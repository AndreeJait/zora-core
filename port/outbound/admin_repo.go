package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// AdminRepository defines the outbound port for admin persistence.
type AdminRepository interface {
	// Add creates or updates an admin entry.
	Add(ctx context.Context, entry *entity.AdminEntry) error
	// Remove deletes an admin entry by phone number.
	Remove(ctx context.Context, phone string) error
	// GetByPhone looks up an admin entry by phone. Returns nil if not found.
	GetByPhone(ctx context.Context, phone string) (*entity.AdminEntry, error)
	// GetByLID looks up an admin entry by LID. Returns nil if not found.
	GetByLID(ctx context.Context, lid string) (*entity.AdminEntry, error)
	// List returns all admin entries.
	List(ctx context.Context) ([]entity.AdminEntry, error)
}