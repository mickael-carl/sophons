package exec

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/mickael-carl/sophons/pkg/util"
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

//	@meta{
//	  "deviations": [
//	    "`state=hard` is not implemented."
//	  ]
//	}
type File struct {
	CommonTask

	Path    jinjaString `sophons:"implemented"`
	Follow  *bool       `sophons:"implemented"`
	Group   jinjaString `sophons:"implemented"`
	Mode    jinjaString `sophons:"implemented"`
	Owner   jinjaString `sophons:"implemented"`
	Recurse bool        `sophons:"implemented"`
	Src     jinjaString `sophons:"implemented"`
	State   FileState   `sophons:"implemented"`

	AccessTime             jinjaString `yaml:"access_time"`
	AccessTimeFormat       jinjaString `yaml:"access_time_format"`
	Attributes             jinjaString
	Force                  bool
	ModificationTime       jinjaString `yaml:"modification_time"`
	ModificationTimeFormat jinjaString `yaml:"modification_time_format"`
	Selevel                jinjaString
	Serole                 jinjaString
	Setype                 jinjaString
	Seuser                 jinjaString
	UnsafeWrites           bool `yaml:"unsafe_writes"`
}

func init() {
	RegisterTaskType("file", func() Task { return &File{} })
	RegisterTaskType("ansible.builtin.file", func() Task { return &File{} })
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

func applyModeAndIDs(path string, mode string, uid, gid int) error {
	if mode != "" {
		// First try to parse mode as octal. If that fails we'll assume it's a
		// string-based mode spec.
		numMode, err := strconv.ParseUint(mode, 10, 32)
		if err == nil {
			return os.Chmod(path, os.FileMode(numMode))
		}

		if err := util.ChmodFromString(path, mode); err != nil {
			return err
		}
	}

	return os.Chown(path, uid, gid)
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

	if string(f.Path) == "" {
		return errors.New("path is required")
	}

	if f.Recurse && f.State != FileDirectory {
		return errors.New("recurse option requires state to be 'directory'")
	}

	// TODO: not exactly true: ansible will use the previous state of the link.
	if (f.State == FileLink || f.State == FileHard) && string(f.Src) == "" {
		return errors.New("src option is required when state is 'link' or 'hard'")
	}

	return nil
}

func (f *File) Apply(_ string) error {
	var follow bool
	// The default for `follow` is true.
	if f.Follow == nil {
		follow = true
	} else {
		follow = *f.Follow
	}

	actualState := f.State

	exists := false
	_, err := os.Lstat(string(f.Path))
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

	uid, err := getUid(string(f.Owner))
	if err != nil {
		return err
	}

	gid, err := getGid(string(f.Owner))
	if err != nil {
		return err
	}

	switch actualState {
	case FileAbsent:
		return os.RemoveAll(string(f.Path))

	case FileDirectory:
		// If f.Mode is not specified, i.e. we don't specify a mode, Ansible
		// says it'll use the default umask. To emulate that, but not do
		// anything on existing files/directories, we call MkdirAll, which
		// won't alter existing things as expected.
		if err := os.MkdirAll(string(f.Path), os.FileMode(0755)); err != nil {
			return err
		}

		if f.Recurse {
			if err := filepath.WalkDir(string(f.Path), func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// In the case where the dir exists, we don't want to change
				// permissions if the mode is unset.

				if d.Type()&os.ModeSymlink != 0 {
					if err := os.Lchown(path, uid, gid); err != nil {
						return err
					}
					if f.Mode != "" {
						if err := util.ChmodFromString(path, string(f.Mode)); err != nil {
							return err
						}
					}
				} else {
					if err := applyModeAndIDs(path, string(f.Mode), uid, gid); err != nil {
						return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
					}
				}

				return nil
			}); err != nil {
				return err
			}
		} else {
			if err := applyModeAndIDs(string(f.Path), string(f.Mode), uid, gid); err != nil {
				return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileFile:
		if !exists {
			return errors.New("file does not exist")
		}

		// Per Ansible docs: if no property are set, state=file does nothing.
		if f.Mode == "" && f.Owner == "" && f.Group == "" {
			return nil
		}

		if err := applyModeAndIDs(string(f.Path), string(f.Mode), uid, gid); err != nil {
			return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
		}

	case FileHard:
		return errors.New("not implemented")

	case FileLink:
		if !exists {
			if err := os.Symlink(string(f.Src), string(f.Path)); err != nil {
				return err
			}
		} else {
			existingSrc, err := os.Readlink(string(f.Path))
			if err != nil {
				return err
			}

			if existingSrc != string(f.Src) {
				if err := os.Remove(string(f.Path)); err != nil {
					return err
				}

				if err := os.Symlink(string(f.Src), string(f.Path)); err != nil {
					return err
				}
			}
		}

		if follow {
			if err := applyModeAndIDs(string(f.Path), string(f.Mode), uid, gid); err != nil {
				return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileTouch:
		if !exists {
			if _, err := os.Create(string(f.Path)); err != nil {
				return fmt.Errorf("failed to create %s: %w", f.Path, err)
			}
		}

		// The Ansible docs say that if the file exists, atime and mtime will
		// be updated but not more. That proves to not be accurate:
		// permissions, uid and gid will be updated too.
		if err := applyModeAndIDs(string(f.Path), string(f.Mode), uid, gid); err != nil {
			return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
		}

	default:
		return errors.New("unsupported state parameter")
	}

	return nil
}
