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

package safedisk

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// mockFilePermission is the default file permission for mock files.
	mockFilePermission fs.FileMode = 0o644

	// flagWrite is the flag bit for write-only file access.
	flagWrite = 0x1

	// flagReadWrite is the flag bit for read-write file access mode.
	flagReadWrite = 0x2

	// flagCreate is the flag for creating a file if it does not exist.
	flagCreate = 0x40

	// flagTrunc is the file truncate flag for clearing file contents on open.
	flagTrunc = 0x200

	// flagAppend is the file append flag for opening files in append mode.
	flagAppend = 0x400
)

// walkAction represents the action to take after processing a path.
type walkAction int

const (
	// walkContinue signals that the walk should proceed to the next path.
	walkContinue walkAction = iota

	// walkSkipDir signals that the current directory should be skipped.
	walkSkipDir

	// walkStop signals that the walk should stop entirely.
	walkStop

	// walkError signals that the walk should stop and return an error.
	walkError
)

// MockFileHandle provides a mock implementation of FileHandle for testing.
// It allows configuring errors for specific operations to test error paths.
type MockFileHandle struct {
	// ReadErr is an error injection field; set to non-nil to make Read fail.
	ReadErr error

	// ReadAtErr is the error to return from ReadAt; nil means success.
	ReadAtErr error

	// WriteErr is the error to return from Write calls; nil means success.
	WriteErr error

	// WriteAtErr is the error to return from WriteAt; nil means success.
	WriteAtErr error

	// SeekErr error // SeekErr is returned by Seek when set; nil means no error.
	SeekErr error

	// SyncErr is returned by Sync when set; nil means Sync succeeds.
	SyncErr error

	// TruncateErr is the error to return from Truncate; nil means success.
	TruncateErr error

	// StatErr error // StatErr is returned by Stat when set; nil means Stat succeeds.
	StatErr error

	// ChmodErr is returned by Chmod when set; nil means Chmod succeeds.
	ChmodErr error

	// CloseErr is the error to return from Close; nil means success.
	CloseErr error

	// ReadDirErr is the error to return from ReadDir; nil means success.
	ReadDirErr error

	// StatInfo holds custom file info to return from Stat; nil uses default info.
	StatInfo fs.FileInfo

	// data holds the file contents as a buffer.
	data *bytes.Buffer

	// name is the relative path of the file within the sandbox.
	name string

	// absolutePath is the full path to the file on disk.
	absolutePath string

	// DirEntries holds custom directory entries to return from ReadDir.
	DirEntries []fs.DirEntry

	// offset tracks the current read and write position in the buffer.
	// Accessed atomically.
	offset int64

	// closed indicates whether the file handle has been closed
	// (0 open, 1 closed), accessed atomically.
	closed int64
}

var (
	_ FileHandle = (*MockFileHandle)(nil)

	_ Sandbox = (*MockSandbox)(nil)
)

// NewMockFileHandle creates a new mock file handle with the given name and
// initial contents.
//
// Takes name (string) which specifies the display name of the file handle.
// Takes absolutePath (string) which specifies the full path to the file.
// Takes initialData ([]byte) which provides the initial file contents.
//
// Returns *MockFileHandle which is ready for use in tests.
func NewMockFileHandle(name, absolutePath string, initialData []byte) *MockFileHandle {
	return &MockFileHandle{
		name:         name,
		absolutePath: absolutePath,
		data:         bytes.NewBuffer(initialData),
	}
}

// Name returns the relative path of the file within the sandbox.
//
// Returns string which is the file's path relative to the sandbox root.
func (m *MockFileHandle) Name() string {
	return m.name
}

// AbsolutePath returns the absolute path of the file on disk.
//
// Returns string which is the full path to the file.
func (m *MockFileHandle) AbsolutePath() string {
	return m.absolutePath
}

// Read reads up to len(p) bytes into p.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when ReadErr is set or the end of file is reached.
func (m *MockFileHandle) Read(p []byte) (n int, err error) {
	if m.ReadErr != nil {
		return 0, m.ReadErr
	}

	data := m.data.Bytes()
	off := atomic.LoadInt64(&m.offset)
	if off >= int64(len(data)) {
		return 0, io.EOF
	}

	n = copy(p, data[off:])
	atomic.AddInt64(&m.offset, int64(n))
	return n, nil
}

