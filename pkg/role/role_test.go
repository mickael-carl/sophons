package role

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestMaybeRole(t *testing.T) {
	vars := []byte(`
hello: "world!"
answer: 42
`)

	defaults := []byte(`
fruit: "banana"
"true": true
`)

	tasks := []byte(`
- ansible.builtin.file:
    path: /foo
    state: touch
`)

	fsys := fstest.MapFS{
		"somerole/vars/main.yml": &fstest.MapFile{
			Data: vars,
		},
		"somerole/defaults/main/main.yml": &fstest.MapFile{
			Data: defaults,
		},
		"somerole/tasks/main": &fstest.MapFile{
			Data: tasks,
		},
	}

	got, ok, err := maybeRole(fsys, "somerole")
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("expected to find a role but didn't")
	}

	expected := Role{
		defaults: variables.Variables{
			"fruit": "banana",
			"true":  true,
		},
		vars: variables.Variables{
			"hello":  "world!",
			"answer": uint64(42),
		},
		tasks: []*proto.Task{
			{
				Content: &proto.Task_File{
					File: &proto.File{
						Path:  "/foo",
						State: exec.FileTouch,
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmp.AllowUnexported(Role{}), cmpopts.IgnoreUnexported(proto.Task{}, proto.File{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMaybeRoleVariables(t *testing.T) {
	vars := []byte(`
hello: "world!"
answer: 42
`)

	defaults := []byte(`
answer: 41
fruit: "banana"
`)

	tasks := []byte(`
- ansible.builtin.file:
    path: "/hello/{{ hello }}"
    state: touch
- ansible.builtin.file:
    path: "/answer/{{ answer }}"
    state: touch
- ansible.builtin.file:
    path: "/fruit/{{ fruit }}"
    state: touch
`)

	fsys := fstest.MapFS{
		"somerole/vars/main.yml": &fstest.MapFile{
			Data: vars,
		},
		"somerole/defaults/main/main.yml": &fstest.MapFile{
			Data: defaults,
		},
		"somerole/tasks/main": &fstest.MapFile{
			Data: tasks,
		},
	}

	got, ok, err := maybeRole(fsys, "somerole")
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("expected to find a role but didn't")
	}

	expected := Role{
		defaults: variables.Variables{
			"answer": uint64(41),
			"fruit":  "banana",
		},
		vars: variables.Variables{
			"hello":  "world!",
			"answer": uint64(42),
		},
		tasks: []*proto.Task{
			{
				Content: &proto.Task_File{
					File: &proto.File{
						Path:  "/hello/{{ hello }}",
						State: exec.FileTouch,
					},
				},
			},
			{
				Content: &proto.Task_File{
					File: &proto.File{
						Path:  "/answer/{{ answer }}",
						State: exec.FileTouch,
					},
				},
			},
			{
				Content: &proto.Task_File{
					File: &proto.File{
						Path:  "/fruit/{{ fruit }}",
						State: exec.FileTouch,
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmp.AllowUnexported(Role{}), cmpopts.IgnoreUnexported(proto.Task{}, proto.File{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscoverRoleNotOK(t *testing.T) {
	randomFile := []byte(`
# Definitely not a role

This is just some test content.
`)

	fsys := fstest.MapFS{
		"somerole/README.md": &fstest.MapFile{
			Data: randomFile,
		},
	}

	_, ok, err := maybeRole(fsys, "somerole")
	if err != nil {
		t.Error(err)
	}

	if ok {
		t.Error("got a role but expected none")
	}
}

func TestMaybeRoleMinimal(t *testing.T) {
	file := []byte(`
This is a very minimal role.
`)

	fsys := fstest.MapFS{
		"somerole/tasks/README.md": &fstest.MapFile{
			Data: file,
		},
	}

	got, ok, err := maybeRole(fsys, "somerole")
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Error("expected a role but didn't find one")
	}

	expected := Role{
		defaults: nil,
		vars:     nil,
		tasks:    []*proto.Task{},
	}

	if diff := cmp.Diff(expected, got, cmp.AllowUnexported(Role{}), cmpopts.IgnoreUnexported(proto.Task{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestDiscoverRole(t *testing.T) {
	tasks1 := []byte(`
- name: "Hello World!"
  ansible.builtin.file:
    path: "/hello"
    state: "touch"`)

	tasks2 := []byte(`
- name: "The Answer"
  ansible.builtin.shell:
    cmd: "echo {{ the_answer }}"`)

	vars := []byte(`the_answer: 42`)

	handler := []byte(`
    - name: Restart myservice
      ansible.builtin.service:
        name: myservice
        state: restarted`)

	fsys := fstest.MapFS{
		"hello/tasks/main.yml":    &fstest.MapFile{Data: tasks1},
		"answer/tasks/main.yaml":  &fstest.MapFile{Data: tasks2},
		"answer/vars/main.yml":    &fstest.MapFile{Data: vars},
		"other/handlers/main.yml": &fstest.MapFile{Data: handler},
	}

	got, err := DiscoverRoles(fsys)
	if err != nil {
		t.Error(err)
	}

	expected := map[string]Role{
		"hello": {
			tasks: []*proto.Task{
				{
					Name: "Hello World!",
					Content: &proto.Task_File{
						File: &proto.File{
							Path:  "/hello",
							State: exec.FileTouch,
						},
					},
				},
			},
		},
		"answer": {
			vars: variables.Variables{
				"the_answer": uint64(42),
			},
			tasks: []*proto.Task{
				{
					Name: "The Answer",
					Content: &proto.Task_Shell{
						Shell: &proto.Shell{
							Cmd: "echo {{ the_answer }}",
						},
					},
				},
			},
		},
		"other": {},
	}

	if diff := cmp.Diff(expected, got, cmp.AllowUnexported(Role{}), cmpopts.IgnoreUnexported(proto.Task{}, proto.File{}, proto.Shell{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRoleApply(t *testing.T) {
	role := Role{
		defaults: variables.Variables{
			"hello": "world!",
			"foos": []string{
				"foo",
				"bar",
				"baz",
			},
		},
		vars: variables.Variables{
			"answer": uint64(42),
		},
		tasks: []*proto.Task{
			{
				Name: "testing1",
				Content: &proto.Task_Shell{
					Shell: &proto.Shell{
						Cmd: "echo {{ hello }}",
					},
				},
			},
			{
				Name: "testing2",
				Loop: func() *structpb.Value { v, _ := structpb.NewValue("{{ foos }}"); return v }(),
				Content: &proto.Task_Shell{
					Shell: &proto.Shell{
						Cmd: "echo {{ item }}",
					},
				},
			},
			{
				Name: "testing3",
				Content: &proto.Task_Shell{
					Shell: &proto.Shell{
						Cmd: "echo {{ answer }}",
					},
				},
			},
		},
	}

	ctx := variables.NewContext(context.Background(), variables.Variables{
		"hello": "tests!",
	})

	if err := role.Apply(ctx, zap.NewNop(), ""); err != nil {
		t.Error(err)
	}

	// Ensure role variables are available in subsequent plays.
	got, ok := variables.FromContext(ctx)
	if !ok {
		t.Error("failed to get variables from context after roles apply")
	}

	expected := variables.Variables{
		"hello":  "tests!",
		"answer": uint64(42),
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
