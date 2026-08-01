package ama

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAskReturnsAnswerAndSources(t *testing.T) {
	searcher := &recordingSearcher{
		results: []semantic.SearchResult{
			{TransitionEventID: 42, Content: "Application: Google Chrome", Distance: 0.125},
		},
	}
	server := newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(searcher, &recordingGenerator{answer: "You were browsing GitHub."}),
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{
		Question:   "  What was I doing? ",
		MaxSources: 99,
	})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if searcher.query != "What was I doing?" {
		t.Fatalf("expected trimmed query, got %q", searcher.query)
	}
	if searcher.k != maxMaxSources {
		t.Fatalf("expected clamped max sources %d, got %d", maxMaxSources, searcher.k)
	}
	if resp.GetAnswer() != "You were browsing GitHub." {
		t.Fatalf("unexpected answer %q", resp.GetAnswer())
	}
	if resp.GetModel() != "test-generator" {
		t.Fatalf("unexpected model %q", resp.GetModel())
	}
	if len(resp.GetSources()) != 1 {
		t.Fatalf("expected one source, got %d", len(resp.GetSources()))
	}
	if resp.GetSources()[0].GetTransitionEventId() != 42 {
		t.Fatalf("unexpected source transition event id %d", resp.GetSources()[0].GetTransitionEventId())
	}
}

func TestAskDefaultsMaxSourcesAndHandlesNoContext(t *testing.T) {
	searcher := &recordingSearcher{}
	server := newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(searcher, &recordingGenerator{answer: "unused"}),
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "anything indexed?"})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if searcher.k != defaultMaxSources {
		t.Fatalf("expected default max sources %d, got %d", defaultMaxSources, searcher.k)
	}
	if !strings.Contains(resp.GetAnswer(), "could not find any relevant transition events") {
		t.Fatalf("unexpected no-context answer %q", resp.GetAnswer())
	}
	if len(resp.GetSources()) != 0 {
		t.Fatalf("expected no sources, got %d", len(resp.GetSources()))
	}
}

func TestAskSchedulesBackfillAndReturnsCatchupMessageWhenNoContextHasMissingRows(t *testing.T) {
	searcher := &recordingSearcher{}
	backfilling := &recordingAmaBackfillCoordinator{hasMissing: true}
	server := newTestServer(testServerParams{
		Answerer:    semantic.NewAnswerer(searcher, &recordingGenerator{answer: "unused"}),
		Backfilling: backfilling,
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "What did I do today?"})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if resp.GetAnswer() != indexingCatchupResponse {
		t.Fatalf("unexpected answer %q", resp.GetAnswer())
	}
	if len(resp.GetSources()) != 0 {
		t.Fatalf("expected no sources while indexing, got %d", len(resp.GetSources()))
	}
	if backfilling.hasMissingCalls != 1 {
		t.Fatalf("expected one missing check, got %d", backfilling.hasMissingCalls)
	}
	if backfilling.startCalls != 1 {
		t.Fatalf("expected one backfill start, got %d", backfilling.startCalls)
	}
	if backfilling.limit != onDemandBackfillLimit {
		t.Fatalf("expected backfill limit %d, got %d", onDemandBackfillLimit, backfilling.limit)
	}
}

func TestAskKeepsNoContextAnswerWhenNoContextHasNoMissingRows(t *testing.T) {
	searcher := &recordingSearcher{}
	backfilling := &recordingAmaBackfillCoordinator{}
	server := newTestServer(testServerParams{
		Answerer:    semantic.NewAnswerer(searcher, &recordingGenerator{answer: "unused"}),
		Backfilling: backfilling,
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "anything indexed?"})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if !strings.Contains(resp.GetAnswer(), "could not find any relevant transition events") {
		t.Fatalf("unexpected no-context answer %q", resp.GetAnswer())
	}
	if backfilling.startCalls != 0 {
		t.Fatalf("expected no backfill start, got %d", backfilling.startCalls)
	}
}

