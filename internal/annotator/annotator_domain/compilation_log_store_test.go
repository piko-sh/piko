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

package annotator_domain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewCompilationLogStore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		enabled         bool
		logDir          string
		wantFileLogging bool
	}{
		{name: "disabled", enabled: false, logDir: "", wantFileLogging: false},
		{name: "enabled without a directory", enabled: true, logDir: "", wantFileLogging: false},
		{name: "enabled with a directory", enabled: true, logDir: "/nonexistent/path/logs", wantFileLogging: true},
		{name: "disabled with a directory", enabled: false, logDir: "/nonexistent/path/logs", wantFileLogging: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewCompilationLogStore(tc.enabled, tc.logDir, slog.LevelDebug)

			require.NotNil(t, store)
			assert.Equal(t, tc.wantFileLogging, store.fileLoggingEnabled.Load())
			assert.Equal(t, tc.logDir, store.logDir)
			assert.Equal(t, slog.LevelDebug, store.minLogLevel)
		})
	}
}

func TestNewCompilationLogStore_PerformsNoIO(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "missing", "logs")

	_ = NewCompilationLogStore(true, logDir, slog.LevelInfo)

	_, err := os.Stat(logDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPrepareLogDirectory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		setup           func(t *testing.T) (logDir string, opts []CompilationLogStoreOption)
		enabled         bool
		wantFileLogging bool
		wantDirectory   bool
	}{
		{
			name: "creates a missing nested directory for a fresh project",
			setup: func(t *testing.T) (string, []CompilationLogStoreOption) {
				t.Helper()
				return filepath.Join(t.TempDir(), ".piko", "nested", "logs"), nil
			},
			enabled:         true,
			wantFileLogging: true,
			wantDirectory:   true,
		},
		{
			name: "uses an injected sandbox factory",
			setup: func(t *testing.T) (string, []CompilationLogStoreOption) {
				t.Helper()
				root := t.TempDir()
				factory, err := safedisk.NewFactory(safedisk.FactoryConfig{CWD: root, AllowedPaths: []string{root}, Enabled: true})
				require.NoError(t, err)
				return filepath.Join(root, "logs"), []CompilationLogStoreOption{WithLogStoreSandboxFactory(factory)}
			},
			enabled:         true,
			wantFileLogging: true,
			wantDirectory:   true,
		},
		{
			name: "falls back to memory when the injected factory refuses the directory",
			setup: func(t *testing.T) (string, []CompilationLogStoreOption) {
				t.Helper()
				allowed := t.TempDir()
				factory, err := safedisk.NewFactory(safedisk.FactoryConfig{CWD: allowed, AllowedPaths: []string{allowed}, Enabled: true})
				require.NoError(t, err)
				return filepath.Join(t.TempDir(), "logs"), []CompilationLogStoreOption{WithLogStoreSandboxFactory(factory)}
			},
			enabled:         true,
			wantFileLogging: false,
			wantDirectory:   false,
		},
		{
			name: "falls back to memory when the directory cannot be created",
			setup: func(t *testing.T) (string, []CompilationLogStoreOption) {
				t.Helper()
				blocker := filepath.Join(t.TempDir(), "blocker")
				require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))
				return filepath.Join(blocker, "logs"), nil
			},
			enabled:         true,
			wantFileLogging: false,
			wantDirectory:   false,
		},
		{
			name: "does nothing when file logging is disabled",
			setup: func(t *testing.T) (string, []CompilationLogStoreOption) {
				t.Helper()
				return filepath.Join(t.TempDir(), "logs"), nil
			},
			enabled:         false,
			wantFileLogging: false,
			wantDirectory:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logDir, opts := tc.setup(t)
			store := NewCompilationLogStore(tc.enabled, logDir, slog.LevelInfo, opts...)

			store.PrepareLogDirectory(context.Background())

			assert.Equal(t, tc.wantFileLogging, store.fileLoggingEnabled.Load())
			info, err := os.Stat(logDir)
			if !tc.wantDirectory {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestPrepareLogDirectory_RunsOnce(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "logs")
	store := NewCompilationLogStore(true, logDir, slog.LevelInfo)

	store.PrepareLogDirectory(context.Background())
	require.NoError(t, os.RemoveAll(logDir))
	store.PrepareLogDirectory(context.Background())

	_, err := os.Stat(logDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.True(t, store.fileLoggingEnabled.Load())
}

func TestStartSession_MemoryOnlyMode(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logger := store.StartSession(context.Background(), "/path/to/entry.pk", "components/button.pk")

	assert.NotNil(t, logger)

	_, found := store.GetLogs("/path/to/entry.pk")
	assert.True(t, found)
}

func TestStartSession_CreatesBuffer(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelDebug)

	entryPoint := "/project/src/main.pk"
	logger := store.StartSession(context.Background(), entryPoint, "src/main.pk")

	assert.NotNil(t, logger)

	logger.Info("Test message")

	logs, found := store.GetLogs(entryPoint)
	assert.True(t, found)
	assert.Contains(t, logs, "Test message")
}

func TestStartSession_MultipleSessions(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logger1 := store.StartSession(context.Background(), "/path/file1.pk", "file1.pk")
	logger2 := store.StartSession(context.Background(), "/path/file2.pk", "file2.pk")
	logger3 := store.StartSession(context.Background(), "/path/file3.pk", "file3.pk")

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)
	assert.NotNil(t, logger3)

	logger1.Info("Message 1")
	logger2.Info("Message 2")
	logger3.Info("Message 3")

	logs1, found1 := store.GetLogs("/path/file1.pk")
	logs2, found2 := store.GetLogs("/path/file2.pk")
	logs3, found3 := store.GetLogs("/path/file3.pk")

	assert.True(t, found1)
	assert.True(t, found2)
	assert.True(t, found3)

	assert.Contains(t, logs1, "Message 1")
	assert.NotContains(t, logs1, "Message 2")
	assert.NotContains(t, logs1, "Message 3")

	assert.Contains(t, logs2, "Message 2")
	assert.NotContains(t, logs2, "Message 1")

	assert.Contains(t, logs3, "Message 3")
}

