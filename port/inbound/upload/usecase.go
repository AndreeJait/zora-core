package upload

import "context"

// UploadResult holds the result of a file upload.
type UploadResult struct {
	Bucket      string `json:"bucket"`
	ObjectKey   string `json:"object_key"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// UseCase defines the inbound port for file uploads.
type UseCase interface {
	Upload(ctx context.Context, bucket string, prefix string, filename string, contentType string, data []byte) (*UploadResult, error)
}