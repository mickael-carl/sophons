package exec

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/variables"
)

func TestTemplateValidate(t *testing.T) {
	tests := []struct {
		name     string
		template *Template
		wantErr  bool
		errMsg   string
	}{
		{
			name: "absolute path",
			template: &Template{
				Template: &proto.Template{
					Src:  "/etc/shadow",
					Dest: "/hacking-passwords",
				},
			},
			wantErr: true,
			errMsg:  "template from an absolute path is not supported",
		},
		{
			name: "missing src",
			template: &Template{
				Template: &proto.Template{
					Dest: "/something",
				},
			},
			wantErr: true,
			errMsg:  "src is required",
		},
		{
			name: "missing dest",
			template: &Template{
				Template: &proto.Template{
					Src: "foo",
				},
			},
			wantErr: true,
			errMsg:  "dest is required",
		},
		{
			name: "valid",
			template: &Template{
				Template: &proto.Template{
					Src:  "foo",
					Dest: "bar",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
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

// Helper functions for Apply tests

func createTemplateFile(t *testing.T, dir, name, content string) {
	t.Helper()
	templateDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(templateDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func createDestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func verifyFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(content) != expected {
		t.Errorf("content mismatch: got %q, want %q", string(content), expected)
	}
}

func calculateChecksums(content []byte) (sha1Hash, md5Hash string) {
	sha1Sum := sha1.Sum(content)
	md5Sum := md5.Sum(content)
	return hex.EncodeToString(sha1Sum[:]), hex.EncodeToString(md5Sum[:])
}

func TestTemplateApply(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T, string) (*Template, context.Context)
		verify   func(*testing.T, *TemplateResult, string)
		wantErr  bool
		errCheck func(*testing.T, error)
	}{
		{
			name: "skip on matching MD5",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				content := "Hello World"
				createTemplateFile(t, tempDir, "test.j2", content)
				destFile := filepath.Join(tempDir, "dest.txt")
				createDestFile(t, destFile, content)

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				expectedChecksum, _ := calculateChecksums([]byte("Hello World"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					CommonResult: CommonResult{
						Skipped: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "change on different content",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "New Content")
				destFile := filepath.Join(tempDir, "dest.txt")
				createDestFile(t, destFile, "Old Content")

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				// Verify actual file content
				destFile := filepath.Join(tempDir, "dest.txt")
				verifyFileContent(t, destFile, "New Content")

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("New Content"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					MD5Sum:   expectedMD5,
					Size:     uint64(len("New Content")),
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "change when destination missing",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "New File Content")
				destFile := filepath.Join(tempDir, "dest.txt")

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				destFile := filepath.Join(tempDir, "dest.txt")
				verifyFileContent(t, destFile, "New File Content")

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("New File Content"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					MD5Sum:   expectedMD5,
					Size:     uint64(len("New File Content")),
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "template rendering with variables",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "Hello {{ name }}!")
				destFile := filepath.Join(tempDir, "dest.txt")

				vars := variables.Variables{"name": "World"}
				ctx := variables.NewContext(context.Background(), vars)

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
					},
				}, ctx
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				destFile := filepath.Join(tempDir, "dest.txt")
				verifyFileContent(t, destFile, "Hello World!")

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("Hello World!"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					MD5Sum:   expectedMD5,
					Size:     uint64(len("Hello World!")),
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "error - missing template file",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				destFile := filepath.Join(tempDir, "dest.txt")

				return &Template{
					Template: &proto.Template{
						Src:  "nonexistent.j2",
						Dest: destFile,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				want := &TemplateResult{
					CommonResult: CommonResult{
						Failed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error for missing template file")
				}
			},
		},
		{
			name: "error - invalid template syntax",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				// Create template with invalid Jinja2 syntax
				createTemplateFile(t, tempDir, "test.j2", "{{ unclosed")
				destFile := filepath.Join(tempDir, "dest.txt")

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				want := &TemplateResult{
					CommonResult: CommonResult{
						Failed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected error for invalid template syntax")
				}
			},
		},
		{
			name: "set mode only",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "Content with mode")
				destFile := filepath.Join(tempDir, "dest.txt")

				return &Template{
					Template: &proto.Template{
						Src:  "test.j2",
						Dest: destFile,
						Mode: &proto.Mode{Value: "0600"},
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				destFile := filepath.Join(tempDir, "dest.txt")

				// Verify actual file permissions
				info, err := os.Stat(destFile)
				if err != nil {
					t.Fatalf("failed to stat file: %v", err)
				}
				actualMode := fmt.Sprintf("%04o", info.Mode().Perm())
				if actualMode != "0600" {
					t.Errorf("Actual file mode mismatch: got %s, want 0600", actualMode)
				}

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("Content with mode"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					MD5Sum:   expectedMD5,
					Mode:     "0600",
					Size:     uint64(len("Content with mode")),
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "set owner and group",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "Content with owner")
				destFile := filepath.Join(tempDir, "dest.txt")

				// Get current user and group
				currentUser, err := user.Current()
				if err != nil {
					t.Skipf("Cannot get current user: %v", err)
				}

				group, err := user.LookupGroupId(currentUser.Gid)
				if err != nil {
					t.Skipf("Cannot lookup group: %v", err)
				}

				return &Template{
					Template: &proto.Template{
						Src:   "test.j2",
						Dest:  destFile,
						Owner: currentUser.Username,
						Group: group.Name,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				destFile := filepath.Join(tempDir, "dest.txt")

				currentUser, err := user.Current()
				if err != nil {
					t.Fatalf("Cannot get current user: %v", err)
				}

				uid, _ := strconv.ParseUint(currentUser.Uid, 10, 64)
				gid, _ := strconv.ParseUint(currentUser.Gid, 10, 64)
				group, _ := user.LookupGroupId(currentUser.Gid)

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("Content with owner"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					Gid:      gid,
					Group:    group.Name,
					MD5Sum:   expectedMD5,
					Owner:    currentUser.Username,
					Size:     uint64(len("Content with owner")),
					Uid:      uid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "set mode, owner, and group",
			setup: func(t *testing.T, tempDir string) (*Template, context.Context) {
				createTemplateFile(t, tempDir, "test.j2", "Complete permissions test")
				destFile := filepath.Join(tempDir, "dest.txt")

				currentUser, err := user.Current()
				if err != nil {
					t.Skipf("Cannot get current user: %v", err)
				}

				group, err := user.LookupGroupId(currentUser.Gid)
				if err != nil {
					t.Skipf("Cannot lookup group: %v", err)
				}

				return &Template{
					Template: &proto.Template{
						Src:   "test.j2",
						Dest:  destFile,
						Mode:  &proto.Mode{Value: "0640"},
						Owner: currentUser.Username,
						Group: group.Name,
					},
				}, context.Background()
			},
			verify: func(t *testing.T, result *TemplateResult, tempDir string) {
				destFile := filepath.Join(tempDir, "dest.txt")

				// Verify actual file permissions
				info, err := os.Stat(destFile)
				if err != nil {
					t.Fatalf("failed to stat file: %v", err)
				}
				actualMode := fmt.Sprintf("%04o", info.Mode().Perm())
				if actualMode != "0640" {
					t.Errorf("Actual file mode mismatch: got %s, want 0640", actualMode)
				}

				currentUser, err := user.Current()
				if err != nil {
					t.Fatalf("Cannot get current user: %v", err)
				}

				uid, _ := strconv.ParseUint(currentUser.Uid, 10, 64)
				gid, _ := strconv.ParseUint(currentUser.Gid, 10, 64)
				group, _ := user.LookupGroupId(currentUser.Gid)

				expectedChecksum, expectedMD5 := calculateChecksums([]byte("Complete permissions test"))
				want := &TemplateResult{
					Checksum: expectedChecksum,
					Dest:     destFile,
					Gid:      gid,
					Group:    group.Name,
					MD5Sum:   expectedMD5,
					Mode:     "0640",
					Owner:    currentUser.Username,
					Size:     uint64(len("Complete permissions test")),
					Uid:      uid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result, cmpopts.IgnoreUnexported(TemplateResult{})); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup
			tmpl, ctx := tt.setup(t, tempDir)

			// Execute
			got, err := tmpl.Apply(ctx, tempDir, false)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			// Verify result
			result, ok := got.(*TemplateResult)
			if !ok {
				t.Fatal("result is not *TemplateResult")
			}

			tt.verify(t, result, tempDir)
		})
	}
}
