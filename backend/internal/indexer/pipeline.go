package indexer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dpolishuk/neograph/backend/internal/db"
	"github.com/dpolishuk/neograph/backend/internal/embedding"
	"github.com/dpolishuk/neograph/backend/internal/models"
)

type Pipeline struct {
	dbClient  *db.Neo4jClient
	extractor *Extractor
	teiClient *embedding.TEIClient
}

func NewPipeline(dbClient *db.Neo4jClient) *Pipeline {
	return &Pipeline{
		dbClient:  dbClient,
		extractor: NewExtractor(),
		teiClient: nil, // Optional, set with SetTEIClient
	}
}

// SetTEIClient optionally enables embedding generation
func (p *Pipeline) SetTEIClient(client *embedding.TEIClient) {
	p.teiClient = client
}

func (p *Pipeline) Close() {
	p.extractor.Close()
}

func (p *Pipeline) IndexDirectory(ctx context.Context, dirPath, repoID string) (*models.IndexResult, error) {
	result := &models.IndexResult{
		RepoID: repoID,
	}

	// Walk directory and find supported files
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common non-code directories
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "__pycache__" || name == ".venv" || name == "dist" ||
				name == "build" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file is supported
		relPath, _ := filepath.Rel(dirPath, path)
		lang := models.DetectLanguage(path)
		if lang != "" {
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Process files sequentially to avoid tree-sitter CGO concurrency issues
	for _, relPath := range files {
		fullPath := filepath.Join(dirPath, relPath)
		file, entities, err := p.processFile(ctx, fullPath, relPath, repoID)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", relPath, err))
			continue
		}

		result.FilesProcessed++
		result.Files = append(result.Files, file)
		result.Entities = append(result.Entities, entities...)
		result.EntitiesFound += len(entities)
	}

	// Generate embeddings for all entities if TEIClient is available
	if p.teiClient != nil && len(result.Entities) > 0 {
		if err := p.generateEmbeddings(ctx, result.Entities); err != nil {
			log.Printf("Warning: failed to generate embeddings: %v", err)
			// Don't fail the entire indexing if embeddings fail
		}
	}

	return result, nil
}

func (p *Pipeline) processFile(ctx context.Context, fullPath, relPath, repoID string) (*models.File, []models.CodeEntity, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	lang := models.DetectLanguage(relPath)

	file := &models.File{
		RepoID:   repoID,
		Path:     relPath,
		Language: lang,
		Size:     info.Size(),
		Hash:     hashContent(content),
	}

	// Extract code entities
	entities, err := p.extractor.Extract(ctx, content, lang, relPath)
	if err != nil {
		return file, nil, fmt.Errorf("extraction failed: %w", err)
	}

	return file, entities, nil
}

func hashContent(content []byte) string {
	// Simple hash for change detection
	var h uint64 = 5381
	for _, b := range content {
		h = ((h << 5) + h) + uint64(b)
	}
	return fmt.Sprintf("%x", h)
}

// generateEmbeddings generates embeddings for entities in batches
func (p *Pipeline) generateEmbeddings(ctx context.Context, entities []models.CodeEntity) error {
	const batchSize = 32

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]

		// Prepare embedding texts
		texts := make([]string, len(batch))
		for j, entity := range batch {
			// Create embedding text from: signature + " " + docstring + " " + name
			text := entity.Signature
			if entity.Docstring != "" {
				text += " " + entity.Docstring
			}
			text += " " + entity.Name
			texts[j] = text
		}

		// Generate embeddings
		embeddings, err := p.teiClient.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("failed to generate embeddings for batch %d-%d: %w", i, end, err)
		}

		// Store embeddings back in entities
		for j, embedding := range embeddings {
			entities[i+j].Embedding = embedding
		}

		log.Printf("Generated embeddings for entities %d-%d", i, end)
	}

	return nil
}
