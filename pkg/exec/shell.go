package exec

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/registry"
)

//	@meta{
//	  "deviations": []
//	}
type Shell struct {
	*proto.Shell `yaml:",inline"`

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
	reg := registry.TaskRegistration{
		ProtoFactory: func() any { return &proto.Shell{} },
		ProtoWrapper: func(msg any) any { return &proto.Task_Shell{Shell: msg.(*proto.Shell)} },
		ExecAdapter: func(content any) any {
			if c, ok := content.(*proto.Task_Shell); ok {
				return &Shell{Shell: c.Shell}
			}
			return nil
		},
	}
	registry.Register("shell", reg, (*proto.Task_Shell)(nil))
	registry.Register("ansible.builtin.shell", reg, (*proto.Task_Shell)(nil))
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
