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

package wasm_adapters

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/seo/seo_dto"
)

const (
	// pkJSTranspileCacheCap caps the long-lived transpile cache between
	// Generate runs. Entries beyond this count are evicted at the next
	// Sweep; transpile re-runs cost ~ms so the cap keeps WASM memory
	// bounded without hurting keystroke-rate cache hit rate.
	pkJSTranspileCacheCap = 256

	// pkJSEmitterComponent identifies the in-memory PKJS emitter for
	// safecall panic logging; see goroutine.SafeCall1.
	pkJSEmitterComponent = "wasm_adapters.InMemoryPKJSEmitter.Transpile"

	// pkJSCacheVersion participates in every cache key so a deliberate
	// bump invalidates every long-lived entry across an upgrade. Bump
	// when TransformPKSource semantics or the JSTranspiler import-rewrite
	// rules change so a hot-reloaded WASM module never serves stale
	// output.
	pkJSCacheVersion = "v2"
)

// errPKJSPathInvalid is returned by EmitJS / Put when the caller-supplied
// page path is empty after normalisation, contains a parent-directory
// segment, or otherwise resolves outside the artefact namespace.
var errPKJSPathInvalid = errors.New("invalid PKJS artefact path")

var (
	_ generator_domain.CollectionEmitterPort = (*NoOpCollectionEmitter)(nil)

	_ generator_domain.SearchIndexEmitterPort = (*NoOpSearchIndexEmitter)(nil)

	_ generator_domain.I18nEmitterPort = (*NoOpI18nEmitter)(nil)

	_ generator_domain.ActionGeneratorPort = (*NoOpActionGenerator)(nil)

	_ generator_domain.RegisterEmitterPort = (*InMemoryRegisterEmitter)(nil)

	_ generator_domain.SEOServicePort = (*NoOpSEOService)(nil)

	_ generator_domain.PKJSEmitterPort = (*InMemoryPKJSEmitter)(nil)

	_ generator_domain.ManifestEmitterPort = (*InMemoryManifestEmitter)(nil)
)

// NoOpCollectionEmitter implements CollectionEmitterPort as a no-op.
// It is used for WASM contexts where collection binary emission is not needed.
type NoOpCollectionEmitter struct{}

// NewNoOpCollectionEmitter creates a new no-op collection emitter.
//
// Returns *NoOpCollectionEmitter which silently ignores all emission requests.
func NewNoOpCollectionEmitter() *NoOpCollectionEmitter {
	return &NoOpCollectionEmitter{}
}

// EmitCollection is a no-op that returns an empty package path.
//
// Returns string which is always empty.
// Returns error which is always nil.
func (*NoOpCollectionEmitter) EmitCollection(
	_ context.Context,
	_ string,
	_ []collection_dto.ContentItem,
	_ string,
) (packagePath string, err error) {
	return "", nil
}

// NoOpSearchIndexEmitter implements SearchIndexEmitterPort as a no-op.
// It is used for WASM contexts where search index emission is not needed.
type NoOpSearchIndexEmitter struct{}

// NewNoOpSearchIndexEmitter creates a new no-op search index emitter.
//
// Returns *NoOpSearchIndexEmitter which silently ignores all emission requests.
func NewNoOpSearchIndexEmitter() *NoOpSearchIndexEmitter {
	return &NoOpSearchIndexEmitter{}
}

// EmitSearchIndex is a no-op that does nothing.
//
// Returns error which is always nil.
func (*NoOpSearchIndexEmitter) EmitSearchIndex(
	_ context.Context,
	_ string,
	_ []collection_dto.ContentItem,
	_ string,
	_ []string,
) error {
	return nil
}

// NoOpI18nEmitter implements I18nEmitterPort as a no-op.
// It is used for WASM contexts where i18n binary emission is not needed.
type NoOpI18nEmitter struct{}

// NewNoOpI18nEmitter creates a new no-op i18n emitter.
//
// Returns *NoOpI18nEmitter which silently ignores all emission requests.
func NewNoOpI18nEmitter() *NoOpI18nEmitter {
	return &NoOpI18nEmitter{}
}

