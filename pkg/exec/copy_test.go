package exec

import "testing"

func TestCopyValidateAbsPath(t *testing.T) {
	c := &Copy{
		Src:  "/etc/shadow",
		Dest: "/hacking-passwords",
	}

	err := c.Validate()
	if err == nil {
		t.Error("copying from an absolute path from the control node is not supported and should fail")
	}

	if err.Error() != "copying from an absolute path without remote_src is not supported" {
		t.Error(err)
	}
}

func TestCopyValidateAbsPathRemote(t *testing.T) {
	c := &Copy{
		Src:       "/tmp/someconfig",
		Dest:      "/etc/someconfig",
		RemoteSrc: true,
	}

	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}

func TestCopyValidateMissingSrc(t *testing.T) {
	c := &Copy{
		Dest: "/something",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a copy without src or content set is not valid")
	}

	if err.Error() != "either src or content need to be specified" {
		t.Error(err)
	}
}

func TestCopyValidateContentDestDirectory(t *testing.T) {
	c := &Copy{
		Content: "hello world",
		Dest:    "/some/directory/",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a copy task with content set and dest being a directory is invalid")
	}

	if err.Error() != "can't use content when dest is a directory" {
		t.Error(err)
	}
}

func TestCopyValidateSrcContentSet(t *testing.T) {
	c := &Copy{
		Src:     "somefile",
		Content: "hello world!",
		Dest:    "/somefile",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a copy task with both content and src set is invalid")
	}

	if err.Error() != "src and content can't both be specified" {
		t.Error(err)
	}
}
