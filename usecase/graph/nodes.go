package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// AgentDeps holds the dependencies injected into graph nodes.
type AgentDeps struct {
	LLM                   llmw.LLM
	Embedder              outbound.Embedder
	ToolRegistry          outbound.ToolRegistryClient
	Knowledge             outbound.KnowledgeClient
	TagExtractor          outbound.TagExtractor
	ToolLimit             int
	DocsLimit             int
	ToolMinScore          float64
	DocsMinScore          float64
	MaxToolCtxTokens      int
	MaxKnowledgeCtxTokens int
	ToolsEnabled          bool
	KnowledgeEnabled      bool
	WahaClient            outbound.WahaClient
	PlanStore             outbound.PlanStore
	MessageStore          outbound.MessageStore
}

// NewAgentDeps constructs AgentDeps from config and injected services.
func NewAgentDeps(
	cfg *config.AppConfig,
	llm llmw.LLM,
	embedder outbound.Embedder,
	toolReg outbound.ToolRegistryClient,
	knowledge outbound.KnowledgeClient,
	tagExtractor outbound.TagExtractor,
	wahaClient outbound.WahaClient,
	planStore outbound.PlanStore,
	messageStore outbound.MessageStore,
) AgentDeps {
	return AgentDeps{
		LLM:                   llm,
		Embedder:              embedder,
		ToolRegistry:          toolReg,
		Knowledge:             knowledge,
		TagExtractor:          tagExtractor,
		ToolLimit:             cfg.Agent.ToolLimit,
		DocsLimit:             cfg.Agent.KnowledgeLimit,
		ToolMinScore:          cfg.Agent.ToolMinScore,
		DocsMinScore:          cfg.Agent.KnowledgeMinScore,
		MaxToolCtxTokens:      cfg.Agent.MaxToolContextTokens,
		MaxKnowledgeCtxTokens: cfg.Agent.MaxKnowledgeContextTokens,
		ToolsEnabled:          cfg.Agent.ToolsEnabled,
		KnowledgeEnabled:      cfg.Agent.KnowledgeEnabled,
		WahaClient:            wahaClient,
		PlanStore:             planStore,
		MessageStore:          messageStore,
	}
}

