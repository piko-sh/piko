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
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
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
)

var _ resolver_domain.ResolverPort = (*inMemoryResolver)(nil)

// GeneratorAdapter implements GeneratorPort using the real generator service
// with in-memory adapters for file system operations.
//
// This adapter is designed for WASM and testing contexts where actual file
// system access is not available or desired.
type GeneratorAdapter struct {
	// stdlibDataGetter retrieves the pre-bundled standard library type
	// information. This is a function because the data may not be available
	// at construction time (it's loaded during orchestrator initialisation).
	stdlibDataGetter func() (*inspector_dto.TypeData, error)

	// pathsConfig holds the resolved path settings.
	pathsConfig generator_domain.GeneratorPathsConfig

	// i18nDefaultLocale is the default locale.
	i18nDefaultLocale string

	// moduleName is the Go module name used for generated code.
	moduleName string

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
		moduleName: "playground",
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
func (a *GeneratorAdapter) Generate(
	ctx context.Context,
	request *wasm_dto.GenerateFromSourcesRequest,
) (*wasm_dto.GenerateFromSourcesResponse, error) {
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

	fsWriter, generatorService, errResp := a.createGeneratorService(ctx, request, moduleName, annotator)
	if errResp != nil {
		return errResp, nil
	}

	artefacts, manifest, err := generatorService.GenerateProject(ctx, entryPoints)
	if err != nil {
		return a.errorResponse(fmt.Sprintf("generation failed: %v", err)), nil
	}

	return &wasm_dto.GenerateFromSourcesResponse{
		Success:   true,
		Artefacts: a.convertArtefacts(artefacts, fsWriter),
		Manifest:  a.convertManifest(manifest),
	}, nil
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
		PKJSEmitter:        NewInMemoryPKJSEmitter(),
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

// discoverEntryPoints finds .pk files in the sources map and returns them as
// entry points. The paths are prefixed with the module name to create
// fully-qualified import paths.
//
// Takes sources (map[string]string) which contains the source files to scan.
// Takes moduleName (string) which is the prefix for fully-qualified paths.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry
// points with their paths and page status.
func (*GeneratorAdapter) discoverEntryPoints(sources map[string]string, moduleName string) []annotator_dto.EntryPoint {
	entryPoints := make([]annotator_dto.EntryPoint, 0, len(sources))

	for path := range sources {
		if !strings.HasSuffix(path, ".pk") {
			continue
		}

		isPage := strings.Contains(path, "pages/") || strings.HasPrefix(path, "pages/")

		fullPath := moduleName + "/" + path

		entryPoints = append(entryPoints, annotator_dto.EntryPoint{
			Path:   fullPath,
			IsPage: isPage,
		})
	}

	return entryPoints
}

// convertArtefacts converts generator artefacts to WASM DTOs.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which holds the
// generated output from the generator.
// Takes fsWriter (*InMemoryFSWriter) which provides access to files written
// to the in-memory filesystem.
//
// Returns []wasm_dto.GeneratedArtefact which contains the combined artefacts
// from both the generator output and the in-memory filesystem.
func (*GeneratorAdapter) convertArtefacts(
	artefacts []*generator_dto.GeneratedArtefact,
	fsWriter *InMemoryFSWriter,
) []wasm_dto.GeneratedArtefact {
	result := make([]wasm_dto.GeneratedArtefact, 0, len(artefacts))

	for _, art := range artefacts {
		artefactType := wasm_dto.ArtefactTypePartial
		if art.Component != nil && art.Component.IsPage {
			artefactType = wasm_dto.ArtefactTypePage
		}

		result = append(result, wasm_dto.GeneratedArtefact{
			Path:    art.SuggestedPath,
			Content: string(art.Content),
			Type:    artefactType,
		})
	}

	for path, content := range fsWriter.GetWrittenFiles() {
		found := false
		for _, art := range result {
			if art.Path == path {
				found = true
				break
			}
		}
		if found {
			continue
		}

		artefactType := determineArtefactType(path)
		result = append(result, wasm_dto.GeneratedArtefact{
			Path:    path,
			Content: string(content),
			Type:    artefactType,
		})
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
			PackagePath: partial.PackagePath,
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

// determineArtefactType guesses the artefact type from the file path.
//
// Takes path (string) which is the file path to analyse.
//
// Returns wasm_dto.ArtefactType which is the guessed type based on path
// patterns, defaulting to ArtefactTypePage if no pattern matches.
func determineArtefactType(path string) wasm_dto.ArtefactType {
	switch {
	case strings.Contains(path, "/pages/"):
		return wasm_dto.ArtefactTypePage
	case strings.Contains(path, "/partials/"):
		return wasm_dto.ArtefactTypePartial
	case strings.Contains(path, "/actions/"):
		return wasm_dto.ArtefactTypeAction
	case strings.HasSuffix(path, ".js"):
		return wasm_dto.ArtefactTypeJS
	case strings.Contains(path, "register"):
		return wasm_dto.ArtefactTypeRegister
	case strings.Contains(path, "manifest"):
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
