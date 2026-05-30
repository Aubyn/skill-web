package syncer

import (
	"fmt"
	"os"
	"path/filepath"
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

// Sync deploys skills from storeDir to target dirs.
// Each skill is a symlink: targetDir/<skillName> -> storeDir/<skillName>.
func Sync(storeDir string, skillIDs []string, targetPaths []string) (*Result, error) {
	// Create backup dir
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("2006-01-02T150405")
	backupDir := filepath.Join(home, ".skill-web", "backups", ts)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup dir: %w", err)
	}

	res := &Result{
		Results:        make([]TargetResult, 0, len(targetPaths)),
		ResolvedSkills: skillIDs,
		BackupPath:     backupDir,
	}

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
		for _, e := range existing {
			// Check if it's a symlink we manage
			fullPath := filepath.Join(targetPath, e.Name())
			if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
				// Backup
				backupPath := filepath.Join(backupDir, filepath.Base(targetPath), e.Name())
				os.MkdirAll(filepath.Dir(backupPath), 0755)
				os.Rename(fullPath, backupPath)
				tr.RemovedOld = append(tr.RemovedOld, e.Name())
			}
		}

		// Create symlinks for each skill
		for _, sid := range skillIDs {
			src := filepath.Join(storeDir, sid)
			dst := filepath.Join(targetPath, sid)

			// Check if source exists
			if _, err := os.Stat(src); os.IsNotExist(err) {
				tr.Failed = append(tr.Failed, sid+" (source not found)")
				continue
			}

			// Remove existing if any
			os.Remove(dst)

			// Create symlink
			// Use absolute path for reliable linking
			absSrc, err := filepath.Abs(src)
			if err != nil {
				tr.Failed = append(tr.Failed, sid+" (path error)")
				continue
			}

			if err := os.Symlink(absSrc, dst); err != nil {
				tr.Failed = append(tr.Failed, sid)
				continue
			}
			tr.Synced = append(tr.Synced, sid)
		}

		res.TotalSynced += len(tr.Synced)
		res.Results = append(res.Results, tr)
	}

	return res, nil
}
