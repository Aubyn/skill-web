//go:build windows

package syncer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// linkDir creates a directory junction at dst pointing to src.
// Junctions are the Windows directory-link equivalent and do not require the
// privileges that symbolic links do.
func linkDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absDst), 0755); err != nil {
		return err
	}

	out, err := exec.Command("cmd", "/c", "mklink", "/J", absDst, absSrc).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("mklink /J: %w: %s", err, msg)
		}
		return fmt.Errorf("mklink /J: %w", err)
	}
	return nil
}
