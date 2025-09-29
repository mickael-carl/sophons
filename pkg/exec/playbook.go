package exec

import (
	"errors"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

type Playbook []Play

type Play struct {
	Hosts string `yaml:"hosts"`
	Tasks []Task
}

type Task interface {
	Validate() error
	Apply() error
}

func init() {
	yaml.RegisterCustomUnmarshaler[Play](playUnmarshalYAML)
}

func playUnmarshalYAML(p *Play, b []byte) error {
	var raw struct {
		Hosts string
		Tasks []map[string]ast.Node
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return err
	}

	p.Hosts = raw.Hosts

	var tasksOut []Task
	for _, task := range raw.Tasks {
		for taskType, node := range task {
			switch taskType {
			case "file", "ansible.builtin.file":
				var f File
				if err := yaml.NodeToValue(node, &f); err != nil {
					return err
				}
				tasksOut = append(tasksOut, &f)
			default:
				return errors.New("unsupported task type")
			}
		}
	}

	p.Tasks = tasksOut
	return nil
}
