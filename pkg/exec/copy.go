package exec

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//	@meta {
//	  "deviations": ["`src` doesn't support absolute paths when `remote_src` is false."]
//	}
type Copy struct {
	Content   jinjaString `sophons:"implemented"`
	RemoteSrc bool        `yaml:"remote_src"`
	Src       jinjaString `sophons:"implemented"`
	Dest      jinjaString `sophons:"implemented"`

	Attributes    jinjaString
	Backup        bool
	Checksum      jinjaString
	Decrypt       *bool
	DirectoryMode jinjaString `yaml:"directory_mode"`
	Follow        bool
	Force         *bool
	Group         jinjaString
	LocalFollow   bool `yaml:"local_follow"`
	Mode          jinjaString
	Owner         jinjaString
	Selevel       jinjaString
	Serole        jinjaString
	Setype        jinjaString
	Seuser        jinjaString
	UnsafeWrites  bool        `yaml:"unsafe_writes"`
	AValidate     jinjaString `yaml:"validate"`
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
	if filepath.IsAbs(string(c.Src)) && !c.RemoteSrc {
		return errors.New("copying from an absolute path without remote_src is not supported")
	}

	if c.Src == "" && c.Content == "" {
		return errors.New("either src or content need to be specified")
	}

	if c.Src != "" && c.Content != "" {
		return errors.New("src and content can't both be specified")
	}

	if c.Content != "" && strings.HasSuffix(string(c.Dest), string(os.PathSeparator)) {
		return errors.New("can't use content when dest is a directory")
	}

	if c.Dest == "" {
		return errors.New("dest is required")
	}

	return nil
}

func (c *Copy) copyDir(actualSrc string) error {
	copyContentsOnly := strings.HasSuffix(actualSrc, string(os.PathSeparator))

	dstDir := string(c.Dest)
	if !copyContentsOnly {
		dstDir = filepath.Join(string(c.Dest), filepath.Base(string(c.Src)))
	}

	if err := os.MkdirAll(dstDir, 0777); err != nil {
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
			return os.MkdirAll(filepath.Join(dstDir, relPath), 0777)
		}

		return copySingleFile(path, filepath.Join(dstDir, relPath))
	})
}

func (c *Copy) copyContent() error {
	if err := os.WriteFile(string(c.Dest), []byte(c.Content), 0666); err != nil {
		return fmt.Errorf("failed to write content to %s: %w", c.Dest, err)
	}
	return nil
}

func (c *Copy) copyFile(actualSrc string) error {
	if strings.HasSuffix(string(c.Dest), "/") {
		if err := os.Mkdir(string(c.Dest), 0777); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
		return copySingleFile(actualSrc, filepath.Join(string(c.Dest), string(c.Src)))
	}

	d, err := os.Stat(string(c.Dest))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return copySingleFile(actualSrc, string(c.Dest))
		}
		return err
	}

	if d.IsDir() {
		return copySingleFile(actualSrc, filepath.Join(string(c.Dest), string(c.Src)))
	}

	return copySingleFile(actualSrc, string(c.Dest))
}

func (c *Copy) Apply(parentPath string, _ bool) error {
	if c.Content != "" {
		return c.copyContent()
	}

	srcPath := filepath.Join(parentPath, "files", string(c.Src))
	f, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", c.Src, err)
	}

	if f.IsDir() {
		err = c.copyDir(srcPath)
	} else {
		err = c.copyFile(srcPath)
	}

	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", c.Src, c.Dest, err)
	}

	return nil
}
