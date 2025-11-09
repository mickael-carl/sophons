package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	"github.com/mickael-carl/sophons/pkg/variables"
	"github.com/nikolalohinski/gonja/v2"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"
)

//	@meta {
//	  "deviations": ["`src` doesn't support absolute paths."]
//	}
type Template struct {
	Dest  string `sophons:"implemented"`
	Group string `sophons:"implemented"`
	Mode  string `sophons:"implemented"`
	Owner string `sophons:"implemented"`
	Src   string `sophons:"implemented"`

	Attributes          string
	Backup              bool
	BlockEndString      string `yaml:"block_end_string"`
	BlockStartString    string `yaml:"block_start_string"`
	CommentEndString    string `yaml:"comment_end_string"`
	CommentStartString  string `yaml:"comment_start_string"`
	Follow              bool
	Force               *bool
	LStripBlocks        bool   `yaml:"lstrip_blocks"`
	NewlineSequence     string `yaml:"newline_sequence"`
	OutputEncoding      string `yaml:"output_encoding"`
	Selevel             string
	Serole              string
	Setype              string
	Seuser              string
	TrimBlocks          *bool  `yaml:"trim_blocks"`
	UnsafeWrites        bool   `yaml:"unsafe_writes"`
	AValidate           string `yaml:"validate"`
	VariableEndString   string `yaml:"variable_end_string"`
	VariableStartString string `yaml:"variable_start_string"`
}

func init() {
	RegisterTaskType("template", func() TaskContent { return &Template{} })
	RegisterTaskType("ansible.builtin.template", func() TaskContent { return &Template{} })
}

func (c *Template) Validate() error {
	// We don't support copying random files from the controller, as it seems
	// like a bad idea. All files should belong in a role. This might change
	// eventually, provided there is a genuinely good usecase for it.
	if filepath.IsAbs(c.Src) {
		return errors.New("template from an absolute path is not supported")
	}

	if c.Src == "" {
		return errors.New("src is required")
	}

	if c.Dest == "" {
		return errors.New("dest is required")
	}

	return nil
}

func (c *Template) Apply(ctx context.Context, parentPath string, isRole bool) error {
	f, err := os.Create(c.Dest)
	if err != nil {
		return fmt.Errorf("failed to create destination %s: %w", c.Dest, err)
	}

	srcPath := filepath.Join(parentPath, "templates", c.Src)
	template, err := gonja.FromFile(srcPath)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to read template file %s: %w", srcPath, err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		vars = variables.Variables{}
	}
	varsCtx := gonjaexec.NewContext(vars)

	if err := template.Execute(f, varsCtx); err != nil {
		f.Close()
		return fmt.Errorf("failed to template file %s to %s: %w", srcPath, c.Dest, err)
	}

	if err := f.Close(); err != nil {
		return err
	}

	if c.Mode == "" && c.Owner == "" && c.Group == "" {
		return nil
	}

	uid, err := util.GetUid(c.Owner)
	if err != nil {
		return err
	}

	gid, err := util.GetGid(c.Group)
	if err != nil {
		return err
	}

	if err := util.ApplyModeAndIDs(c.Dest, c.Mode, uid, gid); err != nil {
		return fmt.Errorf("failed to apply mode and IDs to %s: %w", c.Dest, err)
	}

	return nil
}
