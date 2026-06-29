package session

import (
	"context"
	"testing"
	"time"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/idle"
	"github.com/skulpturenz/timeboxxing/sidecar/monitor/platform"
)

// fakeTracker returns a fixed WindowInfo.
type fakeTracker struct {
	infos []platform.WindowInfo
	pos   int
}

func (f *fakeTracker) Poll(context.Context) (platform.WindowInfo, error) {
	if f.pos >= len(f.infos) {
		return f.infos[len(f.infos)-1], nil
	}
	info := f.infos[f.pos]
	f.pos++
	return info, nil
}

func (f *fakeTracker) Permissions() []platform.PermissionStatus { return nil }

func win(app, title string) platform.WindowInfo {
	return platform.WindowInfo{
		AppName:     app,
		WindowTitle: title,
		TitleSource: platform.TitleSourceWindowAPI,
		Timestamp:   time.Now(),
	}
}

func winWithIdentity(app, title string, identifier string, path string) platform.WindowInfo {
	info := win(app, title)
	info.AppIdentifier = identifier
	info.AppPath = path
	return info
}

func drain(ch <-chan Transition) []Transition {
	var out []Transition
	for t := range ch {
		out = append(out, t)
	}
	return out
}

func TestManagerFirstSession(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{win("VSCode", "main.go")}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	if len(transitions) == 0 {
		t.Fatal("expected at least one transition")
	}
	first := transitions[0]
	if first.From != nil {
		t.Errorf("first transition From should be nil, got %v", first.From)
	}
	if first.To == nil || first.To.Key.AppName != "VSCode" {
		t.Errorf("first transition To should be VSCode, got %v", first.To)
	}
	if first.Reason != ReasonStart {
		t.Errorf("first transition reason should be %q, got %q", ReasonStart, first.Reason)
	}
}

