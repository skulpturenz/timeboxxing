package timeline

import (
	"context"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/skulpturenz/timeboxxing/sidecar/utils"
)

type QueryGetForegroundProcesses struct {
	lastItemId int
}

func (q *QueryGetForegroundProcesses) Stream(ctx context.Context, svcs *services.Services[any, any]) utils.StreamFn[any] {
	return func(ctx context.Context, pageSize int) []any {
		return []any{}
	}
}
