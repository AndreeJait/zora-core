package outbound

import (
	"context"
	"io"
	"time"
)

// Storage defines the outbound port for object storage operations.
type Storage interface {
	// Upload stores an object in the bucket with the given key and content type.
	Upload(ctx context.Context, bucket, objectKey string, reader io.Reader, objectSize int64, contentType string) error

	// GetObject retrieves an object's content from the object store.
	GetObject(ctx context.Context, bucket, objectKey string) ([]byte, error)

	// GetPresignedURL generates a time-limited download URL.
	GetPresignedURL(ctx context.Context, bucket, objectKey string, expiry time.Duration) (string, error)

	// Delete removes an object from the bucket.
	Delete(ctx context.Context, bucket, objectKey string) error
}