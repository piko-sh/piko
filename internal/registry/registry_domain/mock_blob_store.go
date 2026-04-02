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

package registry_domain

import (
	"context"
	"io"
	"sync/atomic"
)

// MockBlobStore is a test double for BlobStore where nil function
// fields return zero values and call counts are tracked atomically.
type MockBlobStore struct {
	// PutFunc is the function called by Put.
	PutFunc func(ctx context.Context, key string, data io.Reader) error

	// GetFunc is the function called by Get.
	GetFunc func(ctx context.Context, key string) (io.ReadCloser, error)

	// RangeGetFunc is the function called by RangeGet.
	RangeGetFunc func(ctx context.Context, key string, offset int64, length int64) (io.ReadCloser, error)

	// DeleteFunc is the function called by Delete.
	DeleteFunc func(ctx context.Context, key string) error

	// RenameFunc is the function called by Rename.
	RenameFunc func(ctx context.Context, tempKey string, key string) error

	// ExistsFunc is the function called by Exists.
	ExistsFunc func(ctx context.Context, key string) (bool, error)

	// ListKeysFunc is the function called by ListKeys.
	ListKeysFunc func(ctx context.Context) ([]string, error)

	// PutCallCount tracks how many times Put was
	// called.
	PutCallCount int64

	// GetCallCount tracks how many times Get was
	// called.
	GetCallCount int64

	// RangeGetCallCount tracks how many times RangeGet
	// was called.
	RangeGetCallCount int64

	// DeleteCallCount tracks how many times Delete was
	// called.
	DeleteCallCount int64

	// RenameCallCount tracks how many times Rename was
	// called.
	RenameCallCount int64

	// ExistsCallCount tracks how many times Exists was
	// called.
	ExistsCallCount int64

	// ListKeysCallCount tracks how many times ListKeys was
	// called.
	ListKeysCallCount int64
}

// Put writes blob data under the given key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the blob to write.
// Takes data (io.Reader) which provides the blob data to store.
//
// Returns error, or nil if PutFunc is nil.
func (m *MockBlobStore) Put(ctx context.Context, key string, data io.Reader) error {
	atomic.AddInt64(&m.PutCallCount, 1)
	if m.PutFunc != nil {
		return m.PutFunc(ctx, key, data)
	}
	return nil
}

// Get retrieves blob data by key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the blob to retrieve.
//
// Returns (io.ReadCloser, error), or (nil, nil) if GetFunc is nil.
func (m *MockBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	atomic.AddInt64(&m.GetCallCount, 1)
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, nil
}

// RangeGet retrieves a byte range of blob data.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the blob to read from.
// Takes offset (int64) which is the byte position to start reading from.
// Takes length (int64) which is the number of bytes to read.
//
// Returns (io.ReadCloser, error), or (nil, nil) if RangeGetFunc is nil.
func (m *MockBlobStore) RangeGet(ctx context.Context, key string, offset int64, length int64) (io.ReadCloser, error) {
	atomic.AddInt64(&m.RangeGetCallCount, 1)
	if m.RangeGetFunc != nil {
		return m.RangeGetFunc(ctx, key, offset, length)
	}
	return nil, nil
}

// Delete removes a blob by key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the blob to delete.
//
// Returns error, or nil if DeleteFunc is nil.
func (m *MockBlobStore) Delete(ctx context.Context, key string) error {
	atomic.AddInt64(&m.DeleteCallCount, 1)
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	return nil
}

// Rename moves a blob from tempKey to key.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes tempKey (string) which is the current storage key of the blob.
// Takes key (string) which is the new storage key for the blob.
//
// Returns error, or nil if RenameFunc is nil.
func (m *MockBlobStore) Rename(ctx context.Context, tempKey string, key string) error {
	atomic.AddInt64(&m.RenameCallCount, 1)
	if m.RenameFunc != nil {
		return m.RenameFunc(ctx, tempKey, key)
	}
	return nil
}

// Exists checks whether a blob exists.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes key (string) which identifies the blob to check.
//
// Returns (bool, error), or (false, nil) if ExistsFunc is nil.
func (m *MockBlobStore) Exists(ctx context.Context, key string) (bool, error) {
	atomic.AddInt64(&m.ExistsCallCount, 1)
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, key)
	}
	return false, nil
}

// ListKeys returns all storage keys in the blob store.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
//
// Returns ([]string, error), or (nil, nil) if ListKeysFunc is nil.
func (m *MockBlobStore) ListKeys(ctx context.Context) ([]string, error) {
	atomic.AddInt64(&m.ListKeysCallCount, 1)
	if m.ListKeysFunc != nil {
		return m.ListKeysFunc(ctx)
	}
	return nil, nil
}
