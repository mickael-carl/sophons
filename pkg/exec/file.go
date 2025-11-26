package exec

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/exec/util"
)

const (
	FileAbsent    string = Absent
	FileDirectory string = "directory"
	FileFile      string = "file"
	FileHard      string = "hard"
	FileLink      string = "link"
	FileTouch     string = "touch"
)

//	@meta{
//	  "deviations": [
//	    "`state=hard` is not implemented.",
//	    "There is no default for `state`, it is required."
//	  ]
//	}
type File struct {
	Path    string `sophons:"implemented"`
	Follow  *bool  `sophons:"implemented"`
	Group   string `sophons:"implemented"`
	Mode    any    `sophons:"implemented"`
	Owner   string `sophons:"implemented"`
	Recurse bool   `sophons:"implemented"`
	Src     string `sophons:"implemented"`
	State   string `sophons:"implemented"`

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

type FileResult struct {
	CommonResult `yaml:",inline"`
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

func (f *File) Validate() error {
	validStates := map[string]struct{}{
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

func (f *File) Apply(_ context.Context, _ string, _ bool) (Result, error) {
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
	} else if !errors.Is(err, fs.ErrNotExist) {
		return &FileResult{}, err
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

	uid, err := util.GetUid(f.Owner)
	if err != nil {
		return &FileResult{}, err
	}

	gid, err := util.GetGid(f.Owner)
	if err != nil {
		return &FileResult{}, err
	}

	switch actualState {
	case FileAbsent:
		return &FileResult{}, os.RemoveAll(f.Path)

	case FileDirectory:
		// If f.Mode is not specified, i.e. we don't specify a mode, Ansible
		// says it'll use the default umask. To emulate that, but not do
		// anything on existing files/directories, we call MkdirAll, which
		// won't alter existing things as expected.
		if err := os.MkdirAll(f.Path, os.FileMode(0o755)); err != nil {
			return &FileResult{}, err
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
					if f.Mode != nil {
						if err := util.ApplyModeAndIDs(path, f.Mode, -1, -1); err != nil {
							return err
						}
					}
				} else {
					if err := util.ApplyModeAndIDs(path, f.Mode, uid, gid); err != nil {
						return fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
					}
				}

				return nil
			}); err != nil {
				return &FileResult{}, err
			}
		} else {
			if err := util.ApplyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
				return &FileResult{}, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileFile:
		if !exists {
			return &FileResult{}, errors.New("file does not exist")
		}

		// Per Ansible docs: if no property are set, state=file does nothing.
		if f.Mode == nil && f.Owner == "" && f.Group == "" {
			return &FileResult{}, nil
		}

		if err := util.ApplyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return &FileResult{}, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
		}

	case FileHard:
		return &FileResult{}, errors.New("not implemented")

	case FileLink:
		if !exists {
			if err := os.Symlink(f.Src, f.Path); err != nil {
				return &FileResult{}, err
			}
		} else {
			existingSrc, err := os.Readlink(f.Path)
			if err != nil {
				return &FileResult{}, err
			}

			if existingSrc != f.Src {
				if err := os.Remove(f.Path); err != nil {
					return &FileResult{}, err
				}

				if err := os.Symlink(f.Src, f.Path); err != nil {
					return &FileResult{}, err
				}
			}
		}

		if follow {
			if err := util.ApplyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
				return &FileResult{}, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileTouch:
		if !exists {
			if _, err := os.Create(f.Path); err != nil {
				return &FileResult{}, fmt.Errorf("failed to create %s: %w", f.Path, err)
			}
		}

		// The Ansible docs say that if the file exists, atime and mtime will
		// be updated but not more. That proves to not be accurate:
		// permissions, uid and gid will be updated too.
		if err := util.ApplyModeAndIDs(f.Path, f.Mode, uid, gid); err != nil {
			return &FileResult{}, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
		}

	default:
		return &FileResult{}, errors.New("unsupported state parameter")
	}

	return &FileResult{}, nil
}
