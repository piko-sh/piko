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

package compiler_adapters

import (
	"context"
	"fmt"
	"io/fs"
	"time"

	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// diskInputReader reads SFC files from disk using a sandboxed filesystem.
// All file operations are constrained to the sandbox root directory.
type diskInputReader struct {
	// sandbox provides safe file system access for reading source files.
	sandbox safedisk.Sandbox
}

// ReadSFC reads an SFC file from disk using the sandboxed filesystem.
//
// Takes sourceIdentifier (string) which specifies the file path relative to
// the sandbox root.
//
// Returns []byte which contains the file contents.
// Returns error when the file cannot be read or the path is a directory.
func (reader *diskInputReader) ReadSFC(ctx context.Context, sourceIdentifier string) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DiskInputReader.ReadSFC",
		logger_domain.String("sourceIdentifier", sourceIdentifier),
	)
	defer span.End()

	l = l.With(logger_domain.String("sourceIdentifier", sourceIdentifier))

	var fileInfo fs.FileInfo
	err := l.RunInSpan(ctx, "StatFile", func(_ context.Context, _ logger_domain.Logger) error {
		var err error
		fileInfo, err = reader.sandbox.Stat(sourceIdentifier)
		if err != nil {
			return fmt.Errorf("stat file %q: %w", sourceIdentifier, err)
		}
		return nil
	})

	if err != nil {
		l.ReportError(span, err, "Cannot stat file")
		fileReadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("cannot stat file at %s: %w", sourceIdentifier, err)
	}

	if fileInfo.IsDir() {
		l.Warn("Source path is a directory, not a file")
		fileReadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("reading SFC at %q: source path is a directory, not a file", sourceIdentifier)
	}

	var contentBytes []byte
	startTime := time.Now()

	err = l.RunInSpan(ctx, "ReadFile", func(_ context.Context, _ logger_domain.Logger) error {
		var err error
		contentBytes, err = reader.sandbox.ReadFile(sourceIdentifier)
		if err != nil {
			return fmt.Errorf("reading file %q: %w", sourceIdentifier, err)
		}
		return nil
	})

	readDuration := time.Since(startTime)
	fileReadDuration.Record(ctx, float64(readDuration.Milliseconds()))

	if err != nil {
		l.ReportError(span, err, "Failed reading file")
		fileReadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("failed reading file at %s: %w", sourceIdentifier, err)
	}

	contentSize := len(contentBytes)
	fileReadSize.Record(ctx, int64(contentSize))
	fileReadCount.Add(ctx, 1)

	l.Trace("Read SFC from disk", logger_domain.Int("contentSize", contentSize))

	return contentBytes, nil
}

// NewDiskInputReader creates a new input reader that uses the provided sandbox
// for secure file system access.
//
// Takes sandbox (safedisk.Sandbox) which provides secure file system access.
//
// Returns compiler_domain.InputReaderPort which is the configured input reader.
func NewDiskInputReader(sandbox safedisk.Sandbox) compiler_domain.InputReaderPort {
	return &diskInputReader{
		sandbox: sandbox,
	}
}
