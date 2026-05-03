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

package annotator_domain

// Implements the main annotator service that orchestrates the entire
// compilation pipeline from parsing to code generation. Coordinates graph
// building, module virtualisation, type inspection, partial expansion, semantic
// analysis, and asset processing.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/sfcparser"
)

const (
	// pathSlash is the forward slash character used to separate URL path parts.
	pathSlash = "/"

	// attributeKeyDiagnosticCount is the logging attribute key for diagnostic counts.
	attributeKeyDiagnosticCount = "diagnostic_count"
)

// annotationOptions holds settings that control how the annotation pipeline
// handles errors during processing.
type annotationOptions struct {
	// resolver overrides the default resolver for this request. Used in LSP mode
	// to provide per-module resolvers.
	resolver resolver_domain.ResolverPort

	// changedComponents limits the fast-path rebuild to only the specified
	// components (by hashed name).
	//
	// When nil, all components are processed. When set, only components in
	// this set are refreshed, annotated, and have artefacts generated.
	changedComponents map[string]bool

	// faultTolerant allows annotation to continue when errors occur. Used in LSP
	// mode to return partial results instead of stopping on the first error.
	faultTolerant bool
}

// AnnotationOption is a function type that changes how annotations behave.
type AnnotationOption func(*annotationOptions)

// AnnotatorService is the main part of the Piko compilation pipeline.
// It implements AnnotatorPort and manages the full annotation workflow.
type AnnotatorService struct {
	// resolver provides path resolution and module information lookup.
	resolver resolver_domain.ResolverPort

	// fsReader reads source files from the file system.
	fsReader FSReaderPort

	// cache stores parsed components for faster repeated lookups.
	cache ComponentCachePort

	// collectionService processes collection directives within components.
	collectionService CollectionServicePort

	// typeInspector builds type information for the code being analysed.
	typeInspector TypeInspectorBuilderPort

	// componentRegistry provides lookup of registered PKC components. If nil,
	// custom tag collection uses heuristic detection instead.
	componentRegistry ComponentRegistryPort

	// cssProcessor parses and transforms CSS during component expansion.
	cssProcessor *CSSProcessor

	// ignoreMatcher specifies file paths to skip when collecting files.
	ignoreMatcher *ignoreMatcher

	// logStore holds compilation log entries for the current operation.
	logStore *CompilationLogStore

	// pathsConfig holds the path settings needed by the annotator for
	// directory resolution, route calculation, and asset URL generation.
	pathsConfig AnnotatorPathsConfig

	// assetsConfig holds asset profiles, screen sizes, and densities for
	// compile-time static asset analysis. Separate from ServerConfig because
	// assets are configured programmatically via WithAssets().
	assetsConfig config.AssetsConfig

	// inMemoryMode skips file system operations such as reading directories.
	inMemoryMode bool
}

// AnnotatorServiceConfig groups the dependencies needed to create an
// AnnotatorService.
type AnnotatorServiceConfig struct {
	// Resolver provides path resolution and base directory lookup.
	Resolver resolver_domain.ResolverPort

	// FSReader provides file system read operations.
	FSReader FSReaderPort

	// Cache stores processed components to avoid repeated work.
	Cache ComponentCachePort

	// CollectionService handles collection operations for the annotator.
	CollectionService CollectionServicePort

	// TypeInspector provides type information for template
	// compilation.
	TypeInspector TypeInspectorBuilderPort

	// ComponentRegistry provides lookup of registered PKC components.
	// If nil, custom tag collection uses heuristic detection instead.
	ComponentRegistry ComponentRegistryPort

	// CSSProcessor handles CSS extraction and scoping for
	// component templates.
	CSSProcessor *CSSProcessor

	// AssetsConfig holds asset profiles and responsive image settings for
	// compile-time analysis. If nil, an empty config is used.
	AssetsConfig *config.AssetsConfig

	// PathsConfig holds the path settings needed by the annotator for
	// directory resolution, route calculation, and asset URL generation.
	PathsConfig AnnotatorPathsConfig

	// DebugLogDir specifies the directory for debug log files.
	// Defaults to config.CompilerDebugLogDir.
	DebugLogDir string

	// CompilationLogLevel sets the log level for compilation output.
	CompilationLogLevel slog.Level

	// EnableDebugLogFiles controls whether the compiler writes a detailed debug
	// log for each component. Defaults to config.CompilerEnableDebugLogFiles.
	EnableDebugLogFiles bool

	// InMemoryMode skips filesystem operations such as walking the directory for
	// Go files. Use this for WASM or testing where file I/O is not available.
	InMemoryMode bool
}

// NewAnnotatorService creates a new AnnotatorService with the provided
// configuration.
//
// Use slog.LevelDebug for development and compiled modes, slog.LevelWarn for
// interpreted mode.
//
// Takes serviceConfig (*AnnotatorServiceConfig) which specifies the service
// settings.
//
// Returns *AnnotatorService which is the configured service ready for use.
// Returns error when the compilation log store cannot be initialised.
func NewAnnotatorService(ctx context.Context, serviceConfig *AnnotatorServiceConfig) (*AnnotatorService, error) {
	excludePatterns := []string{}

	logStore, err := NewCompilationLogStore(
		ctx,
		serviceConfig.EnableDebugLogFiles,
		serviceConfig.DebugLogDir,
		serviceConfig.CompilationLogLevel,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise compilation log store: %w", err)
	}

	var assetsConfig config.AssetsConfig
	if serviceConfig.AssetsConfig != nil {
		assetsConfig = *serviceConfig.AssetsConfig
	}

	return &AnnotatorService{
		resolver:          serviceConfig.Resolver,
		fsReader:          serviceConfig.FSReader,
		typeInspector:     serviceConfig.TypeInspector,
		cssProcessor:      serviceConfig.CSSProcessor,
		pathsConfig:       serviceConfig.PathsConfig,
		assetsConfig:      assetsConfig,
		cache:             serviceConfig.Cache,
		ignoreMatcher:     newIgnoreMatcher(serviceConfig.Resolver.GetBaseDir(), excludePatterns),
		logStore:          logStore,
		collectionService: serviceConfig.CollectionService,
		componentRegistry: serviceConfig.ComponentRegistry,
		inMemoryMode:      serviceConfig.InMemoryMode,
	}, nil
}

