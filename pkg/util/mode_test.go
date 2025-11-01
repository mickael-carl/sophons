package util

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"
)

func TestNewModeFromSpec(t *testing.T) {
	fsys := fstest.MapFS{
		"hello": &fstest.MapFile{
			Mode: 0o000,
		},
		"world": &fstest.MapFile{
			Mode: 0o777,
		},
		"notevil": &fstest.MapFile{
			Mode: 0o444,
		},
	}

	got, err := NewModeFromSpec(fsys, "hello", "u+rw,g=x")
	if err != nil {
		t.Error(err)
	}

	expected := os.FileMode(0o610)
	if got != expected {
		t.Errorf("expected %o but got %o", expected, got)
	}

	got, err = NewModeFromSpec(fsys, "world", "u=x,g-rw,o+w")
	if err != nil {
		t.Error(err)
	}

	expected = os.FileMode(0o117)
	if got != expected {
		t.Errorf("expected %o but got %o", expected, got)
	}

	got, err = NewModeFromSpec(fsys, "notevil", "a=rw")
	if err != nil {
		t.Error(err)
	}

	expected = os.FileMode(0o666)
	if got != expected {
		t.Errorf("expected %o but got %o", expected, got)
	}
}

func TestNewModeFromSpecInvalid(t *testing.T) {
	fsys := fstest.MapFS{
		"hello": &fstest.MapFile{
			Mode: 0o000,
		},
	}

	_, err := NewModeFromSpec(fsys, "hello", "invalid")
	if err == nil {
		t.Error("an invalid spec should throw an error")
	}
	if !errors.Is(err, ErrInvalidMode) {
		t.Error(err)
	}
}
