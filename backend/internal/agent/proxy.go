package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
