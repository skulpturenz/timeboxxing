package semantic

import (
	"context"
	"log/slog"
	"sync"
)

type BackfillCoordinator struct {
	ctx        context.Context
	backfiller *Backfiller
	logger     *slog.Logger

	mu         sync.Mutex
	running    bool
	lastResult BackfillResult
	lastError  string
}

type BackfillStatus struct {
	Running    bool
	LastResult BackfillResult
	LastError  string
}

func NewBackfillCoordinator(ctx context.Context, backfiller *Backfiller, logger *slog.Logger) *BackfillCoordinator {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &BackfillCoordinator{
		ctx:        ctx,
		backfiller: backfiller,
		logger:     logger,
	}
}

func (c *BackfillCoordinator) HasMissing(ctx context.Context) (bool, error) {
	if c == nil || c.backfiller == nil {
		return false, nil
	}
	return c.backfiller.HasMissing(ctx)
}

func (c *BackfillCoordinator) Start(limit int64) bool {
	if c == nil || c.backfiller == nil {
		return false
	}

	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return false
	}
	c.running = true
	c.mu.Unlock()

	go func() {
		result, err := c.backfiller.BackfillMissing(c.ctx, limit)
		lastError := ""
		if err != nil {
			lastError = safeBackfillStatusMessage(err)
			c.logger.ErrorContext(c.ctx, "semantic backfill failed", "error", err)
		} else {
			c.logger.InfoContext(c.ctx, "semantic backfill completed",
				"checked", result.Checked,
				"enqueued", result.Enqueued,
				"failed", result.Failed,
			)
		}

		c.mu.Lock()
		c.lastResult = result
		c.lastError = lastError
		c.running = false
		c.mu.Unlock()
	}()

	return true
}

func (c *BackfillCoordinator) Status() BackfillStatus {
	if c == nil {
		return BackfillStatus{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return BackfillStatus{
		Running:    c.running,
		LastResult: c.lastResult,
		LastError:  c.lastError,
	}
}

func safeBackfillStatusMessage(err error) string {
	if err == nil {
		return ""
	}
	if message, ok := AIRequestUserMessage(err); ok {
		return message
	}
	return "Semantic backfill failed. Check sidecar logs."
}
