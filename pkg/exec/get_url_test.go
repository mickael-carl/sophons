package exec

import "testing"

func TestGetURLValidateMissingURL(t *testing.T) {
	g := &GetURL{
		Dest: "/somewhere",
	}

	err := g.Validate()
	if err == nil {
		t.Error("url is required")
	}

	if err.Error() != "url is required" {
		t.Error(err)
	}
}

func TestGetURLValidateMissingDest(t *testing.T) {
	g := &GetURL{
		URL: "https://example.com",
	}

	err := g.Validate()
	if err == nil {
		t.Error("dest is required")
	}

	if err.Error() != "dest is required" {
		t.Error(err)
	}
}

func TestGetURLValidateInvalidURL(t *testing.T) {
	g := &GetURL{
		URL:  "foo_bar:baz",
		Dest: "/somewhere",
	}

	err := g.Validate()
	if err == nil {
		t.Error("invalid URLs should be rejected")
	}

	if err.Error() != "invalid URL provided" {
		t.Error(err)
	}
}
