# DeepWiki Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add DeepWiki-style auto-generated documentation wiki with sidebar navigation, markdown content, Mermaid diagrams, and AI chat integration.

**Architecture:** Pre-generate wiki pages during repository indexing using Claude doc_writer agent. Store WikiPage nodes in Neo4j. Frontend renders markdown with react-markdown and Mermaid diagrams. Navigation via hierarchical sidebar.

**Tech Stack:** Go/Fiber backend, Python/FastAPI agents with Claude API, React/TypeScript frontend, Neo4j graph database, react-markdown, mermaid.js

---

## Phase 1: Backend Wiki Infrastructure

### Task 1: Create WikiPage Model Types

**Files:**
- Create: `/root/work/neograph/backend/internal/models/wiki.go`

**Step 1: Create wiki model file**

```go
package models

import "time"

// WikiPage represents a generated documentation page
type WikiPage struct {
	ID          string     `json:"id"`
	RepoID      string     `json:"repoId"`
	Slug        string     `json:"slug"`        // URL-friendly identifier
	Title       string     `json:"title"`
	Content     string     `json:"content"`     // Markdown content
	Order       int        `json:"order"`       // Navigation order
	ParentSlug  string     `json:"parentSlug"`  // For nested navigation (empty = root)
	Diagrams    []Diagram  `json:"diagrams"`
	GeneratedAt time.Time  `json:"generatedAt"`
}

// Diagram represents a Mermaid diagram
type Diagram struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"` // Mermaid syntax
}

// WikiNavItem represents a navigation tree item
type WikiNavItem struct {
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	Order    int           `json:"order"`
	Children []WikiNavItem `json:"children,omitempty"`
}

// WikiNavigation is the full navigation tree
type WikiNavigation struct {
	Items []WikiNavItem `json:"items"`
}

// TOCItem represents a table of contents entry
type TOCItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Level int    `json:"level"` // h1=1, h2=2, etc.
}

// WikiPageResponse is the API response for a wiki page
type WikiPageResponse struct {
	WikiPage
	TableOfContents []TOCItem `json:"tableOfContents"`
}

// WikiStatus represents generation progress
type WikiStatus struct {
	Status       string `json:"status"` // pending, generating, ready, error
	Progress     int    `json:"progress"` // 0-100
	CurrentPage  string `json:"currentPage,omitempty"`
	TotalPages   int    `json:"totalPages"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
```

**Step 2: Verify file compiles**

Run: `cd /root/work/neograph/backend && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/models/wiki.go
git commit -m "feat(wiki): add WikiPage model types"
```

---

### Task 2: Add WikiPage Neo4j Queries

**Files:**
- Create: `/root/work/neograph/backend/internal/db/wiki_reader.go`

**Step 1: Create wiki reader with GetNavigation method**

```go
package db

import (
	"context"
	"sort"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type WikiReader struct {
	client *Neo4jClient
}

func NewWikiReader(client *Neo4jClient) *WikiReader {
	return &WikiReader{client: client}
}

// GetNavigation returns the wiki navigation tree for a repository
func (r *WikiReader) GetNavigation(ctx context.Context, repoID string) (*models.WikiNavigation, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:HAS_WIKI]->(w:WikiPage)
			RETURN w.slug as slug, w.title as title, w.order as order,
			       w.parentSlug as parentSlug
			ORDER BY w.order
		`
		records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		if err != nil {
			return nil, err
		}

		// Build flat list first
		type pageInfo struct {
			Slug       string
			Title      string
			Order      int
			ParentSlug string
		}
		var pages []pageInfo

		for records.Next(ctx) {
			rec := records.Record()
			slug, _ := rec.Get("slug")
			title, _ := rec.Get("title")
			order, _ := rec.Get("order")
			parentSlug, _ := rec.Get("parentSlug")

			p := pageInfo{
				Slug:  slug.(string),
				Title: title.(string),
				Order: int(order.(int64)),
			}
			if parentSlug != nil {
				p.ParentSlug = parentSlug.(string)
			}
			pages = append(pages, p)
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		// Build tree structure
		return buildNavTree(pages), nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return &models.WikiNavigation{Items: []models.WikiNavItem{}}, nil
	}
	return result.(*models.WikiNavigation), nil
}

