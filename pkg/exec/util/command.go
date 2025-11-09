package util

import (
	"errors"
	"os/exec"
	"strings"
)

func ValidateCmd(argv []string, cmd, stdin string, stdinAddNewline *bool) error {
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

// ApplyCmd expects an *exec.Cmd that already has Args set, e.g. by calling
// exec.Command("foo").
func ApplyCmd(cmdFunc func() *exec.Cmd, creates, removes, chdir, stdin string, stdinAddNewline *bool) ([]byte, error) {
	ok, err := shouldApply(creates, removes)
	if err != nil {
		return []byte{}, err
	}

	if !ok {
		return []byte{}, nil
	}

	cmd := cmdFunc()

	if chdir != "" {
		cmd.Dir = chdir
	}

	if stdin != "" {
		cmdStdin := stdin

		if stdinAddNewline == nil || stdinAddNewline != nil && *stdinAddNewline {
			cmdStdin += "\n"
		}
		cmd.Stdin = strings.NewReader(cmdStdin)
	}

	return cmd.CombinedOutput()
}
