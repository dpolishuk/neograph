# NeoGraph MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a self-hosted code intelligence system with Neo4j graph database, semantic search, and graph visualization for analyzing 15-20 small repositories.

**Architecture:** Go backend (Fiber) handles git cloning, Tree-sitter parsing, and Neo4j graph construction. HuggingFace TEI serves Qodo-Embed-1-1.5B embeddings. React frontend provides search UI and neovis.js graph visualization. Claude Agent SDK integration planned for Phase 2.

**Tech Stack:** Go 1.22+, Fiber v3, Neo4j 5.x, Tree-sitter, HuggingFace TEI, Qodo-Embed-1-1.5B, React 18, TypeScript, Vite, shadcn/ui, neovis.js, Docker Compose

---

## Phase 1: Infrastructure Setup

### Task 1: Project Structure

**Files:**
- Create: `backend/` directory structure
- Create: `frontend/` directory structure
- Create: `agents/` directory structure
- Create: `docker/` directory structure

**Step 1: Create monorepo structure**

```bash
cd /root/work/neograph

# Backend (Go)
mkdir -p backend/cmd/server
mkdir -p backend/internal/{config,db,git,indexer,api,models}
mkdir -p backend/pkg/treesitter

# Frontend (React)
mkdir -p frontend/src/{components,hooks,lib,pages,types}

# Agents (Python) - Phase 2
mkdir -p agents/src/{mcp,agents}

# Docker
mkdir -p docker/{neo4j,tei}

# Data volumes
mkdir -p data/{repos,neo4j}
```

**Step 2: Verify structure**

Run: `find . -type d | head -30`
Expected: All directories created

**Step 3: Create .gitignore**

Create file `.gitignore`:

```gitignore
# Go
backend/bin/
backend/vendor/

# Node
frontend/node_modules/
frontend/dist/

# Python
agents/__pycache__/
agents/.venv/
agents/*.egg-info/

# Data
data/repos/*
data/neo4j/*
!data/repos/.gitkeep
!data/neo4j/.gitkeep

# IDE
.idea/
.vscode/
*.swp

# Env
.env
.env.local

# OS
.DS_Store
```

**Step 4: Create placeholder files**

```bash
touch data/repos/.gitkeep
touch data/neo4j/.gitkeep
```

**Step 5: Commit**

```bash
git init
git add .
git commit -m "chore: initialize monorepo structure"
```

---

### Task 2: Docker Compose Base

**Files:**
- Create: `docker-compose.yml`
- Create: `docker/neo4j/Dockerfile`
- Create: `.env.example`

**Step 1: Create .env.example**

Create file `.env.example`:

```env
# Neo4j
NEO4J_AUTH=neo4j/neograph_password
NEO4J_PLUGINS=["apoc"]

# TEI (Text Embeddings Inference)
TEI_MODEL=Qodo/Qodo-Embed-1-1.5B

# Backend
BACKEND_PORT=3001
NEO4J_URI=bolt://neo4j:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=neograph_password
TEI_URL=http://tei:8080

# Frontend
VITE_API_URL=http://localhost:3001
```

**Step 2: Create docker-compose.yml**

Create file `docker-compose.yml`:

```yaml
version: '3.8'

services:
  neo4j:
    image: neo4j:5.26.0-community
    container_name: neograph-neo4j
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    environment:
      - NEO4J_AUTH=${NEO4J_AUTH}
      - NEO4J_PLUGINS=${NEO4J_PLUGINS}
      - NEO4J_dbms_security_procedures_unrestricted=apoc.*
      - NEO4J_dbms_security_procedures_allowlist=apoc.*
    volumes:
      - ./data/neo4j:/data
      - ./docker/neo4j/init:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:7474"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - neograph

  tei:
    image: ghcr.io/huggingface/text-embeddings-inference:cpu-1.5
    container_name: neograph-tei
    ports:
      - "8080:80"
    environment:
      - MODEL_ID=${TEI_MODEL}
    volumes:
      - ~/.cache/huggingface:/data
    command: --model-id ${TEI_MODEL} --port 80
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - neograph

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: neograph-backend
    ports:
      - "${BACKEND_PORT}:3001"
    environment:
      - NEO4J_URI=${NEO4J_URI}
      - NEO4J_USER=${NEO4J_USER}
      - NEO4J_PASSWORD=${NEO4J_PASSWORD}
      - TEI_URL=${TEI_URL}
    volumes:
      - ./data/repos:/app/repos
    depends_on:
      neo4j:
        condition: service_healthy
      tei:
        condition: service_healthy
    networks:
      - neograph

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: neograph-frontend
    ports:
      - "5173:5173"
    environment:
      - VITE_API_URL=${VITE_API_URL}
    depends_on:
      - backend
    networks:
      - neograph

networks:
  neograph:
    driver: bridge
```

**Step 3: Create Neo4j init script**

Create directory and file `docker/neo4j/init/01-schema.cypher`:

```cypher
// Constraints
CREATE CONSTRAINT repo_url IF NOT EXISTS FOR (r:Repository) REQUIRE r.url IS UNIQUE;
CREATE CONSTRAINT file_path IF NOT EXISTS FOR (f:File) REQUIRE (f.repoId, f.path) IS UNIQUE;

// Indexes for lookups
CREATE INDEX file_language IF NOT EXISTS FOR (f:File) ON (f.language);
CREATE INDEX function_name IF NOT EXISTS FOR (fn:Function) ON (fn.name);
CREATE INDEX class_name IF NOT EXISTS FOR (c:Class) ON (c.name);
CREATE INDEX method_name IF NOT EXISTS FOR (m:Method) ON (m.name);

// Full-text index for code search
CREATE FULLTEXT INDEX code_fulltext IF NOT EXISTS
FOR (n:Function|Class|Method) ON EACH [n.name, n.docstring, n.signature];

// Vector indexes will be created after first embeddings are generated
// CREATE VECTOR INDEX function_embeddings IF NOT EXISTS
// FOR (n:Function) ON (n.embedding)
// OPTIONS {indexConfig: {`vector.dimensions`: 1024, `vector.similarity_function`: 'cosine'}}
```

**Step 4: Commit**

```bash
git add .
git commit -m "chore: add Docker Compose configuration"
```

---

### Task 3: Go Backend Scaffold

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/Dockerfile`

**Step 1: Initialize Go module**

```bash
cd /root/work/neograph/backend
go mod init github.com/neograph/backend
```

**Step 2: Create config package**

Create file `backend/internal/config/config.go`:

```go
package config

import (
	"os"
)

