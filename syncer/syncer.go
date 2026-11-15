package syncer

import (
	"fmt"
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
}

// SkillRef points at one skill in the library. Name is the directory name
// deployed into targets; StorePath is the source directory to hard-link from.
type SkillRef struct {
	ID        string
	Name      string
	StorePath string
}

// Sync deploys skills from the library to target dirs.
// Each skill is hard-linked file-by-file from StorePath to targetDir/<name>.
func Sync(skills []SkillRef, targetPaths []string) (*Result, error) {
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

			if err := hardLinkDir(sk.StorePath, dst); err != nil {
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

// hardLinkDir recursively creates hard links for each file in src at dst.
// Directories are created as needed; symlinks in source are re-created as symlinks.
func hardLinkDir(src, dst string) error {
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

		// Create parent dirs
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Hard link regular files; re-create symlinks (hard links on symlinks not useful)
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
