package db

import (
	"testing"

	"github.com/dpolishuk/neograph/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestExtractTOC(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []models.TOCItem
	}{
		{
			name: "Multiple heading levels",
			content: `# Introduction
This is some content.

## Getting Started
More content here.

### Installation
Details about installation.

#### Prerequisites
Some prerequisites.

##### Advanced Setup
Advanced details.

###### Notes
Final notes.`,
			expected: []models.TOCItem{
				{ID: "introduction", Title: "Introduction", Level: 1},
				{ID: "getting-started", Title: "Getting Started", Level: 2},
				{ID: "installation", Title: "Installation", Level: 3},
				{ID: "prerequisites", Title: "Prerequisites", Level: 4},
				{ID: "advanced-setup", Title: "Advanced Setup", Level: 5},
				{ID: "notes", Title: "Notes", Level: 6},
			},
		},
		{
			name: "Mixed heading levels",
			content: `# Main Title

## Section One

#### Subsection (skipping h3)

## Section Two

# Another Top Level`,
			expected: []models.TOCItem{
				{ID: "main-title", Title: "Main Title", Level: 1},
				{ID: "section-one", Title: "Section One", Level: 2},
				{ID: "subsection-skipping-h3", Title: "Subsection (skipping h3)", Level: 4},
				{ID: "section-two", Title: "Section Two", Level: 2},
				{ID: "another-top-level", Title: "Another Top Level", Level: 1},
			},
		},
		{
			name:     "No headings",
			content:  `This is just regular text.\nNo headings here.\nJust paragraphs.`,
			expected: []models.TOCItem{},
		},
		{
			name: "Headings with special characters",
			content: `# Hello, World!

## API Reference (v2.0)

### User's Guide & Tips

#### Questions? Comments! Feedback@example.com

## Code: The $100 Solution`,
			expected: []models.TOCItem{
				{ID: "hello-world", Title: "Hello, World!", Level: 1},
				{ID: "api-reference-v20", Title: "API Reference (v2.0)", Level: 2},
				{ID: "users-guide--tips", Title: "User's Guide & Tips", Level: 3},
				{ID: "questions-comments-feedbackexamplecom", Title: "Questions? Comments! Feedback@example.com", Level: 4},
				{ID: "code-the-100-solution", Title: "Code: The $100 Solution", Level: 2},
			},
		},
		{
			name: "Headings mid-line should not match",
			content: `This is a paragraph with # in the middle.
And another line with text before # Heading.
Also this: some text # More text.

# Valid Heading
This is content.
Not a heading: ## invalid`,
			expected: []models.TOCItem{
				{ID: "valid-heading", Title: "Valid Heading", Level: 1},
			},
		},
		{
			name: "Empty headings ignored",
			content: `#
##
###
#
##
# Valid Heading`,
			expected: []models.TOCItem{
				{ID: "valid-heading", Title: "Valid Heading", Level: 1},
			},
		},
		{
			name: "Headings with extra spaces",
			content: `#     Introduction

##   Getting Started

###      Installation`,
			expected: []models.TOCItem{
				{ID: "introduction", Title: "Introduction", Level: 1},
				{ID: "getting-started", Title: "Getting Started", Level: 2},
				{ID: "installation", Title: "Installation", Level: 3},
			},
		},
		{
			name: "Headings with numbers",
			content: `# Chapter 1: Introduction

## Section 2.1

### 3.1.4 Subsection`,
			expected: []models.TOCItem{
				{ID: "chapter-1-introduction", Title: "Chapter 1: Introduction", Level: 1},
				{ID: "section-21", Title: "Section 2.1", Level: 2},
				{ID: "314-subsection", Title: "3.1.4 Subsection", Level: 3},
			},
		},
		{
			name: "Headings with unicode characters",
			content: `# Über uns

## Café Menu

### 日本語タイトル`,
			expected: []models.TOCItem{
				{ID: "ber-uns", Title: "Über uns", Level: 1},
				{ID: "caf-menu", Title: "Café Menu", Level: 2},
				{ID: "", Title: "日本語タイトル", Level: 3},
			},
		},
		{
			name: "More than 6 hashes should be ignored",
			content: `# Valid H1

####### Invalid (7 hashes)

## Valid H2`,
			expected: []models.TOCItem{
				{ID: "valid-h1", Title: "Valid H1", Level: 1},
				{ID: "valid-h2", Title: "Valid H2", Level: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTOC(tt.content)
			// Handle nil vs empty slice comparison
			if len(tt.expected) == 0 && len(result) == 0 {
				return
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildNavTree(t *testing.T) {
	tests := []struct {
		name     string
		pages    []pageInfo
		expected *models.WikiNavigation
	}{
		{
			name:  "Empty pages",
			pages: []pageInfo{},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{},
			},
		},
		{
			name: "Flat list - no parents",
			pages: []pageInfo{
				{Slug: "intro", Title: "Introduction", Order: 1, ParentSlug: ""},
				{Slug: "guide", Title: "Guide", Order: 2, ParentSlug: ""},
				{Slug: "api", Title: "API Reference", Order: 3, ParentSlug: ""},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{Slug: "intro", Title: "Introduction", Order: 1, Children: []models.WikiNavItem{}},
					{Slug: "guide", Title: "Guide", Order: 2, Children: []models.WikiNavItem{}},
					{Slug: "api", Title: "API Reference", Order: 3, Children: []models.WikiNavItem{}},
				},
			},
		},
		{
			name: "Nested hierarchy - one level",
			pages: []pageInfo{
				{Slug: "guide", Title: "Guide", Order: 1, ParentSlug: ""},
				{Slug: "guide-install", Title: "Installation", Order: 1, ParentSlug: "guide"},
				{Slug: "guide-config", Title: "Configuration", Order: 2, ParentSlug: "guide"},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{
						Slug:  "guide",
						Title: "Guide",
						Order: 1,
						Children: []models.WikiNavItem{
							{Slug: "guide-install", Title: "Installation", Order: 1, Children: []models.WikiNavItem{}},
							{Slug: "guide-config", Title: "Configuration", Order: 2, Children: []models.WikiNavItem{}},
						},
					},
				},
			},
		},
		{
			name: "Nested hierarchy - multiple levels",
			pages: []pageInfo{
				{Slug: "root", Title: "Root", Order: 1, ParentSlug: ""},
				{Slug: "child1", Title: "Child 1", Order: 1, ParentSlug: "root"},
				{Slug: "child2", Title: "Child 2", Order: 2, ParentSlug: "root"},
				{Slug: "grandchild1", Title: "Grandchild 1", Order: 1, ParentSlug: "child1"},
				{Slug: "grandchild2", Title: "Grandchild 2", Order: 2, ParentSlug: "child1"},
				{Slug: "greatgrandchild", Title: "Great Grandchild", Order: 1, ParentSlug: "grandchild1"},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{
						Slug:  "root",
						Title: "Root",
						Order: 1,
						Children: []models.WikiNavItem{
							{
								Slug:  "child1",
								Title: "Child 1",
								Order: 1,
								Children: []models.WikiNavItem{
									{
										Slug:  "grandchild1",
										Title: "Grandchild 1",
										Order: 1,
										Children: []models.WikiNavItem{
											{Slug: "greatgrandchild", Title: "Great Grandchild", Order: 1, Children: []models.WikiNavItem{}},
										},
									},
									{Slug: "grandchild2", Title: "Grandchild 2", Order: 2, Children: []models.WikiNavItem{}},
								},
							},
							{Slug: "child2", Title: "Child 2", Order: 2, Children: []models.WikiNavItem{}},
						},
					},
				},
			},
		},
		{
			name: "Sorting by order",
			pages: []pageInfo{
				{Slug: "page3", Title: "Third Page", Order: 30, ParentSlug: ""},
				{Slug: "page1", Title: "First Page", Order: 10, ParentSlug: ""},
				{Slug: "page2", Title: "Second Page", Order: 20, ParentSlug: ""},
				{Slug: "child-b", Title: "Child B", Order: 20, ParentSlug: "page1"},
				{Slug: "child-a", Title: "Child A", Order: 10, ParentSlug: "page1"},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{
						Slug:  "page1",
						Title: "First Page",
						Order: 10,
						Children: []models.WikiNavItem{
							{Slug: "child-a", Title: "Child A", Order: 10, Children: []models.WikiNavItem{}},
							{Slug: "child-b", Title: "Child B", Order: 20, Children: []models.WikiNavItem{}},
						},
					},
					{Slug: "page2", Title: "Second Page", Order: 20, Children: []models.WikiNavItem{}},
					{Slug: "page3", Title: "Third Page", Order: 30, Children: []models.WikiNavItem{}},
				},
			},
		},
		{
			name: "Mixed root and nested pages",
			pages: []pageInfo{
				{Slug: "intro", Title: "Introduction", Order: 1, ParentSlug: ""},
				{Slug: "guide", Title: "Guide", Order: 2, ParentSlug: ""},
				{Slug: "guide-start", Title: "Getting Started", Order: 1, ParentSlug: "guide"},
				{Slug: "api", Title: "API", Order: 3, ParentSlug: ""},
				{Slug: "api-auth", Title: "Authentication", Order: 1, ParentSlug: "api"},
				{Slug: "api-endpoints", Title: "Endpoints", Order: 2, ParentSlug: "api"},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{Slug: "intro", Title: "Introduction", Order: 1, Children: []models.WikiNavItem{}},
					{
						Slug:  "guide",
						Title: "Guide",
						Order: 2,
						Children: []models.WikiNavItem{
							{Slug: "guide-start", Title: "Getting Started", Order: 1, Children: []models.WikiNavItem{}},
						},
					},
					{
						Slug:  "api",
						Title: "API",
						Order: 3,
						Children: []models.WikiNavItem{
							{Slug: "api-auth", Title: "Authentication", Order: 1, Children: []models.WikiNavItem{}},
							{Slug: "api-endpoints", Title: "Endpoints", Order: 2, Children: []models.WikiNavItem{}},
						},
					},
				},
			},
		},
		{
			name: "Same order values - stable sort",
			pages: []pageInfo{
				{Slug: "page-a", Title: "Page A", Order: 10, ParentSlug: ""},
				{Slug: "page-b", Title: "Page B", Order: 10, ParentSlug: ""},
				{Slug: "page-c", Title: "Page C", Order: 10, ParentSlug: ""},
			},
			expected: &models.WikiNavigation{
				Items: []models.WikiNavItem{
					{Slug: "page-a", Title: "Page A", Order: 10, Children: []models.WikiNavItem{}},
					{Slug: "page-b", Title: "Page B", Order: 10, Children: []models.WikiNavItem{}},
					{Slug: "page-c", Title: "Page C", Order: 10, Children: []models.WikiNavItem{}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNavTree(tt.pages)
			assertNavTreeEqual(t, tt.expected, result)
		})
	}
}

// assertNavTreeEqual compares two WikiNavigation structures, handling nil vs empty slices
func assertNavTreeEqual(t *testing.T, expected, actual *models.WikiNavigation) {
	t.Helper()

	if expected == nil && actual == nil {
		return
	}

	if expected == nil || actual == nil {
		t.Errorf("One tree is nil: expected=%v, actual=%v", expected, actual)
		return
	}

	assertNavItemsEqual(t, expected.Items, actual.Items)
}

// assertNavItemsEqual compares two slices of WikiNavItem, handling nil vs empty slices
func assertNavItemsEqual(t *testing.T, expected, actual []models.WikiNavItem) {
	t.Helper()

	// Handle nil vs empty slice
	if len(expected) == 0 && len(actual) == 0 {
		return
	}

	if len(expected) != len(actual) {
		t.Errorf("Different lengths: expected=%d, actual=%d\nExpected: %+v\nActual: %+v",
			len(expected), len(actual), expected, actual)
		return
	}

	for i := range expected {
		if expected[i].Slug != actual[i].Slug {
			t.Errorf("Item %d: Slug mismatch: expected=%s, actual=%s", i, expected[i].Slug, actual[i].Slug)
		}
		if expected[i].Title != actual[i].Title {
			t.Errorf("Item %d: Title mismatch: expected=%s, actual=%s", i, expected[i].Title, actual[i].Title)
		}
		if expected[i].Order != actual[i].Order {
			t.Errorf("Item %d: Order mismatch: expected=%d, actual=%d", i, expected[i].Order, actual[i].Order)
		}

		// Recursively check children
		assertNavItemsEqual(t, expected[i].Children, actual[i].Children)
	}
}
