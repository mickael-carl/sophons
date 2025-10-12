package role

import (
	"context"
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

	got, err := processTasks(context.Background(), fsys, "somerole/tasks")
	if err != nil {
		t.Fatal(err)
	}

	expected := []exec.Task{
		&exec.Command{
			Cmd: "echo 'hello world!'",
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
