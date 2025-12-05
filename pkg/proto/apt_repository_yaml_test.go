package proto_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestAptRepositoryUnmarshalYAML(t *testing.T) {
	pTrue := true

	tests := []struct {
		name string
		yaml string
		want *proto.AptRepository
	}{
		{
			name: "unmarshal with update_cache",
			yaml: `
repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable"
state: "present"
update_cache: true`,
			want: &proto.AptRepository{
				Repo:        "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
				State:       "present",
				UpdateCache: &pTrue,
			},
		},
		{
			name: "unmarshal with update-cache alias",
			yaml: `
repo: "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable"
state: "present"
update-cache: true`,
			want: &proto.AptRepository{
				Repo:        "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
				State:       "present",
				UpdateCache: &pTrue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got proto.AptRepository
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, &got, cmpopts.IgnoreUnexported(proto.AptRepository{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
