package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	domainError "github.com/AndreeJait/zora-core/domain/error"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/webhook"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
)

type webhookUseCase struct {
	wahaClient    portOutbound.WahaClient
	agentUC       agent.UseCase
	whitelistUC   whitelist.UseCase
	taskRepo      portOutbound.TaskRepository
	dispatcher    *TaskDispatcher
	planStore     portOutbound.PlanStore
	messageStore  portOutbound.MessageStore
	chainResolver *ChainResolver
	maxRetry      int

	dedup   map[string]time.Time
	dedupMu sync.Mutex
}

// NewWebhookUseCase creates a new WebhookUseCase implementation.
func NewWebhookUseCase(wahaClient portOutbound.WahaClient, agentUC agent.UseCase, whitelistUC whitelist.UseCase, taskRepo portOutbound.TaskRepository, dispatcher *TaskDispatcher, planStore portOutbound.PlanStore, messageStore portOutbound.MessageStore, chainResolver *ChainResolver, maxRetry int) webhook.UseCase {
	return &webhookUseCase{
		wahaClient:    wahaClient,
		agentUC:       agentUC,
		whitelistUC:   whitelistUC,
		taskRepo:      taskRepo,
		dispatcher:    dispatcher,
		planStore:     planStore,
		messageStore:  messageStore,
		chainResolver: chainResolver,
		maxRetry:      maxRetry,
		dedup:         make(map[string]time.Time),
	}
}

