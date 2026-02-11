// ABOUTME: Log subcommand - displays full message history from the relay
// ABOUTME: Fetches all messages and shows them with timestamps

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/ff6347/colony-relay/pkg/discover"
)

func runLog(args []string) int {
	fs := flag.NewFlagSet("colony-relay log", flag.ContinueOnError)
	server := fs.String("server", "", "Server URL (default: auto-discover)")
	limit := fs.Int("limit", 0, "Maximum messages to show (0 = all)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	serverURL, err := discover.ResolveServerURL(*server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	messages, err := fetchLog(serverURL, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	formatLog(os.Stdout, messages)
	return 0
}

func fetchLog(serverURL string, limit int) ([]hearMessage, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	u.Path = "/messages"

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var messages []hearMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return messages, nil
}

func formatLog(w io.Writer, messages []hearMessage) {
	for _, msg := range messages {
		fmt.Fprintf(w, "[%s] %s: %s\n", msg.TS, msg.Sender, msg.Body)
	}
}
