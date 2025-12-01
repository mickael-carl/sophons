package util

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrInvalidMode = errors.New("invalid mode specification")

const (
	userRead   = 0o400
	userWrite  = 0o200
	userExec   = 0o100
	groupRead  = 0o040
	groupWrite = 0o020
	groupExec  = 0o010
	otherRead  = 0o004
	otherWrite = 0o002
	otherExec  = 0o001
)

func ChmodFromString(path, spec string) error {
	mode, err := NewModeFromSpec(os.DirFS(filepath.Dir(path)), path, spec)
	if err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func NewModeFromSpec(fsys fs.FS, path, spec string) (os.FileMode, error) {
	info, err := fs.Stat(fsys, filepath.Base(path))
	if err != nil {
		return 0, err
	}
	mode := info.Mode().Perm()

	for part := range strings.SplitSeq(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		opIndex := strings.IndexAny(part, "=+-")
		if opIndex == -1 || opIndex == len(part)-1 {
			return 0, ErrInvalidMode
		}

		who := part[:opIndex]
		if strings.Contains(who, "a") {
			who = "ugo"
		}

		op := part[opIndex]
		perms := part[opIndex+1:]

		var mask os.FileMode
		for _, p := range perms {
			switch p {
			case 'r':
				if strings.Contains(who, "u") {
					mask |= userRead
				}
				if strings.Contains(who, "g") {
					mask |= groupRead
				}
				if strings.Contains(who, "o") {
					mask |= otherRead
				}
			case 'w':
				if strings.Contains(who, "u") {
					mask |= userWrite
				}
				if strings.Contains(who, "g") {
					mask |= groupWrite
				}
				if strings.Contains(who, "o") {
					mask |= otherWrite
				}
			case 'x':
				if strings.Contains(who, "u") {
					mask |= userExec
				}
				if strings.Contains(who, "g") {
					mask |= groupExec
				}
				if strings.Contains(who, "o") {
					mask |= otherExec
				}
			default:
				return 0, ErrInvalidMode
			}
		}

		switch op {
		case '+':
			mode |= mask
		case '-':
			mode &^= mask
		case '=':
			if strings.Contains(who, "u") {
				mode &^= 0o700
				mode |= mask & 0o700
			}
			if strings.Contains(who, "g") {
				mode &^= 0o070
				mode |= mask & 0o070
			}
			if strings.Contains(who, "o") {
				mode &^= 0o007
				mode |= mask & 0o007
			}
		default:
			return 0, ErrInvalidMode
		}
	}

	return mode, nil
}

func ApplyModeAndIDs(path string, mode any, uid, gid int) error {
	if mode != nil {
		var chmodErr error
		switch v := mode.(type) {
		case string:
			if v == "" {
				break
			}
			if numMode, err := strconv.ParseUint(v, 8, 32); err == nil {
				chmodErr = os.Chmod(path, os.FileMode(numMode))
			} else {
				chmodErr = ChmodFromString(path, v)
			}

		case int:
			chmodErr = os.Chmod(path, os.FileMode(v))
		case int64:
			chmodErr = os.Chmod(path, os.FileMode(v))
		case uint64:
			chmodErr = os.Chmod(path, os.FileMode(v))
		case *uint64:
			if v != nil {
				chmodErr = os.Chmod(path, os.FileMode(*v))
			}
		default:
			return fmt.Errorf("unsupported mode type %T", mode)
		}

		if chmodErr != nil {
			return chmodErr
		}
	}

	return os.Chown(path, uid, gid)
}
