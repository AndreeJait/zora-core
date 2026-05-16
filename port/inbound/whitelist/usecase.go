package whitelist

import (
	"context"

	"github.com/AndreeJait/zora-core/domain/entity"
)

// SenderContext carries full identity and chat context for access checks.
type SenderContext struct {
	// SenderPhone is the resolved phone number (without @c.us), may be empty in group chats.
	SenderPhone string
	// SenderLID is the raw WhatsApp LID (e.g. "130605761167448@lid"), may be empty in private chats.
	SenderLID string
	// ChatID is the chat to reply to (phone@c.us for private, group@g.us for group).
	ChatID string
	// IsGroup is true if the message comes from a group chat.
	IsGroup bool
}

// UseCase defines the inbound port for whitelist management and access control.
type UseCase interface {
	// CheckAccess verifies that the sender is allowed to use the bot in the given context.
	// Returns nil if access is granted, or an error (ErrNotWhitelisted, ErrTokenExhausted) if denied.
	CheckAccess(ctx context.Context, sender SenderContext) error

	// ConsumeToken deducts one token from the user's hourly budget.
	// Should be called after CheckAccess succeeds and before running the agent.
	ConsumeToken(ctx context.Context, sender SenderContext) error

	// HandleCommand processes admin whitelist commands.
	// Returns a human-readable response string.
	HandleCommand(ctx context.Context, sender SenderContext, args string, mentionedIDs []string) (string, error)

	// IsAdmin checks if the sender is an admin in the given context.
	IsAdmin(sender SenderContext) bool

	// ListWhitelist returns all whitelist entries.
	ListWhitelist(ctx context.Context) ([]entity.WhitelistEntry, error)

	// AddWhitelist creates or updates a whitelist entry.
	AddWhitelist(ctx context.Context, phone, name string, tokensPerHour int, scope string, chatIDs []string) error

	// RemoveWhitelist deletes a whitelist entry by phone.
	RemoveWhitelist(ctx context.Context, phone string) error

	// ListAdmins returns all admin entries from the database.
	ListAdmins(ctx context.Context) ([]entity.AdminEntry, error)

	// AddAdmin creates or updates an admin entry in the database.
	AddAdmin(ctx context.Context, phone, name string, scope string, chatIDs []string) error

	// RemoveAdmin deletes an admin entry from the database by phone.
	RemoveAdmin(ctx context.Context, phone string) error
}