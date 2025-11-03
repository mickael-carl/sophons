package exec

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/arduino/go-apt-client"
)

type AptRepositoryState State

const (
	AptRepositoryAbsent  AptRepositoryState = AptRepositoryState(Absent)
	AptRepositoryPresent AptRepositoryState = "present"
)

//	@meta{
//	  "deviations": []
//	}
type AptRepository struct {
	Codename                 jinjaString
	Filename                 jinjaString
	InstallPythonApt         bool `yaml:"install_python_apt"`
	Mode                     jinjaString
	Repo                     jinjaString        `sophons:"implemented"`
	State                    AptRepositoryState `sophons:"implemented"`
	UpdateCache              *bool              `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64             `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64             `yaml:"update_cache_retry_max_delay"`
	ValidateCerts            *bool              `yaml:"validate_certs"`
}

func init() {
	RegisterTaskType("apt_repository", func() TaskContent { return &AptRepository{} })
	RegisterTaskType("ansible.builtin.apt_repository", func() TaskContent { return &AptRepository{} })
}

func uriToFilename(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("failed to parse repository URL: %w", err)
	}

	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	// Drop any port and then slugify.
	return re.ReplaceAllString(u.Hostname()+u.Path, "_") + ".list", nil
}

func (ar *AptRepository) Validate() error {
	if ar.Repo == "" {
		return errors.New("repo is required")
	}
	if ar.State != AptRepositoryAbsent && ar.State != AptRepositoryPresent && ar.State != "" {
		return fmt.Errorf("unsupported state: %s", ar.State)
	}
	return nil
}

func (ar *AptRepository) Apply(_ string, _ bool) error {
	repos, err := apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		return fmt.Errorf("failed to parse existing repositories: %w", err)
	}

	repo := apt.ParseAPTConfigLine(string(ar.Repo))
	if repo == nil {
		return errors.New("failed to parse repo line")
	}

	if ar.State == AptRepositoryAbsent {
		toRemove := repos.Find(repo)
		if toRemove != nil {
			if err := apt.RemoveRepository(toRemove, "/etc/apt"); err != nil {
				return fmt.Errorf("failed to remove repository: %w", err)
			}
		}
	} else {
		existing := repos.Find(repo)
		if existing == nil {
			filename, err := uriToFilename(repo.URI)
			if err != nil {
				return fmt.Errorf("failed to infer filename from repo: %w", err)
			}
			if err := apt.AddRepository(repo, "/etc/apt", filename); err != nil {
				return fmt.Errorf("failed to add repository: %w", err)
			}
		}
	}

	if ar.UpdateCache == nil || ar.UpdateCache != nil && *ar.UpdateCache {
		if _, err := apt.CheckForUpdates(); err != nil {
			return fmt.Errorf("failed to update apt cache: %w", err)
		}
	}

	return nil
}
