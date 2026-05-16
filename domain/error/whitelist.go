package error

import "github.com/AndreeJait/go-utility/v2/statusw"

var (
	ErrNotWhitelisted = statusw.InvalidAccess.WithCustomMessage("you are not whitelisted to use this bot")
	ErrTokenExhausted = statusw.InvalidAccess.WithCustomMessage("hourly token limit exceeded, try again later")
	ErrNotAdmin       = statusw.InvalidAccess.WithCustomMessage("only admins can manage the whitelist")
)