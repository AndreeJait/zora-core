package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/AndreeJait/go-utility/v2/brokerw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// NSQ topic constants.
const (
	TopicTask      = "zora-task"
	TopicGraphStep = "zora-graph-step"
)

// TaskMessage is the NSQ payload for the zora-task topic.
type TaskMessage struct {
	TaskID string `json:"task_id"`
}

// GraphStepMessage is the NSQ payload for the zora-graph-step topic.
type GraphStepMessage struct {
	ThreadID string `json:"thread_id"`
	TaskID   string `json:"task_id"`
}

// TaskDispatcher publishes task IDs and graph step messages to NSQ.
type TaskDispatcher struct {
	producer brokerw.Producer
}

func NewTaskDispatcher(producer brokerw.Producer) *TaskDispatcher {
	return &TaskDispatcher{producer: producer}
}

// Dispatch publishes a task ID to the zora-task NSQ topic.
func (d *TaskDispatcher) Dispatch(ctx context.Context, taskID string) error {
	payload, err := json.Marshal(TaskMessage{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal task message: %w", err)
	}
	return d.producer.Send(ctx, TopicTask, nil, payload)
}

// DispatchGraphStep publishes a graph step message to the zora-graph-step NSQ topic.
func (d *TaskDispatcher) DispatchGraphStep(ctx context.Context, threadID, taskID string) error {
	payload, err := json.Marshal(GraphStepMessage{ThreadID: threadID, TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal graph step message: %w", err)
	}
	return d.producer.Send(ctx, TopicGraphStep, nil, payload)
}

// Close shuts down the NSQ producer.
func (d *TaskDispatcher) Close() error {
	return d.producer.Close()
}

// TaskHandler processes zora-task NSQ messages.
// It fetches the task, executes the first graph superstep via StepExecute(),
// and either completes the task or enqueues the next step.
type TaskHandler struct {
	taskRepo    outbound.TaskRepository
	agentUC     agent.UseCase
	settingRepo outbound.SettingRepository
	whitelistUC whitelist.UseCase
	wahaClient  outbound.WahaClient
	dispatcher  *TaskDispatcher
}

func NewTaskHandler(
	taskRepo outbound.TaskRepository,
	agentUC agent.UseCase,
	settingRepo outbound.SettingRepository,
	whitelistUC whitelist.UseCase,
	wahaClient outbound.WahaClient,
	dispatcher *TaskDispatcher,
) *TaskHandler {
	return &TaskHandler{
		taskRepo:    taskRepo,
		agentUC:     agentUC,
		settingRepo: settingRepo,
		whitelistUC: whitelistUC,
		wahaClient:  wahaClient,
		dispatcher:  dispatcher,
	}
}

// HandleTask processes a zora-task NSQ message.
func (h *TaskHandler) HandleTask(ctx context.Context, msg *brokerw.Message) error {
	var taskMsg TaskMessage
	if err := json.Unmarshal(msg.Payload, &taskMsg); err != nil {
		logw.CtxErrorf(ctx, "task handler: invalid message: %v", err)
		return err // NSQ will requeue
	}

	t, err := h.taskRepo.GetByID(ctx, taskMsg.TaskID)
	if err != nil || t == nil {
		logw.CtxErrorf(ctx, "task handler: task %s not found: %v", taskMsg.TaskID, err)
		return nil // ACK — task may have been deleted
	}

	// Skip if already completed or cancelled
	if t.Status == entity.TaskStatusCompleted || t.Status == entity.TaskStatusCancelled {
		return nil
	}

	// Mark as running
	t.Status = entity.TaskStatusRunning
	if err := h.taskRepo.Update(ctx, t); err != nil {
		logw.CtxErrorf(ctx, "task handler: failed to mark task %s as running: %v", taskMsg.TaskID, err)
		return nil
	}

	// Reconstruct agent input from task entity columns (primary) with Input map as fallback
	input := agent.ExecuteInput{
		Task:      strFromMap(t.Input, "task"),
		SessionID: strPtrSafe(t.SessionID, strFromMap(t.Input, "session_id")),
		ChatID:    strPtrSafe(t.ChatID, strFromMap(t.Input, "chat_id")),
		MessageID: strPtrSafe(t.MessageID, strFromMap(t.Input, "message_id")),
		Source:    strCoalesce(t.Source, strFromMap(t.Input, "source")),
		UserID:    strFromMap(t.Input, "user_id"),
		IsAdmin:   boolFromMap(t.Input, "is_admin"),
		PlanMode:  boolFromMap(t.Input, "plan_mode"),
		TaskID:    t.ID,
	}

	// Reconstruct PreLoadedState if present in task input
	if preLoaded := preLoadedStateFromMap(t.Input); preLoaded != nil {
		input.PreLoadedState = preLoaded
	}

	// Reconstruct QuotedContext if present in task input
	if quotedCtx := quotedContextFromMap(t.Input); quotedCtx != nil {
		input.QuotedContext = quotedCtx
	}

	// Execute the first graph superstep
	stepResult, err := h.agentUC.StepExecute(ctx, input)
	if err != nil {
		h.handleFailure(ctx, t, err)
		return nil
	}

	if stepResult.IsDone {
		h.completeTask(ctx, t, stepResult.State)
	} else {
		// Store threadID on task for later reference
		threadID := stepResult.State.ThreadID
		t.ThreadID = &threadID
		if err := h.taskRepo.Update(ctx, t); err != nil {
			logw.CtxErrorf(ctx, "task handler: failed to update task %s threadID: %v", t.ID, err)
		}
		// Enqueue next graph step
		if err := h.dispatcher.DispatchGraphStep(ctx, threadID, t.ID); err != nil {
			logw.CtxErrorf(ctx, "task handler: failed to dispatch graph step for task %s: %v", t.ID, err)
			h.handleFailure(ctx, t, err)
			return nil
		}
		logw.CtxInfof(ctx, "task handler: task %s step complete, next nodes: %v", t.ID, stepResult.Next)
	}

	return nil
}

// GraphStepHandler processes zora-graph-step NSQ messages.
// It executes one superstep of an already-running graph and either completes
// the task or enqueues the next step.
type GraphStepHandler struct {
	taskRepo    outbound.TaskRepository
	agentUC     agent.UseCase
	settingRepo outbound.SettingRepository
	whitelistUC whitelist.UseCase
	wahaClient  outbound.WahaClient
	dispatcher  *TaskDispatcher
}

func NewGraphStepHandler(
	taskRepo outbound.TaskRepository,
	agentUC agent.UseCase,
	settingRepo outbound.SettingRepository,
	whitelistUC whitelist.UseCase,
	wahaClient outbound.WahaClient,
	dispatcher *TaskDispatcher,
) *GraphStepHandler {
	return &GraphStepHandler{
		taskRepo:    taskRepo,
		agentUC:     agentUC,
		settingRepo: settingRepo,
		whitelistUC: whitelistUC,
		wahaClient:  wahaClient,
		dispatcher:  dispatcher,
	}
}

// HandleGraphStep processes a zora-graph-step NSQ message.
func (h *GraphStepHandler) HandleGraphStep(ctx context.Context, msg *brokerw.Message) error {
	var stepMsg GraphStepMessage
	if err := json.Unmarshal(msg.Payload, &stepMsg); err != nil {
		logw.CtxErrorf(ctx, "graph step handler: invalid message: %v", err)
		return err
	}

	// Execute next step — state is restored from checkpoint
	stepResult, err := h.agentUC.StepContinue(ctx, stepMsg.ThreadID)
	if err != nil {
		t, terr := h.taskRepo.GetByID(ctx, stepMsg.TaskID)
		if terr != nil || t == nil {
			logw.CtxErrorf(ctx, "graph step handler: task %s not found: %v", stepMsg.TaskID, terr)
			return nil
		}
		h.handleFailure(ctx, t, err)
		return nil
	}

	if stepResult.IsDone {
		t, terr := h.taskRepo.GetByID(ctx, stepMsg.TaskID)
		if terr != nil || t == nil {
			logw.CtxErrorf(ctx, "graph step handler: task %s not found: %v", stepMsg.TaskID, terr)
			return nil
		}
		h.completeTask(ctx, t, stepResult.State)
	} else {
		// Enqueue next step
		if err := h.dispatcher.DispatchGraphStep(ctx, stepMsg.ThreadID, stepMsg.TaskID); err != nil {
			logw.CtxErrorf(ctx, "graph step handler: failed to dispatch next step for task %s: %v", stepMsg.TaskID, err)
			t, terr := h.taskRepo.GetByID(ctx, stepMsg.TaskID)
			if terr == nil && t != nil {
				h.handleFailure(ctx, t, err)
			}
			return nil
		}
		logw.CtxInfof(ctx, "graph step handler: task %s step complete, next nodes: %v", stepMsg.TaskID, stepResult.Next)
	}

	return nil
}

// --- Shared helpers ---

func (h *TaskHandler) completeTask(ctx context.Context, t *entity.Task, state entity.ZoraState) {
	completeTaskHelper(ctx, h.agentUC, h.taskRepo, t, state)
}

func (h *GraphStepHandler) completeTask(ctx context.Context, t *entity.Task, state entity.ZoraState) {
	completeTaskHelper(ctx, h.agentUC, h.taskRepo, t, state)
}

func completeTaskHelper(ctx context.Context, agentUC agent.UseCase, taskRepo outbound.TaskRepository, t *entity.Task, state entity.ZoraState) {
	if mermaid := agentUC.MermaidDiagram(); mermaid != "" {
		t.GraphMermaid = &mermaid
	}

	t.Status = entity.TaskStatusCompleted
	t.Result = map[string]any{
		"trace_id":    state.TraceID,
		"session_id":  state.SessionID,
		"thread_id":   state.ThreadID,
		"resolution":  state.Resolution,
		"iterations":  state.Iteration,
		"is_resolved": state.IsResolved,
	}
	if err := taskRepo.Update(ctx, t); err != nil {
		logw.CtxErrorf(ctx, "worker: failed to update completed task %s: %v", t.ID, err)
	}
}

func (h *TaskHandler) handleFailure(ctx context.Context, t *entity.Task, execErr error) {
	handleFailureHelper(ctx, h.settingRepo, h.whitelistUC, h.wahaClient, h.taskRepo, t, execErr)
}

func (h *GraphStepHandler) handleFailure(ctx context.Context, t *entity.Task, execErr error) {
	handleFailureHelper(ctx, h.settingRepo, h.whitelistUC, h.wahaClient, h.taskRepo, t, execErr)
}

func handleFailureHelper(
	ctx context.Context,
	settingRepo outbound.SettingRepository,
	whitelistUC whitelist.UseCase,
	wahaClient outbound.WahaClient,
	taskRepo outbound.TaskRepository,
	t *entity.Task,
	execErr error,
) {
	errMsg := execErr.Error()
	t.Error = &errMsg

	maxRetry := t.MaxRetry
	if s, err := settingRepo.Get(ctx, "task.max_retry"); err == nil && s != nil {
		if val, parseErr := strconv.Atoi(s.Value); parseErr == nil {
			maxRetry = val
		}
	}

	if t.RetryCount < maxRetry {
		t.RetryCount++

		retryDelay := 30 * time.Second
		if s, err := settingRepo.Get(ctx, "task.retry_delay_seconds"); err == nil && s != nil {
			if val, parseErr := strconv.Atoi(s.Value); parseErr == nil {
				retryDelay = time.Duration(val) * time.Second
			}
		}

		backoffDelay := retryDelay * time.Duration(1<<(t.RetryCount-1))
		nextRetry := time.Now().Add(backoffDelay)
		t.Status = entity.TaskStatusRetrying
		t.NextRetryAt = &nextRetry
		if err := taskRepo.Update(ctx, t); err != nil {
			logw.CtxErrorf(ctx, "worker: failed to update retrying task %s: %v", t.ID, err)
		}
		logw.CtxInfof(ctx, "worker: task %s retrying (attempt %d/%d, next retry at %v)", t.ID, t.RetryCount, maxRetry, nextRetry)
		return
	}

	t.Status = entity.TaskStatusFailed
	if err := taskRepo.Update(ctx, t); err != nil {
		logw.CtxErrorf(ctx, "worker: failed to update failed task %s: %v", t.ID, err)
	}
	logw.CtxErrorf(ctx, "worker: task %s failed after %d retries: %v", t.ID, maxRetry, execErr)

	notifyFailure(ctx, whitelistUC, wahaClient, settingRepo, t)
}

func notifyFailure(ctx context.Context, whitelistUC whitelist.UseCase, wahaClient outbound.WahaClient, settingRepo outbound.SettingRepository, t *entity.Task) {
	if t.Source != entity.TaskSourceWaha {
		return
	}

	msg := fmt.Sprintf("Task %s failed after max retries.\nError: %s", t.ID, *t.Error)

	admins, err := whitelistUC.ListAdmins(ctx)
	if err != nil {
		logw.CtxErrorf(ctx, "worker: failed to list admins for failure notification: %v", err)
		return
	}

	notified := false
	for _, admin := range admins {
		chatID := admin.Phone + "@c.us"
		if err := wahaClient.SendText(ctx, chatID, msg); err != nil {
			logw.CtxErrorf(ctx, "worker: failed to notify admin %s for task %s: %v", admin.Phone, t.ID, err)
			continue
		}
		notified = true
	}

	if !notified {
		adminChatID := ""
		if s, sErr := settingRepo.Get(ctx, "notification.admin_chat_id"); sErr == nil && s != nil {
			adminChatID = s.Value
		}
		if adminChatID == "" {
			logw.CtxInfof(ctx, "worker: no admins to notify for task %s", t.ID)
			return
		}
		if err := wahaClient.SendText(ctx, adminChatID, msg); err != nil {
			logw.CtxErrorf(ctx, "worker: failed to notify admin_chat_id for task %s: %v", t.ID, err)
		}
	}
}

// --- Map helpers ---

func strFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolFromMap(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func preLoadedStateFromMap(m map[string]any) *agent.PreLoadedState {
	v, ok := m["pre_loaded_state"]
	if !ok {
		return nil
	}
	if pls, ok := v.(*agent.PreLoadedState); ok {
		return pls
	}
	return nil
}

func quotedContextFromMap(m map[string]any) *entity.QuotedContext {
	v, ok := m["quoted_context"]
	if !ok {
		return nil
	}
	if qc, ok := v.(*entity.QuotedContext); ok {
		return qc
	}
	return nil
}

func strPtrSafe(ptr *string, fallback string) string {
	if ptr != nil && *ptr != "" {
		return *ptr
	}
	return fallback
}

func strCoalesce(first, fallback string) string {
	if first != "" {
		return first
	}
	return fallback
}