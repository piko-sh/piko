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
	"maps"
	"path/filepath"
	"strings"
	"sync"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// dirPermissions is the permission mode for created directories.
	dirPermissions = 0750

	// filePermissions is the file mode for created files (owner read/write only).
	filePermissions = 0600
)

var _ generator_domain.PKJSEmitterPort = (*DiskPKJSEmitter)(nil)

// DiskPKJSEmitter implements PKJSEmitterPort to transpile and write JavaScript
// directly to disk. This is used for testing to capture generated JS files
// for golden file verification.
//
// Unlike PKJSEmitter which stores to a registry, this adapter writes files
// directly to the filesystem, making them available for test assertions.
type DiskPKJSEmitter struct {
	// transpiler converts TypeScript source code to JavaScript.
	transpiler *generator_domain.JSTranspiler

	// writtenFiles maps artefact IDs to their output file paths. Key format:
	// "pk-js/pages/checkout.js".
	writtenFiles map[string]string

	// sandbox provides sandboxed filesystem access for writing JS files.
	sandbox safedisk.Sandbox

	// factory creates sandboxes with validated paths. When set and sandbox is
	// nil, the factory is used before falling back to NewNoOpSandbox.
	factory safedisk.Factory

	// outputDir is the base folder where JS files are saved.
	outputDir string

	// mu guards writtenFiles for safe concurrent access.
	mu sync.Mutex

	// minify controls whether the output is minified.
	minify bool
}

// DiskPKJSEmitterOption configures a DiskPKJSEmitter.
type DiskPKJSEmitterOption func(*DiskPKJSEmitter)

// NewDiskPKJSEmitter creates a new disk-based PK JavaScript emitter.
//
// Takes outputDir (string) which is the base directory where JS files will be
// written. Files are written as <outputDir>/<artefactID>.
// Takes minify (bool) which controls whether to minify the output JavaScript.
// Takes opts (...DiskPKJSEmitterOption) which provides optional configuration.
//
// Returns *DiskPKJSEmitter which is the configured emitter ready for use.
func NewDiskPKJSEmitter(outputDir string, minify bool, opts ...DiskPKJSEmitterOption) *DiskPKJSEmitter {
	e := &DiskPKJSEmitter{
		transpiler:   generator_domain.NewJSTranspiler(),
		writtenFiles: make(map[string]string),
		outputDir:    outputDir,
		mu:           sync.Mutex{},
		minify:       minify,
	}

	for _, opt := range opts {
		opt(e)
	}

	if e.sandbox == nil {
		var sandbox safedisk.Sandbox
		var err error
		if e.factory != nil {
			sandbox, err = e.factory.Create("disk PK JS emitter", outputDir, safedisk.ModeReadWrite)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(outputDir, safedisk.ModeReadWrite)
		}
		if err == nil {
			e.sandbox = sandbox
		}
	}

	return e
}

// EmitJS transpiles TypeScript/JavaScript source and writes it to disk. The
// artefact ID uses the pk-js/ prefix for consistency with the registry adapter.
//
// For example:
// pagePath: "pages/checkout"
// outputDir: "/tmp/golden"
// -> artefact ID: "pk-js/pages/checkout.js"
// -> file written to: /tmp/golden/pk-js/pages/checkout.js
//
// Takes source (string) which is the TypeScript/JavaScript source code
// to transpile.
// Takes pagePath (string) which identifies the page this script
// belongs to.
//
// Returns string which is the artefact ID that can be used to look up
// the file path via GetWrittenFilePath.
// Returns error when transpilation or file writing fails.
//
// Safe for concurrent use. Access to the written files map is
// serialised by an internal mutex.
func (e *DiskPKJSEmitter) EmitJS(
	ctx context.Context,
	source string,
	pagePath string,
	moduleName string,
	_ string,
	_ bool,
) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}

	cleanPath := strings.TrimSuffix(pagePath, ".pk")

	componentName := strings.TrimPrefix(cleanPath, "partials/")
	if componentName == cleanPath {
		componentName = ""
	}

	transformedSource := generator_domain.TransformPKSource(source, componentName)

	filename := filepath.Base(cleanPath) + ".ts"
	result, err := e.transpiler.Transpile(ctx, transformedSource, generator_domain.TranspileOptions{
		Filename:   filename,
		Minify:     e.minify,
		ModuleName: moduleName,
	})
	if err != nil {
		return "", fmt.Errorf("transpiling PK JS for %s: %w", pagePath, err)
	}

	artefactID := fmt.Sprintf("pk-js/%s.js", cleanPath)

	if err := e.sandbox.MkdirAll(filepath.Dir(artefactID), dirPermissions); err != nil {
		return "", fmt.Errorf("creating directory for PK JS %s: %w", artefactID, err)
	}

	if err := e.sandbox.WriteFile(artefactID, []byte(result.Code), filePermissions); err != nil {
		return "", fmt.Errorf("writing PK JS %s to disk: %w", artefactID, err)
	}

	e.mu.Lock()
	e.writtenFiles[artefactID] = filepath.Join(e.sandbox.Root(), artefactID)
	e.mu.Unlock()

	return artefactID, nil
}

// GetWrittenFilePath returns the absolute path where a given artefact was
// written.
//
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns string which is the file path, or empty if the artefact was not
// written.
//
// Safe for concurrent use.
func (e *DiskPKJSEmitter) GetWrittenFilePath(artefactID string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.writtenFiles[artefactID]
}

// GetAllWrittenFiles returns a copy of all written files.
// Key: artefactID, Value: absolute path.
//
// Returns map[string]string which maps artefact IDs to absolute file paths.
//
// Safe for concurrent use.
func (e *DiskPKJSEmitter) GetAllWrittenFiles() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make(map[string]string, len(e.writtenFiles))
	maps.Copy(result, e.writtenFiles)
	return result
}

// Reset clears the record of written files.
// Use it between test cases.
//
// Safe for concurrent use.
func (e *DiskPKJSEmitter) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.writtenFiles = make(map[string]string)
}

// WithEmitterFactory sets the sandbox factory for the disk PK JS emitter. When
// no sandbox is injected, the factory is tried before falling back to
// NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns DiskPKJSEmitterOption which configures the emitter with the factory.
func WithEmitterFactory(factory safedisk.Factory) DiskPKJSEmitterOption {
	return func(e *DiskPKJSEmitter) {
		e.factory = factory
	}
}

// WithEmitterSandbox returns an option that injects a sandbox for filesystem
// operations. The emitter uses this sandbox instead of creating one from the
// output directory, allowing testing with MockSandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides the filesystem sandbox to
// use.
//
// Returns DiskPKJSEmitterOption which configures the emitter with the sandbox.
func WithEmitterSandbox(sandbox safedisk.Sandbox) DiskPKJSEmitterOption {
	return func(e *DiskPKJSEmitter) {
		e.sandbox = sandbox
	}
}