// ReadAt reads len(p) bytes starting at offset off.
//
// Takes p ([]byte) which is the buffer to read data into.
// Takes off (int64) which is the byte offset to start reading from.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when ReadAtErr is set on the mock.
func (m *MockFileHandle) ReadAt(p []byte, off int64) (n int, err error) {
	if m.ReadAtErr != nil {
		return 0, m.ReadAtErr
	}

	data := m.data.Bytes()
	if off >= int64(len(data)) {
		return 0, nil
	}

	n = copy(p, data[off:])
	return n, nil
}

// Write writes len(p) bytes from p to the file.
//
// Takes p ([]byte) which contains the data to write.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when WriteErr is set on the mock.
func (m *MockFileHandle) Write(p []byte) (n int, err error) {
	if m.WriteErr != nil {
		return 0, m.WriteErr
	}

	off := atomic.LoadInt64(&m.offset)
	data := m.data.Bytes()
	newLen := int(off) + len(p)
	if newLen > len(data) {
		newData := make([]byte, newLen)
		copy(newData, data)
		copy(newData[off:], p)
		m.data = bytes.NewBuffer(newData)
	} else {
		copy(data[off:], p)
	}

	n = len(p)
	atomic.AddInt64(&m.offset, int64(n))
	return n, nil
}

// WriteAt writes len(p) bytes starting at offset off.
//
// Takes p ([]byte) which contains the data to write.
// Takes off (int64) which is the byte offset to start writing at.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when WriteAtErr is set on the mock.
func (m *MockFileHandle) WriteAt(p []byte, off int64) (n int, err error) {
	if m.WriteAtErr != nil {
		return 0, m.WriteAtErr
	}

	data := m.data.Bytes()
	newLen := int(off) + len(p)
	if newLen > len(data) {
		newData := make([]byte, newLen)
		copy(newData, data)
		copy(newData[off:], p)
		m.data = bytes.NewBuffer(newData)
	} else {
		copy(data[off:], p)
	}

	return len(p), nil
}

// WriteString writes a string to the file.
//
// Takes s (string) which is the text to write.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when WriteErr is set on the mock.
func (m *MockFileHandle) WriteString(s string) (n int, err error) {
	return m.Write([]byte(s)) //nolint:gocritic // WriteString would recurse
}

// Seek sets the offset for the next Read or Write.
//
// Takes offset (int64) which specifies the position relative to whence.
// Takes whence (int) which indicates the reference point: 0 for start, 1 for
// current position, 2 for end.
//
// Returns int64 which is the new offset position.
// Returns error when SeekErr is set on the mock.
func (m *MockFileHandle) Seek(offset int64, whence int) (int64, error) {
	if m.SeekErr != nil {
		return 0, m.SeekErr
	}

	var newOffset int64
	switch whence {
	case 0:
		newOffset = offset
	case 1:
		newOffset = atomic.LoadInt64(&m.offset) + offset
	case 2:
		newOffset = int64(m.data.Len()) + offset
	}

	if newOffset < 0 {
		newOffset = 0
	}
	atomic.StoreInt64(&m.offset, newOffset)
	return newOffset, nil
}

// Sync saves the file's contents to stable storage.
//
// Returns error when SyncErr is set on the mock.
func (m *MockFileHandle) Sync() error {
	if m.SyncErr != nil {
		return m.SyncErr
	}
	return nil
}

// Truncate changes the size of the file.
//
// Takes size (int64) which specifies the new file size in bytes.
//
// Returns error when TruncateErr is set on the mock.
func (m *MockFileHandle) Truncate(size int64) error {
	if m.TruncateErr != nil {
		return m.TruncateErr
	}

	data := m.data.Bytes()
	if size < int64(len(data)) {
		m.data = bytes.NewBuffer(data[:size])
	} else if size > int64(len(data)) {
		newData := make([]byte, size)
		copy(newData, data)
		m.data = bytes.NewBuffer(newData)
	}
	return nil
}

// Stat returns information about the file.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when StatErr is set on the mock.
func (m *MockFileHandle) Stat() (fs.FileInfo, error) {
	if m.StatErr != nil {
		return nil, m.StatErr
	}

	if m.StatInfo != nil {
		return m.StatInfo, nil
	}

	return &mockFileInfo{
		name:    m.name,
		size:    int64(m.data.Len()),
		mode:    mockFilePermission,
		modTime: time.Now(),
		isDir:   false,
	}, nil
}

