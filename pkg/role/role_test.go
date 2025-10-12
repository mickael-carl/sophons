package role

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
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

	got, ok, err := maybeRole(context.Background(), fsys, "somerole")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected to find a role but didn't")
	}

	expected := Role{
		Defaults: inventory.Variables{
			"fruit": "banana",
			"true":  true,
		},
		Variables: inventory.Variables{
			"hello":  "world!",
			"answer": uint64(42),
		},
		Tasks: []exec.Task{
			&exec.File{
				Path:  "/foo",
				State: exec.FileTouch,
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
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

	got, ok, err := maybeRole(context.Background(), fsys, "somerole")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected to find a role but didn't")
	}

	expected := Role{
		Defaults: inventory.Variables{
			"fruit":  "banana",
			"answer": uint64(41),
		},
		Variables: inventory.Variables{
			"hello":  "world!",
			"answer": uint64(42),
		},
		Tasks: []exec.Task{
			&exec.File{
				Path:  "/hello/world!",
				State: exec.FileTouch,
			},
			&exec.File{
				Path:  "/answer/42",
				State: exec.FileTouch,
			},
			&exec.File{
				Path:  "/fruit/banana",
				State: exec.FileTouch,
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
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

	_, ok, err := maybeRole(context.Background(), fsys, "somerole")
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("got a role but expected none")
	}
}

func TestDiscoverRoleMinimal(t *testing.T) {
	file := []byte(`
This is a very minimal role.
`)

	fsys := fstest.MapFS{
		"somerole/tasks/README.md": &fstest.MapFile{
			Data: file,
		},
	}

	got, ok, err := maybeRole(context.Background(), fsys, "somerole")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected a role but didn't find one")
	}

	expected := Role{
		Defaults:  inventory.Variables(nil),
		Variables: inventory.Variables(nil),
		Tasks:     []exec.Task{},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
