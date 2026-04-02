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

package logrotate

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func newTestWriter(t *testing.T, config Config) *Writer {
	t.Helper()
	if config.Filename == "" {
		config.Filename = "test.log"
	}
	if config.Directory == "" && config.Sandbox == nil {
		config.Directory = t.TempDir()
	}
	writer, err := New(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(func() { writer.Close() })
	return writer
}

func TestNew_ValidConfig(t *testing.T) {
	t.Parallel()

	writer, err := New(context.Background(), Config{
		Directory: t.TempDir(),
		Filename:  "app.log",
		MaxSize:   1,
	})
	require.NoError(t, err)
	defer writer.Close()

	assert.NotNil(t, writer.sandbox)
	assert.True(t, writer.ownsSandbox)
}

func TestNew_EmptyFilename(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{Directory: t.TempDir()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "filename must not be empty")
}

func TestNew_EmptyDirectory(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{Filename: "app.log"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory must not be empty")
}

func TestNew_NegativeValues(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	_, err := New(context.Background(), Config{Directory: directory, Filename: "a.log", MaxSize: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max size must not be negative")

	_, err = New(context.Background(), Config{Directory: directory, Filename: "b.log", MaxBackups: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max backups must not be negative")

	_, err = New(context.Background(), Config{Directory: directory, Filename: "c.log", MaxAge: -1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max age must not be negative")
}

func TestNew_DefaultMaxSize(t *testing.T) {
	t.Parallel()

	writer := newTestWriter(t, Config{})
	assert.Equal(t, int64(defaultMaxSizeMB)*megabyte, writer.maxSizeBytes)
}

func TestWrite_CreatesFileOnFirstWrite(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writer := newTestWriter(t, Config{Directory: directory, Filename: "app.log"})

	_, err := writer.Write([]byte("hello\n"))
	require.NoError(t, err)

	_, statError := os.Stat(filepath.Join(directory, "app.log"))
	assert.NoError(t, statError)
}

func TestWrite_AppendsToExistingFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writer := newTestWriter(t, Config{Directory: directory, Filename: "app.log"})

	_, err := writer.Write([]byte("line one\n"))
	require.NoError(t, err)

	_, err = writer.Write([]byte("line two\n"))
	require.NoError(t, err)

	content, readError := os.ReadFile(filepath.Join(directory, "app.log"))
	require.NoError(t, readError)
	assert.Equal(t, "line one\nline two\n", string(content))
}

func TestWrite_RotatesWhenSizeExceeded(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 3, 26, 14, 30, 45, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	for i := range payload {
		payload[i] = 'A'
	}

	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("after rotation\n"))
	require.NoError(t, err)

	_, statError := os.Stat(filepath.Join(directory, "app-20260326T143045.log"))
	assert.NoError(t, statError)

	currentContent, readError := os.ReadFile(filepath.Join(directory, "app.log"))
	require.NoError(t, readError)
	assert.Equal(t, "after rotation\n", string(currentContent))
}

func TestWrite_OversizedSingleWrite(t *testing.T) {
	t.Parallel()

	writer := newTestWriter(t, Config{
		Filename: "app.log",
		MaxSize:  1,
	})

	oversizedPayload := make([]byte, 2*megabyte)
	for i := range oversizedPayload {
		oversizedPayload[i] = 'X'
	}

	bytesWritten, err := writer.Write(oversizedPayload)
	require.NoError(t, err)
	assert.Equal(t, len(oversizedPayload), bytesWritten)
}

func TestWrite_OnClosedWriter(t *testing.T) {
	t.Parallel()

	writer, err := New(context.Background(), Config{
		Directory: t.TempDir(),
		Filename:  "app.log",
	})
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	_, writeError := writer.Write([]byte("data"))
	require.ErrorIs(t, writeError, ErrClosed)
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	writer, err := New(context.Background(), Config{
		Directory: t.TempDir(),
		Filename:  "app.log",
	})
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	require.NoError(t, writer.Close())
}

func TestRotation_BackupFilenameFormat(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 1, 15, 8, 5, 3, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "server.log",
		MaxSize:   1,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger rotation\n"))
	require.NoError(t, err)

	_, statError := os.Stat(filepath.Join(directory, "server-20260115T080503.log"))
	assert.NoError(t, statError)
}

func TestRotation_BackupTimestampUTC(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("x"))
	require.NoError(t, err)

	_, statError := os.Stat(filepath.Join(directory, "app-20260615T170000.log"))
	assert.NoError(t, statError)
}

func TestRotation_BackupTimestampLocal(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		LocalTime: true,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("x"))
	require.NoError(t, err)

	localTimestamp := fixedTime.In(time.Local).Format(backupTimestampFormat)
	_, statError := os.Stat(filepath.Join(directory, "app-"+localTimestamp+".log"))
	assert.NoError(t, statError)
}

func TestCleanup_MaxBackups(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	writer := newTestWriter(t, Config{
		Directory:  directory,
		Filename:   "app.log",
		MaxSize:    1,
		MaxBackups: 2,
		Clock:      mockClock,
	})

	payload := make([]byte, megabyte+1)
	for i := range 4 {
		mockClock.Set(time.Date(2026, 1, 1, 0, 0, i+1, 0, time.UTC))
		_, err := writer.Write(payload)
		require.NoError(t, err)
	}

	writer.Close()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)

	var backupCount int
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "app-") {
			backupCount++
		}
	}
	assert.LessOrEqual(t, backupCount, 2)
}

func TestCleanup_MaxAge(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	oldTimestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	oldBackupName := "app-" + oldTimestamp.Format(backupTimestampFormat) + ".log"
	require.NoError(t, os.WriteFile(filepath.Join(directory, oldBackupName), []byte("old data"), 0o640))

	currentTime := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		MaxAge:    7,
		Clock:     clock.NewMockClock(currentTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger"))
	require.NoError(t, err)

	writer.Close()

	_, statError := os.Stat(filepath.Join(directory, oldBackupName))
	assert.True(t, os.IsNotExist(statError))
}

func TestCleanup_Compression(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Compress:  true,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("after rotation"))
	require.NoError(t, err)

	writer.Close()

	compressedData, readError := os.ReadFile(filepath.Join(directory, "app-20260326T100000.log.gz"))
	require.NoError(t, readError)

	gzipReader, gzipError := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, gzipError)
	defer gzipReader.Close()

	decompressed, readAllError := io.ReadAll(gzipReader)
	require.NoError(t, readAllError)
	assert.Len(t, decompressed, megabyte+1)
}