func buildNavTree(pages []pageInfo) *models.WikiNavigation {
	// Group by parent
	childrenMap := make(map[string][]models.WikiNavItem)

	for _, p := range pages {
		item := models.WikiNavItem{
			Slug:  p.Slug,
			Title: p.Title,
			Order: p.Order,
		}
		childrenMap[p.ParentSlug] = append(childrenMap[p.ParentSlug], item)
	}

	// Sort children by order
	for key := range childrenMap {
		sort.Slice(childrenMap[key], func(i, j int) bool {
			return childrenMap[key][i].Order < childrenMap[key][j].Order
		})
	}

	// Build tree recursively
	var buildChildren func(parentSlug string) []models.WikiNavItem
	buildChildren = func(parentSlug string) []models.WikiNavItem {
		children := childrenMap[parentSlug]
		for i := range children {
			children[i].Children = buildChildren(children[i].Slug)
		}
		return children
	}

	return &models.WikiNavigation{
		Items: buildChildren(""), // Root items have empty parent
	}
}

// GetPage returns a specific wiki page by slug
func (r *WikiReader) GetPage(ctx context.Context, repoID, slug string) (*models.WikiPageResponse, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:HAS_WIKI]->(w:WikiPage {slug: $slug})
			RETURN w.id as id, w.repoId as repoId, w.slug as slug, w.title as title,
			       w.content as content, w.order as order, w.parentSlug as parentSlug,
			       w.diagrams as diagrams, w.generatedAt as generatedAt
		`
		records, err := tx.Run(ctx, query, map[string]any{
			"repoId": repoID,
			"slug":   slug,
		})
		if err != nil {
			return nil, err
		}

		if !records.Next(ctx) {
			return nil, nil
		}

		rec := records.Record()

		id, _ := rec.Get("id")
		repoId, _ := rec.Get("repoId")
		slugVal, _ := rec.Get("slug")
		title, _ := rec.Get("title")
		content, _ := rec.Get("content")
		order, _ := rec.Get("order")
		parentSlug, _ := rec.Get("parentSlug")
		generatedAt, _ := rec.Get("generatedAt")

		page := &models.WikiPageResponse{
			WikiPage: models.WikiPage{
				ID:      id.(string),
				RepoID:  repoId.(string),
				Slug:    slugVal.(string),
				Title:   title.(string),
				Content: content.(string),
				Order:   int(order.(int64)),
			},
		}

		if parentSlug != nil {
			page.ParentSlug = parentSlug.(string)
		}

		if generatedAt != nil {
			page.GeneratedAt = generatedAt.(neo4j.Time).Time()
		}

		// Parse diagrams from JSON string if stored that way
		diagramsRaw, _ := rec.Get("diagrams")
		if diagramsRaw != nil {
			if diagrams, ok := diagramsRaw.([]any); ok {
				for _, d := range diagrams {
					if dm, ok := d.(map[string]any); ok {
						page.Diagrams = append(page.Diagrams, models.Diagram{
							ID:    dm["id"].(string),
							Title: dm["title"].(string),
							Code:  dm["code"].(string),
						})
					}
				}
			}
		}

		// Generate TOC from content
		page.TableOfContents = extractTOC(page.Content)

		if err := records.Err(); err != nil {
			return nil, err
		}

		return page, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.WikiPageResponse), nil
}

// extractTOC parses markdown headings to build table of contents
func extractTOC(content string) []models.TOCItem {
	var toc []models.TOCItem
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}

		level := 0
		for _, ch := range line {
			if ch == '#' {
				level++
			} else {
				break
			}
		}

		if level > 0 && level <= 6 {
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if title != "" {
				// Create URL-friendly ID
				id := strings.ToLower(title)
				id = strings.ReplaceAll(id, " ", "-")
				id = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(id, "")

				toc = append(toc, models.TOCItem{
					ID:    id,
					Title: title,
					Level: level,
				})
			}
		}
	}

	return toc
}
```

**Step 2: Add missing imports at top of file**

Add to imports:
```go
import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)
```

**Step 3: Verify compilation**

Run: `cd /root/work/neograph/backend && go build ./...`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/db/wiki_reader.go
git commit -m "feat(wiki): add WikiReader for Neo4j queries"
```

---

### Task 3: Add WikiWriter for Saving Pages

**Files:**
- Create: `/root/work/neograph/backend/internal/db/wiki_writer.go`

**Step 1: Create wiki writer**

```go
package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type WikiWriter struct {
	client *Neo4jClient
}

func NewWikiWriter(client *Neo4jClient) *WikiWriter {
	return &WikiWriter{client: client}
}

