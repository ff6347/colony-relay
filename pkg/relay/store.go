// ABOUTME: SQLite storage layer for relay messages
// ABOUTME: Provides CRUD operations for message persistence

package relay

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite"
)

// Message represents a single message in the relay
type Message struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"ts"`
	Sender    string    `json:"from"`
	Body      string    `json:"body"`
	Mentions  []string  `json:"mentions,omitempty"`
}

// Presence represents an agent's presence on the relay
type Presence struct {
	Name     string    `json:"name"`
	LastSeen time.Time `json:"last_seen"`
}

// Store provides SQLite-backed message storage
type Store struct {
	db *sql.DB
}

// NewStore creates a new message store with the given SQLite database path.
// Use ":memory:" for an in-memory database (useful for testing).
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Insert adds a new message to the store
func (s *Store) Insert(sender, body string, mentions []string) (*Message, error) {
	mentionsJSON, err := json.Marshal(mentions)
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(
		`INSERT INTO messages (sender, body, mentions) VALUES (?, ?, ?)`,
		sender, body, string(mentionsJSON),
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Fetch the inserted message to get the timestamp
	return s.getByID(id)
}

// GetSince returns all messages with ID greater than sinceID
func (s *Store) GetSince(sinceID int64) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, ts, sender, body, mentions FROM messages WHERE id > ? ORDER BY id ASC`,
		sinceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetForEntity returns messages containing the entity name or "@all", since the given ID.
// Search is case-insensitive and matches the name anywhere in the body.
func (s *Store) GetForEntity(entity string, sinceID int64) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, ts, sender, body, mentions FROM messages
		 WHERE id > ?
		 AND (
			 LOWER(body) LIKE LOWER(?)
			 OR LOWER(body) LIKE '%@all%'
		 )
		 ORDER BY id ASC`,
		sinceID,
		`%`+entity+`%`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// Clear removes all messages from the store
func (s *Store) Clear() error {
	_, err := s.db.Exec(`DELETE FROM messages`)
	return err
}

// UpdatePresence updates the last_seen timestamp for an agent
func (s *Store) UpdatePresence(name string) error {
	return s.UpdatePresenceAt(name, time.Now())
}

// UpdatePresenceAt updates the last_seen timestamp for an agent to a specific time
func (s *Store) UpdatePresenceAt(name string, when time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO presence (name, last_seen) VALUES (?, ?)
		 ON CONFLICT(name) DO UPDATE SET last_seen = excluded.last_seen`,
		name, when.UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

// GetPresence returns all agents seen within the given time window
func (s *Store) GetPresence(windowMinutes float64) ([]Presence, error) {
	cutoff := time.Now().Add(-time.Duration(windowMinutes * float64(time.Minute)))

	rows, err := s.db.Query(
		`SELECT name, last_seen FROM presence WHERE last_seen > ? ORDER BY last_seen DESC`,
		cutoff.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Presence
	for rows.Next() {
		var name string
		var lastSeenStr string

		if err := rows.Scan(&name, &lastSeenStr); err != nil {
			return nil, err
		}

		lastSeen := parseTimestamp(lastSeenStr)
		result = append(result, Presence{
			Name:     name,
			LastSeen: lastSeen,
		})
	}

	return result, rows.Err()
}

// ClearPresence removes all presence data (for testing)
func (s *Store) ClearPresence() error {
	_, err := s.db.Exec(`DELETE FROM presence`)
	return err
}

// GetRecent returns the most recent n messages
func (s *Store) GetRecent(limit int) ([]*Message, error) {
	rows, err := s.db.Query(
		`SELECT id, ts, sender, body, mentions FROM messages ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}

func (s *Store) getByID(id int64) (*Message, error) {
	row := s.db.QueryRow(
		`SELECT id, ts, sender, body, mentions FROM messages WHERE id = ?`,
		id,
	)

	var msg Message
	var tsStr string
	var mentionsJSON string

	err := row.Scan(&msg.ID, &tsStr, &msg.Sender, &msg.Body, &mentionsJSON)
	if err != nil {
		return nil, err
	}

	msg.Timestamp = parseTimestamp(tsStr)
	json.Unmarshal([]byte(mentionsJSON), &msg.Mentions)

	return &msg, nil
}

func scanMessages(rows *sql.Rows) ([]*Message, error) {
	var msgs []*Message

	for rows.Next() {
		var msg Message
		var tsStr string
		var mentionsJSON string

		err := rows.Scan(&msg.ID, &tsStr, &msg.Sender, &msg.Body, &mentionsJSON)
		if err != nil {
			return nil, err
		}

		msg.Timestamp = parseTimestamp(tsStr)
		json.Unmarshal([]byte(mentionsJSON), &msg.Mentions)

		msgs = append(msgs, &msg)
	}

	return msgs, rows.Err()
}

// parseTimestamp tries multiple SQLite timestamp formats
func parseTimestamp(s string) time.Time {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Now() // fallback to current time if parsing fails
}

func initSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ts DATETIME DEFAULT CURRENT_TIMESTAMP,
			sender TEXT NOT NULL,
			body TEXT NOT NULL,
			mentions TEXT DEFAULT '[]'
		);
		CREATE INDEX IF NOT EXISTS idx_messages_ts ON messages(ts);

		CREATE TABLE IF NOT EXISTS presence (
			name TEXT PRIMARY KEY,
			last_seen DATETIME NOT NULL
		);
	`
	_, err := db.Exec(schema)
	return err
}
