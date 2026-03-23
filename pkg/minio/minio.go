package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"main/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStorage interface {
	Upload(name string, data []byte) (string, error)
	Delete(objectID string) error
}

type minioStorage struct {
	mc         *minio.Client
	bucketName string
	publicURL  string
}

func New(cfg config.MinioConfig, publicURL string) (FileStorage, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.RootUser, cfg.RootPassword, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio connection: %w", err)
	}

	ctx := context.Background()

	exists, err := mc.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}

	if !exists {
		if err := mc.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}

		policy := `{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + cfg.BucketName + `/*"
				}
			]
		}`

		if err := mc.SetBucketPolicy(ctx, cfg.BucketName, policy); err != nil {
			return nil, fmt.Errorf("minio set policy: %w", err)
		}
	}

	return &minioStorage{
		mc:         mc,
		bucketName: cfg.BucketName,
		publicURL:  publicURL,
	}, nil
}

// detectContentType identifies the MIME type of data.
// Falls back to http.DetectContentType for non-WebP content.
func detectContentType(data []byte) string {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(data)
}

func (s *minioStorage) Upload(name string, data []byte) (string, error) {
	if s.mc == nil {
		return "", errors.New("minio client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID := uuid.New().String()

	reader := bytes.NewReader(data)
	_, err := s.mc.PutObject(ctx, s.bucketName, objectID, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: detectContentType(data),
	})
	if err != nil {
		return "", fmt.Errorf("minio upload %s: %w", name, err)
	}

	url := fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucketName, objectID)
	return url, nil
}

func (s *minioStorage) Delete(objectID string) error {
	return s.mc.RemoveObject(context.Background(), s.bucketName, objectID, minio.RemoveObjectOptions{})
}
