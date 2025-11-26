package exec

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/goccy/go-yaml"
)

const (
	AptRepositoryAbsent  string = Absent
	AptRepositoryPresent string = "present"
)

//	@meta{
//	  "deviations": []
//	}
type AptRepository struct {
	Codename                 string
	Filename                 string
	InstallPythonApt         bool `yaml:"install_python_apt"`
	Mode                     string
	Repo                     string `sophons:"implemented"`
	State                    string `sophons:"implemented"`
	UpdateCache              *bool  `yaml:"update_cache" sophons:"implemented"`
	UpdateCacheRetries       uint64 `yaml:"update_cache_retries"`
	UpdateCacheRetryMaxDelay uint64 `yaml:"update_cache_retry_max_delay"`
	ValidateCerts            *bool  `yaml:"validate_certs"`

	apt aptClient
}

type AptRepositoryResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("apt_repository", func() TaskContent { return &AptRepository{} })
	RegisterTaskType("ansible.builtin.apt_repository", func() TaskContent { return &AptRepository{} })
}

func (a *AptRepository) UnmarshalYAML(b []byte) error {
	type plain AptRepository
	if err := yaml.Unmarshal(b, (*plain)(a)); err != nil {
		return err
	}

	type aptRepository struct {
		UpdateCache *bool `yaml:"update-cache"`
	}

	var aux aptRepository
	if err := yaml.Unmarshal(b, &aux); err != nil {
		return err
	}

	if a.UpdateCache == nil {
		a.UpdateCache = aux.UpdateCache
	}

	return nil
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

func (ar *AptRepository) Apply(ctx context.Context, _ string, _ bool) (Result, error) {
	if clientFromCtx, ok := ctx.Value(aptClientContextKey).(aptClient); ok {
		ar.apt = clientFromCtx
	} else {
		ar.apt = &realAptClient{}
	}

	result := AptRepositoryResult{}
	repos, err := ar.apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to parse existing repositories: %w", err)
	}

	repo := ar.apt.ParseAPTConfigLine(ar.Repo)
	if repo == nil {
		result.TaskFailed()
		return &result, errors.New("failed to parse repo line")
	}

	if ar.State == AptRepositoryAbsent {
		toRemove := repos.Find(repo)
		if toRemove != nil {
			result.TaskChanged()
			if err := ar.apt.RemoveRepository(toRemove, "/etc/apt"); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to remove repository: %w", err)
			}
		}
	} else {
		existing := repos.Find(repo)
		if existing == nil {
			filename, err := uriToFilename(repo.URI)
			if err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to infer filename from repo: %w", err)
			}
			result.TaskChanged()
			if err := ar.apt.AddRepository(repo, "/etc/apt", filename); err != nil {
				result.TaskFailed()
				return &result, fmt.Errorf("failed to add repository: %w", err)
			}
		}
	}

	if ar.UpdateCache == nil || ar.UpdateCache != nil && *ar.UpdateCache {
		result.TaskChanged()
		if _, err := ar.apt.CheckForUpdates(); err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to update apt cache: %w", err)
		}
	}

	return &result, nil
}
