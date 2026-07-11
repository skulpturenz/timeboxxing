package settings

import (
	"context"

	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListModelOptions(ctx context.Context, _ *settingsv1.ListModelOptionsRequest) (*settingsv1.ListModelOptionsResponse, error) {
	if s.readQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	embeddingModels, err := s.readQuerier.ListEmbeddingModels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list embedding models: %v", err)
	}
	semanticModels, err := s.readQuerier.ListSemanticModels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list semantic models: %v", err)
	}

	return &settingsv1.ListModelOptionsResponse{
		EmbeddingModels: embeddingModelsToProto(embeddingModels),
		SemanticModels:  semanticModelsToProto(semanticModels),
	}, nil
}