func TestAskDoesNotScheduleBackfillWhenSourcesExist(t *testing.T) {
	searcher := &recordingSearcher{
		results: []semantic.SearchResult{{TransitionEventID: 42, Content: "Application: Calendar"}},
	}
	backfilling := &recordingAmaBackfillCoordinator{hasMissing: true}
	server := newTestServer(testServerParams{
		Answerer:    semantic.NewAnswerer(searcher, &recordingGenerator{answer: "Calendar work."}),
		Backfilling: backfilling,
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if resp.GetAnswer() != "Calendar work." {
		t.Fatalf("unexpected answer %q", resp.GetAnswer())
	}
	if backfilling.hasMissingCalls != 0 {
		t.Fatalf("expected no missing check, got %d", backfilling.hasMissingCalls)
	}
}

func TestAskValidationAndErrors(t *testing.T) {
	if _, err := newTestServer(testServerParams{}).Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", err)
	}

	server := newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(&recordingSearcher{}, &recordingGenerator{answer: "ok"}),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "   "}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}

	server = newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(&recordingSearcher{err: fmt.Errorf("search failed")}, &recordingGenerator{answer: "ok"}),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal search error, got %v", err)
	}

	server = newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(
			&recordingSearcher{results: []semantic.SearchResult{{TransitionEventID: 1, Content: "Application: Calendar"}}},
			&recordingGenerator{err: fmt.Errorf("generate failed")},
		),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal generation error, got %v", err)
	}
}

func TestAskMapsAIRequestErrorsToSafeGrpcStatus(t *testing.T) {
	server := newTestServer(testServerParams{
		Answerer: semantic.NewAnswerer(
			&recordingSearcher{err: &semantic.AIRequestError{
				Provider:   semantic.ProviderOpenRouter,
				Endpoint:   "/embeddings",
				StatusCode: 429,
				RetryAfter: "2",
				Message:    "Rate limit exceeded for sk-secret",
			}},
			&recordingGenerator{answer: "ok"},
		),
	})

	_, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected resource exhausted, got %v", err)
	}
	if got := status.Convert(err).Message(); got != "OpenRouter rate limit exceeded. Try again in 2 seconds." {
		t.Fatalf("unexpected error message %q", got)
	}
}

func TestAnswerToProtoMapsUsageTimelineArtifact(t *testing.T) {
	startedAt := time.Date(2026, 6, 13, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)

	resp := answerToProto(&semantic.Answer{
		Answer: "You spent the day in Chrome.",
		Model:  "test-model",
		Artifacts: []semantic.Artifact{{
			Type: semantic.ArtifactTypeUsageTimeline,
			UsageTimeline: &semantic.UsageTimeline{
				StartedAt:            startedAt,
				EndedAt:              endedAt,
				TimeZone:             "UTC",
				TotalDurationSeconds: 3600,
				TotalEventCount:      1,
				Events: []semantic.UsageTimelineEvent{{
					TransitionEventID: 42,
					Title:             "Docs",
					SourceName:        "Google Chrome",
					SourceType:        "browser",
					StartedAt:         startedAt,
					EndedAt:           startedAt.Add(time.Hour),
					DurationSeconds:   3600,
					URLHost:           "example.com",
				}},
			},
		}},
	}, semantic.IndexStatus{})

	if len(resp.GetArtifacts()) != 1 {
		t.Fatalf("expected one artifact, got %#v", resp.GetArtifacts())
	}
	timeline := resp.GetArtifacts()[0].GetUsageTimeline()
	if timeline == nil {
		t.Fatalf("expected usage timeline artifact, got %#v", resp.GetArtifacts()[0])
	}
	if timeline.GetTotalDurationSeconds() != 3600 || timeline.GetEvents()[0].GetSourceName() != "Google Chrome" {
		t.Fatalf("unexpected timeline artifact %#v", timeline)
	}
}

func TestSemanticIndexStatusIncludesUnavailableReason(t *testing.T) {
	server := newTestServer(testServerParams{
		UnavailableReason: "OpenRouter API key is invalid or expired.",
	})

	resp, err := server.GetSemanticIndexStatus(context.Background(), &amav1.GetSemanticIndexStatusRequest{})
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if resp.GetState() != amav1.SemanticIndexStatus_UNAVAILABLE {
		t.Fatalf("expected unavailable state, got %v", resp.GetState())
	}
	if resp.GetMessage() != "OpenRouter API key is invalid or expired." {
		t.Fatalf("unexpected unavailable message %q", resp.GetMessage())
	}
}

