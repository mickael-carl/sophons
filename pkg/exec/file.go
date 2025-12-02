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
	"syscall"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	"github.com/mickael-carl/sophons/pkg/proto"
)

const (
	FileAbsent    string = "absent"
	FileDirectory string = "directory"
	FileFile      string = "file"
	FileHard      string = "hard"
	FileLink      string = "link"
	FileTouch     string = "touch"
)

//	@meta{
//	  "deviations": [
//	    "`state=hard` is not implemented."
//	  ]
//	}
type File struct {
	proto.File `yaml:",inline"`
}

type FileResult struct {
	CommonResult `yaml:",inline"`

	Dest  string
	Uid   uint32
	Gid   uint32
	Owner string
	Group string
	Mode  string
	Path  string
	State string
	Size  uint64
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

// needsModeOrOwnershipChange checks if a file/directory requires changes to
// mode, uid, or gid. Returns true if any of the specified attributes differ
// from current values.
func needsModeOrOwnershipChange(path string, mode any, uid, gid int) (bool, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return false, err
	}

	st, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("couldn't get metadata for %s", path)
	}

	if mode != nil {
		currentMode := stat.Mode().Perm()
		var desiredMode os.FileMode

		switch v := mode.(type) {
		case string:
			if v != "" {
				// Try parsing as octal first
				if numMode, err := strconv.ParseUint(v, 8, 32); err == nil {
					desiredMode = os.FileMode(numMode)
				} else {
					// It's a symbolic mode like "u+x", compute what it would be
					newMode, err := util.NewModeFromSpec(os.DirFS(filepath.Dir(path)), path, v)
					if err != nil {
						return false, err
					}
					desiredMode = newMode
				}
			}
		case int:
			desiredMode = os.FileMode(v)
		case int64:
			desiredMode = os.FileMode(v)
		case uint64:
			desiredMode = os.FileMode(v)
		case *uint64:
			if v != nil {
				desiredMode = os.FileMode(*v)
			}
		default:
			return false, fmt.Errorf("unsupported mode type %T", mode)
		}

		if desiredMode != 0 && currentMode != desiredMode {
			return true, nil
		}
	}

	if uid != -1 && st.Uid != uint32(uid) {
		return true, nil
	}

	if gid != -1 && st.Gid != uint32(gid) {
		return true, nil
	}

	return false, nil
}

