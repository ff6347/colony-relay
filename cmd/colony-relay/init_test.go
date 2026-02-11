// ABOUTME: Tests for the init subcommand
// ABOUTME: Validates hook and skill installation into project directories

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHooks(t *testing.T) {
	dir := t.TempDir()

	exitCode := installHooks(dir)
	if exitCode != 0 {
		t.Fatalf("installHooks returned %d", exitCode)
	}

	hooksDir := filepath.Join(dir, ".claude", "hooks")

	scripts := []string{
		"relay-start.sh",
		"relay-poll.sh",
		"relay-end.sh",
		"relay-resolve.sh",
	}

	for _, name := range scripts {
		path := filepath.Join(hooksDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("missing hook file %s: %v", name, err)
			continue
		}

		// Check executable permission
		if info.Mode()&0111 == 0 {
			t.Errorf("hook %s is not executable: %v", name, info.Mode())
		}
	}
}

func TestInstallSettingsNew(t *testing.T) {
	dir := t.TempDir()

	exitCode := installSettings(dir)
	if exitCode != 0 {
		t.Fatalf("installSettings returned %d", exitCode)
	}

	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON in settings.json: %v", err)
	}

	if _, ok := settings["hooks"]; !ok {
		t.Error("settings.json missing hooks key")
	}
}

func TestInstallSettingsMerge(t *testing.T) {
	dir := t.TempDir()

	// Pre-create settings.json with existing config
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	existing := map[string]interface{}{
		"someOtherSetting": true,
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	exitCode := installSettings(dir)
	if exitCode != 0 {
		t.Fatalf("installSettings returned %d", exitCode)
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	result, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not readable: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(result, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Existing setting preserved
	if _, ok := settings["someOtherSetting"]; !ok {
		t.Error("existing setting was lost during merge")
	}

	// Hooks added
	if _, ok := settings["hooks"]; !ok {
		t.Error("hooks not added during merge")
	}
}
