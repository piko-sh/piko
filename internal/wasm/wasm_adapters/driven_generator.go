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
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/generator/generator_adapters/driven_code_emitter_go_literal"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

const (
	// modulePathComponents is the number of path parts in a standard Go module
	// path (domain/org/repo).
	modulePathComponents = 3

	// moduleSplitParts is the number of parts to split an import path into when
	// extracting the module path and subpath.
	moduleSplitParts = 4

	// defaultMaxGenerateSourceFiles caps how many source files Generate /
	// DynamicRender will accept in a single call. Mirrors the convention
	// used elsewhere in the framework: bound the user-controlled input
	// before walking it.
	defaultMaxGenerateSourceFiles = 4096

	// defaultMaxGenerateSourceBytes caps the aggregated payload size of
	// every source file in a single Generate / DynamicRender. Defaults to
	// 8 MiB; raise via WithGeneratorLimits if a project legitimately needs
	// more.
	defaultMaxGenerateSourceBytes = 8 * 1024 * 1024

	// defaultMaxGenerateFileBytes caps an individual source file's size
	// before it is fed to the SFC compiler / annotator. Per-file gate so a
	// pathological 100 MiB single-line file cannot drive the lexer's
	// PositionAt into O(n) per token.
	defaultMaxGenerateFileBytes = 1 * 1024 * 1024
)

const (
	// errorPositionLineGroup is the submatch index for the line component
	// of errorPositionPattern.
	errorPositionLineGroup = 2

	// errorPositionColumnGroup is the submatch index for the column
	// component of errorPositionPattern.
	errorPositionColumnGroup = 3

	// errorPositionMinMatches is the minimum number of submatches for
	// errorPositionPattern to yield a usable line/column pair.
	errorPositionMinMatches = 4
)

var (
	// errGenerateLimitsExceeded is returned by validateGenerateRequest
	// when caller-supplied sources exceed the adapter's configured
	// limits.
	errGenerateLimitsExceeded = errors.New("generate request exceeds configured limits")

	// errorPositionPattern recovers a line/column pair from rendered Go-
	// scanner-style "file:line:col: message" prefixes, used as a best-
	// effort fallback for SFC error positions that aren't yet exposed via
	// a typed error.
	errorPositionPattern = regexp.MustCompile(`^([^:]+):(\d+):(\d+)?:?\s*(.*)$`)
)

var _ resolver_domain.ResolverPort = (*inMemoryResolver)(nil)

// parsePositiveInt returns the integer value of s, or an error when s is
// non-positive or unparseable. Used to validate line/column captures.
//
// Takes s (string) which is the captured digit run.
//
// Returns int which is the parsed positive integer.
// Returns error when s is not a positive base-10 integer.
func parsePositiveInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty integer")
	}
	value, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("non-positive integer %d", value)
	}
	return value, nil
}

// generatorLimits caps caller-supplied input on every Generate so the WASM
// heap cannot blow up under untrusted source maps. Defaults are sourced
// from the const block above; override per adapter via WithGeneratorLimits.
type generatorLimits struct {
	// MaxFileCount is the maximum number of entries allowed in
	// request.Sources.
	MaxFileCount int

	// MaxTotalBytes is the maximum aggregate length (in bytes) of all
	// source files in request.Sources combined.
	MaxTotalBytes int

	// MaxFileBytes is the maximum length (in bytes) of any single source
	// file.
	MaxFileBytes int
}

// GeneratorAdapter implements GeneratorPort using the real generator service
// with in-memory adapters for file system operations.
//
// This adapter is designed for WASM and testing contexts where actual file
// system access is not available or desired.
//
// Concurrent Generate calls on the same adapter are serialised by an
// internal mutex so the shared pkJSEmitter cannot observe a half-mutated
// per-run state from a parallel run.
type GeneratorAdapter struct {
	// stdlibDataGetter retrieves the pre-bundled standard library type
	// information. This is a function because the data may not be available
	// at construction time (it's loaded during orchestrator initialisation).
	stdlibDataGetter func() (*inspector_dto.TypeData, error)

	// pkJSEmitter captures compiled client-side JavaScript across
	// Generate calls.
	//
	// Lifted to adapter scope so the content-hash cache amortises
	// transpile work between keystroke-rate renders. Reset at the start
	// of each Generate to ensure responses contain only the current
	// run's artefacts.
	pkJSEmitter *InMemoryPKJSEmitter

	// pathsConfig holds the resolved path settings.
	pathsConfig generator_domain.GeneratorPathsConfig

	// i18nDefaultLocale is the default locale.
	i18nDefaultLocale string

	// moduleName is the Go module name used for generated code.
	moduleName string

	// limits caps caller-supplied source map size; see generatorLimits.
	limits generatorLimits

	// generateMu serialises Generate calls so concurrent invocations on
	// the same adapter cannot race on the shared pkJSEmitter's per-run
	// state (artefacts, producedThisRun).
	generateMu sync.Mutex

	// hasConfig indicates whether explicit configuration was provided.
	hasConfig bool
}

