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

package coordinator_domain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultCacheWriteTimeout is the default time limit for writing to the cache.
	defaultCacheWriteTimeout = 10 * time.Second

	// logKeyHashedName is the log field key for the hashed name of a component.
	logKeyHashedName = "hashed_name"
)

// outputDiagnosticsIfPresent outputs build diagnostics using the set output
// port. Returns true if there were diagnostics, regardless of severity.
//
// Takes buildResult (*annotator_dto.ProjectAnnotationResult) which holds the
// diagnostics to output.
// Takes buildErr (error) which shows whether the build failed.
// Takes allSourceContents (map[string][]byte) which provides source code for
// context in diagnostic output.
//
// Returns bool which is true when diagnostics were present and output.
func (s *coordinatorService) outputDiagnosticsIfPresent(
	ctx context.Context,
	buildResult *annotator_dto.ProjectAnnotationResult,
	buildErr error,
	allSourceContents map[string][]byte,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	if buildResult == nil || len(buildResult.AllDiagnostics) == 0 {
		return false
	}

	isError := buildErr != nil
	if isError {
		l.Error("Build failed with the following errors:", logger_domain.Int("diagnostic_count", len(buildResult.AllDiagnostics)))
	} else {
		l.Warn("Build succeeded with warnings:", logger_domain.Int("diagnostic_count", len(buildResult.AllDiagnostics)))
	}

	if s.diagnosticOutput != nil {
		s.diagnosticOutput.OutputDiagnostics(buildResult.AllDiagnostics, allSourceContents, isError)
	}
	return true
}

// handleSemanticError processes a semantic error and returns the partial build
// result. Semantic errors indicate user errors (syntax/type errors) but may
// still have a partial result that LSP features can use.
//
// Takes semanticErr (*annotator_domain.SemanticError) which contains the
// semantic error to process.
// Takes buildResult (*annotator_dto.ProjectAnnotationResult) which holds the
// partial build result, if available.
// Takes allSourceContents (map[string][]byte) which maps file paths to their
// source content.
// Takes compilationLogs (*annotator_domain.CompilationLogStore) which stores
// internal compiler logs.
// Takes request (*coordinator_dto.BuildRequest) which provides the original
// build request for status updates.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the partial
// build result with source contents attached, or nil if no result exists.
// Returns error when semantic errors occurred during the build.
func (s *coordinatorService) handleSemanticError(
	ctx context.Context,
	semanticErr *annotator_domain.SemanticError,
	buildResult *annotator_dto.ProjectAnnotationResult,
	allSourceContents map[string][]byte,
	compilationLogs *annotator_domain.CompilationLogStore,
	request *coordinator_dto.BuildRequest,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(semanticErr.Diagnostics) > 0 {
		l.Error("Build failed with the following errors:", logger_domain.Int("diagnostic_count", len(semanticErr.Diagnostics)))
		if s.diagnosticOutput != nil {
			s.diagnosticOutput.OutputDiagnostics(semanticErr.Diagnostics, allSourceContents, true)
		}
		outputInternalCompilerLogs(semanticErr.Diagnostics, compilationLogs)
	}

	if buildResult != nil {
		buildResult.AllSourceContents = allSourceContents
		l.Warn("Returning partial build result despite errors", logger_domain.Int("component_count", len(buildResult.ComponentResults)))
	}
	s.updateStatus(ctx, stateFailed, buildResult, semanticErr, request.CausationID)
	return buildResult, semanticErr
}

