package role

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"

	"github.com/mickael-carl/sophons/pkg/exec"
)

func TestProcessTasks(t *testing.T) {
	tasksFile := []byte(`
- ansible.builtin.command:
    cmd: "echo 'hello world!'"
`)

	fsys := fstest.MapFS{
		"somerole/tasks/main.yml": &fstest.MapFile{
			Data: tasksFile,
		},
	}

	got, err := processTasks(fsys, "somerole/tasks")
	if err != nil {
		t.Error(err)
	}

	expected := []exec.Task{
		{
			Content: &exec.Command{
				Cmd: "echo 'hello world!'",
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}

	fsys = fstest.MapFS{
		"somerole/tasks/main.yaml": &fstest.MapFile{
			Data: tasksFile,
		},
	}

	got, err = processTasks(fsys, "somerole/tasks")
	if err != nil {
		t.Error(err)
	}

	expected = []exec.Task{
		{
			Content: &exec.Command{
				Cmd: "echo 'hello world!'",
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
