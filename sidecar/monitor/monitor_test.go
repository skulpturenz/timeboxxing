package monitor

import (
	"testing"
	"time"
)

func TestResolveConfigUsesForegroundPollDefault(t *testing.T) {
	cfg := resolveConfig(Config{})

	if cfg.PollInterval != 500*time.Millisecond {
		t.Fatalf("expected default poll interval to be 500ms, got %s", cfg.PollInterval)
	}
	if cfg.MinDuration != time.Second {
		t.Fatalf("expected default min duration to stay 1s, got %s", cfg.MinDuration)
	}
	if cfg.IdleThreshold != 5*time.Minute {
		t.Fatalf("expected default idle threshold to stay 5m, got %s", cfg.IdleThreshold)
	}
}

func TestResolveConfigPreservesOverrides(t *testing.T) {
	cfg := resolveConfig(Config{
		PollInterval:  250 * time.Millisecond,
		MinDuration:   2 * time.Second,
		IdleThreshold: 10 * time.Minute,
	})

	if cfg.PollInterval != 250*time.Millisecond {
		t.Fatalf("expected custom poll interval, got %s", cfg.PollInterval)
	}
	if cfg.MinDuration != 2*time.Second {
		t.Fatalf("expected custom min duration, got %s", cfg.MinDuration)
	}
	if cfg.IdleThreshold != 10*time.Minute {
		t.Fatalf("expected custom idle threshold, got %s", cfg.IdleThreshold)
	}
}
