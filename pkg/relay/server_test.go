// ABOUTME: Tests for HTTP API handlers
// ABOUTME: Uses httptest for isolated endpoint testing

package relay

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestServer(t *testing.T) *Server {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return NewServer(store)
}

func TestPostMessage(t *testing.T) {
	srv := setupTestServer(t)

	body := `{"from": "agent-one", "body": "hello @agent-two"}`
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		ID int64  `json:"id"`
		TS string `json:"ts"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if resp.TS == "" {
		t.Error("expected non-empty timestamp")
	}
}

func TestPostMessageBadJSON(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestPostMessageMissingFields(t *testing.T) {
	srv := setupTestServer(t)

	// Missing "from"
	body := `{"body": "hello"}`
	req := httptest.NewRequest("POST", "/messages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestGetMessagesSince(t *testing.T) {
	srv := setupTestServer(t)

	// Insert some messages
	postTestMessage(t, srv, "a", "first")
	msg2 := postTestMessage(t, srv, "b", "second")
	postTestMessage(t, srv, "c", "third")

	// Get messages since msg2
	req := httptest.NewRequest("GET", "/messages?since="+itoa(msg2.ID), nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var msgs []*Message
	if err := json.NewDecoder(rec.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "third" {
		t.Errorf("expected 'third', got %q", msgs[0].Body)
	}
}

func TestGetMessagesForEntity(t *testing.T) {
	srv := setupTestServer(t)

	postTestMessage(t, srv, "a", "hello @bob")
	postTestMessage(t, srv, "b", "hello @alice")
	postTestMessage(t, srv, "c", "hello @all")

	// Get messages for bob
	req := httptest.NewRequest("GET", "/messages?for=bob", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var msgs []*Message
	if err := json.NewDecoder(rec.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Should get "hello @bob" and "hello @all"
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages for bob, got %d", len(msgs))
	}
}

func TestGetMessagesWithLimit(t *testing.T) {
	srv := setupTestServer(t)

	for i := 0; i < 5; i++ {
		postTestMessage(t, srv, "sender", "message")
	}

	req := httptest.NewRequest("GET", "/messages?limit=3", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var msgs []*Message
	if err := json.NewDecoder(rec.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestGetMessagesAll(t *testing.T) {
	srv := setupTestServer(t)

	postTestMessage(t, srv, "a", "hello @bob")
	postTestMessage(t, srv, "b", "no mentions")
	postTestMessage(t, srv, "c", "hello @alice")

	// Get all messages for bob with all=true
	req := httptest.NewRequest("GET", "/messages?for=bob&all=true", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var msgs []*Message
	if err := json.NewDecoder(rec.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// With all=true, should get all 3 messages
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages with all=true, got %d", len(msgs))
	}
}

// postTestMessage posts a message and returns the result
func postTestMessage(t *testing.T, srv *Server, from, body string) *Message {
	t.Helper()
	reqBody, _ := json.Marshal(map[string]string{"from": from, "body": body})
	req := httptest.NewRequest("POST", "/messages", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("postTestMessage failed: %d %s", rec.Code, rec.Body.String())
	}

	var msg Message
	json.NewDecoder(rec.Body).Decode(&msg)
	return &msg
}

func itoa(i int64) string {
	return fmt.Sprintf("%d", i)
}

func TestStreamSSE(t *testing.T) {
	srv := setupTestServer(t)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to /stream: %v", err)
	}
	defer resp.Body.Close()

	// Check SSE headers
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", resp.Header.Get("Cache-Control"))
	}

	reader := bufio.NewReader(resp.Body)

	dataCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}
			if strings.HasPrefix(line, ":") || strings.TrimSpace(line) == "" {
				continue
			}
			dataCh <- line
			return
		}
	}()

	time.Sleep(50 * time.Millisecond)

	// Post a message
	reqBody, _ := json.Marshal(map[string]string{"from": "agent-one", "body": "hello @agent-two"})
	postReq, _ := http.NewRequest("POST", ts.URL+"/messages", bytes.NewBuffer(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		t.Fatalf("failed to post message: %v", err)
	}
	postResp.Body.Close()

	select {
	case data := <-dataCh:
		if !bytes.HasPrefix([]byte(data), []byte("data: ")) {
			t.Errorf("expected SSE data prefix, got %q", data)
		}

		jsonData := bytes.TrimPrefix([]byte(data), []byte("data: "))
		jsonData = bytes.TrimSpace(jsonData)

		var msg Message
		if err := json.Unmarshal(jsonData, &msg); err != nil {
			t.Fatalf("failed to parse SSE message JSON: %v (data: %q)", err, jsonData)
		}

		if msg.Sender != "agent-one" {
			t.Errorf("expected sender 'agent-one', got %q", msg.Sender)
		}
		if msg.Body != "hello @agent-two" {
			t.Errorf("expected body 'hello @agent-two', got %q", msg.Body)
		}
	case err := <-errCh:
		t.Fatalf("error reading from stream: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE message")
	}
}

func TestStreamSSEMultipleClients(t *testing.T) {
	srv := setupTestServer(t)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req1, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/stream", nil)
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("client 1 failed to connect: %v", err)
	}
	defer resp1.Body.Close()

	req2, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/stream", nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("client 2 failed to connect: %v", err)
	}
	defer resp2.Body.Close()

	reader1 := bufio.NewReader(resp1.Body)
	reader2 := bufio.NewReader(resp2.Body)

	dataCh1 := make(chan string, 1)
	dataCh2 := make(chan string, 1)
	go func() {
		for {
			line, _ := reader1.ReadString('\n')
			if strings.HasPrefix(line, ":") || strings.TrimSpace(line) == "" {
				continue
			}
			dataCh1 <- line
			return
		}
	}()
	go func() {
		for {
			line, _ := reader2.ReadString('\n')
			if strings.HasPrefix(line, ":") || strings.TrimSpace(line) == "" {
				continue
			}
			dataCh2 <- line
			return
		}
	}()

	time.Sleep(50 * time.Millisecond)

	reqBody, _ := json.Marshal(map[string]string{"from": "sender", "body": "broadcast test"})
	postReq, _ := http.NewRequest("POST", ts.URL+"/messages", bytes.NewBuffer(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		t.Fatalf("failed to post message: %v", err)
	}
	postResp.Body.Close()

	for i, dataCh := range []chan string{dataCh1, dataCh2} {
		select {
		case data := <-dataCh:
			if !bytes.Contains([]byte(data), []byte("broadcast test")) {
				t.Errorf("client %d did not receive message: %q", i+1, data)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("client %d timeout waiting for message", i+1)
		}
	}
}

func TestStreamSSEDisconnect(t *testing.T) {
	srv := setupTestServer(t)

	ts := httptest.NewServer(srv)
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL+"/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to /stream: %v", err)
	}
	defer resp.Body.Close()

	time.Sleep(50 * time.Millisecond)

	srv.subscribersMu.RLock()
	countBefore := len(srv.subscribers)
	srv.subscribersMu.RUnlock()

	if countBefore != 1 {
		t.Errorf("expected 1 subscriber, got %d", countBefore)
	}

	cancel()

	time.Sleep(100 * time.Millisecond)

	srv.subscribersMu.RLock()
	countAfter := len(srv.subscribers)
	srv.subscribersMu.RUnlock()

	if countAfter != 0 {
		t.Errorf("expected 0 subscribers after disconnect, got %d", countAfter)
	}
}

func TestPresenceEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/presence", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var presence []Presence
	if err := json.NewDecoder(rec.Body).Decode(&presence); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(presence) != 0 {
		t.Errorf("expected 0 agents initially, got %d", len(presence))
	}
}

func TestPresenceAfterMessage(t *testing.T) {
	srv := setupTestServer(t)

	postTestMessage(t, srv, "builder", "hello world")

	req := httptest.NewRequest("GET", "/presence", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var presence []Presence
	if err := json.NewDecoder(rec.Body).Decode(&presence); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(presence) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(presence))
	}

	if presence[0].Name != "builder" {
		t.Errorf("expected name 'builder', got %q", presence[0].Name)
	}
}

func TestPresenceExpired(t *testing.T) {
	srv := setupTestServer(t)

	// Set presence from 60 minutes ago (beyond default 30 minute window)
	longAgo := time.Now().Add(-60 * time.Minute)
	if err := srv.store.UpdatePresenceAt("gone-agent", longAgo); err != nil {
		t.Fatalf("failed to set presence: %v", err)
	}

	req := httptest.NewRequest("GET", "/presence", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	var presence []Presence
	json.NewDecoder(rec.Body).Decode(&presence)

	if len(presence) != 0 {
		t.Errorf("expected 0 agents (expired), got %d", len(presence))
	}
}

func TestPresenceOnGetForEntity(t *testing.T) {
	srv := setupTestServer(t)

	postTestMessage(t, srv, "alice", "hello @bob")

	// Bob fetches messages - this should register bob's presence
	req := httptest.NewRequest("GET", "/messages?for=bob", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/presence", nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var presence []Presence
	json.NewDecoder(rec.Body).Decode(&presence)

	// Should have both alice (from posting) and bob (from fetching)
	if len(presence) != 2 {
		t.Fatalf("expected 2 agents (alice and bob), got %d", len(presence))
	}

	var foundBob bool
	for _, p := range presence {
		if p.Name == "bob" {
			foundBob = true
			break
		}
	}
	if !foundBob {
		t.Error("expected bob to be in presence list after fetching messages")
	}
}

func TestMessageLog(t *testing.T) {
	srv := setupTestServer(t)

	var buf bytes.Buffer
	srv.SetLog(&buf)

	postTestMessage(t, srv, "alice", "hello @bob")
	postTestMessage(t, srv, "bob", "hi alice")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %q", len(lines), output)
	}

	if !strings.Contains(lines[0], "alice") {
		t.Errorf("expected alice in first line, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "hello @bob") {
		t.Errorf("expected message body in first line, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "bob") {
		t.Errorf("expected bob in second line, got: %s", lines[1])
	}
}

func TestMessageLogDisabledByDefault(t *testing.T) {
	srv := setupTestServer(t)

	// No SetLog call - should not panic
	postTestMessage(t, srv, "alice", "hello")
}

func TestUIEndpoint(t *testing.T) {
	srv := setupTestServer(t)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type 'text/html; charset=utf-8', got %q", contentType)
	}

	body := rec.Body.String()

	if !bytes.Contains([]byte(body), []byte("<!DOCTYPE html>")) {
		t.Error("expected HTML doctype")
	}

	if !bytes.Contains([]byte(body), []byte("monospace")) {
		t.Error("expected monospace font styling")
	}

	if !bytes.Contains([]byte(body), []byte("EventSource")) {
		t.Error("expected EventSource for SSE connection")
	}

	if !bytes.Contains([]byte(body), []byte("/messages")) {
		t.Error("expected POST to /messages endpoint")
	}

	if !bytes.Contains([]byte(body), []byte("status")) {
		t.Error("expected connection status indicator")
	}
}
