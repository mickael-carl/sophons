package exec

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mickael-carl/sophons/pkg/exec/util"
)

//	@meta {
//	  "deviations": []
//	}
type Command struct {
	Cmd                string   `sophons:"implemented"`
	Argv               []string `sophons:"implemented"`
	Creates            string   `sophons:"implemented"`
	Removes            string   `sophons:"implemented"`
	Chdir              string   `sophons:"implemented"`
	ExpandArgumentVars bool     `yaml:"expand_argument_vars" sophons:"implemented"`
	Stdin              string   `sophons:"implemented"`
	StdinAddNewline    *bool    `yaml:"stdin_add_newline" sophons:"implemented"`
	StripEmptyEnds     *bool    `yaml:"strip_empty_ends"`
}

type CommandResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("command", func() TaskContent { return &Command{} })
	RegisterTaskType("ansible.builtin.command", func() TaskContent { return &Command{} })
}

func (c *Command) Validate() error {
	return util.ValidateCmd(c.Argv, c.Cmd, c.Stdin, c.StdinAddNewline)
}

func (c *Command) Apply(_ context.Context, _ string, _ bool) (Result, error) {
	cmdFunc := func() *exec.Cmd {
		var cmd *exec.Cmd
		if c.Cmd != "" {
			var splitCmd []string
			if c.ExpandArgumentVars {
				splitCmd = strings.Split(os.ExpandEnv(c.Cmd), " ")
			} else {
				splitCmd = strings.Split(c.Cmd, " ")
			}
			var args []string
			if len(splitCmd) > 1 {
				args = splitCmd[1:]
			}
			cmd = exec.Command(splitCmd[0], args...)
		}
		if len(c.Argv) != 0 {
			var argv []string
			if c.ExpandArgumentVars {
				for _, arg := range c.Argv {
					argv = append(argv, os.ExpandEnv(arg))
				}
			} else {
				argv = c.Argv
			}

			if len(argv) > 1 {
				cmd = exec.Command(argv[0], argv[1:]...)
			} else {
				cmd = exec.Command(argv[0])
			}
		}
		return cmd
	}

	out, err := util.ApplyCmd(cmdFunc, c.Creates, c.Removes, c.Chdir, c.Stdin, c.StdinAddNewline)
	if err != nil {
		return &CommandResult{}, fmt.Errorf("failed to execute command: %s", string(out))
	}

	// TODO: Debug.
	if len(out) > 0 {
		log.Print(string(out))
	}

	return &CommandResult{}, nil
}
