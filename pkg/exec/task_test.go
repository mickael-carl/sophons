package exec

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestTasksUnmarshalYAML(t *testing.T) {
	ctx := variables.NewContext(context.Background(), variables.Variables{
		"foo":   "bar",
		"input": "world",
	})

	b := []byte(`
- name: "testing"
  ansible.builtin.file:
    path: "{{ foo }}"
    state: "touch"
- someunknownfield: ignored
  ansible.builtin.command:
    cmd: "echo hello"
    stdin: "{{ input }}"
`)

	var got []Task
	if err := tasksUnmarshalYAML(ctx, &got, b); err != nil {
		t.Error(err)
	}

	expected := []Task{
		Task{
			Name: "testing",
			Content: &File{
				Path:  "bar",
				State: FileTouch,
			},
		},
		Task{
			Content: &Command{
				Cmd:   "echo hello",
				Stdin: "world",
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestJinjaStringUnmarshalYAML(t *testing.T) {
	ctx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "bar",
	})

	b := []byte(`"{{ foo }}"`)

	var got jinjaString

	if err := jinjaStringUnmarshalYAML(ctx, &got, b); err != nil {
		t.Error(err)
	}

	expected := "bar"

	if string(got) != expected {
		t.Errorf("got %s but expected %s", got, expected)
	}
}
