package semantic

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type QueryAnswer struct{}

func (q QueryAnswer) Exec(ctx context.Context, _ *services.Services[any, any]) {}
