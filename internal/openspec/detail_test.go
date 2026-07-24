package openspec

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestReadChangeDetail(t *testing.T) {
	detail, err := ReadChangeDetail(os.DirFS("testdata/real-project"), "build-storyboard-v1")
	if err != nil {
		t.Fatalf("ReadChangeDetail() error = %v", err)
	}
	if detail.Name != "build-storyboard-v1" {
		t.Errorf("Name = %q, want build-storyboard-v1", detail.Name)
	}
	if len(detail.ArtifactFiles) < 4 {
		t.Fatalf("len(ArtifactFiles) = %d, want at least 4", len(detail.ArtifactFiles))
	}

	for _, artifact := range detail.ArtifactFiles {
		if artifact.Content == "" {
			t.Errorf("artifact %q has empty content", artifact.Path)
		}
		if len(artifact.Version.Hash) != 64 {
			t.Errorf("artifact %q hash length = %d, want 64", artifact.Path, len(artifact.Version.Hash))
		}
	}

	foundProposal := false
	for _, artifact := range detail.ArtifactFiles {
		if artifact.Path == "proposal.md" {
			foundProposal = true
			if artifact.Kind != "proposal" || !strings.Contains(artifact.Content, "# Proposal") {
				t.Errorf("proposal artifact = %+v", artifact)
			}
		}
	}
	if !foundProposal {
		t.Error("proposal.md not returned")
	}
}

func TestMarkdownArtifactPathsMatchChangeDetailArtifacts(t *testing.T) {
	fsys := os.DirFS("testdata/real-project")
	const changePath = "openspec/changes/build-storyboard-v1"
	paths, err := MarkdownArtifactPaths(fsys, changePath)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := ReadChangeDetail(fsys, "build-storyboard-v1")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != len(detail.ArtifactFiles) {
		t.Fatalf("paths = %d, artifacts = %d", len(paths), len(detail.ArtifactFiles))
	}
	for i, artifactPath := range paths {
		want := strings.TrimPrefix(artifactPath, changePath+"/")
		if got := detail.ArtifactFiles[i].Path; got != want {
			t.Errorf("artifact %d path = %q, want %q", i, got, want)
		}
	}
}

func TestReadChangeDetail_Archived(t *testing.T) {
	detail, err := ReadChangeDetail(os.DirFS("testdata/with-archive"), "old-feature")
	if err != nil {
		t.Fatalf("ReadChangeDetail() error = %v", err)
	}
	if !detail.Archived || detail.Status != StatusArchived {
		t.Errorf("archived detail = %+v", detail.Change)
	}
}

func TestReadChangeDetail_NotFoundOrInvalidName(t *testing.T) {
	for _, name := range []string{"missing", "../old-feature", "archive/old-feature", ""} {
		_, err := ReadChangeDetail(os.DirFS("testdata/with-archive"), name)
		if !errors.Is(err, ErrChangeNotFound) {
			t.Errorf("ReadChangeDetail(%q) error = %v, want ErrChangeNotFound", name, err)
		}
	}
}
