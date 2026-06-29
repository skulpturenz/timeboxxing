package settings

import (
	"context"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultOllamaBaseURL     = "http://127.0.0.1:11434"
)

type Server struct {
	settingsv1.UnimplementedSettingsServiceServer

	querier queries.Querier
}

type NewServerParams struct {
	Querier queries.Querier
}

func NewServer(params NewServerParams) *Server {
	return &Server{querier: params.Querier}
}

func (s *Server) ListModelOptions(ctx context.Context, _ *settingsv1.ListModelOptionsRequest) (*settingsv1.ListModelOptionsResponse, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	embeddingModels, err := s.querier.ListEmbeddingModels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list embedding models: %v", err)
	}
	semanticModels, err := s.querier.ListSemanticModels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list semantic models: %v", err)
	}

	return &settingsv1.ListModelOptionsResponse{
		EmbeddingModels: embeddingModelsToProto(embeddingModels),
		SemanticModels:  semanticModelsToProto(semanticModels),
	}, nil
}

func (s *Server) GetAiSettings(ctx context.Context, req *settingsv1.GetAiSettingsRequest) (*settingsv1.AiSettings, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	row, err := s.querier.GetAISettings(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get AI settings: %v", err)
	}
	return aiSettingsToProto(row, req.GetOpenrouterSecretExists(), req.GetOllamaSecretExists()), nil
}

func (s *Server) SaveAiSettings(ctx context.Context, req *settingsv1.SaveAiSettingsRequest) (*settingsv1.AiSettings, error) {
	if s.querier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	provider, err := providerFromProto(req.GetProvider())
	if err != nil {
		return nil, err
	}

	openRouterBaseURL := strings.TrimSpace(req.GetOpenrouterBaseUrl())
	if openRouterBaseURL == "" {
		openRouterBaseURL = defaultOpenRouterBaseURL
	}
	ollamaBaseURL := strings.TrimSpace(req.GetOllamaBaseUrl())
	if ollamaBaseURL == "" {
		ollamaBaseURL = defaultOllamaBaseURL
	}

	if err := s.validateModelSelection(ctx, provider, req.GetEmbeddingModelId(), req.GetSemanticModelId()); err != nil {
		return nil, err
	}

	err = s.querier.UpsertAISettings(ctx, queries.UpsertAISettingsParams{
		Provider:          string(provider),
		OpenrouterBaseUrl: openRouterBaseURL,
		OllamaBaseUrl:     ollamaBaseURL,
		EmbeddingModelID:  req.GetEmbeddingModelId(),
		SemanticModelID:   req.GetSemanticModelId(),
		UpdatedAt:         time.Now().UTC(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save AI settings: %v", err)
	}

	row, err := s.querier.GetAISettings(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload AI settings: %v", err)
	}
	return aiSettingsToProto(row, req.GetOpenrouterSecretExists(), req.GetOllamaSecretExists()), nil
}

func (s *Server) validateModelSelection(ctx context.Context, provider semantic.Provider, embeddingModelID int64, semanticModelID int64) error {
	embeddingModels, err := s.querier.ListEmbeddingModels(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "list embedding models: %v", err)
	}
	semanticModels, err := s.querier.ListSemanticModels(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "list semantic models: %v", err)
	}

	embeddingModel, ok := findEmbeddingModel(embeddingModels, embeddingModelID)
	if !ok {
		return status.Error(codes.InvalidArgument, "embedding model is invalid")
	}
	semanticModel, ok := findSemanticModel(semanticModels, semanticModelID)
	if !ok {
		return status.Error(codes.InvalidArgument, "semantic model is invalid")
	}

	switch provider {
	case semantic.ProviderOpenRouter:
		if strings.TrimSpace(embeddingModel.OpenrouterSlug) == "" || strings.TrimSpace(semanticModel.OpenrouterSlug) == "" {
			return status.Error(codes.InvalidArgument, "selected models are unavailable for OpenRouter")
		}
	case semantic.ProviderOllama:
		if strings.TrimSpace(embeddingModel.OllamaSlug) == "" || strings.TrimSpace(semanticModel.OllamaSlug) == "" {
			return status.Error(codes.InvalidArgument, "selected models are unavailable for Ollama")
		}
	default:
		return status.Error(codes.InvalidArgument, "provider is invalid")
	}

	return nil
}

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
