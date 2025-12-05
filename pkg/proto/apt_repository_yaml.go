package proto

import (
	"github.com/goccy/go-yaml"
)

// UnmarshalYAML is a custom unmarshaler that handles the update-cache alias.
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
