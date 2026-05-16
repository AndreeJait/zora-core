package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AndreeJait/zora-core/port/outbound"
	"github.com/redis/go-redis/v9"
)

const planKeyPrefix = "zora:plan:"
const planTTL = 1 * time.Hour

type redisPlanStore struct {
	client *redis.Client
}

// NewRedisPlanStore creates a Redis-backed PlanStore.
func NewRedisPlanStore(client *redis.Client) outbound.PlanStore {
	return &redisPlanStore{client: client}
}

func (s *redisPlanStore) Save(ctx context.Context, chatID string, entry *outbound.PlanEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal plan entry: %w", err)
	}
	key := planKeyPrefix + chatID
	if err := s.client.Set(ctx, key, data, planTTL).Err(); err != nil {
		return fmt.Errorf("redis set plan: %w", err)
	}
	return nil
}

func (s *redisPlanStore) Get(ctx context.Context, chatID string) (*outbound.PlanEntry, error) {
	key := planKeyPrefix + chatID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // no pending plan
		}
		return nil, fmt.Errorf("redis get plan: %w", err)
	}
	var entry outbound.PlanEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal plan entry: %w", err)
	}
	return &entry, nil
}

func (s *redisPlanStore) Delete(ctx context.Context, chatID string) error {
	key := planKeyPrefix + chatID
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del plan: %w", err)
	}
	return nil
}