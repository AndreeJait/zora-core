package health

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// UseCase defines the inbound port for health checking.
type UseCase interface {
	Check(ctx context.Context) *entity.Health
}