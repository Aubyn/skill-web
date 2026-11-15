package syncer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type TargetResult struct {
	Target     string   `json:"target"`
	Synced     []string `json:"synced"`
	Failed     []string `json:"failed"`
	RemovedOld []string `json:"removed_old"`
}

type Result struct {
	Results        []TargetResult `json:"results"`
	ResolvedSkills []string       `json:"resolved_skills"`
	BackupPath     string         `json:"backup_path"`
	TotalSynced    int            `json:"total_synced"`
	Mode           string         `json:"mode"`
}

// Mode selects how skills are deployed into target directories.
type Mode string

const (
	// ModeLink links to the library: hard links on Unix, junctions on Windows.
	ModeLink Mode = "link"
	// ModeCopy copies files so targets are independent of the library.
	ModeCopy Mode = "copy"
)

func normalizeMode(mode Mode) Mode {
	if mode == ModeCopy {
		return ModeCopy
	}
	return ModeLink
}

// SkillRef points at one skill in the library. Name is the directory name
// deployed into targets; StorePath is the source directory to hard-link from.
type SkillRef struct {
	ID        string
	Name      string
	StorePath string
}

type PreviewConflict struct {
	Name     string   `json:"name"`
	Winner   string   `json:"winner"`
	SkillIDs []string `json:"skill_ids"`
}

type PreviewTarget struct {
	Target     string   `json:"target"`
	Exists     bool     `json:"exists"`
	Planned    []string `json:"planned"`
	RemovedOld []string `json:"removed_old"`
	Failed     []string `json:"failed"`
}

type PreviewResult struct {
	Results        []PreviewTarget   `json:"results"`
	ResolvedSkills []string          `json:"resolved_skills"`
	Conflicts      []PreviewConflict `json:"conflicts"`
	TotalPlanned   int               `json:"total_planned"`
	MissingTargets []string          `json:"missing_targets"`
}

// Preview computes what Sync would do without touching the filesystem.
func Preview(skills []SkillRef, targetPaths []string) *PreviewResult {
	res := &PreviewResult{
		Results:        make([]PreviewTarget, 0, len(targetPaths)),
		ResolvedSkills: make([]string, 0, len(skills)),
		Conflicts:      []PreviewConflict{},
		MissingTargets: []string{},
	}

	// A flat target directory can hold only one skill per name. Detect duplicate
	// names across the selected skill groups once, independent of target count.
	byName := map[string][]string{}
	nameOrder := []string{}
	for _, sk := range skills {
		name := skillName(sk)
		res.ResolvedSkills = append(res.ResolvedSkills, name)
		if _, ok := byName[name]; !ok {
			nameOrder = append(nameOrder, name)
		}
		byName[name] = append(byName[name], sk.ID)
	}
	for _, name := range nameOrder {
		ids := byName[name]
		if len(ids) > 1 {
			res.Conflicts = append(res.Conflicts, PreviewConflict{Name: name, Winner: ids[0], SkillIDs: ids})
		}
	}
	sort.Slice(res.Conflicts, func(i, j int) bool { return res.Conflicts[i].Name < res.Conflicts[j].Name })

	for _, targetPath := range targetPaths {
		pt := PreviewTarget{
			Target:     targetPath,
			Planned:    []string{},
			RemovedOld: []string{},
			Failed:     []string{},
		}

		if info, err := os.Stat(targetPath); err == nil && info.IsDir() {
			pt.Exists = true
			entries, _ := os.ReadDir(targetPath)
			for _, e := range entries {
				if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
					pt.RemovedOld = append(pt.RemovedOld, e.Name())
				}
			}
		} else {
			res.MissingTargets = append(res.MissingTargets, targetPath)
		}

		usedNames := map[string]string{}
		for _, sk := range skills {
			name := skillName(sk)
			if prev, ok := usedNames[name]; ok {
				// Duplicate name: Sync skips it and reports a failure.
				pt.Failed = append(pt.Failed, fmt.Sprintf("%s (name conflict with %s)", name, prev))
				continue
			}
			usedNames[name] = sk.ID

			if sk.StorePath == "" {
				pt.Failed = append(pt.Failed, name+" (source not found)")
				continue
			}
			if _, err := os.Stat(sk.StorePath); err != nil {
				pt.Failed = append(pt.Failed, name+" (source not found)")
				continue
			}
			pt.Planned = append(pt.Planned, name)
		}

		res.TotalPlanned += len(pt.Planned)
		res.Results = append(res.Results, pt)
	}

	return res
}

