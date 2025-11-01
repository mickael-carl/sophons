package exec

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//	@meta {
//	  "deviations": [
//	    "`url` doesn't support `ftp` or `file` schemes."
//	  ]
//	}
type GetURL struct {
	Attributes          jinjaString
	Backup              bool
	Checksum            jinjaString
	Ciphers             []jinjaString
	ClientCert          jinjaString `yaml:"client_cert"`
	ClientKey           jinjaString `yaml:"client_key"`
	Decompress          *bool
	Dest                jinjaString `sophons:"implemented"`
	Force               *bool
	ForceBasicAuth      bool `yaml:"force_basic_auth"`
	Group               jinjaString
	Headers             map[jinjaString]jinjaString
	Mode                jinjaString
	Owner               jinjaString
	Selevel             jinjaString
	Serole              jinjaString
	Setype              jinjaString
	Seuser              jinjaString
	Timeout             uint64
	TmpDest             jinjaString   `yaml:"tmp_dest"`
	UnredirectedHeaders []jinjaString `yaml:"unredirected_headers"`
	UnsafeWrites        bool          `yaml:"unsafe_writes"`
	URL                 jinjaString   `sophons:"implemented"`
	URLPassword         jinjaString   `yaml:"url_password"`
	URLUsername         jinjaString   `yaml:"url_username"`
	UseGSSAPI           bool          `yaml:"use_gssapi"`
	UseNetRC            *bool         `yaml:"use_netrc"`
	UseProxy            *bool         `yaml:"use_proxy"`
	ValidateCerts       *bool         `yaml:"validate_certs"`
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

func (c *GetURL) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}

	if c.Dest == "" {
		return errors.New("dest is required")
	}

	if _, err := url.Parse(string(c.URL)); err != nil {
		return fmt.Errorf("invalid URL provided: %w", err)
	}
	return nil
}

func (c *GetURL) Apply(parentPath string, _ bool) error {
	resp, err := http.Get(string(c.URL))
	if err != nil {
		return fmt.Errorf("failed to get URL %s: %w", c.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status getting URL %s: %s", c.URL, resp.Status)
	}

	d, err := os.Stat(string(c.Dest))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat %s: %w", c.Dest, err)
	}

	actualDest := string(c.Dest)
	if err == nil && d.IsDir() {
		actualDest, err = dirDest(resp.Header, string(c.URL), string(c.Dest))
		if err != nil {
			return fmt.Errorf("failed to determine path from dest: %w", err)
		}
	}

	out, err := os.Create(actualDest)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", c.Dest, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", c.Dest, err)
	}

	return nil
}
