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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

// noOpSandbox is a pass-through implementation that provides no actual sandboxing.
// Use this when sandboxing is disabled in configuration or for testing.
//
// WARNING: This implementation does NOT prevent path traversal attacks.
// It performs basic path validation but relies on filepath.Clean which
// can still be bypassed in certain edge cases. Use OSSandbox for production.
type noOpSandbox struct {
	// rootPath is the base directory for all file operations.
	rootPath string

	// mode is the current sandbox operating mode.
	mode Mode

	// closed indicates whether the sandbox has been closed.
	closed atomic.Bool
}

var _ Sandbox = (*noOpSandbox)(nil)

// Open opens a file for reading.
//
// Takes name (string) which specifies the path to the file to open.
//
// Returns FileHandle which wraps the opened file for reading.
// Returns error when the sandbox is closed, the path is invalid, or the file
// cannot be opened.
func (s *noOpSandbox) Open(name string) (FileHandle, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullPath) //nolint:gosec // path validated by sandbox
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanPath(name)}, nil
}

// ReadFile reads the entire contents of a file.
//
// Takes name (string) which specifies the path to the file to read.
//
// Returns []byte which contains the complete file contents.
// Returns error when the file cannot be opened or read.
func (s *noOpSandbox) ReadFile(name string) ([]byte, error) {
	return readFileViaOpen(s.Open, name)
}

// Stat returns file information.
//
// Takes name (string) which specifies the path to the file.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the sandbox is closed or the path is invalid.
func (s *noOpSandbox) Stat(name string) (fs.FileInfo, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	return os.Stat(fullPath)
}

// Lstat returns file information without following symlinks.
//
// Takes name (string) which specifies the path to query.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the sandbox is closed or the path is invalid.
func (s *noOpSandbox) Lstat(name string) (fs.FileInfo, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	return os.Lstat(fullPath)
}

// ReadDir reads directory contents.
//
// Takes name (string) which specifies the directory path to read.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when the sandbox is closed or the path is invalid.
func (s *noOpSandbox) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadDir(fullPath)
}

// WalkDir walks the directory tree rooted at the given path.
//
// Takes root (string) which specifies the starting directory path.
// Takes walkFunction (fs.WalkDirFunc) which is called for each file and directory.
//
// Returns error when the sandbox is closed or the root path is invalid.
func (s *noOpSandbox) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	fullRoot, err := s.fullPath(root)
	if err != nil {
		return err
	}

	return filepath.WalkDir(fullRoot, func(path string, d fs.DirEntry, err error) error {
		relPath, relErr := filepath.Rel(s.rootPath, path)
		if relErr != nil {
			relPath = path
		}
		return walkFunction(relPath, d, err)
	})
}

// Create creates or truncates a file.
//
// Takes name (string) which specifies the file path to create.
//
// Returns FileHandle which wraps the created file handle.
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) Create(name string) (FileHandle, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(fullPath) //nolint:gosec // path validated by sandbox
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanPath(name)}, nil
}

// OpenFile opens a file with the given flags and permissions.
//
// Takes name (string) which is the path to the file to open.
// Takes flag (int) which sets the file open mode (e.g. os.O_RDONLY).
// Takes perm (fs.FileMode) which sets the file permissions when creating.
//
// Returns FileHandle which wraps the opened file with a cleaned path.
// Returns error when the sandbox is closed, write access is not allowed, the
// path is not valid, or the file operation fails.
func (s *noOpSandbox) OpenFile(name string, flag int, perm fs.FileMode) (FileHandle, error) {
	isWrite := flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0
	if isWrite {
		if err := s.checkWritable(); err != nil {
			return nil, err
		}
	} else {
		if err := s.checkClosed(); err != nil {
			return nil, err
		}
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(fullPath, flag, perm) //nolint:gosec // path validated by sandbox
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanPath(name)}, nil
}

// WriteFile writes data to a file.
//
// Takes name (string) which specifies the path to the file.
// Takes data ([]byte) which contains the content to write.
// Takes perm (fs.FileMode) which sets the file permissions.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) WriteFile(name string, data []byte, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, perm)
}

