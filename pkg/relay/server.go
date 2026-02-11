// ABOUTME: HTTP API handlers for the relay server
// ABOUTME: Provides POST /messages, GET /messages, GET /stream (SSE), GET /presence, and web UI

package relay

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

//go:embed web/index.html
var webContent embed.FS

// DefaultPresenceMinutes is the default time window for presence tracking (30 minutes)
const DefaultPresenceMinutes = 30.0

// Server handles HTTP requests for the relay API
type Server struct {
	store           *Store
	mux             *http.ServeMux
	presenceMinutes float64

	// SSE subscriber management
	subscribersMu sync.RWMutex
	subscribers   map[chan *Message]struct{}
}

// NewServer creates a new HTTP server with the given store
func NewServer(store *Store) *Server {
	s := &Server{
		store:           store,
		mux:             http.NewServeMux(),
		subscribers:     make(map[chan *Message]struct{}),
		presenceMinutes: DefaultPresenceMinutes,
	}
	s.mux.HandleFunc("/", s.handleUI)
	s.mux.HandleFunc("/messages", s.handleMessages)
	s.mux.HandleFunc("/stream", s.handleStream)
	s.mux.HandleFunc("/presence", s.handlePresence)
	return s
}

// SetPresenceMinutes sets the presence timeout window
func (s *Server) SetPresenceMinutes(minutes float64) {
	s.presenceMinutes = minutes
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handleUI serves the web UI at the root path
func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	content, err := webContent.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.postMessage(w, r)
	case http.MethodGet:
		s.getMessages(w, r)
	case http.MethodDelete:
		s.clearMessages(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// clearMessages handles DELETE /messages
func (s *Server) clearMessages(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Clear(); err != nil {
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// postMessage handles POST /messages
func (s *Server) postMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.From == "" {
		http.Error(w, "missing 'from' field", http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		http.Error(w, "missing 'body' field", http.StatusBadRequest)
		return
	}

	// Parse mentions from body
	mentions := ParseMentions(req.Body)
	mentionList := mentions.Names
	if mentions.All {
		mentionList = append(mentionList, "all")
	}
	if mentions.Here {
		mentionList = append(mentionList, "here")
	}

	msg, err := s.store.Insert(req.From, req.Body, mentionList)
	if err != nil {
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update presence for sender
	s.store.UpdatePresence(req.From)

	// Broadcast to SSE subscribers
	s.broadcast(msg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": msg.ID,
		"ts": msg.Timestamp.Format("2006-01-02T15:04:05Z"),
	})
}

// getMessages handles GET /messages
func (s *Server) getMessages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	// Parse query parameters
	forEntity := query.Get("for")
	sinceStr := query.Get("since")
	limitStr := query.Get("limit")
	all := query.Get("all") == "true"

	var sinceID int64
	if sinceStr != "" {
		var err error
		sinceID, err = strconv.ParseInt(sinceStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid 'since' parameter", http.StatusBadRequest)
			return
		}
	}

	var limit int
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid 'limit' parameter", http.StatusBadRequest)
			return
		}
	}

	var msgs []*Message
	var err error

	if forEntity != "" && !all {
		// Get messages for specific entity (filtered by mentions)
		msgs, err = s.store.GetForEntity(forEntity, sinceID)
		// Update presence for the fetching entity
		s.store.UpdatePresence(forEntity)
	} else if limit > 0 {
		// Get recent messages with limit
		msgs, err = s.store.GetRecent(limit)
	} else {
		// Get all messages since ID
		msgs, err = s.store.GetSince(sinceID)
	}

	if err != nil {
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

// handleStream handles GET /stream for Server-Sent Events
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if flushing is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Send initial comment to establish connection
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	// Create a channel for this subscriber
	msgCh := make(chan *Message, 10)
	s.subscribe(msgCh)
	defer s.unsubscribe(msgCh)

	// Get the client's context for disconnect detection
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgCh:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// subscribe adds a channel to the subscriber list
func (s *Server) subscribe(ch chan *Message) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	s.subscribers[ch] = struct{}{}
}

// unsubscribe removes a channel from the subscriber list
func (s *Server) unsubscribe(ch chan *Message) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()
	delete(s.subscribers, ch)
	close(ch)
}

// broadcast sends a message to all SSE subscribers
func (s *Server) broadcast(msg *Message) {
	s.subscribersMu.RLock()
	defer s.subscribersMu.RUnlock()

	for ch := range s.subscribers {
		select {
		case ch <- msg:
		default:
			// Channel full, skip this subscriber
		}
	}
}

// handlePresence handles GET /presence
func (s *Server) handlePresence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	presences, err := s.store.GetPresence(s.presenceMinutes)
	if err != nil {
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presences)
}
