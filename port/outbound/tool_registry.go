package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// ToolSearchQuery defines parameters for semantic tool retrieval.
type ToolSearchQuery struct {
	Embedding []float64 // vector embedding of the task text
	Tags      []string  // extracted tags for hybrid retrieval
	UserID    string    // user identity for scoped retrieval
	Limit     int       // maximum number of results
}

// TagInfo represents a tag from the tool registry with its ID, name, and description.
type TagInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolRegistryClient defines the outbound port for semantic tool retrieval
// and execution via the MCP server.
type ToolRegistryClient interface {
	// SearchTools retrieves tools matching the query parameters.
	// Results are filtered by tags (if provided), scoped to the user (if provided),
	// and sorted by combined vector similarity + tag match score.
	SearchTools(ctx context.Context, query ToolSearchQuery) ([]entity.ToolContext, error)

	// CallTool invokes a specific tool by name with the given arguments.
	CallTool(ctx context.Context, name string, arguments string) (entity.ToolResult, error)

	// ListTags retrieves all available tags from the tool registry.
	// Used to constrain LLM tag extraction to known tags.
	ListTags(ctx context.Context) ([]TagInfo, error)
}