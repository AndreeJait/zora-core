package entity

import "time"

// Health represents the health status of the service.
type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

// NewHealth creates a Health entity indicating the service is healthy.
func NewHealth(serviceName string) *Health {
	return &Health{
		Status:  "healthy",
		Service: serviceName,
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
}