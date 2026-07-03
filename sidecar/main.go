package main

//go:generate go tool buf generate

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/skulpturenz/timeboxxing/sidecar/app"
	"github.com/skulpturenz/timeboxxing/sidecar/observability"
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

	if err := app.Run(ctx, logger); err != nil {
		observability.CaptureException(err)
		logger.ErrorContext(ctx, "run sidecar", "error", err)
		panic(err)
	}
}
