// ABOUTME: Start subcommand - runs the relay server in the foreground
// ABOUTME: Auto-increments port if default is in use, writes port/pid files for discovery

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/ff6347/colony-relay/pkg/discover"
	"github.com/ff6347/colony-relay/pkg/relay"
)

const defaultPort = 4100
const maxPortAttempts = 100

func runStart(args []string) int {
	fs := flag.NewFlagSet("colony-relay start", flag.ContinueOnError)
	port := fs.Int("port", defaultPort, "Port to listen on (auto-increments if in use)")
	dbPath := fs.String("db", "", "Database path (default: .colony-relay/relay.db)")
	presenceMinutes := fs.Float64("presence-timeout", relay.DefaultPresenceMinutes, "Presence timeout in minutes")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Find or create .colony-relay/ directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	relayDir := filepath.Join(cwd, discover.RelayDir)
	if err := os.MkdirAll(relayDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating %s: %v\n", discover.RelayDir, err)
		return 1
	}

	// Resolve DB path
	if *dbPath == "" {
		*dbPath = filepath.Join(relayDir, discover.DBFile)
	}

	// Create store
	store, err := relay.NewStore(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		return 1
	}
	defer store.Close()

	// Create server
	srv := relay.NewServer(store)
	srv.SetPresenceMinutes(*presenceMinutes)
	srv.SetLog(os.Stdout)

	// Find available port
	listener, actualPort, err := listenWithAutoIncrement(*port, maxPortAttempts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Write port and PID files
	portFile := filepath.Join(relayDir, discover.PortFile)
	pidFile := filepath.Join(relayDir, discover.PIDFile)

	if err := os.WriteFile(portFile, []byte(strconv.Itoa(actualPort)), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing port file: %v\n", err)
		listener.Close()
		return 1
	}

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing pid file: %v\n", err)
		os.Remove(portFile)
		listener.Close()
		return 1
	}

	// Clean up files on exit
	cleanup := func() {
		os.Remove(portFile)
		os.Remove(pidFile)
	}

	httpServer := &http.Server{
		Handler:     srv,
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	// Start serving
	go func() {
		addrs := discover.ListenAddresses(actualPort)
		fmt.Fprintf(os.Stderr, "relay listening on:\n")
		for _, addr := range addrs {
			fmt.Fprintf(os.Stderr, "  %s\n", addr)
		}
		if err := httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Fprintf(os.Stderr, "shutting down...\n")
	cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "relay stopped\n")
	return 0
}

// listenWithAutoIncrement tries to listen on startPort, incrementing on failure.
func listenWithAutoIncrement(startPort, maxAttempts int) (net.Listener, int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			return listener, port, nil
		}
	}
	return nil, 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+maxAttempts-1)
}
