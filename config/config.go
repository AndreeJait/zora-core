package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/configw"
	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/go-utility/v2/valuew"
)

// AppConfig holds all application configuration.
type AppConfig struct {
	App struct {
		Name     string `mapstructure:"name"`
		Env      string `mapstructure:"env"`
		HTTPPort int    `mapstructure:"http_port"`
	} `mapstructure:"app"`

	HTTP struct {
		Engine        string `mapstructure:"engine"`
		EnableSwagger bool   `mapstructure:"enable_swagger"`
		DebugMode     bool   `mapstructure:"debug_mode"`
		APIKey        string `mapstructure:"api_key"`
	} `mapstructure:"http"`

	Log struct {
		Level       string         `mapstructure:"level"`
		Format      logw.LogFormat `mapstructure:"format"`
		WriteToFile bool           `mapstructure:"write_to_file"`
		FilePath    string         `mapstructure:"file_path"`
	} `mapstructure:"log"`

	DB struct {
		Driver          string        `mapstructure:"driver"`
		Dialect         string        `mapstructure:"dialect"`
		DSN             string        `mapstructure:"dsn"`
		MaxOpenConns    int           `mapstructure:"max_open_conns"`
		MaxIdleConns    int           `mapstructure:"max_idle_conns"`
		ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
		DebugMode       bool          `mapstructure:"debug_mode"`
	} `mapstructure:"db"`

	Redis struct {
		Address   string `mapstructure:"address"`
		Password  string `mapstructure:"password"`
		DB        int    `mapstructure:"db"`
		PoolSize  int    `mapstructure:"pool_size"`
		DebugMode bool   `mapstructure:"debug_mode"`
	} `mapstructure:"redis"`

	Agent struct {
		MaxSteps                  int     `mapstructure:"max_steps"`
		ToolLimit                 int     `mapstructure:"tool_limit"`
		KnowledgeLimit            int     `mapstructure:"knowledge_limit"`
		ToolMinScore              float64 `mapstructure:"tool_min_score"`
		KnowledgeMinScore         float64 `mapstructure:"knowledge_min_score"`
		MaxToolContextTokens       int     `mapstructure:"max_tool_context_tokens"`
		MaxKnowledgeContextTokens  int     `mapstructure:"max_knowledge_context_tokens"`
		ToolsEnabled               bool    `mapstructure:"tools_enabled"`
		KnowledgeEnabled           bool    `mapstructure:"knowledge_enabled"`
	} `mapstructure:"agent"`

	LLM struct {
		Provider     string  `mapstructure:"provider"`
		Model        string  `mapstructure:"model"`
		EmbedModel   string  `mapstructure:"embed_model"`
		BaseURL      string  `mapstructure:"base_url"`
		EmbedBaseURL string  `mapstructure:"embed_base_url"`
		APIKey       string  `mapstructure:"api_key"`
		Temperature  float64 `mapstructure:"temperature"`
		MaxTokens    int     `mapstructure:"max_tokens"`
	} `mapstructure:"llm"`

	MinIO struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
		UseSSL    bool   `mapstructure:"use_ssl"`
		Region    string `mapstructure:"region"`
	} `mapstructure:"minio"`

	Knowledge struct {
		BaseURL string `mapstructure:"base_url"`
	} `mapstructure:"knowledge"`

	MCPServer struct {
		BaseURL string `mapstructure:"base_url"`
		APIKey  string `mapstructure:"api_key"`
	} `mapstructure:"mcp_server"`

	WAHA struct {
		BaseURL     string `mapstructure:"base_url"`
		APIKey      string `mapstructure:"api_key"`
		Session     string `mapstructure:"session"`
		AdminChatID string `mapstructure:"admin_chat_id"`
	} `mapstructure:"waha"`

	Whitelist struct {
		Admins             []string `mapstructure:"admins"`
		DefaultTokensPerHour int     `mapstructure:"default_tokens_per_hour"`
	} `mapstructure:"whitelist"`

	Graceful struct {
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	} `mapstructure:"graceful"`

	Task struct {
		WorkerCount       int           `mapstructure:"worker_count"`
		ChannelSize       int           `mapstructure:"channel_size"`
		WorkerTimeout     time.Duration `mapstructure:"worker_timeout"`
		DefaultMaxRetry   int           `mapstructure:"default_max_retry"`
		DefaultRetryDelay time.Duration `mapstructure:"default_retry_delay"`
	} `mapstructure:"task"`

	MessageStore struct {
		TTL time.Duration `mapstructure:"ttl"`
	} `mapstructure:"message_store"`

	NSQ struct {
		NSQdAddr     string   `mapstructure:"nsqd_addr"`
		LookupdAddrs []string `mapstructure:"lookupd_addrs"`
		Channel      string   `mapstructure:"channel"`
	} `mapstructure:"nsq"`
}