// Chmod changes the file's permissions.
//
// Takes mode (fs.FileMode) which specifies the new permission bits.
//
// Returns error when ChmodErr is set on the mock.
func (m *MockFileHandle) Chmod(_ fs.FileMode) error {
	if m.ChmodErr != nil {
		return m.ChmodErr
	}
	return nil
}

// Close releases all resources held by the file.
//
// Returns error when CloseErr is set on the mock.
func (m *MockFileHandle) Close() error {
	if m.CloseErr != nil {
		return m.CloseErr
	}

	atomic.StoreInt64(&m.closed, 1)
	return nil
}

// ReadDir reads the directory contents (if this is a directory).
//
// Takes n (int) which limits the number of entries returned.
//
// Returns []fs.DirEntry which contains the mock directory entries.
// Returns error when ReadDirErr is set on the mock.
func (m *MockFileHandle) ReadDir(n int) ([]fs.DirEntry, error) {
	if m.ReadDirErr != nil {
		return nil, m.ReadDirErr
	}

	if m.DirEntries != nil {
		if n <= 0 || n > len(m.DirEntries) {
			return m.DirEntries, nil
		}
		return m.DirEntries[:n], nil
	}

	return nil, nil
}

// Fd returns the underlying file descriptor.
//
// Returns uintptr which is always zero for mock handles.
func (*MockFileHandle) Fd() uintptr {
	return 0
}

// Data returns the current contents of the mock file.
//
// Returns []byte which contains the data written to the mock file.
func (m *MockFileHandle) Data() []byte {
	return m.data.Bytes()
}

// IsClosed returns whether the file has been closed.
//
// Returns bool which is true if the file handle has been closed.
func (m *MockFileHandle) IsClosed() bool {
	return atomic.LoadInt64(&m.closed) == 1
}

// mockFileInfo implements fs.FileInfo for MockFileHandle.
type mockFileInfo struct {
	// modTime is the last modification time of the file.
	modTime time.Time

	// name is the base name of the file.
	name string

	// size is the file size in bytes.
	size int64

	// mode fs.FileMode // mode holds the file's mode and permission bits.
	mode fs.FileMode

	// isDir indicates whether this mock represents a directory.
	isDir bool
}

// Name returns the file's base name.
//
// Returns string which is the name of the file.
func (m *mockFileInfo) Name() string { return m.name }

// Size returns the file size in bytes.
//
// Returns int64 which is the mock file size value.
func (m *mockFileInfo) Size() int64 { return m.size }

// Mode returns the file mode bits.
//
// Returns fs.FileMode which contains the permission and type bits.
func (m *mockFileInfo) Mode() fs.FileMode { return m.mode }

// ModTime returns the modification time of the file.
//
// Returns time.Time which is the last modification timestamp.
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }

// IsDir returns whether the file is a directory.
//
// Returns bool which is true if this represents a directory.
func (m *mockFileInfo) IsDir() bool { return m.isDir }

// Sys returns the underlying data source.
//
// Returns any which is always nil for this mock implementation.
func (*mockFileInfo) Sys() any { return nil }

// mockDirEntry implements fs.DirEntry for testing WalkDir.
type mockDirEntry struct {
	// info holds the file metadata returned by the Info method.
	info fs.FileInfo

	// name is the base name of the directory entry.
	name string

	// isDir indicates whether this entry represents a directory.
	isDir bool
}

// Name returns the directory entry's name.
//
// Returns string which is the base name of the file or directory.
func (m *mockDirEntry) Name() string { return m.name }

// IsDir returns whether the entry represents a directory.
//
// Returns bool which is true if the entry is a directory.
func (m *mockDirEntry) IsDir() bool { return m.isDir }

// Type returns the file mode type bits for this directory entry.
//
// Returns fs.FileMode which contains only the type portion of the file mode.
func (m *mockDirEntry) Type() fs.FileMode { return m.info.Mode().Type() }

// Info returns the file information for this directory entry.
//
// Returns fs.FileInfo which provides the underlying file metadata.
// Returns error when the file information cannot be retrieved.
func (m *mockDirEntry) Info() (fs.FileInfo, error) { return m.info, nil }