func TestGetLogs_NotFound(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logs, found := store.GetLogs("/nonexistent/path.pk")

	assert.False(t, found)
	assert.Empty(t, logs)
}

func TestGetLogs_EmptyBuffer(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	_ = store.StartSession(context.Background(), "/path/empty.pk", "empty.pk")

	logs, found := store.GetLogs("/path/empty.pk")

	assert.True(t, found)
	assert.Empty(t, logs)
}

func TestGetLogs_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	entryPoint := "/concurrent/test.pk"
	logger := store.StartSession(context.Background(), entryPoint, "test.pk")
	logger.Info("Initial message")

	done := make(chan bool, 10)
	for range 10 {
		go func() {
			logs, found := store.GetLogs(entryPoint)
			assert.True(t, found)
			assert.Contains(t, logs, "Initial message")
			done <- true
		}()
	}

	for range 10 {
		<-done
	}
}

func TestClear_RemovesAllBuffers(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logger1 := store.StartSession(context.Background(), "/path/file1.pk", "file1.pk")
	logger2 := store.StartSession(context.Background(), "/path/file2.pk", "file2.pk")
	logger1.Info("Log 1")
	logger2.Info("Log 2")

	_, found1 := store.GetLogs("/path/file1.pk")
	_, found2 := store.GetLogs("/path/file2.pk")
	assert.True(t, found1)
	assert.True(t, found2)

	store.Clear(context.Background())

	_, found1After := store.GetLogs("/path/file1.pk")
	_, found2After := store.GetLogs("/path/file2.pk")
	assert.False(t, found1After)
	assert.False(t, found2After)
}

