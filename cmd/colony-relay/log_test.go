// ABOUTME: Tests for the log subcommand
// ABOUTME: Validates message history display with timestamps

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatLog(t *testing.T) {
	messages := []hearMessage{
		{ID: 1, TS: "2026-02-11T10:00:00Z", Sender: "alice", Body: "hello everyone"},
		{ID: 2, TS: "2026-02-11T10:01:00Z", Sender: "bob", Body: "@alice hi there"},
	}

	var buf bytes.Buffer
	formatLog(&buf, messages)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), output)
	}

	if !strings.Contains(lines[0], "10:00:00") {
		t.Errorf("expected timestamp in first line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "alice") {
		t.Errorf("expected sender in first line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "hello everyone") {
		t.Errorf("expected body in first line, got: %s", lines[0])
	}

	if !strings.Contains(lines[1], "10:01:00") {
		t.Errorf("expected timestamp in second line, got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "bob") {
		t.Errorf("expected sender in second line, got: %s", lines[1])
	}
}

func TestFetchLog(t *testing.T) {
	messages := []hearMessage{
		{ID: 1, TS: "2026-02-11T10:00:00Z", Sender: "alice", Body: "hello"},
		{ID: 2, TS: "2026-02-11T10:01:00Z", Sender: "bob", Body: "world"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		// Verify no "for" param is set
		if r.URL.Query().Get("for") != "" {
			t.Error("log should not set 'for' param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}))
	defer srv.Close()

	got, err := fetchLog(srv.URL, 0)
	if err != nil {
		t.Fatalf("fetchLog failed: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Sender != "alice" {
		t.Errorf("expected alice, got %s", got[0].Sender)
	}
	if got[1].Sender != "bob" {
		t.Errorf("expected bob, got %s", got[1].Sender)
	}
}

func TestFetchLogWithLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "5" {
			t.Errorf("expected limit=5, got %q", limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]hearMessage{})
	}))
	defer srv.Close()

	_, err := fetchLog(srv.URL, 5)
	if err != nil {
		t.Fatalf("fetchLog failed: %v", err)
	}
}
