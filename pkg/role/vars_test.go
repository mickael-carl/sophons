package role

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestProcessVars(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
		path string
		want variables.Variables
	}{
		{
			name: "main.yml file with other files ignored",
			fsys: fstest.MapFS{
				"somerole/defaults/foo.yml": &fstest.MapFile{
					Data: []byte(`
hello: "ignored!"
fruit: "banana"
`),
				},
				"somerole/defaults/main.yml": &fstest.MapFile{
					Data: []byte(`
hello: "world!"
"true": true
`),
				},
			},
			path: "somerole/defaults",
			want: variables.Variables{
				"hello": "world!",
				"true":  true,
			},
		},
		{
			name: "main file without extension",
			fsys: fstest.MapFS{
				"somerole/variables/main": &fstest.MapFile{
					Data: []byte(`
hello: "world!"
main: "is a valid variables file too"
`),
				},
			},
			path: "somerole/variables",
			want: variables.Variables{
				"hello": "world!",
				"main":  "is a valid variables file too",
			},
		},
		{
			name: "main directory with multiple files",
			fsys: fstest.MapFS{
				"somerole/defaults/main/foo.yml": &fstest.MapFile{
					Data: []byte(`
foo: "foo"
fruit: "banana"
`),
				},
				"somerole/defaults/main/bar.yml": &fstest.MapFile{
					Data: []byte(`
bar: "bar"
"true": true
`),
				},
			},
			path: "somerole/defaults",
			want: variables.Variables{
				"foo":   "foo",
				"fruit": "banana",
				"bar":   "bar",
				"true":  true,
			},
		},
		{
			name: "nested directory with override",
			fsys: fstest.MapFS{
				"somerole/variables/main/foo.yml": &fstest.MapFile{
					Data: []byte(`
"hello": "world!"
"answer": 42
`),
				},
				"somerole/variables/main/somedir/region.yml": &fstest.MapFile{
					Data: []byte(`
"hello": "region!"
"false": false
`),
				},
			},
			path: "somerole/variables",
			want: variables.Variables{
				"hello":  "region!",
				"answer": uint64(42),
				"false":  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processVars(tt.fsys, tt.path)
			if err != nil {
				t.Errorf("processVars() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
