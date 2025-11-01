package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

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

func playbookApply(ctx context.Context, playbookPath, node string, groups map[string]struct{}, roles map[string]role.Role, rolesDir string) error {
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		return fmt.Errorf("failed to read playbook from %s: %w", playbookPath, err)
	}

	var playbook playbook.Playbook
	if err := yaml.UnmarshalContext(ctx, playbookData, &playbook); err != nil {
		return fmt.Errorf("failed to unmarshal playbook from %s: %w", playbookPath, err)
	}

	for _, play := range playbook {
		if _, ok := groups[play.Hosts]; ok || play.Hosts == node {
			// Ansible executes roles first, then tasks. See
			// https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_reuse_roles.html#using-roles-at-the-play-level.
			for _, roleName := range play.Roles {
				// TODO: debug.
				log.Printf("executing %s", roleName)

				role, ok := roles[roleName]
				if !ok {
					log.Fatalf("no such role: %s", roleName)
				}

				for _, task := range role.Tasks {
					// TODO: remove the duplication with the play level tasks
					// execution: add a Apply() to Play and Role and call that.

					// TODO: better formatting or maybe make that a new method.
					log.Printf("%+v", task)

					if err := task.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					if err := task.Apply(filepath.Join(rolesDir, roleName), true); err != nil {
						return fmt.Errorf("failed to apply task: %w", err)
					}
				}
			}
			for _, task := range play.Tasks {
				// TODO: better formatting or maybe make that a new method.
				log.Printf("%+v", task)

				if err := task.Validate(); err != nil {
					return fmt.Errorf("validation failed: %w", err)
				}

				if err := task.Apply(filepath.Dir(playbookPath), false); err != nil {
					return fmt.Errorf("failed to apply task: %w", err)
				}
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatal("usage: executer spec.yaml")
	}

	if *dataArchive != "" && *playbooksDirName == "" || *dataArchive == "" && *playbooksDirName != "" {
		log.Fatal("when either -d or -p is set, both flags must be set")
	}

	groups := map[string]struct{}{"all": {}}
	vars := variables.Variables{}

	if *inventoryPath != "" {
		inventoryData, err := os.ReadFile(*inventoryPath)
		if err != nil {
			log.Fatalf("failed to read inventory from %s: %v", *inventoryPath, err)
		}

		var inventory inventory.Inventory
		if err := yaml.Unmarshal(inventoryData, &inventory); err != nil {
			log.Fatalf("failed to unmarshal inventory from %s: %v", *inventoryPath, err)
		}

		groups = inventory.Find(*node)
		vars = inventory.NodeVars(*node)
	}

	ctx := variables.NewContext(context.Background(), vars)

	playbookDir := filepath.Dir(flag.Args()[0])
	if *dataArchive != "" {
		if err := util.Untar(*dataArchive, filepath.Dir(*dataArchive)); err != nil {
			log.Fatalf("failed to untar archive at %s: %v", *dataArchive, err)
		}
		playbookDir = filepath.Join(filepath.Dir(*dataArchive), *playbooksDirName)
	}

	rolesDir := filepath.Join(playbookDir, "roles")

	roles, err := role.DiscoverRoles(ctx, rolesDir)
	if err != nil {
		log.Fatalf("failed to discover roles: %v", err)
	}

	playbookPath := flag.Args()[0]
	if err := playbookApply(ctx, playbookPath, *node, groups, roles, rolesDir); err != nil {
		log.Fatal(err)
	}
}