// AnnotateProject is the primary entry point for a full project build.
//
// Performs a complete, multi-stage analysis of all components in the dependency
// graph (both pages and partials) and returns a single, full result. Runs both
// Phase 1 (expensive type introspection) and Phase 2 (annotation). For
// template-only changes, use AnnotateProjectWithCachedIntrospection to skip
// Phase 1.
//
// Actions are auto-discovered from the actions/ directory during Phase 1.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the
// components to analyse.
// Takes scriptHashes (map[string]string) which contains SHA1 hashes of script
// block content for cache invalidation. Pass nil if script hashes are not
// available (will disable script-based cache invalidation).
// Takes opts (...AnnotationOption) which configures annotation behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the complete
// analysis results for all components.
// Returns *CompilationLogStore which contains logs from the compilation.
// Returns error when introspection or annotation fails.
func (s *AnnotatorService) AnnotateProject(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	scriptHashes map[string]string,
	opts ...AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error) {
	options := &annotationOptions{faultTolerant: false}
	for _, opt := range opts {
		opt(options)
	}

	ctx, span, l := log.Span(ctx, "AnnotatorService.AnnotateProject")
	defer span.End()

	s.logStore.Clear(ctx)
	defer s.logStore.Shutdown(ctx)

	l.Internal("--- [PHASE 1] Starting: Type Introspection (Full Build Path) ---")
	phase1Result, err := s.runPhase1Introspection(ctx, entryPoints, scriptHashes, options)
	if err != nil {
		if phase1Result != nil && len(phase1Result.Diagnostics) > 0 && phase1Result.ComponentGraph != nil {
			return &annotator_dto.ProjectAnnotationResult{
				ComponentResults:        nil,
				AllDiagnostics:          phase1Result.Diagnostics,
				AllSourceContents:       phase1Result.ComponentGraph.AllSourceContents,
				FinalAssetManifest:      nil,
				VirtualModule:           nil,
				FinalGeneratedArtefacts: nil,
			}, s.logStore, err
		}
		return nil, s.logStore, err
	}
	componentGraph := phase1Result.ComponentGraph
	virtualModule := phase1Result.VirtualModule
	typeResolver := phase1Result.TypeResolver
	phase1Diagnostics := phase1Result.Diagnostics
	if ast_domain.HasErrors(phase1Diagnostics) {
		return &annotator_dto.ProjectAnnotationResult{
			ComponentResults:        nil,
			AllDiagnostics:          phase1Diagnostics,
			AllSourceContents:       componentGraph.AllSourceContents,
			FinalAssetManifest:      nil,
			VirtualModule:           nil,
			FinalGeneratedArtefacts: nil,
		}, s.logStore, NewSemanticError(phase1Diagnostics)
	}
	l.Internal("--- [PHASE 1] Finished: Type Introspection Complete ---")

	return s.runPhase2Annotation(ctx, componentGraph, virtualModule, typeResolver, phase1Diagnostics, nil, options)
}

// AnnotateProjectWithCachedIntrospection is the fast path for template-only
// changes. It skips Phase 1 (expensive type introspection) and jumps directly
// to Phase 2 (annotation) using cached introspection data from a previous
// build.
//
// Achieves 5-10x performance improvement when only <template>, <style>, or
// <i18n> blocks have changed, as it avoids the expensive packages.Load() call
// for Go type introspection.
//
// Should be called by the coordinator when Tier 2 cache (full annotation) misses,
// Tier 1 cache (introspection) hits, and the introspection hash matches the
// cached entry.
//
// Actions are auto-discovered from the cached ActionManifest in VirtualModule.
//
// Takes cachedComponentGraph (*annotator_dto.ComponentGraph) which provides
// the component graph from a previous build.
// Takes cachedVirtualModule (*annotator_dto.VirtualModule) which provides the
// virtual module data from a previous build.
// Takes cachedTypeResolver (*TypeResolver) which provides the type resolver
// from a previous build.
// Takes opts (...AnnotationOption) which provides optional behaviour controls.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the annotation
// results for the project.
// Returns *CompilationLogStore which contains the compilation logs.
// Returns error when refreshing mutable component data fails.
func (s *AnnotatorService) AnnotateProjectWithCachedIntrospection(
	ctx context.Context,
	cachedComponentGraph *annotator_dto.ComponentGraph,
	cachedVirtualModule *annotator_dto.VirtualModule,
	cachedTypeResolver *TypeResolver,
	opts ...AnnotationOption,
) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error) {
	options := &annotationOptions{faultTolerant: false}
	for _, opt := range opts {
		opt(options)
	}

	ctx, span, l := log.Span(ctx, "AnnotatorService.AnnotateProjectWithCachedIntrospection")
	defer span.End()

	s.logStore.Clear(ctx)
	defer s.logStore.Shutdown(ctx)

	l.Internal("--- [FAST PATH] Skipping Phase 1, using cached type introspection data ---")

	l.Internal("[FAST PATH] Re-parsing templates, styles, and i18n from current disk state...")
	if err := s.refreshMutableComponentData(ctx, cachedComponentGraph, options.changedComponents); err != nil {
		l.ReportError(span, err, "Failed to refresh mutable component data")
		return nil, s.logStore, fmt.Errorf("fast path failed to refresh templates/styles: %w", err)
	}
	l.Internal("[FAST PATH] Successfully refreshed mutable content from disk")

	initialDiagnostics := make([]*ast_domain.Diagnostic, 0)

	return s.runPhase2Annotation(ctx, cachedComponentGraph, cachedVirtualModule, cachedTypeResolver, initialDiagnostics, nil, options)
}

// Annotate runs the full annotation process for a single entry point. It is a
// helper for development and single-file generation that calls AnnotateProject
// and extracts the result for the given path.
//
// Takes mainSourcePath (string) which specifies the path to the entry point.
// Takes isPage (bool) which indicates whether the entry point is a page.
//
// Returns *annotator_dto.AnnotationResult which contains the annotation for the
// single entry point.
// Returns *CompilationLogStore which contains the compilation logs.
// Returns error when path resolution fails or the entry point is not found in
// the project result.
func (s *AnnotatorService) Annotate(ctx context.Context, mainSourcePath string, isPage bool) (*annotator_dto.AnnotationResult, *CompilationLogStore, error) {
	ctx, span, l := log.Span(ctx, "AnnotatorService.Annotate", logger_domain.String("entryPoint", mainSourcePath))
	defer span.End()

	entryPoints := []annotator_dto.EntryPoint{{Path: mainSourcePath, IsPage: isPage, IsPublic: false, IsEmail: false, VirtualPageSource: nil, IsE2EOnly: false}}
	projectResult, compilationLogs, err := s.AnnotateProject(ctx, entryPoints, nil)
	if err != nil {
		return nil, compilationLogs, err
	}

	if ast_domain.HasDiagnostics(projectResult.AllDiagnostics) {
		l.Warn("Compilation succeeded with warnings.",
			logger_domain.Int(attributeKeyDiagnosticCount, len(projectResult.AllDiagnostics)))
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n", FormatAllDiagnostics(projectResult.AllDiagnostics, projectResult.AllSourceContents))
	}

	resolvedPath, err := s.resolver.ResolvePKPath(ctx, mainSourcePath, "")
	if err != nil {
		return nil, compilationLogs, fmt.Errorf("could not resolve path to find single annotation result: %w", err)
	}
	hashedName, ok := projectResult.VirtualModule.Graph.PathToHashedName[resolvedPath]
	if !ok {
		return nil, compilationLogs, fmt.Errorf("internal error: could not find hash for path '%s' in project result", resolvedPath)
	}
	singleResult, ok := projectResult.ComponentResults[hashedName]
	if !ok {
		return nil, compilationLogs, fmt.Errorf("internal error: AnnotateProject returned no result for entry point '%s'", mainSourcePath)
	}

	singleResult.AnnotatedAST.Diagnostics = projectResult.AllDiagnostics
	return singleResult, compilationLogs, nil
}

