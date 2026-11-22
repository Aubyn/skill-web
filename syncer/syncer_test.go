package syncer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func writeSkillDir(t *testing.T, dir string) {
	t.Helper()
	writeSkill(t, dir, "---\nname: test\n---\n")
}

func writeSkill(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseStrategy(t *testing.T) {
	cases := []struct {
		in   string
		want Strategy
	}{
		{"", StrategyApply},
		{"apply", StrategyApply},
		{"append", StrategyAppend},
		{" APPEND ", StrategyAppend},
		{"Append", StrategyAppend},
		{"apply ", StrategyApply},
	}
	for _, c := range cases {
		got, err := ParseStrategy(c.in)
		if err != nil {
			t.Fatalf("ParseStrategy(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("ParseStrategy(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if _, err := ParseStrategy("overwrite"); err == nil {
		t.Fatal("expected an error for an unknown strategy")
	}
}

func TestPreviewApplyPlan(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "store", "g1", "x")
	writeSkillDir(t, src)

	target := filepath.Join(root, "target")
	for _, name := range []string{"old1", "old2"} {
		writeSkillDir(t, filepath.Join(target, name))
	}
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	preview := Preview([]SkillRef{{ID: "g1:x", Name: "x", StorePath: src}}, []string{target}, Options{Strategy: StrategyApply})

	if preview.Strategy != string(StrategyApply) {
		t.Fatalf("expected strategy apply, got %q", preview.Strategy)
	}
	pt := preview.Results[0]
	if !pt.Exists {
		t.Fatal("expected target to exist")
	}
	if len(pt.RemovedOld) != 2 || pt.RemovedOld[0] != "old1" || pt.RemovedOld[1] != "old2" {
		t.Fatalf("unexpected removed_old: %+v", pt.RemovedOld)
	}
	if len(pt.Kept) != 0 || len(pt.Replaced) != 0 {
		t.Fatalf("apply must not keep or replace: kept=%+v replaced=%+v", pt.Kept, pt.Replaced)
	}
	if len(pt.Planned) != 1 || pt.Planned[0] != "x" {
		t.Fatalf("unexpected planned: %+v", pt.Planned)
	}
	if preview.TotalPlanned != 1 {
		t.Fatalf("expected 1 planned skill, got %d", preview.TotalPlanned)
	}

	// Preview must not mutate the target.
	if _, err := os.Stat(filepath.Join(target, "old1")); err != nil {
		t.Fatalf("preview removed existing target content: %v", err)
	}
	if readFile(t, filepath.Join(target, "README.md")) != "keep me" {
		t.Fatal("preview touched a regular file")
	}
}

func TestPreviewAppendPlan(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "store", "g1", "x")
	writeSkillDir(t, src)

	target := filepath.Join(root, "target")
	writeSkill(t, filepath.Join(target, "x"), "old version")
	writeSkillDir(t, filepath.Join(target, "other"))

	preview := Preview([]SkillRef{{ID: "g1:x", Name: "x", StorePath: src}}, []string{target}, Options{Strategy: StrategyAppend})

	pt := preview.Results[0]
	if len(pt.Replaced) != 1 || pt.Replaced[0] != "x" {
		t.Fatalf("expected x to be replaced, got %+v", pt.Replaced)
	}
	if len(pt.Kept) != 1 || pt.Kept[0] != "other" {
		t.Fatalf("expected other to be kept, got %+v", pt.Kept)
	}
	if len(pt.RemovedOld) != 0 {
		t.Fatalf("append must not remove content: %+v", pt.RemovedOld)
	}
	if len(pt.Planned) != 1 || pt.Planned[0] != "x" {
		t.Fatalf("unexpected planned: %+v", pt.Planned)
	}
	if preview.MissingTargets == nil || len(preview.MissingTargets) != 0 {
		t.Fatalf("unexpected missing targets: %+v", preview.MissingTargets)
	}
}

func TestPreviewNameConflictIsStrategyIndependent(t *testing.T) {
	for _, strategy := range []Strategy{StrategyApply, StrategyAppend} {
		t.Run(string(strategy), func(t *testing.T) {
			root := t.TempDir()
			a := filepath.Join(root, "store", "g1", "x")
			b := filepath.Join(root, "store", "g2", "x")
			writeSkillDir(t, a)
			writeSkillDir(t, b)

			target := filepath.Join(root, "target")
			writeSkillDir(t, filepath.Join(target, "oldskill"))

			skills := []SkillRef{
				{ID: "g1:x", Name: "x", StorePath: a},
				{ID: "g2:x", Name: "x", StorePath: b},
			}
			preview := Preview(skills, []string{target}, Options{Strategy: strategy})

			if len(preview.Conflicts) != 1 {
				t.Fatalf("expected 1 conflict, got %+v", preview.Conflicts)
			}
			if preview.Conflicts[0].Winner != "g1:x" {
				t.Fatalf("expected winner g1:x, got %q", preview.Conflicts[0].Winner)
			}
			if len(preview.Conflicts[0].SkillIDs) != 2 {
				t.Fatalf("expected 2 conflicting skill ids, got %+v", preview.Conflicts[0].SkillIDs)
			}
			if preview.TotalPlanned != 1 {
				t.Fatalf("expected 1 planned skill, got %d", preview.TotalPlanned)
			}
			if len(preview.Results[0].Failed) != 1 {
				t.Fatalf("expected duplicate name reported as failed, got %+v", preview.Results[0].Failed)
			}
		})
	}
}

func TestPreviewMissingTargetAndSource(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "missing-target")

	for _, strategy := range []Strategy{StrategyApply, StrategyAppend} {
		preview := Preview([]SkillRef{{ID: "g:x", Name: "x"}}, []string{target}, Options{Strategy: strategy})
		if len(preview.MissingTargets) != 1 || preview.MissingTargets[0] != target {
			t.Fatalf("unexpected missing targets: %+v", preview.MissingTargets)
		}
		if len(preview.Results[0].Failed) != 1 {
			t.Fatalf("expected missing source to be reported, got %+v", preview.Results[0].Failed)
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("preview created the missing target: %v", err)
		}
	}
}

func TestSyncCopyAndLinkModes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	src := filepath.Join(root, "store", "x")
	writeSkillDir(t, src)
	refs := []SkillRef{{ID: "g:x", Name: "x", StorePath: src}}
	srcInfo, err := os.Stat(filepath.Join(src, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	copyTarget := filepath.Join(root, "copy-target")
	if err := os.MkdirAll(copyTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Sync(refs, []string{copyTarget}, Options{Mode: ModeCopy, Strategy: StrategyApply})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != string(ModeCopy) {
		t.Fatalf("expected copy mode, got %q", res.Mode)
	}
	if res.Strategy != string(StrategyApply) {
		t.Fatalf("expected apply strategy, got %q", res.Strategy)
	}
	copyInfo, err := os.Stat(filepath.Join(copyTarget, "x", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(srcInfo, copyInfo) {
		t.Fatal("copy mode must not create hard links")
	}

	linkTarget := filepath.Join(root, "link-target")
	res, err = Sync(refs, []string{linkTarget}, Options{Mode: ModeLink, Strategy: StrategyApply})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != string(ModeLink) {
		t.Fatalf("expected link mode, got %q", res.Mode)
	}
	linkInfo, err := os.Stat(filepath.Join(linkTarget, "x", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(srcInfo, linkInfo) {
		t.Fatal("link mode should share the source file")
	}
}

func TestSyncApplyBacksUpAndClearsTarget(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	src := filepath.Join(root, "store", "g1", "x")
	writeSkillDir(t, src)

	target := filepath.Join(root, "target")
	writeSkill(t, filepath.Join(target, "old"), "old content")

	res, err := Sync([]SkillRef{{ID: "g1:x", Name: "x", StorePath: src}}, []string{target}, Options{Mode: ModeCopy, Strategy: StrategyApply})
	if err != nil {
		t.Fatal(err)
	}

	tr := res.Results[0]
	if len(tr.Synced) != 1 || tr.Synced[0] != "x" {
		t.Fatalf("unexpected synced: %+v", tr.Synced)
	}
	if len(tr.RemovedOld) != 1 || tr.RemovedOld[0] != "old" {
		t.Fatalf("unexpected removed_old: %+v", tr.RemovedOld)
	}
	if len(tr.Kept) != 0 || len(tr.Replaced) != 0 {
		t.Fatalf("apply must not keep or replace: kept=%+v replaced=%+v", tr.Kept, tr.Replaced)
	}
	if _, err := os.Stat(filepath.Join(target, "old")); !os.IsNotExist(err) {
		t.Fatal("apply should have removed the old skill")
	}
	if got := readFile(t, filepath.Join(res.BackupPath, filepath.Base(target), "old", "SKILL.md")); got != "old content" {
		t.Fatalf("backup content mismatch: %q", got)
	}
}

func TestSyncAppendKeepsUnrelatedAndBacksUpOnlyReplaced(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	src := filepath.Join(root, "store", "g1", "x")
	writeSkill(t, src, "new version")

	target := filepath.Join(root, "target")
	writeSkill(t, filepath.Join(target, "x"), "old version")
	writeSkill(t, filepath.Join(target, "other"), "keep me")

	res, err := Sync([]SkillRef{{ID: "g1:x", Name: "x", StorePath: src}}, []string{target}, Options{Mode: ModeCopy, Strategy: StrategyAppend})
	if err != nil {
		t.Fatal(err)
	}

	tr := res.Results[0]
	if len(tr.Synced) != 1 || tr.Synced[0] != "x" {
		t.Fatalf("unexpected synced: %+v", tr.Synced)
	}
	if len(tr.Replaced) != 1 || tr.Replaced[0] != "x" {
		t.Fatalf("unexpected replaced: %+v", tr.Replaced)
	}
	if len(tr.Kept) != 1 || tr.Kept[0] != "other" {
		t.Fatalf("unexpected kept: %+v", tr.Kept)
	}
	if len(tr.RemovedOld) != 0 {
		t.Fatalf("append must not report removed_old: %+v", tr.RemovedOld)
	}

	if got := readFile(t, filepath.Join(target, "x", "SKILL.md")); got != "new version" {
		t.Fatalf("replaced skill content = %q", got)
	}
	if got := readFile(t, filepath.Join(target, "other", "SKILL.md")); got != "keep me" {
		t.Fatalf("unrelated skill was modified: %q", got)
	}

	backupTarget := filepath.Join(res.BackupPath, filepath.Base(target))
	if got := readFile(t, filepath.Join(backupTarget, "x", "SKILL.md")); got != "old version" {
		t.Fatalf("backup content mismatch: %q", got)
	}
	if _, err := os.Stat(filepath.Join(backupTarget, "other")); !os.IsNotExist(err) {
		t.Fatal("append must not back up unrelated content")
	}
}

func TestSyncAppendIsIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	root := t.TempDir()
	src := filepath.Join(root, "store", "g1", "x")
	writeSkill(t, src, "new version")

	target := filepath.Join(root, "target")
	writeSkill(t, filepath.Join(target, "other"), "keep me")

	refs := []SkillRef{{ID: "g1:x", Name: "x", StorePath: src}}
	opts := Options{Mode: ModeCopy, Strategy: StrategyAppend}
	if _, err := Sync(refs, []string{target}, opts); err != nil {
		t.Fatal(err)
	}
	res, err := Sync(refs, []string{target}, opts)
	if err != nil {
		t.Fatal(err)
	}

	tr := res.Results[0]
	if len(tr.Synced) != 1 || len(tr.Replaced) != 1 || len(tr.Kept) != 1 {
		t.Fatalf("unexpected second run result: %+v", tr)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 target entries, got %d", len(entries))
	}
	if got := readFile(t, filepath.Join(target, "other", "SKILL.md")); got != "keep me" {
		t.Fatalf("unrelated skill was modified: %q", got)
	}
}

func TestBackupEntryFallsBackToCopy(t *testing.T) {
	failingRename := func(string, string) error { return errors.New("cross-device link") }

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "target", "x")
		writeSkill(t, src, "old version")
		dst := filepath.Join(root, "backup", "x")

		if err := backupEntryWith(failingRename, src, dst); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(src); !os.IsNotExist(err) {
			t.Fatal("source should be removed after a successful backup")
		}
		if got := readFile(t, filepath.Join(dst, "SKILL.md")); got != "old version" {
			t.Fatalf("backup content mismatch: %q", got)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		link := filepath.Join(root, "target", "linked")
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("/some/where", link); err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(root, "backup", "linked")

		if err := backupEntryWith(failingRename, link, dst); err != nil {
			t.Fatal(err)
		}
		got, err := os.Readlink(dst)
		if err != nil {
			t.Fatalf("backup is not a symlink: %v", err)
		}
		if got != "/some/where" {
			t.Fatalf("symlink target = %q", got)
		}
	})
}

func TestPruneBackupsKeepsTenNewest(t *testing.T) {
	parent := t.TempDir()
	targetBase := "target"
	for i := 0; i < 12; i++ {
		dir := filepath.Join(parent, fmt.Sprintf("2026-01-01T0000%02d", i), targetBase)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pruneBackups(parent, []string{targetBase})

	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	var remaining []string
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(parent, e.Name(), targetBase)); err == nil {
			remaining = append(remaining, e.Name())
		}
	}
	if len(remaining) != 10 {
		t.Fatalf("expected 10 backups to remain, got %d: %+v", len(remaining), remaining)
	}
	if remaining[0] != "2026-01-01T000002" {
		t.Fatalf("expected the two oldest backups to be pruned, first kept = %q", remaining[0])
	}
}
