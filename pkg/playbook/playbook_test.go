package playbook

import (
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestPlaybookUnmarshalYAML(t *testing.T) {
	a := []byte(`
 - hosts: all
   vars:
     var1: foo
     var2: bar
   vars_files:
     - myvars.yml
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
	ctx := variables.NewContext(context.Background(), variables.Variables{
		"filepath": "/banana",
		"hello":    "hello world!",
	})

	var got Playbook
	if err := yaml.UnmarshalContext(ctx, a, &got); err != nil {
		t.Error(err)
	}

	pTrue := true

	expected := Playbook{
		Play{
			Hosts: "all",
			Vars: variables.Variables{
				"var1": "foo",
				"var2": "bar",
			},
			VarsFiles: []string{"myvars.yml"},
			Tasks: []exec.Task{
				{
					Content: &exec.File{
						File: proto.File{
							Path:  "/foo",
							State: exec.FileDirectory,
						},
					},
				},
				{
					Content: &exec.File{
						File: proto.File{
							Path:  "/foo/bar",
							State: exec.FileFile,
						},
					},
				},
			},
		},
		Play{
			Hosts: "some-group",
			Tasks: []exec.Task{
				{
					Content: &exec.File{
						File: proto.File{
							Path:  "/foo/bar/baz",
							State: exec.FileTouch,
						},
					},
				},
				{
					Content: &exec.File{
						File: proto.File{
							Path:    "/foo/bar",
							State:   exec.FileDirectory,
							Recurse: true,
							Mode: &proto.Mode{
								Value: "600",
							},
						},
					},
				},
			},
		},
		Play{
			Hosts: "jinja-test",
			Tasks: []exec.Task{
				{
					Content: &exec.File{
						File: proto.File{
							Path:  "{{ filepath }}",
							State: exec.FileTouch,
						},
					},
				},
				{
					Content: &exec.Command{
						Command: proto.Command{
							Cmd:             "dd of=/tmp/hello",
							Stdin:           "{{ hello }}",
							StdinAddNewline: &pTrue,
						},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmpopts.IgnoreUnexported(exec.Command{}, proto.Command{}, proto.File{}, proto.Mode{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