func TestClear_AllowsNewSessions(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logger1 := store.StartSession(context.Background(), "/path/file.pk", "file.pk")
	logger1.Info("Old message")

	store.Clear(context.Background())

	logger2 := store.StartSession(context.Background(), "/path/file.pk", "file.pk")
	logger2.Info("New message")

	logs, found := store.GetLogs("/path/file.pk")
	assert.True(t, found)
	assert.Contains(t, logs, "New message")
	assert.NotContains(t, logs, "Old message")
}

func TestShutdown_ClearsClosers(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	_ = store.StartSession(context.Background(), "/path/file1.pk", "file1.pk")
	_ = store.StartSession(context.Background(), "/path/file2.pk", "file2.pk")

	store.Shutdown(context.Background())

	store.mu.RLock()
	closersLen := len(store.closers)
	store.mu.RUnlock()
	assert.Zero(t, closersLen)
}

func TestShutdown_PreservesBuffers(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	logger := store.StartSession(context.Background(), "/path/file.pk", "file.pk")
	logger.Info("Message before shutdown")

	store.Shutdown(context.Background())

	logs, found := store.GetLogs("/path/file.pk")
	assert.True(t, found)
	assert.Contains(t, logs, "Message before shutdown")
}

func TestStartSession_FileBasedLogging(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logDir := tempDir + "/logs"

	store := NewCompilationLogStore(true, logDir, slog.LevelDebug)

	logger := store.StartSession(context.Background(), "/path/component.pk", "components/button.pk")
	assert.NotNil(t, logger)

	logger.Info("File-based log message")

	logs, found := store.GetLogs("/path/component.pk")
	assert.True(t, found)
	assert.Contains(t, logs, "File-based log message")

	store.Shutdown(context.Background())
}

func TestStartSession_SanitisesFilename(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logDir := tempDir + "/logs"

	store := NewCompilationLogStore(true, logDir, slog.LevelInfo)

	logger := store.StartSession(context.Background(), "/path/deep/nested/file.pk", "src/components/deep/nested/button.pk")
	assert.NotNil(t, logger)

	store.Shutdown(context.Background())
}

func TestStartSession_LogLevelFiltering(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelWarn)

	logger := store.StartSession(context.Background(), "/path/file.pk", "file.pk")

	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")

	logs, found := store.GetLogs("/path/file.pk")
	assert.True(t, found)

	assert.NotContains(t, logs, "Debug message")
	assert.NotContains(t, logs, "Info message")

	assert.Contains(t, logs, "Warn message")
	assert.Contains(t, logs, "Error message")
}

func TestConcurrentStartSession(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)

	const numGoroutines = 20
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(index int) {
			entryPoint := "/path/file" + strings.Repeat("x", index) + ".pk"
			logger := store.StartSession(context.Background(), entryPoint, "file.pk")
			logger.Info("Message from goroutine")
			done <- true
		}(i)
	}

	for range numGoroutines {
		<-done
	}

	store.mu.RLock()
	bufferCount := len(store.buffers)
	store.mu.RUnlock()

	assert.Equal(t, numGoroutines, bufferCount)
}

func TestPrepareLogDirectory_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		enabled         bool
		logDir          string
		mkdirErr        error
		wantFileLogging bool
		wantDirectory   bool
	}{
		{name: "creates the directory", enabled: true, logDir: "/logs/compiler", wantFileLogging: true, wantDirectory: true},
		{
			name:            "falls back to memory when MkdirAll fails",
			enabled:         true,
			logDir:          "/logs/compiler",
			mkdirErr:        errors.New("disk full"),
			wantFileLogging: false,
		},
		{name: "skips directory creation when disabled", enabled: false, logDir: "/logs/compiler"},
		{name: "skips directory creation when the directory is empty", enabled: true, logDir: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
			defer func() { _ = sandbox.Close() }()
			sandbox.MkdirAllErr = tc.mkdirErr

			store := NewCompilationLogStore(tc.enabled, tc.logDir, slog.LevelDebug, WithLogStoreSandbox(sandbox))
			store.PrepareLogDirectory(context.Background())

			assert.Equal(t, tc.wantFileLogging, store.fileLoggingEnabled.Load())
			info, statErr := sandbox.Stat("compiler")
			if !tc.wantDirectory {
				assert.Error(t, statErr)
				return
			}
			require.NoError(t, statErr)
			assert.True(t, info.IsDir())
		})
	}
}