var _ wasm_domain.GeneratorPort = (*GeneratorAdapter)(nil)

// GeneratorAdapterOption configures a GeneratorAdapter instance.
type GeneratorAdapterOption func(*GeneratorAdapter)

// NewGeneratorAdapter creates a new generator adapter with the given options.
//
// Takes opts (...GeneratorAdapterOption) which configure the adapter.
//
// Returns *GeneratorAdapter which is ready to generate code.
func NewGeneratorAdapter(opts ...GeneratorAdapterOption) *GeneratorAdapter {
	a := &GeneratorAdapter{
		moduleName:  "playground",
		pkJSEmitter: NewInMemoryPKJSEmitter(),
		limits: generatorLimits{
			MaxFileCount:  defaultMaxGenerateSourceFiles,
			MaxTotalBytes: defaultMaxGenerateSourceBytes,
			MaxFileBytes:  defaultMaxGenerateFileBytes,
		},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Generate produces code artefacts from in-memory sources.
//
// Takes request (*wasm_dto.GenerateFromSourcesRequest) which contains the source
// files and configuration for code generation.
//
// Returns *wasm_dto.GenerateFromSourcesResponse which contains the generated
// artefacts and manifest on success, or error details on failure.
// Returns error when an unexpected error occurs during generation.
//
// Concurrency: Concurrent Generate calls on the same adapter are
// serialised by an internal mutex so the shared pkJSEmitter cannot
// observe half-mutated per-run state from a parallel run.
func (a *GeneratorAdapter) Generate(
	ctx context.Context,
	request *wasm_dto.GenerateFromSourcesRequest,
) (*wasm_dto.GenerateFromSourcesResponse, error) {
	if request == nil {
		return a.errorResponse("nil generate request"), nil
	}
	if errResp := validateGenerateRequest(request, a.limits); errResp != nil {
		return errResp, nil
	}

	a.generateMu.Lock()
	defer a.generateMu.Unlock()

	stdlibData, errResp := a.validateAndGetStdlib()
	if errResp != nil {
		return errResp, nil
	}

	moduleName := request.ModuleName
	if moduleName == "" {
		moduleName = a.moduleName
	}

	annotator, errResp := a.createAnnotator(request.Sources, moduleName, stdlibData)
	if errResp != nil {
		return errResp, nil
	}

	entryPoints := a.discoverEntryPoints(request.Sources, moduleName)
	if len(entryPoints) == 0 {
		return a.errorResponse("no .pk files found in sources"), nil
	}

	a.pkJSEmitter.Reset()

	fsWriter, generatorService, errResp := a.createGeneratorService(ctx, request, moduleName, annotator)
	if errResp != nil {
		return errResp, nil
	}

	artefacts, manifest, err := generatorService.GenerateProject(ctx, entryPoints)
	if err != nil {
		return a.errorResponse(fmt.Sprintf("generation failed: %v", err)), nil
	}

	componentDiagnostics, err := a.compileClientComponents(ctx, request.Sources, moduleName)
	if err != nil {
		return a.errorResponse(fmt.Sprintf("compiling client components cancelled: %v", err)), nil
	}

	response := &wasm_dto.GenerateFromSourcesResponse{
		Success:     true,
		Artefacts:   a.convertArtefacts(artefacts, fsWriter, a.pkJSEmitter),
		Manifest:    a.convertManifest(manifest),
		Diagnostics: componentDiagnostics,
	}

	a.pkJSEmitter.Sweep()

	return response, nil
}

// validateGenerateRequest enforces caller-supplied limits.
//
// Mirrors validateAnalyseRequest's gate so expensive work never runs on
// an oversize request. Returns nil when the request is acceptable, or a
// failed response carrying a developer-facing message when a limit is
// exceeded.
//
// Takes request (*wasm_dto.GenerateFromSourcesRequest) which is the request
// to validate.
// Takes limits (generatorLimits) which configures the caps.
//
// Returns *wasm_dto.GenerateFromSourcesResponse which is non-nil when a
// limit is exceeded.
func validateGenerateRequest(request *wasm_dto.GenerateFromSourcesRequest, limits generatorLimits) *wasm_dto.GenerateFromSourcesResponse {
	fileCount := len(request.Sources)
	if fileCount == 0 {
		return nil
	}
	if limits.MaxFileCount > 0 && fileCount > limits.MaxFileCount {
		return &wasm_dto.GenerateFromSourcesResponse{
			Success: false,
			Error: fmt.Sprintf("%v: %d source files exceeds limit %d",
				errGenerateLimitsExceeded, fileCount, limits.MaxFileCount),
		}
	}
	totalBytes := 0
	for filePath, content := range request.Sources {
		size := len(content)
		if limits.MaxFileBytes > 0 && size > limits.MaxFileBytes {
			return &wasm_dto.GenerateFromSourcesResponse{
				Success: false,
				Error: fmt.Sprintf("%v: source %s size %d exceeds per-file limit %d",
					errGenerateLimitsExceeded, filePath, size, limits.MaxFileBytes),
			}
		}
		totalBytes += size
		if limits.MaxTotalBytes > 0 && totalBytes > limits.MaxTotalBytes {
			return &wasm_dto.GenerateFromSourcesResponse{
				Success: false,
				Error: fmt.Sprintf("%v: aggregate source size exceeds limit %d",
					errGenerateLimitsExceeded, limits.MaxTotalBytes),
			}
		}
	}
	return nil
}

// validateAndGetStdlib validates the adapter and returns stdlib data.
//
// Returns *inspector_dto.TypeData which contains the standard library type
// information.
// Returns *wasm_dto.GenerateFromSourcesResponse which contains an error
// response when validation fails or stdlib data cannot be retrieved, or nil
// on success.
func (a *GeneratorAdapter) validateAndGetStdlib() (*inspector_dto.TypeData, *wasm_dto.GenerateFromSourcesResponse) {
	if a.stdlibDataGetter == nil {
		return nil, a.errorResponse("generator adapter not configured: stdlib data getter is nil")
	}
	stdlibData, err := a.stdlibDataGetter()
	if err != nil {
		return nil, a.errorResponse(fmt.Sprintf("failed to get stdlib data: %v", err))
	}
	return stdlibData, nil
}

// createAnnotator creates the in-memory annotator service.
//
// Takes sources (map[string]string) which provides the source files to parse.
// Takes moduleName (string) which specifies the Go module name.
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information.
//
// Returns annotator_domain.AnnotatorPort which is the configured annotator.
// Returns *wasm_dto.GenerateFromSourcesResponse which contains the error
// response when creation fails, or nil on success.
func (a *GeneratorAdapter) createAnnotator(sources map[string]string, moduleName string, stdlibData *inspector_dto.TypeData) (annotator_domain.AnnotatorPort, *wasm_dto.GenerateFromSourcesResponse) {
	annotator, err := NewInMemoryAnnotatorService(sources, moduleName, stdlibData)
	if err != nil {
		return nil, a.errorResponse(fmt.Sprintf("failed to create annotator service: %v", err))
	}
	return annotator, nil
}

// createGeneratorService creates the generator service with in-memory ports.
//
// Takes ctx (context.Context) which provides the base context for the service.
// Takes request (*wasm_dto.GenerateFromSourcesRequest) which contains the source
// files and configuration for generation.
// Takes moduleName (string) which specifies the Go module name for resolution.
// Takes annotator (annotator_domain.AnnotatorPort) which provides annotation
// services for the coordinator.
//
// Returns *InMemoryFSWriter which captures generated file output.
// Returns generator_domain.GeneratorService which is the configured service.
// Returns *wasm_dto.GenerateFromSourcesResponse which contains any error
// response, or nil on success.
func (a *GeneratorAdapter) createGeneratorService(
	ctx context.Context,
	request *wasm_dto.GenerateFromSourcesRequest,
	moduleName string,
	annotator annotator_domain.AnnotatorPort,
) (*InMemoryFSWriter, generator_domain.GeneratorService, *wasm_dto.GenerateFromSourcesResponse) {
	fsWriter := NewInMemoryFSWriter()
	coordinator := NewInMemoryCoordinator(annotator)

	baseDir := request.BaseDir
	if baseDir == "" && a.hasConfig {
		baseDir = a.pathsConfig.BaseDir
	}

	ports := generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		ManifestEmitter:    NewInMemoryManifestEmitter(),
		Coordinator:        coordinator,
		Resolver:           newInMemoryResolver(moduleName, baseDir),
		RegisterEmitter:    NewInMemoryRegisterEmitter(fsWriter),
		CodeEmitterFactory: driven_code_emitter_go_literal.NewEmitterFactory(ctx, nil),
		CollectionEmitter:  NewNoOpCollectionEmitter(),
		SearchIndexEmitter: NewNoOpSearchIndexEmitter(),
		PKJSEmitter:        a.pkJSEmitter,
		I18nEmitter:        NewNoOpI18nEmitter(),
		ActionGenerator:    NewNoOpActionGenerator(),
		SEOService:         nil,
	}

	pathsConfig := generator_domain.GeneratorPathsConfig{BaseDir: "."}
	i18nLocale := ""
	if a.hasConfig {
		pathsConfig = a.pathsConfig
		i18nLocale = a.i18nDefaultLocale
	}

	generatorService, err := generator_domain.NewGeneratorService(ctx, pathsConfig, i18nLocale, ports, generator_domain.WithInMemoryMode())
	if err != nil {
		return nil, nil, a.errorResponse(fmt.Sprintf("failed to create generator service: %v", err))
	}

	return fsWriter, generatorService, nil
}

// errorResponse creates a failed response with the given error message.
//
// Takes message (string) which provides the error message to include.
//
// Returns *wasm_dto.GenerateFromSourcesResponse which contains the failure
// status and error message.
func (*GeneratorAdapter) errorResponse(message string) *wasm_dto.GenerateFromSourcesResponse {
	return &wasm_dto.GenerateFromSourcesResponse{Success: false, Error: message}
}

// discoverEntryPoints finds .pk files in the sources map and returns them
// as entry points. The paths are prefixed with the module name to create
// fully-qualified import paths.
//
// .pkc client-side components are NOT entry points; they're compiled
// separately by compileClientComponents because the annotator's
// page/partial-style type checking would mistakenly demand a Render
// function or a Response DTO that .pkc components don't have.
//
// Takes sources (map[string]string) which contains the source files to scan.
// Takes moduleName (string) which is the prefix for fully-qualified paths.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry
// points with their paths and page status.
func (*GeneratorAdapter) discoverEntryPoints(sources map[string]string, moduleName string) []annotator_dto.EntryPoint {
	entryPoints := make([]annotator_dto.EntryPoint, 0, len(sources))

	for sourcePath := range sources {
		if !strings.HasSuffix(sourcePath, ".pk") {
			continue
		}

		isPage := strings.Contains(sourcePath, "pages/") || strings.HasPrefix(sourcePath, "pages/")

		fullPath := moduleName + "/" + sourcePath

		entryPoints = append(entryPoints, annotator_dto.EntryPoint{
			Path:   fullPath,
			IsPage: isPage,
		})
	}

	return entryPoints
}

// compileClientComponents walks sources for .pkc client-side components
// and runs each one through the SFC compiler. This produces a class
// extending PPElement with a customElements.define call so the browser
// can upgrade <pp-foo> elements in the page.
//
// .pkc is deliberately not routed through pkJSEmitter.EmitJS. That path
// applies the partial-style TransformPKSource (factory + _createPKContext
// wrapper) intended for inline <script> blocks inside .pk pages and
// partials, not for client-side Web Components. The disk pipeline keeps
// the two paths separate; the WASM adapter mirrors that.
//
// Per-component errors land in the returned Diagnostic slice; a
// well-formed component that produces no JS surfaces a warning so a
// silently-broken playground preview is impossible.
//
// Takes ctx (context.Context) which propagates through compilation; ctx
// cancellation is checked between iterations so a long-running run can be
// cancelled by the WASM Promise's deadline.
// Takes sources (map[string]string) which contains every source file the
// playground sent.
// Takes moduleName (string) which the SFC compiler uses to resolve "@/"
// alias paths in component imports.
//
// Returns []wasm_dto.Diagnostic with one entry per failed or empty
// component compile.
// Returns error only when ctx is cancelled mid-run (the partial diagnostic
// list is still returned alongside).
func (a *GeneratorAdapter) compileClientComponents(
	ctx context.Context,
	sources map[string]string,
	moduleName string,
) ([]wasm_dto.Diagnostic, error) {
	var diagnostics []wasm_dto.Diagnostic
	sfcCompiler := compiler_domain.NewSFCCompiler(moduleName, nil)

	for sourcePath, content := range sources {
		if err := ctx.Err(); err != nil {
			return diagnostics, fmt.Errorf("compileClientComponents cancelled: %w", err)
		}
		if !strings.HasSuffix(sourcePath, ".pkc") {
			continue
		}
		diagnostics = append(diagnostics, a.compileSingleClientComponent(ctx, sfcCompiler, sourcePath, content)...)
	}
	return diagnostics, nil
}

// compileSingleClientComponent runs one .pkc through the SFC compiler and
// stores its output in the emitter. Returns the slice of diagnostics that
// should be appended to the caller's collection (empty on a clean success).
//
// Takes ctx (context.Context) which threads through CompileSFC.
// Takes sfcCompiler which compiles the SFC.
// Takes sourcePath (string) which is the user-supplied source key.
// Takes content (string) which is the user-supplied raw .pkc text.
//
// Returns []wasm_dto.Diagnostic which is the per-component diagnostic
// stream (compiler-internal warnings plus this layer's own findings).
func (a *GeneratorAdapter) compileSingleClientComponent(
	ctx context.Context,
	sfcCompiler compiler_domain.SFCCompiler,
	sourcePath, content string,
) []wasm_dto.Diagnostic {
	artefact, err := sfcCompiler.CompileSFC(ctx, sourcePath, []byte(content))
	if err != nil {
		return []wasm_dto.Diagnostic{{
			Severity: "error",
			Message:  fmt.Sprintf("compiling component %s: %v", sourcePath, err),
			Location: locationFromError(err, sourcePath),
		}}
	}
	if artefact == nil || artefact.BaseJSPath == "" {
		return []wasm_dto.Diagnostic{{
			Severity: "warning",
			Message:  fmt.Sprintf("component %s produced no JavaScript artefact", sourcePath),
			Location: wasm_dto.Location{FilePath: sourcePath},
		}}
	}
	outputJS, ok := artefact.Files[artefact.BaseJSPath]
	if !ok || outputJS == "" {
		return []wasm_dto.Diagnostic{{
			Severity: "warning",
			Message:  fmt.Sprintf("component %s compiled but emitted empty JS body", sourcePath),
			Location: wasm_dto.Location{FilePath: sourcePath},
		}}
	}

	emitterPath := strings.TrimSuffix(sourcePath, ".pkc")
	artefactID := path.Join("pk-js", emitterPath) + ".js"
	if err := a.pkJSEmitter.Put(artefactID, outputJS); err != nil {
		return []wasm_dto.Diagnostic{{
			Severity: "error",
			Message:  fmt.Sprintf("rejecting component artefact id for %s: %v", sourcePath, err),
			Location: wasm_dto.Location{FilePath: sourcePath},
		}}
	}

	diagnostics := make([]wasm_dto.Diagnostic, 0, len(artefact.Diagnostics))
	for _, compilerDiagnostic := range artefact.Diagnostics {
		diagnostics = append(diagnostics, wasm_dto.Diagnostic{
			Severity: compilerDiagnostic.Severity,
			Message:  compilerDiagnostic.Message,
			Location: wasm_dto.Location{FilePath: sourcePath},
		})
	}
	return diagnostics
}

// locationFromError walks the error chain looking for a structured
// sfcparser location or a Go scanner-style "file:line:col" prefix. Falls
// back to (path, 0, 0) when no location can be recovered.
//
// Takes err (error) which is the error returned from CompileSFC.
// Takes sourcePath (string) which is used as the FilePath of the location.
//
// Returns wasm_dto.Location with whatever Line/Column could be parsed.
func locationFromError(err error, sourcePath string) wasm_dto.Location {
	location := wasm_dto.Location{FilePath: sourcePath}
	if err == nil {
		return location
	}

	message := err.Error()
	if matches := errorPositionPattern.FindStringSubmatch(message); len(matches) >= errorPositionMinMatches {
		if line, parseErr := parsePositiveInt(matches[errorPositionLineGroup]); parseErr == nil {
			location.Line = line
		}
		if column, parseErr := parsePositiveInt(matches[errorPositionColumnGroup]); parseErr == nil {
			location.Column = column
		}
	}
	return location
}

// convertArtefacts converts generator artefacts to WASM DTOs, merging output
// from three sources: the generator's direct artefact slice, the in-memory
// filesystem (for register/manifest emitters that write through it), and the
// PKJS emitter (which captures compiled client-side JavaScript directly).
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which holds the
// generated Go output from the generator.
// Takes fsWriter (*InMemoryFSWriter) which provides access to files written
// to the in-memory filesystem.
// Takes pkJSEmitter (*InMemoryPKJSEmitter) which captures transpiled
// client-side JavaScript artefacts.
//
// Returns []wasm_dto.GeneratedArtefact with deduplication by Path.
func (*GeneratorAdapter) convertArtefacts(
	artefacts []*generator_dto.GeneratedArtefact,
	fsWriter *InMemoryFSWriter,
	pkJSEmitter *InMemoryPKJSEmitter,
) []wasm_dto.GeneratedArtefact {
	totalCapacity := len(artefacts) + len(fsWriter.GetWrittenFiles())
	if pkJSEmitter != nil {
		totalCapacity += len(pkJSEmitter.GetArtefacts())
	}
	result := make([]wasm_dto.GeneratedArtefact, 0, totalCapacity)
	seen := make(map[string]struct{}, totalCapacity)

	for _, artefact := range artefacts {
		artefactType := wasm_dto.ArtefactTypePartial
		if artefact.Component != nil && artefact.Component.IsPage {
			artefactType = wasm_dto.ArtefactTypePage
		}

		result = append(result, wasm_dto.GeneratedArtefact{
			Path:    artefact.SuggestedPath,
			Content: string(artefact.Content),
			Type:    artefactType,
		})
		seen[artefact.SuggestedPath] = struct{}{}
	}

	for filePath, content := range fsWriter.GetWrittenFiles() {
		if _, found := seen[filePath]; found {
			continue
		}

		artefactType := determineArtefactType(filePath)
		result = append(result, wasm_dto.GeneratedArtefact{
			Path:    filePath,
			Content: string(content),
			Type:    artefactType,
		})
		seen[filePath] = struct{}{}
	}

	if pkJSEmitter != nil {
		for filePath, content := range pkJSEmitter.GetArtefacts() {
			if _, found := seen[filePath]; found {
				continue
			}
			result = append(result, wasm_dto.GeneratedArtefact{
				Path:    filePath,
				Content: content,
				Type:    wasm_dto.ArtefactTypeJS,
			})
			seen[filePath] = struct{}{}
		}
	}

	return result
}

// convertManifest converts a generator manifest to the WASM DTO format.
//
// Takes manifest (*generator_dto.Manifest) which is the source manifest to
// convert.
//
// Returns *wasm_dto.GeneratedManifest which contains the converted manifest
// data, or nil if the input is nil.
func (*GeneratorAdapter) convertManifest(manifest *generator_dto.Manifest) *wasm_dto.GeneratedManifest {
	if manifest == nil {
		return nil
	}

	result := &wasm_dto.GeneratedManifest{
		Pages:    make(map[string]wasm_dto.ManifestPageEntry),
		Partials: make(map[string]wasm_dto.ManifestPartialEntry),
	}

	for id := range manifest.Pages {
		page := manifest.Pages[id]
		result.Pages[id] = wasm_dto.ManifestPageEntry{
			PackagePath:   page.PackagePath,
			RoutePatterns: page.RoutePatterns,
			JSArtefactIDs: page.JSArtefactIDs,
			StyleBlock:    page.StyleBlock,
		}
	}

	for id, partial := range manifest.Partials {
		result.Partials[id] = wasm_dto.ManifestPartialEntry{
			PackagePath:  partial.PackagePath,
			SourcePath:   partial.OriginalSourcePath,
			JSArtefactID: partial.JSArtefactID,
			StyleBlock:   partial.StyleBlock,
		}
	}

	return result
}

// inMemoryResolver provides path resolution for in-memory file systems.
// It implements resolver_domain.ResolverPort.
type inMemoryResolver struct {
	// moduleName is the Go module path used to resolve import paths.
	moduleName string

	// baseDir is the base directory path; empty for in-memory resolution.
	baseDir string
}

// DetectLocalModule is a no-op for in-memory resolver.
//
// Returns error when detection fails; always returns nil for this resolver.
func (*inMemoryResolver) DetectLocalModule(_ context.Context) error {
	return nil
}

// GetModuleName returns the module name.
//
// Returns string which is the module name for this resolver.
func (r *inMemoryResolver) GetModuleName() string {
	return r.moduleName
}

// GetBaseDir returns the base directory (empty for in-memory).
//
// Returns string which is the base directory path, or empty if in-memory.
func (r *inMemoryResolver) GetBaseDir() string {
	return r.baseDir
}

// ResolvePKPath resolves a Piko component import path to an absolute path.
//
// Takes importPath (string) which is the import path to resolve.
//
// Returns string which is the resolved path with module prefix removed.
// Returns error which is always nil for this resolver.
func (r *inMemoryResolver) ResolvePKPath(_ context.Context, importPath string, _ string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(importPath))

	if trimmed, ok := strings.CutPrefix(clean, "@/"); ok {
		clean = trimmed
	}

	if trimmed, ok := strings.CutPrefix(clean, r.moduleName+"/"); ok {
		clean = trimmed
	}

	return clean, nil
}

