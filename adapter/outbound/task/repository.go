package task

import (
	"context"
	"fmt"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"gorm.io/gorm"
)

// Repository implements outbound.TaskRepository using GORM.
type Repository struct {
	db *gorm.DB
}

var _ outbound.TaskRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, task *entity.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*entity.Task, error) {
	var task entity.Task
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return &task, nil
}

func (r *Repository) GetByChatID(ctx context.Context, chatID string, page, perPage int) ([]entity.Task, int64, error) {
	var tasks []entity.Task
	var total int64
	offset := (page - 1) * perPage

	if err := r.db.WithContext(ctx).Model(&entity.Task{}).Where("chat_id = ?", chatID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tasks by chat_id: %w", err)
	}
	if err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Order("created_at DESC").Offset(offset).Limit(perPage).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("get tasks by chat_id: %w", err)
	}
	return tasks, total, nil
}

func (r *Repository) Update(ctx context.Context, task *entity.Task) error {
	if err := r.db.WithContext(ctx).Save(task).Error; err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (r *Repository) ListByStatus(ctx context.Context, status string, page, perPage int) ([]entity.Task, int64, error) {
	var tasks []entity.Task
	var total int64
	offset := (page - 1) * perPage

	query := r.db.WithContext(ctx).Model(&entity.Task{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}
	if err := r.db.WithContext(ctx).Scopes(r.withStatus(status)).Order("created_at DESC").Offset(offset).Limit(perPage).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, total, nil
}

func (r *Repository) ListPendingRetry(ctx context.Context, before time.Time) ([]entity.Task, error) {
	var tasks []entity.Task
	if err := r.db.WithContext(ctx).Where("status = ? AND next_retry_at <= ?", entity.TaskStatusRetrying, before).Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("list pending retry: %w", err)
	}
	return tasks, nil
}

func (r *Repository) withStatus(status string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if status == "" {
			return db
		}
		return db.Where("status = ?", status)
	}
}