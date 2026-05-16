package entity

import (
	"encoding/json"
	"time"
)

// WhitelistEntry represents a whitelisted user who can access the bot.
type WhitelistEntry struct {
	ID            string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Phone         string    `json:"phone" gorm:"uniqueIndex;not null"`
	LID           string    `json:"lid" gorm:"column:lid;index"`                      // WhatsApp LID for group fallback lookup
	Name          string    `json:"name" gorm:"not null"`
	Scope         string    `json:"scope" gorm:"not null;default:'both'"`            // "personal", "group", "both"
	ChatIDs       string    `json:"chat_ids" gorm:"type:text"`                       // JSON array of group IDs; empty = no restriction
	TokensPerHour int       `json:"tokens_per_hour" gorm:"not null;default:0"` // 0 = unlimited
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// GetChatIDs parses the ChatIDs JSON array. Returns nil if empty or no restriction.
func (e WhitelistEntry) GetChatIDs() []string {
	if e.ChatIDs == "" || e.ChatIDs == "[]" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(e.ChatIDs), &ids); err != nil {
		return nil
	}
	return ids
}

// IsUnlimited returns true if the entry has no token limit.
func (e WhitelistEntry) IsUnlimited() bool {
	return e.TokensPerHour == 0
}

// TableName overrides the default table name.
func (WhitelistEntry) TableName() string {
	return "whitelist_entries"
}

// TokenUsage tracks hourly token consumption per user.
type TokenUsage struct {
	ID          string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Phone       string    `json:"phone" gorm:"not null"`
	TokensUsed  int       `json:"tokens_used" gorm:"not null;default:1"`
	WindowStart time.Time `json:"window_start" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName overrides the default table name.
func (TokenUsage) TableName() string {
	return "token_usages"
}