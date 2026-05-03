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
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/gzip"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// backupTimestampFormat is the time layout used for rotated backup filenames.
	backupTimestampFormat = "20060102T150405"

	// compressedSuffix is the file extension appended to gzip-compressed backups.
	compressedSuffix = ".gz"

	// temporarySuffix is the file extension used for in-progress compressed files.
	temporarySuffix = ".tmp"

	// defaultMaxSizeMB is the default maximum log file size in megabytes.
	defaultMaxSizeMB = 100

	// megabyte is the number of bytes in one megabyte.
	megabyte = 1024 * 1024

	// filePermissions is the permission mode used when creating log files.
	filePermissions fs.FileMode = 0o640
)

var (
	// ErrClosed is returned by [Writer.Write] when the writer has already
	// been closed.
	ErrClosed = errors.New("logrotate: writer is closed")

	// ErrInvalidConfig is returned by [New] when the configuration fails
	// validation. Use [errors.Is] to check for this sentinel; the returned
	// error wraps it with a descriptive message.
	ErrInvalidConfig = errors.New("logrotate: invalid configuration")
)

// Config holds the settings for a rotating log file writer.
type Config struct {
	// Sandbox is the filesystem sandbox used for all file operations. When
	// nil a sandbox is created from Directory.
	Sandbox safedisk.Sandbox

	// Clock provides the current time.
	//
	// When nil [clock.RealClock] is used. Use [clock.NewMockClock] in tests
	// to control timestamps.
	Clock clock.Clock

	// Directory is the directory where the log file and its rotated backups
	// are stored.
	//
	// When Sandbox is nil this must be set and a real sandbox is created from
	// it. When Sandbox is provided the value is ignored.
	Directory string

	// Filename is the base name of the log file (e.g. "app.log"), which must
	// not contain path separators.
	Filename string

	// MaxSize is the maximum size of the log file in megabytes before it is
	// rotated. Zero uses the default of 100 MB.
	MaxSize int

	// MaxBackups is the maximum number of old log files to retain. Zero
	// means keep all backups.
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files based on
	// the timestamp encoded in their filename. Zero means no age limit.
	MaxAge int

	// Compress controls whether rotated log files are compressed with gzip.
	Compress bool

	// LocalTime determines whether backup timestamps use local time instead
	// of UTC. The default is UTC.
	LocalTime bool
}

// Writer is a thread-safe [io.WriteCloser] that automatically rotates the
// underlying log file when it exceeds the configured size.
//
// Rotated files are named by appending a timestamp to the original filename.
// A background goroutine compresses rotated files and removes old backups
// according to MaxBackups and MaxAge. The goroutine respects context
// cancellation and exits promptly during shutdown.
type Writer struct {
	// sandbox holds the filesystem sandbox for all file operations.
	sandbox safedisk.Sandbox

	// file holds the currently open log file handle.
	file safedisk.FileHandle

	// ctx controls the lifetime of the background cleanup goroutine.
	ctx context.Context

	// done is closed when the background cleanup goroutine exits.
	done chan struct{}

	// clock provides the current time for backup timestamps.
	clock clock.Clock

	// cancel cancels the writer's internal context.
	cancel context.CancelCauseFunc

	// cleanupSignal notifies the background goroutine to run cleanup.
	cleanupSignal chan struct{}

	// filename is the base name of the log file.
	filename string

	// prefix is the filename without its extension.
	prefix string

	// extension is the file extension including the leading dot.
	extension string

	// config holds the rotation settings.
	config Config

	// maxSizeBytes is the maximum log file size in bytes before rotation.
	maxSizeBytes int64

	// currentSize tracks the current log file size in bytes.
	currentSize int64

	// closeOnce ensures Close executes its body at most once.
	closeOnce sync.Once

	// mu serialises writes and file operations.
	mu sync.Mutex

	// closed indicates whether the writer has been closed.
	closed atomic.Bool

	// ownsSandbox indicates whether Close should release the sandbox.
	ownsSandbox bool
}

