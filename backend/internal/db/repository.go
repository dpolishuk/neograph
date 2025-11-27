package db

import (
	"context"
	"fmt"
	"time"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func CreateRepository(ctx context.Context, client *Neo4jClient, repo *models.Repository) (*models.Repository, error) {
	repo.ID = uuid.New().String()

	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CREATE (r:Repository {
				id: $id,
				url: $url,
				name: $name,
				defaultBranch: $defaultBranch,
				status: $status,
				lastIndexed: $lastIndexed,
				filesCount: 0,
				functionsCount: 0
			})
			RETURN r
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":            repo.ID,
			"url":           repo.URL,
			"name":          repo.Name,
			"defaultBranch": repo.DefaultBranch,
			"status":        repo.Status,
			"lastIndexed":   time.Now().UTC(),
		})
		return nil, err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	return repo, nil
}

func GetRepository(ctx context.Context, client *Neo4jClient, id string) (*models.Repository, error) {
	result, err := client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			RETURN r.id AS id, r.url AS url, r.name AS name,
			       r.defaultBranch AS defaultBranch, r.status AS status,
			       r.lastIndexed AS lastIndexed, r.filesCount AS filesCount,
			       r.functionsCount AS functionsCount
		`
		result, err := tx.Run(ctx, query, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}

		if result.Next(ctx) {
			record := result.Record()
			return recordToRepository(record), nil
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*models.Repository), nil
}

func ListRepositories(ctx context.Context, client *Neo4jClient) ([]*models.Repository, error) {
	result, err := client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository)
			RETURN r.id AS id, r.url AS url, r.name AS name,
			       r.defaultBranch AS defaultBranch, r.status AS status,
			       r.lastIndexed AS lastIndexed, r.filesCount AS filesCount,
			       r.functionsCount AS functionsCount
			ORDER BY r.lastIndexed DESC
		`
		result, err := tx.Run(ctx, query, nil)
		if err != nil {
			return nil, err
		}

		var repos []*models.Repository
		for result.Next(ctx) {
			repos = append(repos, recordToRepository(result.Record()))
		}
		return repos, result.Err()
	})

	if err != nil {
		return nil, err
	}
	return result.([]*models.Repository), nil
}

func UpdateRepositoryStatus(ctx context.Context, client *Neo4jClient, id, status string) error {
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $id})
			SET r.status = $status, r.lastIndexed = $lastIndexed
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":          id,
			"status":      status,
			"lastIndexed": time.Now().UTC(),
		})
		return nil, err
	})
	return err
}

func DeleteRepository(ctx context.Context, client *Neo4jClient, id string) error {
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Delete all related nodes first
		query := `
			MATCH (r:Repository {id: $id})
			OPTIONAL MATCH (r)-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(e)
			DETACH DELETE e, f, r
		`
		_, err := tx.Run(ctx, query, map[string]any{"id": id})
		return nil, err
	})
	return err
}

func recordToRepository(record *neo4j.Record) *models.Repository {
	repo := &models.Repository{}

	if id, ok := record.Get("id"); ok && id != nil {
		repo.ID = id.(string)
	}
	if url, ok := record.Get("url"); ok && url != nil {
		repo.URL = url.(string)
	}
	if name, ok := record.Get("name"); ok && name != nil {
		repo.Name = name.(string)
	}
	if branch, ok := record.Get("defaultBranch"); ok && branch != nil {
		repo.DefaultBranch = branch.(string)
	}
	if status, ok := record.Get("status"); ok && status != nil {
		repo.Status = status.(string)
	}
	if lastIndexed, ok := record.Get("lastIndexed"); ok && lastIndexed != nil {
		if t, ok := lastIndexed.(time.Time); ok {
			repo.LastIndexed = t
		}
	}
	if filesCount, ok := record.Get("filesCount"); ok && filesCount != nil {
		repo.FilesCount = int(filesCount.(int64))
	}
	if functionsCount, ok := record.Get("functionsCount"); ok && functionsCount != nil {
		repo.FunctionsCount = int(functionsCount.(int64))
	}

	return repo
}