// Sync deploys skills from the library to target dirs.
// Link mode hard-links on Unix and creates junctions on Windows; copy mode
// copies files so each target is an independent copy.
func Sync(skills []SkillRef, targetPaths []string, mode Mode) (*Result, error) {
	mode = normalizeMode(mode)
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("2006-01-02T150405")
	backupDir := filepath.Join(home, ".skill-web", "backups", ts)

	resolved := make([]string, 0, len(skills))
	for _, sk := range skills {
		resolved = append(resolved, skillName(sk))
	}

	res := &Result{
		Results:        make([]TargetResult, 0, len(targetPaths)),
		ResolvedSkills: resolved,
		BackupPath:     backupDir,
		Mode:           string(mode),
	}

	var backedUpTargets []string // targets that had content backed up

	for _, targetPath := range targetPaths {
		tr := TargetResult{
			Target:     targetPath,
			Synced:     []string{},
			Failed:     []string{},
			RemovedOld: []string{},
		}

		// Ensure target dir exists
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			tr.Failed = append(tr.Failed, fmt.Sprintf("mkdir target: %v", err))
			res.Results = append(res.Results, tr)
			continue
		}

		// List existing entries in target
		existing, _ := os.ReadDir(targetPath)

		// Backup non-empty targets only
		if len(existing) > 0 {
			// Lazy-create backup dir (only if there's content to back up)
			if err := os.MkdirAll(backupDir, 0755); err != nil {
				tr.Failed = append(tr.Failed, fmt.Sprintf("create backup dir: %v", err))
				res.Results = append(res.Results, tr)
				continue
			}

			for _, e := range existing {
				fullPath := filepath.Join(targetPath, e.Name())
				if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
					// Backup
					backupPath := filepath.Join(backupDir, filepath.Base(targetPath), e.Name())
					os.MkdirAll(filepath.Dir(backupPath), 0755)
					os.Rename(fullPath, backupPath)
					tr.RemovedOld = append(tr.RemovedOld, e.Name())
				}
			}
			backedUpTargets = append(backedUpTargets, filepath.Base(targetPath))
		}

		// Create hard links for each skill (file-by-file)
		usedNames := map[string]string{}
		for _, sk := range skills {
			name := skillName(sk)
			dst := filepath.Join(targetPath, name)

			// Target directory names are flat: reject duplicate names across groups.
			if prev, ok := usedNames[name]; ok {
				tr.Failed = append(tr.Failed, fmt.Sprintf("%s (name conflict with %s)", name, prev))
				continue
			}
			usedNames[name] = sk.ID

			// Check if source exists
			if sk.StorePath == "" {
				tr.Failed = append(tr.Failed, name+" (source not found)")
				continue
			}
			if _, err := os.Stat(sk.StorePath); os.IsNotExist(err) {
				tr.Failed = append(tr.Failed, name+" (source not found)")
				continue
			}

			// Remove existing if any
			os.RemoveAll(dst)

			if err := linkOrCopy(sk.StorePath, dst, mode); err != nil {
				tr.Failed = append(tr.Failed, name)
				continue
			}
			tr.Synced = append(tr.Synced, name)
		}

		res.TotalSynced += len(tr.Synced)
		res.Results = append(res.Results, tr)
	}

	// Prune old backups: keep 10 newest per target
	if len(backedUpTargets) > 0 {
		pruneBackups(filepath.Dir(backupDir), backedUpTargets)
	}

	return res, nil
}

// skillName returns the flat directory name to deploy for a skill.
func skillName(sk SkillRef) string {
	name := sk.Name
	if name == "" {
		name = filepath.Base(sk.StorePath)
	}
	if name == "" || name == "." {
		name = sk.ID
	}
	return name
}

// linkOrCopy deploys one skill directory using the selected mode.
func linkOrCopy(src, dst string, mode Mode) error {
	if mode == ModeCopy {
		return copyDir(src, dst)
	}
	return linkDir(src, dst)
}

// copyDir recursively copies files and recreates symlinks.
func copyDir(src, dst string) error {
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

		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, destPath)
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	})
}

// pruneBackups keeps at most 10 backup versions per target, deleting older ones.
// backupsParent is ~/.skill-web/backups/; targetBases are target basenames that were just backed up.
func pruneBackups(backupsParent string, targetBases []string) {
	entries, err := os.ReadDir(backupsParent)
	if err != nil {
		return
	}

	// Sort by name ascending (oldest first, since timestamps are lexicographically ordered)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, targetBase := range targetBases {
		// Collect indices of backup dirs containing this target
		var containing []int
		for i, e := range entries {
			if !e.IsDir() {
				continue
			}
			targetDir := filepath.Join(backupsParent, e.Name(), targetBase)
			if _, err := os.Stat(targetDir); err == nil {
				containing = append(containing, i)
			}
		}

		// Keep 10 newest (last in sorted order), delete the rest
		keep := 10
		if len(containing) > keep {
			toDelete := containing[:len(containing)-keep]
			for _, idx := range toDelete {
				dir := filepath.Join(backupsParent, entries[idx].Name(), targetBase)
				os.RemoveAll(dir)
			}
		}
	}
}
