package settings

import (
	"context"
	"strings"

	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) validateModelSelection(ctx context.Context, provider semantic.Provider, embeddingModelID int64, semanticModelID int64) error {
	models, err := s.readQuerier.ListModels(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "list models: %v", err)
	}

	embeddingModel, ok := findModel(models, embeddingModelID)
	if !ok || !embeddingModel.Embedding {
		return status.Error(codes.InvalidArgument, "embedding model is invalid")
	}
	semanticModel, ok := findModel(models, semanticModelID)
	if !ok || !semanticModel.Semantic {
		return status.Error(codes.InvalidArgument, "semantic model is invalid")
	}

	switch provider {
	case semantic.ProviderOpenRouter:
		if strings.TrimSpace(embeddingModel.OpenrouterSlug.String) == "" || strings.TrimSpace(semanticModel.OpenrouterSlug.String) == "" {
			return status.Error(codes.InvalidArgument, "selected models are unavailable for OpenRouter")
		}
	case semantic.ProviderOllama:
		if strings.TrimSpace(embeddingModel.OllamaSlug.String) == "" || strings.TrimSpace(semanticModel.OllamaSlug.String) == "" {
			return status.Error(codes.InvalidArgument, "selected models are unavailable for Ollama")
		}
	default:
		return status.Error(codes.InvalidArgument, "provider is invalid")
	}

	return nil
}
