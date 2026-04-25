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
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	logger "piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxTempAttempts is the maximum number of attempts to create a unique temp file.
	maxTempAttempts = 10000

	// tempRandomBytes is the number of random bytes used for temporary file names.
	tempRandomBytes = 8

	// defaultDirPerm is the file mode used when creating new directories.
	// Uses 0750: owner rwx, group rx, others none.
	defaultDirPerm = fs.FileMode(0750)

	// defaultFilePerm is the file permission used when creating new files.
	// Uses 0640: owner rw, group r, others none.
	defaultFilePerm = fs.FileMode(0640)

	// currentDir is the current directory marker used as a default when handling
	// paths.
	currentDir = "."
)

// osSandbox implements Sandbox using Go 1.24's os.Root for kernel-level
// protection. It uses the operating system's own security mechanisms
// (openat2 with RESOLVE_BENEATH on Linux) to prevent path traversal attacks.
type osSandbox struct {
	// root is the OS root handle for sandboxed file operations.
	root *os.Root

	// rootPath is the absolute path to the sandbox root folder.
	rootPath string

	// mode specifies whether the sandbox allows write operations.
	mode Mode

	// closed tracks whether the sandbox has been closed.
	closed atomic.Bool
}

var _ Sandbox = (*osSandbox)(nil)
var _ fs.DirEntry = (*dirEntry)(nil)

// Open opens a file for reading within the sandbox.
//
// Takes name (string) which specifies the path to the file to open.
//
// Returns FileHandle which wraps the opened file with the cleaned path.
// Returns error when the sandbox is closed or the file cannot be opened.
func (s *osSandbox) Open(name string) (FileHandle, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	cleanName := cleanPath(name)
	f, err := s.root.Open(cleanName)
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanName}, nil
}

// ReadFile reads the entire contents of a file within the sandbox.
//
// Takes name (string) which specifies the path to the file to read.
//
// Returns []byte which contains the complete file contents.
// Returns error when the file cannot be opened or read.
func (s *osSandbox) ReadFile(name string) ([]byte, error) {
	return readFileViaOpen(s.Open, name)
}

// ReadFileLimit reads up to maxBytes from a file within the sandbox.
// See Sandbox.ReadFileLimit for the contract.
//
// Takes name (string) which specifies the path to the file to read.
// Takes maxBytes (int64) which caps the byte count read into memory.
//
// Returns []byte which contains the file content (up to maxBytes).
// Returns int64 which is the stat-reported file size at stat time.
// Returns error which wraps ErrFileExceedsLimit, ErrInvalidLimit, or any
// underlying stat or read error.
func (s *osSandbox) ReadFileLimit(name string, maxBytes int64) ([]byte, int64, error) {
	return readFileLimitViaOpen(s.Open, s.Stat, name, maxBytes)
}

// Stat returns file information for a path within the sandbox.
//
// Takes name (string) which specifies the path to query.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the sandbox is closed or the path is invalid.
func (s *osSandbox) Stat(name string) (fs.FileInfo, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	cleanName := cleanPath(name)
	return s.root.Stat(cleanName)
}

// Lstat returns file information without following symlinks.
//
// Takes name (string) which specifies the path to query.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the sandbox is closed or the path is invalid.
func (s *osSandbox) Lstat(name string) (fs.FileInfo, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	cleanName := cleanPath(name)
	return s.root.Lstat(cleanName)
}

// ReadDir reads the contents of a directory within the sandbox.
//
// Takes name (string) which specifies the path to the directory to read.
//
// Returns []fs.DirEntry which contains the directory entries found.
// Returns error when the directory cannot be opened or read.
func (s *osSandbox) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			_, l := logger.From(context.Background(), log)
			l.Warn("Failed to close directory after read", logger.Error(closeErr), logger.String("dir", name))
		}
	}()

	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("reading directory entries in sandbox %q: %w", name, err)
	}
	return entries, nil
}

// WalkDir walks the directory tree rooted at root within the sandbox.
//
// Takes root (string) which specifies the starting directory path.
// Takes walkFunction (fs.WalkDirFunc) which is called for each file and directory.
//
// Returns error when the sandbox is closed or the walk fails.
func (s *osSandbox) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	cleanRoot := cleanPath(root)
	return s.walkDirRecursive(cleanRoot, walkFunction)
}

// dirEntry wraps fs.FileInfo to implement the fs.DirEntry interface.
type dirEntry struct {
	// info holds the file metadata used to provide fs.DirEntry methods.
	info fs.FileInfo
}

// Name returns the file's base name.
//
// Returns string which is the base name of the file.
func (d *dirEntry) Name() string { return d.info.Name() }

// IsDir reports whether the entry describes a directory.
//
// Returns bool which is true if this entry is a directory.
func (d *dirEntry) IsDir() bool { return d.info.IsDir() }

