package exec

import "testing"

func TestTemplateValidateAbsPath(t *testing.T) {
	c := &Template{
		Src:  "/etc/shadow",
		Dest: "/hacking-passwords",
	}

	err := c.Validate()
	if err == nil {
		t.Error("templating from an absolute path from the control node is not supported and should fail")
	}

	if err.Error() != "template from an absolute path is not supported" {
		t.Error(err)
	}
}

func TestTemplateValidateMissingSrc(t *testing.T) {
	c := &Template{
		Dest: "/something",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a template without src set is not valid")
	}

	if err.Error() != "src is required" {
		t.Error(err)
	}
}

func TestTemplateValidateMissingDest(t *testing.T) {
	c := &Template{
		Src: "foo",
	}

	err := c.Validate()
	if err == nil {
		t.Error("a template without dest set is not valid")
	}

	if err.Error() != "dest is required" {
		t.Error(err)
	}
}
