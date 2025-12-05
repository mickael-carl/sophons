package proto_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestAptUnmarshalYAML(t *testing.T) {
	pTrue := true

	tests := []struct {
		name string
		yaml string
		want *proto.Apt
	}{
		{
			name: "unmarshal with name",
			yaml: `
clean: false
name:
  - "foo"
  - "bar"
state: "present"
update_cache: true
upgrade: "full"`,
			want: &proto.Apt{
				Clean: false,
				Name: &proto.PackageList{
					Items: []string{
						"foo",
						"bar",
					},
				},
				State:       "present",
				UpdateCache: &pTrue,
				Upgrade:     "full",
			},
		},
		{
			name: "unmarshal with package alias",
			yaml: `
clean: false
package:
  - "foo"
  - "bar"
state: "present"
update-cache: true
upgrade: "full"`,
			want: &proto.Apt{
				Clean: false,
				Name: &proto.PackageList{
					Items: []string{
						"foo",
						"bar",
					},
				},
				State:       "present",
				UpdateCache: &pTrue,
				Upgrade:     "full",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got proto.Apt
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, &got, cmpopts.IgnoreUnexported(proto.Apt{}, proto.PackageList{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
