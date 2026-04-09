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
	"fmt"
	"maps"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/seo/seo_dto"
)

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

// InMemoryPKJSEmitter implements PKJSEmitterPort using an in-memory map.
// It captures emitted JavaScript artefacts for later retrieval.
type InMemoryPKJSEmitter struct {
	// artefacts maps artefact IDs to their JavaScript content.
	artefacts map[string]string

	// mu protects concurrent access to artefacts.
	mu sync.RWMutex
}

// NewInMemoryPKJSEmitter creates a new in-memory PKJS emitter.
//
// Returns *InMemoryPKJSEmitter which captures emitted JavaScript.
func NewInMemoryPKJSEmitter() *InMemoryPKJSEmitter {
	return &InMemoryPKJSEmitter{
		artefacts: make(map[string]string),
	}
}

// EmitJS stores JavaScript content in memory and returns an
// artefact ID.
//
// Takes source (string) which is the JavaScript source code.
// Takes pagePath (string) which is the page path for the artefact
// ID.
//
// Returns string which is the generated artefact ID.
// Returns error which is always nil.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (e *InMemoryPKJSEmitter) EmitJS(
	_ context.Context,
	source string,
	pagePath string,
	_ string,
	_ string,
	_ bool,
) (artefactID string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	artefactID = "pk-js/" + pagePath + ".js"
	e.artefacts[artefactID] = source
	return artefactID, nil
}

// GetArtefacts returns all emitted JavaScript artefacts.
//
// Returns map[string]string which maps artefact IDs to JavaScript content.
//
// Safe for concurrent use. Returns a copy of the internal map.
func (e *InMemoryPKJSEmitter) GetArtefacts() map[string]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]string, len(e.artefacts))
	maps.Copy(result, e.artefacts)
	return result
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
