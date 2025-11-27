package api

import (
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(app *fiber.App, h *Handler) {
	api := app.Group("/api")

	// Search endpoints
	api.Get("/search", h.GlobalSearch)

	// Agent proxy endpoints
	agents := api.Group("/agents")
	agents.Post("/chat", h.ProxyAgentChat)

	// Repositories
	repos := api.Group("/repositories")
	repos.Get("/", h.ListRepositories)
	repos.Post("/", h.CreateRepository)
	repos.Get("/:id", h.GetRepository)
	repos.Delete("/:id", h.DeleteRepository)
	repos.Post("/:id/reindex", h.ReindexRepository)
	repos.Get("/:id/files", h.GetRepositoryFiles)
	repos.Get("/:id/graph", h.GetRepositoryGraph)
	repos.Get("/:id/nodes/:nodeId", h.GetNodeDetail)
	repos.Get("/:id/search", h.RepoSearch)

	// Wiki endpoints
	repos.Get("/:id/wiki", h.GetWikiNavigation)
	repos.Get("/:id/wiki/status", h.GetWikiStatus)
	repos.Post("/:id/wiki/generate", h.GenerateWiki)
	repos.Get("/:id/wiki/:slug", h.GetWikiPage)
}
