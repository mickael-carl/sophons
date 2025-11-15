package exec

import (
	"context"
	"testing"

	"github.com/arduino/go-apt-client"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/mock/gomock"
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

	if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(AptRepository{})) {
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

	if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(AptRepository{})) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestURIToFilename(t *testing.T) {
	got, err := uriToFilename("https://download.docker.com/linux/debian")
	if err != nil {
		t.Error(err)
	}

	expected := "download_docker_com_linux_debian.list"
	if expected != got {
		t.Errorf("expected %s but got %s", expected, got)
	}

	_, err = uriToFilename("some_invalid:url")
	if err == nil {
		t.Error("uriToFilename should return an error on invalid URLs")
	}
}

func TestAptRepositoryApply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	a := &AptRepository{
		Repo:  "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
		State: AptRepositoryPresent,
		apt:   m,
	}

	repo := &apt.Repository{
		Enabled:      true,
		SourceRepo:   false,
		Options:      "signed-by=/etc/apt/keyrings/docker.asc",
		URI:          "https://download.docker.com/linux/debian",
		Distribution: "bookworm",
		Components:   "stable",
		Comment:      "",
	}

	m.EXPECT().ParseAPTConfigFolder("/etc/apt").Return(nil, nil)
	m.EXPECT().ParseAPTConfigLine("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable").Return(repo)
	m.EXPECT().AddRepository(repo, "/etc/apt", "download_docker_com_linux_debian.list").Return(nil)
	m.EXPECT().CheckForUpdates().Return("", nil)

	if err := a.Apply(context.Background(), "", false); err != nil {
		t.Error(err)
	}

	pFalse := false
	a.State = AptRepositoryAbsent
	a.UpdateCache = &pFalse

	m.EXPECT().ParseAPTConfigFolder("/etc/apt").Return(apt.RepositoryList{repo}, nil)
	m.EXPECT().ParseAPTConfigLine("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable").Return(repo)
	m.EXPECT().RemoveRepository(repo, "/etc/apt").Return(nil)

	if err := a.Apply(context.Background(), "", false); err != nil {
		t.Error(err)
	}
}