func TestSemanticIndexStatusPreservesCountsWhenAnsweringUnavailable(t *testing.T) {
	server := newTestServer(testServerParams{
		IndexStatus: &recordingIndexStatusProvider{status: semantic.IndexStatus{
			State:               semantic.IndexStateReady,
			CompletedEventCount: 10,
			IndexedEventCount:   8,
			PendingEventCount:   2,
			Message:             "Semantic index is ready.",
		}},
		UnavailableReason: "OpenRouter provider is temporarily unavailable.",
	})

	resp, err := server.GetSemanticIndexStatus(context.Background(), &amav1.GetSemanticIndexStatusRequest{})
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if resp.GetState() != amav1.SemanticIndexStatus_UNAVAILABLE {
		t.Fatalf("expected unavailable state, got %v", resp.GetState())
	}
	if resp.GetMessage() != "OpenRouter provider is temporarily unavailable." {
		t.Fatalf("unexpected unavailable message %q", resp.GetMessage())
	}
	if resp.GetCompletedEventCount() != 10 || resp.GetIndexedEventCount() != 8 || resp.GetPendingEventCount() != 2 {
		t.Fatalf("expected counts to be preserved, got %+v", resp)
	}
}

func TestSemanticIndexStatusProviderErrorReturnsUnavailableStatus(t *testing.T) {
	server := newTestServer(testServerParams{
		IndexStatus: &recordingIndexStatusProvider{err: fmt.Errorf("sqlite status failed")},
	})

	resp, err := server.GetSemanticIndexStatus(context.Background(), &amav1.GetSemanticIndexStatusRequest{})
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if resp.GetState() != amav1.SemanticIndexStatus_UNAVAILABLE {
		t.Fatalf("expected unavailable state, got %v", resp.GetState())
	}
	if resp.GetMessage() != "Semantic index status is unavailable." {
		t.Fatalf("unexpected unavailable message %q", resp.GetMessage())
	}
}

func TestAskSurvivesSemanticIndexStatusProviderError(t *testing.T) {
	searcher := &recordingSearcher{
		results: []semantic.SearchResult{
			{TransitionEventID: 42, Content: "Application: Calendar", Distance: 0.125},
		},
	}
	server := newTestServer(testServerParams{
		Answerer:    semantic.NewAnswerer(searcher, &recordingGenerator{answer: "You used Calendar."}),
		IndexStatus: &recordingIndexStatusProvider{err: fmt.Errorf("sqlite status failed")},
	})

	resp, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "What did I do today?"})
	if err != nil {
		t.Fatalf("ask: %v", err)
	}
	if resp.GetAnswer() != "You used Calendar." {
		t.Fatalf("unexpected answer %q", resp.GetAnswer())
	}
	if resp.GetSemanticIndexStatus().GetState() != amav1.SemanticIndexStatus_UNAVAILABLE {
		t.Fatalf("expected unavailable attached status, got %v", resp.GetSemanticIndexStatus().GetState())
	}
	if resp.GetSemanticIndexStatus().GetMessage() != "Semantic index status is unavailable." {
		t.Fatalf("unexpected attached status message %q", resp.GetSemanticIndexStatus().GetMessage())
	}
}

type testServerParams struct {
	Answerer          *semantic.Answerer
	Backfilling       BackfillCoordinator
	IndexStatus       IndexStatusProvider
	UnavailableReason string
}

func newTestServer(params testServerParams) *Server {
	registry := services.New()
	semantic.RegisterRuntime(registry, &semantic.Runtime{
		Answerer:          params.Answerer,
		UnavailableReason: params.UnavailableReason,
	})
	server := NewServer(registry)
	server.backfilling = params.Backfilling
	server.indexStatus = params.IndexStatus
	return server
}

type recordingSearcher struct {
	query   string
	k       int64
	results []semantic.SearchResult
	err     error
}

func (r *recordingSearcher) Search(_ context.Context, query string, k int64) ([]semantic.SearchResult, error) {
	r.query = query
	r.k = k
	return r.results, r.err
}

type recordingGenerator struct {
	answer string
	err    error
}

func (r *recordingGenerator) Model() string { return "test-generator" }

func (r *recordingGenerator) Generate(context.Context, string) (string, error) {
	return r.answer, r.err
}

type recordingAmaBackfillCoordinator struct {
	hasMissing      bool
	hasMissingCalls int
	startCalls      int
	limit           int64
	err             error
}

func (r *recordingAmaBackfillCoordinator) HasMissing(context.Context) (bool, error) {
	r.hasMissingCalls++
	return r.hasMissing, r.err
}

func (r *recordingAmaBackfillCoordinator) Start(limit int64) bool {
	r.startCalls++
	r.limit = limit
	return true
}

type recordingIndexStatusProvider struct {
	status semantic.IndexStatus
	err    error
}

func (r *recordingIndexStatusProvider) Status(context.Context) (semantic.IndexStatus, error) {
	return r.status, r.err
}
