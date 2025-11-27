# NeoGraph Phase 2 Design

## Overview

Phase 2 extends the MVP with repository exploration, semantic search, and Claude AI agents for intelligent code analysis.

## Phase 2.1: Repository Detail View

### Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  NeoGraph    [Search all repos...]                    [Cmd+K]   │
├───────────────┬─────────────────────────────────────────────────┤
│ < Back        │                                                 │
│               │         GRAPH VISUALIZATION                     │
│ envconfig     │              (neovis.js)                        │
│ ├─ envconfig.go   │                                             │
│ │  ├─ fn Process    │    [File] ──> [Function] ──> [Function]  │
│ │  ├─ fn MustProcess│                                           │
│ │  └─ fn gatherInfo │        Toggle: [Structure] [Calls]        │
│ ├─ usage.go        │                                            │
│ └─ doc.go          │                                            │
│               │                                                 │
│ ──────────────│─────────────────────────────────────────────────│
│ Stats:        │  Selected: Process()                            │
│ 8 files       │  File: envconfig.go:45-89                       │
│ 76 functions  │  Calls: gatherInfo, processField                │
│               │  Called by: MustProcess                         │
└───────────────┴─────────────────────────────────────────────────┘
```

### Components

- **Left sidebar**: File tree with expandable functions, click to focus in graph
- **Main area**: neovis.js graph with zoom/pan, node colors by type (File=blue, Function=green)
- **Toggle buttons**: Switch between "Structure" (hierarchy) and "Calls" (call graph) views
- **Detail panel**: Shows selected node info - signature, location, relationships

### API Endpoints

- `GET /api/repositories/:id/files` - File tree with functions
- `GET /api/repositories/:id/graph?type=structure|calls` - Graph data for neovis.js

---

## Phase 2.2: Semantic Search

### Architecture

```
User Query → TEI (Qodo-Embed-1) → Vector → Neo4j Vector Index → Results
                                              ↓
                                    + Graph context (callers, files)
```

### Features

1. **Global Search** (`/search` page):
   - Search across all indexed repositories
   - Results grouped by repo, ranked by similarity score
   - Click result → navigate to repo detail with node highlighted

2. **Per-Repo Search** (in detail view):
   - Search box in sidebar header
   - Results filter the file tree + highlight matching nodes in graph
   - "Find similar code" button on selected function

### Backend Changes

- New endpoints: `GET /api/search?q=...` (global), `GET /api/repositories/:id/search?q=...` (scoped)
- Background job: Generate embeddings for all functions via TEI service
- Store embeddings in Neo4j using `db.create.setNodeVectorProperty()`
- Create HNSW vector index on Function nodes

### Indexing Pipeline Update

```
Parse File → Extract Functions → Call TEI API → Store embedding in Neo4j
```

---

## Phase 2.3: Claude Agents

### Architecture

```
Frontend ←→ Go Backend ←→ Python Agent Service ←→ Claude API
                ↓                    ↓
              Neo4j ←───── MCP Tools (cypher, vector_search, blast_radius)
```

### Agents

| Agent | Trigger | Capability |
|-------|---------|------------|
| **Explorer** | "Find authentication code" | Semantic search + graph traversal |
| **Analyzer** | "What depends on X?" | Blast radius, impact analysis |
| **Doc Writer** | "Document this module" | Generate markdown docs from code |

### UI Components

1. **Command Bar** (Cmd+K):
   - Quick input, auto-routes to appropriate agent
   - Inline results with "Open in chat" option
   - Recent queries history

2. **Chat Panel** (slide-out drawer):
   - Full conversation with streaming responses
   - Code blocks with syntax highlighting
   - "Show in graph" button for referenced functions
   - Cypher query preview (expandable)

### MCP Server Tools

- `neo4j_query` - Run arbitrary Cypher
- `vector_search` - Semantic code search
- `blast_radius` - Find dependents of a function
- `trace_imports` - Follow import chains

---

## Implementation Tasks

### Phase 2.1 - Repository Detail View (6 tasks)

1. Backend: Add `/api/repositories/:id/files` and `/api/repositories/:id/graph` endpoints
2. Frontend: Add React Router, create RepositoryDetail page
3. Frontend: File tree sidebar component with expandable nodes
4. Frontend: Integrate neovis.js for graph visualization
5. Frontend: Node detail panel (signature, location, relationships)
6. Frontend: Toggle between Structure/Calls graph views

### Phase 2.2 - Semantic Search (5 tasks)

1. Backend: TEI client for embedding generation
2. Backend: Update indexer to generate embeddings for functions
3. Backend: Create Neo4j vector index, add search endpoints
4. Frontend: Global search page with results list
5. Frontend: Per-repo search in sidebar with graph highlighting

### Phase 2.3 - Claude Agents (6 tasks)

1. Create Python agent service with Claude SDK
2. Implement MCP server with Neo4j tools
3. Backend: Proxy endpoints for agent communication
4. Frontend: Command bar component (Cmd+K)
5. Frontend: Chat panel drawer with streaming
6. Implement Explorer, Analyzer, Doc Writer agents

**Total: 17 tasks**

---

## Technology Stack

| Component | Technology |
|-----------|------------|
| Graph Visualization | neovis.js |
| Embeddings | Qodo-Embed-1 via HuggingFace TEI |
| Vector Index | Neo4j HNSW |
| Agent Framework | Claude Agent SDK (Python) |
| MCP Server | Python with neo4j driver |
| Frontend Routing | React Router |
| Command Bar | cmdk (pacocoursey/cmdk) |
