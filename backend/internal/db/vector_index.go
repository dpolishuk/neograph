package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateVectorIndex creates a vector index for function embeddings
func (c *Neo4jClient) CreateVectorIndex(ctx context.Context) error {
	_, err := c.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CREATE VECTOR INDEX function_embeddings IF NOT EXISTS
			FOR (f:Function) ON (f.embedding)
			OPTIONS {indexConfig: {
				` + "`" + `vector.dimensions` + "`" + `: 1536,
				` + "`" + `vector.similarity_function` + "`" + `: 'cosine'
			}}
		`
		_, err := tx.Run(ctx, query, nil)
		return nil, err
	})
	return err
}

// SearchResult represents a single search result
type SearchResult struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Signature string  `json:"signature"`
	FilePath  string  `json:"filePath"`
	RepoID    string  `json:"repoId"`
	RepoName  string  `json:"repoName"`
	Score     float64 `json:"score"`
}

// VectorSearch performs semantic search using vector embeddings
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

		// Prepare parameters
		params := map[string]any{
			"embedding": embedding,
			"limit":     limit,
		}

		// Handle optional repoId filter
		if repoID == "" {
			params["repoId"] = nil
		} else {
			params["repoId"] = repoID
		}

		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, fmt.Errorf("failed to run vector search query: %w", err)
		}

		var results []SearchResult
		for records.Next(ctx) {
			rec := records.Record()

			// Extract values safely
			id, _ := rec.Get("node.id")
			name, _ := rec.Get("node.name")
			signature, _ := rec.Get("node.signature")
			filePath, _ := rec.Get("node.filePath")
			repoID, _ := rec.Get("r.id")
			repoName, _ := rec.Get("r.name")
			score, _ := rec.Get("score")

			result := SearchResult{
				ID:        fmt.Sprintf("%v", id),
				Name:      fmt.Sprintf("%v", name),
				Signature: fmt.Sprintf("%v", signature),
				FilePath:  fmt.Sprintf("%v", filePath),
				RepoID:    fmt.Sprintf("%v", repoID),
				RepoName:  fmt.Sprintf("%v", repoName),
				Score:     0.0,
			}

			// Handle score conversion
			if score != nil {
				switch v := score.(type) {
				case float64:
					result.Score = v
				case int64:
					result.Score = float64(v)
				}
			}

			results = append(results, result)
		}

		if err := records.Err(); err != nil {
			return nil, fmt.Errorf("error iterating search results: %w", err)
		}

		return results, nil
	})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return []SearchResult{}, nil
	}

	return result.([]SearchResult), nil
}
