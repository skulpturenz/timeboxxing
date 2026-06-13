package semantic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterEmbedder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth header %q", r.Header.Get("Authorization"))
		}

		var req struct {
			Model string `json:"model"`
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "embedding-model" || req.Input != "hello" {
			t.Fatalf("unexpected request: %+v", req)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer server.Close()

	embedder, err := NewOpenRouterEmbedder(OpenRouterConfig{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "embedding-model",
		Dimension: 3,
		Client:    server.Client(),
	})
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}

	embedding, err := embedder.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(embedding) != 3 || embedding[0] != float32(0.1) || embedding[2] != float32(0.3) {
		t.Fatalf("unexpected embedding: %#v", embedding)
	}
}

func TestOpenRouterEmbedderDimensionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer server.Close()

	embedder, err := NewOpenRouterEmbedder(OpenRouterConfig{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "embedding-model",
		Dimension: 3,
		Client:    server.Client(),
	})
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}

	if _, err := embedder.Embed(context.Background(), "hello"); err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}

func TestOpenRouterGenerator(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"answer"}}]}`))
	}))
	defer server.Close()

	generator, err := NewOpenRouterGenerator(OpenRouterConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "gemma-model",
		Client:  server.Client(),
	})
	if err != nil {
		t.Fatalf("create generator: %v", err)
	}

	answer, err := generator.Generate(context.Background(), "question")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if answer != "answer" {
		t.Fatalf("unexpected answer %q", answer)
	}
}
