package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatRequest represents the request body for chat endpoint
type ChatRequest struct {
	Message   string  `json:"message"`
	RepoID    *string `json:"repo_id,omitempty"`
	AgentType string  `json:"agent_type"`
}

// ChatResponse represents the response from the agent service
type ChatResponse struct {
	Response  string `json:"response"`
	ToolCalls []any  `json:"tool_calls"`
}

// WikiGenerateRequest represents the request body for wiki generation
type WikiGenerateRequest struct {
	RepoID   string `json:"repo_id"`
	RepoName string `json:"repo_name"`
}

// WikiDiagram represents a mermaid diagram
type WikiDiagram struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Code  string `json:"code"`
}

// WikiPageResponse represents a single wiki page from the agent
type WikiPageResponse struct {
	Slug       string        `json:"slug"`
	Title      string        `json:"title"`
	Content    string        `json:"content"`
	Order      int           `json:"order"`
	ParentSlug *string       `json:"parent_slug"`
	Diagrams   []WikiDiagram `json:"diagrams"`
}

// WikiGenerateResponse represents the response from wiki generation
type WikiGenerateResponse struct {
	Pages []WikiPageResponse `json:"pages"`
}

// AgentProxy handles communication with the Python agent service
type AgentProxy struct {
	baseURL    string
	httpClient *http.Client
}

// NewAgentProxy creates a new agent proxy instance
func NewAgentProxy(baseURL string) *AgentProxy {
	return &AgentProxy{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// Chat sends a message to the agent service and returns the response
func (p *AgentProxy) Chat(ctx context.Context, message string, repoID *string, agentType string) (*ChatResponse, error) {
	// Construct request
	reqBody := ChatRequest{
		Message:   message,
		RepoID:    repoID,
		AgentType: agentType,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &chatResp, nil
}

// GenerateWiki calls the agent service to generate wiki pages
func (p *AgentProxy) GenerateWiki(ctx context.Context, repoID, repoName string) (*WikiGenerateResponse, error) {
	// Construct request
	reqBody := WikiGenerateRequest{
		RepoID:   repoID,
		RepoName: repoName,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/wiki/generate", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request with longer timeout for wiki generation (5 minutes for large repos)
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var wikiResp WikiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&wikiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wikiResp, nil
}
