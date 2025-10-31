package exec

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/arduino/go-apt-client"
)

type AptState State

const (
	AptAbsent   AptState = AptState(Absent)
	AptBuildDep AptState = "build-dep"
	AptLatest   AptState = "latest"
	AptPresent  AptState = "present"
	AptFixed    AptState = "fixed"
)

//	@meta{
//	  "deviations": [
//	    "`state` only supports `present`, `latest` and `absent`",
//	    "`upgrade` only supports `dist` and `yes`",
//	    "aliases for `name` are not supported",
//	    "version strings in package names are not supported",
//	    "`name` needs to be a list (one element is ok), a single string is not supported"
//	  ]
//	}
type Apt struct {
	AllowChangeHeldPackages  bool  `yaml:"allow_change_held_packages"`
	AllowDowngrade           bool  `yaml:"allow_downgrade"`
	AllowUnauthenticated     bool  `yaml:"allow_unauthenticated"`
	AutoInstallModuleDeps    *bool `yaml:"auto_install_module_deps"`
	Autoclean                bool
	Autoremove               bool
	CacheValidTime           int64 `yaml:"cache_valid_time" sophons:"implemented"`
	Clean                    bool
	Deb                      jinjaString
	DefaultRelease           jinjaString `yaml:"default_release"`
	DpkgOptions              jinjaString `yaml:"dpkg_options"`
	FailOnAutoremove         bool        `yaml:"fail_on_autoremove"`
	Force                    bool
	ForceAptGet              bool          `yaml:"force_apt_get"`
	InstallRecommends        bool          `yaml:"install_recommends"`
	LockTimeout              int64         `yaml:"lock_timeout"`
	Name                     []jinjaString `sophons:"implemented"`
	OnlyUpgrade              bool          `yaml:"only_upgrade"`
	PolicyRCD                int64         `yaml:"policy_rc_d"`
	Purge                    bool
	State                    jinjaString `sophons:"implemented"`
	UpdateCache              bool        `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64      `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64      `yaml:"update_cache_retry_max_delay"`
	Upgrade                  jinjaString `sophons:"implemented"`
}

func init() {
	RegisterTaskType("apt", func() TaskContent { return &Apt{} })
	RegisterTaskType("ansible.builtin.apt", func() TaskContent { return &Apt{} })
}

func (a *Apt) Validate() error {
	// TODO
	return nil
}

func (a *Apt) Apply(_ string, _ bool) error {
	actualState := AptState(a.State)
	if a.State == "" {
		actualState = AptPresent
	}

	cacheInfo, err := os.Stat("/var/lib/apt/lists")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check cache last refresh time: %w", err)
	}

	if err == nil {
		if a.UpdateCache && time.Since(cacheInfo.ModTime()).Seconds() > float64(a.CacheValidTime) {
			if _, err := apt.CheckForUpdates(); err != nil {
				return fmt.Errorf("failed to update apt cache: %w", err)
			}
		}
	}

	if a.Upgrade != "" {
		switch a.Upgrade {
		case "yes":
			if _, err := apt.UpgradeAll(); err != nil {
				return fmt.Errorf("failed to upgrade: %w", err)
			}
		case "dist":
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
