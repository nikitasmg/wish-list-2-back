package minio

import "main/pkg/imageconv"

type optimizingStorage struct {
	inner FileStorage
}

// NewOptimizing returns a FileStorage decorator that converts images to WebP
// before delegating to the inner storage.
func NewOptimizing(inner FileStorage) FileStorage {
	return &optimizingStorage{inner: inner}
}

func (s *optimizingStorage) Upload(name string, data []byte) (string, error) {
	return s.inner.Upload(name, imageconv.Convert(data))
}

func (s *optimizingStorage) Delete(objectID string) error {
	return s.inner.Delete(objectID)
}