var _ io.WriteCloser = (*Writer)(nil)

// New creates a new rotating log file writer with the given configuration.
//
// The provided context controls the lifetime of the background cleanup
// goroutine; cancelling it causes the goroutine to stop accepting new work.
// Call [Writer.Close] for deterministic shutdown regardless of context state.
//
// Takes ctx ([context.Context]) which controls the background goroutine
// lifetime.
// Takes config ([Config]) which specifies the rotation settings.
//
// Returns *Writer which is ready for use.
// Returns error which wraps [ErrInvalidConfig] when the
// configuration is invalid, or a filesystem error when the
// sandbox cannot be created.
//
// Concurrency: the returned Writer is safe for concurrent
// use. Writes are serialised by an internal mutex. A
// background goroutine handles compression and old backup
// removal; it exits when the context is cancelled or Close
// is called.
func New(ctx context.Context, config Config) (*Writer, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	maxSizeMB := config.MaxSize
	if maxSizeMB == 0 {
		maxSizeMB = defaultMaxSizeMB
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	sandbox, ownsSandbox, sandboxError := resolveSandbox(config)
	if sandboxError != nil {
		return nil, sandboxError
	}

	writerContext, writerCancel := context.WithCancelCause(ctx)

	extension := filepath.Ext(config.Filename)
	prefix := strings.TrimSuffix(config.Filename, extension)

	writer := &Writer{
		filename:      config.Filename,
		prefix:        prefix,
		extension:     extension,
		maxSizeBytes:  int64(maxSizeMB) * megabyte,
		config:        config,
		clock:         clk,
		sandbox:       sandbox,
		ownsSandbox:   ownsSandbox,
		ctx:           writerContext,
		cancel:        writerCancel,
		cleanupSignal: make(chan struct{}, 1),
		done:          make(chan struct{}),
	}

	go writer.runCleanup()

	return writer, nil
}

// Write writes the payload to the current log file, rotating first if the
// write would exceed the configured maximum size.
//
// Takes payload ([]byte) which contains the data to write.
//
// Returns int which is the number of bytes written.
// Returns error which wraps [ErrClosed] when the writer
// has been closed, or a filesystem error when writing
// fails.
//
// Safe for concurrent use by multiple goroutines.
func (w *Writer) Write(payload []byte) (int, error) {
	if w.closed.Load() {
		return 0, ErrClosed
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.openExistingOrCreate(); err != nil {
			return 0, err
		}
	}

	if w.currentSize > 0 && w.currentSize+int64(len(payload)) > w.maxSizeBytes {
		if err := w.rotateFile(); err != nil {
			return 0, err
		}
	}

	bytesWritten, writeError := w.file.Write(payload)
	w.currentSize += int64(bytesWritten)
	return bytesWritten, writeError
}

// Close flushes and closes the current log file, stops the background
// cleanup goroutine, and releases the sandbox if it was created internally.
//
// Close is idempotent; subsequent calls return nil.
//
// Returns error when closing the underlying file fails.
func (w *Writer) Close() error {
	var fileError error
	w.closeOnce.Do(func() {
		w.closed.Store(true)

		w.mu.Lock()
		if w.file != nil {
			fileError = w.file.Close()
			w.file = nil
		}
		w.mu.Unlock()

		close(w.cleanupSignal)
		<-w.done
		w.cancel(errors.New("logrotate writer shutdown"))

		if w.ownsSandbox {
			if sandboxError := w.sandbox.Close(); sandboxError != nil && fileError == nil {
				fileError = sandboxError
			}
		}
	})
	return fileError
}

// validateConfig checks that the configuration is valid.
//
// Takes config ([Config]) which holds the settings to validate.
//
// Returns error wrapping [ErrInvalidConfig] when validation fails.
func validateConfig(config Config) error {
	if config.Filename == "" {
		return fmt.Errorf("%w: filename must not be empty", ErrInvalidConfig)
	}
	if config.Sandbox == nil && config.Directory == "" {
		return fmt.Errorf("%w: directory must not be empty when sandbox is nil", ErrInvalidConfig)
	}
	if config.MaxSize < 0 {
		return fmt.Errorf("%w: max size must not be negative, got %d", ErrInvalidConfig, config.MaxSize)
	}
	if config.MaxBackups < 0 {
		return fmt.Errorf("%w: max backups must not be negative, got %d", ErrInvalidConfig, config.MaxBackups)
	}
	if config.MaxAge < 0 {
		return fmt.Errorf("%w: max age must not be negative, got %d", ErrInvalidConfig, config.MaxAge)
	}
	return nil
}

// resolveSandbox creates or reuses the filesystem sandbox.
//
// Takes config ([Config]) which provides the sandbox or directory to use.
//
// Returns safedisk.Sandbox which is the sandbox to use.
// Returns bool which indicates whether the caller owns the sandbox.
// Returns error when sandbox creation fails.
func resolveSandbox(config Config) (safedisk.Sandbox, bool, error) {
	if config.Sandbox != nil {
		return config.Sandbox, false, nil
	}

	factory, factoryError := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      true,
		AllowedPaths: []string{config.Directory},
	})
	if factoryError != nil {
		return nil, false, fmt.Errorf("logrotate: failed to create sandbox factory for %q: %w", config.Directory, factoryError)
	}

	sandbox, sandboxError := factory.Create("logrotate", config.Directory, safedisk.ModeReadWrite)
	if sandboxError != nil {
		return nil, false, fmt.Errorf("logrotate: failed to create sandbox for %q: %w", config.Directory, sandboxError)
	}

	return sandbox, true, nil
}