// tryGenerateArtefacts generates code artefacts if a code emitter is set.
//
// Takes span (trace.Span) which records tracing data for the operation.
// Takes buildResult (*annotator_dto.ProjectAnnotationResult) which receives the
// generated artefacts.
// Takes request (*coordinator_dto.BuildRequest) which contains the build
// request for status updates.
// Takes pathDescription (string) which gives context for log messages.
//
// Returns error when artefact generation fails.
func (s *coordinatorService) tryGenerateArtefacts(
	ctx context.Context,
	span trace.Span,
	buildResult *annotator_dto.ProjectAnnotationResult,
	request *coordinator_dto.BuildRequest,
	pathDescription string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if s.codeEmitter == nil {
		return nil
	}

	l.Internal("Code emitter available, generating artefacts for dev-i mode" + pathDescription + "...")
	artefacts, emitErr := s.generateArtefacts(ctx, buildResult)
	if emitErr != nil {
		l.ReportError(span, emitErr, "Failed to generate artefacts")
		s.updateStatus(ctx, stateFailed, s.GetStatus().Result, emitErr, request.CausationID)
		return fmt.Errorf("artefact generation failed: %w", emitErr)
	}
	buildResult.FinalGeneratedArtefacts = artefacts
	l.Internal("Successfully generated artefacts"+pathDescription, logger_domain.Int("artefact_count", len(artefacts)))
	return nil
}

// tier1CacheResult holds the result of a tier 1 (introspection) cache lookup.
// Fields are ordered for optimal memory alignment.
type tier1CacheResult struct {
	// scriptHashes maps script paths to their content hashes.
	scriptHashes map[string]string

	// entry holds the cached introspection data to reuse in the fast path.
	entry *IntrospectionCacheEntry

	// introspectionHash is the cache key for introspection results.
	introspectionHash string

	// useFastPath indicates whether to skip Phase 1 and reuse cached data.
	useFastPath bool
}

// checkTier1Cache attempts to retrieve cached introspection data for the
// fast path.
//
// Takes span (trace.Span) which records cache lookup metrics and status.
// Takes request (*coordinator_dto.BuildRequest) which contains entry points
// and actions to hash.
//
// Returns tier1CacheResult with useFastPath=true if a valid cache entry
// exists, or useFastPath=false when the cache misses, hash calculation fails,
// or script hashes have changed.
func (s *coordinatorService) checkTier1Cache(
	ctx context.Context,
	span trace.Span,
	request *coordinator_dto.BuildRequest,
) tier1CacheResult {
	ctx, l := logger_domain.From(ctx, log)
	buildOpts := &buildOptions{
		InspectionCacheHints: nil,
		CausationID:          "",
		ChangedFiles:         nil,
		Resolver:             request.Resolver,
		SkipInspection:       false,
		FaultTolerant:        false,
	}
	introspectionHash, scriptHashes, hashErr := s.calculateIntrospectionHash(ctx, request.EntryPoints, buildOpts)
	if hashErr != nil {
		l.Warn("Failed to calculate introspection hash, falling back to full build.", logger_domain.Error(hashErr))
		return tier1CacheResult{
			scriptHashes:      nil,
			entry:             nil,
			introspectionHash: "",
			useFastPath:       false,
		}
	}

	introspectionEntry, introspErr := s.introspectionCache.Get(ctx, introspectionHash)
	if introspErr != nil {
		if !errors.Is(introspErr, ErrCacheMiss) {
			l.ReportError(span, introspErr, "Tier 1 cache adapter returned an unexpected error")
			cacheErrorCount.Add(ctx, 1)
		} else {
			l.Trace("Tier 1 cache MISS.")
			span.SetAttributes(attribute.String("cache.tier1.status", "MISS"))
			introspectionCacheMissCount.Add(ctx, 1)
		}
		return tier1CacheResult{
			scriptHashes:      scriptHashes,
			entry:             nil,
			introspectionHash: introspectionHash,
			useFastPath:       false,
		}
	}

	if !introspectionEntry.MatchesScriptHashes(scriptHashes) {
		l.Warn("Tier 1 cache entry found but script hashes don't match. Invalidating entry.")
		span.SetAttributes(attribute.String("cache.tier1.status", "STALE"))
		introspectionCacheMissCount.Add(ctx, 1)
		return tier1CacheResult{
			scriptHashes:      scriptHashes,
			entry:             nil,
			introspectionHash: introspectionHash,
			useFastPath:       false,
		}
	}

	l.Trace("Tier 1 cache HIT (introspection cache). Using FAST PATH (skip Phase 1).")
	span.SetAttributes(attribute.String("cache.tier1.status", "HIT"))
	introspectionCacheHitCount.Add(ctx, 1)
	return tier1CacheResult{
		scriptHashes:      scriptHashes,
		entry:             introspectionEntry,
		introspectionHash: introspectionHash,
		useFastPath:       true,
	}
}

