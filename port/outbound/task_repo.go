package outbound

import (
	"context"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// TaskRepository defines the outbound port for task persistence.
type TaskRepository interface {
	// Create inserts a new task.
	Create(ctx context.Context, task *entity.Task) error
	// GetByID returns a task by ID. Returns nil if not found.
	GetByID(ctx context.Context, id string) (*entity.Task, error)
	// GetByChatID returns paginated tasks filtered by chat ID.
	GetByChatID(ctx context.Context, chatID string, page, perPage int) ([]entity.Task, int64, error)
	// Update saves task changes.
	Update(ctx context.Context, task *entity.Task) error
	// ListByStatus returns paginated tasks filtered by status.
	ListByStatus(ctx context.Context, status string, page, perPage int) ([]entity.Task, int64, error)
	// ListPendingRetry returns tasks in "retrying" status with next_retry_at before the given time.
	ListPendingRetry(ctx context.Context, before time.Time) ([]entity.Task, error)
}