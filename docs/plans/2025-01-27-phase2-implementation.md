# NeoGraph Phase 2 Implementation Plan

## Overview

This document provides detailed implementation steps for Phase 2 features. Each task includes exact file paths, code snippets, and verification commands.

---

## Phase 2.1: Repository Detail View

### Task 1: Backend - Add File Tree and Graph Endpoints

**File:** `backend/internal/api/routes.go`

Add new routes:
```go
// In SetupRoutes function, add after existing routes:
repos.Get("/:id/files", h.GetRepositoryFiles)
repos.Get("/:id/graph", h.GetRepositoryGraph)
```

**File:** `backend/internal/api/handler.go`

Add handlers:
```go
// GetRepositoryFiles returns file tree with functions for a repository
func (h *Handler) GetRepositoryFiles(c fiber.Ctx) error {
    id := c.Params("id")
    files, err := h.graphReader.GetFileTree(c.Context(), id)
    if err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    return c.JSON(files)
}

// GetRepositoryGraph returns neovis.js compatible graph data
func (h *Handler) GetRepositoryGraph(c fiber.Ctx) error {
    id := c.Params("id")
    graphType := c.Query("type", "structure") // "structure" or "calls"
    graph, err := h.graphReader.GetGraph(c.Context(), id, graphType)
    if err != nil {
        return fiber.NewError(fiber.StatusInternalServerError, err.Error())
    }
    return c.JSON(graph)
}
```

**File:** `backend/internal/db/graph_reader.go` (NEW)

```go
package db

import (
    "context"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphReader struct {
    client *Neo4jClient
}

func NewGraphReader(client *Neo4jClient) *GraphReader {
    return &GraphReader{client: client}
}

type FileNode struct {
    ID        string        `json:"id"`
    Path      string        `json:"path"`
    Language  string        `json:"language"`
    Functions []FunctionRef `json:"functions"`
}

type FunctionRef struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Signature string `json:"signature"`
    StartLine int    `json:"startLine"`
    EndLine   int    `json:"endLine"`
}

type GraphData struct {
    Nodes []GraphNode `json:"nodes"`
    Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
    ID    string            `json:"id"`
    Label string            `json:"label"`
    Type  string            `json:"type"`
    Props map[string]any    `json:"props"`
}

type GraphEdge struct {
    ID     string `json:"id"`
    Source string `json:"source"`
    Target string `json:"target"`
    Type   string `json:"type"`
}

func (r *GraphReader) GetFileTree(ctx context.Context, repoID string) ([]FileNode, error) {
    result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        query := `
            MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)
            OPTIONAL MATCH (f)-[:DECLARES]->(fn:Function)
            WITH f, collect({
                id: fn.id,
                name: fn.name,
                signature: fn.signature,
                startLine: fn.startLine,
                endLine: fn.endLine
            }) as functions
            RETURN f.id as id, f.path as path, f.language as language, functions
            ORDER BY f.path
        `
        records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
        if err != nil {
            return nil, err
        }

        var files []FileNode
        for records.Next(ctx) {
            rec := records.Record()
            file := FileNode{
                ID:       rec.Values[0].(string),
                Path:     rec.Values[1].(string),
                Language: rec.Values[2].(string),
            }
            // Parse functions...
            files = append(files, file)
        }
        return files, nil
    })
    if err != nil {
        return nil, err
    }
    return result.([]FileNode), nil
}

func (r *GraphReader) GetGraph(ctx context.Context, repoID, graphType string) (*GraphData, error) {
    var query string
    if graphType == "calls" {
        query = `
            MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)-[:DECLARES]->(fn)
            OPTIONAL MATCH (fn)-[c:CALLS]->(target)
            RETURN fn, f, c, target
        `
    } else {
        query = `
            MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)
            OPTIONAL MATCH (f)-[:DECLARES]->(fn)
            RETURN f, fn
        `
    }
    // Execute and transform to GraphData...
    return &GraphData{}, nil
}
```

