# Wiki Generation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement Claude-powered wiki generation that auto-generates documentation when a repository is indexed.

**Architecture:** Backend triggers agents service via new `/wiki/generate` endpoint. Agents queries Neo4j for code structure, calls Claude to generate pages, returns structured JSON. Backend stores pages and updates progress.

**Tech Stack:** Python/FastAPI (agents), Go/Fiber (backend), Neo4j (storage), Claude API (generation)

---

## Task 1: Create Wiki Generator Module

**Files:**
- Create: `agents/src/wiki/__init__.py`
- Create: `agents/src/wiki/generator.py`

**Step 1: Create wiki package init**

Create `agents/src/wiki/__init__.py`:

```python
"""Wiki generation module."""
from .generator import generate_wiki

__all__ = ["generate_wiki"]
```

**Step 2: Create generator module with Neo4j query function**

Create `agents/src/wiki/generator.py`:

```python
"""Wiki generation using Claude."""
import json
import logging
from typing import Any

import anthropic

from ..config import settings
from ..tools.neo4j_tools import neo4j_query

logger = logging.getLogger(__name__)

# Initialize Anthropic client
client = anthropic.Anthropic(api_key=settings.anthropic_api_key)


def get_code_structure(repo_id: str) -> dict[str, Any]:
    """
    Query Neo4j for repository code structure.

    Args:
        repo_id: Repository ID

    Returns:
        Dictionary with files grouped by directory
    """
    query = """
    MATCH (repo:Repository {id: $repo_id})-[:CONTAINS]->(file:File)
    OPTIONAL MATCH (file)-[:DECLARES]->(fn:Function)
    WITH file, collect({
        name: fn.name,
        signature: fn.signature,
        startLine: fn.startLine,
        endLine: fn.endLine
    }) as functions
    RETURN {
        path: file.path,
        language: file.language,
        functions: functions
    } as file
    ORDER BY file.path
    """

    results = neo4j_query(query.replace("$repo_id", f"'{repo_id}'"))

    # Group files by directory
    modules = {}
    for record in results:
        if "error" in record:
            continue
        file_data = record.get("file", record)
        path = file_data.get("path", "")

        # Extract directory as module name
        parts = path.split("/")
        if len(parts) > 1:
            module = parts[-2] if parts[-2] != "src" else parts[-1].replace(".py", "").replace(".go", "").replace(".ts", "")
        else:
            module = "root"

        if module not in modules:
            modules[module] = []
        modules[module].append(file_data)

    return modules


WIKI_PROMPT = """You are generating documentation for a code repository.

## Repository: {repo_name}

## Code Structure:
{code_structure}

## Instructions:
Generate a multi-page wiki with:

1. **Overview page** (slug: "overview", order: 1)
   - Project purpose based on the code
   - Architecture diagram in mermaid format
   - Key modules summary (one line each)

2. **Module pages** (one per directory, order: 2+)
   - Module purpose
   - Key functions with brief descriptions
   - How this module relates to others

## Output Format:
Return ONLY valid JSON (no markdown, no explanation) with this exact structure:
{{
  "pages": [
    {{
      "slug": "overview",
      "title": "Overview",
      "content": "# Project Name\\n\\nmarkdown content...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [
        {{
          "id": "architecture",
          "title": "Architecture",
          "code": "graph TD\\n  A[Module] --> B[Module]"
        }}
      ]
    }},
    {{
      "slug": "module-name",
      "title": "Module Name",
      "content": "# Module Name\\n\\nmarkdown content...",
      "order": 2,
      "parent_slug": "overview",
      "diagrams": []
    }}
  ]
}}

IMPORTANT:
- Return ONLY the JSON, no other text
- Use \\n for newlines in content strings
- Keep content concise (200-400 words per page)
- Generate 3-6 pages total
"""


def generate_wiki(repo_id: str, repo_name: str) -> dict[str, Any]:
    """
    Generate wiki pages for a repository using Claude.

    Args:
        repo_id: Repository ID
        repo_name: Repository name for display

    Returns:
        Dictionary with 'pages' list
    """
    # Get code structure from Neo4j
    modules = get_code_structure(repo_id)

    if not modules:
        return {
            "pages": [{
                "slug": "overview",
                "title": "Overview",
                "content": f"# {repo_name}\n\nNo code structure found. Please ensure the repository has been indexed.",
                "order": 1,
                "parent_slug": None,
                "diagrams": []
            }]
        }

    # Format code structure for prompt
    code_structure = json.dumps(modules, indent=2)

    # Build prompt
    prompt = WIKI_PROMPT.format(
        repo_name=repo_name,
        code_structure=code_structure
    )

    logger.info(f"Generating wiki for {repo_name} with {len(modules)} modules")

    # Call Claude
    response = client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=8192,
        messages=[{"role": "user", "content": prompt}]
    )

    # Extract response text
    response_text = ""
    for block in response.content:
        if hasattr(block, "text"):
            response_text += block.text

    # Parse JSON response
    try:
        # Try to find JSON in response
        response_text = response_text.strip()
        if response_text.startswith("```"):
            # Remove markdown code blocks
            lines = response_text.split("\n")
            response_text = "\n".join(lines[1:-1])

        result = json.loads(response_text)
        logger.info(f"Generated {len(result.get('pages', []))} wiki pages")
        return result
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse Claude response: {e}")
        logger.error(f"Response was: {response_text[:500]}")
        return {
            "pages": [{
                "slug": "overview",
                "title": "Overview",
                "content": f"# {repo_name}\n\nWiki generation failed. Please try again.",
                "order": 1,
                "parent_slug": None,
                "diagrams": []
            }]
        }
