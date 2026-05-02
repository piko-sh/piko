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

package migration_sql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"piko.sh/piko/internal/querier/querier_domain"
)

const (
	// defaultMaxMigrationFileBytes caps the size of a migration or seed file loaded by
	// FSFileReader. Migrations are typically small; oversized files are likely a
	// misconfiguration or denial-of-service vector.
	defaultMaxMigrationFileBytes int64 = 4 * 1024 * 1024
)

var (
	// ErrMigrationFileTooLarge is returned when a migration or seed file exceeds the
	// configured size cap. Callers can use errors.Is to detect this condition without
	// parsing the message.
	ErrMigrationFileTooLarge = errors.New("migration file exceeds configured size limit")
)

// FSFileReader implements FileReaderPort using an io/fs.FS. This allows reading migration
// files from embed.FS without requiring OS-level filesystem access.
type FSFileReader struct {
	// filesystem holds the underlying fs.FS used for file operations.
	filesystem fs.FS

	// maxFileBytes caps the size of files returned from ReadFile. A non-positive value falls
	// back to defaultMaxMigrationFileBytes.
	maxFileBytes int64
}

// FSFileReaderOption configures optional behaviour for FSFileReader instances.
type FSFileReaderOption func(*FSFileReader)

// WithMaxMigrationFileBytes overrides the per-file size cap enforced when reading
// migration or seed content. Non-positive values reset the cap to
// defaultMaxMigrationFileBytes.
//
// Takes maxBytes (int64) which is the maximum file size in bytes.
//
// Returns FSFileReaderOption which applies the cap to the reader when passed to
// NewFSFileReader.
func WithMaxMigrationFileBytes(maxBytes int64) FSFileReaderOption {
	return func(r *FSFileReader) {
		if maxBytes <= 0 {
			r.maxFileBytes = defaultMaxMigrationFileBytes
			return
		}
		r.maxFileBytes = maxBytes
	}
}

var (
	_ querier_domain.FileReaderPort = (*FSFileReader)(nil)
)

// NewFSFileReader creates a FileReaderPort backed by the given filesystem.
//
// Takes filesystem (fs.FS) which is the filesystem to read from (typically an embed.FS).
// Takes opts (...FSFileReaderOption) which configure optional behaviour such as the
// maximum file size.
//
// Returns *FSFileReader which is ready to read files.
func NewFSFileReader(filesystem fs.FS, opts ...FSFileReaderOption) *FSFileReader {
	reader := &FSFileReader{
		filesystem:   filesystem,
		maxFileBytes: defaultMaxMigrationFileBytes,
	}
	for _, opt := range opts {
		opt(reader)
	}
	return reader
}

// ReadFile reads the contents of a file from the embedded filesystem.
//
// The read is capped at the configured maxFileBytes; files exceeding this cap return
// ErrMigrationFileTooLarge wrapped with %w.
//
// Takes path (string) which is the file path relative to the filesystem root.
//
// Returns []byte which holds the file contents.
// Returns error when ctx is cancelled, the file cannot be read, or the size cap is
// exceeded.
func (reader *FSFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	maxBytes := reader.effectiveMaxFileBytes()
	file, openErr := reader.filesystem.Open(path)
	if openErr != nil {
		return nil, openErr
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr == nil && info.Size() > maxBytes {
		return nil, fmt.Errorf("file %q is %d bytes (cap %d): %w",
			path, info.Size(), maxBytes, ErrMigrationFileTooLarge)
	}

	limited := io.LimitReader(file, maxBytes+1)
	content, readErr := io.ReadAll(limited)
	if readErr != nil {
		return nil, fmt.Errorf("reading migration file %q: %w", path, readErr)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("file %q exceeds %d bytes: %w",
			path, maxBytes, ErrMigrationFileTooLarge)
	}
	return content, nil
}

// ReadDir reads the directory entries from the embedded filesystem, sorted by name.
//
// Context cancellation is honoured before the synchronous fs.ReadDir call dispatches; the
// underlying io/fs.FS interface has no cancellation hook so an in-flight read runs to
// completion.
//
// Takes directory (string) which is the directory path relative to the filesystem root.
//
// Returns []os.DirEntry which holds the directory entries.
// Returns error when ctx is cancelled or the directory cannot be read.
func (reader *FSFileReader) ReadDir(ctx context.Context, directory string) ([]os.DirEntry, error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	entries, readError := fs.ReadDir(reader.filesystem, directory)
	if readError != nil {
		return nil, fmt.Errorf("reading directory %q: %w", directory, readError)
	}

	result := make([]os.DirEntry, len(entries))
	copy(result, entries)

	return result, nil
}

// effectiveMaxFileBytes returns the configured cap, defaulting to
// defaultMaxMigrationFileBytes when no positive override has been supplied.
//
// Returns int64 which is the active per-file byte cap.
func (reader *FSFileReader) effectiveMaxFileBytes() int64 {
	if reader.maxFileBytes <= 0 {
		return defaultMaxMigrationFileBytes
	}
	return reader.maxFileBytes
}