type Config struct {
	Port        string
	Neo4jURI    string
	Neo4jUser   string
	Neo4jPass   string
	TEI_URL     string
	ReposPath   string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("BACKEND_PORT", "3001"),
		Neo4jURI:    getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:   getEnv("NEO4J_USER", "neo4j"),
		Neo4jPass:   getEnv("NEO4J_PASSWORD", "neograph_password"),
		TEI_URL:     getEnv("TEI_URL", "http://localhost:8080"),
		ReposPath:   getEnv("REPOS_PATH", "./repos"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
```

**Step 3: Create main entry point**

Create file `backend/cmd/server/main.go`:

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/neograph/backend/internal/config"
)

func main() {
	cfg := config.Load()

	app := fiber.New(fiber.Config{
		AppName: "NeoGraph API",
	})

	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"service": "neograph-backend",
		})
	})

	// API routes will be added here
	api := app.Group("/api")
	api.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "NeoGraph API v1",
		})
	})

	log.Printf("Starting NeoGraph backend on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
```

**Step 4: Add Go dependencies**

```bash
cd /root/work/neograph/backend
go get github.com/gofiber/fiber/v3
go get github.com/neo4j/neo4j-go-driver/v5
go mod tidy
```

**Step 5: Create Dockerfile**

Create file `backend/Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates git

# Copy binary
COPY --from=builder /build/server .

# Create repos directory
RUN mkdir -p /app/repos

EXPOSE 3001

CMD ["./server"]
```

**Step 6: Verify build locally**

```bash
cd /root/work/neograph/backend
go build -o bin/server ./cmd/server
./bin/server &
curl http://localhost:3001/health
kill %1
```

Expected: `{"status":"ok","service":"neograph-backend"}`

**Step 7: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Go backend scaffold with Fiber"
```

---

## Phase 2: Neo4j Integration

### Task 4: Neo4j Database Client

**Files:**
- Create: `backend/internal/db/neo4j.go`
- Create: `backend/internal/db/neo4j_test.go`

**Step 1: Write failing test for Neo4j connection**

Create file `backend/internal/db/neo4j_test.go`:

```go
package db

import (
	"context"
	"testing"
)

func TestNewNeo4jClient(t *testing.T) {
	// This test requires Neo4j running
	// Skip in CI without Neo4j
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := Neo4jConfig{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "neograph_password",
	}

	client, err := NewNeo4jClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Verify connection
	err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Failed to ping: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /root/work/neograph/backend
go test ./internal/db/... -v
```

Expected: FAIL - package not found or type not defined

**Step 3: Implement Neo4j client**

Create file `backend/internal/db/neo4j.go`:

```go
package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}

type Neo4jClient struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jClient(ctx context.Context, cfg Neo4jConfig) (*Neo4jClient, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	return &Neo4jClient{driver: driver}, nil
}

func (c *Neo4jClient) Close() error {
	return c.driver.Close(context.Background())
}

func (c *Neo4jClient) Ping(ctx context.Context) error {
	return c.driver.VerifyConnectivity(ctx)
}

func (c *Neo4jClient) Session(ctx context.Context) neo4j.SessionWithContext {
	return c.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
	})
}

// ExecuteWrite runs a write transaction
func (c *Neo4jClient) ExecuteWrite(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := c.Session(ctx)
	defer session.Close(ctx)

	return session.ExecuteWrite(ctx, work)
}

// ExecuteRead runs a read transaction
func (c *Neo4jClient) ExecuteRead(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	session := c.Session(ctx)
	defer session.Close(ctx)

	return session.ExecuteRead(ctx, work)
}
```

**Step 4: Run test (with Neo4j running)**

```bash
# Start Neo4j first
docker run -d --name neo4j-test -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/neograph_password \
  neo4j:5.26.0-community

# Wait for startup
sleep 30

# Run test
cd /root/work/neograph/backend
go test ./internal/db/... -v

# Cleanup
docker stop neo4j-test && docker rm neo4j-test
```

Expected: PASS

**Step 5: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Neo4j client with connection management"
```

---

### Task 5: Neo4j Models

**Files:**
- Create: `backend/internal/models/repository.go`
- Create: `backend/internal/models/file.go`
- Create: `backend/internal/models/code_entity.go`

**Step 1: Create Repository model**

Create file `backend/internal/models/repository.go`:

```go
package models

import "time"

type Repository struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`
	Name          string    `json:"name"`
	DefaultBranch string    `json:"defaultBranch"`
	LastIndexed   time.Time `json:"lastIndexed"`
	Status        string    `json:"status"` // pending, indexing, ready, error
	FilesCount    int       `json:"filesCount"`
	FunctionsCount int      `json:"functionsCount"`
}

type CreateRepositoryInput struct {
	URL    string `json:"url" validate:"required,url"`
	Branch string `json:"branch"`
}
```

**Step 2: Create File model**

Create file `backend/internal/models/file.go`:

```go
package models

type File struct {
	ID       string `json:"id"`
	RepoID   string `json:"repoId"`
	Path     string `json:"path"`
	Language string `json:"language"`
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
}

// Language detection by extension
var LanguageByExtension = map[string]string{
	".go":   "go",
	".py":   "python",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".java": "java",
	".kt":   "kotlin",
	".kts":  "kotlin",
}

func DetectLanguage(path string) string {
	for ext, lang := range LanguageByExtension {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return lang
		}
	}
	return ""
}
```

**Step 3: Create CodeEntity models**

Create file `backend/internal/models/code_entity.go`:

```go
package models

type CodeEntityType string

const (
	EntityFunction CodeEntityType = "Function"
	EntityClass    CodeEntityType = "Class"
	EntityMethod   CodeEntityType = "Method"
	EntityVariable CodeEntityType = "Variable"
)

type CodeEntity struct {
	ID            string         `json:"id"`
	Type          CodeEntityType `json:"type"`
	Name          string         `json:"name"`
	Signature     string         `json:"signature,omitempty"`
	Docstring     string         `json:"docstring,omitempty"`
	StartLine     int            `json:"startLine"`
	EndLine       int            `json:"endLine"`
	FilePath      string         `json:"filePath"`
	FileID        string         `json:"fileId"`
	RepoID        string         `json:"repoId"`

	// For embeddings
	NLDescription string    `json:"nlDescription,omitempty"`
	Embedding     []float32 `json:"embedding,omitempty"`

	// Relationships (populated on query)
	Calls   []string `json:"calls,omitempty"`
	Imports []string `json:"imports,omitempty"`
}

type CallRelation struct {
	CallerID string `json:"callerId"`
	CalleeID string `json:"calleeId"`
	Line     int    `json:"line"`
}

type ImportRelation struct {
	FileID     string `json:"fileId"`
	ImportPath string `json:"importPath"`
	Alias      string `json:"alias,omitempty"`
}
```

**Step 4: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add domain models for Repository, File, CodeEntity"
```

---

### Task 6: Neo4j Repository for Graph Operations

**Files:**
- Create: `backend/internal/db/repository.go`
- Create: `backend/internal/db/repository_test.go`

**Step 1: Write failing test**

Create file `backend/internal/db/repository_test.go`:

