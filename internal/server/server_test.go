package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"storyboard/internal/openspec"
)

func testServer(t *testing.T) (*httptest.Server, fs.FS) {
	t.Helper()
	projectFS := os.DirFS("../openspec/testdata/real-project")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(NewWithFS("test-project", projectFS, logger).Handler())
	t.Cleanup(server.Close)
	return server, projectFS
}

func TestHealth(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if response.Header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", response.Header.Get("Content-Type"))
	}
}

func TestCurrentProject(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/api/projects/current")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	var project openspec.Project
	if err := json.NewDecoder(response.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	if project.Root != "test-project" || len(project.Changes) != 1 {
		t.Errorf("project = %+v", project)
	}
}

func TestChangeDetail(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/api/changes/build-storyboard-v1")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var detail openspec.ChangeDetail
	if err := json.NewDecoder(response.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Name != "build-storyboard-v1" || len(detail.ArtifactFiles) == 0 {
		t.Errorf("detail = %+v", detail)
	}
}

func TestChangeDetailNotFound(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/api/changes/missing")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.StatusCode)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "change_not_found" {
		t.Errorf("error code = %q", body.Error.Code)
	}
}
