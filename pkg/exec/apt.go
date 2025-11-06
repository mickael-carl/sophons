package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arduino/go-apt-client"
	"github.com/goccy/go-yaml"
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

// TODO: this is fixable: if we implement a custom unmarshaller for Apt, we can
// implement aliases and name being either a list of a single string.
//
//	@meta{
//	  "deviations": [
//	    "`state` only supports `present`, `latest` and `absent`",
//	    "version strings in package names are not supported",
//	    "`name` needs to be a list (one element is ok), a single string is not supported"
//	  ]
//	}
type Apt struct {
	AllowChangeHeldPackages bool    `yaml:"allow_change_held_packages"`
	AllowDowngrade          bool    `yaml:"allow_downgrade"`
	AllowUnauthenticated    bool    `yaml:"allow_unauthenticated"`
	AutoInstallModuleDeps   *bool   `yaml:"auto_install_module_deps"`
	AutoClean               bool    `yaml:"autoclean"`
	AutoRemove              bool    `yaml:"autoremove"`
	CacheValidTime          *uint64 `yaml:"cache_valid_time" sophons:"implemented"`
	Clean                   bool    `sophons:"implemented"`
	Deb                     jinjaString
	DefaultRelease          jinjaString `yaml:"default_release"`
	DpkgOptions             jinjaString `yaml:"dpkg_options"`
	FailOnAutoremove        bool        `yaml:"fail_on_autoremove"`
	Force                   bool
	ForceAptGet             bool          `yaml:"force_apt_get"`
	InstallRecommends       bool          `yaml:"install_recommends"`
	LockTimeout             int64         `yaml:"lock_timeout"`
	Name                    []jinjaString `sophons:"implemented"`

	OnlyUpgrade              bool  `yaml:"only_upgrade"`
	PolicyRCD                int64 `yaml:"policy_rc_d"`
	Purge                    bool
	State                    jinjaString `sophons:"implemented"`
	UpdateCache              *bool       `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64      `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64      `yaml:"update_cache_retry_max_delay"`
	Upgrade                  jinjaString `sophons:"implemented"`
}

func init() {
	RegisterTaskType("apt", func() TaskContent { return &Apt{} })
	RegisterTaskType("ansible.builtin.apt", func() TaskContent { return &Apt{} })
}

func (a *Apt) UnmarshalYAML(ctx context.Context, b []byte) error {
	type plain Apt
	if err := yaml.UnmarshalContext(ctx, b, (*plain)(a)); err != nil {
		return err
	}

	type apt struct {
		Pkg         []jinjaString
		Package     []jinjaString
		UpdateCache bool `yaml:"update-cache"`
	}

	var aux apt
	if err := yaml.UnmarshalContext(ctx, b, &aux); err != nil {
		return err
	}

	if len(a.Name) == 0 {
		if len(aux.Package) != 0 {
			a.Name = aux.Package
		} else if len(aux.Pkg) != 0 {
			a.Name = aux.Pkg
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

	if _, ok := supportedUpgrade[string(a.Upgrade)]; !ok {
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
		_, cacheErr := apt.CheckForUpdates()
		return cacheErr
	}

	if err == nil {
		if a.UpdateCache != nil && *a.UpdateCache || a.CacheValidTime != nil && time.Since(cacheInfo.ModTime()).Seconds() > float64(*a.CacheValidTime) {
			_, cacheErr := apt.CheckForUpdates()
			return cacheErr
		}
	}

	return nil
}

func (a *Apt) Apply(_ string, _ bool) error {
	if a.Clean {
		if _, err := apt.Clean(); err != nil {
			return fmt.Errorf("failed to clean apt cache: %w", err)
		}

		// This is similar to Ansible's implementation. See
		// https://github.com/ansible/ansible/issues/82611 and
		// https://github.com/ansible/ansible/pull/82800 for some context.
		if len(a.Name) == 0 && (a.Upgrade == "no" || a.Upgrade == "") && a.Deb == "" {
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
			if _, err := apt.UpgradeAll(); err != nil {
				return fmt.Errorf("failed to upgrade: %w", err)
			}
		case "dist", "full":
			if _, err := apt.DistUpgrade(); err != nil {
				return fmt.Errorf("failed to dist-upgrade: %w", err)
			}
		default:
			return fmt.Errorf("unsupported value of upgrade: %s", a.Upgrade)
		}
	}

	switch actualState {
	case AptPresent:
		installed, err := apt.ListInstalled()
		if err != nil {
			return fmt.Errorf("failed to list installed packages: %w", err)
		}

		toInstall := []*apt.Package{}
		for _, wanted := range a.Name {
			found := false
			for _, p := range installed {
				if p.Name == string(wanted) {
					found = true
					break
				}
			}
			if !found {
				toInstall = append(toInstall, &apt.Package{Name: string(wanted)})
			}
		}

		if _, err := apt.Install(toInstall...); err != nil {
			return fmt.Errorf("failed to install package list: %w", err)
		}
	case AptLatest:
		toInstall := []*apt.Package{}
		for _, p := range a.Name {
			toInstall = append(toInstall, &apt.Package{
				Name: string(p),
			})
		}

		// We don't need to think about upgrading vs installing: installing an
		// already installed package will upgrade it if a new version is
		// available.
		if _, err := apt.Install(toInstall...); err != nil {
			return fmt.Errorf("failed to install package list: %w", err)
		}
	case AptAbsent:
		toRemove := []*apt.Package{}
		for _, p := range a.Name {
			toRemove = append(toRemove, &apt.Package{
				Name: string(p),
			})
		}

		if _, err := apt.Remove(toRemove...); err != nil {
			return fmt.Errorf("failed to remove package list: %w", err)
		}
	default:
		return fmt.Errorf("state %s is not implemented for apt", actualState)
	}

	return nil
}
