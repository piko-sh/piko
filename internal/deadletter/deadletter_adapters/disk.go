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

package deadletter_adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// filePermissions is the Unix file mode for dead letter files (owner read and
	// write only).
	filePermissions = 0o600

	// errReadingDeadLetter is the error format for failed dead letter file reads.
	errReadingDeadLetter = "reading dead letter file: %w"

	// defaultMaxDLQBytes caps the on-disk size of a dead-letter file at
	// 256 MiB.
	//
	// Callers can override via WithMaxDLQBytes. The default protects
	// against a runaway producer turning the DLQ into a liability.
	defaultMaxDLQBytes int64 = 256 * 1024 * 1024
)

// ErrDLQFull is returned by Add when the dead-letter file already exceeds the
// configured maximum size. Callers should treat this as a backpressure signal
// rather than a transient error: log the dropped entry and surface a monitoring
// alert so an operator can drain the queue.
var ErrDLQFull = errors.New("dead-letter queue exceeded maximum size")

// newlineSeparator is the byte slice used to delimit JSON lines entries in
// dead letter files.
var newlineSeparator = []byte("\n")

// DiskDeadLetterQueue is a generic disk-based dead letter queue using JSON
// lines format. Entries are appended in O(1) writes and the file is bounded
// by maxBytes to prevent unbounded growth from a misbehaving producer.
type DiskDeadLetterQueue[T any] struct {
	// sandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil and sandbox is nil, this factory is used instead of
	// safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox provides file system access for reading and writing dead letter
	// files.
	sandbox safedisk.Sandbox

	// fileName is the base name for the dead letter file within the sandbox.
	fileName string

	// maxBytes is the size cap for the dead-letter file; further appends fail
	// with ErrDLQFull once exceeded.
	maxBytes int64

	// mu guards access to the dead letter queue state.
	mu sync.Mutex
}

// DiskDeadLetterOption configures a DiskDeadLetterQueue during creation.
type DiskDeadLetterOption[T any] func(*DiskDeadLetterQueue[T])

// Add appends an item to the dead letter file in O(1) time.
//
// The file is opened with O_APPEND and a single JSON-line is written, then
// fsynced for durability. The mutex serialises concurrent appends so the
// stat-then-append size check stays consistent for a single process.
//
// Takes entry (T) which is the item to store in the dead letter queue.
//
// Returns ErrDLQFull when the on-disk file has exceeded the configured size
// cap. Returns other errors when the entry cannot be marshalled or the write
// fails.
//
// Safe for concurrent use; protected by a mutex.
func (d *DiskDeadLetterQueue[T]) Add(ctx context.Context, entry T) error {
	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	defer d.mu.Unlock()

	wrapped := wrappedEntry[T]{
		ID:        uuid.NewString(),
		Timestamp: time.Now(),
		Data:      entry,
	}

	line, err := json.Marshal(wrapped)
	if err != nil {
		return fmt.Errorf("marshalling dead letter entry: %w", err)
	}

	currentSize, err := d.statSizeUnlocked()
	if err != nil {
		return fmt.Errorf(errReadingDeadLetter, err)
	}

	projectedSize := currentSize + int64(len(line)) + 1
	if d.maxBytes > 0 && projectedSize > d.maxBytes {
		return fmt.Errorf("%w: would reach %d bytes, cap is %d",
			ErrDLQFull, projectedSize, d.maxBytes)
	}

	if err := d.appendLineUnlocked(line); err != nil {
		return fmt.Errorf("writing dead letter entry: %w", err)
	}

	l.Warn("Item added to disk dead letter queue",
		logger_domain.String("entry_id", wrapped.ID),
		logger_domain.String("file", d.fileName))

	return nil
}