func (f *File) Apply(_ context.Context, _ string, _ bool) (Result, error) {
	var follow bool
	// The default for `follow` is true.
	if f.Follow == nil {
		follow = true
	} else {
		follow = *f.Follow
	}

	result := FileResult{}

	exists := false
	_, err := os.Lstat(f.Path)
	if err == nil {
		exists = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		result.TaskFailed()
		return &result, err
	}

	actualState := f.State
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
		result.TaskFailed()
		return &result, err
	}

	gid, err := util.GetGid(f.Group)
	if err != nil {
		result.TaskFailed()
		return &result, err
	}

	changed := false

	switch actualState {
	case FileAbsent:
		result.Path = f.Path
		result.State = FileAbsent
		if exists {
			if err := os.RemoveAll(f.Path); err != nil {
				result.TaskFailed()
				return &result, err
			}
			result.TaskChanged()
		}
		return &result, nil

	case FileDirectory:
		result.Path = f.Path
		// If f.Mode is not specified, i.e. we don't specify a mode, Ansible
		// says it'll use the default umask. To emulate that, but not do
		// anything on existing files/directories, we call MkdirAll, which
		// won't alter existing things as expected.
		if err := os.MkdirAll(f.Path, os.FileMode(0o755)); err != nil {
			result.TaskFailed()
			return &result, err
		}
		changed = !exists

		if f.Recurse {
			if err := filepath.WalkDir(f.Path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.Type()&os.ModeSymlink != 0 {
					// For symlinks, only check uid/gid (mode changes don't
					// apply to symlinks themselves)
					needsUpdate, err := needsModeOrOwnershipChange(path, nil, uid, gid)
					if err != nil {
						return err
					}

					if needsUpdate {
						changed = true
						if err := os.Lchown(path, uid, gid); err != nil {
							return err
						}
					}
				} else {
					needsUpdate, err := needsModeOrOwnershipChange(path, f.Mode.GetValue(), uid, gid)
					if err != nil {
						return err
					}

					if needsUpdate {
						changed = true
						if err := util.ApplyModeAndIDs(path, f.Mode.GetValue(), uid, gid); err != nil {
							return err
						}
					}
				}

				return nil
			}); err != nil {
				result.TaskFailed()
				return &result, err
			}
		} else {
			needsUpdate, err := needsModeOrOwnershipChange(f.Path, f.Mode.GetValue(), uid, gid)
			if err != nil {
				result.TaskFailed()
				return &result, err
			}

			if needsUpdate {
				changed = true
				if err := util.ApplyModeAndIDs(f.Path, f.Mode.GetValue(), uid, gid); err != nil {
					result.TaskFailed()
					return &result, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
				}
			}
		}

	case FileFile:
		result.Path = f.Path
		if !exists {
			result.TaskFailed()
			return &result, errors.New("file does not exist")
		}

		// Per Ansible docs: if no property are set, state=file does nothing.
		if f.Mode == nil && f.Owner == "" && f.Group == "" {
			result.TaskSkipped()
			return &result, nil
		}

		needsUpdate, err := needsModeOrOwnershipChange(f.Path, f.Mode.GetValue(), uid, gid)
		if err != nil {
			result.TaskFailed()
			return &result, err
		}

		if needsUpdate {
			changed = true
			if err := util.ApplyModeAndIDs(f.Path, f.Mode.GetValue(), uid, gid); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
		}

	case FileHard:
		result.Dest = f.Path
		result.TaskFailed()
		return &result, errors.New("not implemented")

	case FileLink:
		result.Dest = f.Path
		if !exists {
			if err := os.Symlink(f.Src, f.Path); err != nil {
				result.TaskFailed()
				return &result, err
			}
			changed = true
		} else {
			existingSrc, err := os.Readlink(f.Path)
			if err != nil {
				result.TaskFailed()
				return &result, err
			}

			if existingSrc != f.Src {
				if err := os.Remove(f.Path); err != nil {
					result.TaskFailed()
					return &result, err
				}

				if err := os.Symlink(f.Src, f.Path); err != nil {
					result.TaskFailed()
					return &result, err
				}
				changed = true
			}
		}

		if follow {
			needsUpdate, err := needsModeOrOwnershipChange(f.Path, f.Mode.GetValue(), uid, gid)
			if err != nil {
				result.TaskFailed()
				return &result, err
			}

			if needsUpdate {
				if err := util.ApplyModeAndIDs(f.Path, f.Mode.GetValue(), uid, gid); err != nil {
					result.TaskFailed()
					return &result, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
				}
				changed = true
			}
		}

	case FileTouch:
		result.Dest = f.Path
		if !exists {
			if _, err := os.Create(f.Path); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to create %s: %w", f.Path, err)
			}
			changed = true
		}

		// The Ansible docs say that if the file exists, atime and mtime will
		// be updated but not more. That proves to not be accurate:
		// permissions, uid and gid will be updated too.
		needsUpdate, err := needsModeOrOwnershipChange(f.Path, f.Mode.GetValue(), uid, gid)
		if err != nil {
			result.TaskFailed()
			return &result, err
		}

		if needsUpdate {
			if err := util.ApplyModeAndIDs(f.Path, f.Mode.GetValue(), uid, gid); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("couldn't apply mode and IDs to %s: %w", f.Path, err)
			}
			changed = true
		}

	default:
		result.TaskFailed()
		return &result, errors.New("unsupported state parameter")
	}

	if changed {
		result.TaskChanged()
	}

	if actualState != FileAbsent {
		stat, err := os.Lstat(f.Path)
		if err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to stat %s: %w", f.Path, err)
		}
		result.Mode = fmt.Sprintf("%#o", stat.Mode().Perm())

		st, ok := stat.Sys().(*syscall.Stat_t)
		if !ok {
			result.TaskFailed()
			return &result, fmt.Errorf("couldn't get metadata for %s: %w", f.Path, err)
		}

		switch mode := stat.Mode(); {
		case mode&os.ModeSymlink != 0:
			result.State = FileLink
		case mode.IsDir():
			result.State = FileDirectory
		default:
			// We still need to check if the target is actually a hard link.
			if st.Nlink > 1 {
				result.State = FileHard
			} else {
				// It's either a regular file, at which point this is correct,
				// or it's e.g. a char device, or something that's basically a
				// special file. Ansible treats the latter as a file so we do
				// the same.
				result.State = FileFile
			}
		}

		result.Uid = st.Uid
		result.Gid = st.Gid
		result.Size = uint64(st.Size)

		owner, err := user.LookupId(strconv.Itoa(int(st.Uid)))
		// Ignore errors, it could be the UID doesn't have a matching user.
		if err == nil {
			result.Owner = owner.Username
		}

		group, err := user.LookupGroupId(strconv.Itoa(int(st.Gid)))
		// Ignore errors, it could be the GID doesn't have a matching group.
		if err == nil {
			result.Group = group.Name
		}
	}

	return &result, nil
}