**Verification:**
```bash
cd /root/work/neograph/backend && go build ./...
curl http://localhost:3001/api/repositories/{id}/files
curl http://localhost:3001/api/repositories/{id}/graph?type=structure
curl http://localhost:3001/api/repositories/{id}/graph?type=calls
```

---

### Task 2: Frontend - Add React Router

**Install dependencies:**
```bash
cd /root/work/neograph/frontend
npm install react-router-dom
```

**File:** `frontend/src/main.tsx`

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import App from './App.tsx'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
)
```

**File:** `frontend/src/App.tsx`

```tsx
import { Routes, Route } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import RepositoryListPage from './pages/RepositoryListPage'
import RepositoryDetailPage from './pages/RepositoryDetailPage'

const queryClient = new QueryClient()

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div className="min-h-screen bg-gray-50">
        <header className="bg-white border-b">
          <div className="max-w-7xl mx-auto px-4 py-4">
            <h1 className="text-2xl font-bold">NeoGraph</h1>
            <p className="text-gray-500">Code Intelligence with Neo4j</p>
          </div>
        </header>
        <main className="max-w-7xl mx-auto px-4 py-6">
          <Routes>
            <Route path="/" element={<RepositoryListPage />} />
            <Route path="/repository/:id" element={<RepositoryDetailPage />} />
          </Routes>
        </main>
      </div>
    </QueryClientProvider>
  )
}

export default App
```

**File:** `frontend/src/pages/RepositoryListPage.tsx` (NEW)

```tsx
import { RepositoryList } from '@/components/RepositoryList'
import { AddRepositoryForm } from '@/components/AddRepositoryForm'

export default function RepositoryListPage() {
  return (
    <>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold">Repositories</h2>
        <AddRepositoryForm />
      </div>
      <RepositoryList />
    </>
  )
}
```

**File:** `frontend/src/pages/RepositoryDetailPage.tsx` (NEW)

```tsx
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { FileTree } from '@/components/FileTree'
import { GraphVisualization } from '@/components/GraphVisualization'
import { NodeDetailPanel } from '@/components/NodeDetailPanel'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useState } from 'react'

export default function RepositoryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [graphType, setGraphType] = useState<'structure' | 'calls'>('structure')
  const [selectedNode, setSelectedNode] = useState<string | null>(null)

  const { data: repo, isLoading } = useQuery({
    queryKey: ['repository', id],
    queryFn: () => repositoryApi.get(id!),
    enabled: !!id,
  })

  if (isLoading) return <div>Loading...</div>
  if (!repo) return <div>Repository not found</div>

  return (
    <div className="h-[calc(100vh-120px)] flex flex-col">
      <div className="flex items-center gap-4 mb-4">
        <Link to="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> Back
          </Button>
        </Link>
        <h2 className="text-xl font-semibold">{repo.name}</h2>
      </div>

      <div className="flex-1 grid grid-cols-[280px_1fr_300px] gap-4">
        <FileTree repoId={id!} onNodeSelect={setSelectedNode} />
        <GraphVisualization
          repoId={id!}
          type={graphType}
          onTypeChange={setGraphType}
          selectedNode={selectedNode}
          onNodeClick={setSelectedNode}
        />
        <NodeDetailPanel nodeId={selectedNode} repoId={id!} />
      </div>
    </div>
  )
}
```

**Verification:**
```bash
npm run dev
# Navigate to http://localhost:5173/repository/some-id
```

---

### Task 3: Frontend - File Tree Sidebar Component

**File:** `frontend/src/components/FileTree.tsx` (NEW)

```tsx
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { ChevronRight, ChevronDown, FileCode, Box } from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'

interface FileTreeProps {
  repoId: string
  onNodeSelect: (nodeId: string) => void
}

interface FileNode {
  id: string
  path: string
  language: string
  functions: Array<{
    id: string
    name: string
    signature: string
    startLine: number
    endLine: number
  }>
}

