package proto

import (
	"fmt"
	"strconv"

	"github.com/goccy/go-yaml"
)

// UnmarshalYAML handles Mode as either string ("0644", "u+rwx") or int (0644).
func (m *Mode) UnmarshalYAML(b []byte) error {
	// Try string first (most common case)
	var str string
	if err := yaml.Unmarshal(b, &str); err == nil {
		m.Value = str
		return nil
	}

	// Try int (for unquoted octal like 0644)
	var num int64
	if err := yaml.Unmarshal(b, &num); err == nil {
		m.Value = strconv.FormatInt(num, 8)
		return nil
	}

	// Try uint64 (just in case)
	var unum uint64
	if err := yaml.Unmarshal(b, &unum); err == nil {
		m.Value = strconv.FormatUint(unum, 8)
		return nil
	}

	return fmt.Errorf("failed to unmarshal mode: %s", b)
}

// MarshalYAML outputs Mode as a string.
func (m *Mode) MarshalYAML() ([]byte, error) {
	if m == nil || m.Value == "" {
		return yaml.Marshal(nil)
	}
	return yaml.Marshal(m.Value)
}
