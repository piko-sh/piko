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

package lifecycle_adapters

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
)

func TestMapToFileEventType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		op       fsnotify.Op
		expected lifecycle_dto.FileEventType
	}{
		{
			name:     "create event",
			op:       fsnotify.Create,
			expected: lifecycle_dto.FileEventTypeCreate,
		},
		{
			name:     "write event",
			op:       fsnotify.Write,
			expected: lifecycle_dto.FileEventTypeWrite,
		},
		{
			name:     "remove event",
			op:       fsnotify.Remove,
			expected: lifecycle_dto.FileEventTypeRemove,
		},
		{
			name:     "rename event",
			op:       fsnotify.Rename,
			expected: lifecycle_dto.FileEventTypeRename,
		},
		{
			name:     "chmod event maps to unknown",
			op:       fsnotify.Chmod,
			expected: lifecycle_dto.FileEventTypeUnknown,
		},
		{
			name:     "zero op maps to unknown",
			op:       0,
			expected: lifecycle_dto.FileEventTypeUnknown,
		},
		{
			name:     "create takes precedence in combined ops",
			op:       fsnotify.Create | fsnotify.Write,
			expected: lifecycle_dto.FileEventTypeCreate,
		},
		{
			name:     "write takes precedence over chmod",
			op:       fsnotify.Write | fsnotify.Chmod,
			expected: lifecycle_dto.FileEventTypeWrite,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := mapToFileEventType(tc.op)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldIgnoreEvent(t *testing.T) {
	t.Parallel()

	watcher := &fsNotifyWatcher{}

	testCases := []struct {
		name     string
		event    fsnotify.Event
		expected bool
	}{
		{
			name:     "ignores editor backup files with tilde",
			event:    fsnotify.Event{Name: "/path/to/file.go~"},
			expected: true,
		},
		{
			name:     "ignores vim swap files ending with tilde",
			event:    fsnotify.Event{Name: "/path/to/.file.swp~"},
			expected: true,
		},
		{
			name:     "allows normal go files",
			event:    fsnotify.Event{Name: "/path/to/main.go"},
			expected: false,
		},
		{
			name:     "allows pk files",
			event:    fsnotify.Event{Name: "/path/to/page.pk"},
			expected: false,
		},
		{
			name:     "allows files with tilde in middle",
			event:    fsnotify.Event{Name: "/path/to/file~backup.go"},
			expected: false,
		},
		{
			name:     "allows hidden files without tilde",
			event:    fsnotify.Event{Name: "/path/to/.gitignore"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := watcher.shouldIgnoreEvent(context.Background(), tc.event)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPathCoveredByStaticWatch(t *testing.T) {
	t.Parallel()

	t.Run("returns true when parent directory is watched", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			staticDirs: map[string]bool{
				"/project/src": true,
			},
		}

		result := watcher.isPathCoveredByStaticWatch("/project/src/main.go")
		assert.True(t, result)
	})

	t.Run("returns true when ancestor directory is watched", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			staticDirs: map[string]bool{
				"/project": true,
			},
		}

		result := watcher.isPathCoveredByStaticWatch("/project/src/pkg/file.go")
		assert.True(t, result)
	})

	t.Run("returns false when no ancestor is watched", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			staticDirs: map[string]bool{
				"/other/path": true,
			},
		}

		result := watcher.isPathCoveredByStaticWatch("/project/src/main.go")
		assert.False(t, result)
	})

	t.Run("returns false with empty static dirs", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			staticDirs: map[string]bool{},
		}

		result := watcher.isPathCoveredByStaticWatch("/any/path/file.go")
		assert.False(t, result)
	})

	t.Run("handles root path correctly", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			staticDirs: map[string]bool{
				"/": true,
			},
		}

		result := watcher.isPathCoveredByStaticWatch("/any/deep/path/file.go")
		assert.True(t, result)
	})
}

func TestShouldDebounceEvent(t *testing.T) {
	t.Parallel()

	t.Run("does not debounce first event", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: make(map[string]time.Time),
			mu:        sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file.go",
			Op:   fsnotify.Write,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.False(t, result)
	})

	t.Run("debounces rapid write events", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: map[string]time.Time{
				"/path/to/file.go": time.Now(),
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file.go",
			Op:   fsnotify.Write,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.True(t, result)
	})

	t.Run("does not debounce create events", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: map[string]time.Time{
				"/path/to/file.go": time.Now(),
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file.go",
			Op:   fsnotify.Create,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.False(t, result)
	})

	t.Run("does not debounce remove events", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: map[string]time.Time{
				"/path/to/file.go": time.Now(),
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file.go",
			Op:   fsnotify.Remove,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.False(t, result)
	})

	t.Run("allows event after debounce interval", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: map[string]time.Time{
				"/path/to/file.go": time.Now().Add(-100 * time.Millisecond),
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file.go",
			Op:   fsnotify.Write,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.False(t, result)
	})

	t.Run("different files are not debounced against each other", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{
			debounced: map[string]time.Time{
				"/path/to/file1.go": time.Now(),
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/path/to/file2.go",
			Op:   fsnotify.Write,
		}

		result := watcher.shouldDebounceEvent(context.Background(), event)
		assert.False(t, result)
	})
}

func TestShouldSkipDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("skips node_modules", func(t *testing.T) {
		t.Parallel()

		nodeModulesDir := filepath.Join(tempDir, "node_modules_test")
		require.NoError(t, os.Mkdir(nodeModulesDir, 0755))

		entry, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		var nodeModulesEntry fs.DirEntry
		for _, e := range entry {
			if e.Name() == "node_modules_test" {
				nodeModulesEntry = e
				break
			}
		}
		require.NotNil(t, nodeModulesEntry)

		watcher := &fsNotifyWatcher{}

		result := watcher.shouldSkipDirectory(context.Background(), filepath.Join(tempDir, "node_modules"), nodeModulesEntry)
		assert.True(t, result)
	})

	t.Run("skips .git directory", func(t *testing.T) {
		t.Parallel()

		gitDir := filepath.Join(tempDir, "git_test")
		require.NoError(t, os.Mkdir(gitDir, 0755))

		entry, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		var gitEntry fs.DirEntry
		for _, e := range entry {
			if e.Name() == "git_test" {
				gitEntry = e
				break
			}
		}
		require.NotNil(t, gitEntry)

		watcher := &fsNotifyWatcher{}

		result := watcher.shouldSkipDirectory(context.Background(), filepath.Join(tempDir, ".git"), gitEntry)
		assert.True(t, result)
	})

	t.Run("allows normal directories", func(t *testing.T) {
		t.Parallel()

		srcDir := filepath.Join(tempDir, "src_test")
		require.NoError(t, os.Mkdir(srcDir, 0755))

		entry, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		var srcEntry fs.DirEntry
		for _, e := range entry {
			if e.Name() == "src_test" {
				srcEntry = e
				break
			}
		}
		require.NotNil(t, srcEntry)

		watcher := &fsNotifyWatcher{}
		result := watcher.shouldSkipDirectory(context.Background(), srcDir, srcEntry)
		assert.False(t, result)
	})
}

func TestNewFSNotifyWatcher(t *testing.T) {
	t.Parallel()

	t.Run("creates watcher successfully", func(t *testing.T) {
		t.Parallel()

		watcher, err := NewFSNotifyWatcher(nil)
		require.NoError(t, err)
		require.NotNil(t, watcher)

		err = watcher.Close()
		assert.NoError(t, err)
	})
}

func TestFSNotifyWatcher_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes successfully", func(t *testing.T) {
		t.Parallel()

		watcher, err := NewFSNotifyWatcher(nil)
		require.NoError(t, err)

		err = watcher.Close()
		assert.NoError(t, err)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		t.Parallel()

		watcher, err := NewFSNotifyWatcher(nil)
		require.NoError(t, err)

		err = watcher.Close()
		assert.NoError(t, err)

		err = watcher.Close()
		assert.NoError(t, err)
	})
}

func TestFSNotifyWatcher_UpdateWatchedFiles(t *testing.T) {
	t.Parallel()

	t.Run("returns error when closed", func(t *testing.T) {
		t.Parallel()

		watcher, err := NewFSNotifyWatcher(nil)
		require.NoError(t, err)

		err = watcher.Close()
		require.NoError(t, err)

		err = watcher.UpdateWatchedFiles(context.Background(), []string{"/path/to/file.png"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("updates dynamic files map", func(t *testing.T) {
		t.Parallel()

		watcher, err := NewFSNotifyWatcher(nil)
		require.NoError(t, err)
		defer func() { _ = watcher.Close() }()

		tempDir := t.TempDir()
		tempFile := filepath.Join(tempDir, "test.png")
		require.NoError(t, os.WriteFile(tempFile, []byte("test"), 0644))

		err = watcher.UpdateWatchedFiles(context.Background(), []string{tempFile})
		assert.NoError(t, err)
	})
}

func TestFSNotifyWatcher_handleDirectoryRemoval(t *testing.T) {
	t.Parallel()

	t.Run("removes static watch on remove event", func(t *testing.T) {
		t.Parallel()

		fsWatcher, err := fsnotify.NewWatcher()
		require.NoError(t, err)
		defer func() { _ = fsWatcher.Close() }()

		watcher := &fsNotifyWatcher{
			watcher: fsWatcher,
			staticDirs: map[string]bool{
				"/project/src": true,
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/project/src",
			Op:   fsnotify.Remove,
		}

		watcher.handleDirectoryRemoval(context.Background(), event)

		watcher.mu.Lock()
		_, exists := watcher.staticDirs["/project/src"]
		watcher.mu.Unlock()

		assert.False(t, exists)
	})

	t.Run("removes static watch on rename event", func(t *testing.T) {
		t.Parallel()

		fsWatcher, err := fsnotify.NewWatcher()
		require.NoError(t, err)
		defer func() { _ = fsWatcher.Close() }()

		watcher := &fsNotifyWatcher{
			watcher: fsWatcher,
			staticDirs: map[string]bool{
				"/project/old": true,
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/project/old",
			Op:   fsnotify.Rename,
		}

		watcher.handleDirectoryRemoval(context.Background(), event)

		watcher.mu.Lock()
		_, exists := watcher.staticDirs["/project/old"]
		watcher.mu.Unlock()

		assert.False(t, exists)
	})

	t.Run("ignores write events", func(t *testing.T) {
		t.Parallel()

		fsWatcher, err := fsnotify.NewWatcher()
		require.NoError(t, err)
		defer func() { _ = fsWatcher.Close() }()

		watcher := &fsNotifyWatcher{
			watcher: fsWatcher,
			staticDirs: map[string]bool{
				"/project/src": true,
			},
			mu: sync.Mutex{},
		}

		event := fsnotify.Event{
			Name: "/project/src",
			Op:   fsnotify.Write,
		}

		watcher.handleDirectoryRemoval(context.Background(), event)

		watcher.mu.Lock()
		_, exists := watcher.staticDirs["/project/src"]
		watcher.mu.Unlock()

		assert.True(t, exists)
	})
}
