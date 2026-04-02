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

package wasm_adapters

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/generator/generator_domain"
)

const (
	// dirPermission is the permission mode for directories (rwxr-xr-x).
	dirPermission fs.FileMode = 0755

	// filePermission is the file permission mode (owner read-write, others read).
	filePermission fs.FileMode = 0644

	// pathSep is the forward slash used to separate parts of normalised paths.
	pathSep = "/"
)

// InMemoryFSReader implements FSReaderPort using an in-memory file map.
// It is used for WASM contexts where file system access is not available.
//
// The reader supports both exact path matches and path normalisation,
// making it flexible for different path formats used by callers.
type InMemoryFSReader struct {
	// files maps file paths to their content as byte slices.
	files map[string][]byte

	// mu guards access to files for safe concurrent reads and writes.
	mu sync.RWMutex
}

var (
	_ annotator_domain.FSReaderPort = (*InMemoryFSReader)(nil)

	_ generator_domain.FSReaderPort = (*InMemoryFSReader)(nil)

	_ generator_domain.FSWriterPort = (*InMemoryFSWriter)(nil)
)

// NewInMemoryFSReader creates a new in-memory file reader from a map of file
// paths to their contents.
//
// Takes files (map[string]string) which maps file paths to their contents.
//
// Returns *InMemoryFSReader which is ready for reading files.
func NewInMemoryFSReader(files map[string]string) *InMemoryFSReader {
	byteFiles := make(map[string][]byte, len(files))
	for path, content := range files {
		normalised := filepath.ToSlash(filepath.Clean(path))
		byteFiles[normalised] = []byte(content)
	}
	return &InMemoryFSReader{files: byteFiles}
}

// NewInMemoryFSReaderFromBytes creates an in-memory file reader from a map of
// file paths to their byte contents.
//
// Takes files (map[string][]byte) which maps file paths to their contents.
//
// Returns *InMemoryFSReader which is ready for reading files.
func NewInMemoryFSReaderFromBytes(files map[string][]byte) *InMemoryFSReader {
	byteFiles := make(map[string][]byte, len(files))
	for path, content := range files {
		normalised := filepath.ToSlash(filepath.Clean(path))
		byteFiles[normalised] = content
	}
	return &InMemoryFSReader{files: byteFiles}
}

// ReadFile reads the contents of a file from the in-memory file system.
//
// Takes filePath (string) which is the path to the file to read.
//
// Returns []byte which contains the file contents.
// Returns error when the file is not found.
//
// Safe for concurrent use.
func (r *InMemoryFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalised := filepath.ToSlash(filepath.Clean(filePath))

	if content, ok := r.files[normalised]; ok {
		return content, nil
	}

	withoutLeading := strings.TrimPrefix(normalised, "/")
	if content, ok := r.files[withoutLeading]; ok {
		return content, nil
	}

	withLeading := "/" + withoutLeading
	if content, ok := r.files[withLeading]; ok {
		return content, nil
	}

	return nil, fmt.Errorf("file not found in in-memory FS: %s", filePath)
}

// AddFile adds or updates a file in the in-memory file system.
//
// Takes path (string) which is the file path.
// Takes content ([]byte) which is the file content.
//
// Safe for concurrent use.
func (r *InMemoryFSReader) AddFile(path string, content []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	normalised := filepath.ToSlash(filepath.Clean(path))
	r.files[normalised] = content
}

// GetFiles returns a copy of all files in the in-memory file system.
//
// Returns map[string][]byte which maps paths to contents.
//
// Safe for concurrent use.
func (r *InMemoryFSReader) GetFiles() map[string][]byte {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]byte, len(r.files))
	maps.Copy(result, r.files)
	return result
}

// InMemoryFSWriter implements FSWriterPort using an in-memory file map.
// It captures written files for retrieval in WASM or testing contexts.
type InMemoryFSWriter struct {
	// written maps file paths to their contents.
	written map[string][]byte

	// mu guards concurrent access to the written map.
	mu sync.RWMutex
}

// NewInMemoryFSWriter creates a new in-memory file writer.
//
// Returns *InMemoryFSWriter which is ready to capture file writes.
func NewInMemoryFSWriter() *InMemoryFSWriter {
	return &InMemoryFSWriter{
		written: make(map[string][]byte),
	}
}

// WriteFile writes data to the in-memory file system.
//
// Takes filePath (string) which specifies the destination file path.
// Takes data ([]byte) which contains the content to write.
//
// Returns error which is always nil for in-memory writes.
//
// Safe for concurrent use.
func (w *InMemoryFSWriter) WriteFile(_ context.Context, filePath string, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	normalised := filepath.ToSlash(filepath.Clean(filePath))
	contentCopy := make([]byte, len(data))
	copy(contentCopy, data)
	w.written[normalised] = contentCopy
	return nil
}

