package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
)

type Provider string

const (
	ProviderOpenRouter Provider = "openrouter"
	ProviderOllama     Provider = "ollama"
)

func ProviderModelKey(provider Provider, model string) string {
	return string(provider) + ":" + strings.TrimSpace(model)
}

type AISettings struct {
	Provider            Provider
	OpenRouterBaseURL   string
	OllamaBaseURL       string
	EmbeddingModelID    int64
	SemanticModelID     int64
	EmbeddingOpenRouter string
	EmbeddingOllama     string
	SemanticOpenRouter  string
	SemanticOllama      string
	EmbeddingModelLabel string
	SemanticModelLabel  string
}

func LoadAISettings(ctx context.Context, querier queries.Querier) (AISettings, error) {
	if querier == nil {
		return AISettings{}, fmt.Errorf("settings querier is required")
	}
	row, err := querier.GetAISettings(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return AISettings{}, fmt.Errorf("AI settings are not configured")
		}
		return AISettings{}, fmt.Errorf("get AI settings: %w", err)
	}
	return AISettings{
		Provider:            Provider(row.Provider),
		OpenRouterBaseURL:   row.OpenrouterBaseUrl,
		OllamaBaseURL:       row.OllamaBaseUrl,
		EmbeddingModelID:    row.EmbeddingModelID,
		SemanticModelID:     row.SemanticModelID,
		EmbeddingOpenRouter: row.EmbeddingOpenrouterSlug,
		EmbeddingOllama:     row.EmbeddingOllamaSlug,
		SemanticOpenRouter:  row.SemanticOpenrouterSlug,
		SemanticOllama:      row.SemanticOllamaSlug,
		EmbeddingModelLabel: row.EmbeddingLabel,
		SemanticModelLabel:  row.SemanticLabel,
	}, nil
}

func (s AISettings) EmbeddingSlug() (string, error) {
	switch s.Provider {
	case ProviderOpenRouter:
		return requireProviderSlug(s.EmbeddingOpenRouter, "OpenRouter embedding")
	case ProviderOllama:
		return requireProviderSlug(s.EmbeddingOllama, "Ollama embedding")
	default:
		return "", fmt.Errorf("unsupported AI provider %q", s.Provider)
	}
}

func (s AISettings) SemanticSlug() (string, error) {
	switch s.Provider {
	case ProviderOpenRouter:
		return requireProviderSlug(s.SemanticOpenRouter, "OpenRouter semantic")
	case ProviderOllama:
		return requireProviderSlug(s.SemanticOllama, "Ollama semantic")
	default:
		return "", fmt.Errorf("unsupported AI provider %q", s.Provider)
	}
}

func (s AISettings) BaseURL() string {
	switch s.Provider {
	case ProviderOpenRouter:
		return s.OpenRouterBaseURL
	case ProviderOllama:
		return s.OllamaBaseURL
	default:
		return ""
	}
}

func requireProviderSlug(value string, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s model is unavailable for the selected provider", label)
	}
	return value, nil
}
