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
	"io"
	"io/fs"
	"os"
	"time"
)

// Mode specifies whether a sandbox allows write operations.
// It implements fmt.Stringer.
type Mode uint8

const (
	// ModeReadOnly is a sandbox mode that allows only read operations such as
	// Open, ReadFile, and Stat.
	ModeReadOnly Mode = iota

	// ModeReadWrite allows both read and write operations within the sandbox.
	ModeReadWrite
)

var (
	_ io.Reader = (*File)(nil)

	_ io.ReaderAt = (*File)(nil)

	_ io.Writer = (*File)(nil)

	_ io.WriterAt = (*File)(nil)

	_ io.Seeker = (*File)(nil)

	_ io.Closer = (*File)(nil)

	_ fs.ReadDirFile = (*File)(nil)

	_ FileHandle = (*File)(nil)
)

var _ fs.FileInfo = fileInfo{}

// String returns the string representation of the mode.
//
// Returns string which is "read-only" for ModeReadOnly, or "read-write"
// otherwise.
func (m Mode) String() string {
	if m == ModeReadOnly {
		return "read-only"
	}
	return "read-write"
}

// FileHandle provides an interface for sandboxed file operations.
// This interface enables dependency injection and testing with mock
// implementations that can simulate errors for test coverage.
//
// The default implementation is *File, which wraps os.File.
type FileHandle interface {
	// Name returns the relative path of the file within the sandbox.
	Name() string

	// AbsolutePath returns the absolute path of the file on disk.
	// This should only be used for logging/debugging.
	AbsolutePath() string

	// Read reads up to len(p) bytes into p.
	Read(p []byte) (n int, err error)

	// ReadAt reads len(p) bytes starting at offset off.
	ReadAt(p []byte, off int64) (n int, err error)

	// Write writes len(p) bytes from p to the file.
	Write(p []byte) (n int, err error)

	// WriteAt writes len(p) bytes starting at offset off.
	WriteAt(p []byte, off int64) (n int, err error)

	// WriteString writes a string to the file.
	WriteString(s string) (n int, err error)

	// Seek sets the offset for the next Read or Write.
	Seek(offset int64, whence int) (int64, error)

	// Sync saves the file's contents to stable storage.
	Sync() error

	// Truncate changes the size of the file.
	Truncate(size int64) error

	// Stat returns information about the file.
	Stat() (fs.FileInfo, error)

	// Chmod changes the file's permissions.
	Chmod(mode fs.FileMode) error

	// Close releases all resources held by the file.
	Close() error

	// ReadDir reads the directory contents (if this is a directory).
	ReadDir(n int) ([]fs.DirEntry, error)

	// Fd returns the underlying file descriptor.
	Fd() uintptr
}