// executeSlowPathBuild runs the full build (Phase 1 + Phase 2) when both
// caches miss.
//
// Takes span (trace.Span) which receives tracing events for the build.
// Takes request (*coordinator_dto.BuildRequest) which specifies the build
// parameters.
// Takes allSourceContents (map[string][]byte) which provides the source files
// to build.
// Takes inputHash (string) which identifies the build inputs for caching.
// Takes tier1Result (tier1CacheResult) which contains the tier 1 cache lookup
// result.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the build
// annotations.
// Returns error when the annotation pipeline fails catastrophically.
func (s *coordinatorService) executeSlowPathBuild(
	ctx context.Context,
	span trace.Span,
	request *coordinator_dto.BuildRequest,
	allSourceContents map[string][]byte,
	inputHash string,
	tier1Result tier1CacheResult,
	buildEpoch uint64,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Both cache tiers MISS. Starting SLOW PATH (full build with Phase 1 + Phase 2).")
	s.updateStatus(ctx, stateBuilding, s.GetStatus().Result, nil, request.CausationID)

	startTime := time.Now()
	var annotatorOpts []annotator_domain.AnnotationOption
	if request.FaultTolerant {
		annotatorOpts = append(annotatorOpts, annotator_domain.WithFaultTolerance())
	}
	if request.Resolver != nil {
		annotatorOpts = append(annotatorOpts, annotator_domain.WithResolver(request.Resolver))
	}
	phase1Result, buildErr := s.annotator.RunPhase1IntrospectionAndAnnotate(
		ctx, request.EntryPoints, tier1Result.scriptHashes, annotatorOpts...)
	duration := time.Since(startTime)
	buildDuration.Record(ctx, float64(duration.Milliseconds()))
	buildCount.Add(ctx, 1)
	slowPathBuildCount.Add(ctx, 1)

	if phase1Result != nil {
		s.outputDiagnosticsIfPresent(ctx, phase1Result.Annotations, buildErr, allSourceContents)
	}

	if buildErr != nil {
		l.ReportError(span, buildErr, "Annotation pipeline failed")

		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](buildErr); ok && phase1Result != nil {
			return s.handleSemanticError(ctx, semanticErr, phase1Result.Annotations, allSourceContents, phase1Result.Logs, request)
		}

		s.updateStatus(ctx, stateFailed, s.GetStatus().Result, buildErr, request.CausationID)
		return nil, buildErr
	}

	if phase1Result == nil {
		return nil, errors.New("unexpected nil annotation result")
	}

	phase1Result.Annotations.AllSourceContents = allSourceContents

	if err := s.tryGenerateArtefacts(ctx, span, phase1Result.Annotations, request, ""); err != nil {
		return nil, fmt.Errorf("generating artefacts for slow path build: %w", err)
	}

	if s.invalidationEpoch.Load() == buildEpoch {
		s.cacheIntrospectionResults(ctx, tier1Result, phase1Result.ComponentGraph, phase1Result.VirtualModule, phase1Result.TypeResolver)
	}

	s.writeTier2Cache(ctx, span, inputHash, buildEpoch, phase1Result.Annotations)
	s.updateStatus(ctx, stateReady, phase1Result.Annotations, nil, request.CausationID)
	return phase1Result.Annotations, nil
}

