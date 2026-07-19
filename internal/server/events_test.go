package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	projectwatch "storyboard/internal/watch"
)

func TestEventsEndpointBroadcasts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := NewWithFS("test-project", nil, logger)
	testServer := httptest.NewServer(app.Handler())
	defer testServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("Content-Type = %q", response.Header.Get("Content-Type"))
	}

	lines := make(chan string, 4)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "data: ") {
				lines <- strings.TrimPrefix(scanner.Text(), "data: ")
			}
		}
	}()

	ready := readLiveEvent(t, lines)
	if ready.Type != "ready" {
		t.Fatalf("first event type = %q, want ready", ready.Type)
	}
	app.events.publish(liveEvent{
		Type:       "project_changed",
		Activities: []projectwatch.Activity{{Message: "tasks.md · 1.1 checked"}},
		Timestamp:  time.Now(),
	})
	changed := readLiveEvent(t, lines)
	if changed.Type != "project_changed" || len(changed.Activities) != 1 {
		t.Errorf("event = %+v", changed)
	}
}

func readLiveEvent(t *testing.T, lines <-chan string) liveEvent {
	t.Helper()
	select {
	case line := <-lines:
		var event liveEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE event")
		return liveEvent{}
	}
}
