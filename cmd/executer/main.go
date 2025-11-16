package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/nikolalohinski/gonja/v2"

	"go.uber.org/zap"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
	"github.com/mickael-carl/sophons/pkg/playbook"
	"github.com/mickael-carl/sophons/pkg/role"
	"github.com/mickael-carl/sophons/pkg/util"
	"github.com/mickael-carl/sophons/pkg/variables"
)

var (
	inventoryPath    = flag.String("i", "", "path to inventory file")
	dataArchive      = flag.String("d", "", "path to data archive")
	playbooksDirName = flag.String("p", "", "name of the directory containing playbooks")
	node             = flag.String("n", "localhost", "name of the node to run the playbook against")
)

func playbookApply(ctx context.Context, logger *zap.Logger, playbookPath, node string, groups map[string]struct{}, roles map[string]role.Role, rolesDir string) error {
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		return fmt.Errorf("failed to read playbook from %s: %w", playbookPath, err)
	}

	var playbook playbook.Playbook
	if err := yaml.Unmarshal(playbookData, &playbook); err != nil {
		return fmt.Errorf("failed to unmarshal playbook from %s: %w", playbookPath, err)
	}

	for _, play := range playbook {
		if _, ok := groups[play.Hosts]; ok || play.Hosts == node {
			inventoryVars, ok := variables.FromContext(ctx)
			if !ok {
				inventoryVars = variables.Variables{}
			}

			playVars := variables.Variables{}
			playVars.Merge(inventoryVars)

			playVars.Merge(play.Vars)

			for _, varsFile := range play.VarsFiles {
				absVarsFilePath := filepath.Join(filepath.Dir(playbookPath), varsFile)
				fileVars, err := variables.LoadFromFile(absVarsFilePath)
				if err != nil {
					return fmt.Errorf("failed to load vars file %s for play: %w", absVarsFilePath, err)
				}
				playVars.Merge(fileVars)
			}

			playCtx := variables.NewContext(ctx, playVars)

			// Ansible executes roles first, then tasks. See
			// https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_reuse_roles.html#using-roles-at-the-play-level.
			for _, roleName := range play.Roles {
				logger.Debug("executing role", zap.String("role", roleName))

				role, ok := roles[roleName]
				if !ok {
					return fmt.Errorf("no such role: %s", roleName)
				}

				// Headsup: roles variables are *not* scoped to only the role
				// itself. This means this call actually *has to mutate*
				// playCtx, so that variables defined in a role can be used in
				// subsequent ones as well as the rest of the play. Sorry
				// Ansible but this is STUPID.
				if err := role.Apply(playCtx, logger, filepath.Join(rolesDir, roleName)); err != nil {
					return fmt.Errorf("failed to apply role %s: %w", roleName, err)
				}
			}
			for _, task := range play.Tasks {
				if err := exec.ExecuteTask(playCtx, logger, task, filepath.Dir(playbookPath), false); err != nil { // use playCtx
					return fmt.Errorf("failed to execute task: %w", err)
				}
			}
		}
	}
	return nil
}

func main() {
	gonja.DefaultConfig.StrictUndefined = true
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync() //nolint:errcheck

	if len(flag.Args()) != 1 {
		logger.Fatal("usage: executer spec.yaml")
	}

	if *dataArchive != "" && *playbooksDirName == "" || *dataArchive == "" && *playbooksDirName != "" {
		logger.Fatal("when either -d or -p is set, both flags must be set")
	}

	groups := map[string]struct{}{"all": {}}
	vars := variables.Variables{}

	if *inventoryPath != "" {
		inventoryData, err := os.ReadFile(*inventoryPath)
		if err != nil {
			logger.Fatal("failed to read inventory", zap.String("path", *inventoryPath), zap.Error(err))
		}

		var inventory inventory.Inventory
		if err := yaml.Unmarshal(inventoryData, &inventory); err != nil {
			logger.Fatal("failed to unmarshal inventory", zap.String("path", *inventoryPath), zap.Error(err))
		}

		groups = inventory.Find(*node)
		vars = inventory.NodeVars(*node)
	}

	ctx := variables.NewContext(context.Background(), vars)

	playbookDir := filepath.Dir(flag.Args()[0])
	if *dataArchive != "" {
		if err := util.Untar(*dataArchive, filepath.Dir(*dataArchive)); err != nil {
			logger.Fatal("failed to untar archive", zap.String("path", *dataArchive), zap.Error(err))
		}
		playbookDir = filepath.Join(filepath.Dir(*dataArchive), *playbooksDirName)
	}

	rolesDir := filepath.Join(playbookDir, "roles")
	fsys := os.DirFS(rolesDir)

	roles, err := role.DiscoverRoles(fsys)
	if err != nil {
		logger.Fatal("failed to discover roles", zap.Error(err))
	}

	playbookPath := flag.Args()[0]
	if err := playbookApply(ctx, logger, playbookPath, *node, groups, roles, rolesDir); err != nil {
		logger.Fatal("failed to run playbook", zap.String("path", playbookPath), zap.Error(err))
	}
}
