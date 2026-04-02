// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package registry_adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/registry/registry_domain"
)

var _ registry_domain.BlobStore = (*MockBlobStore)(nil)

// MockBlobStore is a thread-safe, in-memory implementation of BlobStore.
// It is used for testing and does not save data to disk.
type MockBlobStore struct {
	// blobs stores blob data as byte slices keyed by their string identifier.
	blobs map[string][]byte

	// mu guards concurrent access to the blobs map.
	mu sync.RWMutex
}

// NewMockBlobStore creates a new in-memory blob store for testing.
//
// Returns *MockBlobStore which is ready for use in test scenarios.
func NewMockBlobStore() *MockBlobStore {
	return &MockBlobStore{
		blobs: make(map[string][]byte),
		mu:    sync.RWMutex{},
	}
}

// Name returns the display name of this blob store.
//
// Returns string which is the human-readable name for this blob store.
func (*MockBlobStore) Name() string {
	return "BlobStore (Mock)"
}

// Check implements the healthprobe_domain.Probe interface.
// The mock blob store is always healthy as it operates in-memory.
//
// Returns healthprobe_dto.Status which always reports healthy.
func (m *MockBlobStore) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	return healthprobe_dto.Status{
		Name:      m.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Mock blob store operational",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// Put stores blob data in the mock store.
//
// Takes key (string) which identifies the blob in the store.
// Takes data (io.Reader) which provides the blob content to store.
//
// Returns error when reading from the data reader fails.
//
// Safe for concurrent use.
func (m *MockBlobStore) Put(_ context.Context, key string, data io.Reader) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blobData, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read blob data: %w", err)
	}

	dataCopy := make([]byte, len(blobData))
	copy(dataCopy, blobData)
	m.blobs[key] = dataCopy

	return nil
}

// Get retrieves blob data from the mock store.
//
// Takes key (string) which identifies the blob to retrieve.
//
// Returns io.ReadCloser which provides access to a copy of the blob data.
// Returns error when the blob does not exist.
//
// Safe for concurrent use.
func (m *MockBlobStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blobData, exists := m.blobs[key]
	if !exists {
		return nil, fmt.Errorf("blob not found: %s", key)
	}

	dataCopy := make([]byte, len(blobData))
	copy(dataCopy, blobData)

	return io.NopCloser(bytes.NewReader(dataCopy)), nil
}

// RangeGet retrieves a range of bytes from a blob.
//
// Takes key (string) which identifies the blob to read from.
// Takes offset (int64) which specifies the starting byte position.
// Takes length (int64) which specifies the number of bytes to read.
//
// Returns io.ReadCloser which provides access to the requested byte range.
// Returns error when the blob is not found or the range is invalid.
//
// Safe for concurrent use.
func (m *MockBlobStore) RangeGet(_ context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	blobData, exists := m.blobs[key]
	if !exists {
		return nil, registry_domain.ErrBlobNotFound
	}

	if offset < 0 || length <= 0 {
		return nil, registry_domain.ErrRangeNotSatisfiable
	}

	blobSize := int64(len(blobData))
	if offset >= blobSize {
		return nil, registry_domain.ErrRangeNotSatisfiable
	}

	actualLength := length
	if offset+length > blobSize {
		actualLength = blobSize - offset
	}

	rangeData := make([]byte, actualLength)
	copy(rangeData, blobData[offset:offset+actualLength])

	return io.NopCloser(bytes.NewReader(rangeData)), nil
}

// Delete removes a blob from the mock store.
//
// Takes key (string) which identifies the blob to remove.
//
// Returns error when the blob does not exist.
//
// Safe for concurrent use.
func (m *MockBlobStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.blobs[key]; !exists {
		return fmt.Errorf("blob not found: %s", key)
	}

	delete(m.blobs, key)
	return nil
}

// Rename moves a blob from one key to another.
//
// Takes tempKey (string) which is the source key to rename from.
// Takes key (string) which is the destination key to rename to.
//
// Returns error when the source blob does not exist.
//
// Safe for concurrent use.
func (m *MockBlobStore) Rename(_ context.Context, tempKey string, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	blobData, exists := m.blobs[tempKey]
	if !exists {
		return fmt.Errorf("source blob not found: %s", tempKey)
	}

	m.blobs[key] = blobData
	delete(m.blobs, tempKey)

	return nil
}

// Exists checks if a blob exists in the mock store.
//
// Takes key (string) which identifies the blob to look up.
//
// Returns bool which indicates whether the blob exists.
// Returns error when the lookup fails.
//
// Safe for concurrent use.
func (m *MockBlobStore) Exists(_ context.Context, key string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.blobs[key]
	return exists, nil
}

// ListKeys returns all storage keys in the mock blob store.
//
// Returns []string which contains all keys currently in the store.
// Returns error which is always nil.
//
// Safe for concurrent use.
func (m *MockBlobStore) ListKeys(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.blobs))
	for key := range m.blobs {
		keys = append(keys, key)
	}
	return keys, nil
}

// GetBlobCount returns the number of blobs currently stored (for testing).
//
// Returns int which is the count of blobs in the store.
//
// Safe for concurrent use.
func (m *MockBlobStore) GetBlobCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.blobs)
}

// Clear removes all blobs from the store.
//
// Safe for concurrent use.
func (m *MockBlobStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blobs = make(map[string][]byte)
}
