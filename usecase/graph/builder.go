package graph

import (
	"context"
	"fmt"

	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/zora-core/domain/entity"
)

// BuildAgentGraph creates and compiles the think-act-observe state graph.
//
// Topology:
//
//	START → think → [has tool_calls? → act → observe → think | → END]
//	observe → [resolved? → END | → think]
func BuildAgentGraph(deps AgentDeps) (graphw.Graph[entity.ZoraState], error) {
	builder := graphw.NewBuilder[entity.ZoraState](reducer)

	builder.AddNode("think", thinkNode(deps))
	builder.AddNode("act", actNode(deps))
	builder.AddNode("observe", observeNode(deps))

	builder.AddEdge(graphw.START, "think")

	// think routing: NodeResult.Route takes priority.
	// This conditional edge acts as a fallback safety net.
	builder.AddConditionalEdge("think",
		func(ctx context.Context, state entity.ZoraState) (string, error) {
			if state.IsResolved {
				return graphw.END, nil
			}
			if len(state.PendingCalls) > 0 {
				return "act", nil
			}
			return graphw.END, nil
		},
		map[string]string{
			"act":      "act",
			"__end__":  graphw.END,
		},
	)

	builder.AddEdge("act", "observe")

	// observe routing
	builder.AddConditionalEdge("observe",
		func(ctx context.Context, state entity.ZoraState) (string, error) {
			if state.IsResolved {
				return graphw.END, nil
			}
			return "think", nil
		},
		map[string]string{
			"think":   "think",
			"__end__": graphw.END,
		},
	)

	graph, err := builder.Compile(
		graphw.WithRecursionLimit(deps.ToolLimit*2+10),
	)
	if err != nil {
		return nil, fmt.Errorf("compile agent graph: %w", err)
	}

	return graph, nil
}
