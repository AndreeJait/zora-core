package echo

import (
	"fmt"

	httpw "github.com/AndreeJait/go-utility/v2/httpw/echow"
	"github.com/AndreeJait/go-utility/v2/responsew"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/webhook"
	"github.com/labstack/echo/v5"
)

// RegisterWebhookRoutes registers the WAHA webhook route.
//
// @Summary      Handle WAHA webhook
// @Description  Process incoming WhatsApp messages from the WAHA webhook. Requires API key via X-API-Key header.
// @Tags         webhook
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        body  body  entity.WAHAWebhookEvent  true  "Webhook event payload"
// @Success      200  {object}  responsew.BaseResponse
// @Failure      400  {object}  responsew.BaseResponse
// @Router       /webhook [post]
func RegisterWebhookRoutes(r RouteRegistrar, webhookUC webhook.UseCase) {
	r.POST("/webhook", httpw.Bind(handleWebhook(webhookUC)))
}

func handleWebhook(webhookUC webhook.UseCase) func(c *echo.Context) (any, error) {
	return func(c *echo.Context) (any, error) {
		var event entity.WAHAWebhookEvent
		if err := c.Bind(&event); err != nil {
			return nil, fmt.Errorf("failed to bind webhook event: %w", err)
		}
		if err := webhookUC.HandleIncomingMessage(c.Request().Context(), &event); err != nil {
			return nil, err
		}
		return responsew.Success(nil, "Webhook processed"), nil
	}
}
