package settings

import (
	"context"
	"strings"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/db/queries"
	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
