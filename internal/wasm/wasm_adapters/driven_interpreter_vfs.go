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
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

const (
	// vfsDirPermission is the permission mode for directories (rwxr-xr-x).
	vfsDirPermission fs.FileMode = 0755

	// vfsFilePermission is the permission mode for files (rw-r--r--).
	vfsFilePermission fs.FileMode = 0644

	// vfsPathSep is the separator used between parts of virtual file paths.
	vfsPathSep = "/"

	// vfsCurrentDir is the current directory marker for virtual file system paths.
	vfsCurrentDir = "."
)

// InterpreterVFS implements fs.FS for the interpreter's
// SetSourcecodeFilesystem. It provides an in-memory virtual
// filesystem that allows the interpreter to resolve imports from
// generated code without real filesystem access.
type InterpreterVFS struct {
	// files maps file paths to their contents.
	files map[string]string
}

var (
	_ fs.FS = (*InterpreterVFS)(nil)

	_ fs.ReadDirFS = (*InterpreterVFS)(nil)
)

// NewInterpreterVFS creates a new in-memory filesystem for the
// interpreter.
//
// Takes files (map[string]string) which maps file paths to their contents.
// Paths should use forward slashes and be relative to the module root.
//
// Returns *InterpreterVFS which implements fs.FS for use with the interpreter.
func NewInterpreterVFS(files map[string]string) *InterpreterVFS {
	normalised := make(map[string]string, len(files))
	for path, content := range files {
		clean := filepath.ToSlash(filepath.Clean(path))
		clean = strings.TrimPrefix(clean, "/")
		normalised[clean] = content
	}
	return &InterpreterVFS{files: normalised}
}

// Open opens the named file for reading.
// This implements fs.FS.
//
// Takes name (string) which is the path to the file to open.
//
// Returns fs.File which provides read access to the file.
// Returns error when the file is not found or the name is invalid.
func (v *InterpreterVFS) Open(name string) (fs.File, error) {
	clean := filepath.ToSlash(filepath.Clean(name))
	clean = strings.TrimPrefix(clean, vfsPathSep)

	if clean == vfsCurrentDir || clean == "" {
		return &vfsDir{
			name:  vfsCurrentDir,
			vfs:   v,
			path:  "",
			isDir: true,
		}, nil
	}

	if content, ok := v.files[clean]; ok {
		return &vfsFile{
			name:    filepath.Base(clean),
			content: content,
			reader:  strings.NewReader(content),
		}, nil
	}

	prefix := clean + vfsPathSep
	for path := range v.files {
		if strings.HasPrefix(path, prefix) || path == clean {
			return &vfsDir{
				name:  filepath.Base(clean),
				vfs:   v,
				path:  clean,
				isDir: true,
			}, nil
		}
	}

	return nil, fs.ErrNotExist
}

// ReadDir reads the named directory and returns a list of directory entries
// sorted by filename.
// This implements fs.ReadDirFS.
//
// Takes name (string) which is the directory path to read.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when the directory cannot be read.
func (v *InterpreterVFS) ReadDir(name string) ([]fs.DirEntry, error) {
	clean := filepath.ToSlash(filepath.Clean(name))
	clean = strings.TrimPrefix(clean, vfsPathSep)

	prefix := ""
	if clean != vfsCurrentDir && clean != "" {
		prefix = clean + vfsPathSep
	}

	seen := make(map[string]bool)
	entries := make([]fs.DirEntry, 0, len(v.files))

	for path := range v.files {
		if prefix != "" && !strings.HasPrefix(path, prefix) {
			continue
		}
		if prefix == "" && clean != vfsCurrentDir && clean != "" {
			continue
		}

		rel := strings.TrimPrefix(path, prefix)
		if rel == "" {
			continue
		}

		parts := strings.SplitN(rel, vfsPathSep, 2)
		entryName := parts[0]

		if seen[entryName] {
			continue
		}
		seen[entryName] = true

		isDir := len(parts) > 1
		entries = append(entries, &vfsDirEntry{
			name:  entryName,
			isDir: isDir,
			size:  int64(len(v.files[path])),
		})
	}

	return entries, nil
}

// AddFile adds or updates a file in the virtual filesystem.
//
// Takes path (string) which is the file path.
// Takes content (string) which is the file content.
func (v *InterpreterVFS) AddFile(path, content string) {
	clean := filepath.ToSlash(filepath.Clean(path))
	clean = strings.TrimPrefix(clean, "/")
	v.files[clean] = content
}

// vfsFile implements io.ReadCloser for in-memory file content.
type vfsFile struct {
	// reader provides sequential access to the file's string content.
	reader *strings.Reader

	// name is the base name of the file.
	name string

	// content holds the file data as a string.
	content string
}

