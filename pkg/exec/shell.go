package exec

import (
	"context"
	"fmt"
	"log"
	"strings"
)

//	@meta{
//	  "deviations": []
//	}
type Shell struct {
	Argv            []string `sophons:"implemented"`
	Chdir           string   `sophons:"implemented"`
	Cmd             string   `sophons:"implemented"`
	Creates         string   `sophons:"implemented"`
	Executable      string   `sophons:"implemented"`
	Removes         string   `sophons:"implemented"`
	Stdin           string   `sophons:"implemented"`
	StdinAddNewline *bool    `yaml:"stdin_add_newline" sophons:"implemented"`

	cmdFactory cmdFactory
}

type ShellResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("shell", func() TaskContent { return &Shell{} })
	RegisterTaskType("ansible.builtin.shell", func() TaskContent { return &Shell{} })
}

func (s *Shell) Validate() error {
	return validateCmd(s.Argv, s.Cmd, s.Stdin, s.StdinAddNewline)
}

func (s *Shell) Apply(_ context.Context, _ string, _ bool) (Result, error) {
	if s.cmdFactory == nil {
		s.cmdFactory = realCmdFactory
	}

	result := ShellResult{}

	if ok, err := shouldApply(s.Creates, s.Removes); err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to check creates/removes: %w", err)
	} else if !ok {
		result.TaskSkipped()
		return &result, nil
	}

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

	stdout, _, _, err := ApplyCommand(s.cmdFactory, s.Chdir, s.Stdin, s.StdinAddNewline, name, args)
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
