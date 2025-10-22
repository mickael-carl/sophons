package exec

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	if c.Cmd != "" && len(c.Argv) != 0 {
		return errors.New("cmd and argv can't be both specified at the same time")
	}

	if c.Cmd == "" && len(c.Argv) == 0 {
		return errors.New("either cmd or argv need to be specified")
	}

	if c.Stdin == "" && c.StdinAddNewline != nil && *c.StdinAddNewline {
		return errors.New("stdin_add_newline can't be set if stdin is unset")
	}

	return nil
}

func (c *Command) shouldApply() (bool, error) {
	if string(c.Creates) != "" {
		matches, err := filepath.Glob(string(c.Creates))
		if err != nil {
			return false, err
		}
		return len(matches) == 0, nil
	}

	if string(c.Removes) != "" {
		matches, err := filepath.Glob(string(c.Removes))
		if err != nil {
			return false, err
		}
		return len(matches) > 0, nil
	}

	return true, nil
}

func (c *Command) Apply(_ string) error {
	ok, err := c.shouldApply()
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

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
	} else if len(c.Argv) != 0 {
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
	} else {
		return errors.New("invalid command")
	}

	if c.Chdir != "" {
		cmd.Dir = string(c.Chdir)
	}

	if c.Stdin != "" {
		stdin := string(c.Stdin)

		if c.StdinAddNewline == nil || c.StdinAddNewline != nil && *c.StdinAddNewline {
			stdin += "\n"
		}
		cmd.Stdin = strings.NewReader(stdin)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s", string(out))
	}

	// TODO: Debug.
	if len(out) > 0 {
		log.Print(string(out))
	}

	return nil
}
