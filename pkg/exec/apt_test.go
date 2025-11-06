package exec

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

func TestAptValidateInvalidState(t *testing.T) {
	a := &Apt{
		State: "banana",
	}

	err := a.Validate()
	if err == nil {
		t.Error("banana is not a valid state")
	}

	if err.Error() != "unsupported state: banana" {
		t.Error(err)
	}
}

func TestAptValidateInvalidUpgrade(t *testing.T) {
	a := &Apt{
		Upgrade: "banana",
	}

	err := a.Validate()
	if err == nil {
		t.Error("banana is not a valid upgrade")
	}

	if err.Error() != "unsupported upgrade mode: banana" {
		t.Error(err)
	}
}

func TestAptValidate(t *testing.T) {
	cacheValidTime := uint64(360)
	pTrue := true
	a := &Apt{
		Name: []string{
			"curl",
		},
		CacheValidTime: &cacheValidTime,
		UpdateCache:    &pTrue,
		Upgrade:        "dist",
	}

	if err := a.Validate(); err != nil {
		t.Error(err)
	}
}

func TestAptUnmarshalYAML(t *testing.T) {
	b := []byte(`
clean: false
name:
  - "foo"
  - "bar"
state: "present"
update_cache: true
upgrade: "full"`)

	var got Apt
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pTrue := true
	expected := Apt{
		Clean: false,
		Name: []string{
			"foo",
			"bar",
		},
		State:       string(AptPresent),
		UpdateCache: &pTrue,
		Upgrade:     AptUpgradeFull,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestAptUnmarshalYAMLAliases(t *testing.T) {
	b := []byte(`
clean: false
package:
  - "foo"
  - "bar"
state: "present"
update-cache: true
upgrade: "full"`)

	var got Apt
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Error(err)
	}

	pTrue := true
	expected := Apt{
		Clean: false,
		Name: []string{
			"foo",
			"bar",
		},
		State:       string(AptPresent),
		UpdateCache: &pTrue,
		Upgrade:     AptUpgradeFull,
	}

	if !cmp.Equal(got, expected) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}