// EmitI18n is a no-op that does nothing.
//
// Returns error which is always nil.
func (*NoOpI18nEmitter) EmitI18n(_ context.Context, _ string) error {
	return nil
}

// NoOpActionGenerator implements ActionGeneratorPort as a no-op.
// It is used for WASM contexts where action generation is not needed.
type NoOpActionGenerator struct{}

// NewNoOpActionGenerator creates a new no-op action generator.
//
// Returns *NoOpActionGenerator which silently ignores all generation requests.
func NewNoOpActionGenerator() *NoOpActionGenerator {
	return &NoOpActionGenerator{}
}

// GenerateActions is a no-op that does nothing.
//
// Returns error which is always nil.
func (*NoOpActionGenerator) GenerateActions(
	_ context.Context,
	_ *annotator_dto.ActionManifest,
	_ string,
	_ string,
) error {
	return nil
}

// InMemoryRegisterEmitter implements RegisterEmitterPort using in-memory storage.
// It generates proper register code for WASM contexts and writes to an FSWriter.
type InMemoryRegisterEmitter struct {
	// fsWriter writes the generated output to the in-memory file system.
	fsWriter generator_domain.FSWriterPort

	// content stores the last generated register file content.
	content []byte

	// mu protects concurrent access.
	mu sync.RWMutex
}

// NewInMemoryRegisterEmitter creates a new in-memory register emitter.
//
// Takes fsWriter (generator_domain.FSWriterPort) which handles file system writes.
//
// Returns *InMemoryRegisterEmitter which generates and stores register code.
func NewInMemoryRegisterEmitter(fsWriter generator_domain.FSWriterPort) *InMemoryRegisterEmitter {
	return &InMemoryRegisterEmitter{fsWriter: fsWriter}
}

// Emit generates register code and writes it to the file system.
//
// Takes outputPath (string) which is the output path for the register file.
// Takes allPackagePaths ([]string) which are the package paths to import.
//
// Returns error when generation or writing fails.
//
// Safe for concurrent use. Uses a mutex to protect the internal content field.
func (e *InMemoryRegisterEmitter) Emit(ctx context.Context, outputPath string, allPackagePaths []string) error {
	content, err := e.Generate(ctx, allPackagePaths)
	if err != nil {
		return fmt.Errorf("generating register content: %w", err)
	}
	e.mu.Lock()
	e.content = content
	e.mu.Unlock()

	return e.fsWriter.WriteFile(ctx, outputPath, content)
}

// Generate creates register file content for the given package paths.
//
// Takes allPackagePaths ([]string) which are the package paths to import.
//
// Returns []byte which contains the register file content.
// Returns error which is always nil.
func (*InMemoryRegisterEmitter) Generate(_ context.Context, allPackagePaths []string) ([]byte, error) {
	if len(allPackagePaths) == 0 {
		return []byte(`// Code generated by Piko - DO NOT EDIT.
// This file imports all compiled component packages to ensure they are included
// in the final binary and their init() functions are executed.

package dist
`), nil
	}

	var buffer strings.Builder
	buffer.WriteString("// Code generated by Piko - DO NOT EDIT.\n")
	buffer.WriteString("// This file imports all compiled component packages to ensure they are included\n")
	buffer.WriteString("// in the final binary and their init() functions are executed.\n\n")
	buffer.WriteString("package dist\n\n")
	buffer.WriteString("import (\n")
	for _, packagePath := range allPackagePaths {
		buffer.WriteString("\t_ \"")
		buffer.WriteString(packagePath)
		buffer.WriteString("\"\n")
	}
	buffer.WriteString(")\n")

	return []byte(buffer.String()), nil
}

// GetContent returns the last generated register file content.
//
// Returns []byte which contains the stored content.
//
// Safe for concurrent use.
func (e *InMemoryRegisterEmitter) GetContent() []byte {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.content
}

// NoOpSEOService implements SEOServicePort as a no-op.
// It is used for WASM contexts where SEO artefact generation is not needed.
type NoOpSEOService struct{}

