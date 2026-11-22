package syncer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type TargetResult struct {
	Target     string   `json:"target"`
	Synced     []string `json:"synced"`
	Failed     []string `json:"failed"`
	RemovedOld []string `json:"removed_old"`
	Replaced   []string `json:"replaced"`
	Kept       []string `json:"kept"`
}

type Result struct {
	Results        []TargetResult `json:"results"`
	ResolvedSkills []string       `json:"resolved_skills"`
	BackupPath     string         `json:"backup_path"`
	TotalSynced    int            `json:"total_synced"`
	Mode           string         `json:"mode"`
	Strategy       string         `json:"strategy"`
}

// Mode selects how skills are deployed into target directories.
type Mode string

const (
	// ModeLink links to the library: hard links on Unix, junctions on Windows.
	ModeLink Mode = "link"
	// ModeCopy copies files so targets are independent of the library.
	ModeCopy Mode = "copy"
)

// Strategy selects what happens to content already present in a target dir.
type Strategy string

const (
	// StrategyApply backs up and clears the target, then deploys the selection.
	StrategyApply Strategy = "apply"
	// StrategyAppend keeps unrelated target content and replaces same-name skills.
	StrategyAppend Strategy = "append"
)

// ParseStrategy parses a strategy name, defaulting to apply for empty input.
// Case and surrounding spaces are ignored.
func ParseStrategy(raw string) (Strategy, error) {
	switch Strategy(strings.ToLower(strings.TrimSpace(raw))) {
	case "", StrategyApply:
		return StrategyApply, nil
	case StrategyAppend:
		return StrategyAppend, nil
	default:
		return "", fmt.Errorf("invalid strategy %q (want %q or %q)", raw, StrategyApply, StrategyAppend)
	}
}

// Options bundles the deploy knobs so callers do not pass positional flags.
type Options struct {
	Mode     Mode
	Strategy Strategy
}

// normalized fills in defaults: link mode and apply strategy.
func (o Options) normalized() Options {
	strategy := o.Strategy
	if strategy != StrategyAppend {
		strategy = StrategyApply
	}
	return Options{Mode: normalizeMode(o.Mode), Strategy: strategy}
}

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
	Replaced   []string `json:"replaced"`
	Kept       []string `json:"kept"`
	Failed     []string `json:"failed"`
}

type PreviewResult struct {
	Results        []PreviewTarget   `json:"results"`
	ResolvedSkills []string          `json:"resolved_skills"`
	Conflicts      []PreviewConflict `json:"conflicts"`
	TotalPlanned   int               `json:"total_planned"`
	MissingTargets []string          `json:"missing_targets"`
	Strategy       string            `json:"strategy"`
}

// deployPlan is the per-target work list. Preview renders it and Sync executes
// it, so the preview can never drift from what a sync really does.
type deployPlan struct {
	Target  string
	Exists  bool
	Refs    []SkillRef // skills to deploy, in order, names de-duplicated
	Names   []string   // deploy names, parallel to Refs
	Failed  []string
	Keep    []string // entries left untouched (append only)
	Replace []string // existing same-name entries that get backed up and replaced
	Remove  []string // existing entries that get backed up and removed (apply)
}

// planTarget computes what a sync would do to one target directory without
// touching the filesystem.
func planTarget(targetPath string, skills []SkillRef, strategy Strategy) *deployPlan {
	plan := &deployPlan{
		Target:  targetPath,
		Failed:  []string{},
		Keep:    []string{},
		Replace: []string{},
		Remove:  []string{},
	}

	planned := map[string]bool{}
	seen := map[string]string{}
	for _, sk := range skills {
		name := skillName(sk)
		if prev, ok := seen[name]; ok {
			plan.Failed = append(plan.Failed, fmt.Sprintf("%s (name conflict with %s)", name, prev))
			continue
		}
		seen[name] = sk.ID

		if sk.StorePath == "" {
			plan.Failed = append(plan.Failed, name+" (source not found)")
			continue
		}
		if _, err := os.Stat(sk.StorePath); err != nil {
			plan.Failed = append(plan.Failed, name+" (source not found)")
			continue
		}

		plan.Refs = append(plan.Refs, sk)
		plan.Names = append(plan.Names, name)
		planned[name] = true
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return plan // missing or unreadable target: nothing existing to handle
	}
	plan.Exists = true

	for _, e := range entries {
		if !isDeployableEntry(e) {
			continue // regular files in the target are never touched
		}
		if strategy != StrategyAppend {
			plan.Remove = append(plan.Remove, e.Name())
			continue
		}
		if planned[e.Name()] {
			plan.Replace = append(plan.Replace, e.Name())
			continue
		}
		plan.Keep = append(plan.Keep, e.Name())
	}

	return plan
}

// isDeployableEntry reports whether a target entry is something a sync owns:
// directories (skills) and symlinks (legacy skill links).
func isDeployableEntry(e os.DirEntry) bool {
	return e.IsDir() || e.Type()&os.ModeSymlink != 0
}