// openExistingOrCreate opens the log file for appending if it already exists
// and is within the size limit, otherwise creates a new file.
//
// Returns error when the file cannot be opened or created.
func (w *Writer) openExistingOrCreate() error {
	info, statError := w.sandbox.Stat(w.filename)
	if statError == nil && info.Size() < w.maxSizeBytes {
		handle, openError := w.sandbox.OpenFile(
			w.filename, os.O_WRONLY|os.O_APPEND, filePermissions,
		)
		if openError == nil {
			w.file = handle
			w.currentSize = info.Size()
			return nil
		}
	}

	if statError == nil {
		if err := w.rotateExistingFile(); err != nil {
			return fmt.Errorf("rotating existing file on open: %w", err)
		}
	}

	return w.createFreshFile()
}

// rotateExistingFile renames the existing log file to a timestamped backup
// name and signals the background cleanup goroutine.
//
// Returns error when the rename fails.
func (w *Writer) rotateExistingFile() error {
	backupName := w.backupFilename()
	if err := w.sandbox.Rename(w.filename, backupName); err != nil {
		return fmt.Errorf("logrotate: failed to rename %q to %q: %w", w.filename, backupName, err)
	}
	w.signalCleanup()
	return nil
}

// createFreshFile creates a new empty log file.
//
// Returns error when the file cannot be created.
func (w *Writer) createFreshFile() error {
	handle, createError := w.sandbox.OpenFile(
		w.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions,
	)
	if createError != nil {
		return fmt.Errorf("logrotate: failed to create %q: %w", w.filename, createError)
	}
	w.file = handle
	w.currentSize = 0
	return nil
}

// rotateFile closes the current log file, renames it to a backup, and opens
// a fresh file.
//
// Returns error when any step fails.
func (w *Writer) rotateFile() error {
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("logrotate: failed to close file before rotation: %w", err)
	}
	w.file = nil

	if err := w.rotateExistingFile(); err != nil {
		return fmt.Errorf("rotating existing file: %w", err)
	}

	return w.createFreshFile()
}

