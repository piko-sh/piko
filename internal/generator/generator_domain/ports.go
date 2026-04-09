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

package generator_domain

import (
	"context"
	"os"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

// GeneratorService is the primary driving port for the entire compilation
// process. It orchestrates the annotation and code generation stages.
type GeneratorService interface {
	// Generate orchestrates the full compilation pipeline for a single component.
	//
	// It runs the annotator to build the dependency graph, then calls the code
	// emitter. This is the primary method for development servers, single-file
	// builds, and testing.
	//
	// Actions are auto-discovered from the actions/ directory during annotation.
	//
	// Takes request (GenerateRequest) which details the source file to compile.
	//
	// Returns *GeneratedArtefact which contains the final Go code and metadata.
	// Returns error when compilation fails.
	Generate(ctx context.Context, request generator_dto.GenerateRequest) (*generator_dto.GeneratedArtefact, error)

	// GenerateProject orchestrates the full compilation pipeline for a project.
	//
	// It takes the annotator result, generates all page artefacts in parallel,
	// and produces a final manifest file. This is the primary method for
	// production builds. Actions are auto-discovered from the actions/ directory
	// during annotation.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which contains the annotated
	// source files to compile.
	//
	// Returns []*generator_dto.GeneratedArtefact which contains all generated
	// page artefacts.
	// Returns *generator_dto.Manifest which describes the build output.
	// Returns error when compilation or manifest generation fails.
	GenerateProject(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
	) ([]*generator_dto.GeneratedArtefact, *generator_dto.Manifest, error)

	// AnnotateProject runs only the annotation phase of the compilation
	// pipeline, returning the project analysis result without generating any
	// code. This is used by assets-only builds where the FinalAssetManifest
	// (containing template-derived image sizes, densities, and formats) is
	// needed, but Go code emission and formatting are not.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which contains the
	// source files to analyse.
	//
	// Returns *annotator_dto.ProjectAnnotationResult which contains the
	// FinalAssetManifest and all component metadata.
	// Returns error when annotation fails.
	AnnotateProject(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
	) (*annotator_dto.ProjectAnnotationResult, error)

	// EmitProject runs the post-annotation emission pipeline: static
	// collections, search indexes, i18n, actions, parallel code generation,
	// manifest building, and SEO artefacts. It takes a pre-computed
	// annotation result so the caller can run annotation once and fan out
	// emission and asset building in parallel.
	//
	// Takes projectResult (*annotator_dto.ProjectAnnotationResult) which
	// contains the annotated project data from AnnotateProject.
	//
	// Returns []*generator_dto.GeneratedArtefact which contains all
	// generated Go source files.
	// Returns *generator_dto.Manifest which describes the project's pages,
	// partials, and routes.
	// Returns error when any emission step fails.
	EmitProject(
		ctx context.Context,
		projectResult *annotator_dto.ProjectAnnotationResult,
	) ([]*generator_dto.GeneratedArtefact, *generator_dto.Manifest, error)

	// Resolver returns the resolver port for cross-reference resolution.
	//
	// Returns resolver_domain.ResolverPort which provides symbol lookup.
	Resolver() resolver_domain.ResolverPort
}

// CodeEmitterFactoryPort defines a factory that creates CodeEmitterPort
// instances, letting the domain service create new emitters for each task
// without being tied to a specific implementation.
type CodeEmitterFactoryPort interface {
	// NewEmitter creates a new code emitter for output generation.
	//
	// Returns CodeEmitterPort which handles code output formatting.
	NewEmitter() CodeEmitterPort
}

// CodeEmitterPort defines the contract for turning a fully annotated AST into
// its final Go source code representation. It is a simple translator that
// relies on the metadata provided by the annotator's AnnotationResult.
type CodeEmitterPort interface {
	// EmitCode generates output code from the given annotation result.
	//
	// Takes annotationResult (*annotator_dto.AnnotationResult) which contains the
	// parsed annotations to process.
	// Takes request (generator_dto.GenerateRequest) which specifies generation
	// options.
	//
	// Returns []byte which contains the generated code.
	// Returns []*ast_domain.Diagnostic which contains any warnings or issues
	// found.
	// Returns error when code generation fails.
	EmitCode(
		ctx context.Context,
		annotationResult *annotator_dto.AnnotationResult,
		request generator_dto.GenerateRequest,
	) ([]byte, []*ast_domain.Diagnostic, error)
}

// ManifestEmitterPort defines the contract for generating the final manifest
// file from the metadata of all compiled pages.
type ManifestEmitterPort interface {
	// EmitCode generates source code from the manifest and writes it to the output
	// path.
	//
	// Takes manifest (*generator_dto.Manifest) which contains the parsed
	// documentation data.
	// Takes outputPath (string) which specifies where to write the generated code.
	//
	// Returns error when code generation or file writing fails.
	EmitCode(ctx context.Context, manifest *generator_dto.Manifest, outputPath string) error
}

// FSReaderPort defines the contract for reading files from the filesystem.
// It provides an abstraction layer for file I/O operations during generation.
type FSReaderPort interface {
	// ReadFile reads the contents of a file at the given path.
	//
	// Takes filePath (string) which is the path to the file to read.
	//
	// Returns []byte which contains the file contents.
	// Returns error when the file cannot be read.
	ReadFile(ctx context.Context, filePath string) ([]byte, error)
}

// FSWriterPort defines a way to write files to the filesystem.
// It provides a simple interface for file output during generation.
type FSWriterPort interface {
	// WriteFile writes data to the file at the given path.
	//
	// Takes filePath (string) which specifies the destination file path.
	// Takes data ([]byte) which contains the content to write.
	//
	// Returns error when the write operation fails.
	WriteFile(ctx context.Context, filePath string, data []byte) error

	// ReadDir reads the directory named by dirname and returns a list of
	// directory entries sorted by filename.
	//
	// Takes dirname (string) which specifies the directory path to read.
	//
	// Returns []os.DirEntry which contains the directory entries.
	// Returns error when the directory cannot be read.
	ReadDir(dirname string) ([]os.DirEntry, error)

	// RemoveAll removes path and any children it contains.
	//
	// Takes path (string) which specifies the path to remove.
	//
	// Returns error when the removal fails.
	RemoveAll(path string) error
}

// ManifestProviderPort defines the contract for loading a project manifest. It
// abstracts the source of manifest data, enabling different storage backends.
type ManifestProviderPort interface {
	// Load retrieves the manifest data from the underlying source.
	//
	// Returns *generator_dto.Manifest which contains the loaded manifest data.
	// Returns error when the manifest cannot be loaded.
	Load(ctx context.Context) (*generator_dto.Manifest, error)
}

// RegisterEmitterPort defines the contract for generating the component
// registration file. This file aggregates all generated components for runtime
// discovery.
type RegisterEmitterPort interface {
	// Emit writes the collected check results to the specified output path.
	//
	// Takes outputPath (string) which is the file path to write results to.
	// Takes allPackagePaths ([]string) which contains the packages that were
	// checked.
	//
	// Returns error when writing the output fails.
	Emit(ctx context.Context, outputPath string, allPackagePaths []string) error

	// Generate creates documentation output for the given package paths.
	//
	// Takes allPackagePaths ([]string) which lists the packages to document.
	//
	// Returns []byte which contains the generated documentation.
	// Returns error when generation fails.
	Generate(ctx context.Context, allPackagePaths []string) ([]byte, error)
}

// SEOServicePort defines the contract for generating SEO artefacts such as
// sitemap.xml and robots.txt from a project view. This is an optional
// dependency - if nil, SEO artefact generation is skipped.
type SEOServicePort interface {
	// GenerateArtefacts creates SEO artefacts for the given project view.
	//
	// Takes view (*seo_dto.ProjectView) which contains the project data to
	// process.
	//
	// Returns error when artefact generation fails.
	GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error
}

// CollectionEmitterPort defines the contract for generating static collection
// binary artefacts. It is implemented by the collection emitter service.
//
// This port is responsible for emitting:
//  1. Binary data files (data.bin) containing encoded collection items
//  2. Go wrapper files (generated.go) with //go:embed directives and init()
//     registration
//
// The emitted artefacts are placed in dist/collections/{collectionName}/ and
// are designed to be embedded directly into the compiled binary for zero-copy
// runtime access.
//
// Architecture:
//   - Called during GenerateProject after annotation but before page
//     generation
//   - Uses CollectionEncoderPort from collection hexagon to encode data
//   - Generates minimal Go code (just embedding + registration, no business
//     logic)
//   - Output location: dist/collections/{collectionName}/
type CollectionEmitterPort interface {
	// EmitCollection creates the binary and Go wrapper files for a static
	// collection.
	//
	// This method:
	//   1. Encodes all items into a FlatBuffer binary using
	//      CollectionSerialiserPort.
	//   2. Writes the binary to dist/collections/{collectionName}/data.bin.
	//   3. Creates a Go file with //go:embed directive and init() registration.
	//   4. Writes the Go file to dist/collections/{collectionName}/generated.go.
	//
	// Takes collectionName (string) which is the name of the collection (for
	// example "docs" or "blog").
	// Takes items ([]collection_dto.ContentItem) which is the slice of items to
	// encode.
	// Takes outputDir (string) which is the base output folder (for example
	// "dist").
	//
	// Returns packagePath (string) which is the Go package path for the created
	// collection package.
	// Returns err (error) when encoding or file writing fails.
	EmitCollection(
		ctx context.Context,
		collectionName string,
		items []collection_dto.ContentItem,
		outputDir string,
	) (packagePath string, err error)
}

// SearchIndexEmitterPort defines the contract for generating search index
// binary artefacts. It implements generator_domain.SearchIndexEmitterPort.
//
// This port emits search indexes for static collections:
//   - Fast mode index (search_fast.bin) - basic tokenisation + BM25
//   - Smart mode index (search_smart.bin) - full NLP pipeline with stemming
//
// The emitted artefacts are placed in dist/collections/{collectionName}/
// alongside the collection data and are designed to be embedded for
// zero-copy runtime access.
type SearchIndexEmitterPort interface {
	// EmitSearchIndex generates search index binaries for a collection.
	//
	// This method builds inverted indexes from collection items, writes Fast and
	// Smart mode index files to dist/collections/{collectionName}/, and updates
	// the collection's generated.go to embed and register the indexes.
	//
	// Takes collectionName (string) which identifies the collection to index.
	// Takes items ([]ContentItem) which contains the content to index.
	// Takes outputDir (string) which specifies the base output directory.
	// Takes modes ([]string) which specifies search modes ("fast", "smart", or
	// both).
	//
	// Returns error when index building or file writing fails.
	EmitSearchIndex(
		ctx context.Context,
		collectionName string,
		items []collection_dto.ContentItem,
		outputDir string,
		modes []string,
	) error
}

// PKJSEmitterPort defines the contract for emitting client-side JavaScript
// from PK files.
//
// This port is responsible for:
//  1. Transpiling TypeScript to JavaScript (stripping type annotations)
//  2. Storing the output in the registry with appropriate profiles
//
// The emitted JavaScript is stored in the registry and served via
// /_piko/assets/{artefactID}. The registry's capabilities pipeline handles
// minification (PriorityNeed) and compression.
//
// Architecture:
//   - Called during GenerateProject for each PK file with a client script
//   - Uses JSTranspiler to convert TypeScript to JavaScript
//   - Stores in registry with pk-js/ prefix for profile matching
//   - Orchestrator processes profiles: minified (NEED), gzip (WANT), br (WANT)
type PKJSEmitterPort interface {
	// EmitJS transpiles and stores JavaScript for a PK page in the registry.
	//
	// Takes source (string) which is the TypeScript/JavaScript source code from
	// the PK <script> block.
	// Takes pagePath (string) which is the relative path of the page (e.g.
	// "pages/checkout").
	// Takes moduleName (string) which is the Go module name for @/ alias
	// resolution in imports.
	// Takes outputDir (string) which is ignored (registry handles storage).
	// Takes minify (bool) which is ignored (capabilities pipeline handles
	// minification).
	//
	// Returns artefactID (string) which is the registry key (e.g.
	// "pk-js/pages/checkout.js") for storage in the manifest.
	// Returns error when transpilation or registry storage fails.
	EmitJS(
		ctx context.Context,
		source string,
		pagePath string,
		moduleName string,
		outputDir string,
		minify bool,
	) (artefactID string, err error)
}

// I18nEmitterPort defines the contract for emitting i18n FlatBuffer binaries.
//
// This port emits a binary FlatBuffer file containing all global translations.
// The binary is designed for zero-copy loading at runtime.
//
// Architecture:
//   - Called during GenerateProject after annotation
//   - Reads translations from the i18n directory configured in the project
//   - Encodes translations to dist/i18n.bin using FlatBuffer schema
//   - The runtime loads this file for Store-based translation lookups
type I18nEmitterPort interface {
	// EmitI18n encodes translations to a FlatBuffer binary file.
	//
	// Takes outputPath (string) which is the full path for the output file
	// (e.g. "dist/i18n.bin").
	//
	// Returns error when loading translations or file writing fails.
	EmitI18n(ctx context.Context, outputPath string) error
}

// ActionGeneratorPort defines the contract for generating action code artefacts
// from an ActionManifest.
//
// This port is responsible for generating:
//   - Registry code (dist/actions/registry.go) - maps action names to handlers
//   - Wrapper code (dist/actions/wrappers.go) - type-safe wrapper functions
//   - TypeScript types (dist/ts/actions.gen.ts) - client-side type definitions
//
// Architecture:
//   - Called during GenerateProject after annotation phase completes
//   - Receives ActionManifest from VirtualModule (auto-discovered actions)
//   - Uses emitter adapters from generator_adapters and typegen_adapters
type ActionGeneratorPort interface {
	// GenerateActions generates all action code artefacts from the manifest.
	//
	// Takes manifest (*annotator_dto.ActionManifest) which contains the
	// discovered actions from the actions/ directory.
	// Takes moduleName (string) which is the Go module name for imports.
	// Takes outputDir (string) which is the project root directory for output.
	//
	// Returns error when code generation or file writing fails.
	GenerateActions(
		ctx context.Context,
		manifest *annotator_dto.ActionManifest,
		moduleName string,
		outputDir string,
	) error
}

// StaticPrerenderer renders fully-static AST nodes to HTML bytes at generation
// time. This enables precomputing HTML for static subtrees, avoiding recursive
// AST walking at render time.
//
// The renderer implements this interface. The generator uses it during code
// emission to prerender nodes marked with IsFullyPrerenderable=true.
//
// Architecture:
//   - Interface defined here (generator owns) to maintain clean dependency
//     direction
//   - Renderer implements this with minimal context (no registry, no CSRF)
//   - Generator calls this for nodes where entire subtree has no piko:* tags
type StaticPrerenderer interface {
	// RenderStaticNode renders a static node subtree to HTML bytes.
	//
	// The node must have IsFullyPrerenderable=true, meaning its entire subtree
	// contains no piko:svg, piko:img, piko:a, or piko:video tags that require
	// runtime processing.
	//
	// Takes node (*ast_domain.TemplateNode) which is the root of the static
	// subtree to render.
	//
	// Returns []byte which contains the rendered HTML.
	// Returns error if rendering fails (should not happen for valid static nodes).
	RenderStaticNode(node *ast_domain.TemplateNode) ([]byte, error)
}
