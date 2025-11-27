package models

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWikiPageJSONSerialization tests WikiPage struct JSON marshaling and unmarshaling
func TestWikiPageJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)

	original := WikiPage{
		ID:          "page-123",
		RepoID:      "repo-456",
		Slug:        "getting-started",
		Title:       "Getting Started Guide",
		Content:     "# Getting Started\n\nWelcome to the documentation.",
		Order:       1,
		ParentSlug:  "docs",
		Diagrams:    []Diagram{
			{
				ID:    "diagram-1",
				Title: "Architecture",
				Code:  "graph TD\n  A-->B",
			},
		},
		GeneratedAt: now,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal WikiPage: %v", err)
	}

	// Unmarshal back to struct
	var unmarshaled WikiPage
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WikiPage: %v", err)
	}

	// Verify all fields
	if unmarshaled.ID != original.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, original.ID)
	}
	if unmarshaled.RepoID != original.RepoID {
		t.Errorf("RepoID mismatch: got %v, want %v", unmarshaled.RepoID, original.RepoID)
	}
	if unmarshaled.Slug != original.Slug {
		t.Errorf("Slug mismatch: got %v, want %v", unmarshaled.Slug, original.Slug)
	}
	if unmarshaled.Title != original.Title {
		t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, original.Title)
	}
	if unmarshaled.Content != original.Content {
		t.Errorf("Content mismatch: got %v, want %v", unmarshaled.Content, original.Content)
	}
	if unmarshaled.Order != original.Order {
		t.Errorf("Order mismatch: got %v, want %v", unmarshaled.Order, original.Order)
	}
	if unmarshaled.ParentSlug != original.ParentSlug {
		t.Errorf("ParentSlug mismatch: got %v, want %v", unmarshaled.ParentSlug, original.ParentSlug)
	}
	if len(unmarshaled.Diagrams) != len(original.Diagrams) {
		t.Errorf("Diagrams length mismatch: got %v, want %v", len(unmarshaled.Diagrams), len(original.Diagrams))
	}
	if !unmarshaled.GeneratedAt.Equal(original.GeneratedAt) {
		t.Errorf("GeneratedAt mismatch: got %v, want %v", unmarshaled.GeneratedAt, original.GeneratedAt)
	}
}

// TestWikiPageEmptyDiagrams tests WikiPage with empty diagrams array
func TestWikiPageEmptyDiagrams(t *testing.T) {
	page := WikiPage{
		ID:          "page-1",
		RepoID:      "repo-1",
		Slug:        "test",
		Title:       "Test Page",
		Content:     "Test content",
		Order:       0,
		ParentSlug:  "",
		Diagrams:    []Diagram{},
		GeneratedAt: time.Now(),
	}

	jsonData, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Failed to marshal WikiPage with empty diagrams: %v", err)
	}

	var unmarshaled WikiPage
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WikiPage with empty diagrams: %v", err)
	}

	if unmarshaled.Diagrams == nil {
		t.Error("Diagrams should be empty array, not nil")
	}
	if len(unmarshaled.Diagrams) != 0 {
		t.Errorf("Expected empty diagrams array, got %d items", len(unmarshaled.Diagrams))
	}
}

// TestDiagramJSONSerialization tests Diagram struct JSON marshaling
func TestDiagramJSONSerialization(t *testing.T) {
	original := Diagram{
		ID:    "diag-789",
		Title: "Component Diagram",
		Code:  "graph LR\n  A[Frontend] --> B[Backend]\n  B --> C[Database]",
	}

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Diagram: %v", err)
	}

	var unmarshaled Diagram
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Diagram: %v", err)
	}

	if unmarshaled.ID != original.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, original.ID)
	}
	if unmarshaled.Title != original.Title {
		t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, original.Title)
	}
	if unmarshaled.Code != original.Code {
		t.Errorf("Code mismatch: got %v, want %v", unmarshaled.Code, original.Code)
	}
}

