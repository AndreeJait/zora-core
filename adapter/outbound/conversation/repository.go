package conversation

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"gorm.io/gorm"
)

// Repository implements outbound.ConversationRepository using GORM.
type Repository struct {
	db *gorm.DB
}

var _ outbound.ConversationRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, conv *entity.Conversation) error {
	result := r.db.WithContext(ctx).Where("session_id = ?", conv.SessionID).Assign(conv).FirstOrCreate(conv)
	if result.Error != nil {
		return fmt.Errorf("save conversation: %w", result.Error)
	}
	return nil
}

func (r *Repository) GetBySessionID(ctx context.Context, sessionID string) (*entity.Conversation, error) {
	var conv entity.Conversation
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&conv).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return &conv, nil
}

func (r *Repository) Delete(ctx context.Context, sessionID string) error {
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&entity.Conversation{}).Error; err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}
