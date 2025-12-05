package proto_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestFileUnmarshalYAML(t *testing.T) {
	pFalse := false

	tests := []struct {
		name string
		yaml string
		want *proto.File
	}{
		{
			name: "unmarshal with path",
			yaml: `
path: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`,
			want: &proto.File{
				Path:   "/foo",
				Follow: &pFalse,
				Group:  "bar",
				Mode: &proto.Mode{
					Value: "0644",
				},
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   "file",
			},
		},
		{
			name: "unmarshal with dest alias",
			yaml: `
dest: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`,
			want: &proto.File{
				Path:   "/foo",
				Follow: &pFalse,
				Group:  "bar",
				Mode: &proto.Mode{
					Value: "0644",
				},
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   "file",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got proto.File
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, &got, cmpopts.IgnoreUnexported(proto.File{}, proto.Mode{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
