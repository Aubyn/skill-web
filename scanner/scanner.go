package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
)

type Entry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SkillType   string `json:"skill_type"`
	Size        string `json:"size"`
	Description string `json:"description"`
	SourcePath  string `json:"-"`
}

// Scan discovers skills recursively from dir. Each skill is a directory containing SKILL.md
// (or <dir>/<dir>.md). If dir itself is a skill, only that root skill is returned.
// rootFallback names the root skill when its frontmatter has no usable name
// (for zip imports this is the package name, not the temp directory name).
func Scan(dir string, rootFallback string, maxDepth int) ([]Entry, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("stat dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}

	entries := []Entry{}

	// Root is a skill: take only the root skill, never recurse into it.
	if isSkillDir(dir) {
		entries = append(entries, makeEntry(dir, rootFallback))
		return entries, nil
	}

	seen := map[string]bool{}
	scanDir(dir, 0, maxDepth, seen, &entries)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
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

func makeEntry(dir string, fallback string) Entry {
	base := filepath.Base(dir)
	meta := readMeta(dir, base)

	name := strings.TrimSpace(meta.name)
	if !safeName(name) {
		name = strings.TrimSpace(fallback)
		if !safeName(name) {
			name = base
		}
	}

	entry := Entry{
		ID:          name,
		Name:        name,
		SkillType:   "dir",
		Description: meta.description,
		SourcePath:  dir,
	}

	var size int64
	filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && fi != nil {
			size += fi.Size()
		}
		return nil
	})
	entry.Size = humanize.Bytes(uint64(size))

	return entry
}

func scanDir(dir string, depth, maxDepth int, seen map[string]bool, entries *[]Entry) {
	if depth > maxDepth {
		return
	}

	// Check if this directory itself is a recognized skill
	if isSkillDir(dir) {
		entry := makeEntry(dir, "")

		// Keep the first skill with a given name within this scan.
		if !seen[entry.Name] {
			seen[entry.Name] = true
			*entries = append(*entries, entry)
		}

		// Continue recursion to find sub-skills
		scanSubdirs(dir, depth+1, maxDepth, seen, entries)
		return
	}

	// Not a skill directory, scan children
	scanSubdirs(dir, depth+1, maxDepth, seen, entries)
}

func scanSubdirs(dir string, depth, maxDepth int, seen map[string]bool, entries *[]Entry) {
	if depth > maxDepth {
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return // skip unreadable dirs
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

		scanDir(subPath, depth, maxDepth, seen, entries)
	}
}

type meta struct {
	name        string
	description string
}

// readMeta reads frontmatter from SKILL.md, falling back to <base>.md.
func readMeta(dir, base string) meta {
	candidates := []string{
		filepath.Join(dir, "SKILL.md"),
		filepath.Join(dir, base+".md"),
	}
	for _, path := range candidates {
		if m, ok := parseMeta(path); ok {
			return m
		}
	}
	return meta{}
}

// parseMeta reads a small YAML frontmatter subset: name and description.
func parseMeta(path string) (meta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return meta{}, false
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024), 1024*1024)

	if !sc.Scan() || strings.TrimSpace(sc.Text()) != "---" {
		return meta{}, false
	}

	var lines []string
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		lines = append(lines, line)
	}

	var m meta
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		switch {
		case strings.HasPrefix(line, "name:"):
			if m.name == "" {
				m.name = trimScalar(strings.TrimSpace(line[len("name:"):]))
			}
		case strings.HasPrefix(line, "description:"):
			if m.description != "" {
				continue
			}
			val := trimScalar(strings.TrimSpace(line[len("description:"):]))
			if isBlockScalar(val) {
				var parts []string
				for j := i + 1; j < len(lines); j++ {
					cont := lines[j]
					if strings.TrimSpace(cont) == "" || (cont[0] != ' ' && cont[0] != '\t') {
						break
					}
					parts = append(parts, strings.TrimSpace(cont))
					i = j
				}
				val = strings.Join(parts, " ")
			}
			m.description = val
		}
	}

	return m, true
}

func trimScalar(s string) string {
	return strings.Trim(s, ` "'`)
}

func isBlockScalar(s string) bool {
	switch s {
	case "", ">", "|", ">-", "|-", ">+", "|+":
		return true
	}
	return false
}

func safeName(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if r == '/' || r == '\\' || r < 32 {
			return false
		}
	}
	return true
}
