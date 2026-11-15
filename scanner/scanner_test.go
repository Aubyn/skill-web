package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanRootSkillOnly(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "SKILL.md"), "---\nname: root-skill\ndescription: root\n---\n")
	writeFile(t, filepath.Join(root, "nested", "SKILL.md"), "---\nname: nested-skill\n---\n")

	entries, err := Scan(root, "fallback", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only the root skill, got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Name != "root-skill" {
		t.Fatalf("expected name root-skill, got %q", entries[0].Name)
	}
	if entries[0].Description != "root" {
		t.Fatalf("expected description root, got %q", entries[0].Description)
	}
}

func TestScanNestedSkillsUseFrontmatterName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a", "SKILL.md"), "---\nname: alpha\n---\n")
	writeFile(t, filepath.Join(root, "b", "SKILL.md"), "---\nname: beta\n---\n")

	entries, err := Scan(root, filepath.Base(root), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Name != "alpha" || entries[1].Name != "beta" {
		t.Fatalf("unexpected names: %q, %q", entries[0].Name, entries[1].Name)
	}
}

func TestScanRootFallbackWhenNameMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "SKILL.md"), "---\ndescription: no name\n---\n")

	entries, err := Scan(root, "package-name", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "package-name" {
		t.Fatalf("expected fallback package-name, got %q", entries[0].Name)
	}
}

func TestScanDirConventionName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reasonix-skill", "reasonix-skill.md"), "---\nname: reasonix-name\n---\n")

	entries, err := Scan(root, filepath.Base(root), 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "reasonix-name" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
