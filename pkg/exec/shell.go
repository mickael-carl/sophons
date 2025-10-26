package exec

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

//	@meta{
//	  "deviations": []
//	}
type Shell struct {
	Argv            []jinjaString `sophons:"implemented"`
	Chdir           jinjaString   `sophons:"implemented"`
	Cmd             jinjaString   `sophons:"implemented"`
	Creates         jinjaString   `sophons:"implemented"`
	Executable      jinjaString   `sophons:"implemented"`
	Removes         jinjaString   `sophons:"implemented"`
	Stdin           jinjaString   `sophons:"implemented"`
	StdinAddNewline *bool         `yaml:"stdin_add_newline" sophons:"implemented"`
}

func init() {
	RegisterTaskType("shell", func() TaskContent { return &Shell{} })
	RegisterTaskType("ansible.builtin.shell", func() TaskContent { return &Shell{} })
}

func (s *Shell) Validate() error {
	return validateCmd(s.Argv, s.Cmd, s.Stdin, s.StdinAddNewline)
}

func (s *Shell) Apply(_ string, _ bool) error {
	cmdFunc := func() *exec.Cmd {
		var cmd *exec.Cmd
		exe := "/bin/sh"
		if s.Executable != "" {
			exe = string(s.Executable)
		}

		if s.Cmd != "" {
			cmd = exec.Command(exe, "-c", string(s.Cmd))
		}
		if len(s.Argv) != 0 {
			args := []string{}
			for _, arg := range s.Argv {
				args = append(args, string(arg))
			}
			cmd = exec.Command(exe, "-c", strings.Join(args, " "))
		}
		return cmd
	}

	out, err := applyCmd(cmdFunc, s.Creates, s.Removes, s.Chdir, s.Stdin, s.StdinAddNewline)
	if err != nil {
		return fmt.Errorf("failed to execute command: %s", string(out))
	}

	// TODO: Debug.
	if len(out) > 0 {
		log.Print(string(out))
	}

	return nil
}
