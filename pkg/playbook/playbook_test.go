package playbook

import (
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestPlaybookUnmarshalYAML(t *testing.T) {
	a := []byte(`
 - hosts: all
   tasks:
     - ansible.builtin.file:
         path: /foo
         state: directory
     - file:
         path: /foo/bar
         state: file
 - hosts: some-group
   tasks:
     - ansible.builtin.file:
         path: /foo/bar/baz
         state: touch
     - ansible.builtin.file:
         path: /foo/bar
         state: directory
         recurse: true
         mode: 0600
 - hosts: jinja-test
   tasks:
     - ansible.builtin.file:
         path: "{{ filepath }}"
         state: touch
     - ansible.builtin.command:
         cmd: "dd of=/tmp/hello"
         stdin: "{{ hello }}"
         stdin_add_newline: true
`)
	vars := variables.Variables{
		"filepath": "/banana",
		"hello":    "hello world!",
	}
	ctx := context.WithValue(context.Background(), "vars", vars)

	var got Playbook
	err := yaml.UnmarshalContext(ctx, a, &got)
	if err != nil {
		t.Error(err)
	}

	pTrue := true

	expected := Playbook{
		Play{
			Hosts: "all",
			Tasks: []exec.Task{
				&exec.File{
					Path:  "/foo",
					State: exec.FileDirectory,
				},
				&exec.File{
					Path:  "/foo/bar",
					State: exec.FileFile,
				},
			},
		},
		Play{
			Hosts: "some-group",
			Tasks: []exec.Task{
				&exec.File{
					Path:  "/foo/bar/baz",
					State: exec.FileTouch,
				},
				&exec.File{
					Path:    "/foo/bar",
					State:   exec.FileDirectory,
					Recurse: true,
					Mode:    0600,
				},
			},
		},
		Play{
			Hosts: "jinja-test",
			Tasks: []exec.Task{
				&exec.File{
					Path:  "/banana",
					State: exec.FileTouch,
				},
				&exec.Command{
					Cmd:             "dd of=/tmp/hello",
					Stdin:           "hello world!",
					StdinAddNewline: &pTrue,
				},
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
