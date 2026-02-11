// ABOUTME: Tests for message store SQLite implementation
// ABOUTME: Uses in-memory SQLite for fast, isolated tests

package relay

import (
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()
}

func TestInsertAndGetByID(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	msg, err := store.Insert("agent-one", "hello world", []string{})
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if msg.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if msg.Sender != "agent-one" {
		t.Errorf("expected sender 'agent-one', got %q", msg.Sender)
	}
	if msg.Body != "hello world" {
		t.Errorf("expected body 'hello world', got %q", msg.Body)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestInsertWithMentions(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	mentions := []string{"agent-two", "agent-three"}
	msg, err := store.Insert("agent-one", "hello @agent-two and @agent-three", mentions)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	if len(msg.Mentions) != 2 {
		t.Fatalf("expected 2 mentions, got %d", len(msg.Mentions))
	}
	if msg.Mentions[0] != "agent-two" || msg.Mentions[1] != "agent-three" {
		t.Errorf("unexpected mentions: %v", msg.Mentions)
	}
}

func TestGetSince(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert 3 messages
	msg1, _ := store.Insert("a", "first", []string{})
	store.Insert("b", "second", []string{})
	store.Insert("c", "third", []string{})

	// Get messages since first one
	msgs, err := store.GetSince(msg1.ID)
	if err != nil {
		t.Fatalf("GetSince failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Body != "second" {
		t.Errorf("expected 'second', got %q", msgs[0].Body)
	}
	if msgs[1].Body != "third" {
		t.Errorf("expected 'third', got %q", msgs[1].Body)
	}
}

func TestGetSinceWithZero(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	store.Insert("a", "first", []string{})
	store.Insert("b", "second", []string{})

	// Get all messages (since ID 0)
	msgs, err := store.GetSince(0)
	if err != nil {
		t.Fatalf("GetSince failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestGetForEntity(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert messages with various mentions
	store.Insert("a", "hello @bob", []string{"bob"})
	store.Insert("b", "hello @alice", []string{"alice"})
	store.Insert("c", "hello @bob and @alice", []string{"bob", "alice"})
	store.Insert("d", "hello @all", []string{"all"})
	store.Insert("e", "no mentions", []string{})

	// Get messages for bob (should include @bob and @all)
	msgs, err := store.GetForEntity("bob", 0)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}

	// Should get: "hello @bob", "hello @bob and @alice", "hello @all"
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages for bob, got %d", len(msgs))
	}
}

func TestGetForEntitySinceID(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	msg1, _ := store.Insert("a", "old @bob", []string{"bob"})
	store.Insert("b", "new @bob", []string{"bob"})

	// Get messages for bob since first message
	msgs, err := store.GetForEntity("bob", msg1.ID)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Body != "new @bob" {
		t.Errorf("expected 'new @bob', got %q", msgs[0].Body)
	}
}

func TestGetRecent(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert 5 messages
	for i := 0; i < 5; i++ {
		store.Insert("sender", "message", []string{})
	}

	// Get last 3
	msgs, err := store.GetRecent(3)
	if err != nil {
		t.Fatalf("GetRecent failed: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestMessageTimestamp(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	before := time.Now().Add(-time.Second)
	msg, _ := store.Insert("sender", "body", []string{})
	after := time.Now().Add(time.Second)

	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Errorf("timestamp %v not in expected range [%v, %v]", msg.Timestamp, before, after)
	}
}

func TestGetForEntityCaseInsensitive(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert message with uppercase mention
	store.Insert("a", "hello @BOB how are you", []string{"bob"})
	store.Insert("b", "hello AGENT wake up", []string{})

	// Search with lowercase should find uppercase
	msgs, err := store.GetForEntity("bob", 0)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for bob, got %d", len(msgs))
	}

	// Search for agent (lowercase) should find AGENT (uppercase)
	msgs, err = store.GetForEntity("agent", 0)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for agent, got %d", len(msgs))
	}
}

func TestGetForEntityWithoutAtSymbol(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	// Insert message without @ symbol - just the name
	store.Insert("a", "hey bob what's up", []string{})
	store.Insert("b", "agent-alpha says hello", []string{})

	// Should find messages containing the name without @
	msgs, err := store.GetForEntity("bob", 0)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for bob, got %d", len(msgs))
	}

	msgs, err = store.GetForEntity("agent-alpha", 0)
	if err != nil {
		t.Fatalf("GetForEntity failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message for agent-alpha, got %d", len(msgs))
	}
}
