package entity

import "time"

// Setting represents a runtime-configurable key-value setting.
type Setting struct {
	Key         string    `json:"key" gorm:"primaryKey;type:varchar(255)"`
	Value       string    `json:"value" gorm:"type:text;not null"`
	Description *string   `json:"description" gorm:"type:text"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName overrides the default table name.
func (Setting) TableName() string {
	return "zora_settings"
}