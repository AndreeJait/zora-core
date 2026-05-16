package usecase

import (
	"context"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// RetrySweeper scans for overdue retrying tasks and re-enqueues them via NSQ.
type RetrySweeper struct {
	taskRepo   outbound.TaskRepository
	dispatcher *TaskDispatcher
}

func NewRetrySweeper(taskRepo outbound.TaskRepository, dispatcher *TaskDispatcher) *RetrySweeper {
	return &RetrySweeper{taskRepo: taskRepo, dispatcher: dispatcher}
}

func (s *RetrySweeper) Sweep(ctx context.Context) error {
	tasks, err := s.taskRepo.ListPendingRetry(ctx, time.Now())
	if err != nil {
		return err
	}

	for _, t := range tasks {
		if err := s.dispatcher.Dispatch(ctx, t.ID); err != nil {
			logw.CtxErrorf(ctx, "retry sweeper: failed to dispatch task %s: %v", t.ID, err)
			continue
		}
		logw.CtxInfof(ctx, "retry sweeper: re-enqueued task %s (retry %d)", t.ID, t.RetryCount)
	}

	return nil
}

// RecoverStuckTasks resets tasks stuck in "running" status (from server crash)
// and re-enqueues overdue retrying tasks. Call on worker startup.
func RecoverStuckTasks(ctx context.Context, taskRepo outbound.TaskRepository, dispatcher *TaskDispatcher) {
	// Recover tasks stuck in "running" (server crashed mid-execution)
	tasks, _, err := taskRepo.ListByStatus(ctx, entity.TaskStatusRunning, 1, 100)
	if err != nil {
		logw.Errorf("recover: failed to list running tasks: %v", err)
	} else {
		for _, t := range tasks {
			t.Status = entity.TaskStatusPending
			if err := taskRepo.Update(ctx, &t); err != nil {
				logw.Errorf("recover: failed to reset task %s: %v", t.ID, err)
				continue
			}
			if err := dispatcher.Dispatch(ctx, t.ID); err != nil {
				logw.Errorf("recover: failed to dispatch task %s: %v", t.ID, err)
				continue
			}
			logw.Infof("recover: re-enqueued stuck task %s", t.ID)
		}
	}

	// Re-enqueue overdue retrying tasks
	pendingRetry, err := taskRepo.ListPendingRetry(ctx, time.Now())
	if err != nil {
		logw.Errorf("recover: failed to list pending retry tasks: %v", err)
	} else {
		for _, t := range pendingRetry {
			if err := dispatcher.Dispatch(ctx, t.ID); err != nil {
				logw.Errorf("recover: failed to dispatch retry task %s: %v", t.ID, err)
				continue
			}
			logw.Infof("recover: re-enqueued retry task %s", t.ID)
		}
	}
}