package outbound

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// SettingRepository defines the outbound port for runtime settings.
type SettingRepository interface {
	// Get returns a setting by key. Returns nil if not found.
	Get(ctx context.Context, key string) (*entity.Setting, error)
	// GetAll returns all settings.
	GetAll(ctx context.Context) ([]entity.Setting, error)
	// Upsert creates or updates a setting.
	Upsert(ctx context.Context, setting *entity.Setting) error
}