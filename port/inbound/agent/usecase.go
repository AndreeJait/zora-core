package agent

import (
	"context"
	"iter"

	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/zora-core/domain/entity"
)

// UseCase defines the inbound port for agent execution.
type UseCase interface {
	// Execute runs the agent think-act-observe loop and returns the final state.
	Execute(ctx context.Context, input ExecuteInput) (ExecuteOutput, error)

	// Stream executes the agent and yields a Step after each superstep.
	Stream(ctx context.Context, input ExecuteInput) iter.Seq2[graphw.Step[entity.ZoraState], error]

	// Resume continues a paused (HITL) agent execution.
	Resume(ctx context.Context, threadID string, resumeValue any) (ExecuteOutput, error)

	// StepExecute runs one superstep of the graph for a new execution.
	// On the first call, it resolves the START edges and executes those nodes.
	StepExecute(ctx context.Context, input ExecuteInput) (*StepOutput, error)

	// StepContinue runs one superstep of an existing execution.
	// State is restored from the checkpointer for the given threadID.
	StepContinue(ctx context.Context, threadID string) (*StepOutput, error)

	// GetState retrieves the current state snapshot for a given thread.
	GetState(threadID string) (*graphw.StateSnapshot[entity.ZoraState], error)

	// ListCheckpoints returns checkpoint history for a thread.
	ListCheckpoints(threadID string) ([]graphw.Checkpoint, error)

	// Redirect changes the next nodes to execute for a given thread.
	Redirect(threadID string, nextNodes []string) error

	// RevertTo time-travels to a previous checkpoint.
	RevertTo(threadID, checkpointID string) error

	// MermaidDiagram returns the graph structure as a Mermaid flowchart string.
	MermaidDiagram() string
}

// StepOutput contains the result of a single graph superstep execution.
type StepOutput struct {
	State  entity.ZoraState
	Next   []string
	IsDone bool
}
