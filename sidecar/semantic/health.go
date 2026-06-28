package semantic

import (
	"context"
	"fmt"
)

func CheckEmbedderHealth(ctx context.Context, embedder Embedder) error {
	if embedder == nil {
		return fmt.Errorf("embedder is required")
	}
	embedding, err := embedder.Embed(ctx, "timeboxxing semantic health check")
	if err != nil {
		return fmt.Errorf("semantic embedding health check failed: %w", err)
	}
	if len(embedding) != embedder.Dimension() {
		return fmt.Errorf("semantic embedding health check dimension mismatch: got %d, want %d", len(embedding), embedder.Dimension())
	}
	return nil
}
