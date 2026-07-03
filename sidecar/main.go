package main

//go:generate go tool buf generate

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/skulpturenz/timeboxxing/sidecar/app"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	"github.com/skulpturenz/timeboxxing/sidecar/observability"
	"github.com/skulpturenz/timeboxxing/sidecar/secrets"
)

func main() {
	ctx := context.Background()
	sentryConfig := observability.SentryConfigFromEnv()
	flushSentry, sentryErr := observability.InitSentry(sentryConfig)
	defer flushSentry()

	handlers := []slog.Handler{tint.NewHandler(os.Stderr, &tint.Options{})}
	if sentryErr == nil {
		handlers = append(handlers, observability.NewSentryLogHandler(ctx, sentryConfig))
	}

	logger := slog.New(observability.NewTeeHandler(handlers...))
	slog.SetDefault(logger)
	if sentryErr != nil {
		logger.ErrorContext(ctx, "initialize sentry", "error", sentryErr)
	}

	// Receive API keys handed off by the parent process over stdin, keeping them out of the
	// environment block. Falls back to environment variables when stdin is not piped (dev runs).
	handoff := secrets.LoadFromStdin()
	envs.SetSecretOverrides(handoff.OpenRouterAPIKey, handoff.OllamaAPIKey)

	if err := app.Run(ctx, logger); err != nil {
		observability.CaptureException(err)
		logger.ErrorContext(ctx, "run sidecar", "error", err)
		panic(err)
	}
}
