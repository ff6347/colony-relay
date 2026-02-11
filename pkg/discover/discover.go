// ABOUTME: Discovers the relay server by walking up from CWD to find .colony-relay/port
// ABOUTME: Provides shared discovery logic for say, hear, and status subcommands

package discover

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const RelayDir = ".colony-relay"
const PortFile = "port"
const PIDFile = "pid"
const DBFile = "relay.db"

// ServerURL finds the relay server URL by walking up directories from startDir
// looking for a .colony-relay/port file. Returns empty string if not found.
func ServerURL(startDir string) (string, error) {
	dir, err := FindRelayDir(startDir)
	if err != nil {
		return "", err
	}

	port, err := ReadPort(dir)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://localhost:%d", port), nil
}

// FindRelayDir walks up from startDir looking for a .colony-relay/ directory.
// Returns the full path to the .colony-relay/ directory, or an error if not found.
func FindRelayDir(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	for {
		candidate := filepath.Join(dir, RelayDir)
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s directory found (walked up from %s)", RelayDir, startDir)
		}
		dir = parent
	}
}

// ReadPort reads the port number from a .colony-relay/port file.
func ReadPort(relayDir string) (int, error) {
	data, err := os.ReadFile(filepath.Join(relayDir, PortFile))
	if err != nil {
		return 0, fmt.Errorf("read port file: %w", err)
	}

	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse port: %w", err)
	}

	return port, nil
}

// ReadPID reads the PID from a .colony-relay/pid file.
func ReadPID(relayDir string) (int, error) {
	data, err := os.ReadFile(filepath.Join(relayDir, PIDFile))
	if err != nil {
		return 0, fmt.Errorf("read pid file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse pid: %w", err)
	}

	return pid, nil
}

// ResolveServerURL determines the server URL from flag, env var, or discovery.
func ResolveServerURL(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if envURL := os.Getenv("RELAY_SERVER"); envURL != "" {
		return envURL, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	return ServerURL(cwd)
}
