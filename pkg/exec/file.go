package exec

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

type FileState State

const (
	FileAbsent    FileState = FileState(Absent)
	FileDirectory FileState = "directory"
	FileFile      FileState = "file"
	FileHard      FileState = "hard"
	FileLink      FileState = "link"
	FileTouch     FileState = "touch"
)

type File struct {
	Path   string
	Follow *bool
	Group  string
	// TODO: Mode should support string syntax, e.g. "u+rw,o=r".
	Mode    uint32
	Owner   string
	Recurse bool
	Src     string
	State   FileState
}

func getUid(uidOrUserName string) (int, error) {
	// -1 is the value for not changing owner in calls to Chown/Lchown.
	uid := int(-1)
	if uidOrUserName != "" {
		u, err := user.LookupId(uidOrUserName)
		if err == nil {
			uid, err = strconv.Atoi(u.Uid)
			if err != nil {
				return -1, err
			}
		} else {
			u, err := user.Lookup(uidOrUserName)
			if err != nil {
				return -1, err
			}
			uid, err = strconv.Atoi(u.Uid)
			if err != nil {
				return -1, err
			}
		}
	}

	return uid, nil
}

func getGid(gidOrGroupName string) (int, error) {
	// -1 is the value for not changing group in calls to Chown/Lchown.
	gid := int(-1)
	if gidOrGroupName != "" {
		g, err := user.LookupGroupId(gidOrGroupName)
		if err == nil {
			gid, err = strconv.Atoi(g.Gid)
			if err != nil {
				return -1, err
			}
		} else {
			g, err := user.LookupGroup(gidOrGroupName)
			if err != nil {
				return -1, err
			}
			gid, err = strconv.Atoi(g.Gid)
			if err != nil {
				return -1, err
			}
		}
	}

	return gid, nil
}

func applyModeAndIDs(path string, mode uint32, uid, gid int) error {
	if mode != 0 {
		if err := os.Chmod(path, os.FileMode(mode)); err != nil {
			return err
		}
	}

	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}

	return nil
}

func (f *File) Validate() error {
	validStates := map[FileState]struct{}{
		FileAbsent:    struct{}{},
		FileDirectory: struct{}{},
		FileFile:      struct{}{},
		FileHard:      struct{}{},
		FileLink:      struct{}{},
		FileTouch:     struct{}{},
	}

	if _, ok := validStates[f.State]; !ok {
		return errors.New("invalid state")
	}

	if f.Path == "" {
		return errors.New("path is required")
	}

	if f.Recurse && f.State != FileDirectory {
		return errors.New("recurse option requires state to be 'directory'")
	}

	// TODO: not exactly true: ansible will use the previous state of the link.
	if (f.State == FileLink || f.State == FileHard) && f.Src == "" {
		return errors.New("src option is required when state is 'link' or 'hard'")
	}

	return nil
}

func (f *File) Apply() error {
	var follow bool
	// The default for `follow` is true.
	if f.Follow == nil {
		follow = true
	} else {
		follow = *f.Follow
	}

	actualState := f.State

	exists := false
	_, err := os.Lstat(f.Path)
	if err == nil {
		exists = true
	} else {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	if f.State == "" {
		if !exists {
			if f.Recurse {
				actualState = FileDirectory
			} else {
				actualState = FileFile
			}
		}
	}

	uid, err := getUid(f.Owner)
	if err != nil {
		return err
	}

	gid, err := getGid(f.Owner)
	if err != nil {
		return err
	}

	switch actualState {
	case FileAbsent:
		return os.RemoveAll(f.Path)

	case FileDirectory:
		// If f.Mode is 0, i.e. we don't specify a mode, Ansible says it'll use
		// the default umask. To emulate that, but not do anything on existing
		// files/directories, we call MkdirAll, which won't alter existing
		// things as expected.
		mode := f.Mode
		if mode == 0 {
			mode = 0755
		}
		if err := os.MkdirAll(f.Path, os.FileMode(mode)); err != nil {
			return err
		}

		if f.Recurse {
			if err := filepath.WalkDir(f.Path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// In the case where the dir exists, we don't want to change
				// permissions if the mode is unset.
				if f.Mode != 0 {
					if err := os.Chmod(path, os.FileMode(f.Mode)); err != nil {
						return err
					}
				}

				if d.Type()&os.ModeSymlink != 0 {
					if err := os.Lchown(path, uid, gid); err != nil {
						return err
					}
				} else {
					if err := os.Chown(path, uid, gid); err != nil {
						return err
					}
				}

				return nil
			}); err != nil {
				return err
			}
		} else {
			if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
				return err
			}
		}

	case FileFile:
		if !exists {
			return errors.New("file does not exist")
		}

		// Per Ansible docs: if no property are set, state=file does nothing.
		if f.Mode == 0 && f.Owner == "" && f.Group == "" {
			return nil
		}

		if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return err
		}

	case FileHard:
		return errors.New("not implemented")

	case FileLink:
		if !exists {
			if err := os.Symlink(f.Src, f.Path); err != nil {
				return err
			}
		} else {
			existingSrc, err := os.Readlink(f.Path)
			if err != nil {
				return err
			}

			if existingSrc != f.Src {
				if err := os.Remove(f.Path); err != nil {
					return err
				}

				if err := os.Symlink(f.Src, f.Path); err != nil {
					return err
				}
			}
		}

		if follow {
			if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
				return err
			}
		}

	case FileTouch:
		if !exists {
			if _, err := os.Create(f.Path); err != nil {
				return err
			}
		}

		// The Ansible docs say that if the file exists, atime and mtime will
		// be updated but not more. That proves to not be accurate:
		// permissions, uid and gid will be updated too.
		if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return err
		}

	default:
		return errors.New("unsupported state parameter")
	}

	return nil
}
