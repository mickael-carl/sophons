package exec

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

func TestFileValidate(t *testing.T) {
	tests := []struct {
		name    string
		file    File
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid state",
			file: File{
				State: "banana",
			},
			wantErr: true,
			errMsg:  "invalid state",
		},
		{
			name: "missing path",
			file: File{
				State: "file",
			},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name: "recurse without directory state",
			file: File{
				State:   "file",
				Path:    "/foo",
				Recurse: true,
			},
			wantErr: true,
			errMsg:  "recurse option requires state to be 'directory'",
		},
		{
			name: "link without src",
			file: File{
				State: "link",
				Path:  "/foo/bar",
			},
			wantErr: true,
			errMsg:  "src option is required when state is 'link' or 'hard'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestFileUnmarshalYAML(t *testing.T) {
	pFalse := false

	tests := []struct {
		name string
		yaml string
		want File
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
			want: File{
				Path:    "/foo",
				Follow:  &pFalse,
				Group:   "bar",
				Mode:    "0644",
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   FileFile,
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
			want: File{
				Path:    "/foo",
				Follow:  &pFalse,
				Group:   "bar",
				Mode:    "0644",
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   FileFile,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got File
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
