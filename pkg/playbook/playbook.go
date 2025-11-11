package playbook

import (
	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/variables"
)

type Playbook []Play

type Play struct {
	Hosts     string `yaml:"hosts"`
	Roles     []string
	Tasks     []exec.Task
	Vars      variables.Variables
	VarsFiles []string `yaml:"vars_files"`
}