// statSizeUnlocked reports the current on-disk size of the DLQ file or zero
// if the file does not yet exist. Must be called with the mutex held.
//
// Returns int64 which is the current file size in bytes.
// Returns error when an unexpected stat error is encountered.
func (d *DiskDeadLetterQueue[T]) statSizeUnlocked() (int64, error) {
	info, err := d.sandbox.Stat(d.fileName)
	if err != nil {
		if safedisk.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

// appendLineUnlocked appends a JSON-encoded line plus newline to the
// dead-letter file.
//
// Opens the file with O_APPEND so the kernel atomically positions writes
// at end-of-file even if multiple processes share the file. Must be
// called with the mutex held.
//
// Takes line ([]byte) which is the marshalled entry without trailing newline.
//
// Returns error when the file cannot be opened, written, fsynced, or closed.
func (d *DiskDeadLetterQueue[T]) appendLineUnlocked(line []byte) error {
	file, err := d.sandbox.OpenFile(d.fileName,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, filePermissions)
	if err != nil {
		return fmt.Errorf("opening dead letter file for append: %w", err)
	}

	buffer := make([]byte, 0, len(line)+1)
	buffer = append(buffer, line...)
	buffer = append(buffer, '\n')

	if _, writeErr := file.Write(buffer); writeErr != nil {
		_ = file.Close()
		return fmt.Errorf("writing dead letter line: %w", writeErr)
	}

	if syncErr := file.Sync(); syncErr != nil {
		_ = file.Close()
		return fmt.Errorf("fsyncing dead letter file: %w", syncErr)
	}

	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("closing dead letter file: %w", closeErr)
	}

	return nil
}

// Get retrieves items from the dead letter queue.
//
// Takes limit (int) which specifies the maximum items to return, or zero for
// no limit.
//
// Returns []T which contains the retrieved items, possibly empty.
// Returns error when the file cannot be read.
//
// Safe for concurrent use; access is protected by a mutex.
func (d *DiskDeadLetterQueue[T]) Get(ctx context.Context, limit int) ([]T, error) {
	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.sandbox.ReadFile(d.fileName)
	if err != nil {
		if safedisk.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf(errReadingDeadLetter, err)
	}

	initialCap := bytes.Count(data, newlineSeparator)
	if limit > 0 && limit < initialCap {
		initialCap = limit
	}
	entries := make([]T, 0, initialCap)

	for line := range bytes.SplitSeq(data, newlineSeparator) {
		if len(line) == 0 {
			continue
		}
		if limit > 0 && len(entries) >= limit {
			break
		}

		var wrapped wrappedEntry[T]
		if err := json.Unmarshal(line, &wrapped); err != nil {
			l.Warn("Failed to unmarshal dead letter entry", logger_domain.Error(err))
			continue
		}
		entries = append(entries, wrapped.Data)
	}

	return entries, nil
}

// Remove removes entries from the dead letter queue.
//
// Rewrites the file, which is slow but needed for the JSON lines format. Without
// a way to identify entries, actual removal is not yet supported. Services should
// create their own DLQ port with ID-based removal.
//
// Takes entries ([]T) which specifies the entries to remove.
//
// Returns error when reading existing entries or rewriting the file fails.
//
// Safe for concurrent use; protected by a mutex.
func (d *DiskDeadLetterQueue[T]) Remove(ctx context.Context, entries []T) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	defer d.mu.Unlock()

	allEntries, err := d.getAllUnlocked(ctx)
	if err != nil {
		return fmt.Errorf("reading dead letter entries for removal: %w", err)
	}

	l.Warn("Disk DLQ Remove requires entry comparison, consider implementing service-specific DLQ",
		logger_domain.Int("requested_to_remove", len(entries)))

	return d.rewriteFileUnlocked(allEntries)
}

// Count returns the number of entries in the dead letter queue.
//
// Returns int which is the count of entries, or zero if the file does not
// exist.
// Returns error when the file cannot be read.
//
// Safe for concurrent use; holds a mutex lock during execution.
func (d *DiskDeadLetterQueue[T]) Count(_ context.Context) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	data, err := d.sandbox.ReadFile(d.fileName)
	if err != nil {
		if safedisk.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf(errReadingDeadLetter, err)
	}

	count := 0
	for line := range bytes.SplitSeq(data, newlineSeparator) {
		if len(line) > 0 {
			count++
		}
	}

	return count, nil
}

// Clear removes all entries from the dead letter queue.
//
// Returns error when the queue file cannot be removed.
//
// Safe for concurrent use. Holds the mutex for the whole operation.
func (d *DiskDeadLetterQueue[T]) Clear(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.sandbox.Remove(d.fileName); err != nil && !safedisk.IsNotExist(err) {
		return fmt.Errorf("clearing dead letter file: %w", err)
	}

	l.Internal("Cleared all entries from disk dead letter queue",
		logger_domain.String("file", d.fileName))

	return nil
}

// GetOlderThan retrieves entries older than the specified duration.
//
// Takes duration (time.Duration) which specifies the age threshold.
//
// Returns []T which contains entries with timestamps before the cutoff.
// Returns error when reading entries from disk fails.
//
// Safe for concurrent use; protects access with a mutex.
func (d *DiskDeadLetterQueue[T]) GetOlderThan(ctx context.Context, duration time.Duration) ([]T, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	allEntries, err := d.getAllUnlocked(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading dead letter entries for age filter: %w", err)
	}

	cutoff := time.Now().Add(-duration)
	var oldEntries []T

	for _, wrapped := range allEntries {
		if wrapped.Timestamp.Before(cutoff) {
			oldEntries = append(oldEntries, wrapped.Data)
		}
	}

	return oldEntries, nil
}

