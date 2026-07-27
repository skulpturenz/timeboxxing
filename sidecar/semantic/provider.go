package semantic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
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
	BaseURLValue        string
	EmbeddingModelID    int64
	SemanticModelID     int64
	EmbeddingOpenRouter string
	EmbeddingOllama     string
	SemanticOpenRouter  string
	SemanticOllama      string
	EmbeddingModelLabel string
	SemanticModelLabel  string
}

func LoadAISettings(ctx context.Context, querier readqueries.Querier) (AISettings, error) {
	if querier == nil {
		return AISettings{}, fmt.Errorf("settings querier is required")
	}
	row, err := querier.GetApplicationSettings(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return AISettings{}, fmt.Errorf("AI settings are not configured")
		}
		return AISettings{}, fmt.Errorf("get AI settings: %w", err)
	}
	return AISettings{
		Provider:            providerFromLabel(row.ModelProviderLabel),
		BaseURLValue:        utils.Coalesce(row.ModelProviderBaseUrl, ""),
		EmbeddingModelID:    row.EmbeddingModelID,
		SemanticModelID:     row.SemanticModelID,
		EmbeddingOpenRouter: utils.Coalesce(row.EmbeddingOpenrouterSlug, ""),
		EmbeddingOllama:     utils.Coalesce(row.EmbeddingOllamaSlug, ""),
		SemanticOpenRouter:  utils.Coalesce(row.SemanticOpenrouterSlug, ""),
		SemanticOllama:      utils.Coalesce(row.SemanticOllamaSlug, ""),
		EmbeddingModelLabel: row.EmbeddingLabel,
		SemanticModelLabel:  row.SemanticLabel,
	}, nil
}

// providerFromLabel maps a model_providers.label to a Provider.
func providerFromLabel(label string) Provider {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case string(ProviderOpenRouter):
		return ProviderOpenRouter
	case string(ProviderOllama):
		return ProviderOllama
	default:
		return Provider(strings.ToLower(strings.TrimSpace(label)))
	}
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
	return s.BaseURLValue
}

func requireProviderSlug(value string, label string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s model is unavailable for the selected provider", label)
	}
	return value, nil
}
