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

package generator_domain

import (
	"context"
	"os"
	"sync/atomic"
)

// MockFSWriter is a test double for FSWriterPort where nil function
// fields return zero values and call counts are tracked atomically.
type MockFSWriter struct {
	// WriteFileFunc is the function called by WriteFile.
	WriteFileFunc func(ctx context.Context, filePath string, data []byte) error

	// ReadDirFunc is the function called by ReadDir.
	ReadDirFunc func(dirname string) ([]os.DirEntry, error)

	// RemoveAllFunc is the function called by RemoveAll.
	RemoveAllFunc func(path string) error

	// WriteFileCallCount tracks how many times WriteFile
	// was called.
	WriteFileCallCount int64

	// ReadDirCallCount tracks how many times ReadDir
	// was called.
	ReadDirCallCount int64

	// RemoveAllCallCount tracks how many times RemoveAll
	// was called.
	RemoveAllCallCount int64
}

var _ FSWriterPort = (*MockFSWriter)(nil)

// WriteFile delegates to WriteFileFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes filePath (string) which is the destination file path.
// Takes data ([]byte) which is the content to write.
//
// Returns nil if WriteFileFunc is nil.
func (m *MockFSWriter) WriteFile(ctx context.Context, filePath string, data []byte) error {
	atomic.AddInt64(&m.WriteFileCallCount, 1)
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(ctx, filePath, data)
	}
	return nil
}

// ReadDir delegates to ReadDirFunc if set.
//
// Takes dirname (string) which is the directory path to read.
//
// Returns (nil, nil) if ReadDirFunc is nil.
func (m *MockFSWriter) ReadDir(dirname string) ([]os.DirEntry, error) {
	atomic.AddInt64(&m.ReadDirCallCount, 1)
	if m.ReadDirFunc != nil {
		return m.ReadDirFunc(dirname)
	}
	return nil, nil
}

// RemoveAll delegates to RemoveAllFunc if set.
//
// Takes path (string) which is the file or directory path to remove.
//
// Returns nil if RemoveAllFunc is nil.
func (m *MockFSWriter) RemoveAll(path string) error {
	atomic.AddInt64(&m.RemoveAllCallCount, 1)
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(path)
	}
	return nil
}
