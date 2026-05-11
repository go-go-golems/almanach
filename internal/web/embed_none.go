//go:build !embed

package web

import (
	"io/fs"
	"os"
	"path/filepath"
)

// PublicFS is the on-disk fallback when the embed build tag is not set.
var PublicFS fs.FS

func init() {
	root := findRoot()
	PublicFS = os.DirFS(filepath.Join(root, "internal", "web", "embed", "public"))
}

func findRoot() string {
	dir, _ := os.Getwd()
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "."
}
