package util

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

func shouldApply(creates, removes string) (bool, error) {
	if creates != "" {
		matches, err := filepath.Glob(creates)
		if err != nil {
			return false, err
		}
		return len(matches) == 0, nil
	}

	if removes != "" {
		matches, err := filepath.Glob(removes)
		if err != nil {
			return false, err
		}
		return len(matches) > 0, nil
	}

	return true, nil
}

func GetStringSlice(i interface{}) []string {
	if i == nil {
		return nil
	}
	if str, ok := i.(string); ok {
		return []string{str}
	}
	if slice, ok := i.([]string); ok {
		return slice
	}
	return nil
}

func GetUid(uidOrUserName string) (int, error) {
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

func GetGid(gidOrGroupName string) (int, error) {
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

func ApplyModeAndIDs(path, mode string, uid, gid int) error {
	if mode != "" {
		// First try to parse mode as octal. If that fails we'll assume it's a
		// string-based mode spec.
		numMode, err := strconv.ParseUint(mode, 10, 32)
		if err == nil {
			return os.Chmod(path, os.FileMode(numMode))
		}

		if err := ChmodFromString(path, mode); err != nil {
			return err
		}
	}

	return os.Chown(path, uid, gid)
}