// Sandbox defines a restricted filesystem interface that confines all
// operations within a root directory. All paths are relative to the sandbox
// root, and attempts to access paths outside the boundary will fail.
type Sandbox interface {
	// Open opens a file for reading within the sandbox.
	// The path must be relative to the sandbox root.
	//
	// Takes name (string) the relative path within the sandbox.
	//
	// Returns FileHandle the opened file handle.
	// Returns error if the file cannot be opened or path escapes sandbox.
	Open(name string) (FileHandle, error)

	// ReadFile reads the entire contents of a file within the sandbox.
	// This is equivalent to Open followed by io.ReadAll.
	//
	// Takes name (string) the relative path within the sandbox.
	//
	// Returns []byte the file contents.
	// Returns error if the file cannot be read or path escapes sandbox.
	ReadFile(name string) ([]byte, error)

	// Stat returns file information for a path within the sandbox.
	//
	// Takes name (string) which is the relative path within the sandbox.
	//
	// Returns fs.FileInfo which contains the file information.
	// Returns error when the path is invalid or escapes the sandbox.
	Stat(name string) (fs.FileInfo, error)

	// Lstat returns file information without following symlinks.
	//
	// Takes name (string) the relative path within the sandbox.
	//
	// Returns fs.FileInfo the file information.
	// Returns error if the path is invalid or escapes sandbox.
	Lstat(name string) (fs.FileInfo, error)

	// ReadDir reads the contents of a directory within the sandbox.
	//
	// Takes name (string) the relative path to the directory.
	//
	// Returns []fs.DirEntry the directory entries.
	// Returns error if the directory cannot be read or path escapes sandbox.
	ReadDir(name string) ([]fs.DirEntry, error)

	// WalkDir walks the directory tree rooted at root within the sandbox.
	// It calls walkFunction for each file or directory in the tree.
	//
	// Takes root (string) the relative path to start walking from.
	// Takes walkFunction (fs.WalkDirFunc) the function to call for each entry.
	//
	// Returns error if walking fails or path escapes sandbox.
	WalkDir(root string, walkFunction fs.WalkDirFunc) error

	// Create creates or truncates a file for writing.
	//
	// Takes name (string) the relative path within the sandbox.
	//
	// Returns FileHandle the created file handle.
	// Returns error if creation fails, path escapes sandbox, or sandbox is read-only.
	Create(name string) (FileHandle, error)

	// OpenFile opens a file with the specified flags and permissions.
	// This is the most flexible file opening function.
	//
	// Takes name (string) which is the relative path within the sandbox.
	// Takes flag (int) which specifies the file open flags (os.O_RDONLY,
	// os.O_WRONLY, etc.).
	// Takes perm (fs.FileMode) which sets the file permissions for creation.
	//
	// Returns FileHandle which is the opened file handle.
	// Returns error when opening fails, path escapes sandbox, or write is
	// attempted on a read-only sandbox.
	OpenFile(name string, flag int, perm fs.FileMode) (FileHandle, error)

	// WriteFile writes data to a file, creating it if necessary.
	// This is equivalent to Create followed by Write and Close.
	//
	// Takes name (string) the relative path within the sandbox.
	// Takes data ([]byte) the content to write.
	// Takes perm (fs.FileMode) the file permissions.
	//
	// Returns error if writing fails, path escapes sandbox, or sandbox is read-only.
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// WriteFileAtomic writes data to a file atomically.
	//
	// The method first writes to a temporary file in the same directory and then
	// renames it to the target path. This means that if the process crashes
	// during the write, the original file remains intact and no partially written
	// file is left behind.
	//
	// Takes name (string) which is the relative path within the sandbox.
	// Takes data ([]byte) which is the content to write.
	// Takes perm (fs.FileMode) which specifies the file permissions.
	//
	// Returns error when writing fails, path escapes the sandbox, or the sandbox
	// is read-only.
	WriteFileAtomic(name string, data []byte, perm fs.FileMode) error

	// Mkdir creates a directory within the sandbox.
	//
	// Takes name (string) the relative path for the new directory.
	// Takes perm (fs.FileMode) the directory permissions.
	//
	// Returns error if creation fails, path escapes sandbox, or sandbox is read-only.
	Mkdir(name string, perm fs.FileMode) error

	// MkdirAll creates a directory and all necessary parent directories.
	//
	// Takes path (string) the relative path for the new directory tree.
	// Takes perm (fs.FileMode) the directory permissions.
	//
	// Returns error if creation fails, path escapes sandbox, or sandbox is read-only.
	MkdirAll(path string, perm fs.FileMode) error

	// Remove deletes a file or empty directory within the sandbox.
	//
	// Takes name (string) the relative path to remove.
	//
	// Returns error if removal fails, path escapes sandbox, or sandbox is read-only.
	Remove(name string) error

	// RemoveAll deletes a path and all its children within the sandbox.
	//
	// Takes path (string) the relative path to remove recursively.
	//
	// Returns error if removal fails, path escapes sandbox, or sandbox is read-only.
	RemoveAll(path string) error

	// Rename renames a file or directory within the sandbox.
	// IMPORTANT: Both oldpath and newpath must be within the same sandbox.
	//
	// Takes oldpath (string) the current relative path.
	// Takes newpath (string) the new relative path.
	//
	// Returns error if renaming fails, paths escape sandbox, or sandbox is read-only.
	Rename(oldpath, newpath string) error

	// Chmod changes the permissions of a file within the sandbox.
	//
	// Takes name (string) the relative path.
	// Takes mode (fs.FileMode) the new permissions.
	//
	// Returns error if changing fails, path escapes sandbox, or sandbox is read-only.
	Chmod(name string, mode fs.FileMode) error

	// CreateTemp creates a temporary file within the sandbox.
	// The file is created in the specified directory with a name
	// generated from the pattern (using * as placeholder for random string).
	//
	// Takes directory (string) the relative directory for the temp file.
	// Takes pattern (string) the filename pattern (e.g., "upload-*.tmp").
	//
	// Returns FileHandle the created temporary file.
	// Returns error if creation fails, path escapes sandbox, or sandbox is read-only.
	CreateTemp(directory, pattern string) (FileHandle, error)

	// MkdirTemp creates a temporary directory within the sandbox.
	//
	// Takes directory (string) the relative parent directory.
	// Takes pattern (string) the directory name pattern.
	//
	// Returns string the path to the created directory.
	// Returns error if creation fails, path escapes sandbox, or sandbox is read-only.
	MkdirTemp(directory, pattern string) (string, error)

	// Root returns the absolute path of the sandbox root directory.
	Root() string

	// Mode returns whether this sandbox allows file changes.
	//
	// Returns Mode which shows if access is read-only or read-write.
	Mode() Mode

	// IsReadOnly returns true if the sandbox does not allow write operations.
	IsReadOnly() bool

	// Close releases any resources held by the sandbox. After Close is called,
	// all operations will fail.
	//
	// Returns error when releasing resources fails.
	Close() error

	// RelPath converts a path to a sandbox-relative path.
	//
	// This handles three cases:
	//
	//  1. Absolute paths (e.g., "/project/dist/file.go") are converted to
	//     paths relative to the sandbox root using filepath.Rel.
	//
	//  2. Relative paths that include the sandbox directory name as a prefix
	//     (e.g., "dist/file.go" when sandbox is at "dist") have that
	//     redundant prefix stripped. This occurs when paths are calculated
	//     relative to a parent directory but include the sandbox folder name.
	//
	//  3. Relative paths that do not match the above cases are returned as-is.
	//
	// Takes path (string) which is the path to convert.
	//
	// Returns string which is the path relative to the sandbox root.
	RelPath(path string) string
}