// WritePage saves or updates a wiki page
func (w *WikiWriter) WritePage(ctx context.Context, page *models.WikiPage) error {
	if page.ID == "" {
		page.ID = uuid.New().String()
	}
	page.GeneratedAt = time.Now()

	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Serialize diagrams to JSON
		diagramsJSON, _ := json.Marshal(page.Diagrams)

		query := `
			MATCH (r:Repository {id: $repoId})
			MERGE (w:WikiPage {repoId: $repoId, slug: $slug})
			SET w.id = $id,
			    w.title = $title,
			    w.content = $content,
			    w.order = $order,
			    w.parentSlug = $parentSlug,
			    w.diagrams = $diagrams,
			    w.generatedAt = datetime()
			MERGE (r)-[:HAS_WIKI]->(w)
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":         page.ID,
			"repoId":     page.RepoID,
			"slug":       page.Slug,
			"title":      page.Title,
			"content":    page.Content,
			"order":      page.Order,
			"parentSlug": page.ParentSlug,
			"diagrams":   string(diagramsJSON),
		})
		return nil, err
	})

	return err
}

// ClearWiki removes all wiki pages for a repository
func (w *WikiWriter) ClearWiki(ctx context.Context, repoID string) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:HAS_WIKI]->(w:WikiPage)
			DETACH DELETE w
		`
		_, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		return nil, err
	})

	return err
}

// UpdateWikiStatus updates the wiki generation status on a repository
func (w *WikiWriter) UpdateWikiStatus(ctx context.Context, repoID string, status *models.WikiStatus) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})
			SET r.wikiStatus = $status,
			    r.wikiProgress = $progress,
			    r.wikiCurrentPage = $currentPage,
			    r.wikiTotalPages = $totalPages,
			    r.wikiError = $errorMessage
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"repoId":       repoID,
			"status":       status.Status,
			"progress":     status.Progress,
			"currentPage":  status.CurrentPage,
			"totalPages":   status.TotalPages,
			"errorMessage": status.ErrorMessage,
		})
		return nil, err
	})

	return err
}

// GetWikiStatus retrieves the wiki generation status for a repository
func (w *WikiWriter) GetWikiStatus(ctx context.Context, repoID string) (*models.WikiStatus, error) {
	result, err := w.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})
			RETURN r.wikiStatus as status, r.wikiProgress as progress,
			       r.wikiCurrentPage as currentPage, r.wikiTotalPages as totalPages,
			       r.wikiError as errorMessage
		`
		records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		if err != nil {
			return nil, err
		}

		if !records.Next(ctx) {
			return nil, nil
		}

		rec := records.Record()
		status := &models.WikiStatus{}

		if s, _ := rec.Get("status"); s != nil {
			status.Status = s.(string)
		} else {
			status.Status = "none"
		}

		if p, _ := rec.Get("progress"); p != nil {
			status.Progress = int(p.(int64))
		}

		if cp, _ := rec.Get("currentPage"); cp != nil {
			status.CurrentPage = cp.(string)
		}

		if tp, _ := rec.Get("totalPages"); tp != nil {
			status.TotalPages = int(tp.(int64))
		}

		if em, _ := rec.Get("errorMessage"); em != nil {
			status.ErrorMessage = em.(string)
		}

		return status, records.Err()
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return &models.WikiStatus{Status: "none"}, nil
	}
	return result.(*models.WikiStatus), nil
}
```

**Step 2: Verify compilation**

Run: `cd /root/work/neograph/backend && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/db/wiki_writer.go
git commit -m "feat(wiki): add WikiWriter for saving pages"
```

---

### Task 4: Add Wiki API Handlers

**Files:**
- Modify: `/root/work/neograph/backend/internal/api/handlers.go`

**Step 1: Add wikiReader and wikiWriter to Handler struct**

In `handlers.go`, modify the Handler struct (around line 16-25):

```go
type Handler struct {
	cfg         *config.Config
	dbClient    *db.Neo4jClient
	gitSvc      *git.GitService
	pipeline    *indexer.Pipeline
	writer      *db.GraphWriter
	graphReader *db.GraphReader
	wikiReader  *db.WikiReader   // ADD THIS
	wikiWriter  *db.WikiWriter   // ADD THIS
	teiClient   *embedding.TEIClient
	agentProxy  *agent.AgentProxy
}
```

**Step 2: Initialize wiki reader/writer in NewHandler**

