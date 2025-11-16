package exec

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/mickael-carl/sophons/pkg/exec/util"
)

//	@meta{
//	  "deviations": []
//	}
type Shell struct {
	Argv            []string `sophons:"implemented"`
	Chdir           string   `sophons:"implemented"`
	Cmd             string   `sophons:"implemented"`
	Creates         string   `sophons:"implemented"`
	Executable      string   `sophons:"implemented"`
	Removes         string   `sophons:"implemented"`
	Stdin           string   `sophons:"implemented"`
	StdinAddNewline *bool    `yaml:"stdin_add_newline" sophons:"implemented"`
}

type ShellResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("shell", func() TaskContent { return &Shell{} })
	RegisterTaskType("ansible.builtin.shell", func() TaskContent { return &Shell{} })
}

func (s *Shell) Validate() error {
	return util.ValidateCmd(s.Argv, s.Cmd, s.Stdin, s.StdinAddNewline)
}

func (s *Shell) Apply(_ context.Context, _ string, _ bool) (Result, error) {
	cmdFunc := func() *exec.Cmd {
		var cmd *exec.Cmd
		exe := "/bin/sh"
		if s.Executable != "" {
			exe = s.Executable
		}

		if s.Cmd != "" {
			cmd = exec.Command(exe, "-c", s.Cmd)
		}
		if len(s.Argv) != 0 {
			cmd = exec.Command(exe, "-c", strings.Join(s.Argv, " "))
		}
		return cmd
	}

	out, err := util.ApplyCmd(cmdFunc, s.Creates, s.Removes, s.Chdir, s.Stdin, s.StdinAddNewline)
	if err != nil {
		return &ShellResult{}, fmt.Errorf("failed to execute command: %s", string(out))
	}

	// TODO: Debug.
	if len(out) > 0 {
		log.Print(string(out))
	}

	return &ShellResult{}, nil
}
