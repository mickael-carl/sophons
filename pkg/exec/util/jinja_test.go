package util

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
		Content any
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
		Content any
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
		Content: GetStringSlice(is.Content),
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
		Content any
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
