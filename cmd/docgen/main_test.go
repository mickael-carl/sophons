package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileNodeToStructDoc(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		filepath    string
		want        *structDoc
	}{
		{
			name: "with annotations",
			fileContent: "package exec\n" +
				"\n" +
				"//	@meta{\n" +
				"//	  \"deviations\": [\"something is done differently somehow\"]\n" +
				"//	}\n" +
				"type Foo struct {\n" +
				"\tParameter string `sophons:\"implemented\"`\n" +
				"\tLongParamater string `yaml:\"long_parameter\" sophons:\"implemented\"`\n" +
				"\tUnimplementedParameter string `yaml:\"unimplemented_parameter\"`\n" +
				"}\n",
			filepath: "pkg/exec/foo.go",
			want: &structDoc{
				Filename: "foo.go",
				Filepath: "pkg/exec/foo.go",
				Name:     "foo",
				Parameters: map[string]bool{
					"parameter":               true,
					"long_parameter":          true,
					"unimplemented_parameter": false,
				},
				Meta: &docMeta{
					Deviations: []string{"something is done differently somehow"},
				},
			},
		},
		{
			name: "missing annotation",
			fileContent: "package exec\n" +
				"\n" +
				"type someRandomType struct {}",
			filepath: "pkg/exec/test.go",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			fileNode, err := parser.ParseFile(fset, tt.filepath, tt.fileContent, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			got, err := fileNodeToStructDoc(fileNode, tt.filepath)
			if err != nil {
				t.Fatalf("fileNodeToStructDoc() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "camel case",
			input: "IncludeTasks",
			want:  "include_tasks",
		},
		{
			name:  "with acronym",
			input: "SomeHTTPString",
			want:  "some_http_string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase() = %v, want %v", got, tt.want)
			}
		})
	}
}
