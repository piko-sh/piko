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
			name:     "ignores vim swap files",
			event:    fsnotify.Event{Name: "/path/to/.page.pk.swp"},
			expected: true,
		},
		{
			name:     "ignores tmp files",
			event:    fsnotify.Event{Name: "/path/to/logo.svg.tmp"},
			expected: true,
		},
		{
			name:     "ignores vim write-probe file",
			event:    fsnotify.Event{Name: "/path/to/4913"},
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

func TestIsCoalescedEvent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		op       fsnotify.Op
		expected bool
	}{
		{name: "write is held back", op: fsnotify.Write, expected: true},
		{name: "chmod is held back", op: fsnotify.Chmod, expected: true},
		{name: "write with chmod is held back", op: fsnotify.Write | fsnotify.Chmod, expected: true},
		{name: "create is forwarded", op: fsnotify.Create, expected: false},
		{name: "create with write is forwarded", op: fsnotify.Create | fsnotify.Write, expected: false},
		{name: "remove is forwarded", op: fsnotify.Remove, expected: false},
		{name: "rename is forwarded", op: fsnotify.Rename, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isCoalescedEvent(fsnotify.Event{Name: "/path/to/file.go", Op: tc.op}))
		})
	}
}

func TestWriteDebouncer(t *testing.T) {
	t.Parallel()

	const (
		quiet   = 50 * time.Millisecond
		maxWait = 500 * time.Millisecond
		path    = "/path/to/logo.svg"
	)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("releases a single write after the quiet period", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start)

		deadline, pending := debouncer.nextDeadline()
		require.True(t, pending)
		assert.Equal(t, start.Add(quiet), deadline)
		assert.Empty(t, debouncer.takeDue(start.Add(quiet-time.Millisecond)), "nothing is due before the quiet period ends")

		due := debouncer.takeDue(start.Add(quiet))
		require.Len(t, due, 1)
		assert.Equal(t, path, due[0].Name)
		assert.True(t, due[0].Has(fsnotify.Write))
	})

	t.Run("each write restarts the quiet period so the trailing write wins", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start.Add(20*time.Millisecond))

		assert.Empty(t, debouncer.takeDue(start.Add(quiet)), "the second write must hold the event back")

		deadline, pending := debouncer.nextDeadline()
		require.True(t, pending)
		assert.Equal(t, start.Add(20*time.Millisecond+quiet), deadline)
		assert.Len(t, debouncer.takeDue(deadline), 1, "exactly one event is released for the burst")
	})

	t.Run("a continuous burst is released at the maximum wait", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		for offset := time.Duration(0); offset <= maxWait; offset += 10 * time.Millisecond {
			debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start.Add(offset))
		}

		deadline, pending := debouncer.nextDeadline()
		require.True(t, pending)
		assert.Equal(t, start.Add(maxWait), deadline)
		assert.Len(t, debouncer.takeDue(start.Add(maxWait)), 1)
	})

	t.Run("a chmod after a write is still reported as a write", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Chmod}, start.Add(time.Millisecond))

		due := debouncer.takeDue(start.Add(time.Second))
		require.Len(t, due, 1)
		assert.Equal(t, lifecycle_dto.FileEventTypeWrite, mapToFileEventType(due[0].Op))
	})

	t.Run("paths are debounced independently", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		debouncer.record(fsnotify.Event{Name: "/a.go", Op: fsnotify.Write}, start)
		debouncer.record(fsnotify.Event{Name: "/b.go", Op: fsnotify.Write}, start.Add(40*time.Millisecond))

		due := debouncer.takeDue(start.Add(quiet))
		require.Len(t, due, 1)
		assert.Equal(t, "/a.go", due[0].Name)
	})

	t.Run("discard drops a pending write", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		debouncer.record(fsnotify.Event{Name: path, Op: fsnotify.Write}, start)
		debouncer.discard(path)

		_, pending := debouncer.nextDeadline()
		assert.False(t, pending)
		assert.Empty(t, debouncer.takeDue(start.Add(time.Second)))
	})

	t.Run("released paths are pruned", func(t *testing.T) {
		t.Parallel()

		debouncer := newWriteDebouncer(quiet, maxWait)
		for i := range 100 {
			debouncer.record(fsnotify.Event{Name: filepath.Join("/assets", string(rune('a'+i%26)), "file.svg"), Op: fsnotify.Write}, start)
		}
		debouncer.takeDue(start.Add(time.Second))

		assert.Empty(t, debouncer.pending)
	})
}

