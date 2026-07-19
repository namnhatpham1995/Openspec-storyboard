package server

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestTaskToggleAndConflict(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Work\n- [ ] 1.1 Toggle me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(New(root, logger).Handler())
	defer server.Close()

	detailResponse, err := http.Get(server.URL + "/api/changes/demo")
	if err != nil {
		t.Fatal(err)
	}
	defer detailResponse.Body.Close()
	var detail openspec.ChangeDetail
	if err := json.NewDecoder(detailResponse.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	var version openspec.FileVersion
	for _, artifact := range detail.ArtifactFiles {
		if artifact.Path == "tasks.md" {
			version = artifact.Version
		}
	}
	body, _ := json.Marshal(toggleTaskRequest{Version: version})

	response := postJSON(t, server.URL+"/api/changes/demo/tasks/1.1/toggle", body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("toggle status = %d, want 200", response.StatusCode)
	}
	var result openspec.ToggleResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if !result.Task.Checked || result.Version.Hash == version.Hash {
		t.Errorf("result = %+v", result)
	}

	conflict := postJSON(t, server.URL+"/api/changes/demo/tasks/1.1/toggle", body)
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Errorf("stale toggle status = %d, want 409", conflict.StatusCode)
	}
}

func TestTaskToggleRejectsInvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(New(t.TempDir(), logger).Handler())
	defer server.Close()

	response := postJSON(t, server.URL+"/api/changes/demo/tasks/1.1/toggle", []byte(`{"unknown":true}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", response.StatusCode)
	}
}

func postJSON(t *testing.T, url string, body []byte) *http.Response {
	t.Helper()
	response, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return response
}