// Type returns the file mode bits for this directory entry.
//
// Returns fs.FileMode which holds only the type portion of the mode bits.
func (d *dirEntry) Type() fs.FileMode { return d.info.Mode().Type() }

// Info returns the file information for this directory entry.
//
// Returns fs.FileInfo which holds the file metadata.
// Returns error which is always nil in this implementation.
func (d *dirEntry) Info() (fs.FileInfo, error) { return d.info, nil }

// Create creates or truncates a file for writing.
//
// Takes name (string) which specifies the path of the file to create.
//
// Returns FileHandle which is the opened file ready for writing.
// Returns error when the sandbox is not writable or file creation fails.
func (s *osSandbox) Create(name string) (FileHandle, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	cleanName := cleanPath(name)
	f, err := s.root.Create(cleanName)
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanName}, nil
}

// OpenFile opens a file with the specified flags and permissions.
//
// Takes name (string) which specifies the path to the file.
// Takes flag (int) which specifies the file open flags (e.g. os.O_RDONLY).
// Takes perm (fs.FileMode) which specifies the file permissions for creation.
//
// Returns FileHandle which wraps the opened file.
// Returns error when the sandbox is closed or not writable for write operations.
func (s *osSandbox) OpenFile(name string, flag int, perm fs.FileMode) (FileHandle, error) {
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

	cleanName := cleanPath(name)
	f, err := s.root.OpenFile(cleanName, flag, perm)
	if err != nil {
		return nil, err
	}

	return &File{file: f, name: cleanName}, nil
}

