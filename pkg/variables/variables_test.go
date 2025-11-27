package variables

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	testYAMLContent := `
key1: value1
key2:
  subkey1: subvalue1
  subkey2: 123`

	testFilePath := filepath.Join(tmpDir, "test_vars.yaml")
	err := os.WriteFile(testFilePath, []byte(testYAMLContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write test YAML file: %v", err)
	}

	expectedVars := Variables{
		"key1": "value1",
		"key2": map[string]any{
			"subkey1": "subvalue1",
			"subkey2": uint64(123),
		},
	}

	loadedVars, err := LoadFromFile(testFilePath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if diff := cmp.Diff(expectedVars, loadedVars); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestLoadFromInexistantFile(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := LoadFromFile(filepath.Join(tmpDir, "non_existent.yaml")); err == nil {
		t.Error("LoadFromFile did not return an error for a non-existent file")
	}
}

func TestLoadFromInvalidYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()

	testContent := `
[something]
key1 = "foo"

[something.subkey]
subkey1 = subvalue1
subkey2 = 123`

	testFilePath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(testFilePath, []byte(testContent), 0o644)
	if err != nil {
		t.Fatalf("failed to write test YAML file: %v", err)
	}

	if _, err := LoadFromFile(filepath.Join(tmpDir, "invalid.yaml")); err == nil {
		t.Error("LoadFromFile did not return an error for an invalid YAML file")
	}
}

func TestMerge(t *testing.T) {
	baseVars := Variables{
		"key1": "value1",
		"key2": "value2",
		"key3": map[string]any{
			"subkey1": "subvalue1",
		},
	}

	overrideVars := Variables{
		"key2": "new_value2",
		"key3": map[string]any{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}

	expectedMergedVars := Variables{
		"key1": "value1",
		"key2": "new_value2",
		"key3": map[string]any{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}

	baseVars.Merge(overrideVars)

	if diff := cmp.Diff(expectedMergedVars, baseVars); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestMergeEmpty(t *testing.T) {
	emptyVars := Variables{}
	overrideVars := Variables{
		"key2": "new_value2",
		"key3": map[string]any{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}
	emptyVars.Merge(overrideVars)

	if diff := cmp.Diff(overrideVars, emptyVars); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

func TestNewContextAndFromContext(t *testing.T) {
	initialVars := Variables{"testKey": "testValue"}
	ctx := context.Background()

	newCtx := NewContext(ctx, initialVars)
	if newCtx == ctx {
		t.Error("NewContext returned the same context as the parent")
	}

	retrievedVars, ok := FromContext(newCtx)
	if !ok {
		t.Error("FromContext failed to retrieve variables")
	}
	if diff := cmp.Diff(initialVars, retrievedVars); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	_, ok = FromContext(context.Background())
	if ok {
		t.Error("FromContext returned variables from a context without variables")
	}
}
