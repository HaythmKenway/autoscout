package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSkillsFromDirReadsMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "idor.md"), []byte("# IDOR\nCheck numeric IDs."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	skills := LoadSkillsFromDir(dir)

	if !strings.Contains(skills, "## idor.md") {
		t.Fatalf("expected skill filename in output, got %q", skills)
	}
	if !strings.Contains(skills, "Check numeric IDs.") {
		t.Fatalf("expected markdown content in output, got %q", skills)
	}
	if strings.Contains(skills, "ignore me") {
		t.Fatalf("expected non-markdown files to be ignored, got %q", skills)
	}
}

func TestLoadSkillsFromDirReadsSkillMarkdownInDirectories(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "api-auth")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# API Auth\nCheck token scope."), 0644); err != nil {
		t.Fatal(err)
	}

	skills := LoadSkillsFromDir(dir)

	if !strings.Contains(skills, "## api-auth/SKILL.md") {
		t.Fatalf("expected nested SKILL.md path in output, got %q", skills)
	}
	if !strings.Contains(skills, "Check token scope.") {
		t.Fatalf("expected nested skill content in output, got %q", skills)
	}
}

func TestLoadSkillsFromDirReturnsHelpWhenMissing(t *testing.T) {
	skills := LoadSkillsFromDir(filepath.Join(t.TempDir(), "missing"))

	if !strings.Contains(skills, "No Autoscout skills found.") {
		t.Fatalf("expected setup help for missing skill dir, got %q", skills)
	}
}