// WriteFile writes data to a file, creating it if necessary.
//
// Takes name (string) which specifies the path to the file.
// Takes data ([]byte) which contains the content to write.
// Takes perm (fs.FileMode) which sets the file permissions if created.
//
// Returns error when the file cannot be opened, written, or closed.
func (s *osSandbox) WriteFile(name string, data []byte, perm fs.FileMode) error {
	f, err := s.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	_, writeErr := f.Write(data)
	closeErr := f.Close()

	if writeErr != nil {
		return fmt.Errorf("writing file data in sandbox %q: %w", name, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing file after write in sandbox %q: %w", name, closeErr)
	}
	return nil
}

// WriteFileAtomic writes data to a file atomically within the sandbox. It first
// writes to a temporary file in the same directory and then renames it to the
// target path, which provides crash safety.
//
// Takes name (string) which is the relative path within the sandbox.
// Takes data ([]byte) which contains the content to write.
// Takes perm (fs.FileMode) which sets the file permissions.
//
// Returns error when writing fails or the sandbox is read-only.
func (s *osSandbox) WriteFileAtomic(name string, data []byte, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanName := cleanPath(name)
	directory := filepath.Dir(cleanName)
	base := filepath.Base(cleanName)

	tmpFile, err := s.CreateTemp(directory, "."+base+".tmp.*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = s.Remove(tmpPath)
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

	if err := s.Rename(tmpPath, cleanName); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	success = true
	return nil
}

// Mkdir creates a directory within the sandbox.
//
// Takes name (string) which is the path of the directory to create.
// Takes perm (fs.FileMode) which specifies the permission bits for the new
// directory.
//
// Returns error when the sandbox is not writable or the directory cannot be
// created.
func (s *osSandbox) Mkdir(name string, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanName := cleanPath(name)
	return s.root.Mkdir(cleanName, perm)
}

// MkdirAll creates a directory and all necessary parent directories.
//
// Takes path (string) which specifies the directory path to create.
// Takes perm (fs.FileMode) which sets the permission bits for new directories.
//
// Returns error when the sandbox is read-only or directory creation fails.
func (s *osSandbox) MkdirAll(path string, perm fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanPath := cleanPath(path)
	if cleanPath == currentDir || cleanPath == "" {
		return nil
	}

	parts := strings.Split(cleanPath, string(filepath.Separator))
	current := ""

	for _, part := range parts {
		if part == "" || part == currentDir {
			continue
		}

		if current == "" {
			current = part
		} else {
			current = filepath.Join(current, part)
		}

		err := s.root.Mkdir(current, perm)
		if err != nil && !os.IsExist(err) {
			return fmt.Errorf("safedisk: failed to create directory %q: %w", current, err)
		}
	}

	return nil
}

// Remove deletes a file or empty directory within the sandbox.
//
// Takes name (string) which is the path to the file or directory to delete.
//
// Returns error when the sandbox is read-only or the removal fails.
func (s *osSandbox) Remove(name string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanName := cleanPath(name)
	return s.root.Remove(cleanName)
}

// RemoveAll deletes a path and all its contents within the sandbox.
//
// Takes path (string) which specifies the path to remove.
//
// Returns error when the path cannot be removed.
func (s *osSandbox) RemoveAll(path string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanPath := cleanPath(path)

	info, err := s.root.Lstat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return s.root.Remove(cleanPath)
	}

	var paths []string
	walkErr := s.walkDirRecursive(cleanPath, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if walkErr != nil {
		return fmt.Errorf("walking directory tree for removal in sandbox %q: %w", cleanPath, walkErr)
	}

	for i := len(paths) - 1; i >= 0; i-- {
		if err := s.root.Remove(paths[i]); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// Rename moves a file or directory to a new path within the sandbox.
//
// Uses absolute paths internally but both must be within the sandbox.
// The source is validated via os.Root; the destination is validated by
// checking that the resolved absolute path remains under the sandbox root,
// since os.Root does not provide a Rename method.
//
// Takes oldpath (string) which is the current path of the file or directory.
// Takes newpath (string) which is the destination path.
//
// Returns error when the sandbox is read-only, either path escapes the
// sandbox, or the source path is invalid.
func (s *osSandbox) Rename(oldpath, newpath string) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanOld := cleanPath(oldpath)
	cleanNew := cleanPath(newpath)

	if _, err := s.root.Lstat(cleanOld); err != nil {
		return fmt.Errorf("safedisk: source path %q: %w", cleanOld, err)
	}

	absOld := filepath.Join(s.rootPath, cleanOld)
	absNew := filepath.Join(s.rootPath, cleanNew)

	if !isWithinRoot(s.rootPath, absNew) {
		return fmt.Errorf("safedisk: destination path %q escapes sandbox root", newpath)
	}

	return os.Rename(absOld, absNew)
}

// Chmod changes the permissions of a file within the sandbox.
//
// Takes name (string) which is the path to the file.
// Takes mode (fs.FileMode) which specifies the new permissions.
//
// Returns error when the sandbox is not writable or the file cannot be opened.
func (s *osSandbox) Chmod(name string, mode fs.FileMode) error {
	if err := s.checkWritable(); err != nil {
		return err
	}

	cleanName := cleanPath(name)

	f, err := s.root.OpenFile(cleanName, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			_, l := logger.From(context.Background(), log)
			l.Warn("Failed to close file after chmod", logger.Error(closeErr), logger.String("file", name))
		}
	}()

	return f.Chmod(mode)
}

// CreateTemp creates a temporary file within the sandbox.
//
// Takes directory (string) which specifies the directory for the
// file, or empty for the current directory.
// Takes pattern (string) which specifies the filename pattern with optional
// prefix and suffix around a random string.
//
// Returns FileHandle which is the created temporary file.
// Returns error when the sandbox is read-only, the directory cannot be
// created, or a unique filename cannot be generated.
func (s *osSandbox) CreateTemp(directory, pattern string) (FileHandle, error) {
	if err := s.checkWritable(); err != nil {
		return nil, err
	}

	cleanDir := cleanPath(directory)
	if cleanDir == "" {
		cleanDir = currentDir
	}

	if err := s.MkdirAll(cleanDir, defaultDirPerm); err != nil {
		return nil, fmt.Errorf("creating temp file directory in sandbox %q: %w", cleanDir, err)
	}

	prefix, suffix := parsePattern(pattern)

	for range maxTempAttempts {
		name := prefix + randomString() + suffix
		path := filepath.Join(cleanDir, name)

		f, err := s.root.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, defaultFilePerm)
		if err == nil {
			return &File{file: f, name: path}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("safedisk: failed to create temp file: %w", err)
		}
	}

	return nil, errTempFileExhausted
}

// MkdirTemp creates a temporary directory within the sandbox.
//
// Takes directory (string) which specifies the parent directory for the temp
// directory, or empty string to use the current directory.
// Takes pattern (string) which specifies a prefix and suffix for the directory
// name, with a random string inserted between them.
//
// Returns string which is the path to the created temporary directory.
// Returns error when the sandbox is not writable, the parent directory cannot
// be created, or no unique name can be found after maximum attempts.
func (s *osSandbox) MkdirTemp(directory, pattern string) (string, error) {
	if err := s.checkWritable(); err != nil {
		return "", err
	}

	cleanDir := cleanPath(directory)
	if cleanDir == "" {
		cleanDir = currentDir
	}

	if err := s.MkdirAll(cleanDir, defaultDirPerm); err != nil {
		return "", fmt.Errorf("creating temp directory parent in sandbox %q: %w", cleanDir, err)
	}

	prefix, suffix := parsePattern(pattern)

	for range maxTempAttempts {
		name := prefix + randomString() + suffix
		path := filepath.Join(cleanDir, name)

		err := s.root.Mkdir(path, defaultDirPerm)
		if err == nil {
			return path, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("safedisk: failed to create temp directory: %w", err)
		}
	}

	return "", errTempFileExhausted
}

// Root returns the absolute path of the sandbox root directory.
//
// Returns string which is the absolute filesystem path.
func (s *osSandbox) Root() string {
	return s.rootPath
}

// Mode returns whether this sandbox is read-only or read-write.
//
// Returns Mode which indicates the sandbox access mode.
func (s *osSandbox) Mode() Mode {
	return s.mode
}

// IsReadOnly returns true if this sandbox does not allow write operations.
//
// Returns bool which is true when the sandbox mode is read-only.
func (s *osSandbox) IsReadOnly() bool {
	return s.mode == ModeReadOnly
}

// Close releases any resources held by the sandbox.
//
// Returns error when the underlying root cannot be closed.
func (s *osSandbox) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	return s.root.Close()
}

