package exec

import (
	"context"
	"testing"

	"github.com/arduino/go-apt-client"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.uber.org/mock/gomock"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestAptRepositoryValidate(t *testing.T) {
	tests := []struct {
		name    string
		aptRepo *AptRepository
		wantErr bool
		errMsg  string
	}{
		{
			name: "invalid state",
			aptRepo: &AptRepository{
				AptRepository: &proto.AptRepository{
					Repo:  "foo",
					State: "banana",
				},
			},
			wantErr: true,
			errMsg:  "unsupported state: banana",
		},
		{
			name: "missing repo",
			aptRepo: &AptRepository{
				AptRepository: &proto.AptRepository{
					State: "present",
				},
			},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name: "valid",
			aptRepo: &AptRepository{
				AptRepository: &proto.AptRepository{
					Repo:  "foo",
					State: "present",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.aptRepo.Validate()
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

func TestURIToFilename(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid URL",
			uri:     "https://download.docker.com/linux/debian",
			want:    "download_docker_com_linux_debian.list",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			uri:     "some_invalid:url",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uriToFilename(tt.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("uriToFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("uriToFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAptRepositoryApply(t *testing.T) {
	tests := []struct {
		name     string
		aptRepo  *AptRepository
		mockFunc func(*MockaptClient)
		want     *AptRepositoryResult
	}{
		{
			name: "add repository",
			aptRepo: &AptRepository{
				AptRepository: &proto.AptRepository{
					Repo:  "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
					State: AptRepositoryPresent,
				},
			},
			mockFunc: func(m *MockaptClient) {
				repo := &apt.Repository{
					Enabled:      true,
					SourceRepo:   false,
					Options:      "signed-by=/etc/apt/keyrings/docker.asc",
					URI:          "https://download.docker.com/linux/debian",
					Distribution: "bookworm",
					Components:   "stable",
					Comment:      "",
				}
				m.EXPECT().ParseAPTConfigFolder("/etc/apt").Return(nil, nil)
				m.EXPECT().ParseAPTConfigLine("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable").Return(repo)
				m.EXPECT().AddRepository(repo, "/etc/apt", "download_docker_com_linux_debian.list").Return(nil)
				m.EXPECT().CheckForUpdates().Return("", nil)
			},
			want: &AptRepositoryResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
			},
		},
		{
			name: "remove repository",
			aptRepo: func() *AptRepository {
				pFalse := false
				return &AptRepository{
					AptRepository: &proto.AptRepository{
						Repo:        "deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable",
						State:       AptRepositoryAbsent,
						UpdateCache: &pFalse,
					},
				}
			}(),
			mockFunc: func(m *MockaptClient) {
				repo := &apt.Repository{
					Enabled:      true,
					SourceRepo:   false,
					Options:      "signed-by=/etc/apt/keyrings/docker.asc",
					URI:          "https://download.docker.com/linux/debian",
					Distribution: "bookworm",
					Components:   "stable",
					Comment:      "",
				}
				m.EXPECT().ParseAPTConfigFolder("/etc/apt").Return(apt.RepositoryList{repo}, nil)
				m.EXPECT().ParseAPTConfigLine("deb [signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable").Return(repo)
				m.EXPECT().RemoveRepository(repo, "/etc/apt").Return(nil)
			},
			want: &AptRepositoryResult{
				CommonResult: CommonResult{
					Changed: true,
					Failed:  false,
					Skipped: false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			m := NewMockaptClient(ctrl)

			tt.mockFunc(m)

			ctx := context.WithValue(context.Background(), aptClientContextKey, m)
			got, err := tt.aptRepo.Apply(ctx, "", false)
			if err != nil {
				t.Errorf("Apply() error = %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreUnexported(AptRepositoryResult{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
