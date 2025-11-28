# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

NeoGraph is a code intelligence platform that indexes Git repositories into a Neo4j graph database, enabling semantic search and AI-powered code exploration. It consists of three services:

- **Backend** (Go/Fiber) - REST API, repository indexing, Neo4j graph operations
- **Frontend** (React/TypeScript/Vite) - Web UI with graph visualization and wiki
- **Agents** (Python/FastAPI) - Claude-powered code analysis service

## Build & Run Commands

### Backend (Go)
```bash
cd backend
go run cmd/server/main.go                    # Run server
go test ./...                                # Run all tests
go test ./internal/db/...                    # Run tests for specific package
go test -v -run TestFunctionName ./pkg/...   # Run single test
go build -o server cmd/server/main.go        # Build binary
```

### Frontend (React/Vite)
```bash
cd frontend
npm install                    # Install dependencies
npm run dev                    # Development server (port 5173)
npm run build                  # Production build (runs tsc first)
npx tsc --noEmit              # Type check without emit
```

### Agents (Python)
```bash
cd agents
pip install -r requirements.txt
uvicorn src.server:app --host 0.0.0.0 --port 8001 --reload
```

### Docker (full stack)
```bash
docker-compose up neo4j tei     # Start Neo4j and embeddings service first
docker-compose up               # Start all services
```

## Environment Variables

Backend reads from environment (see `.env.example`):
- `NEO4J_URI` (default: bolt://localhost:7687)
- `NEO4J_USER` (default: neo4j)
- `NEO4J_PASSWORD` (default: neograph_password)
- `TEI_URL` (default: http://localhost:8080)
- `BACKEND_PORT` (default: 3001)

Frontend:
- `VITE_API_URL` (default: http://localhost:3001)

## Architecture

### Backend Structure (`backend/internal/`)
- `api/` - Fiber HTTP handlers and routes
- `db/` - Neo4j client, graph reader/writer, wiki storage, vector index
- `indexer/` - Code parsing pipeline using tree-sitter
- `git/` - Repository cloning
- `embedding/` - TEI client for semantic embeddings
- `models/` - Domain types (Repository, File, CodeEntity, WikiPage)
- `agent/` - Proxy to Python agents service
- `config/` - Environment configuration

### Data Flow
1. User adds repository URL → `git/clone.go` clones repo
2. `indexer/pipeline.go` parses files with tree-sitter → extracts functions/methods
3. `db/graph_writer.go` stores entities as Neo4j nodes with relationships
4. `embedding/tei_client.go` generates vectors → stored via `db/vector_index.go`
5. Frontend queries via REST API → graph visualization with vis-network

### Frontend Structure (`frontend/src/`)
- `pages/` - Route components (RepositoryList, RepositoryDetail, Wiki, Search)
- `components/` - Reusable UI (GraphVisualization, FileTree, WikiSidebar, WikiContent)
- `lib/api.ts` - API client with typed endpoints

### API Endpoints
- `GET/POST /api/repositories` - List/create repositories
- `GET /api/repositories/:id/graph` - Get graph data for visualization
- `GET /api/repositories/:id/wiki/:slug` - Get wiki page content
- `POST /api/repositories/:id/wiki/generate` - Generate wiki documentation
- `GET /api/search?q=` - Global semantic search
- `POST /api/agents/chat` - Chat with Claude agent

## Testing Notes

Backend tests require a running Neo4j instance. Unit tests for pure functions (extractTOC, buildNavTree) can run without Neo4j.

## Additional Instructions

- Use perplexity and context7 for research on all tasks
