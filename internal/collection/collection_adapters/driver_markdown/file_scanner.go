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

package driver_markdown

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// logKeyPath is the logger key for file paths.
const logKeyPath = "path"

// fileScanner handles recursive discovery of markdown files in a directory.
// All file operations are sandboxed for security.
//
// Design Philosophy:
//   - Single responsibility: Only concerned with file discovery
//   - Path-based: Works with paths relative to the sandbox root
//   - Recursive: Scans subdirectories automatically
//   - Filtered: Only returns .md files
//   - Secure: All operations are sandboxed
//
// Performance:
//   - O(n) where n = total files in directory tree
//   - Memory: ~1KB per discovered file
type fileScanner struct {
	// sandbox provides file system access for directory walking and file stats.
	sandbox safedisk.Sandbox
}

// discoveredFile holds metadata for a markdown file found during scanning.
type discoveredFile struct {
	// absolutePath is the full path to the file on the filesystem.
	absolutePath string

	// relativePath is the file path from the collection root.
	relativePath string

	// size is the file size in bytes.
	size int64

	// modTime is the last modification time as a Unix timestamp.
	modTime int64
}

// scanDirectory recursively scans a directory for markdown files.
// The rootPath should be relative to the sandbox root (use "." for the root).
//
// Takes rootPath (string) which is the path relative to the sandbox root to
// scan.
//
// Returns []*discoveredFile which contains the discovered markdown files.
// Returns error when the directory cannot be read.
//
// Skips hidden directories (starting with '.'), node_modules, and .git.
// Only returns files with .md extension. Returns an empty slice if the
// directory is empty.
func (s *fileScanner) scanDirectory(ctx context.Context, rootPath string) ([]*discoveredFile, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Scanning directory for markdown files",
		logger_domain.String(logKeyPath, rootPath))

	if err := s.validateDirectory(rootPath); err != nil {
		return nil, fmt.Errorf("validating directory %q: %w", rootPath, err)
	}

	var files []*discoveredFile

	err := s.sandbox.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		return s.processWalkEntry(ctx, rootPath, path, d, err, &files)
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	l.Trace("Directory scan complete",
		logger_domain.String(logKeyPath, rootPath),
		logger_domain.Int("file_count", len(files)))

	return files, nil
}

// validateDirectory checks that the path exists and is a directory.
//
// Takes rootPath (string) which specifies the directory path to validate.
//
// Returns error when the path cannot be accessed or is not a directory.
func (s *fileScanner) validateDirectory(rootPath string) error {
	info, err := s.sandbox.Stat(rootPath)
	if err != nil {
		return fmt.Errorf("cannot access directory %q: %w", rootPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", rootPath)
	}
	return nil
}

// processWalkEntry handles a single entry during directory traversal.
//
// Takes rootPath (string) which is the base directory being scanned.
// Takes path (string) which is the full path to the current entry.
// Takes d (fs.DirEntry) which describes the current directory entry.
// Takes walkErr (error) which is any error from the walk function.
// Takes files (*[]*discoveredFile) which collects discovered markdown files.
//
// Returns error when the context is cancelled.
func (s *fileScanner) processWalkEntry(
	ctx context.Context,
	rootPath, path string,
	d fs.DirEntry,
	walkErr error,
	files *[]*discoveredFile,
) error {
	ctx, l := logger_domain.From(ctx, log)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if walkErr != nil {
		l.Warn("Error accessing path during scan",
			logger_domain.String(logKeyPath, path),
			logger_domain.Error(walkErr))
		return nil
	}

	if d.IsDir() {
		return s.handleDirectory(ctx, d.Name())
	}

	if !isMarkdownFile(d.Name()) {
		return nil
	}

	file, err := s.buildDiscoveredFile(ctx, rootPath, path, d)
	if err != nil {
		return nil
	}

	*files = append(*files, file)
	l.Trace("Discovered markdown file",
		logger_domain.String("relative_path", file.relativePath),
		logger_domain.Int64("size_bytes", file.size))

	return nil
}

// handleDirectory checks whether to skip or descend into a directory.
//
// Takes name (string) which is the directory name to check.
//
// Returns error when the directory should be skipped (returns fs.SkipDir).
func (*fileScanner) handleDirectory(ctx context.Context, name string) error {
	_, l := logger_domain.From(ctx, log)
	if shouldSkipDirectory(name) {
		l.Trace("Skipping directory",
			logger_domain.String("name", name))
		return fs.SkipDir
	}
	return nil
}

// buildDiscoveredFile creates a discoveredFile from a directory entry.
//
// Takes rootPath (string) which is the base path for relative path calculation.
// Takes path (string) which is the file path to process.
// Takes d (fs.DirEntry) which provides the directory entry information.
//
// Returns *discoveredFile which contains the file metadata and paths.
// Returns error when file info cannot be retrieved or path calculation fails.
func (s *fileScanner) buildDiscoveredFile(ctx context.Context, rootPath, path string, d fs.DirEntry) (*discoveredFile, error) {
	_, l := logger_domain.From(ctx, log)
	info, err := d.Info()
	if err != nil {
		l.Warn("Cannot get file info",
			logger_domain.String(logKeyPath, path),
			logger_domain.Error(err))
		return nil, fmt.Errorf("getting file info for %q: %w", path, err)
	}

	relPath := path
	if rootPath != "." {
		relPath, err = filepath.Rel(rootPath, path)
		if err != nil {
			l.Warn("Cannot calculate relative path",
				logger_domain.String(logKeyPath, path),
				logger_domain.Error(err))
			return nil, fmt.Errorf("calculating relative path for %q: %w", path, err)
		}
	}

	return &discoveredFile{
		absolutePath: filepath.Join(s.sandbox.Root(), path),
		relativePath: relPath,
		size:         info.Size(),
		modTime:      info.ModTime().Unix(),
	}, nil
}

// newFileScanner creates a new file scanner that uses the provided sandbox
// for secure file system access.
//
// Takes sandbox (safedisk.Sandbox) which provides secure file system access.
//
// Returns *fileScanner which is ready to scan files within the sandbox.
func newFileScanner(sandbox safedisk.Sandbox) *fileScanner {
	return &fileScanner{
		sandbox: sandbox,
	}
}

// isMarkdownFile checks if a filename has a .md extension.
//
// Takes name (string) which is the filename to check.
//
// Returns bool which is true if the filename ends with .md, ignoring case.
func isMarkdownFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".md")
}

// shouldSkipDirectory reports whether a directory should be skipped during
// scanning.
//
// Takes name (string) which is the directory name to check.
//
// Returns bool which is true when the directory should be skipped.
func shouldSkipDirectory(name string) bool {
	if name == "." {
		return false
	}

	if strings.HasPrefix(name, ".") {
		return true
	}

	skipDirs := map[string]bool{
		"node_modules": true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		"out":          true,
		".next":        true,
		".cache":       true,
		"coverage":     true,
		"__pycache__":  true,
	}

	return skipDirs[name]
}
