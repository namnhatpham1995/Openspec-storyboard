package directorybrowser

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBrowserList(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "Alpha", "empty"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	browser := New()
	browser.homeDir = func() (string, error) { return root, nil }
	browser.locations = func(home string) ([]Location, error) {
		return []Location{{Name: "Home", Path: home}, {Name: "Duplicate", Path: home}}, nil
	}

	tests := []struct {
		name      string
		path      string
		wantPath  string
		wantNames []string
	}{
		{name: "default home", path: "", wantPath: root, wantNames: []string{"Alpha", "empty", "zeta"}},
		{name: "explicit empty directory", path: filepath.Join(root, "empty"), wantPath: filepath.Join(root, "empty"), wantNames: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listing, err := browser.List(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if listing.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", listing.Path, tt.wantPath)
			}
			gotNames := make([]string, 0, len(listing.Directories))
			for _, directory := range listing.Directories {
				gotNames = append(gotNames, directory.Name)
			}
			if !equalStrings(gotNames, tt.wantNames) {
				t.Errorf("directories = %q, want %q", gotNames, tt.wantNames)
			}
			if len(listing.Locations) != 1 || listing.Locations[0].Path != root {
				t.Errorf("locations = %+v", listing.Locations)
			}
		})
	}
}

func TestBrowserListSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("creating symlink requires additional Windows privileges: %v", err)
		}
		t.Fatal(err)
	}

	browser := New()
	listing, err := browser.List(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, directory := range listing.Directories {
		if directory.Name == "linked" {
			found = true
		}
	}
	if !found {
		t.Errorf("symlinked directory missing from %+v", listing.Directories)
	}

	linkedListing, err := browser.List(link)
	if err != nil {
		t.Fatal(err)
	}
	if linkedListing.Path != target {
		t.Errorf("canonical linked path = %q, want %q", linkedListing.Path, target)
	}
}

func TestBrowserListErrors(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{name: "relative", path: "relative", wantErr: ErrPathNotAbsolute},
		{name: "file", path: filePath, wantErr: ErrNotDirectory},
		{name: "missing", path: filepath.Join(root, "missing"), wantErr: fs.ErrNotExist},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New().List(tt.path)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrowserListPermissionError(t *testing.T) {
	root := t.TempDir()
	browser := New()
	browser.readDir = func(string) ([]os.DirEntry, error) { return nil, fs.ErrPermission }

	_, err := browser.List(root)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("error = %v, want permission error", err)
	}
}

func TestBrowserParentNavigation(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	listing, err := New().List(child)
	if err != nil {
		t.Fatal(err)
	}
	if listing.ParentPath != root {
		t.Errorf("parent = %q, want %q", listing.ParentPath, root)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
