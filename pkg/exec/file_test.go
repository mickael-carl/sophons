package exec

import "testing"

func TestFileValidateInvalidState(t *testing.T) {
	f := &File{
		State: "banana",
	}

	err := f.Validate()
	if err == nil {
		t.Error("banana is not a valid state")
	}

	if err.Error() != "invalid state" {
		t.Error(err)
	}
}

func TestFileValidateMissingPath(t *testing.T) {
	f := &File{
		State: "file",
	}

	err := f.Validate()
	if err == nil {
		t.Error("a file without path is not valid")
	}

	if err.Error() != "path is required" {
		t.Error(err)
	}
}

func TestFileValidateRecurseWithoutDirectoryState(t *testing.T) {
	f := &File{
		State:   "file",
		Path:    "/foo",
		Recurse: true,
	}

	err := f.Validate()
	if err == nil {
		t.Error("a file task with recurse set to true without state being 'directory' is invalid")
	}

	if err.Error() != "recurse option requires state to be 'directory'" {
		t.Error(err)
	}
}

func TestFileValidateLinkWithoutSrc(t *testing.T) {
	f := &File{
		State: "link",
		Path:  "/foo/bar",
	}

	err := f.Validate()
	if err == nil {
		t.Error("a link without Src attribute is not valid")
	}

	if err.Error() != "src option is required when state is 'link' or 'hard'" {
		t.Error(err)
	}
}
