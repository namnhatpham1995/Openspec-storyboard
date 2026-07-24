package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"storyboard/internal/directorybrowser"
)

type fakeDirectoryLister struct {
	listing   directorybrowser.Listing
	err       error
	requested string
}

func (f *fakeDirectoryLister) List(path string) (directorybrowser.Listing, error) {
	f.requested = path
	return f.listing, f.err
}

func TestDirectoriesEndpointDefaultAndExplicitPath(t *testing.T) {
	explicit := filepath.Join(t.TempDir(), "project")
	tests := []struct {
		name      string
		path      string
		wantQuery string
	}{
		{name: "default", path: "", wantQuery: ""},
		{name: "explicit", path: explicit, wantQuery: explicit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeDirectoryLister{listing: directorybrowser.Listing{
				Path:        "C:\\Users\\demo",
				Directories: []directorybrowser.Directory{},
				Locations:   []directorybrowser.Location{{Name: "Home", Path: "C:\\Users\\demo"}},
			}}
			app := testApp(t, "")
			app.directories = fake
			server := httptest.NewServer(app.Handler())
			defer server.Close()

			endpoint := server.URL + "/api/filesystem/directories"
			if tt.path != "" {
				endpoint += "?path=" + url.QueryEscape(tt.path)
			}
			response, err := http.Get(endpoint)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}
			if fake.requested != tt.wantQuery {
				t.Errorf("requested path = %q, want %q", fake.requested, tt.wantQuery)
			}
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Errorf("Cache-Control = %q", response.Header.Get("Cache-Control"))
			}
			var listing directorybrowser.Listing
			if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
				t.Fatal(err)
			}
			if len(listing.Locations) != 1 || listing.Locations[0].Name != "Home" {
				t.Errorf("locations = %+v", listing.Locations)
			}
		})
	}
}

func TestDirectoriesEndpointReturnsSortedDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "Alpha"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hidden from response"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := testApp(t, "")
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	response, err := http.Get(server.URL + "/api/filesystem/directories?path=" + url.QueryEscape(root))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var listing directorybrowser.Listing
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		t.Fatal(err)
	}
	if len(listing.Directories) != 2 || listing.Directories[0].Name != "Alpha" || listing.Directories[1].Name != "zeta" {
		t.Errorf("directories = %+v", listing.Directories)
	}
}

func TestDirectoriesEndpointErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "relative", err: directorybrowser.ErrPathNotAbsolute, wantStatus: http.StatusBadRequest, wantCode: "invalid_directory_path"},
		{name: "not directory", err: directorybrowser.ErrNotDirectory, wantStatus: http.StatusUnprocessableEntity, wantCode: "not_a_directory"},
		{name: "missing", err: fs.ErrNotExist, wantStatus: http.StatusNotFound, wantCode: "directory_not_found"},
		{name: "inaccessible", err: fs.ErrPermission, wantStatus: http.StatusForbidden, wantCode: "directory_inaccessible"},
		{name: "unexpected", err: errors.New("disk failure"), wantStatus: http.StatusInternalServerError, wantCode: "directory_read_failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeDirectoryLister{err: tt.err}
			app := testApp(t, "")
			app.directories = fake
			request := httptest.NewRequest(http.MethodGet, "/api/filesystem/directories?path=relative", nil)
			request.Host = "127.0.0.1"
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", body.Error.Code, tt.wantCode)
			}
		})
	}
}
