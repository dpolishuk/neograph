# GitHub Copilot Instructions for NeoGraph

## Project Overview

**NeoGraph** is a code intelligence platform that indexes Git repositories into a Neo4j graph database, enabling semantic search and AI-powered code exploration.

**Key Features:**
- Repository indexing with tree-sitter parsing
- Graph-based code navigation
- Semantic search with embeddings
- AI-powered code analysis via Claude
- Auto-generated wiki documentation

## Tech Stack

- **Backend**: Go 1.25+ with Fiber framework
- **Frontend**: React 18 + TypeScript + Vite
- **Database**: Neo4j 5.x
- **Agents**: Python with FastAPI
- **Embeddings**: HuggingFace TEI

## Issue Tracking with bd

**CRITICAL**: This project uses **bd (beads)** for ALL task tracking. Do NOT create markdown TODO lists.

### Essential Commands

```bash
# Find work
bd ready --json                    # Unblocked issues

# Create and manage
bd create "Title" -t bug|feature|task -p 0-4 --json
bd update <id> --status in_progress --json
bd close <id> --reason "Done" --json
```

### Workflow

1. **Check ready work**: `bd ready --json`
2. **Claim task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** `bd create "Found bug" -p 1 --deps discovered-from:<parent-id> --json`
5. **Complete**: `bd close <id> --reason "Done" --json`

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

## Project Structure

```
neograph/
├── backend/             # Go backend
│   ├── cmd/server/      # Main entrypoint
│   └── internal/        # Core packages
│       ├── api/         # HTTP handlers
│       ├── db/          # Neo4j operations
│       ├── indexer/     # Code parsing
│       └── models/      # Domain types
├── frontend/            # React frontend
│   └── src/
│       ├── pages/       # Route components
│       ├── components/  # UI components
│       └── lib/         # API client
├── agents/              # Python AI service
└── .beads/
    └── issues.jsonl     # Git-synced issue storage
```

## Build Commands

```bash
# Backend
cd backend && go run cmd/server/main.go
cd backend && go test ./...

# Frontend
cd frontend && npm run dev
cd frontend && npm run build

# Agents
cd agents && uvicorn src.server:app --port 8001 --reload
```

## Important Rules

- Use bd for ALL task tracking
- Always use `--json` flag for programmatic bd commands
- Link discovered work with `discovered-from` dependencies
- Check `bd ready` before asking "what should I work on?"
- Do NOT create markdown TODO lists
- Do NOT duplicate tracking systems

---

**For detailed workflows, see [AGENTS.md](../AGENTS.md) and [CLAUDE.md](../CLAUDE.md)**
