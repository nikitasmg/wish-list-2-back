package upload

import (
	"context"
	"fmt"

	"main/internal/usecase"
	minioPkg "main/pkg/minio"
)

type uploadUseCase struct {
	fileStorage minioPkg.FileStorage
}

func New(fileStorage minioPkg.FileStorage) usecase.UploadUseCase {
	return &uploadUseCase{fileStorage: fileStorage}
}

func (uc *uploadUseCase) Upload(ctx context.Context, name string, data []byte) (usecase.UploadResult, error) {
	url, err := uc.fileStorage.Upload(name, data)
	if err != nil {
		return usecase.UploadResult{}, fmt.Errorf("upload: %w", err)
	}
	return usecase.UploadResult{URL: url}, nil
}

func (uc *uploadUseCase) BulkUpload(ctx context.Context, files []usecase.FileInput) ([]usecase.BulkUploadResult, error) {
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