// MockSandbox provides a mock implementation of Sandbox for testing.
// It uses an in-memory file system and allows configuring errors.
type MockSandbox struct {
	// OpenErr is the error to return from Open; nil means success.
	OpenErr error

	// ReadFileErr is returned by ReadFile when set.
	ReadFileErr error

	// StatErr is the error to return from Stat; nil means success.
	StatErr error

	// LstatErr is returned by Lstat when set; nil means Lstat succeeds.
	LstatErr error

	// ReadDirErr is the error to return from ReadDir; nil means success.
	ReadDirErr error

	// WalkDirErr is the error to return from WalkDir; nil means success.
	WalkDirErr error

	// CreateErr error // CreateErr is returned by Create when set.
	CreateErr error

	// OpenFileErr is returned by OpenFile when set; nil means no error.
	OpenFileErr error

	// WriteFileErr is returned by WriteFile when set; nil allows normal behaviour.
	WriteFileErr error

	// WriteFileAtomicErr is returned by WriteFileAtomic when set.
	WriteFileAtomicErr error

	// MkdirErr error // MkdirErr is returned by Mkdir when set; nil means success.
	MkdirErr error

	// MkdirAllErr is returned by MkdirAll when set; nil means success.
	MkdirAllErr error

	// RemoveErr is returned by Remove when set; nil means success.
	RemoveErr error

	// RemoveAllErr is returned by RemoveAll when set; nil means success.
	RemoveAllErr error

	// RenameErr is returned by Rename when set; nil allows normal behaviour.
	RenameErr error

	// ChmodErr is returned by Chmod when set; nil means Chmod succeeds.
	ChmodErr error

	// CreateTempErr is returned by CreateTemp when set to a non-nil value.
	CreateTempErr error

	// MkdirTempErr is returned by MkdirTemp when set; nil means success.
	MkdirTempErr error

	// CloseErr is the error to return from Close; nil means success.
	CloseErr error

	// NextTempFileWriteErr injects a write error into the next temp file
	// created by CreateTemp. It is cleared after use to allow single-use
	// error injection.
	NextTempFileWriteErr error

	// NextTempFileSyncErr is the error to return from Sync on the next temp file.
	NextTempFileSyncErr error

	// NextTempFileCloseErr is the error to return from Close on the next
	// temporary file created; nil means no error.
	NextTempFileCloseErr error

	// files stores the mock files by their path.
	files map[string]*MockFileHandle

	// ChmodFunc sets a custom Chmod function for testing.
	ChmodFunc func(name string, mode fs.FileMode) error

	// CallCounts tracks how many times each method was called.
	CallCounts map[string]int

	// root is the base path for the mock sandbox.
	root string

	// mu guards access to files and CallCounts.
	mu sync.RWMutex

	// mode is the sandbox access mode; ModeReadOnly blocks write operations.
	mode Mode
}

// NewMockSandbox creates a new mock sandbox with the given root and mode.
//
// Takes root (string) which specifies the root directory path for the sandbox.
// Takes mode (Mode) which sets the sandbox operating mode.
//
// Returns *MockSandbox which is the initialised mock sandbox ready for use.
func NewMockSandbox(root string, mode Mode) *MockSandbox {
	return &MockSandbox{
		root:       root,
		mode:       mode,
		files:      make(map[string]*MockFileHandle),
		CallCounts: make(map[string]int),
	}
}

// AddFile adds a mock file to the sandbox with the given contents.
//
// Takes name (string) which specifies the file name.
// Takes data ([]byte) which provides the file contents.
//
// Returns *MockFileHandle which is the handle for the added file.
//
// Safe for concurrent use.
func (m *MockSandbox) AddFile(name string, data []byte) *MockFileHandle {
	m.mu.Lock()
	defer m.mu.Unlock()

	file := NewMockFileHandle(name, m.root+"/"+name, data)
	m.files[name] = file
	return file
}

// GetFile retrieves a mock file from the sandbox.
//
// Takes name (string) which specifies the file to retrieve.
//
// Returns *MockFileHandle which is the file handle, or nil if not found.
//
// Safe for concurrent use.
func (m *MockSandbox) GetFile(name string) *MockFileHandle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.files[name]
}

// Open opens a file for reading within the sandbox.
//
// Takes name (string) which specifies the file path to open.
//
// Returns FileHandle which provides access to the file contents.
// Returns error when the file does not exist or an error is configured.
//
// Safe for concurrent use.
func (m *MockSandbox) Open(name string) (FileHandle, error) {
	m.incrementCall("Open")

	if m.OpenErr != nil {
		return nil, m.OpenErr
	}

	m.mu.RLock()
	file, exists := m.files[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fs.ErrNotExist
	}

	return NewMockFileHandle(file.name, file.absolutePath, file.Data()), nil
}

