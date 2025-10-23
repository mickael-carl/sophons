package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFileNodeToStructDoc(t *testing.T) {
	fileContent := "package exec\n" +
		"\n" +
		"//	@meta{\n" +
		"//	  \"deviations\": [\"something is done differently somehow\"]\n" +
		"//	}\n" +
		"type Foo struct {\n" +
		"\tParameter string `sophons:\"implemented\"`\n" +
		"\tLongParamater string `yaml:\"long_parameter\" sophons:\"implemented\"`\n" +
		"\tUnimplementedParameter string `yaml:\"unimplemented_parameter\"`\n" +
		"}\n"

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, "pkg/exec/foo.go", fileContent, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	got, err := fileNodeToStructDoc(fileNode, "pkg/exec/foo.go")
	if err != nil {
		t.Fatal(err)
	}

	expected := &structDoc{
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
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestFileNodeToStructDocMissingAnnotation(t *testing.T) {
	fileContent := "package exec\n" +
		"\n" +
		"type someRandomType struct {}"

	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, "pkg/exec/test.go", fileContent, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	got, err := fileNodeToStructDoc(fileNode, "pkg/exec/test.go")
	if err != nil {
		t.Fatal(err)
	}

	expected := (*structDoc)(nil)

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
