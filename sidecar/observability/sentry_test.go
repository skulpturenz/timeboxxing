package observability

import (
	"log/slog"
	"os"
	"reflect"
	"testing"

	sentryslog "github.com/getsentry/sentry-go/slog"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
)

func TestSentryConfigUsesFerriteDefaults(t *testing.T) {
	unsetEnv(t, "GO_ENV", "SIDECAR_SENTRY_DSN", "SENTRY_RELEASE")

	config := SentryConfigFromEnv()

	if config.DSN != envs.PlaceholderSidecarSentryDSN {
		t.Fatalf("expected placeholder DSN %q, got %q", envs.PlaceholderSidecarSentryDSN, config.DSN)
	}
	if config.Environment != "production" {
		t.Fatalf("expected production environment, got %q", config.Environment)
	}
}

func TestSentryConfigUsesFerriteOverrides(t *testing.T) {
	t.Setenv("SIDECAR_SENTRY_DSN", "https://public@example.com/42")
	t.Setenv("SENTRY_RELEASE", " timeboxxing-sidecar@2.0.0 ")
	t.Setenv("GO_ENV", "local")

	config := SentryConfigFromEnv()

	if config.DSN != "https://public@example.com/42" {
		t.Fatalf("expected DSN override, got %q", config.DSN)
	}
	if config.Release != "timeboxxing-sidecar@2.0.0" {
		t.Fatalf("expected release override, got %q", config.Release)
	}
	if config.Environment != "local" {
		t.Fatalf("expected environment override, got %q", config.Environment)
	}
}

func TestSentryConfigSetsPrivacyTracingAndLogDefaults(t *testing.T) {
	unsetEnv(t, "GO_ENV", "SIDECAR_SENTRY_DSN", "SENTRY_RELEASE")

	config := SentryConfigFromEnv()

	if config.SendDefaultPII {
		t.Fatal("expected default PII collection to be disabled")
	}
	if !config.EnableTracing {
		t.Fatal("expected tracing to be enabled")
	}
	if config.TracesSampleRate != 0.2 {
		t.Fatalf("expected traces sample rate 0.2, got %f", config.TracesSampleRate)
	}
	if !config.LogsEnabled {
		t.Fatal("expected logs to be enabled")
	}
	expectedLogLevels := []slog.Level{slog.LevelWarn, slog.LevelError, sentryslog.LevelFatal}
	if !reflect.DeepEqual(expectedLogLevels, config.LogLevels) {
		t.Fatalf("expected log levels %v, got %v", expectedLogLevels, config.LogLevels)
	}
	if config.Tags["process"] != "sidecar" {
		t.Fatalf("expected sidecar process tag, got %q", config.Tags["process"])
	}
	if config.Tags["go_env"] != "production" {
		t.Fatalf("expected production go_env tag, got %q", config.Tags["go_env"])
	}
}

func TestClientOptionsReflectConfig(t *testing.T) {
	unsetEnv(t, "GO_ENV", "SIDECAR_SENTRY_DSN", "SENTRY_RELEASE")

	config := SentryConfigFromEnv()

	options := config.ClientOptions()

	if options.Dsn != config.DSN {
		t.Fatalf("expected options DSN %q, got %q", config.DSN, options.Dsn)
	}
	if options.SendDefaultPII != config.SendDefaultPII {
		t.Fatalf("expected SendDefaultPII %v, got %v", config.SendDefaultPII, options.SendDefaultPII)
	}
	if options.EnableTracing != config.EnableTracing {
		t.Fatalf("expected EnableTracing %v, got %v", config.EnableTracing, options.EnableTracing)
	}
	if options.TracesSampleRate != config.TracesSampleRate {
		t.Fatalf("expected TracesSampleRate %f, got %f", config.TracesSampleRate, options.TracesSampleRate)
	}
	if options.DisableLogs {
		t.Fatal("expected Sentry logs to be enabled")
	}
}

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		key := key
		previous, hadPrevious := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if hadPrevious {
				_ = os.Setenv(key, previous)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}
