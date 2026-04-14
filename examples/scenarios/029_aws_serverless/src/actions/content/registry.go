package content

import (
	"sync"
	"time"
)

// ContentRecord represents a piece of content stored in S3.
type ContentRecord struct {
	Key       string `json:"key"`
	Preview   string `json:"preview"`
	Size      int    `json:"size"`
	StoredAt  string `json:"stored_at"`
}

type contentStore struct {
	mu      sync.RWMutex
	records []ContentRecord
}

var store = &contentStore{}

// Add appends a content record to the store.
func (s *contentStore) Add(record ContentRecord) {
	if record.StoredAt == "" {
		record.StoredAt = time.Now().Format(time.RFC3339)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
}

// All returns a copy of all content records.
func (s *contentStore) All() []ContentRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ContentRecord, len(s.records))
	copy(out, s.records)
	return out
}

// AllRecords returns a copy of all stored content records.
func AllRecords() []ContentRecord {
	return store.All()
}
