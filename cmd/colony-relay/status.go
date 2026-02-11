// ABOUTME: Status subcommand - checks if the relay server is running
// ABOUTME: Reads port/pid files and verifies the process is alive

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/ff6347/colony-relay/pkg/discover"
)

func runStatus(args []string) int {
	fs := flag.NewFlagSet("colony-relay status", flag.ContinueOnError)
	server := fs.String("server", "", "Server URL (default: auto-discover)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Try to find relay dir
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	relayDir, dirErr := discover.FindRelayDir(cwd)

	// If we have a relay dir, check PID
	if dirErr == nil {
		pid, pidErr := discover.ReadPID(relayDir)
		if pidErr == nil {
			if processAlive(pid) {
				port, _ := discover.ReadPort(relayDir)
				fmt.Printf("relay running (pid %d, port %d)\n", pid, port)
				addrs := discover.ListenAddresses(port)
				for _, addr := range addrs {
					fmt.Printf("  %s\n", addr)
				}
				printServerInfo(*server, port)
				return 0
			}
			fmt.Println("relay not running (stale pid file)")
			return 1
		}
	}

	// No PID file, try direct connection
	serverURL := *server
	if serverURL == "" {
		if dirErr == nil {
			url, err := discover.ServerURL(cwd)
			if err == nil {
				serverURL = url
			}
		}
	}

	if serverURL == "" {
		fmt.Println("relay not running (no .colony-relay/ found)")
		return 1
	}

	if checkServerReachable(serverURL) {
		fmt.Printf("relay reachable at %s\n", serverURL)
		return 0
	}

	fmt.Println("relay not running")
	return 1
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func printServerInfo(serverFlag string, port int) {
	serverURL := serverFlag
	if serverURL == "" {
		serverURL = fmt.Sprintf("http://localhost:%d", port)
	}

	// Try to get presence info
	resp, err := http.Get(strings.TrimSuffix(serverURL, "/") + "/presence")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var presence []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&presence); err != nil {
		return
	}

	if len(presence) > 0 {
		names := make([]string, len(presence))
		for i, p := range presence {
			names[i] = p.Name
		}
		fmt.Printf("active agents: %s\n", strings.Join(names, ", "))
	}
}

func checkServerReachable(serverURL string) bool {
	resp, err := http.Get(strings.TrimSuffix(serverURL, "/") + "/presence")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
