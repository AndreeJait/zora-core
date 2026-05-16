package main

import (
	"fmt"
	"net/http"

	echoAdapter "github.com/AndreeJait/zora-core/adapter/inbound/echo"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/health"
	"github.com/AndreeJait/zora-core/port/inbound/setting"
	"github.com/AndreeJait/zora-core/port/inbound/task"
	"github.com/AndreeJait/zora-core/port/inbound/upload"
	"github.com/AndreeJait/zora-core/port/inbound/webhook"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	httpwEcho "github.com/AndreeJait/go-utility/v2/httpw/echow"
	"go.uber.org/dig"
)

// provideRouter registers the HTTP router provider into the dig container.
func provideRouter(c *dig.Container) {
	c.Provide(newRouter)
}

// newRouter selects the HTTP engine and registers all routes.
func newRouter(
	// kyan:param:start
	cfg *config.AppConfig,
	healthUC health.UseCase,
	agentUC agent.UseCase,
	webhookUC webhook.UseCase,
	uploadUC upload.UseCase,
	taskUC task.UseCase,
	settingUC setting.UseCase,
	whitelistUC whitelist.UseCase,
	// kyan:param:end
) (http.Handler, error) {
	switch cfg.HTTP.Engine {
	case "echo":
		e := httpwEcho.New(&httpwEcho.Config{
			DebugMode:     cfg.HTTP.DebugMode,
			EnableSwagger: cfg.HTTP.EnableSwagger,
		})
		echoAdapter.RegisterRoutes(e, healthUC, agentUC, webhookUC, uploadUC, taskUC, settingUC, whitelistUC, cfg.HTTP.APIKey)
		return e, nil
	default:
		return nil, fmt.Errorf("unknown engine: %s (must be echo)", cfg.HTTP.Engine)
	}
}