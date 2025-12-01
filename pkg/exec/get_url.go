package exec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mickael-carl/sophons/pkg/exec/util"
	"github.com/mickael-carl/sophons/pkg/proto"
)

//	@meta {
//	  "deviations": [
//	    "`url` doesn't support `ftp` or `file` schemes."
//	  ]
//	}
type GetURL struct {
	proto.GetURL `yaml:",inline"`
}

type GetURLResult struct {
	CommonResult `yaml:",inline"`
}

func init() {
	RegisterTaskType("get_url", func() TaskContent { return &GetURL{} })
	RegisterTaskType("ansible.builtin.get_url", func() TaskContent { return &GetURL{} })
}

// filenameFromHeader extracts filename from Content-Disposition if present.
func filenameFromHeader(h http.Header) string {
	cd := h.Get("Content-Disposition")
	if cd == "" {
		return ""
	}

	key := "filename="
	i := strings.Index(cd, key)
	if i == -1 {
		return ""
	}

	// TODO: maybe don't assume that `filename` is the last key.
	filename := cd[i+len(key):]
	filename = strings.Trim(filename, "\"'; ")
	return filepath.Base(filename)
}

func dirDest(h http.Header, src, dest string) (string, error) {
	if filename := filenameFromHeader(h); filename != "" {
		return filepath.Join(dest, filename), nil
	}

	pURL, err := url.Parse(src)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	if pURL.Path != "" {
		return filepath.Join(dest, path.Base(pURL.Path)), nil
	}

	return filepath.Join(dest, "index.html"), nil
}

func (g *GetURL) Validate() error {
	if g.Url == "" {
		return errors.New("url is required")
	}

	if g.Dest == "" {
		return errors.New("dest is required")
	}

	if _, err := url.Parse(g.Url); err != nil {
		return fmt.Errorf("invalid URL provided")
	}
	return nil
}

func (g *GetURL) Apply(_ context.Context, parentPath string, _ bool) (Result, error) {
	resp, err := http.Get(g.Url)
	if err != nil {
		return &GetURLResult{}, fmt.Errorf("failed to get URL %s: %w", g.Url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &GetURLResult{}, fmt.Errorf("unexpected status getting.Url %s: %s", g.Url, resp.Status)
	}

	d, err := os.Stat(g.Dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return &GetURLResult{}, fmt.Errorf("failed to stat %s: %w", g.Dest, err)
	}

	actualDest := g.Dest
	if err == nil && d.IsDir() {
		actualDest, err = dirDest(resp.Header, g.Url, g.Dest)
		if err != nil {
			return &GetURLResult{}, fmt.Errorf("failed to determine path from dest: %w", err)
		}
	}

	out, err := os.Create(actualDest)
	if err != nil {
		return &GetURLResult{}, fmt.Errorf("failed to create file %s: %w", g.Dest, err)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		out.Close()
		return &GetURLResult{}, fmt.Errorf("failed to write to file %s: %w", g.Dest, err)
	}

	if err := out.Close(); err != nil {
		return &GetURLResult{}, err
	}

	if g.Mode == nil && g.Owner == "" && g.Group == "" {
		return &GetURLResult{}, nil
	}

	uid, err := util.GetUid(g.Owner)
	if err != nil {
		return &GetURLResult{}, err
	}

	gid, err := util.GetGid(g.Group)
	if err != nil {
		return &GetURLResult{}, err
	}

	if err := util.ApplyModeAndIDs(actualDest, g.Mode.GetValue(), uid, gid); err != nil {
		return &GetURLResult{}, fmt.Errorf("failed to apply mode and IDs to %s: %w", actualDest, err)
	}

	return &GetURLResult{}, nil
}
