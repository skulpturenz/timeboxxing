package settings

import (
	"database/sql"
	"strings"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Model-provider ids, matching the model_providers seed (db/seeds/model_providers).
const (
	modelProviderOpenRouterID = 1
	modelProviderOllamaID     = 2
)

func providerFromProto(provider settingsv1.AiProvider) (semantic.Provider, error) {
	switch provider {
	case settingsv1.AiProvider_OPENROUTER:
		return semantic.ProviderOpenRouter, nil
	case settingsv1.AiProvider_OLLAMA:
		return semantic.ProviderOllama, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func providerToModelProviderID(provider semantic.Provider) sql.NullInt64 {
	switch provider {
	case semantic.ProviderOpenRouter:
		return sql.NullInt64{Int64: modelProviderOpenRouterID, Valid: true}
	case semantic.ProviderOllama:
		return sql.NullInt64{Int64: modelProviderOllamaID, Valid: true}
	default:
		return sql.NullInt64{}
	}
}

func providerToProtoFromLabel(label string) settingsv1.AiProvider {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case string(semantic.ProviderOpenRouter):
		return settingsv1.AiProvider_OPENROUTER
	case string(semantic.ProviderOllama):
		return settingsv1.AiProvider_OLLAMA
	default:
		return settingsv1.AiProvider_AI_PROVIDER_UNSPECIFIED
	}
}

func modelsToProto(models []readqueries.Model) []*settingsv1.ModelOption {
	out := make([]*settingsv1.ModelOption, 0, len(models))
	for _, model := range models {
		out = append(out, &settingsv1.ModelOption{
			Id:             model.ID,
			OpenrouterSlug: model.OpenrouterSlug.String,
			OllamaSlug:     model.OllamaSlug.String,
			Label:          model.Label,
			Semantic:       model.Semantic,
			Embedding:      model.Embedding,
		})
	}
	return out
}

func aiSettingsToProto(row readqueries.GetApplicationSettingsRow, openRouterSecretExists bool, ollamaSecretExists bool) *settingsv1.AiSettings {
	return &settingsv1.AiSettings{
		Provider:               providerToProtoFromLabel(row.ModelProviderLabel),
		ModelProviderBaseUrl:   row.ModelProviderBaseUrl.String,
		EmbeddingModelId:       row.EmbeddingModelID,
		SemanticModelId:        row.SemanticModelID,
		OpenrouterSecretExists: openRouterSecretExists,
		OllamaSecretExists:     ollamaSecretExists,
	}
}

func findModel(models []readqueries.Model, id int64) (readqueries.Model, bool) {
	for _, model := range models {
		if model.ID == id {
			return model, true
		}
	}
	return readqueries.Model{}, false
}
