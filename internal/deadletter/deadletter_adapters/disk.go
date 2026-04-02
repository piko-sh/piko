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
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// filePermissions is the Unix file mode for dead letter files (owner read and
	// write only).
	filePermissions = 0o600

	// errReadingDeadLetter is the error format for failed dead letter file reads.
	errReadingDeadLetter = "reading dead letter file: %w"
)

// newlineSeparator is the byte slice used to delimit JSON lines entries in
// dead letter files.
var newlineSeparator = []byte("\n")

// DiskDeadLetterQueue is a generic disk-based dead letter queue using JSON
// lines format.
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

	// mu guards access to the dead letter queue state.
	mu sync.Mutex
}

// DiskDeadLetterOption configures a DiskDeadLetterQueue during creation.
type DiskDeadLetterOption[T any] func(*DiskDeadLetterQueue[T])

// Add appends an item to the dead letter file.
//
// Takes entry (T) which is the item to store in the dead letter queue.
//
// Returns error when the file cannot be read, the entry cannot be marshalled,
// or the write fails.
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

	existing, err := d.sandbox.ReadFile(d.fileName)
	if err != nil && !safedisk.IsNotExist(err) {
		return fmt.Errorf(errReadingDeadLetter, err)
	}

	line, err := json.Marshal(wrapped)
	if err != nil {
		return fmt.Errorf("marshalling dead letter entry: %w", err)
	}

	data := make([]byte, 0, len(existing)+len(line)+1)
	data = append(data, existing...)
	data = append(data, line...)
	data = append(data, '\n')

	if err := d.sandbox.WriteFile(d.fileName, data, filePermissions); err != nil {
		return fmt.Errorf("writing dead letter entry: %w", err)
	}

	l.Warn("Item added to disk dead letter queue",
		logger_domain.String("entry_id", wrapped.ID),
		logger_domain.String("file", d.fileName))

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
// This method rewrites the file, which is slow but needed for the JSON lines
// format. Without a way to identify entries, actual removal is not yet
// supported. Services should create their own DLQ port with ID-based removal.
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
// Safe for concurrent use. The method holds the mutex for the whole operation.
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
