package settings

import (
	"context"
	"strings"

	writequeries "github.com/skulpturenz/timeboxxing/sidecar/db/write_queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) SaveAiSettings(ctx context.Context, req *settingsv1.SaveAiSettingsRequest) (*settingsv1.AiSettings, error) {
	if s.writeQuerier == nil || s.readQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	provider, err := providerFromProto(req.GetProvider())
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimSpace(req.GetModelProviderBaseUrl())
	if baseURL == "" {
		baseURL = defaultBaseURLForProvider(provider)
	}

	if err := s.validateModelSelection(ctx, provider, req.GetEmbeddingModelId(), req.GetSemanticModelId()); err != nil {
		return nil, err
	}

	// ReleaseChannel is left NULL; the upsert preserves the existing value via COALESCE.
	err = s.writeQuerier.UpsertApplicationSettings(ctx, writequeries.UpsertApplicationSettingsParams{
		ModelProviderID:      providerToModelProviderID(provider),
		ModelProviderBaseUrl: &baseURL,
		EmbeddingModelID:     req.GetEmbeddingModelId(),
		SemanticModelID:      req.GetSemanticModelId(),
		ReleaseChannel:       nil,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "save AI settings: %v", err)
	}

	row, err := s.readQuerier.GetApplicationSettings(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reload AI settings: %v", err)
	}
	return aiSettingsToProto(row, req.GetOpenrouterSecretExists(), req.GetOllamaSecretExists()), nil
}

func defaultBaseURLForProvider(provider semantic.Provider) string {
	if provider == semantic.ProviderOllama {
		return defaultOllamaBaseURL
	}
	return defaultOpenRouterBaseURL
}
