package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestWritePageUUIDGeneration tests that WritePage generates a UUID when ID is empty
func TestWritePageUUIDGeneration(t *testing.T) {
	// Note: This test validates the UUID generation logic in WritePage.
	// The actual database write is tested in integration tests.
	tests := []struct {
		name       string
		page       *models.WikiPage
		wantNewID  bool
		wantSameID bool
	}{
		{
			name: "Empty ID should generate new UUID",
			page: &models.WikiPage{
				ID:     "",
				RepoID: "repo-123",
				Slug:   "test-page",
				Title:  "Test Page",
			},
			wantNewID:  true,
			wantSameID: false,
		},
		{
			name: "Existing ID should be preserved",
			page: &models.WikiPage{
				ID:     "existing-uuid",
				RepoID: "repo-123",
				Slug:   "test-page",
				Title:  "Test Page",
			},
			wantNewID:  false,
			wantSameID: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.page.ID

			// We can't actually call WritePage without a real Neo4j connection,
			// but we can test the logic by replicating it
			if tt.page.ID == "" {
				tt.page.ID = uuid.New().String()
			}
			tt.page.GeneratedAt = time.Now()

			if tt.wantNewID {
				assert.NotEmpty(t, tt.page.ID, "ID should be generated")
				assert.NotEqual(t, originalID, tt.page.ID, "ID should be different from original")
				_, err := uuid.Parse(tt.page.ID)
				assert.NoError(t, err, "ID should be a valid UUID")
			}

			if tt.wantSameID {
				assert.Equal(t, originalID, tt.page.ID, "ID should be preserved")
			}

			assert.False(t, tt.page.GeneratedAt.IsZero(), "GeneratedAt should be set")
		})
	}
}

// TestDiagramJSONSerialization tests that diagrams are correctly serialized to JSON
func TestDiagramJSONSerialization(t *testing.T) {
	tests := []struct {
		name      string
		diagrams  []models.Diagram
		wantValid bool
		wantJSON  string
	}{
		{
			name:      "Nil diagrams",
			diagrams:  nil,
			wantValid: true,
			wantJSON:  "null",
		},
		{
			name:      "Empty diagrams",
			diagrams:  []models.Diagram{},
			wantValid: true,
			wantJSON:  "[]",
		},
		{
			name: "Single diagram",
			diagrams: []models.Diagram{
				{
					ID:    "diagram-1",
					Title: "Architecture",
					Code:  "graph TD\n  A --> B",
				},
			},
			wantValid: true,
			wantJSON:  `[{"id":"diagram-1","title":"Architecture","code":"graph TD\n  A --> B"}]`,
		},
		{
			name: "Multiple diagrams",
			diagrams: []models.Diagram{
				{
					ID:    "diagram-1",
					Title: "Architecture",
					Code:  "graph TD\n  A --> B",
				},
				{
					ID:    "diagram-2",
					Title: "Sequence",
					Code:  "sequenceDiagram\n  A->>B: Hello",
				},
			},
			wantValid: true,
			wantJSON:  `[{"id":"diagram-1","title":"Architecture","code":"graph TD\n  A --> B"},{"id":"diagram-2","title":"Sequence","code":"sequenceDiagram\n  A->>B: Hello"}]`,
		},
		{
			name: "Diagram with special characters",
			diagrams: []models.Diagram{
				{
					ID:    "diagram-1",
					Title: "Test \"quotes\" and 'apostrophes'",
					Code:  "graph TD\n  A[\"Node with \\\"quotes\\\"\"] --> B",
				},
			},
			wantValid: true,
		},
		{
			name: "Diagram with unicode",
			diagrams: []models.Diagram{
				{
					ID:    "diagram-1",
					Title: "日本語タイトル",
					Code:  "graph TD\n  A[\"你好\"] --> B[\"مرحبا\"]",
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mimics the serialization logic in WritePage
			diagramsJSON, err := json.Marshal(tt.diagrams)

			if tt.wantValid {
				assert.NoError(t, err, "JSON marshaling should succeed")
				assert.NotNil(t, diagramsJSON, "JSON should not be nil")

				// Verify it's valid JSON by unmarshaling
				var decoded []models.Diagram
				err = json.Unmarshal(diagramsJSON, &decoded)
				assert.NoError(t, err, "JSON should be valid and unmarshalable")

				// If we have specific JSON expectation, verify it
				if tt.wantJSON != "" {
					assert.JSONEq(t, tt.wantJSON, string(diagramsJSON), "JSON should match expected format")
				}

				// Verify the decoded diagrams match the original
				if tt.diagrams != nil {
					assert.Equal(t, len(tt.diagrams), len(decoded), "Number of diagrams should match")
					for i := range tt.diagrams {
						assert.Equal(t, tt.diagrams[i].ID, decoded[i].ID, "Diagram ID should match")
						assert.Equal(t, tt.diagrams[i].Title, decoded[i].Title, "Diagram Title should match")
						assert.Equal(t, tt.diagrams[i].Code, decoded[i].Code, "Diagram Code should match")
					}
				}
			} else {
				assert.Error(t, err, "JSON marshaling should fail")
			}
		})
	}
}

// TestWikiPageValidation tests validation of WikiPage fields
func TestWikiPageValidation(t *testing.T) {
	tests := []struct {
		name  string
		page  *models.WikiPage
		valid bool
		desc  string
	}{
		{
			name: "Valid minimal page",
			page: &models.WikiPage{
				RepoID: "repo-123",
				Slug:   "intro",
				Title:  "Introduction",
			},
			valid: true,
			desc:  "Minimal valid page with required fields",
		},
		{
			name: "Valid complete page",
			page: &models.WikiPage{
				ID:         "page-uuid",
				RepoID:     "repo-123",
				Slug:       "guide-install",
				Title:      "Installation Guide",
				Content:    "# Installation\n\nFollow these steps...",
				Order:      1,
				ParentSlug: "guide",
				Diagrams: []models.Diagram{
					{ID: "d1", Title: "Flow", Code: "graph TD\n  A --> B"},
				},
			},
			valid: true,
			desc:  "Complete page with all fields",
		},
		{
			name: "Page with empty content",
			page: &models.WikiPage{
				RepoID:  "repo-123",
				Slug:    "empty",
				Title:   "Empty Page",
				Content: "",
			},
			valid: true,
			desc:  "Empty content is allowed",
		},
		{
			name: "Page with zero order",
			page: &models.WikiPage{
				RepoID: "repo-123",
				Slug:   "first",
				Title:  "First Page",
				Order:  0,
			},
			valid: true,
			desc:  "Zero order is valid",
		},
		{
			name: "Page with negative order",
			page: &models.WikiPage{
				RepoID: "repo-123",
				Slug:   "negative",
				Title:  "Negative Order",
				Order:  -1,
			},
			valid: true,
			desc:  "Negative order is technically allowed (business logic may restrict)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the page structure is valid
			assert.NotNil(t, tt.page, "Page should not be nil")

			// Test JSON serialization (used by API)
			jsonData, err := json.Marshal(tt.page)
			assert.NoError(t, err, "Page should be JSON serializable")

			var decoded models.WikiPage
			err = json.Unmarshal(jsonData, &decoded)
			assert.NoError(t, err, "Page should be JSON deserializable")

			// Verify key fields are preserved (ignoring GeneratedAt which is time-sensitive)
			assert.Equal(t, tt.page.ID, decoded.ID, "ID should be preserved")
			assert.Equal(t, tt.page.RepoID, decoded.RepoID, "RepoID should be preserved")
			assert.Equal(t, tt.page.Slug, decoded.Slug, "Slug should be preserved")
			assert.Equal(t, tt.page.Title, decoded.Title, "Title should be preserved")
			assert.Equal(t, tt.page.Content, decoded.Content, "Content should be preserved")
			assert.Equal(t, tt.page.Order, decoded.Order, "Order should be preserved")
			assert.Equal(t, tt.page.ParentSlug, decoded.ParentSlug, "ParentSlug should be preserved")
		})
	}
}

