package agent

import "github.com/AndreeJait/zora-core/domain/entity"

// ExecuteInput carries the request parameters for an agent run.
type ExecuteInput struct {
	Task      string `json:"task" validate:"required"`
	SessionID string `json:"session_id"`
	ThreadID  string `json:"thread_id"` // optional: resume an existing thread
	ChatID    string `json:"chat_id"`   // WAHA chat ID (set when source is WhatsApp)
	MessageID string `json:"message_id"` // WAHA message ID (set when source is WhatsApp)
	Source    string `json:"source"`     // "waha" when coming from WhatsApp webhook
	UserID    string `json:"user_id"`    // user identity for scoped retrieval
	IsAdmin   bool   `json:"is_admin"`   // whether user has admin privileges
	TaskID    string `json:"task_id"`   // correlates with task entity for NSQ step tracking

	// PlanMode triggers plan generation instead of execution.
	// When true, the agent generates a step-by-step plan and sends it to the user
	// without executing any tools. The plan is saved for later approval.
	PlanMode       bool            `json:"plan_mode"`
	PreLoadedState *PreLoadedState `json:"pre_loaded_state,omitempty"`

	// QuotedContext carries the reply chain when the user replied to a message.
	QuotedContext *entity.QuotedContext `json:"quoted_context,omitempty"`
}

// PreLoadedState carries cached search state from a previous plan generation.
// When set, thinkNode skips tool/knowledge search and reuses the cached state.
type PreLoadedState struct {
	PlanText      string                    `json:"plan_text"`
	SystemPrompt  string                    `json:"system_prompt"`
	RelevantTools []entity.ToolContext      `json:"relevant_tools"`
	RetrievedDocs []entity.KnowledgeSnippet `json:"retrieved_docs"`
	ExtractedTags []string                  `json:"extracted_tags"`
	ForceSearch   bool                      `json:"force_search"` // true = re-run search steps despite cached state
}

// ExecuteOutput is the final result of an agent execution.
type ExecuteOutput struct {
	TraceID    string `json:"trace_id"`
	SessionID  string `json:"session_id"`
	ThreadID   string `json:"thread_id"`
	Resolution string `json:"resolution"`
	Iterations  int    `json:"iterations"`
	IsResolved  bool   `json:"is_resolved"`
}