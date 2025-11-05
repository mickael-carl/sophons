package exec

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

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

func TestAptRepositoryUnmarshalYAML(t *testing.T) {
	b := []byte(`
repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable"
state: "present"
update_cache: true`)

	var got AptRepository
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pTrue := true
	expected := AptRepository{
		Repo:        "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
		State:       AptRepositoryPresent,
		UpdateCache: &pTrue,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestAptRepositoryUnmarshalYAMLAliases(t *testing.T) {
	b := []byte(`
repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable"
state: "present"
update-cache: true`)

	var got AptRepository
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pTrue := true
	expected := AptRepository{
		Repo:        "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
		State:       AptRepositoryPresent,
		UpdateCache: &pTrue,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