// Preview computes what Sync would do without touching the filesystem.
func Preview(skills []SkillRef, targetPaths []string, opts Options) *PreviewResult {
	o := opts.normalized()
	res := &PreviewResult{
		Results:        make([]PreviewTarget, 0, len(targetPaths)),
		ResolvedSkills: make([]string, 0, len(skills)),
		Conflicts:      []PreviewConflict{},
		MissingTargets: []string{},
		Strategy:       string(o.Strategy),
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
		plan := planTarget(targetPath, skills, o.Strategy)
		if !plan.Exists {
			res.MissingTargets = append(res.MissingTargets, targetPath)
		}
		res.TotalPlanned += len(plan.Names)
		res.Results = append(res.Results, PreviewTarget{
			Target:     plan.Target,
			Exists:     plan.Exists,
			Planned:    plan.Names,
			RemovedOld: plan.Remove,
			Replaced:   plan.Replace,
			Kept:       plan.Keep,
			Failed:     plan.Failed,
		})
	}

	return res
}

// Sync deploys skills from the library to target dirs following opts.
// In append mode unrelated target content is kept and only same-name skills are
// backed up and replaced; in apply mode the whole target is backed up and cleared.
func Sync(skills []SkillRef, targetPaths []string, opts Options) (*Result, error) {
	o := opts.normalized()
	home, _ := os.UserHomeDir()
	ts := time.Now().Format("2006-01-02T150405")
	backupDir := filepath.Join(home, ".skill-web", "backups", ts)

	res := &Result{
		Results:        make([]TargetResult, 0, len(targetPaths)),
		ResolvedSkills: make([]string, 0, len(skills)),
		BackupPath:     backupDir,
		Mode:           string(o.Mode),
		Strategy:       string(o.Strategy),
	}
	for _, sk := range skills {
		res.ResolvedSkills = append(res.ResolvedSkills, skillName(sk))
	}

	var backedUpTargets []string // targets that had content backed up

	for _, targetPath := range targetPaths {
		tr := TargetResult{
			Target:     targetPath,
			Synced:     []string{},
			Failed:     []string{},
			RemovedOld: []string{},
			Replaced:   []string{},
			Kept:       []string{},
		}

		// Ensure target dir exists
		if err := os.MkdirAll(targetPath, 0o755); err != nil {
			tr.Failed = append(tr.Failed, fmt.Sprintf("mkdir target: %v", err))
			res.Results = append(res.Results, tr)
			continue
		}

		plan := planTarget(targetPath, skills, o.Strategy)
		tr.Kept = append(tr.Kept, plan.Keep...)
		tr.Replaced = append(tr.Replaced, plan.Replace...)
		tr.RemovedOld = append(tr.RemovedOld, plan.Remove...)
		tr.Failed = append(tr.Failed, plan.Failed...)

		// Back up everything that is about to be replaced or removed. A failed
		// backup keeps the entry in place instead of silently destroying it.
		blocked := map[string]bool{}
		toBackup := append(append([]string{}, plan.Remove...), plan.Replace...)
		if len(toBackup) > 0 {
			if err := os.MkdirAll(backupDir, 0o755); err != nil {
				tr.Failed = append(tr.Failed, fmt.Sprintf("create backup dir: %v", err))
				res.Results = append(res.Results, tr)
				continue
			}
			for _, name := range toBackup {
				src := filepath.Join(targetPath, name)
				dst := filepath.Join(backupDir, filepath.Base(targetPath), name)
				if err := backupEntry(src, dst); err != nil {
					blocked[name] = true
					tr.Failed = append(tr.Failed, fmt.Sprintf("%s (backup failed: %v)", name, err))
				}
			}
			backedUpTargets = append(backedUpTargets, filepath.Base(targetPath))
		}

		// Deploy each planned skill.
		for i, sk := range plan.Refs {
			name := plan.Names[i]
			if blocked[name] {
				continue // already reported as failed
			}
			dst := filepath.Join(targetPath, name)

			// Remove existing if any
			os.RemoveAll(dst)

			if err := linkOrCopy(sk.StorePath, dst, o.Mode); err != nil {
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

// backupEntry moves src into the backup tree. When a rename is impossible
// (e.g. the backup tree lives on another filesystem) it falls back to a copy.
func backupEntry(src, dst string) error {
	return backupEntryWith(os.Rename, src, dst)
}

func backupEntryWith(rename func(string, string) error, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := rename(src, dst); err == nil {
		return nil
	}

	fi, err := os.Lstat(src)
	if err != nil {
		return err
	}

	var copyErr error
	switch {
	case fi.Mode()&os.ModeSymlink != 0:
		copyErr = copySymlink(src, dst)
	case fi.IsDir():
		copyErr = copyDir(src, dst)
	default:
		return fmt.Errorf("unsupported entry: %s", src)
	}
	if copyErr != nil {
		return copyErr
	}
	return os.RemoveAll(src)
}

// copySymlink re-creates the symlink src at dst.
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	os.RemoveAll(dst)
	return os.Symlink(target, dst)
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