// writeTier2Cache writes the build result to the Tier 2 cache if the build
// epoch has not been invalidated.
//
// Takes span (trace.Span) which records any cache write errors.
// Takes inputHash (string) which is the cache key for storing the result.
// Takes buildEpoch (uint64) which identifies the epoch to check for
// invalidation before writing.
// Takes annotations (*annotator_dto.ProjectAnnotationResult) which contains
// the build result to cache.
func (s *coordinatorService) writeTier2Cache(
	ctx context.Context,
	span trace.Span,
	inputHash string,
	buildEpoch uint64,
	annotations *annotator_dto.ProjectAnnotationResult,
) {
	_, l := logger_domain.From(ctx, log)
	if s.invalidationEpoch.Load() != buildEpoch {
		l.Internal("Skipping Tier 2 cache write - cache was invalidated during build.")
		return
	}
	if cacheErr := s.cache.Set(ctx, inputHash, annotations); cacheErr != nil {
		l.ReportError(span, cacheErr, "Failed to write build result to Tier 2 cache")
		cacheErrorCount.Add(ctx, 1)
		return
	}
	l.Internal("Successfully cached new build result to Tier 2.")
}

// cacheIntrospectionResults stores introspection results in the Tier 1 cache
// for faster builds in the future.
//
// Takes tier1Result (tier1CacheResult) which provides the cache key and script
// hashes from the first processing step.
// Takes componentGraph (*annotator_dto.ComponentGraph) which contains the
// parsed component structure.
// Takes virtualModule (*annotator_dto.VirtualModule) which holds the module
// data to cache.
// Takes typeResolver (*annotator_domain.TypeResolver) which provides type
// resolution state to cache.
func (s *coordinatorService) cacheIntrospectionResults(
	ctx context.Context,
	tier1Result tier1CacheResult,
	componentGraph *annotator_dto.ComponentGraph,
	virtualModule *annotator_dto.VirtualModule,
	typeResolver *annotator_domain.TypeResolver,
) {
	ctx, l := logger_domain.From(ctx, log)
	if tier1Result.introspectionHash == "" || tier1Result.scriptHashes == nil {
		return
	}
	if componentGraph == nil || virtualModule == nil || typeResolver == nil {
		return
	}

	introspectionEntry := &IntrospectionCacheEntry{
		VirtualModule:  virtualModule,
		TypeResolver:   typeResolver,
		ComponentGraph: componentGraph,
		ScriptHashes:   tier1Result.scriptHashes,
		Timestamp:      time.Now(),
		Version:        1,
	}
	if cacheErr := s.introspectionCache.Set(ctx, tier1Result.introspectionHash, introspectionEntry); cacheErr != nil {
		l.Warn("Failed to write introspection results to Tier 1 cache", logger_domain.Error(cacheErr))
		cacheErrorCount.Add(ctx, 1)
	} else {
		l.Internal("Successfully cached introspection results to Tier 1 for future fast-path builds.")
	}
}

// buildWaiter passes the result of a build back to callers that are waiting.
type buildWaiter struct {
	// result holds the annotation output after a build finishes; nil until done.
	result *annotator_dto.ProjectAnnotationResult

	// err holds any error that occurred during the build.
	err error

	// done signals when the build has finished; closed by notifyWaiters.
	done chan struct{}
}

