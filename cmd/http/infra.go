package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/brokerw"
	"github.com/AndreeJait/go-utility/v2/brokerw/nsqw"
	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/go-utility/v2/llmw/openaiw"
	"github.com/AndreeJait/zora-core/adapter/outbound"
	"github.com/AndreeJait/zora-core/config"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

// provideInfrastructure registers infrastructure providers into the dig container.
func provideInfrastructure(c *dig.Container) {
	c.Provide(newDB)
	c.Provide(newGormDB)
	c.Provide(newRedisConn)
	c.Provide(newRedisClient)
	c.Provide(newLLM)
	c.Provide(newEmbedder)
	c.Provide(newCheckpointer)
	c.Provide(newNSQProducer)
}

func newDB(cfg *config.AppConfig, cc *CleanupCollector) (*outbound.DB, error) {
	db, cleanup, err := outbound.ConnectSQL(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	cc.Add(cleanup)
	return db, nil
}

func newGormDB(db *outbound.DB) *gorm.DB {
	return db.GormDB
}

func newRedisConn(cfg *config.AppConfig, cc *CleanupCollector) (*outbound.RedisConn, error) {
	conn, cleanup, err := outbound.ConnectRedis(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	cc.Add(cleanup)
	return conn, nil
}

func newRedisClient(conn *outbound.RedisConn) *redis.Client {
	return conn.Client
}

// newLLM creates the LLM provider (OpenAI-compatible, pointed at Ollama).
// Cloud models (containing "-cloud") use the Ollama cloud endpoint instead of the configured base URL.
func newLLM(cfg *config.AppConfig, cc *CleanupCollector) (llmw.LLM, error) {
	baseURL := cfg.LLM.BaseURL
	if strings.Contains(cfg.LLM.Model, "cloud") {
		baseURL = "https://ollama.com"
	}
	llm, err := openaiw.New(&openaiw.Config{
		BaseURL: baseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
	})
	if err != nil {
		return nil, err
	}
	cc.Add(func(ctx context.Context) error {
		return llm.Close()
	})
	return llm, nil
}

// newEmbedder creates the embedding provider using Ollama's native /api/embed endpoint.
// EmbedBaseURL is used (defaults to BaseURL without /v1 suffix) because Ollama's
// native API doesn't use the /v1 prefix, unlike the OpenAI-compatible chat API.
func newEmbedder(cfg *config.AppConfig) portOutbound.Embedder {
	return outbound.NewOllamaEmbedder(outbound.OllamaEmbedderConfig{
		BaseURL: cfg.LLM.EmbedBaseURL,
		Model:   cfg.LLM.EmbedModel,
	})
}

// newCheckpointer creates the Redis-backed graph checkpointer.
func newCheckpointer(client *redis.Client) graphw.Checkpointer {
	return outbound.NewRedisCheckpointer(client)
}

// newNSQProducer creates the NSQ producer for task dispatching.
func newNSQProducer(cfg *config.AppConfig, cc *CleanupCollector) (brokerw.Producer, error) {
	if cfg.NSQ.NSQdAddr == "" {
		return nil, fmt.Errorf("nsq.nsqd_addr is required in config")
	}
	producer, err := nsqw.NewProducerWithRetry(nsqw.ProducerConfig{
		NSQdAddr:   cfg.NSQ.NSQdAddr,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQ producer: %w", err)
	}
	cc.Add(func(ctx context.Context) error {
		return producer.Close()
	})
	return producer, nil
}
