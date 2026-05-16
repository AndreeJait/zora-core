package setting

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements outbound.SettingRepository using GORM.
type Repository struct {
	db *gorm.DB
}

var _ outbound.SettingRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, key string) (*entity.Setting, error) {
	var s entity.Setting
	if err := r.db.WithContext(ctx).Where("key = ?", key).First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get setting: %w", err)
	}
	return &s, nil
}

func (r *Repository) GetAll(ctx context.Context) ([]entity.Setting, error) {
	var settings []entity.Setting
	if err := r.db.WithContext(ctx).Order("key ASC").Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	return settings, nil
}

func (r *Repository) Upsert(ctx context.Context, s *entity.Setting) error {
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "description", "updated_at"}),
	}).Create(s).Error; err != nil {
		return fmt.Errorf("upsert setting: %w", err)
	}
	return nil
}