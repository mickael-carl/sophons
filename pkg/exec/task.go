package exec

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goccy/go-yaml"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	protopackage "github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/registry"
	"github.com/mickael-carl/sophons/pkg/variables"
)

type Task struct {
	Name     string
	When     string
	Loop     any
	Content  TaskContent
	Register string
}

func (t Task) Validate() error {
	return t.Content.Validate()
}

func (t Task) Apply(ctx context.Context, parentPath string, isRole bool) (Result, error) {
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
	Apply(context.Context, string, bool) (Result, error)
}

// FromProto converts a proto.Task to an exec.Task for execution.
func FromProto(pt *protopackage.Task) (*Task, error) {
	t := &Task{
		Name:     pt.Name,
		When:     pt.When,
		Register: pt.Register,
	}

	if pt.Loop != nil {
		t.Loop = pt.Loop.AsInterface()
	}

	if pt.Content == nil {
		return t, nil
	}

	reg, ok := registry.TypeRegistry[reflect.TypeOf(pt.Content)]
	if !ok {
		return nil, fmt.Errorf("unknown proto content type %T: not registered", pt.Content)
	}
	if reg.ExecAdapter == nil {
		return nil, fmt.Errorf("no exec adapter registered for type %T", pt.Content)
	}

	execContent := reg.ExecAdapter(pt.Content)
	t.Content, ok = execContent.(TaskContent)
	if !ok {
		return nil, fmt.Errorf("exec adapter returned non-TaskContent type %T", execContent)
	}

	return t, nil
}

// deepCopyContent takes any task's content and returns a copy of it alongside
// any error while doing so. We need this to support loops, since when `loop`
// is set we need to evaluate Jinja templates with a new variable, `item`, for
// every iteration on a copy of the task.
func deepCopyContent(content TaskContent) (TaskContent, error) {
	contentType := reflect.TypeOf(content).Elem()
	newContentPtr := reflect.New(contentType)
	newContentValue := newContentPtr.Elem()
	oldContentValue := reflect.ValueOf(content).Elem()

	// Find and clone any embedded proto fields
	for i := 0; i < newContentValue.NumField(); i++ {
		oldField := oldContentValue.Field(i)

		// Skip non-pointer fields or fields that can't be interfaced
		if oldField.Kind() != reflect.Ptr || !oldField.CanInterface() {
			continue
		}

		// Check if this field is a proto.Message and clone it
		if protoMsg, ok := oldField.Interface().(proto.Message); ok {
			cloned := proto.Clone(protoMsg)
			newContentValue.Field(i).Set(reflect.ValueOf(cloned))
		}
	}

	return newContentPtr.Interface().(TaskContent), nil
}

func processAndRunTask(ctx context.Context, logger *zap.Logger, task Task, parentPath string, isRole bool) (Result, error) {
	if err := util.ProcessJinjaTemplates(ctx, &task); err != nil {
		return &CommonResult{}, fmt.Errorf("failed to process Jinja templating: %w", err)
	}

	whenResult, err := util.JinjaProcessWhen(ctx, task.When)
	if err != nil {
		return &CommonResult{}, fmt.Errorf("failed to process when condition: %w", err)
	}

	if !whenResult {
		logger.Debug("skipping task due to when condition", zap.String("task", task.Name))
		return &CommonResult{}, nil
	}

	logger.Debug("executing task", zap.Any("task", task))
	if err := task.Validate(); err != nil {
		return &CommonResult{}, fmt.Errorf("validation failed: %w", err)
	}
	return task.Apply(ctx, parentPath, isRole)
}

// ExecuteTask executes a single task, processing any loop items and rendering
// Jinja templates.
func ExecuteTask(ctx context.Context, logger *zap.Logger, task Task, parentPath string, isRole bool) error {
	if task.Loop == nil {
		result, err := processAndRunTask(ctx, logger, task, parentPath, isRole)
		if err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}

		if task.Register != "" {
			vars, ok := variables.FromContext(ctx)
			if !ok {
				vars = variables.Variables{}
			}
			resultMap, err := resultToMap(result)
			if err != nil {
				return fmt.Errorf("failed to convert result to map: %w", err)
			}
			vars[task.Register] = resultMap
		}
		return nil
	}

	tempLoopHolder := struct{ Loop any }{Loop: task.Loop}
	if err := util.ProcessJinjaTemplates(ctx, &tempLoopHolder); err != nil {
		return fmt.Errorf("failed to process Jinja templating for loop: %w", err)
	}
	task.Loop = tempLoopHolder.Loop

	loopValues, ok := task.Loop.([]any)
	if !ok {
		// It might be a slice of strings if the Jinja processing resulted in
		// that.
		loopStrValues, okStr := task.Loop.([]string)
		if !okStr {
			return fmt.Errorf("loop variable is not a list: %T", task.Loop)
		}
		loopValues = make([]any, len(loopStrValues))
		for i, v := range loopStrValues {
			loopValues[i] = v
		}
	}

	loopResults := LoopResult{
		Results: []Result{},
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

		result, err := processAndRunTask(loopCtx, logger, iterTask, parentPath, isRole)
		if err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}

		if result.IsChanged() {
			loopResults.TaskChanged()
		}
		if result.IsSkipped() {
			loopResults.TaskSkipped()
		}
		if result.IsFailed() {
			loopResults.TaskFailed()
		}
		loopResults.Results = append(loopResults.Results, result)
	}

	if task.Register != "" {
		vars, ok := variables.FromContext(ctx)
		if !ok {
			vars = variables.Variables{}
		}
		resultMap, err := resultToMap(&loopResults)
		if err != nil {
			return fmt.Errorf("failed to convert loop result to map: %w", err)
		}
		vars[task.Register] = resultMap
	}

	return nil
}

type CommonResult struct {
	Changed bool `yaml:"changed" json:"changed"`
	// TODO: add diff.
	Failed      bool     `yaml:"failed" json:"failed"`
	Msg         string   `yaml:"msg" json:"msg"`
	RC          int      `yaml:"rc"`
	Skipped     bool     `yaml:"skipped" json:"skipped"`
	Stderr      string   `yaml:"stderr" json:"stderr"`
	StderrLines []string `yaml:"stderr_lines" json:"stderr_lines"`
	Stdout      string   `yaml:"stdout" json:"stdout"`
	StdoutLines []string `yaml:"stdout_lines" json:"stdout_lines"`
}

func (c *CommonResult) TaskChanged() {
	c.Changed = true
}

func (c *CommonResult) TaskSkipped() {
	c.Skipped = true
}

func (c *CommonResult) TaskFailed() {
	c.Failed = true
}

func (c *CommonResult) IsChanged() bool {
	return c.Changed
}

func (c *CommonResult) IsSkipped() bool {
	return c.Skipped
}

func (c *CommonResult) IsFailed() bool {
	return c.Failed
}

type Result interface {
	TaskChanged()
	TaskSkipped()
	TaskFailed()
	IsChanged() bool
	IsSkipped() bool
	IsFailed() bool
}

type LoopResult struct {
	CommonResult `yaml:",inline"`
	Results      []Result `yaml:"results" json:"results"`
}

// resultToMap converts a Result interface to a map[string]any by
// marshalling it to YAML and then unmarshalling it. This is to make sure that
// the registered variables have snake_case keys.
func resultToMap(result Result) (map[string]any, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return nil, err
	}
	var resultMap map[string]any
	err = yaml.Unmarshal(data, &resultMap)
	if err != nil {
		return nil, err
	}
	return resultMap, nil
}