// WriteFileAtomic writes data to a file atomically by first writing to a
// temporary file in the same directory and then renaming it to the target path.
// This provides crash safety.
//
// Takes name (string) which is the relative path within the sandbox.
// Takes data ([]byte) which contains the content to write.
// Takes perm (fs.FileMode) which sets the file permissions.
//
// Returns error when writing fails or the sandbox is read-only.
func (s *noOpSandbox) WriteFileAtomic(name string, data []byte, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return err
	}

	directory := filepath.Dir(fullPath)
	base := filepath.Base(fullPath)

	tmpFile, err := os.CreateTemp(directory, "."+base+".tmp.*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("writing data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, fullPath); err != nil { //nolint:gosec // path validated by sandbox
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}

// Mkdir creates a directory.
//
// Takes name (string) which specifies the path of the directory to create.
// Takes perm (fs.FileMode) which sets the permission bits for the directory.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) Mkdir(name string, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return err
	}

	return os.Mkdir(fullPath, perm)
}

// MkdirAll creates a directory and all parents.
//
// Takes path (string) which specifies the directory path to create.
// Takes perm (fs.FileMode) which sets the permission bits for new directories.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) MkdirAll(path string, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(fullPath, perm)
}

// Remove deletes a file or empty directory.
//
// Takes name (string) which specifies the path to remove.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) Remove(name string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return err
	}

	return os.Remove(fullPath)
}

// RemoveAll deletes a path and all children.
//
// Takes path (string) which specifies the path to delete.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) RemoveAll(path string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(path)
	if err != nil {
		return err
	}

	return os.RemoveAll(fullPath)
}

// Rename renames a file or directory.
//
// Takes oldpath (string) which specifies the current path to rename.
// Takes newpath (string) which specifies the new path for the file or
// directory.
//
// Returns error when the sandbox is not writable or either path is invalid.
func (s *noOpSandbox) Rename(oldpath, newpath string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullOld, err := s.fullPath(oldpath)
	if err != nil {
		return err
	}

	fullNew, err := s.fullPath(newpath)
	if err != nil {
		return err
	}

	return os.Rename(fullOld, fullNew)
}

// Chmod changes file permissions.
//
// Takes name (string) which specifies the path to the file.
// Takes mode (fs.FileMode) which specifies the new permission bits.
//
// Returns error when the sandbox is not writable or the path is invalid.
func (s *noOpSandbox) Chmod(name string, mode fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	fullPath, err := s.fullPath(name)
	if err != nil {
		return err
	}

	return os.Chmod(fullPath, mode)
}

// CreateTemp creates a temporary file.
//
// Takes directory (string) which specifies the directory for the temporary file.
// Takes pattern (string) which specifies the filename pattern to use.
//
// Returns FileHandle which wraps the created temporary file with a relative path.
// Returns error when the sandbox is not writable or file creation fails.
func (s *noOpSandbox) CreateTemp(directory, pattern string) (FileHandle, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	fullDir, err := s.fullPath(directory)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(fullDir, defaultDirPerm); err != nil {
		return nil, fmt.Errorf("creating temp file directory %q: %w", fullDir, err)
	}

	f, err := os.CreateTemp(fullDir, pattern)
	if err != nil {
		return nil, fmt.Errorf("creating temp file in %q: %w", fullDir, err)
	}

	relPath, err := filepath.Rel(s.rootPath, f.Name())
	if err != nil {
		relPath = f.Name()
	}

	return &File{file: f, name: relPath}, nil
}

// MkdirTemp creates a temporary directory.
//
// Takes directory (string) which specifies the parent directory path.
// Takes pattern (string) which specifies the prefix for the directory name.
//
// Returns string which is the relative path to the created directory.
// Returns error when the sandbox is read-only, the path is invalid, or
// directory creation fails.
func (s *noOpSandbox) MkdirTemp(directory, pattern string) (string, error) {
	if err := s.checkWritable(); err != nil {
		return "", err
	}

	fullDir, err := s.fullPath(directory)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(fullDir, defaultDirPerm); err != nil {
		return "", fmt.Errorf("creating temp directory parent %q: %w", fullDir, err)
	}

	tmpDir, err := os.MkdirTemp(fullDir, pattern)
	if err != nil {
		return "", fmt.Errorf("creating temp directory in %q: %w", fullDir, err)
	}

	relPath, err := filepath.Rel(s.rootPath, tmpDir)
	if err != nil {
		relPath = tmpDir
	}

	return relPath, nil
}

