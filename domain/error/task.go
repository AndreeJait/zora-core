package error

import "github.com/AndreeJait/go-utility/v2/statusw"

var (
	ErrTaskNotFound = statusw.NotFound.WithCustomMessage("task not found")
	ErrGraphNotReady = statusw.NotFound.WithCustomMessage("graph not available for this task")
)