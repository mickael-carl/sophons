package exec

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestProcessJinjaTemplates(t *testing.T) {
	type testStruct struct {
		Foo string
		Bar []string
		Baz struct {
			Qux string
		}
	}

	ctx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "bar",
		"qux": "quux",
	})

	ts := &testStruct{
		Foo: "{{ foo }}",
		Bar: []string{"{{ foo }}", "{{ qux }}"},
		Baz: struct {
			Qux string
		}{
			Qux: "{{ qux }}",
		},
	}

	if err := ProcessJinjaTemplates(ctx, ts); err != nil {
		t.Error(err)
	}

	expected := &testStruct{
		Foo: "bar",
		Bar: []string{"bar", "quux"},
		Baz: struct {
			Qux string
		}{
			Qux: "quux",
		},
	}

	if !cmp.Equal(ts, expected) {
		t.Errorf("got %#v but expected %#v", ts, expected)
	}
}

func TestProcessJinjaTemplatesInterface(t *testing.T) {
	type interfaceStruct struct {
		Content interface{}
	}

	type contentStruct struct {
		Foo string
	}

	interfaceCtx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "bar",
	})

	is := &interfaceStruct{
		Content: &contentStruct{
			Foo: "{{ foo }}",
		},
	}

	if err := ProcessJinjaTemplates(interfaceCtx, is); err != nil {
		t.Error(err)
	}

	expectedInterface := &interfaceStruct{
		Content: &contentStruct{
			Foo: "bar",
		},
	}

	if !cmp.Equal(is, expectedInterface) {
		t.Errorf("got %#v but expected %#v", is, expectedInterface)
	}
}

func TestProcessJinjaTemplatesInterfaceExpressionList(t *testing.T) {
	type interfaceStruct struct {
		Content interface{}
	}

	interfaceCtx := variables.NewContext(context.Background(), variables.Variables{
		"foo": []string{"bar", "baz"},
	})

	is := &interfaceStruct{
		Content: "{{ foo }}",
	}

	if err := ProcessJinjaTemplates(interfaceCtx, is); err != nil {
		t.Error(err)
	}

	got := &interfaceStruct{
		Content: getStringSlice(is.Content),
	}

	expectedInterface := &interfaceStruct{
		Content: []string{"bar", "baz"},
	}

	if !cmp.Equal(got, expectedInterface) {
		t.Errorf("got %#v but expected %#v", got, expectedInterface)
	}
}

func TestProcessJinjaTemplatesInterfaceSlice(t *testing.T) {
	type interfaceStruct struct {
		Content interface{}
	}

	interfaceCtx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "bar",
	})

	is := &interfaceStruct{
		Content: []string{"foo", "{{ foo }}"},
	}

	if err := ProcessJinjaTemplates(interfaceCtx, is); err != nil {
		t.Error(err)
	}

	expectedInterface := &interfaceStruct{
		Content: []string{"foo", "bar"},
	}

	if !cmp.Equal(is, expectedInterface) {
		t.Errorf("got %#v but expected %#v", is, expectedInterface)
	}
}

func TestValidateCmdMissingCommand(t *testing.T) {
	pFalse := false

	err := validateCmd([]string{}, "", "", &pFalse)
	if err == nil {
		t.Error("a command with cmd or argv set is not valid")
	}

	if err.Error() != "either cmd or argv need to be specified" {
		t.Error(err)
	}
}

func TestValidateCmdConflictingParameters(t *testing.T) {
	pFalse := false

	err := validateCmd(
		[]string{
			"ls",
			"-l",
		},
		"ls -l",
		"",
		&pFalse,
	)

	if err == nil {
		t.Error("a command with both cmd and argv set is not valid")
	}

	if err.Error() != "cmd and argv can't be both specified at the same time" {
		t.Error(err)
	}
}

func TestValidateCmd(t *testing.T) {
	pFalse := false

	if err := validateCmd(
		[]string{},
		"ls -l",
		"",
		&pFalse,
	); err != nil {
		t.Error(err)
	}
}

func TestValidateCmdStdinMissing(t *testing.T) {
	pTrue := true

	err := validateCmd(
		[]string{},
		"cat",
		"",
		&pTrue,
	)
	if err == nil {
		t.Error("a command with stdin_add_newline and without stdin is not valid")
	}

	if err.Error() != "stdin_add_newline can't be set if stdin is unset" {
		t.Error(err)
	}
}