// NewNoOpSEOService creates a new no-op SEO service.
//
// Returns *NoOpSEOService which silently ignores all generation requests.
func NewNoOpSEOService() *NoOpSEOService {
	return &NoOpSEOService{}
}

// GenerateArtefacts is a no-op that does nothing.
//
// Returns error which is always nil.
func (*NoOpSEOService) GenerateArtefacts(_ context.Context, _ *seo_dto.ProjectView) error {
	return nil
}

// InMemoryPKJSEmitter implements PKJSEmitterPort by transforming and
// transpiling client-side TypeScript in memory. It mirrors DiskPKJSEmitter's
// compile pipeline (TransformPKSource then JSTranspiler.Transpile) without
// writing to a filesystem, and exposes the captured JavaScript via
// GetArtefacts so the WASM response surface can include it.
type InMemoryPKJSEmitter struct {
	// transpiler converts TypeScript source code to JavaScript using
	// esbuild's parser/printer. Reused across EmitJS calls.
	transpiler *generator_domain.JSTranspiler

	// artefacts maps artefact IDs to their compiled JavaScript content.
	// Reset at the start of each Generate so a response only carries the
	// JS produced for that run.
	artefacts map[string]string

	// transpileCache memoises transpile output keyed by a content hash
	// of (transformed source + moduleName + filename + cache version).
	// Long-lived across Generate calls; eviction happens in Sweep.
	transpileCache map[string]string

	// producedThisRun records every content-hash hit in EmitJS during
	// the current Generate so Sweep can drop cache entries that didn't
	// participate this run.
	producedThisRun map[string]struct{}

	// mu serialises every public method end-to-end so per-run state
	// (artefacts, producedThisRun) cannot be observed mid-mutation by a
	// concurrent caller.
	mu sync.Mutex
}

// NewInMemoryPKJSEmitter creates a new in-memory PKJS emitter.
//
// Returns *InMemoryPKJSEmitter which captures emitted JavaScript.
func NewInMemoryPKJSEmitter() *InMemoryPKJSEmitter {
	return &InMemoryPKJSEmitter{
		transpiler:      generator_domain.NewJSTranspiler(),
		artefacts:       make(map[string]string),
		transpileCache:  make(map[string]string),
		producedThisRun: make(map[string]struct{}),
	}
}

// EmitJS transforms PK source and stores the transpiled JavaScript.
//
// Transforms client-side PK source via TransformPKSource, transpiles the
// result to ES module JavaScript, and stores it under a pk-js/<path>.js
// artefact ID. The WASM build produces JavaScript byte-identical to the
// disk build for equivalent inputs. Empty or whitespace-only source
// returns ("", nil) without storing anything.
//
// Takes source (string) which is the raw client-script TypeScript extracted
// by the annotator from a .pkc <script> block.
// Takes pagePath (string) which identifies the page or partial. Cleaned via
// path.Clean and rejected if it escapes the artefact namespace.
// Takes moduleName (string) which is the Go module name; used by the
// transpiler to rewrite "@/" import aliases.
//
// Returns string which is the artefact ID ("pk-js/<cleanPath>.js"), or empty
// when source is whitespace-only.
// Returns error when the path is invalid or transpilation fails or panics.
//
// Concurrency: Safe for concurrent use; the entire body runs under the
// emitter mutex.
func (e *InMemoryPKJSEmitter) EmitJS(
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

	cleanPath, err := normalisePKJSPath(pagePath)
	if err != nil {
		return "", err
	}

	componentName := strings.TrimPrefix(cleanPath, "partials/")
	if componentName == cleanPath {
		componentName = ""
	}

	transformedSource := generator_domain.TransformPKSource(source, componentName)
	filename := filepath.Base(cleanPath) + ".ts"
	cacheKey := pkJSCacheKey(transformedSource, moduleName, filename)
	artefactID := path.Join("pk-js", cleanPath) + ".js"

	e.mu.Lock()
	defer e.mu.Unlock()

	if cached, hit := e.transpileCache[cacheKey]; hit {
		e.producedThisRun[cacheKey] = struct{}{}
		e.artefacts[artefactID] = cached
		return artefactID, nil
	}

	code, err := goroutine.SafeCall1(ctx, pkJSEmitterComponent, func() (string, error) {
		result, transpileErr := e.transpiler.Transpile(ctx, transformedSource, generator_domain.TranspileOptions{
			Filename:   filename,
			Minify:     false,
			ModuleName: moduleName,
		})
		if transpileErr != nil {
			return "", transpileErr
		}
		return result.Code, nil
	})
	if err != nil {
		return "", fmt.Errorf("transpiling PK JS for %s: %w", pagePath, err)
	}

	e.transpileCache[cacheKey] = code
	e.producedThisRun[cacheKey] = struct{}{}
	e.artefacts[artefactID] = code
	return artefactID, nil
}

