package role

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/inventory"
)

// processVars processes variables/defaults for a particular directory. It does
// so in the "correct" load order, as defined by Ansible: first look for a
// main.yml, if not found look for a main.yaml, and if again not found look for
// a `main/` and load everything underneath.
func processVars(fsys fs.FS, root string) (inventory.Variables, error) {
	vars := inventory.Variables{}

	data, err := fs.ReadFile(fsys, filepath.Join(root, "main.yml"))
	// Careful here: we look for no error first on purpose as it makes the code
	// much more readable.
	if err == nil {
		if err = yaml.Unmarshal(data, &vars); err != nil {
			return inventory.Variables{}, err
		}
		return vars, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return inventory.Variables{}, err
	}

	data, err = fs.ReadFile(fsys, filepath.Join(root, "main.yaml"))
	if err == nil {
		if err = yaml.Unmarshal(data, &vars); err != nil {
			return inventory.Variables{}, err
		}
		return vars, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return inventory.Variables{}, err
	}

	f, err := fs.Stat(fsys, filepath.Join(root, "main"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return inventory.Variables{}, nil
		}
		return inventory.Variables{}, err
	}

	// `main` can just also be a file, which Ansible also considers valid and
	// loads automagically.
	if !f.IsDir() {
		data, err := fs.ReadFile(fsys, filepath.Join(root, "main"))
		if err != nil {
			return inventory.Variables{}, err
		}

		if err = yaml.Unmarshal(data, &vars); err != nil {
			return inventory.Variables{}, err
		}

		return vars, nil
	}

	// Final valid case: main is a directory. Then load everything underneath.
	if err := fs.WalkDir(fsys, filepath.Join(root, "main"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		var fileVars inventory.Variables
		if err := yaml.Unmarshal(f, &fileVars); err != nil {
			return err
		}

		// NOTE: Ansible's documentation does not say which level takes
		// precedence: more nested over less, or the other way around? We'll
		// just go with more nested wins.
		maps.Copy(vars, fileVars)

		return nil
	}); err != nil {
		return inventory.Variables{}, err
	}

	return vars, nil
}
