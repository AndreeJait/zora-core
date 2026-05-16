package echo

import (
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/agent"
	"github.com/AndreeJait/zora-core/port/inbound/health"
	"github.com/AndreeJait/zora-core/port/inbound/setting"
	"github.com/AndreeJait/zora-core/port/inbound/task"
	"github.com/AndreeJait/zora-core/port/inbound/upload"
	"github.com/AndreeJait/zora-core/port/inbound/webhook"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	httpw "github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/labstack/echo/v5"
)

// RouteRegistrar is implemented by both *echo.Echo and *echo.Group,
// allowing route registration functions to accept either.
type RouteRegistrar interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) echo.RouteInfo
}

// Required for swagger annotations
var _ = entity.Health{}
var _ = agent.ExecuteInput{}
var _ = agent.ExecuteOutput{}
var _ = whitelist.UseCase(nil)

// RegisterRoutes registers all HTTP routes on the Echo engine.
func RegisterRoutes(e *echo.Echo, healthUC health.UseCase, agentUC agent.UseCase, webhookUC webhook.UseCase, uploadUC upload.UseCase, taskUC task.UseCase, settingUC setting.UseCase, whitelistUC whitelist.UseCase, apiKey string) {
	e.GET("/health", httpw.Bind(checkHealth(healthUC)))

	// Protected routes — API key middleware applied at group level
	mgmt := e.Group("", APIKeyMiddleware(apiKey))
	RegisterWebhookRoutes(mgmt, webhookUC)
	RegisterAgentRoutes(mgmt, agentUC)
	RegisterUploadRoutes(mgmt, uploadUC)
	RegisterTaskRoutes(mgmt, taskUC)
	RegisterSettingRoutes(mgmt, settingUC)
	RegisterWhitelistRoutes(mgmt, whitelistUC)
	RegisterAdminRoutes(mgmt, whitelistUC)
	RegisterGraphStateRoutes(mgmt, agentUC)
}

// checkHealth returns a handler for the health endpoint.
//
// @Summary      Health check
// @Description  Check if the service is healthy including DB and Redis connectivity
// @Tags         health
// @Success      200  {object}  responsew.BaseResponse{data=entity.Health}
// @Router       /health [get]
func checkHealth(healthUC health.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		health := healthUC.Check(c.Request().Context())
		return responsew.Success(health, "Service is healthy"), nil
	}
}