// File provides a sandboxed file handle that implements io.ReadWriteCloser.
// It wraps os.File with path checks to keep all operations within the sandbox.
type File struct {
	// file is the underlying OS file handle for read and write operations.
	file *os.File

	// name is the file path relative to the sandbox root.
	name string
}

// Name returns the relative path of the file within the sandbox.
//
// Returns string which is the file's path relative to the sandbox root.
func (f *File) Name() string {
	return f.name
}

// AbsolutePath returns the absolute path of the file on disk.
// This should only be used for logging/debugging, not for further file
// operations.
//
// Returns string which is the file's absolute path, or empty if the file is
// nil.
func (f *File) AbsolutePath() string {
	if f.file == nil {
		return ""
	}
	return f.file.Name()
}

// Read reads up to len(p) bytes into p.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the read operation fails.
func (f *File) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

// ReadAt reads len(p) bytes starting at offset off.
//
// Takes p ([]byte) which is the buffer to read data into.
// Takes off (int64) which is the byte offset to start reading from.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the read operation fails.
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	return f.file.ReadAt(p, off)
}

// Write writes len(p) bytes from p to the file.
//
// Takes p ([]byte) which contains the data to write.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when the write operation fails.
func (f *File) Write(p []byte) (n int, err error) {
	return f.file.Write(p)
}

// WriteAt writes len(p) bytes starting at offset off.
//
// Takes p ([]byte) which contains the data to write.
// Takes off (int64) which is the byte offset to start writing at.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when the write operation fails.
func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	return f.file.WriteAt(p, off)
}

// WriteString writes a string to the file.
//
// Takes s (string) which is the text to write.
//
// Returns n (int) which is the number of bytes written.
// Returns err (error) when the write operation fails.
func (f *File) WriteString(s string) (n int, err error) {
	return f.file.WriteString(s)
}

// Seek sets the offset for the next Read or Write.
//
// Takes offset (int64) which specifies the position relative to whence.
// Takes whence (int) which defines the reference point: 0 for start, 1 for
// current position, 2 for end of file.
//
// Returns int64 which is the new offset from the start of the file.
// Returns error when the seek operation fails.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

// Sync saves the file's contents to stable storage.
//
// Returns error when the sync operation fails.
func (f *File) Sync() error {
	return f.file.Sync()
}

// Truncate changes the size of the file.
//
// Takes size (int64) which specifies the new length in bytes.
//
// Returns error when the file cannot be resized.
func (f *File) Truncate(size int64) error {
	return f.file.Truncate(size)
}

// Stat returns information about the file.
//
// Returns fs.FileInfo which describes the file's metadata.
// Returns error when the file cannot be read.
func (f *File) Stat() (fs.FileInfo, error) {
	return f.file.Stat()
}

// Chmod changes the file's permissions.
//
// Takes mode (fs.FileMode) which specifies the new permission bits.
//
// Returns error when the permission change fails.
func (f *File) Chmod(mode fs.FileMode) error {
	return f.file.Chmod(mode)
}

// Close releases all resources held by the file.
//
// Returns error when closing the underlying file fails.
func (f *File) Close() error {
	return f.file.Close()
}

// ReadDir reads the directory contents (if this is a directory).
//
// Takes n (int) which limits the number of entries to return. If n <= 0, all
// entries are returned.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when the file is not a directory or reading fails.
func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	return f.file.ReadDir(n)
}

// Fd returns the underlying file descriptor.
//
// Returns uintptr which is the operating system file descriptor.
func (f *File) Fd() uintptr {
	return f.file.Fd()
}

// fileInfo wraps fs.FileInfo with extra sandbox context.
type fileInfo struct {
	// info holds the file metadata from the file system.
	info fs.FileInfo

	// path is the file path relative to the sandbox root.
	path string
}

// Name returns the base name of the file.
//
// Returns string which is the file name without the directory path.
func (fi fileInfo) Name() string { return fi.info.Name() }

// Size returns the file size in bytes.
//
// Returns int64 which is the size of the file in bytes.
func (fi fileInfo) Size() int64 { return fi.info.Size() }

// Mode returns the file mode bits.
//
// Returns fs.FileMode which holds the file permission and type bits.
func (fi fileInfo) Mode() fs.FileMode { return fi.info.Mode() }

// ModTime returns the modification time.
//
// Returns time.Time which is the last modification time of the file.
func (fi fileInfo) ModTime() time.Time { return fi.info.ModTime() }

// IsDir reports whether this is a directory.
//
// Returns bool which is true if the file is a directory, false otherwise.
func (fi fileInfo) IsDir() bool { return fi.info.IsDir() }

// Sys returns the underlying data source.
//
// Returns any which is nil or system-specific file information.
func (fi fileInfo) Sys() any { return fi.info.Sys() }

// Path returns the relative path within the sandbox.
//
// Returns string which is the path relative to the sandbox root.
func (fi fileInfo) Path() string { return fi.path }