func TestGetLogs_ReadsWhileSessionWrites(t *testing.T) {
	t.Parallel()

	store := NewCompilationLogStore(false, "", slog.LevelInfo)
	entryPoint := "/project/busy.pk"
	logger := store.StartSession(context.Background(), entryPoint, "busy.pk")

	const messageCount = 200
	var wg sync.WaitGroup
	wg.Go(func() {
		for index := range messageCount {
			logger.Info("busy message", logger_domain.Int("index", index))
		}
	})
	wg.Go(func() {
		for range messageCount {
			_, found := store.GetLogs(entryPoint)
			assert.True(t, found)
		}
	})
	wg.Wait()

	logs, found := store.GetLogs(entryPoint)
	require.True(t, found)
	assert.Equal(t, messageCount, strings.Count(logs, "busy message"))
}

type recordingCloser struct {
	err    error
	closed bool
}

func (c *recordingCloser) Close() error {
	c.closed = true
	return c.err
}

func TestSessionLogFileName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		relativePath string
		want         string
	}{
		{name: "escapes directory separators", relativePath: "components/button.pk", want: "components%2Fbutton.pk.log"},
		{name: "keeps underscores", relativePath: "a/b_c.pk", want: "a%2Fb_c.pk.log"},
		{name: "keeps a flat name readable", relativePath: "page.pk", want: "page.pk.log"},
		{name: "escapes characters that are unsafe in file names", relativePath: "a b:c?.pk", want: "a+b%3Ac%3F.pk.log"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, sessionLogFileName(tc.relativePath))
		})
	}
}

func TestSessionLogFileName_DistinctPathsGiveDistinctNames(t *testing.T) {
	t.Parallel()

	paths := []string{
		"a/b_c.pk",
		"a_b/c.pk",
		"a_b_c.pk",
		"a/_b.pk",
		"a_/b.pk",
		"a%2Fb.pk",
		"a/b.pk",
		"a b.pk",
		"a+b.pk",
	}

	seen := make(map[string]string, len(paths))
	for _, path := range paths {
		name := sessionLogFileName(path)
		previous, exists := seen[name]
		assert.False(t, exists, "%q and %q both map to %q", previous, path, name)
		seen[name] = path
	}
}

func TestStartSession_WritesSeparateFilesForCollidingFlatNames(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "logs")
	store := NewCompilationLogStore(true, logDir, slog.LevelInfo)
	store.PrepareLogDirectory(context.Background())
	require.True(t, store.fileLoggingEnabled.Load())

	first := store.StartSession(context.Background(), "/project/a/b_c.pk", "a/b_c.pk")
	second := store.StartSession(context.Background(), "/project/a_b/c.pk", "a_b/c.pk")
	first.Info("first component message")
	second.Info("second component message")

	store.Shutdown(context.Background())

	firstContent, err := os.ReadFile(filepath.Join(logDir, sessionLogFileName("a/b_c.pk")))
	require.NoError(t, err)
	secondContent, err := os.ReadFile(filepath.Join(logDir, sessionLogFileName("a_b/c.pk")))
	require.NoError(t, err)

	assert.Contains(t, string(firstContent), "first component message")
	assert.NotContains(t, string(firstContent), "second component message")
	assert.Contains(t, string(secondContent), "second component message")
	assert.NotContains(t, string(secondContent), "first component message")
}

