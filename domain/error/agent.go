package error

import "github.com/AndreeJait/go-utility/v2/statusw"

var (
	ErrMaxIterations   = statusw.InternalServerError.WithCustomMessage("agent exceeded maximum iterations")
	ErrToolExecution   = statusw.InternalServerError.WithCustomMessage("tool execution failed")
	ErrNoToolsRelevant = statusw.InvalidReqParam.WithCustomMessage("no relevant tools found for task")
	ErrEmptyTask       = statusw.InvalidReqParam.WithCustomMessage("task must not be empty")
	ErrSessionNotFound = statusw.NotFound.WithCustomMessage("session not found")
	ErrGraphCompile    = statusw.InternalServerError.WithCustomMessage("failed to compile agent graph")
)
