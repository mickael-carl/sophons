package exec

import (
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/arduino/go-apt-client"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"go.uber.org/mock/gomock"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestAptValidate(t *testing.T) {
	tests := []struct {
		name    string
		apt     *Apt
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid state",
			apt: &Apt{
				Apt: &proto.Apt{
					State: "banana",
				},
			},
			wantErr: true,
			errMsg:  "unsupported state: banana",
		},
		{
			name: "invalid upgrade",
			apt: &Apt{
				Apt: &proto.Apt{
					Upgrade: "banana",
				},
			},
			wantErr: true,
			errMsg:  "unsupported upgrade mode: banana",
		},
		{
			name: "valid",
			apt: func() *Apt {
				cacheValidTime := uint64(360)
				pTrue := true
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"curl"},
						},
						CacheValidTime: &cacheValidTime,
						UpdateCache:    &pTrue,
						Upgrade:        "dist",
					},
				}
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.apt.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestAptApply(t *testing.T) {
	tests := []struct {
		name        string
		apt         *Apt
		mockFunc    func(*MockaptClient)
		want        *AptResult
		useSyncTest bool
	}{
		{
			name: "install packages",
			apt: &Apt{
				Apt: &proto.Apt{
					Name: &proto.PackageList{
						Items: []string{
							"foo",
							"bar",
						},
					},
					State: AptPresent,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().ListInstalled().Return([]*apt.Package{{Name: "foo"}}, nil)
				m.EXPECT().Install([]*apt.Package{{Name: "bar"}})
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "install latest",
			apt: &Apt{
				Apt: &proto.Apt{
					Name: &proto.PackageList{
						Items: []string{
							"foo",
						},
					},
					State: AptLatest,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "remove packages",
			apt: &Apt{
				Apt: &proto.Apt{
					Name: &proto.PackageList{
						Items: []string{
							"foo",
							"bar",
						},
					},
					State: AptAbsent,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().Remove(&apt.Package{Name: "foo"}, &apt.Package{Name: "bar"}).Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "remove with no packages",
			apt: &Apt{
				Apt: &proto.Apt{
					State: AptAbsent,
				},
			},
			mockFunc: func(m *MockaptClient) {},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: false,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "upgrade full",
			apt: &Apt{
				Apt: &proto.Apt{
					Upgrade: AptUpgradeFull,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().DistUpgrade().Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "upgrade safe",
			apt: &Apt{
				Apt: &proto.Apt{
					Upgrade: AptUpgradeSafe,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().UpgradeAll().Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "clean",
			apt: &Apt{
				Apt: &proto.Apt{
					Clean: true,
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().Clean().Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.Time{},
			},
		},
		{
			name: "clean with packages",
			apt: &Apt{
				Apt: &proto.Apt{
					Clean: true,
					Name: &proto.PackageList{
						Items: []string{
							"foo",
						},
					},
				},
			},
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().Clean().Return("", nil)
				m.EXPECT().ListInstalled().Return(nil, nil)
				m.EXPECT().Install(&apt.Package{Name: "foo"}).Return("", nil)
			},
			want: &AptResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
				CacheUpdated:    false,
				CacheUpdateTime: time.UnixMilli(0),
			},
		},
		{
			name: "update cache",
			apt: func() *Apt {
				pTrue := true
				return &Apt{
					Apt: &proto.Apt{
						UpdateCache: &pTrue,
					},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
			want:        nil, // Will be constructed inside synctest.Test
			useSyncTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFunc := func(t *testing.T) {
				// Use helper to reduce boilerplate
				ctx := newMockAptContext(t, tt.mockFunc)

				got, err := tt.apt.Apply(ctx, "", false)
				if err != nil {
					t.Errorf("Apply() error = %v", err)
					return
				}

				// Special handling for "update cache" test which needs time.Now() inside synctest
				want := tt.want
				if tt.name == "update cache" && tt.useSyncTest {
					want = &AptResult{
						CommonResult: CommonResult{
							Changed: true,
						},
						CacheUpdateTime: time.Now(),
						CacheUpdated:    true,
					}
				}

				if want != nil {
					if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(AptResult{})); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				}
			}

			if tt.useSyncTest {
				synctest.Test(t, testFunc)
			} else {
				testFunc(t)
			}
		})
	}
}

func TestHandleUpdate(t *testing.T) {
	tests := []struct {
		name     string
		apt      *Apt
		mockFunc func(*MockaptClient)
	}{
		{
			name: "cache valid time expired",
			apt: func() *Apt {
				cacheValidTime := uint64(1)
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"foo"},
						},
						State:          AptLatest,
						CacheValidTime: &cacheValidTime,
					},
					aptFS: fstest.MapFS{
						"lists": &fstest.MapFile{
							ModTime: time.Unix(0, 0),
						},
					},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
		},
		{
			name: "cache valid time not expired",
			apt: func() *Apt {
				cacheValidTime := uint64(1000)
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"foo"},
						},
						State:          AptLatest,
						CacheValidTime: &cacheValidTime,
					},
					aptFS: fstest.MapFS{
						"lists": &fstest.MapFile{
							ModTime: time.Now(),
						},
					},
				}
			}(),
			mockFunc: func(m *MockaptClient) {},
		},
		{
			name: "missing lists with cache valid time",
			apt: func() *Apt {
				cacheValidTime := uint64(1)
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"foo"},
						},
						State:          AptLatest,
						CacheValidTime: &cacheValidTime,
					},
					aptFS: fstest.MapFS{},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
		},
		{
			name: "missing lists with update cache",
			apt: func() *Apt {
				updateCache := true
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"foo"},
						},
						State:       AptLatest,
						UpdateCache: &updateCache,
					},
					aptFS: fstest.MapFS{},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
		},
		{
			name: "missing lists with nothing set",
			apt: &Apt{
				Apt: &proto.Apt{
					Name: &proto.PackageList{
						Items: []string{"foo"},
					},
					State: AptLatest,
				},
				aptFS: fstest.MapFS{},
			},
			mockFunc: func(m *MockaptClient) {},
		},
		{
			name: "explicit update cache",
			apt: func() *Apt {
				updateCache := true
				return &Apt{
					Apt: &proto.Apt{
						Name: &proto.PackageList{
							Items: []string{"foo"},
						},
						State:       AptLatest,
						UpdateCache: &updateCache,
					},
					aptFS: fstest.MapFS{
						"lists": &fstest.MapFile{},
					},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			m := NewMockaptClient(ctrl)

			tt.mockFunc(m)
			tt.apt.apt = m

			if _, _, err := tt.apt.handleUpdate(); err != nil {
				t.Errorf("handleUpdate() error = %v", err)
			}
		})
	}
}