// executeBuild runs the core build logic on a cache miss, orchestrated by the
// buildLoop. This implements the two-tier caching strategy: first check Tier 2
// (annotation cache) for a complete result, then Tier 1 (introspection cache)
// for the fast path, otherwise run the full build (slow path).
//
// Takes inputHash (string) which identifies the unique build input for caching.
// Takes request (*coordinator_dto.BuildRequest) which contains the build
// configuration and source files to process.
// Takes allSourceContents (map[string][]byte) which provides the raw source
// file contents keyed by file path.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the annotation
// data, possibly partial on semantic errors (fault-tolerant pattern).
// Returns error when the build fails or an unexpected type is returned.
func (s *coordinatorService) executeBuild(
	ctx context.Context,
	inputHash string,
	request *coordinator_dto.BuildRequest,
	allSourceContents map[string][]byte,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.executeBuild")
	defer span.End()

	buildEpoch := s.invalidationEpoch.Load()
	v, err, _ := s.buildGroup.Do(inputHash, func() (any, error) {
		cachedResult, err := s.cache.Get(ctx, inputHash)
		if err == nil {
			l.Trace("Tier 2 cache HIT (annotation cache). Returning complete result.")
			span.SetAttributes(attribute.String("cache.tier2.status", "HIT"))
			cacheHitCount.Add(ctx, 1)
			s.updateStatus(ctx, stateReady, cachedResult, nil, request.CausationID)
			return cachedResult, nil
		}
		if !errors.Is(err, ErrCacheMiss) {
			l.ReportError(span, err, "Tier 2 cache adapter returned an unexpected error")
			cacheErrorCount.Add(ctx, 1)
		}

		l.Trace("Tier 2 cache MISS. Checking Tier 1 (introspection cache)...")
		span.SetAttributes(attribute.String("cache.tier2.status", "MISS"))
		cacheMissCount.Add(ctx, 1)

		tier1Result := s.checkTier1Cache(ctx, span, request)
		if tier1Result.useFastPath {
			return s.executePartialBuild(ctx, tier1Result.entry, request, allSourceContents, inputHash, buildEpoch)
		}

		return s.executeSlowPathBuild(ctx, span, request, allSourceContents, inputHash, tier1Result, buildEpoch)
	})

	if err != nil {
		if v != nil {
			result, ok := v.(*annotator_dto.ProjectAnnotationResult)
			if !ok {
				return nil, fmt.Errorf("unexpected type from singleflight: %w", err)
			}
			return result, err
		}
		return nil, fmt.Errorf("executing build via singleflight: %w", err)
	}
	result, ok := v.(*annotator_dto.ProjectAnnotationResult)
	if !ok {
		return nil, errors.New("unexpected type from singleflight, expected *ProjectAnnotationResult")
	}
	return result, nil
}

// executePartialBuild implements the fast path for template-only changes.
// It skips Phase 1 (expensive type introspection) and jumps directly to
// Phase 2 (annotation) using cached introspection data.
//
// Takes introspectionEntry (*IntrospectionCacheEntry) which provides the
// cached type introspection data from a previous full build.
// Takes request (*coordinator_dto.BuildRequest) which specifies the build
// configuration and actions to perform.
// Takes allSourceContents (map[string][]byte) which contains the source file
// contents keyed by path.
// Takes fullHash (string) which identifies the cache key for storing results.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the annotated
// project output.
// Returns error when the annotation pipeline fails or artefact generation
// fails.
func (s *coordinatorService) executePartialBuild(
	ctx context.Context,
	introspectionEntry *IntrospectionCacheEntry,
	request *coordinator_dto.BuildRequest,
	allSourceContents map[string][]byte,
	fullHash string,
	buildEpoch uint64,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.executePartialBuild")
	defer span.End()

	l.Internal("Starting FAST PATH build (reusing cached introspection, only running Phase 2).")
	s.updateStatus(ctx, stateBuilding, s.GetStatus().Result, nil, request.CausationID)

	startTime := time.Now()
	annotatorOpts := s.buildPartialAnnotatorOptions(ctx, request, introspectionEntry)

	buildResult, compilationLogs, buildErr := s.annotator.AnnotateProjectWithCachedIntrospection(
		ctx,
		introspectionEntry.ComponentGraph,
		introspectionEntry.VirtualModule,
		introspectionEntry.TypeResolver,
		annotatorOpts...,
	)
	duration := time.Since(startTime)
	partialBuildDuration.Record(ctx, float64(duration.Milliseconds()))
	fastPathBuildCount.Add(ctx, 1)

	l.Internal("FAST PATH build completed.", logger_domain.Int64("duration_ms", duration.Milliseconds()))

	s.outputDiagnosticsIfPresent(ctx, buildResult, buildErr, allSourceContents)

	if buildErr != nil {
		l.ReportError(span, buildErr, "Fast path annotation pipeline failed")

		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](buildErr); ok {
			return s.handleSemanticError(ctx, semanticErr, buildResult, allSourceContents, compilationLogs, request)
		}

		s.updateStatus(ctx, stateFailed, s.GetStatus().Result, buildErr, request.CausationID)
		return nil, buildErr
	}

	buildResult.AllSourceContents = allSourceContents

	if err := s.tryGenerateArtefacts(ctx, span, buildResult, request, " (fast path)"); err != nil {
		return nil, fmt.Errorf("generating artefacts for fast path build: %w", err)
	}

	if s.invalidationEpoch.Load() == buildEpoch {
		if cacheErr := s.cache.Set(ctx, fullHash, buildResult); cacheErr != nil {
			l.ReportError(span, cacheErr, "Failed to write fast path build result to Tier 2 cache")
			cacheErrorCount.Add(ctx, 1)
		} else {
			l.Internal("Successfully cached fast path build result to Tier 2.")
		}
	} else {
		l.Internal("Skipping Tier 2 cache write - cache was invalidated during build.")
	}

	s.updateStatus(ctx, stateReady, buildResult, nil, request.CausationID)
	return buildResult, nil
}