// pkJSCacheKey derives a 64-bit content hash for the transpile cache.
//
// moduleName is required because RewriteImportRecords embeds it into
// rewritten "@/" alias paths; filename keeps transpile error messages
// and source-map metadata accurate; pkJSCacheVersion gives us a
// kill-switch for stale entries after upgrade. xxhash collisions only
// cost a redundant transpile, not a correctness or security bug.
//
// Takes transformedSource (string) which is the post-transform TypeScript fed
// to the transpiler.
// Takes moduleName (string) which the transpiler embeds into rewritten "@/"
// alias paths.
// Takes filename (string) which the transpiler uses for error messages and
// source-map metadata.
//
// Returns string which is the hex-encoded 64-bit cache key.
func pkJSCacheKey(transformedSource, moduleName, filename string) string {
	h := xxhash.New()
	_, _ = h.WriteString(pkJSCacheVersion)
	_, _ = h.WriteString("\x00")
	_, _ = h.WriteString(transformedSource)
	_, _ = h.WriteString("\x00")
	_, _ = h.WriteString(moduleName)
	_, _ = h.WriteString("\x00")
	_, _ = h.WriteString(filename)
	var digest [8]byte
	binary.BigEndian.PutUint64(digest[:], h.Sum64())
	return hex.EncodeToString(digest[:])
}

// normalisePKJSPath cleans the caller-supplied page path.
//
// Rejects values that would escape the artefact namespace. The rules
// match the safedisk-sandboxed disk path: no parent-directory segments,
// no absolute paths, no empty result.
//
// Takes pagePath (string) which is the caller-supplied page path.
//
// Returns string which is the cleaned path with the .pk suffix stripped.
// Returns error which wraps errPKJSPathInvalid when the path is unusable as an
// artefact ID.
func normalisePKJSPath(pagePath string) (string, error) {
	if pagePath == "" {
		return "", fmt.Errorf("%w: empty page path", errPKJSPathInvalid)
	}
	withoutSuffix := strings.TrimSuffix(pagePath, ".pk")
	cleaned := path.Clean(withoutSuffix)
	if cleaned == "." || cleaned == "/" || cleaned == "" {
		return "", fmt.Errorf("%w: %q", errPKJSPathInvalid, pagePath)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("%w: absolute path %q", errPKJSPathInvalid, pagePath)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: parent traversal in %q", errPKJSPathInvalid, pagePath)
	}
	return cleaned, nil
}

// GetArtefacts returns a copy of all emitted JavaScript artefacts.
//
// The returned map is a snapshot so the caller can iterate without
// holding the emitter lock.
//
// Returns map[string]string which maps artefact IDs to JavaScript content.
//
// Concurrency: Safe for concurrent use; copies under the emitter mutex.
func (e *InMemoryPKJSEmitter) GetArtefacts() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make(map[string]string, len(e.artefacts))
	maps.Copy(result, e.artefacts)
	return result
}

// Reset clears per-run state.
//
// Leaves the long-lived transpile cache untouched. The GeneratorAdapter
// calls Reset at the start of each Generate.
//
// Concurrency: Safe for concurrent use; mutates under the emitter mutex.
func (e *InMemoryPKJSEmitter) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	clear(e.artefacts)
	clear(e.producedThisRun)
}

