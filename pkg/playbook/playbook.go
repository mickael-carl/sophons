package playbook

import (
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/variables"
)

type Playbook []Play

type Play struct {
	Hosts     string `yaml:"hosts"`
	Roles     []string
	Tasks     []*proto.Task
	Vars      variables.Variables
	VarsFiles []string `yaml:"vars_files"`
}
