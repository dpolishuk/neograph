package db

import (
	"context"
	"testing"
)

func TestNewNeo4jClient(t *testing.T) {
	// This test requires Neo4j running
	// Skip in CI without Neo4j
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := Neo4jConfig{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "neograph_password",
	}

	client, err := NewNeo4jClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Verify connection
	err = client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Failed to ping: %v", err)
	}
}