// backupFilename generates a timestamped backup filename from the current
// time and the configured location.
//
// Returns string which is the backup filename relative to the sandbox root.
func (w *Writer) backupFilename() string {
	timestamp := w.clock.Now().In(w.location()).Format(backupTimestampFormat)
	return w.prefix + "-" + timestamp + w.extension
}

// signalCleanup sends a non-blocking signal to the background cleanup
// goroutine. If the goroutine is already processing a cleanup the signal
// is dropped.
func (w *Writer) signalCleanup() {
	select {
	case w.cleanupSignal <- struct{}{}:
	default:
	}
}

// runCleanup is the background goroutine that processes
// cleanup signals, exiting when the writer's context is
// cancelled or the channel is closed.
func (w *Writer) runCleanup() {
	defer close(w.done)
	defer goroutine.RecoverPanic(w.ctx, "logrotate.cleanup")

	for {
		select {
		case _, ok := <-w.cleanupSignal:
			if !ok {
				return
			}
			w.performCleanup()
		case <-w.ctx.Done():
			return
		}
	}
}

// performCleanup compresses uncompressed backups and removes old backups
// that exceed MaxBackups or MaxAge. Each step checks context cancellation
// to exit promptly during shutdown.
func (w *Writer) performCleanup() {
	if w.ctx.Err() != nil {
		return
	}

	if w.config.Compress {
		w.compressBackups()
	}

	if w.ctx.Err() != nil {
		return
	}

	backups, err := w.oldBackups()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logrotate: failed to list backups: %v\n", err)
		return
	}

	removals := w.selectRemovals(backups)
	for _, name := range removals {
		if w.ctx.Err() != nil {
			return
		}
		if removeError := w.sandbox.Remove(name); removeError != nil {
			fmt.Fprintf(os.Stderr, "logrotate: failed to remove %q: %v\n", name, removeError)
		}
	}
}

// compressBackups finds uncompressed backup files and compresses each one
// using gzip with atomic rename. Exits early if the context is cancelled.
func (w *Writer) compressBackups() {
	backups, err := w.oldBackups()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logrotate: failed to list backups for compression: %v\n", err)
		return
	}

	for _, backup := range backups {
		if w.ctx.Err() != nil {
			return
		}
		if strings.HasSuffix(backup.name, compressedSuffix) {
			continue
		}
		if compressError := w.compressFile(backup.name); compressError != nil {
			fmt.Fprintf(os.Stderr, "logrotate: failed to compress %q: %v\n", backup.name, compressError)
		}
	}
}

// compressFile compresses a single backup file using gzip with an atomic
// write-then-rename strategy.
//
// Data is written to a temporary file which is then renamed to the final
// compressed name. The original uncompressed file is removed on success.
//
// Takes source (string) which is the filename relative to the sandbox root.
//
// Returns error when compression fails.
func (w *Writer) compressFile(source string) error {
	sourceHandle, openError := w.sandbox.Open(source)
	if openError != nil {
		return fmt.Errorf("opening source: %w", openError)
	}

	destinationName := source + compressedSuffix
	temporaryName := destinationName + temporarySuffix

	destinationHandle, createError := w.sandbox.OpenFile(
		temporaryName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions,
	)
	if createError != nil {
		return errors.Join(
			fmt.Errorf("creating temp file: %w", createError),
			sourceHandle.Close(),
		)
	}

	gzipWriter, gzipError := gzip.NewWriterLevel(destinationHandle, gzip.BestCompression)
	if gzipError != nil {
		return errors.Join(
			fmt.Errorf("creating gzip writer: %w", gzipError),
			destinationHandle.Close(),
			sourceHandle.Close(),
		)
	}
	_, copyError := io.Copy(gzipWriter, sourceHandle)

	closeErrors := errors.Join(
		gzipWriter.Close(),
		destinationHandle.Close(),
		sourceHandle.Close(),
	)

	if err := errors.Join(copyError, closeErrors); err != nil {
		return errors.Join(
			fmt.Errorf("compressing data: %w", err),
			w.sandbox.Remove(temporaryName),
		)
	}

	if err := w.sandbox.Rename(temporaryName, destinationName); err != nil {
		return errors.Join(
			fmt.Errorf("renaming compressed file: %w", err),
			w.sandbox.Remove(temporaryName),
		)
	}

	if err := w.sandbox.Remove(source); err != nil {
		return fmt.Errorf("removing original after compression: %w", err)
	}

	return nil
}