// buildPartialAnnotatorOptions assembles annotator options for a partial
// (fast-path) build, including fault tolerance, resolver overrides, and
// targeted component scoping.
//
// Takes ctx (context.Context) which provides the logging context.
// Takes request (*coordinator_dto.BuildRequest) which supplies the build
// configuration.
//
// Returns []annotator_domain.AnnotationOption which contains the assembled
// options.
func (s *coordinatorService) buildPartialAnnotatorOptions(
	ctx context.Context,
	request *coordinator_dto.BuildRequest,
	introspectionEntry *IntrospectionCacheEntry,
) []annotator_domain.AnnotationOption {
	_, l := logger_domain.From(ctx, log)
	var opts []annotator_domain.AnnotationOption
	if request.FaultTolerant {
		opts = append(opts, annotator_domain.WithFaultTolerance())
	}
	if request.Resolver != nil {
		opts = append(opts, annotator_domain.WithResolver(request.Resolver))
	}

	if len(request.EntryPoints) > 0 && len(request.EntryPoints) < len(introspectionEntry.ComponentGraph.Components) {
		changedHashedNames := s.resolveEntryPointsToHashedNames(ctx, request.EntryPoints, introspectionEntry.ComponentGraph)
		if len(changedHashedNames) > 0 {
			l.Internal("Scoping fast-path build to targeted components",
				logger_domain.Int("targeted", len(changedHashedNames)),
				logger_domain.Int("total", len(introspectionEntry.ComponentGraph.Components)))
			opts = append(opts, annotator_domain.WithChangedComponents(changedHashedNames))
		}
	}

	return opts
}

// resolveEntryPointsToHashedNames converts entry point paths to the hashed
// names used in the component graph.
//
// Entry point paths may be relative (e.g. "pages/main.pk") or module-prefixed
// (e.g. "mymodule/pages/main.pk"). Both forms are resolved to absolute paths
// and matched against PathToHashedName.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which lists the entry points
// to resolve.
// Takes graph (*annotator_dto.ComponentGraph) which maps paths to hashed names.
//
// Returns []string which contains the resolved hashed names.
func (s *coordinatorService) resolveEntryPointsToHashedNames(
	_ context.Context,
	entryPoints []annotator_dto.EntryPoint,
	graph *annotator_dto.ComponentGraph,
) []string {
	baseDir := s.resolver.GetBaseDir()
	moduleName := s.resolver.GetModuleName()
	result := make([]string, 0, len(entryPoints))

	for _, ep := range entryPoints {
		path := ep.Path
		if moduleName != "" {
			path = strings.TrimPrefix(path, moduleName+"/")
		}
		absPath := filepath.Join(baseDir, path)

		if hashedName, ok := graph.PathToHashedName[absPath]; ok {
			result = append(result, hashedName)
		}
	}
	return result
}

