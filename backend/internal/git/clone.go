package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitService struct {
	basePath string
}

func NewGitService(basePath string) *GitService {
	return &GitService{basePath: basePath}
}

// Clone clones a repository to the base path
func (s *GitService) Clone(ctx context.Context, url, branch string) (string, error) {
	repoName := ExtractRepoName(url)
	repoPath := filepath.Join(s.basePath, repoName)

	// Check if already cloned
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Already exists, do a pull instead
		return repoPath, s.Pull(ctx, repoPath)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Clone with depth 1 for faster clone
	args := []string{"clone", "--depth", "1"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, repoPath)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}

	return repoPath, nil
}

// Pull pulls latest changes
func (s *GitService) Pull(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "pull", "--ff-only")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}

// GetCurrentCommit returns the current commit hash
func (s *GitService) GetCurrentCommit(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// ListFiles returns all files in the repository
func (s *GitService) ListFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// ExtractRepoName extracts repository name from URL
func ExtractRepoName(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle HTTPS URLs
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.Split(url, "/")
		return parts[len(parts)-1]
	}

	// Handle SSH URLs (git@github.com:owner/repo)
	if strings.Contains(url, ":") {
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			pathParts := strings.Split(parts[1], "/")
			return pathParts[len(pathParts)-1]
		}
	}

	return url
}

// GetRepoPath returns the full path for a repository
func (s *GitService) GetRepoPath(repoName string) string {
	return filepath.Join(s.basePath, repoName)
}
