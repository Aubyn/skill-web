package syncer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkillDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: test\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewReportsConflictsAndExistingContent(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "store", "g1", "x")
	b := filepath.Join(root, "store", "g2", "x")
	writeSkillDir(t, a)
	writeSkillDir(t, b)

	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(target, "oldskill"), 0o755); err != nil {
		t.Fatal(err)
	}

	skills := []SkillRef{
		{ID: "g1:x", Name: "x", StorePath: a},
		{ID: "g2:x", Name: "x", StorePath: b},
	}
	preview := Preview(skills, []string{target})

	if len(preview.Conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %+v", preview.Conflicts)
	}
	if preview.Conflicts[0].Name != "x" {
		t.Fatalf("expected conflict name x, got %q", preview.Conflicts[0].Name)
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

	pt := preview.Results[0]
	if !pt.Exists {
		t.Fatal("expected target to exist")
	}
	if len(pt.Planned) != 1 || pt.Planned[0] != "x" {
		t.Fatalf("unexpected planned skills: %+v", pt.Planned)
	}
	if len(pt.Failed) != 1 {
		t.Fatalf("expected duplicate name to be reported as failed, got %+v", pt.Failed)
	}
	if len(pt.RemovedOld) != 1 || pt.RemovedOld[0] != "oldskill" {
		t.Fatalf("unexpected removed_old: %+v", pt.RemovedOld)
	}

	// Preview must not mutate the target.
	if _, err := os.Stat(filepath.Join(target, "oldskill")); err != nil {
		t.Fatalf("preview removed existing target content: %v", err)
	}
}

func TestPreviewMissingTargetAndSource(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "missing-target")

	preview := Preview([]SkillRef{{ID: "g:x", Name: "x"}}, []string{target})
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
	res, err := Sync(refs, []string{copyTarget}, ModeCopy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Mode != string(ModeCopy) {
		t.Fatalf("expected copy mode, got %q", res.Mode)
	}
	copyInfo, err := os.Stat(filepath.Join(copyTarget, "x", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(srcInfo, copyInfo) {
		t.Fatal("copy mode must not create hard links")
	}

	linkTarget := filepath.Join(root, "link-target")
	res, err = Sync(refs, []string{linkTarget}, ModeLink)
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
