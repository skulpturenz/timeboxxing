package semantic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["model"] != "embedding-model" || req["input"] != "hello" || req["encoding_format"] != "float" {
			t.Fatalf("unexpected request: %+v", req)
		}
		if _, ok := req["dimensions"]; ok {
			t.Fatalf("request should not force dimensions: %+v", req)
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

func TestOpenRouterEmbedderPadsShorterVectorToStoreDimension(t *testing.T) {
	values := make([]float64, 3072)
	values[0] = 0.1
	values[len(values)-1] = 0.9
	response, err := json.Marshal(map[string]any{
		"data": []map[string]any{{"embedding": values}},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(response)
	}))
	defer server.Close()

	embedder, err := NewOpenRouterEmbedder(OpenRouterConfig{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "embedding-model",
		Dimension: StoreEmbeddingDimension,
		Client:    server.Client(),
	})
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}

	embedding, err := embedder.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(embedding) != StoreEmbeddingDimension {
		t.Fatalf("expected padded vector length %d, got %d", StoreEmbeddingDimension, len(embedding))
	}
	if embedding[0] != float32(0.1) || embedding[3071] != float32(0.9) || embedding[3072] != 0 {
		t.Fatalf("unexpected padded embedding values around boundary: %#v %#v %#v", embedding[0], embedding[3071], embedding[3072])
	}
}

func TestOpenRouterEmbedderDimensionMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3,0.4]}]}`))
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

func TestOpenRouterGeneratorToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := req["tools"]; !ok {
			t.Fatalf("expected tools in request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"id":"call-1","type":"function","function":{"name":"get_app_usage_totals","arguments":"{\"started_at\":\"2026-06-13T00:00:00Z\"}"}}]}}]}`))
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

	response, err := generator.GenerateWithTools(context.Background(), []ChatMessage{{Role: ChatRoleUser, Content: "question"}}, []ChatTool{{
		Name:        "get_app_usage_totals",
		Description: "usage",
		Parameters:  map[string]any{"type": "object"},
	}})
	if err != nil {
		t.Fatalf("generate with tools: %v", err)
	}
	if len(response.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %#v", response.ToolCalls)
	}
	if response.ToolCalls[0].ID != "call-1" || response.ToolCalls[0].Name != "get_app_usage_totals" {
		t.Fatalf("unexpected tool call %#v", response.ToolCalls[0])
	}
	if !strings.Contains(response.ToolCalls[0].Arguments, "started_at") {
		t.Fatalf("unexpected tool arguments %q", response.ToolCalls[0].Arguments)
	}
}

func TestOpenRouterErrorsExposeSafeUserMessages(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		retryAfter string
		body       string
		want       string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"No auth credentials found"}}`,
			want:       "OpenRouter API key is invalid or expired.",
		},
		{
			name:       "credits",
			statusCode: http.StatusPaymentRequired,
			body:       `{"error":{"message":"Insufficient credits"}}`,
			want:       "OpenRouter account has insufficient credits.",
		},
		{
			name:       "model unavailable",
			statusCode: http.StatusNotFound,
			body:       `{"error":{"message":"No endpoints found for model qwen/foo","metadata":{"error_type":"model_not_found"}}}`,
			want:       "Selected OpenRouter model is unavailable or the request is invalid: No endpoints found for model qwen/foo",
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			retryAfter: "75",
			body:       `{"error":{"message":"Rate limit exceeded"}}`,
			want:       "OpenRouter rate limit exceeded. Try again in 2 minutes.",
		},
		{
			name:       "provider overload",
			statusCode: 529,
			body:       `{"error":{"message":"Provider overloaded"}}`,
			want:       "OpenRouter provider is temporarily unavailable.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.retryAfter != "" {
					w.Header().Set("Retry-After", tt.retryAfter)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
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

			_, err = embedder.Embed(context.Background(), "hello")
			if err == nil {
				t.Fatal("expected error")
			}
			message, ok := AIRequestUserMessage(err)
			if !ok {
				t.Fatalf("expected AI request error, got %T %v", err, err)
			}
			if message != tt.want {
				t.Fatalf("expected message %q, got %q", tt.want, message)
			}
			if strings.Contains(message, "test-key") {
				t.Fatalf("message leaked secret: %q", message)
			}
		})
	}
}
