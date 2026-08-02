package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/zalando/go-keyring"
)

type CommandDeleteSecret struct {
	Namespace string
	Key       string
}

//nolint:revive // command/query shape is uniform; svcs is threaded through even where unused
func (c CommandDeleteSecret) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	if err := keyring.Delete(c.Namespace, c.Key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("delete secret %s/%s: %w", c.Namespace, c.Key, err)
	}

	return nil
}
