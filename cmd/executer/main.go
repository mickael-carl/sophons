package main

import (
	"context"
	"errors"
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
	inventoryPath = flag.String("i", "", "path to inventory file")
	node          = flag.String("n", "localhost", "name of the node to run the playbook against")
)

func untarIfExists(playbookDir, name string) error {
	archivePath := filepath.Join(playbookDir, fmt.Sprintf("%s.tar.gz", name))

	_, err := os.Stat(archivePath)
	if err == nil {
		if err := util.UntarRoles(archivePath, playbookDir); err != nil {
			return fmt.Errorf("failed to unpack %s.tar.gz: %w", name, err)
		}
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to stat %s.tar.gz: %w", name, err)
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatal("usage: executer spec.yaml")
	}

	groups := map[string]struct{}{"all": struct{}{}}
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

	for _, dir := range []string{"roles", "files", "templates"} {
		if err := untarIfExists(playbookDir, dir); err != nil {
			log.Fatalf("failed to untar archive for %s: %v", dir, err)
		}
	}

	rolesDir := filepath.Join(playbookDir, "roles")

	roles, err := role.DiscoverRoles(ctx, rolesDir)
	if err != nil {
		log.Fatalf("failed to discover roles: %v", err)
	}

	playbookPath := flag.Args()[0]
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		log.Fatalf("failed to read playbook from %s: %v", playbookPath, err)
	}

	var playbook playbook.Playbook
	if err := yaml.UnmarshalContext(ctx, playbookData, &playbook); err != nil {
		log.Fatalf("failed to unmarshal playbook from %s: %v", playbookPath, err)
	}

	for _, play := range playbook {
		if _, ok := groups[play.Hosts]; ok || play.Hosts == *node {
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
						log.Fatalf("validation failed: %v", err)
					}

					if err := task.Apply(filepath.Join(rolesDir, roleName)); err != nil {
						log.Fatalf("failed to apply task: %v", err)
					}
				}
			}
			for _, task := range play.Tasks {
				// TODO: better formatting or maybe make that a new method.
				log.Printf("%+v", task)

				if err := task.Validate(); err != nil {
					log.Fatalf("validation failed: %v", err)
				}

				if err := task.Apply(playbookDir); err != nil {
					log.Fatalf("failed to apply task: %v", err)
				}
			}
		}
	}
}
