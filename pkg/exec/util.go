package exec

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func validateCmd(argv []jinjaString, cmd, stdin jinjaString, stdinAddNewline *bool) error {
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

// applyCmd expects an *exec.Cmd that already has Args set, e.g. by calling
// exec.Command("foo").
func applyCmd(cmdFunc func() *exec.Cmd, creates, removes, chdir, stdin jinjaString, stdinAddNewline *bool) ([]byte, error) {
	ok, err := shouldApply(string(creates), string(removes))
	if err != nil {
		return []byte{}, err
	}

	if !ok {
		return []byte{}, nil
	}

	cmd := cmdFunc()

	if chdir != "" {
		cmd.Dir = string(chdir)
	}

	if stdin != "" {
		cmdStdin := string(stdin)

		if stdinAddNewline == nil || stdinAddNewline != nil && *stdinAddNewline {
			cmdStdin += "\n"
		}
		cmd.Stdin = strings.NewReader(cmdStdin)
	}

	return cmd.CombinedOutput()

}
