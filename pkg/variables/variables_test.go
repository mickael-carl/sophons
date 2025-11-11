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
		"key2": map[string]interface{}{
			"subkey1": "subvalue1",
			"subkey2": uint64(123),
		},
	}

	loadedVars, err := LoadFromFile(testFilePath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if !cmp.Equal(loadedVars, expectedVars) {
		t.Errorf("expected %#v but got %#v", expectedVars, loadedVars)
	}
}

func TestLoadFromInexistantFile(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := LoadFromFile(filepath.Join(tmpDir, "non_existent.yaml")); err == nil {
		t.Error("LoadFromFile did not return an error for a non-existent file")
	}
}

func TestMerge(t *testing.T) {
	baseVars := Variables{
		"key1": "value1",
		"key2": "value2",
		"key3": map[string]interface{}{
			"subkey1": "subvalue1",
		},
	}

	overrideVars := Variables{
		"key2": "new_value2",
		"key3": map[string]interface{}{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}

	expectedMergedVars := Variables{
		"key1": "value1",
		"key2": "new_value2",
		"key3": map[string]interface{}{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}

	baseVars.Merge(overrideVars)

	if !cmp.Equal(baseVars, expectedMergedVars) {
		t.Errorf("expected %#v but got %#v", expectedMergedVars, baseVars)
	}
}

func TestMergeEmpty(t *testing.T) {
	emptyVars := Variables{}
	overrideVars := Variables{
		"key2": "new_value2",
		"key3": map[string]interface{}{
			"subkey2": "subvalue2",
		},
		"key4": "value4",
	}
	emptyVars.Merge(overrideVars)

	if !cmp.Equal(emptyVars, overrideVars) {
		t.Errorf("expected %#v but got %#v", overrideVars, emptyVars)
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
	if !cmp.Equal(retrievedVars, initialVars) {
		t.Errorf("expected %#v but got %#v", initialVars, retrievedVars)
	}

	_, ok = FromContext(context.Background())
	if ok {
		t.Error("FromContext returned variables from a context without variables")
	}
}