Modify NewHandler function (around line 27-38):

```go
func NewHandler(cfg *config.Config, dbClient *db.Neo4jClient) *Handler {
	return &Handler{
		cfg:         cfg,
		dbClient:    dbClient,
		gitSvc:      git.NewGitService(cfg.ReposPath),
		pipeline:    indexer.NewPipeline(dbClient),
		writer:      db.NewGraphWriter(dbClient),
		graphReader: db.NewGraphReader(dbClient),
		wikiReader:  db.NewWikiReader(dbClient),   // ADD THIS
		wikiWriter:  db.NewWikiWriter(dbClient),   // ADD THIS
		teiClient:   embedding.NewTEIClient(cfg.TEI_URL),
		agentProxy:  agent.NewAgentProxy(cfg.AgentURL),
	}
}
```

**Step 3: Add wiki handler methods at end of file**

```go
// GetWikiNavigation returns the wiki navigation tree
func (h *Handler) GetWikiNavigation(c fiber.Ctx) error {
	id := c.Params("id")
	nav, err := h.wikiReader.GetNavigation(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(nav)
}

// GetWikiPage returns a specific wiki page
func (h *Handler) GetWikiPage(c fiber.Ctx) error {
	repoID := c.Params("id")
	slug := c.Params("slug")

	if slug == "" {
		slug = "overview" // Default page
	}

	page, err := h.wikiReader.GetPage(c.Context(), repoID, slug)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if page == nil {
		return c.Status(404).JSON(fiber.Map{"error": "wiki page not found"})
	}
	return c.JSON(page)
}

// GenerateWiki triggers wiki generation for a repository
func (h *Handler) GenerateWiki(c fiber.Ctx) error {
	repoID := c.Params("id")

	// Verify repository exists
	repo, err := db.GetRepository(c.Context(), h.dbClient, repoID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if repo == nil {
		return c.Status(404).JSON(fiber.Map{"error": "repository not found"})
	}

	// Update status to generating
	status := &models.WikiStatus{
		Status:     "generating",
		Progress:   0,
		TotalPages: 5, // Estimate
	}
	h.wikiWriter.UpdateWikiStatus(c.Context(), repoID, status)

	// Start generation in background
	go h.generateWikiPages(repo)

	return c.JSON(fiber.Map{"status": "generation started"})
}

// GetWikiStatus returns the current wiki generation status
func (h *Handler) GetWikiStatus(c fiber.Ctx) error {
	repoID := c.Params("id")
	status, err := h.wikiWriter.GetWikiStatus(c.Context(), repoID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(status)
}

// generateWikiPages generates all wiki pages for a repository (stub for now)
func (h *Handler) generateWikiPages(repo *models.Repository) {
	ctx := context.Background()

	// Clear existing wiki
	h.wikiWriter.ClearWiki(ctx, repo.ID)

	// Create placeholder overview page
	overviewPage := &models.WikiPage{
		RepoID:     repo.ID,
		Slug:       "overview",
		Title:      "Overview",
		Order:      1,
		ParentSlug: "",
		Content:    fmt.Sprintf("# %s\n\nDocumentation for %s.\n\n*Wiki generation coming soon...*", repo.Name, repo.Name),
		Diagrams:   []models.Diagram{},
	}
	h.wikiWriter.WritePage(ctx, overviewPage)

	// Update status to ready
	status := &models.WikiStatus{
		Status:     "ready",
		Progress:   100,
		TotalPages: 1,
	}
	h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, status)
}
```

**Step 4: Add missing import for fmt**

At top of handlers.go, ensure `fmt` is imported:
```go
import (
	"context"
	"fmt"  // ADD THIS if not present
	...
)
```

**Step 5: Verify compilation**

Run: `cd /root/work/neograph/backend && go build ./...`
Expected: No errors

**Step 6: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat(wiki): add wiki API handlers"
```

---

### Task 5: Add Wiki Routes

**Files:**
- Modify: `/root/work/neograph/backend/internal/api/routes.go`

**Step 1: Add wiki routes in SetupRoutes function**

In `routes.go`, add after line 27 (after the existing repo routes):

```go
	// Wiki endpoints
	repos.Get("/:id/wiki", h.GetWikiNavigation)
	repos.Get("/:id/wiki/status", h.GetWikiStatus)
	repos.Post("/:id/wiki/generate", h.GenerateWiki)
	repos.Get("/:id/wiki/:slug", h.GetWikiPage)