// TestWikiNavItemNestedChildren tests WikiNavItem with nested children serialization
func TestWikiNavItemNestedChildren(t *testing.T) {
	original := WikiNavItem{
		Slug:  "root",
		Title: "Root Section",
		Order: 1,
		Children: []WikiNavItem{
			{
				Slug:  "child-1",
				Title: "First Child",
				Order: 1,
				Children: []WikiNavItem{
					{
						Slug:     "grandchild-1",
						Title:    "First Grandchild",
						Order:    1,
						Children: nil,
					},
				},
			},
			{
				Slug:     "child-2",
				Title:    "Second Child",
				Order:    2,
				Children: []WikiNavItem{},
			},
		},
	}

	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal nested WikiNavItem: %v", err)
	}

	var unmarshaled WikiNavItem
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal nested WikiNavItem: %v", err)
	}

	// Verify root level
	if unmarshaled.Slug != original.Slug {
		t.Errorf("Root Slug mismatch: got %v, want %v", unmarshaled.Slug, original.Slug)
	}
	if len(unmarshaled.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(unmarshaled.Children))
	}

	// Verify first child has grandchild
	if len(unmarshaled.Children[0].Children) != 1 {
		t.Errorf("Expected first child to have 1 grandchild, got %d", len(unmarshaled.Children[0].Children))
	}

	// Verify grandchild
	grandchild := unmarshaled.Children[0].Children[0]
	if grandchild.Slug != "grandchild-1" {
		t.Errorf("Grandchild slug mismatch: got %v, want %v", grandchild.Slug, "grandchild-1")
	}

	// Verify omitempty behavior for empty children
	// The second child should not have "children" field in JSON due to empty array and omitempty
	var rawJSON map[string]interface{}
	json.Unmarshal(jsonData, &rawJSON)
	children := rawJSON["children"].([]interface{})
	secondChild := children[1].(map[string]interface{})
	if _, exists := secondChild["children"]; exists {
		t.Error("Empty children array should be omitted from JSON")
	}
}

// TestWikiNavItemNoChildren tests WikiNavItem with no children (omitempty behavior)
func TestWikiNavItemNoChildren(t *testing.T) {
	navItem := WikiNavItem{
		Slug:     "leaf-node",
		Title:    "Leaf Node",
		Order:    5,
		Children: nil,
	}

	jsonData, err := json.Marshal(navItem)
	if err != nil {
		t.Fatalf("Failed to marshal WikiNavItem with no children: %v", err)
	}

	// Verify that children field is omitted
	var rawJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &rawJSON)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := rawJSON["children"]; exists {
		t.Error("Children field should be omitted when nil (omitempty)")
	}

	// Verify unmarshaling works correctly
	var unmarshaled WikiNavItem
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Slug != navItem.Slug {
		t.Errorf("Slug mismatch: got %v, want %v", unmarshaled.Slug, navItem.Slug)
	}
}

// TestWikiNavigationEmptyItems tests WikiNavigation with empty items array
func TestWikiNavigationEmptyItems(t *testing.T) {
	nav := WikiNavigation{
		Items: []WikiNavItem{},
	}

	jsonData, err := json.Marshal(nav)
	if err != nil {
		t.Fatalf("Failed to marshal empty WikiNavigation: %v", err)
	}

	var unmarshaled WikiNavigation
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty WikiNavigation: %v", err)
	}

	if unmarshaled.Items == nil {
		t.Error("Items should be empty array, not nil")
	}
	if len(unmarshaled.Items) != 0 {
		t.Errorf("Expected empty items array, got %d items", len(unmarshaled.Items))
	}
}

// TestWikiNavigationWithMultipleItems tests WikiNavigation with multiple root items
func TestWikiNavigationWithMultipleItems(t *testing.T) {
	nav := WikiNavigation{
		Items: []WikiNavItem{
			{
				Slug:  "intro",
				Title: "Introduction",
				Order: 1,
			},
			{
				Slug:  "guide",
				Title: "User Guide",
				Order: 2,
				Children: []WikiNavItem{
					{
						Slug:  "installation",
						Title: "Installation",
						Order: 1,
					},
				},
			},
			{
				Slug:  "api",
				Title: "API Reference",
				Order: 3,
			},
		},
	}

	jsonData, err := json.Marshal(nav)
	if err != nil {
		t.Fatalf("Failed to marshal WikiNavigation: %v", err)
	}

	var unmarshaled WikiNavigation
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WikiNavigation: %v", err)
	}

	if len(unmarshaled.Items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(unmarshaled.Items))
	}

	if len(unmarshaled.Items[1].Children) != 1 {
		t.Errorf("Expected second item to have 1 child, got %d", len(unmarshaled.Items[1].Children))
	}
}

