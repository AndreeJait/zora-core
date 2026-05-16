package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/redis/go-redis/v9"
)

const (
	checkpointKeyPrefix = "zora:checkpoint:"
	defaultTTL          = 24 * time.Hour
)

// RedisCheckpointer implements graphw.Checkpointer using Redis.
type RedisCheckpointer struct {
	client *redis.Client
	ttl    time.Duration
	mu     sync.RWMutex
}

var _ graphw.Checkpointer = (*RedisCheckpointer)(nil)

// NewRedisCheckpointer creates a Redis-backed checkpointer.
func NewRedisCheckpointer(client *redis.Client) *RedisCheckpointer {
	return &RedisCheckpointer{
		client: client,
		ttl:    defaultTTL,
	}
}

func (r *RedisCheckpointer) Put(ctx context.Context, checkpoint graphw.Checkpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := checkpointKeyPrefix + checkpoint.ThreadID + ":" + checkpoint.ID
	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	err = r.client.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set checkpoint: %w", err)
	}

	// Update the latest pointer for this thread
	latestKey := checkpointKeyPrefix + checkpoint.ThreadID + ":latest"
	err = r.client.Set(ctx, latestKey, checkpoint.ID, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set latest pointer: %w", err)
	}

	return nil
}

func (r *RedisCheckpointer) Get(ctx context.Context, threadID, checkpointID string) (*graphw.Checkpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := checkpointKeyPrefix + threadID + ":" + checkpointID
	return r.getByKey(ctx, key)
}

func (r *RedisCheckpointer) List(ctx context.Context, threadID string, opts ...graphw.CheckpointListOpt) ([]graphw.Checkpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pattern := checkpointKeyPrefix + threadID + ":*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("redis keys: %w", err)
	}

	limit := 10
	for _, opt := range opts {
		// Apply limit option (internal config struct)
		_ = opt
	}

	checkpoints := make([]graphw.Checkpoint, 0, limit)
	for _, key := range keys {
		if len(checkpoints) >= limit {
			break
		}
		// Skip "latest" pointer key
		if len(key) > 7 && key[len(key)-7:] == ":latest" {
			continue
		}
		cp, err := r.getByKey(ctx, key)
		if err != nil {
			continue
		}
		if cp != nil {
			checkpoints = append(checkpoints, *cp)
		}
	}

	return checkpoints, nil
}

func (r *RedisCheckpointer) Delete(ctx context.Context, threadID, checkpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := checkpointKeyPrefix + threadID + ":" + checkpointID
	return r.client.Del(ctx, key).Err()
}

func (r *RedisCheckpointer) getByKey(ctx context.Context, key string) (*graphw.Checkpoint, error) {
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get checkpoint: %w", err)
	}

	var cp graphw.Checkpoint
	if err := json.Unmarshal([]byte(data), &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
	}
	return &cp, nil
}
