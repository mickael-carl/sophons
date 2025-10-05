package exec

import (
	"bytes"
	"context"
	"errors"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"

	"github.com/mickael-carl/sophons/pkg/inventory"
)

type Playbook []Play

type Play struct {
	Hosts string `yaml:"hosts"`
	Tasks []Task
}

type Task interface {
	Validate() error
	Apply() error
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
			switch taskType {
			case "file", "ansible.builtin.file":
				var f File
				var buf bytes.Buffer
				if err := yaml.NewDecoder(&buf).DecodeFromNodeContext(ctx, node, &f); err != nil {
					return err
				}
				tasksOut = append(tasksOut, &f)
			case "command", "ansible.builtin.command":
				var c Command
				var buf bytes.Buffer
				if err := yaml.NewDecoder(&buf).DecodeFromNodeContext(ctx, node, &c); err != nil {
					return err
				}
				tasksOut = append(tasksOut, &c)
			default:
				return errors.New("unsupported task type")
			}
		}
	}

	*t = tasksOut
	return nil
}

// TODO: move to support also Jinja in non-string types.
type jinjaString string

func jinjaStringUnmarshalYAML(ctx context.Context, j *jinjaString, b []byte) error {
	var vars inventory.Variables
	vars, ok := ctx.Value("vars").(inventory.Variables)
	if !ok {
		vars = inventory.Variables{}
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