```go
package db

import (
	"context"
	"testing"

	"github.com/neograph/backend/internal/models"
)

func TestCreateRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := setupTestClient(t)
	defer client.Close()

	repo := &models.Repository{
		URL:           "https://github.com/test/repo",
		Name:          "repo",
		DefaultBranch: "main",
		Status:        "pending",
	}

	created, err := CreateRepository(context.Background(), client, repo)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	if created.ID == "" {
		t.Error("Expected repository ID to be set")
	}

	// Cleanup
	_, _ = client.ExecuteWrite(context.Background(), func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(context.Background(),
			"MATCH (r:Repository {url: $url}) DELETE r",
			map[string]any{"url": repo.URL})
		return nil, err
	})
}

func setupTestClient(t *testing.T) *Neo4jClient {
	cfg := Neo4jConfig{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "neograph_password",
	}
	client, err := NewNeo4jClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}
```

**Step 2: Run test to verify it fails**

```bash
cd /root/work/neograph/backend
go test ./internal/db/... -v -run TestCreateRepository
```

Expected: FAIL - CreateRepository not defined

**Step 3: Implement repository operations**

Create file `backend/internal/db/repository.go`:

```go
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neograph/backend/internal/models"
)

func CreateRepository(ctx context.Context, client *Neo4jClient, repo *models.Repository) (*models.Repository, error) {
	repo.ID = uuid.New().String()

	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CREATE (r:Repository {
				id: $id,
				url: $url,
				name: $name,
				defaultBranch: $defaultBranch,
				status: $status,
				lastIndexed: $lastIndexed,
				filesCount: 0,
				functionsCount: 0
			})
			RETURN r
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":            repo.ID,
			"url":           repo.URL,
			"name":          repo.Name,
			"defaultBranch": repo.DefaultBranch,
			"status":        repo.Status,
			"lastIndexed":   time.Now().UTC(),
		})
		return nil, err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return repo, nil
}

func GetRepository(ctx context.Context, client *Neo4jClient, id string) (*models.Repository, error) {
	result, err := client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			RETURN r.id AS id, r.url AS url, r.name AS name,
			       r.defaultBranch AS defaultBranch, r.status AS status,
			       r.lastIndexed AS lastIndexed, r.filesCount AS filesCount,
			       r.functionsCount AS functionsCount
		`
		result, err := tx.Run(ctx, query, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return recordToRepository(record), nil
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.Repository), nil
}

func ListRepositories(ctx context.Context, client *Neo4jClient) ([]*models.Repository, error) {
	result, err := client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository)
			RETURN r.id AS id, r.url AS url, r.name AS name,
			       r.defaultBranch AS defaultBranch, r.status AS status,
			       r.lastIndexed AS lastIndexed, r.filesCount AS filesCount,
			       r.functionsCount AS functionsCount
			ORDER BY r.lastIndexed DESC
		`
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		var repos []*models.Repository
		for result.Next(ctx) {
			repos = append(repos, recordToRepository(result.Record()))
		}
		return repos, result.Err()
	})

	if err != nil {
		return nil, err
	}
	return result.([]*models.Repository), nil
}

func UpdateRepositoryStatus(ctx context.Context, client *Neo4jClient, id, status string) error {
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			SET r.status = $status, r.lastIndexed = $lastIndexed
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":          id,
			"status":      status,
			"lastIndexed": time.Now().UTC(),
		})
		return nil, err
	})
	return err
}

func DeleteRepository(ctx context.Context, client *Neo4jClient, id string) error {
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Delete all related nodes first
		query := `
			MATCH (r:Repository {id: $id})
			OPTIONAL MATCH (r)-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(e)
			DETACH DELETE e, f, r
		`
		_, err := tx.Run(ctx, query, map[string]any{"id": id})
		return nil, err
	})
	return err
}

func recordToRepository(record *neo4j.Record) *models.Repository {
	repo := &models.Repository{}

	if id, ok := record.Get("id"); ok && id != nil {
		repo.ID = id.(string)
	}
	if url, ok := record.Get("url"); ok && url != nil {
		repo.URL = url.(string)
	}
	if name, ok := record.Get("name"); ok && name != nil {
		repo.Name = name.(string)
	}
	if branch, ok := record.Get("defaultBranch"); ok && branch != nil {
		repo.DefaultBranch = branch.(string)
	}
	if status, ok := record.Get("status"); ok && status != nil {
		repo.Status = status.(string)
	}
	if lastIndexed, ok := record.Get("lastIndexed"); ok && lastIndexed != nil {
		if t, ok := lastIndexed.(time.Time); ok {
			repo.LastIndexed = t
		}
	}
	if filesCount, ok := record.Get("filesCount"); ok && filesCount != nil {
		repo.FilesCount = int(filesCount.(int64))
	}
	if functionsCount, ok := record.Get("functionsCount"); ok && functionsCount != nil {
		repo.FunctionsCount = int(functionsCount.(int64))
	}

	return repo
}
```

**Step 4: Add uuid dependency**

```bash
cd /root/work/neograph/backend
go get github.com/google/uuid
```

**Step 5: Run test**

```bash
cd /root/work/neograph/backend
go test ./internal/db/... -v -run TestCreateRepository
```

Expected: PASS (with Neo4j running)

