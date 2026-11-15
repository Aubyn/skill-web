//go:build !windows

package syncer

import (
	"os"
	"path/filepath"
)

// linkDir creates hard links for every file in src at dst.
// Directories are created as needed; symlinks in source are re-created as symlinks.
func linkDir(src, dst string) error {
	os.RemoveAll(dst)

	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, fi.Mode())
		}

		destPath := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(destPath, fi.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Hard link regular files; re-create symlinks (hard links on symlinks are not useful).
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, destPath)
		}

		return os.Link(path, destPath)
	})
}
