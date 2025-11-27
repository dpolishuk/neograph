package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexRepository(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "neograph-index-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test Go file
	goFile := filepath.Join(tmpDir, "main.go")
	goContent := []byte(`package main

func Hello() string {
	return "Hello"
}

func main() {
	println(Hello())
}
`)
	if err := os.WriteFile(goFile, goContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create test Python file
	pyFile := filepath.Join(tmpDir, "utils.py")
	pyContent := []byte(`def greet(name):
    """Greet someone."""
    return f"Hello, {name}"
`)
	if err := os.WriteFile(pyFile, pyContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	pipeline := NewPipeline(nil) // nil db client for unit test
	defer pipeline.Close()

	result, err := pipeline.IndexDirectory(context.Background(), tmpDir, "test-repo")
	if err != nil {
		t.Fatalf("IndexDirectory failed: %v", err)
	}

	if result.FilesProcessed != 2 {
		t.Errorf("Expected 2 files, got %d", result.FilesProcessed)
	}

	if result.EntitiesFound < 3 {
		t.Errorf("Expected at least 3 entities, got %d", result.EntitiesFound)
	}
}

func TestSkipIgnoredDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "neograph-skip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file in node_modules (should be skipped)
	nodeModules := filepath.Join(tmpDir, "node_modules")
	os.MkdirAll(nodeModules, 0755)
	os.WriteFile(filepath.Join(nodeModules, "test.js"), []byte("function x(){}"), 0644)

	// Create a file in root (should be processed)
	os.WriteFile(filepath.Join(tmpDir, "app.js"), []byte("function main(){}"), 0644)

	pipeline := NewPipeline(nil)
	defer pipeline.Close()

	result, _ := pipeline.IndexDirectory(context.Background(), tmpDir, "test-repo")

	if result.FilesProcessed != 1 {
		t.Errorf("Expected 1 file (node_modules should be skipped), got %d", result.FilesProcessed)
	}
}
