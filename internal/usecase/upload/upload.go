package upload

import (
	"context"
	"fmt"

	"main/internal/usecase"
	"main/pkg/imagefile"
	minioPkg "main/pkg/minio"
)

type uploadUseCase struct {
	fileStorage minioPkg.FileStorage
}

func New(fileStorage minioPkg.FileStorage) usecase.UploadUseCase {
	return &uploadUseCase{fileStorage: fileStorage}
}

func (uc *uploadUseCase) Upload(ctx context.Context, name string, data []byte) (usecase.UploadResult, error) {
	if err := imagefile.Validate(data); err != nil {
		return usecase.UploadResult{}, err
	}

	url, err := uc.fileStorage.Upload(name, data)
	if err != nil {
		return usecase.UploadResult{}, fmt.Errorf("upload: %w", err)
	}
	return usecase.UploadResult{URL: url}, nil
}

func (uc *uploadUseCase) BulkUpload(ctx context.Context, files []usecase.FileInput) ([]usecase.BulkUploadResult, error) {
	// Проверяем всё до первой загрузки: иначе половина пачки уже лежит в MinIO,
	// а пользователь видит ошибку и жмёт «загрузить» ещё раз.
	for _, f := range files {
		if err := imagefile.Validate(f.Data); err != nil {
			return nil, fmt.Errorf("файл %q: %w", f.Name, err)
		}
	}

	results := make([]usecase.BulkUploadResult, 0, len(files))
	for _, f := range files {
		url, err := uc.fileStorage.Upload(f.Name, f.Data)
		if err != nil {
			return nil, fmt.Errorf("bulk upload file[%d] %q: %w", f.Index, f.Name, err)
		}
		results = append(results, usecase.BulkUploadResult{Index: f.Index, URL: url})
	}
	return results, nil
}
