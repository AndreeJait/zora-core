package entity

import (
	"fmt"
	"strings"

	"github.com/AndreeJait/go-utility/v2/llmw"
)

// AgentStep represents the current phase in the Think-Act-Observe loop.
type AgentStep string

const (
	StepThink   AgentStep = "think"
	StepAct     AgentStep = "act"
	StepObserve AgentStep = "observe"
)

// ZoraState is the state payload carried through the graphw execution loop.
// It is JSON-serializable for checkpointing.
type ZoraState struct {
	// Identity
	TraceID   string `json:"trace_id"`
	ThreadID  string `json:"thread_id"`
	SessionID string `json:"session_id"`
	TaskID    string `json:"task_id"` // correlates graph execution with task entity

	// Task
	Task       string `json:"task"`
	IsResolved bool   `json:"is_resolved"`
	Resolution string `json:"resolution"`

	// Conversation
	Messages []llmw.Message `json:"messages"`

	// Loop control
	CurrentStep AgentStep `json:"current_step"`
	Iteration   int       `json:"iteration"`
	MaxSteps    int       `json:"max_steps"`

	// Tool context
	RelevantTools []ToolContext `json:"relevant_tools"`
	PendingCalls  []ToolCall    `json:"pending_calls"`
	ToolResults   []ToolResult  `json:"tool_results"`

	// Knowledge context
	RetrievedDocs []KnowledgeSnippet `json:"retrieved_docs"`

	// Tag context (extracted once at start)
	ExtractedTags []string `json:"extracted_tags"`

	// System prompt (built once at start)
	SystemPrompt string `json:"system_prompt"`

	// Plan mode — when true, generate plan instead of executing tools
	PlanMode bool `json:"plan_mode"`

	// Error and retry
	LastError  string `json:"last_error,omitempty"`
	RetryCount int    `json:"retry_count"`

	// WAHA context (set when source is WhatsApp)
	ChatID    string `json:"chat_id"`
	MessageID string `json:"message_id"`
	Source    string `json:"source"` // "waha" when coming from WhatsApp

	// User identity (for scoped retrieval)
	UserID  string `json:"user_id"`  // user identity for scoped retrieval
	IsAdmin bool   `json:"is_admin"` // whether user has admin privileges

	// Reply chain context (set when message is a reply)
	QuotedContext *QuotedContext `json:"quoted_context,omitempty"`

	// Extensible
	Metadata map[string]any `json:"metadata"`
}

// QuotedContext holds the reply chain context for the current message.
type QuotedContext struct {
	Chain        []QuotedMessage `json:"chain"`
	ReplyToMsgID string          `json:"reply_to_msg_id,omitempty"` // user's message ID — zora should reply to this
}

// QuotedMessage represents a single message in a reply chain.
type QuotedMessage struct {
	SenderPhone string `json:"sender_phone"`
	Body        string `json:"body"`
	FromMe      bool   `json:"from_me"` // true = zora's own message
}

// FormatChain formats the reply chain as human-readable text for the LLM system prompt.
func (qc *QuotedContext) FormatChain() string {
	if qc == nil || len(qc.Chain) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("The user is replying to a message chain. Here is the context:\n\n")

	for i, msg := range qc.Chain {
		if msg.FromMe {
			sb.WriteString(fmt.Sprintf("[Zora's message]: %s", msg.Body))
		} else {
			sender := strings.TrimSuffix(msg.SenderPhone, "@c.us")
			sb.WriteString(fmt.Sprintf("[Quoted message from %s]: %s", sender, msg.Body))
		}
		if i < len(qc.Chain)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n\nThe user's current message refers to the above context.")
	return sb.String()
}

// TagInfo holds a tag with name and description as returned by the MCP server.
type TagInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolContext holds a tool's metadata retrieved via semantic search.
type ToolContext struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Language    string         `json:"language"`
	Parameters  map[string]any `json:"parameters"`
	Score       float64        `json:"score"`
	Tags        []TagInfo      `json:"tags"`
}

// ToolCall represents an LLM-requested tool invocation.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult holds the outcome of a tool execution.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// KnowledgeSnippet is a RAG-retrieved document chunk.
type KnowledgeSnippet struct {
	DocID    string         `json:"doc_id"`
	Content  string         `json:"content"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata"`
	Tags     []string       `json:"tags"`
}