// RelPath converts a path to a sandbox-relative path.
// See the Sandbox interface for full documentation.
//
// Takes path (string) which is the path to convert.
//
// Returns string which is the path relative to the sandbox root.
func (s *osSandbox) RelPath(path string) string {
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

// checkClosed returns errClosed if the sandbox has been closed.
//
// Returns error when the sandbox has been closed.
func (s *osSandbox) checkClosed() error {
	if s.closed.Load() {
		return errClosed
	}
	return nil
}

// checkWritable checks whether write operations are allowed.
//
// Returns error when the sandbox is closed or in read-only mode.
func (s *osSandbox) checkWritable() error {
	if err := s.checkClosed(); err != nil {
		return err
	}
	if s.mode == ModeReadOnly {
		return errReadOnly
	}
	return nil
}

// walkDirRecursive is the internal recursive implementation of WalkDir.
//
// Takes path (string) which specifies the directory path to walk.
// Takes walkFunction (fs.WalkDirFunc) which handles each visited entry.
//
// Returns error when the walk function returns an error other than SkipDir.
func (s *osSandbox) walkDirRecursive(path string, walkFunction fs.WalkDirFunc) error {
	info, err := s.root.Lstat(path)
	if err != nil {
		return walkFunction(path, nil, err)
	}

	entry := &dirEntry{info: info}

	if err := walkFunction(path, entry, nil); err != nil {
		if errors.Is(err, filepath.SkipDir) {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	entries, err := s.ReadDir(path)
	if err != nil {
		if err := walkFunction(path, entry, err); err != nil {
			if errors.Is(err, filepath.SkipDir) {
				return nil
			}
			return err
		}
		return nil
	}

	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	for _, e := range entries {
		entryPath := filepath.Join(path, e.Name())
		if err := s.walkDirRecursive(entryPath, walkFunction); err != nil {
			return err
		}
	}

	return nil
}

// NewSandbox creates a sandboxed filesystem rooted at the given directory.
// All file operations through this sandbox are limited to the specified
// directory and its subdirectories.
//
// Takes directory (string) which is the path to the sandbox root directory.
// Takes mode (Mode) which sets whether the sandbox allows write operations.
//
// Returns Sandbox which is the sandboxed filesystem.
// Returns error when the directory path is empty or cannot be opened.
//
// For read-only mode, the directory must exist.
// For read-write mode, the directory is created if it does not exist.
func NewSandbox(directory string, mode Mode) (Sandbox, error) {
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

	root, err := os.OpenRoot(absDir)
	if err != nil {
		return nil, fmt.Errorf("safedisk: failed to open root at %q: %w", absDir, err)
	}

	return &osSandbox{
		root:     root,
		rootPath: absDir,
		mode:     mode,
		closed:   atomic.Bool{},
	}, nil
}

// cleanPath makes a path ready for use within the sandbox.
// It removes any leading slash and cleans the path to make it relative.
//
// Takes name (string) which is the path to clean.
//
// Returns string which is the cleaned, relative path.
func cleanPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	return filepath.Clean(name)
}

// parsePattern splits a pattern like "upload-*.tmp" into prefix and suffix.
//
// Takes pattern (string) which is the glob pattern to split at the last
// asterisk.
//
// Returns prefix (string) which is the part before the asterisk.
// Returns suffix (string) which is the part after the asterisk.
func parsePattern(pattern string) (prefix, suffix string) {
	if pattern == "" {
		return "", ""
	}

	if index := strings.LastIndex(pattern, "*"); index != -1 {
		prefix = pattern[:index]
		suffix = pattern[index+1:]
	} else {
		prefix = pattern
	}

	return prefix, suffix
}

// randomString generates a random hex string for temporary file names.
//
// Returns string which is a random hex string, or a fallback based on the
// process ID if random generation fails.
func randomString() string {
	b := make([]byte, tempRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}
