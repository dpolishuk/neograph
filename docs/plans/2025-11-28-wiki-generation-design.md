# Wiki Generation Design

## Overview

This document describes the design for Claude-powered wiki generation in NeoGraph. When a repository is indexed, the system automatically generates a multi-page wiki with an overview and module-level documentation.

## Design Decisions

| Decision | Choice | Reasoning |
|----------|--------|-----------|
| Content scope | Multi-page structure | Overview + one page per module. Balances depth with practicality |
| Generation trigger | Auto on index | Wiki generates automatically when repository is indexed |
| Claude integration | New dedicated endpoint | Adds `POST /wiki/generate` to agents service |
| Page structure | Dynamic by module | Groups files by directory/package into logical pages |
| Diagrams | Overview diagram only | One architecture diagram on main page (YAGNI) |
| Progress feedback | Page-by-page | Shows "Generating page 2 of 5: Database Module" |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Frontend     │────▶│  Backend (Go)   │────▶│  Agents (Python)│
│  React + Vite   │     │     Fiber       │     │    FastAPI      │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │                        │
                               ▼                        ▼
                        ┌─────────────────┐     ┌─────────────────┐
                        │     Neo4j       │     │   Claude API    │
                        │  (code graph)   │     │  (generation)   │
                        └─────────────────┘     └─────────────────┘
```

**Flow:**
1. Repository indexing completes → Backend calls `generateWikiPages()`
2. Backend marks status as "generating" and calls `POST http://agents:8001/wiki/generate`
3. Agents service queries Neo4j for code structure, calls Claude to generate content
4. Agents returns structured pages → Backend stores each page in Neo4j
5. Frontend polls status, shows progress, displays pages when ready

## Agents Service: New Endpoint

**Endpoint:** `POST /wiki/generate`

### Request
```json
{
  "repo_id": "5f89c606-5652-4474-9496-1a81c59c1d6c",
  "repo_name": "envconfig"
}
```

### Response
```json
{
  "pages": [
    {
      "slug": "overview",
      "title": "Overview",
      "content": "# envconfig\n\nA Go library for...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [
        {
          "id": "arch-diagram",
          "title": "Architecture",
          "code": "graph TD\n  A[envconfig] --> B[Process]..."
        }
      ]
    },
    {
      "slug": "envconfig-module",
      "title": "envconfig Module",
      "content": "# envconfig Module\n\n## Functions\n...",
      "order": 2,
      "parent_slug": "overview"
    }
  ]
}
```

### Logic
1. Query Neo4j for all files/functions in the repo
2. Group files by directory to identify modules
3. Call Claude once with full context to generate all pages
4. Return structured JSON response

## Backend Changes

**File:** `backend/internal/api/handlers.go`

```go
func (h *Handler) generateWikiPages(repo *models.Repository) {
    ctx := context.Background()

    // 1. Set status to "generating"
    h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
        Status:   "generating",
        Progress: 0,
    })

    // 2. Call agents service
    resp, err := http.Post(
        h.cfg.AgentURL + "/wiki/generate",
        "application/json",
        // body: { repo_id, repo_name }
    )

    // 3. Parse response and store each page
    for i, page := range resp.Pages {
        h.wikiWriter.WritePage(ctx, &models.WikiPage{...})

        // Update progress
        h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
            Status:      "generating",
            Progress:    (i+1) * 100 / len(resp.Pages),
            CurrentPage: page.Title,
            TotalPages:  len(resp.Pages),
        })
    }

    // 4. Set status to "ready"
    h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, &models.WikiStatus{
        Status:     "ready",
        Progress:   100,
        TotalPages: len(resp.Pages),
    })
}
```

## Claude Prompt Strategy

One Claude call generates all pages using structured JSON output:

```python
WIKI_GENERATION_PROMPT = """
You are generating documentation for a code repository.

## Repository: {repo_name}

## Code Structure:
{code_structure}

## Instructions:
Generate a multi-page wiki with:

1. **Overview page** (slug: "overview")
   - Project purpose and description
   - Architecture diagram (mermaid flowchart)
   - Key modules summary
   - Getting started info

2. **Module pages** (one per directory)
   - Module purpose
   - Key functions with descriptions
   - Usage examples
   - Dependencies on other modules

## Output Format:
Return valid JSON with this structure:
{
  "pages": [
    {
      "slug": "overview",
      "title": "Overview",
      "content": "markdown content...",
      "order": 1,
      "parent_slug": null,
      "diagrams": [{"id": "...", "title": "...", "code": "mermaid..."}]
    }
  ]
}
"""
```

## Implementation Tasks

| # | Task | File | Description |
|---|------|------|-------------|
| 1 | Add wiki generation endpoint | `agents/src/server.py` | New `POST /wiki/generate` endpoint |
| 2 | Create wiki generator module | `agents/src/wiki/generator.py` | Logic to query Neo4j, call Claude, parse response |
| 3 | Update backend handler | `backend/internal/api/handlers.go` | Call agents service instead of stub |
| 4 | Auto-trigger on index | `backend/internal/api/handlers.go` | Call `generateWikiPages()` after indexing completes |

## Resource Estimates

**Per wiki generation:**
- ~1 Claude API call (all pages in one request)
- Input: ~2-5K tokens (code structure)
- Output: ~3-8K tokens (wiki content)
- Time: ~10-30 seconds depending on repo size
