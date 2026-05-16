package outbound

import (
	"context"
	"fmt"

	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// EmbedderAdapter wraps llmw.Embedder to implement outbound.Embedder.
type EmbedderAdapter struct {
	embedder llmw.Embedder
}

var _ outbound.Embedder = (*EmbedderAdapter)(nil)

func NewEmbedderAdapter(embedder llmw.Embedder) *EmbedderAdapter {
	return &EmbedderAdapter{embedder: embedder}
}

func (e *EmbedderAdapter) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings, err := e.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	return embeddings, nil
}
