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
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewCompilationLogStore_DisabledMode(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)

	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.False(t, store.enabled)
	assert.Empty(t, store.logDir)
}

func TestNewCompilationLogStore_EnabledModeEmptyDir(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), true, "", slog.LevelInfo)

	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.True(t, store.enabled)
}

func TestNewCompilationLogStore_EnabledModeWithTempDir(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logDir := tempDir + "/logs"

	store, err := NewCompilationLogStore(context.Background(), true, logDir, slog.LevelDebug)

	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.True(t, store.enabled)
	assert.Equal(t, logDir, store.logDir)
	assert.Equal(t, slog.LevelDebug, store.minLogLevel)
}

func TestNewCompilationLogStore_InvalidDirectory(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), true, "/nonexistent/path/logs", slog.LevelInfo)

	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestStartSession_MemoryOnlyMode(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

	logger := store.StartSession(context.Background(), "/path/to/entry.pk", "components/button.pk")

	assert.NotNil(t, logger)

	_, found := store.GetLogs("/path/to/entry.pk")
	assert.True(t, found)
}

func TestStartSession_CreatesBuffer(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelDebug)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

	logs, found := store.GetLogs("/nonexistent/path.pk")

	assert.False(t, found)
	assert.Empty(t, logs)
}

func TestGetLogs_EmptyBuffer(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

	_ = store.StartSession(context.Background(), "/path/empty.pk", "empty.pk")

	logs, found := store.GetLogs("/path/empty.pk")

	assert.True(t, found)
	assert.Empty(t, logs)
}

func TestGetLogs_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), true, logDir, slog.LevelDebug)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), true, logDir, slog.LevelInfo)
	require.NoError(t, err)

	logger := store.StartSession(context.Background(), "/path/deep/nested/file.pk", "src/components/deep/nested/button.pk")
	assert.NotNil(t, logger)

	store.Shutdown(context.Background())
}

func TestStartSession_LogLevelFiltering(t *testing.T) {
	t.Parallel()

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelWarn)
	require.NoError(t, err)

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

	store, err := NewCompilationLogStore(context.Background(), false, "", slog.LevelInfo)
	require.NoError(t, err)

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

func TestNewCompilationLogStore_WithInjectedSandbox(t *testing.T) {
	t.Parallel()

	t.Run("creates log directory with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		store, err := NewCompilationLogStore(context.Background(), true, "/logs/compiler", slog.LevelDebug, WithLogStoreSandbox(sandbox))

		require.NoError(t, err)
		require.NotNil(t, store)

		info, statErr := sandbox.Stat("compiler")
		require.NoError(t, statErr)
		assert.True(t, info.IsDir())
	})

	t.Run("returns error when MkdirAll fails with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.MkdirAllErr = errors.New("disk full")

		store, err := NewCompilationLogStore(context.Background(), true, "/logs/compiler", slog.LevelDebug, WithLogStoreSandbox(sandbox))

		require.Error(t, err)
		assert.Nil(t, store)
		assert.Contains(t, err.Error(), "failed to create compiler debug log directory")
		assert.Contains(t, err.Error(), "disk full")
	})

	t.Run("skips directory creation when disabled with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		store, err := NewCompilationLogStore(context.Background(), false, "/logs/compiler", slog.LevelDebug, WithLogStoreSandbox(sandbox))

		require.NoError(t, err)
		require.NotNil(t, store)

		_, statErr := sandbox.Stat("compiler")
		assert.Error(t, statErr)
	})

	t.Run("skips directory creation when logDir is empty with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/logs", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		store, err := NewCompilationLogStore(context.Background(), true, "", slog.LevelDebug, WithLogStoreSandbox(sandbox))

		require.NoError(t, err)
		require.NotNil(t, store)
	})
}
