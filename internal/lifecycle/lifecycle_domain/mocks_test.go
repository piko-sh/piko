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
	"errors"
	"io"
	"io/fs"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockFileSystem_AddFile(t *testing.T) {
	t.Parallel()

	t.Run("adds file and creates parent directories", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/project/src/main.go", []byte("package main"))

		info, err := mockFS.Stat("/project/src/main.go")
		require.NoError(t, err)
		assert.False(t, info.IsDir())
		assert.Equal(t, "main.go", info.Name())

		info, err = mockFS.Stat("/project/src")
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestMockFileSystem_AddDir(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()
	mockFS.AddDir("/project/src")

	info, err := mockFS.Stat("/project/src")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestMockFileSystem_Open(t *testing.T) {
	t.Parallel()

	t.Run("opens existing file", func(t *testing.T) {
		t.Parallel()

		content := []byte("hello world")
		mockFS := NewMockFileSystem()
		mockFS.AddFile("/test.txt", content)

		reader, err := mockFS.Open("/test.txt")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		_, err := mockFS.Open("/nonexistent.txt")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestMockFileSystem_Stat(t *testing.T) {
	t.Parallel()

	t.Run("returns file info for file", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddFile("/test.txt", []byte("content"))

		info, err := mockFS.Stat("/test.txt")
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.Equal(t, int64(7), info.Size())
		assert.False(t, info.IsDir())
	})

	t.Run("returns file info for directory", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/mydir")

		info, err := mockFS.Stat("/mydir")
		require.NoError(t, err)
		assert.Equal(t, "mydir", info.Name())
		assert.True(t, info.IsDir())
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		_, err := mockFS.Stat("/nonexistent")
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})
}

func TestMockFileSystem_WalkDir(t *testing.T) {
	t.Parallel()

	t.Run("walks all files and directories", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project")
		mockFS.AddDir("/project/src")
		mockFS.AddFile("/project/src/main.go", []byte("package main"))
		mockFS.AddFile("/project/src/util.go", []byte("package main"))

		var paths []string
		err := mockFS.WalkDir("/project", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			paths = append(paths, path)
			return nil
		})

		require.NoError(t, err)
		assert.Contains(t, paths, "/project")
		assert.Contains(t, paths, "/project/src")
		assert.Contains(t, paths, "/project/src/main.go")
		assert.Contains(t, paths, "/project/src/util.go")
	})

	t.Run("returns error for non-existent root", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		err := mockFS.WalkDir("/nonexistent", func(path string, d fs.DirEntry, err error) error {
			return nil
		})

		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("handles SkipDir", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project")
		mockFS.AddFile("/project/a.txt", []byte("a"))
		mockFS.AddFile("/project/b.txt", []byte("b"))

		var count int
		err := mockFS.WalkDir("/project", func(path string, d fs.DirEntry, err error) error {
			count++
			return fs.SkipDir
		})

		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("propagates non-SkipDir error from callback", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()
		mockFS.AddDir("/project")
		mockFS.AddFile("/project/a.txt", []byte("a"))

		walkErr := errors.New("walk failed")
		err := mockFS.WalkDir("/project", func(path string, d fs.DirEntry, err error) error {
			return walkErr
		})

		assert.ErrorIs(t, err, walkErr)
	})
}

func TestMockFileSystem_makeDirEntry(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for path not in files or dirs", func(t *testing.T) {
		t.Parallel()

		mockFS := NewMockFileSystem()

		entry := mockFS.makeDirEntry("/nonexistent/path")

		assert.Nil(t, entry)
	})
}

func TestMockFileSystem_Rel(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	rel, err := mockFS.Rel("/project", "/project/src/main.go")
	require.NoError(t, err)
	assert.Equal(t, "src/main.go", rel)
}

func TestMockFileSystem_Join(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	result := mockFS.Join("/project", "src", "main.go")
	assert.Equal(t, "/project/src/main.go", result)
}

func TestMockFileSystem_IsNotExist(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	assert.True(t, mockFS.IsNotExist(fs.ErrNotExist))
	assert.False(t, mockFS.IsNotExist(fs.ErrPermission))
	assert.False(t, mockFS.IsNotExist(nil))
}

func TestMockReadCloser(t *testing.T) {
	t.Parallel()

	t.Run("reads all data", func(t *testing.T) {
		t.Parallel()

		data := []byte("hello world")
		rc := &mockReadCloser{data: data, offset: 0}

		buffer := make([]byte, 5)
		n, err := rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte("hello"), buffer)

		n, err = rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, []byte(" worl"), buffer)

		n, err = rc.Read(buffer)
		require.NoError(t, err)
		assert.Equal(t, 1, n)
		assert.Equal(t, byte('d'), buffer[0])

		n, err = rc.Read(buffer)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("close is no-op", func(t *testing.T) {
		t.Parallel()

		rc := &mockReadCloser{data: []byte("test"), offset: 0}
		err := rc.Close()
		assert.NoError(t, err)
	})
}

func TestMockDirEntry(t *testing.T) {
	t.Parallel()

	t.Run("file entry", func(t *testing.T) {
		t.Parallel()

		entry := &mockDirEntry{
			name:  "test.txt",
			isDir: false,
			mode:  0644,
			size:  100,
		}

		assert.Equal(t, "test.txt", entry.Name())
		assert.False(t, entry.IsDir())
		assert.Equal(t, fs.FileMode(0644).Type(), entry.Type())

		info, err := entry.Info()
		assert.Nil(t, info)
		assert.NoError(t, err)
	})

	t.Run("directory entry", func(t *testing.T) {
		t.Parallel()

		entry := &mockDirEntry{
			name:  "mydir",
			isDir: true,
			mode:  fs.ModeDir | 0755,
			size:  0,
		}

		assert.Equal(t, "mydir", entry.Name())
		assert.True(t, entry.IsDir())
		assert.Equal(t, fs.ModeDir, entry.Type())
	})
}

func TestMockFileInfo(t *testing.T) {
	t.Parallel()

	t.Run("file info methods", func(t *testing.T) {
		t.Parallel()

		testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		fi := &mockFileInfo{
			name:    "test.txt",
			size:    1024,
			mode:    0644,
			modTime: testTime,
			isDir:   false,
		}

		assert.Equal(t, "test.txt", fi.Name())
		assert.Equal(t, int64(1024), fi.Size())
		assert.Equal(t, fs.FileMode(0644), fi.Mode())
		assert.False(t, fi.IsDir())
		assert.Nil(t, fi.Sys())
		assert.Equal(t, testTime, fi.ModTime())
	})

	t.Run("directory info", func(t *testing.T) {
		t.Parallel()

		fi := &mockFileInfo{
			name:  "mydir",
			isDir: true,
			mode:  fs.ModeDir | 0755,
		}

		assert.True(t, fi.IsDir())
	})
}

func TestMockFileSystem_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := NewMockFileSystem()

	_, err := m.Open("/nonexistent")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	_, err = m.Stat("/nonexistent")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	err = m.WalkDir("/nonexistent", func(_ string, _ fs.DirEntry, _ error) error {
		return nil
	})
	assert.ErrorIs(t, err, fs.ErrNotExist)

	rel, err := m.Rel("/a", "/a/b")
	require.NoError(t, err)
	assert.Equal(t, "b", rel)

	assert.Equal(t, "/a/b", m.Join("/a", "b"))
	assert.True(t, m.IsNotExist(fs.ErrNotExist))
	assert.False(t, m.IsNotExist(nil))
}

func TestMockFileSystem_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := NewMockFileSystem()
	m.AddDir("/project")
	m.AddFile("/project/main.go", []byte("package main"))

	var wg sync.WaitGroup
	wg.Add(goroutines * 6)

	for range goroutines {
		go func() {
			defer wg.Done()
			m.AddFile("/project/concurrent.go", []byte("package main"))
		}()
		go func() {
			defer wg.Done()
			m.AddDir("/project/concurrent")
		}()
		go func() {
			defer wg.Done()
			_, _ = m.Open("/project/main.go")
		}()
		go func() {
			defer wg.Done()
			_, _ = m.Stat("/project/main.go")
		}()
		go func() {
			defer wg.Done()
			_ = m.WalkDir("/project", func(_ string, _ fs.DirEntry, _ error) error {
				return nil
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = m.Rel("/project", "/project/main.go")
		}()
	}

	wg.Wait()

	info, err := m.Stat("/project/main.go")
	require.NoError(t, err)
	assert.Equal(t, "main.go", info.Name())
}
