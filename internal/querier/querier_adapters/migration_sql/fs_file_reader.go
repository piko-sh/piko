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
	"fmt"
	"io/fs"
	"os"

	"piko.sh/piko/internal/querier/querier_domain"
)

// FSFileReader implements FileReaderPort using an io/fs.FS. This allows
// reading migration files from embed.FS without requiring OS-level filesystem
// access.
type FSFileReader struct {
	// filesystem holds the underlying fs.FS used for file operations.
	filesystem fs.FS
}

var _ querier_domain.FileReaderPort = (*FSFileReader)(nil)

// NewFSFileReader creates a FileReaderPort backed by the given filesystem.
//
// Takes filesystem (fs.FS) which is the filesystem to read from (typically
// an embed.FS).
//
// Returns *FSFileReader which is ready to read files.
func NewFSFileReader(filesystem fs.FS) *FSFileReader {
	return &FSFileReader{filesystem: filesystem}
}

// ReadFile reads the contents of a file from the embedded filesystem.
//
// Takes path (string) which is the file path relative to the filesystem root.
//
// Returns []byte which holds the file contents.
// Returns error when the file cannot be read.
func (reader *FSFileReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	return fs.ReadFile(reader.filesystem, path)
}

// ReadDir reads the directory entries from the embedded filesystem, sorted by
// name.
//
// Takes directory (string) which is the directory path relative to the
// filesystem root.
//
// Returns []os.DirEntry which holds the directory entries.
// Returns error when the directory cannot be read.
func (reader *FSFileReader) ReadDir(_ context.Context, directory string) ([]os.DirEntry, error) {
	entries, readError := fs.ReadDir(reader.filesystem, directory)
	if readError != nil {
		return nil, fmt.Errorf("reading directory %q: %w", directory, readError)
	}

	result := make([]os.DirEntry, len(entries))
	copy(result, entries)

	return result, nil
}
