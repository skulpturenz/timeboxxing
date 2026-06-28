package ama

import (
	"context"
	"strings"
	"time"

	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultMaxSources       = int64(5)
	minMaxSources           = int64(1)
	maxMaxSources           = int64(10)
	onDemandBackfillLimit   = int64(200)
	indexingCatchupResponse = "I'm still indexing your usage history. Try again in a moment."
)

func (s *Server) Ask(ctx context.Context, req *amav1.AskRequest) (*amav1.AskResponse, error) {
	if s.answerer == nil {
		return nil, status.Error(codes.FailedPrecondition, "AMA answerer is unavailable")
	}

	question := strings.TrimSpace(req.GetQuestion())
	if question == "" {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}

	maxSources := maxSourcesFromRequest(req)
	answer, err := s.answerer.Answer(ctx, question, maxSources)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "answer question: %v", err)
	}
	indexStatus, err := s.semanticIndexStatus(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get semantic index status: %v", err)
	}
	if answer != nil && len(answer.Sources) == 0 && s.backfilling != nil {
		switch indexStatus.State {
		case semantic.IndexStateEmpty:
			answer = &semantic.Answer{
				Question: question,
				Answer:   "No completed usage events have been indexed yet. Ask again after you switch apps and the current session closes.",
				Model:    answer.Model,
				Sources:  nil,
			}
		case semantic.IndexStateIndexing:
			started := s.backfilling.Start(onDemandBackfillLimit)
			s.logger.InfoContext(ctx, "semantic backfill scheduled for empty AMA context", "started", started)
			answer = &semantic.Answer{
				Question: question,
				Answer:   indexingCatchupResponse,
				Model:    answer.Model,
				Sources:  nil,
			}
		default:
			hasMissing, err := s.backfilling.HasMissing(ctx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "check semantic index: %v", err)
			}
			if hasMissing {
				started := s.backfilling.Start(onDemandBackfillLimit)
				s.logger.InfoContext(ctx, "semantic backfill scheduled for empty AMA context", "started", started)
				answer = &semantic.Answer{
					Question: question,
					Answer:   indexingCatchupResponse,
					Model:    answer.Model,
					Sources:  nil,
				}
			}
		}
	}

	sourceCount := 0
	if answer != nil {
		sourceCount = len(answer.Sources)
	}
	s.logger.InfoContext(ctx, "AMA answer completed", "source_count", sourceCount, "max_sources", maxSources)

	return answerToProto(answer, indexStatus), nil
}

func (s *Server) GetSemanticIndexStatus(ctx context.Context, _ *amav1.GetSemanticIndexStatusRequest) (*amav1.SemanticIndexStatus, error) {
	indexStatus, err := s.semanticIndexStatus(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get semantic index status: %v", err)
	}
	return indexStatusToProto(indexStatus), nil
}

func maxSourcesFromRequest(req *amav1.AskRequest) int64 {
	maxSources := int64(req.GetMaxSources())
	if maxSources <= 0 {
		return defaultMaxSources
	}
	if maxSources < minMaxSources {
		return minMaxSources
	}
	if maxSources > maxMaxSources {
		return maxMaxSources
	}
	return maxSources
}

func answerToProto(answer *semantic.Answer, indexStatus semantic.IndexStatus) *amav1.AskResponse {
	if answer == nil {
		return &amav1.AskResponse{SemanticIndexStatus: indexStatusToProto(indexStatus)}
	}
	resp := &amav1.AskResponse{
		Answer:              answer.Answer,
		Model:               answer.Model,
		Sources:             make([]*amav1.Source, 0, len(answer.Sources)),
		SemanticIndexStatus: indexStatusToProto(indexStatus),
	}
	for _, source := range answer.Sources {
		resp.Sources = append(resp.Sources, &amav1.Source{
			TransitionEventId: source.TransitionEventID,
			Content:           source.Content,
			Distance:          source.Distance,
			DocumentKey:       source.DocumentKey,
			DocumentType:      source.DocumentType,
			StartedAt:         timestampOrNil(source.StartedAt),
			EndedAt:           timestampOrNil(source.EndedAt),
		})
	}
	return resp
}

func (s *Server) semanticIndexStatus(ctx context.Context) (semantic.IndexStatus, error) {
	if s.indexStatus == nil {
		return semantic.IndexStatus{
			State:   semantic.IndexStateUnavailable,
			Message: "Semantic index is unavailable.",
		}, nil
	}
	return s.indexStatus.Status(ctx)
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

func timestampOrNil(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