```

**Step 3: Verify syntax**

Run:
```bash
cd /root/work/neograph/agents && python -m py_compile src/wiki/__init__.py src/wiki/generator.py
```

Expected: No output (success)

**Step 4: Commit**

```bash
git add agents/src/wiki/
git commit -m "feat(agents): add wiki generator module"
```

---

## Task 2: Add Wiki Generate Endpoint

**Files:**
- Modify: `agents/src/server.py`

**Step 1: Add imports and request model**

In `agents/src/server.py`, add after existing imports:

```python
from .wiki import generate_wiki
```

Add after `ChatResponse` class:

```python
class WikiGenerateRequest(BaseModel):
    """Request model for wiki generation."""
    repo_id: str
    repo_name: str


class WikiPage(BaseModel):
    """Single wiki page."""
    slug: str
    title: str
    content: str
    order: int
    parent_slug: Optional[str] = None
    diagrams: List[Dict[str, Any]] = []


class WikiGenerateResponse(BaseModel):
    """Response model for wiki generation."""
    pages: List[WikiPage]
```

**Step 2: Add endpoint**

Add before `@app.get("/health")`:

```python
@app.post("/wiki/generate", response_model=WikiGenerateResponse)
async def wiki_generate(request: WikiGenerateRequest):
    """
    Generate wiki pages for a repository.

    Args:
        request: Wiki generation request with repo_id and repo_name

    Returns:
        WikiGenerateResponse with generated pages
    """
    logger.info(f"Generating wiki for repo {request.repo_id} ({request.repo_name})")

    result = generate_wiki(request.repo_id, request.repo_name)

    return WikiGenerateResponse(pages=result.get("pages", []))
```

**Step 3: Verify syntax**

Run:
```bash
cd /root/work/neograph/agents && python -m py_compile src/server.py
```

Expected: No output (success)

**Step 4: Commit**

```bash
git add agents/src/server.py
git commit -m "feat(agents): add /wiki/generate endpoint"
```

---

## Task 3: Add Wiki Generation to Backend Agent Proxy

**Files:**
- Modify: `backend/internal/agent/proxy.go`

**Step 1: Add request/response types**

In `backend/internal/agent/proxy.go`, add after `ChatResponse`:

```go
// WikiGenerateRequest represents the request body for wiki generation
type WikiGenerateRequest struct {
	RepoID   string `json:"repo_id"`
	RepoName string `json:"repo_name"`
}

// WikiDiagram represents a mermaid diagram
type WikiDiagram struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"`
}

// WikiPageResponse represents a single wiki page from the agent
type WikiPageResponse struct {
	Slug       string        `json:"slug"`
	Title      string        `json:"title"`
	Content    string        `json:"content"`
	Order      int           `json:"order"`
	ParentSlug *string       `json:"parent_slug"`
	Diagrams   []WikiDiagram `json:"diagrams"`
}

