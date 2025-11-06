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
		{
			Name: "testing",
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