// Root returns the sandbox root path.
//
// Returns string which is the path to the sandbox root directory.
func (s *noOpSandbox) Root() string {
	return s.rootPath
}

// Mode returns the sandbox mode.
//
// Returns Mode which indicates the current sandbox operational mode.
func (s *noOpSandbox) Mode() Mode {
	return s.mode
}

// IsReadOnly returns true if write operations are not allowed.
//
// Returns bool which indicates whether the sandbox is in read-only mode.
func (s *noOpSandbox) IsReadOnly() bool {
	return s.mode == ModeReadOnly
}

// Close releases resources.
//
// Returns error when resources cannot be released, though this implementation
// always returns nil.
func (s *noOpSandbox) Close() error {
	s.closed.Store(true)
	return nil
}

// RelPath converts a path to a sandbox-relative path.
// See the Sandbox interface for full documentation.
//
// Takes path (string) which is the path to convert.
//
// Returns string which is the path relative to the sandbox root. If the path
// cannot be made relative, the original path is returned unchanged.
func (s *noOpSandbox) RelPath(path string) string {
	if filepath.IsAbs(path) {
		relPath, err := filepath.Rel(s.rootPath, path)
		if err != nil {
			return path
		}
		return relPath
	}

	sandboxDirName := filepath.Base(s.rootPath)
	prefix := sandboxDirName + string(filepath.Separator)
	if trimmed, found := strings.CutPrefix(path, prefix); found {
		return trimmed
	}

	return path
}

// fullPath turns a relative path into an absolute path within the root.
// It does basic path traversal checks but is not as secure as OSSandbox.
//
// Takes name (string) which is the relative path to convert.
//
// Returns string which is the absolute path within the sandbox root.
// Returns error when the path escapes the sandbox root.
func (s *noOpSandbox) fullPath(name string) (string, error) {
	cleanName := cleanPath(name)
	fullPath := filepath.Join(s.rootPath, cleanName)
	cleanFull := filepath.Clean(fullPath)

	if !isWithinRoot(s.rootPath, cleanFull) {
		return "", fmt.Errorf("safedisk: path %q escapes sandbox root", name)
	}

	return cleanFull, nil
}

// checkClosed returns an error if the sandbox has been closed.
//
// Returns error when the sandbox has been closed.
func (s *noOpSandbox) checkClosed() error {
	if s.closed.Load() {
		return errClosed
	}
	return nil
}

// checkWritable checks if write operations are allowed on the sandbox.
//
// Returns error when the sandbox is closed or in read-only mode.
func (s *noOpSandbox) checkWritable() error {
	if err := s.checkClosed(); err != nil {
		return err
	}
	if s.mode == ModeReadOnly {
		return errReadOnly
	}
	return nil
}

// NewNoOpSandbox creates a sandbox that wraps standard file operations
// without path restrictions. All paths are resolved relative to the given
// root directory.
//
// Takes directory (string) which is the root directory path.
// Takes mode (Mode) which sets whether to allow write operations.
//
// Returns Sandbox which is the filesystem wrapper.
// Returns error when the directory path is empty or cannot be resolved.
//
// For read-write mode, the directory is created if it does not exist.
func NewNoOpSandbox(directory string, mode Mode) (Sandbox, error) {
	if directory == "" {
		return nil, errEmptyPath
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("safedisk: failed to resolve absolute path: %w", err)
	}

	if mode == ModeReadWrite {
		if err := os.MkdirAll(absDir, defaultDirPerm); err != nil {
			return nil, fmt.Errorf("safedisk: failed to create directory %q: %w", absDir, err)
		}
	}

	return &noOpSandbox{
		rootPath: absDir,
		mode:     mode,
		closed:   atomic.Bool{},
	}, nil
}

// isWithinRoot checks if a path is within the root directory.
// This is a basic string prefix check and is NOT safe against symlinks.
//
// Takes root (string) which is the root directory path to check against.
// Takes path (string) which is the path to check.
//
// Returns bool which is true if the path is within root, false otherwise.
func isWithinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)

	if path == root {
		return true
	}

	return len(path) > len(root) && path[:len(root)] == root && path[len(root)] == filepath.Separator
}
