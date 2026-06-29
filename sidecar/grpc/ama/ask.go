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
		return nil, status.Error(codes.FailedPrecondition, s.semanticUnavailableMessage())
	}

	question := strings.TrimSpace(req.GetQuestion())
	if question == "" {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}

	maxSources := maxSourcesFromRequest(req)
	answer, err := s.answerer.Answer(ctx, question, maxSources)
	if err != nil {
		return nil, answerErrorStatus(err)
	}
	indexStatus, err := s.semanticIndexStatus(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "semantic index status unavailable while answering", "error", err)
		indexStatus = semanticIndexStatusUnavailable()
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
		s.logger.WarnContext(ctx, "semantic index status unavailable", "error", err)
		indexStatus = semanticIndexStatusUnavailable()
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
		Artifacts:           make([]*amav1.Artifact, 0, len(answer.Artifacts)),
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
	for _, artifact := range answer.Artifacts {
		if protoArtifact := artifactToProto(artifact); protoArtifact != nil {
			resp.Artifacts = append(resp.Artifacts, protoArtifact)
		}
	}
	return resp
}

func artifactToProto(artifact semantic.Artifact) *amav1.Artifact {
	switch artifact.Type {
	case semantic.ArtifactTypeAppUsageChart:
		if artifact.AppUsageChart == nil {
			return nil
		}
		chart := artifact.AppUsageChart
		buckets := make([]*amav1.AppUsageBucket, 0, len(chart.Buckets))
		for _, bucket := range chart.Buckets {
			buckets = append(buckets, &amav1.AppUsageBucket{
				Name:                  bucket.Name,
				SourceType:            bucket.SourceType,
				DurationSeconds:       bucket.DurationSeconds,
				SessionCount:          bucket.SessionCount,
				ApplicationIdentifier: bucket.ApplicationIdentifier,
				ApplicationPath:       bucket.ApplicationPath,
			})
		}
		return &amav1.Artifact{
			Value: &amav1.Artifact_AppUsageChart{
				AppUsageChart: &amav1.AppUsageChart{
					StartedAt:            timestampOrNil(chart.StartedAt),
					EndedAt:              timestampOrNil(chart.EndedAt),
					Timezone:             chart.TimeZone,
					TotalDurationSeconds: chart.TotalDurationSeconds,
					Buckets:              buckets,
					PeriodLabel:          chart.PeriodLabel,
				},
			},
		}
	default:
		return nil
	}
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

func answerErrorStatus(err error) error {
	message, ok := semantic.AIRequestUserMessage(err)
	if !ok {
		return status.Error(codes.Internal, "AMA could not answer right now. Please try again in a moment.")
	}

	code := codes.Internal
	if requestCode, ok := semantic.AIRequestStatusCode(err); ok {
		switch requestCode {
		case 401, 402, 400, 404:
			code = codes.FailedPrecondition
		case 429:
			code = codes.ResourceExhausted
		case 502, 503, 529:
			code = codes.Unavailable
		}
	}
	return status.Error(code, message)
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
