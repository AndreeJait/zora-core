package error

import "github.com/AndreeJait/go-utility/v2/statusw"

var (
	ErrInternalServer = statusw.InternalServerError
	ErrNotFound       = statusw.NotFound
	ErrInvalidParam   = statusw.InvalidReqParam
)