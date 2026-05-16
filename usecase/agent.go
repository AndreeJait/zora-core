package usecase

import (
	"context"
	"fmt"
	"iter"

	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/domain/entity"
	domainError "github.com/AndreeJait/zora-core/domain/error"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/outbound"
	"github.com/AndreeJait/zora-core/usecase/graph"
)

type agentUseCase struct {
	cfg          *config.AppConfig
	llm          llmw.LLM
	embedder     outbound.Embedder
	toolReg      outbound.ToolRegistryClient
	knowledge    outbound.KnowledgeClient
	tagExtractor outbound.TagExtractor
	convRepo     outbound.ConversationRepository
	checkpointer graphw.Checkpointer
	wahaClient   outbound.WahaClient
	planStore    outbound.PlanStore
	messageStore outbound.MessageStore
}

var _ agent.UseCase = (*agentUseCase)(nil)

// NewAgentUseCase creates the agent use case implementation.
func NewAgentUseCase(
	cfg *config.AppConfig,
	llm llmw.LLM,
	embedder outbound.Embedder,
	toolReg outbound.ToolRegistryClient,
	knowledge outbound.KnowledgeClient,
	tagExtractor outbound.TagExtractor,
	convRepo outbound.ConversationRepository,
	checkpointer graphw.Checkpointer,
	wahaClient outbound.WahaClient,
	planStore outbound.PlanStore,
	messageStore outbound.MessageStore,
) agent.UseCase {
	return &agentUseCase{
		cfg:          cfg,
		llm:          llm,
		embedder:     embedder,
		toolReg:      toolReg,
		knowledge:    knowledge,
		tagExtractor: tagExtractor,
		convRepo:     convRepo,
		checkpointer:  checkpointer,
		wahaClient:   wahaClient,
		planStore:    planStore,
		messageStore: messageStore,
	}
}

func (uc *agentUseCase) Execute(ctx context.Context, input agent.ExecuteInput) (agent.ExecuteOutput, error) {
	if input.Task == "" {
		return agent.ExecuteOutput{}, domainError.ErrEmptyTask.WithError(fmt.Errorf("task is empty"))
	}

	g, err := uc.buildGraph()
	if err != nil {
		return agent.ExecuteOutput{}, domainError.ErrGraphCompile.WithError(err)
	}

	initialState := uc.initialState(ctx, input)
	runOpts := []graphw.RunOption{graphw.WithThreadID(initialState.ThreadID)}

	finalState, err := g.Invoke(ctx, initialState, runOpts...)
	if err != nil {
		return agent.ExecuteOutput{}, fmt.Errorf("agent invoke: %w", err)
	}

	_ = uc.persistConversation(ctx, finalState)

	return agent.ExecuteOutput{
		TraceID:    finalState.TraceID,
		SessionID:  finalState.SessionID,
		ThreadID:   finalState.ThreadID,
		Resolution: finalState.Resolution,
		Iterations: finalState.Iteration,
		IsResolved: finalState.IsResolved,
	}, nil
}

func (uc *agentUseCase) Stream(ctx context.Context, input agent.ExecuteInput) iter.Seq2[graphw.Step[entity.ZoraState], error] {
	return func(yield func(graphw.Step[entity.ZoraState], error) bool) {
		if input.Task == "" {
			yield(graphw.Step[entity.ZoraState]{}, domainError.ErrEmptyTask.WithError(fmt.Errorf("task is empty")))
			return
		}

		g, err := uc.buildGraph()
		if err != nil {
			yield(graphw.Step[entity.ZoraState]{}, domainError.ErrGraphCompile.WithError(err))
			return
		}

		initialState := uc.initialState(ctx, input)
		runOpts := []graphw.RunOption{graphw.WithThreadID(initialState.ThreadID)}

		for step, err := range g.Stream(ctx, initialState, runOpts...) {
			if err != nil {
				yield(step, err)
				return
			}
			if !yield(step, nil) {
				return
			}
		}
	}
}

func (uc *agentUseCase) Resume(ctx context.Context, threadID string, resumeValue any) (agent.ExecuteOutput, error) {
	g, err := uc.buildGraph()
	if err != nil {
		return agent.ExecuteOutput{}, domainError.ErrGraphCompile.WithError(err)
	}

	ctx = graphw.WithResumeValue(ctx, resumeValue)
	finalState, err := g.Invoke(ctx, entity.ZoraState{ThreadID: threadID}, graphw.WithThreadID(threadID))
	if err != nil {
		return agent.ExecuteOutput{}, fmt.Errorf("agent resume: %w", err)
	}

	return agent.ExecuteOutput{
		TraceID:    finalState.TraceID,
		SessionID:  finalState.SessionID,
		ThreadID:   finalState.ThreadID,
		Resolution: finalState.Resolution,
		Iterations: finalState.Iteration,
		IsResolved: finalState.IsResolved,
	}, nil
}