func TestFSNotifyWatcher_TruncateThenWriteDeliversFinalContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	require.NoError(t, os.MkdirAll(assets, 0o755))
	target := filepath.Join(assets, "logo.svg")
	require.NoError(t, os.WriteFile(target, []byte(`<svg>old</svg>`), 0o644))

	watcher, err := NewFSNotifyWatcher(nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watcher.Close()) })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	events, err := watcher.Watch(ctx, []string{assets}, nil)
	require.NoError(t, err)

	finalContent := `<svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"></path></svg>`

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_TRUNC, 0o644)
	require.NoError(t, err)
	time.Sleep(20 * time.Millisecond)
	_, err = file.WriteString(finalContent)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	var lastRead string
	received := 0
	quiet := time.NewTimer(3 * time.Second)
	defer quiet.Stop()
	for {
		select {
		case event, ok := <-events:
			require.True(t, ok, "watcher closed before the write was reported")
			if event.Path != target {
				continue
			}
			received++
			content, readErr := os.ReadFile(target)
			require.NoError(t, readErr)
			lastRead = string(content)
			quiet.Reset(4 * writeQuietPeriod)
			continue
		case <-quiet.C:
		}
		break
	}

	require.Positive(t, received, "the write burst must be reported")
	assert.Equal(t, finalContent, lastRead,
		"the last event for a truncate-then-write burst must be handled after the final write")
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

func TestFSNotifyWatcher_WatchReadyForImmediateRename(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pages := filepath.Join(root, "pages")
	nested := filepath.Join(pages, "news")
	assets := filepath.Join(root, "assets")
	require.NoError(t, os.MkdirAll(nested, 0o755))
	require.NoError(t, os.MkdirAll(assets, 0o755))
	oldPath := filepath.Join(nested, "main.pk")
	newPath := filepath.Join(nested, "home.pk")
	require.NoError(t, os.WriteFile(oldPath, []byte("page"), 0o644))

	watcher, err := NewFSNotifyWatcher(nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watcher.Close()) })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	events, err := watcher.Watch(ctx, []string{pages, assets, filepath.Join(root, "partials")}, nil)
	require.NoError(t, err)
	fsWatcher, ok := watcher.(*fsNotifyWatcher)
	require.True(t, ok)
	require.ElementsMatch(t, []string{pages, nested, assets}, fsWatcher.watcher.WatchList())

	require.NoError(t, os.Rename(oldPath, newPath))
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	var removed, created bool
	for !removed || !created {
		select {
		case event, ok := <-events:
			require.True(t, ok, "watcher closed before reporting the rename")
			removed = removed || event.Path == oldPath && event.Type == lifecycle_dto.FileEventTypeRemove
			created = created || event.Path == newPath && event.Type == lifecycle_dto.FileEventTypeCreate
		case <-timeout.C:
			t.Fatalf("rename events missing, removed %v, created %v", removed, created)
		}
	}
}

func TestFSNotifyWatcher_WatchReportsInstallationFailure(t *testing.T) {
	t.Parallel()

	watcher, err := NewFSNotifyWatcher(nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, watcher.Close()) })
	fsWatcher, ok := watcher.(*fsNotifyWatcher)
	require.True(t, ok)
	require.NoError(t, fsWatcher.watcher.Close())

	events, err := watcher.Watch(context.Background(), []string{t.TempDir()}, nil)
	require.ErrorIs(t, err, fsnotify.ErrClosed)
	assert.Nil(t, events)
	assert.Empty(t, fsWatcher.staticDirs)
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

func TestResolveEventType(t *testing.T) {
	t.Parallel()

	t.Run("non-rename op passes through unchanged", func(t *testing.T) {
		t.Parallel()

		watcher := &fsNotifyWatcher{}

		result := watcher.resolveEventType(fsnotify.Event{Name: "/path/to/page.pk", Op: fsnotify.Write})
		assert.Equal(t, lifecycle_dto.FileEventTypeWrite, result)
	})

	t.Run("rename of existing path resolves to create", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		existing := filepath.Join(dir, "new.pk")
		require.NoError(t, os.WriteFile(existing, []byte("page"), 0600))

		watcher := &fsNotifyWatcher{}

		result := watcher.resolveEventType(fsnotify.Event{Name: existing, Op: fsnotify.Rename})
		assert.Equal(t, lifecycle_dto.FileEventTypeCreate, result)
	})

	t.Run("rename of missing path resolves to remove", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		missing := filepath.Join(dir, "gone.pk")

		watcher := &fsNotifyWatcher{}

		result := watcher.resolveEventType(fsnotify.Event{Name: missing, Op: fsnotify.Rename})
		assert.Equal(t, lifecycle_dto.FileEventTypeRemove, result)
	})
}