// getAllUnlocked reads all wrapped entries from the file. Must be called with
// lock held.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
//
// Returns []wrappedEntry[T] which contains the deserialised entries.
// Returns error when the file cannot be read.
func (d *DiskDeadLetterQueue[T]) getAllUnlocked(ctx context.Context) ([]wrappedEntry[T], error) {
	ctx, l := logger_domain.From(ctx, log)
	data, err := d.sandbox.ReadFile(d.fileName)
	if err != nil {
		if safedisk.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf(errReadingDeadLetter, err)
	}

	entries := make([]wrappedEntry[T], 0, bytes.Count(data, newlineSeparator))

	for line := range bytes.SplitSeq(data, newlineSeparator) {
		if len(line) == 0 {
			continue
		}

		var wrapped wrappedEntry[T]
		if err := json.Unmarshal(line, &wrapped); err != nil {
			l.Warn("Failed to unmarshal dead letter entry", logger_domain.Error(err))
			continue
		}
		entries = append(entries, wrapped)
	}

	return entries, nil
}

// rewriteFileUnlocked rewrites the file with the given entries.
// Must be called with lock held.
//
// Takes entries ([]wrappedEntry[T]) which contains the entries to write.
//
// Returns error when the temp file cannot be written, entries cannot be
// marshalled, or the file cannot be replaced.
func (d *DiskDeadLetterQueue[T]) rewriteFileUnlocked(entries []wrappedEntry[T]) error {
	var buffer bytes.Buffer

	for _, wrapped := range entries {
		data, err := json.Marshal(wrapped)
		if err != nil {
			return fmt.Errorf("marshalling entry: %w", err)
		}
		_, _ = buffer.Write(data)
		_ = buffer.WriteByte('\n')
	}

	tempName := d.fileName + ".tmp"

	if err := d.sandbox.WriteFile(tempName, buffer.Bytes(), filePermissions); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := d.sandbox.Rename(tempName, d.fileName); err != nil {
		_ = d.sandbox.Remove(tempName)
		return fmt.Errorf("replacing dead letter file: %w", err)
	}

	return nil
}

var _ deadletter_domain.DeadLetterPort[any] = (*DiskDeadLetterQueue[any])(nil)

// WithDeadLetterSandbox sets a custom sandbox for the dead letter queue. This
// allows injection of mock sandboxes for testing filesystem operations.
//
// If not provided, a real sandbox is created using safedisk.NewNoOpSandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access within
// the dead letter directory.
//
// Returns DiskDeadLetterOption[T] which configures the queue with the given
// sandbox.
func WithDeadLetterSandbox[T any](sandbox safedisk.Sandbox) DiskDeadLetterOption[T] {
	return func(d *DiskDeadLetterQueue[T]) {
		d.sandbox = sandbox
	}
}

// WithDeadLetterFactory sets a factory for creating sandboxes when no sandbox
// is directly injected. When non-nil and no sandbox is set, this factory is
// used instead of safedisk.NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes for the dead letter
// directory.
//
// Returns DiskDeadLetterOption[T] which configures the queue with the given
// factory.
func WithDeadLetterFactory[T any](factory safedisk.Factory) DiskDeadLetterOption[T] {
	return func(d *DiskDeadLetterQueue[T]) {
		d.sandboxFactory = factory
	}
}

// WithMaxDLQBytes overrides the size cap on the dead-letter file.
//
// Once the file exceeds this many bytes, further Add calls fail with
// ErrDLQFull. Pass a zero or negative value to disable the cap (not
// recommended in production).
//
// Takes maxBytes (int64) which is the maximum on-disk size in bytes.
//
// Returns DiskDeadLetterOption[T] which configures the queue with the given
// cap.
func WithMaxDLQBytes[T any](maxBytes int64) DiskDeadLetterOption[T] {
	return func(d *DiskDeadLetterQueue[T]) {
		d.maxBytes = maxBytes
	}
}

// NewDiskDeadLetterQueue creates a new disk-based dead letter queue.
//
// Takes filePath (string) which is the path to the file for storing dead
// letters.
// Takes opts (...DiskDeadLetterOption[T]) which provides optional configuration
// such as WithDeadLetterSandbox for testing.
//
// Returns deadletter_domain.DeadLetterPort[T] which is ready to use.
func NewDiskDeadLetterQueue[T any](filePath string, opts ...DiskDeadLetterOption[T]) deadletter_domain.DeadLetterPort[T] {
	directory := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	d := &DiskDeadLetterQueue[T]{
		fileName: fileName,
		maxBytes: defaultMaxDLQBytes,
	}

	for _, opt := range opts {
		opt(d)
	}

	if d.sandbox == nil && d.sandboxFactory != nil {
		sandbox, err := d.sandboxFactory.Create("deadletter", directory, safedisk.ModeReadWrite)
		if err == nil {
			d.sandbox = sandbox
		}
	}
	if d.sandbox == nil {
		sandbox, err := safedisk.NewNoOpSandbox(directory, safedisk.ModeReadWrite)
		if err != nil {
			return d
		}
		d.sandbox = sandbox
	}

	return d
}