// Stat returns file information.
//
// Returns fs.FileInfo which provides the file's metadata.
// Returns error when the file information cannot be retrieved.
func (f *vfsFile) Stat() (fs.FileInfo, error) {
	return &vfsFileInfo{
		name:  f.name,
		size:  int64(len(f.content)),
		isDir: false,
	}, nil
}

// Read reads up to len(b) bytes from the file.
//
// Takes b ([]byte) which is the buffer to read into.
//
// Returns int which is the number of bytes read.
// Returns error when the end of the file is reached or reading fails.
func (f *vfsFile) Read(b []byte) (int, error) {
	return f.reader.Read(b)
}

// Close closes the file. For in-memory files, this is a no-op.
//
// Returns error which is always nil for in-memory files.
func (*vfsFile) Close() error {
	return nil
}

// vfsDir implements fs.File and fs.ReadDirFile for directories.
type vfsDir struct {
	// name is the directory name.
	name string

	// vfs provides file system operations for directory reading.
	vfs *InterpreterVFS

	// path is the directory path within the virtual filesystem.
	path string

	// isDir indicates whether the vfsDir represents a directory.
	isDir bool
}

// Stat returns directory information.
//
// Returns fs.FileInfo which provides the directory's metadata.
// Returns error when the directory information cannot be retrieved.
func (d *vfsDir) Stat() (fs.FileInfo, error) {
	return &vfsFileInfo{
		name:  d.name,
		size:  0,
		isDir: true,
	}, nil
}

// Read returns an error as directories cannot be read directly.
//
// Takes p ([]byte) which is the buffer to read into.
//
// Returns int which is always zero as directories cannot be read.
// Returns error when called, as directories do not support
// reading.
func (d *vfsDir) Read(_ []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.path, Err: fs.ErrInvalid}
}

// Close closes the directory.
//
// Returns error when the directory cannot be closed; always returns nil.
func (*vfsDir) Close() error {
	return nil
}

// ReadDir reads the directory contents.
//
// Takes n (int) which limits the number of entries returned. If n <= 0, all
// entries are returned.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when reading fails or when n > 0 and no entries remain.
func (d *vfsDir) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := d.vfs.ReadDir(d.path)
	if err != nil {
		return nil, fmt.Errorf("reading virtual directory %q: %w", d.path, err)
	}

	if n <= 0 {
		return entries, nil
	}

	if n > len(entries) {
		n = len(entries)
	}
	if n == 0 {
		return nil, io.EOF
	}
	return entries[:n], nil
}

// vfsFileInfo implements fs.FileInfo for in-memory files.
type vfsFileInfo struct {
	// name is the base name of the file.
	name string

	// size is the file size in bytes.
	size int64

	// isDir bool // isDir indicates whether this entry is a directory.
	isDir bool
}

// Name returns the base name of the file.
//
// Returns string which is the file's base name.
func (fi *vfsFileInfo) Name() string {
	return fi.name
}

// Size returns the length in bytes.
//
// Returns int64 which is the file size in bytes.
func (fi *vfsFileInfo) Size() int64 {
	return fi.size
}

// Mode returns the file mode bits.
//
// Returns fs.FileMode which is the permission and mode bits for the file.
func (fi *vfsFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir | vfsDirPermission
	}
	return vfsFilePermission
}

// ModTime returns a zero time as modification time is not tracked.
//
// Returns time.Time which is always the zero value.
func (*vfsFileInfo) ModTime() time.Time {
	return time.Time{}
}

// IsDir returns whether this is a directory.
//
// Returns bool which is true if this entry is a directory.
func (fi *vfsFileInfo) IsDir() bool {
	return fi.isDir
}

// Sys returns the underlying data source.
//
// Returns any which is always nil for in-memory files.
func (*vfsFileInfo) Sys() any {
	return nil
}

// vfsDirEntry implements fs.DirEntry for directory entries.
type vfsDirEntry struct {
	// name is the base name of the file or directory entry.
	name string

	// isDir indicates whether the entry is a directory.
	isDir bool

	// size is the file size in bytes.
	size int64
}

// Name returns the name of the entry.
//
// Returns string which is the base name of the file or directory.
func (e *vfsDirEntry) Name() string {
	return e.name
}

// IsDir reports whether the entry is a directory.
//
// Returns bool which is true if the entry represents a directory.
func (e *vfsDirEntry) IsDir() bool {
	return e.isDir
}

// Type returns the type bits for the entry.
//
// Returns fs.FileMode which is ModeDir for directories or zero for files.
func (e *vfsDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}

// Info returns the FileInfo for the entry.
//
// Returns fs.FileInfo which describes the file or directory.
// Returns error when the file info cannot be retrieved.
func (e *vfsDirEntry) Info() (fs.FileInfo, error) {
	return &vfsFileInfo{
		name:  e.name,
		size:  e.size,
		isDir: e.isDir,
	}, nil
}
