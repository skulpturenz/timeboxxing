package main

//go:generate go tool buf generate

import (
	"context"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"github.com/skulpturenz/timeboxxing/sidecar/app"
)

func main() {
	ctx := context.Background()
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{}))
	slog.SetDefault(logger)

	if err := app.Run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "run sidecar", "error", err)
		panic(err)
	}
}
