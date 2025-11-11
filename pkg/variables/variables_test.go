package variables

import (
	"context"
	"reflect"
	"testing"
)

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

	if !reflect.DeepEqual(baseVars, expectedMergedVars) {
		t.Errorf("Merged variables mismatch.\nExpected: %+v\nGot: %+v", expectedMergedVars, baseVars)
	}

	// Test merging into an empty map
	emptyVars := Variables{}
	emptyVars.Merge(overrideVars)
	if !reflect.DeepEqual(emptyVars, overrideVars) {
		t.Errorf("Merging into empty map failed.\nExpected: %+v\nGot: %+v", overrideVars, emptyVars)
	}
}

func TestNewContextAndFromContext(t *testing.T) {
	initialVars := Variables{"testKey": "testValue"}
	ctx := context.Background()

	// Test NewContext
	newCtx := NewContext(ctx, initialVars)
	if newCtx == ctx {
		t.Error("NewContext returned the same context as the parent")
	}

	// Test FromContext
	retrievedVars, ok := FromContext(newCtx)
	if !ok {
		t.Error("FromContext failed to retrieve variables")
	}
	if !reflect.DeepEqual(retrievedVars, initialVars) {
		t.Errorf("Retrieved variables mismatch.\nExpected: %+v\nGot: %+v", initialVars, retrievedVars)
	}

	// Test FromContext with a context without variables
	_, ok = FromContext(context.Background())
	if ok {
		t.Error("FromContext returned variables from a context without variables")
	}
}
