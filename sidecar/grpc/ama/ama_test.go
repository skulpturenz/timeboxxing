package ama

import (
	"context"
	"fmt"
	"strings"
	"testing"

	amav1 "github.com/skulpturenz/timeboxxing/sidecar/gen/ama/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/semantic"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAskReturnsAnswerAndSources(t *testing.T) {
	searcher := &recordingSearcher{
		results: []semantic.SearchResult{
			{TransitionEventID: 42, Content: "Application: Google Chrome", Distance: 0.125},
		},
	}
	server := NewServer(NewServerParams{
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
	server := NewServer(NewServerParams{
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
	server := NewServer(NewServerParams{
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
	server := NewServer(NewServerParams{
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
	server := NewServer(NewServerParams{
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
	if _, err := NewServer(NewServerParams{}).Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", err)
	}

	server := NewServer(NewServerParams{
		Answerer: semantic.NewAnswerer(&recordingSearcher{}, &recordingGenerator{answer: "ok"}),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "   "}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}

	server = NewServer(NewServerParams{
		Answerer: semantic.NewAnswerer(&recordingSearcher{err: fmt.Errorf("search failed")}, &recordingGenerator{answer: "ok"}),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal search error, got %v", err)
	}

	server = NewServer(NewServerParams{
		Answerer: semantic.NewAnswerer(
			&recordingSearcher{results: []semantic.SearchResult{{TransitionEventID: 1, Content: "Application: Calendar"}}},
			&recordingGenerator{err: fmt.Errorf("generate failed")},
		),
	})
	if _, err := server.Ask(context.Background(), &amav1.AskRequest{Question: "hello"}); status.Code(err) != codes.Internal {
		t.Fatalf("expected internal generation error, got %v", err)
	}
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
