package files

import (
	"sync"
	"time"
)

// FileRecord represents a file that has been uploaded to S3 storage.
type FileRecord struct {
	Key         string `json:"key"`
	FileName    string `json:"file_name"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	UploadedAt  string `json:"uploaded_at"`
	Method      string `json:"method"` // "direct" or "presigned"
}

type fileStore struct {
	mu    sync.RWMutex
	files []FileRecord
}

var fileRegistry = &fileStore{}

// Add appends a file record to the registry.
func (s *fileStore) Add(r FileRecord) {
	if r.UploadedAt == "" {
		r.UploadedAt = time.Now().Format(time.RFC3339)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = append(s.files, r)
}

// All returns a copy of all file records.
func (s *fileStore) All() []FileRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FileRecord, len(s.files))
	copy(out, s.files)
	return out
}

// AllFiles returns a copy of all uploaded file records.
func AllFiles() []FileRecord {
	return fileRegistry.All()
}
