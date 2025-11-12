package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arduino/go-apt-client"
	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/exec/util"
)

type AptState State

const (
	AptAbsent   AptState = AptState(Absent)
	AptBuildDep AptState = "build-dep"
	AptLatest   AptState = "latest"
	AptPresent  AptState = "present"
	AptFixed    AptState = "fixed"
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
//	    "version strings in package names are not supported",
//	    "`name` needs to be a list (one element is ok), a single string is not supported"
//	  ]
//	}
type Apt struct {
	AllowChangeHeldPackages  bool    `yaml:"allow_change_held_packages"`
	AllowDowngrade           bool    `yaml:"allow_downgrade"`
	AllowUnauthenticated     bool    `yaml:"allow_unauthenticated"`
	AutoInstallModuleDeps    *bool   `yaml:"auto_install_module_deps"`
	AutoClean                bool    `yaml:"autoclean"`
	AutoRemove               bool    `yaml:"autoremove"`
	CacheValidTime           *uint64 `yaml:"cache_valid_time" sophons:"implemented"`
	Clean                    bool    `sophons:"implemented"`
	Deb                      string
	DefaultRelease           string `yaml:"default_release"`
	DpkgOptions              string `yaml:"dpkg_options"`
	FailOnAutoremove         bool   `yaml:"fail_on_autoremove"`
	Force                    bool
	ForceAptGet              bool        `yaml:"force_apt_get"`
	InstallRecommends        bool        `yaml:"install_recommends"`
	LockTimeout              int64       `yaml:"lock_timeout"`
	Name                     interface{} `sophons:"implemented"`
	OnlyUpgrade              bool        `yaml:"only_upgrade"`
	PolicyRCD                int64       `yaml:"policy_rc_d"`
	Purge                    bool
	State                    AptState `sophons:"implemented"`
	UpdateCache              *bool    `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64   `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64   `yaml:"update_cache_retry_max_delay"`
	Upgrade                  string   `sophons:"implemented"`

	apt aptClient
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
		Pkg         interface{}
		Package     interface{}
		UpdateCache bool `yaml:"update-cache"`
	}

	var aux apt
	if err := yaml.Unmarshal(b, &aux); err != nil {
		return err
	}

	if a.Name == nil {
		if aux.Package != nil {
			a.Name = aux.Package
		} else if aux.Pkg != nil {
			a.Name = aux.Pkg
		}
	}

	if name, ok := a.Name.([]interface{}); ok {
		s := []string{}
		for _, v := range name {
			vStr, ok := v.(string)
			if !ok {
				return fmt.Errorf("package name is not a string: %v", v)
			}
			s = append(s, vStr)
		}
		a.Name = s
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

	supportedState := map[AptState]struct{}{
		AptAbsent:  {},
		AptPresent: {},
		AptLatest:  {},
		"":         {},
	}

	if _, ok := supportedState[AptState(a.State)]; !ok {
		return fmt.Errorf("unsupported state: %s", a.State)
	}

	return nil
}

func (a *Apt) handleUpdate() error {
	cacheInfo, err := os.Stat("/var/lib/apt/lists")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check cache last refresh time: %w", err)
	}

	if errors.Is(err, os.ErrNotExist) && a.UpdateCache != nil && *a.UpdateCache {
		_, cacheErr := a.apt.CheckForUpdates()
		return cacheErr
	}

	if err == nil {
		if a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil && time.Since(cacheInfo.ModTime()).Seconds() > float64(*a.CacheValidTime) {
			_, cacheErr := a.apt.CheckForUpdates()
			return cacheErr
		}
	}

	return nil
}

func (a *Apt) Apply(_ context.Context, _ string, _ bool) error {
	if a.apt == nil {
		a.apt = &realAptClient{}
	}
	name := util.GetStringSlice(a.Name)

	if a.Clean {
		if _, err := a.apt.Clean(); err != nil {
			return fmt.Errorf("failed to clean apt cache: %w", err)
		}

		// This is similar to Ansible's implementation. See
		// https://github.com/ansible/ansible/issues/82611 and
		// https://github.com/ansible/ansible/pull/82800 for some context.
		if len(name) == 0 && (a.Upgrade == "no" || a.Upgrade == "") && a.Deb == "" {
			return nil
		}
	}

	actualState := AptState(a.State)
	if a.State == "" {
		actualState = AptPresent
	}

	if err := a.handleUpdate(); err != nil {
		return fmt.Errorf("failed to update apt cache: %w", err)
	}

	if a.Upgrade != "" {
		switch a.Upgrade {
		case "yes", "safe":
			if _, err := a.apt.UpgradeAll(); err != nil {
				return fmt.Errorf("failed to upgrade: %w", err)
			}
		case "dist", "full":
			if _, err := a.apt.DistUpgrade(); err != nil {
				return fmt.Errorf("failed to dist-upgrade: %w", err)
			}
		default:
			return fmt.Errorf("unsupported value of upgrade: %s", a.Upgrade)
		}
	}

	switch actualState {
	case AptPresent:
		installed, err := a.apt.ListInstalled()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}

		toInstall := []*apt.Package{}
		for _, wanted := range name {
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
				return fmt.Errorf("failed to install package list: %w. Output: %s", err, out)
			}
		}
	case AptLatest:
		toInstall := []*apt.Package{}
		for _, p := range name {
			toInstall = append(toInstall, &apt.Package{
				Name: p,
			})
		}

		// We don't need to think about upgrading vs installing: installing an
		// already installed package will upgrade it if a new version is
		// available.
		if len(toInstall) > 0 {
			if out, err := a.apt.Install(toInstall...); err != nil {
				return fmt.Errorf("failed to install package list: %w. Output: %s", err, out)
			}
		}
	case AptAbsent:
		toRemove := []*apt.Package{}
		for _, p := range name {
			toRemove = append(toRemove, &apt.Package{
				Name: p,
			})
		}

		if len(toRemove) > 0 {
			if out, err := a.apt.Remove(toRemove...); err != nil {
				return fmt.Errorf("failed to remove package list: %w. Output: %s", err, out)
			}
		}
	default:
		return fmt.Errorf("state %s is not implemented for apt", actualState)
	}

	return nil
}
