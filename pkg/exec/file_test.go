package exec

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFileValidate(t *testing.T) {
	tests := []ValidationTestCase[*File]{
		{
			Name: "invalid state",
			Input: &File{
				State: "banana",
			},
			WantErr: true,
			ErrMsg:  "invalid state",
		},
		{
			Name: "missing path",
			Input: &File{
				State: "file",
			},
			WantErr: true,
			ErrMsg:  "path is required",
		},
		{
			Name: "recurse without directory state",
			Input: &File{
				State:   "file",
				Path:    "/foo",
				Recurse: true,
			},
			WantErr: true,
			ErrMsg:  "recurse option requires state to be 'directory'",
		},
		{
			Name: "link without src",
			Input: &File{
				State: "link",
				Path:  "/foo/bar",
			},
			WantErr: true,
			ErrMsg:  "src option is required when state is 'link' or 'hard'",
		},
		{
			Name: "valid file state",
			Input: &File{
				State: "file",
				Path:  "/tmp/test.txt",
			},
			WantErr: false,
		},
		{
			Name: "valid directory state",
			Input: &File{
				State: "directory",
				Path:  "/tmp/testdir",
			},
			WantErr: false,
		},
		{
			Name: "valid touch state",
			Input: &File{
				State: "touch",
				Path:  "/tmp/touched",
			},
			WantErr: false,
		},
		{
			Name: "valid absent state",
			Input: &File{
				State: "absent",
				Path:  "/tmp/removed",
			},
			WantErr: false,
		},
		{
			Name: "hard link without src",
			Input: &File{
				State: "hard",
				Path:  "/foo/hardlink",
			},
			WantErr:     true,
			ErrContains: "src",
		},
		{
			Name: "valid link with src",
			Input: &File{
				State: "link",
				Path:  "/foo/link",
				Src:   "/foo/target",
			},
			WantErr: false,
		},
		{
			Name: "recurse with directory state",
			Input: &File{
				State:   "directory",
				Path:    "/foo",
				Recurse: true,
			},
			WantErr: false,
		},
	}

	RunValidationTests(t, tests)
}

func TestFileUnmarshalYAML(t *testing.T) {
	pFalse := false

	tests := []struct {
		name string
		yaml string
		want File
	}{
		{
			name: "unmarshal with path",
			yaml: `
path: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`,
			want: File{
				Path:    "/foo",
				Follow:  &pFalse,
				Group:   "bar",
				Mode:    "0644",
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   FileFile,
			},
		},
		{
			name: "unmarshal with dest alias",
			yaml: `
dest: "/foo"
follow: false
group: "bar"
mode: "0644"
owner: "baz"
recurse: false
src: "/hello"
state: "file"`,
			want: File{
				Path:    "/foo",
				Follow:  &pFalse,
				Group:   "bar",
				Mode:    "0644",
				Owner:   "baz",
				Recurse: false,
				Src:     "/hello",
				State:   FileFile,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got File
			if err := yaml.Unmarshal([]byte(tt.yaml), &got); err != nil {
				t.Errorf("Unmarshal() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func createTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
}

func createTestDir(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, mode); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func createTestSymlink(t *testing.T, src, dest string) {
	t.Helper()
	if err := os.Symlink(src, dest); err != nil {
		t.Fatalf("failed to create symlink %s -> %s: %v", dest, src, err)
	}
}

func verifyFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Errorf("file %s should exist: %v", path, err)
	}
}

func verifyFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("file %s should not exist", path)
	}
}

func verifyFileMode(t *testing.T, path, expectedMode string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}
	actualMode := fmt.Sprintf("%#o", info.Mode().Perm())
	if actualMode != expectedMode {
		t.Errorf("mode mismatch for %s: got %s, want %s", path, actualMode, expectedMode)
	}
}

func verifySymlinkTarget(t *testing.T, path, expectedTarget string) {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("failed to read symlink %s: %v", path, err)
	}
	if target != expectedTarget {
		t.Errorf("symlink target mismatch for %s: got %s, want %s", path, target, expectedTarget)
	}
}

func getFileInfo(t *testing.T, path string) (uid, gid uint32, mode string, size uint64) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}

	mode = fmt.Sprintf("%#o", info.Mode().Perm())
	size = uint64(info.Size())

	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		uid = st.Uid
		gid = st.Gid
	}

	return uid, gid, mode, size
}

