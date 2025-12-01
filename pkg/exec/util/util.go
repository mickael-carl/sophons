package util

import (
	"os/user"
	"strconv"
)

func GetStringSlice(i any) []string {
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
