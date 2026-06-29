package ama

import (
	"context"
	"strings"

	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
)

func (s *Server) GetSemanticIndexStatus(ctx context.Context, _ *amav1.GetSemanticIndexStatusRequest) (*amav1.SemanticIndexStatus, error) {
	indexStatus, err := s.semanticIndexStatus(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "semantic index status unavailable", "error", err)
		indexStatus = semanticIndexStatusUnavailable()
	}
	return indexStatusToProto(indexStatus), nil
}

func (s *Server) semanticIndexStatus(ctx context.Context) (semantic.IndexStatus, error) {
	if s.indexStatus == nil {
		return semantic.IndexStatus{
			State:   semantic.IndexStateUnavailable,
			Message: s.semanticUnavailableMessage(),
		}, nil
	}
	indexStatus, err := s.indexStatus.Status(ctx)
	if err != nil {
		return semantic.IndexStatus{}, err
	}
	if s.answerer == nil && strings.TrimSpace(s.unavailableReason) != "" {
		indexStatus.State = semantic.IndexStateUnavailable
		indexStatus.Message = s.semanticUnavailableMessage()
	}
	return indexStatus, nil
}

func (s *Server) semanticUnavailableMessage() string {
	if s != nil && strings.TrimSpace(s.unavailableReason) != "" {
		return s.unavailableReason
	}
	return "Semantic index is unavailable."
}

func semanticIndexStatusUnavailable() semantic.IndexStatus {
	return semantic.IndexStatus{
		State:   semantic.IndexStateUnavailable,
		Message: "Semantic index status is unavailable.",
	}
}

func indexStatusToProto(status semantic.IndexStatus) *amav1.SemanticIndexStatus {
	return &amav1.SemanticIndexStatus{
		State:               indexStateToProto(status.State),
		CompletedEventCount: status.CompletedEventCount,
		IndexedEventCount:   status.IndexedEventCount,
		PendingEventCount:   status.PendingEventCount,
		BackfillRunning:     status.BackfillRunning,
		Message:             status.Message,
	}
}

func indexStateToProto(state semantic.IndexState) amav1.SemanticIndexStatus_State {
	switch state {
	case semantic.IndexStateReady:
		return amav1.SemanticIndexStatus_READY
	case semantic.IndexStateIndexing:
		return amav1.SemanticIndexStatus_INDEXING
	case semantic.IndexStateEmpty:
		return amav1.SemanticIndexStatus_EMPTY
	case semantic.IndexStateUnavailable:
		return amav1.SemanticIndexStatus_UNAVAILABLE
	default:
		return amav1.SemanticIndexStatus_STATE_UNSPECIFIED
	}
}
