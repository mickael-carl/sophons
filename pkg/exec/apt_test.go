package exec

import "testing"

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
		Name: []jinjaString{
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