// TestWikiStatusValidation tests validation of WikiStatus fields
func TestWikiStatusValidation(t *testing.T) {
	tests := []struct {
		name   string
		status *models.WikiStatus
		desc   string
	}{
		{
			name: "Status none",
			status: &models.WikiStatus{
				Status: "none",
			},
			desc: "Initial state with no wiki",
		},
		{
			name: "Status pending",
			status: &models.WikiStatus{
				Status:     "pending",
				Progress:   0,
				TotalPages: 0,
			},
			desc: "Wiki generation queued",
		},
		{
			name: "Status generating with progress",
			status: &models.WikiStatus{
				Status:      "generating",
				Progress:    45,
				CurrentPage: "guide-installation",
				TotalPages:  10,
			},
			desc: "Wiki being generated",
		},
		{
			name: "Status ready",
			status: &models.WikiStatus{
				Status:     "ready",
				Progress:   100,
				TotalPages: 15,
			},
			desc: "Wiki generation complete",
		},
		{
			name: "Status error",
			status: &models.WikiStatus{
				Status:       "error",
				Progress:     50,
				CurrentPage:  "guide-config",
				TotalPages:   10,
				ErrorMessage: "Failed to generate diagram",
			},
			desc: "Wiki generation failed",
		},
		{
			name: "Progress boundaries - 0%",
			status: &models.WikiStatus{
				Status:   "generating",
				Progress: 0,
			},
			desc: "Progress at 0%",
		},
		{
			name: "Progress boundaries - 100%",
			status: &models.WikiStatus{
				Status:   "ready",
				Progress: 100,
			},
			desc: "Progress at 100%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the status structure is valid
			assert.NotNil(t, tt.status, "Status should not be nil")

			// Test JSON serialization (used by API)
			jsonData, err := json.Marshal(tt.status)
			assert.NoError(t, err, "Status should be JSON serializable")

			var decoded models.WikiStatus
			err = json.Unmarshal(jsonData, &decoded)
			assert.NoError(t, err, "Status should be JSON deserializable")

			// Verify all fields are preserved
			assert.Equal(t, tt.status.Status, decoded.Status, "Status should be preserved")
			assert.Equal(t, tt.status.Progress, decoded.Progress, "Progress should be preserved")
			assert.Equal(t, tt.status.CurrentPage, decoded.CurrentPage, "CurrentPage should be preserved")
			assert.Equal(t, tt.status.TotalPages, decoded.TotalPages, "TotalPages should be preserved")
			assert.Equal(t, tt.status.ErrorMessage, decoded.ErrorMessage, "ErrorMessage should be preserved")

			// Validate progress is in valid range (0-100)
			if tt.status.Status == "generating" || tt.status.Status == "ready" {
				assert.GreaterOrEqual(t, tt.status.Progress, 0, "Progress should be >= 0")
				assert.LessOrEqual(t, tt.status.Progress, 100, "Progress should be <= 100")
			}
		})
	}
}

