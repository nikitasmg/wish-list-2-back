package upload_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"main/internal/usecase"
	uploadUC "main/internal/usecase/upload"
	mockminio "main/mock/minio"
	"main/pkg/imagefile"
	"main/pkg/imagefile/imagefiletest"
)

func TestUpload_Success(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)
	data := imagefiletest.PNG(t)

	fs.On("Upload", "photo.png", data).Return("https://minio/photo.png", nil)

	result, err := uc.Upload(context.Background(), "photo.png", data)
	require.NoError(t, err)
	assert.Equal(t, "https://minio/photo.png", result.URL)
}

func TestUpload_RejectsNonImage(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)

	_, err := uc.Upload(context.Background(), "doc.pdf", []byte("%PDF-1.7 не картинка"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "формат не поддерживается")
	fs.AssertNotCalled(t, "Upload", "doc.pdf", mock.Anything)
}

func TestUpload_RejectsOversized(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)
	oversized := append(imagefiletest.PNG(t), make([]byte, imagefile.MaxSize)...)

	_, err := uc.Upload(context.Background(), "huge.png", oversized)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "10 МБ")
}

func TestBulkUpload_OneFileFails(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)
	ok, fail := imagefiletest.PNG(t), imagefiletest.PNG(t)

	fs.On("Upload", "ok.png", ok).Return("https://minio/ok.png", nil)
	fs.On("Upload", "fail.png", fail).Return("", errors.New("storage error"))

	files := []usecase.FileInput{
		{Index: 0, Name: "ok.png", Data: ok},
		{Index: 1, Name: "fail.png", Data: fail},
	}

	_, err := uc.BulkUpload(context.Background(), files)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1")
}

// Пачка проверяется до первой загрузки: иначе половина файлов уже в хранилище,
// а пользователь видит ошибку и жмёт «загрузить» ещё раз.
func TestBulkUpload_ValidatesBeforeUploadingAnything(t *testing.T) {
	fs := &mockminio.MockFileStorage{}
	uc := uploadUC.New(fs)

	files := []usecase.FileInput{
		{Index: 0, Name: "ok.png", Data: imagefiletest.PNG(t)},
		{Index: 1, Name: "doc.pdf", Data: []byte("%PDF-1.7 не картинка")},
	}

	_, err := uc.BulkUpload(context.Background(), files)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "doc.pdf")
	fs.AssertNotCalled(t, "Upload", "ok.png", mock.Anything)
}
