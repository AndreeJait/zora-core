package entity

import "time"

// Task status constants.
const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusCompleted = "completed"
	TaskStatusFailed    = "failed"
	TaskStatusRetrying  = "retrying"
	TaskStatusCancelled = "cancelled"
)

// Task source constants.
const (
	TaskSourceWaha = "waha"
	TaskSourceAPI  = "api"
)

// Task type constants.
const (
	TaskTypeWebhook = "webhook"
	TaskTypeAPI     = "api"
)

// Task represents a background task for agent execution.
type Task struct {
	ID           string         `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Type         string         `json:"type" gorm:"type:varchar(20);not null"`
	Status       string         `json:"status" gorm:"type:varchar(20);not null;default:pending"`
	Source       string         `json:"source" gorm:"type:varchar(20);not null"`
	Input        map[string]any `json:"input" gorm:"serializer:json;default:'{}'"`
	Result       map[string]any `json:"result" gorm:"serializer:json;default:'{}'"`
	Error        *string        `json:"error" gorm:"type:text"`
	RetryCount   int            `json:"retry_count" gorm:"not null;default:0"`
	MaxRetry     int            `json:"max_retry" gorm:"not null;default:3"`
	ChatID       *string        `json:"chat_id" gorm:"index"`
	MessageID    *string        `json:"message_id"`
	SessionID    *string        `json:"session_id" gorm:"index"`
	ThreadID     *string        `json:"thread_id"`
	GraphMermaid *string        `json:"graph_mermaid" gorm:"type:text"`
	NextRetryAt  *time.Time     `json:"next_retry_at" gorm:"index"`
	CreatedAt    time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName overrides the default table name.
func (Task) TableName() string {
	return "zora_tasks"
}