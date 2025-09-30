package exec

import "testing"

func TestCommandValidateMissingCommand(t *testing.T) {
	c := Command{
		Chdir: "/tmp",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a command with cmd or argv set is not valid")
	}

	if err.Error() != "either cmd or argv need to be specified" {
		t.Error(err)
	}
}

func TestCommandValidateConflictingParameters(t *testing.T) {
	c := Command{
		Cmd: "ls -l",
		Argv: []string{
			"ls",
			"-l",
		},
	}

	err := c.Validate()
	if err == nil {
		t.Error("a command with both cmd and argv set is not valid")
	}

	if err.Error() != "cmd and argv can't be both specified at the same time" {
		t.Error(err)
	}
}

func TestCommandValidate(t *testing.T) {
	c := Command{
		Cmd:   "ls -l",
		Chdir: "/tmp",
	}

	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}

func TestCommandValidateStdinMissing(t *testing.T) {
	c := Command{
		Cmd:             "cat",
		StdinAddNewline: true,
	}

	err := c.Validate()
	if err == nil {
		t.Error("a command with stdin_add_newline and without stdin is not valid")
	}

	if err.Error() != "stdin_add_newline can't be set if stdin is unset" {
		t.Error(err)
	}
}
