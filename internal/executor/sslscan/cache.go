package sslscan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type cache struct{}

func (c *cache) get(target string, key string) (string, []string, bool) {
	path := fmt.Sprintf("./%s/%s", target, key)
	output, err := os.ReadFile(path)
	if err != nil {
		return "", nil, false
	}
	reportPaths := findFileStartsWith("./"+target, "sslReport")
	return string(output), reportPaths, true
}

func findFileStartsWith(root string, pattern string) []string {
	a := make([]string, 0)
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if strings.HasPrefix(d.Name(), pattern) {
			a = append(a, s)
		}
		return nil
	})
	return a
}
