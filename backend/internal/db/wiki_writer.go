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
