// ABOUTME: Init subcommand - sets up colony-relay in the current project
// ABOUTME: Creates .colony-relay/ directory and installs Claude Code skill

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ff6347/colony-relay/pkg/discover"
	"github.com/ff6347/colony-relay/pkg/skill"
)

func runInit(args []string) int {
	fs := flag.NewFlagSet("colony-relay init", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Create .colony-relay/ directory
	relayDir := filepath.Join(cwd, discover.RelayDir)
	if err := os.MkdirAll(relayDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", discover.RelayDir, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "created %s/\n", discover.RelayDir)

	// Install Claude Code skill
	skillDir := filepath.Join(cwd, ".claude", "commands")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating .claude/commands: %v\n", err)
		return 1
	}

	skillPath := filepath.Join(skillDir, "relay.md")
	if err := os.WriteFile(skillPath, skill.Content, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing skill: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "installed .claude/commands/relay.md\n")

	// Check .gitignore
	gitignorePath := filepath.Join(cwd, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		data, err := os.ReadFile(gitignorePath)
		if err == nil {
			content := string(data)
			if !containsLine(content, discover.RelayDir) && !containsLine(content, discover.RelayDir+"/") {
				fmt.Fprintf(os.Stderr, "\nRemember to add %s/ to .gitignore\n", discover.RelayDir)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "\nRemember to add %s/ to .gitignore\n", discover.RelayDir)
	}

	return 0
}

func containsLine(content, target string) bool {
	for _, line := range splitLines(content) {
		if line == target {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
