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
	modelKey  string
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
		cfg.Model = "qwen/qwen3-embedding-8b"
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
		modelKey:  ProviderModelKey(ProviderOpenRouter, cfg.Model),
		dimension: cfg.Dimension,
		client:    cfg.Client,
	}, nil
}

func (e *OpenRouterEmbedder) Model() string { return e.modelKey }

func (e *OpenRouterEmbedder) Dimension() int { return e.dimension }

func (e *OpenRouterEmbedder) Embed(ctx context.Context, input string) ([]float32, error) {
	var out struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}

	body := map[string]any{
		"model":           e.model,
		"input":           input,
		"encoding_format": "float",
	}
	if err := e.post(ctx, "/embeddings", body, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("openrouter embeddings response contained no data")
	}
	embedding := make([]float32, len(out.Data[0].Embedding))
	for i, v := range out.Data[0].Embedding {
		embedding[i] = float32(v)
	}
	normalized, err := NormalizeFloat32Vector(embedding, e.dimension)
	if err != nil {
		return nil, fmt.Errorf("openrouter embedding dimension mismatch: %w", err)
	}
	return normalized, nil
}

type OpenRouterGenerator struct {
	apiKey   string
	baseURL  string
	model    string
	modelKey string
	client   *http.Client
}

func NewOpenRouterGenerator(cfg OpenRouterConfig) (*OpenRouterGenerator, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("openrouter API key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://openrouter.ai/api/v1"
	}
	if cfg.Model == "" {
		cfg.Model = "minimax/minimax-m3"
	}
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 60 * time.Second}
	}

	return &OpenRouterGenerator{
		apiKey:   cfg.APIKey,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		model:    cfg.Model,
		modelKey: ProviderModelKey(ProviderOpenRouter, cfg.Model),
		client:   cfg.Client,
	}, nil
}

func (g *OpenRouterGenerator) Model() string { return g.modelKey }

func (g *OpenRouterGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	response, err := g.GenerateWithTools(ctx, []ChatMessage{{Role: ChatRoleUser, Content: prompt}}, nil)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

func (g *OpenRouterGenerator) GenerateWithTools(ctx context.Context, messages []ChatMessage, tools []ChatTool) (ChatResponse, error) {
	var out struct {
		Choices []struct {
			Message openAIChatMessage `json:"message"`
		} `json:"choices"`
	}

	body := map[string]any{
		"model":    g.model,
		"messages": openAIChatMessages(messages),
	}
	if len(tools) > 0 {
		body["tools"] = openAIChatTools(tools)
		body["tool_choice"] = "auto"
	}
	if err := g.post(ctx, "/chat/completions", body, &out); err != nil {
		return ChatResponse{}, err
	}
	if len(out.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openrouter chat response contained no choices")
	}
	return chatResponseFromOpenAIMessage(out.Choices[0].Message), nil
}

func (e *OpenRouterEmbedder) post(ctx context.Context, endpoint string, body any, out any) error {
	return postJSON(ctx, aiPostRequest{
		Client:   e.client,
		Provider: ProviderOpenRouter,
		URL:      e.baseURL + endpoint,
		Endpoint: endpoint,
		APIKey:   e.apiKey,
		Body:     body,
		Out:      out,
		Headers: map[string]string{
			"HTTP-Referer":       "https://github.com/skulpturenz/timeboxxing",
			"X-OpenRouter-Title": "Timeboxxing",
		},
	})
}

func (g *OpenRouterGenerator) post(ctx context.Context, endpoint string, body any, out any) error {
	return postJSON(ctx, aiPostRequest{
		Client:   g.client,
		Provider: ProviderOpenRouter,
		URL:      g.baseURL + endpoint,
		Endpoint: endpoint,
		APIKey:   g.apiKey,
		Body:     body,
		Out:      out,
		Headers: map[string]string{
			"HTTP-Referer":       "https://github.com/skulpturenz/timeboxxing",
			"X-OpenRouter-Title": "Timeboxxing",
		},
	})
}

type aiPostRequest struct {
	Client   *http.Client
	Provider Provider
	URL      string
	Endpoint string
	APIKey   string
	Body     any
	Out      any
	Headers  map[string]string
}

func postJSON(ctx context.Context, reqSpec aiPostRequest) error {
	payload, err := json.Marshal(reqSpec.Body)
	if err != nil {
		return fmt.Errorf("marshal AI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqSpec.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create AI request: %w", err)
	}
	if reqSpec.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+reqSpec.APIKey)
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range reqSpec.Headers {
		if strings.TrimSpace(value) != "" {
			req.Header.Set(name, value)
		}
	}

	resp, err := reqSpec.Client.Do(req)
	if err != nil {
		return &AIRequestError{
			Provider:     reqSpec.Provider,
			Endpoint:     reqSpec.Endpoint,
			TransportErr: err,
		}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read AI response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseAIError(reqSpec.Provider, reqSpec.Endpoint, resp.StatusCode, resp.Header.Get("Retry-After"), data)
	}
	if err := json.Unmarshal(data, reqSpec.Out); err != nil {
		return fmt.Errorf("decode AI response: %w", err)
	}

	return nil
}

type openAIChatMessage struct {
	Role       string               `json:"role,omitempty"`
	Content    string               `json:"content,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIChatToolCall `json:"tool_calls,omitempty"`
}

type openAIChatToolCall struct {
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIChatFunction `json:"function"`
}

type openAIChatFunction struct {
	Name      string          `json:"name,omitempty"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func openAIChatMessages(messages []ChatMessage) []map[string]any {
	encoded := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{
			"role":    string(message.Role),
			"content": message.Content,
		}
		if message.Role == ChatRoleTool {
			item["tool_call_id"] = message.ToolCallID
		}
		if len(message.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(message.ToolCalls))
			for _, call := range message.ToolCalls {
				id := strings.TrimSpace(call.ID)
				if id == "" {
					id = "tool_call"
				}
				calls = append(calls, map[string]any{
					"id":   id,
					"type": "function",
					"function": map[string]any{
						"name":      call.Name,
						"arguments": call.Arguments,
					},
				})
			}
			item["tool_calls"] = calls
		}
		encoded = append(encoded, item)
	}
	return encoded
}

func openAIChatTools(tools []ChatTool) []map[string]any {
	encoded := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		encoded = append(encoded, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		})
	}
	return encoded
}

func chatResponseFromOpenAIMessage(message openAIChatMessage) ChatResponse {
	calls := make([]ChatToolCall, 0, len(message.ToolCalls))
	for i, call := range message.ToolCalls {
		if call.Type != "" && call.Type != "function" {
			continue
		}
		id := strings.TrimSpace(call.ID)
		if id == "" {
			id = fmt.Sprintf("tool_call_%d", i+1)
		}
		calls = append(calls, ChatToolCall{
			ID:        id,
			Name:      call.Function.Name,
			Arguments: chatToolArgumentsString(call.Function.Arguments),
		})
	}
	return ChatResponse{
		Content:   message.Content,
		ToolCalls: calls,
	}
}

func chatToolArgumentsString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		if strings.TrimSpace(value) == "" {
			return "{}"
		}
		return value
	}
	return string(raw)
}
