package secrets

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/envs"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryIsAuthenticated struct{}

func (q QueryIsAuthenticated) Exec(ctx context.Context, svcs *services.Services[any, any]) bool {
	query := QueryGetSecret{Namespace: "sidecar_auth", Key: "secret"}
	secret, err := query.Exec(ctx, svcs)
	if err != nil || secret == nil {
		return false
	}

	if utils.IsZero(envs.LAUNCH_TOKEN.Value()) {
		return false
	}

	return utils.Coalesce(secret, "") == envs.LAUNCH_TOKEN.Value()
}
