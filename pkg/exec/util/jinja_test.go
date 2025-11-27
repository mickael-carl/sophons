package util

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"

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

	if diff := cmp.Diff(expected, ts); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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

	if diff := cmp.Diff(expectedInterface, is); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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

	if diff := cmp.Diff(expectedInterface, got); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
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

	if diff := cmp.Diff(expectedInterface, is); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderJinjaStringToSlice(t *testing.T) {
	vars := variables.Variables{
		"package_list":  []any{"git", "man-db"},
		"postgres_type": "client",
	}
	varsCtx := gonjaexec.NewContext(vars)

	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "plain string",
			input:    "htop",
			expected: []string{"htop"},
		},
		{
			name:     "template to list",
			input:    "{{ package_list }}",
			expected: []string{"git", "man-db"},
		},
		{
			name:     "template in string",
			input:    "postgresql-{{ postgres_type }}",
			expected: []string{"postgresql-client"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderJinjaStringToSlice(tt.input, varsCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderJinjaStringToSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("renderJinjaStringToSlice() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
