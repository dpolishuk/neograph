package db

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphReader_GetFileTree(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := setupTestNeo4j(t)
	defer client.Close()

	// Create test data
	repoID := setupTestRepository(t, ctx, client)
	defer cleanupTestRepository(t, ctx, client, repoID)

	reader := NewGraphReader(client)

	// Test getting file tree
	files, err := reader.GetFileTree(ctx, repoID)
	require.NoError(t, err)
	assert.NotNil(t, files)

	// Verify we get files
	if len(files) > 0 {
		// Check first file structure
		file := files[0]
		assert.NotEmpty(t, file.ID)
		assert.NotEmpty(t, file.Path)
		assert.NotEmpty(t, file.Language)
		assert.NotNil(t, file.Functions)
	}
}

func TestGraphReader_GetGraph_Structure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := setupTestNeo4j(t)
	defer client.Close()

	// Create test data
	repoID := setupTestRepository(t, ctx, client)
	defer cleanupTestRepository(t, ctx, client, repoID)

	reader := NewGraphReader(client)

	// Test getting structure graph
	graph, err := reader.GetGraph(ctx, repoID, "structure")
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify graph structure
	assert.NotNil(t, graph.Nodes)
	assert.NotNil(t, graph.Edges)

	// Nodes should include both files and functions
	if len(graph.Nodes) > 0 {
		node := graph.Nodes[0]
		assert.NotEmpty(t, node.ID)
		assert.NotEmpty(t, node.Label)
		assert.NotEmpty(t, node.Type)
		assert.Contains(t, []string{"File", "Function"}, node.Type)
	}

	// Edges should be DECLARES relationships
	if len(graph.Edges) > 0 {
		edge := graph.Edges[0]
		assert.NotEmpty(t, edge.ID)
		assert.NotEmpty(t, edge.Source)
		assert.NotEmpty(t, edge.Target)
		assert.Equal(t, "DECLARES", edge.Type)
	}
}

func TestGraphReader_GetGraph_Calls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := setupTestNeo4j(t)
	defer client.Close()

	// Create test data
	repoID := setupTestRepository(t, ctx, client)
	defer cleanupTestRepository(t, ctx, client, repoID)

	reader := NewGraphReader(client)

	// Test getting calls graph
	graph, err := reader.GetGraph(ctx, repoID, "calls")
	require.NoError(t, err)
	require.NotNil(t, graph)

	// Verify graph structure
	assert.NotNil(t, graph.Nodes)
	assert.NotNil(t, graph.Edges)

	// All nodes should be functions in calls graph
	for _, node := range graph.Nodes {
		assert.Equal(t, "Function", node.Type)
	}

	// Edges should be CALLS relationships
	for _, edge := range graph.Edges {
		assert.Equal(t, "CALLS", edge.Type)
	}
}

func TestGraphReader_EmptyRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	client := setupTestNeo4j(t)
	defer client.Close()

	// Create empty repository
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `CREATE (r:Repository {id: $id, name: $name}) RETURN r`
		_, err := tx.Run(ctx, query, map[string]any{
			"id":   "test-empty",
			"name": "Empty Test Repo",
		})
		return nil, err
	})
	require.NoError(t, err)
	defer cleanupTestRepository(t, ctx, client, "test-empty")

	reader := NewGraphReader(client)

	// Test empty file tree
	files, err := reader.GetFileTree(ctx, "test-empty")
	require.NoError(t, err)
	assert.Empty(t, files)

	// Test empty structure graph
	graph, err := reader.GetGraph(ctx, "test-empty", "structure")
	require.NoError(t, err)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)

	// Test empty calls graph
	graph, err = reader.GetGraph(ctx, "test-empty", "calls")
	require.NoError(t, err)
	assert.Empty(t, graph.Nodes)
	assert.Empty(t, graph.Edges)
}

// Helper functions for test setup

func setupTestNeo4j(t *testing.T) *Neo4jClient {
	t.Helper()

	cfg := Neo4jConfig{
		URI:      getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnvOrDefault("NEO4J_USER", "neo4j"),
		Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
	}

	client, err := NewNeo4jClient(context.Background(), cfg)
	require.NoError(t, err)

	return client
}

func setupTestRepository(t *testing.T, ctx context.Context, client *Neo4jClient) string {
	t.Helper()

	repoID := "test-repo-" + t.Name()

	// Create repository with some test files and functions
	_, err := client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CREATE (r:Repository {id: $repoId, name: $name})
			CREATE (f1:File {id: $file1Id, repoId: $repoId, path: 'main.go', language: 'go'})
			CREATE (f2:File {id: $file2Id, repoId: $repoId, path: 'utils.go', language: 'go'})
			CREATE (fn1:Function {id: $fn1Id, name: 'main', signature: 'func main()', filePath: 'main.go', startLine: 5, endLine: 10, repoId: $repoId})
			CREATE (fn2:Function {id: $fn2Id, name: 'helper', signature: 'func helper()', filePath: 'utils.go', startLine: 3, endLine: 7, repoId: $repoId})
			CREATE (r)-[:CONTAINS]->(f1)
			CREATE (r)-[:CONTAINS]->(f2)
			CREATE (f1)-[:DECLARES]->(fn1)
			CREATE (f2)-[:DECLARES]->(fn2)
			CREATE (fn1)-[:CALLS]->(fn2)
		`
		_, err := tx.Run(ctx, query, map[string]any{
			"repoId":  repoID,
			"name":    "Test Repository",
			"file1Id": "file1",
			"file2Id": "file2",
			"fn1Id":   "fn1",
			"fn2Id":   "fn2",
		})
		return nil, err
	})
	require.NoError(t, err)

	return repoID
}

func cleanupTestRepository(t *testing.T, ctx context.Context, client *Neo4jClient, repoID string) {
	t.Helper()

	_, _ = client.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})
			OPTIONAL MATCH (r)-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(fn)
			DETACH DELETE r, f, fn
		`
		_, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		return nil, err
	})
}

func getEnvOrDefault(key, defaultValue string) string {
	// For testing, we'll use defaults
	// In real implementation, use os.Getenv
	return defaultValue
}
