package api

import (
	"context"

	"github.com/dpolishuk/neograph/backend/internal/config"
	"github.com/dpolishuk/neograph/backend/internal/db"
	"github.com/dpolishuk/neograph/backend/internal/git"
	"github.com/dpolishuk/neograph/backend/internal/indexer"
	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	cfg      *config.Config
	dbClient *db.Neo4jClient
	gitSvc   *git.GitService
	pipeline *indexer.Pipeline
	writer   *db.GraphWriter
}

func NewHandler(cfg *config.Config, dbClient *db.Neo4jClient) *Handler {
	return &Handler{
		cfg:      cfg,
		dbClient: dbClient,
		gitSvc:   git.NewGitService(cfg.ReposPath),
		pipeline: indexer.NewPipeline(dbClient),
		writer:   db.NewGraphWriter(dbClient),
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
