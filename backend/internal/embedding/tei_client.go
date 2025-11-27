package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TEIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTEIClient(baseURL string) *TEIClient {
	return &TEIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type EmbedRequest struct {
	Inputs []string `json:"inputs"`
}

func (c *TEIClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	reqBody, err := json.Marshal(EmbedRequest{Inputs: texts})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TEI error (status %d): %s", resp.StatusCode, string(body))
	}

	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embeddings, nil
}