func getCurrentUserInfo(t *testing.T) (uid, gid uint32, owner, group string) {
	t.Helper()
	currentUser, err := user.Current()
	if err != nil {
		t.Fatalf("failed to get current user: %v", err)
	}

	uidInt, err := strconv.ParseUint(currentUser.Uid, 10, 32)
	if err != nil {
		t.Fatalf("failed to parse UID: %v", err)
	}

	gidInt, err := strconv.ParseUint(currentUser.Gid, 10, 32)
	if err != nil {
		t.Fatalf("failed to parse GID: %v", err)
	}

	grp, err := user.LookupGroupId(currentUser.Gid)
	if err != nil {
		t.Fatalf("failed to lookup group: %v", err)
	}

	return uint32(uidInt), uint32(gidInt), currentUser.Username, grp.Name
}

func TestFileApply(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T, string) (*File, context.Context)
		verify   func(*testing.T, *FileResult, string)
		wantErr  bool
		errCheck func(*testing.T, error)
	}{
		{
			name: "directory - create new",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "newdir")
				return &File{
					Path:  dirPath,
					State: FileDirectory,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "newdir")
				verifyFileExists(t, dirPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, mode, _ := getFileInfo(t, dirPath)

				want := &FileResult{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  mode,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size for directories as it varies by filesystem
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "directory - no state but recurse",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "newdir-recurse")
				return &File{
					Path:    dirPath,
					Recurse: true,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "newdir-recurse")
				verifyFileExists(t, dirPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, mode, _ := getFileInfo(t, dirPath)

				want := &FileResult{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  mode,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size for directories as it varies by filesystem
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "directory - create with mode",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "dirwithmode")
				return &File{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  "0700",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "dirwithmode")
				verifyFileExists(t, dirPath)
				verifyFileMode(t, dirPath, "0700")

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  "0700",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size for directories as it varies by filesystem
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "directory - recurse with nested structure",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "recurse")
				createTestDir(t, dirPath, 0o755)
				createTestDir(t, filepath.Join(dirPath, "subdir"), 0o755)
				createTestFile(t, filepath.Join(dirPath, "file1.txt"), "content", 0o644)
				createTestFile(t, filepath.Join(dirPath, "subdir", "file2.txt"), "content", 0o644)
				createTestSymlink(t, filepath.Join(dirPath, "file1.txt"), filepath.Join(dirPath, "subdir", "linkfile"))

				return &File{
					Path:    dirPath,
					State:   FileDirectory,
					Mode:    "0700",
					Recurse: true,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "recurse")

				verifyFileMode(t, dirPath, "0700")
				verifyFileMode(t, filepath.Join(dirPath, "subdir"), "0700")
				verifyFileMode(t, filepath.Join(dirPath, "file1.txt"), "0700")
				verifyFileMode(t, filepath.Join(dirPath, "subdir", "file2.txt"), "0700")

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  "0700",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size for directories as it varies by filesystem
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "file - does not exist should fail",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "nonexistent.txt")
				return &File{
					Path:  filePath,
					State: FileFile,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "nonexistent.txt")

				want := &FileResult{
					Path: filePath,
					CommonResult: CommonResult{
						Failed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				if err == nil || err.Error() != "file does not exist" {
					t.Errorf("expected 'file does not exist' error, got %v", err)
				}
			},
		},
		{
			name: "file - no properties set should skip",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "existing.txt")
				createTestFile(t, filePath, "content", 0o644)

				return &File{
					Path:  filePath,
					State: FileFile,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "existing.txt")

				want := &FileResult{
					Path: filePath,
					CommonResult: CommonResult{
						Skipped: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "file - change mode",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "chmod.txt")
				createTestFile(t, filePath, "content", 0o644)

				return &File{
					Path:  filePath,
					State: FileFile,
					Mode:  "0600",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "chmod.txt")
				verifyFileMode(t, filePath, "0600")

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, _, size := getFileInfo(t, filePath)

				want := &FileResult{
					Path:  filePath,
					State: FileFile,
					Mode:  "0600",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "file - change mode symbolic",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "chmod-symbolic.txt")
				createTestFile(t, filePath, "content", 0o600)

				return &File{
					Path:  filePath,
					State: FileFile,
					Mode:  "g+rw,o+r",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "chmod-symbolic.txt")
				verifyFileMode(t, filePath, "0664")

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, _, size := getFileInfo(t, filePath)

				want := &FileResult{
					Path:  filePath,
					State: FileFile,
					Mode:  "0664",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "link - create new symlink",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "link.txt")
				createTestFile(t, targetPath, "content", 0o644)

				return &File{
					Path:  linkPath,
					Src:   targetPath,
					State: FileLink,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "link.txt")

				verifyFileExists(t, linkPath)
				verifySymlinkTarget(t, linkPath, targetPath)

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Dest:  linkPath,
					State: FileLink,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size and Mode for symlinks as they vary by system
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size", "Mode")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "link - symlink exists with same target",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "existinglink.txt")
				createTestFile(t, targetPath, "content", 0o644)
				createTestSymlink(t, targetPath, linkPath)

				return &File{
					Path:  linkPath,
					Src:   targetPath,
					State: FileLink,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "existinglink.txt")

				verifySymlinkTarget(t, linkPath, targetPath)

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Dest:  linkPath,
					State: FileLink,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: false,
					},
				}

				// Ignore Size and Mode for symlinks as they vary by system
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size", "Mode")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "link - update symlink to different target",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				oldTarget := filepath.Join(tempDir, "old.txt")
				newTarget := filepath.Join(tempDir, "new.txt")
				linkPath := filepath.Join(tempDir, "updatelink.txt")

				createTestFile(t, oldTarget, "old", 0o644)
				createTestFile(t, newTarget, "new", 0o644)
				createTestSymlink(t, oldTarget, linkPath)

				return &File{
					Path:  linkPath,
					Src:   newTarget,
					State: FileLink,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				newTarget := filepath.Join(tempDir, "new.txt")
				linkPath := filepath.Join(tempDir, "updatelink.txt")

				verifySymlinkTarget(t, linkPath, newTarget)

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Dest:  linkPath,
					State: FileLink,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size and Mode for symlinks as they vary by system
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size", "Mode")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "link - change mode",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "follow-mode.txt")
				createTestFile(t, targetPath, "content", 0o644)

				return &File{
					Path:  linkPath,
					Src:   targetPath,
					State: FileLink,
					Mode:  "0600",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "follow-mode.txt")
				verifyFileExists(t, linkPath)
				verifyFileMode(t, targetPath, "0600")

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Dest:  linkPath,
					State: FileLink,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Mode:  "0600",
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size and Mode for symlinks as they vary by system
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size", "Mode")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "link - follow=false does not apply mode to target",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				targetPath := filepath.Join(tempDir, "target.txt")
				linkPath := filepath.Join(tempDir, "nofollow.txt")
				createTestFile(t, targetPath, "content", 0o644)

				followFalse := false
				return &File{
					Path:   linkPath,
					Src:    targetPath,
					State:  FileLink,
					Follow: &followFalse,
					Mode:   "0600",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				linkPath := filepath.Join(tempDir, "nofollow.txt")
				verifyFileExists(t, linkPath)

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Dest:  linkPath,
					State: FileLink,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size and Mode for symlinks as they vary by system
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size", "Mode")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "touch - create new file",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				touchPath := filepath.Join(tempDir, "touched.txt")

				return &File{
					Path:  touchPath,
					State: FileTouch,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				touchPath := filepath.Join(tempDir, "touched.txt")
				verifyFileExists(t, touchPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, mode, size := getFileInfo(t, touchPath)

				want := &FileResult{
					Dest:  touchPath,
					State: FileFile,
					Mode:  mode,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "touch - existing file",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				touchPath := filepath.Join(tempDir, "existing.txt")
				createTestFile(t, touchPath, "existing content", 0o644)

				return &File{
					Path:  touchPath,
					State: FileTouch,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				touchPath := filepath.Join(tempDir, "existing.txt")
				verifyFileExists(t, touchPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, mode, size := getFileInfo(t, touchPath)

				want := &FileResult{
					Dest:  touchPath,
					State: FileFile,
					Mode:  mode,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: false,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "touch - existing file with mode",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				touchPath := filepath.Join(tempDir, "existing-mode.txt")
				createTestFile(t, touchPath, "existing content", 0o600)

				return &File{
					Path:  touchPath,
					State: FileTouch,
					Mode:  0o644,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				touchPath := filepath.Join(tempDir, "existing-mode.txt")
				verifyFileExists(t, touchPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, _, size := getFileInfo(t, touchPath)

				want := &FileResult{
					Dest:  touchPath,
					State: FileFile,
					Mode:  "0644",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "touch - with mode",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				touchPath := filepath.Join(tempDir, "touchmode.txt")

				return &File{
					Path:  touchPath,
					State: FileTouch,
					Mode:  "0600",
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				touchPath := filepath.Join(tempDir, "touchmode.txt")
				verifyFileExists(t, touchPath)
				verifyFileMode(t, touchPath, "0600")

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, _, size := getFileInfo(t, touchPath)

				want := &FileResult{
					Dest:  touchPath,
					State: FileFile,
					Mode:  "0600",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "touch - with owner and group",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				touchPath := filepath.Join(tempDir, "touchowner.txt")

				currentUser, err := user.Current()
				if err != nil {
					t.Skipf("Cannot get current user: %v", err)
				}

				group, err := user.LookupGroupId(currentUser.Gid)
				if err != nil {
					t.Skipf("Cannot lookup group: %v", err)
				}

				return &File{
					Path:  touchPath,
					State: FileTouch,
					Owner: currentUser.Username,
					Group: group.Name,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				touchPath := filepath.Join(tempDir, "touchowner.txt")
				verifyFileExists(t, touchPath)

				uid, gid, owner, group := getCurrentUserInfo(t)
				_, _, mode, size := getFileInfo(t, touchPath)

				want := &FileResult{
					Dest:  touchPath,
					State: FileFile,
					Mode:  mode,
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  size,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "absent - remove existing file",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "remove.txt")
				createTestFile(t, filePath, "content", 0o644)

				return &File{
					Path:  filePath,
					State: FileAbsent,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "remove.txt")
				verifyFileNotExists(t, filePath)

				want := &FileResult{
					Path:  filePath,
					State: FileAbsent,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "absent - remove existing directory",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "removedir")
				createTestDir(t, dirPath, 0o755)
				createTestFile(t, filepath.Join(dirPath, "file.txt"), "content", 0o644)

				return &File{
					Path:  dirPath,
					State: FileAbsent,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "removedir")
				verifyFileNotExists(t, dirPath)

				want := &FileResult{
					Path:  dirPath,
					State: FileAbsent,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "absent - non-existent path",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "doesnotexist.txt")

				return &File{
					Path:  filePath,
					State: FileAbsent,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "doesnotexist.txt")

				want := &FileResult{
					Path:  filePath,
					State: FileAbsent,
					CommonResult: CommonResult{
						Changed: false,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "error - state=hard not implemented",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				srcPath := filepath.Join(tempDir, "src.txt")
				hardPath := filepath.Join(tempDir, "hard.txt")
				createTestFile(t, srcPath, "content", 0o644)

				return &File{
					Path:  hardPath,
					Src:   srcPath,
					State: FileHard,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				hardPath := filepath.Join(tempDir, "hard.txt")

				want := &FileResult{
					Dest: hardPath,
					CommonResult: CommonResult{
						Failed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: true,
			errCheck: func(t *testing.T, err error) {
				if err == nil || err.Error() != "not implemented" {
					t.Errorf("expected 'not implemented' error, got %v", err)
				}
			},
		},
		{
			name: "directory - all properties set",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				dirPath := filepath.Join(tempDir, "fulldir")

				currentUser, err := user.Current()
				if err != nil {
					t.Skipf("Cannot get current user: %v", err)
				}

				group, err := user.LookupGroupId(currentUser.Gid)
				if err != nil {
					t.Skipf("Cannot lookup group: %v", err)
				}

				return &File{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  "0750",
					Owner: currentUser.Username,
					Group: group.Name,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				dirPath := filepath.Join(tempDir, "fulldir")
				verifyFileExists(t, dirPath)
				verifyFileMode(t, dirPath, "0750")

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Path:  dirPath,
					State: FileDirectory,
					Mode:  "0750",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				// Ignore Size for directories as it varies by filesystem
				if diff := cmp.Diff(want, result, cmpopts.IgnoreFields(FileResult{}, "Size")); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
		{
			name: "file - verify all return values",
			setup: func(t *testing.T, tempDir string) (*File, context.Context) {
				filePath := filepath.Join(tempDir, "fullfile.txt")
				createTestFile(t, filePath, "test content", 0o644)

				currentUser, err := user.Current()
				if err != nil {
					t.Skipf("Cannot get current user: %v", err)
				}

				group, err := user.LookupGroupId(currentUser.Gid)
				if err != nil {
					t.Skipf("Cannot lookup group: %v", err)
				}

				return &File{
					Path:  filePath,
					State: FileFile,
					Mode:  "0640",
					Owner: currentUser.Username,
					Group: group.Name,
				}, context.Background()
			},
			verify: func(t *testing.T, result *FileResult, tempDir string) {
				filePath := filepath.Join(tempDir, "fullfile.txt")
				verifyFileMode(t, filePath, "0640")

				uid, gid, owner, group := getCurrentUserInfo(t)

				want := &FileResult{
					Path:  filePath,
					State: FileFile,
					Mode:  "0640",
					Owner: owner,
					Group: group,
					Uid:   uid,
					Gid:   gid,
					Size:  uint64(len("test content")),
					CommonResult: CommonResult{
						Changed: true,
					},
				}

				if diff := cmp.Diff(want, result); diff != "" {
					t.Errorf("mismatch (-want +got):\n%s", diff)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			file, ctx := tt.setup(t, tempDir)

			got, err := file.Apply(ctx, tempDir, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("Apply() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errCheck != nil {
				tt.errCheck(t, err)
			}

			result, ok := got.(*FileResult)
			if !ok {
				t.Fatal("result is not *FileResult")
			}

			tt.verify(t, result, tempDir)
		})
	}
}
