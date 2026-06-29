package semantic

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OllamaConfig struct {
	APIKey    string
	BaseURL   string
	Model     string
	Dimension int
	Client    *http.Client
}

type OllamaEmbedder struct {
	apiKey    string
	baseURL   string
	model     string
	modelKey  string
	dimension int
	client    *http.Client
}

func NewOllamaEmbedder(cfg OllamaConfig) (*OllamaEmbedder, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "qwen3-embedding:8b"
	}
	if cfg.Dimension <= 0 {
		return nil, fmt.Errorf("embedding dimension must be positive")
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 30 * time.Second}
	}

	return &OllamaEmbedder{
		apiKey:    cfg.APIKey,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		model:     cfg.Model,
		modelKey:  ProviderModelKey(ProviderOllama, cfg.Model),
		dimension: cfg.Dimension,
		client:    cfg.Client,
	}, nil
}

func (e *OllamaEmbedder) Model() string { return e.modelKey }

func (e *OllamaEmbedder) Dimension() int { return e.dimension }

func (e *OllamaEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	var out struct {
		Embeddings [][]float64 `json:"embeddings"`
		Embedding  []float64   `json:"embedding"`
	}

	body := map[string]any{
		"model": e.model,
		"input": input,
	}
	if err := postJSON(ctx, aiPostRequest{
		Client:   e.client,
		Provider: ProviderOllama,
		URL:      e.baseURL + "/api/embed",
		Endpoint: "/api/embed",
		APIKey:   e.apiKey,
		Body:     body,
		Out:      &out,
	}); err != nil {
		return nil, err
	}

	raw := out.Embedding
	if len(raw) == 0 && len(out.Embeddings) > 0 {
		raw = out.Embeddings[0]
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("ollama embeddings response contained no data")
	}

	embedding := make([]float32, len(raw))
	for i, v := range raw {
		embedding[i] = float32(v)
	}
	normalized, err := NormalizeFloat32Vector(embedding, e.dimension)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding dimension mismatch: %w", err)
	}
	return normalized, nil
}

type OllamaGenerator struct {
	apiKey   string
	baseURL  string
	model    string
	modelKey string
	client   *http.Client
}

func NewOllamaGenerator(cfg OllamaConfig) (*OllamaGenerator, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://127.0.0.1:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "gemma4:26b"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OllamaGenerator{
		apiKey:   cfg.APIKey,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		model:    cfg.Model,
		modelKey: ProviderModelKey(ProviderOllama, cfg.Model),
		client:   cfg.Client,
	}, nil
}

func (g *OllamaGenerator) Model() string { return g.modelKey }

func (g *OllamaGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := g.GenerateWithTools(ctx, []ChatMessage{{Role: ChatRoleUser, Content: prompt}}, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Content) == "" {
		return "", fmt.Errorf("ollama chat response contained no message content")
	}
	return response.Content, nil
}

func (g *OllamaGenerator) GenerateWithTools(ctx context.Context, messages []ChatMessage, tools []ChatTool) (ChatResponse, error) {
	var out struct {
		Message openAIChatMessage `json:"message"`
	}

	body := map[string]any{
		"model":    g.model,
		"messages": openAIChatMessages(messages),
		"stream":   false,
	}
	if len(tools) > 0 {
		body["tools"] = openAIChatTools(tools)
	}
	if err := postJSON(ctx, aiPostRequest{
		Client:   g.client,
		Provider: ProviderOllama,
		URL:      g.baseURL + "/api/chat",
		Endpoint: "/api/chat",
		APIKey:   g.apiKey,
		Body:     body,
		Out:      &out,
	}); err != nil {
		return ChatResponse{}, err
	}
	return chatResponseFromOpenAIMessage(out.Message), nil
}
