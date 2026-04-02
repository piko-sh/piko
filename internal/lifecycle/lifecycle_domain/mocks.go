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

package lifecycle_domain

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// mockFilePermission is the default file mode for mock files.
	mockFilePermission fs.FileMode = 0644

	// mockDirPermission is the file mode used for mock directory entries.
	mockDirPermission fs.FileMode = 0755
)

var _ FileSystem = (*MockFileSystem)(nil)

// MockFileSystem is an in-memory implementation of FileSystem for
// testing, allowing tests to simulate file system operations without
// touching the real file system.
//
// Map access is guarded by a sync.RWMutex because tests may populate
// files during setup while parallel subtests read concurrently.
type MockFileSystem struct {
	// files maps cleaned file paths to their mock file data.
	files map[string]*mockFile

	// dirs holds paths that are directories in the mock file system.
	dirs map[string]bool

	// mu guards access to the files and dirs maps.
	mu sync.RWMutex
}

// mockFile represents a file in the mock file system.
type mockFile struct {
	// modTime is the simulated file modification time.
	modTime time.Time

	// content holds the file data returned when the mock file is opened.
	content []byte

	// mode is the file permission and type bits.
	mode fs.FileMode
}

// NewMockFileSystem creates an empty mock file system for testing.
//
// Returns *MockFileSystem which is ready for use.
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string]*mockFile),
		dirs:  make(map[string]bool),
	}
}

// AddFile adds a file to the mock file system.
//
// Takes path (string) which specifies the file path to create.
// Takes content ([]byte) which provides the file contents.
//
// Safe for use from multiple goroutines.
func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.files[filepath.Clean(path)] = &mockFile{
		content: content,
		modTime: time.Now(),
		mode:    mockFilePermission,
	}

	directory := filepath.Dir(path)
	for directory != "." && directory != "/" {
		m.dirs[filepath.Clean(directory)] = true
		directory = filepath.Dir(directory)
	}
}

// AddDir adds a directory to the mock filesystem.
//
// Takes path (string) which specifies the directory path to add.
//
// Safe for concurrent use.
func (m *MockFileSystem) AddDir(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dirs[filepath.Clean(path)] = true
}

// WalkDir walks the file tree rooted at root.
//
// Takes root (string) which specifies the starting directory path.
// Takes walkFunction (fs.WalkDirFunc) which handles each file or
// directory visited.
//
// Returns error when the root path does not exist.
//
// Safe for concurrent use; acquires a read lock during traversal.
func (m *MockFileSystem) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanRoot := filepath.Clean(root)
	if !m.rootExists(cleanRoot) {
		return fs.ErrNotExist
	}

	entries := m.collectEntriesUnder(cleanRoot)
	return m.walkEntries(entries, walkFunction)
}

// Open opens the named file for reading.
//
// Takes name (string) which is the path to the file to open.
//
// Returns io.ReadCloser which provides access to the file content.
// Returns error when the file does not exist.
//
// Safe for concurrent use.
func (m *MockFileSystem) Open(name string) (io.ReadCloser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanName := filepath.Clean(name)
	file, exists := m.files[cleanName]
	if !exists {
		return nil, fs.ErrNotExist
	}

	return &mockReadCloser{data: file.content, offset: 0}, nil
}

// Stat returns file info for the named file.
//
// Takes name (string) which specifies the file or directory path to query.
//
// Returns fs.FileInfo which describes the file or directory.
// Returns error when the path does not exist.
//
// Safe for concurrent use.
func (m *MockFileSystem) Stat(name string) (fs.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cleanName := filepath.Clean(name)

	if file, exists := m.files[cleanName]; exists {
		return &mockFileInfo{
			name:    filepath.Base(cleanName),
			size:    int64(len(file.content)),
			mode:    file.mode,
			modTime: file.modTime,
			isDir:   false,
		}, nil
	}

	if _, exists := m.dirs[cleanName]; exists {
		return &mockFileInfo{
			name:    filepath.Base(cleanName),
			size:    0,
			mode:    fs.ModeDir | mockDirPermission,
			modTime: time.Now(),
			isDir:   true,
		}, nil
	}

	return nil, fs.ErrNotExist
}

