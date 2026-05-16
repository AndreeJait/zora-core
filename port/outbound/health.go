package outbound

import "context"

// HealthRepository defines the outbound port for health-checking infrastructure.
type HealthRepository interface {
	PingDB(ctx context.Context) error
	PingRedis(ctx context.Context) error
}