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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pathutil"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// maxModuleRootSearchDepth bounds the upward walk that locates an external file's owning
	// module directory, guarding against a runaway loop on a pathological path.
	maxModuleRootSearchDepth = 256
)

var (
	_ generator_domain.FSReaderPort = (*FSReader)(nil)
)

var (
	// errNoModuleRoot is returned when an absolute path outside the project root has no
	// go.mod in any parent directory, so its owning module cannot be determined.
	errNoModuleRoot = errors.New("no owning module directory found for external file")
)

// FSReader implements FSReaderPort for reading files from the file system. It reads
// project files through the project sandbox and external Go module files through
// per-module read-only sandboxes, each confined to its module's own directory to prevent
// path traversal.
type FSReader struct {
	// sandbox provides safe file access within the project's source directory.
	sandbox safedisk.Sandbox

	// externalSandboxes caches one read-only sandbox per external module directory so files
	// from the same module reuse a single confined root rather than opening one per read.
	externalSandboxes map[string]safedisk.Sandbox

	// externalMu guards externalSandboxes.
	externalMu sync.Mutex
}

// NewFSReader creates a new file system reader that works within the given sandbox.
//
// The sandbox should be set up for the project's source folder.
//
// Takes sandbox (safedisk.Sandbox) which sets the allowed file system limits for reading
// files.
//
// Returns *FSReader which provides sandboxed file reading.
func NewFSReader(sandbox safedisk.Sandbox) *FSReader {
	return &FSReader{
		sandbox:           sandbox,
		externalSandboxes: make(map[string]safedisk.Sandbox),
		externalMu:        sync.Mutex{},
	}
}

// ReadFile reads the content of a file at the given path. It includes logging and
// OpenTelemetry metrics for observability.
//
// The filePath may be relative to the sandbox root, absolute within the sandbox root, or
// absolute and outside the root. The last case covers Piko components and assets imported
// from another Go module, which live outside the project base directory. Those absolute
// paths are read through a read-only sandbox rooted at the owning module's directory, so
// a read can never escape that module.
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

	data, err := r.readFile(filePath)
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

// Close releases every cached external-module sandbox. It is safe to call more than once.
//
// Returns error which joins any failures encountered while closing sandboxes.
//
// Concurrency: Safe for concurrent use; guarded by externalMu.
func (r *FSReader) Close() error {
	r.externalMu.Lock()
	defer r.externalMu.Unlock()

	var errs []error
	for moduleRoot, sandbox := range r.externalSandboxes {
		if err := sandbox.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing external sandbox %q: %w", moduleRoot, err))
		}
		delete(r.externalSandboxes, moduleRoot)
	}
	return errors.Join(errs...)
}

// readFile reads filePath through the appropriate sandbox for its location.
//
// A path within the project root is read through the project sandbox. An absolute path
// outside the root (a resolved external-module file) is read through a read-only sandbox
// rooted at the owning module's directory, which gives a confinement boundary a read
// cannot escape. The read is bounded by the sandbox's DefaultReadFileMaxBytes cap, which
// surfaces safedisk.ErrFileExceedsLimit rather than silently truncating an oversized
// file.
//
// Takes filePath (string) which is the path to read, relative or absolute.
//
// Returns []byte which is the file content.
// Returns error when the owning module cannot be located or the file cannot be read.
func (r *FSReader) readFile(filePath string) ([]byte, error) {
	if !filepath.IsAbs(filePath) || pathutil.Contains(r.sandbox.Root(), filePath) {
		return r.sandbox.ReadFile(r.sandbox.RelPath(filePath))
	}

	moduleRoot, err := findModuleRoot(filePath)
	if err != nil {
		return nil, err
	}

	sandbox, err := r.externalSandbox(moduleRoot)
	if err != nil {
		return nil, err
	}

	relativePath, err := filepath.Rel(moduleRoot, filePath)
	if err != nil {
		return nil, fmt.Errorf("locating %q within module %q: %w", filePath, moduleRoot, err)
	}

	return sandbox.ReadFile(relativePath)
}

// externalSandbox returns a cached read-only sandbox rooted at moduleRoot, creating one
// on first use. Concurrent first uses for the same module reuse a single sandbox; a
// sandbox that loses the creation race is closed.
//
// Takes moduleRoot (string) which is the absolute directory of an external module.
//
// Returns safedisk.Sandbox which is confined to moduleRoot.
// Returns error when the sandbox cannot be created.
func (r *FSReader) externalSandbox(moduleRoot string) (safedisk.Sandbox, error) {
	r.externalMu.Lock()
	existing, ok := r.externalSandboxes[moduleRoot]
	r.externalMu.Unlock()
	if ok {
		return existing, nil
	}

	created, err := safedisk.NewSandbox(moduleRoot, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating read-only sandbox for module directory %q: %w", moduleRoot, err)
	}

	r.externalMu.Lock()
	defer r.externalMu.Unlock()
	if existing, ok := r.externalSandboxes[moduleRoot]; ok {
		_ = created.Close()
		return existing, nil
	}
	r.externalSandboxes[moduleRoot] = created
	return created, nil
}

// findModuleRoot walks upward from filePath to the nearest ancestor directory containing
// a go.mod, which is the owning module's root. The walk is bounded to guard against a
// runaway loop.
//
// Takes filePath (string) which is the absolute path of an external module file.
//
// Returns string which is the absolute module root directory.
// Returns error which wraps errNoModuleRoot when no go.mod is found in any parent.
func findModuleRoot(filePath string) (string, error) {
	directory := filepath.Dir(filePath)
	for range maxModuleRootSearchDepth {
		info, err := os.Stat(filepath.Join(directory, "go.mod"))
		if err == nil && !info.IsDir() {
			return directory, nil
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", fmt.Errorf("%w: %q", errNoModuleRoot, filePath)
}
