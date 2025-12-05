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

// UnmarshalYAML is a custom unmarshaler that handles the name field (pkg and
// package) and update-cache aliases.
func (a *Apt) UnmarshalYAML(b []byte) error {
	type plain Apt
	if err := yaml.Unmarshal(b, (*plain)(a)); err != nil {
		return err
	}

	type apt struct {
		Pkg         PackageList
		Package     PackageList
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