// RunPhase1IntrospectionAndAnnotate runs the full two-phase annotation
// pipeline and returns both the intermediate Phase 1 introspection results
// (for Tier 1 caching) and the final Phase 2 annotation results (for Tier 2
// caching).
//
// Designed for the coordinator's slow path to enable populating the
// introspection cache after a full build.
//
// Actions are auto-discovered from the actions/ directory during Phase 1.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the entry
// points to process.
// Takes scriptHashes (map[string]string) which contains SHA1 hashes of script
// block content for cache invalidation.
// Takes opts (...AnnotationOption) which provides optional behaviour controls.
//
// Returns *Phase1Result which contains component graph, virtual module, type
// resolver, annotations, and logs.
// Returns error when introspection or annotation fails.
func (s *AnnotatorService) RunPhase1IntrospectionAndAnnotate(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	scriptHashes map[string]string,
	opts ...AnnotationOption,
) (*Phase1Result, error) {
	options := &annotationOptions{faultTolerant: false}
	for _, opt := range opts {
		opt(options)
	}

	ctx, span, l := log.Span(ctx, "AnnotatorService.RunPhase1IntrospectionAndAnnotate")
	defer span.End()

	if options.faultTolerant {
		l.Internal("Running in FAULT-TOLERANT mode (LSP)")
	} else {
		l.Internal("Running in FAST-FAIL mode (Generator)")
	}

	l.Internal("--- [PHASE 1] Starting: Type Introspection (Full Build Path) ---")
	phase1Result, phase1Err := s.runPhase1Introspection(ctx, entryPoints, scriptHashes, options)
	if phase1Err != nil {
		return s.handlePhase1Error(ctx, phase1Result, phase1Err)
	}

	result, logs, err := s.runPhase2Annotation(ctx, phase1Result.ComponentGraph, phase1Result.VirtualModule, phase1Result.TypeResolver, phase1Result.Diagnostics, nil, options)
	if err != nil {
		return &Phase1Result{
			ComponentGraph: phase1Result.ComponentGraph,
			VirtualModule:  phase1Result.VirtualModule,
			TypeResolver:   phase1Result.TypeResolver,
			Annotations:    result,
			Logs:           logs,
		}, err
	}

	return &Phase1Result{
		ComponentGraph: phase1Result.ComponentGraph,
		VirtualModule:  phase1Result.VirtualModule,
		TypeResolver:   phase1Result.TypeResolver,
		Annotations:    result,
		Logs:           logs,
	}, nil
}

// getEffectiveResolver returns the resolver from options if set, or the
// service's default resolver otherwise.
//
// Takes opts (*annotationOptions) which may contain a resolver override.
//
// Returns resolver_domain.ResolverPort which is the resolver to use.
func (s *AnnotatorService) getEffectiveResolver(opts *annotationOptions) resolver_domain.ResolverPort {
	if opts != nil && opts.resolver != nil {
		return opts.resolver
	}
	return s.resolver
}

// runPhase2Annotation executes Phase 2 of the annotation pipeline:
// per-component annotation, asset aggregation, and srcset annotation. This is
// extracted as a separate method so it can be reused by both the full build
// path and the fast path (cached introspection).
//
// Takes componentGraph (*annotator_dto.ComponentGraph) which provides the
// graph of components and their relationships.
// Takes virtualModule (*annotator_dto.VirtualModule) which contains the
// components to annotate, keyed by hash.
// Takes typeResolver (*TypeResolver) which resolves type information during
// annotation.
// Takes initialDiagnostics ([]*ast_domain.Diagnostic) which contains any
// diagnostics from earlier phases.
// Takes actions (map[string]ActionInfoProvider) which provides action metadata
// for annotation.
// Takes options (*annotationOptions) which configures the annotation behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the aggregated
// annotation results, diagnostics, and asset manifest.
// Returns *CompilationLogStore which provides the compilation log for this run.
// Returns error when the worker pool fails or severe semantic errors occur.
func (s *AnnotatorService) runPhase2Annotation(
	ctx context.Context,
	componentGraph *annotator_dto.ComponentGraph,
	virtualModule *annotator_dto.VirtualModule,
	typeResolver *TypeResolver,
	initialDiagnostics []*ast_domain.Diagnostic,
	actions map[string]ActionInfoProvider,
	options *annotationOptions,
) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [PHASE 2] Starting: Per-Component Annotation ---")

	if actions == nil {
		actions = buildActionsFromManifest(virtualModule.ActionManifest)
	}

	allComponentsToAnnotate := virtualModule.ComponentsByHash
	if options.changedComponents != nil {
		filtered := make(map[string]*annotator_dto.VirtualComponent, len(options.changedComponents))
		for name, vc := range allComponentsToAnnotate {
			if options.changedComponents[name] {
				filtered[name] = vc
			}
		}
		l.Internal("[FAST PATH] Scoped Phase 2 annotation to changed components only",
			logger_domain.Int("total_components", len(virtualModule.ComponentsByHash)),
			logger_domain.Int("targeted_components", len(filtered)))
		allComponentsToAnnotate = filtered
	}

	workerConfig := &annotationWorkerConfig{
		componentGraph: componentGraph,
		virtualModule:  virtualModule,
		typeResolver:   typeResolver,
		actions:        actions,
		options:        options,
	}

	resultsChan, err := s.runAnnotationWorkerPool(ctx, allComponentsToAnnotate, workerConfig)
	if err != nil {
		return nil, s.logStore, err
	}

	finalResult := &annotator_dto.ProjectAnnotationResult{
		ComponentResults:        make(map[string]*annotator_dto.AnnotationResult, len(allComponentsToAnnotate)),
		AllDiagnostics:          initialDiagnostics,
		AllSourceContents:       componentGraph.AllSourceContents,
		FinalAssetManifest:      nil,
		VirtualModule:           virtualModule,
		FinalGeneratedArtefacts: nil,
	}

	severeErrors := aggregateAnnotationResults(resultsChan, finalResult)
	finalResult.AnnotatedComponentCount = len(allComponentsToAnnotate)

	if len(severeErrors) > 0 {
		l.Error("Compilation failed with severe errors during per-component annotation.")
		return finalResult, s.logStore, NewSemanticError(finalResult.AllDiagnostics)
	}

	runAssetAggregation(ctx, finalResult)

	s.runSrcsetAnnotation(ctx, finalResult)

	return handlePhase2Completion(ctx, finalResult, s.logStore, options)
}