**Step 6: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Neo4j repository operations (CRUD)"
```

---

## Phase 3: Git Integration

### Task 7: Git Clone Service

**Files:**
- Create: `backend/internal/git/clone.go`
- Create: `backend/internal/git/clone_test.go`

**Step 1: Write failing test**

Create file `backend/internal/git/clone_test.go`:

```go
package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCloneRepository(t *testing.T) {
	// Use a small public repo for testing
	repoURL := "https://github.com/kelseyhightower/nocode"

	tmpDir, err := os.MkdirTemp("", "neograph-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	service := NewGitService(tmpDir)

	repoPath, err := service.Clone(context.Background(), repoURL, "main")
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

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/owner/repo", "repo"},
		{"https://github.com/owner/repo.git", "repo"},
		{"git@github.com:owner/repo.git", "repo"},
	}

	for _, tt := range tests {
		got := ExtractRepoName(tt.url)
		if got != tt.expected {
			t.Errorf("ExtractRepoName(%s) = %s, want %s", tt.url, got, tt.expected)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /root/work/neograph/backend
go test ./internal/git/... -v
```

Expected: FAIL - package not found

**Step 3: Implement Git service**

Create file `backend/internal/git/clone.go`:

```go
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
```

**Step 4: Run tests**

```bash
cd /root/work/neograph/backend
go test ./internal/git/... -v
```

Expected: PASS (requires git installed)

**Step 5: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Git clone service"
```

---

## Phase 4: Tree-sitter Indexing

### Task 8: Tree-sitter Parser Setup

**Files:**
- Create: `backend/pkg/treesitter/parser.go`
- Create: `backend/pkg/treesitter/languages.go`

**Step 1: Install Tree-sitter dependencies**

```bash
cd /root/work/neograph/backend
go get github.com/smacker/go-tree-sitter
go get github.com/smacker/go-tree-sitter/golang
go get github.com/smacker/go-tree-sitter/python
go get github.com/smacker/go-tree-sitter/typescript
go get github.com/smacker/go-tree-sitter/java
go get github.com/smacker/go-tree-sitter/kotlin
go get github.com/smacker/go-tree-sitter/javascript
```

**Step 2: Create languages registry**

Create file `backend/pkg/treesitter/languages.go`:

```go
package treesitter

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

var languages = map[string]*sitter.Language{
	"go":         golang.GetLanguage(),
	"python":     python.GetLanguage(),
	"typescript": typescript.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"java":       java.GetLanguage(),
	"kotlin":     kotlin.GetLanguage(),
}

func GetLanguage(name string) *sitter.Language {
	return languages[name]
}

func SupportedLanguages() []string {
	keys := make([]string, 0, len(languages))
	for k := range languages {
		keys = append(keys, k)
	}
	return keys
}
```

**Step 3: Create parser wrapper**

Create file `backend/pkg/treesitter/parser.go`:

```go
package treesitter

import (
	"context"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type Parser struct {
	parser *sitter.Parser
}

func NewParser() *Parser {
	return &Parser{
		parser: sitter.NewParser(),
	}
}

func (p *Parser) Parse(ctx context.Context, content []byte, language string) (*sitter.Tree, error) {
	lang := GetLanguage(language)
	if lang == nil {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	p.parser.SetLanguage(lang)

	tree, err := p.parser.ParseCtx(ctx, nil, content)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return tree, nil
}

func (p *Parser) Close() {
	p.parser.Close()
}
```

**Step 4: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Tree-sitter parser setup with language support"
```

---

### Task 9: Code Entity Extractor

**Files:**
- Create: `backend/internal/indexer/extractor.go`
- Create: `backend/internal/indexer/extractor_test.go`
- Create: `backend/internal/indexer/queries/` (query patterns)

**Step 1: Write failing test**

Create file `backend/internal/indexer/extractor_test.go`:

```go
package indexer

import (
	"context"
	"testing"
)

func TestExtractGoFunctions(t *testing.T) {
	code := []byte(`
package main

// HelloWorld prints a greeting
func HelloWorld(name string) string {
	return "Hello, " + name
}

func main() {
	println(HelloWorld("World"))
}
`)

	extractor := NewExtractor()
	defer extractor.Close()

	entities, err := extractor.Extract(context.Background(), code, "go", "test.go")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find 2 functions
	var functions int
	for _, e := range entities {
		if e.Type == "Function" {
			functions++
		}
	}

	if functions != 2 {
		t.Errorf("Expected 2 functions, got %d", functions)
	}

	// Check HelloWorld function
	var helloWorld *CodeEntity
	for _, e := range entities {
		if e.Name == "HelloWorld" {
			helloWorld = e
			break
		}
	}

	if helloWorld == nil {
		t.Fatal("HelloWorld function not found")
	}

	if helloWorld.Docstring != "HelloWorld prints a greeting" {
		t.Errorf("Unexpected docstring: %s", helloWorld.Docstring)
	}
}

func TestExtractPythonFunctions(t *testing.T) {
	code := []byte(`
def greet(name: str) -> str:
    """Greet a person by name."""
    return f"Hello, {name}"

class Greeter:
    """A class that greets people."""

    def __init__(self, prefix: str):
        self.prefix = prefix

    def greet(self, name: str) -> str:
        return f"{self.prefix}, {name}"
`)

	extractor := NewExtractor()
	defer extractor.Close()

	entities, err := extractor.Extract(context.Background(), code, "python", "test.py")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find: 1 function + 1 class + 2 methods
	var functions, classes, methods int
	for _, e := range entities {
		switch e.Type {
		case "Function":
			functions++
		case "Class":
			classes++
		case "Method":
			methods++
		}
	}

	if functions != 1 {
		t.Errorf("Expected 1 function, got %d", functions)
	}
	if classes != 1 {
		t.Errorf("Expected 1 class, got %d", classes)
	}
	if methods != 2 {
		t.Errorf("Expected 2 methods, got %d", methods)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /root/work/neograph/backend
go test ./internal/indexer/... -v
```

Expected: FAIL - package not found

**Step 3: Implement code entity extractor**

Create file `backend/internal/indexer/extractor.go`:

```go
package indexer

import (
	"context"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/neograph/backend/pkg/treesitter"
)

type CodeEntity struct {
	Type      string   // Function, Class, Method
	Name      string
	Signature string
	Docstring string
	StartLine int
	EndLine   int
	FilePath  string
	Calls     []string
	Content   string
}

type Extractor struct {
	parser *treesitter.Parser
}

func NewExtractor() *Extractor {
	return &Extractor{
		parser: treesitter.NewParser(),
	}
}

func (e *Extractor) Close() {
	e.parser.Close()
}

func (e *Extractor) Extract(ctx context.Context, content []byte, language, filePath string) ([]*CodeEntity, error) {
	tree, err := e.parser.Parse(ctx, content, language)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()

	switch language {
	case "go":
		return e.extractGo(root, content, filePath), nil
	case "python":
		return e.extractPython(root, content, filePath), nil
	case "typescript", "javascript":
		return e.extractTypeScript(root, content, filePath), nil
	case "java":
		return e.extractJava(root, content, filePath), nil
	case "kotlin":
		return e.extractKotlin(root, content, filePath), nil
	default:
		return nil, nil
	}
}

func (e *Extractor) extractGo(root *sitter.Node, content []byte, filePath string) []*CodeEntity {
	var entities []*CodeEntity

	// Traverse tree looking for function declarations
	var traverse func(node *sitter.Node)
	traverse = func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			entity := e.parseGoFunction(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "method_declaration":
			entity := e.parseGoMethod(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "type_declaration":
			// Check for struct/interface types
			for i := 0; i < int(node.NamedChildCount()); i++ {
				child := node.NamedChild(i)
				if child.Type() == "type_spec" {
					entity := e.parseGoType(child, content, filePath)
					if entity != nil {
						entities = append(entities, entity)
					}
				}
			}
		}

		// Recurse into children
		for i := 0; i < int(node.ChildCount()); i++ {
			traverse(node.Child(i))
		}
	}

	traverse(root)
	return entities
}

func (e *Extractor) parseGoFunction(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)

	// Get docstring from preceding comment
	docstring := e.getPrecedingComment(node, content)

	// Build signature
	signature := e.getNodeContent(node, content)
	// Truncate to just the signature part
	if idx := strings.Index(signature, "{"); idx > 0 {
		signature = strings.TrimSpace(signature[:idx])
	}

	// Extract function calls
	calls := e.extractCalls(node, content)

	return &CodeEntity{
		Type:      "Function",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parseGoMethod(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPrecedingComment(node, content)

	signature := e.getNodeContent(node, content)
	if idx := strings.Index(signature, "{"); idx > 0 {
		signature = strings.TrimSpace(signature[:idx])
	}

	calls := e.extractCalls(node, content)

	return &CodeEntity{
		Type:      "Method",
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Calls:     calls,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parseGoType(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPrecedingComment(node.Parent(), content)

	return &CodeEntity{
		Type:      "Class", // Using Class for Go structs/interfaces
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) extractPython(root *sitter.Node, content []byte, filePath string) []*CodeEntity {
	var entities []*CodeEntity

	var traverse func(node *sitter.Node, inClass bool)
	traverse = func(node *sitter.Node, inClass bool) {
		switch node.Type() {
		case "function_definition":
			entity := e.parsePythonFunction(node, content, filePath, inClass)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "class_definition":
			entity := e.parsePythonClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
			// Recurse into class body
			body := node.ChildByFieldName("body")
			if body != nil {
				for i := 0; i < int(body.ChildCount()); i++ {
					traverse(body.Child(i), true)
				}
			}
			return // Don't recurse normally for class
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			traverse(node.Child(i), inClass)
		}
	}

	traverse(root, false)
	return entities
}

func (e *Extractor) parsePythonFunction(node *sitter.Node, content []byte, filePath string, inClass bool) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)

	// Get docstring
	docstring := e.getPythonDocstring(node, content)

	// Build signature
	params := node.ChildByFieldName("parameters")
	returnType := node.ChildByFieldName("return_type")

	signature := "def " + name
	if params != nil {
		signature += params.Content(content)
	}
	if returnType != nil {
		signature += " -> " + returnType.Content(content)
	}

	entityType := "Function"
	if inClass {
		entityType = "Method"
	}

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Signature: signature,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parsePythonClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPythonDocstring(node, content)

	return &CodeEntity{
		Type:      "Class",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) extractTypeScript(root *sitter.Node, content []byte, filePath string) []*CodeEntity {
	var entities []*CodeEntity

	var traverse func(node *sitter.Node, inClass bool)
	traverse = func(node *sitter.Node, inClass bool) {
		switch node.Type() {
		case "function_declaration", "arrow_function", "function":
			entity := e.parseTSFunction(node, content, filePath, inClass)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "method_definition":
			entity := e.parseTSFunction(node, content, filePath, true)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "class_declaration":
			entity := e.parseTSClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
			// Continue traversing to find methods
			for i := 0; i < int(node.ChildCount()); i++ {
				traverse(node.Child(i), true)
			}
			return
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			traverse(node.Child(i), inClass)
		}
	}

	traverse(root, false)
	return entities
}

func (e *Extractor) parseTSFunction(node *sitter.Node, content []byte, filePath string, inClass bool) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	if name == "" {
		return nil
	}

	docstring := e.getPrecedingComment(node, content)

	entityType := "Function"
	if inClass {
		entityType = "Method"
	}

	return &CodeEntity{
		Type:      entityType,
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parseTSClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "Class",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) extractJava(root *sitter.Node, content []byte, filePath string) []*CodeEntity {
	var entities []*CodeEntity

	var traverse func(node *sitter.Node)
	traverse = func(node *sitter.Node) {
		switch node.Type() {
		case "method_declaration":
			entity := e.parseJavaMethod(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "class_declaration", "interface_declaration":
			entity := e.parseJavaClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			traverse(node.Child(i))
		}
	}

	traverse(root)
	return entities
}

func (e *Extractor) parseJavaMethod(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "Method",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parseJavaClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	name := nameNode.Content(content)
	docstring := e.getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "Class",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) extractKotlin(root *sitter.Node, content []byte, filePath string) []*CodeEntity {
	var entities []*CodeEntity

	var traverse func(node *sitter.Node)
	traverse = func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			entity := e.parseKotlinFunction(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "class_declaration", "object_declaration":
			entity := e.parseKotlinClass(node, content, filePath)
			if entity != nil {
				entities = append(entities, entity)
			}
		}

		for i := 0; i < int(node.ChildCount()); i++ {
			traverse(node.Child(i))
		}
	}

	traverse(root)
	return entities
}

func (e *Extractor) parseKotlinFunction(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	// Find simple_identifier child
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "simple_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	docstring := e.getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "Function",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

func (e *Extractor) parseKotlinClass(node *sitter.Node, content []byte, filePath string) *CodeEntity {
	var name string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_identifier" || child.Type() == "simple_identifier" {
			name = child.Content(content)
			break
		}
	}

	if name == "" {
		return nil
	}

	docstring := e.getPrecedingComment(node, content)

	return &CodeEntity{
		Type:      "Class",
		Name:      name,
		Docstring: docstring,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		FilePath:  filePath,
		Content:   e.getNodeContent(node, content),
	}
}

// Helper functions

func (e *Extractor) getNodeContent(node *sitter.Node, content []byte) string {
	return node.Content(content)
}

func (e *Extractor) getPrecedingComment(node *sitter.Node, content []byte) string {
	prev := node.PrevSibling()
	if prev == nil {
		return ""
	}

	if prev.Type() == "comment" {
		comment := prev.Content(content)
		// Clean up comment markers
		comment = strings.TrimPrefix(comment, "//")
		comment = strings.TrimPrefix(comment, "/*")
		comment = strings.TrimSuffix(comment, "*/")
		comment = strings.TrimPrefix(comment, "#")
		return strings.TrimSpace(comment)
	}

	return ""
}

func (e *Extractor) getPythonDocstring(node *sitter.Node, content []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil {
		return ""
	}

	// First statement in body might be docstring
	if body.ChildCount() > 0 {
		first := body.Child(0)
		if first.Type() == "expression_statement" {
			if first.ChildCount() > 0 {
				expr := first.Child(0)
				if expr.Type() == "string" {
					doc := expr.Content(content)
					// Clean up quotes
					doc = strings.Trim(doc, "\"'")
					doc = strings.TrimPrefix(doc, "\"\"")
					doc = strings.TrimSuffix(doc, "\"\"")
					return strings.TrimSpace(doc)
				}
			}
		}
	}

	return ""
}

func (e *Extractor) extractCalls(node *sitter.Node, content []byte) []string {
	var calls []string
	seen := make(map[string]bool)

	var traverse func(n *sitter.Node)
	traverse = func(n *sitter.Node) {
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				name := fn.Content(content)
				// Extract just the function name
				if idx := strings.LastIndex(name, "."); idx >= 0 {
					name = name[idx+1:]
				}
				if !seen[name] {
					seen[name] = true
					calls = append(calls, name)
				}
			}
		}

		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(i))
		}
	}

	traverse(node)
	return calls
}
```

**Step 4: Run tests**

```bash
cd /root/work/neograph/backend
go test ./internal/indexer/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Tree-sitter code entity extractor for multiple languages"
```

---

### Task 10: Indexing Pipeline

**Files:**
- Create: `backend/internal/indexer/pipeline.go`
- Create: `backend/internal/indexer/pipeline_test.go`

**Step 1: Write failing test**

Create file `backend/internal/indexer/pipeline_test.go`:

```go
package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/neograph/backend/internal/models"
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
```

**Step 2: Run test to verify it fails**

```bash
cd /root/work/neograph/backend
go test ./internal/indexer/... -v -run TestIndexRepository
```

Expected: FAIL - NewPipeline not defined

**Step 3: Implement indexing pipeline**

Create file `backend/internal/indexer/pipeline.go`:

```go
package indexer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/neograph/backend/internal/db"
	"github.com/neograph/backend/internal/models"
)

type Pipeline struct {
	dbClient  *db.Neo4jClient
	extractor *Extractor
}

type IndexResult struct {
	RepoID          string
	FilesProcessed  int
	EntitiesFound   int
	Errors          []string
	Files           []*models.File
	Entities        []*CodeEntity
}

func NewPipeline(dbClient *db.Neo4jClient) *Pipeline {
	return &Pipeline{
		dbClient:  dbClient,
		extractor: NewExtractor(),
	}
}

func (p *Pipeline) Close() {
	p.extractor.Close()
}

func (p *Pipeline) IndexDirectory(ctx context.Context, dirPath, repoID string) (*IndexResult, error) {
	result := &IndexResult{
		RepoID: repoID,
	}

	// Walk directory and find supported files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common non-code directories
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
			   name == "__pycache__" || name == ".venv" || name == "dist" ||
			   name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is supported
		relPath, _ := filepath.Rel(dirPath, path)
		lang := models.DetectLanguage(path)
		if lang != "" {
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Process files concurrently (limited concurrency)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4) // Max 4 concurrent
	var mu sync.Mutex

	for _, relPath := range files {
		wg.Add(1)
		go func(relPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullPath := filepath.Join(dirPath, relPath)
			file, entities, err := p.processFile(ctx, fullPath, relPath, repoID)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", relPath, err))
				return
			}

			result.FilesProcessed++
			result.Files = append(result.Files, file)
			result.Entities = append(result.Entities, entities...)
			result.EntitiesFound += len(entities)
		}(relPath)
	}

	wg.Wait()

	return result, nil
}

func (p *Pipeline) processFile(ctx context.Context, fullPath, relPath, repoID string) (*models.File, []*CodeEntity, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	lang := models.DetectLanguage(relPath)

	file := &models.File{
		RepoID:   repoID,
		Path:     relPath,
		Language: lang,
		Size:     info.Size(),
		Hash:     hashContent(content),
	}

	// Extract code entities
	entities, err := p.extractor.Extract(ctx, content, lang, relPath)
	if err != nil {
		return file, nil, fmt.Errorf("extraction failed: %w", err)
	}

	return file, entities, nil
}

func hashContent(content []byte) string {
	// Simple hash for change detection
	var h uint64 = 5381
	for _, b := range content {
		h = ((h << 5) + h) + uint64(b)
	}
	return fmt.Sprintf("%x", h)
}
```

**Step 4: Run tests**

```bash
cd /root/work/neograph/backend
go test ./internal/indexer/... -v
```

Expected: PASS

**Step 5: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add indexing pipeline with concurrent file processing"
```

---

### Task 11: Neo4j Graph Writer

**Files:**
- Create: `backend/internal/db/graph_writer.go`

**Step 1: Implement graph writer**

Create file `backend/internal/db/graph_writer.go`:

```go
package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neograph/backend/internal/indexer"
	"github.com/neograph/backend/internal/models"
)

type GraphWriter struct {
	client *Neo4jClient
}

func NewGraphWriter(client *Neo4jClient) *GraphWriter {
	return &GraphWriter{client: client}
}

// WriteIndexResult writes all indexed data to Neo4j
func (w *GraphWriter) WriteIndexResult(ctx context.Context, result *indexer.IndexResult) error {
	// Write files
	for _, file := range result.Files {
		if err := w.WriteFile(ctx, file); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	// Write entities
	for _, entity := range result.Entities {
		if err := w.WriteEntity(ctx, result.RepoID, entity); err != nil {
			return fmt.Errorf("failed to write entity %s: %w", entity.Name, err)
		}
	}

	// Write call relationships
	for _, entity := range result.Entities {
		if len(entity.Calls) > 0 {
			if err := w.WriteCallRelationships(ctx, entity); err != nil {
				return fmt.Errorf("failed to write calls for %s: %w", entity.Name, err)
			}
		}
	}

	// Update repository stats
	return w.UpdateRepositoryStats(ctx, result.RepoID, len(result.Files), result.EntitiesFound)
}

func (w *GraphWriter) WriteFile(ctx context.Context, file *models.File) error {
	file.ID = uuid.New().String()

	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})
			MERGE (f:File {repoId: $repoId, path: $path})
			SET f.id = $id,
			    f.language = $language,
			    f.hash = $hash,
			    f.size = $size
			MERGE (r)-[:CONTAINS]->(f)
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":       file.ID,
			"repoId":   file.RepoID,
			"path":     file.Path,
			"language": file.Language,
			"hash":     file.Hash,
			"size":     file.Size,
		})
		return nil, err
	})

	return err
}

func (w *GraphWriter) WriteEntity(ctx context.Context, repoID string, entity *indexer.CodeEntity) error {
	entityID := uuid.New().String()

	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create entity node with appropriate label
		var query string
		switch entity.Type {
		case "Function":
			query = `
				MATCH (f:File {repoId: $repoId, path: $filePath})
				CREATE (e:Function {
					id: $id,
					name: $name,
					signature: $signature,
					docstring: $docstring,
					startLine: $startLine,
					endLine: $endLine,
					filePath: $filePath,
					repoId: $repoId
				})
				CREATE (f)-[:DECLARES]->(e)
			`
		case "Class":
			query = `
				MATCH (f:File {repoId: $repoId, path: $filePath})
				CREATE (e:Class {
					id: $id,
					name: $name,
					docstring: $docstring,
					startLine: $startLine,
					endLine: $endLine,
					filePath: $filePath,
					repoId: $repoId
				})
				CREATE (f)-[:DECLARES]->(e)
			`
		case "Method":
			query = `
				MATCH (f:File {repoId: $repoId, path: $filePath})
				CREATE (e:Method {
					id: $id,
					name: $name,
					signature: $signature,
					docstring: $docstring,
					startLine: $startLine,
					endLine: $endLine,
					filePath: $filePath,
					repoId: $repoId
				})
				CREATE (f)-[:DECLARES]->(e)
			`
		default:
			return nil, nil
		}

		_, err := tx.Run(ctx, query, map[string]any{
			"id":        entityID,
			"name":      entity.Name,
			"signature": entity.Signature,
			"docstring": entity.Docstring,
			"startLine": entity.StartLine,
			"endLine":   entity.EndLine,
			"filePath":  entity.FilePath,
			"repoId":    repoID,
		})
		return nil, err
	})

	return err
}

func (w *GraphWriter) WriteCallRelationships(ctx context.Context, entity *indexer.CodeEntity) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, calledName := range entity.Calls {
			query := `
				MATCH (caller:Function|Method {name: $callerName, filePath: $filePath})
				MATCH (callee:Function|Method {name: $calleeName})
				WHERE callee.repoId = caller.repoId
				MERGE (caller)-[:CALLS]->(callee)
			`
			_, err := tx.Run(ctx, query, map[string]any{
				"callerName": entity.Name,
				"filePath":   entity.FilePath,
				"calleeName": calledName,
			})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

func (w *GraphWriter) UpdateRepositoryStats(ctx context.Context, repoID string, filesCount, entitiesCount int) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			SET r.filesCount = $filesCount,
			    r.functionsCount = $entitiesCount,
			    r.status = 'ready'
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":            repoID,
			"filesCount":    filesCount,
			"entitiesCount": entitiesCount,
		})
		return nil, err
	})

	return err
}

// ClearRepository removes all indexed data for a repository
func (w *GraphWriter) ClearRepository(ctx context.Context, repoID string) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			OPTIONAL MATCH (r)-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(e)
			DETACH DELETE e, f
		`
		_, err := tx.Run(ctx, query, map[string]any{"id": repoID})
		return nil, err
	})

	return err
}
```

**Step 2: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add Neo4j graph writer for persisting indexed data"
```

