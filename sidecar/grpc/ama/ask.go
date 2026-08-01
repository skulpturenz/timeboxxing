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
	structuredRequest := req.GetStructuredQuery()
	if question == "" && structuredRequest == nil {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}

	maxSources := maxSourcesFromRequest(req)
	var answer *semantic.Answer
	var err error
	if structuredRequest != nil {
		query, parseErr := structuredQueryFromProto(structuredRequest)
		if parseErr != nil {
			return nil, parseErr
		}
		answer, err = s.answerer.AnswerStructured(ctx, query)
	} else {
		answer, err = s.answerer.Answer(ctx, question, maxSources)
	}
	if err != nil {
		return nil, answerErrorStatus(err)
	}
	indexStatus, err := s.semanticIndexStatus(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "semantic index status unavailable while answering", "error", err)
		indexStatus = semanticIndexStatusUnavailable()
	}
	if structuredRequest == nil && answer != nil && len(answer.Sources) == 0 && s.backfilling != nil {
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

func structuredQueryFromProto(req *amav1.StructuredQuery) (semantic.StructuredQuery, error) {
	if req == nil {
		return semantic.StructuredQuery{}, status.Error(codes.InvalidArgument, "structured_query is required")
	}
	window, err := timeWindowFromProto(req.GetWindow(), "window")
	if err != nil {
		return semantic.StructuredQuery{}, err
	}
	query := semantic.StructuredQuery{
		Window:      window,
		Limit:       int(req.GetLimit()),
		IncludeIdle: req.GetIncludeIdle(),
		PeriodLabel: strings.TrimSpace(req.GetPeriodLabel()),
	}
	switch req.GetKind() {
	case amav1.QueryKind_QUERY_KIND_TIMELINE:
		query.Kind = semantic.StructuredQueryKindTimeline
	default:
		return semantic.StructuredQuery{}, status.Error(codes.InvalidArgument, "structured_query kind is required")
	}
	return query, nil
}

func timeWindowFromProto(req *amav1.TimeWindow, fieldName string) (semantic.TimeWindow, error) {
	if req == nil {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s is required", fieldName)
	}
	if req.GetStartedAt() == nil {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s.started_at is required", fieldName)
	}
	if req.GetEndedAt() == nil {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s.ended_at is required", fieldName)
	}
	if err := req.GetStartedAt().CheckValid(); err != nil {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s.started_at is invalid: %v", fieldName, err)
	}
	if err := req.GetEndedAt().CheckValid(); err != nil {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s.ended_at is invalid: %v", fieldName, err)
	}
	window := semantic.TimeWindow{
		StartedAt: req.GetStartedAt().AsTime().UTC(),
		EndedAt:   req.GetEndedAt().AsTime().UTC(),
	}
	if !window.EndedAt.After(window.StartedAt) {
		return semantic.TimeWindow{}, status.Errorf(codes.InvalidArgument, "%s.ended_at must be after started_at", fieldName)
	}
	return window, nil
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
	case semantic.ArtifactTypeUsageTimeline:
		if artifact.UsageTimeline == nil {
			return nil
		}
		timeline := artifact.UsageTimeline
		events := make([]*amav1.UsageTimelineEvent, 0, len(timeline.Events))
		for _, event := range timeline.Events {
			events = append(events, usageTimelineEventToProto(event))
		}
		return &amav1.Artifact{
			Value: &amav1.Artifact_UsageTimeline{
				UsageTimeline: &amav1.UsageTimeline{
					StartedAt:            timestampOrNil(timeline.StartedAt),
					EndedAt:              timestampOrNil(timeline.EndedAt),
					Timezone:             timeline.TimeZone,
					TotalDurationSeconds: timeline.TotalDurationSeconds,
					Events:               events,
					PeriodLabel:          timeline.PeriodLabel,
					TotalEventCount:      int32(timeline.TotalEventCount),
					Truncated:            timeline.Truncated,
				},
			},
		}
	default:
		return nil
	}
}

func usageTimelineEventToProto(event semantic.UsageTimelineEvent) *amav1.UsageTimelineEvent {
	return &amav1.UsageTimelineEvent{
		TransitionEventId:     event.TransitionEventID,
		Title:                 event.Title,
		SourceName:            event.SourceName,
		SourceType:            event.SourceType,
		StartedAt:             timestampOrNil(event.StartedAt),
		EndedAt:               timestampOrNil(event.EndedAt),
		DurationSeconds:       event.DurationSeconds,
		ApplicationIdentifier: event.ApplicationIdentifier,
		ApplicationPath:       event.ApplicationPath,
		UrlHost:               event.URLHost,
		Idle:                  event.Idle,
	}
}

func timestampOrNil(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
