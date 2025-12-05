package proto

import (
	"github.com/goccy/go-yaml"
)

// UnmarshalYAML is a custom unmarshaler that handles the path field aliases
// (dest and name).
func (f *File) UnmarshalYAML(b []byte) error {
	type plain File
	if err := yaml.Unmarshal(b, (*plain)(f)); err != nil {
		return err
	}

	type file struct {
		Dest string
		Name string
	}

	var aux file
	if err := yaml.Unmarshal(b, &aux); err != nil {
		return err
	}

	if f.Path == "" {
		if aux.Dest != "" {
			f.Path = aux.Dest
		} else if aux.Name != "" {
			f.Path = aux.Name
		}
	}

	return nil
}
