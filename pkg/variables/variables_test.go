package variables

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		fileName    string
		want        Variables
		wantErr     bool
	}{
		{
			name: "valid YAML file",
			fileContent: `
key1: value1
key2:
  subkey1: subvalue1
  subkey2: 123`,
			fileName: "test_vars.yaml",
			want: Variables{
				"key1": "value1",
				"key2": map[string]any{
					"subkey1": "subvalue1",
					"subkey2": uint64(123),
				},
			},
			wantErr: false,
		},
		{
			name:        "non-existent file",
			fileContent: "",
			fileName:    "non_existent.yaml",
			wantErr:     true,
		},
		{
			name: "invalid YAML file",
			fileContent: `
[something]
key1 = "foo"

[something.subkey]
subkey1 = subvalue1
subkey2 = 123`,
			fileName: "invalid.yaml",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFilePath := filepath.Join(tmpDir, tt.fileName)

			// Write file only if content is provided
			if tt.fileContent != "" {
				err := os.WriteFile(testFilePath, []byte(tt.fileContent), 0o644)
				if err != nil {
					t.Fatalf("failed to write test YAML file: %v", err)
				}
			}

			loadedVars, err := LoadFromFile(testFilePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, loadedVars); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     Variables
		override Variables
		want     Variables
	}{
		{
			name: "merge with overrides",
			base: Variables{
				"key1": "value1",
				"key2": "value2",
				"key3": map[string]any{
					"subkey1": "subvalue1",
				},
			},
			override: Variables{
				"key2": "new_value2",
				"key3": map[string]any{
					"subkey2": "subvalue2",
				},
				"key4": "value4",
			},
			want: Variables{
				"key1": "value1",
				"key2": "new_value2",
				"key3": map[string]any{
					"subkey2": "subvalue2",
				},
				"key4": "value4",
			},
		},
		{
			name: "merge into empty base",
			base: Variables{},
			override: Variables{
				"key2": "new_value2",
				"key3": map[string]any{
					"subkey2": "subvalue2",
				},
				"key4": "value4",
			},
			want: Variables{
				"key2": "new_value2",
				"key3": map[string]any{
					"subkey2": "subvalue2",
				},
				"key4": "value4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.override)
			if diff := cmp.Diff(tt.want, tt.base); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewContextAndFromContext(t *testing.T) {
	tests := []struct {
		name        string
		initialVars Variables
		useNewCtx   bool
		wantOk      bool
		wantSameCtx bool
		wantVars    Variables
	}{
		{
			name:        "new context with variables",
			initialVars: Variables{"testKey": "testValue"},
			useNewCtx:   true,
			wantOk:      true,
			wantSameCtx: false,
			wantVars:    Variables{"testKey": "testValue"},
		},
		{
			name:        "context without variables",
			initialVars: nil,
			useNewCtx:   false,
			wantOk:      false,
			wantSameCtx: true,
			wantVars:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var testCtx context.Context

			if tt.useNewCtx {
				testCtx = NewContext(ctx, tt.initialVars)
				if (testCtx == ctx) == !tt.wantSameCtx {
					if tt.wantSameCtx {
						t.Error("NewContext returned a different context than the parent")
					} else {
						t.Error("NewContext returned the same context as the parent")
					}
				}
			} else {
				testCtx = ctx
			}

			retrievedVars, ok := FromContext(testCtx)
			if ok != tt.wantOk {
				t.Errorf("FromContext() ok = %v, want %v", ok, tt.wantOk)
			}
			if tt.wantOk {
				if diff := cmp.Diff(tt.wantVars, retrievedVars); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
