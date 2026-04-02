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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestAtomicWriteFile(t *testing.T) {
	t.Parallel()

	t.Run("writes file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		data := []byte("hello world")

		err := AtomicWriteFile(context.Background(), sandbox, "subdir/test.txt", data, 0644)

		require.NoError(t, err)

		written, readErr := sandbox.ReadFile("subdir/test.txt")
		require.NoError(t, readErr)
		assert.Equal(t, data, written)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		err := AtomicWriteFile(context.Background(), sandbox, "a/b/c/file.txt", []byte("data"), 0644)

		require.NoError(t, err)
		assert.Equal(t, 1, sandbox.CallCounts["MkdirAll"])
	})

	t.Run("returns error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()

		mkdirErr := errors.New("mkdir failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.MkdirAllErr = mkdirErr

		err := AtomicWriteFile(context.Background(), sandbox, "dir/file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, mkdirErr)
		assert.Contains(t, err.Error(), "failed to create directory")
	})

	t.Run("returns error when CreateTemp fails", func(t *testing.T) {
		t.Parallel()

		createTempErr := errors.New("createtemp failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.CreateTempErr = createTempErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, createTempErr)
		assert.Contains(t, err.Error(), "failed to create temporary file")
	})

	t.Run("returns error when Write fails", func(t *testing.T) {
		t.Parallel()

		writeErr := errors.New("write failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileWriteErr = writeErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, writeErr)
		assert.Contains(t, err.Error(), "failed to write to temporary file")
	})

	t.Run("returns error when Sync fails", func(t *testing.T) {
		t.Parallel()

		syncErr := errors.New("sync failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileSyncErr = syncErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, syncErr)
		assert.Contains(t, err.Error(), "failed to sync temporary file")
	})

	t.Run("returns error when Close fails", func(t *testing.T) {
		t.Parallel()

		closeErr := errors.New("close failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileCloseErr = closeErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, closeErr)
		assert.Contains(t, err.Error(), "failed to close temporary file")
	})

	t.Run("returns error when Chmod fails", func(t *testing.T) {
		t.Parallel()

		chmodErr := errors.New("chmod failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.ChmodErr = chmodErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, chmodErr)
		assert.Contains(t, err.Error(), "failed to set permissions")
	})

	t.Run("returns error when Rename fails", func(t *testing.T) {
		t.Parallel()

		renameErr := errors.New("rename failed")
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.RenameErr = renameErr

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		require.Error(t, err)
		assert.ErrorIs(t, err, renameErr)
		assert.Contains(t, err.Error(), "failed to atomically rename")
	})

	t.Run("cleans up temp file on write error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileWriteErr = errors.New("write failed")

		_ = AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("cleans up temp file on sync error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileSyncErr = errors.New("sync failed")

		_ = AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("removes temp file on close error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.NextTempFileCloseErr = errors.New("close failed")

		_ = AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("removes temp file on chmod error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.ChmodErr = errors.New("chmod failed")

		_ = AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("removes temp file on rename error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.RenameErr = errors.New("rename failed")

		_ = AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0644)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("handles empty data", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		err := AtomicWriteFile(context.Background(), sandbox, "empty.txt", []byte{}, 0644)

		require.NoError(t, err)
		written, _ := sandbox.ReadFile("empty.txt")
		assert.Empty(t, written)
	})

	t.Run("respects file permissions", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		var capturedPerm fs.FileMode
		sandbox.ChmodFunc = func(name string, mode fs.FileMode) error {
			capturedPerm = mode
			return nil
		}

		err := AtomicWriteFile(context.Background(), sandbox, "file.txt", []byte("data"), 0755)

		require.NoError(t, err)
		assert.Equal(t, fs.FileMode(0755), capturedPerm)
	})
}

func TestCleanupTempFile(t *testing.T) {
	t.Parallel()

	t.Run("handles nil file gracefully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		cleanupTempFile(context.Background(), sandbox, nil)
	})

	t.Run("closes and removes file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		file := safedisk.NewMockFileHandle("temp.txt", "/sandbox/temp.txt", nil)

		cleanupTempFile(context.Background(), sandbox, file)

		assert.True(t, file.IsClosed())

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("continues cleanup even when close fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		file := safedisk.NewMockFileHandle("temp.txt", "/sandbox/temp.txt", nil)
		file.CloseErr = errors.New("close failed")

		cleanupTempFile(context.Background(), sandbox, file)

		assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1)
	})

	t.Run("logs warning when remove fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.RemoveErr = errors.New("remove failed")
		file := safedisk.NewMockFileHandle("temp.txt", "/sandbox/temp.txt", nil)

		cleanupTempFile(context.Background(), sandbox, file)
	})
}

func TestRemoveTempFile(t *testing.T) {
	t.Parallel()

	t.Run("removes file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.AddFile("temp.txt", []byte("data"))

		removeTempFile(context.Background(), sandbox, "temp.txt")

		_, err := sandbox.ReadFile("temp.txt")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("logs warning when remove fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.RemoveErr = errors.New("remove failed")

		removeTempFile(context.Background(), sandbox, "temp.txt")

		assert.Equal(t, 1, sandbox.CallCounts["Remove"])
	})
}

func TestAtomicWriteFile_ErrorScenarios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		setupMock      func(*safedisk.MockSandbox)
		wantErrMessage string
		wantCleanup    bool
	}{
		{
			name: "MkdirAll error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.MkdirAllErr = errors.New("permission denied")
			},
			wantErrMessage: "failed to create directory",
			wantCleanup:    false,
		},
		{
			name: "CreateTemp error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.CreateTempErr = errors.New("disk full")
			},
			wantErrMessage: "failed to create temporary file",
			wantCleanup:    false,
		},
		{
			name: "Write error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileWriteErr = errors.New("disk full")
			},
			wantErrMessage: "failed to write to temporary file",
			wantCleanup:    true,
		},
		{
			name: "Sync error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileSyncErr = errors.New("io error")
			},
			wantErrMessage: "failed to sync temporary file",
			wantCleanup:    true,
		},
		{
			name: "Close error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileCloseErr = errors.New("io error")
			},
			wantErrMessage: "failed to close temporary file",
			wantCleanup:    true,
		},
		{
			name: "Chmod error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.ChmodErr = errors.New("permission denied")
			},
			wantErrMessage: "failed to set permissions",
			wantCleanup:    true,
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("cross-device link")
			},
			wantErrMessage: "failed to atomically rename",
			wantCleanup:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
			defer sandbox.Close()
			tc.setupMock(sandbox)

			err := AtomicWriteFile(context.Background(), sandbox, "dir/file.txt", []byte("data"), 0644)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErrMessage)

			if tc.wantCleanup {
				assert.GreaterOrEqual(t, sandbox.CallCounts["Remove"], 1,
					"expected cleanup to remove temp file")
			}
		})
	}
}
