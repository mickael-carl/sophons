package exec

import (
	"testing"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestGetURLValidate(t *testing.T) {
	tests := []struct {
		name    string
		getURL  *GetURL
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing url",
			getURL: &GetURL{
				GetURL: proto.GetURL{
					Dest: "/somewhere",
				},
			},
			wantErr: true,
			errMsg:  "url is required",
		},
		{
			name: "missing dest",
			getURL: &GetURL{
				GetURL: proto.GetURL{
					Url: "https://example.com",
				},
			},
			wantErr: true,
			errMsg:  "dest is required",
		},
		{
			name: "invalid url",
			getURL: &GetURL{
				GetURL: proto.GetURL{
					Url:  "foo_bar:baz",
					Dest: "/somewhere",
				},
			},
			wantErr: true,
			errMsg:  "invalid URL provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.getURL.Validate()
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
