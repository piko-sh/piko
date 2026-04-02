package files

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/storage"
)

var log = logger.GetLogger("files")

// RequestUploadAction generates a presigned PUT URL for client-side S3 upload.
type RequestUploadAction struct {
	piko.ActionMetadata
}

// RequestUploadInput specifies the file to upload.
type RequestUploadInput struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
}

// RequestUploadOutput contains the presigned URL and temporary key.
type RequestUploadOutput struct {
	UploadURL string `json:"upload_url"`
	TempKey   string `json:"temp_key"`
}

// Call generates a presigned PUT URL for the client to upload directly to S3.
func (a RequestUploadAction) Call(input RequestUploadInput) (RequestUploadOutput, error) {
	ctx := a.Ctx()
	service, err := storage.GetDefaultService()
	if err != nil {
		return RequestUploadOutput{}, fmt.Errorf("getting storage service: %w", err)
	}

	tempKey := fmt.Sprintf("tmp/%s-%s", uuid.New().String(), input.FileName)

	url, err := service.GeneratePresignedUploadURL(ctx, "s3", storage.PresignParams{
		Repository:  "uploads",
		Key:         tempKey,
		ContentType: input.ContentType,
		ExpiresIn:   15 * time.Minute,
	})
	if err != nil {
		return RequestUploadOutput{}, fmt.Errorf("generating presigned URL: %w", err)
	}

	return RequestUploadOutput{
		UploadURL: url,
		TempKey:   tempKey,
	}, nil
}
