package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// ConversationRepository defines the outbound port for persisting conversations.
type ConversationRepository interface {
	// Save persists or updates a conversation.
	Save(ctx context.Context, conv *entity.Conversation) error

	// GetBySessionID retrieves a conversation by its session identifier.
	GetBySessionID(ctx context.Context, sessionID string) (*entity.Conversation, error)

	// Delete removes a conversation by session ID.
	Delete(ctx context.Context, sessionID string) error
}
