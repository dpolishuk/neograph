package api

import (
	"context"
	"fmt"

	"github.com/dpolishuk/neograph/backend/internal/agent"
	"github.com/dpolishuk/neograph/backend/internal/config"
	"github.com/dpolishuk/neograph/backend/internal/db"
	"github.com/dpolishuk/neograph/backend/internal/embedding"
	"github.com/dpolishuk/neograph/backend/internal/git"
	"github.com/dpolishuk/neograph/backend/internal/indexer"
	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	cfg         *config.Config
	dbClient    *db.Neo4jClient
	gitSvc      *git.GitService
	pipeline    *indexer.Pipeline
	writer      *db.GraphWriter
	graphReader *db.GraphReader
	wikiReader  *db.WikiReader
	wikiWriter  *db.WikiWriter
	teiClient   *embedding.TEIClient
	agentProxy  *agent.AgentProxy
}

func NewHandler(cfg *config.Config, dbClient *db.Neo4jClient) *Handler {
	return &Handler{
		cfg:         cfg,
		dbClient:    dbClient,
		gitSvc:      git.NewGitService(cfg.ReposPath),
		pipeline:    indexer.NewPipeline(dbClient),
		writer:      db.NewGraphWriter(dbClient),
		graphReader: db.NewGraphReader(dbClient),
		wikiReader:  db.NewWikiReader(dbClient),
		wikiWriter:  db.NewWikiWriter(dbClient),
		teiClient:   embedding.NewTEIClient(cfg.TEI_URL),
		agentProxy:  agent.NewAgentProxy(cfg.AgentURL),
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
	if repos == nil {
		repos = []*models.Repository{}
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
		DefaultBranch: input.DefaultBranch,
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

// GetRepositoryFiles returns file tree with functions for a repository
func (h *Handler) GetRepositoryFiles(c fiber.Ctx) error {
	id := c.Params("id")
	files, err := h.graphReader.GetFileTree(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if files == nil {
		files = []db.FileNode{}
	}
	return c.JSON(files)
}

// GetRepositoryGraph returns graph data for visualization
func (h *Handler) GetRepositoryGraph(c fiber.Ctx) error {
	id := c.Params("id")
	graphType := c.Query("type", "structure") // "structure" or "calls"

	// Validate graph type
	if graphType != "structure" && graphType != "calls" {
		return c.Status(400).JSON(fiber.Map{"error": "invalid graph type, must be 'structure' or 'calls'"})
	}

	graph, err := h.graphReader.GetGraph(c.Context(), id, graphType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(graph)
}

// GetNodeDetail returns detailed information about a specific node
func (h *Handler) GetNodeDetail(c fiber.Ctx) error {
	repoID := c.Params("id")
	nodeID := c.Params("nodeId")

	nodeDetail, err := h.graphReader.GetNodeDetail(c.Context(), repoID, nodeID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if nodeDetail == nil {
		return c.Status(404).JSON(fiber.Map{"error": "node not found"})
	}
	return c.JSON(nodeDetail)
}

// GlobalSearch performs semantic search across all repositories
func (h *Handler) GlobalSearch(c fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}

	// Get optional limit parameter
	limit := fiber.Query[int](c, "limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Generate embedding for the query
	embeddings, err := h.teiClient.Embed(c.Context(), []string{query})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate embedding: " + err.Error()})
	}

	if len(embeddings) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "no embedding generated"})
	}

	// Search Neo4j vector index (empty repoID means search all repos)
	results, err := h.graphReader.VectorSearch(c.Context(), embeddings[0], limit, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "search failed: " + err.Error()})
	}

	if results == nil {
		results = []db.SearchResult{}
	}

	return c.JSON(results)
}

// RepoSearch performs semantic search within a specific repository
func (h *Handler) RepoSearch(c fiber.Ctx) error {
	repoID := c.Params("id")
	query := c.Query("q")

	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": "query parameter 'q' is required"})
	}

	// Get optional limit parameter
	limit := fiber.Query[int](c, "limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Generate embedding for the query
	embeddings, err := h.teiClient.Embed(c.Context(), []string{query})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate embedding: " + err.Error()})
	}

	if len(embeddings) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "no embedding generated"})
	}

	// Search Neo4j vector index filtered by repository
	results, err := h.graphReader.VectorSearch(c.Context(), embeddings[0], limit, repoID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "search failed: " + err.Error()})
	}

	if results == nil {
		results = []db.SearchResult{}
	}

	return c.JSON(results)
}

// ProxyAgentChat forwards chat requests to the Python agent service
func (h *Handler) ProxyAgentChat(c fiber.Ctx) error {
	var req agent.ChatRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Validate required fields
	if req.Message == "" {
		return c.Status(400).JSON(fiber.Map{"error": "message is required"})
	}
	if req.AgentType == "" {
		req.AgentType = "explorer" // Default agent type
	}

	// Forward to agent service
	response, err := h.agentProxy.Chat(c.Context(), req.Message, req.RepoID, req.AgentType)
	if err != nil {
		return c.Status(502).JSON(fiber.Map{"error": "failed to communicate with agent service: " + err.Error()})
	}

	return c.JSON(response)
}

// GetWikiNavigation returns the wiki navigation tree
func (h *Handler) GetWikiNavigation(c fiber.Ctx) error {
	id := c.Params("id")
	nav, err := h.wikiReader.GetNavigation(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(nav)
}

// GetWikiPage returns a specific wiki page by slug
func (h *Handler) GetWikiPage(c fiber.Ctx) error {
	repoID := c.Params("id")
	slug := c.Params("slug")

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

// generateWikiPages generates all wiki pages for a repository (placeholder implementation)
func (h *Handler) generateWikiPages(repo *models.Repository) {
	ctx := context.Background()

	// Helper to set error status
	setError := func(msg string) {
		status := &models.WikiStatus{
			Status:       "error",
			Progress:     0,
			ErrorMessage: msg,
		}
		h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, status)
	}

	// Clear existing wiki
	if err := h.wikiWriter.ClearWiki(ctx, repo.ID); err != nil {
		setError("failed to clear existing wiki: " + err.Error())
		return
	}

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
	if err := h.wikiWriter.WritePage(ctx, overviewPage); err != nil {
		setError("failed to write overview page: " + err.Error())
		return
	}

	// Update status to ready
	status := &models.WikiStatus{
		Status:     "ready",
		Progress:   100,
		TotalPages: 1,
	}
	h.wikiWriter.UpdateWikiStatus(ctx, repo.ID, status)
}
