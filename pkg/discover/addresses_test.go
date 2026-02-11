// ABOUTME: Tests for network address detection.
// ABOUTME: Verifies that listen addresses always include localhost.

package discover

import (
	"fmt"
	"strings"
	"testing"
)

func TestListenAddresses_AlwaysIncludesLocalhost(t *testing.T) {
	port := 4100
	addrs := ListenAddresses(port)

	if len(addrs) == 0 {
		t.Fatal("expected at least one address")
	}

	expected := fmt.Sprintf("http://localhost:%d", port)
	if addrs[0] != expected {
		t.Errorf("first address = %q, want %q", addrs[0], expected)
	}
}

func TestListenAddresses_AllAddressesContainPort(t *testing.T) {
	port := 5555
	addrs := ListenAddresses(port)

	for _, addr := range addrs {
		if !strings.Contains(addr, fmt.Sprintf(":%d", port)) {
			t.Errorf("address %q does not contain port %d", addr, port)
		}
	}
}

func TestListenAddresses_AllAddressesAreHTTP(t *testing.T) {
	addrs := ListenAddresses(4100)

	for _, addr := range addrs {
		if !strings.HasPrefix(addr, "http://") {
			t.Errorf("address %q does not start with http://", addr)
		}
	}
}

func TestListenAddresses_NoDuplicates(t *testing.T) {
	addrs := ListenAddresses(4100)
	seen := make(map[string]bool)

	for _, addr := range addrs {
		if seen[addr] {
			t.Errorf("duplicate address: %q", addr)
		}
		seen[addr] = true
	}
}
