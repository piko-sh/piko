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

package lifecycle_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
)

// MockFileSystemWatcher is a test double for FileSystemWatcher that
// returns zero values from nil function fields and tracks call counts
// atomically.
type MockFileSystemWatcher struct {
	// WatchFunc is the function called by Watch.
	WatchFunc func(ctx context.Context, recursiveDirs []string, nonRecursiveDirs []string) (<-chan lifecycle_dto.FileEvent, error)

	// UpdateWatchedFilesFunc is the function called by
	// UpdateWatchedFiles.
	UpdateWatchedFilesFunc func(ctx context.Context, files []string) error

	// CloseFunc is the function called by Close.
	CloseFunc func() error

	// WatchCallCount tracks how many times Watch was called.
	WatchCallCount int64

	// UpdateWatchedFilesCallCount tracks how many times
	// UpdateWatchedFiles was called.
	UpdateWatchedFilesCallCount int64

	// CloseCallCount tracks how many times Close was called.
	CloseCallCount int64
}

var _ FileSystemWatcher = (*MockFileSystemWatcher)(nil)

// Watch begins watching the specified directories for file changes.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes recursiveDirs ([]string) which lists directories to watch recursively.
// Takes nonRecursiveDirs ([]string) which lists directories to
// watch non-recursively.
//
// Returns (<-chan FileEvent, error), or (nil, nil) if WatchFunc is nil.
func (m *MockFileSystemWatcher) Watch(ctx context.Context, recursiveDirs []string, nonRecursiveDirs []string) (<-chan lifecycle_dto.FileEvent, error) {
	atomic.AddInt64(&m.WatchCallCount, 1)
	if m.WatchFunc != nil {
		return m.WatchFunc(ctx, recursiveDirs, nonRecursiveDirs)
	}
	return nil, nil
}

// UpdateWatchedFiles adds or removes files from the watch list.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes files ([]string) which lists the file paths to watch.
//
// Returns error, or nil if UpdateWatchedFilesFunc is nil.
func (m *MockFileSystemWatcher) UpdateWatchedFiles(ctx context.Context, files []string) error {
	atomic.AddInt64(&m.UpdateWatchedFilesCallCount, 1)
	if m.UpdateWatchedFilesFunc != nil {
		return m.UpdateWatchedFilesFunc(ctx, files)
	}
	return nil
}

// Close stops all file watching and releases resources.
//
// Returns error, or nil if CloseFunc is nil.
func (m *MockFileSystemWatcher) Close() error {
	atomic.AddInt64(&m.CloseCallCount, 1)
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
