package semantic

import "context"

type Embedder interface {
	Model() string
	Dimension() int
	Embed(ctx context.Context, input string) ([]float32, error)
}

type Generator interface {
	Model() string
	Generate(ctx context.Context, prompt string) (string, error)
}
