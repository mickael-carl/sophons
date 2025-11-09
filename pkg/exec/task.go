package exec

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"github.com/mickael-carl/sophons/pkg/variables"
)

type Task struct {
	Name    string
	Loop    interface{}
	Content TaskContent
}

func (t Task) Validate() error {
	return t.Content.Validate()
}

func (t Task) Apply(ctx context.Context, parentPath string, isRole bool) error {
	return t.Content.Apply(ctx, parentPath, isRole)
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
	Apply(context.Context, string, bool) error
}

var taskRegistry = map[string]func() TaskContent{}

func RegisterTaskType(name string, factory func() TaskContent) {
	taskRegistry[name] = factory
}

func init() {
	yaml.RegisterCustomUnmarshaler[[]Task](tasksUnmarshalYAML)
}

func tasksUnmarshalYAML(t *[]Task, b []byte) error {
	type unmarshalTask struct {
		Task       `yaml:",inline"`
		RawContent map[string]ast.Node `yaml:",inline"`
	}

	var raw []unmarshalTask
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}

	var tasksOut []Task
	for _, task := range raw {
		t := task.Task

		for taskType, node := range task.RawContent {
			factory, ok := taskRegistry[taskType]
			if !ok {
				continue
			}

			f := factory()

			if err := yaml.NodeToValue(node, f); err != nil {
				return err
			}
			t.Content = f
		}
		tasksOut = append(tasksOut, t)
	}

	*t = tasksOut
	return nil
}

// deepCopyContent takes any task's content and returns a copy of it alongside
// any error while doing so. We need this to support loops, since when `loop`
// is set we need to evaluate Jinja templates with a new variable, `item`, for
// every iteration on a copy of the task.
func deepCopyContent(content TaskContent) (TaskContent, error) {
	data, err := yaml.Marshal(content)
	if err != nil {
		return nil, err
	}

	// Create a new instance of the same type. `content` is a pointer to a
	// struct, so we get the type of the pointed-to struct.
	contentType := reflect.TypeOf(content).Elem()
	// Create a new pointer to a struct of that type.
	newContentPtr := reflect.New(contentType)

	// Get the interface value of the new pointer.
	newContent := newContentPtr.Interface().(TaskContent)

	if err := yaml.Unmarshal(data, newContent); err != nil {
		return nil, err
	}

	return newContent, nil
}

func processAndRunTask(ctx context.Context, task Task, parentPath string, isRole bool) error {
	if err := ProcessJinjaTemplates(ctx, &task); err != nil {
		return fmt.Errorf("failed to process Jinja templating: %w", err)
	}
	log.Printf("%+v", task)
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := task.Apply(ctx, parentPath, isRole); err != nil {
		return fmt.Errorf("failed to apply task: %w", err)
	}
	return nil
}

// ExecuteTask executes a single task, processing any loop items and rendering
// Jinja templates.
func ExecuteTask(ctx context.Context, task Task, parentPath string, isRole bool) error {
	if task.Loop != nil {
		tempLoopHolder := struct{ Loop interface{} }{Loop: task.Loop}
		if err := ProcessJinjaTemplates(ctx, &tempLoopHolder); err != nil {
			return fmt.Errorf("failed to process Jinja templating for loop: %w", err)
		}
		task.Loop = tempLoopHolder.Loop
	}

	if task.Loop == nil {
		if err := processAndRunTask(ctx, task, parentPath, isRole); err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}
		return nil
	}

	loopValues, ok := task.Loop.([]interface{})
	if !ok {
		// It might be a slice of strings if the Jinja processing resulted in
		// that.
		loopStrValues, okStr := task.Loop.([]string)
		if !okStr {
			return fmt.Errorf("loop variable is not a list: %T", task.Loop)
		}
		loopValues = make([]interface{}, len(loopStrValues))
		for i, v := range loopStrValues {
			loopValues[i] = v
		}
	}

	for _, item := range loopValues {
		newContent, err := deepCopyContent(task.Content)
		if err != nil {
			return fmt.Errorf("failed to copy task content: %w", err)
		}

		iterTask := Task{
			Name:    task.Name,
			Content: newContent,
		}

		currentVars, ok := variables.FromContext(ctx)
		if !ok {
			currentVars = variables.Variables{}
		}
		currentVars["item"] = item
		loopCtx := variables.NewContext(ctx, currentVars)

		if err := processAndRunTask(loopCtx, iterTask, parentPath, isRole); err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}
	}
	return nil
}
