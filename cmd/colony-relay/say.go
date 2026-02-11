// ABOUTME: Say subcommand - posts a message to the relay server
// ABOUTME: Reads message from arguments or stdin, discovers server via port file

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strings"

	"github.com/ff6347/colony-relay/pkg/discover"
)

func runSay(args []string) int {
	fs := flag.NewFlagSet("colony-relay say", flag.ContinueOnError)
	from := fs.String("from", "", "Sender name (default: $USER)")
	server := fs.String("server", "", "Server URL (default: auto-discover from .colony-relay/port)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Resolve sender name
	senderName := *from
	if senderName == "" {
		if u, err := user.Current(); err == nil {
			senderName = u.Username
		}
	}
	if senderName == "" {
		fmt.Fprintln(os.Stderr, "error: --from is required (or $USER must be set)")
		return 1
	}

	// Resolve server URL
	serverURL, err := discover.ResolveServerURL(*server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Get message: from remaining args or stdin
	var message string
	if fs.NArg() > 0 {
		message = strings.Join(fs.Args(), " ")
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading stdin: %v\n", err)
			return 1
		}
		message = string(data)
	}

	message = strings.TrimSpace(message)
	if message == "" {
		fmt.Fprintln(os.Stderr, "error: no message provided")
		return 1
	}

	if err := postMessage(serverURL, senderName, message); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	return 0
}

func postMessage(serverURL, from, body string) error {
	payload := map[string]string{
		"from": from,
		"body": body,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	url := strings.TrimSuffix(serverURL, "/") + "/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}