// handlePhase1Error builds a Phase1Result when Phase 1 fails.
//
// This helper makes sure the LSP can show error messages even when Phase 1
// does not work. It creates a basic result with any error messages that are
// ready, or an empty result if there are none.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
// Takes phase1Result (*Phase1IntrospectionResult) which holds partial results
// and error messages from the failed step.
// Takes phase1Err (error) which is the error from Phase 1.
//
// Returns *Phase1Result which holds a basic result with any error messages
// for the LSP to show.
// Returns error when Phase 1 has failed.
func (s *AnnotatorService) handlePhase1Error(
	ctx context.Context,
	phase1Result *Phase1IntrospectionResult,
	phase1Err error,
) (*Phase1Result, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Phase 1 returned error",
		logger_domain.Error(phase1Err),
		logger_domain.Int(attributeKeyDiagnosticCount, len(phase1Result.Diagnostics)),
		logger_domain.Bool("has_component_graph", phase1Result.ComponentGraph != nil))

	if len(phase1Result.Diagnostics) > 0 {
		minimalResult := &annotator_dto.ProjectAnnotationResult{
			ComponentResults:        nil,
			AllDiagnostics:          phase1Result.Diagnostics,
			AllSourceContents:       nil,
			FinalAssetManifest:      nil,
			VirtualModule:           nil,
			FinalGeneratedArtefacts: nil,
		}
		l.Warn("Phase 1 failed, returning minimal result with diagnostics for LSP",
			logger_domain.Int(attributeKeyDiagnosticCount, len(phase1Result.Diagnostics)))
		return &Phase1Result{
			ComponentGraph: phase1Result.ComponentGraph,
			VirtualModule:  phase1Result.VirtualModule,
			TypeResolver:   phase1Result.TypeResolver,
			Annotations:    minimalResult,
			Logs:           s.logStore,
		}, phase1Err
	}

	l.Warn("Phase 1 failed but no diagnostics available, returning nil result")
	return &Phase1Result{
		ComponentGraph: phase1Result.ComponentGraph,
		VirtualModule:  phase1Result.VirtualModule,
		TypeResolver:   phase1Result.TypeResolver,
		Annotations:    nil,
		Logs:           s.logStore,
	}, phase1Err
}

// Phase1IntrospectionResult contains the artefacts from Phase 1 introspection.
type Phase1IntrospectionResult struct {
	// ComponentGraph holds the dependency graph of components found in the source.
	ComponentGraph *annotator_dto.ComponentGraph

	// VirtualModule holds data about the virtual module found during
	// introspection.
	VirtualModule *annotator_dto.VirtualModule

	// TypeResolver provides type information lookup for documentation analysis.
	TypeResolver *TypeResolver

	// Diagnostics holds any errors or warnings found during phase 1 analysis.
	Diagnostics []*ast_domain.Diagnostic
}

// Phase1Result contains all artefacts from Phase 1 including annotation results
// and logs.
type Phase1Result struct {
	// ComponentGraph holds the links between components that depend on each other.
	ComponentGraph *annotator_dto.ComponentGraph

	// VirtualModule holds the generated module data for this phase result.
	VirtualModule *annotator_dto.VirtualModule

	// TypeResolver holds the resolver used to look up type information.
	TypeResolver *TypeResolver

	// Annotations holds the results from phase one processing.
	Annotations *annotator_dto.ProjectAnnotationResult

	// Logs holds compiler output and error messages from the analysis phase.
	Logs *CompilationLogStore
}

// runPhase1Introspection executes the expensive type introspection phase of
// annotation. This includes building the component graph, virtualising the
// module, and initialising the type resolver via packages.Load().
//
// This is Phase 1 of the annotation pipeline and can be cached separately from
// Phase 2 because it only depends on <script> blocks and .go files, not on
// <template>, <style>, or <i18n> blocks.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the
// components to process.
// Takes scriptHashes (map[string]string) which provides hashes for cache
// validation.
// Takes options (*annotationOptions) which controls annotation behaviour.
//
// Returns *Phase1IntrospectionResult which contains the component graph,
// virtual module, type resolver, and diagnostics.
// Returns error when graph building, expansion, virtualisation, or type
// resolver initialisation fails.
func (s *AnnotatorService) runPhase1Introspection(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	scriptHashes map[string]string,
	options *annotationOptions,
) (*Phase1IntrospectionResult, error) {
	componentGraph, allDiagnostics, err := s.buildUnifiedGraph(ctx, entryPoints, options)
	if err != nil {
		return newPhase1Result(componentGraph, nil, nil, allDiagnostics), err
	}

	if err := s.checkGraphErrors(ctx, allDiagnostics, options); err != nil {
		return newPhase1Result(componentGraph, nil, nil, allDiagnostics), err
	}

	expandedEntryPoints, collectionDiags, expandErr := s.expandCollectionDirectives(ctx, componentGraph, entryPoints, options)
	allDiagnostics = append(allDiagnostics, collectionDiags...)
	if expandErr != nil {
		return newPhase1Result(componentGraph, nil, nil, allDiagnostics), expandErr
	}

	actionManifest, actionDiscoveryDiags := s.discoverActions(ctx, options)
	allDiagnostics = append(allDiagnostics, actionDiscoveryDiags...)

	virtualModule, err := s.virtualiseModule(ctx, componentGraph, expandedEntryPoints, options)
	if err != nil {
		return newPhase1Result(componentGraph, nil, nil, allDiagnostics), err
	}
	allDiagnostics = append(allDiagnostics, virtualModule.Diagnostics...)

	virtualModule.ActionManifest = actionManifest

	typeResolver, err := s.initialiseTypeResolver(ctx, virtualModule, scriptHashes, options)
	if err != nil {
		return newPhase1Result(componentGraph, virtualModule, nil, allDiagnostics), err
	}

	if actionManifest != nil && len(actionManifest.Actions) > 0 {
		actionTypeDiags := ResolveActionTypes(ctx, actionManifest, typeResolver)
		allDiagnostics = append(allDiagnostics, actionTypeDiags...)
	}

	return newPhase1Result(componentGraph, virtualModule, typeResolver, allDiagnostics), nil
}

