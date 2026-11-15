package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// New connects to an existing S3 bucket. Provisioning is an explicit opt-in for local development.
func New(cfg config.S3Config) (FileStorage, error) {
	if cfg.Endpoint == "" || cfg.BucketName == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, errors.New("s3: endpoint, bucket and access credentials are required")
	}
	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil || publicURL.Host == "" || (publicURL.Scheme != "https" && publicURL.Scheme != "http") ||
		publicURL.User != nil || publicURL.RawQuery != "" || publicURL.ForceQuery || publicURL.Fragment != "" {
		return nil, errors.New("s3: public bucket URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := mc.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("s3 bucket check: %w", err)
	}
	if !exists {
		if !cfg.AutoCreateBucket {
			return nil, errors.New("s3: bucket does not exist; create it and configure access in the storage provider console")
		}
		if err := mc.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("s3 create local bucket: %w", err)
		}
		policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"s3:GetObject","Resource":"arn:aws:s3:::%s/*"}]}`, cfg.BucketName)
		if err := mc.SetBucketPolicy(ctx, cfg.BucketName, policy); err != nil {
			return nil, fmt.Errorf("s3 set local bucket policy: %w", err)
		}
	}
	return &minioStorage{
		mc:         mc,
		bucketName: cfg.BucketName,
		publicURL:  strings.TrimRight(cfg.PublicURL, "/"),
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
		return "", errors.New("s3 client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	objectID := uuid.New().String()
	_, err := s.mc.PutObject(ctx, s.bucketName, objectID, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: detectContentType(data),
	})
	if err != nil {
		return "", fmt.Errorf("s3 upload %s: %w", name, err)
	}
	return s.publicURL + "/" + objectID, nil
}

func (s *minioStorage) Delete(objectID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.mc.RemoveObject(ctx, s.bucketName, objectID, minio.RemoveObjectOptions{})
}
