package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestServesEmbeddedSPAAndDeepLinks(t *testing.T) {
	server, _ := testServer(t)
	for _, route := range []string{"/", "/projects/example/changes/add-login"} {
		response, err := http.Get(server.URL + route)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", route, response.StatusCode)
		}
		if !bytes.Contains(body, []byte("<title>Storyboard</title>")) {
			t.Errorf("GET %s did not return the Storyboard index", route)
		}
	}
}

func TestServesEmbeddedStaticAsset(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", contentType)
	}
}

func TestUnknownAPIRouteDoesNotFallBackToSPA(t *testing.T) {
	server, _ := testServer(t)
	response, err := http.Get(server.URL + "/api/not-a-route")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.StatusCode)
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

func TestTaskTextAndProposalEndpoints(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proposal := []byte("# Original\n")
	tasks := []byte("## Work\n- [ ] 1.1 Original task\n")
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), proposal, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), tasks, 0o644); err != nil {
		t.Fatal(err)
	}
	proposalInfo, _ := os.Stat(filepath.Join(changeDir, "proposal.md"))
	tasksInfo, _ := os.Stat(filepath.Join(changeDir, "tasks.md"))
	proposalVersion := testVersion(proposal, proposalInfo.ModTime())
	tasksVersion := testVersion(tasks, tasksInfo.ModTime())

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(New(root, logger).Handler())
	defer server.Close()

	taskBody, _ := json.Marshal(taskTextRequest{Text: "Edited task", Version: tasksVersion})
	taskResponse := putJSON(t, server.URL+"/api/changes/demo/tasks/1.1/text", taskBody)
	defer taskResponse.Body.Close()
	if taskResponse.StatusCode != http.StatusOK {
		t.Fatalf("task text status = %d, want 200", taskResponse.StatusCode)
	}
	var taskResult openspec.TaskTextResult
	if err := json.NewDecoder(taskResponse.Body).Decode(&taskResult); err != nil {
		t.Fatal(err)
	}
	if taskResult.Task.Text != "Edited task" || taskResult.Version.Hash == tasksVersion.Hash {
		t.Errorf("task result = %+v", taskResult)
	}

	proposalBody, _ := json.Marshal(proposalTextRequest{Content: "# Edited\n", Version: proposalVersion})
	proposalResponse := putJSON(t, server.URL+"/api/changes/demo/artifacts/proposal", proposalBody)
	defer proposalResponse.Body.Close()
	if proposalResponse.StatusCode != http.StatusOK {
		t.Fatalf("proposal status = %d, want 200", proposalResponse.StatusCode)
	}
	var proposalResult openspec.ArtifactWriteResult
	if err := json.NewDecoder(proposalResponse.Body).Decode(&proposalResult); err != nil {
		t.Fatal(err)
	}
	if proposalResult.Artifact.Content != "# Edited\n" || proposalResult.Artifact.Version.Hash == proposalVersion.Hash {
		t.Errorf("proposal result = %+v", proposalResult)
	}

	conflict := putJSON(t, server.URL+"/api/changes/demo/artifacts/proposal", proposalBody)
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Errorf("stale proposal status = %d, want 409", conflict.StatusCode)
	}
}

func TestProjectRegistryAPIAndDisconnectedState(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "alpha")
	changeDir := filepath.Join(projectRoot, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, err := NewPersistent(filepath.Join(root, "registry.json"), "", logger)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	addBody, _ := json.Marshal(addProjectRequest{Path: projectRoot})
	added := postJSON(t, server.URL+"/api/projects", addBody)
	if added.StatusCode != http.StatusCreated {
		t.Fatalf("add status = %d, want 201", added.StatusCode)
	}
	var project registeredProjectView
	if err := json.NewDecoder(added.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	added.Body.Close()
	if !project.Connected || project.Name != "alpha" || len(project.Changes) != 1 {
		t.Fatalf("added project = %+v", project)
	}

	detail, err := http.Get(server.URL + "/api/projects/" + project.ID + "/changes/demo")
	if err != nil {
		t.Fatal(err)
	}
	defer detail.Body.Close()
	if detail.StatusCode != http.StatusOK {
		t.Fatalf("registered detail status = %d, want 200", detail.StatusCode)
	}

	disconnectedPath := projectRoot + "-offline"
	if err := os.Rename(projectRoot, disconnectedPath); err != nil {
		t.Fatal(err)
	}
	projectsResponse, err := http.Get(server.URL + "/api/projects")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Projects []registeredProjectView `json:"projects"`
	}
	if err := json.NewDecoder(projectsResponse.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	projectsResponse.Body.Close()
	if len(list.Projects) != 1 || list.Projects[0].Connected {
		t.Fatalf("disconnected projects = %+v", list.Projects)
	}

	request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/projects/"+project.ID, nil)
	removed, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	removed.Body.Close()
	if removed.StatusCode != http.StatusNoContent {
		t.Fatalf("remove status = %d, want 204", removed.StatusCode)
	}
}

func TestProjectWatcherStartsAndStopsWithRegistry(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "watched")
	if err := os.MkdirAll(filepath.Join(projectRoot, "openspec", "changes"), 0o755); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, err := NewPersistent(filepath.Join(root, "registry.json"), "", logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go app.WatchProjects(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		app.watchMu.Lock()
		ready := app.watchCtx != nil
		app.watchMu.Unlock()
		if ready {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	project, err := app.registry.Add(projectRoot)
	if err != nil {
		t.Fatal(err)
	}
	app.startProjectWatcher(project)
	waitForWatcherCount(t, app, 1)
	app.stopProjectWatcher(project.ID)
	waitForWatcherCount(t, app, 0)
}

func waitForWatcherCount(t *testing.T, app *Server, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		app.watchMu.Lock()
		got := len(app.watchers)
		app.watchMu.Unlock()
		if got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("watcher count did not become %d", want)
}

func postJSON(t *testing.T, url string, body []byte) *http.Response {
	t.Helper()
	response, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func putJSON(t *testing.T, url string, body []byte) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func testVersion(content []byte, modTime time.Time) openspec.FileVersion {
	sum := sha256.Sum256(content)
	return openspec.FileVersion{ModTime: modTime, Hash: hex.EncodeToString(sum[:])}
}
