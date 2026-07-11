package settings

import (
	"context"

	settingsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/settings/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetAiSettings(ctx context.Context, req *settingsv1.GetAiSettingsRequest) (*settingsv1.AiSettings, error) {
	if s.readQuerier == nil {
		return nil, status.Error(codes.FailedPrecondition, "settings store is unavailable")
	}

	row, err := s.readQuerier.GetAISettings(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get AI settings: %v", err)
	}
	return aiSettingsToProto(row, req.GetOpenrouterSecretExists(), req.GetOllamaSecretExists()), nil
}
