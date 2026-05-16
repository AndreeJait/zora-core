package entity

import (
	"time"

	"github.com/AndreeJait/go-utility/v2/llmw"
)

// Conversation represents a persisted agent conversation.
type Conversation struct {
	ID        string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SessionID string         `json:"session_id" gorm:"uniqueIndex;not null"`
	Task      string         `json:"task" gorm:"not null"`
	Messages  []llmw.Message `json:"messages" gorm:"serializer:json"`
	Status    string         `json:"status" gorm:"default:'active'"` // active, resolved, failed
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName overrides the default table name.
func (Conversation) TableName() string {
	return "conversations"
}
