package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/zalando/go-keyring"
)

type CommandSetSecret struct {
	Namespace string
	Key       string
	Value     string
	Overwrite bool
}

//nolint:revive // command/query shape is uniform; svcs is threaded through even where unused
func (c CommandSetSecret) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	if !c.Overwrite {
		_, err := keyring.Get(c.Namespace, c.Key)
		if err == nil {
			return fmt.Errorf("set secret %s/%s: %w", c.Namespace, c.Key, ErrSecretExists)
		}

		if !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("set secret %s/%s: read existing: %w", c.Namespace, c.Key, err)
		}
	}

	if err := keyring.Set(c.Namespace, c.Key, c.Value); err != nil {
		return fmt.Errorf("set secret %s/%s: %w", c.Namespace, c.Key, err)
	}

	return nil
}
