package exec

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
			Loop: []interface{}{"foo", "bar"},
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

	if !cmp.Equal(got, expected) {
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

	if !cmp.Equal(originalContent, copiedContent) {
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
