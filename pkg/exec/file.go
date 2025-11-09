package exec

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/goccy/go-yaml"
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
//	    "`state=hard` is not implemented.",
//	    "There is no default for `state`, it is required."
//	  ]
//	}
type File struct {
	Path    string    `sophons:"implemented"`
	Follow  *bool     `sophons:"implemented"`
	Group   string    `sophons:"implemented"`
	Mode    string    `sophons:"implemented"`
	Owner   string    `sophons:"implemented"`
	Recurse bool      `sophons:"implemented"`
	Src     string    `sophons:"implemented"`
	State   FileState `sophons:"implemented"`

	AccessTime             string `yaml:"access_time"`
	AccessTimeFormat       string `yaml:"access_time_format"`
	Attributes             string
	Force                  bool
	ModificationTime       string `yaml:"modification_time"`
	ModificationTimeFormat string `yaml:"modification_time_format"`
	Selevel                string
	Serole                 string
	Setype                 string
	Seuser                 string
	UnsafeWrites           bool `yaml:"unsafe_writes"`
}

func init() {
	RegisterTaskType("file", func() TaskContent { return &File{} })
	RegisterTaskType("ansible.builtin.file", func() TaskContent { return &File{} })
}

func (f *File) UnmarshalYAML(b []byte) error {
	type plain File
	if err := yaml.Unmarshal(b, (*plain)(f)); err != nil {
		return err
	}

	type file struct {
		Dest string
		Name string
	}

	var aux file
	if err := yaml.Unmarshal(b, &aux); err != nil {
		return err
	}

	if f.Path == "" {
		if aux.Dest != "" {
			f.Path = aux.Dest
		} else if aux.Name != "" {
			f.Path = aux.Name
		}
	}

	return nil
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

func applyModeAndIDs(path, mode string, uid, gid int) error {
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
		FileAbsent:    {},
		FileDirectory: {},
		FileFile:      {},
		FileHard:      {},
		FileLink:      {},
		FileTouch:     {},
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

func (f *File) Apply(_ context.Context, _ string, _ bool) error {
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
		// If f.Mode is not specified, i.e. we don't specify a mode, Ansible
		// says it'll use the default umask. To emulate that, but not do
		// anything on existing files/directories, we call MkdirAll, which
		// won't alter existing things as expected.
		if err := os.MkdirAll(f.Path, os.FileMode(0o755)); err != nil {
			return err
		}

		if f.Recurse {
			if err := filepath.WalkDir(f.Path, func(path string, d fs.DirEntry, err error) error {
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
						if err := util.ChmodFromString(path, f.Mode); err != nil {
							return err
						}
					}
				} else {
					if err := applyModeAndIDs(path, f.Mode, uid, gid); err != nil {
						return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
					}
				}

				return nil
			}); err != nil {
				return err
			}
		} else {
			if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
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

		if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
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
				return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileTouch:
		if !exists {
			if _, err := os.Create(f.Path); err != nil {
				return fmt.Errorf("failed to create %s: %w", f.Path, err)
			}
		}

		// The Ansible docs say that if the file exists, atime and mtime will
		// be updated but not more. That proves to not be accurate:
		// permissions, uid and gid will be updated too.
		if err := applyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
		}

	default:
		return errors.New("unsupported state parameter")
	}

	return nil
}
