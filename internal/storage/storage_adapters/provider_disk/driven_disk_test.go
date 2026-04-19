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

package provider_disk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_dto"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewDiskProvider(t *testing.T) {
	t.Parallel()

	t.Run("creates provider with sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, err := NewDiskProvider(Config{Sandbox: sandbox})

		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("returns error when no sandbox or base directory", func(t *testing.T) {
		t.Parallel()

		_, err := NewDiskProvider(Config{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseDirectory or sandbox must be provided")
	})
}

func TestDiskProvider_Put(t *testing.T) {
	t.Parallel()

	t.Run("writes file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "test.txt",
			Reader:     bytes.NewReader([]byte("hello world")),
		}

		err := provider.Put(context.Background(), params)

		require.NoError(t, err)
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object key cannot be empty")
	})

	t.Run("returns error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirAllErr = errors.New("mkdir failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "file.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directories")
	})

	t.Run("returns error when CreateTemp fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.CreateTempErr = errors.New("disk full")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "file.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create temporary file")
	})

	t.Run("returns error when Rename fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.RenameErr = errors.New("rename failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "file.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to atomically move file")
	})
}

func TestDiskProvider_Get(t *testing.T) {
	t.Parallel()

	t.Run("gets file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("hello world"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		reader, err := provider.Get(context.Background(), params)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, _ := io.ReadAll(reader)
		assert.Equal(t, "hello world", string(data))
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "",
		}

		_, err := provider.Get(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object key cannot be empty")
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "nonexistent.txt",
		}

		_, err := provider.Get(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("returns error when Open fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		sandbox.OpenErr = errors.New("permission denied")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		_, err := provider.Get(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})
}

func TestDiskProvider_Stat(t *testing.T) {
	t.Parallel()

	t.Run("returns file info", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("hello"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		info, err := provider.Stat(context.Background(), params)

		require.NoError(t, err)
		assert.Equal(t, int64(5), info.Size)
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "",
		}

		_, err := provider.Stat(context.Background(), params)

		require.Error(t, err)
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "nonexistent.txt",
		}

		_, err := provider.Stat(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object not found")
	})

	t.Run("returns error when Stat fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		sandbox.StatErr = errors.New("io error")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		_, err := provider.Stat(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat file")
	})
}

func TestDiskProvider_Remove(t *testing.T) {
	t.Parallel()

	t.Run("removes file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		err := provider.Remove(context.Background(), params)

		require.NoError(t, err)
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "",
		}

		err := provider.Remove(context.Background(), params)

		require.Error(t, err)
	})

	t.Run("returns error when Remove fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		sandbox.RemoveErr = errors.New("permission denied")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		err := provider.Remove(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove file")
	})
}

func TestDiskProvider_Exists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		exists, err := provider.Exists(context.Background(), params)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for non-existing file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "nonexistent.txt",
		}

		exists, err := provider.Exists(context.Background(), params)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "",
		}

		_, err := provider.Exists(context.Background(), params)

		require.Error(t, err)
	})

	t.Run("returns error when Stat fails unexpectedly", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/test.txt", []byte("data"))
		sandbox.StatErr = errors.New("io error")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "test.txt",
		}

		_, err := provider.Exists(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat")
	})
}

func TestDiskProvider_Rename(t *testing.T) {
	t.Parallel()

	t.Run("renames file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/old.txt", []byte("data"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Rename(context.Background(), "repo", "old.txt", "new.txt")

		require.NoError(t, err)
	})

	t.Run("returns error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/old.txt", []byte("data"))
		sandbox.MkdirAllErr = errors.New("mkdir failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Rename(context.Background(), "repo", "old.txt", "subdir/new.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create destination directories")
	})

	t.Run("returns error when Rename fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/old.txt", []byte("data"))
		sandbox.RenameErr = errors.New("rename failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Rename(context.Background(), "repo", "old.txt", "new.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to rename")
	})
}

func TestDiskProvider_Copy(t *testing.T) {
	t.Parallel()

	t.Run("copies file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/src.txt", []byte("content"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Copy(context.Background(), "repo", "src.txt", "dst.txt")

		require.NoError(t, err)
	})

	t.Run("returns error when source file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Copy(context.Background(), "repo", "nonexistent.txt", "dst.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open source file")
	})

	t.Run("returns error when CreateTemp fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/src.txt", []byte("content"))
		sandbox.CreateTempErr = errors.New("disk full")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Copy(context.Background(), "repo", "src.txt", "dst.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create temporary file")
	})

	t.Run("returns error when MkdirAll fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/src.txt", []byte("content"))
		sandbox.MkdirAllErr = errors.New("permission denied")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Copy(context.Background(), "repo", "src.txt", "dst.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create destination directories")
	})

	t.Run("returns error when Rename fails during copy", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/src.txt", []byte("content"))
		sandbox.RenameErr = errors.New("cross-device link")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Copy(context.Background(), "repo", "src.txt", "dst.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to atomically move file")
	})
}