// ResolveCSSPath resolves a CSS import path.
//
// Takes importPath (string) which is the path to resolve.
// Takes containingDir (string) which is the directory containing the import.
//
// Returns string which is the resolved path.
// Returns error when resolution fails.
func (*inMemoryResolver) ResolveCSSPath(_ context.Context, importPath string, containingDir string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(importPath))

	if trimmed, ok := strings.CutPrefix(clean, "@/"); ok {
		return trimmed, nil
	}

	if strings.HasPrefix(clean, "./") || strings.HasPrefix(clean, "../") {
		return filepath.ToSlash(filepath.Join(containingDir, clean)), nil
	}

	return clean, nil
}

// ResolveAssetPath resolves an asset path.
//
// Takes importPath (string) which is the asset import path to resolve.
//
// Returns string which is the resolved asset path.
// Returns error which is always nil for this resolver.
func (r *inMemoryResolver) ResolveAssetPath(_ context.Context, importPath string, _ string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(importPath))

	if trimmed, ok := strings.CutPrefix(clean, "@/"); ok {
		return trimmed, nil
	}

	if trimmed, ok := strings.CutPrefix(clean, r.moduleName+"/"); ok {
		return trimmed, nil
	}

	return clean, nil
}

// ConvertEntryPointPathToManifestKey converts a module-absolute path to a
// project-relative key.
//
// Takes entryPointPath (string) which is the module-absolute path to convert.
//
// Returns string which is the project-relative key with the module name prefix
// removed, or the original path if no prefix was present.
func (r *inMemoryResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	if trimmed, ok := strings.CutPrefix(entryPointPath, r.moduleName+"/"); ok {
		return trimmed
	}
	return entryPointPath
}