func (u *webhookUseCase) HandleIncomingMessage(ctx context.Context, event *entity.WAHAWebhookEvent) error {
	msg := entity.NewWAHAIncomingMessage(event)
	if msg == nil {
		return nil
	}

	if u.isProcessed(msg.MessageID) {
		return nil
	}

	logw.CtxInfof(ctx, "incoming WAHA message: from=%s chatID=%s senderPhone=%s senderLID=%s isGroup=%v body=%q",
		msg.From, msg.ChatID, msg.SenderPhone, msg.SenderLID, entity.IsGroupChat(msg.ChatID), msg.Body)

	// Store incoming message for future reply chain resolution
	if u.messageStore != nil {
		storeMsg := &entity.WAHAMessage{
			ID:          msg.MessageID,
			ChatID:      msg.ChatID,
			SenderPhone: msg.From,
			Body:        msg.Body,
			IsReply:     msg.IsReply,
			QuotedMsgID: msg.QuotedMsgID,
			FromMe:      false,
			Timestamp:   msg.Timestamp.Unix(),
		}
		if err := u.messageStore.Store(ctx, storeMsg); err != nil {
			logw.CtxWarningf(ctx, "webhook: failed to store incoming message %s: %v", msg.MessageID, err)
		}
	}

	// Build sender context for access checks
	sender := whitelist.SenderContext{
		SenderPhone: msg.SenderPhone,
		SenderLID:   msg.SenderLID,
		ChatID:      msg.ChatID,
		IsGroup:     entity.IsGroupChat(msg.ChatID),
	}

	// Mark as seen
	_ = u.wahaClient.SendSeen(ctx, msg.ChatID)

	ok, query := parseZora(msg.Body)
	if !ok {
		return nil
	}

	if query == "" {
		_ = u.wahaClient.SendText(ctx, msg.ChatID,
			"Hi! Ask me anything by typing *!zora* followed by your question.\n\nExample:\n!zora search a latest news about valorant")
		return nil
	}

	// Route whitelist commands
	if strings.HasPrefix(strings.ToLower(query), "whitelist") {
		args := strings.TrimSpace(strings.TrimPrefix(query, "whitelist"))
		result, err := u.whitelistUC.HandleCommand(ctx, sender, args, msg.MentionedIDs)
		if err != nil {
			if err == domainError.ErrNotAdmin {
				_ = u.wahaClient.SendText(ctx, msg.ChatID, "Only admins can manage the whitelist.")
				return nil
			}
			logw.CtxErrorf(ctx, "whitelist command failed: %v", err)
			_ = u.wahaClient.SendText(ctx, msg.ChatID, "Failed to process whitelist command.")
			return nil
		}
		_ = u.wahaClient.SendText(ctx, msg.ChatID, result)
		return nil
	}

	// Access check
	if err := u.whitelistUC.CheckAccess(ctx, sender); err != nil {
		if err == domainError.ErrNotWhitelisted {
			_ = u.wahaClient.SendText(ctx, msg.ChatID, "You are not authorized to use this bot. Contact an admin to get access.")
			return nil
		}
		if err == domainError.ErrTokenExhausted {
			_ = u.wahaClient.SendText(ctx, msg.ChatID, "Hourly token limit reached. Try again later.")
			return nil
		}
		logw.CtxErrorf(ctx, "access check failed: %v", err)
		return nil
	}

	// Consume token after access check passes
	if err := u.whitelistUC.ConsumeToken(ctx, sender); err != nil {
		logw.CtxErrorf(ctx, "token consumption failed: %v", err)
	}

	// Resolve reply chain context
	var quotedCtx *entity.QuotedContext
	if msg.IsReply && msg.QuotedMsgID != "" && u.chainResolver != nil {
		chain, chainErr := u.chainResolver.Resolve(ctx, msg.QuotedMsgID)
		if chainErr != nil {
			logw.CtxWarningf(ctx, "webhook: chain resolution failed: %v", chainErr)
		}
		if len(chain) > 0 {
			quotedCtx = &entity.QuotedContext{
				Chain:        chain,
				ReplyToMsgID: msg.MessageID,
			}
		}
	}

	// Check for pending plan — if exists, this message is a plan response
	if u.planStore != nil {
		plan, planErr := u.planStore.Get(ctx, msg.ChatID)
		if planErr != nil {
			logw.CtxWarningf(ctx, "plan store check failed: %v", planErr)
		} else if plan != nil {
			intent := parsePlanIntent(query)
			logw.CtxInfof(ctx, "pending plan found for chat %s, intent=%s", msg.ChatID, intent)

			switch intent {
			case "approve":
				// Delete plan and execute with pre-loaded state
				_ = u.planStore.Delete(ctx, msg.ChatID)
				taskText := fmt.Sprintf("Execute the following plan:\n%s", plan.PlanText)
				u.createAndDispatchTask(ctx, taskText, plan.SessionID, msg.ChatID, msg.MessageID, plan.UserID, plan.IsAdmin, false, &agent.PreLoadedState{
					PlanText:      plan.PlanText,
					SystemPrompt:  plan.SystemPrompt,
					RelevantTools: plan.RelevantTools,
					RetrievedDocs: plan.RetrievedDocs,
					ExtractedTags: plan.ExtractedTags,
				}, quotedCtx)
				return nil

			case "modify":
				// Keep plan in Redis (will be overwritten with updated plan)
				taskText := fmt.Sprintf("You previously created this plan:\n%s\n\nThe user wants changes: %s\nGenerate an UPDATED plan incorporating these changes.",
					plan.PlanText, query)
				u.createAndDispatchTask(ctx, taskText, plan.SessionID, msg.ChatID, msg.MessageID, plan.UserID, plan.IsAdmin, true, &agent.PreLoadedState{
					PlanText:      plan.PlanText,
					SystemPrompt:  plan.SystemPrompt,
					RelevantTools: plan.RelevantTools,
					RetrievedDocs: plan.RetrievedDocs,
					ExtractedTags: plan.ExtractedTags,
					ForceSearch:   true, // user modifications may need new search
				}, quotedCtx)
				return nil

			case "reject":
				_ = u.planStore.Delete(ctx, msg.ChatID)
				_ = u.wahaClient.SendText(ctx, msg.ChatID, "Plan cancelled. Let me know if you need anything else!")
				return nil
			}
		}
	}

	// Detect plan mode in user's request
	isAdmin := u.whitelistUC.IsAdmin(sender)
	planMode := detectPlanMode(query)

	// Create background task instead of synchronous execution
	task := &entity.Task{
		Type:      entity.TaskTypeWebhook,
		Source:    entity.TaskSourceWaha,
		Status:    entity.TaskStatusPending,
		MaxRetry:  u.maxRetry,
		ChatID:    &msg.ChatID,
		MessageID: &msg.MessageID,
		SessionID: &msg.ChatID,
		Input: map[string]any{
			"task":           query,
			"session_id":     msg.ChatID,
			"chat_id":        msg.ChatID,
			"message_id":    msg.MessageID,
			"source":        "waha",
			"user_id":       msg.From,
			"is_admin":      isAdmin,
			"plan_mode":      planMode,
			"quoted_context": quotedCtx,
		},
	}

	if err := u.taskRepo.Create(ctx, task); err != nil {
		logw.CtxErrorf(ctx, "failed to create task: %v", err)
		// Fallback to synchronous execution
		input := agent.ExecuteInput{
			Task:          query,
			SessionID:     msg.ChatID,
			ChatID:        msg.ChatID,
			MessageID:     msg.MessageID,
			Source:        "waha",
			UserID:        msg.From,
			IsAdmin:       isAdmin,
			QuotedContext: quotedCtx,
		}
		output, execErr := u.agentUC.Execute(ctx, input)
		if execErr != nil {
			_ = u.wahaClient.SendText(ctx, msg.ChatID, fmt.Sprintf("Sorry, something went wrong: %v", execErr))
			return nil
		}
		response := output.Resolution
		if response == "" {
			response = "I couldn't find a good answer for that. Can you try rephrasing?"
		}
		_ = u.wahaClient.SendText(ctx, msg.ChatID, response)
		return nil
	}

	// Dispatch task to worker pool
	u.dispatcher.Dispatch(ctx, task.ID)
	logw.CtxInfof(ctx, "task %s created and dispatched for chat %s", task.ID, msg.ChatID)

	return nil
}

