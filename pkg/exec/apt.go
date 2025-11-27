package exec

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/arduino/go-apt-client"
	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/proto"
)

const (
	AptAbsent   string = Absent
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
	proto.Apt `yaml:",inline"`

	apt   aptClient
	aptFS fs.FS
}

type AptResult struct {
	CommonResult `yaml:",inline"`

	CacheUpdateTime time.Time `yaml:"cache_update_time"`
	CacheUpdated    bool      `yaml:"cache_updated"`
}

func init() {
	RegisterTaskType("apt", func() TaskContent { return &Apt{} })
	RegisterTaskType("ansible.builtin.apt", func() TaskContent { return &Apt{} })
}

func (a *Apt) UnmarshalYAML(b []byte) error {
	type plain Apt
	if err := yaml.Unmarshal(b, (*plain)(a)); err != nil {
		return err
	}

	type apt struct {
		Pkg         proto.PackageList
		Package     proto.PackageList
		UpdateCache bool `yaml:"update-cache"`
	}

	var aux apt
	if err := yaml.Unmarshal(b, &aux); err != nil {
		return err
	}

	if a.Name == nil || len(a.Name.Items) == 0 {
		if len(aux.Package.Items) != 0 {
			a.Name = &aux.Package
		} else if len(aux.Pkg.Items) != 0 {
			a.Name = &aux.Pkg
		}
	}

	if a.UpdateCache == nil {
		a.UpdateCache = &aux.UpdateCache
	}

	return nil
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

	if errors.Is(err, os.ErrNotExist) && (a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil) {
		_, cacheErr := a.apt.CheckForUpdates()
		return true, time.Now().UTC(), cacheErr
	}

	if a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil && time.Since(cacheInfo.ModTime()).Seconds() > float64(*a.CacheValidTime) {
		_, cacheErr := a.apt.CheckForUpdates()
		return true, time.Now().UTC(), cacheErr
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
