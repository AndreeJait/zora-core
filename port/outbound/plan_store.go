package outbound

import (
	"context"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// PlanEntry holds a pending execution plan and its cached search state.
type PlanEntry struct {
	ThreadID      string
	SessionID     string
	ChatID        string
	MessageID     string
	PlanText      string
	SystemPrompt  string
	RelevantTools []entity.ToolContext
	RetrievedDocs []entity.KnowledgeSnippet
	ExtractedTags []string
	UserID        string
	IsAdmin       bool
	CreatedAt     time.Time
}

// PlanStore manages pending execution plans per chat.
type PlanStore interface {
	Save(ctx context.Context, chatID string, entry *PlanEntry) error
	Get(ctx context.Context, chatID string) (*PlanEntry, error)
	Delete(ctx context.Context, chatID string) error
}