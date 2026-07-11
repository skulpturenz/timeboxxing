package settings

import (
	readqueries "github.com/skulpturenz/timeboxxing/sidecar/db/read_queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func providerToProto(provider string) settingsv1.AiProvider {
	switch semantic.Provider(provider) {
	case semantic.ProviderOpenRouter:
		return settingsv1.AiProvider_OPENROUTER
	case semantic.ProviderOllama:
		return settingsv1.AiProvider_OLLAMA
	default:
		return settingsv1.AiProvider_AI_PROVIDER_UNSPECIFIED
	}
}

func embeddingModelsToProto(models []readqueries.EmbeddingModel) []*settingsv1.ModelOption {
	out := make([]*settingsv1.ModelOption, 0, len(models))
	for _, model := range models {
		out = append(out, &settingsv1.ModelOption{
			Id:             model.ID,
			OpenrouterSlug: model.OpenrouterSlug,
			OllamaSlug:     model.OllamaSlug,
			Label:          model.Label,
		})
	}
	return out
}

func semanticModelsToProto(models []readqueries.SemanticModel) []*settingsv1.ModelOption {
	out := make([]*settingsv1.ModelOption, 0, len(models))
	for _, model := range models {
		out = append(out, &settingsv1.ModelOption{
			Id:             model.ID,
			OpenrouterSlug: model.OpenrouterSlug,
			OllamaSlug:     model.OllamaSlug,
			Label:          model.Label,
		})
	}
	return out
}

func aiSettingsToProto(row readqueries.GetAISettingsRow, openRouterSecretExists bool, ollamaSecretExists bool) *settingsv1.AiSettings {
	return &settingsv1.AiSettings{
		Provider:               providerToProto(row.Provider),
		OpenrouterBaseUrl:      row.OpenrouterBaseUrl,
		OllamaBaseUrl:          row.OllamaBaseUrl,
		EmbeddingModelId:       row.EmbeddingModelID,
		SemanticModelId:        row.SemanticModelID,
		OpenrouterSecretExists: openRouterSecretExists,
		OllamaSecretExists:     ollamaSecretExists,
	}
}

func findEmbeddingModel(models []readqueries.EmbeddingModel, id int64) (readqueries.EmbeddingModel, bool) {
	for _, model := range models {
		if model.ID == id {
			return model, true
		}
	}
	return readqueries.EmbeddingModel{}, false
}

func findSemanticModel(models []readqueries.SemanticModel, id int64) (readqueries.SemanticModel, bool) {
	for _, model := range models {
		if model.ID == id {
			return model, true
		}
	}
	return readqueries.SemanticModel{}, false
}
