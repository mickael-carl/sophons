package exec

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
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
`)
	var got Playbook
	err := yaml.Unmarshal(a, &got)
	if err != nil {
		t.Error(err)
	}

	expected := Playbook{
		Play{
			Hosts: "all",
			Tasks: []Task{
				&File{
					Path:  "/foo",
					State: FileDirectory,
				},
				&File{
					Path:  "/foo/bar",
					State: FileFile,
				},
			},
		},
		Play{
			Hosts: "some-group",
			Tasks: []Task{
				&File{
					Path:  "/foo/bar/baz",
					State: FileTouch,
				},
				&File{
					Path:    "/foo/bar",
					State:   FileDirectory,
					Recurse: true,
					Mode:    0600,
				},
			},
		},
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
