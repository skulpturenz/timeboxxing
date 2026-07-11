package envs

import (
	"os"
	"testing"

	enumsenv "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_env"
)

func TestRuntimeEnvironmentDefaultsToDevelopment(t *testing.T) {
	unsetEnv(t, "GO_ENV")

	if got := GO_ENV.Value(); got != enumsenv.Development.String() {
		t.Fatalf("expected development default, got %q", got)
	}
}

func TestRuntimeEnvironmentAcceptsSupportedValues(t *testing.T) {
	cases := map[string]enumsenv.Environment{
		"production":  enumsenv.Production,
		"development": enumsenv.Development,
		"test":        enumsenv.Test,
		"local":       enumsenv.Local,
	}

	for value, expected := range cases {
		t.Run(value, func(t *testing.T) {
			t.Setenv("GO_ENV", value)

			if got, err := enumsenv.Parse(GO_ENV.Value()); err != nil || got != expected {
				t.Fatalf("expected %q, got %q", expected, got)
			}
		})
	}
}

func TestRuntimeEnvironmentRejectsUnsupportedValues(t *testing.T) {
	t.Setenv("GO_ENV", "staging")

	assertPanics(t, func() {
		_ = GO_ENV.Value()
	})
}

func TestSentryDSNDefaultsToPlaceholder(t *testing.T) {
	unsetEnv(t, "SIDECAR_SENTRY_DSN")

	if got := SENTRY_DSN.Value(); got != PlaceholderSidecarSentryDSN {
		t.Fatalf("expected placeholder DSN %q, got %q", PlaceholderSidecarSentryDSN, got)
	}
}

func TestSentryDSNUsesOverride(t *testing.T) {
	t.Setenv("SIDECAR_SENTRY_DSN", "https://public@example.com/42")

	if got := SENTRY_DSN.Value(); got != "https://public@example.com/42" {
		t.Fatalf("expected DSN override, got %q", got)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
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
