package role

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/proto"
)

// processTasks processes tasks for a given directory. It does hopefully in a
// way that imitates Ansible's quirks. Indeed, just like for variables, Ansible
// looks for a `tasks/main.yml` first, then if not found tries
// `tasks/main.yaml`, then `tasks/main` but stops there. Contrary to variables,
// it does not go into a subdirectory called `tasks/main`.
func processTasks(fsys fs.FS, root string) ([]*proto.Task, error) {
	// NOTE: the code looks very similar to `processVars` and it could be tempting
	// to try and refactor both those functions to not duplicate it so much. Given
	// how quirky Ansible is though and the fact that variables and tasks are
	// wildly different concepts, I think it best to leave the duplication.
	tasks := []*proto.Task{}

	data, err := fs.ReadFile(fsys, filepath.Join(root, "main.yml"))
	// Careful here: we look for no error first on purpose as it makes the code
	// much more readable.
	if err == nil {
		if err = yaml.Unmarshal(data, &tasks); err != nil {
			return []*proto.Task{}, fmt.Errorf("failed to unmarshal tasks: %w", err)
		}
		return tasks, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return []*proto.Task{}, fmt.Errorf("failed to read main.yml: %w", err)
	}

	data, err = fs.ReadFile(fsys, filepath.Join(root, "main.yaml"))
	if err == nil {
		if err = yaml.Unmarshal(data, &tasks); err != nil {
			return []*proto.Task{}, fmt.Errorf("failed to unmarshal tasks: %w", err)
		}
		return tasks, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return []*proto.Task{}, fmt.Errorf("failed to read main.yaml: %w", err)
	}

	f, err := fs.Stat(fsys, filepath.Join(root, "main"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*proto.Task{}, nil
		}
		return []*proto.Task{}, fmt.Errorf("failed to stat main: %w", err)
	}

	// In case `main` is a directory, we don't actually load anything directly.
	if f.IsDir() {
		return []*proto.Task{}, nil
	}

	data, err = fs.ReadFile(fsys, filepath.Join(root, "main"))
	if err != nil {
		return []*proto.Task{}, fmt.Errorf("failed to read content of main/ directory: %w", err)
	}

	if err = yaml.Unmarshal(data, &tasks); err != nil {
		return []*proto.Task{}, fmt.Errorf("failed to unmarshal tasks: %w", err)
	}
	return tasks, nil
}
