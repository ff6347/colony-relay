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

func TestParseTailscaleHost(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "valid dns name with trailing dot",
			json: `{"Self":{"DNSName":"myhost.tailnet.ts.net."}}`,
			want: "myhost.tailnet.ts.net",
		},
		{
			name: "valid dns name without trailing dot",
			json: `{"Self":{"DNSName":"myhost.tailnet.ts.net"}}`,
			want: "myhost.tailnet.ts.net",
		},
		{
			name: "empty dns name",
			json: `{"Self":{"DNSName":""}}`,
			want: "",
		},
		{
			name: "missing self",
			json: `{}`,
			want: "",
		},
		{
			name: "invalid json",
			json: `not json`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTailscaleHost([]byte(tt.json))
			if got != tt.want {
				t.Errorf("parseTailscaleHost() = %q, want %q", got, tt.want)
			}
		})
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
