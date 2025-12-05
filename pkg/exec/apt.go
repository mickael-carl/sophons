package exec

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/arduino/go-apt-client"

	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/registry"
)

const (
	AptAbsent   string = "absent"
	AptBuildDep string = "build-dep"
	AptLatest   string = "latest"
	AptPresent  string = "present"
	AptFixed    string = "fixed"
)

const (
	AptUpgradeYes  = "yes"
	AptUpgradeFull = "full"
	AptUpgradeDist = "dist"
	AptUpgradeSafe = "safe"
	AptUpgradeNo   = "no"
)

//	@meta{
//	  "deviations": [
//	    "`state` only supports `present`, `latest` and `absent`",
//	    "version strings in package names are not supported"
//	  ]
//	}
type Apt struct {
	*proto.Apt `yaml:",inline"`

	apt   aptClient
	aptFS fs.FS
}

type AptResult struct {
	CommonResult `yaml:",inline"`

	CacheUpdateTime time.Time `yaml:"cache_update_time"`
	CacheUpdated    bool      `yaml:"cache_updated"`
}

func init() {
	reg := registry.TaskRegistration{
		ProtoFactory: func() any { return &proto.Apt{} },
		ProtoWrapper: func(msg any) any { return &proto.Task_Apt{Apt: msg.(*proto.Apt)} },
		ExecAdapter: func(content any) any {
			if c, ok := content.(*proto.Task_Apt); ok {
				return &Apt{Apt: c.Apt}
			}
			return nil
		},
	}
	registry.Register("apt", reg, (*proto.Task_Apt)(nil))
	registry.Register("ansible.builtin.apt", reg, (*proto.Task_Apt)(nil))
}

func (a *Apt) Validate() error {
	supportedUpgrade := map[string]struct{}{
		AptUpgradeYes:  {},
		AptUpgradeNo:   {},
		AptUpgradeFull: {},
		AptUpgradeDist: {},
		AptUpgradeSafe: {},
		"":             {},
	}

	if _, ok := supportedUpgrade[a.Upgrade]; !ok {
		return fmt.Errorf("unsupported upgrade mode: %s", a.Upgrade)
	}

	supportedState := map[string]struct{}{
		AptAbsent:  {},
		AptPresent: {},
		AptLatest:  {},
		"":         {},
	}

	if _, ok := supportedState[string(a.State)]; !ok {
		return fmt.Errorf("unsupported state: %s", a.State)
	}

	return nil
}

func (a *Apt) handleUpdate() (bool, time.Time, error) {
	cacheInfo, err := fs.Stat(a.aptFS, "lists")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, time.UnixMilli(0).UTC(), fmt.Errorf("failed to check cache last refresh time: %w", err)
	}

	var beforeModTime time.Time
	if err == nil {
		beforeModTime = cacheInfo.ModTime()
	}

	if errors.Is(err, os.ErrNotExist) && (a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil) {
		_, cacheErr := a.apt.CheckForUpdates()
		if cacheErr != nil {
			return false, time.UnixMilli(0).UTC(), cacheErr
		}

		// Check if cache actually changed
		afterInfo, statErr := fs.Stat(a.aptFS, "lists")
		if statErr != nil {
			// If the directory still doesn't exist after update (e.g., in tests with mocks),
			// assume the cache was updated since CheckForUpdates succeeded
			if errors.Is(statErr, os.ErrNotExist) {
				return true, time.Now().UTC(), nil
			}
			return false, time.UnixMilli(0).UTC(), fmt.Errorf("failed to check cache after update: %w", statErr)
		}
		afterModTime := afterInfo.ModTime()
		// We only actually updated the cache if the list changed.
		return !afterModTime.Equal(beforeModTime), afterModTime.UTC(), nil
	}

	if a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil && time.Since(cacheInfo.ModTime()).Seconds() > float64(*a.CacheValidTime) {
		_, cacheErr := a.apt.CheckForUpdates()
		if cacheErr != nil {
			return false, beforeModTime.UTC(), cacheErr
		}

		// Check if cache actually changed
		afterInfo, statErr := fs.Stat(a.aptFS, "lists")
		if statErr != nil {
			// If the directory doesn't exist after update (shouldn't happen in real usage,
			// but can happen in tests), treat it as an error since it existed before
			if errors.Is(statErr, os.ErrNotExist) {
				return false, beforeModTime.UTC(), fmt.Errorf("cache directory disappeared after update")
			}
			return false, beforeModTime.UTC(), fmt.Errorf("failed to check cache after update: %w", statErr)
		}
		afterModTime := afterInfo.ModTime()
		// We only actually updated the cache if the list changed.
		return !afterModTime.Equal(beforeModTime), afterModTime.UTC(), nil
	}

	cacheUpdateTime := time.UnixMilli(0)
	if err == nil {
		cacheUpdateTime = cacheInfo.ModTime()
	}

	return false, cacheUpdateTime.UTC(), nil
}