// generateArtefacts generates fully-emitted Go code artefacts for all
// components in the build result. Called after annotation completes and only
// used in dev-i mode to provide executable code to the interpreted runner.
//
// Takes buildResult (*annotator_dto.ProjectAnnotationResult) which contains
// the annotated components to generate code for.
//
// Returns []*generator_dto.GeneratedArtefact which contains the generated
// code artefacts for each component.
// Returns error when the build result has no virtual module or when
// generating any single artefact fails.
func (s *coordinatorService) generateArtefacts(
	ctx context.Context,
	buildResult *annotator_dto.ProjectAnnotationResult,
) ([]*generator_dto.GeneratedArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.generateArtefacts")
	defer span.End()

	if buildResult.VirtualModule == nil {
		return nil, errors.New("build result has no virtual module")
	}

	artefacts := make([]*generator_dto.GeneratedArtefact, 0, len(buildResult.ComponentResults))

	for hashedName, vc := range buildResult.VirtualModule.ComponentsByHash {
		annotationResult, ok := buildResult.ComponentResults[hashedName]
		if !ok {
			l.Warn("No annotation result for component", logger_domain.String(logKeyHashedName, hashedName))
			continue
		}

		artefact, err := s.generateSingleArtefact(ctx, buildResult, hashedName, vc, annotationResult)
		if err != nil {
			return nil, fmt.Errorf("generating artefact for component %s: %w", hashedName, err)
		}
		artefacts = append(artefacts, artefact)
	}

	buildResult.GeneratedArtefactCount = len(artefacts)
	l.Internal("Successfully generated all artefacts", logger_domain.Int("artefact_count", len(artefacts)))
	return artefacts, nil
}

// generateSingleArtefact creates code for a single component and returns the
// generated artefact.
//
// Takes buildResult (*annotator_dto.ProjectAnnotationResult) which provides the
// full project annotation including the virtual module.
// Takes hashedName (string) which identifies the component.
// Takes vc (*annotator_dto.VirtualComponent) which specifies the component to
// generate code for.
// Takes annotationResult (*annotator_dto.AnnotationResult) which contains the
// parsed annotations for the component.
//
// Returns *generator_dto.GeneratedArtefact which contains the generated code
// and metadata for the component.
// Returns error when code generation fails.
func (s *coordinatorService) generateSingleArtefact(
	ctx context.Context,
	buildResult *annotator_dto.ProjectAnnotationResult,
	hashedName string,
	vc *annotator_dto.VirtualComponent,
	annotationResult *annotator_dto.AnnotationResult,
) (*generator_dto.GeneratedArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	if annotationResult.VirtualModule == nil {
		annotationResult.VirtualModule = buildResult.VirtualModule
	}

	request := generator_dto.GenerateRequest{
		SourcePath:                vc.Source.SourcePath,
		OutputPath:                vc.VirtualGoFilePath,
		PackagePrefix:             "",
		PackageName:               vc.HashedName,
		BaseDir:                   "",
		HashedName:                hashedName,
		CanonicalGoPackagePath:    vc.CanonicalGoPackagePath,
		IsPage:                    vc.IsPage,
		VirtualInstances:          generator_dto.ConvertVirtualInstances(vc.VirtualInstances),
		CollectionName:            vc.Source.CollectionName,
		ModuleName:                "",
		VerifyGeneratedCode:       false,
		IsEmail:                   vc.IsEmail,
		EnablePrerendering:        s.enablePrerendering && !vc.IsEmail,
		EnableStaticHoisting:      s.enableStaticHoisting,
		StripHTMLComments:         s.stripHTMLComments,
		EnableDwarfLineDirectives: s.enableDwarfLineDirectives,
	}

	emittedCode, emitDiags, err := s.codeEmitter.EmitCode(ctx, annotationResult, request)
	if err != nil {
		l.Error("Fatal error emitting code for component",
			logger_domain.String(logKeyHashedName, hashedName),
			logger_domain.String("source", vc.Source.SourcePath),
			logger_domain.Error(err))
		return nil, fmt.Errorf("failed to emit code for %s: %w", vc.Source.SourcePath, err)
	}

	if len(emitDiags) > 0 {
		l.Warn("Code emitter returned diagnostics",
			logger_domain.String(logKeyHashedName, hashedName),
			logger_domain.Int("diag_count", len(emitDiags)))
	}

	l.Trace("Successfully emitted code for component",
		logger_domain.String(logKeyHashedName, hashedName),
		logger_domain.Int("code_size", len(emittedCode)))

	jsArtefactID := s.emitClientScript(ctx, vc, annotationResult)

	return &generator_dto.GeneratedArtefact{
		Result:        annotationResult,
		Component:     vc,
		SuggestedPath: vc.VirtualGoFilePath,
		Content:       emittedCode,
		JSArtefactID:  jsArtefactID,
	}, nil
}

