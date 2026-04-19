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
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"piko.sh/piko/wdk/safedisk"
)

// FileSystem defines file operations for reading and traversing files.
// Tests can use mock versions instead of real filesystem access.
type FileSystem interface {
	// WalkDir walks the file tree rooted at root, calling
	// walkFunction for each file or directory.
	//
	// Takes root (string) which is the starting directory for the walk.
	// Takes walkFunction (fs.WalkDirFunc) which is called for each file or directory.
	//
	// Returns error when the walk fails.
	WalkDir(root string, walkFunction fs.WalkDirFunc) error

	// Open opens the named file for reading.
	//
	// Takes name (string) which is the path of the file to open.
	//
	// Returns io.ReadCloser which provides access to the file contents.
	// Returns error when the file cannot be opened.
	Open(name string) (io.ReadCloser, error)

	// Stat returns file information for the named file.
	//
	// Takes name (string) which is the path to the file.
	//
	// Returns fs.FileInfo which describes the file.
	// Returns error when the file does not exist or cannot be accessed.
	Stat(name string) (fs.FileInfo, error)

	// Rel returns a path that, when joined to basepath, gives the same path as
	// targpath.
	//
	// Takes basepath (string) which is the starting path.
	// Takes targpath (string) which is the target path to reach.
	//
	// Returns string which is the relative path from basepath to targpath.
	// Returns error when a relative path cannot be found.
	Rel(basepath, targpath string) (string, error)

	// Join combines path elements into a single path.
	//
	// Takes element (...string) which contains the path elements to join.
	//
	// Returns string which is the combined path.
	Join(element ...string) string

	// IsNotExist returns whether the error is known to report that a file does
	// not exist.
	//
	// Takes err (error) which is the error to check.
	//
	// Returns bool which is true if the error indicates a missing file.
	IsNotExist(err error) bool
}

var (
	_ FileSystem = (*osFileSystem)(nil)

	_ FileSystem = (*sandboxedFileSystem)(nil)
)

// osFileSystem implements the filesystem interface using the os package.
type osFileSystem struct{}

// WalkDir walks the file tree rooted at root.
//
// Takes root (string) which specifies the starting directory for the walk.
// Takes walkFunction (fs.WalkDirFunc) which is called for each file or directory.
//
// Returns error when the walk cannot complete or walkFunction returns an error.
func (*osFileSystem) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	return filepath.WalkDir(root, walkFunction)
}

// Open opens the named file for reading.
//
// Takes name (string) which specifies the path to the file to open.
//
// Returns io.ReadCloser which provides access to the file contents.
// Returns error when the file cannot be opened.
func (*osFileSystem) Open(name string) (io.ReadCloser, error) {
	return os.Open(name) //nolint:gosec // caller validates path
}

// Stat returns file information for the named file.
//
// Takes name (string) which is the path to the file.
//
// Returns fs.FileInfo which provides file metadata.
// Returns error when the file does not exist or cannot be read.
func (*osFileSystem) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// Rel returns a relative path from the base path to the target path.
//
// Takes basepath (string) which is the starting directory.
// Takes targpath (string) which is the destination path.
//
// Returns string which is the relative path from base to target.
// Returns error when a relative path cannot be computed.
func (*osFileSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Join joins path elements into a single path.
//
// Takes element (...string) which specifies the path elements to join.
//
// Returns string which is the combined path using the OS-specific separator.
func (*osFileSystem) Join(element ...string) string {
	return filepath.Join(element...)
}

// IsNotExist returns whether the error reports that a file does not exist.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error indicates a missing file.
func (*osFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// sandboxedFileSystem provides safe filesystem access within a restricted directory.
// It implements the filesystem interface using safedisk.Sandbox for secure,
// sandboxed operations.
type sandboxedFileSystem struct {
	// sandbox provides filesystem operations within a safe boundary.
	sandbox safedisk.Sandbox
}

// WalkDir walks the file tree rooted at root using the sandbox.
//
// Takes root (string) which specifies the starting directory for the walk.
// Takes walkFunction (fs.WalkDirFunc) which is called for each file or directory.
//
// Returns error when the walk cannot complete or walkFunction returns an error.
func (s *sandboxedFileSystem) WalkDir(root string, walkFunction fs.WalkDirFunc) error {
	return s.sandbox.WalkDir(root, walkFunction)
}

// Open opens the named file for reading using the sandbox.
//
// Takes name (string) which specifies the file path to open.
//
// Returns io.ReadCloser which provides access to the file contents.
// Returns error when the file cannot be opened.
func (s *sandboxedFileSystem) Open(name string) (io.ReadCloser, error) {
	return s.sandbox.Open(name)
}

// Stat returns file information for the named file within the sandbox.
//
// Takes name (string) which specifies the path to the file.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the file does not exist or cannot be accessed.
func (s *sandboxedFileSystem) Stat(name string) (fs.FileInfo, error) {
	return s.sandbox.Stat(name)
}

// Rel returns a relative path from base to target.
//
// Takes basepath (string) which is the starting directory.
// Takes targpath (string) which is the destination path.
//
// Returns string which is the relative path from base to target.
// Returns error when a relative path cannot be computed.
//
// This is a pure path operation, not affected by sandboxing.
func (*sandboxedFileSystem) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Join combines path elements into a single path.
//
// Takes element (...string) which provides the path elements to combine.
//
// Returns string which is the combined path.
//
// This is a pure path operation and is not affected by sandboxing.
func (*sandboxedFileSystem) Join(element ...string) string {
	return filepath.Join(element...)
}

// IsNotExist reports whether the error indicates that a file does not exist.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error indicates a missing file.
func (*sandboxedFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// newOSFileSystem creates a filesystem that uses the operating system.
//
// Returns FileSystem which provides access to the real filesystem.
func newOSFileSystem() FileSystem {
	return &osFileSystem{}
}
