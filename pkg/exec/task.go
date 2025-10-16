package exec

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"

	"github.com/mickael-carl/sophons/pkg/variables"
)

type CommonTask struct {
	Name jinjaString
}

type Task interface {
	Validate() error
	Apply() error
}

var taskRegistry = map[string]func() Task{}

func RegisterTaskType(name string, factory func() Task) {
	taskRegistry[name] = factory
}

func init() {
	yaml.RegisterCustomUnmarshalerContext[[]Task](tasksUnmarshalYAML)
	yaml.RegisterCustomUnmarshalerContext[jinjaString](jinjaStringUnmarshalYAML)
}

func tasksUnmarshalYAML(ctx context.Context, t *[]Task, b []byte) error {
	var raw []map[string]ast.Node
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}

	var tasksOut []Task
	for _, task := range raw {
		for taskType, node := range task {
			factory, ok := taskRegistry[taskType]
			if !ok {
				return fmt.Errorf("unsupported task type: %s", taskType)
			}

			t := factory()

			var buf bytes.Buffer
			if err := yaml.NewDecoder(&buf).DecodeFromNodeContext(ctx, node, t); err != nil {
				return err
			}
			tasksOut = append(tasksOut, t)
		}
	}

	*t = tasksOut
	return nil
}

// TODO: move to support also Jinja in non-string types.
type jinjaString string

func jinjaStringUnmarshalYAML(ctx context.Context, j *jinjaString, b []byte) error {
	var vars variables.Variables
	vars, ok := variables.FromContext(ctx)
	if !ok {
		vars = variables.Variables{}
	}

	varsCtx := exec.NewContext(vars)

	var raw string
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}

	template, err := gonja.FromString(raw)
	if err != nil {
		return err
	}

	expanded, err := template.ExecuteToString(varsCtx)
	if err != nil {
		return err
	}

	*j = jinjaString(expanded)

	return nil
}
