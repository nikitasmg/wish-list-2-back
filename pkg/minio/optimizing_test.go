package minio_test

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	minioPkg "main/pkg/minio"
	mockminio "main/mock/minio"
)

func makeJPEGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

func isWebP(data []byte) bool {
	return len(data) >= 12 &&
		string(data[:4]) == "RIFF" &&
		string(data[8:12]) == "WEBP"
}

func TestOptimizingStorage_Upload_ConvertsJPEGToWebP(t *testing.T) {
	inner := new(mockminio.MockFileStorage)
	inner.On("Upload", mock.AnythingOfType("string"), mock.MatchedBy(isWebP)).
		Return("http://files.example.com/img", nil)

	storage := minioPkg.NewOptimizing(inner)

	url, err := storage.Upload("photo.jpg", makeJPEGBytes(t))

	require.NoError(t, err)
	assert.Equal(t, "http://files.example.com/img", url)
	inner.AssertExpectations(t)
}

func TestOptimizingStorage_Upload_PassthroughNonImage(t *testing.T) {
	original := []byte("not an image")

	inner := new(mockminio.MockFileStorage)
	inner.On("Upload", "file.bin", original).Return("http://files.example.com/file", nil)

	storage := minioPkg.NewOptimizing(inner)

	url, err := storage.Upload("file.bin", original)

	require.NoError(t, err)
	assert.Equal(t, "http://files.example.com/file", url)
	inner.AssertExpectations(t)
}

func TestOptimizingStorage_Delete_Delegates(t *testing.T) {
	inner := new(mockminio.MockFileStorage)
	inner.On("Delete", "some-object-id").Return(nil)

	storage := minioPkg.NewOptimizing(inner)

	err := storage.Delete("some-object-id")

	require.NoError(t, err)
	inner.AssertExpectations(t)
}
