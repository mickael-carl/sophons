package exec

import "testing"

func TestIncludeTasksValidateMissingFile(t *testing.T) {
	it := &IncludeTasks{}

	err := it.Validate()
	if err == nil {
		t.Error("an include_tasks without `file` is not valid")
	}

	if err.Error() != "`file` is required" {
		t.Error(err)
	}
}