func TestDiskProvider_Put_WriteFailure(t *testing.T) {
	t.Parallel()

	t.Run("returns error when temp file write fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.NextTempFileWriteErr = errors.New("disk full during write")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "file.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write data to temporary file")
	})

	t.Run("returns error when temp file fsync fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.NextTempFileSyncErr = errors.New("fsync failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "file.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fsync temporary file")
	})
}

func TestDiskProvider_Get_ByteRange(t *testing.T) {
	t.Parallel()

	t.Run("returns partial content for byte range", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/data.txt", []byte("0123456789ABCDEF"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "data.txt",
			ByteRange:  &storage_dto.ByteRange{Start: 5, End: 9},
		}

		reader, err := provider.Get(context.Background(), params)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, _ := io.ReadAll(reader)
		assert.Equal(t, "56789", string(data))
	})

	t.Run("returns content from offset to end for open-ended range", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/data.txt", []byte("0123456789ABCDEF"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "data.txt",
			ByteRange:  &storage_dto.ByteRange{Start: 10, End: -1},
		}

		reader, err := provider.Get(context.Background(), params)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, _ := io.ReadAll(reader)
		assert.Equal(t, "ABCDEF", string(data))
	})

	t.Run("returns single byte for single-byte range", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/data.txt", []byte("ABCDEFGHIJ"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := storage_dto.GetParams{
			Repository: "repo",
			Key:        "data.txt",
			ByteRange:  &storage_dto.ByteRange{Start: 3, End: 3},
		}

		reader, err := provider.Get(context.Background(), params)

		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, _ := io.ReadAll(reader)
		assert.Equal(t, "D", string(data))
	})
}

func TestDiskProvider_GetHash(t *testing.T) {
	t.Parallel()

	t.Run("returns hash for existing file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/file.txt", []byte("hash me"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		hash, err := provider.GetHash(context.Background(), storage_dto.GetParams{
			Repository: "repo",
			Key:        "file.txt",
		})

		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Len(t, hash, 64, "SHA256 hex should be 64 characters")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		_, err := provider.GetHash(context.Background(), storage_dto.GetParams{
			Repository: "repo",
			Key:        "missing.txt",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for empty key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		_, err := provider.GetHash(context.Background(), storage_dto.GetParams{
			Repository: "repo",
			Key:        "",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "object key cannot be empty")
	})
}

func TestDiskProvider_Metadata(t *testing.T) {
	t.Parallel()

	t.Run("writes and reads metadata sidecar", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "with-meta.txt",
			Reader:     bytes.NewReader([]byte("data")),
			Metadata:   map[string]string{"x-custom": "value", "author": "test"},
		}

		err := provider.Put(context.Background(), params)
		require.NoError(t, err)

		info, err := provider.Stat(context.Background(), storage_dto.GetParams{
			Repository: "repo",
			Key:        "with-meta.txt",
		})
		require.NoError(t, err)
		assert.Equal(t, "value", info.Metadata["x-custom"])
		assert.Equal(t, "test", info.Metadata["author"])
	})

	t.Run("nil metadata does not create sidecar", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		params := &storage_dto.PutParams{
			Repository: "repo",
			Key:        "no-meta.txt",
			Reader:     bytes.NewReader([]byte("data")),
		}

		err := provider.Put(context.Background(), params)
		require.NoError(t, err)

		info, err := provider.Stat(context.Background(), storage_dto.GetParams{
			Repository: "repo",
			Key:        "no-meta.txt",
		})
		require.NoError(t, err)
		assert.Nil(t, info.Metadata)
	})
}

func TestDiskProvider_CopyToAnotherRepository(t *testing.T) {
	t.Parallel()

	t.Run("copies between repositories", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("src-repo/file.txt", []byte("cross-repo data"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.CopyToAnotherRepository(
			context.Background(),
			"src-repo", "file.txt", "dst-repo", "copied.txt",
		)

		require.NoError(t, err)
	})

	t.Run("returns error for empty source key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.CopyToAnotherRepository(
			context.Background(),
			"src-repo", "", "dst-repo", "dst.txt",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source key")
	})

	t.Run("returns error for empty destination key", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.CopyToAnotherRepository(
			context.Background(),
			"src-repo", "file.txt", "dst-repo", "",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid destination key")
	})
}

