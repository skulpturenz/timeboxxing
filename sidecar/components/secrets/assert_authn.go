package secrets

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

func AssertAuthenticated(ctx context.Context, svcs *services.Services[any, any]) {
	query := QueryIsAuthenticated{}
	ok := query.Exec(ctx, svcs)

	if !ok {
		panic("unable to start sidecar")
	}
}
