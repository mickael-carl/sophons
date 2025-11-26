package proto

import (
	"fmt"

	"github.com/goccy/go-yaml"
)

// UnmarshalYAML is a customer unmarshaler that handles the package list being
// either a scalar or list.
func (p *PackageList) UnmarshalYAML(b []byte) error {
	var packageName string
	if err := yaml.Unmarshal(b, &packageName); err == nil {
		p.Items = []string{packageName}
		return nil
	}

	var packages []string
	if err := yaml.Unmarshal(b, &packages); err == nil {
		p.Items = packages
		return nil
	}

	return fmt.Errorf("failed to unmarshal package name: %s", b)
}

func (p *PackageList) MarshalYAML() ([]byte, error) {
	if p == nil || len(p.Items) == 0 {
		return yaml.Marshal(nil)
	}
	if len(p.Items) == 1 {
		return yaml.Marshal(p.Items[0])
	}
	return yaml.Marshal(p.Items)
}
