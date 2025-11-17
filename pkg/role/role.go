package role

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"go.uber.org/zap"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/variables"
)

type Role struct {
	// TODO: add files and templates.
	defaults variables.Variables
	vars     variables.Variables
	tasks    []exec.Task
}

func DiscoverRoles(fsys fs.FS) (map[string]Role, error) {
	roles := map[string]Role{}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return map[string]Role{}, fmt.Errorf("failed to discover roles: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		role, ok, err := maybeRole(fsys, entry.Name())
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
func maybeRole(fsys fs.FS, name string) (Role, bool, error) {
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

	role.defaults = defaults
	role.vars = vars

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

func (r *Role) Apply(ctx context.Context, logger *zap.Logger, parentPath string) error {
	inventoryAndPlayVars, ok := variables.FromContext(ctx)
	if !ok {
		inventoryAndPlayVars = variables.Variables{}
	}

	roleCtxVars := make(variables.Variables)
	roleCtxVars.Merge(r.defaults)
	roleCtxVars.Merge(inventoryAndPlayVars)
	roleCtxVars.Merge(r.vars)

	roleCtx := variables.NewContext(ctx, roleCtxVars)
	for _, task := range r.tasks {
		if err := exec.ExecuteTask(roleCtx, logger, task, parentPath, true); err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}
	}

	// After the role has been executed, its variables should be merged back
	// into the main context for the rest of the play. Role defaults have
	// the lowest precedence, so we don't merge them back. See
	// https://docs.ansible.com/projects/ansible/latest/playbook_guide/playbooks_variables.html#tips-on-where-to-set-variables
	// for more details, but specifically:
	// > Variables set in one role are available to later roles. You can set
	// > variables in the role’s vars directory [...] and use them in other roles
	// > and elsewhere in your playbook
	// Author's note: this is utterly insane. So much for scoping. If this code
	// ever makes it to production somewhere, we should absolutely disable
	// this madness. It is *DANGEROUS*.
	inventoryAndPlayVars.Merge(r.vars)

	return nil
}
