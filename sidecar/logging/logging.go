package logging

import (
	"log/slog"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type loggerKey struct{}

func RegisterLogger(registry *services.Services[any, any], logger *slog.Logger) {
	services.Set(registry, loggerKey{}, logger)
}

func LoggerFromServices(registry *services.Services[any, any]) *slog.Logger {
	service, ok := services.Get[*slog.Logger](registry, loggerKey{})
	if !ok || service.Unwrap() == nil {
		return slog.Default()
	}
	return service.Unwrap()
}
