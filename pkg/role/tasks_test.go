package role

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/proto"
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
				Command: proto.Command{
					Cmd: "echo 'hello world!'",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmpopts.IgnoreUnexported(exec.Command{}, proto.Command{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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
				Command: proto.Command{
					Cmd: "echo 'hello world!'",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmpopts.IgnoreUnexported(exec.Command{}, proto.Command{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
