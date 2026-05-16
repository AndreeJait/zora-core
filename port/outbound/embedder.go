package outbound

import "context"

// Embedder defines the outbound port for text-to-vector embedding.
type Embedder interface {
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}