// GetModuleDir resolves a Go module path to its filesystem directory.
// For in-memory resolver, this returns an error as external modules are not
// available.
//
// Takes modulePath (string) which specifies the Go module path to resolve.
//
// Returns string which would be the directory path if resolution succeeded.
// Returns error when called, as external modules are not available.
func (*inMemoryResolver) GetModuleDir(_ context.Context, modulePath string) (string, error) {
	return "", fmt.Errorf("external module %q not available in in-memory resolver", modulePath)
}

// FindModuleBoundary splits an import path into module and subpath.
// For in-memory resolver, this assumes everything is in the local module.
//
// Takes importPath (string) which is the full import path to split.
//
// Returns modulePath (string) which is the module portion of the path.
// Returns subpath (string) which is the path within the module.
// Returns err (error) which is always nil for this resolver.
func (r *inMemoryResolver) FindModuleBoundary(_ context.Context, importPath string) (modulePath, subpath string, err error) {
	if sub, ok := strings.CutPrefix(importPath, r.moduleName+"/"); ok {
		return r.moduleName, sub, nil
	}

	parts := strings.SplitN(importPath, "/", moduleSplitParts)
	if len(parts) >= modulePathComponents {
		modPath := strings.Join(parts[:modulePathComponents], "/")
		sub := ""
		if len(parts) > modulePathComponents {
			sub = parts[modulePathComponents]
		}
		return modPath, sub, nil
	}

	return importPath, "", nil
}

