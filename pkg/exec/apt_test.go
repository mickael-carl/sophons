package exec

import (
	"context"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

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
	}

	m.EXPECT().ListInstalled().Return([]*apt.Package{
		{Name: "foo"},
	}, nil)
	m.EXPECT().Install(&apt.Package{Name: "bar"}).Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})

	got, err := a.Apply(ctx, "", false)
	if err != nil {
		t.Error(err)
	}

	expected := &AptResult{
		CommonResult: CommonResult{
			Changed: true,
			Failed:  false,
			Skipped: false,
		},
		CacheUpdated:    false,
		CacheUpdateTime: time.UnixMilli(0),
	}

	if !cmp.Equal(expected, got) {
		t.Errorf("expected %#v but got %#v", expected, got)
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
	}

	m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})

	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}
}

func TestHandleUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	listsFile := fstest.MapFile{
		ModTime: time.Unix(0, 0),
	}

	cacheValidTime := uint64(1)

	a := &Apt{
		Name: []string{
			"foo",
		},
		State:          AptLatest,
		CacheValidTime: &cacheValidTime,
		apt:            m,
		aptFS: fstest.MapFS{
			"lists": &listsFile,
		},
	}

	m.EXPECT().CheckForUpdates().Return("", nil)

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}

	listsFile = fstest.MapFile{
		ModTime: time.Now(),
	}

	cacheValidTime = uint64(1000)

	a = &Apt{
		Name: []string{
			"foo",
		},
		State:          AptLatest,
		CacheValidTime: &cacheValidTime,
		apt:            m,
		aptFS: fstest.MapFS{
			"lists": &listsFile,
		},
	}

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}
}

func TestHandleUpdateMissingLists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	// Non-existing lists file with CacheValidTime set should update the cache.
	cacheValidTime := uint64(1)

	a := &Apt{
		Name: []string{
			"foo",
		},
		State:          AptLatest,
		CacheValidTime: &cacheValidTime,
		apt:            m,
		aptFS:          fstest.MapFS{},
	}

	m.EXPECT().CheckForUpdates().Return("", nil)

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}

	// Non-existing lists file with UpdateCache set should also update the
	// cache.
	updateCache := true
	a = &Apt{
		Name: []string{
			"foo",
		},
		State:       AptLatest,
		UpdateCache: &updateCache,
		apt:         m,
		aptFS:       fstest.MapFS{},
	}

	m.EXPECT().CheckForUpdates().Return("", nil)

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}

	// Finally not lists file and nothing set means we shouldn't update.
	a = &Apt{
		Name: []string{
			"foo",
		},
		State: AptLatest,
		apt:   m,
		aptFS: fstest.MapFS{},
	}

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}
}

func TestHandleUpdateExplicit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	listsFile := fstest.MapFile{}

	updateCache := true

	a := &Apt{
		Name: []string{
			"foo",
		},
		State:       AptLatest,
		UpdateCache: &updateCache,
		apt:         m,
		aptFS: fstest.MapFS{
			"lists": &listsFile,
		},
	}

	m.EXPECT().CheckForUpdates().Return("", nil)

	if _, _, err := a.handleUpdate(); err != nil {
		t.Error(err)
	}
}

func TestAptApplyAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	a := &Apt{
		Name: []string{
			"foo",
			"bar",
		},
		State: AptAbsent,
	}

	m.EXPECT().Remove(&apt.Package{Name: "foo"}, &apt.Package{Name: "bar"}).Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})
	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}

	a = &Apt{
		State: AptAbsent,
	}

	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}
}

func TestAptApplyUpgrade(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	a := &Apt{
		Upgrade: AptUpgradeFull,
	}

	m.EXPECT().DistUpgrade().Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})

	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}

	a = &Apt{
		Upgrade: AptUpgradeSafe,
	}

	m.EXPECT().UpgradeAll().Return("", nil)
	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}
}

func TestAptApplyClean(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockaptClient(ctrl)

	a := &Apt{
		Clean: true,
	}

	m.EXPECT().Clean().Return("", nil)

	ctx := context.WithValue(context.Background(), aptClientContextKey, m)
	ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})

	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}

	a = &Apt{
		Clean: true,
		Name: []string{
			"foo",
		},
	}

	m.EXPECT().Clean().Return("", nil)
	m.EXPECT().ListInstalled().Return(nil, nil)
	m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)

	if _, err := a.Apply(ctx, "", false); err != nil {
		t.Error(err)
	}
}

func TestAptApplyUpdateCache(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := NewMockaptClient(ctrl)

		pTrue := true
		a := &Apt{
			UpdateCache: &pTrue,
		}

		m.EXPECT().CheckForUpdates().Return("", nil)

		ctx := context.WithValue(context.Background(), aptClientContextKey, m)
		ctx = context.WithValue(ctx, aptFSContextKey, fstest.MapFS{})

		got, err := a.Apply(ctx, "", false)
		if err != nil {
			t.Error(err)
		}

		expected := &AptResult{
			CommonResult: CommonResult{
				Changed: true,
			},
			// In synctest.Test, time doesn't progress, so this works.
			CacheUpdateTime: time.Now(),
			CacheUpdated:    true,
		}

		if !cmp.Equal(got, expected, cmpopts.IgnoreUnexported(AptResult{})) {
			t.Errorf("expected %#v but got %#v", expected, got)
		}
	})
}