export function FileTree({ repoId, onNodeSelect }: FileTreeProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const { data: files, isLoading } = useQuery({
    queryKey: ['repository-files', repoId],
    queryFn: () => repositoryApi.getFiles(repoId),
  })

  const toggleExpand = (fileId: string) => {
    const next = new Set(expanded)
    if (next.has(fileId)) {
      next.delete(fileId)
    } else {
      next.add(fileId)
    }
    setExpanded(next)
  }

  if (isLoading) return <div className="p-4">Loading files...</div>

  return (
    <div className="bg-white rounded-lg border overflow-auto">
      <div className="p-3 border-b font-medium text-sm">Files</div>
      <div className="p-2">
        {files?.map((file: FileNode) => (
          <div key={file.id}>
            <button
              className="flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm"
              onClick={() => {
                toggleExpand(file.id)
                onNodeSelect(file.id)
              }}
            >
              {file.functions.length > 0 ? (
                expanded.has(file.id) ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />
              ) : <span className="w-4" />}
              <FileCode className="w-4 h-4 text-blue-500" />
              <span className="truncate">{file.path.split('/').pop()}</span>
            </button>

            {expanded.has(file.id) && (
              <div className="ml-6">
                {file.functions.map((fn) => (
                  <button
                    key={fn.id}
                    className="flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm"
                    onClick={() => onNodeSelect(fn.id)}
                  >
                    <Box className="w-4 h-4 text-green-500" />
                    <span className="truncate">{fn.name}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
```

**File:** `frontend/src/lib/api.ts` (UPDATE)

Add to repositoryApi:
```typescript
getFiles: async (id: string): Promise<FileNode[]> => {
  const { data } = await api.get(`/api/repositories/${id}/files`)
  return data
},

getGraph: async (id: string, type: 'structure' | 'calls' = 'structure') => {
  const { data } = await api.get(`/api/repositories/${id}/graph?type=${type}`)
  return data
},
```

---

### Task 4: Frontend - Integrate neovis.js for Graph Visualization

**Install neovis.js:**
```bash
cd /root/work/neograph/frontend
npm install neovis.js
```

**File:** `frontend/src/components/GraphVisualization.tsx` (NEW)

```tsx
import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface GraphVisualizationProps {
  repoId: string
  type: 'structure' | 'calls'
  onTypeChange: (type: 'structure' | 'calls') => void
  selectedNode: string | null
  onNodeClick: (nodeId: string) => void
}

export function GraphVisualization({
  repoId,
  type,
  onTypeChange,
  selectedNode,
  onNodeClick,
}: GraphVisualizationProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const vizRef = useRef<any>(null)

  const { data: graphData, isLoading } = useQuery({
    queryKey: ['repository-graph', repoId, type],
    queryFn: () => repositoryApi.getGraph(repoId, type),
  })

  useEffect(() => {
    if (!containerRef.current || !graphData) return

    // Initialize vis.js network
    const nodes = graphData.nodes.map((n: any) => ({
      id: n.id,
      label: n.label,
      color: n.type === 'File' ? '#3b82f6' : '#22c55e',
      shape: n.type === 'File' ? 'box' : 'ellipse',
    }))

    const edges = graphData.edges.map((e: any) => ({
      from: e.source,
      to: e.target,
      arrows: 'to',
      label: e.type,
    }))

    // Use vis-network for visualization
    import('vis-network/standalone').then(({ Network, DataSet }) => {
      const nodesDS = new DataSet(nodes)
      const edgesDS = new DataSet(edges)

      vizRef.current = new Network(
        containerRef.current!,
        { nodes: nodesDS, edges: edgesDS },
        {
          physics: { stabilization: { iterations: 100 } },
          interaction: { hover: true },
        }
      )

      vizRef.current.on('click', (params: any) => {
        if (params.nodes.length > 0) {
          onNodeClick(params.nodes[0])
        }
      })
    })

    return () => {
      vizRef.current?.destroy()
    }
  }, [graphData, onNodeClick])

  // Highlight selected node
  useEffect(() => {
    if (vizRef.current && selectedNode) {
      vizRef.current.selectNodes([selectedNode])
    }
  }, [selectedNode])

  return (
    <div className="bg-white rounded-lg border flex flex-col">
      <div className="p-3 border-b flex items-center justify-between">
        <span className="font-medium text-sm">Graph</span>
        <div className="flex gap-1">
          <Button
            variant={type === 'structure' ? 'default' : 'outline'}
            size="sm"
            onClick={() => onTypeChange('structure')}
          >
            Structure
          </Button>
          <Button
            variant={type === 'calls' ? 'default' : 'outline'}
            size="sm"
            onClick={() => onTypeChange('calls')}
          >
            Calls
          </Button>
        </div>
      </div>
      <div ref={containerRef} className="flex-1 min-h-[400px]">
        {isLoading && <div className="p-4">Loading graph...</div>}
      </div>
    </div>
  )
}
```

---

### Task 5: Frontend - Node Detail Panel

**File:** `frontend/src/components/NodeDetailPanel.tsx` (NEW)

```tsx
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { FileCode, Box, ArrowRight, ArrowLeft } from 'lucide-react'

interface NodeDetailPanelProps {
  nodeId: string | null
  repoId: string
}

export function NodeDetailPanel({ nodeId, repoId }: NodeDetailPanelProps) {
  const { data: nodeDetail, isLoading } = useQuery({
    queryKey: ['node-detail', repoId, nodeId],
    queryFn: () => repositoryApi.getNodeDetail(repoId, nodeId!),
    enabled: !!nodeId,
  })

  if (!nodeId) {
    return (
      <div className="bg-white rounded-lg border p-4 text-gray-500 text-sm">
        Select a node to view details
      </div>
    )
  }

  if (isLoading) return <div className="bg-white rounded-lg border p-4">Loading...</div>

  return (
    <div className="bg-white rounded-lg border overflow-auto">
      <div className="p-3 border-b font-medium text-sm">Details</div>
      <div className="p-3 space-y-4">
        <div>
          <h3 className="text-lg font-medium flex items-center gap-2">
            {nodeDetail?.type === 'File' ? (
              <FileCode className="w-5 h-5 text-blue-500" />
            ) : (
              <Box className="w-5 h-5 text-green-500" />
            )}
            {nodeDetail?.name}
          </h3>
          {nodeDetail?.signature && (
            <code className="text-sm text-gray-600 block mt-1">
              {nodeDetail.signature}
            </code>
          )}
        </div>

        {nodeDetail?.filePath && (
          <div>
            <h4 className="text-sm font-medium text-gray-500">Location</h4>
            <p className="text-sm">
              {nodeDetail.filePath}:{nodeDetail.startLine}-{nodeDetail.endLine}
            </p>
          </div>
        )}

        {nodeDetail?.calls?.length > 0 && (
          <div>
            <h4 className="text-sm font-medium text-gray-500 flex items-center gap-1">
              <ArrowRight className="w-4 h-4" /> Calls
            </h4>
            <ul className="text-sm mt-1 space-y-1">
              {nodeDetail.calls.map((name: string) => (
                <li key={name} className="text-blue-600 hover:underline cursor-pointer">
                  {name}
                </li>
              ))}
            </ul>
          </div>
        )}

        {nodeDetail?.calledBy?.length > 0 && (
          <div>
            <h4 className="text-sm font-medium text-gray-500 flex items-center gap-1">
              <ArrowLeft className="w-4 h-4" /> Called By
            </h4>
            <ul className="text-sm mt-1 space-y-1">
              {nodeDetail.calledBy.map((name: string) => (
                <li key={name} className="text-blue-600 hover:underline cursor-pointer">
                  {name}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
```

---

### Task 6: Make Repository Cards Clickable

**File:** `frontend/src/components/RepositoryList.tsx` (UPDATE)

```tsx
import { Link } from 'react-router-dom'

// In RepositoryCard component, wrap Card with Link:
function RepositoryCard({ repo }: { repo: Repository }) {
  // ... existing code ...

  return (
    <Link to={`/repository/${repo.id}`}>
      <Card className="hover:shadow-md transition-shadow cursor-pointer">
        {/* ... existing card content ... */}
      </Card>
    </Link>
  )
}
```

---

## Phase 2.2: Semantic Search

### Task 7: Backend - TEI Client for Embedding Generation

**File:** `backend/internal/embedding/tei_client.go` (NEW)

```go
package embedding

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type TEIClient struct {
    baseURL    string
    httpClient *http.Client
}

func NewTEIClient(baseURL string) *TEIClient {
    return &TEIClient{
        baseURL:    baseURL,
        httpClient: &http.Client{},
    }
}

type EmbedRequest struct {
    Inputs []string `json:"inputs"`
}

func (c *TEIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    reqBody, err := json.Marshal(EmbedRequest{Inputs: texts})
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewReader(reqBody))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("TEI error: %s", string(body))
    }

    var embeddings [][]float32
    if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
        return nil, err
    }

    return embeddings, nil
}
```

---

### Task 8: Backend - Update Indexer for Embeddings

**File:** `backend/internal/indexer/pipeline.go` (UPDATE)

Add embedding generation to the indexing pipeline:
```go
// In Pipeline struct, add:
teiClient *embedding.TEIClient

// In Index function, after extracting entities:
if p.teiClient != nil {
    for i := range entities {
        // Create embedding text from signature + docstring
        text := entities[i].Signature + " " + entities[i].Docstring
        embeddings, err := p.teiClient.Embed(ctx, []string{text})
        if err != nil {
            log.Printf("Failed to generate embedding for %s: %v", entities[i].Name, err)
            continue
        }
        entities[i].Embedding = embeddings[0]
    }
}
```

**File:** `backend/internal/models/code_entity.go` (UPDATE)

Add embedding field:
```go
type CodeEntity struct {
    // ... existing fields ...
    Embedding []float32 `json:"embedding,omitempty"`
}
```

---

### Task 9: Backend - Create Neo4j Vector Index and Search Endpoints

**File:** `backend/internal/db/vector_index.go` (NEW)

```go
package db

import (
    "context"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Neo4jClient) CreateVectorIndex(ctx context.Context) error {
    _, err := c.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        query := `
            CREATE VECTOR INDEX function_embeddings IF NOT EXISTS
            FOR (f:Function) ON (f.embedding)
            OPTIONS {indexConfig: {
                `+"`"+`vector.dimensions`+"`"+`: 1536,
                `+"`"+`vector.similarity_function`+"`"+`: 'cosine'
            }}
        `
        _, err := tx.Run(ctx, query, nil)
        return nil, err
    })
    return err
}

type SearchResult struct {
    ID         string  `json:"id"`
    Name       string  `json:"name"`
    Signature  string  `json:"signature"`
    FilePath   string  `json:"filePath"`
    RepoID     string  `json:"repoId"`
    RepoName   string  `json:"repoName"`
    Score      float64 `json:"score"`
}

func (r *GraphReader) VectorSearch(ctx context.Context, embedding []float32, limit int, repoID string) ([]SearchResult, error) {
    result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        query := `
            CALL db.index.vector.queryNodes('function_embeddings', $limit, $embedding)
            YIELD node, score
            MATCH (node)<-[:DECLARES]-(f:File)<-[:CONTAINS]-(r:Repository)
            WHERE ($repoId IS NULL OR r.id = $repoId)
            RETURN node.id, node.name, node.signature, node.filePath, r.id, r.name, score
            ORDER BY score DESC
        `
        records, err := tx.Run(ctx, query, map[string]any{
            "embedding": embedding,
            "limit":     limit,
            "repoId":    repoID,
        })
        if err != nil {
            return nil, err
        }

        var results []SearchResult
        for records.Next(ctx) {
            rec := records.Record()
            results = append(results, SearchResult{
                ID:        rec.Values[0].(string),
                Name:      rec.Values[1].(string),
                Signature: rec.Values[2].(string),
                FilePath:  rec.Values[3].(string),
                RepoID:    rec.Values[4].(string),
                RepoName:  rec.Values[5].(string),
                Score:     rec.Values[6].(float64),
            })
        }
        return results, nil
    })
    if err != nil {
        return nil, err
    }
    return result.([]SearchResult), nil
}
```

**File:** `backend/internal/api/routes.go` (UPDATE)

Add search routes:
```go
api.Get("/search", h.GlobalSearch)
repos.Get("/:id/search", h.RepoSearch)
```

---

### Task 10: Frontend - Global Search Page

**File:** `frontend/src/pages/SearchPage.tsx` (NEW)

```tsx
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { searchApi } from '@/lib/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Search, FileCode } from 'lucide-react'

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const [inputValue, setInputValue] = useState(query)

  const { data: results, isLoading } = useQuery({
    queryKey: ['search', query],
    queryFn: () => searchApi.global(query),
    enabled: query.length > 2,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setSearchParams({ q: inputValue })
  }

  return (
    <div className="max-w-4xl mx-auto">
      <form onSubmit={handleSearch} className="flex gap-2 mb-6">
        <Input
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          placeholder="Search code across all repositories..."
          className="flex-1"
        />
        <Button type="submit">
          <Search className="w-4 h-4 mr-1" /> Search
        </Button>
      </form>

      {isLoading && <div>Searching...</div>}

      {results && (
        <div className="space-y-4">
          {results.map((result: any) => (
            <Link
              key={result.id}
              to={`/repository/${result.repoId}?node=${result.id}`}
              className="block p-4 bg-white rounded-lg border hover:shadow-md"
            >
              <div className="flex items-center gap-2">
                <FileCode className="w-4 h-4 text-blue-500" />
                <span className="font-medium">{result.name}</span>
                <span className="text-gray-400 text-sm">in {result.repoName}</span>
              </div>
              <code className="text-sm text-gray-600 block mt-1">{result.signature}</code>
              <p className="text-sm text-gray-500 mt-1">{result.filePath}</p>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
```

---

### Task 11: Frontend - Per-Repo Search with Graph Highlighting

Add search input to FileTree component and highlight matching nodes in the graph.

**File:** `frontend/src/components/FileTree.tsx` (UPDATE)

Add search functionality at the top of the file tree that filters visible files/functions.

---

## Phase 2.3: Claude Agents

### Task 12: Create Python Agent Service

**File:** `agents/pyproject.toml` (NEW)

```toml
[project]
name = "neograph-agents"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = [
    "anthropic>=0.39.0",
    "fastapi>=0.115.0",
    "uvicorn>=0.32.0",
    "neo4j>=5.25.0",
    "pydantic>=2.9.0",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

**File:** `agents/src/server.py` (NEW)

```python
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
from typing import Optional
import anthropic
from .tools import neo4j_tools

app = FastAPI()
app.add_middleware(CORSMiddleware, allow_origins=["*"])

client = anthropic.Anthropic()

class ChatRequest(BaseModel):
    message: str
    repo_id: Optional[str] = None
    agent_type: str = "explorer"

@app.post("/chat")
async def chat(request: ChatRequest):
    tools = neo4j_tools.get_tools()

    response = client.messages.create(
        model="claude-sonnet-4-20250514",
        max_tokens=4096,
        tools=tools,
        messages=[{"role": "user", "content": request.message}],
        system=get_system_prompt(request.agent_type, request.repo_id),
    )

    # Handle tool use and return result
    return {"response": response.content}
```

---

### Task 13: Implement MCP Server with Neo4j Tools

**File:** `agents/src/tools/neo4j_tools.py` (NEW)

```python
from neo4j import GraphDatabase
import os

driver = GraphDatabase.driver(
    os.environ["NEO4J_URI"],
    auth=(os.environ["NEO4J_USER"], os.environ["NEO4J_PASSWORD"])
)

def get_tools():
    return [
        {
            "name": "neo4j_query",
            "description": "Execute a Cypher query against the Neo4j database",
            "input_schema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Cypher query to execute"},
                },
                "required": ["query"],
            },
        },
        {
            "name": "vector_search",
            "description": "Search for similar code using vector embeddings",
            "input_schema": {
                "type": "object",
                "properties": {
                    "query": {"type": "string", "description": "Natural language search query"},
                    "limit": {"type": "integer", "default": 10},
                },
                "required": ["query"],
            },
        },
        {
            "name": "blast_radius",
            "description": "Find all functions that depend on a given function",
            "input_schema": {
                "type": "object",
                "properties": {
                    "function_name": {"type": "string"},
                    "depth": {"type": "integer", "default": 3},
                },
                "required": ["function_name"],
            },
        },
    ]

def execute_tool(name: str, args: dict):
    if name == "neo4j_query":
        return run_cypher(args["query"])
    elif name == "vector_search":
        return vector_search(args["query"], args.get("limit", 10))
    elif name == "blast_radius":
        return find_blast_radius(args["function_name"], args.get("depth", 3))

def run_cypher(query: str):
    with driver.session() as session:
        result = session.run(query)
        return [dict(record) for record in result]
```

---

### Task 14: Backend - Proxy Endpoints for Agent Communication

**File:** `backend/internal/api/routes.go` (UPDATE)

```go
// Add agent proxy routes
api.Post("/agents/chat", h.ProxyAgentChat)
```

**File:** `backend/internal/api/handler.go` (UPDATE)

```go
func (h *Handler) ProxyAgentChat(c fiber.Ctx) error {
    // Forward request to Python agent service
    resp, err := http.Post(
        "http://localhost:8001/chat",
        "application/json",
        bytes.NewReader(c.Body()),
    )
    if err != nil {
        return fiber.NewError(fiber.StatusBadGateway, err.Error())
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    return c.Send(body)
}
```

---

### Task 15: Frontend - Command Bar Component (Cmd+K)

**Install cmdk:**
```bash
npm install cmdk
```

**File:** `frontend/src/components/CommandBar.tsx` (NEW)

```tsx
import { Command } from 'cmdk'
import { useEffect, useState } from 'react'
import { Search } from 'lucide-react'

export function CommandBar() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((open) => !open)
      }
    }
    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  return (
    <Command.Dialog open={open} onOpenChange={setOpen} className="fixed inset-0 z-50">
      <div className="fixed inset-0 bg-black/50" onClick={() => setOpen(false)} />
      <div className="fixed top-1/4 left-1/2 -translate-x-1/2 w-full max-w-xl bg-white rounded-lg shadow-xl">
        <Command.Input
          value={query}
          onValueChange={setQuery}
          placeholder="Ask about code..."
          className="w-full p-4 border-b outline-none"
        />
        <Command.List className="max-h-80 overflow-auto p-2">
          <Command.Empty>No results found.</Command.Empty>
          <Command.Item>Find authentication code</Command.Item>
          <Command.Item>What depends on Process()?</Command.Item>
          <Command.Item>Document this module</Command.Item>
        </Command.List>
      </div>
    </Command.Dialog>
  )
}
```

---

### Task 16: Frontend - Chat Panel Drawer with Streaming

**File:** `frontend/src/components/ChatPanel.tsx` (NEW)

```tsx
import { useState, useRef, useEffect } from 'react'
import { X, Send, MessageSquare } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

interface Message {
  role: 'user' | 'assistant'
  content: string
}

interface ChatPanelProps {
  open: boolean
  onClose: () => void
  repoId?: string
}

export function ChatPanel({ open, onClose, repoId }: ChatPanelProps) {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const sendMessage = async () => {
    if (!input.trim() || isLoading) return

    const userMessage = input
    setInput('')
    setMessages((prev) => [...prev, { role: 'user', content: userMessage }])
    setIsLoading(true)

    try {
      const response = await fetch('/api/agents/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: userMessage,
          repo_id: repoId,
          agent_type: 'explorer',
        }),
      })
      const data = await response.json()
      setMessages((prev) => [...prev, { role: 'assistant', content: data.response }])
    } catch (error) {
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: 'Error: Failed to get response' },
      ])
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div
      className={cn(
        'fixed right-0 top-0 h-full w-96 bg-white shadow-xl transform transition-transform',
        open ? 'translate-x-0' : 'translate-x-full'
      )}
    >
      <div className="flex items-center justify-between p-4 border-b">
        <h3 className="font-medium flex items-center gap-2">
          <MessageSquare className="w-4 h-4" /> Chat
        </h3>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="w-4 h-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-auto p-4 space-y-4 h-[calc(100vh-140px)]">
        {messages.map((msg, i) => (
          <div
            key={i}
            className={cn(
              'p-3 rounded-lg',
              msg.role === 'user' ? 'bg-blue-100 ml-8' : 'bg-gray-100 mr-8'
            )}
          >
            {msg.content}
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      <div className="p-4 border-t flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && sendMessage()}
          placeholder="Ask about code..."
          disabled={isLoading}
        />
        <Button onClick={sendMessage} disabled={isLoading}>
          <Send className="w-4 h-4" />
        </Button>
      </div>
    </div>
  )
}
```

---

### Task 17: Implement Explorer, Analyzer, Doc Writer Agents

**File:** `agents/src/agents/explorer.py` (NEW)

```python
SYSTEM_PROMPT = """You are a code exploration agent with access to a Neo4j graph database.
Your job is to help users find and understand code in their repositories.

Available tools:
- neo4j_query: Run Cypher queries
- vector_search: Semantic code search

When asked to find code:
1. Use vector_search to find relevant functions
2. Use neo4j_query to explore relationships
3. Provide clear explanations with file paths and line numbers
"""

def get_system_prompt(repo_id: str = None):
    base = SYSTEM_PROMPT
    if repo_id:
        base += f"\n\nCurrently exploring repository: {repo_id}"
    return base
```

**File:** `agents/src/agents/analyzer.py` (NEW)

```python
SYSTEM_PROMPT = """You are a code impact analysis agent.
Your job is to analyze dependencies and potential blast radius of changes.

Available tools:
- blast_radius: Find all dependents of a function
- neo4j_query: Custom graph queries

When asked about dependencies:
1. Use blast_radius to find impacted code
2. Categorize by severity (direct vs transitive)
3. Suggest testing priorities
"""
```

**File:** `agents/src/agents/doc_writer.py` (NEW)

```python
SYSTEM_PROMPT = """You are a documentation generation agent.
Your job is to create clear, comprehensive documentation for code.

Available tools:
- neo4j_query: Fetch code structure
- vector_search: Find related code

When asked to document:
1. Fetch the code and its relationships
2. Generate markdown documentation
3. Include examples and usage patterns
"""
```

---

## Verification Checklist

### Phase 2.1
- [ ] Backend endpoints return file tree and graph data
- [ ] React Router navigates between pages
- [ ] File tree shows expandable files with functions
- [ ] Graph visualization renders with nodes and edges
- [ ] Clicking node shows detail panel
- [ ] Toggle between Structure/Calls views works

### Phase 2.2
- [ ] TEI client generates embeddings
- [ ] Indexer stores embeddings in Neo4j
- [ ] Vector index created successfully
- [ ] Global search returns ranked results
- [ ] Per-repo search filters correctly

### Phase 2.3
- [ ] Python agent service starts
- [ ] MCP tools execute against Neo4j
- [ ] Command bar opens with Cmd+K
- [ ] Chat panel streams responses
- [ ] All three agents respond appropriately

---

## Running Commands

```bash
# Backend
cd /root/work/neograph/backend && go build ./... && go test ./...

# Frontend
cd /root/work/neograph/frontend && npm run dev

# Agent Service
cd /root/work/neograph/agents && uvicorn src.server:app --port 8001

# Full E2E
curl http://localhost:3001/api/repositories
curl http://localhost:3001/api/search?q=process
```