func TestCleanup_CompressionDisabled(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	fixedTime := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Compress:  false,
		Clock:     clock.NewMockClock(fixedTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger"))
	require.NoError(t, err)

	writer.Close()

	backupPath := filepath.Join(directory, "app-20260326T100000.log")
	_, statError := os.Stat(backupPath)
	assert.NoError(t, statError)

	_, statError = os.Stat(backupPath + ".gz")
	assert.True(t, os.IsNotExist(statError))
}

func TestCleanup_AtomicCompression(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Compress:  true,
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger"))
	require.NoError(t, err)

	writer.Close()

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), temporarySuffix),
			"temporary file should not remain: %s", entry.Name())
	}
}

func TestConcurrentWrites(t *testing.T) {
	t.Parallel()

	writer := newTestWriter(t, Config{
		Filename: "app.log",
		MaxSize:  1,
	})

	var waitGroup sync.WaitGroup
	goroutineCount := 10
	writesPerGoroutine := 100

	waitGroup.Add(goroutineCount)
	for range goroutineCount {
		go func() {
			defer waitGroup.Done()
			for range writesPerGoroutine {
				_, err := writer.Write([]byte("concurrent log line\n"))
				assert.NoError(t, err)
			}
		}()
	}
	waitGroup.Wait()
}

