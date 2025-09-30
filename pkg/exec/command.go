package exec

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Cmd  string
	Argv []string
	// TODO: support glob patterns.
	Creates            string
	Removes            string
	Chdir              string
	ExpandArgumentVars bool `yaml:"expand_argument_vars"`
	Stdin              string
	StdinAddNewline    bool `yaml:"stdin_add_newline"`
}

func (c *Command) Validate() error {
	if c.Cmd != "" && len(c.Argv) != 0 {
		return errors.New("cmd and argv can't be both specified at the same time")
	}

	if c.Cmd == "" && len(c.Argv) == 0 {
		return errors.New("either cmd or argv need to be specified")
	}

	if c.Stdin == "" && c.StdinAddNewline {
		return errors.New("stdin_add_newline can't be set if stdin is unset")
	}

	return nil
}

func (c *Command) shouldApply() (bool, error) {
	if c.Creates != "" {
		_, err := os.Stat(c.Creates)
		if err == nil {
			return false, nil
		} else {
			if errors.Is(err, os.ErrNotExist) {
				return true, nil
			} else {
				return false, err
			}
		}
	}

	if c.Removes != "" {
		_, err := os.Stat(c.Removes)
		if err == nil {
			return true, nil
		} else {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			} else {
				return false, err
			}
		}
	}

	return true, nil
}

func (c *Command) Apply() error {
	ok, err := c.shouldApply()
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

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
	} else if len(c.Argv) != 0 {
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
	} else {
		return errors.New("invalid command")
	}

	if c.Chdir != "" {
		cmd.Dir = c.Chdir
	}

	if c.Stdin != "" {
		stdin := c.Stdin
		if c.StdinAddNewline {
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
