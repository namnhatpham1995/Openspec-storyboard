package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	projectregistry "storyboard/internal/registry"
	projectwatch "storyboard/internal/watch"
)

type liveEvent struct {
	Type        string                  `json:"type"`
	ProjectID   string                  `json:"projectId,omitempty"`
	ProjectName string                  `json:"projectName,omitempty"`
	ProjectRoot string                  `json:"projectRoot,omitempty"`
	Activities  []projectwatch.Activity `json:"activities,omitempty"`
	Timestamp   time.Time               `json:"timestamp"`
}

type projectWatcher struct {
	cancel context.CancelFunc
	done   chan struct{}
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

// WatchProjects maintains one watcher for every registered project and reacts
// to projects added or removed through the API.
func (s *Server) WatchProjects(ctx context.Context) error {
	s.watchMu.Lock()
	s.watchCtx = ctx
	s.watchMu.Unlock()
	for _, project := range s.registry.List() {
		s.startProjectWatcher(project)
	}
	<-ctx.Done()
	s.watchMu.Lock()
	watchers := make([]*projectWatcher, 0, len(s.watchers))
	for id, watcher := range s.watchers {
		watcher.cancel()
		watchers = append(watchers, watcher)
		delete(s.watchers, id)
	}
	s.watchCtx = nil
	s.watchMu.Unlock()
	for _, watcher := range watchers {
		<-watcher.done
	}
	return nil
}

func (s *Server) startProjectWatcher(project projectregistry.Project) {
	s.watchMu.Lock()
	if s.watchCtx == nil {
		s.watchMu.Unlock()
		return
	}
	if _, exists := s.watchers[project.ID]; exists {
		s.watchMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(s.watchCtx)
	handle := &projectWatcher{cancel: cancel, done: make(chan struct{})}
	s.watchers[project.ID] = handle
	s.watchMu.Unlock()

	go func() {
		defer close(handle.done)
		err := s.watchRoot(ctx, project.ID, project.Name, project.Path)
		s.watchMu.Lock()
		if s.watchers[project.ID] == handle {
			delete(s.watchers, project.ID)
		}
		s.watchMu.Unlock()
		if err != nil && ctx.Err() == nil {
			s.logger.Error("project live updates stopped", "project", project.Path, "error", err)
		}
	}()
}

func (s *Server) stopProjectWatcher(id string) {
	s.watchMu.Lock()
	watcher := s.watchers[id]
	if watcher != nil {
		delete(s.watchers, id)
	}
	s.watchMu.Unlock()
	if watcher != nil {
		watcher.cancel()
		<-watcher.done
	}
}

func (s *Server) watchRoot(ctx context.Context, projectID, projectName, projectRoot string) error {
	snapshot, err := projectwatch.Capture(projectRoot)
	if err != nil {
		return err
	}
	watcher, err := projectwatch.New(projectRoot, 300*time.Millisecond)
	if err != nil {
		return err
	}
	return watcher.Run(ctx, func() {
		next, err := projectwatch.Capture(projectRoot)
		if err != nil {
			s.logger.Error("reloading changed project", "error", err)
			return
		}
		now := time.Now()
		activities := projectwatch.Diff(snapshot, next, now)
		snapshot = next
		s.events.publish(liveEvent{
			Type:        "project_changed",
			ProjectID:   projectID,
			ProjectName: projectName,
			ProjectRoot: projectRoot,
			Activities:  activities,
			Timestamp:   now,
		})
	})
}