func TestClockInjection(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	specificTime := time.Date(2030, 12, 25, 23, 59, 59, 0, time.UTC)

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Clock:     clock.NewMockClock(specificTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger"))
	require.NoError(t, err)

	_, statError := os.Stat(filepath.Join(directory, "app-20301225T235959.log"))
	assert.NoError(t, statError)
}

func TestWrite_AppendsToExistingFileOnReopen(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "app.log"), []byte("existing content\n"), 0o640))

	writer := newTestWriter(t, Config{Directory: directory, Filename: "app.log"})

	_, err := writer.Write([]byte("new content\n"))
	require.NoError(t, err)

	writer.Close()

	content, readError := os.ReadFile(filepath.Join(directory, "app.log"))
	require.NoError(t, readError)
	assert.Equal(t, "existing content\nnew content\n", string(content))
}

func TestWrite_OpenFileError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.OpenFileErr = errors.New("disk full")

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
	})
	require.NoError(t, err)
	defer writer.Close()

	_, writeError := writer.Write([]byte("hello"))
	require.Error(t, writeError)
	assert.Contains(t, writeError.Error(), "disk full")
}

func TestWrite_WriteError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
	})
	require.NoError(t, err)
	defer writer.Close()

	_, firstWriteError := writer.Write([]byte("hello"))
	require.NoError(t, firstWriteError)

	mockHandle, ok := writer.file.(*safedisk.MockFileHandle)
	require.True(t, ok)
	mockHandle.WriteErr = errors.New("io error")

	_, writeError := writer.Write([]byte("world"))
	require.Error(t, writeError)
	assert.Contains(t, writeError.Error(), "io error")
}

func TestRotation_CloseError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	writer.file.Close()

	_, writeError := writer.Write([]byte("trigger rotation"))
	require.Error(t, writeError)
	assert.Contains(t, writeError.Error(), "close file before rotation")
}

func TestRotation_RenameError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", make([]byte, 2*megabyte))
	mock.RenameErr = errors.New("rename failed")

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
		MaxSize:  1,
	})
	require.NoError(t, err)
	defer writer.Close()

	_, writeError := writer.Write([]byte("trigger"))
	require.Error(t, writeError)
	assert.Contains(t, writeError.Error(), "rename")
}

func TestRotation_CreateAfterRotateError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(directory, 0o444))
	t.Cleanup(func() { os.Chmod(directory, 0o755) })

	_, writeError := writer.Write([]byte("trigger rotation"))
	require.Error(t, writeError)
}

func TestCleanup_ContextCancellation(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	writer, err := New(ctx, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		Compress:  true,
	})
	require.NoError(t, err)

	payload := make([]byte, megabyte+1)
	_, writeError := writer.Write(payload)
	require.NoError(t, writeError)

	cancel()

	_, writeError = writer.Write([]byte("second write to trigger rotation"))
	require.NoError(t, writeError)

	writer.Close()
}

func TestCleanup_RemoveError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	currentTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	oldBackupName := "app-20251201T000000.log"
	require.NoError(t, os.WriteFile(filepath.Join(directory, oldBackupName), []byte("old"), 0o640))
	require.NoError(t, os.Chmod(filepath.Join(directory, oldBackupName), 0o000))
	t.Cleanup(func() { os.Chmod(filepath.Join(directory, oldBackupName), 0o644) })

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxSize:   1,
		MaxAge:    1,
		Clock:     clock.NewMockClock(currentTime),
	})

	payload := make([]byte, megabyte+1)
	_, err := writer.Write(payload)
	require.NoError(t, err)

	_, err = writer.Write([]byte("trigger"))
	require.NoError(t, err)

	writer.Close()
}

func TestCleanup_ReadDirError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)

	writer, err := New(context.Background(), Config{
		Filename:   "app.log",
		Sandbox:    mock,
		MaxSize:    1,
		MaxBackups: 1,
	})
	require.NoError(t, err)

	mock.ReadDirErr = errors.New("permission denied")

	writer.performCleanup()
	writer.Close()
}

