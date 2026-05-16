package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	domainError "github.com/AndreeJait/zora-core/domain/error"
	"github.com/AndreeJait/zora-core/port/inbound/task"
	"github.com/AndreeJait/zora-core/port/outbound"
)

type taskUseCase struct {
	taskRepo outbound.TaskRepository
	storage  outbound.Storage
}

var _ task.UseCase = (*taskUseCase)(nil)

func NewTaskUseCase(taskRepo outbound.TaskRepository, storage outbound.Storage) task.UseCase {
	return &taskUseCase{taskRepo: taskRepo, storage: storage}
}

func (uc *taskUseCase) Get(ctx context.Context, id string) (*entity.Task, error) {
	t, err := uc.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if t == nil {
		return nil, domainError.ErrTaskNotFound.WithError(fmt.Errorf("task %s not found", id))
	}
	return t, nil
}

func (uc *taskUseCase) GetByChatID(ctx context.Context, chatID string, page, perPage int) ([]entity.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	return uc.taskRepo.GetByChatID(ctx, chatID, page, perPage)
}

func (uc *taskUseCase) List(ctx context.Context, status string, page, perPage int) ([]entity.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	return uc.taskRepo.ListByStatus(ctx, status, page, perPage)
}

func (uc *taskUseCase) RenderGraph(ctx context.Context, id string, format string) (string, error) {
	t, err := uc.taskRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get task for graph: %w", err)
	}
	if t == nil {
		return "", domainError.ErrTaskNotFound.WithError(fmt.Errorf("task %s not found", id))
	}

	if t.GraphMermaid == nil || *t.GraphMermaid == "" {
		return "", domainError.ErrGraphNotReady.WithError(fmt.Errorf("graph not available for task %s", id))
	}

	switch format {
	case "presigned":
		objectKey := fmt.Sprintf("graphs/%s.mmd", t.ID)
		reader := strings.NewReader(*t.GraphMermaid)
		if err := uc.storage.Upload(ctx, "zora-graphs", objectKey, reader, int64(len(*t.GraphMermaid)), "text/plain"); err != nil {
			return "", fmt.Errorf("upload graph to storage: %w", err)
		}
		url, err := uc.storage.GetPresignedURL(ctx, "zora-graphs", objectKey, 24*time.Hour)
		if err != nil {
			return "", fmt.Errorf("get presigned url: %w", err)
		}
		return url, nil
	default:
		return *t.GraphMermaid, nil
	}
}