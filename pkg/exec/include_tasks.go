package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	"github.com/mickael-carl/sophons/pkg/proto"
)

//	@meta{
//	  "deviations": []
//	}
type IncludeTasks struct {
	proto.IncludeTasks `yaml:",inline"`
}

type IncludeTasksResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("include_tasks", func() TaskContent { return &IncludeTasks{} })
	RegisterTaskType("ansible.builtin.include_tasks", func() TaskContent { return &IncludeTasks{} })
}

func (it *IncludeTasks) Validate() error {
	if it.File == "" {
		return errors.New("`file` is required")
	}

	return nil
}

func (it *IncludeTasks) Apply(ctx context.Context, parentPath string, isRole bool) (Result, error) {
	// This is Ansible madness: include_tasks' File is relative to where the
	// task is defined. If the task is within a role, then it can be found in
	// the same directory as other tasks (i.e. in `tasks/`); but if the task is
	// within a play, then it's found in the play's directory.
	var taskPath string
	if isRole {
		taskPath = filepath.Join(parentPath, "tasks", it.File)
	} else {
		taskPath = filepath.Join(parentPath, it.File)
	}

	taskData, err := os.ReadFile(taskPath)
	if err != nil {
		return &IncludeTasksResult{}, fmt.Errorf("failed to read tasks from %s: %w", taskPath, err)
	}

	var tasks []Task
	if err := yaml.Unmarshal(taskData, &tasks); err != nil {
		return &IncludeTasksResult{}, fmt.Errorf("failed to parse tasks from %s: %w", taskPath, err)
	}

	for _, task := range tasks {
		if err := util.ProcessJinjaTemplates(ctx, &task); err != nil {
			return &IncludeTasksResult{}, fmt.Errorf("failed to render Jinja templating from %s: %w", taskPath, err)
		}

		if err := task.Validate(); err != nil {
			return &IncludeTasksResult{}, fmt.Errorf("failed to validate task from %s: %w", taskPath, err)
		}

		// TODO: handle result values. It's likely not possible to do register
		// on this.
		_, err := task.Apply(ctx, parentPath, isRole)
		if err != nil {
			return &IncludeTasksResult{}, fmt.Errorf("failed to apply task from %s: %w", taskPath, err)
		}
	}

	return &IncludeTasksResult{}, nil
}