// checkGraphErrors handles error recovery for graph processing failures.
//
// Takes allDiagnostics ([]*ast_domain.Diagnostic) which contains errors found
// during graph processing.
// Takes options (*annotationOptions) which controls error handling behaviour.
//
// Returns error when diagnostics contain errors and fast-fail mode is enabled.
func (*AnnotatorService) checkGraphErrors(
	ctx context.Context,
	allDiagnostics []*ast_domain.Diagnostic,
	options *annotationOptions,
) error {
	_, l := logger_domain.From(ctx, log)
	if !ast_domain.HasErrors(allDiagnostics) {
		return nil
	}

	if !options.faultTolerant {
		l.Internal("Graph has errors, halting compilation (fast-fail mode)")
		return NewSemanticError(allDiagnostics)
	}

	l.Internal("Graph has errors, but continuing annotation (fault-tolerant mode)",
		logger_domain.Int("error_count", len(allDiagnostics)))
	return nil
}

// buildUnifiedGraph builds a unified component graph from the given entry
// points.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the paths to
// process.
// Takes options (*annotationOptions) which controls how errors are handled.
//
// Returns *annotator_dto.ComponentGraph which contains the parsed components.
// Returns []*ast_domain.Diagnostic which contains any parsing warnings or
// errors found.
// Returns error when graph building fails or when parsing errors occur in
// strict mode.
func (s *AnnotatorService) buildUnifiedGraph(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	options *annotationOptions,
) (*annotator_dto.ComponentGraph, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 1/9] Starting: Unified Graph Building ---")

	allPaths := make([]string, len(entryPoints))
	for i, ep := range entryPoints {
		allPaths[i] = ep.Path
	}

	graphBuilder := NewGraphBuilder(s.getEffectiveResolver(options), s.fsReader, s.cache, s.pathsConfig, options.faultTolerant)
	componentGraph, graphDiags, err := graphBuilder.Build(ctx, allPaths)
	if err != nil {
		return componentGraph, graphDiags, fmt.Errorf("stage 1 (GraphBuilder) failed fatally: %w", err)
	}

	if ast_domain.HasErrors(graphDiags) {
		if options.faultTolerant {
			l.Warn("Graph building found parsing errors, but continuing in fault-tolerant mode (LSP)",
				logger_domain.Int("error_count", len(graphDiags)))
			return componentGraph, graphDiags, nil
		}
		l.Error("Graph building failed with critical diagnostics. Halting compilation.")
		return componentGraph, graphDiags, NewSemanticError(graphDiags)
	}

	l.Internal("--- [STAGE 1/9] Finished: Graph Building ---", logger_domain.Int("diagnostics_found", len(graphDiags)), logger_domain.Int("components_found", len(componentGraph.Components)))
	return componentGraph, graphDiags, nil
}

// expandSingleCollectionComponent expands a single collection component into
// multiple entry points.
//
// For static providers (markdown): creates one entry point per content item.
// For dynamic providers (CMS): creates one entry point with a dynamic route
// pattern.
//
// Takes parsedComp (*annotator_dto.ParsedComponent) which contains the parsed
// collection component with its provider and collection name.
// Takes options (*annotationOptions) which provides settings for the process.
//
// Returns []annotator_dto.EntryPoint which contains the expanded entry points.
// Returns error when the collection directive cannot be processed.
func (s *AnnotatorService) expandSingleCollectionComponent(
	ctx context.Context,
	parsedComp *annotator_dto.ParsedComponent,
	options *annotationOptions,
) ([]annotator_dto.EntryPoint, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Found collection component",
		logger_domain.String(logKeyPath, parsedComp.SourcePath),
		logger_domain.String("collection", parsedComp.CollectionName),
		logger_domain.String("provider", parsedComp.CollectionProvider))

	resolver := s.getEffectiveResolver(options)
	directive := &collection_dto.CollectionDirectiveInfo{
		CacheConfig:       nil,
		Filters:           nil,
		ProviderName:      parsedComp.CollectionProvider,
		CollectionName:    parsedComp.CollectionName,
		LayoutPath:        parsedComp.SourcePath,
		RoutePath:         s.calculateBaseRoutePathWithResolver(parsedComp.SourcePath, resolver),
		BasePath:          resolver.GetBaseDir(),
		ContentModulePath: parsedComp.ContentModulePath,
		ParamName:         parsedComp.CollectionParamName,
	}

	collectionEntryPoints, err := s.collectionService.ProcessCollectionDirective(ctx, directive)
	if err != nil {
		return nil, fmt.Errorf("expanding collection directive for %s: %w", parsedComp.SourcePath, err)
	}

	l.Internal("Expanded collection directive",
		logger_domain.String(logKeyPath, parsedComp.SourcePath),
		logger_domain.Int("entry_points", len(collectionEntryPoints)))

	return s.convertCollectionToAnnotatorEntryPointsWithResolver(collectionEntryPoints, resolver), nil
}

// convertCollectionToAnnotatorEntryPointsWithResolver converts collection
// entry points into annotator format using the given resolver.
//
// Takes collectionEPs ([]*collection_dto.CollectionEntryPoint) which provides
// the collection entry points to convert.
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution.
//
// Returns []annotator_dto.EntryPoint which contains the converted entry points
// with paths relative to the module.
func (*AnnotatorService) convertCollectionToAnnotatorEntryPointsWithResolver(
	collectionEPs []*collection_dto.CollectionEntryPoint,
	resolver resolver_domain.ResolverPort,
) []annotator_dto.EntryPoint {
	baseDir := resolver.GetBaseDir()
	moduleName := resolver.GetModuleName()
	result := make([]annotator_dto.EntryPoint, 0, len(collectionEPs))

	for _, cep := range collectionEPs {
		relativePath, err := filepath.Rel(baseDir, cep.Path)
		if err != nil {
			relativePath = cep.Path
		}
		pikoPath := filepath.ToSlash(filepath.Join(moduleName, relativePath))

		result = append(result, annotator_dto.EntryPoint{
			Path:     pikoPath,
			IsPage:   cep.IsPage,
			IsPublic: true,
			IsEmail:  false,
			VirtualPageSource: &annotator_dto.VirtualPageSource{
				TemplatePath:      cep.Path,
				CollectionName:    cep.DynamicCollection,
				ProviderName:      cep.DynamicProvider,
				InitialProps:      cep.InitialProps,
				RouteOverride:     cep.RoutePatternOverride,
				CollectionContext: nil,
			},
			IsE2EOnly: false,
		})
	}

	return result
}

