package exec

import (
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestFileValidateInvalidState(t *testing.T) {
	f := &File{
		State: "banana",
	}

	err := f.Validate()
	if err == nil {
		t.Error("banana is not a valid state")
	}

	if err.Error() != "invalid state" {
		t.Error(err)
	}
}

func TestFileValidateMissingPath(t *testing.T) {
	f := &File{
		State: "file",
	}

	err := f.Validate()
	if err == nil {
		t.Error("a file without path is not valid")
	}

	if err.Error() != "path is required" {
		t.Error(err)
	}
}

func TestFileValidateRecurseWithoutDirectoryState(t *testing.T) {
	f := &File{
		State:   "file",
		Path:    "/foo",
		Recurse: true,
	}

	err := f.Validate()
	if err == nil {
		t.Error("a file task with recurse set to true without state being 'directory' is invalid")
	}

	if err.Error() != "recurse option requires state to be 'directory'" {
		t.Error(err)
	}
}

func TestFileValidateLinkWithoutSrc(t *testing.T) {
	f := &File{
		State: "link",
		Path:  "/foo/bar",
	}

	err := f.Validate()
	if err == nil {
		t.Error("a link without Src attribute is not valid")
	}

	if err.Error() != "src option is required when state is 'link' or 'hard'" {
		t.Error(err)
	}
}

func TestFileUnmarshalYAML(t *testing.T) {
	b := []byte(`
path: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`)

	var got File
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pFalse := false
	expected := File{
		Path:    "/foo",
		Follow:  &pFalse,
		Group:   "bar",
		Mode:    "0644",
		Owner:   "baz",
		Recurse: false,
		Src:     "/hello",
		State:   FileFile,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestFileUnmarshalYAMLAliases(t *testing.T) {
	b := []byte(`
dest: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`)

	var got File
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pFalse := false
	expected := File{
		Path:    "/foo",
		Follow:  &pFalse,
		Group:   "bar",
		Mode:    "0644",
		Owner:   "baz",
		Recurse: false,
		Src:     "/hello",
		State:   FileFile,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestFileUnmarshalYAMLAliasesVariables(t *testing.T) {
	ctx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "/bar",
	})

	b := []byte(`
name: "{{ foo }}"
state: "touch"`)

	var got File
	if err := yaml.UnmarshalContext(ctx, b, &got); err != nil {
		t.Error(err)
	}

	expected := File{
		Path:  "/bar",
		State: FileTouch,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
