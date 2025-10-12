package role

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
)

type Role struct {
	// TODO: add files and templates.
	Defaults  inventory.Variables
	Variables inventory.Variables
	Tasks     []exec.Task
}

func DiscoverRoles(ctx context.Context, rolesPath string) (map[string]Role, error) {
	roles := map[string]Role{}

	fsys := os.DirFS(rolesPath)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return map[string]Role{}, fmt.Errorf("failed to read entries for %s: %w", rolesPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		role, ok, err := maybeRole(ctx, fsys, entry.Name())
		if err != nil {
			return map[string]Role{}, err
		}
		if ok {
			roles[entry.Name()] = role
		}
	}

	return roles, nil
}

func maybeRole(ctx context.Context, fsys fs.FS, name string) (Role, bool, error) {
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return Role{}, false, err
	}

	isARole := false
	var role Role
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := path.Join(name, entry.Name())

		// TODO: per the Ansible docs, a directory is a role if any of the
		// below is *populated*. We only check for the directory's existence
		// instead. This may introduce a deviation in behaviour.
		switch entry.Name() {
		case "defaults":
			isARole = true
			defaults, err := processVars(fsys, entryPath)
			if err != nil {
				return Role{}, false, err
			}
			role.Defaults = defaults

		case "vars":
			isARole = true
			vars, err := processVars(fsys, entryPath)
			if err != nil {
				return Role{}, false, err
			}
			role.Variables = vars

		case "tasks":
			isARole = true
			tasks, err := processTasks(ctx, fsys, entryPath)
			if err != nil {
				return Role{}, false, err
			}
			role.Tasks = tasks

		// TODO: those are dirs found in a role, but not implemented currently.
		case "handlers", "templates", "files", "meta", "library", "module_utils", "lookup_plugins":
			isARole = true
		default:
			isARole = false
		}
	}

	return role, isARole, nil
}
