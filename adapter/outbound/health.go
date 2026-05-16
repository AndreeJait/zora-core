package outbound

import (
	"context"

	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// healthRepository implements portOutbound.HealthRepository using GORM and Redis.
type healthRepository struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewHealthRepository creates a new HealthRepository with GORM and Redis clients.
func NewHealthRepository(db *DB, redisConn *RedisConn) portOutbound.HealthRepository {
	return &healthRepository{
		db:    db.GormDB,
		redis: redisConn.Client,
	}
}

func (r *healthRepository) PingDB(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (r *healthRepository) PingRedis(ctx context.Context) error {
	return r.redis.Ping(ctx).Err()
}