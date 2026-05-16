package outbound

import "context"

// TagExtractor extracts structured tags from task text using an LLM.
// Tags are used to improve retrieval precision by filtering tools and knowledge
// before embedding-based similarity search.
type TagExtractor interface {
	// ExtractTags extracts tags from the task text.
	// When availableTags is non-empty, the LLM is constrained to select from those known tags.
	// When availableTags is empty, the LLM generates tags freely (legacy behavior).
	ExtractTags(ctx context.Context, taskText string, availableTags []string) ([]string, error)
}