// WithStdlibDataGetter sets a function to fetch the pre-bundled standard
// library type data. This is a function because the stdlib data may not be
// ready when the adapter is created.
//
// Takes getter (func() (*inspector_dto.TypeData, error)) which fetches
// the stdlib types when called.
//
// Returns GeneratorAdapterOption which configures the adapter.
func WithStdlibDataGetter(getter func() (*inspector_dto.TypeData, error)) GeneratorAdapterOption {
	return func(a *GeneratorAdapter) {
		a.stdlibDataGetter = getter
	}
}

// WithGeneratorConfig sets the generator configuration for the adapter.
//
// Takes pathsConfig (generator_domain.GeneratorPathsConfig) which provides path
// settings.
// Takes i18nDefaultLocale (string) which specifies the default locale.
//
// Returns GeneratorAdapterOption which configures the adapter with the given
// settings.
func WithGeneratorConfig(pathsConfig generator_domain.GeneratorPathsConfig, i18nDefaultLocale string) GeneratorAdapterOption {
	return func(a *GeneratorAdapter) {
		a.pathsConfig = pathsConfig
		a.i18nDefaultLocale = i18nDefaultLocale
		a.hasConfig = true
	}
}

// WithModuleName sets the Go module name for generated code.
//
// Takes moduleName (string) which is the module name to use.
//
// Returns GeneratorAdapterOption which configures the adapter.
func WithModuleName(moduleName string) GeneratorAdapterOption {
	return func(a *GeneratorAdapter) {
		a.moduleName = moduleName
	}
}

