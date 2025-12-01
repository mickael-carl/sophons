package exec

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mickael-carl/sophons/pkg/proto"
)

//	@meta {
//	  "deviations": []
//	}
type Command struct {
	proto.Command `yaml:",inline"`

	cmdFactory cmdFactory
}

type CommandResult struct {
	CommonResult `yaml:",inline"`

	Cmd   []string
	Delta time.Duration
	End   time.Time
	Start time.Time
}

func init() {
	RegisterTaskType("command", func() TaskContent { return &Command{} })
	RegisterTaskType("ansible.builtin.command", func() TaskContent { return &Command{} })
}

func (c *Command) Validate() error {
	return validateCmd(c.Argv, c.Cmd, c.Stdin, c.StdinAddNewline)
}

func (c *Command) Apply(ctx context.Context, _ string, _ bool) (Result, error) {
	if ctxCmdFactory, ok := ctx.Value(commandFactoryContextKey).(cmdFactory); ok {
		c.cmdFactory = ctxCmdFactory
	} else {
		c.cmdFactory = realCmdFactory
	}

	result := CommandResult{}
	var name string
	var args []string
	if c.Cmd != "" {
		if c.ExpandArgumentVars {
			splitCmd := strings.Fields(os.ExpandEnv(c.Cmd))
			name = splitCmd[0]
			if len(splitCmd) > 1 {
				args = splitCmd[1:]
			}
		} else {
			splitCmd := strings.Fields(c.Cmd)
			name = splitCmd[0]
			if len(splitCmd) > 1 {
				args = splitCmd[1:]
			}
		}
	} else if len(c.Argv) != 0 {
		if c.ExpandArgumentVars {
			var expandedArgs []string
			for _, arg := range c.Argv {
				expandedArgs = append(expandedArgs, os.ExpandEnv(arg))
			}
			name = expandedArgs[0]
			if len(expandedArgs) > 1 {
				args = expandedArgs[1:]
			}
		} else {
			name = c.Argv[0]
			if len(c.Argv) > 1 {
				args = c.Argv[1:]
			}
		}
	}

	result.Cmd = append([]string{name}, args...)

	if ok, err := shouldApply(c.Creates, c.Removes); err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to check creates/removes: %w", err)
	} else if !ok {
		result.TaskSkipped()
		return &result, nil
	}

	start := time.Now()
	stdout, stderr, rc, err := ApplyCommand(c.cmdFactory, c.Chdir, c.Stdin, c.StdinAddNewline, name, args)
	end := time.Now()

	result.Start = start
	result.End = end
	result.Delta = end.Sub(start)
	result.Stdout = stdout
	result.Stderr = stderr
	result.RC = rc

	if err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to execute command: %w", err)
	}

	// TODO: Debug.
	if stdout != "" {
		log.Print(stdout)
	}

	result.TaskChanged()
	return &result, nil
}
