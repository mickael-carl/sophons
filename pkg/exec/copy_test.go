package exec

import "testing"

func TestCopyValidate(t *testing.T) {
	tests := []ValidationTestCase[*Copy]{
		{
			Name: "absolute path without remote_src",
			Input: &Copy{
				Src:  "/etc/shadow",
				Dest: "/hacking-passwords",
			},
			WantErr: true,
			ErrMsg:  "copying from an absolute path without remote_src is not supported",
		},
		{
			Name: "absolute path with remote_src",
			Input: &Copy{
				Src:       "/tmp/someconfig",
				Dest:      "/etc/someconfig",
				RemoteSrc: true,
			},
			WantErr: false,
		},
		{
			Name: "missing src and content",
			Input: &Copy{
				Dest: "/something",
			},
			WantErr: true,
			ErrMsg:  "either src or content need to be specified",
		},
		{
			Name: "content with directory dest",
			Input: &Copy{
				Content: "hello world",
				Dest:    "/some/directory/",
			},
			WantErr: true,
			ErrMsg:  "can't use content when dest is a directory",
		},
		{
			Name: "both src and content set",
			Input: &Copy{
				Src:     "somefile",
				Content: "hello world!",
				Dest:    "/somefile",
			},
			WantErr: true,
			ErrMsg:  "src and content can't both be specified",
		},
		{
			Name: "empty src",
			Input: &Copy{
				Src:  "",
				Dest: "/somewhere",
			},
			WantErr:     true,
			ErrContains: "src",
		},
		{
			Name: "valid relative src",
			Input: &Copy{
				Src:  "files/config.txt",
				Dest: "/etc/config.txt",
			},
			WantErr: false,
		},
		{
			Name: "content with newlines",
			Input: &Copy{
				Content: "line1\nline2\nline3",
				Dest:    "/tmp/multiline.txt",
			},
			WantErr: false,
		},
		{
			Name: "empty content still requires src",
			Input: &Copy{
				Content: "",
				Dest:    "/tmp/empty.txt",
			},
			WantErr:     true,
			ErrContains: "src or content",
		},
		{
			Name: "dest with trailing slash is directory",
			Input: &Copy{
				Src:  "file.txt",
				Dest: "/tmp/dir/",
			},
			WantErr: false,
		},
	}

	RunValidationTests(t, tests)
}