func (a *Apt) Apply(ctx context.Context, _ string, _ bool) (Result, error) {
	if clientFromCtx, ok := ctx.Value(aptClientContextKey).(aptClient); ok {
		a.apt = clientFromCtx
	} else {
		a.apt = &realAptClient{}
	}

	if fsFromCtx, ok := ctx.Value(aptFSContextKey).(fs.FS); ok {
		a.aptFS = fsFromCtx
	} else {
		a.aptFS = os.DirFS("/var/lib/apt")
	}

	result := AptResult{}

	if a.Clean {
		if _, err := a.apt.Clean(); err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to clean apt cache: %w", err)
		}

		result.TaskChanged()

		// This is similar to Ansible's implementation. See
		// https://github.com/ansible/ansible/issues/82611 and
		// https://github.com/ansible/ansible/pull/82800 for some context.
		if (a.Name == nil || len(a.Name.Items) == 0) && (a.Upgrade == "no" || a.Upgrade == "") && a.Deb == "" {
			return &result, nil
		}
	}

	actualState := a.State
	if a.State == "" {
		actualState = AptPresent
	}

	cacheUpdated, cacheUpdateTime, err := a.handleUpdate()
	if err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to update apt cache: %w", err)
	}

	result.CacheUpdated = cacheUpdated
	result.CacheUpdateTime = cacheUpdateTime

	if cacheUpdated {
		result.TaskChanged()
	}

	if a.Upgrade != "" {
		switch a.Upgrade {
		case AptUpgradeYes, AptUpgradeSafe:
			if _, err := a.apt.UpgradeAll(); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to upgrade: %w", err)
			}
		case AptUpgradeDist, AptUpgradeFull:
			if _, err := a.apt.DistUpgrade(); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to dist-upgrade: %w", err)
			}
		default:
			result.TaskFailed()
			return &result, fmt.Errorf("unsupported value of upgrade: %s", a.Upgrade)
		}
		result.TaskChanged()
	}

	if a.Name == nil || len(a.Name.Items) == 0 {
		return &result, nil
	}

	switch actualState {
	case AptPresent:
		installed, err := a.apt.ListInstalled()
		if err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to list installed packages: %w", err)
		}

		toInstall := []*apt.Package{}
		for _, wanted := range a.Name.Items {
			found := false
			for _, p := range installed {
				if p.Name == wanted {
					found = true
					break
				}
			}
			if !found {
				toInstall = append(toInstall, &apt.Package{Name: wanted})
			}
		}

		if len(toInstall) > 0 {
			if out, err := a.apt.Install(toInstall...); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to install package list: %w. Output: %s", err, out)
			}
			result.TaskChanged()
		}
	case AptLatest:
		toInstall := []*apt.Package{}
		for _, p := range a.Name.Items {
			toInstall = append(toInstall, &apt.Package{
				Name: p,
			})
		}

		// We don't need to think about upgrading vs installing: installing an
		// already installed package will upgrade it if a new version is
		// available.
		if len(toInstall) > 0 {
			if out, err := a.apt.Install(toInstall...); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to install package list: %w. Output: %s", err, out)
			}
			result.TaskChanged()
		}
	case AptAbsent:
		toRemove := []*apt.Package{}
		for _, p := range a.Name.Items {
			toRemove = append(toRemove, &apt.Package{
				Name: p,
			})
		}

		if len(toRemove) > 0 {
			if out, err := a.apt.Remove(toRemove...); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to remove package list: %w. Output: %s", err, out)
			}
			result.TaskChanged()
		}
	default:
		result.TaskFailed()
		return &result, fmt.Errorf("state %s is not implemented for apt", actualState)
	}

	return &result, nil
}
