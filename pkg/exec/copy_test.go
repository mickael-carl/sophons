package exec

import (
	"testing"

	"github.com/mickael-carl/sophons/pkg/proto"
)

func TestCopyValidate(t *testing.T) {
	tests := []ValidationTestCase[*Copy]{
		{
			Name: "absolute path without remote_src",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:  "/etc/shadow",
					Dest: "/hacking-passwords",
				},
			},
			WantErr: true,
			ErrMsg:  "copying from an absolute path without remote_src is not supported",
		},
		{
			Name: "absolute path with remote_src",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:       "/tmp/someconfig",
					Dest:      "/etc/someconfig",
					RemoteSrc: true,
				},
			},
			WantErr: false,
		},
		{
			Name: "missing src and content",
			Input: &Copy{
				Copy: &proto.Copy{
					Dest: "/something",
				},
			},
			WantErr: true,
			ErrMsg:  "either src or content need to be specified",
		},
		{
			Name: "content with directory dest",
			Input: &Copy{
				Copy: &proto.Copy{
					Content: "hello world",
					Dest:    "/some/directory/",
				},
			},
			WantErr: true,
			ErrMsg:  "can't use content when dest is a directory",
		},
		{
			Name: "both src and content set",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:     "somefile",
					Content: "hello world!",
					Dest:    "/somefile",
				},
			},
			WantErr: true,
			ErrMsg:  "src and content can't both be specified",
		},
		{
			Name: "empty src",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:  "",
					Dest: "/somewhere",
				},
			},
			WantErr:     true,
			ErrContains: "src",
		},
		{
			Name: "valid relative src",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:  "files/config.txt",
					Dest: "/etc/config.txt",
				},
			},
			WantErr: false,
		},
		{
			Name: "content with newlines",
			Input: &Copy{
				Copy: &proto.Copy{
					Content: "line1\nline2\nline3",
					Dest:    "/tmp/multiline.txt",
				},
			},
			WantErr: false,
		},
		{
			Name: "empty content still requires src",
			Input: &Copy{
				Copy: &proto.Copy{
					Content: "",
					Dest:    "/tmp/empty.txt",
				},
			},
			WantErr:     true,
			ErrContains: "src or content",
		},
		{
			Name: "dest with trailing slash is directory",
			Input: &Copy{
				Copy: &proto.Copy{
					Src:  "file.txt",
					Dest: "/tmp/dir/",
				},
			},
			WantErr: false,
		},
	}

	RunValidationTests(t, tests)
}
