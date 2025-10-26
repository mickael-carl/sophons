package exec

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"

	"github.com/mickael-carl/sophons/pkg/variables"
)

type Task struct {
	Name    string
	Content TaskContent
}

func (t Task) Validate() error {
	return t.Content.Validate()
}

func (t Task) Apply(parentPath string, isRole bool) error {
	return t.Content.Apply(parentPath, isRole)
}

func (t Task) String() string {
	if t.Content == nil {
		return fmt.Sprintf("Task{Name:%q, Content:nil}", t.Name)
	}

	v := reflect.ValueOf(t.Content)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	return fmt.Sprintf("Task{Name:%q, Content:%#v}", t.Name, v.Interface())
}

type TaskContent interface {
	Validate() error
	Apply(string, bool) error
}

var taskRegistry = map[string]func() TaskContent{}

func RegisterTaskType(name string, factory func() TaskContent) {
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
		var name string
		if n, ok := task["name"]; ok {
			var buf bytes.Buffer
			if err := yaml.NewDecoder(&buf).DecodeFromNodeContext(ctx, n, &name); err != nil {
				return err
			}
		}
		t := Task{
			Name: name,
		}
		for taskType, node := range task {
			factory, ok := taskRegistry[taskType]
			if !ok {
				continue
			}

			f := factory()

			var buf bytes.Buffer
			if err := yaml.NewDecoder(&buf).DecodeFromNodeContext(ctx, node, f); err != nil {
				return err
			}
			t.Content = f
		}
		tasksOut = append(tasksOut, t)
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