func thinkNode(deps AgentDeps) graphw.NodeFunc[entity.ZoraState] {
	return func(ctx context.Context, state entity.ZoraState) (graphw.NodeResult[entity.ZoraState], error) {
		logw.CtxInfof(ctx, "thinkNode: iteration %d, messages=%d", state.Iteration, len(state.Messages))

		// First invocation: send 💭 reaction to indicate processing
		if state.SystemPrompt == "" && state.Source == "waha" && state.ChatID != "" && state.MessageID != "" {
			_ = deps.WahaClient.SendReaction(ctx, state.ChatID, state.MessageID, "💭")
		}

		// First invocation: tag extraction + semantic retrieval + build system prompt
		if state.SystemPrompt == "" {
			// Step 0: Fetch available tags from tool registry to constrain LLM extraction
			var tagNames []string
			availableTags, err := deps.ToolRegistry.ListTags(ctx)
			if err != nil {
				logw.CtxWarningf(ctx, "thinkNode: failed to list tags: %v, continuing without tag constraints", err)
			} else {
				for _, t := range availableTags {
					tagNames = append(tagNames, t.Name)
				}
			}

			// Step 1: Extract tags from task text using available tags
			tags, err := deps.TagExtractor.ExtractTags(ctx, state.Task, tagNames)
			if err != nil {
				logw.CtxWarningf(ctx, "thinkNode: tag extraction failed: %v, continuing without tags", err)
				tags = nil
			}
			state.ExtractedTags = tags

			// Step 2: Embed task text
			embeddings, err := deps.Embedder.Embed(ctx, []string{state.Task})
			if err != nil {
				logw.CtxErrorf(ctx, "thinkNode: embed task failed: %v", err)
				return graphw.NodeResult[entity.ZoraState]{
					State: entity.ZoraState{LastError: err.Error()},
					Route: graphw.END,
				}, nil
			}
			taskEmb := embeddings[0]

			// Step 3: Search tools with tags + user context
			var tools []entity.ToolContext
			if deps.ToolsEnabled {
				var err error
				tools, err = deps.ToolRegistry.SearchTools(ctx, outbound.ToolSearchQuery{
					Embedding: taskEmb,
					Tags:      state.ExtractedTags,
					UserID:    state.UserID,
					Limit:     deps.ToolLimit,
				})
				if err != nil {
					logw.CtxErrorf(ctx, "thinkNode: search tools failed: %v", err)
					tools = nil
				}
				for _, t := range tools {
					logw.CtxInfof(ctx, "thinkNode: tool result: name=%s score=%.4f tags=%v", t.Name, t.Score, t.Tags)
				}
			} else {
				logw.CtxInfof(ctx, "thinkNode: tools disabled, skipping tool search")
			}

			// Step 4: Search knowledge with tags + user context
			var docs []entity.KnowledgeSnippet
			if deps.KnowledgeEnabled {
				var err error
				docs, err = deps.Knowledge.SearchDocs(ctx, outbound.KnowledgeSearchQuery{
					Embedding: taskEmb,
					Tags:      state.ExtractedTags,
					UserID:    state.UserID,
					IsAdmin:   state.IsAdmin,
					Limit:     deps.DocsLimit,
				})
				if err != nil {
					logw.CtxErrorf(ctx, "thinkNode: search docs failed: %v", err)
					docs = nil
				}
				for _, d := range docs {
					logw.CtxInfof(ctx, "thinkNode: doc result: id=%s score=%.4f tags=%v", d.DocID, d.Score, d.Tags)
				}
			} else {
				logw.CtxInfof(ctx, "thinkNode: knowledge disabled, skipping doc search")
			}

			// Step 5: Filter by minimum score threshold
			logw.CtxInfof(ctx, "thinkNode: filtering tools=%d docs=%d with minToolScore=%.2f minDocScore=%.2f", len(tools), len(docs), deps.ToolMinScore, deps.DocsMinScore)
			tools = filterToolsByScore(tools, deps.ToolMinScore)
			docs = filterDocsByScore(docs, deps.DocsMinScore)
			logw.CtxInfof(ctx, "thinkNode: after filter tools=%d docs=%d", len(tools), len(docs))

			// Step 6: Budget context to stay within token limits
			tools = budgetTools(tools, deps.MaxToolCtxTokens)
			docs = budgetDocs(docs, deps.MaxKnowledgeCtxTokens)

			// Step 7: Inject built-in WAHA tools (always available, not subject to score filtering)
			// Deduplicate: skip builtins that already appear in search results by name
			if state.Source == "waha" {
				existing := make(map[string]bool, len(tools))
				for _, t := range tools {
					existing[t.Name] = true
				}
				for _, bt := range wahaBuiltinTools() {
					if !existing[bt.Name] {
						tools = append(tools, bt)
					}
				}
			}

			// Step 8: Build system prompt
			state.SystemPrompt = buildSystemPrompt(state.Task, tools, docs, state.Source, state.ExtractedTags, state.PlanMode, state.QuotedContext)
			state.RelevantTools = tools
			state.RetrievedDocs = docs
		}

		// Build tool definitions for LLM
		toolInfos := make([]llmw.ToolInfo, 0, len(state.RelevantTools))
		for _, t := range state.RelevantTools {
			toolInfos = append(toolInfos, llmw.ToolInfo{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}

		// Prepend system message to conversation for LLM call
		fullMessages := make([]llmw.Message, 0, 1+len(state.Messages))
		fullMessages = append(fullMessages, llmw.Message{Role: llmw.RoleSystem, Content: state.SystemPrompt})
		fullMessages = append(fullMessages, state.Messages...)

		// Call LLM
		resp, err := deps.LLM.Chat(ctx, fullMessages, llmw.WithTools(toolInfos...))
		if err != nil {
			logw.CtxErrorf(ctx, "thinkNode: LLM chat failed: %v", err)
			return graphw.NodeResult[entity.ZoraState]{
				State: entity.ZoraState{
					Messages:      nil,
					LastError:     err.Error(),
					IsResolved:    true,
					Resolution:    fmt.Sprintf("I encountered an error: %v", err),
					CurrentStep:   entity.StepThink,
					SystemPrompt:  state.SystemPrompt,
					RelevantTools: state.RelevantTools,
					RetrievedDocs: state.RetrievedDocs,
				},
				Route: graphw.END,
			}, nil
		}

		assistantMsg := resp.Message
		logw.CtxInfof(ctx, "thinkNode: LLM response — tool_calls=%d, content_len=%d",
			len(assistantMsg.ToolCalls), len(assistantMsg.Content))

		// LLM decided to call tools
		if len(assistantMsg.ToolCalls) > 0 {
			// Plan mode: intercept tool calls, format as plan, and send to user
			if state.PlanMode {
				planText := formatToolCallsAsPlan(assistantMsg.ToolCalls)
				logw.CtxInfof(ctx, "thinkNode: plan mode — generated plan with %d steps", len(assistantMsg.ToolCalls))

				// Save plan + state to PlanStore for later resumption
				if state.ChatID != "" && deps.PlanStore != nil {
					if err := deps.PlanStore.Save(ctx, state.ChatID, &outbound.PlanEntry{
						ThreadID:      state.ThreadID,
						SessionID:     state.SessionID,
						ChatID:        state.ChatID,
						MessageID:     state.MessageID,
						PlanText:      planText,
						SystemPrompt:  state.SystemPrompt,
						RelevantTools: state.RelevantTools,
						RetrievedDocs: state.RetrievedDocs,
						ExtractedTags: state.ExtractedTags,
						UserID:        state.UserID,
						IsAdmin:        state.IsAdmin,
						CreatedAt:     time.Now(),
					}); err != nil {
						logw.CtxErrorf(ctx, "thinkNode: failed to save plan: %v", err)
					}
				}

				// Send plan to WhatsApp
				if state.Source == "waha" && state.ChatID != "" {
					if state.QuotedContext != nil && state.QuotedContext.ReplyToMsgID != "" {
						_, _ = deps.WahaClient.SendTextReply(ctx, state.ChatID, planText, state.QuotedContext.ReplyToMsgID)
					} else {
						_ = deps.WahaClient.SendText(ctx, state.ChatID, planText)
					}
					if state.MessageID != "" {
						_ = deps.WahaClient.SendReaction(ctx, state.ChatID, state.MessageID, "✅")
					}
				}

				// Return resolved — no tool execution
				return graphw.NodeResult[entity.ZoraState]{
					State: entity.ZoraState{
						IsResolved:     true,
						Resolution:     planText,
						SystemPrompt:   state.SystemPrompt,
						RelevantTools:  state.RelevantTools,
						RetrievedDocs:  state.RetrievedDocs,
						ExtractedTags:  state.ExtractedTags,
					},
					Route: graphw.END,
				}, nil
			}

			pendingCalls := make([]entity.ToolCall, 0, len(assistantMsg.ToolCalls))
			for _, tc := range assistantMsg.ToolCalls {
				pendingCalls = append(pendingCalls, entity.ToolCall{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
			return graphw.NodeResult[entity.ZoraState]{
				State: entity.ZoraState{
					Messages:      []llmw.Message{assistantMsg},
					PendingCalls:  pendingCalls,
					CurrentStep:   entity.StepThink,
					SystemPrompt:  state.SystemPrompt,
					RelevantTools: state.RelevantTools,
					RetrievedDocs: state.RetrievedDocs,
				},
				Route: "act",
			}, nil
		}

		// LLM returned empty content — common with qwen3 models that use /think tags.
		// Retry with a nudge to actually respond or use a tool.
		if strings.TrimSpace(assistantMsg.Content) == "" {
			logw.CtxWarningf(ctx, "thinkNode: LLM returned empty content, retrying with nudge")
			nudgeText := "You did not provide any response. Please answer the user's task directly."
			if len(state.RelevantTools) > 0 {
				nudgeText = "You did not provide any response. Please use one of the available tools to answer the user's task, or provide a direct answer. Do not leave your response empty."
			}
			nudgeMessages := make([]llmw.Message, 0, 1+len(state.Messages)+2)
			nudgeMessages = append(nudgeMessages, llmw.Message{Role: llmw.RoleSystem, Content: state.SystemPrompt})
			nudgeMessages = append(nudgeMessages, state.Messages...)
			nudgeMessages = append(nudgeMessages, assistantMsg)
			nudgeMessages = append(nudgeMessages, llmw.Message{
				Role:    llmw.RoleUser,
				Content: nudgeText,
			})

			retryResp, retryErr := deps.LLM.Chat(ctx, nudgeMessages, llmw.WithTools(toolInfos...))
			if retryErr != nil {
				logw.CtxErrorf(ctx, "thinkNode: LLM retry failed: %v", retryErr)
				return graphw.NodeResult[entity.ZoraState]{
					State: entity.ZoraState{
						IsResolved:    true,
						Resolution:    fmt.Sprintf("I encountered an error: %v", retryErr),
						CurrentStep:   entity.StepThink,
						SystemPrompt:  state.SystemPrompt,
						RelevantTools: state.RelevantTools,
						RetrievedDocs: state.RetrievedDocs,
					},
					Route: graphw.END,
				}, nil
			}

			assistantMsg = retryResp.Message
			logw.CtxInfof(ctx, "thinkNode: LLM retry response — tool_calls=%d, content_len=%d",
				len(assistantMsg.ToolCalls), len(assistantMsg.Content))

			// Check for tool calls in retry
			if len(assistantMsg.ToolCalls) > 0 {
				pendingCalls := make([]entity.ToolCall, 0, len(assistantMsg.ToolCalls))
				for _, tc := range assistantMsg.ToolCalls {
					pendingCalls = append(pendingCalls, entity.ToolCall{
						ID:        tc.ID,
						Name:      tc.Name,
						Arguments: tc.Arguments,
					})
				}
				return graphw.NodeResult[entity.ZoraState]{
					State: entity.ZoraState{
						Messages:      []llmw.Message{assistantMsg},
						PendingCalls:  pendingCalls,
						CurrentStep:   entity.StepThink,
						SystemPrompt:  state.SystemPrompt,
						RelevantTools: state.RelevantTools,
						RetrievedDocs: state.RetrievedDocs,
					},
					Route: "act",
				}, nil
			}
		}

		// LLM responded with a final answer
		response := strings.TrimSpace(assistantMsg.Content)
		if response == "" {
			response = "I couldn't generate a response. Please try again."
		}

		// Auto-send response to WhatsApp when source is "waha"
		if state.Source == "waha" && state.ChatID != "" {
			var sentMsgID string
			if state.QuotedContext != nil && state.QuotedContext.ReplyToMsgID != "" {
				id, err := deps.WahaClient.SendTextReply(ctx, state.ChatID, response, state.QuotedContext.ReplyToMsgID)
				if err != nil {
					logw.CtxErrorf(ctx, "thinkNode: failed to send WhatsApp reply: %v", err)
				} else {
					sentMsgID = id
					logw.CtxInfof(ctx, "thinkNode: auto-sent WhatsApp reply to %s", state.ChatID)
				}
			} else {
				if err := deps.WahaClient.SendText(ctx, state.ChatID, response); err != nil {
					logw.CtxErrorf(ctx, "thinkNode: failed to send WhatsApp response: %v", err)
				} else {
					logw.CtxInfof(ctx, "thinkNode: auto-sent WhatsApp response to %s", state.ChatID)
				}
			}
			// Store outgoing message for future chain resolution
			if sentMsgID != "" && deps.MessageStore != nil {
				outMsg := &entity.WAHAMessage{
					ID:        sentMsgID,
					ChatID:    state.ChatID,
					Body:      response,
					FromMe:    true,
					Timestamp: time.Now().Unix(),
				}
				if state.QuotedContext != nil {
					outMsg.IsReply = true
					outMsg.QuotedMsgID = state.QuotedContext.ReplyToMsgID
				}
				_ = deps.MessageStore.Store(ctx, outMsg)
			}
			// Mark done ✅ when processing is complete
			if state.MessageID != "" {
				_ = deps.WahaClient.SendReaction(ctx, state.ChatID, state.MessageID, "✅")
			}
		}

		return graphw.NodeResult[entity.ZoraState]{
			State: entity.ZoraState{
				Messages:      []llmw.Message{assistantMsg},
				IsResolved:    true,
				Resolution:    response,
				CurrentStep:   entity.StepThink,
				SystemPrompt:  state.SystemPrompt,
				RelevantTools: state.RelevantTools,
				RetrievedDocs: state.RetrievedDocs,
			},
			Route: graphw.END,
		}, nil
	}
}

func actNode(deps AgentDeps) graphw.NodeFunc[entity.ZoraState] {
	return func(ctx context.Context, state entity.ZoraState) (graphw.NodeResult[entity.ZoraState], error) {
		results := make([]entity.ToolResult, 0, len(state.PendingCalls))

		for _, call := range state.PendingCalls {
			logw.CtxInfof(ctx, "actNode: executing tool %s", call.Name)

			// Handle built-in WAHA tools locally
			if isWahaBuiltinTool(call.Name) {
				result := executeWahaBuiltinTool(ctx, deps, &state, &call)
				results = append(results, result)
				continue
			}

			result, err := deps.ToolRegistry.CallTool(ctx, call.Name, call.Arguments)
			if err != nil {
				logw.CtxErrorf(ctx, "actNode: tool %s failed: %v", call.Name, err)
				result = entity.ToolResult{
					ToolCallID: call.ID,
					Name:       call.Name,
					Content:    fmt.Sprintf("error: %v", err),
					IsError:    true,
				}
			}
			results = append(results, result)
		}

		return graphw.NodeResult[entity.ZoraState]{
			State: entity.ZoraState{
				ToolResults: results,
				CurrentStep: entity.StepAct,
			},
		}, nil
	}
}

func observeNode(deps AgentDeps) graphw.NodeFunc[entity.ZoraState] {
	return func(ctx context.Context, state entity.ZoraState) (graphw.NodeResult[entity.ZoraState], error) {
		toolMessages := make([]llmw.Message, 0, len(state.ToolResults))
		for _, tr := range state.ToolResults {
			content := tr.Content
			if tr.IsError {
				content = fmt.Sprintf("[ERROR] %s", content)
			}
			toolMessages = append(toolMessages, llmw.Message{
				Role:       llmw.RoleTool,
				Content:    content,
				ToolCallID: tr.ToolCallID,
				Name:       tr.Name,
			})
		}

		newIteration := state.Iteration + 1
		logw.CtxInfof(ctx, "observeNode: iteration %d/%d, %d tool results",
			newIteration, state.MaxSteps, len(state.ToolResults))

		// Max iterations check
		if newIteration >= state.MaxSteps {
			logw.CtxWarningf(ctx, "observeNode: max iterations reached, forcing resolution")
			toolMessages = append(toolMessages, llmw.Message{
				Role:    llmw.RoleAssistant,
				Content: "Maximum iterations reached. I'll provide my best answer based on what I have so far.",
			})
			return graphw.NodeResult[entity.ZoraState]{
				State: entity.ZoraState{
					Messages:    toolMessages,
					Iteration:   newIteration,
					CurrentStep: entity.StepObserve,
					IsResolved:  true,
					Resolution:  "Maximum iterations reached.",
				},
				Route: graphw.END,
			}, nil
		}

		return graphw.NodeResult[entity.ZoraState]{
			State: entity.ZoraState{
				Messages:    toolMessages,
				Iteration:   newIteration,
				CurrentStep: entity.StepObserve,
			},
		}, nil
	}
}

// buildSystemPrompt constructs the system prompt with tool and knowledge context.
func buildSystemPrompt(task string, tools []entity.ToolContext, docs []entity.KnowledgeSnippet, source string, tags []string, planMode bool, quotedContext *entity.QuotedContext) string {
	prompt := "You are Zora, an autonomous agentic assistant. Use available tools and knowledge to help the user.\n\n"
	prompt += fmt.Sprintf("USER TASK: %s\n\n", task)

	if quotedContext != nil && len(quotedContext.Chain) > 0 {
		prompt += quotedContext.FormatChain() + "\n\n"
	}

	if len(tags) > 0 {
		prompt += fmt.Sprintf("MATCHED CATEGORIES: %s\n\n", strings.Join(tags, ", "))
	}

	if len(tools) > 0 {
		prompt += "AVAILABLE TOOLS:\n"
		for _, t := range tools {
			paramsJSON, _ := json.Marshal(t.Parameters)
			prompt += fmt.Sprintf("- %s: %s\n  Parameters: %s\n", t.Name, t.Description, string(paramsJSON))
		}
		prompt += "\n"
	}

	if len(docs) > 0 {
		prompt += "RELEVANT KNOWLEDGE:\n"
		for _, d := range docs {
			prompt += fmt.Sprintf("- [%s] %s\n", d.DocID, d.Content)
		}
		prompt += "\n"
	}

	if planMode {
		prompt += "PLAN MODE INSTRUCTIONS:\n"
		prompt += "- Do NOT answer the task directly. Instead, call the tools you would use to complete the task.\n"
		prompt += "- Your tool calls will be shown to the user as an execution plan for review.\n"
		prompt += "- Be specific with argument values — they will be visible to the user.\n"
		prompt += "- Call ALL the tools needed to complete the task, in the correct order.\n"
		prompt += "- If the user asked for changes to a previous plan, adjust the tool calls accordingly.\n\n"
	} else {
		prompt += "INSTRUCTIONS:\n"
		prompt += "1. Analyze the user's task carefully.\n"
		prompt += "2. Use available tools when you need to perform actions or retrieve data.\n"
		prompt += "3. Use the provided knowledge to inform your responses.\n"
		prompt += "4. When the task is complete, provide a clear final answer without calling tools.\n"
		prompt += "5. If you cannot complete the task, explain why and suggest alternatives.\n"
		prompt += "6. Never return an empty response. Always provide content or call a tool.\n"
		prompt += "7. When tools generate or save files, include the download URL in your final answer.\n"
		prompt += "8. Use internet-search when you need to find current information from the web.\n"
		prompt += "9. Use file-read to view existing files, file-write to update them, and file-save to create new files.\n"
		prompt += "10. When saving files, always include the download URL in your final answer so the user can access them.\n"
	}

	if source == "waha" {
		prompt += "\nWHATSAPP BEHAVIOR:\n"
		prompt += "- When you start processing a task, call waha-reaction with emoji 💭 to show you are thinking.\n"
		prompt += "- When you finish processing, call waha-reaction with an empty emoji to remove the 💭.\n"
		prompt += "- Your final answer will be delivered to the user automatically. Do NOT call any tool to send your response text.\n"
		prompt += "- Keep responses concise and suitable for WhatsApp chat.\n"
	}

	return prompt
}

// filterToolsByScore removes tools below the minimum similarity score threshold.
func filterToolsByScore(tools []entity.ToolContext, minScore float64) []entity.ToolContext {
	if minScore <= 0 {
		return tools
	}
	filtered := make([]entity.ToolContext, 0, len(tools))
	for _, t := range tools {
		if t.Score >= minScore {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// filterDocsByScore removes knowledge snippets below the minimum similarity score threshold.
func filterDocsByScore(docs []entity.KnowledgeSnippet, minScore float64) []entity.KnowledgeSnippet {
	if minScore <= 0 {
		return docs
	}
	filtered := make([]entity.KnowledgeSnippet, 0, len(docs))
	for _, d := range docs {
		if d.Score >= minScore {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// roughTokenEstimate estimates tokens at ~4 characters per token.
func roughTokenEstimate(text string) int {
	return len(text) / 4
}

// budgetTools truncates the tool list to fit within a token budget.
// Tools are included in order; descriptions are truncated if the last tool
// partially fits. A budget of 0 or less means no budgeting.
func budgetTools(tools []entity.ToolContext, maxTokens int) []entity.ToolContext {
	if maxTokens <= 0 || len(tools) == 0 {
		return tools
	}
	var budget []entity.ToolContext
	used := 0
	for _, t := range tools {
		est := roughTokenEstimate(t.Description)
		if used+est > maxTokens {
			remaining := (maxTokens - used) * 4
			if remaining > 20 {
				desc := t.Description
				if len(desc) > remaining {
					desc = desc[:remaining] + "..."
				}
				budget = append(budget, entity.ToolContext{
					ID: t.ID, Name: t.Name, Description: desc,
					Language: t.Language, Parameters: t.Parameters,
					Score: t.Score, Tags: t.Tags,
				})
			}
			break
		}
		budget = append(budget, t)
		used += est
	}
	return budget
}

// budgetDocs truncates the knowledge snippet list to fit within a token budget.
// Snippets are included in order; content is truncated if the last snippet
// partially fits. A budget of 0 or less means no budgeting.
func budgetDocs(docs []entity.KnowledgeSnippet, maxTokens int) []entity.KnowledgeSnippet {
	if maxTokens <= 0 || len(docs) == 0 {
		return docs
	}
	var budget []entity.KnowledgeSnippet
	used := 0
	for _, d := range docs {
		est := roughTokenEstimate(d.Content)
		if used+est > maxTokens {
			remaining := (maxTokens - used) * 4
			if remaining > 20 {
				content := d.Content
				if len(content) > remaining {
					content = content[:remaining] + "..."
				}
				budget = append(budget, entity.KnowledgeSnippet{
					DocID: d.DocID, Content: content,
					Score: d.Score, Metadata: d.Metadata, Tags: d.Tags,
				})
			}
			break
		}
		budget = append(budget, d)
		used += est
	}
	return budget
}

// wahaBuiltinTools returns the list of built-in WAHA UX tools that are always
// available when the agent is processing a WhatsApp message.
// Text delivery is handled automatically — no waha-send-text tool is needed.
func wahaBuiltinTools() []entity.ToolContext {
	return []entity.ToolContext{
		{
			Name:        "waha-reaction",
			Description: "React to a WhatsApp message with an emoji. Use 💭 when you start thinking and empty string to remove the reaction when done.",
			Parameters: map[string]any{
				"type":     "object",
				"required": []any{"chatId", "messageId", "emoji"},
				"properties": map[string]any{
					"chatId":    map[string]any{"type": "string", "description": "WhatsApp chat ID"},
					"messageId": map[string]any{"type": "string", "description": "WhatsApp message ID to react to"},
					"emoji":     map[string]any{"type": "string", "description": "Emoji reaction (empty string to remove)"},
				},
			},
		},
		{
			Name:        "waha-typing",
			Description: "Start or stop typing indicator in a WhatsApp chat.",
			Parameters: map[string]any{
				"type":     "object",
				"required": []any{"chatId", "action"},
				"properties": map[string]any{
					"chatId": map[string]any{"type": "string", "description": "WhatsApp chat ID"},
					"action": map[string]any{"type": "string", "enum": []any{"start", "stop"}, "description": "'start' or 'stop'"},
				},
			},
		},
		{
			Name:        "waha-send-seen",
			Description: "Mark messages as read in a WhatsApp chat.",
			Parameters: map[string]any{
				"type":     "object",
				"required": []any{"chatId"},
				"properties": map[string]any{
					"chatId": map[string]any{"type": "string", "description": "WhatsApp chat ID"},
				},
			},
		},
	}
}

// isWahaBuiltinTool checks if a tool name is a built-in WAHA tool.
func isWahaBuiltinTool(name string) bool {
	switch name {
	case "waha-reaction", "waha-typing", "waha-send-seen":
		return true
	default:
		return false
	}
}

// formatToolCallsAsPlan formats LLM tool calls as a human-readable execution plan.
func formatToolCallsAsPlan(toolCalls []llmw.ToolCall) string {
	var sb strings.Builder
	sb.WriteString("*Execution Plan:*\n\n")
	for i, tc := range toolCalls {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, tc.Name))
		var args map[string]any
		if json.Unmarshal([]byte(tc.Arguments), &args) == nil {
			// Sort keys for consistent output
			keys := make([]string, 0, len(args))
			for k := range args {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := args[k]
				s := fmt.Sprintf("%v", v)
				// Truncate long values
				if len(s) > 200 {
					s = s[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("   %s: %s\n", k, s))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Reply *execute* to run this plan, or tell me what to change.")
	return sb.String()
}

// executeWahaBuiltinTool handles built-in WAHA tool calls locally without going through MCP.
func executeWahaBuiltinTool(ctx context.Context, deps AgentDeps, state *entity.ZoraState, call *entity.ToolCall) entity.ToolResult {
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return entity.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    fmt.Sprintf("error parsing arguments: %v", err),
			IsError:    true,
		}
	}

	chatID := state.ChatID
	messageID := state.MessageID

	// Allow arguments to override context defaults
	if v, ok := args["chatId"].(string); ok && v != "" {
		chatID = v
	}

	switch call.Name {
	case "waha-reaction":
		emoji, _ := args["emoji"].(string)
		if mid, ok := args["messageId"].(string); ok && mid != "" {
			messageID = mid
		}
		if chatID == "" || messageID == "" {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: chatId and messageId are required for waha-reaction",
				IsError:    true,
			}
		}
		if err := deps.WahaClient.SendReaction(ctx, chatID, messageID, emoji); err != nil {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    fmt.Sprintf("error: %v", err),
				IsError:    true,
			}
		}
		return entity.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    "ok",
		}

	case "waha-typing":
		action, _ := args["action"].(string)
		if action != "start" && action != "stop" {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: action must be 'start' or 'stop'",
				IsError:    true,
			}
		}
		if chatID == "" {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: chatId is required",
				IsError:    true,
			}
		}
		var err error
		if action == "start" {
			err = deps.WahaClient.StartTyping(ctx, chatID)
		} else {
			err = deps.WahaClient.StopTyping(ctx, chatID)
		}
		if err != nil {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    fmt.Sprintf("error: %v", err),
				IsError:    true,
			}
		}
		return entity.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    "ok",
		}

	case "waha-send-seen":
		if chatID == "" {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    "error: chatId is required",
				IsError:    true,
			}
		}
		if err := deps.WahaClient.SendSeen(ctx, chatID); err != nil {
			return entity.ToolResult{
				ToolCallID: call.ID,
				Name:       call.Name,
				Content:    fmt.Sprintf("error: %v", err),
				IsError:    true,
			}
		}
		return entity.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    "ok",
		}

	default:
		return entity.ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Content:    fmt.Sprintf("error: unknown built-in tool %s", call.Name),
			IsError:    true,
		}
	}
}
