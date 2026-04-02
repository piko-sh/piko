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

package generator_adapters

import (
	"context"
	"fmt"
	"os"
	"time"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// FSWriter implements FSWriterPort as a driven adapter that writes generated
// files to disk. It writes files atomically to prevent corruption from build
// interruptions, and restricts all operations to a sandbox to block path
// traversal attacks.
type FSWriter struct {
	// sandbox provides safe file system access within allowed paths.
	sandbox safedisk.Sandbox
}

var _ generator_domain.FSWriterPort = (*FSWriter)(nil)

// NewFSWriter creates a new file system writer that works within the given
// sandbox. The sandbox should be set up with ModeReadWrite for the output
// folder.
//
// Takes sandbox (safedisk.Sandbox) which sets the allowed file paths.
//
// Returns *FSWriter which is ready for writing files within the sandbox.
func NewFSWriter(sandbox safedisk.Sandbox) *FSWriter {
	return &FSWriter{sandbox: sandbox}
}

// WriteFile writes data to the specified file path using an atomic
// write-then-rename strategy. This method is safe from race conditions and
// interruptions.
//
// The filePath can be either absolute (within sandbox root) or relative.
// Absolute paths are changed to relative paths within the sandbox. Relative
// paths that include the sandbox directory name (e.g., "dist/file.go" when
// sandbox is rooted at "dist") have that prefix removed.
//
// Takes filePath (string) which specifies the destination file path.
// Takes data ([]byte) which contains the content to write.
//
// Returns error when the atomic write operation fails.
func (w *FSWriter) WriteFile(ctx context.Context, filePath string, data []byte) error {
	ctx, span, l := log.Span(ctx, "FSWriter.WriteFile", logger_domain.String("path", filePath))
	defer span.End()

	fileWriteCount.Add(ctx, 1)
	startTime := time.Now()

	relPath := w.sandbox.RelPath(filePath)

	if err := generator_domain.AtomicWriteFile(ctx, w.sandbox, relPath, data, generator_domain.FilePermission); err != nil {
		fileWriteErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to write file atomically")
		return fmt.Errorf("writing file atomically to %q: %w", relPath, err)
	}

	duration := time.Since(startTime)
	fileWriteDuration.Record(ctx, float64(duration.Milliseconds()))

	l.Trace("Wrote file successfully.", logger_domain.Int("size_bytes", len(data)))
	return nil
}

// ReadDir reads the directory at the given path and returns a list of
// directory entries sorted by filename.
//
// Takes dirname (string) which specifies the directory path to read.
//
// Returns []os.DirEntry which contains the directory entries.
// Returns error when the directory cannot be read.
func (w *FSWriter) ReadDir(dirname string) ([]os.DirEntry, error) {
	relPath := w.sandbox.RelPath(dirname)
	return w.sandbox.ReadDir(relPath)
}

// RemoveAll removes path and any children it contains.
//
// Takes path (string) which specifies the path to remove.
//
// Returns error when the removal fails.
func (w *FSWriter) RemoveAll(path string) error {
	relPath := w.sandbox.RelPath(path)
	return w.sandbox.RemoveAll(relPath)
}