// expandCollectionDirectives processes collection directives and turns them
// into separate entry points.
//
// Takes componentGraph (*annotator_dto.ComponentGraph) which holds the parsed
// components to check for collection directives.
// Takes entryPoints ([]annotator_dto.EntryPoint) which provides the starting
// entry points to expand.
// Takes options (*annotationOptions) which provides the resolver override.
//
// Returns []annotator_dto.EntryPoint which contains the expanded entry points.
// Returns []*ast_domain.Diagnostic which contains any warning diagnostics for
// skipped collection components.
// Returns error when expanding a collection component fails for a reason other
// than a missing provider.
func (s *AnnotatorService) expandCollectionDirectives(
	ctx context.Context,
	componentGraph *annotator_dto.ComponentGraph,
	entryPoints []annotator_dto.EntryPoint,
	options *annotationOptions,
) ([]annotator_dto.EntryPoint, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 1.5/9] Starting: Collection Directive Expansion ---")

	if s.collectionService == nil {
		l.Internal("No collection service configured, skipping collection expansion")
		return entryPoints, nil, nil
	}

	resolver := s.getEffectiveResolver(options)
	var expandedEntryPoints []annotator_dto.EntryPoint
	var diagnostics []*ast_domain.Diagnostic
	collectionComponentPaths := make(map[string]bool)

	for _, parsedComp := range componentGraph.Components {
		if !parsedComp.HasCollection {
			continue
		}

		componentEPs, err := s.expandSingleCollectionComponent(ctx, parsedComp, options)
		if err != nil {
			if errors.Is(err, collection_dto.ErrProviderNotFound) {
				l.Warn("Skipping collection component: provider not registered",
					logger_domain.String(logKeyPath, parsedComp.SourcePath),
					logger_domain.String("provider", parsedComp.CollectionProvider),
					logger_domain.String("collection", parsedComp.CollectionName))

				diagnostics = append(diagnostics, providerNotFoundDiagnostic(parsedComp))
				collectionComponentPaths[parsedComp.SourcePath] = true

				continue
			}

			return nil, nil, fmt.Errorf("expanding collection component %q: %w", parsedComp.SourcePath, err)
		}
		expandedEntryPoints = append(expandedEntryPoints, componentEPs...)
		collectionComponentPaths[parsedComp.SourcePath] = true
	}

	for _, ep := range entryPoints {
		resolvedPath, _ := resolver.ResolvePKPath(ctx, ep.Path, "")
		if !collectionComponentPaths[resolvedPath] {
			expandedEntryPoints = append(expandedEntryPoints, ep)
		}
	}

	l.Internal("--- [STAGE 1.5/9] Finished: Collection Directive Expansion ---",
		logger_domain.Int("original_count", len(entryPoints)),
		logger_domain.Int("expanded_count", len(expandedEntryPoints)))

	return expandedEntryPoints, diagnostics, nil
}

// providerNotFoundDiagnostic builds a warning diagnostic for a collection
// component whose provider is not registered.
//
// Takes comp (*annotator_dto.ParsedComponent) which provides the source path
// and provider name for the diagnostic.
//
// Returns *ast_domain.Diagnostic which is a warning-severity diagnostic with
// code T153.
func providerNotFoundDiagnostic(comp *annotator_dto.ParsedComponent) *ast_domain.Diagnostic {
	return ast_domain.NewDiagnosticWithCode(
		ast_domain.Warning,
		fmt.Sprintf(
			"Collection provider %q is not registered. "+
				"Pages using p-collection with this provider will not be available. "+
				"To enable it, %s",
			comp.CollectionProvider,
			providerEnableHint(comp.CollectionProvider),
		),
		fmt.Sprintf("p-collection provider=%q", comp.CollectionProvider),
		annotator_dto.CodeCollectionProviderNotFound,
		ast_domain.Location{},
		comp.SourcePath,
	)
}

// providerEnableHint returns a user-facing hint for how to enable a given
// collection provider.
//
// Takes providerName (string) which is the name of the missing provider.
//
// Returns string which is the hint text.
func providerEnableHint(providerName string) string {
	switch providerName {
	case "markdown":
		return "pass piko.WithMarkdownParser(...) to piko.New()."
	default:
		return fmt.Sprintf("register a provider named %q with piko.New().", providerName)
	}
}

// discoverActions scans the actions/ directory and discovers action structs
// that embed piko.ActionMetadata. This is Stage 1.6 of the Phase 1 pipeline.
//
// Takes options (*annotationOptions) which provides the resolver override.
//
// Returns *annotator_dto.ActionManifest which contains the discovered actions.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors.
func (s *AnnotatorService) discoverActions(
	ctx context.Context,
	options *annotationOptions,
) (*annotator_dto.ActionManifest, []*ast_domain.Diagnostic) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 1.6/9] Starting: Action Discovery ---")

	resolver := s.getEffectiveResolver(options)

	var opts []ActionDiscovererOption
	if s.inMemoryMode {
		opts = append(opts, WithActionDiscovererInMemoryMode())
	}
	discoverer := NewActionDiscoverer(resolver, s.fsReader, s.pathsConfig, opts...)

	manifest, diagnostics := discoverer.Discover(ctx)

	l.Internal("--- [STAGE 1.6/9] Finished: Action Discovery ---",
		logger_domain.Int("action_count", len(manifest.Actions)),
		logger_domain.Int(attributeKeyDiagnosticCount, len(diagnostics)))

	return manifest, diagnostics
}

// calculateBaseRoutePathWithResolver converts a source file path into a URL
// route path using the given resolver.
//
// Takes sourcePath (string) which is the full path to the source file.
// Takes resolver (resolver_domain.ResolverPort) which provides the base
// directory.
//
// Returns string which is the URL path for routing.
func (s *AnnotatorService) calculateBaseRoutePathWithResolver(
	sourcePath string,
	resolver resolver_domain.ResolverPort,
) string {
	baseDir := resolver.GetBaseDir()
	pagesDir := filepath.Join(baseDir, s.pathsConfig.PagesSourceDir)

	relPath, err := filepath.Rel(pagesDir, sourcePath)
	if err != nil {
		return pathSlash
	}

	urlPath := filepath.ToSlash(relPath)
	urlPath = strings.TrimSuffix(urlPath, ".pk")

	if urlPath == "index" {
		return pathSlash
	}
	if trimmed, found := strings.CutSuffix(urlPath, "/index"); found {
		return pathSlash + trimmed + pathSlash
	}

	if !strings.HasPrefix(urlPath, pathSlash) {
		urlPath = pathSlash + urlPath
	}

	return urlPath
}

