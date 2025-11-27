package models

import "time"

// WikiPage represents a generated documentation page
type WikiPage struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repoId"`
	Slug        string    `json:"slug"`       // URL-friendly identifier
	Title       string    `json:"title"`
	Content     string    `json:"content"`    // Markdown content
	Order       int       `json:"order"`      // Navigation order
	ParentSlug  string    `json:"parentSlug"` // For nested navigation (empty = root)
	Diagrams    []Diagram `json:"diagrams"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Diagram represents a Mermaid diagram
type Diagram struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"` // Mermaid syntax
}

// WikiNavItem represents a navigation tree item
type WikiNavItem struct {
	Slug     string        `json:"slug"`
	Title    string        `json:"title"`
	Order    int           `json:"order"`
	Children []WikiNavItem `json:"children,omitempty"`
}

// WikiNavigation is the full navigation tree
type WikiNavigation struct {
	Items []WikiNavItem `json:"items"`
}

// TOCItem represents a table of contents entry
type TOCItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Level int    `json:"level"` // h1=1, h2=2, etc.
}

// WikiPageResponse is the API response for a wiki page
type WikiPageResponse struct {
	WikiPage
	TableOfContents []TOCItem `json:"tableOfContents"`
}

// WikiStatus represents generation progress
type WikiStatus struct {
	Status       string `json:"status"`             // pending, generating, ready, error
	Progress     int    `json:"progress"`           // 0-100
	CurrentPage  string `json:"currentPage,omitempty"`
	TotalPages   int    `json:"totalPages"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}
