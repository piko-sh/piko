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
	"errors"
	"io/fs"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockFSWriter_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("nil WriteFileFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc:      nil,
			ReadDirFunc:        nil,
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		ctx := context.Background()
		err := mock.WriteFile(ctx, "/tmp/out.go", []byte("package main"))

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WriteFileCallCount))
	})

	t.Run("delegates to WriteFileFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		mock := &MockFSWriter{
			WriteFileFunc: func(_ context.Context, _ string, _ []byte) error {
				called = true
				return nil
			},
			ReadDirFunc:        nil,
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		ctx := context.Background()
		err := mock.WriteFile(ctx, "/output/generated.go", []byte("package gen"))

		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WriteFileCallCount))
	})

	t.Run("propagates error from WriteFileFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc: func(_ context.Context, _ string, _ []byte) error {
				return errors.New("disk full")
			},
			ReadDirFunc:        nil,
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		ctx := context.Background()
		err := mock.WriteFile(ctx, "/output/file.go", []byte("data"))

		require.Error(t, err)
		assert.Equal(t, "disk full", err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WriteFileCallCount))
	})
}

func TestMockFSWriter_WriteFile_PassesArguments(t *testing.T) {
	t.Parallel()

	var (
		capturedCtx  context.Context
		capturedPath string
		capturedData []byte
	)

	mock := &MockFSWriter{
		WriteFileFunc: func(ctx context.Context, filePath string, data []byte) error {
			capturedCtx = ctx
			capturedPath = filePath
			capturedData = data
			return nil
		},
		ReadDirFunc:        nil,
		RemoveAllFunc:      nil,
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "write-ctx")
	data := []byte("package generated\n\nfunc init() {}\n")

	err := mock.WriteFile(ctx, "/project/dist/output.go", data)

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, "/project/dist/output.go", capturedPath)
	assert.Equal(t, data, capturedData)
}

type fsDirEntry struct {
	name  string
	isDir bool
}

func (e *fsDirEntry) Name() string               { return e.name }
func (e *fsDirEntry) IsDir() bool                { return e.isDir }
func (e *fsDirEntry) Type() fs.FileMode          { return 0 }
func (e *fsDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

var _ os.DirEntry = (*fsDirEntry)(nil)

func TestMockFSWriter_ReadDir(t *testing.T) {
	t.Parallel()

	t.Run("nil ReadDirFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc:      nil,
			ReadDirFunc:        nil,
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		entries, err := mock.ReadDir("/some/directory")

		require.NoError(t, err)
		assert.Nil(t, entries)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReadDirCallCount))
	})

	t.Run("delegates to ReadDirFunc", func(t *testing.T) {
		t.Parallel()

		expected := []os.DirEntry{
			&fsDirEntry{name: "main.go", isDir: false},
			&fsDirEntry{name: "sub", isDir: true},
		}

		mock := &MockFSWriter{
			WriteFileFunc: nil,
			ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
				return expected, nil
			},
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		entries, err := mock.ReadDir("/project/src")

		require.NoError(t, err)
		require.Len(t, entries, 2)
		assert.Equal(t, "main.go", entries[0].Name())
		assert.Equal(t, "sub", entries[1].Name())
		assert.True(t, entries[1].IsDir())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReadDirCallCount))
	})

	t.Run("propagates error from ReadDirFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc: nil,
			ReadDirFunc: func(_ string) ([]os.DirEntry, error) {
				return nil, errors.New("permission denied")
			},
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		entries, err := mock.ReadDir("/restricted")

		require.Error(t, err)
		assert.Equal(t, "permission denied", err.Error())
		assert.Nil(t, entries)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReadDirCallCount))
	})
}