// virtualiseModule creates a virtual module from the component graph.
//
// Takes componentGraph (*annotator_dto.ComponentGraph) which defines how
// components relate to each other.
// Takes entryPoints ([]annotator_dto.EntryPoint) which lists the starting
// points for the analysis.
// Takes options (*annotationOptions) which provides the resolver settings.
//
// Returns *annotator_dto.VirtualModule which contains the virtual module
// with source overlays.
// Returns error when collecting Go files fails or when creating the virtual
// module fails.
func (s *AnnotatorService) virtualiseModule(
	ctx context.Context,
	componentGraph *annotator_dto.ComponentGraph,
	entryPoints []annotator_dto.EntryPoint,
	options *annotationOptions,
) (*annotator_dto.VirtualModule, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 2/9] Starting: Module Virtualisation ---")

	resolver := s.getEffectiveResolver(options)
	originalGoFiles, err := s.collectOriginalGoFilesWithResolver(ctx, resolver)
	if err != nil {
		return nil, fmt.Errorf("stage 2 (ModuleVirtualiser) failed to collect Go files: %w", err)
	}

	moduleVirtualiser := NewModuleVirtualiser(resolver, s.pathsConfig)
	virtualModule, err := moduleVirtualiser.Virtualise(ctx, componentGraph, originalGoFiles, entryPoints)
	if err != nil {
		return nil, fmt.Errorf("stage 2 (ModuleVirtualiser) failed: %w", err)
	}

	l.Internal("--- [STAGE 2/9] Finished: Module Virtualisation ---", logger_domain.Int("overlay_files", len(virtualModule.SourceOverlay)))
	return virtualModule, nil
}

// initialiseTypeResolver builds the type inspector and creates a shared
// TypeResolver for the given virtual module.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which contains the source
// overlay and module settings.
// Takes scriptHashes (map[string]string) which maps script paths to their
// content hashes for finding changes.
// Takes options (*annotationOptions) which provides the resolver override.
//
// Returns *TypeResolver which provides type information queries.
// Returns error when the type inspector fails to build or cannot be fetched.
func (s *AnnotatorService) initialiseTypeResolver(
	ctx context.Context,
	virtualModule *annotator_dto.VirtualModule,
	scriptHashes map[string]string,
	options *annotationOptions,
) (*TypeResolver, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 3/9] Starting: Type Inspection ---")

	if options != nil && options.resolver != nil {
		resolver := options.resolver
		s.typeInspector.SetConfig(inspector_dto.Config{
			BaseDir:    resolver.GetBaseDir(),
			ModuleName: resolver.GetModuleName(),
			BuildFlags: inspector_dto.AnalysisBuildFlags,
		})
	}

	if err := s.typeInspector.Build(ctx, virtualModule.SourceOverlay, scriptHashes); err != nil {
		diagnostics := convertTypeInspectorErrorToDiagnostics(ctx, err, virtualModule, s.fsReader)
		if len(diagnostics) > 0 {
			l.Internal("Converted Go compile errors to diagnostics", logger_domain.Int("count", len(diagnostics)))
			return nil, NewSemanticError(diagnostics)
		}
		return nil, fmt.Errorf("stage 3 (TypeInspector) failed to build from virtual module: %w", err)
	}
	inspector, ok := s.typeInspector.GetQuerier()
	if !ok {
		return nil, errors.New("stage 3 (TypeInspector) failed: inspector could not be retrieved after build")
	}

	typeResolver := NewTypeResolver(inspector, virtualModule, s.collectionService)
	l.Internal("--- [STAGE 3/9] Finished: Type Inspection & Shared TypeResolver created ---")
	return typeResolver, nil
}

// collectOriginalGoFilesWithResolver walks the project's base directory to
// find all non-test .go files using the provided resolver.
//
// Takes resolver (resolver_domain.ResolverPort) which provides the base
// directory to walk.
//
// Returns map[string][]byte which contains the file paths and their contents.
// Returns error when walking the directory fails or a file cannot be read.
func (s *AnnotatorService) collectOriginalGoFilesWithResolver(
	ctx context.Context,
	resolver resolver_domain.ResolverPort,
) (map[string][]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	_, span, _ := l.Span(ctx, "AnnotatorService.collectOriginalGoFiles")
	defer span.End()

	if s.inMemoryMode {
		return make(map[string][]byte), nil
	}

	goFiles := make(map[string][]byte)
	baseDir := resolver.GetBaseDir()

	localIgnoreMatcher := newIgnoreMatcher(baseDir, []string{})

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory %q: %w", path, err)
		}

		if shouldSkipEntry(localIgnoreMatcher, d) {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		if shouldIncludeGoFile(d.Name()) {
			return readAndStoreGoFile(ctx, s.fsReader, path, goFiles)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("collecting Go files from %q: %w", baseDir, err)
	}
	return goFiles, nil
}

// refreshMutableComponentData re-parses only the parts of components that can
// change without script changes (template, styles, i18n) from their current
// files on disk. The fast path uses this to update cached component data
// without running expensive type checking again.
//
// Takes componentGraph (*annotator_dto.ComponentGraph) which contains the
// cached components to refresh.
//
// Returns error when reading or parsing component files fails.
//
// Changes the cached ComponentGraph directly, updating:
//   - Template (parsed TemplateAST)
//   - StyleBlocks (raw style data)
//   - LocalTranslations (i18n data)
//
// It keeps the cached Script data unchanged, as parsing scripts is expensive
// and script content has not changed if we are on the fast path.
//
// Design note: The Tier 1 cache stores ComponentGraph, which includes parsed
// templates. This is not ideal because templates can change on their own,
// without script changes. Rather than changing the whole cache structure, we
// fix this by refreshing templates on the fast path.
func (s *AnnotatorService) refreshMutableComponentData(ctx context.Context, componentGraph *annotator_dto.ComponentGraph, changedComponents map[string]bool) error {
	refreshCount := len(componentGraph.Components)
	if changedComponents != nil {
		refreshCount = len(changedComponents)
	}
	ctx, span, l := log.Span(ctx, "refreshMutableComponentData",
		logger_domain.Int("component_count", refreshCount))
	defer span.End()

	for hashedName, component := range componentGraph.Components {
		if changedComponents != nil && !changedComponents[hashedName] {
			continue
		}
		sourcePath := component.SourcePath
		if sourcePath == "" {
			l.Warn("Component has no source path, skipping refresh",
				logger_domain.String("hashed_name", hashedName))
			continue
		}

		fileData, err := s.fsReader.ReadFile(ctx, sourcePath)
		if err != nil {
			return fmt.Errorf("failed to read component file '%s': %w", sourcePath, err)
		}

		sfcResult, err := sfcparser.Parse(fileData)
		if err != nil {
			return fmt.Errorf("failed to parse SFC for '%s': %w", sourcePath, err)
		}

		templateStartLocation := ast_domain.Location{
			Line:   sfcResult.TemplateContentLocation.Line,
			Column: sfcResult.TemplateContentLocation.Column,
			Offset: 0,
		}

		freshTemplate, err := parseTemplateBlock(ctx, sfcResult.Template, sourcePath, templateStartLocation)
		if err != nil {
			l.Warn("Template parse error during refresh (will be caught in annotation phase)",
				logger_domain.String(logKeyPath, sourcePath),
				logger_domain.Error(err))
		}

		freshTranslations, err := parseI18nBlocks(sfcResult, sourcePath)
		if err != nil {
			return fmt.Errorf("failed to parse i18n blocks for '%s': %w", sourcePath, err)
		}

		component.Template = freshTemplate
		component.StyleBlocks = sfcResult.Styles
		component.LocalTranslations = freshTranslations

		l.Trace("Refreshed mutable data for component",
			logger_domain.String(logKeyPath, sourcePath),
			logger_domain.String("hashed_name", hashedName))
	}

	l.Internal("Successfully refreshed mutable content for all components")
	return nil
}

