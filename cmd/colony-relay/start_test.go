// ABOUTME: Tests for the start subcommand
// ABOUTME: Validates port auto-increment when default port is in use

package main

import (
	"net"
	"strconv"
	"testing"
)

func TestListenWithAutoIncrement(t *testing.T) {
	// Get a listener on an available port
	listener, port, err := listenWithAutoIncrement(0, 1)
	if err != nil {
		t.Fatalf("listenWithAutoIncrement failed: %v", err)
	}
	listener.Close()

	if port != 0 {
		t.Errorf("expected port 0 (any), got %d", port)
	}
}

func TestListenWithAutoIncrementOccupiedPort(t *testing.T) {
	// Occupy a port
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer occupied.Close()

	occupiedPort := occupied.Addr().(*net.TCPAddr).Port

	// Try to listen starting from the occupied port - should get next port
	listener, port, err := listenWithAutoIncrement(occupiedPort, 10)
	if err != nil {
		t.Fatalf("listenWithAutoIncrement failed: %v", err)
	}
	defer listener.Close()

	if port == occupiedPort {
		t.Error("should not have gotten the occupied port")
	}
	if port < occupiedPort || port > occupiedPort+10 {
		t.Errorf("expected port in range [%d, %d], got %d", occupiedPort+1, occupiedPort+10, port)
	}
}

func TestListenWithAutoIncrementNoAvailable(t *testing.T) {
	// Occupy several consecutive ports
	var listeners []net.Listener
	basePort := 19700 // unlikely to be in use
	for i := 0; i < 3; i++ {
		l, err := net.Listen("tcp", ":"+strconv.Itoa(basePort+i))
		if err != nil {
			// Port already in use, skip this test
			for _, existing := range listeners {
				existing.Close()
			}
			t.Skip("could not occupy test ports")
		}
		listeners = append(listeners, l)
	}
	defer func() {
		for _, l := range listeners {
			l.Close()
		}
	}()

	// Try with maxAttempts = 3 (all occupied)
	_, _, err := listenWithAutoIncrement(basePort, 3)
	if err == nil {
		t.Error("expected error when no ports available")
	}
}