// WithGeneratorLimits overrides the default per-call caps.
//
// Bounds how many sources Generate / DynamicRender accept and how large
// each source can be. Pass zero on any field to disable that specific
// limit. Defaults are intentionally generous so they do not get in the
// way of real projects but still close the unbounded-input attack
// surface.
//
// Takes maxFileCount (int) which caps len(request.Sources).
// Takes maxTotalBytes (int) which caps the aggregate byte size of every
// source file.
// Takes maxFileBytes (int) which caps any single source file's byte size.
//
// Returns GeneratorAdapterOption which configures the adapter.
func WithGeneratorLimits(maxFileCount, maxTotalBytes, maxFileBytes int) GeneratorAdapterOption {
	return func(a *GeneratorAdapter) {
		a.limits = generatorLimits{
			MaxFileCount:  maxFileCount,
			MaxTotalBytes: maxTotalBytes,
			MaxFileBytes:  maxFileBytes,
		}
	}
}

// determineArtefactType guesses the artefact type from the file path.
//
// Takes filePath (string) which is the file path to analyse.
//
// Returns wasm_dto.ArtefactType which is the guessed type based on path
// patterns, defaulting to ArtefactTypePage if no pattern matches.
func determineArtefactType(filePath string) wasm_dto.ArtefactType {
	switch {
	case strings.Contains(filePath, "/pages/"):
		return wasm_dto.ArtefactTypePage
	case strings.Contains(filePath, "/partials/"):
		return wasm_dto.ArtefactTypePartial
	case strings.Contains(filePath, "/actions/"):
		return wasm_dto.ArtefactTypeAction
	case strings.HasSuffix(filePath, ".js"):
		return wasm_dto.ArtefactTypeJS
	case strings.Contains(filePath, "register"):
		return wasm_dto.ArtefactTypeRegister
	case strings.Contains(filePath, "manifest"):
		return wasm_dto.ArtefactTypeManifest
	default:
		return wasm_dto.ArtefactTypePage
	}
}

// newInMemoryResolver creates a resolver for in-memory generation.
//
// Takes moduleName (string) which specifies the Go module name.
// Takes baseDir (string) which specifies the base directory for resolution.
//
// Returns *inMemoryResolver which is the configured resolver ready for use.
func newInMemoryResolver(moduleName string, baseDir string) *inMemoryResolver {
	return &inMemoryResolver{
		moduleName: moduleName,
		baseDir:    baseDir,
	}
}
