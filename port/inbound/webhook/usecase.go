package webhook

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// UseCase defines the inbound port for processing WhatsApp webhook events.
type UseCase interface {
	HandleIncomingMessage(ctx context.Context, event *entity.WAHAWebhookEvent) error
}
