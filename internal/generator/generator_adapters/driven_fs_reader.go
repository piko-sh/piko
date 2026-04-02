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
	"time"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// FSReader implements FSReaderPort for reading files from the file system.
// It uses a sandbox to prevent path traversal attacks.
type FSReader struct {
	// sandbox provides safe file access within a set directory.
	sandbox safedisk.Sandbox
}

var _ generator_domain.FSReaderPort = (*FSReader)(nil)

// NewFSReader creates a new file system reader that works within the given
// sandbox.
//
// The sandbox should be set up for the project's source folder.
//
// Takes sandbox (safedisk.Sandbox) which sets the allowed file system limits
// for reading files.
//
// Returns *FSReader which provides sandboxed file reading.
func NewFSReader(sandbox safedisk.Sandbox) *FSReader {
	return &FSReader{sandbox: sandbox}
}

// ReadFile reads the content of a file at the given path.
// It includes logging and OpenTelemetry metrics for observability.
//
// The filePath can be absolute (within the sandbox root) or relative.
// Absolute paths are changed to relative paths within the sandbox.
// Relative paths with extra sandbox directory prefixes are cleaned.
//
// Takes filePath (string) which specifies the path to the file to read.
//
// Returns []byte which contains the file content.
// Returns error when the file cannot be read.
func (r *FSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	ctx, span, l := log.Span(ctx, "FSReader.ReadFile", logger_domain.String("path", filePath))
	defer span.End()

	fileReadCount.Add(ctx, 1)
	startTime := time.Now()

	relPath := r.sandbox.RelPath(filePath)

	data, err := r.sandbox.ReadFile(relPath)
	if err != nil {
		fileReadErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to read file from disk")
		return nil, fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	duration := time.Since(startTime)
	fileReadDuration.Record(ctx, float64(duration.Milliseconds()))

	l.Trace("Read file successfully.", logger_domain.Int("size_bytes", len(data)))
	return data, nil
}
