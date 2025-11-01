package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func Tar(src string) (string, error) {
	f, err := os.CreateTemp("", "sophons-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("could not create target file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	err = filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		headerName, err := filepath.Rel(filepath.Dir(src), file)
		if err != nil {
			return err
		}

		var link string
		if fi.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(file)
			if err != nil {
				return fmt.Errorf("could not read symlink %s: %w", file, err)
			}
		}

		header, err := tar.FileInfoHeader(fi, link)
		if err != nil {
			return fmt.Errorf("could not create header for %s: %w", file, err)
		}
		header.Name = headerName

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("could not write header for %s: %w", file, err)
		}

		if fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		fh, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("could not open %s: %w", file, err)
		}
		defer fh.Close()

		if _, err := io.Copy(tw, fh); err != nil {
			return fmt.Errorf("could not copy data for %s: %w", file, err)
		}

		return nil
	})

	return f.Name(), err
}

func Untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source file: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("could not create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("could not create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("could not create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("could not create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("could not write file %s: %w", target, err)
			}
			outFile.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("could not create parent directory for symlink: %w", err)
			}

			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("could not create symlink %s -> %s: %w", target, header.Linkname, err)
			}

		default:
			fmt.Printf("Skipping unsupported type: %v in %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}
