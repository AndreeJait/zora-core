package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// MessageStore persists WhatsApp messages for reply chain resolution.
type MessageStore interface {
	// Store saves a message by its WhatsApp message ID with a TTL.
	Store(ctx context.Context, msg *entity.WAHAMessage) error

	// Get retrieves a message by its WhatsApp message ID.
	// Returns nil if not found or expired.
	Get(ctx context.Context, messageID string) (*entity.WAHAMessage, error)
}