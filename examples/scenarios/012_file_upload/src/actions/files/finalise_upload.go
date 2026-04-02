package files

import (
	"fmt"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/storage"
)

// FinaliseUploadAction verifies and finalises a presigned URL upload.
type FinaliseUploadAction struct {
	piko.ActionMetadata
}

// FinaliseUploadInput identifies the temp upload to finalise.
type FinaliseUploadInput struct {
	TempKey  string `json:"temp_key"`
	FileName string `json:"file_name"`
}

// FinaliseUploadOutput confirms the finalised upload.
type FinaliseUploadOutput struct {
	Success    bool   `json:"success"`
	StorageKey string `json:"storage_key"`
	FileName   string `json:"file_name"`
	Size       int64  `json:"size"`
}

// Call verifies the temp file exists in S3, copies it to a final location,
// removes the temp file, and registers it in the file list.
func (a FinaliseUploadAction) Call(input FinaliseUploadInput) (FinaliseUploadOutput, error) {
	ctx := a.Ctx()
	service, err := storage.GetDefaultService()
	if err != nil {
		return FinaliseUploadOutput{}, fmt.Errorf("getting storage service: %w", err)
	}

	// Verify the temp file exists.
	statReq, err := storage.NewRequestBuilder(service, "uploads", input.TempKey)
	if err != nil {
		return FinaliseUploadOutput{}, fmt.Errorf("creating request builder: %w", err)
	}
	info, err := statReq.Provider("s3").Stat(ctx)
	if err != nil {
		return FinaliseUploadOutput{}, fmt.Errorf("temp file not found in S3: %w", err)
	}

	// Copy to final location.
	finalKey := fmt.Sprintf("presigned/%d-%s", time.Now().UnixNano(), input.FileName)
	err = service.CopyObject(ctx, "s3", storage.CopyParams{
		SourceRepository:      "uploads",
		SourceKey:             input.TempKey,
		DestinationRepository: "uploads",
		DestinationKey:        finalKey,
	})
	if err != nil {
		return FinaliseUploadOutput{}, fmt.Errorf("copying to final location: %w", err)
	}

	// Remove the temp file.
	if removeReq, err := storage.NewRequestBuilder(service, "uploads", input.TempKey); err == nil {
		_ = removeReq.Provider("s3").Remove(ctx)
	}

	fileRegistry.Add(FileRecord{
		Key:         finalKey,
		FileName:    input.FileName,
		Size:        info.Size,
		ContentType: info.ContentType,
		Method:      "presigned",
	})

	return FinaliseUploadOutput{
		Success:    true,
		StorageKey: finalKey,
		FileName:   input.FileName,
		Size:       info.Size,
	}, nil
}
