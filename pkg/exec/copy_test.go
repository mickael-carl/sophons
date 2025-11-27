package exec

import "testing"

func TestCopyValidate(t *testing.T) {
	tests := []struct {
		name    string
		copy    Copy
		wantErr bool
		errMsg  string
	}{
		{
			name: "absolute path without remote_src",
			copy: Copy{
				Src:  "/etc/shadow",
				Dest: "/hacking-passwords",
			},
			wantErr: true,
			errMsg:  "copying from an absolute path without remote_src is not supported",
		},
		{
			name: "absolute path with remote_src",
			copy: Copy{
				Src:       "/tmp/someconfig",
				Dest:      "/etc/someconfig",
				RemoteSrc: true,
			},
			wantErr: false,
		},
		{
			name: "missing src and content",
			copy: Copy{
				Dest: "/something",
			},
			wantErr: true,
			errMsg:  "either src or content need to be specified",
		},
		{
			name: "content with directory dest",
			copy: Copy{
				Content: "hello world",
				Dest:    "/some/directory/",
			},
			wantErr: true,
			errMsg:  "can't use content when dest is a directory",
		},
		{
			name: "both src and content set",
			copy: Copy{
				Src:     "somefile",
				Content: "hello world!",
				Dest:    "/somefile",
			},
			wantErr: true,
			errMsg:  "src and content can't both be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.copy.Validate()
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
