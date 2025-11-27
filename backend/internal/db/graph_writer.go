package db

import (
	"context"
	"fmt"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphWriter struct {
	client *Neo4jClient
}

func NewGraphWriter(client *Neo4jClient) *GraphWriter {
	return &GraphWriter{client: client}
}

// WriteIndexResult writes all indexed data to Neo4j
func (w *GraphWriter) WriteIndexResult(ctx context.Context, result *models.IndexResult) error {
	// Write files
	for _, file := range result.Files {
		if err := w.WriteFile(ctx, file); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	// Write entities
	for i := range result.Entities {
		if err := w.WriteEntity(ctx, result.RepoID, &result.Entities[i]); err != nil {
			return fmt.Errorf("failed to write entity %s: %w", result.Entities[i].Name, err)
		}
	}

	// Write call relationships
	for i := range result.Entities {
		if len(result.Entities[i].Calls) > 0 {
			if err := w.WriteCallRelationships(ctx, &result.Entities[i]); err != nil {
				return fmt.Errorf("failed to write calls for %s: %w", result.Entities[i].Name, err)
			}
		}
	}

	// Update repository stats
	return w.UpdateRepositoryStats(ctx, result.RepoID, len(result.Files), result.EntitiesFound)
}

func (w *GraphWriter) WriteFile(ctx context.Context, file *models.File) error {
	file.ID = uuid.New().String()

	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})
			MERGE (f:File {repoId: $repoId, path: $path})
			SET f.id = $id,
			    f.language = $language,
			    f.hash = $hash,
			    f.size = $size
			MERGE (r)-[:CONTAINS]->(f)
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":       file.ID,
			"repoId":   file.RepoID,
			"path":     file.Path,
			"language": file.Language,
			"hash":     file.Hash,
			"size":     file.Size,
		})
		return nil, err
	})

	return err
}

func (w *GraphWriter) WriteEntity(ctx context.Context, repoID string, entity *models.CodeEntity) error {
	entityID := uuid.New().String()

	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Create entity node with appropriate label
		var query string
		params := map[string]any{
			"id":        entityID,
			"name":      entity.Name,
			"signature": entity.Signature,
			"docstring": entity.Docstring,
			"startLine": entity.StartLine,
			"endLine":   entity.EndLine,
			"filePath":  entity.FilePath,
			"repoId":    repoID,
		}

		// Add embedding if available
		if len(entity.Embedding) > 0 {
			params["embedding"] = entity.Embedding
		}

		switch entity.Type {
		case models.EntityFunction:
			if len(entity.Embedding) > 0 {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Function {
						id: $id,
						name: $name,
						signature: $signature,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId,
						embedding: $embedding
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			} else {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Function {
						id: $id,
						name: $name,
						signature: $signature,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			}
		case models.EntityClass:
			if len(entity.Embedding) > 0 {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Class {
						id: $id,
						name: $name,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId,
						embedding: $embedding
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			} else {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Class {
						id: $id,
						name: $name,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			}
		case models.EntityMethod:
			if len(entity.Embedding) > 0 {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Method {
						id: $id,
						name: $name,
						signature: $signature,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId,
						embedding: $embedding
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			} else {
				query = `
					MATCH (f:File {repoId: $repoId, path: $filePath})
					CREATE (e:Method {
						id: $id,
						name: $name,
						signature: $signature,
						docstring: $docstring,
						startLine: $startLine,
						endLine: $endLine,
						filePath: $filePath,
						repoId: $repoId
					})
					CREATE (f)-[:DECLARES]->(e)
				`
			}
		default:
			return nil, nil
		}

		_, err := tx.Run(ctx, query, params)
		return nil, err
	})

	return err
}

func (w *GraphWriter) WriteCallRelationships(ctx context.Context, entity *models.CodeEntity) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, calledName := range entity.Calls {
			query := `
				MATCH (caller:Function|Method {name: $callerName, filePath: $filePath})
				MATCH (callee:Function|Method {name: $calleeName})
				WHERE callee.repoId = caller.repoId
				MERGE (caller)-[:CALLS]->(callee)
			`
			_, err := tx.Run(ctx, query, map[string]any{
				"callerName": entity.Name,
				"filePath":   entity.FilePath,
				"calleeName": calledName,
			})
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	return err
}

func (w *GraphWriter) UpdateRepositoryStats(ctx context.Context, repoID string, filesCount, entitiesCount int) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			SET r.filesCount = $filesCount,
			    r.functionsCount = $entitiesCount,
			    r.status = 'ready'
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":            repoID,
			"filesCount":    filesCount,
			"entitiesCount": entitiesCount,
		})
		return nil, err
	})

	return err
}

// ClearRepository removes all indexed data for a repository
func (w *GraphWriter) ClearRepository(ctx context.Context, repoID string) error {
	_, err := w.client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			OPTIONAL MATCH (r)-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(e)
			DETACH DELETE e, f
		`
		_, err := tx.Run(ctx, query, map[string]any{"id": repoID})
		return nil, err
	})

	return err
}
