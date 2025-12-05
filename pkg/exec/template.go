package exec

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nikolalohinski/gonja/v2"
	gonjaexec "github.com/nikolalohinski/gonja/v2/exec"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/registry"
	"github.com/mickael-carl/sophons/pkg/variables"
)

//	@meta {
//	  "deviations": ["`src` doesn't support absolute paths."]
//	}
type Template struct {
	*proto.Template `yaml:",inline"`
}

type TemplateResult struct {
	Checksum string
	Dest     string
	Gid      uint64
	Group    string
	MD5Sum   string
	Mode     string
	Owner    string
	Size     uint64
	// We can't support this: Ansible fills it in with the copied template file
	// on the target node but our execution model is fundamentally not working
	// like that.
	// Src      string
	Uid uint64

	CommonResult `yaml:",inline"`
}

func init() {
	reg := registry.TaskRegistration{
		ProtoFactory: func() any { return &proto.Template{} },
		ProtoWrapper: func(msg any) any { return &proto.Task_Template{Template: msg.(*proto.Template)} },
		ExecAdapter: func(content any) any {
			if c, ok := content.(*proto.Task_Template); ok {
				return &Template{Template: c.Template}
			}
			return nil
		},
	}
	registry.Register("template", reg, (*proto.Task_Template)(nil))
	registry.Register("ansible.builtin.template", reg, (*proto.Task_Template)(nil))
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

func (c *Template) Apply(ctx context.Context, parentPath string, isRole bool) (Result, error) {
	result := TemplateResult{}

	srcPath := filepath.Join(parentPath, "templates", c.Src)
	template, err := gonja.FromFile(srcPath)
	if err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to read template file %s: %w", srcPath, err)
	}

	vars, ok := variables.FromContext(ctx)
	if !ok {
		vars = variables.Variables{}
	}
	varsCtx := gonjaexec.NewContext(vars)

	var buf bytes.Buffer
	if err := template.Execute(&buf, varsCtx); err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to template file %s: %w", srcPath, err)
	}
	renderedContent := buf.Bytes()

	checksum := sha1.Sum(renderedContent)
	result.Checksum = hex.EncodeToString(checksum[:])

	md5sum := md5.Sum(renderedContent)
	renderedMD5 := hex.EncodeToString(md5sum[:])

	stat, err := os.Stat(c.Dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to state %s: %w", srcPath, err)
	}

	if err == nil && stat.Mode().IsRegular() {
		existingContent, err := os.ReadFile(c.Dest)
		if err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to read from %s: %w", srcPath, err)
		}
		existingmd5sum := md5.Sum(existingContent)
		existingMD5 := hex.EncodeToString(existingmd5sum[:])

		if existingMD5 == renderedMD5 {
			result.TaskSkipped()
			return &result, nil
		}
	}

	if err := os.WriteFile(c.Dest, renderedContent, 0o644); err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to write destination %s: %w", c.Dest, err)
	}
	result.TaskChanged()

	// MD5Sum is only populated when changed.
	result.MD5Sum = renderedMD5

	if c.Mode != nil && c.Mode.Value != "" || c.Owner != "" || c.Group != "" {
		uid, err := util.GetUid(c.Owner)
		if err != nil {
			result.TaskFailed()
			return &result, err
		}

		gid, err := util.GetGid(c.Group)
		if err != nil {
			result.TaskFailed()
			return &result, err
		}

		var modeValue any
		if c.Mode != nil {
			modeValue = c.Mode.Value
		}
		if err := util.ApplyModeAndIDs(c.Dest, modeValue, uid, gid); err != nil {
			result.TaskFailed()
			return &result, fmt.Errorf("failed to apply mode and IDs to %s: %w", c.Dest, err)
		}

		// Populate result fields with explicitly set values.
		if c.Owner != "" {
			result.Owner = c.Owner
			if uid != -1 {
				result.Uid = uint64(uid)
			}
		}
		if c.Group != "" {
			result.Group = c.Group
			if gid != -1 {
				result.Gid = uint64(gid)
			}
		}
	}

	stat, err = os.Stat(c.Dest)
	if err != nil {
		result.TaskFailed()
		return &result, fmt.Errorf("failed to stat %s: %w", c.Dest, err)
	}

	// Populate common fields only on success.
	result.Dest = c.Dest
	result.Size = uint64(stat.Size())

	// Only populate mode if it was explicitly set.
	if c.Mode != nil && c.Mode.Value != "" {
		result.Mode = fmt.Sprintf("%04o", stat.Mode().Perm())
	}

	return &result, nil
}
