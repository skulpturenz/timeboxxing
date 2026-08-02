package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
	"github.com/zalando/go-keyring"
)

type QueryGetSecret struct {
	Namespace string
	Key       string
}

//nolint:revive // command/query shape is uniform; svcs is threaded through even where unused
func (q QueryGetSecret) Exec(ctx context.Context, svcs *services.Services[any, any]) (*string, error) {
	value, err := keyring.Get(q.Namespace, q.Key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return nil, fmt.Errorf("get secret %s/%s: %w", q.Namespace, q.Key, err)
	}

	return utils.ZeroNil(value), nil
}
