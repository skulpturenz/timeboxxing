package semantic

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type QuerySearch struct{}

func (q QuerySearch) Exec(ctx context.Context, _ *services.Services[any, any]) {}