// parseZora checks if the message starts with !zora and returns the query after it.
func parseZora(body string) (bool, string) {
	trimmed := strings.TrimSpace(body)
	if len(trimmed) < 5 {
		return false, ""
	}
	prefix := strings.ToLower(trimmed[:5])
	if prefix != "!zora" {
		return false, ""
	}
	query := strings.TrimSpace(trimmed[5:])
	return true, query
}

func (u *webhookUseCase) isProcessed(msgID string) bool {
	u.dedupMu.Lock()
	defer u.dedupMu.Unlock()

	if _, ok := u.dedup[msgID]; ok {
		return true
	}
	u.dedup[msgID] = time.Now()

	cutoff := time.Now().Add(-5 * time.Minute)
	for id, t := range u.dedup {
		if t.Before(cutoff) {
			delete(u.dedup, id)
		}
	}
	return false
}

// createAndDispatchTask creates a task and dispatches it to the worker pool.
func (u *webhookUseCase) createAndDispatchTask(ctx context.Context, taskText, sessionID, chatID, messageID, userID string, isAdmin, planMode bool, preLoadedState *agent.PreLoadedState, quotedCtx *entity.QuotedContext) {
	input := map[string]any{
		"task":           taskText,
		"session_id":     sessionID,
		"chat_id":        chatID,
		"message_id":     messageID,
		"source":        "waha",
		"user_id":       userID,
		"is_admin":      isAdmin,
		"plan_mode":      planMode,
		"quoted_context": quotedCtx,
	}
	if preLoadedState != nil {
		input["pre_loaded_state"] = preLoadedState
	}

	task := &entity.Task{
		Type:      entity.TaskTypeWebhook,
		Source:    entity.TaskSourceWaha,
		Status:    entity.TaskStatusPending,
		MaxRetry:  u.maxRetry,
		ChatID:    &chatID,
		MessageID: &messageID,
		SessionID: &sessionID,
		Input:     input,
	}

	if err := u.taskRepo.Create(ctx, task); err != nil {
		logw.CtxErrorf(ctx, "failed to create plan task: %v", err)
		// Fallback to sync execution
		execInput := agent.ExecuteInput{
			Task:            taskText,
			SessionID:       sessionID,
			ChatID:          chatID,
			MessageID:       messageID,
			Source:          "waha",
			UserID:         userID,
			IsAdmin:         isAdmin,
			PlanMode:        planMode,
			PreLoadedState: preLoadedState,
			QuotedContext:  quotedCtx,
		}
		output, execErr := u.agentUC.Execute(ctx, execInput)
		if execErr != nil {
			_ = u.wahaClient.SendText(ctx, chatID, fmt.Sprintf("Sorry, something went wrong: %v", execErr))
			return
		}
		response := output.Resolution
		if response == "" {
			response = "I couldn't find a good answer for that. Can you try rephrasing?"
		}
		_ = u.wahaClient.SendText(ctx, chatID, response)
		return
	}

	u.dispatcher.Dispatch(ctx, task.ID)
	logw.CtxInfof(ctx, "plan task %s created and dispatched for chat %s (plan_mode=%v)", task.ID, chatID, planMode)
}

// parsePlanIntent detects whether the user is approving, modifying, or rejecting a plan.
func parsePlanIntent(query string) string {
	lower := strings.ToLower(strings.TrimSpace(query))
	// Approve signals
	if lower == "execute" || lower == "yes" || lower == "go" || lower == "run" ||
		lower == "do it" || lower == "proceed" || lower == "go ahead" ||
		strings.HasPrefix(lower, "execute ") || lower == "ok" || lower == "done" {
		return "approve"
	}
	// Reject signals
	if lower == "no" || lower == "cancel" || lower == "stop" || lower == "never mind" ||
		lower == "batal" || lower == "tidak" {
		return "reject"
	}
	// Default: treat as modification (user is giving feedback/changes)
	return "modify"
}

// detectPlanMode checks if the user wants to see a plan before execution.
func detectPlanMode(query string) bool {
	lower := strings.ToLower(query)
	return strings.Contains(lower, "send plan") ||
		strings.Contains(lower, "send me the plan") ||
		strings.Contains(lower, "plan first") ||
		strings.Contains(lower, "don't execute") ||
		strings.Contains(lower, "show plan") ||
		strings.Contains(lower, "create a plan") ||
		strings.Contains(lower, "buat plan") ||
		strings.Contains(lower, "kirim plan")
}