// TestTOCItemStructFields tests TOCItem struct fields and JSON serialization
func TestTOCItemStructFields(t *testing.T) {
	tocItem := TOCItem{
		ID:    "heading-introduction",
		Title: "Introduction",
		Level: 1,
	}

	// Test JSON tag names
	jsonData, err := json.Marshal(tocItem)
	if err != nil {
		t.Fatalf("Failed to marshal TOCItem: %v", err)
	}

	var rawJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &rawJSON)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// Verify JSON field names
	if _, exists := rawJSON["id"]; !exists {
		t.Error("Expected 'id' field in JSON")
	}
	if _, exists := rawJSON["title"]; !exists {
		t.Error("Expected 'title' field in JSON")
	}
	if _, exists := rawJSON["level"]; !exists {
		t.Error("Expected 'level' field in JSON")
	}

	// Test unmarshaling
	var unmarshaled TOCItem
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal TOCItem: %v", err)
	}

	if unmarshaled.ID != tocItem.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, tocItem.ID)
	}
	if unmarshaled.Title != tocItem.Title {
		t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, tocItem.Title)
	}
	if unmarshaled.Level != tocItem.Level {
		t.Errorf("Level mismatch: got %v, want %v", unmarshaled.Level, tocItem.Level)
	}
}

// TestTOCItemVariousLevels tests TOCItem with different heading levels
func TestTOCItemVariousLevels(t *testing.T) {
	testCases := []struct {
		level int
		desc  string
	}{
		{1, "h1 heading"},
		{2, "h2 heading"},
		{3, "h3 heading"},
		{4, "h4 heading"},
		{5, "h5 heading"},
		{6, "h6 heading"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			item := TOCItem{
				ID:    "heading-" + tc.desc,
				Title: tc.desc,
				Level: tc.level,
			}

			jsonData, err := json.Marshal(item)
			if err != nil {
				t.Fatalf("Failed to marshal TOCItem: %v", err)
			}

			var unmarshaled TOCItem
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal TOCItem: %v", err)
			}

			if unmarshaled.Level != tc.level {
				t.Errorf("Level mismatch: got %v, want %v", unmarshaled.Level, tc.level)
			}
		})
	}
}

// TestWikiPageResponseEmbedding tests WikiPageResponse embedded WikiPage fields
func TestWikiPageResponseEmbedding(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)

	response := WikiPageResponse{
		WikiPage: WikiPage{
			ID:          "page-100",
			RepoID:      "repo-200",
			Slug:        "api-docs",
			Title:       "API Documentation",
			Content:     "# API\n\n## Endpoints",
			Order:       1,
			ParentSlug:  "",
			Diagrams:    []Diagram{},
			GeneratedAt: now,
		},
		TableOfContents: []TOCItem{
			{
				ID:    "api",
				Title: "API",
				Level: 1,
			},
			{
				ID:    "endpoints",
				Title: "Endpoints",
				Level: 2,
			},
		},
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal WikiPageResponse: %v", err)
	}

	var unmarshaled WikiPageResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WikiPageResponse: %v", err)
	}

	// Verify embedded WikiPage fields
	if unmarshaled.ID != response.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, response.ID)
	}
	if unmarshaled.Slug != response.Slug {
		t.Errorf("Slug mismatch: got %v, want %v", unmarshaled.Slug, response.Slug)
	}

	// Verify TableOfContents
	if len(unmarshaled.TableOfContents) != 2 {
		t.Errorf("Expected 2 TOC items, got %d", len(unmarshaled.TableOfContents))
	}
}

