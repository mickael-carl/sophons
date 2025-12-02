package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mickael-carl/sophons/pkg/proto"
)

//	@meta {
//	  "deviations": ["`src` doesn't support absolute paths when `remote_src` is false."]
//	}
type Copy struct {
	proto.Copy `yaml:",inline"`
}

type CopyResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("copy", func() TaskContent { return &Copy{} })
	RegisterTaskType("ansible.builtin.copy", func() TaskContent { return &Copy{} })
}

func copySingleFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s for reading: %w", src, err)
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", dst, err)
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", src, dst, err)
	}

	return nil
}

func (c *Copy) Validate() error {
	// We don't support copying random files from the controller, as it seems
	// like a bad idea. All files should belong in a role. This might change
	// eventually, provided there is a genuinely good usecase for it.
	if filepath.IsAbs(c.Src) && !c.RemoteSrc {
		return errors.New("copying from an absolute path without remote_src is not supported")
	}

	if c.Src == "" && c.Content == "" {
		return errors.New("either src or content need to be specified")
	}

	if c.Src != "" && c.Content != "" {
		return errors.New("src and content can't both be specified")
	}

	if c.Content != "" && strings.HasSuffix(c.Dest, string(os.PathSeparator)) {
		return errors.New("can't use content when dest is a directory")
	}

	if c.Dest == "" {
		return errors.New("dest is required")
	}

	return nil
}

func (c *Copy) copyDir(actualSrc string) error {
	copyContentsOnly := strings.HasSuffix(actualSrc, string(os.PathSeparator))

	dstDir := c.Dest
	if !copyContentsOnly {
		dstDir = filepath.Join(c.Dest, filepath.Base(c.Src))
	}

	if err := os.MkdirAll(dstDir, 0o777); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dstDir, err)
	}

	return filepath.WalkDir(actualSrc, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(actualSrc, path)
		if err != nil {
			return err
		}

		if d.IsDir() {
			// TODO: change when adding c.Mode.
			return os.MkdirAll(filepath.Join(dstDir, relPath), 0o777)
		}

		return copySingleFile(path, filepath.Join(dstDir, relPath))
	})
}

func (c *Copy) copyContent() error {
	if err := os.WriteFile(c.Dest, []byte(c.Content), 0o666); err != nil {
		return fmt.Errorf("failed to write content to %s: %w", c.Dest, err)
	}
	return nil
}

func (c *Copy) copyFile(actualSrc string) error {
	if strings.HasSuffix(c.Dest, "/") {
		if err := os.Mkdir(c.Dest, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		return copySingleFile(actualSrc, filepath.Join(c.Dest, c.Src))
	}

	d, err := os.Stat(c.Dest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return copySingleFile(actualSrc, c.Dest)
		}
		return err
	}

	if d.IsDir() {
		return copySingleFile(actualSrc, filepath.Join(c.Dest, c.Src))
	}

	return copySingleFile(actualSrc, c.Dest)
}

func (c *Copy) Apply(_ context.Context, parentPath string, _ bool) (Result, error) {
	if c.Content != "" {
		return &CopyResult{}, c.copyContent()
	}

	srcPath := filepath.Join(parentPath, "files", c.Src)
	f, err := os.Stat(srcPath)
	if err != nil {
		return &CopyResult{}, fmt.Errorf("failed to read %s: %w", c.Src, err)
	}

	if f.IsDir() {
		err = c.copyDir(srcPath)
	} else {
		err = c.copyFile(srcPath)
	}

	if err != nil {
		return &CopyResult{}, fmt.Errorf("failed to copy %s to %s: %w", c.Src, c.Dest, err)
	}

	return &CopyResult{}, nil
}
