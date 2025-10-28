package exec

import (
	"fmt"

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
//	    "`state` only supports `present` and `absent`",
//	    "aliases for `name` are not supported"
//	  ]
//	}
type Apt struct {
	AllowChangeHeldPackages  bool  `yaml:"allow_change_held_packages"`
	AllowDowngrade           bool  `yaml:"allow_downgrade"`
	AllowUnauthenticated     bool  `yaml:"allow_unauthenticated"`
	AutoInstallModuleDeps    *bool `yaml:"auto_install_module_deps"`
	Autoclean                bool
	Autoremove               bool
	CacheValidTime           int64 `yaml:"cache_valid_time"`
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
	State                    jinjaString
	UpdateCache              bool   `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64 `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64 `yaml:"update_cache_retry_max_delay"`
	Upgrade                  jinjaString
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

	if a.UpdateCache {
		if _, err := apt.CheckForUpdates(); err != nil {
			return fmt.Errorf("failed to update apt cache: %w", err)
		}
	}

	packages := []*apt.Package{}
	for _, n := range a.Name {
		packages = append(packages, &apt.Package{
			Name: string(n),
		})
	}

	switch actualState {
	case AptPresent:
		if _, err := apt.Install(packages...); err != nil {
			return fmt.Errorf("failed to install package list: %w", err)
		}
	case AptAbsent:
		if _, err := apt.Remove(packages...); err != nil {
			return fmt.Errorf("failed to remove package list: %w", err)
		}
	default:
		return fmt.Errorf("state %s is not implemented for apt", actualState)
	}

	return nil
}