// emitClientScript returns the artefact ID for the component's client script.
//
// Takes vc (*annotator_dto.VirtualComponent) which identifies the
// source file and is used to derive the component's relative path.
// Takes annotationResult (*annotator_dto.AnnotationResult) which holds
// the parsed ClientScript source.
//
// Returns string which is the registered artefact ID, or "" when no
// JS was emitted.
func (s *coordinatorService) emitClientScript(
	ctx context.Context,
	vc *annotator_dto.VirtualComponent,
	annotationResult *annotator_dto.AnnotationResult,
) string {
	if s.clientScriptEmitter == nil || annotationResult.ClientScript == "" {
		return ""
	}
	ctx, l := logger_domain.From(ctx, log)
	pagePath := deriveComponentPagePath(vc.Source.SourcePath, s.resolver.GetBaseDir())
	moduleName := s.resolver.GetModuleName()
	jsPath, err := s.clientScriptEmitter.EmitJS(ctx, annotationResult.ClientScript, pagePath, moduleName, "", false)
	if err != nil {
		l.Warn("Failed to emit client-side JS; continuing without it",
			logger_domain.String("source", vc.Source.SourcePath),
			logger_domain.Error(err))
		return ""
	}
	l.Trace("Emitted client-side JS",
		logger_domain.String("source", vc.Source.SourcePath),
		logger_domain.String("jsPath", jsPath))
	return jsPath
}

// deriveComponentPagePath returns the project-relative path without the .pk suffix.
//
// The result is the human-readable segment the JS emitter uses when
// building an artefact URL (e.g. "partials/integrations/grid"). It
// mirrors derivePagePath in the generator service so the compiled and
// dev-i paths produce identical artefact URLs.
//
// Takes sourcePath (string) which is the component's source file path.
// Takes baseDir (string) which is the project root.
//
// Returns string which is the relative component path without the
// ".pk" suffix.
func deriveComponentPagePath(sourcePath, baseDir string) string {
	absSource, absErr := filepath.Abs(sourcePath)
	absBase, baseErr := filepath.Abs(baseDir)
	if absErr == nil && baseErr == nil {
		relativePath, err := filepath.Rel(absBase, absSource)
		if err == nil && !strings.HasPrefix(relativePath, "..") {
			return strings.TrimSuffix(filepath.ToSlash(relativePath), ".pk")
		}
	}
	relativePath, err := filepath.Rel(baseDir, sourcePath)
	if err != nil || strings.HasPrefix(relativePath, "..") {
		relativePath = filepath.Base(sourcePath)
	}
	return strings.TrimSuffix(filepath.ToSlash(relativePath), ".pk")
}

// outputInternalCompilerLogs writes internal compiler logs to stderr for
// debugging. This always writes to stderr, not to the set output, as it is
// only meant for internal debugging.
//
// Takes diagnostics ([]*ast_domain.Diagnostic) which provides the list of
// diagnostics to check for compiler logs.
// Takes compilationLogs (*annotator_domain.CompilationLogStore) which stores
// the internal compiler logs to output.
func outputInternalCompilerLogs(diagnostics []*ast_domain.Diagnostic, compilationLogs *annotator_domain.CompilationLogStore) {
	if len(diagnostics) == 0 {
		return
	}
	firstErrorFile := diagnostics[0].SourcePath
	logs, found := compilationLogs.GetLogs(firstErrorFile)
	if !found {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\n--- Internal compiler logs for %s ---\n", firstErrorFile)
	_, _ = fmt.Fprintln(os.Stderr, logs)
	_, _ = fmt.Fprintln(os.Stderr, "--- End of internal logs ---")
}
