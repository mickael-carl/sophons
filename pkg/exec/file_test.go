package exec

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

func TestFileValidate(t *testing.T) {
	tests := []ValidationTestCase[*File]{
		{
			Name: "invalid state",
			Input: &File{
				State: "banana",
			},
			WantErr: true,
			ErrMsg:  "invalid state",
		},
		{
			Name: "missing path",
			Input: &File{
				State: "file",
			},
			WantErr: true,
			ErrMsg:  "path is required",
		},
		{
			Name: "recurse without directory state",
			Input: &File{
				State:   "file",
				Path:    "/foo",
				Recurse: true,
			},
			WantErr: true,
			ErrMsg:  "recurse option requires state to be 'directory'",
		},
		{
			Name: "link without src",
			Input: &File{
				State: "link",
				Path:  "/foo/bar",
			},
			WantErr: true,
			ErrMsg:  "src option is required when state is 'link' or 'hard'",
		},
		{
			Name: "valid file state",
			Input: &File{
				State: "file",
				Path:  "/tmp/test.txt",
			},
			WantErr: false,
		},
		{
			Name: "valid directory state",
			Input: &File{
				State: "directory",
				Path:  "/tmp/testdir",
			},
			WantErr: false,
		},
		{
			Name: "valid touch state",
			Input: &File{
				State: "touch",
				Path:  "/tmp/touched",
			},
			WantErr: false,
		},
		{
			Name: "valid absent state",
			Input: &File{
				State: "absent",
				Path:  "/tmp/removed",
			},
			WantErr: false,
		},
		{
			Name: "hard link without src",
			Input: &File{
				State: "hard",
				Path:  "/foo/hardlink",
			},
			WantErr:     true,
			ErrContains: "src",
		},
		{
			Name: "valid link with src",
			Input: &File{
				State: "link",
				Path:  "/foo/link",
				Src:   "/foo/target",
			},
			WantErr: false,
		},
		{
			Name: "recurse with directory state",
			Input: &File{
				State:   "directory",
				Path:    "/foo",
				Recurse: true,
			},
			WantErr: false,
		},
	}

	RunValidationTests(t, tests)
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
