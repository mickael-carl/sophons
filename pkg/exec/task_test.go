package exec

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/arduino/go-apt-client"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestTasksUnmarshalYAML(t *testing.T) {
	b := []byte(`
- name: "testing"
  loop:
    - foo
    - bar
  ansible.builtin.file:
    path: "{{ foo }}"
    state: "touch"
- someunknownfield: ignored
  ansible.builtin.command:
    cmd: "echo hello"
    stdin: "{{ input }}"
`)

	var got []Task
	if err := tasksUnmarshalYAML(&got, b); err != nil {
		t.Error(err)
	}

	expected := []Task{
		{
			Name: "testing",
			Loop: []any{"foo", "bar"},
			Content: &File{
				Path:  "{{ foo }}",
				State: FileTouch,
			},
		},
		{
			Content: &Command{
				Cmd:   "echo hello",
				Stdin: "{{ input }}",
			},
		},
	}

	if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(Command{})) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestDeepCopyContent(t *testing.T) {
	originalContent := &Command{
		Cmd:   "echo hello",
		Chdir: "/tmp",
		Argv:  []string{},
	}

	copiedContent, err := deepCopyContent(originalContent)
	if err != nil {
		t.Fatalf("deepCopyContent failed: %v", err)
	}

	if originalContent == copiedContent {
		t.Error("copied content is the same instance as original")
	}

	if !cmp.Equal(originalContent, copiedContent, cmpopts.IgnoreUnexported(Command{})) {
		t.Errorf("copied content %#v is not equal to original %#v initially", copiedContent, originalContent)
	}

	copiedCommand, ok := copiedContent.(*Command)
	if !ok {
		t.Error("copied content is not of type *Command")
	}
	copiedCommand.Cmd = "echo world"
	copiedCommand.Chdir = "/var"

	if originalContent.Cmd != "echo hello" {
		t.Errorf("original content's Cmd changed to %q, expected %q", originalContent.Cmd, "echo hello")
	}
	if originalContent.Chdir != "/tmp" {
		t.Errorf("original content's Chdir changed to %q, expected %q", originalContent.Chdir, "/tmp")
	}

	if copiedCommand.Cmd != "echo world" {
		t.Errorf("copied content's Cmd is %q, expected %q", copiedCommand.Cmd, "echo world")
	}
	if copiedCommand.Chdir != "/var" {
		t.Errorf("copied content's Chdir is %q, expected %q", copiedCommand.Chdir, "/var")
	}
}

func TestTaskApply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	task := Task{
		Name: "Add docker repo",
		Content: &AptRepository{
			Repo:  "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
			State: AptRepositoryPresent,
		},
		Register: "out",
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

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = variables.NewContext(ctx, variables.Variables{})
	logger := zap.NewNop()
	if err := ExecuteTask(ctx, logger, task, "", false); err != nil {
		t.Error(err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		t.Error("failed to get variables from context")
	}

	got, ok := vars["out"]
	if !ok {
		t.Error("result is not registered in variables")
	}

	expected := map[string]any{
		"changed":      true,
		"failed":       false,
		"msg":          "",
		"rc":           uint64(0),
		"skipped":      false,
		"stderr":       "",
		"stderr_lines": []any{},
		"stdout":       "",
		"stdout_lines": []any{},
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}

func TestTaskApplyLoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	task := Task{
		Name: "install foo and {{ bar }}",
		When: "hello",
		Loop: []string{
			"foo",
			"{{ bar }}",
		},
		Content: &Apt{
			Name: "{{ item }}",
		},
		Register: "out",
	}

	m.EXPECT().ListInstalled().Return(nil, nil)
	m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)
	m.EXPECT().ListInstalled().Return(nil, nil)
	m.EXPECT().Install(&apt.Package{Name: "bar"}).Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})
	ctx = variables.NewContext(ctx, variables.Variables{
		"bar": "bar",
	})

	logger := zap.NewNop()
	if err := ExecuteTask(ctx, logger, task, "", false); err != nil {
		t.Error(err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		t.Error("failed to get variables from context")
	}

	got, ok := vars["out"]
	if !ok {
		t.Error("result is not registered in variables")
	}

	expected := map[string]any{
		"changed": bool(true),
		"failed":  bool(false),
		"msg":     string(""),
		"results": []any{
			map[string]any{
				"cache_update_time": "1970-01-01T00:00:00Z",
				"cache_updated":     false,
				"changed":           true,
				"failed":            false,
				"msg":               "",
				"rc":                uint64(0),
				"skipped":           false,
				"stderr":            "",
				"stderr_lines":      []any{},
				"stdout":            "",
				"stdout_lines":      []any{},
			},
			map[string]any{
				"cache_update_time": "1970-01-01T00:00:00Z",
				"cache_updated":     false,
				"changed":           true,
				"failed":            false,
				"msg":               "",
				"rc":                uint64(0),
				"skipped":           false,
				"stderr":            "",
				"stderr_lines":      []any{},
				"stdout":            "",
				"stdout_lines":      []any{},
			},
		},
		"rc":           uint64(0),
		"skipped":      false,
		"stderr":       "",
		"stderr_lines": []any{},
		"stdout":       "",
		"stdout_lines": []any{},
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("expected %#v but got %#v", expected, got)
	}
}
