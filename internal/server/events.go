package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	projectwatch "storyboard/internal/watch"
)

type liveEvent struct {
	Type        string                  `json:"type"`
	ProjectRoot string                  `json:"projectRoot,omitempty"`
	Activities  []projectwatch.Activity `json:"activities,omitempty"`
	Timestamp   time.Time               `json:"timestamp"`
}

type eventHub struct {
	mu      sync.Mutex
	clients map[chan liveEvent]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{clients: make(map[chan liveEvent]struct{})}
}

func (h *eventHub) subscribe() (<-chan liveEvent, func()) {
	client := make(chan liveEvent, 8)
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
	return client, func() {
		h.mu.Lock()
		if _, exists := h.clients[client]; exists {
			delete(h.clients, client)
			close(client)
		}
		h.mu.Unlock()
	}
}

func (h *eventHub) publish(event liveEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client <- event:
		default:
			// A slow browser will recover by reconnecting and refetching; never
			// let one client block watcher progress for everyone else.
		}
	}
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, "streaming_unavailable", "event streaming is unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	events, unsubscribe := s.events.subscribe()
	defer unsubscribe()
	if err := writeSSE(w, liveEvent{Type: "ready", Timestamp: time.Now()}); err != nil {
		return
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if err := writeSSE(w, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event liveEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", payload)
	return err
}

// WatchProject runs the project's recursive watcher until ctx is cancelled.
// Failed reparses are logged and retried on the next filesystem event.
func (s *Server) WatchProject(ctx context.Context) error {
	if s.writeRoot == "" {
		return nil
	}
	snapshot, err := projectwatch.Capture(s.projectRoot)
	if err != nil {
		return err
	}
	watcher, err := projectwatch.New(s.projectRoot, 300*time.Millisecond)
	if err != nil {
		return err
	}
	return watcher.Run(ctx, func() {
		next, err := projectwatch.Capture(s.projectRoot)
		if err != nil {
			s.logger.Error("reloading changed project", "error", err)
			return
		}
		now := time.Now()
		activities := projectwatch.Diff(snapshot, next, now)
		snapshot = next
		s.events.publish(liveEvent{
			Type:        "project_changed",
			ProjectRoot: s.projectRoot,
			Activities:  activities,
			Timestamp:   now,
		})
	})
}
