package role

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestProcessVars(t *testing.T) {
	ignoredVarsFile := []byte(`
hello: "ignored!"
fruit: "banana"
`)

	mainVarsFile := []byte(`
hello: "world!"
"true": true
`)

	fsys := fstest.MapFS{
		"somerole/defaults/foo.yml": &fstest.MapFile{
			Data: ignoredVarsFile,
		},
		"somerole/defaults/main.yml": &fstest.MapFile{
			Data: mainVarsFile,
		},
	}

	got, err := processVars(fsys, "somerole/defaults")
	if err != nil {
		t.Error(err)
	}

	expected := variables.Variables{
		"hello": "world!",
		"true":  true,
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessVarsMainFile(t *testing.T) {
	mainFile := []byte(`
hello: "world!"
main: "is a valid variables file too"
`)

	fsys := fstest.MapFS{
		"somerole/variables/main": &fstest.MapFile{
			Data: mainFile,
		},
	}

	got, err := processVars(fsys, "somerole/variables")
	if err != nil {
		t.Error(err)
	}

	expected := variables.Variables{
		"hello": "world!",
		"main":  "is a valid variables file too",
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessVarsMainDirectory(t *testing.T) {
	fooVarsFile := []byte(`
foo: "foo"
fruit: "banana"
`)

	barVarsFile := []byte(`
bar: "bar"
"true": true
`)

	fsys := fstest.MapFS{
		"somerole/defaults/main/foo.yml": &fstest.MapFile{
			Data: fooVarsFile,
		},
		"somerole/defaults/main/bar.yml": &fstest.MapFile{
			Data: barVarsFile,
		},
	}

	got, err := processVars(fsys, "somerole/defaults")
	if err != nil {
		t.Error(err)
	}

	expected := variables.Variables{
		"foo":   "foo",
		"fruit": "banana",
		"bar":   "bar",
		"true":  true,
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestProcessVarsOverride(t *testing.T) {
	varsFile := []byte(`
"hello": "world!"
"answer": 42
`)

	nestedVarsFile := []byte(`
"hello": "region!"
"false": false
`)

	fsys := fstest.MapFS{
		"somerole/variables/main/foo.yml": &fstest.MapFile{
			Data: varsFile,
		},
		"somerole/variables/main/somedir/region.yml": &fstest.MapFile{
			Data: nestedVarsFile,
		},
	}

	got, err := processVars(fsys, "somerole/variables")
	if err != nil {
		t.Error(err)
	}

	expected := variables.Variables{
		"hello":  "region!",
		"answer": uint64(42),
		"false":  false,
	}

	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
