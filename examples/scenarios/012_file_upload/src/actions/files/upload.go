package files

import (
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/storage"
)

// UploadAction handles direct multipart file uploads to S3.
type UploadAction struct {
	piko.ActionMetadata
}

// UploadOutput is the response after a successful upload.
type UploadOutput struct {
	Success     bool   `json:"success"`
	FileName    string `json:"file_name"`
	StorageKey  string `json:"storage_key"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

// Call receives a file upload and stores it in S3 via the storage service.
func (a UploadAction) Call(file piko.FileUpload) (UploadOutput, error) {
	ctx := a.Ctx()
	reader, err := file.Open()
	if err != nil {
		return UploadOutput{}, fmt.Errorf("opening uploaded file: %w", err)
	}
	defer func() { _ = reader.Close() }()

	service, err := storage.GetDefaultService()
	if err != nil {
		return UploadOutput{}, fmt.Errorf("getting storage service: %w", err)
	}

	key := fmt.Sprintf("direct/%d-%s", time.Now().UnixNano(), file.Name)

	builder, err := storage.NewUploadBuilder(service, reader)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("creating upload builder: %w", err)
	}

	err = builder.
		Provider("s3").
		Key(key).
		Repository("uploads").
		ContentType(file.ContentType).
		Size(file.Size).
		Do(ctx)
	if err != nil {
		return UploadOutput{}, fmt.Errorf("uploading to S3: %w", err)
	}

	fileRegistry.Add(FileRecord{
		Key:         key,
		FileName:    file.Name,
		Size:        file.Size,
		ContentType: file.ContentType,
		Method:      "direct",
	})

	return UploadOutput{
		Success:     true,
		FileName:    file.Name,
		StorageKey:  key,
		Size:        file.Size,
		ContentType: file.ContentType,
	}, nil
}
