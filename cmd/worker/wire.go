package main

import (
	"context"
	"time"

	"github.com/AndreeJait/go-utility/v2/brokerw"
	"github.com/AndreeJait/go-utility/v2/brokerw/nsqw"
	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/go-utility/v2/llmw/openaiw"
	adminAdp "github.com/AndreeJait/zora-core/adapter/outbound/admin"
	"github.com/AndreeJait/zora-core/adapter/outbound"
	convRepo "github.com/AndreeJait/zora-core/adapter/outbound/conversation"
	msgStoreAdp "github.com/AndreeJait/zora-core/adapter/outbound/message_store"
	settingAdp "github.com/AndreeJait/zora-core/adapter/outbound/setting"
	taskAdp "github.com/AndreeJait/zora-core/adapter/outbound/task"
	wlRepo "github.com/AndreeJait/zora-core/adapter/outbound/whitelist"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
	"github.com/AndreeJait/zora-core/usecase"
	"github.com/redis/go-redis/v9"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

var digContainer *dig.Container

// CleanupCollector accumulates cleanup functions from infrastructure providers.
type CleanupCollector struct {
	cleanups []func(ctx context.Context) error
}

// Add appends a cleanup function.
func (cc *CleanupCollector) Add(fn func(ctx context.Context) error) {
	cc.cleanups = append(cc.cleanups, fn)
}

// Cleanup runs all collected cleanup functions, returning the first error.
func (cc *CleanupCollector) Cleanup(ctx context.Context) error {
	var firstErr error
	for _, fn := range cc.cleanups {
		if err := fn(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func wireWorker(cfg *config.AppConfig) (func(ctx context.Context) error, error) {
	c := dig.New()
	digContainer = c

	cc := &CleanupCollector{}
	c.Provide(func() *CleanupCollector { return cc })
	c.Provide(func() *config.AppConfig { return cfg })

	provideWorkerInfrastructure(c)
	provideWorkerServices(c)

	return cc.Cleanup, nil
}

func provideWorkerInfrastructure(c *dig.Container) {
	c.Provide(newWorkerDB)
	c.Provide(newWorkerGormDB)
	c.Provide(newWorkerRedisConn)
	c.Provide(newWorkerRedisClient)
	c.Provide(newWorkerLLM)
	c.Provide(newWorkerEmbedder)
	c.Provide(newWorkerCheckpointer)
	c.Provide(newWorkerNSQProducer)
	c.Provide(newWorkerNSQConsumer)
}

func provideWorkerServices(c *dig.Container) {
	c.Provide(newWorkerToolRegistryClient)
	c.Provide(newWorkerKnowledgeClient)
	c.Provide(newWorkerConversationRepository)
	c.Provide(newWorkerWhitelistRepository)
	c.Provide(newWorkerAdminRepository)
	c.Provide(newWorkerWhitelistUseCase)
	c.Provide(newWorkerTagExtractor)
	c.Provide(newWorkerAgentUseCase)
	c.Provide(newWorkerWahaClient)
	c.Provide(newWorkerPlanStore)
	c.Provide(newWorkerMessageStore)
	c.Provide(newWorkerChainResolver)
	c.Provide(newWorkerTaskRepository)
	c.Provide(newWorkerSettingRepository)
	c.Provide(newWorkerTaskDispatcher)
	c.Provide(newWorkerRetrySweeper)
	c.Provide(newWorkerTaskHandler)
	c.Provide(newWorkerGraphStepHandler)
}

// --- Infrastructure providers ---

func newWorkerDB(cfg *config.AppConfig, cc *CleanupCollector) (*outbound.DB, error) {
	db, cleanup, err := outbound.ConnectSQL(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	cc.Add(cleanup)
	return db, nil
}

func newWorkerGormDB(db *outbound.DB) *gorm.DB {
	return db.GormDB
}

func newWorkerRedisConn(cfg *config.AppConfig, cc *CleanupCollector) (*outbound.RedisConn, error) {
	conn, cleanup, err := outbound.ConnectRedis(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	cc.Add(cleanup)
	return conn, nil
}

func newWorkerRedisClient(conn *outbound.RedisConn) *redis.Client {
	return conn.Client
}

func newWorkerLLM(cfg *config.AppConfig) (llmw.LLM, error) {
	return openaiw.New(&openaiw.Config{
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
		Model:   cfg.LLM.Model,
	})
}

func newWorkerEmbedder(cfg *config.AppConfig) portOutbound.Embedder {
	return outbound.NewOllamaEmbedder(outbound.OllamaEmbedderConfig{
		BaseURL: cfg.LLM.EmbedBaseURL,
		Model:   cfg.LLM.EmbedModel,
	})
}

func newWorkerCheckpointer(client *redis.Client) graphw.Checkpointer {
	return outbound.NewRedisCheckpointer(client)
}

func newWorkerNSQProducer(cfg *config.AppConfig) (brokerw.Producer, error) {
	return nsqw.NewProducerWithRetry(nsqw.ProducerConfig{
		NSQdAddr:   cfg.NSQ.NSQdAddr,
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	})
}

func newWorkerNSQConsumer(cfg *config.AppConfig) brokerw.Consumer {
	return nsqw.NewConsumer(cfg.NSQ.LookupdAddrs, cfg.NSQ.Channel)
}

// --- Service providers ---

func newWorkerToolRegistryClient(cfg *config.AppConfig) portOutbound.ToolRegistryClient {
	return outbound.NewToolRegistryHTTP(cfg.MCPServer.BaseURL, cfg.MCPServer.APIKey)
}

func newWorkerKnowledgeClient(cfg *config.AppConfig) portOutbound.KnowledgeClient {
	return outbound.NewKnowledgeHTTP(cfg.Knowledge.BaseURL)
}

func newWorkerConversationRepository(db *gorm.DB) portOutbound.ConversationRepository {
	return convRepo.NewRepository(db)
}

func newWorkerWhitelistRepository(db *gorm.DB) portOutbound.WhitelistRepository {
	return wlRepo.NewRepository(db)
}

func newWorkerAdminRepository(db *gorm.DB) portOutbound.AdminRepository {
	return adminAdp.NewRepository(db)
}

func newWorkerWhitelistUseCase(repo portOutbound.WhitelistRepository, adminRepo portOutbound.AdminRepository, wahaClient portOutbound.WahaClient, cfg *config.AppConfig) whitelist.UseCase {
	return usecase.NewWhitelistUseCase(repo, adminRepo, wahaClient, cfg)
}

func newWorkerTagExtractor(llm llmw.LLM) portOutbound.TagExtractor {
	return outbound.NewTagExtractorAdapter(llm)
}

func newWorkerAgentUseCase(
	cfg *config.AppConfig,
	llm llmw.LLM,
	embedder portOutbound.Embedder,
	toolReg portOutbound.ToolRegistryClient,
	knowledge portOutbound.KnowledgeClient,
	tagExtractor portOutbound.TagExtractor,
	convRepo portOutbound.ConversationRepository,
	checkpointer graphw.Checkpointer,
	wahaClient portOutbound.WahaClient,
	planStore portOutbound.PlanStore,
	messageStore portOutbound.MessageStore,
) agent.UseCase {
	return usecase.NewAgentUseCase(cfg, llm, embedder, toolReg, knowledge, tagExtractor, convRepo, checkpointer, wahaClient, planStore, messageStore)
}

func newWorkerWahaClient(cfg *config.AppConfig) portOutbound.WahaClient {
	return outbound.NewWahaClient(cfg)
}

func newWorkerPlanStore(redisConn *outbound.RedisConn) portOutbound.PlanStore {
	return outbound.NewRedisPlanStore(redisConn.Client)
}

func newWorkerMessageStore(cfg *config.AppConfig, redisConn *outbound.RedisConn) portOutbound.MessageStore {
	return msgStoreAdp.NewRedisMessageStore(redisConn.Client, cfg.MessageStore.TTL)
}

func newWorkerChainResolver(messageStore portOutbound.MessageStore) *usecase.ChainResolver {
	return usecase.NewChainResolver(messageStore)
}

func newWorkerTaskRepository(db *gorm.DB) portOutbound.TaskRepository {
	return taskAdp.NewRepository(db)
}

func newWorkerSettingRepository(db *gorm.DB) portOutbound.SettingRepository {
	return settingAdp.NewRepository(db)
}

func newWorkerTaskDispatcher(producer brokerw.Producer) *usecase.TaskDispatcher {
	return usecase.NewTaskDispatcher(producer)
}

func newWorkerRetrySweeper(taskRepo portOutbound.TaskRepository, dispatcher *usecase.TaskDispatcher) *usecase.RetrySweeper {
	return usecase.NewRetrySweeper(taskRepo, dispatcher)
}

func newWorkerTaskHandler(
	taskRepo portOutbound.TaskRepository,
	agentUC agent.UseCase,
	settingRepo portOutbound.SettingRepository,
	whitelistUC whitelist.UseCase,
	wahaClient portOutbound.WahaClient,
	dispatcher *usecase.TaskDispatcher,
) *usecase.TaskHandler {
	return usecase.NewTaskHandler(taskRepo, agentUC, settingRepo, whitelistUC, wahaClient, dispatcher)
}

func newWorkerGraphStepHandler(
	taskRepo portOutbound.TaskRepository,
	agentUC agent.UseCase,
	settingRepo portOutbound.SettingRepository,
	whitelistUC whitelist.UseCase,
	wahaClient portOutbound.WahaClient,
	dispatcher *usecase.TaskDispatcher,
) *usecase.GraphStepHandler {
	return usecase.NewGraphStepHandler(taskRepo, agentUC, settingRepo, whitelistUC, wahaClient, dispatcher)
}