func TestDiskProvider_PutMany(t *testing.T) {
	t.Parallel()

	t.Run("uploads batch successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		objects := make([]storage_dto.PutObjectSpec, 3)
		for i := range 3 {
			objects[i] = storage_dto.PutObjectSpec{
				Key:         fmt.Sprintf("batch/%d.txt", i),
				Reader:      strings.NewReader(fmt.Sprintf("content %d", i)),
				Size:        int64(len(fmt.Sprintf("content %d", i))),
				ContentType: "text/plain",
			}
		}

		result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
			Repository:      "repo",
			Objects:         objects,
			Concurrency:     3,
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRequested)
		assert.Equal(t, 3, result.TotalSuccessful)
		assert.Equal(t, 0, result.TotalFailed)
		assert.Len(t, result.SuccessfulKeys, 3)
	})

	t.Run("returns empty result for empty batch", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
			Repository:      "repo",
			Objects:         nil,
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalRequested)
		assert.Equal(t, 0, result.TotalSuccessful)
	})

	t.Run("continues on error when ContinueOnError is true", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		objects := []storage_dto.PutObjectSpec{
			{Key: "good.txt", Reader: strings.NewReader("data"), Size: 4, ContentType: "text/plain"},
			{Key: "", Reader: strings.NewReader("data"), Size: 4, ContentType: "text/plain"},
			{Key: "also-good.txt", Reader: strings.NewReader("data"), Size: 4, ContentType: "text/plain"},
		}

		result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
			Repository:      "repo",
			Objects:         objects,
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRequested)
		assert.Equal(t, 2, result.TotalSuccessful)
		assert.Equal(t, 1, result.TotalFailed)
		assert.Len(t, result.FailedKeys, 1)
	})

	t.Run("stops on first error when ContinueOnError is false", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		objects := []storage_dto.PutObjectSpec{
			{Key: "", Reader: strings.NewReader("data"), Size: 4, ContentType: "text/plain"},
			{Key: "never-reached.txt", Reader: strings.NewReader("data"), Size: 4, ContentType: "text/plain"},
		}

		result, err := provider.PutMany(context.Background(), &storage_dto.PutManyParams{
			Repository:      "repo",
			Objects:         objects,
			ContinueOnError: false,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, result.TotalRequested)
		assert.Equal(t, 0, result.TotalSuccessful)
		assert.Equal(t, 1, result.TotalFailed, "should stop after first failure")
	})
}

func TestDiskProvider_RemoveMany(t *testing.T) {
	t.Parallel()

	t.Run("removes batch successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		for i := range 3 {
			sandbox.AddFile(fmt.Sprintf("repo/%d.txt", i), []byte("data"))
		}
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		result, err := provider.RemoveMany(context.Background(), storage_dto.RemoveManyParams{
			Repository:      "repo",
			Keys:            []string{"0.txt", "1.txt", "2.txt"},
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRequested)
		assert.Equal(t, 3, result.TotalSuccessful)
		assert.Equal(t, 0, result.TotalFailed)
	})

	t.Run("returns empty result for empty batch", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		result, err := provider.RemoveMany(context.Background(), storage_dto.RemoveManyParams{
			Repository:      "repo",
			Keys:            nil,
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalRequested)
	})

	t.Run("continues on error when ContinueOnError is true", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("repo/good.txt", []byte("data"))
		sandbox.AddFile("repo/also-good.txt", []byte("data"))
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		result, err := provider.RemoveMany(context.Background(), storage_dto.RemoveManyParams{
			Repository:      "repo",
			Keys:            []string{"good.txt", "", "also-good.txt"},
			ContinueOnError: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalRequested)
		assert.Equal(t, 2, result.TotalSuccessful)
		assert.Equal(t, 1, result.TotalFailed)
	})
}

func TestDiskProvider_Remove_Idempotent(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

	err := provider.Remove(context.Background(), storage_dto.GetParams{
		Repository: "repo",
		Key:        "never-existed.txt",
	})
	assert.NoError(t, err, "removing non-existent file should succeed silently")
}

func TestDiskProvider_PresignURL(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

	_, err := provider.PresignURL(context.Background(), storage_dto.PresignParams{
		Repository: "repo",
		Key:        "file.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestDiskProvider_PresignDownloadURL(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

	_, err := provider.PresignDownloadURL(context.Background(), storage_dto.PresignDownloadParams{
		Repository: "repo",
		Key:        "file.txt",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestDiskProvider_Capabilities(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()
	provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

	assert.False(t, provider.SupportsMultipart())
	assert.False(t, provider.SupportsBatchOperations())
	assert.False(t, provider.SupportsRetry())
	assert.False(t, provider.SupportsCircuitBreaking())
	assert.False(t, provider.SupportsRateLimiting())
	assert.False(t, provider.SupportsPresignedURLs())
}

func TestDiskProvider_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes sandbox successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Close(context.Background())
		assert.NoError(t, err)
	})

	t.Run("returns error when sandbox close fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.CloseErr = errors.New("close failed")
		provider, _ := NewDiskProvider(Config{Sandbox: sandbox})

		err := provider.Close(context.Background())
		assert.Error(t, err)
	})
}
