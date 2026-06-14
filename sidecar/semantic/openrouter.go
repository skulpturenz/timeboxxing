package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenRouterConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	Dimension int
	Client    *http.Client
}

type OpenRouterEmbedder struct {
	apiKey    string
	baseURL   string
	model     string
	dimension int
	client    *http.Client
}

func NewOpenRouterEmbedder(cfg OpenRouterConfig) (*OpenRouterEmbedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "nvidia/llama-nemotron-embed-vl-1b-v2:free"
	}
	if cfg.Dimension <= 0 {
		return nil, fmt.Errorf("embedding dimension must be positive")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 30 * time.Second}
	}

	return &OpenRouterEmbedder{
		apiKey:    cfg.APIKey,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		model:     cfg.Model,
		dimension: cfg.Dimension,
		client:    cfg.Client,
	}, nil
}

func (e *OpenRouterEmbedder) Model() string { return e.model }

func (e *OpenRouterEmbedder) Dimension() int { return e.dimension }

func (e *OpenRouterEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	var out struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	body := map[string]any{
		"model": e.model,
		"input": input,
	}
	if err := e.post(ctx, "/embeddings", body, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("openrouter embeddings response contained no data")
	}
	if len(out.Data[0].Embedding) != e.dimension {
		return nil, fmt.Errorf("openrouter embedding dimension mismatch: got %d, want %d", len(out.Data[0].Embedding), e.dimension)
	}

	embedding := make([]float32, len(out.Data[0].Embedding))
	for i, v := range out.Data[0].Embedding {
		embedding[i] = float32(v)
	}
	return embedding, nil
}

type OpenRouterGenerator struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenRouterGenerator(cfg OpenRouterConfig) (*OpenRouterGenerator, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "google/gemma-3-27b-it:free"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenRouterGenerator{
		apiKey:  cfg.APIKey,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		model:   cfg.Model,
		client:  cfg.Client,
	}, nil
}

func (g *OpenRouterGenerator) Model() string { return g.model }

func (g *OpenRouterGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	body := map[string]any{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if err := g.post(ctx, "/chat/completions", body, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openrouter chat response contained no choices")
	}
	return out.Choices[0].Message.Content, nil
}

func (e *OpenRouterEmbedder) post(ctx context.Context, endpoint string, body any, out any) error {
	return postJSON(ctx, e.client, e.baseURL+endpoint, e.apiKey, body, out)
}

func (g *OpenRouterGenerator) post(ctx context.Context, endpoint string, body any, out any) error {
	return postJSON(ctx, g.client, g.baseURL+endpoint, g.apiKey, body, out)
}

func postJSON(ctx context.Context, client *http.Client, url string, apiKey string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal openrouter request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create openrouter request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send openrouter request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read openrouter response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("openrouter request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode openrouter response: %w", err)
	}

	return nil
}
