package proto_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/types/known/structpb"

	_ "github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestTasksUnmarshalYAML(t *testing.T) {
	b := []byte(`
- name: "testing"
  loop:
    - foo
    - bar
  ansible.builtin.file:
    path: "{{ foo }}"
    state: "touch"
- someunknownfield: ignored
  ansible.builtin.command:
    cmd: "echo hello"
    stdin: "{{ input }}"
`)

	var got []*proto.Task
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	expected := []*proto.Task{
		{
			Name: "testing",
			Loop: &structpb.Value{
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "foo"}},
							{Kind: &structpb.Value_StringValue{StringValue: "bar"}},
						},
					},
				},
			},
			Content: &proto.Task_File{
				File: &proto.File{
					Path:  "{{ foo }}",
					State: "touch",
				},
			},
		},
		{
			Content: &proto.Task_Command{
				Command: &proto.Command{
					Cmd:   "echo hello",
					Stdin: "{{ input }}",
				},
			},
		},
	}

	if diff := cmp.Diff(expected, got, cmpopts.IgnoreUnexported(proto.Task{}, proto.Command{}, proto.File{}, structpb.Value{}, structpb.ListValue{})); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