func TestCleanup_CompressReadDirError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
		MaxSize:  1,
		Compress: true,
	})
	require.NoError(t, err)

	mock.ReadDirErr = errors.New("permission denied")

	writer.compressBackups()
	writer.Close()
}

func TestCompress_OpenSourceError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
		Compress: true,
	})
	require.NoError(t, err)
	defer writer.Close()

	mock.OpenErr = errors.New("open failed")
	compressError := writer.compressFile("backup.log")
	require.Error(t, compressError)
	assert.Contains(t, compressError.Error(), "opening source")
}

func TestCompress_CreateDestinationError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)
	mock.AddFile("backup.log", []byte("data"))

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
		Compress: true,
	})
	require.NoError(t, err)
	defer writer.Close()

	mock.OpenFileErr = errors.New("cannot create")
	compressError := writer.compressFile("backup.log")
	require.Error(t, compressError)
	assert.Contains(t, compressError.Error(), "creating temp file")
}

func TestCompress_RenameError(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	backupPath := filepath.Join(directory, "app-20260101T000000.log")
	require.NoError(t, os.WriteFile(backupPath, []byte("backup data"), 0o640))

	writer := newTestWriter(t, Config{
		Directory: directory,
		Filename:  "app.log",
		Compress:  true,
	})

	require.NoError(t, os.Chmod(directory, 0o555))
	t.Cleanup(func() { os.Chmod(directory, 0o755) })

	compressError := writer.compressFile("app-20260101T000000.log")
	require.Error(t, compressError)
}

func TestClose_SandboxCloseError(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)
	mock.CloseErr = errors.New("sandbox close failed")

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  mock,
	})
	require.NoError(t, err)

	assert.False(t, writer.ownsSandbox)
	closeError := writer.Close()
	assert.NoError(t, closeError)
}

func TestNew_FactoryError(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Config{
		Directory: "/nonexistent/path/that/should/fail/\x00invalid",
		Filename:  "app.log",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox")
}

func TestPerformCleanup_ContextCancelledBeforeCompress(t *testing.T) {
	t.Parallel()

	mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
	mock.AddFile("app.log", nil)

	ctx, cancel := context.WithCancel(context.Background())
	writer, err := New(ctx, Config{
		Filename:   "app.log",
		Sandbox:    mock,
		Compress:   true,
		MaxBackups: 1,
	})
	require.NoError(t, err)

	cancel()

	writer.performCleanup()
	writer.Close()
}

func TestPerformCleanup_ContextCancelledBeforeRemove(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	currentTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	oldBackupName := "app-20260101T000000.log"
	require.NoError(t, os.WriteFile(filepath.Join(directory, oldBackupName), []byte("old"), 0o640))

	ctx, cancel := context.WithCancel(context.Background())
	writer, err := New(ctx, Config{
		Directory: directory,
		Filename:  "app.log",
		MaxAge:    1,
		Clock:     clock.NewMockClock(currentTime),
	})
	require.NoError(t, err)

	cancel()

	writer.performCleanup()
	writer.Close()

	_, statError := os.Stat(filepath.Join(directory, oldBackupName))
	assert.NoError(t, statError)
}

func TestCompressBackups_ContextCancelledMidLoop(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "app-20260101T000000.log"), []byte("a"), 0o640))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "app-20260102T000000.log"), []byte("b"), 0o640))

	ctx, cancel := context.WithCancel(context.Background())
	writer, err := New(ctx, Config{
		Directory: directory,
		Filename:  "app.log",
		Compress:  true,
	})
	require.NoError(t, err)

	cancel()
	writer.compressBackups()
	writer.Close()
}

func TestProvidedSandboxNotClosed(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	defer sandbox.Close()

	writer, err := New(context.Background(), Config{
		Filename: "app.log",
		Sandbox:  sandbox,
	})
	require.NoError(t, err)

	_, writeError := writer.Write([]byte("test\n"))
	require.NoError(t, writeError)

	require.NoError(t, writer.Close())

	_, statError := sandbox.Stat("app.log")
	assert.NoError(t, statError)
}
