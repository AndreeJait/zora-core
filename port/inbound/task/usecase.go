package task

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// UseCase defines the inbound port for task management.
type UseCase interface {
	// Get returns a task by ID.
	Get(ctx context.Context, id string) (*entity.Task, error)
	// GetByChatID returns paginated tasks filtered by chat ID.
	GetByChatID(ctx context.Context, chatID string, page, perPage int) ([]entity.Task, int64, error)
	// List returns paginated tasks filtered by status.
	List(ctx context.Context, status string, page, perPage int) ([]entity.Task, int64, error)
	// RenderGraph returns the Mermaid diagram for a task.
	// format: "mmd" returns Mermaid text, "presigned" uploads to MinIO and returns a URL.
	RenderGraph(ctx context.Context, id string, format string) (string, error)
}