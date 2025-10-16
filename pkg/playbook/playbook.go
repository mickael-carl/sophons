package playbook

import "github.com/mickael-carl/sophons/pkg/exec"

type Playbook []Play

type Play struct {
	Hosts string `yaml:"hosts"`
	Roles []string
	Tasks []exec.Task
}
