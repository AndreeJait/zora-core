package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/AndreeJait/go-utility/v2/storagew"
	"github.com/AndreeJait/go-utility/v2/storagew/miniow"
	"github.com/AndreeJait/zora-core/config"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
)

// minioStorage implements portOutbound.Storage using the go-utility storagew interface.
type minioStorage struct {
	inner storagew.Storage
}

var _ portOutbound.Storage = (*minioStorage)(nil)

func NewStorage(cfg *config.AppConfig) (portOutbound.Storage, error) {
	s, err := miniow.New(&miniow.Config{
		Endpoint:      cfg.MinIO.Endpoint,
		AccessKeyID:   cfg.MinIO.AccessKey,
		SecretAccessKey: cfg.MinIO.SecretKey,
		UseSSL:        cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio connect: %w", err)
	}
	return &minioStorage{inner: s}, nil
}

func (m *minioStorage) Upload(ctx context.Context, bucket, objectKey string, reader io.Reader, objectSize int64, contentType string) error {
	return m.inner.Upload(ctx, bucket, objectKey, reader, objectSize, contentType)
}

func (m *minioStorage) GetObject(ctx context.Context, bucket, objectKey string) ([]byte, error) {
	rc, err := m.inner.Download(ctx, bucket, objectKey)
	if err != nil {
		return nil, fmt.Errorf("download object: %w", err)
	}
	defer rc.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	return buf.Bytes(), nil
}

func (m *minioStorage) GetPresignedURL(ctx context.Context, bucket, objectKey string, expiry time.Duration) (string, error) {
	return m.inner.GetPresignedURL(ctx, bucket, objectKey, expiry)
}

func (m *minioStorage) Delete(ctx context.Context, bucket, objectKey string) error {
	return m.inner.Delete(ctx, bucket, objectKey)
}