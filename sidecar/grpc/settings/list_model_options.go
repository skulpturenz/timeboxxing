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

	models, err := s.readQuerier.ListModels(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list models: %v", err)
	}

	return &settingsv1.ListModelOptionsResponse{
		Models: modelsToProto(models),
	}, nil
}
