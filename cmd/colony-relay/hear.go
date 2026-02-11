// ABOUTME: Hear subcommand - receives messages from the relay server
// ABOUTME: Supports polling (default) and SSE streaming modes

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ff6347/colony-relay/pkg/discover"
)

type hearMessage struct {
	ID       int64    `json:"id"`
	TS       string   `json:"ts"`
	Sender   string   `json:"from"`
	Body     string   `json:"body"`
	Mentions []string `json:"mentions"`
}

func runHear(args []string) int {
	fs := flag.NewFlagSet("colony-relay hear", flag.ContinueOnError)
	forAgent := fs.String("for", "", "Agent name to receive messages for (default: $USER)")
	server := fs.String("server", "", "Server URL (default: auto-discover)")
	all := fs.Bool("all", false, "Hear all messages, not just @mentions")
	stream := fs.Bool("stream", false, "Stream messages via SSE instead of polling")
	limit := fs.Int("limit", 0, "Maximum messages to return (0 = all)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	// Resolve agent name
	agentName := *forAgent
	if agentName == "" {
		if u, err := user.Current(); err == nil {
			agentName = u.Username
		}
	}
	if agentName == "" {
		fmt.Fprintln(os.Stderr, "error: --for is required (or $USER must be set)")
		return 1
	}

	// Resolve server URL
	serverURL, err := discover.ResolveServerURL(*server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if *stream {
		return hearStream(serverURL)
	}
	return hearPoll(serverURL, agentName, *all, *limit)
}

func hearPoll(serverURL, agentName string, all bool, limit int) int {
	// Find relay dir for tracking last ID
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	relayDir, err := discover.FindRelayDir(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	lastIDPath := filepath.Join(relayDir, agentName+".lastid")

	lastID, err := readLastID(lastIDPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading lastid: %v\n", err)
		return 1
	}

	allMessages, err := fetchMessages(serverURL, agentName, lastID, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching messages: %v\n", err)
		return 1
	}

	// Apply limit (most recent N messages) for output
	output := limitMessages(allMessages, limit)
	formatOutput(os.Stdout, output)

	// Track highest ID from ALL fetched messages (not just limited ones)
	if len(allMessages) > 0 {
		newLastID := highestID(allMessages)
		if err := writeLastID(lastIDPath, newLastID); err != nil {
			fmt.Fprintf(os.Stderr, "error writing lastid: %v\n", err)
			return 1
		}
	}

	return 0
}

func hearStream(serverURL string) int {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return 0
		default:
		}

		err := streamMessages(ctx, serverURL, os.Stdout)
		if err == nil || ctx.Err() != nil {
			return 0
		}

		delay := backoff(attempt)
		fmt.Fprintf(os.Stderr, "connection lost, retrying in %v: %v\n", delay, err)
		attempt++

		select {
		case <-ctx.Done():
			return 0
		case <-time.After(delay):
		}
	}
}

func readLastID(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	id, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, nil
	}

	return id, nil
}

func writeLastID(path string, id int64) error {
	return os.WriteFile(path, []byte(strconv.FormatInt(id, 10)), 0644)
}

func fetchMessages(serverURL, forAgent string, since int64, all bool) ([]hearMessage, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("parse server URL: %w", err)
	}
	u.Path = "/messages"

	q := u.Query()
	q.Set("for", forAgent)
	q.Set("since", strconv.FormatInt(since, 10))
	if all {
		q.Set("all", "true")
	}
	u.RawQuery = q.Encode()

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

func formatOutput(w io.Writer, messages []hearMessage) {
	for _, msg := range messages {
		fmt.Fprintf(w, "%s: %s\n", msg.Sender, msg.Body)
	}
}

func limitMessages(messages []hearMessage, limit int) []hearMessage {
	if limit <= 0 || len(messages) <= limit {
		return messages
	}
	return messages[len(messages)-limit:]
}

func highestID(messages []hearMessage) int64 {
	var max int64
	for _, msg := range messages {
		if msg.ID > max {
			max = msg.ID
		}
	}
	return max
}

func streamMessages(ctx context.Context, serverURL string, stdout io.Writer) error {
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}
	u.Path = "/stream"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read stream: %w", err)
		}

		msg, err := parseSSELine(line)
		if err != nil {
			continue
		}
		if msg == nil {
			continue
		}

		fmt.Fprintf(stdout, "%s: %s\n", msg.Sender, msg.Body)
	}
}

func parseSSELine(line string) (*hearMessage, error) {
	line = strings.TrimSpace(line)

	if line == "" || strings.HasPrefix(line, ":") {
		return nil, nil
	}

	if strings.HasPrefix(line, "event:") {
		return nil, nil
	}

	if !strings.HasPrefix(line, "data:") {
		return nil, nil
	}

	jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	var msg hearMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	return &msg, nil
}

func backoff(attempt int) time.Duration {
	base := time.Second
	max := 30 * time.Second

	delay := base * (1 << attempt)
	if delay > max {
		delay = max
	}

	return delay
}
