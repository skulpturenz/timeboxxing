package semantic

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type CommandIngest struct{}

func (c CommandIngest) Exec(ctx context.Context, _ *services.Services[any, any]) {}