```

**Step 2: Verify compilation**

Run: `cd /root/work/neograph/backend && go build ./...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/api/routes.go
git commit -m "feat(wiki): add wiki API routes"
```

---

### Task 6: Build and Test Backend

**Step 1: Build backend**

Run: `cd /root/work/neograph/backend && go build -o bin/server ./cmd/server`
Expected: No errors

**Step 2: Start backend server**

Run: `cd /root/work/neograph && source .env && cd backend && ./bin/server &`
Expected: Server starts on port 3001

**Step 3: Test wiki endpoints**

Run: `curl -s http://localhost:3001/api/repositories/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki | jq`
Expected: Returns `{"items":[]}` (empty wiki)

Run: `curl -X POST http://localhost:3001/api/repositories/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki/generate`
Expected: Returns `{"status":"generation started"}`

Run: `curl -s http://localhost:3001/api/repositories/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki/overview | jq`
Expected: Returns the overview page

**Step 4: Commit**

```bash
git add -A
git commit -m "feat(wiki): Phase 1 complete - backend wiki infrastructure"
```

---

## Phase 2: Frontend Wiki UI

### Task 7: Add Frontend Dependencies

**Files:**
- Modify: `/root/work/neograph/frontend/package.json`

**Step 1: Install markdown and mermaid packages**

Run:
```bash
cd /root/work/neograph/frontend && npm install react-markdown remark-gfm rehype-highlight mermaid
```

**Step 2: Verify installation**

Run: `cd /root/work/neograph/frontend && npm ls react-markdown mermaid`
Expected: Shows installed versions

**Step 3: Commit**

```bash
git add package.json package-lock.json
git commit -m "feat(wiki): add react-markdown and mermaid dependencies"
```

---

### Task 8: Add Wiki API Types and Methods

**Files:**
- Modify: `/root/work/neograph/frontend/src/lib/api.ts`

**Step 1: Add wiki types after existing interfaces (around line 100)**

```typescript
// Wiki types
export interface WikiNavItem {
  slug: string
  title: string
  order: number
  children?: WikiNavItem[]
}

export interface WikiNavigation {
  items: WikiNavItem[]
}

export interface Diagram {
  id: string
  title: string
  code: string
}

export interface TOCItem {
  id: string
  title: string
  level: number
}

export interface WikiPage {
  id: string
  repoId: string
  slug: string
  title: string
  content: string
  order: number
  parentSlug: string
  diagrams: Diagram[]
  tableOfContents: TOCItem[]
  generatedAt: string
}

export interface WikiStatus {
  status: 'none' | 'generating' | 'ready' | 'error'
  progress: number
  currentPage?: string
  totalPages: number
  errorMessage?: string
}
```

**Step 2: Add wiki API methods after repositoryApi object**

```typescript
export const wikiApi = {
  getNavigation: async (repoId: string): Promise<WikiNavigation> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki`)
    return data
  },

  getPage: async (repoId: string, slug: string): Promise<WikiPage> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki/${slug}`)
    return data
  },

  getStatus: async (repoId: string): Promise<WikiStatus> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki/status`)
    return data
  },

  generate: async (repoId: string): Promise<void> => {
    await api.post(`/api/repositories/${repoId}/wiki/generate`)
  },
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add src/lib/api.ts
git commit -m "feat(wiki): add wiki API types and methods"
```

---

### Task 9: Create WikiSidebar Component

**Files:**
- Create: `/root/work/neograph/frontend/src/components/WikiSidebar.tsx`

**Step 1: Create the sidebar component**

```tsx
import { useQuery } from '@tanstack/react-query'
import { wikiApi, WikiNavItem } from '@/lib/api'
import { ChevronRight, ChevronDown, FileText, Book } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

interface WikiSidebarProps {
  repoId: string
  currentSlug?: string
}