func TestStartSession_FallsBackToMemoryWhenLogFileUnavailable(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "logs")
	store := NewCompilationLogStore(true, logDir, slog.LevelInfo)
	store.PrepareLogDirectory(context.Background())
	require.True(t, store.fileLoggingEnabled.Load())
	require.NoError(t, os.RemoveAll(logDir))
	require.NoError(t, os.WriteFile(logDir, []byte("not a directory"), 0o600))

	first := store.StartSession(context.Background(), "/project/first.pk", "first.pk")
	second := store.StartSession(context.Background(), "/project/second.pk", "second.pk")
	first.Info("kept in memory")
	second.Info("also kept in memory")

	firstLogs, found := store.GetLogs("/project/first.pk")
	require.True(t, found)
	assert.Contains(t, firstLogs, "kept in memory")
	secondLogs, found := store.GetLogs("/project/second.pk")
	require.True(t, found)
	assert.Contains(t, secondLogs, "also kept in memory")

	store.mu.RLock()
	closerCount := len(store.closers)
	store.mu.RUnlock()
	assert.Zero(t, closerCount)
	assert.True(t, store.fileFailureReported.Load())

	store.Clear(context.Background())
	assert.False(t, store.fileFailureReported.Load())
}

func TestClearAndShutdown_CloseEveryWriter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		release func(*CompilationLogStore)
	}{
		{name: "clear", release: func(store *CompilationLogStore) { store.Clear(context.Background()) }},
		{name: "shutdown", release: func(store *CompilationLogStore) { store.Shutdown(context.Background()) }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := NewCompilationLogStore(false, "", slog.LevelInfo)

			failing := &recordingCloser{err: errors.New("disk detached")}
			succeeding := &recordingCloser{}
			store.closers = append(store.closers, failing, succeeding)

			tc.release(store)

			assert.True(t, failing.closed)
			assert.True(t, succeeding.closed)
			store.mu.RLock()
			closerCount := len(store.closers)
			store.mu.RUnlock()
			assert.Zero(t, closerCount)
		})
	}
}

func TestCloseLogWriters(t *testing.T) {
	t.Parallel()

	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")

	testCases := []struct {
		name     string
		closers  []*recordingCloser
		wantErrs []error
	}{
		{name: "no writers", closers: nil},
		{name: "all succeed", closers: []*recordingCloser{{}, {}}},
		{
			name:     "joins every failure",
			closers:  []*recordingCloser{{err: firstErr}, {}, {err: secondErr}},
			wantErrs: []error{firstErr, secondErr},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			closers := make([]io.Closer, 0, len(tc.closers))
			for _, closer := range tc.closers {
				closers = append(closers, closer)
			}

			err := closeLogWriters(closers)

			for _, closer := range tc.closers {
				assert.True(t, closer.closed)
			}
			if len(tc.wantErrs) == 0 {
				assert.NoError(t, err)
				return
			}
			for _, wantErr := range tc.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
		})
	}
}

func TestConcurrentFileSessions(t *testing.T) {
	t.Parallel()

	logDir := filepath.Join(t.TempDir(), "logs")
	store := NewCompilationLogStore(true, logDir, slog.LevelInfo)

	const sessionCount = 16
	var wg sync.WaitGroup
	for index := range sessionCount {
		wg.Go(func() {
			relativePath := fmt.Sprintf("components/c%d.pk", index)
			logger := store.StartSession(context.Background(), "/project/"+relativePath, relativePath)
			logger.Info("concurrent message")
		})
	}
	wg.Wait()

	store.mu.RLock()
	bufferCount := len(store.buffers)
	closerCount := len(store.closers)
	store.mu.RUnlock()
	assert.Equal(t, sessionCount, bufferCount)
	assert.Equal(t, sessionCount, closerCount)

	store.Shutdown(context.Background())

	entries, err := os.ReadDir(logDir)
	require.NoError(t, err)
	assert.Len(t, entries, sessionCount)
}
