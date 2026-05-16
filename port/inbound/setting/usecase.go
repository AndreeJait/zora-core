package setting

import "context"

// UseCase defines the inbound port for runtime settings management.
type UseCase interface {
	// Get returns the value of a setting by key. Returns empty string if not found.
	Get(ctx context.Context, key string) (string, error)
	// GetAll returns all settings as a key-value map.
	GetAll(ctx context.Context) (map[string]string, error)
	// Set creates or updates a setting.
	Set(ctx context.Context, key, value, description string) error
}