func TestMockFSWriter_ReadDir_PassesArguments(t *testing.T) {
	t.Parallel()

	var capturedDirname string

	mock := &MockFSWriter{
		WriteFileFunc: nil,
		ReadDirFunc: func(dirname string) ([]os.DirEntry, error) {
			capturedDirname = dirname
			return nil, nil
		},
		RemoveAllFunc:      nil,
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	_, err := mock.ReadDir("/project/dist/generated")

	require.NoError(t, err)
	assert.Equal(t, "/project/dist/generated", capturedDirname)
}

func TestMockFSWriter_RemoveAll(t *testing.T) {
	t.Parallel()

	t.Run("nil RemoveAllFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc:      nil,
			ReadDirFunc:        nil,
			RemoveAllFunc:      nil,
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		err := mock.RemoveAll("/tmp/old-output")

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveAllCallCount))
	})

	t.Run("delegates to RemoveAllFunc", func(t *testing.T) {
		t.Parallel()

		var called bool

		mock := &MockFSWriter{
			WriteFileFunc: nil,
			ReadDirFunc:   nil,
			RemoveAllFunc: func(_ string) error {
				called = true
				return nil
			},
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		err := mock.RemoveAll("/project/dist")

		require.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveAllCallCount))
	})

	t.Run("propagates error from RemoveAllFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockFSWriter{
			WriteFileFunc: nil,
			ReadDirFunc:   nil,
			RemoveAllFunc: func(_ string) error {
				return errors.New("directory in use")
			},
			WriteFileCallCount: 0,
			ReadDirCallCount:   0,
			RemoveAllCallCount: 0,
		}

		err := mock.RemoveAll("/locked/path")

		require.Error(t, err)
		assert.Equal(t, "directory in use", err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveAllCallCount))
	})
}

func TestMockFSWriter_RemoveAll_PassesArguments(t *testing.T) {
	t.Parallel()

	var capturedPath string

	mock := &MockFSWriter{
		WriteFileFunc: nil,
		ReadDirFunc:   nil,
		RemoveAllFunc: func(path string) error {
			capturedPath = path
			return nil
		},
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	err := mock.RemoveAll("/project/dist/stale-artefacts")

	require.NoError(t, err)
	assert.Equal(t, "/project/dist/stale-artefacts", capturedPath)
}

func TestMockFSWriter_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockFSWriter{
		WriteFileFunc:      nil,
		ReadDirFunc:        nil,
		RemoveAllFunc:      nil,
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	ctx := context.Background()

	_ = mock.WriteFile(ctx, "/a", nil)
	_ = mock.WriteFile(ctx, "/b", nil)
	_ = mock.WriteFile(ctx, "/c", nil)
	_, _ = mock.ReadDir("/d")
	_, _ = mock.ReadDir("/e")
	_ = mock.RemoveAll("/f")

	assert.Equal(t, int64(3), atomic.LoadInt64(&mock.WriteFileCallCount))
	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.ReadDirCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveAllCallCount))
}

func TestMockFSWriter_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockFSWriter

	ctx := context.Background()

	err := mock.WriteFile(ctx, "/tmp/zero.go", []byte("package zero"))
	require.NoError(t, err)

	entries, err := mock.ReadDir("/tmp")
	require.NoError(t, err)
	assert.Nil(t, entries)

	err = mock.RemoveAll("/tmp/old")
	require.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.WriteFileCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ReadDirCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveAllCallCount))
}

func TestMockFSWriter_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockFSWriter{
		WriteFileFunc:      nil,
		ReadDirFunc:        nil,
		RemoveAllFunc:      nil,
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.WriteFile(ctx, "/concurrent/file.go", []byte("data"))
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.ReadDir("/concurrent")
		}()
		go func() {
			defer wg.Done()
			_ = mock.RemoveAll("/concurrent/old")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.WriteFileCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ReadDirCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.RemoveAllCallCount))
}

func TestMockFSWriter_ImplementsFSWriterPort(t *testing.T) {
	t.Parallel()

	mock := &MockFSWriter{
		WriteFileFunc:      nil,
		ReadDirFunc:        nil,
		RemoveAllFunc:      nil,
		WriteFileCallCount: 0,
		ReadDirCallCount:   0,
		RemoveAllCallCount: 0,
	}

	var _ FSWriterPort = mock
}
