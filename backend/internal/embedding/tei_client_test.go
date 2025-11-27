package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewTEIClient(t *testing.T) {
	client := NewTEIClient("http://localhost:8080")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("expected baseURL http://localhost:8080, got %s", client.baseURL)
	}
	if client.httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}
}

func TestEmbed_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/embed" {
			t.Errorf("expected /embed, got %s", r.URL.Path)
		}

		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}

		// Decode and verify request body
		var req EmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}
		if len(req.Inputs) != 2 {
			t.Errorf("expected 2 inputs, got %d", len(req.Inputs))
		}

		// Return mock embeddings
		mockEmbeddings := [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockEmbeddings)
	}))
	defer server.Close()

	client := NewTEIClient(server.URL)
	embeddings, err := client.Embed(context.Background(), []string{"text1", "text2"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 2 {
		t.Errorf("expected 2 embeddings, got %d", len(embeddings))
	}
	if len(embeddings[0]) != 3 {
		t.Errorf("expected embedding dimension 3, got %d", len(embeddings[0]))
	}
	if embeddings[0][0] != 0.1 {
		t.Errorf("expected first value 0.1, got %f", embeddings[0][0])
	}
}

func TestEmbed_EmptyInput(t *testing.T) {
	client := NewTEIClient("http://localhost:8080")
	embeddings, err := client.Embed(context.Background(), []string{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embeddings) != 0 {
		t.Errorf("expected 0 embeddings, got %d", len(embeddings))
	}
}

func TestEmbed_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := NewTEIClient(server.URL)
	_, err := client.Embed(context.Background(), []string{"text1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "TEI error (status 500)"
	if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
	}
}

func TestEmbed_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewTEIClient(server.URL)
	_, err := client.Embed(context.Background(), []string{"text1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "failed to decode response"
	if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
	}
}

func TestEmbed_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewTEIClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Embed(ctx, []string{"text1"})

	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

func TestEmbed_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewTEIClient("http://invalid-host-that-does-not-exist:9999")
	_, err := client.Embed(context.Background(), []string{"text1"})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expectedMsg := "failed to send request"
	if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
	}
}
