package timeline

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryGetUnindexedForegroundProcesses struct {
	lastItemId int
}

func (q *QueryGetUnindexedForegroundProcesses) Stream(ctx context.Context, svcs *services.Services[any, any]) utils.StreamFn[any] {
	return func(ctx context.Context, pageSize int) []any {
		return []any{}
	}
}