// WithFaultTolerance enables fault-tolerant mode for the annotator.
//
// In fault-tolerant mode, the annotator keeps processing code even when it
// finds errors. This lets LSP features work on valid parts of the code.
// Without this option, the annotator stops at the first error (generator mode).
//
// Returns AnnotationOption which sets up the annotator for fault tolerance.
func WithFaultTolerance() AnnotationOption {
	return func(opts *annotationOptions) {
		opts.faultTolerant = true
	}
}

// WithResolver sets a custom resolver for this request, replacing the default.
//
// Use this in LSP mode to provide a resolver that has been set up for the
// specific project being checked.
//
// Takes resolver (resolver_domain.ResolverPort) which provides path resolution
// for the current request.
//
// Returns AnnotationOption which configures the resolver for the request.
func WithResolver(resolver resolver_domain.ResolverPort) AnnotationOption {
	return func(opts *annotationOptions) {
		opts.resolver = resolver
	}
}

// WithChangedComponents limits a fast-path rebuild to only the specified
// components (identified by their hashed names).
//
// Components not in this set are skipped during template refresh, Phase 2
// annotation, and artefact generation. This dramatically reduces rebuild time
// when only a small subset of components are affected by a file change.
//
// When this option is not provided, all components are processed (existing
// behaviour).
//
// Takes hashedNames ([]string) which lists the hashed names of components
// that need rebuilding (the changed file plus its transitive dependents).
//
// Returns AnnotationOption which scopes the rebuild to the given components.
func WithChangedComponents(hashedNames []string) AnnotationOption {
	return func(opts *annotationOptions) {
		opts.changedComponents = make(map[string]bool, len(hashedNames))
		for _, name := range hashedNames {
			opts.changedComponents[name] = true
		}
	}
}

// newPhase1Result creates a Phase1IntrospectionResult with fields in a
// set order.
//
// Takes graph (*annotator_dto.ComponentGraph) which provides the
// component dependency graph.
// Takes vm (*annotator_dto.VirtualModule) which represents the virtual
// module.
// Takes tr (*TypeResolver) which resolves type information.
// Takes diagnostics ([]*ast_domain.Diagnostic) which contains any
// diagnostics found.
//
// Returns *Phase1IntrospectionResult which is the assembled result.
func newPhase1Result(
	graph *annotator_dto.ComponentGraph,
	vm *annotator_dto.VirtualModule,
	tr *TypeResolver,
	diagnostics []*ast_domain.Diagnostic,
) *Phase1IntrospectionResult {
	return &Phase1IntrospectionResult{
		ComponentGraph: graph,
		VirtualModule:  vm,
		TypeResolver:   tr,
		Diagnostics:    diagnostics,
	}
}

// shouldSkipEntry checks whether a directory entry should be skipped during
// traversal.
//
// Takes matcher (*ignoreMatcher) which provides the ignore pattern matching.
// Takes entry (os.DirEntry) which is the directory entry to check.
//
// Returns bool which is true when the entry matches an ignore pattern.
func shouldSkipEntry(matcher *ignoreMatcher, entry os.DirEntry) bool {
	return matcher.Matches(entry.Name())
}

// shouldIncludeGoFile reports whether a file is a Go source file that is not a
// test file.
//
// Takes filename (string) which is the name of the file to check.
//
// Returns bool which is true if the file ends with .go but not _test.go.
func shouldIncludeGoFile(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasSuffix(filename, "_test.go")
}

// readAndStoreGoFile reads a Go source file and stores it in the given map.
//
// Takes reader (FSReaderPort) which provides file system read access.
// Takes path (string) which is the path of the file to read.
// Takes goFiles (map[string][]byte) which holds file contents by path.
//
// Returns error when the file cannot be read.
func readAndStoreGoFile(ctx context.Context, reader FSReaderPort, path string, goFiles map[string][]byte) error {
	content, err := reader.ReadFile(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to read go source file '%s': %w", path, err)
	}
	goFiles[path] = content
	return nil
}

// getMainComponent finds the main virtual component for an annotation result.
// This is a key helper for reading the output of the annotation pipeline.
//
// Takes result (*annotator_dto.AnnotationResult) which holds the annotated AST
// and virtual module data to search.
//
// Returns *annotator_dto.VirtualComponent which is the found main component.
// Returns error when result is nil, required fields are missing, or internal
// data is not consistent.
func getMainComponent(result *annotator_dto.AnnotationResult) (*annotator_dto.VirtualComponent, error) {
	if result == nil || result.VirtualModule == nil {
		return nil, errors.New("internal error: result or its virtual module is nil")
	}

	if result.AnnotatedAST == nil || result.AnnotatedAST.SourcePath == nil {
		return nil, errors.New("internal error: annotation result for a component is missing its AST or source path")
	}

	sourcePath := *result.AnnotatedAST.SourcePath
	hashedName, ok := result.VirtualModule.Graph.PathToHashedName[sourcePath]
	if !ok {
		return nil, fmt.Errorf("internal error: inconsistency, could not find hash for source path '%s'", sourcePath)
	}

	vc, ok := result.VirtualModule.ComponentsByHash[hashedName]
	if !ok {
		return nil, fmt.Errorf("internal error: inconsistency, could not find virtual component for hash '%s'", hashedName)
	}

	return vc, nil
}
