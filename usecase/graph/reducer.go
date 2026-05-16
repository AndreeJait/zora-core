package graph

import (
	"github.com/AndreeJait/go-utility/v2/valuew"
	"github.com/AndreeJait/zora-core/domain/entity"
)

// reducer merges an update into the current ZoraState.
// It is used as the graphw.Reducer[entity.ZoraState].
func reducer(current, update entity.ZoraState) entity.ZoraState {
	// Append conversation messages
	current.Messages = append(current.Messages, update.Messages...)

	// Accumulate tool calls and results
	current.PendingCalls = append(current.PendingCalls, update.PendingCalls...)
	current.ToolResults = append(current.ToolResults, update.ToolResults...)
	current.RetrievedDocs = append(current.RetrievedDocs, update.RetrievedDocs...)

	// Scalar fields: non-zero update overwrites
	current.TraceID = valuew.Coalesce(update.TraceID, current.TraceID)
	current.Task = valuew.Coalesce(update.Task, current.Task)
	current.ThreadID = valuew.Coalesce(update.ThreadID, current.ThreadID)
	current.SessionID = valuew.Coalesce(update.SessionID, current.SessionID)
	current.TaskID = valuew.Coalesce(update.TaskID, current.TaskID)
	current.IsResolved = valuew.Coalesce(update.IsResolved, current.IsResolved)
	current.Resolution = valuew.Coalesce(update.Resolution, current.Resolution)
	current.CurrentStep = valuew.Coalesce(update.CurrentStep, current.CurrentStep)
	current.LastError = valuew.Coalesce(update.LastError, current.LastError)
	current.SystemPrompt = valuew.Coalesce(update.SystemPrompt, current.SystemPrompt)
	current.Iteration = valuew.Coalesce(update.Iteration, current.Iteration)
	current.RetryCount = valuew.Coalesce(update.RetryCount, current.RetryCount)
	current.MaxSteps = valuew.Coalesce(update.MaxSteps, current.MaxSteps)
	current.UserID = valuew.Coalesce(update.UserID, current.UserID)

	// IsAdmin uses explicit check because Coalesce(true, false) would be true
	// (the zero value for bool), but we only want to propagate explicit admin status.
	if update.IsAdmin {
		current.IsAdmin = true
	}

	// Deep-merge metadata maps
	if update.Metadata != nil {
		if current.Metadata == nil {
			current.Metadata = make(map[string]any)
		}
		for k, v := range update.Metadata {
			current.Metadata[k] = v
		}
	}

	// Slice fields: replace if non-nil (not append)
	if update.RelevantTools != nil {
		current.RelevantTools = update.RelevantTools
	}
	if update.ExtractedTags != nil {
		current.ExtractedTags = update.ExtractedTags
	}

	return current
}