function NavItem({ item, repoId, currentSlug, level = 0 }: {
  item: WikiNavItem
  repoId: string
  currentSlug?: string
  level?: number
}) {
  const [expanded, setExpanded] = useState(true)
  const hasChildren = item.children && item.children.length > 0
  const isActive = item.slug === currentSlug

  return (
    <div>
      <Link
        to={`/repository/${repoId}/wiki/${item.slug}`}
        className={`flex items-center gap-2 px-3 py-2 text-sm rounded-md transition-colors ${
          isActive
            ? 'bg-blue-100 text-blue-700 font-medium'
            : 'text-gray-700 hover:bg-gray-100'
        }`}
        style={{ paddingLeft: `${12 + level * 16}px` }}
        onClick={(e) => {
          if (hasChildren) {
            e.preventDefault()
            setExpanded(!expanded)
          }
        }}
      >
        {hasChildren ? (
          expanded ? (
            <ChevronDown className="w-4 h-4 flex-shrink-0" />
          ) : (
            <ChevronRight className="w-4 h-4 flex-shrink-0" />
          )
        ) : (
          <FileText className="w-4 h-4 flex-shrink-0 text-gray-400" />
        )}
        <span className="truncate">{item.title}</span>
      </Link>

      {hasChildren && expanded && (
        <div>
          {item.children!.map((child) => (
            <NavItem
              key={child.slug}
              item={child}
              repoId={repoId}
              currentSlug={currentSlug}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function WikiSidebar({ repoId, currentSlug }: WikiSidebarProps) {
  const { data: navigation, isLoading, error } = useQuery({
    queryKey: ['wiki-navigation', repoId],
    queryFn: () => wikiApi.getNavigation(repoId),
  })

  if (isLoading) {
    return (
      <div className="p-4 text-sm text-gray-500">
        Loading navigation...
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-4 text-sm text-red-500">
        Failed to load navigation
      </div>
    )
  }

  if (!navigation?.items?.length) {
    return (
      <div className="p-4 text-sm text-gray-500">
        <Book className="w-8 h-8 mx-auto mb-2 text-gray-300" />
        <p className="text-center">No wiki pages yet</p>
      </div>
    )
  }

  return (
    <nav className="py-2">
      {navigation.items.map((item) => (
        <NavItem
          key={item.slug}
          item={item}
          repoId={repoId}
          currentSlug={currentSlug}
        />
      ))}
    </nav>
  )
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add src/components/WikiSidebar.tsx
git commit -m "feat(wiki): add WikiSidebar navigation component"
```

---

### Task 10: Create WikiContent Component with Markdown

**Files:**
- Create: `/root/work/neograph/frontend/src/components/WikiContent.tsx`

**Step 1: Create the content renderer component**

```tsx
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useEffect, useRef } from 'react'
import mermaid from 'mermaid'
import { WikiPage, TOCItem } from '@/lib/api'

// Initialize mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'neutral',
  securityLevel: 'loose',
})

interface WikiContentProps {
  page: WikiPage
}

function MermaidDiagram({ code, id }: { code: string; id: string }) {
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const renderDiagram = async () => {
      if (containerRef.current) {
        containerRef.current.innerHTML = ''
        try {
          const { svg } = await mermaid.render(`mermaid-${id}`, code)
          containerRef.current.innerHTML = svg
        } catch (err) {
          console.error('Mermaid render error:', err)
          containerRef.current.innerHTML = `<pre class="text-red-500">${code}</pre>`
        }
      }
    }
    renderDiagram()
  }, [code, id])

  return <div ref={containerRef} className="my-4 overflow-x-auto" />
}

function TableOfContents({ items }: { items: TOCItem[] }) {
  if (!items || items.length === 0) return null

  return (
    <div className="sticky top-4">
      <h4 className="text-sm font-semibold text-gray-500 mb-3">On this page</h4>
      <nav className="space-y-1">
        {items.map((item) => (
          <a
            key={item.id}
            href={`#${item.id}`}
            className={`block text-sm text-gray-600 hover:text-blue-600 transition-colors ${
              item.level === 1 ? 'font-medium' : ''
            }`}
            style={{ paddingLeft: `${(item.level - 1) * 12}px` }}
          >
            {item.title}
          </a>
        ))}
      </nav>
    </div>
  )
}

export function WikiContent({ page }: WikiContentProps) {
  return (
    <div className="flex gap-8">
      {/* Main content */}
      <div className="flex-1 min-w-0">
        <article className="prose prose-slate max-w-none">
          <ReactMarkdown
            remarkPlugins={[remarkGfm]}
            components={{
              // Add IDs to headings for TOC links
              h1: ({ children, ...props }) => {
                const id = String(children)
                  .toLowerCase()
                  .replace(/\s+/g, '-')
                  .replace(/[^a-z0-9-]/g, '')
                return <h1 id={id} {...props}>{children}</h1>
              },
              h2: ({ children, ...props }) => {
                const id = String(children)
                  .toLowerCase()
                  .replace(/\s+/g, '-')
                  .replace(/[^a-z0-9-]/g, '')
                return <h2 id={id} {...props}>{children}</h2>
              },
              h3: ({ children, ...props }) => {
                const id = String(children)
                  .toLowerCase()
                  .replace(/\s+/g, '-')
                  .replace(/[^a-z0-9-]/g, '')
                return <h3 id={id} {...props}>{children}</h3>
              },
              // Render code blocks
              code({ className, children, ...props }) {
                const match = /language-(\w+)/.exec(className || '')
                const language = match ? match[1] : ''

                // Check if it's a mermaid diagram
                if (language === 'mermaid') {
                  return (
                    <MermaidDiagram
                      code={String(children).trim()}
                      id={Math.random().toString(36).substr(2, 9)}
                    />
                  )
                }

                return (
                  <code className={className} {...props}>
                    {children}
                  </code>
                )
              },
            }}
          >
            {page.content}
          </ReactMarkdown>

          {/* Render explicit diagrams */}
          {page.diagrams?.map((diagram) => (
            <div key={diagram.id} className="my-6">
              <h4 className="text-sm font-medium text-gray-700 mb-2">
                {diagram.title}
              </h4>
              <MermaidDiagram code={diagram.code} id={diagram.id} />
            </div>
          ))}
        </article>
      </div>

      {/* Table of contents sidebar */}
      {page.tableOfContents && page.tableOfContents.length > 0 && (
        <div className="hidden lg:block w-64 flex-shrink-0">
          <TableOfContents items={page.tableOfContents} />
        </div>
      )}
    </div>
  )
}
```

**Step 2: Add Tailwind typography plugin**

Run: `cd /root/work/neograph/frontend && npm install @tailwindcss/typography`

**Step 3: Update tailwind.config.js to include typography**

Add to plugins array in `/root/work/neograph/frontend/tailwind.config.js`:
```javascript
plugins: [
  require('@tailwindcss/typography'),
],
```

**Step 4: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add src/components/WikiContent.tsx tailwind.config.js package.json package-lock.json
git commit -m "feat(wiki): add WikiContent component with markdown and mermaid"
```

---

### Task 11: Create WikiPage Page Component

**Files:**
- Create: `/root/work/neograph/frontend/src/pages/WikiPage.tsx`

**Step 1: Create the wiki page component**

```tsx
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { wikiApi, repositoryApi } from '@/lib/api'
import { WikiSidebar } from '@/components/WikiSidebar'
import { WikiContent } from '@/components/WikiContent'
import { ArrowLeft, RefreshCw, Book, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'

export default function WikiPageView() {
  const { id: repoId, slug = 'overview' } = useParams<{ id: string; slug?: string }>()

  const { data: repo } = useQuery({
    queryKey: ['repository', repoId],
    queryFn: () => repositoryApi.get(repoId!),
    enabled: !!repoId,
  })

  const { data: page, isLoading: pageLoading, error: pageError } = useQuery({
    queryKey: ['wiki-page', repoId, slug],
    queryFn: () => wikiApi.getPage(repoId!, slug!),
    enabled: !!repoId && !!slug,
  })

  const { data: status } = useQuery({
    queryKey: ['wiki-status', repoId],
    queryFn: () => wikiApi.getStatus(repoId!),
    enabled: !!repoId,
    refetchInterval: (data) =>
      data?.status === 'generating' ? 2000 : false,
  })

  const handleGenerate = async () => {
    if (repoId) {
      await wikiApi.generate(repoId)
    }
  }

  if (!repoId) {
    return <div>Repository ID required</div>
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <div className="border-b bg-gray-50 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to={`/repository/${repoId}`}
              className="flex items-center gap-2 text-gray-600 hover:text-gray-900"
            >
              <ArrowLeft className="w-4 h-4" />
              Back to Graph
            </Link>
            <div className="h-6 w-px bg-gray-300" />
            <div className="flex items-center gap-2">
              <Book className="w-5 h-5 text-blue-600" />
              <h1 className="text-lg font-semibold">{repo?.name || 'Repository'} Wiki</h1>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {status?.status === 'generating' && (
              <span className="text-sm text-gray-500">
                Generating... {status.progress}%
              </span>
            )}
            <Button
              variant="outline"
              size="sm"
              onClick={handleGenerate}
              disabled={status?.status === 'generating'}
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${status?.status === 'generating' ? 'animate-spin' : ''}`} />
              {status?.status === 'generating' ? 'Generating...' : 'Regenerate'}
            </Button>
          </div>
        </div>
      </div>

      <div className="flex">
        {/* Sidebar */}
        <div className="w-64 border-r bg-gray-50 min-h-[calc(100vh-73px)] overflow-y-auto">
          <WikiSidebar repoId={repoId} currentSlug={slug} />
        </div>

        {/* Main content */}
        <div className="flex-1 p-8 overflow-y-auto">
          {pageLoading ? (
            <div className="text-center py-12 text-gray-500">
              Loading page...
            </div>
          ) : pageError ? (
            <div className="text-center py-12">
              <AlertCircle className="w-12 h-12 mx-auto text-gray-300 mb-4" />
              <p className="text-gray-500 mb-4">Wiki page not found</p>
              <Button onClick={handleGenerate}>
                Generate Wiki
              </Button>
            </div>
          ) : page ? (
            <WikiContent page={page} />
          ) : (
            <div className="text-center py-12">
              <Book className="w-12 h-12 mx-auto text-gray-300 mb-4" />
              <p className="text-gray-500 mb-4">No wiki content available</p>
              <Button onClick={handleGenerate}>
                Generate Wiki
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add src/pages/WikiPage.tsx
git commit -m "feat(wiki): add WikiPage page component"
```

---

### Task 12: Add Wiki Route to App.tsx

**Files:**
- Modify: `/root/work/neograph/frontend/src/App.tsx`

**Step 1: Import WikiPage component**

Add import at top of App.tsx (after line 5):
```tsx
import WikiPageView from './pages/WikiPage'
```

**Step 2: Add wiki route**

In the Routes component (around line 56-60), add before the closing `</Routes>`:
```tsx
<Route path="/repository/:id/wiki" element={<WikiPageView />} />
<Route path="/repository/:id/wiki/:slug" element={<WikiPageView />} />
```

**Step 3: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add src/App.tsx
git commit -m "feat(wiki): add wiki routes to App.tsx"
```

---

### Task 13: Add Wiki Link to Repository Detail Page

**Files:**
- Modify: `/root/work/neograph/frontend/src/pages/RepositoryDetailPage.tsx`

**Step 1: Import Book icon**

At top of file, add to lucide-react import:
```tsx
import { ArrowLeft, Book } from 'lucide-react'
```

**Step 2: Add wiki link button**

In the header section (find the "Back" link), add a Wiki link next to it:

After the Back link and before the repo name, add:
```tsx
<Link
  to={`/repository/${id}/wiki`}
  className="flex items-center gap-2 text-gray-600 hover:text-blue-600 transition-colors"
>
  <Book className="w-4 h-4" />
  <span>Wiki</span>
</Link>
```

**Step 3: Verify TypeScript compiles**

Run: `cd /root/work/neograph/frontend && npx tsc --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add src/pages/RepositoryDetailPage.tsx
git commit -m "feat(wiki): add wiki link to repository detail page"
```

---

### Task 14: Build and Test Frontend

**Step 1: Build frontend**

Run: `cd /root/work/neograph/frontend && npm run build`
Expected: Build succeeds

**Step 2: Start dev server**

Run: `cd /root/work/neograph/frontend && npm run dev &`
Expected: Dev server starts on port 5173

**Step 3: Test in browser**

Navigate to: `http://localhost:5173/repository/{repo-id}/wiki`
Expected: Wiki page loads with sidebar and content area

**Step 4: Final commit**

```bash
git add -A
git commit -m "feat(wiki): Phase 2 complete - frontend wiki UI"
```

---

## Phase 3: Agent Wiki Generation (Future)

### Task 15: Enhance Doc Writer Agent

**Files:**
- Modify: `/root/work/neograph/agents/src/agents/doc_writer.py`

*Implementation details for full wiki generation via Claude agent - to be defined in next planning session.*

---

## Summary

This plan implements a DeepWiki-like feature in 14 tasks across 2 phases:

**Phase 1 (Tasks 1-6):** Backend infrastructure
- WikiPage model types
- Neo4j WikiReader and WikiWriter
- API handlers and routes
- Build verification

**Phase 2 (Tasks 7-14):** Frontend UI
- Dependencies (react-markdown, mermaid)
- API types and methods
- WikiSidebar navigation
- WikiContent with markdown/mermaid
- WikiPage layout
- Route integration

**Phase 3 (Future):** Full AI generation via Claude agent
