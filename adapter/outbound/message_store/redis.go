package message_store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
	"github.com/redis/go-redis/v9"
)

const msgKeyPrefix = "zora:msg:"

type redisMessageStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisMessageStore creates a Redis-backed MessageStore.
func NewRedisMessageStore(client *redis.Client, ttl time.Duration) outbound.MessageStore {
	return &redisMessageStore{client: client, ttl: ttl}
}

func (s *redisMessageStore) Store(ctx context.Context, msg *entity.WAHAMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal waha message: %w", err)
	}
	key := msgKeyPrefix + msg.ID
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("redis set message: %w", err)
	}
	return nil
}

func (s *redisMessageStore) Get(ctx context.Context, messageID string) (*entity.WAHAMessage, error) {
	key := msgKeyPrefix + messageID
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis get message: %w", err)
	}
	var msg entity.WAHAMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal waha message: %w", err)
	}
	return &msg, nil
}