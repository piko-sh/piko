package content

import (
	"fmt"
	"strings"
	"time"

	"piko.sh/piko"
	"piko.sh/piko/wdk/storage"
)

// StoreAction stores content text in S3 via the storage service.
type StoreAction struct {
	piko.ActionMetadata
}

// StoreOutput is the response after a successful store operation.
type StoreOutput struct {
	Success    bool   `json:"success"`
	StorageKey string `json:"storage_key"`
	Size       int    `json:"size"`
}

// Call receives content text and stores it in S3 as a text file.
func (a StoreAction) Call(contentText string) (StoreOutput, error) {
	ctx := a.Ctx()

	service, err := storage.GetDefaultService()
	if err != nil {
		return StoreOutput{}, fmt.Errorf("getting storage service: %w", err)
	}

	key := fmt.Sprintf("content/%d.txt", time.Now().UnixNano())
	reader := strings.NewReader(contentText)
	contentSize := len(contentText)

	builder, err := storage.NewUploadBuilder(service, reader)
	if err != nil {
		return StoreOutput{}, fmt.Errorf("creating upload builder: %w", err)
	}

	err = builder.
		Provider("s3").
		Key(key).
		Repository("content").
		ContentType("text/plain").
		Size(int64(contentSize)).
		Do(ctx)
	if err != nil {
		return StoreOutput{}, fmt.Errorf("storing content in S3: %w", err)
	}

	// Keep a short preview for the in-memory list.
	preview := contentText
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}

	store.Add(ContentRecord{
		Key:     key,
		Preview: preview,
		Size:    contentSize,
	})

	return StoreOutput{
		Success:    true,
		StorageKey: key,
		Size:       contentSize,
	}, nil
}
