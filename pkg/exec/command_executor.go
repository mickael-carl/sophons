package exec

import (
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:generate mockgen -destination=mock_command_executor_test.go -package=exec . commandExecutor

var commandFactoryContextKey = &struct{ name string }{"command"}

// CommandExecutor is an interface that allows mocking exec.Cmd's `Run`.
type commandExecutor interface {
	Run() error
	SetDir(string)
	SetStdin(io.Reader)
	SetStdout(io.Writer)
	SetStderr(io.Writer)
}

// realCommandExecutor is a wrapper around *exec.Cmd that implements CommandExecutor.
type realCommandExecutor struct {
	cmd *exec.Cmd
}

func (r *realCommandExecutor) Run() error {
	return r.cmd.Run()
}

func (r *realCommandExecutor) SetDir(dir string) {
	r.cmd.Dir = dir
}

func (r *realCommandExecutor) SetStdin(stdin io.Reader) {
	r.cmd.Stdin = stdin
}

func (r *realCommandExecutor) SetStdout(stdout io.Writer) {
	r.cmd.Stdout = stdout
}

func (r *realCommandExecutor) SetStderr(stderr io.Writer) {
	r.cmd.Stderr = stderr
}

// CmdFactory creates a CommandExecutor from a command name and arguments.
type cmdFactory func(name string, args ...string) commandExecutor

// DefaultCmdFactory creates a real exec.Cmd and wraps it to implement CommandExecutor.
func realCmdFactory(name string, args ...string) commandExecutor {
	return &realCommandExecutor{cmd: exec.Command(name, args...)}
}

type exitCoder interface {
	ExitCode() int
}

// ApplyCommand is a helper function to apply a command-like task.
func ApplyCommand(
	factory cmdFactory,
	chdir, stdin string,
	stdinAddNewline *bool,
	name string, args []string,
) (string, string, int, error) {
	cmdExecutor := factory(name, args...)

	if chdir != "" {
		cmdExecutor.SetDir(chdir)
	}

	if stdin != "" {
		cmdStdin := stdin
		if stdinAddNewline == nil || (stdinAddNewline != nil && *stdinAddNewline) {
			cmdStdin += "\n"
		}
		cmdExecutor.SetStdin(strings.NewReader(cmdStdin))
	}

	var stdout strings.Builder
	cmdExecutor.SetStdout(&stdout)

	var stderr strings.Builder
	cmdExecutor.SetStderr(&stderr)

	rc := 0
	err := cmdExecutor.Run()
	if err != nil {
		var ec exitCoder
		if errors.As(err, &ec) {
			rc = ec.ExitCode()
		} else {
			rc = -1
		}
	}

	return stdout.String(), stderr.String(), rc, err
}

func validateCmd(argv []string, cmd, stdin string, stdinAddNewline *bool) error {
	if cmd != "" && len(argv) != 0 {
		return errors.New("cmd and argv can't be both specified at the same time")
	}

	if cmd == "" && len(argv) == 0 {
		return errors.New("either cmd or argv need to be specified")
	}

	if stdin == "" && stdinAddNewline != nil && *stdinAddNewline {
		return errors.New("stdin_add_newline can't be set if stdin is unset")
	}
	return nil
}

func shouldApply(creates, removes string) (bool, error) {
	if creates != "" {
		matches, err := filepath.Glob(creates)
		if err != nil {
			return false, err
		}
		return len(matches) == 0, nil
	}

	if removes != "" {
		matches, err := filepath.Glob(removes)
		if err != nil {
			return false, err
		}
		return len(matches) > 0, nil
	}

	return true, nil
}
