package outbound

import (
	"context"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// WhitelistRepository defines the outbound port for whitelist persistence.
type WhitelistRepository interface {
	// Add creates or updates a whitelist entry.
	Add(ctx context.Context, entry *entity.WhitelistEntry) error
	// Remove deletes a whitelist entry by phone number.
	Remove(ctx context.Context, phone string) error
	// GetByPhone looks up a whitelist entry by phone. Returns nil if not found.
	GetByPhone(ctx context.Context, phone string) (*entity.WhitelistEntry, error)
	// GetByLID looks up a whitelist entry by LID. Returns nil if not found.
	GetByLID(ctx context.Context, lid string) (*entity.WhitelistEntry, error)
	// List returns all whitelist entries.
	List(ctx context.Context) ([]entity.WhitelistEntry, error)
	// IncrementUsage increments token usage for the given hour window.
	// Returns the new total tokens used in the window.
	IncrementUsage(ctx context.Context, phone string, windowStart time.Time) (int, error)
	// GetUsage returns tokens used in the given hour window.
	GetUsage(ctx context.Context, phone string, windowStart time.Time) (int, error)
	// CleanupExpiredTokens removes token usage records older than the given time.
	CleanupExpiredTokens(ctx context.Context, before time.Time) error
}