---

## Phase 5: REST API

### Task 12: API Handlers

**Files:**
- Create: `backend/internal/api/handlers.go`
- Create: `backend/internal/api/routes.go`
- Modify: `backend/cmd/server/main.go`

**Step 1: Create API handlers**

Create file `backend/internal/api/handlers.go`:

```go
package api

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/neograph/backend/internal/config"
	"github.com/neograph/backend/internal/db"
	"github.com/neograph/backend/internal/git"
	"github.com/neograph/backend/internal/indexer"
	"github.com/neograph/backend/internal/models"
)

type Handler struct {
	cfg       *config.Config
	dbClient  *db.Neo4jClient
	gitSvc    *git.GitService
	pipeline  *indexer.Pipeline
	writer    *db.GraphWriter
}

func NewHandler(cfg *config.Config, dbClient *db.Neo4jClient) *Handler {
	return &Handler{
		cfg:       cfg,
		dbClient:  dbClient,
		gitSvc:    git.NewGitService(cfg.ReposPath),
		pipeline:  indexer.NewPipeline(dbClient),
		writer:    db.NewGraphWriter(dbClient),
	}
}

func (h *Handler) Close() {
	h.pipeline.Close()
}

// ListRepositories returns all repositories
func (h *Handler) ListRepositories(c fiber.Ctx) error {
	repos, err := db.ListRepositories(c.Context(), h.dbClient)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(repos)
}

// GetRepository returns a single repository
func (h *Handler) GetRepository(c fiber.Ctx) error {
	id := c.Params("id")
	repo, err := db.GetRepository(c.Context(), h.dbClient, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if repo == nil {
		return c.Status(404).JSON(fiber.Map{"error": "repository not found"})
	}
	return c.JSON(repo)
}

// CreateRepository adds a new repository and starts indexing
func (h *Handler) CreateRepository(c fiber.Ctx) error {
	var input models.CreateRepositoryInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if input.URL == "" {
		return c.Status(400).JSON(fiber.Map{"error": "url is required"})
	}

	// Create repository record
	repo := &models.Repository{
		URL:           input.URL,
		Name:          git.ExtractRepoName(input.URL),
		DefaultBranch: input.Branch,
		Status:        "pending",
	}

	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	created, err := db.CreateRepository(c.Context(), h.dbClient, repo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Start indexing in background
	go h.indexRepository(created)

	return c.Status(201).JSON(created)
}

// DeleteRepository removes a repository
func (h *Handler) DeleteRepository(c fiber.Ctx) error {
	id := c.Params("id")

	if err := db.DeleteRepository(c.Context(), h.dbClient, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(204)
}

// ReindexRepository triggers re-indexing
func (h *Handler) ReindexRepository(c fiber.Ctx) error {
	id := c.Params("id")

	repo, err := db.GetRepository(c.Context(), h.dbClient, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if repo == nil {
		return c.Status(404).JSON(fiber.Map{"error": "repository not found"})
	}

	// Update status and reindex
	db.UpdateRepositoryStatus(c.Context(), h.dbClient, id, "indexing")
	go h.indexRepository(repo)

	return c.JSON(fiber.Map{"status": "indexing started"})
}

func (h *Handler) indexRepository(repo *models.Repository) {
	ctx := context.Background()

	// Clone or update repository
	repoPath, err := h.gitSvc.Clone(ctx, repo.URL, repo.DefaultBranch)
	if err != nil {
		db.UpdateRepositoryStatus(ctx, h.dbClient, repo.ID, "error")
		return
	}

	// Clear existing data
	h.writer.ClearRepository(ctx, repo.ID)

	// Update status
	db.UpdateRepositoryStatus(ctx, h.dbClient, repo.ID, "indexing")

	// Run indexing pipeline
	result, err := h.pipeline.IndexDirectory(ctx, repoPath, repo.ID)
	if err != nil {
		db.UpdateRepositoryStatus(ctx, h.dbClient, repo.ID, "error")
		return
	}

	// Write to Neo4j
	if err := h.writer.WriteIndexResult(ctx, result); err != nil {
		db.UpdateRepositoryStatus(ctx, h.dbClient, repo.ID, "error")
		return
	}

	// Status will be updated to 'ready' by WriteIndexResult
}
```

