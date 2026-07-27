package settings

import (
	"strings"

	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
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

func providerToModelProviderID(provider semantic.Provider) *int64 {
	switch provider {
	case semantic.ProviderOpenRouter:
		id := int64(modelProviderOpenRouterID)
		return &id
	case semantic.ProviderOllama:
		id := int64(modelProviderOllamaID)
		return &id
	default:
		return nil
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
			OpenrouterSlug: utils.Coalesce(model.OpenrouterSlug, ""),
			OllamaSlug:     utils.Coalesce(model.OllamaSlug, ""),
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
		ModelProviderBaseUrl:   utils.Coalesce(row.ModelProviderBaseUrl, ""),
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
