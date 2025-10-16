package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
	"github.com/mickael-carl/sophons/pkg/role"
	"github.com/mickael-carl/sophons/pkg/util"
	"github.com/mickael-carl/sophons/pkg/variables"
)

var (
	inventoryPath = flag.String("i", "", "path to inventory file")
	node          = flag.String("n", "localhost", "name of the node to run the playbook against")
)

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatal("usage: executer spec.yaml")
	}

	groups := map[string]struct{}{"all": struct{}{}}
	variables := variables.Variables{}

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
		variables = inventory.NodeVars(*node)
	}

	ctx := context.WithValue(context.Background(), "vars", variables)

	playbookDir := filepath.Dir(flag.Args()[0])

	rolesArchivePath := filepath.Join(playbookDir, "roles.tar.gz")
	rolesDir := filepath.Join(playbookDir, "roles")

	_, err := os.Stat(rolesArchivePath)
	if err == nil {
		tmpDir, err := os.MkdirTemp("", "sophons-roles")
		if err != nil {
			log.Fatalf("failed to create temp dir for roles: %v", err)
		}
		// TODO: not going to work because log.Fatal.
		//defer os.RemoveAll(tmpDir)

		if err := util.UntarRoles(rolesArchivePath, tmpDir); err != nil {
			log.Fatalf("failed to unpack roles archive: %v", err)
		}

		rolesDir = filepath.Join(tmpDir, "roles")
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatalf("failed to stat roles archive file: %v", err)
		}
	}

	roles, err := role.DiscoverRoles(ctx, rolesDir)
	if err != nil {
		log.Fatalf("failed to discover roles: %v", err)
	}

	playbookPath := flag.Args()[0]
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		log.Fatalf("failed to read playbook from %s: %v", playbookPath, err)
	}

	var playbook exec.Playbook
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

					if err := task.Apply(); err != nil {
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

				if err := task.Apply(); err != nil {
					log.Fatalf("failed to apply task: %v", err)
				}
			}
		}
	}
}