// ReadFile reads the entire contents of a file within the sandbox.
//
// Takes name (string) which is the path of the file to read.
//
// Returns []byte which contains the file contents.
// Returns error when ReadFileErr is set or the file does not exist.
//
// Safe for concurrent use.
func (m *MockSandbox) ReadFile(name string) ([]byte, error) {
	m.incrementCall("ReadFile")

	if m.ReadFileErr != nil {
		return nil, m.ReadFileErr
	}

	m.mu.RLock()
	file, exists := m.files[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fs.ErrNotExist
	}

	return file.Data(), nil
}

// Stat returns file information for a path within the sandbox.
//
// Takes name (string) which is the path to the file within the sandbox.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the file does not exist or stat fails.
//
// Safe for concurrent use.
func (m *MockSandbox) Stat(name string) (fs.FileInfo, error) {
	m.incrementCall("Stat")

	if m.StatErr != nil {
		return nil, m.StatErr
	}

	m.mu.RLock()
	file, exists := m.files[name]
	m.mu.RUnlock()

	if !exists {
		return nil, fs.ErrNotExist
	}

	return file.Stat()
}

// Lstat returns file information without following symlinks.
//
// Takes name (string) which specifies the path to query.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when LstatErr is set or the underlying Stat call fails.
func (m *MockSandbox) Lstat(name string) (fs.FileInfo, error) {
	m.incrementCall("Lstat")

	if m.LstatErr != nil {
		return nil, m.LstatErr
	}

	return m.Stat(name)
}

// ReadDir reads the contents of a directory within the sandbox.
//
// Returns []fs.DirEntry which contains the directory entries found.
// Returns error when ReadDirErr is set on the mock.
func (m *MockSandbox) ReadDir(_ string) ([]fs.DirEntry, error) {
	m.incrementCall("ReadDir")

	if m.ReadDirErr != nil {
		return nil, m.ReadDirErr
	}

	return nil, nil
}

// WalkDir walks the directory tree rooted at root within the sandbox.
//
// Takes root (string) which specifies the starting directory path.
// Takes walkFunction (fs.WalkDirFunc) which is called for each file and directory.
//
// Returns error when the walk function returns an error or WalkDirErr is set.
func (m *MockSandbox) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	m.incrementCall("WalkDir")

	if m.WalkDirErr != nil {
		return m.WalkDirErr
	}

	paths := m.collectMatchingPaths(root)
	sortStrings(paths)

	var skipPrefixes []string
	for _, path := range paths {
		if isPathSkipped(path, skipPrefixes) {
			continue
		}

		result := m.walkSinglePath(path, walkFunction)
		switch result.action {
		case walkContinue:
			continue
		case walkSkipDir:
			if result.isDir {
				skipPrefixes = append(skipPrefixes, path)
			}
		case walkStop:
			return nil
		case walkError:
			return result.err
		}
	}

	return nil
}

// Create creates or truncates a file for writing.
//
// Takes name (string) which specifies the file path to create.
//
// Returns FileHandle which provides access to the created file.
// Returns error when CreateErr is set or the sandbox is in read-only mode.
//
// Safe for concurrent use.
func (m *MockSandbox) Create(name string) (FileHandle, error) {
	m.incrementCall("Create")

	if m.CreateErr != nil {
		return nil, m.CreateErr
	}

	if m.mode == ModeReadOnly {
		return nil, errReadOnly
	}

	file := NewMockFileHandle(name, m.root+"/"+name, nil)
	m.mu.Lock()
	m.files[name] = file
	m.mu.Unlock()

	return file, nil
}

// walkResult holds the outcome of processing a single path during a walk.
type walkResult struct {
	// err error // err holds the error encountered during directory walking.
	err error

	// action indicates the next step for the directory walk.
	action walkAction

	// isDir indicates whether the path is a directory.
	isDir bool
}

// OpenFile opens a file with the specified flags and permissions.
//
// Takes name (string) which specifies the file path to open.
// Takes flag (int) which specifies the file open flags.
//
// Returns FileHandle which provides access to the opened file.
// Returns error when OpenFileErr is set, the sandbox is read-only, or the
// file does not exist and create was not requested.
//
// Safe for concurrent use.
func (m *MockSandbox) OpenFile(name string, flag int, _ fs.FileMode) (FileHandle, error) {
	m.incrementCall("OpenFile")

	if m.OpenFileErr != nil {
		return nil, m.OpenFileErr
	}

	writeFlags := flagWrite | flagReadWrite | flagCreate | flagTrunc | flagAppend
	if m.mode == ModeReadOnly && (flag&writeFlags) != 0 {
		return nil, errReadOnly
	}

	m.mu.RLock()
	file, exists := m.files[name]
	m.mu.RUnlock()

	if !exists {
		if flag&flagCreate == 0 {
			return nil, fs.ErrNotExist
		}
		file = NewMockFileHandle(name, m.root+"/"+name, nil)
		m.mu.Lock()
		m.files[name] = file
		m.mu.Unlock()
	}

	return NewMockFileHandle(file.name, file.absolutePath, file.Data()), nil
}

