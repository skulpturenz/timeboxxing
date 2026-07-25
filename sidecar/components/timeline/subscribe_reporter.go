package timeline

import (
	"context"

	"github.com/negrel/assert"
	"github.com/skulpturenz/timeboxxing/sidecar/services"
)

type CommandSubscribeReporter struct {
	Chan          <-chan ForegroundProcess
	activeProcess *ForegroundProcess
}

func (c *CommandSubscribeReporter) Exec(ctx context.Context, svcs *services.Services[any, any]) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case fp := <-c.Chan:
			cmd := CommandUpsertForegroundProcess{
				PreviousProcess: c.activeProcess,
				ActiveProcess:   fp,
			}

			err := cmd.Exec(ctx, svcs)
			assert.NoError(err)
			if err != nil {
				return err
			}

			c.activeProcess = &fp
		}
	}
}
