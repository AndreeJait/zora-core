package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// KnowledgeSearchQuery defines parameters for semantic knowledge retrieval.
type KnowledgeSearchQuery struct {
	Embedding []float64 // vector embedding of the task text
	Tags      []string  // extracted tags for hybrid retrieval
	UserID    string    // user identity for scoped retrieval
	IsAdmin   bool      // whether to include admin-level knowledge
	Limit     int       // maximum number of results
}

// KnowledgeClient defines the outbound port for semantic knowledge retrieval
// from the zora-knowledge service.
type KnowledgeClient interface {
	// SearchDocs retrieves knowledge documents matching the query parameters.
	// Results are filtered by tags (if provided), scoped to the user (if provided),
	// and include admin-level docs when IsAdmin is true.
	SearchDocs(ctx context.Context, query KnowledgeSearchQuery) ([]entity.KnowledgeSnippet, error)
}