// selectRemovals determines which backup files should be removed based on
// MaxBackups and MaxAge. The backups slice must be sorted newest-first.
//
// Takes backups ([]backupInfo) which contains the backup files to evaluate.
//
// Returns []string which contains the filenames to remove.
func (w *Writer) selectRemovals(backups []backupInfo) []string {
	if w.config.MaxBackups == 0 && w.config.MaxAge == 0 {
		return nil
	}

	marked := make(map[string]bool)

	if w.config.MaxBackups > 0 && len(backups) > w.config.MaxBackups {
		for _, backup := range backups[w.config.MaxBackups:] {
			marked[backup.name] = true
		}
	}

	if w.config.MaxAge > 0 {
		cutoff := w.clock.Now().Add(-time.Duration(w.config.MaxAge) * 24 * time.Hour)
		for _, backup := range backups {
			if backup.timestamp.Before(cutoff) {
				marked[backup.name] = true
			}
		}
	}

	removals := make([]string, 0, len(marked))
	for name := range marked {
		removals = append(removals, name)
	}
	return removals
}

// backupInfo holds the filename and parsed timestamp of a rotated backup.
type backupInfo struct {
	// timestamp is the time extracted from the backup filename.
	timestamp time.Time

	// name is the backup filename relative to the sandbox root.
	name string
}

// oldBackups scans the sandbox root directory and returns all backup files
// matching the expected naming pattern, sorted newest-first.
//
// Returns []backupInfo which contains the discovered backups.
// Returns error when the directory cannot be read.
func (w *Writer) oldBackups() ([]backupInfo, error) {
	entries, readError := w.sandbox.ReadDir(".")
	if readError != nil {
		return nil, readError
	}

	var backups []backupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		timestamp, ok := w.parseBackupTimestamp(name)
		if !ok {
			continue
		}
		backups = append(backups, backupInfo{name: name, timestamp: timestamp})
	}

	slices.SortFunc(backups, func(a, b backupInfo) int {
		return b.timestamp.Compare(a.timestamp)
	})

	return backups, nil
}

// parseBackupTimestamp extracts the timestamp from a backup filename that
// matches the pattern {prefix}-{timestamp}{extension}[.gz].
//
// Takes name (string) which is the filename to parse.
//
// Returns time.Time which is the parsed timestamp.
// Returns bool which indicates whether the filename matched the pattern.
func (w *Writer) parseBackupTimestamp(name string) (time.Time, bool) {
	candidate, _ := strings.CutSuffix(name, compressedSuffix)

	if !strings.HasPrefix(candidate, w.prefix+"-") {
		return time.Time{}, false
	}
	if !strings.HasSuffix(candidate, w.extension) {
		return time.Time{}, false
	}

	timestampPart := strings.TrimPrefix(candidate, w.prefix+"-")
	timestampPart = strings.TrimSuffix(timestampPart, w.extension)

	parsed, parseError := time.ParseInLocation(backupTimestampFormat, timestampPart, w.location())
	if parseError != nil {
		return time.Time{}, false
	}

	return parsed, true
}

// location returns the time location for backup timestamps based on the
// LocalTime configuration.
//
// Returns *time.Location which is [time.Local] when LocalTime is true,
// or [time.UTC] otherwise.
func (w *Writer) location() *time.Location {
	if w.config.LocalTime {
		return time.Local
	}
	return time.UTC
}