// Rel returns a relative path from basepath to targpath.
//
// Takes basepath (string) which is the base directory path.
// Takes targpath (string) which is the target path to make relative.
//
// Returns string which is the relative path from base to target.
// Returns error when a relative path cannot be determined.
func (*MockFileSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Join joins path elements into a single path.
//
// Takes element (...string) which specifies the path elements to join.
//
// Returns string which is the combined path using the OS path separator.
func (*MockFileSystem) Join(element ...string) string {
	return filepath.Join(element...)
}

// IsNotExist returns whether the error reports that a file does not exist.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error indicates a missing file.
func (*MockFileSystem) IsNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

// rootExists checks if the given path exists as a directory or file.
//
// Takes cleanRoot (string) which specifies the path to check.
//
// Returns bool which indicates whether the path exists.
func (m *MockFileSystem) rootExists(cleanRoot string) bool {
	if _, exists := m.dirs[cleanRoot]; exists {
		return true
	}
	_, exists := m.files[cleanRoot]
	return exists
}

// collectEntriesUnder gathers all paths under a given root path.
//
// Takes cleanRoot (string) which specifies the root path to search under.
//
// Returns []string which contains all file and directory paths that start
// with the given root.
func (m *MockFileSystem) collectEntriesUnder(cleanRoot string) []string {
	var entries []string
	for path := range m.files {
		if strings.HasPrefix(path, cleanRoot) {
			entries = append(entries, path)
		}
	}
	for path := range m.dirs {
		if strings.HasPrefix(path, cleanRoot) {
			entries = append(entries, path)
		}
	}
	return entries
}

// walkEntries calls the callback for each entry path.
//
// Takes entries ([]string) which specifies the paths to walk.
// Takes walkFunction (fs.WalkDirFunc) which handles each directory entry.
//
// Returns error when the callback returns an error other than fs.SkipDir.
func (m *MockFileSystem) walkEntries(entries []string, walkFunction fs.WalkDirFunc) error {
	for _, path := range entries {
		entry := m.makeDirEntry(path)
		if err := walkFunction(path, entry, nil); err != nil {
			if errors.Is(err, fs.SkipDir) {
				continue
			}
			return err
		}
	}
	return nil
}

// makeDirEntry creates an fs.DirEntry for the given path.
//
// Takes path (string) which specifies the file or directory path to look up.
//
// Returns fs.DirEntry which is the directory entry, or nil if the path does
// not exist in the mock file system.
func (m *MockFileSystem) makeDirEntry(path string) fs.DirEntry {
	if _, isDir := m.dirs[path]; isDir {
		return &mockDirEntry{name: filepath.Base(path), isDir: true, mode: fs.ModeDir, size: 0}
	}
	if file, isFile := m.files[path]; isFile {
		return &mockDirEntry{
			name:  filepath.Base(path),
			isDir: false,
			mode:  file.mode,
			size:  int64(len(file.content)),
		}
	}
	return nil
}

// mockReadCloser implements io.ReadCloser for testing purposes.
type mockReadCloser struct {
	// data holds the bytes to be read.
	data []byte

	// offset is the current read position in data.
	offset int
}

// Read implements io.Reader.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes copied into p.
// Returns err (error) which is io.EOF when all data has been read.
func (r *mockReadCloser) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

// Close implements io.Closer.
//
// Returns error when the close operation fails. Always returns nil in this
// mock implementation.
func (*mockReadCloser) Close() error {
	return nil
}

// mockDirEntry implements fs.DirEntry for testing purposes.
type mockDirEntry struct {
	// name is the base name of the file or directory.
	name string

	// isDir is true if the entry is a directory.
	isDir bool

	// mode is the file mode bits returned by the Type method.
	mode fs.FileMode

	// size is the file size in bytes.
	size int64
}

// Name returns the name of the file or directory.
//
// Returns string which is the base name of the entry.
func (e *mockDirEntry) Name() string { return e.name }

// IsDir reports whether the entry describes a directory.
//
// Returns bool which is true if the entry is a directory.
func (e *mockDirEntry) IsDir() bool { return e.isDir }

// Type returns the type bits for the entry.
//
// Returns fs.FileMode which contains only the type bits of the file mode.
func (e *mockDirEntry) Type() fs.FileMode { return e.mode.Type() }

// Info returns the file information for this directory entry.
//
// Returns fs.FileInfo which describes the file or subdirectory.
// Returns error when the file information cannot be retrieved.
func (*mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// mockFileInfo implements fs.FileInfo for testing purposes.
type mockFileInfo struct {
	// modTime is the mock file's last modification time.
	modTime time.Time

	// name is the base name of the file.
	name string

	// size is the file size in bytes.
	size int64

	// mode holds the file mode and permission bits for the mock.
	mode fs.FileMode

	// isDir indicates whether this mock represents a directory.
	isDir bool
}

// Name returns the base name of the file.
//
// Returns string which is the file's base name.
func (fi *mockFileInfo) Name() string { return fi.name }

// Size returns the length of the file in bytes.
//
// Returns int64 which is the file size.
func (fi *mockFileInfo) Size() int64 { return fi.size }

// Mode returns the file mode bits.
//
// Returns fs.FileMode which holds the file's mode and permission bits.
func (fi *mockFileInfo) Mode() fs.FileMode { return fi.mode }

// ModTime returns the modification time.
//
// Returns time.Time which is the last modification time of the file.
func (fi *mockFileInfo) ModTime() time.Time { return fi.modTime }

// IsDir reports whether this file info represents a directory.
//
// Returns bool which is true if it is a directory.
func (fi *mockFileInfo) IsDir() bool { return fi.isDir }

// Sys returns the underlying data source.
//
// Returns any which is always nil for mock file info.
func (*mockFileInfo) Sys() any { return nil }
