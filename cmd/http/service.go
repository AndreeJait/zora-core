package main

import (
	"github.com/AndreeJait/go-utility/v2/brokerw"
	"github.com/AndreeJait/go-utility/v2/graphw"
	"github.com/AndreeJait/go-utility/v2/llmw"
	adminAdp "github.com/AndreeJait/zora-core/adapter/outbound/admin"
	"github.com/AndreeJait/zora-core/adapter/outbound"
	convRepo "github.com/AndreeJait/zora-core/adapter/outbound/conversation"
	msgStoreAdp "github.com/AndreeJait/zora-core/adapter/outbound/message_store"
	settingAdp "github.com/AndreeJait/zora-core/adapter/outbound/setting"
	storageAdp "github.com/AndreeJait/zora-core/adapter/outbound/storage"
	taskAdp "github.com/AndreeJait/zora-core/adapter/outbound/task"
	wlRepo "github.com/AndreeJait/zora-core/adapter/outbound/whitelist"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/health"
	"github.com/AndreeJait/zora-core/port/inbound/setting"
	"github.com/AndreeJait/zora-core/port/inbound/task"
	"github.com/AndreeJait/zora-core/port/inbound/upload"
	"github.com/AndreeJait/zora-core/port/inbound/webhook"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
	"github.com/AndreeJait/zora-core/usecase"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

// provideServices registers repository and use case providers into the dig container.
func provideServices(c *dig.Container) {
	// kyan:provider:start
	c.Provide(newHealthRepository)
	c.Provide(newHealthUseCase)

	c.Provide(newToolRegistryClient)
	c.Provide(newKnowledgeClient)
	c.Provide(newConversationRepository)
	c.Provide(newWhitelistRepository)
	c.Provide(newAdminRepository)
	c.Provide(newWhitelistUseCase)
	c.Provide(newTagExtractor)
	c.Provide(newAgentUseCase)

	c.Provide(newStorage)
	c.Provide(newUploadUseCase)

	c.Provide(newWahaClient)
	c.Provide(newPlanStore)
	c.Provide(newMessageStore)
	c.Provide(newChainResolver)
	c.Provide(newWebhookUseCase)

	c.Provide(newTaskRepository)
	c.Provide(newSettingRepository)
	c.Provide(newTaskUseCase)
	c.Provide(newSettingUseCase)
	c.Provide(newTaskDispatcher)
	c.Provide(newRetrySweeper)
	// kyan:provider:end
}

func newHealthRepository(db *outbound.DB, redisConn *outbound.RedisConn) portOutbound.HealthRepository {
	return outbound.NewHealthRepository(db, redisConn)
}

func newHealthUseCase(cfg *config.AppConfig, repo portOutbound.HealthRepository) health.UseCase {
	return usecase.NewHealthUseCase(cfg.App.Name, repo)
}

func newToolRegistryClient(cfg *config.AppConfig) portOutbound.ToolRegistryClient {
	return outbound.NewToolRegistryHTTP(cfg.MCPServer.BaseURL, cfg.MCPServer.APIKey)
}

func newKnowledgeClient(cfg *config.AppConfig) portOutbound.KnowledgeClient {
	return outbound.NewKnowledgeHTTP(cfg.Knowledge.BaseURL)
}

func newConversationRepository(db *gorm.DB) portOutbound.ConversationRepository {
	return convRepo.NewRepository(db)
}

func newWhitelistRepository(db *gorm.DB) portOutbound.WhitelistRepository {
	return wlRepo.NewRepository(db)
}

func newAdminRepository(db *gorm.DB) portOutbound.AdminRepository {
	return adminAdp.NewRepository(db)
}

func newWhitelistUseCase(repo portOutbound.WhitelistRepository, adminRepo portOutbound.AdminRepository, wahaClient portOutbound.WahaClient, cfg *config.AppConfig) whitelist.UseCase {
	return usecase.NewWhitelistUseCase(repo, adminRepo, wahaClient, cfg)
}

func newTagExtractor(llm llmw.LLM) portOutbound.TagExtractor {
	return outbound.NewTagExtractorAdapter(llm)
}

func newAgentUseCase(
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

func newWahaClient(cfg *config.AppConfig) portOutbound.WahaClient {
	return outbound.NewWahaClient(cfg)
}

func newPlanStore(redisConn *outbound.RedisConn) portOutbound.PlanStore {
	return outbound.NewRedisPlanStore(redisConn.Client)
}

func newMessageStore(cfg *config.AppConfig, redisConn *outbound.RedisConn) portOutbound.MessageStore {
	return msgStoreAdp.NewRedisMessageStore(redisConn.Client, cfg.MessageStore.TTL)
}

func newChainResolver(messageStore portOutbound.MessageStore) *usecase.ChainResolver {
	return usecase.NewChainResolver(messageStore)
}

func newWebhookUseCase(wahaClient portOutbound.WahaClient, agentUC agent.UseCase, whitelistUC whitelist.UseCase, taskRepo portOutbound.TaskRepository, dispatcher *usecase.TaskDispatcher, planStore portOutbound.PlanStore, messageStore portOutbound.MessageStore, chainResolver *usecase.ChainResolver, cfg *config.AppConfig) webhook.UseCase {
	return usecase.NewWebhookUseCase(wahaClient, agentUC, whitelistUC, taskRepo, dispatcher, planStore, messageStore, chainResolver, cfg.Task.DefaultMaxRetry)
}

func newStorage(cfg *config.AppConfig) (portOutbound.Storage, error) {
	return storageAdp.NewStorage(cfg)
}

func newUploadUseCase(storage portOutbound.Storage) upload.UseCase {
	return usecase.NewUploadUseCase(storage)
}

func newTaskRepository(db *gorm.DB) portOutbound.TaskRepository {
	return taskAdp.NewRepository(db)
}

func newSettingRepository(db *gorm.DB) portOutbound.SettingRepository {
	return settingAdp.NewRepository(db)
}

func newTaskUseCase(taskRepo portOutbound.TaskRepository, storage portOutbound.Storage) task.UseCase {
	return usecase.NewTaskUseCase(taskRepo, storage)
}

func newSettingUseCase(settingRepo portOutbound.SettingRepository) setting.UseCase {
	return usecase.NewSettingUseCase(settingRepo)
}

func newTaskDispatcher(producer brokerw.Producer) *usecase.TaskDispatcher {
	return usecase.NewTaskDispatcher(producer)
}

func newRetrySweeper(taskRepo portOutbound.TaskRepository, dispatcher *usecase.TaskDispatcher) *usecase.RetrySweeper {
	return usecase.NewRetrySweeper(taskRepo, dispatcher)
}