**Step 2: Create routes**

Create file `backend/internal/api/routes.go`:

```go
package api

import (
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(app *fiber.App, h *Handler) {
	api := app.Group("/api")

	// Repositories
	repos := api.Group("/repos")
	repos.Get("/", h.ListRepositories)
	repos.Post("/", h.CreateRepository)
	repos.Get("/:id", h.GetRepository)
	repos.Delete("/:id", h.DeleteRepository)
	repos.Post("/:id/reindex", h.ReindexRepository)

	// Search (to be implemented)
	// api.Post("/search", h.Search)

	// Graph queries (to be implemented)
	// api.Post("/graph/query", h.GraphQuery)
}
```

**Step 3: Update main.go**

Update file `backend/cmd/server/main.go`:

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/neograph/backend/internal/api"
	"github.com/neograph/backend/internal/config"
	"github.com/neograph/backend/internal/db"
)

func main() {
	cfg := config.Load()

	// Connect to Neo4j
	dbClient, err := db.NewNeo4jClient(context.Background(), db.Neo4jConfig{
		URI:      cfg.Neo4jURI,
		Username: cfg.Neo4jUser,
		Password: cfg.Neo4jPass,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer dbClient.Close()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "NeoGraph API",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	// Health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "neograph-backend",
		})
	})

	// Setup API routes
	handler := api.NewHandler(cfg, dbClient)
	defer handler.Close()
	api.SetupRoutes(app, handler)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		app.Shutdown()
	}()

	log.Printf("Starting NeoGraph backend on port %s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
```

**Step 4: Update dependencies**

```bash
cd /root/work/neograph/backend
go mod tidy
```

**Step 5: Test build**

```bash
cd /root/work/neograph/backend
go build -o bin/server ./cmd/server
```

Expected: Build succeeds

**Step 6: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add REST API with repository management endpoints"
```

---

## Phase 6: Frontend (React)

### Task 13: Frontend Setup

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/index.html`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`

**Step 1: Initialize Vite project**

```bash
cd /root/work/neograph/frontend
npm create vite@latest . -- --template react-ts
```

**Step 2: Install dependencies**

```bash
cd /root/work/neograph/frontend
npm install
npm install @tanstack/react-query axios
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

**Step 3: Install shadcn/ui**

```bash
cd /root/work/neograph/frontend
npx shadcn@latest init
```

Select: TypeScript, Default style, CSS variables

**Step 4: Add shadcn components**

```bash
npx shadcn@latest add button card input table badge
```

**Step 5: Create API client**

Create file `frontend/src/lib/api.ts`:

```typescript
import axios from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

export const api = axios.create({
  baseURL: `${API_URL}/api`,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface Repository {
  id: string;
  url: string;
  name: string;
  defaultBranch: string;
  lastIndexed: string;
  status: 'pending' | 'indexing' | 'ready' | 'error';
  filesCount: number;
  functionsCount: number;
}

export interface CreateRepositoryInput {
  url: string;
  branch?: string;
}

export const repositoryApi = {
  list: () => api.get<Repository[]>('/repos').then(r => r.data),
  get: (id: string) => api.get<Repository>(`/repos/${id}`).then(r => r.data),
  create: (input: CreateRepositoryInput) => api.post<Repository>('/repos', input).then(r => r.data),
  delete: (id: string) => api.delete(`/repos/${id}`),
  reindex: (id: string) => api.post(`/repos/${id}/reindex`),
};
```

**Step 6: Create repository list component**

Create file `frontend/src/components/RepositoryList.tsx`:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { repositoryApi, Repository } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Trash2, RefreshCw } from 'lucide-react';

export function RepositoryList() {
  const queryClient = useQueryClient();

  const { data: repos, isLoading } = useQuery({
    queryKey: ['repositories'],
    queryFn: repositoryApi.list,
    refetchInterval: 5000, // Poll for status updates
  });

  const deleteMutation = useMutation({
    mutationFn: repositoryApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['repositories'] }),
  });

  const reindexMutation = useMutation({
    mutationFn: repositoryApi.reindex,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['repositories'] }),
  });

  if (isLoading) {
    return <div className="text-center py-4">Loading repositories...</div>;
  }

  if (!repos || repos.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No repositories yet. Add one to get started.
      </div>
    );
  }

  const getStatusBadge = (status: Repository['status']) => {
    const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
      ready: 'default',
      indexing: 'secondary',
      pending: 'outline',
      error: 'destructive',
    };
    return <Badge variant={variants[status]}>{status}</Badge>;
  };

  return (
    <div className="space-y-4">
      {repos.map((repo) => (
        <Card key={repo.id}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-lg font-medium">{repo.name}</CardTitle>
            {getStatusBadge(repo.status)}
          </CardHeader>
          <CardContent>
            <div className="text-sm text-muted-foreground mb-2">
              {repo.url}
            </div>
            <div className="flex items-center gap-4 text-sm">
              <span>{repo.filesCount} files</span>
              <span>{repo.functionsCount} functions</span>
              <span>Branch: {repo.defaultBranch}</span>
            </div>
            <div className="flex gap-2 mt-4">
              <Button
                variant="outline"
                size="sm"
                onClick={() => reindexMutation.mutate(repo.id)}
                disabled={repo.status === 'indexing'}
              >
                <RefreshCw className="h-4 w-4 mr-1" />
                Reindex
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => deleteMutation.mutate(repo.id)}
              >
                <Trash2 className="h-4 w-4 mr-1" />
                Delete
              </Button>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
```

**Step 7: Create add repository form**

Create file `frontend/src/components/AddRepositoryForm.tsx`:

```typescript
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { repositoryApi } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Plus } from 'lucide-react';

export function AddRepositoryForm() {
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('main');
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: repositoryApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] });
      setUrl('');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (url.trim()) {
      mutation.mutate({ url: url.trim(), branch: branch.trim() || undefined });
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Add Repository</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="flex gap-2">
          <Input
            placeholder="https://github.com/owner/repo"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="flex-1"
          />
          <Input
            placeholder="Branch (default: main)"
            value={branch}
            onChange={(e) => setBranch(e.target.value)}
            className="w-32"
          />
          <Button type="submit" disabled={mutation.isPending}>
            <Plus className="h-4 w-4 mr-1" />
            Add
          </Button>
        </form>
        {mutation.isError && (
          <div className="text-red-500 text-sm mt-2">
            Failed to add repository. Please check the URL.
          </div>
        )}
      </CardContent>
    </Card>
  );
}
```

**Step 8: Update App.tsx**

Update file `frontend/src/App.tsx`:

```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RepositoryList } from '@/components/RepositoryList';
import { AddRepositoryForm } from '@/components/AddRepositoryForm';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-background">
        <header className="border-b">
          <div className="container mx-auto px-4 py-4">
            <h1 className="text-2xl font-bold">NeoGraph</h1>
            <p className="text-muted-foreground">Code Intelligence with Neo4j</p>
          </div>
        </header>

        <main className="container mx-auto px-4 py-8">
          <div className="max-w-3xl mx-auto space-y-6">
            <AddRepositoryForm />
            <RepositoryList />
          </div>
        </main>
      </div>
    </QueryClientProvider>
  );
}

export default App;
```

**Step 9: Create Dockerfile**

Create file `frontend/Dockerfile`:

```dockerfile
FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
```

**Step 10: Commit**

```bash
cd /root/work/neograph
git add .
git commit -m "feat: add React frontend with repository management UI"
```

---

## Phase 7: Integration Testing

### Task 14: End-to-End Test

**Step 1: Copy .env.example to .env**

```bash
cd /root/work/neograph
cp .env.example .env
```

**Step 2: Start all services**

```bash
docker-compose up -d neo4j
# Wait for Neo4j to be ready
sleep 60
docker-compose up -d
```

**Step 3: Verify services**

```bash
# Check Neo4j
curl http://localhost:7474

# Check Backend
curl http://localhost:3001/health

# Check Frontend
curl http://localhost:5173
```

**Step 4: Test API**

```bash
# List repos (empty)
curl http://localhost:3001/api/repos

# Add a test repo
curl -X POST http://localhost:3001/api/repos \
  -H "Content-Type: application/json" \
  -d '{"url": "https://github.com/kelseyhightower/nocode"}'

# List repos again
curl http://localhost:3001/api/repos
```

**Step 5: Verify graph in Neo4j**

Open http://localhost:7474 and run:

```cypher
MATCH (n) RETURN n LIMIT 25
```

**Step 6: Commit docker-compose verification**

```bash
cd /root/work/neograph
git add .
git commit -m "chore: verify end-to-end integration"
```

---

## Summary

This plan covers the MVP implementation with:

1. **Infrastructure**: Docker Compose with Neo4j, TEI, Backend, Frontend
2. **Backend (Go)**: Fiber API, Neo4j client, Git cloning, Tree-sitter indexing
3. **Frontend (React)**: Repository management, status display
4. **Testing**: Unit tests for key components, integration test steps

**Not included in MVP** (Phase 2):
- Embeddings generation and vector search
- Claude Agent SDK integration
- NL description generation
- neovis.js graph visualization
- Advanced search features

---

**Plan complete and saved to `docs/plans/2025-01-27-neograph-mvp.md`. Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
