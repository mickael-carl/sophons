package role

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/variables"
)

type Role struct {
	// TODO: add files and templates.
	vars  variables.Variables
	tasks []exec.Task
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

// maybeRole checks if a given directory is indeed an Ansible role or not by
// checking for known directories that are a role may contain. It does so in
// the order required to preserve variables precedence rules.
func maybeRole(ctx context.Context, fsys fs.FS, name string) (Role, bool, error) {
	isARole := false
	var role Role

	var defaults variables.Variables
	// First check for defaults, since those have low precedence.
	_, err := fs.Stat(fsys, path.Join(name, "defaults"))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Role{}, false, err
		}
	} else {
		isARole = true
		defaults, err = processVars(fsys, path.Join(name, "defaults"))
		if err != nil {
			return Role{}, false, err
		}
	}

	var vars variables.Variables
	// Then check for role variables, since those have higher precedence than
	// defaults, but lower than play/inventory-level variables.
	// First check for defaults, since those have low precedence.
	_, err = fs.Stat(fsys, path.Join(name, "vars"))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return Role{}, false, err
		}
	} else {
		isARole = true
		vars, err = processVars(fsys, path.Join(name, "vars"))
		if err != nil {
			return Role{}, false, err
		}
	}

	// Per https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_variables.html#understanding-variable-precedence:
	// defaults have lowest precedence. Over that are inventory vars, then
	// playbook vars, then role vars.
	roleVars := variables.Variables{}
	maps.Copy(roleVars, defaults)

	additionalVars, ok := variables.FromContext(ctx)
	if !ok {
		additionalVars = variables.Variables{}
	}
	maps.Copy(roleVars, additionalVars)

	maps.Copy(roleVars, vars)

	role.vars = roleVars

	// Then process the rest of the directory (order doesn't really matter anymore).
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return Role{}, false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := path.Join(name, entry.Name())

		// TODO: per the Ansible docs, a directory is a role if any of the
		// below is *populated*. We only check for the directory's existence
		// instead. This may introduce a deviation in behaviour.
		switch entry.Name() {
		case "tasks":
			isARole = true
			tasks, err := processTasks(fsys, entryPath)
			if err != nil {
				return Role{}, false, err
			}
			role.tasks = tasks

		// TODO: those are dirs found in a role, but not implemented currently.
		case "handlers", "templates", "files", "meta", "library", "module_utils", "lookup_plugins":
			isARole = true
		default:
		}
	}

	return role, isARole, nil
}

func (r *Role) Apply(parentPath string) error {
	// We need to apply role variables here, since we've resolved variables in
	// DiscoverRoles in the right precedence order.
	ctx := variables.NewContext(context.Background(), r.vars)
	for _, task := range r.tasks {
		if err := exec.ExecuteTask(ctx, task, parentPath, true); err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}
	}

	return nil
}
