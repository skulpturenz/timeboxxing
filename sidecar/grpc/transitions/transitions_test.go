package transitions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	componentTransitions "github.com/skulpturenz/timeboxxing/sidecar/components/transitions"
	"github.com/skulpturenz/timeboxxing/sidecar/db"
	transitionsv1 "github.com/skulpturenz/timeboxxing/sidecar/gen/transitions/v1"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/reporter"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/session"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetTransitionEventsMapsComponentEvents(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service := componentTransitions.NewService(componentTransitions.NewServiceParams{Querier: database.Querier})
	server := NewServer(NewServerParams{Transitions: service})
	startedAt := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	id := createTransitionEvent(t, ctx, database, "Google Chrome", "Docs", startedAt)

	resp, err := server.GetTransitionEvents(ctx, &transitionsv1.GetTransitionEventsRequest{StartedAt: timestamppb.New(startedAt)})
	if err != nil {
		t.Fatalf("get transition events: %v", err)
	}
	events := resp.GetTransitionEvents()
	if len(events) != 1 {
		t.Fatalf("expected one transition event, got %d", len(events))
	}
	if events[0].GetId() != id || events[0].GetApplicationName() != "Google Chrome" || events[0].GetTab() != "Docs" {
		t.Fatalf("unexpected transition event: %#v", events[0])
	}
}

func TestGetTransitionEventStreamSendsComponentEvents(t *testing.T) {
	ctx := context.Background()
	database := newTestDatabase(t, ctx)
	service := componentTransitions.NewService(componentTransitions.NewServiceParams{Querier: database.Querier})
	server := NewServer(NewServerParams{Transitions: service})
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream := newRecordingStream(streamCtx)
	done := make(chan error, 1)
	go func() {
		done <- server.GetTransitionEventStream(&transitionsv1.GetTransitionEventsRequest{}, stream)
	}()

	id := createTransitionEvent(t, ctx, database, "Google Chrome", "Docs", time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC))
	publishTicker := time.NewTicker(10 * time.Millisecond)
	defer publishTicker.Stop()
	timeout := time.After(time.Second)
	for {
		select {
		case event := <-stream.events:
			if event.GetId() != id {
				t.Fatalf("expected event id %d, got %d", id, event.GetId())
			}
			goto received
		case <-timeout:
			t.Fatal("timed out waiting for streamed transition event")
		case <-publishTicker.C:
			if err := service.PublishTransitionEvent(ctx, componentTransitions.PublishTransitionEventParams{ID: id}); err != nil {
				t.Fatalf("publish transition event: %v", err)
			}
		}
	}

received:

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream to exit")
	}
}

func newTestDatabase(t *testing.T, ctx context.Context) *db.Database {
	t.Helper()
	database, err := db.New(ctx, db.Options{
		Engine:         db.EngineSqlite,
		DataSourceName: filepath.Join(t.TempDir(), "test.db"),
	})
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	return database
}

func createTransitionEvent(t *testing.T, ctx context.Context, database *db.Database, appName string, tab string, startedAt time.Time) int64 {
	t.Helper()
	reporter := reporter.NewDatabaseReporter(database.Conn)
	id, err := reporter.Record(ctx, session.Transition{
		From: &session.Session{
			Key: session.AppKey{
				AppName:  appName,
				TabTitle: tab,
			},
			StartedAt: startedAt,
			EndedAt:   startedAt.Add(5 * time.Minute),
			Duration:  5 * time.Minute,
		},
		Reason: session.ReasonFocusChange,
	})
	if err != nil {
		t.Fatalf("record transition event: %v", err)
	}
	if id == 0 {
		t.Fatal("expected persisted transition event id")
	}

	return id
}

type recordingStream struct {
	ctx    context.Context
	events chan *transitionsv1.TransitionEvent
}

func newRecordingStream(ctx context.Context) *recordingStream {
	return &recordingStream{ctx: ctx, events: make(chan *transitionsv1.TransitionEvent, 10)}
}

func (s *recordingStream) Send(event *transitionsv1.TransitionEvent) error {
	select {
	case s.events <- event:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *recordingStream) SetHeader(metadata.MD) error  { return nil }
func (s *recordingStream) SendHeader(metadata.MD) error { return nil }
func (s *recordingStream) SetTrailer(metadata.MD)       {}
func (s *recordingStream) Context() context.Context     { return s.ctx }
func (s *recordingStream) SendMsg(any) error            { return nil }
func (s *recordingStream) RecvMsg(any) error            { return nil }
