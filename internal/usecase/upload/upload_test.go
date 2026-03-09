package upload_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"main/internal/usecase"
	uploadUC "main/internal/usecase/upload"
	mockminio "main/mock/minio"
)

func TestUpload_Success(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)

	fs.On("Upload", "photo.jpg", []byte("data")).Return("https://minio/photo.jpg", nil)

	result, err := uc.Upload(context.Background(), "photo.jpg", []byte("data"))
	require.NoError(t, err)
	assert.Equal(t, "https://minio/photo.jpg", result.URL)
}

func TestBulkUpload_OneFileFails(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)

	fs.On("Upload", "ok.jpg", []byte("ok")).Return("https://minio/ok.jpg", nil)
	fs.On("Upload", "fail.jpg", []byte("fail")).Return("", errors.New("storage error"))

	files := []usecase.FileInput{
		{Index: 0, Name: "ok.jpg", Data: []byte("ok")},
		{Index: 1, Name: "fail.jpg", Data: []byte("fail")},
	}

	_, err := uc.BulkUpload(context.Background(), files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1")
}
