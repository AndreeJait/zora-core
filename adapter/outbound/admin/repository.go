package admin

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements outbound.AdminRepository using GORM.
type Repository struct {
	db *gorm.DB
}

var _ outbound.AdminRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(ctx context.Context, entry *entity.AdminEntry) error {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "phone"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "lid", "scope", "chat_ids", "updated_at"}),
	}).Create(entry)
	if result.Error != nil {
		return fmt.Errorf("add admin entry: %w", result.Error)
	}
	return nil
}

func (r *Repository) Remove(ctx context.Context, phone string) error {
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).Delete(&entity.AdminEntry{}).Error; err != nil {
		return fmt.Errorf("remove admin entry: %w", err)
	}
	return nil
}

func (r *Repository) GetByPhone(ctx context.Context, phone string) (*entity.AdminEntry, error) {
	var entry entity.AdminEntry
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin entry: %w", err)
	}
	return &entry, nil
}

func (r *Repository) GetByLID(ctx context.Context, lid string) (*entity.AdminEntry, error) {
	var entry entity.AdminEntry
	if err := r.db.WithContext(ctx).Where("lid = ?", lid).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin entry by LID: %w", err)
	}
	return &entry, nil
}

func (r *Repository) List(ctx context.Context) ([]entity.AdminEntry, error) {
	var entries []entity.AdminEntry
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list admin entries: %w", err)
	}
	return entries, nil
}