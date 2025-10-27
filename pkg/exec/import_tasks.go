package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

//	@meta{
//	  "deviations": []
//	}
type ImportTasks struct {
	File string `sophons:"implemented"`
}

func init() {
	RegisterTaskType("import_tasks", func() TaskContent { return &ImportTasks{} })
	RegisterTaskType("ansible.builtin.import_tasks", func() TaskContent { return &ImportTasks{} })
}

func (it *ImportTasks) Validate() error {
	if it.File == "" {
		return errors.New("`file` is required")
	}

	return nil
}

func (it *ImportTasks) Apply(parentPath string, isRole bool) error {
	// This is Ansible madness: import_tasks' File is relative to where the
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
		return fmt.Errorf("failed to read tasks from %s: %w", taskPath, err)
	}

	var tasks []Task
	if err := yaml.Unmarshal(taskData, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks from %s: %w", taskPath, err)
	}

	for _, task := range tasks {
		if err := task.Apply(parentPath, isRole); err != nil {
			return fmt.Errorf("failed to apply tasks from %s: %w", taskPath, err)
		}
	}

	return nil
}
