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

package wasm_adapters

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestNoOpConsole(t *testing.T) {
	t.Run("creates new instance", func(t *testing.T) {
		console := newNoOpConsole()
		require.NotNil(t, console)
	})

	t.Run("Debug does not panic", func(t *testing.T) {
		console := newNoOpConsole()
		assert.NotPanics(t, func() {
			console.Debug("test message")
			console.Debug("message with arguments", "arg1", 123)
		})
	})
}

func TestStdoutConsole(t *testing.T) {
	t.Run("creates new instance", func(t *testing.T) {
		console := newStdoutConsole()
		require.NotNil(t, console)
	})
}

func TestNoOpComponentCache(t *testing.T) {
	t.Run("creates new instance", func(t *testing.T) {
		cache := NewNoOpComponentCache()
		require.NotNil(t, cache)
	})

	t.Run("GetOrSet always calls loader", func(t *testing.T) {
		cache := NewNoOpComponentCache()
		loaderCalled := 0
		expected := &annotator_dto.ParsedComponent{}

		loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
			loaderCalled++
			return expected, nil
		}

		result1, err1 := cache.GetOrSet(context.Background(), "key1", loader)
		require.NoError(t, err1)
		assert.Equal(t, expected, result1)
		assert.Equal(t, 1, loaderCalled)

		result2, err2 := cache.GetOrSet(context.Background(), "key1", loader)
		require.NoError(t, err2)
		assert.Equal(t, expected, result2)
		assert.Equal(t, 2, loaderCalled)
	})

	t.Run("GetOrSet propagates loader error", func(t *testing.T) {
		cache := NewNoOpComponentCache()
		expectedErr := errors.New("loader failed")

		loader := func(_ context.Context) (*annotator_dto.ParsedComponent, error) {
			return nil, expectedErr
		}

		result, err := cache.GetOrSet(context.Background(), "key1", loader)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestInMemoryFSReader(t *testing.T) {
	t.Run("creates from string map", func(t *testing.T) {
		files := map[string]string{
			"file1.txt": "content1",
		}
		reader := NewInMemoryFSReader(files)
		require.NotNil(t, reader)
	})

	t.Run("reads existing file", func(t *testing.T) {
		files := map[string]string{
			"test.txt": "hello world",
		}
		reader := NewInMemoryFSReader(files)

		content, err := reader.ReadFile(context.Background(), "test.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello world"), content)
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		reader := NewInMemoryFSReader(map[string]string{})
		_, err := reader.ReadFile(context.Background(), "missing.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})

	t.Run("AddFile adds new file", func(t *testing.T) {
		reader := NewInMemoryFSReader(map[string]string{})
		reader.AddFile("new.txt", []byte("new content"))

		content, err := reader.ReadFile(context.Background(), "new.txt")
		require.NoError(t, err)
		assert.Equal(t, []byte("new content"), content)
	})
}

func TestInMemoryFSWriter(t *testing.T) {
	t.Run("creates new instance", func(t *testing.T) {
		writer := NewInMemoryFSWriter()
		require.NotNil(t, writer)
	})

	t.Run("WriteFile stores content", func(t *testing.T) {
		writer := NewInMemoryFSWriter()
		err := writer.WriteFile(context.Background(), "test.txt", []byte("content"))
		require.NoError(t, err)

		content, ok := writer.GetWrittenFile("test.txt")
		assert.True(t, ok)
		assert.Equal(t, []byte("content"), content)
	})

	t.Run("Clear removes all files", func(t *testing.T) {
		writer := NewInMemoryFSWriter()
		_ = writer.WriteFile(context.Background(), "file1.txt", []byte("content1"))
		writer.Clear()

		files := writer.GetWrittenFiles()
		assert.Empty(t, files)
	})

	t.Run("RemoveAll removes children", func(t *testing.T) {
		writer := NewInMemoryFSWriter()
		_ = writer.WriteFile(context.Background(), "dir/file1.txt", []byte("content1"))
		_ = writer.WriteFile(context.Background(), "other.txt", []byte("other"))

		err := writer.RemoveAll("dir")
		require.NoError(t, err)

		_, ok1 := writer.GetWrittenFile("dir/file1.txt")
		_, okOther := writer.GetWrittenFile("other.txt")
		assert.False(t, ok1)
		assert.True(t, okOther)
	})
}

func TestInMemoryDirEntry(t *testing.T) {
	t.Run("Name returns entry name", func(t *testing.T) {
		entry := &inMemoryDirEntry{name: "test.txt", isDir: false}
		assert.Equal(t, "test.txt", entry.Name())
	})

	t.Run("IsDir returns correct value", func(t *testing.T) {
		dirEntry := &inMemoryDirEntry{name: "subdir", isDir: true}
		fileEntry := &inMemoryDirEntry{name: "file.txt", isDir: false}

		assert.True(t, dirEntry.IsDir())
		assert.False(t, fileEntry.IsDir())
	})

	t.Run("Type returns correct mode", func(t *testing.T) {
		dirEntry := &inMemoryDirEntry{name: "subdir", isDir: true}
		fileEntry := &inMemoryDirEntry{name: "file.txt", isDir: false}

		assert.Equal(t, fs.ModeDir, dirEntry.Type())
		assert.Equal(t, fs.FileMode(0), fileEntry.Type())
	})
}

func TestInMemoryFileInfo(t *testing.T) {
	t.Run("Name returns file name", func(t *testing.T) {
		info := &inMemoryFileInfo{name: "test.txt", isDir: false, size: 100}
		assert.Equal(t, "test.txt", info.Name())
	})

	t.Run("Size returns file size", func(t *testing.T) {
		info := &inMemoryFileInfo{name: "test.txt", isDir: false, size: 12345}
		assert.Equal(t, int64(12345), info.Size())
	})

	t.Run("ModTime returns zero time", func(t *testing.T) {
		info := &inMemoryFileInfo{name: "test.txt"}
		assert.True(t, info.ModTime().IsZero())
	})

	t.Run("Sys returns nil", func(t *testing.T) {
		info := &inMemoryFileInfo{name: "test.txt"}
		assert.Nil(t, info.Sys())
	})
}
