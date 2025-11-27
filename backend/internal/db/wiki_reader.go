package db

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type WikiReader struct {
	client *Neo4jClient
}

func NewWikiReader(client *Neo4jClient) *WikiReader {
	return &WikiReader{client: client}
}

// pageInfo is internal struct for building navigation
type pageInfo struct {
	Slug       string
	Title      string
	Order      int
	ParentSlug string
}

// GetNavigation returns the wiki navigation tree for a repository
func (r *WikiReader) GetNavigation(ctx context.Context, repoID string) (*models.WikiNavigation, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:HAS_WIKI]->(w:WikiPage)
			RETURN w.slug as slug, w.title as title, w.order as order,
			       w.parentSlug as parentSlug
			ORDER BY w.order
		`
		records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		if err != nil {
			return nil, err
		}

		var pages []pageInfo

		for records.Next(ctx) {
			rec := records.Record()
			slug, _ := rec.Get("slug")
			title, _ := rec.Get("title")
			order, _ := rec.Get("order")
			parentSlug, _ := rec.Get("parentSlug")

			p := pageInfo{
				Slug:  slug.(string),
				Title: title.(string),
				Order: int(order.(int64)),
			}
			if parentSlug != nil {
				p.ParentSlug = parentSlug.(string)
			}
			pages = append(pages, p)
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		return buildNavTree(pages), nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return &models.WikiNavigation{Items: []models.WikiNavItem{}}, nil
	}
	return result.(*models.WikiNavigation), nil
}

func buildNavTree(pages []pageInfo) *models.WikiNavigation {
	// Group by parent
	childrenMap := make(map[string][]models.WikiNavItem)

	for _, p := range pages {
		item := models.WikiNavItem{
			Slug:  p.Slug,
			Title: p.Title,
			Order: p.Order,
		}
		childrenMap[p.ParentSlug] = append(childrenMap[p.ParentSlug], item)
	}

	// Sort children by order
	for key := range childrenMap {
		sort.Slice(childrenMap[key], func(i, j int) bool {
			return childrenMap[key][i].Order < childrenMap[key][j].Order
		})
	}

	// Build tree recursively
	var buildChildren func(parentSlug string) []models.WikiNavItem
	buildChildren = func(parentSlug string) []models.WikiNavItem {
		children := childrenMap[parentSlug]
		for i := range children {
			children[i].Children = buildChildren(children[i].Slug)
		}
		return children
	}

	return &models.WikiNavigation{
		Items: buildChildren(""), // Root items have empty parent
	}
}

// GetPage returns a specific wiki page by slug
func (r *WikiReader) GetPage(ctx context.Context, repoID, slug string) (*models.WikiPageResponse, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:HAS_WIKI]->(w:WikiPage {slug: $slug})
			RETURN w.id as id, w.repoId as repoId, w.slug as slug, w.title as title,
			       w.content as content, w.order as order, w.parentSlug as parentSlug,
			       w.diagrams as diagrams, w.generatedAt as generatedAt
		`
		records, err := tx.Run(ctx, query, map[string]any{
			"repoId": repoID,
			"slug":   slug,
		})
		if err != nil {
			return nil, err
		}

		if !records.Next(ctx) {
			return nil, nil
		}

		rec := records.Record()

		id, _ := rec.Get("id")
		repoId, _ := rec.Get("repoId")
		slugVal, _ := rec.Get("slug")
		title, _ := rec.Get("title")
		content, _ := rec.Get("content")
		order, _ := rec.Get("order")
		parentSlug, _ := rec.Get("parentSlug")
		generatedAt, _ := rec.Get("generatedAt")

		page := &models.WikiPageResponse{
			WikiPage: models.WikiPage{
				ID:      id.(string),
				RepoID:  repoId.(string),
				Slug:    slugVal.(string),
				Title:   title.(string),
				Content: content.(string),
				Order:   int(order.(int64)),
			},
		}

		if parentSlug != nil {
			page.ParentSlug = parentSlug.(string)
		}

		if generatedAt != nil {
			// Handle both time.Time and neo4j.Time
			switch t := generatedAt.(type) {
			case time.Time:
				page.GeneratedAt = t
			case neo4j.Time:
				page.GeneratedAt = t.Time()
			}
		}

		// Parse diagrams from JSON string if stored that way
		diagramsRaw, _ := rec.Get("diagrams")
		if diagramsRaw != nil {
			if diagrams, ok := diagramsRaw.([]any); ok {
				for _, d := range diagrams {
					if dm, ok := d.(map[string]any); ok {
						page.Diagrams = append(page.Diagrams, models.Diagram{
							ID:    dm["id"].(string),
							Title: dm["title"].(string),
							Code:  dm["code"].(string),
						})
					}
				}
			}
		}

		// Generate TOC from content
		page.TableOfContents = extractTOC(page.Content)

		if err := records.Err(); err != nil {
			return nil, err
		}

		return page, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.WikiPageResponse), nil
}

// extractTOC parses markdown headings to build table of contents
func extractTOC(content string) []models.TOCItem {
	var toc []models.TOCItem
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}

		level := 0
		for _, ch := range line {
			if ch == '#' {
				level++
			} else {
				break
			}
		}

		if level > 0 && level <= 6 {
			title := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if title != "" {
				// Create URL-friendly ID
				id := strings.ToLower(title)
				id = strings.ReplaceAll(id, " ", "-")
				id = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(id, "")

				toc = append(toc, models.TOCItem{
					ID:    id,
					Title: title,
					Level: level,
				})
			}
		}
	}

	return toc
}