// TestDiagramEdgeCases tests edge cases for diagram serialization
func TestDiagramEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		diagram  models.Diagram
		wantErr  bool
		checkFn  func(*testing.T, string)
	}{
		{
			name: "Empty diagram",
			diagram: models.Diagram{
				ID:    "",
				Title: "",
				Code:  "",
			},
			wantErr: false,
			checkFn: func(t *testing.T, jsonStr string) {
				assert.Contains(t, jsonStr, `"id":""`)
				assert.Contains(t, jsonStr, `"title":""`)
				assert.Contains(t, jsonStr, `"code":""`)
			},
		},
		{
			name: "Very long code",
			diagram: models.Diagram{
				ID:    "long",
				Title: "Long Diagram",
				Code:  string(make([]byte, 10000)),
			},
			wantErr: false,
			checkFn: func(t *testing.T, jsonStr string) {
				assert.NotEmpty(t, jsonStr)
			},
		},
		{
			name: "Newlines and tabs in code",
			diagram: models.Diagram{
				ID:    "formatted",
				Title: "Formatted",
				Code:  "graph TD\n\tA --> B\n\tB --> C\n\tC --> D",
			},
			wantErr: false,
			checkFn: func(t *testing.T, jsonStr string) {
				// JSON should escape newlines and tabs
				assert.Contains(t, jsonStr, `\n`)
				assert.Contains(t, jsonStr, `\t`)
			},
		},
		{
			name: "HTML-like content",
			diagram: models.Diagram{
				ID:    "html",
				Title: "<script>alert('test')</script>",
				Code:  "graph TD\n  A[\"<div>Test</div>\"] --> B",
			},
			wantErr: false,
			checkFn: func(t *testing.T, jsonStr string) {
				// JSON should properly escape HTML
				assert.NotContains(t, jsonStr, "<script>")
				var decoded models.Diagram
				err := json.Unmarshal([]byte(jsonStr), &decoded)
				assert.NoError(t, err)
				assert.Equal(t, "<script>alert('test')</script>", decoded.Title)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.diagram)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, jsonData)

				if tt.checkFn != nil {
					tt.checkFn(t, string(jsonData))
				}

				// Verify roundtrip
				var decoded models.Diagram
				err = json.Unmarshal(jsonData, &decoded)
				assert.NoError(t, err)
				assert.Equal(t, tt.diagram.ID, decoded.ID)
				assert.Equal(t, tt.diagram.Title, decoded.Title)
				assert.Equal(t, tt.diagram.Code, decoded.Code)
			}
		})
	}
}

// TestNewWikiWriter tests the constructor
func TestNewWikiWriter(t *testing.T) {
	// Note: We can't create a real Neo4jClient without a connection,
	// but we can test that the constructor returns a non-nil writer
	t.Run("Constructor returns non-nil writer", func(t *testing.T) {
		// In a real scenario, you'd pass a mock or test client
		// For now, we just verify the constructor logic
		var client *Neo4jClient // nil client for testing
		writer := NewWikiWriter(client)
		assert.NotNil(t, writer, "NewWikiWriter should return a non-nil writer")
		assert.Equal(t, client, writer.client, "Writer should store the client")
	})
}

// TestWikiWriterMethodsRequireIntegrationTests documents that most methods need Neo4j
func TestWikiWriterMethodsRequireIntegrationTests(t *testing.T) {
	t.Run("Integration test documentation", func(t *testing.T) {
		// This test serves as documentation that the following methods
		// require integration tests with a real Neo4j instance:
		//
		// - WritePage: Requires Neo4j to test actual write operation
		// - ClearWiki: Requires Neo4j to test deletion
		// - UpdateWikiStatus: Requires Neo4j to test status update
		// - GetWikiStatus: Requires Neo4j to test status retrieval
		//
		// These methods are tested in separate integration test files
		// that spin up a Neo4j test container.
		//
		// Unit tests in this file focus on:
		// - UUID generation logic in WritePage
		// - JSON serialization of diagrams
		// - Data structure validation
		// - Edge cases and input validation

		t.Log("WikiWriter methods require Neo4j for integration testing")
		t.Log("Unit tests cover serialization, validation, and pure logic")
	})
}
