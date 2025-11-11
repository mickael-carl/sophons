package variables

import (
	"context"
	"maps"
	"os"

	"github.com/goccy/go-yaml"
)

type Variables map[string]any

type key int

var varsKey key

func FromContext(ctx context.Context) (Variables, bool) {
	v, ok := ctx.Value(varsKey).(Variables)
	return v, ok
}

func NewContext(ctx context.Context, vars Variables) context.Context {
	return context.WithValue(ctx, varsKey, vars)
}

func LoadFromFile(path string) (Variables, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Variables{}, err
	}
	var vars Variables
	if err := yaml.Unmarshal(data, &vars); err != nil {
		return Variables{}, err
	}
	return vars, nil
}

func (v Variables) Merge(other Variables) {
	maps.Copy(v, other)
}
