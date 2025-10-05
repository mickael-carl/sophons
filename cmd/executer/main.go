package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
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
	variables := inventory.Variables{}

	if *inventoryPath != "" {
		inventoryData, err := os.ReadFile(*inventoryPath)
		if err != nil {
			log.Fatal(err)
		}

		var inventory inventory.Inventory
		if err := yaml.Unmarshal(inventoryData, &inventory); err != nil {
			log.Fatal(err)
		}

		groups = inventory.Find(*node)
		variables = inventory.NodeVars(*node)
	}

	ctx := context.WithValue(context.Background(), "vars", variables)

	playbookData, err := os.ReadFile(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}

	var playbook exec.Playbook
	if err := yaml.UnmarshalContext(ctx, playbookData, &playbook); err != nil {
		log.Fatal(err)
	}

	for _, play := range playbook {
		if _, ok := groups[play.Hosts]; ok || play.Hosts == *node {
			for _, task := range play.Tasks {
				log.Printf("%+v", task)

				if err := task.Validate(); err != nil {
					log.Fatal(err)
				}

				if err := task.Apply(); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}