// ReadDir reads the directory at the given path and returns a list of
// directory entries sorted by filename.
//
// This implementation synthesises directory entries from the in-memory files
// that have paths starting with the given directory.
//
// Takes dirname (string) which specifies the directory path to read.
//
// Returns []os.DirEntry which contains the directory entries.
// Returns error when the directory cannot be read.
//
// Safe for concurrent use; uses a read lock to protect access to the
// underlying file map.
func (w *InMemoryFSWriter) ReadDir(dirname string) ([]os.DirEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	normalised := filepath.ToSlash(filepath.Clean(dirname))
	if !strings.HasSuffix(normalised, pathSep) {
		normalised += pathSep
	}

	seen := make(map[string]bool)
	entries := make([]os.DirEntry, 0, len(w.written))

	for path := range w.written {
		if !strings.HasPrefix(path, normalised) {
			continue
		}

		rel := strings.TrimPrefix(path, normalised)
		if rel == "" {
			continue
		}

		parts := strings.SplitN(rel, pathSep, 2)
		name := parts[0]

		if seen[name] {
			continue
		}
		seen[name] = true

		isDir := len(parts) > 1
		entries = append(entries, &inMemoryDirEntry{
			name:  name,
			isDir: isDir,
		})
	}

	slices.SortFunc(entries, func(a, b os.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	return entries, nil
}

// RemoveAll removes path and any children from the in-memory file system.
//
// Takes path (string) which specifies the path to remove.
//
// Returns error which is always nil for in-memory removal.
//
// Safe for concurrent use.
func (w *InMemoryFSWriter) RemoveAll(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	normalised := filepath.ToSlash(filepath.Clean(path))

	delete(w.written, normalised)

	prefix := normalised + pathSep
	for p := range w.written {
		if strings.HasPrefix(p, prefix) {
			delete(w.written, p)
		}
	}

	return nil
}

// GetWrittenFiles returns a copy of all files written to the in-memory file
// system.
//
// Returns map[string][]byte which maps paths to contents.
//
// Safe for concurrent use.
func (w *InMemoryFSWriter) GetWrittenFiles() map[string][]byte {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make(map[string][]byte, len(w.written))
	maps.Copy(result, w.written)
	return result
}

// GetWrittenFile returns the content of a single written file.
//
// Takes path (string) which is the file path to retrieve.
//
// Returns []byte which contains the file content.
// Returns bool which is true if the file exists.
//
// Safe for concurrent use.
func (w *InMemoryFSWriter) GetWrittenFile(path string) ([]byte, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	normalised := filepath.ToSlash(filepath.Clean(path))
	content, ok := w.written[normalised]
	return content, ok
}

// Clear removes all written files from the in-memory file system.
//
// Safe for concurrent use.
func (w *InMemoryFSWriter) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.written = make(map[string][]byte)
}

// inMemoryDirEntry implements os.DirEntry for directory entries stored in memory.
type inMemoryDirEntry struct {
	// name is the base name of the file or directory entry.
	name string

	// isDir indicates whether the entry represents a directory.
	isDir bool
}

// Name returns the name of the file or subdirectory.
//
// Returns string which is the base name of the directory entry.
func (e *inMemoryDirEntry) Name() string {
	return e.name
}

// IsDir reports whether the entry describes a directory.
//
// Returns bool which is true if the entry is a directory.
func (e *inMemoryDirEntry) IsDir() bool {
	return e.isDir
}

// Type returns the type bits for the entry.
//
// Returns fs.FileMode which is ModeDir for directories or zero for files.
func (e *inMemoryDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}

// Info returns the FileInfo for the file or subdirectory.
//
// Returns fs.FileInfo which describes the file or directory.
// Returns error when the file information cannot be retrieved.
func (e *inMemoryDirEntry) Info() (fs.FileInfo, error) {
	return &inMemoryFileInfo{
		name:  e.name,
		isDir: e.isDir,
	}, nil
}

// inMemoryFileInfo implements fs.FileInfo for in-memory files.
type inMemoryFileInfo struct {
	// name is the base name of the file.
	name string

	// isDir bool // isDir indicates whether this entry is a directory.
	isDir bool

	// size is the file size in bytes.
	size int64
}

// Name returns the base name of the file.
//
// Returns string which is the file's base name.
func (fi *inMemoryFileInfo) Name() string {
	return fi.name
}

// Size returns the length in bytes.
//
// Returns int64 which is the file size.
func (fi *inMemoryFileInfo) Size() int64 {
	return fi.size
}

// Mode returns the file mode bits.
//
// Returns fs.FileMode which is the permission and mode bits for the file.
func (fi *inMemoryFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir | dirPermission
	}
	return filePermission
}

// ModTime returns the modification time.
//
// Returns time.Time which is the zero time value.
func (*inMemoryFileInfo) ModTime() time.Time {
	return time.Time{}
}

// IsDir returns whether this is a directory.
//
// Returns bool which is true if this file info represents a directory.
func (fi *inMemoryFileInfo) IsDir() bool {
	return fi.isDir
}

// Sys returns the underlying data source.
//
// Returns any which is always nil for in-memory files.
func (*inMemoryFileInfo) Sys() any {
	return nil
}