func (uc *agentUseCase) buildGraph() (graphw.Graph[entity.ZoraState], error) {
	deps := graph.NewAgentDeps(uc.cfg, uc.llm, uc.embedder, uc.toolReg, uc.knowledge, uc.tagExtractor, uc.wahaClient, uc.planStore, uc.messageStore)
	return graph.BuildAgentGraph(deps)
}

func (uc *agentUseCase) MermaidDiagram() string {
	g, err := uc.buildGraph()
	if err != nil {
		return ""
	}
	return graphw.ExportMermaid(g)
}

func (uc *agentUseCase) StepExecute(ctx context.Context, input agent.ExecuteInput) (*agent.StepOutput, error) {
	if input.Task == "" {
		return nil, domainError.ErrEmptyTask.WithError(fmt.Errorf("task is empty"))
	}

	g, err := uc.buildGraph()
	if err != nil {
		return nil, domainError.ErrGraphCompile.WithError(err)
	}

	initialState := uc.initialState(ctx, input)
	stepResult, err := g.Step(ctx, initialState, graphw.WithThreadID(initialState.ThreadID))
	if err != nil {
		return nil, fmt.Errorf("agent step execute: %w", err)
	}

	if stepResult.IsDone {
		_ = uc.persistConversation(ctx, stepResult.State)
	}

	return &agent.StepOutput{
		State:  stepResult.State,
		Next:   stepResult.Next,
		IsDone: stepResult.IsDone,
	}, nil
}

func (uc *agentUseCase) StepContinue(ctx context.Context, threadID string) (*agent.StepOutput, error) {
	g, err := uc.buildGraph()
	if err != nil {
		return nil, domainError.ErrGraphCompile.WithError(err)
	}

	stepResult, err := g.Step(ctx, entity.ZoraState{}, graphw.WithThreadID(threadID))
	if err != nil {
		return nil, fmt.Errorf("agent step continue: %w", err)
	}

	if stepResult.IsDone {
		_ = uc.persistConversation(ctx, stepResult.State)
	}

	return &agent.StepOutput{
		State:  stepResult.State,
		Next:   stepResult.Next,
		IsDone: stepResult.IsDone,
	}, nil
}

func (uc *agentUseCase) GetState(threadID string) (*graphw.StateSnapshot[entity.ZoraState], error) {
	g, err := uc.buildGraph()
	if err != nil {
		return nil, domainError.ErrGraphCompile.WithError(err)
	}
	return g.GetState(threadID)
}

func (uc *agentUseCase) ListCheckpoints(threadID string) ([]graphw.Checkpoint, error) {
	if uc.checkpointer == nil {
		return nil, fmt.Errorf("no checkpointer configured")
	}
	return uc.checkpointer.List(context.Background(), threadID)
}

func (uc *agentUseCase) Redirect(threadID string, nextNodes []string) error {
	g, err := uc.buildGraph()
	if err != nil {
		return domainError.ErrGraphCompile.WithError(err)
	}
	return g.Redirect(threadID, nextNodes)
}

func (uc *agentUseCase) RevertTo(threadID, checkpointID string) error {
	g, err := uc.buildGraph()
	if err != nil {
		return domainError.ErrGraphCompile.WithError(err)
	}
	return g.RevertTo(threadID, checkpointID)
}

func (uc *agentUseCase) initialState(ctx context.Context, input agent.ExecuteInput) entity.ZoraState {
	threadID := input.ThreadID
	if threadID == "" {
		threadID = input.SessionID
	}

	traceID := logw.GetLogID(ctx)

	state := entity.ZoraState{
		TraceID:       traceID,
		ThreadID:      threadID,
		SessionID:     input.SessionID,
		TaskID:        input.TaskID,
		Task:          input.Task,
		MaxSteps:      uc.cfg.Agent.MaxSteps,
		Messages:      nil,
		ChatID:        input.ChatID,
		MessageID:     input.MessageID,
		Source:        input.Source,
		UserID:        input.UserID,
		IsAdmin:       input.IsAdmin,
		PlanMode:      input.PlanMode,
		QuotedContext: input.QuotedContext,
	}

	// Pre-load cached search state from a previous plan generation.
	// When set, thinkNode skips tool/knowledge search steps.
	if input.PreLoadedState != nil {
		state.SystemPrompt = input.PreLoadedState.SystemPrompt
		state.RelevantTools = input.PreLoadedState.RelevantTools
		state.RetrievedDocs = input.PreLoadedState.RetrievedDocs
		state.ExtractedTags = input.PreLoadedState.ExtractedTags
		// If ForceSearch is true, clear SystemPrompt so thinkNode re-runs search
		if input.PreLoadedState.ForceSearch {
			state.SystemPrompt = ""
		}
	}

	return state
}

func (uc *agentUseCase) persistConversation(ctx context.Context, state entity.ZoraState) error {
	if uc.convRepo == nil || state.SessionID == "" {
		return nil
	}
	conv := &entity.Conversation{
		SessionID: state.SessionID,
		Task:      state.Task,
		Messages:  state.Messages,
		Status:    "resolved",
	}
	if !state.IsResolved {
		conv.Status = "failed"
	}
	return uc.convRepo.Save(ctx, conv)
}
