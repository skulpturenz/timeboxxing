package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygrpc "github.com/getsentry/sentry-go/grpc"
	sentryslog "github.com/getsentry/sentry-go/slog"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	"google.golang.org/grpc"
)

const (
	defaultRelease         = "timeboxxing-sidecar@1.0.0"
	sentryTracesSampleRate = 0.2
	sentryFlushTimeout     = 2 * time.Second
)

type SentryConfig struct {
	DSN              string
	Release          string
	Environment      string
	SendDefaultPII   bool
	EnableTracing    bool
	TracesSampleRate float64
	LogsEnabled      bool
	LogLevels        []slog.Level
	Tags             map[string]string
}

func InitSentry(config SentryConfig) (func(), error) {
	if err := sentry.Init(config.ClientOptions()); err != nil {
		return func() {}, err
	}
	return func() {
		sentry.Flush(sentryFlushTimeout)
	}, nil
}

func SentryConfigFromEnv() SentryConfig {
	goEnv := envs.GO_ENV.Value()
	return SentryConfig{
		DSN:              envs.SENTRY_DSN.Value(),
		Release:          trimmedValue(os.Getenv("SENTRY_RELEASE"), defaultRelease),
		Environment:      string(goEnv),
		SendDefaultPII:   false,
		EnableTracing:    true,
		TracesSampleRate: sentryTracesSampleRate,
		LogsEnabled:      true,
		LogLevels:        []slog.Level{slog.LevelWarn, slog.LevelError, sentryslog.LevelFatal},
		Tags: map[string]string{
			"process": "sidecar",
			"go_env":  string(goEnv),
		},
	}
}

func (c SentryConfig) ClientOptions() sentry.ClientOptions {
	return sentry.ClientOptions{
		Dsn:              c.DSN,
		Release:          c.Release,
		Environment:      c.Environment,
		SendDefaultPII:   c.SendDefaultPII,
		EnableTracing:    c.EnableTracing,
		TracesSampleRate: c.TracesSampleRate,
		DisableLogs:      !c.LogsEnabled,
		Tags:             c.Tags,
	}
}

func NewSentryLogHandler(ctx context.Context, config SentryConfig) slog.Handler {
	return sentryslog.Option{
		LogLevel:  config.LogLevels,
		AddSource: true,
	}.NewSentryHandler(ctx)
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return sentrygrpc.UnaryServerInterceptor(sentrygrpc.ServerOptions{
		Repanic: true,
	})
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return sentrygrpc.StreamServerInterceptor(sentrygrpc.ServerOptions{
		Repanic: true,
	})
}

func CaptureException(err error) {
	if err != nil {
		sentry.CaptureException(err)
	}
}

func trimmedValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