func TestManagerFinalizesForegroundAppNameFromPath(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{
		{
			AppPath:     "/Applications/Safari.app",
			TitleSource: platform.TitleSourceWindowAPI,
			Timestamp:   time.Now(),
		},
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	if len(transitions) == 0 {
		t.Fatal("expected finalized foreground transition")
	}
	first := transitions[0]
	if first.To == nil || first.To.Key.AppName != "Safari" {
		t.Fatalf("expected finalized app name Safari, got %#v", first.To)
	}
}

func TestManagerIgnoresEmptyForegroundSample(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{{Timestamp: time.Now()}}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	if len(transitions) != 0 {
		t.Fatalf("expected no transitions for empty sample, got %#v", transitions)
	}
}

func TestManagerFocusChange(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{
		win("VSCode", "main.go"),
		win("VSCode", "main.go"),
		win("Google Chrome", "GitHub - Google Chrome"),
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	// Find the focus_change transition.
	found := false
	for _, tr := range transitions {
		if tr.Reason == ReasonFocusChange {
			found = true
			if tr.From == nil || tr.From.Key.AppName != "VSCode" {
				t.Errorf("focus_change From should be VSCode, got %v", tr.From)
			}
			if tr.To == nil || tr.To.Key.AppName != "Google Chrome" {
				t.Errorf("focus_change To should be Google Chrome, got %v", tr.To)
			}
		}
	}
	if !found {
		t.Error("expected a focus_change transition, none found")
	}
}

func TestManagerApplicationIdentityChangeDoesNotCreateTransition(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{
		winWithIdentity("VSCode", "main.go", "com.microsoft.VSCode", "/Applications/Visual Studio Code.app"),
		winWithIdentity("VSCode", "main.go", "", ""),
		winWithIdentity("VSCode", "main.go", "com.microsoft.VSCode", "/Applications/Visual Studio Code.app"),
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	for _, tr := range transitions {
		if tr.Reason == ReasonFocusChange || tr.Reason == ReasonTabChange {
			t.Fatalf("expected identity changes to be ignored as session keys, got %#v", tr)
		}
	}
	if len(transitions) == 0 || transitions[0].To == nil {
		t.Fatal("expected initial session transition")
	}
	if transitions[0].To.ApplicationIdentity.Path != "/Applications/Visual Studio Code.app" {
		t.Fatalf("expected first session identity to be preserved, got %#v", transitions[0].To.ApplicationIdentity)
	}
}

func TestManagerBrowserTabChange(t *testing.T) {
	tracker := &fakeTracker{infos: []platform.WindowInfo{
		win("Google Chrome", "Gmail - Google Chrome"),
		win("Google Chrome", "Gmail - Google Chrome"),
		win("Google Chrome", "GitHub - Google Chrome"),
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  0,
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	found := false
	for _, tr := range transitions {
		if tr.Reason == ReasonTabChange {
			found = true
			if tr.From.Key.TabTitle != "Gmail" {
				t.Errorf("tab_change From.TabTitle should be Gmail, got %q", tr.From.Key.TabTitle)
			}
			if tr.To.Key.TabTitle != "GitHub" {
				t.Errorf("tab_change To.TabTitle should be GitHub, got %q", tr.To.Key.TabTitle)
			}
		}
	}
	if !found {
		t.Error("expected a tab_change transition, none found")
	}
}

func TestManagerMinDurationDiscards(t *testing.T) {
	// Switch happens faster than minDuration → should NOT appear in stats.
	tracker := &fakeTracker{infos: []platform.WindowInfo{
		win("App1", ""),
		win("App2", ""), // very brief
		win("App3", ""), // stays
		win("App3", ""),
		win("App3", ""),
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:  10 * time.Second, // very high threshold
		IdleDetector: idle.Nop(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	drain(mgr.Transitions)

	stats := mgr.Stats()
	// App1 and App2 are sub-threshold so they must NOT appear in stats.
	for k := range stats {
		if k.AppName == "App1" || k.AppName == "App2" {
			t.Errorf("sub-threshold app %q should not appear in stats", k.AppName)
		}
	}
}

func TestManagerIdleSession(t *testing.T) {
	idleSeconds := 0.0
	type fakeIdle struct{ secs *float64 }
	fi := &fakeIdle{secs: &idleSeconds}
	// We need a fakeIdle that implements idle.IdleDetector.
	// Use a closure-based wrapper below.

	_ = fi // keep linter happy; we use the direct value below.

	type spikeIdle struct{ secs float64 }
	// inline fake to avoid importing a test helper package
	type inlineIdleDetector struct {
		calls int
	}
	_ = inlineIdleDetector{}

	// Simple inline implementation: first 2 polls active, then idle.
	callCount := 0
	var detector idle.IdleDetector = idleFunc(func() (float64, error) {
		callCount++
		if callCount > 2 {
			return 3600, nil // very idle
		}
		return 0, nil
	})

	tracker := &fakeTracker{infos: []platform.WindowInfo{
		win("VSCode", ""),
		win("VSCode", ""),
		win("VSCode", ""),
		win("VSCode", ""),
		win("VSCode", ""),
	}}
	mgr := NewManager(ManagerConfig{
		MinDuration:   0,
		IdleThreshold: time.Second, // small threshold for test
		IdleDetector:  detector,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	go mgr.Run(ctx, tracker, 50*time.Millisecond)
	transitions := drain(mgr.Transitions)

	foundIdle := false
	for _, tr := range transitions {
		if tr.Reason == ReasonIdle {
			foundIdle = true
			if !tr.To.Key.IsIdle {
				t.Error("idle transition To.Key.IsIdle should be true")
			}
		}
	}
	if !foundIdle {
		t.Error("expected an idle transition, none found")
	}
}

// idleFunc is a function that implements idle.IdleDetector.
type idleFunc func() (float64, error)

func (f idleFunc) SecondsSinceLastInput(context.Context) (float64, error) { return f() }
