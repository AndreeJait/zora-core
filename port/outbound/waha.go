package outbound

import "context"

// WahaClient defines the outbound port for interacting with the WAHA API.
type WahaClient interface {
	// SendText sends a text message to the given chat ID.
	SendText(ctx context.Context, chatID, text string) error
	// SendTextReply sends a text message as a reply to a specific message.
	// Returns the sent message's ID for chain storage.
	SendTextReply(ctx context.Context, chatID, text, replyTo string) (string, error)
	// SendReaction sends an emoji reaction to a message. Empty emoji removes the reaction.
	SendReaction(ctx context.Context, chatID, messageID, emoji string) error
	// StartTyping starts the typing indicator in a chat.
	StartTyping(ctx context.Context, chatID string) error
	// StopTyping stops the typing indicator in a chat.
	StopTyping(ctx context.Context, chatID string) error
	// SendSeen marks messages as read in a chat.
	SendSeen(ctx context.Context, chatID string) error
	// ResolveLID resolves a WhatsApp LID to a phone number.
	// Returns the phone number (without @c.us suffix) or empty string if unresolvable.
	ResolveLID(ctx context.Context, lid string) (string, error)
}
