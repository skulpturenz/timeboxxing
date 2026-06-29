package settings

import (
	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
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

func embeddingModelsToProto(models []queries.EmbeddingModel) []*settingsv1.ModelOption {
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

func semanticModelsToProto(models []queries.SemanticModel) []*settingsv1.ModelOption {
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

func aiSettingsToProto(row queries.GetAISettingsRow, openRouterSecretExists bool, ollamaSecretExists bool) *settingsv1.AiSettings {
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

func findEmbeddingModel(models []queries.EmbeddingModel, id int64) (queries.EmbeddingModel, bool) {
	for _, model := range models {
		if model.ID == id {
			return model, true
		}
	}
	return queries.EmbeddingModel{}, false
}

func findSemanticModel(models []queries.SemanticModel, id int64) (queries.SemanticModel, bool) {
	for _, model := range models {
		if model.ID == id {
			return model, true
		}
	}
	return queries.SemanticModel{}, false
}
