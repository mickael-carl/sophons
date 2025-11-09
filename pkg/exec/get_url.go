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
)

//	@meta {
//	  "deviations": [
//	    "`url` doesn't support `ftp` or `file` schemes."
//	  ]
//	}
type GetURL struct {
	Dest  string `sophons:"implemented"`
	URL   string `sophons:"implemented"`
	Group string `sophons:"implemented"`
	Mode  any    `sophons:"implemented"`
	Owner string `sophons:"implemented"`

	Attributes          string
	Backup              bool
	Checksum            string
	Ciphers             []string
	ClientCert          string `yaml:"client_cert"`
	ClientKey           string `yaml:"client_key"`
	Decompress          *bool
	Force               *bool
	ForceBasicAuth      bool `yaml:"force_basic_auth"`
	Headers             map[string]string
	Selevel             string
	Serole              string
	Setype              string
	Seuser              string
	Timeout             uint64
	TmpDest             string   `yaml:"tmp_dest"`
	UnredirectedHeaders []string `yaml:"unredirected_headers"`
	UnsafeWrites        bool     `yaml:"unsafe_writes"`
	URLPassword         string   `yaml:"url_password"`
	URLUsername         string   `yaml:"url_username"`
	UseGSSAPI           bool     `yaml:"use_gssapi"`
	UseNetRC            *bool    `yaml:"use_netrc"`
	UseProxy            *bool    `yaml:"use_proxy"`
	ValidateCerts       *bool    `yaml:"validate_certs"`
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
	if g.URL == "" {
		return errors.New("url is required")
	}

	if g.Dest == "" {
		return errors.New("dest is required")
	}

	if _, err := url.Parse(g.URL); err != nil {
		return fmt.Errorf("invalid URL provided")
	}
	return nil
}

func (g *GetURL) Apply(_ context.Context, parentPath string, _ bool) error {
	resp, err := http.Get(g.URL)
	if err != nil {
		return fmt.Errorf("failed to get URL %s: %w", g.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status getting URL %s: %s", g.URL, resp.Status)
	}

	d, err := os.Stat(g.Dest)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat %s: %w", g.Dest, err)
	}

	actualDest := g.Dest
	if err == nil && d.IsDir() {
		actualDest, err = dirDest(resp.Header, g.URL, g.Dest)
		if err != nil {
			return fmt.Errorf("failed to determine path from dest: %w", err)
		}
	}

	out, err := os.Create(actualDest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", g.Dest, err)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		out.Close()
		return fmt.Errorf("failed to write to file %s: %w", g.Dest, err)
	}

	if err := out.Close(); err != nil {
		return err
	}

	if g.Mode == nil && g.Owner == "" && g.Group == "" {
		return nil
	}

	uid, err := util.GetUid(g.Owner)
	if err != nil {
		return err
	}

	gid, err := util.GetGid(g.Group)
	if err != nil {
		return err
	}

	if err := util.ApplyModeAndIDs(actualDest, g.Mode, uid, gid); err != nil {
		return fmt.Errorf("failed to apply mode and IDs to %s: %w", actualDest, err)
	}

	return nil
}
