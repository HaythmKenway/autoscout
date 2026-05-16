package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultSkillsDir = "$HOME/.config/autoscout/skills"

// LoadSkills reads Markdown skill files used to steer AI analysis behavior.
func LoadSkills() string {
	return LoadSkillsFromDir(os.ExpandEnv(defaultSkillsDir))
}

func LoadSkillsFromDir(skillsDir string) string {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return defaultSkillsHelp(skillsDir)
	}

	var paths []string
	for _, entry := range entries {
		path := filepath.Join(skillsDir, entry.Name())
		if entry.IsDir() {
			skillPath := filepath.Join(path, "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				paths = append(paths, skillPath)
			}
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".md" || ext == ".markdown" {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		return defaultSkillsHelp(skillsDir)
	}

	var sections []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("## %s\n%s", skillName(skillsDir, path), content))
	}

	if len(sections) == 0 {
		return defaultSkillsHelp(skillsDir)
	}

	return strings.Join(sections, "\n\n")
}

func skillName(skillsDir, path string) string {
	rel, err := filepath.Rel(skillsDir, path)
	if err != nil {
		return filepath.Base(path)
	}
	return filepath.ToSlash(rel)
}

func defaultSkillsHelp(skillsDir string) string {
	return fmt.Sprintf(`No Autoscout skills found.

Create Markdown files in %s to train the AI agent with focused playbooks.
Supported formats:
- %s/idor.md
- %s/api-auth/SKILL.md`, skillsDir, skillsDir, skillsDir)
}