// Load reads the base config file using configw.Load, then merges app.local.yaml
// from the same directory if it exists. Environment variables override both.
func Load(configPath string) (*AppConfig, error) {
	cfg := &AppConfig{}
	if err := configw.Load(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Merge local overrides if app.local.yaml exists alongside the base config
	localPath := strings.Replace(configPath, "app.yaml", "app.local.yaml", 1)
	if _, err := os.Stat(localPath); err == nil {
		localCfg := &AppConfig{}
		if err := configw.Load(localPath, localCfg); err != nil {
			return nil, fmt.Errorf("failed to load local config: %w", err)
		}
		mergeNonZero(cfg, localCfg)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// mergeNonZero overwrites dst fields with non-zero values from src.
func mergeNonZero(dst, src *AppConfig) {
	dst.App.Name = valuew.Coalesce(src.App.Name, dst.App.Name)
	dst.App.Env = valuew.Coalesce(src.App.Env, dst.App.Env)
	dst.App.HTTPPort = valuew.Coalesce(src.App.HTTPPort, dst.App.HTTPPort)
	dst.HTTP.Engine = valuew.Coalesce(src.HTTP.Engine, dst.HTTP.Engine)
	dst.HTTP.EnableSwagger = valuew.Coalesce(src.HTTP.EnableSwagger, dst.HTTP.EnableSwagger)
	dst.HTTP.DebugMode = valuew.Coalesce(src.HTTP.DebugMode, dst.HTTP.DebugMode)
	dst.Log.Level = valuew.Coalesce(src.Log.Level, dst.Log.Level)
	dst.Log.Format = valuew.Coalesce(src.Log.Format, dst.Log.Format)
	dst.Log.WriteToFile = valuew.Coalesce(src.Log.WriteToFile, dst.Log.WriteToFile)
	dst.Log.FilePath = valuew.Coalesce(src.Log.FilePath, dst.Log.FilePath)
	dst.DB.Driver = valuew.Coalesce(src.DB.Driver, dst.DB.Driver)
	dst.DB.Dialect = valuew.Coalesce(src.DB.Dialect, dst.DB.Dialect)
	dst.DB.DSN = valuew.Coalesce(src.DB.DSN, dst.DB.DSN)
	dst.DB.MaxOpenConns = valuew.Coalesce(src.DB.MaxOpenConns, dst.DB.MaxOpenConns)
	dst.DB.MaxIdleConns = valuew.Coalesce(src.DB.MaxIdleConns, dst.DB.MaxIdleConns)
	dst.DB.ConnMaxLifetime = valuew.Coalesce(src.DB.ConnMaxLifetime, dst.DB.ConnMaxLifetime)
	dst.DB.DebugMode = valuew.Coalesce(src.DB.DebugMode, dst.DB.DebugMode)
	dst.Redis.Address = valuew.Coalesce(src.Redis.Address, dst.Redis.Address)
	dst.Redis.Password = valuew.Coalesce(src.Redis.Password, dst.Redis.Password)
	dst.Redis.DB = valuew.Coalesce(src.Redis.DB, dst.Redis.DB)
	dst.Redis.PoolSize = valuew.Coalesce(src.Redis.PoolSize, dst.Redis.PoolSize)
	dst.Redis.DebugMode = valuew.Coalesce(src.Redis.DebugMode, dst.Redis.DebugMode)
	dst.Agent.MaxSteps = valuew.Coalesce(src.Agent.MaxSteps, dst.Agent.MaxSteps)
	dst.Agent.ToolLimit = valuew.Coalesce(src.Agent.ToolLimit, dst.Agent.ToolLimit)
	dst.Agent.KnowledgeLimit = valuew.Coalesce(src.Agent.KnowledgeLimit, dst.Agent.KnowledgeLimit)
	dst.Agent.ToolMinScore = valuew.Coalesce(src.Agent.ToolMinScore, dst.Agent.ToolMinScore)
	dst.Agent.KnowledgeMinScore = valuew.Coalesce(src.Agent.KnowledgeMinScore, dst.Agent.KnowledgeMinScore)
	dst.Agent.MaxToolContextTokens = valuew.Coalesce(src.Agent.MaxToolContextTokens, dst.Agent.MaxToolContextTokens)
	dst.Agent.MaxKnowledgeContextTokens = valuew.Coalesce(src.Agent.MaxKnowledgeContextTokens, dst.Agent.MaxKnowledgeContextTokens)
	dst.Agent.ToolsEnabled = valuew.Coalesce(src.Agent.ToolsEnabled, dst.Agent.ToolsEnabled)
	dst.Agent.KnowledgeEnabled = valuew.Coalesce(src.Agent.KnowledgeEnabled, dst.Agent.KnowledgeEnabled)
	dst.LLM.Provider = valuew.Coalesce(src.LLM.Provider, dst.LLM.Provider)
	dst.LLM.Model = valuew.Coalesce(src.LLM.Model, dst.LLM.Model)
	dst.LLM.EmbedModel = valuew.Coalesce(src.LLM.EmbedModel, dst.LLM.EmbedModel)
	dst.LLM.BaseURL = valuew.Coalesce(src.LLM.BaseURL, dst.LLM.BaseURL)
	dst.LLM.EmbedBaseURL = valuew.Coalesce(src.LLM.EmbedBaseURL, dst.LLM.EmbedBaseURL)
	dst.LLM.APIKey = valuew.Coalesce(src.LLM.APIKey, dst.LLM.APIKey)
	dst.LLM.Temperature = valuew.Coalesce(src.LLM.Temperature, dst.LLM.Temperature)
	dst.LLM.MaxTokens = valuew.Coalesce(src.LLM.MaxTokens, dst.LLM.MaxTokens)
	dst.MinIO.Endpoint = valuew.Coalesce(src.MinIO.Endpoint, dst.MinIO.Endpoint)
	dst.MinIO.AccessKey = valuew.Coalesce(src.MinIO.AccessKey, dst.MinIO.AccessKey)
	dst.MinIO.SecretKey = valuew.Coalesce(src.MinIO.SecretKey, dst.MinIO.SecretKey)
	dst.MinIO.UseSSL = valuew.Coalesce(src.MinIO.UseSSL, dst.MinIO.UseSSL)
	dst.MinIO.Region = valuew.Coalesce(src.MinIO.Region, dst.MinIO.Region)
	dst.Knowledge.BaseURL = valuew.Coalesce(src.Knowledge.BaseURL, dst.Knowledge.BaseURL)
	dst.MCPServer.BaseURL = valuew.Coalesce(src.MCPServer.BaseURL, dst.MCPServer.BaseURL)
	dst.MCPServer.APIKey = valuew.Coalesce(src.MCPServer.APIKey, dst.MCPServer.APIKey)
	dst.WAHA.BaseURL = valuew.Coalesce(src.WAHA.BaseURL, dst.WAHA.BaseURL)
	dst.WAHA.APIKey = valuew.Coalesce(src.WAHA.APIKey, dst.WAHA.APIKey)
	dst.WAHA.Session = valuew.Coalesce(src.WAHA.Session, dst.WAHA.Session)
	dst.Whitelist.DefaultTokensPerHour = valuew.Coalesce(src.Whitelist.DefaultTokensPerHour, dst.Whitelist.DefaultTokensPerHour)
	dst.Graceful.ShutdownTimeout = valuew.Coalesce(src.Graceful.ShutdownTimeout, dst.Graceful.ShutdownTimeout)
	dst.HTTP.APIKey = valuew.Coalesce(src.HTTP.APIKey, dst.HTTP.APIKey)
	dst.WAHA.AdminChatID = valuew.Coalesce(src.WAHA.AdminChatID, dst.WAHA.AdminChatID)
	dst.Task.WorkerCount = valuew.Coalesce(src.Task.WorkerCount, dst.Task.WorkerCount)
	dst.Task.ChannelSize = valuew.Coalesce(src.Task.ChannelSize, dst.Task.ChannelSize)
	dst.Task.WorkerTimeout = valuew.Coalesce(src.Task.WorkerTimeout, dst.Task.WorkerTimeout)
	dst.Task.DefaultMaxRetry = valuew.Coalesce(src.Task.DefaultMaxRetry, dst.Task.DefaultMaxRetry)
	dst.Task.DefaultRetryDelay = valuew.Coalesce(src.Task.DefaultRetryDelay, dst.Task.DefaultRetryDelay)
	dst.NSQ.NSQdAddr = valuew.Coalesce(src.NSQ.NSQdAddr, dst.NSQ.NSQdAddr)
	dst.NSQ.Channel = valuew.Coalesce(src.NSQ.Channel, dst.NSQ.Channel)
}

func applyDefaults(cfg *AppConfig) {
	cfg.HTTP.Engine = valuew.Coalesce(cfg.HTTP.Engine, "echo")
	cfg.App.HTTPPort = valuew.Coalesce(cfg.App.HTTPPort, 8080)
	cfg.DB.Driver = valuew.Coalesce(cfg.DB.Driver, "gorm")
	cfg.DB.Dialect = valuew.Coalesce(cfg.DB.Dialect, "postgres")
	cfg.Agent.MaxSteps = valuew.Coalesce(cfg.Agent.MaxSteps, 25)
	cfg.Agent.ToolLimit = valuew.Coalesce(cfg.Agent.ToolLimit, 15)
	cfg.Agent.KnowledgeLimit = valuew.Coalesce(cfg.Agent.KnowledgeLimit, 5)
	cfg.Agent.ToolMinScore = valuew.Coalesce(cfg.Agent.ToolMinScore, 0.3)
	cfg.Agent.KnowledgeMinScore = valuew.Coalesce(cfg.Agent.KnowledgeMinScore, 0.4)
	cfg.Agent.MaxToolContextTokens = valuew.Coalesce(cfg.Agent.MaxToolContextTokens, 4000)
	cfg.Agent.MaxKnowledgeContextTokens = valuew.Coalesce(cfg.Agent.MaxKnowledgeContextTokens, 2000)
	cfg.Agent.ToolsEnabled = valuew.Coalesce(cfg.Agent.ToolsEnabled, true)
	cfg.Agent.KnowledgeEnabled = valuew.Coalesce(cfg.Agent.KnowledgeEnabled, false)
	cfg.LLM.Temperature = valuew.Coalesce(cfg.LLM.Temperature, 0.7)
	cfg.LLM.MaxTokens = valuew.Coalesce(cfg.LLM.MaxTokens, 4096)
	cfg.Graceful.ShutdownTimeout = valuew.Coalesce(cfg.Graceful.ShutdownTimeout, 10*time.Second)
	cfg.Whitelist.DefaultTokensPerHour = valuew.Coalesce(cfg.Whitelist.DefaultTokensPerHour, 100)
	cfg.Task.WorkerCount = valuew.Coalesce(cfg.Task.WorkerCount, 5)
	cfg.Task.ChannelSize = valuew.Coalesce(cfg.Task.ChannelSize, 1000)
	cfg.Task.WorkerTimeout = valuew.Coalesce(cfg.Task.WorkerTimeout, 5*time.Minute)
	cfg.Task.DefaultMaxRetry = valuew.Coalesce(cfg.Task.DefaultMaxRetry, 3)
	cfg.Task.DefaultRetryDelay = valuew.Coalesce(cfg.Task.DefaultRetryDelay, 30*time.Second)
	cfg.MessageStore.TTL = valuew.Coalesce(cfg.MessageStore.TTL, 24*time.Hour)
	cfg.NSQ.Channel = valuew.Coalesce(cfg.NSQ.Channel, "zora-worker")

	// If embed_base_url is not set, derive it from base_url by stripping /v1 suffix.
	// Ollama's native /api/embed endpoint doesn't use /v1, but the OpenAI-compatible
	// chat API (used by the LLM) requires /v1.
	if cfg.LLM.EmbedBaseURL == "" && cfg.LLM.BaseURL != "" {
		cfg.LLM.EmbedBaseURL = strings.TrimSuffix(cfg.LLM.BaseURL, "/v1")
	}
}