// WikiGenerateResponse represents the response from wiki generation
type WikiGenerateResponse struct {
	Pages []WikiPageResponse `json:"pages"`
}
```

**Step 2: Add GenerateWiki method**

Add after `Chat` method:

```go
// GenerateWiki calls the agent service to generate wiki pages
func (p *AgentProxy) GenerateWiki(ctx context.Context, repoID, repoName string) (*WikiGenerateResponse, error) {
	// Construct request
	reqBody := WikiGenerateRequest{
		RepoID:   repoID,
		RepoName: repoName,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/wiki/generate", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request with longer timeout for wiki generation
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var wikiResp WikiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&wikiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wikiResp, nil
}
```

**Step 3: Add time import**

At the top of the file, add `"time"` to imports:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)
```

**Step 4: Verify build**

Run:
```bash
cd /root/work/neograph/backend && go build ./...
```

Expected: No errors

**Step 5: Commit**

```bash
git add backend/internal/agent/proxy.go
git commit -m "feat(backend): add GenerateWiki to agent proxy"
```

---

## Task 4: Update generateWikiPages Handler

**Files:**
- Modify: `backend/internal/api/handlers.go`

**Step 1: Replace generateWikiPages implementation**

Find the `generateWikiPages` function (around line 375-417) and replace it entirely:

```go
func (h *Handler) generateWikiPages(repo *models.Repository) {
	ctx := context.Background()

	setError := func(msg string) {
		status := &models.WikiStatus{
			Status:       "error",
			Progress:     0,
			ErrorMessage: msg,
		}
		h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, status)
	}

	// Set status to generating
	h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
		Status:   "generating",
		Progress: 0,
	})

	// Clear existing wiki
	if err := h.wikiWriter.ClearWiki(ctx, repo.ID); err != nil {
		setError("failed to clear existing wiki: " + err.Error())
		return
	}

	// Call agents service to generate wiki
	wikiResp, err := h.agentProxy.GenerateWiki(ctx, repo.ID, repo.Name)
	if err != nil {
		setError("failed to generate wiki: " + err.Error())
		return
	}

	// Store each page
	totalPages := len(wikiResp.Pages)
	for i, page := range wikiResp.Pages {
		// Convert diagrams
		diagrams := make([]models.Diagram, len(page.Diagrams))
		for j, d := range page.Diagrams {
			diagrams[j] = models.Diagram{
				ID:    d.ID,
				Title: d.Title,
				Code:  d.Code,
			}
		}

		// Create wiki page
		wikiPage := &models.WikiPage{
			RepoID:     repo.ID,
			Slug:       page.Slug,
			Title:      page.Title,
			Content:    page.Content,
			Order:      page.Order,
			ParentSlug: "",
			Diagrams:   diagrams,
		}
		if page.ParentSlug != nil {
			wikiPage.ParentSlug = *page.ParentSlug
		}

		if err := h.wikiWriter.WritePage(ctx, wikiPage); err != nil {
			setError("failed to write page: " + err.Error())
			return
		}

		// Update progress
		progress := ((i + 1) * 100) / totalPages
		h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
			Status:      "generating",
			Progress:    progress,
			CurrentPage: page.Title,
			TotalPages:  totalPages,
		})
	}

	// Set status to ready
	h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
		Status:     "ready",
		Progress:   100,
		TotalPages: totalPages,
	})
}
```

**Step 2: Verify build**

Run:
```bash
cd /root/work/neograph/backend && go build ./...
```

Expected: No errors

**Step 3: Commit**

```bash
git add backend/internal/api/handlers.go
git commit -m "feat(backend): integrate Claude wiki generation"
```

---

## Task 5: Add Auto-Generate on Index

**Files:**
- Modify: `backend/internal/api/handlers.go`

**Step 1: Update indexRepository to call generateWikiPages**

Find the `indexRepository` function and add wiki generation after successful indexing:

```go
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

	// Auto-generate wiki after successful indexing
	go h.generateWikiPages(repo)

	// Status will be updated to 'ready' by WriteIndexResult
}
```

**Step 2: Verify build**

Run:
```bash
cd /root/work/neograph/backend && go build ./...
```

Expected: No errors

**Step 3: Commit**

```bash
git add backend/internal/api/handlers.go
git commit -m "feat(backend): auto-generate wiki on repository index"
```

---

## Task 6: Test End-to-End

**Step 1: Restart agents service**

Run:
```bash
cd /root/work/neograph/agents && pkill -f "uvicorn src.server" || true
cd /root/work/neograph/agents && NEO4J_PASSWORD=telebrain_password nohup uvicorn src.server:app --host 0.0.0.0 --port 8001 > /var/log/neograph-agents.log 2>&1 &
sleep 3
curl http://localhost:8001/health
```

Expected: `{"status":"ok"}`

**Step 2: Restart backend**

Run:
```bash
pkill -f "go run cmd/server/main.go" || true
cd /root/work/neograph/backend && nohup env NEO4J_PASSWORD=telebrain_password AGENT_URL=http://localhost:8001 go run cmd/server/main.go > /var/log/neograph.log 2>&1 &
sleep 5
curl http://localhost:3001/health
```

Expected: `{"service":"neograph-backend","status":"ok"}`

**Step 3: Test wiki generation manually**

Run:
```bash
curl -X POST http://localhost:3001/api/repositories/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki/generate
sleep 30
curl http://localhost:3001/api/repositories/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki/status
```

Expected: Status should show "ready" with multiple pages

**Step 4: Verify in browser**

Open: http://92.118.112.225:5173/repository/5f89c606-5652-4474-9496-1a81c59c1d6c/wiki

Expected: Wiki sidebar should show multiple pages, content should be real documentation

**Step 5: Commit final**

```bash
git add -A
git commit -m "test: verify wiki generation works end-to-end"
```
