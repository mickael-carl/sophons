package exec

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mickael-carl/sophons/pkg/proto"
)

//	@meta{
//	  "deviations": []
//	}
type Shell struct {
	proto.Shell `yaml:",inline"`

	cmdFactory cmdFactory
}

type ShellResult struct {
	CommonResult `yaml:",inline"`

	Cmd   string
	Delta time.Duration
	End   time.Time
	Start time.Time
}

func init() {
	RegisterTaskType("shell", func() TaskContent { return &Shell{} })
	RegisterTaskType("ansible.builtin.shell", func() TaskContent { return &Shell{} })
}

func (s *Shell) Validate() error {
	return validateCmd(s.Argv, s.Cmd, s.Stdin, s.StdinAddNewline)
}

func (s *Shell) Apply(ctx context.Context, _ string, _ bool) (Result, error) {
	if ctxCmdFactory, ok := ctx.Value(commandFactoryContextKey).(cmdFactory); ok {
		s.cmdFactory = ctxCmdFactory
	} else {
		s.cmdFactory = realCmdFactory
	}

	result := ShellResult{}

	var args []string
	name := "/bin/sh"
	if s.Executable != "" {
		name = s.Executable
	}

	var cmdStr string
	if s.Cmd != "" {
		cmdStr = s.Cmd
	} else if len(s.Argv) != 0 {
		cmdStr = strings.Join(s.Argv, " ")
	}
	args = []string{"-c", cmdStr}

	// This is silly: shell uses a string cmd in its return values, where
	// command has a list of strings.
	result.Cmd = cmdStr

	if ok, err := shouldApply(s.Creates, s.Removes); err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to check creates/removes: %w", err)
	} else if !ok {
		result.TaskSkipped()
		return &result, nil
	}

	start := time.Now()
	stdout, stderr, rc, err := ApplyCommand(s.cmdFactory, s.Chdir, s.Stdin, s.StdinAddNewline, name, args)
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
