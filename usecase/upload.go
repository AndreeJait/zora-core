package usecase

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/port/inbound/upload"
	"github.com/AndreeJait/zora-core/port/outbound"
	"github.com/google/uuid"
)

type uploadUseCase struct {
	storage outbound.Storage
}

var _ upload.UseCase = (*uploadUseCase)(nil)

func NewUploadUseCase(storage outbound.Storage) upload.UseCase {
	return &uploadUseCase{storage: storage}
}

func (uc *uploadUseCase) Upload(ctx context.Context, bucket string, prefix string, filename string, contentType string, data []byte) (*upload.UploadResult, error) {
	if bucket == "" {
		bucket = "zora-files"
	}
	if prefix == "" {
		prefix = "uploads"
	}

	objectKey := fmt.Sprintf("%s/%s-%s", prefix, uuid.New().String(), filepath.Base(filename))
	size := int64(len(data))
	reader := bytes.NewReader(data)

	if err := uc.storage.Upload(ctx, bucket, objectKey, reader, size, contentType); err != nil {
		logw.CtxErrorf(ctx, "upload: failed to upload %s to %s: %v", objectKey, bucket, err)
		return nil, fmt.Errorf("upload file: %w", err)
	}

	url, err := uc.storage.GetPresignedURL(ctx, bucket, objectKey, 24*time.Hour)
	if err != nil {
		url = fmt.Sprintf("%s/%s", bucket, objectKey)
	}

	logw.CtxInfof(ctx, "upload: stored %s (%d bytes) in bucket %s", objectKey, size, bucket)

	return &upload.UploadResult{
		Bucket:      bucket,
		ObjectKey:   objectKey,
		URL:         url,
		ContentType: contentType,
		Size:        size,
	}, nil
}