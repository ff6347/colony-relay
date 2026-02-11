// ABOUTME: Tests for relay server discovery mechanism
// ABOUTME: Validates port file walking, reading, and URL resolution

package discover

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestFindRelayDir(t *testing.T) {
	// Create a temp directory structure: /tmp/xxx/subdir/deep
	// with .colony-relay/ at /tmp/xxx/
	tmpDir := t.TempDir()
	relayDir := filepath.Join(tmpDir, RelayDir)
	if err := os.Mkdir(relayDir, 0755); err != nil {
		t.Fatal(err)
	}

	deepDir := filepath.Join(tmpDir, "subdir", "deep")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Should find relay dir from deep subdirectory
	found, err := FindRelayDir(deepDir)
	if err != nil {
		t.Fatalf("FindRelayDir failed: %v", err)
	}
	if found != relayDir {
		t.Errorf("expected %q, got %q", relayDir, found)
	}
}

func TestFindRelayDirFromSameDir(t *testing.T) {
	tmpDir := t.TempDir()
	relayDir := filepath.Join(tmpDir, RelayDir)
	if err := os.Mkdir(relayDir, 0755); err != nil {
		t.Fatal(err)
	}

	found, err := FindRelayDir(tmpDir)
	if err != nil {
		t.Fatalf("FindRelayDir failed: %v", err)
	}
	if found != relayDir {
		t.Errorf("expected %q, got %q", relayDir, found)
	}
}

func TestFindRelayDirNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := FindRelayDir(tmpDir)
	if err == nil {
		t.Error("expected error when no relay dir exists")
	}
}

func TestReadPort(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, PortFile), []byte("4100\n"), 0644); err != nil {
		t.Fatal(err)
	}

	port, err := ReadPort(tmpDir)
	if err != nil {
		t.Fatalf("ReadPort failed: %v", err)
	}
	if port != 4100 {
		t.Errorf("expected 4100, got %d", port)
	}
}

func TestReadPortMissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := ReadPort(tmpDir)
	if err == nil {
		t.Error("expected error when port file missing")
	}
}

func TestReadPID(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, PIDFile), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}

	pid, err := ReadPID(tmpDir)
	if err != nil {
		t.Fatalf("ReadPID failed: %v", err)
	}
	if pid != 12345 {
		t.Errorf("expected 12345, got %d", pid)
	}
}

func TestServerURL(t *testing.T) {
	tmpDir := t.TempDir()
	relayDir := filepath.Join(tmpDir, RelayDir)
	if err := os.Mkdir(relayDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(relayDir, PortFile), []byte(strconv.Itoa(4100)), 0644); err != nil {
		t.Fatal(err)
	}

	url, err := ServerURL(tmpDir)
	if err != nil {
		t.Fatalf("ServerURL failed: %v", err)
	}
	if url != "http://localhost:4100" {
		t.Errorf("expected http://localhost:4100, got %q", url)
	}
}

func TestResolveServerURLFromFlag(t *testing.T) {
	url, err := ResolveServerURL("http://example.com:8080")
	if err != nil {
		t.Fatalf("ResolveServerURL failed: %v", err)
	}
	if url != "http://example.com:8080" {
		t.Errorf("expected flag value, got %q", url)
	}
}

func TestResolveServerURLFromEnv(t *testing.T) {
	t.Setenv("RELAY_SERVER", "http://env-server:9090")

	url, err := ResolveServerURL("")
	if err != nil {
		t.Fatalf("ResolveServerURL failed: %v", err)
	}
	if url != "http://env-server:9090" {
		t.Errorf("expected env value, got %q", url)
	}
}
