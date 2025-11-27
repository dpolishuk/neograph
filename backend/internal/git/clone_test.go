package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/owner/repo", "repo"},
		{"https://github.com/owner/repo.git", "repo"},
		{"git@github.com:owner/repo.git", "repo"},
		{"http://gitlab.com/group/project", "project"},
	}

	for _, tt := range tests {
		got := ExtractRepoName(tt.url)
		if got != tt.expected {
			t.Errorf("ExtractRepoName(%s) = %s, want %s", tt.url, got, tt.expected)
		}
	}
}

func TestCloneRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Use a small public repo for testing
	repoURL := "https://github.com/kelseyhightower/nocode"

	tmpDir, err := os.MkdirTemp("", "neograph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewGitService(tmpDir)

	repoPath, err := service.Clone(context.Background(), repoURL, "master")
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}

	// Verify clone succeeded
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		t.Error("Expected .git directory to exist")
	}

	// Verify README exists
	if _, err := os.Stat(filepath.Join(repoPath, "README.md")); os.IsNotExist(err) {
		t.Error("Expected README.md to exist")
	}
}