// Put stores pre-compiled JavaScript verbatim.
//
// Bypasses TransformPKSource and the transpile cache. This is the entry
// point for .pkc client-side components, which go through the SFC
// compiler rather than the partial-style transform that EmitJS applies
// to inline <script> blocks inside .pk pages and partials. Empty content
// is a no-op.
//
// Takes artefactID (string) which is the relative artefact path (e.g.
// "pk-js/components/pp-counter.js") cleaned and validated before storage.
// Takes content (string) which is the pre-compiled JavaScript.
//
// Returns error which wraps errPKJSPathInvalid when artefactID is empty,
// absolute, or contains a parent-directory segment after cleaning.
//
// Concurrency: Safe for concurrent use; mutates under the emitter mutex.
func (e *InMemoryPKJSEmitter) Put(artefactID, content string) error {
	if content == "" {
		return nil
	}
	cleaned, err := normalisePutArtefactID(artefactID)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.artefacts[cleaned] = content
	return nil
}

// normalisePutArtefactID validates an artefact ID supplied to Put.
//
// The ID is expected to be a forward-slash relative path (e.g.
// "pk-js/components/pp-counter.js"). Empty, absolute, or
// parent-traversal inputs are rejected with errPKJSPathInvalid.
//
// Takes artefactID (string) which is the candidate artefact ID.
//
// Returns string which is the cleaned, validated artefact ID.
// Returns error which wraps errPKJSPathInvalid for unusable inputs.
func normalisePutArtefactID(artefactID string) (string, error) {
	if artefactID == "" {
		return "", fmt.Errorf("%w: empty artefact id", errPKJSPathInvalid)
	}
	cleaned := path.Clean(artefactID)
	if cleaned == "." || cleaned == "/" {
		return "", fmt.Errorf("%w: %q", errPKJSPathInvalid, artefactID)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("%w: absolute artefact id %q", errPKJSPathInvalid, artefactID)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: parent traversal in %q", errPKJSPathInvalid, artefactID)
	}
	return cleaned, nil
}

// Sweep prunes the transpile cache after a Generate run.
//
// Evicts cache entries whose source hashes were not consumed during the
// current Generate, then trims any remaining excess to keep the cache
// under pkJSTranspileCacheCap. producedThisRun is cleared so the next
// EmitJS pass starts from a clean tracking set even if the caller
// forgets to call Reset.
//
// Concurrency: Safe for concurrent use; mutates under the emitter mutex.
func (e *InMemoryPKJSEmitter) Sweep() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for key := range e.transpileCache {
		if _, used := e.producedThisRun[key]; !used {
			delete(e.transpileCache, key)
		}
	}
	clear(e.producedThisRun)

	for len(e.transpileCache) > pkJSTranspileCacheCap {
		for key := range e.transpileCache {
			delete(e.transpileCache, key)
			break
		}
	}
}

// InMemoryManifestEmitter implements ManifestEmitterPort using an in-memory
// buffer. It captures the manifest for later retrieval instead of writing to
// disk.
type InMemoryManifestEmitter struct {
	// manifest is the captured manifest.
	manifest *generator_dto.Manifest

	// mu protects concurrent access.
	mu sync.RWMutex
}

// NewInMemoryManifestEmitter creates a new in-memory manifest emitter.
//
// Returns *InMemoryManifestEmitter which captures the manifest.
func NewInMemoryManifestEmitter() *InMemoryManifestEmitter {
	return &InMemoryManifestEmitter{}
}

// EmitCode stores the manifest in memory instead of writing to
// disk.
//
// Takes manifest (*generator_dto.Manifest) which is the manifest to
// store.
//
// Returns error which is always nil.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (e *InMemoryManifestEmitter) EmitCode(
	_ context.Context,
	manifest *generator_dto.Manifest,
	_ string,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.manifest = manifest
	return nil
}

// GetManifest returns the captured manifest.
//
// Returns *generator_dto.Manifest which is the stored manifest, or nil if none.
//
// Safe for concurrent use.
func (e *InMemoryManifestEmitter) GetManifest() *generator_dto.Manifest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.manifest
}
