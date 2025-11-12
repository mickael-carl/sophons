package exec

import (
	"context"
	"testing"

	"github.com/arduino/go-apt-client"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/mock/gomock"
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
		State:       AptPresent,
		UpdateCache: &pTrue,
		Upgrade:     AptUpgradeFull,
	}

	if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(Apt{})) {
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
		State:       AptPresent,
		UpdateCache: &pTrue,
		Upgrade:     AptUpgradeFull,
	}

	if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(Apt{})) {
		t.Errorf("got %#v but expected %#v", got, expected)
	}
}

func TestAptApply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockaptClient(ctrl)
	a := &Apt{
		Name: []string{
			"foo",
			"bar",
		},
		State: AptPresent,
		apt:   m,
	}

	m.EXPECT().ListInstalled().Return([]*apt.Package{
		{Name: "foo"},
	}, nil)
	m.EXPECT().Install(&apt.Package{Name: "bar"}).Return("", nil)

	if err := a.Apply(context.Background(), "", false); err != nil {
		t.Error(err)
	}
}

func TestAptApplyLatest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := NewMockaptClient(ctrl)
	a := &Apt{
		Name: []string{
			"foo",
		},
		State: AptLatest,
		apt:   m,
	}

	m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)

	if err := a.Apply(context.Background(), "", false); err != nil {
		t.Error(err)
	}
}
