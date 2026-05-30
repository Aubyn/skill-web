package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
)

type Entry struct {
	ID         string `json:"id"`
	SkillType  string `json:"skill_type"`
	Size       string `json:"size"`
	SourcePath string `json:"-"`
}

// Scan discovers skills recursively from dir. Each skill is a directory containing SKILL.md.
// existing is a set of already-imported skill IDs for detecting conflicts.
func Scan(dir string, existing map[string]bool, maxDepth int) ([]Entry, []string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("abs path: %w", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("stat dir: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("not a directory: %s", dir)
	}

	entries := []Entry{}
	conflicts := []string{}
	seen := map[string]bool{}

	err = scanDir(dir, 0, maxDepth, existing, seen, &entries, &conflicts)
	if err != nil {
		return nil, nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	return entries, conflicts, nil
}

func isSkillDir(dir string) bool {
	// A directory is a skill if it contains SKILL.md
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		return true
	}
	// Or if it contains a .md file with the same name as the directory
	// (Reasonix convention: <dir>/<dir>.md)
	base := filepath.Base(dir)
	if _, err := os.Stat(filepath.Join(dir, base+".md")); err == nil {
		return true
	}
	return false
}

func scanDir(dir string, depth, maxDepth int, existing map[string]bool, seen map[string]bool, entries *[]Entry, conflicts *[]string) error {
	if depth > maxDepth {
		return nil
	}

	// Check if this directory itself is a recognized skill
	if isSkillDir(dir) {
		id := filepath.Base(dir)

		// Check for duplicates within this scan
		if seen[id] {
			return nil // already added from another path
		}
		seen[id] = true

		entry := Entry{
			ID:         id,
			SkillType:  "dir",
			SourcePath: dir,
		}
		// Calculate size
		var size int64
		filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
			if err == nil && fi != nil {
				size += fi.Size()
			}
			return nil
		})
		entry.Size = humanize.Bytes(uint64(size))

		*entries = append(*entries, entry)

		if existing[id] {
			*conflicts = append(*conflicts, id)
		}

		// Continue recursion to find sub-skills
		return scanSubdirs(dir, depth+1, maxDepth, existing, seen, entries, conflicts)
	}

	// Not a skill directory, scan children
	return scanSubdirs(dir, depth+1, maxDepth, existing, seen, entries, conflicts)
}

func scanSubdirs(dir string, depth, maxDepth int, existing map[string]bool, seen map[string]bool, entries *[]Entry, conflicts *[]string) error {
	if depth > maxDepth {
		return nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil // skip unreadable dirs
	}

	for _, f := range files {
		// Skip hidden directories
		if strings.HasPrefix(f.Name(), ".") && f.Name() != "." {
			continue
		}

		subPath := filepath.Join(dir, f.Name())

		// Accept regular directories OR symlinks pointing to directories
		if f.IsDir() {
			// regular directory — proceed
		} else if f.Type()&os.ModeSymlink != 0 {
			// symlink — check if it points to a directory
			info, err := os.Stat(subPath)
			if err != nil || !info.IsDir() {
				continue
			}
		} else {
			continue // regular file
		}

		if err := scanDir(subPath, depth, maxDepth, existing, seen, entries, conflicts); err != nil {
			// continue scanning other dirs
			continue
		}
	}
	return nil
}
