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

package lsp_adapters

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// lspFSReader is a driven adapter that implements FSReaderPort for providing
// live diagnostics on unsaved files. It first consults the LSP server's
// in-memory document cache for open files with unsaved changes, then falls
// back to the physical disk for files not in the cache.
type lspFSReader struct {
	// docCache stores live, unsaved document content for cache lookups.
	docCache *lsp_domain.DocumentCache

	// fallback reads from the file system when files are not in the cache.
	fallback annotator_domain.FSReaderPort
}

var (
	_ annotator_domain.FSReaderPort = (*lspFSReader)(nil)

	_ annotator_domain.FSReaderPort = (*osFSReader)(nil)
)

// ReadFile implements the FSReaderPort interface by reading file content.
// It checks the in-memory cache first before falling back to the disk.
//
// Takes filePath (string) which specifies the absolute path to the file.
//
// Returns []byte which contains the file content.
// Returns error when the file cannot be read from disk.
func (r *lspFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)

	fileURI := uri.File(filePath)

	if content, found := r.docCache.Get(protocol.DocumentURI(fileURI)); found {
		l.Trace("lspFSReader: Cache HIT - using in-memory content",
			logger_domain.String("filePath", filePath),
			logger_domain.Int("contentSize", len(content)))
		return content, nil
	}

	l.Debug("lspFSReader: Cache MISS, falling back to disk", logger_domain.String("filePath", filePath))
	return r.fallback.ReadFile(ctx, filePath)
}

// osFSReader implements FSReaderPort by reading files from the file system.
type osFSReader struct {
	// sandbox is an optional sandbox for testing. When nil, a sandbox is
	// created for each ReadFile call based on the file's parent directory.
	sandbox safedisk.Sandbox

	// factory creates sandboxes for filesystem access. When nil, falls back
	// to safedisk.NewNoOpSandbox for each call.
	factory safedisk.Factory
}

// OsFSReaderOption configures an osFSReader during creation.
type OsFSReaderOption func(*osFSReader)

// ReadFile implements the FSReaderPort interface using sandboxed file access.
//
// Takes filePath (string) which specifies the path to the file to read.
//
// Returns []byte which contains the file contents.
// Returns error when the sandbox cannot be created or the file cannot be read.
func (r *osFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fileName := filepath.Base(filePath)

	if r.sandbox != nil {
		content, err := r.sandbox.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("osFSReader failed to read file '%s': %w", filePath, err)
		}
		return content, nil
	}

	parentDir := filepath.Dir(filePath)
	var sandbox safedisk.Sandbox
	var err error
	if r.factory != nil {
		sandbox, err = r.factory.Create("lsp-fs-read", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, fmt.Errorf("osFSReader failed to create sandbox for '%s': %w", parentDir, err)
	}
	defer func() { _ = sandbox.Close() }()

	content, err := sandbox.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("osFSReader failed to read file '%s': %w", filePath, err)
	}
	return content, nil
}

// NewLspFSReader creates a new, configured file reader adapter for the LSP.
//
// Takes docCache (*lsp_domain.DocumentCache) which provides the LSP domain's
// document cache for accessing open documents.
// Takes fallback (annotator_domain.FSReaderPort) which provides disk access
// for files not in the cache.
//
// Returns annotator_domain.FSReaderPort which reads files from the document
// cache with disk fallback.
// Returns error when docCache or fallback is nil.
func NewLspFSReader(docCache *lsp_domain.DocumentCache, fallback annotator_domain.FSReaderPort) (annotator_domain.FSReaderPort, error) {
	switch {
	case docCache == nil:
		return nil, errors.New("lspFSReader: docCache cannot be nil")
	case fallback == nil:
		return nil, errors.New("lspFSReader: fallback reader cannot be nil")
	}
	return &lspFSReader{
		docCache: docCache,
		fallback: fallback,
	}, nil
}

// WithOsFSReaderSandbox injects a sandbox for testing filesystem
// operations, using it for all ReadFile calls instead of creating a
// new sandbox per call.
//
// The caller is responsible for closing the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access.
//
// Returns OsFSReaderOption which configures the reader with the given sandbox.
func WithOsFSReaderSandbox(sandbox safedisk.Sandbox) OsFSReaderOption {
	return func(r *osFSReader) {
		r.sandbox = sandbox
	}
}

// WithOsFSReaderFactory sets a sandbox factory for creating sandboxes per
// ReadFile call instead of falling back to no-op sandboxes.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns OsFSReaderOption which configures the reader with the given factory.
func WithOsFSReaderFactory(factory safedisk.Factory) OsFSReaderOption {
	return func(r *osFSReader) {
		r.factory = factory
	}
}

// NewOsFSReader creates a new file system reader that reads from disk.
//
// Takes opts (...OsFSReaderOption) which provides optional configuration
// such as WithOsFSReaderSandbox for testing.
//
// Returns annotator_domain.FSReaderPort which provides file system access.
func NewOsFSReader(opts ...OsFSReaderOption) annotator_domain.FSReaderPort {
	r := &osFSReader{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}
