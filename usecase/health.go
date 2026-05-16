package usecase

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/health"
	"github.com/AndreeJait/zora-core/port/outbound"
)

type healthUseCase struct {
	serviceName string
	healthRepo  outbound.HealthRepository
}

// NewHealthUseCase creates a new HealthUseCase implementation.
// healthRepo is optional — pass nil to skip DB/Redis ping checks.
func NewHealthUseCase(serviceName string, healthRepo outbound.HealthRepository) health.UseCase {
	return &healthUseCase{serviceName: serviceName, healthRepo: healthRepo}
}

func (u *healthUseCase) Check(ctx context.Context) *entity.Health {
	h := entity.NewHealth(u.serviceName)

	if u.healthRepo != nil {
		if err := u.healthRepo.PingDB(ctx); err != nil {
			h.Status = "unhealthy: db - " + err.Error()
			return h
		}
		if err := u.healthRepo.PingRedis(ctx); err != nil {
			h.Status = "unhealthy: redis - " + err.Error()
			return h
		}
	}

	return h
}
