package whitelist

import (
	"context"
	"fmt"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements outbound.WhitelistRepository using GORM.
type Repository struct {
	db *gorm.DB
}

var _ outbound.WhitelistRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Add(ctx context.Context, entry *entity.WhitelistEntry) error {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "phone"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "lid", "scope", "chat_ids", "tokens_per_hour", "updated_at"}),
	}).Create(entry)
	if result.Error != nil {
		return fmt.Errorf("add whitelist entry: %w", result.Error)
	}
	return nil
}

func (r *Repository) Remove(ctx context.Context, phone string) error {
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).Delete(&entity.WhitelistEntry{}).Error; err != nil {
		return fmt.Errorf("remove whitelist entry: %w", err)
	}
	return nil
}

func (r *Repository) GetByPhone(ctx context.Context, phone string) (*entity.WhitelistEntry, error) {
	var entry entity.WhitelistEntry
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get whitelist entry: %w", err)
	}
	return &entry, nil
}

func (r *Repository) GetByLID(ctx context.Context, lid string) (*entity.WhitelistEntry, error) {
	var entry entity.WhitelistEntry
	if err := r.db.WithContext(ctx).Where("lid = ?", lid).First(&entry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get whitelist entry by LID: %w", err)
	}
	return &entry, nil
}

func (r *Repository) List(ctx context.Context) ([]entity.WhitelistEntry, error) {
	var entries []entity.WhitelistEntry
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list whitelist entries: %w", err)
	}
	return entries, nil
}

func (r *Repository) IncrementUsage(ctx context.Context, phone string, windowStart time.Time) (int, error) {
	var usage entity.TokenUsage
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "phone"}, {Name: "window_start"}},
		DoUpdates: clause.AssignmentColumns([]string{"tokens_used"}),
	}).Where("phone = ? AND window_start = ?", phone, windowStart).FirstOrCreate(&usage)

	// If we found an existing record, increment it manually since FirstOrCreate + OnConflict
	// on Create doesn't trigger the DO UPDATE for existing rows correctly.
	// Use a raw upsert instead.
	var total int
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO token_usages (id, phone, tokens_used, window_start, created_at)
		VALUES (gen_random_uuid(), ?, 1, ?, NOW())
		ON CONFLICT (phone, window_start) DO UPDATE SET tokens_used = token_usages.tokens_used + 1
		RETURNING tokens_used
	`, phone, windowStart).Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("increment token usage: %w", err)
	}
	_ = result
	return total, nil
}

func (r *Repository) GetUsage(ctx context.Context, phone string, windowStart time.Time) (int, error) {
	var usage entity.TokenUsage
	if err := r.db.WithContext(ctx).Where("phone = ? AND window_start = ?", phone, windowStart).First(&usage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("get token usage: %w", err)
	}
	return usage.TokensUsed, nil
}

func (r *Repository) CleanupExpiredTokens(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).Where("window_start < ?", before).Delete(&entity.TokenUsage{}).Error; err != nil {
		return fmt.Errorf("cleanup expired tokens: %w", err)
	}
	return nil
}