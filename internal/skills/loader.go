package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LoadContext searches the current working directory for skills and markdown files
// Returns a combined text block to be injected as a System prompt
func LoadContext() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("You are an autonomous AI Assistant. You have access to tools and must use them to accomplish the user's goals. Below is the project context and available skills loaded from markdown files:\n\n")

	// 1. Files like .cursorrules, .windsurfrules or project README
	importantFiles := []string{".cursorrules", ".windsurfrules", "README.md", "CLAUDE.md"}
	for _, file := range importantFiles {
		content, err := os.ReadFile(filepath.Join(cwd, file))
		if err == nil {
			sb.WriteString(fmt.Sprintf("\n--- BEGIN %s ---\n%s\n--- END %s ---\n", file, string(content), file))
		}
	}

	// 2. Load any .md files from a "skills" or "docs" folder if they exist
	foldersToScan := []string{"skills", "docs"}
	for _, folder := range foldersToScan {
		folderPath := filepath.Join(cwd, folder)
		if stat, err := os.Stat(folderPath); err == nil && stat.IsDir() {
			filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
					content, err := os.ReadFile(path)
					if err == nil {
						relPath, _ := filepath.Rel(cwd, path)
						sb.WriteString(fmt.Sprintf("\n--- BEGIN SKILL: %s ---\n%s\n--- END SKILL: %s ---\n", relPath, string(content), relPath))
					}
				}
				return nil
			})
		}
	}

	return sb.String()
}
