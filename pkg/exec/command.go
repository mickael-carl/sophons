package exec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

//	@meta {
//	  "deviations": []
//	}
type Command struct {
	Cmd                jinjaString   `sophons:"implemented"`
	Argv               []jinjaString `sophons:"implemented"`
	Creates            jinjaString   `sophons:"implemented"`
	Removes            jinjaString   `sophons:"implemented"`
	Chdir              jinjaString   `sophons:"implemented"`
	ExpandArgumentVars bool          `yaml:"expand_argument_vars" sophons:"implemented"`
	Stdin              jinjaString   `sophons:"implemented"`
	StdinAddNewline    *bool         `yaml:"stdin_add_newline" sophons:"implemented"`
	StripEmptyEnds     *bool         `yaml:"strip_empty_ends"`
}

func init() {
	RegisterTaskType("command", func() Task { return &Command{} })
	RegisterTaskType("ansible.builtin.command", func() Task { return &Command{} })
}

func (c *Command) Validate() error {
	return validateCmd(c.Argv, c.Cmd, c.Stdin, c.StdinAddNewline)
}

func (c *Command) Apply(_ string) error {
	cmdFunc := func() *exec.Cmd {
		var cmd *exec.Cmd
		if string(c.Cmd) != "" {
			var splitCmd []string
			if c.ExpandArgumentVars {
				splitCmd = strings.Split(os.ExpandEnv(string(c.Cmd)), " ")
			} else {
				splitCmd = strings.Split(string(c.Cmd), " ")
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
					argv = append(argv, os.ExpandEnv(string(arg)))
				}
			} else {
				for _, arg := range c.Argv {
					argv = append(argv, string(arg))
				}
			}

			if len(argv) > 1 {
				cmd = exec.Command(argv[0], argv[1:]...)
			} else {
				cmd = exec.Command(argv[0])
			}
		}
		return cmd
	}

	out, err := applyCmd(cmdFunc, c.Creates, c.Removes, c.Chdir, c.Stdin, c.StdinAddNewline)
	if err != nil {
		return fmt.Errorf("failed to execute command: %s", string(out))
	}

	// TODO: Debug.
	if len(out) > 0 {
		log.Print(string(out))
	}

	return nil
}
