package usecase

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

const maxChainDepth = 10

// ChainResolver resolves WhatsApp reply chains by looking up messages in the MessageStore.
type ChainResolver struct {
	store outbound.MessageStore
}

// NewChainResolver creates a new ChainResolver.
func NewChainResolver(store outbound.MessageStore) *ChainResolver {
	return &ChainResolver{store: store}
}

// Resolve looks up the reply chain starting from the given message ID.
// Returns the chain in oldest-first order. Returns partial chain on error.
func (r *ChainResolver) Resolve(ctx context.Context, replyToID string) ([]entity.QuotedMessage, error) {
	var chain []entity.QuotedMessage
	currentID := replyToID

	for i := 0; i < maxChainDepth && currentID != ""; i++ {
		msg, err := r.store.Get(ctx, currentID)
		if err != nil {
			logw.CtxWarningf(ctx, "chain resolver: failed to get message %s: %v", currentID, err)
			return chain, err
		}
		if msg == nil {
			break // message not found (expired or never stored)
		}

		// Prepend to maintain oldest-first order
		chain = append([]entity.QuotedMessage{{
			SenderPhone: msg.SenderPhone,
			Body:        msg.Body,
			FromMe:      msg.FromMe,
		}}, chain...)

		currentID = msg.QuotedMsgID
	}

	return chain, nil
}