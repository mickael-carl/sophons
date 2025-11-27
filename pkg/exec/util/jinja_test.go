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

func TestJinjaProcessWhen(t *testing.T) {
	tests := []struct {
		name     string
		when     string
		vars     variables.Variables
		expected bool
		wantErr  bool
	}{
		{
			name:     "empty when clause always true",
			when:     "",
			vars:     variables.Variables{},
			expected: true,
		},
		{
			name:     "true boolean string",
			when:     "foo",
			vars:     variables.Variables{"foo": true},
			expected: true,
		},
		{
			name:     "True string literal",
			when:     "true",
			vars:     variables.Variables{},
			expected: true,
		},
		{
			name:     "Capital True string literal",
			when:     "True",
			vars:     variables.Variables{},
			expected: true,
		},
		{
			name:     "false boolean string",
			when:     "bar",
			vars:     variables.Variables{"bar": false},
			expected: false,
		},
		{
			name:     "False string literal",
			when:     "false",
			vars:     variables.Variables{},
			expected: false,
		},
		{
			name:     "Capital False string literal",
			when:     "False",
			vars:     variables.Variables{},
			expected: false,
		},
		{
			name:     "non-zero number is true",
			when:     "count",
			vars:     variables.Variables{"count": 5},
			expected: true,
		},
		{
			name:     "zero is false",
			when:     "count",
			vars:     variables.Variables{"count": 0},
			expected: false,
		},
		{
			name:     "negative number is true",
			when:     "count",
			vars:     variables.Variables{"count": -1},
			expected: true,
		},
		{
			name:     "expression evaluation - true",
			when:     "count > 3",
			vars:     variables.Variables{"count": 5},
			expected: true,
		},
		{
			name:     "expression evaluation - false",
			when:     "count < 3",
			vars:     variables.Variables{"count": 5},
			expected: false,
		},
		{
			name:     "context without variables",
			when:     "true",
			vars:     nil, // Will trigger the "!ok" path
			expected: true,
		},
		{
			name:    "invalid template",
			when:    "{{ unclosed",
			vars:    variables.Variables{},
			wantErr: true,
		},
		{
			name:     "string that doesn't match true/false patterns defaults to false",
			when:     "'some random string'",
			vars:     variables.Variables{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.vars != nil {
				ctx = variables.NewContext(ctx, tt.vars)
			}

			got, err := JinjaProcessWhen(ctx, tt.when)
			if (err != nil) != tt.wantErr {
				t.Errorf("JinjaProcessWhen() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("JinjaProcessWhen() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProcessJinjaTemplatesPointer(t *testing.T) {
	type nestedStruct struct {
		Value string
	}

	type pointerStruct struct {
		PtrToStruct *nestedStruct
		PtrToString *string
		PtrToSlice  *[]string
		NilPtr      *nestedStruct
	}

	str := "{{ foo }}"
	slice := []string{"{{ foo }}", "bar"}

	ctx := variables.NewContext(context.Background(), variables.Variables{
		"foo": "baz",
	})

	ps := &pointerStruct{
		PtrToStruct: &nestedStruct{Value: "{{ foo }}"},
		PtrToString: &str,
		PtrToSlice:  &slice,
		NilPtr:      nil,
	}

	if err := ProcessJinjaTemplates(ctx, ps); err != nil {
		t.Fatalf("ProcessJinjaTemplates() error = %v", err)
	}

	expectedStr := "baz"
	if *ps.PtrToString != expectedStr {
		t.Errorf("PtrToString = %v, want %v", *ps.PtrToString, expectedStr)
	}

	if ps.PtrToStruct.Value != "baz" {
		t.Errorf("PtrToStruct.Value = %v, want baz", ps.PtrToStruct.Value)
	}

	expectedSlice := []string{"baz", "bar"}
	if diff := cmp.Diff(expectedSlice, *ps.PtrToSlice); diff != "" {
		t.Errorf("PtrToSlice mismatch (-want +got):\n%s", diff)
	}

	if ps.NilPtr != nil {
		t.Errorf("NilPtr should remain nil")
	}
}