// WriteFile writes data to a file, creating it if necessary.
//
// Takes name (string) which specifies the file path to write.
// Takes data ([]byte) which contains the content to write.
//
// Returns error when WriteFileErr is set or the sandbox is in read-only
// mode.
//
// Safe for concurrent use.
func (m *MockSandbox) WriteFile(name string, data []byte, _ fs.FileMode) error {
	m.incrementCall("WriteFile")

	if m.WriteFileErr != nil {
		return m.WriteFileErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	file := NewMockFileHandle(name, m.root+"/"+name, data)
	m.mu.Lock()
	m.files[name] = file
	m.mu.Unlock()

	return nil
}

// WriteFileAtomic writes data to a file atomically.
//
// Takes name (string) which specifies the file path to write.
// Takes data ([]byte) which contains the content to write.
//
// Returns error when WriteFileAtomicErr is set.
func (m *MockSandbox) WriteFileAtomic(name string, data []byte, _ fs.FileMode) error {
	m.incrementCall("WriteFileAtomic")

	if m.WriteFileAtomicErr != nil {
		return m.WriteFileAtomicErr
	}

	return m.WriteFile(name, data, 0)
}

// Mkdir creates a directory within the sandbox.
//
// Returns error when MkdirErr is set or the sandbox is in read-only mode.
func (m *MockSandbox) Mkdir(_ string, _ fs.FileMode) error {
	m.incrementCall("Mkdir")

	if m.MkdirErr != nil {
		return m.MkdirErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	return nil
}

// MkdirAll creates a directory and all necessary parent directories.
//
// Takes path (string) which specifies the directory path to create.
// Takes perm (fs.FileMode) which sets the permission bits for created
// directories.
//
// Returns error when MkdirAllErr is set or the sandbox is in read-only mode.
//
// Safe for concurrent use; protects file creation with a mutex.
func (m *MockSandbox) MkdirAll(path string, perm fs.FileMode) error {
	m.incrementCall("MkdirAll")

	if m.MkdirAllErr != nil {
		return m.MkdirAllErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	parts := splitPath(path)
	current := ""
	m.mu.Lock()
	for _, part := range parts {
		if current == "" {
			current = part
		} else {
			current = current + "/" + part
		}
		if _, exists := m.files[current]; !exists {
			file := NewMockFileHandle(current, m.root+"/"+current, nil)
			file.StatInfo = &mockFileInfo{
				name:    current,
				size:    0,
				mode:    perm | fs.ModeDir,
				modTime: time.Now(),
				isDir:   true,
			}
			m.files[current] = file
		}
	}
	m.mu.Unlock()

	return nil
}

// Remove deletes a file or empty directory within the sandbox.
//
// Takes name (string) which specifies the path to remove.
//
// Returns error when RemoveErr is set or the sandbox is in read-only mode.
//
// Safe for concurrent use; protects file map access with a mutex.
func (m *MockSandbox) Remove(name string) error {
	m.incrementCall("Remove")

	if m.RemoveErr != nil {
		return m.RemoveErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	m.mu.Lock()
	delete(m.files, name)
	m.mu.Unlock()

	return nil
}

// RemoveAll deletes a path and all its children within the sandbox.
//
// Takes path (string) which specifies the root path to remove.
//
// Returns error when removal fails or the sandbox is in read-only mode.
//
// Safe for concurrent use. Holds the mutex while modifying the file map.
func (m *MockSandbox) RemoveAll(path string) error {
	m.incrementCall("RemoveAll")

	if m.RemoveAllErr != nil {
		return m.RemoveAllErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	m.mu.Lock()
	for name := range m.files {
		if len(name) >= len(path) && name[:len(path)] == path {
			delete(m.files, name)
		}
	}
	m.mu.Unlock()

	return nil
}

// Rename renames a file or directory within the sandbox.
//
// Takes oldpath (string) which is the current path of the file or directory.
// Takes newpath (string) which is the new path for the file or directory.
//
// Returns error when the sandbox is in read-only mode or the old path does
// not exist.
//
// Safe for concurrent use; protects file map access with a mutex.
func (m *MockSandbox) Rename(oldpath, newpath string) error {
	m.incrementCall("Rename")

	if m.RenameErr != nil {
		return m.RenameErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	file, exists := m.files[oldpath]
	if !exists {
		return fs.ErrNotExist
	}

	delete(m.files, oldpath)
	file.name = newpath
	file.absolutePath = m.root + "/" + newpath
	m.files[newpath] = file

	return nil
}

// Chmod changes the permissions of a file within the sandbox.
//
// Takes name (string) which specifies the path to the file.
// Takes mode (fs.FileMode) which specifies the new file permissions.
//
// Returns error when ChmodErr is set, the sandbox is read-only, or ChmodFunc
// returns an error.
func (m *MockSandbox) Chmod(name string, mode fs.FileMode) error {
	m.incrementCall("Chmod")

	if m.ChmodErr != nil {
		return m.ChmodErr
	}

	if m.mode == ModeReadOnly {
		return errReadOnly
	}

	if m.ChmodFunc != nil {
		return m.ChmodFunc(name, mode)
	}

	return nil
}

// CreateTemp creates a temporary file within the sandbox.
//
// Takes directory (string) which specifies the directory for the temporary file.
// Takes pattern (string) which specifies the filename pattern to use.
//
// Returns FileHandle which provides access to the created temporary file.
// Returns error when CreateTempErr is set or the sandbox is in read-only mode.
//
// Safe for concurrent use; protects file handle storage with a mutex.
func (m *MockSandbox) CreateTemp(directory, pattern string) (FileHandle, error) {
	m.incrementCall("CreateTemp")

	if m.CreateTempErr != nil {
		return nil, m.CreateTempErr
	}

	if m.mode == ModeReadOnly {
		return nil, errReadOnly
	}

	name := directory + "/temp-" + pattern + "-12345"
	file := NewMockFileHandle(name, m.root+"/"+name, nil)

	m.mu.Lock()
	if m.NextTempFileWriteErr != nil {
		file.WriteErr = m.NextTempFileWriteErr
		m.NextTempFileWriteErr = nil
	}
	if m.NextTempFileSyncErr != nil {
		file.SyncErr = m.NextTempFileSyncErr
		m.NextTempFileSyncErr = nil
	}
	if m.NextTempFileCloseErr != nil {
		file.CloseErr = m.NextTempFileCloseErr
		m.NextTempFileCloseErr = nil
	}
	m.files[name] = file
	m.mu.Unlock()

	return file, nil
}

// MkdirTemp creates a temporary directory within the sandbox.
//
// Takes directory (string) which specifies the parent directory
// for the temp folder.
// Takes pattern (string) which provides a prefix for the directory name.
//
// Returns string which is the path to the created temporary directory.
// Returns error when MkdirTempErr is set or the sandbox is in read-only mode.
func (m *MockSandbox) MkdirTemp(directory, pattern string) (string, error) {
	m.incrementCall("MkdirTemp")

	if m.MkdirTempErr != nil {
		return "", m.MkdirTempErr
	}

	if m.mode == ModeReadOnly {
		return "", errReadOnly
	}

	return directory + "/temp-" + pattern + "-12345", nil
}

// Root returns the absolute path of the sandbox root directory.
//
// Returns string which is the absolute path to the sandbox root.
func (m *MockSandbox) Root() string {
	return m.root
}

// Mode returns whether this sandbox allows file changes.
//
// Returns Mode which indicates the current sandbox mode.
func (m *MockSandbox) Mode() Mode {
	return m.mode
}

// IsReadOnly returns true if the sandbox does not allow write operations.
//
// Returns bool which is true when the sandbox is in read-only mode.
func (m *MockSandbox) IsReadOnly() bool {
	return m.mode == ModeReadOnly
}

// Close releases any resources held by the sandbox.
//
// Returns error when CloseErr is set on the mock.
func (m *MockSandbox) Close() error {
	m.incrementCall("Close")

	if m.CloseErr != nil {
		return m.CloseErr
	}

	return nil
}

// RelPath converts a path to a sandbox-relative path.
//
// Takes path (string) which is the absolute or relative path to convert.
//
// Returns string which is the path relative to the sandbox root.
func (*MockSandbox) RelPath(path string) string {
	return path
}

// incrementCall records that the given method was called.
//
// Takes method (string) which is the name of the method to record.
//
// Safe for concurrent use.
func (m *MockSandbox) incrementCall(method string) {
	m.mu.Lock()
	m.CallCounts[method]++
	m.mu.Unlock()
}

// collectMatchingPaths returns all paths that match the root prefix.
//
// Takes root (string) which specifies the path prefix to match against.
//
// Returns []string which contains all matching file paths.
//
// Safe for concurrent use. Uses a read lock to protect access to the
// files map.
func (m *MockSandbox) collectMatchingPaths(root string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var paths []string
	for path := range m.files {
		if matchesRoot(path, root) {
			paths = append(paths, path)
		}
	}
	return paths
}

// walkSinglePath processes a single path during WalkDir.
//
// Takes path (string) which specifies the file path to process.
// Takes walkFunction (fs.WalkDirFunc) which handles each visited entry.
//
// Returns walkResult which indicates whether to continue or stop walking.
//
// Safe for concurrent use. Uses a read lock when accessing the files map.
func (m *MockSandbox) walkSinglePath(path string, walkFunction fs.WalkDirFunc) walkResult {
	m.mu.RLock()
	file, exists := m.files[path]
	m.mu.RUnlock()

	if !exists {
		return walkResult{action: walkContinue}
	}

	info, err := file.Stat()
	if err != nil {
		return m.handleStatError(path, err, walkFunction)
	}

	entry := &mockDirEntry{name: path, isDir: info.IsDir(), info: info}
	if err := walkFunction(path, entry, nil); err != nil {
		return handleWalkError(err, info.IsDir())
	}

	return walkResult{action: walkContinue}
}

// handleStatError handles an error from Stat during WalkDir.
//
// Takes path (string) which is the file path that failed to stat.
// Takes statErr (error) which is the error returned by the Stat call.
// Takes walkFunction (fs.WalkDirFunc) which is the callback to notify
// of the error.
//
// Returns walkResult which indicates whether to continue or stop walking.
func (*MockSandbox) handleStatError(path string, statErr error, walkFunction fs.WalkDirFunc) walkResult {
	walkErr := walkFunction(path, nil, statErr)
	if walkErr == nil {
		return walkResult{action: walkContinue}
	}
	if errors.Is(walkErr, fs.SkipDir) || errors.Is(walkErr, fs.SkipAll) {
		return walkResult{action: walkContinue}
	}
	return walkResult{action: walkError, err: walkErr}
}

// sortStrings sorts a slice of strings in place.
//
// Takes s ([]string) which is the slice to sort.
func sortStrings(s []string) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// matchesRoot checks if a path matches the given root.
//
// Takes path (string) which is the file path to check.
// Takes root (string) which is the root directory to match against.
//
// Returns bool which is true if path equals root, starts with root followed
// by a separator, or if root is empty or ".".
func matchesRoot(path, root string) bool {
	if root == "." || root == "" {
		return true
	}
	if path == root {
		return true
	}
	return len(path) > len(root) && path[:len(root)] == root && path[len(root)] == '/'
}

// isPathSkipped checks if a path should be skipped due to a parent skip.
//
// Takes path (string) which is the path to check.
// Takes skipPrefixes ([]string) which contains the prefixes to match against.
//
// Returns bool which is true if the path starts with any skip prefix.
func isPathSkipped(path string, skipPrefixes []string) bool {
	for _, prefix := range skipPrefixes {
		if len(path) > len(prefix) && path[:len(prefix)] == prefix && path[len(prefix)] == '/' {
			return true
		}
	}
	return false
}

// handleWalkError handles an error returned from the WalkDirFunc.
//
// Takes err (error) which is the error returned by the walk function.
// Takes isDir (bool) which indicates whether the current path is a directory.
//
// Returns walkResult which specifies the action to take based on the error.
func handleWalkError(err error, isDir bool) walkResult {
	if errors.Is(err, fs.SkipDir) {
		return walkResult{action: walkSkipDir, isDir: isDir}
	}
	if errors.Is(err, fs.SkipAll) {
		return walkResult{action: walkStop}
	}
	return walkResult{action: walkError, err: err}
}

// splitPath splits a path into its component parts.
//
// Takes path (string) which is the file path to split.
//
// Returns []string which contains the individual path segments.
func splitPath(path string) []string {
	if path == "" || path == "." {
		return []string{"."}
	}
	var parts []string
	start := 0
	for i := range len(path) {
		if path[i] == '/' {
			if start < i {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
