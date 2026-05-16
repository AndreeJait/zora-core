package entity

import (
	"encoding/json"
	"time"
)

// AdminEntry represents an admin user who can manage the whitelist and bot.
type AdminEntry struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Phone     string    `json:"phone" gorm:"uniqueIndex;not null"`
	LID       string    `json:"lid" gorm:"index"`                           // WhatsApp LID for group fallback lookup
	Name      string    `json:"name" gorm:"not null"`
	Scope     string    `json:"scope" gorm:"not null;default:'both'"`       // "personal", "group", "both"
	ChatIDs   string    `json:"chat_ids" gorm:"type:text"`                  // JSON array of group IDs; empty = no restriction
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// GetChatIDs parses the ChatIDs JSON array. Returns nil if empty or no restriction.
func (e AdminEntry) GetChatIDs() []string {
	if e.ChatIDs == "" || e.ChatIDs == "[]" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(e.ChatIDs), &ids); err != nil {
		return nil
	}
	return ids
}

// TableName overrides the default table name.
func (AdminEntry) TableName() string {
	return "admin_entries"
}