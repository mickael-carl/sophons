package exec

import "testing"

var pFalse = false

func TestValidateCmdMissingCommand(t *testing.T) {
	err := validateCmd([]jinjaString{}, "", "", &pFalse)
	if err == nil {
		t.Error("a command with cmd or argv set is not valid")
	}

	if err.Error() != "either cmd or argv need to be specified" {
		t.Error(err)
	}
}

func TestValidateCmdConflictingParameters(t *testing.T) {
	err := validateCmd(
		[]jinjaString{
			"ls",
			"-l",
		},
		"ls -l",
		"",
		&pFalse,
	)

	if err == nil {
		t.Error("a command with both cmd and argv set is not valid")
	}

	if err.Error() != "cmd and argv can't be both specified at the same time" {
		t.Error(err)
	}
}

func TestValidateCmd(t *testing.T) {
	if err := validateCmd(
		[]jinjaString{},
		"ls -l",
		"",
		&pFalse,
	); err != nil {
		t.Error(err)
	}
}

func TestValidateCmdStdinMissing(t *testing.T) {
	pTrue := true

	err := validateCmd(
		[]jinjaString{},
		"cat",
		"",
		&pTrue,
	)
	if err == nil {
		t.Error("a command with stdin_add_newline and without stdin is not valid")
	}

	if err.Error() != "stdin_add_newline can't be set if stdin is unset" {
		t.Error(err)
	}
}
