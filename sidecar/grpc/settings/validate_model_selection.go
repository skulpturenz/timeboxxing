package settings

import (
	"context"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) validateModelSelection(ctx context.Context, provider semantic.Provider, embeddingModelID int64, semanticModelID int64) error {
	embeddingModels, err := s.readQuerier.ListEmbeddingModels(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "list embedding models: %v", err)
	}
	semanticModels, err := s.readQuerier.ListSemanticModels(ctx)
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
