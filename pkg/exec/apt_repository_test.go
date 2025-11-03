package exec

import "testing"

func TestAptRepositoryValidateInvalidState(t *testing.T) {
	a := &AptRepository{
		Repo:  "foo",
		State: "banana",
	}

	err := a.Validate()
	if err == nil {
		t.Error("banana is not a valid state")
	}

	if err.Error() != "unsupported state: banana" {
		t.Error(err)
	}
}

func TestAptRepositoryValidateMissingRepo(t *testing.T) {
	a := &AptRepository{
		State: "present",
	}

	err := a.Validate()
	if err == nil {
		t.Error("repo is required")
	}

	if err.Error() != "repo is required" {
		t.Error(err)
	}
}

func TestAptRepositoryValidate(t *testing.T) {
	a := &AptRepository{
		Repo:  "foo",
		State: "present",
	}

	if err := a.Validate(); err != nil {
		t.Error(err)
	}
}