// TestWikiStatusStructFields tests WikiStatus struct fields and JSON serialization
func TestWikiStatusStructFields(t *testing.T) {
	status := WikiStatus{
		Status:       "generating",
		Progress:     45,
		CurrentPage:  "api-reference",
		TotalPages:   10,
		ErrorMessage: "",
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal WikiStatus: %v", err)
	}

	var unmarshaled WikiStatus
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WikiStatus: %v", err)
	}

	if unmarshaled.Status != status.Status {
		t.Errorf("Status mismatch: got %v, want %v", unmarshaled.Status, status.Status)
	}
	if unmarshaled.Progress != status.Progress {
		t.Errorf("Progress mismatch: got %v, want %v", unmarshaled.Progress, status.Progress)
	}
	if unmarshaled.CurrentPage != status.CurrentPage {
		t.Errorf("CurrentPage mismatch: got %v, want %v", unmarshaled.CurrentPage, status.CurrentPage)
	}
	if unmarshaled.TotalPages != status.TotalPages {
		t.Errorf("TotalPages mismatch: got %v, want %v", unmarshaled.TotalPages, status.TotalPages)
	}
	if unmarshaled.ErrorMessage != status.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %v, want %v", unmarshaled.ErrorMessage, status.ErrorMessage)
	}
}

// TestWikiStatusVariousStates tests WikiStatus with different status values
func TestWikiStatusVariousStates(t *testing.T) {
	testCases := []struct {
		name         string
		status       WikiStatus
		expectFields []string
	}{
		{
			name: "pending status",
			status: WikiStatus{
				Status:     "pending",
				Progress:   0,
				TotalPages: 0,
			},
			expectFields: []string{"status", "progress", "totalPages"},
		},
		{
			name: "generating status",
			status: WikiStatus{
				Status:      "generating",
				Progress:    50,
				CurrentPage: "getting-started",
				TotalPages:  20,
			},
			expectFields: []string{"status", "progress", "currentPage", "totalPages"},
		},
		{
			name: "ready status",
			status: WikiStatus{
				Status:     "ready",
				Progress:   100,
				TotalPages: 15,
			},
			expectFields: []string{"status", "progress", "totalPages"},
		},
		{
			name: "error status",
			status: WikiStatus{
				Status:       "error",
				Progress:     0,
				TotalPages:   0,
				ErrorMessage: "Failed to generate documentation",
			},
			expectFields: []string{"status", "progress", "totalPages", "errorMessage"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tc.status)
			if err != nil {
				t.Fatalf("Failed to marshal WikiStatus: %v", err)
			}

			var unmarshaled WikiStatus
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal WikiStatus: %v", err)
			}

			if unmarshaled.Status != tc.status.Status {
				t.Errorf("Status mismatch: got %v, want %v", unmarshaled.Status, tc.status.Status)
			}
			if unmarshaled.Progress != tc.status.Progress {
				t.Errorf("Progress mismatch: got %v, want %v", unmarshaled.Progress, tc.status.Progress)
			}
			if unmarshaled.ErrorMessage != tc.status.ErrorMessage {
				t.Errorf("ErrorMessage mismatch: got %v, want %v", unmarshaled.ErrorMessage, tc.status.ErrorMessage)
			}
		})
	}
}

// TestWikiStatusOmitEmptyFields tests omitempty behavior for optional fields
func TestWikiStatusOmitEmptyFields(t *testing.T) {
	status := WikiStatus{
		Status:     "pending",
		Progress:   0,
		TotalPages: 5,
		// CurrentPage and ErrorMessage are empty
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal WikiStatus: %v", err)
	}

	var rawJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &rawJSON)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	// CurrentPage should be omitted when empty
	if _, exists := rawJSON["currentPage"]; exists {
		t.Error("currentPage should be omitted when empty (omitempty)")
	}

	// ErrorMessage should be omitted when empty
	if _, exists := rawJSON["errorMessage"]; exists {
		t.Error("errorMessage should be omitted when empty (omitempty)")
	}

	// Status, Progress, and TotalPages should always be present
	if _, exists := rawJSON["status"]; !exists {
		t.Error("status field should be present")
	}
	if _, exists := rawJSON["progress"]; !exists {
		t.Error("progress field should be present")
	}
	if _, exists := rawJSON["totalPages"]; !exists {
		t.Error("totalPages field should be present")
	}
}
