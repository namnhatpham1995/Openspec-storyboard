package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"storyboard/internal/openspec"
)

func testApp(t *testing.T, initialProject string) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, err := NewPersistent(filepath.Join(t.TempDir(), "registry.json"), initialProject, logger)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(testApp(t, "").Handler())
	t.Cleanup(server.Close)
	return server
}

func testProjectServer(t *testing.T, projectRoot string) (*httptest.Server, string) {
	t.Helper()
	app := testApp(t, projectRoot)
	projects := app.registry.List()
	if len(projects) != 1 {
		t.Fatalf("registered projects = %d, want 1", len(projects))
	}
	server := httptest.NewServer(app.Handler())
	t.Cleanup(server.Close)
	return server, projects[0].ID
}

func TestHealth(t *testing.T) {
	server := testServer(t)
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
	server := testServer(t)
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
	server := testServer(t)
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
	server := testServer(t)
	for _, path := range []string{"/api/not-a-route", "/api/projects/current", "/api/changes/demo"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404", path, response.StatusCode)
		}
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

	server, projectID := testProjectServer(t, root)

	detailResponse, err := http.Get(server.URL + "/api/projects/" + projectID + "/changes/demo")
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

	response := postJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/tasks/1.1/toggle", body)
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

	conflict := postJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/tasks/1.1/toggle", body)
	defer conflict.Body.Close()
	if conflict.StatusCode != http.StatusConflict {
		t.Errorf("stale toggle status = %d, want 409", conflict.StatusCode)
	}
}

func TestTaskToggleRejectsInvalidJSON(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(projectRoot, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	server, projectID := testProjectServer(t, projectRoot)

	response := postJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/tasks/1.1/toggle", []byte(`{"unknown":true}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", response.StatusCode)
	}
}

func TestArchiveEndpoint(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(t *testing.T, root string)
		stale      bool
		wantStatus int
		wantCode   string
	}{
		{name: "success", wantStatus: http.StatusOK},
		{name: "version conflict", stale: true, wantStatus: http.StatusConflict, wantCode: "file_conflict"},
		{
			name: "archive name collision", wantStatus: http.StatusConflict, wantCode: "archive_name_conflict",
			prepare: func(t *testing.T, root string) {
				t.Helper()
				name := time.Now().Format("2006-01-02") + "-demo"
				if err := os.MkdirAll(filepath.Join(root, "openspec", "changes", "archive", name), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "archive failure includes its cause", wantStatus: http.StatusInternalServerError, wantCode: "archive_failed",
			prepare: func(t *testing.T, root string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(root, "openspec", "changes", "archive"), []byte("not a directory"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			changeDir := filepath.Join(root, "openspec", "changes", "demo")
			if err := os.MkdirAll(changeDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			tasks := []byte("- [x] 1.1 Done\n")
			tasksPath := filepath.Join(changeDir, "tasks.md")
			if err := os.WriteFile(tasksPath, tasks, 0o644); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(tasksPath)
			if err != nil {
				t.Fatal(err)
			}
			version := testVersion(tasks, info.ModTime())
			if tt.stale {
				version.Hash = "stale"
			}
			if tt.prepare != nil {
				tt.prepare(t, root)
			}

			server, projectID := testProjectServer(t, root)
			body, _ := json.Marshal(archiveChangeRequest{Version: version})
			response := postJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/archive", body)
			defer response.Body.Close()
			if response.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, tt.wantStatus)
			}
			if tt.wantCode != "" {
				var payload struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if payload.Error.Code != tt.wantCode {
					t.Errorf("error code = %q, want %q", payload.Error.Code, tt.wantCode)
				}
				return
			}
			var result openspec.ArchiveResult
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				t.Fatal(err)
			}
			if result.Name == "" || result.Path == "" {
				t.Errorf("result = %+v", result)
			}
		})
	}
}

func TestTaskTextAndArtifactEndpoints(t *testing.T) {
	root := t.TempDir()
	changeDir := filepath.Join(root, "openspec", "changes", "demo")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proposal := []byte("# Original\n")
	design := []byte("# Design\n")
	spec := []byte("# Spec\n")
	tasks := []byte("## Work\n- [ ] 1.1 Original task\n")
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), proposal, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), tasks, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "design.md"), design, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(changeDir, "specs", "capability"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, "specs", "capability", "spec.md"), spec, 0o644); err != nil {
		t.Fatal(err)
	}
	proposalInfo, _ := os.Stat(filepath.Join(changeDir, "proposal.md"))
	designInfo, _ := os.Stat(filepath.Join(changeDir, "design.md"))
	specInfo, _ := os.Stat(filepath.Join(changeDir, "specs", "capability", "spec.md"))
	tasksInfo, _ := os.Stat(filepath.Join(changeDir, "tasks.md"))
	proposalVersion := testVersion(proposal, proposalInfo.ModTime())
	designVersion := testVersion(design, designInfo.ModTime())
	specVersion := testVersion(spec, specInfo.ModTime())
	tasksVersion := testVersion(tasks, tasksInfo.ModTime())

	server, projectID := testProjectServer(t, root)

	taskBody, _ := json.Marshal(taskTextRequest{Text: "Edited task", Version: tasksVersion})
	taskResponse := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/tasks/1.1/text", taskBody)
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

	proposalBody, _ := json.Marshal(artifactTextRequest{Content: "# Edited\n", Version: proposalVersion})
	proposalResponse := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/artifacts/proposal.md", proposalBody)
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

	designBody, _ := json.Marshal(artifactTextRequest{Content: "# Edited design\n", Version: designVersion})
	designResponse := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/artifacts/design.md", designBody)
	defer designResponse.Body.Close()
	if designResponse.StatusCode != http.StatusOK {
		t.Fatalf("design status = %d, want 200", designResponse.StatusCode)
	}
	var designResult openspec.ArtifactWriteResult
	if err := json.NewDecoder(designResponse.Body).Decode(&designResult); err != nil {
		t.Fatal(err)
	}
	if designResult.Artifact.Kind != "design" || designResult.Artifact.Path != "design.md" {
		t.Errorf("design result = %+v", designResult)
	}

	specBody, _ := json.Marshal(artifactTextRequest{Content: "# Edited spec\n", Version: specVersion})
	specResponse := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/artifacts/specs/capability/spec.md", specBody)
	defer specResponse.Body.Close()
	if specResponse.StatusCode != http.StatusOK {
		t.Fatalf("spec status = %d, want 200", specResponse.StatusCode)
	}
	var specResult openspec.ArtifactWriteResult
	if err := json.NewDecoder(specResponse.Body).Decode(&specResult); err != nil {
		t.Fatal(err)
	}
	if specResult.Artifact.Kind != "spec" || specResult.Artifact.Path != "specs/capability/spec.md" {
		t.Errorf("spec result = %+v", specResult)
	}

	missingResponse := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/artifacts/missing.md", designBody)
	defer missingResponse.Body.Close()
	if missingResponse.StatusCode != http.StatusNotFound {
		t.Errorf("missing artifact status = %d, want 404", missingResponse.StatusCode)
	}

	conflict := putJSON(t, server.URL+"/api/projects/"+projectID+"/changes/demo/artifacts/proposal.md", proposalBody)
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
