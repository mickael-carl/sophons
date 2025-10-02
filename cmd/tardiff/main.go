package main

import (
	"archive/tar"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/google/go-cmp/cmp"
)

type entry struct {
	Hash     string
	Linkname string
	Size     int64
	Mode     int64
	Uid      int
	Gid      int
	Uname    string
	Gname    string
}

func filterOut(name string) bool {
	if strings.HasPrefix(name, "var/log") {
		return true
	}

	if strings.Contains(name, ".ansible/") {
		return true
	}

	return false
}

func tarToEntries(path string) (map[string]*entry, error) {
	out := map[string]*entry{}

	f, err := os.Open(path)
	if err != nil {
		return map[string]*entry{}, err
	}
	r := tar.NewReader(f)
	h := sha1.New()
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return map[string]*entry{}, err
		}

		if filterOut(header.Name) {
			continue
		}

		h.Reset()
		if _, err := io.Copy(h, r); err != nil {
			return map[string]*entry{}, err
		}
		hash := hex.EncodeToString(h.Sum(nil))

		out["/"+header.Name] = &entry{
			Hash:     hash,
			Linkname: header.Linkname,
			Size:     header.Size,
			Mode:     header.Mode,
			Uid:      header.Uid,
			Gid:      header.Gid,
			Uname:    header.Uname,
			Gname:    header.Gname,
		}
	}

	return out, nil

}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("invalid usage")
	}

	left, err := tarToEntries(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	right, err := tarToEntries(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	onlyLeft := []string{}
	onlyRight := []string{}
	diff := map[string]string{}
	for k, v := range left {
		vRight, ok := right[k]
		if !ok {
			onlyLeft = append(onlyLeft, k)
			continue
		}

		if !cmp.Equal(v, vRight) {
			diff[k] = cmp.Diff(v, vRight)
		}
	}

	for k, _ := range right {
		_, ok := left[k]
		if ok {
			continue
		}

		onlyRight = append(onlyRight, k)
	}

	if len(onlyLeft) > 0 || len(onlyRight) > 0 || len(diff) > 0 {
		fmt.Printf("only in %s: %s\n", os.Args[1], onlyLeft)
		fmt.Printf("only in %s: %s\n", os.Args[2], onlyRight)
		fmt.Printf("diff: %s\n", diff)
		os.Exit(1)
	}
}
