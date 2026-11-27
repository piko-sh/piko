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

package lifecycle_adapters

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/safedisk"
)

const (
	// fieldAbsolutePath is the log field name for absolute file paths.
	fieldAbsolutePath = "absolutePath"

	// fieldProjectRoot is the log field name for the project root directory.
	fieldProjectRoot = "projectRoot"

	// fieldPath is the log field name for recording file paths.
	fieldPath = "path"

	// fieldPackagePath is the log field name for Go package paths.
	fieldPackagePath = "pkg_path"

	// defaultJITCompileTimeout bounds one JIT compilation when no other limit is configured.
	defaultJITCompileTimeout = 5 * time.Minute
)

var (
	// errJITCompileTimeout is the cancellation cause recorded when a JIT compilation exceeds
	// its time limit.
	errJITCompileTimeout = errors.New("JIT compilation exceeded its time limit")

	// errNoInterpreterPool reports a build attempted without an interpreter pool.
	errNoInterpreterPool = errors.New("no interpreter pool is configured")
)

// InterpretedBuildOrchestrator handles the conversion of build artefacts into a runnable
// InterpretedManifestRunner.
type InterpretedBuildOrchestrator struct {
	// compileGroup stops repeated JIT compilations for the same path.
	compileGroup singleflight.Group

	// i18nService provides translation and localisation for manifest runners.
	i18nService i18n_domain.Service

	// cachedManifest holds the manifest from the last build for JIT compilation.
	cachedManifest *generator_dto.Manifest

	// progCache maps relative paths to compiled page entries.
	progCache map[string]*templater_adapters.PageEntry

	// dirtyCodeCache maps relative paths to their updated source code pending JIT
	// compilation.
	dirtyCodeCache map[string][]byte

	// reverseDepsMap maps component paths to the list of components that depend on them.
	reverseDepsMap map[string][]string

	// interpreterPool holds reusable interpreters for template processing. Replaced by a
	// fresh pool when user Go packages change, guarded by stateLock.
	interpreterPool templater_domain.InterpreterPoolPort

	// interpreterProvider builds fresh interpreter pools when user Go packages change, so
	// they are recompiled into a new symbol registry.
	interpreterProvider templater_domain.InterpreterProviderPort

	// clock supplies the time used for compilation metrics.
	clock clock.Clock

	// artefactByPackagePath maps Go package paths to their generated artefacts.
	artefactByPackagePath map[string]*generator_dto.GeneratedArtefact

	// sandboxFactory creates sandboxes for filesystem access within the orchestrator.
	sandboxFactory safedisk.Factory

	// pathsConfig holds the resolved path settings for the generator.
	pathsConfig generator_domain.GeneratorPathsConfig

	// i18nDefaultLocale is the default locale for internationalisation.
	i18nDefaultLocale string

	// projectRoot is the absolute path to the project root folder.
	projectRoot string

	// moduleName is the Go module path used to resolve imports.
	moduleName string

	// hostImports holds the memoised project-relative directories of the host imports.
	hostImports map[string]bool

	// packageStateWarned records the project packages already reported by
	// warnSeparatePackageState, keyed by relative directory.
	packageStateWarned sync.Map

	// jitCompileTimeout bounds one JIT compilation.
	jitCompileTimeout time.Duration

	// freshPoolRequested counts the requests for a fresh interpreter pool made by
	// InvalidateUserPackages, guarded by stateLock.
	freshPoolRequested uint64

	// freshPoolBuilt records the latest request covered by a completed build. stateLock
	// protects the counter.
	freshPoolBuilt uint64

	// hostImportsOnce computes hostImports on the first build.
	hostImportsOnce sync.Once

	// stateLock guards access to orchestrator state fields for safe concurrent use.
	stateLock sync.RWMutex
}

// InterpretedBuildOrchestratorDeps holds the dependencies required to construct an
// InterpretedBuildOrchestrator.
type InterpretedBuildOrchestratorDeps struct {
	// InterpreterPool provides pooled interpreters for template execution. Its modules must
	// already be loaded.
	InterpreterPool templater_domain.InterpreterPoolPort

	// InterpreterProvider rebuilds the interpreter pool after Go package changes. A nil
	// provider disables the refresh.
	InterpreterProvider templater_domain.InterpreterProviderPort

	// Clock supplies the time used for compilation metrics. Nil uses the real clock.
	Clock clock.Clock

	// I18nService provides translation support.
	I18nService i18n_domain.Service

	// SandboxFactory creates sandboxes for filesystem access.
	SandboxFactory safedisk.Factory

	// PathsConfig holds the resolved path settings for the generator.
	PathsConfig generator_domain.GeneratorPathsConfig

	// I18nDefaultLocale specifies the default locale for internationalisation.
	I18nDefaultLocale string

	// ModuleName identifies the Go module being processed.
	ModuleName string

	// ProjectRoot specifies the root directory of the project.
	ProjectRoot string

	// JITCompileTimeout bounds one JIT compilation. Zero or less uses a generous default.
	JITCompileTimeout time.Duration
}

// orchestratorBuildState holds the products of a completed full build, installed into the
// orchestrator in one step.
type orchestratorBuildState struct {
	// pool is the interpreter pool the build compiled with.
	pool templater_domain.InterpreterPoolPort

	// progCache maps relative paths to the linked page entries.
	progCache map[string]*templater_adapters.PageEntry

	// manifest is the build manifest.
	manifest *generator_dto.Manifest

	// reverseDepsMap maps component paths to their dependents.
	reverseDepsMap map[string][]string

	// artefactByPackagePath maps package paths to generated artefacts.
	artefactByPackagePath map[string]*generator_dto.GeneratedArtefact

	// poolGeneration is the fresh-pool request count the build satisfied.
	poolGeneration uint64
}

// NewInterpretedBuildOrchestrator creates a new orchestrator for building interpreted
// runners.
//
// Takes deps (InterpretedBuildOrchestratorDeps) which provides all required dependencies
// for the orchestrator.
//
// Returns *InterpretedBuildOrchestrator which is ready for use.
func NewInterpretedBuildOrchestrator(
	deps InterpretedBuildOrchestratorDeps,
) *InterpretedBuildOrchestrator {
	jitCompileTimeout := deps.JITCompileTimeout
	if jitCompileTimeout <= 0 {
		jitCompileTimeout = defaultJITCompileTimeout
	}
	orchestratorClock := deps.Clock
	if orchestratorClock == nil {
		orchestratorClock = clock.RealClock()
	}
	return &InterpretedBuildOrchestrator{
		compileGroup:          singleflight.Group{},
		i18nService:           deps.I18nService,
		cachedManifest:        nil,
		progCache:             make(map[string]*templater_adapters.PageEntry),
		dirtyCodeCache:        make(map[string][]byte),
		reverseDepsMap:        make(map[string][]string),
		interpreterPool:       deps.InterpreterPool,
		interpreterProvider:   deps.InterpreterProvider,
		clock:                 orchestratorClock,
		artefactByPackagePath: make(map[string]*generator_dto.GeneratedArtefact),
		sandboxFactory:        deps.SandboxFactory,
		pathsConfig:           deps.PathsConfig,
		i18nDefaultLocale:     deps.I18nDefaultLocale,
		projectRoot:           deps.ProjectRoot,
		moduleName:            deps.ModuleName,
		jitCompileTimeout:     jitCompileTimeout,
		freshPoolRequested:    0,
		freshPoolBuilt:        0,
		hostImports:           nil,
		packageStateWarned:    sync.Map{},
		hostImportsOnce:       sync.Once{},
		stateLock:             sync.RWMutex{},
	}
}

// BuildRunner compiles build artefacts and constructs an interpreted manifest runner.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the annotated
// project artefacts to compile.
//
// Returns templater_domain.ManifestRunnerPort which is the configured runner ready for
// template execution.
// Returns error when artefact extraction, sorting, or interpretation fails.
func (o *InterpretedBuildOrchestrator) BuildRunner(
	ctx context.Context,
	result *annotator_dto.ProjectAnnotationResult,
) (templater_domain.ManifestRunnerPort, error) {
	ctx, span, l := log.Span(ctx, "InterpretedBuildOrchestrator.BuildRunner")
	defer span.End()

	l.Internal("[JIT-BUILD] ========== Starting Interpreted Runner Build ==========")

	if o.isEmptyVirtualModule(result) {
		l.Internal("[JIT-BUILD] No components in virtual module, creating empty runner")
		return o.createEmptyRunner(), nil
	}

	artefacts, err := o.extractArtefacts(ctx, result)
	if err != nil {
		return nil, fmt.Errorf("extracting build artefacts: %w", err)
	}
	if len(artefacts) == 0 {
		l.Internal("[JIT-BUILD] No valid artefacts after filtering, creating empty runner")
		return o.createEmptyRunner(), nil
	}

	manifest, err := o.buildManifest(ctx, artefacts)
	if err != nil {
		return nil, fmt.Errorf("building manifest: %w", err)
	}

	l.Internal("[JIT-BUILD] Stage 1/2: Topologically sorting artefacts...")
	sortedArtefacts, err := o.topologicallySortArtefacts(artefacts)
	if err != nil {
		return nil, fmt.Errorf("sorting artefacts topologically: %w", err)
	}
	l.Internal("[JIT-BUILD] Artefacts sorted", logger_domain.Int("count", len(sortedArtefacts)))

	pool, generation, err := o.interpreterPoolForBuild(ctx)
	if err != nil {
		return nil, fmt.Errorf("preparing interpreter pool: %w", err)
	}

	progCache, err := o.interpretArtefacts(ctx, pool, sortedArtefacts, manifest)
	if err != nil {
		return nil, fmt.Errorf("interpreting artefacts: %w", err)
	}

	o.updateOrchestratorState(orchestratorBuildState{
		progCache:             progCache,
		manifest:              manifest,
		reverseDepsMap:        o.buildReverseDependencyMap(sortedArtefacts),
		artefactByPackagePath: o.buildArtefactLookupMap(sortedArtefacts),
		pool:                  pool,
		poolGeneration:        generation,
	})

	l.Internal("[JIT-BUILD] ========== Interpreted Runner Build Complete ==========",
		logger_domain.Int("cached_entries", len(progCache)))

	return templater_adapters.NewInterpretedManifestRunner(o.i18nService, progCache, o, o.getDefaultLocale()), nil
}

// MarkDirty caches changed source and marks components and their dependents for later JIT
// compilation.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the annotation
// results for changed files.
//
// Returns error when artefact extraction or manifest building fails.
//
// Concurrency: The stateLock mutex protects shared state.
func (o *InterpretedBuildOrchestrator) MarkDirty(
	ctx context.Context,
	result *annotator_dto.ProjectAnnotationResult,
) error {
	ctx, span, l := log.Span(ctx, "InterpretedBuildOrchestrator.MarkDirty")
	defer span.End()

	l.Internal("[JIT-MARK-DIRTY] ========== Marking Components Dirty ==========")

	artefacts, err := o.extractMarkDirtyArtefacts(ctx, result)
	if err != nil {
		return fmt.Errorf("extracting mark-dirty artefacts: %w", err)
	}
	if artefacts == nil {
		return nil
	}

	if !resultCoversVirtualModule(result) {
		l.Internal("[JIT-MARK-DIRTY] Result covers part of the project; merging into the cached manifest",
			logger_domain.Int("annotated", len(result.ComponentResults)),
			logger_domain.Int("components", len(result.VirtualModule.ComponentsByHash)))
		return o.markArtefactsDirty(ctx, artefacts)
	}

	manifest, err := o.buildManifest(ctx, artefacts)
	if err != nil {
		return fmt.Errorf("building manifest for mark-dirty: %w", err)
	}

	o.stateLock.Lock()
	defer o.stateLock.Unlock()

	o.cachedManifest = manifest

	o.updateArtefactLookup(artefacts)

	o.rebuildReverseDependencyMapFromState()

	directlyChanged, allDirty := o.markDirectlyChangedComponents(ctx, artefacts)
	o.propagateDirtyFlags(ctx, directlyChanged, allDirty)

	l.Internal("[JIT-MARK-DIRTY] ========== Dirty Marking Complete ==========",
		logger_domain.Int("directly_changed", len(directlyChanged)),
		logger_domain.Int("total_dirty", len(allDirty)),
		logger_domain.Int("dirty_code_stored", len(o.dirtyCodeCache)))

	return nil
}

// IsInitialised returns true if the orchestrator has completed an initial full build.
//
// Returns bool which is true when the interpreter pool exists, the program cache is
// populated and no fresh interpreter pool is pending.
//
// Concurrency: Safe for concurrent use. Uses a read lock to access internal state.
func (o *InterpretedBuildOrchestrator) IsInitialised() bool {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	return o.interpreterPool != nil && len(o.progCache) > 0 && o.freshPoolBuilt == o.freshPoolRequested
}

// InvalidateUserPackages requests a full build with a fresh interpreter pool.
//
// The interpreter cannot unregister a compiled package, and every clone the pool hands
// out shares one symbol registry, so an edited user package would otherwise keep its old
// compiled form. Requests use the current pool until that build succeeds.
//
// Concurrency: Acquires stateLock.
func (o *InterpretedBuildOrchestrator) InvalidateUserPackages() {
	o.stateLock.Lock()
	defer o.stateLock.Unlock()
	o.freshPoolRequested++
}

// GetCachedEntry retrieves a compiled page entry from the cache.
//
// Takes relPath (string) which specifies the relative path to look up.
//
// Returns *templater_adapters.PageEntry which is the cached entry if found.
// Returns bool which indicates whether the entry was present in the cache.
//
// Concurrency: A read lock protects access to the cached entry.
func (o *InterpretedBuildOrchestrator) GetCachedEntry(relPath string) (*templater_adapters.PageEntry, bool) {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	entry, found := o.progCache[relPath]
	return entry, found
}

// GetAllCachedKeys returns all keys in the prog cache.
//
// Returns []string which contains all cached program keys.
//
// Concurrency: A read lock protects access to the cached keys.
func (o *InterpretedBuildOrchestrator) GetAllCachedKeys() []string {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	return slices.Collect(maps.Keys(o.progCache))
}

// JITCompile compiles a dirty component and its dependencies on demand.
//
// Requests for the same component share one compilation under the JIT timeout. Each
// caller can stop waiting when its own context ends.
//
// Takes relPath (string) which specifies the relative path of the component to compile.
//
// Returns error when compilation fails or ctx ends before it completes.
func (o *InterpretedBuildOrchestrator) JITCompile(
	ctx context.Context,
	relPath string,
) error {
	ctx, span, _ := log.Span(ctx, "InterpretedBuildOrchestrator.JITCompile",
		logger_domain.String(fieldPath, relPath))
	defer span.End()

	detachedCtx := daemon_dto.DetachRequestContext(ctx)
	results := o.compileGroup.DoChan(relPath, func() (any, error) {
		compileCtx, cancel := context.WithTimeoutCause(detachedCtx, o.jitCompileTimeout, errJITCompileTimeout)
		defer cancel()
		return nil, goroutine.SafeCall(compileCtx, "lifecycle.JITCompile", func() error {
			return o.executeJITCompilation(compileCtx, relPath)
		})
	})

	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting for JIT compilation of %q: %w", relPath, context.Cause(ctx))
	case result := <-results:
		if result.Err != nil {
			return fmt.Errorf("JIT compiling %q: %w", relPath, result.Err)
		}
		return nil
	}
}

// GetAffectedComponents returns all component paths that transitively depend on the given
// component.
//
// Takes relPath (string) which is the relative path of the changed component.
//
// Returns []string which contains the relative paths of all transitively dependent
// components, not including relPath itself.
//
// Concurrency: Safe for concurrent use. Acquires a read lock on the state.
func (o *InterpretedBuildOrchestrator) GetAffectedComponents(relPath string) []string {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()

	visited := make(map[string]bool)
	queue := []string{relPath}
	var affected []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range o.reverseDepsMap[current] {
			if visited[dep] {
				continue
			}
			visited[dep] = true
			affected = append(affected, dep)
			queue = append(queue, dep)
		}
	}

	return affected
}

// MarkComponentsDirty merges changed components into the cached manifest and marks them
// for recompilation.
//
// The delegated update protects shared state with stateLock.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the annotation
// results for changed files.
//
// Returns error when artefact extraction or manifest building fails.
func (o *InterpretedBuildOrchestrator) MarkComponentsDirty(
	ctx context.Context,
	result *annotator_dto.ProjectAnnotationResult,
) error {
	ctx, span, l := log.Span(ctx, "InterpretedBuildOrchestrator.MarkComponentsDirty")
	defer span.End()

	l.Internal("[JIT-MARK-DIRTY] ========== Marking Components Dirty (targeted) ==========")

	artefacts, err := o.extractMarkDirtyArtefacts(ctx, result)
	if err != nil {
		return fmt.Errorf("extracting mark-dirty artefacts: %w", err)
	}
	if artefacts == nil {
		return nil
	}

	return o.markArtefactsDirty(ctx, artefacts)
}

// resultCoversVirtualModule reports whether a build result annotated every component of
// its virtual module, as a full build does, rather than a subset.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which is the result to check.
//
// Returns bool which is true when no component of the virtual module is missing a result.
func resultCoversVirtualModule(result *annotator_dto.ProjectAnnotationResult) bool {
	if result == nil || result.VirtualModule == nil {
		return true
	}
	return len(result.ComponentResults) >= len(result.VirtualModule.ComponentsByHash)
}

// ProactiveRecompile JIT-compiles all components currently in the dirty code cache.
//
// An error compiling one component is logged and does not stop the batch. When ctx ends
// the batch stops quietly, leaving the remaining components dirty for the next request.
//
// Returns error only when a systemic failure prevents all compilation.
//
// Concurrency: Safe for concurrent use. Reads dirtyCodeCache keys under a read lock, then
// calls JITCompile which acquires its own locks.
func (o *InterpretedBuildOrchestrator) ProactiveRecompile(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "InterpretedBuildOrchestrator.ProactiveRecompile")
	defer span.End()

	o.stateLock.RLock()
	dirtyPaths := slices.Collect(maps.Keys(o.dirtyCodeCache))
	o.stateLock.RUnlock()

	if len(dirtyPaths) == 0 {
		l.Trace("[JIT-PROACTIVE] No dirty components to compile")
		return nil
	}

	l.Internal("[JIT-PROACTIVE] Starting proactive compilation",
		logger_domain.Int("dirty_count", len(dirtyPaths)))

	var compiledCount int
	for _, relPath := range dirtyPaths {
		if ctx.Err() != nil {
			l.Internal("[JIT-PROACTIVE] Context ended, stopping proactive compilation",
				logger_domain.Int("compiled", compiledCount),
				logger_domain.Int("total", len(dirtyPaths)))
			return nil
		}
		if err := o.JITCompile(ctx, relPath); err != nil {
			if ctx.Err() != nil {
				continue
			}
			l.Error("[JIT-PROACTIVE] Failed to compile component",
				logger_domain.String(fieldPath, relPath),
				logger_domain.Error(err))
			continue
		}
		compiledCount++
	}

	l.Internal("[JIT-PROACTIVE] Proactive compilation complete",
		logger_domain.Int("compiled", compiledCount),
		logger_domain.Int("total", len(dirtyPaths)))

	return nil
}

// RemoveComponent removes a component from the program, source, manifest, and artefact
// caches and refreshes reverse dependencies.
//
// Takes relPath (string) which is the project-relative source path of the removed
// component (e.g. "pages/old.pk"). For a regular page this is the program cache key.
//
// Concurrency: The stateLock mutex protects cache mutations.
func (o *InterpretedBuildOrchestrator) RemoveComponent(ctx context.Context, relPath string) {
	_, l := logger_domain.From(ctx, log)
	relPath = filepath.ToSlash(relPath)

	o.stateLock.Lock()
	defer o.stateLock.Unlock()

	delete(o.progCache, relPath)
	delete(o.dirtyCodeCache, relPath)

	if o.cachedManifest != nil {
		delete(o.cachedManifest.Pages, relPath)
		delete(o.cachedManifest.Partials, relPath)
		delete(o.cachedManifest.Emails, relPath)
		delete(o.cachedManifest.Pdfs, relPath)
		delete(o.cachedManifest.ErrorPages, relPath)
	}

	for packagePath, artefact := range o.artefactByPackagePath {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}
		artefactRelPath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			l.Error("Failed to compute relative path for removed component",
				logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
				logger_domain.String(fieldProjectRoot, o.projectRoot),
				logger_domain.Error(err))
			continue
		}
		if filepath.ToSlash(artefactRelPath) == relPath {
			delete(o.artefactByPackagePath, packagePath)
		}
	}

	o.rebuildReverseDependencyMapFromState()
}

// markArtefactsDirty merges partial-build artefacts and marks them and their dependents
// for recompilation.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which are the rebuilt components.
//
// Returns error when the manifest cannot be built.
//
// Concurrency: The stateLock mutex protects the merge and dirty flags.
func (o *InterpretedBuildOrchestrator) markArtefactsDirty(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) error {
	ctx, l := logger_domain.From(ctx, log)

	o.stateLock.Lock()
	defer o.stateLock.Unlock()

	knownArtefacts := slices.Collect(maps.Values(o.artefactByPackagePath))
	manifest, err := o.buildManifestWithContext(ctx, artefacts, knownArtefacts)
	if err != nil {
		return fmt.Errorf("building manifest for targeted mark-dirty: %w", err)
	}

	o.mergeManifest(manifest)

	o.updateArtefactLookup(artefacts)

	o.rebuildReverseDependencyMapFromState()

	directlyChanged, allDirty := o.markDirectlyChangedComponents(ctx, artefacts)
	o.propagateDirtyFlags(ctx, directlyChanged, allDirty)

	l.Internal("[JIT-MARK-DIRTY] ========== Targeted Dirty Marking Complete ==========",
		logger_domain.Int("directly_changed", len(directlyChanged)),
		logger_domain.Int("total_dirty", len(allDirty)),
		logger_domain.Int("dirty_code_stored", len(o.dirtyCodeCache)))

	return nil
}

// isEmptyVirtualModule checks if the virtual module has no components.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the module to
// check.
//
// Returns bool which is true when the virtual module is nil or has no components.
func (*InterpretedBuildOrchestrator) isEmptyVirtualModule(result *annotator_dto.ProjectAnnotationResult) bool {
	return result.VirtualModule == nil || len(result.VirtualModule.ComponentsByHash) == 0
}

// createEmptyRunner creates a runner with no cached entries.
//
// Returns templater_domain.ManifestRunnerPort which is a runner with an empty page cache
// and no manifest.
func (o *InterpretedBuildOrchestrator) createEmptyRunner() templater_domain.ManifestRunnerPort {
	return templater_adapters.NewInterpretedManifestRunner(
		o.i18nService,
		make(map[string]*templater_adapters.PageEntry),
		nil,
		o.getDefaultLocale(),
	)
}

// getDefaultLocale returns the default locale from the configuration.
//
// Returns string which is the configured default locale, or "en" if none is set.
func (o *InterpretedBuildOrchestrator) getDefaultLocale() string {
	if o.i18nDefaultLocale != "" {
		return o.i18nDefaultLocale
	}
	return "en"
}

// extractArtefacts gets and checks the artefacts from a build result.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the build output with
// the generated artefacts.
//
// Returns []*generator_dto.GeneratedArtefact which holds the checked artefacts ready for
// use.
// Returns error when FinalGeneratedArtefacts is nil, has the wrong type, or is empty.
func (*InterpretedBuildOrchestrator) extractArtefacts(
	ctx context.Context,
	result *annotator_dto.ProjectAnnotationResult,
) ([]*generator_dto.GeneratedArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	if result.FinalGeneratedArtefacts == nil {
		l.Error("[JIT-BUILD] CRITICAL: FinalGeneratedArtefacts is empty. The coordinator did not populate this field.")
		return nil, errors.New("build result missing FinalGeneratedArtefacts - coordinator pipeline may be broken")
	}

	artefacts, ok := result.FinalGeneratedArtefacts.([]*generator_dto.GeneratedArtefact)
	if !ok {
		l.Error("[JIT-BUILD] CRITICAL: FinalGeneratedArtefacts has wrong type")
		return nil, errors.New("finalGeneratedArtefacts type assertion failed")
	}

	if len(artefacts) == 0 {
		l.Warn("[JIT-BUILD] Build result contained no generated artefacts")
		l.Error("[JIT-BUILD] CRITICAL: FinalGeneratedArtefacts is empty. The coordinator did not populate this field.")
		return nil, errors.New("build result missing FinalGeneratedArtefacts - coordinator pipeline may be broken")
	}

	l.Internal("[JIT-BUILD] Using final generated artefacts from build result",
		logger_domain.Int("artefact_count", len(artefacts)))

	return artefacts, nil
}

// buildManifest creates a manifest from the given artefacts.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the generated items
// to include in the manifest.
//
// Returns *generator_dto.Manifest which contains the organised pages, partials, and
// emails.
// Returns error when the manifest builder fails to process the artefacts.
func (o *InterpretedBuildOrchestrator) buildManifest(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) (*generator_dto.Manifest, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Building manifest from artefacts...")
	manifestBuilder := generator_domain.NewManifestBuilder(o.pathsConfig, o.i18nDefaultLocale, o.projectRoot)
	manifest, err := manifestBuilder.Build(artefacts)
	if err != nil {
		return nil, fmt.Errorf("failed to build manifest from artefacts: %w", err)
	}
	l.Internal("[JIT-BUILD] Manifest built",
		logger_domain.Int("pages", len(manifest.Pages)),
		logger_domain.Int("partials", len(manifest.Partials)),
		logger_domain.Int("emails", len(manifest.Emails)))
	return manifest, nil
}

// buildManifestWithContext builds entries against all known artefacts.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which are the artefacts to create
// entries for.
// Takes contextArtefacts ([]*generator_dto.GeneratedArtefact) which are the project's
// other known artefacts.
//
// Returns *generator_dto.Manifest which contains entries for artefacts only.
// Returns error when the manifest builder fails to process the artefacts.
func (o *InterpretedBuildOrchestrator) buildManifestWithContext(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
	contextArtefacts []*generator_dto.GeneratedArtefact,
) (*generator_dto.Manifest, error) {
	_, l := logger_domain.From(ctx, log)
	manifestBuilder := generator_domain.NewManifestBuilder(o.pathsConfig, o.i18nDefaultLocale, o.projectRoot)
	manifest, err := manifestBuilder.BuildWithContext(artefacts, contextArtefacts)
	if err != nil {
		return nil, fmt.Errorf("failed to build manifest from artefacts: %w", err)
	}
	l.Internal("[JIT-BUILD] Partial manifest built",
		logger_domain.Int("pages", len(manifest.Pages)),
		logger_domain.Int("partials", len(manifest.Partials)),
		logger_domain.Int("context_artefacts", len(contextArtefacts)))
	return manifest, nil
}

// interpreterPoolForBuild selects the current pool or creates a fresh pool for a pending
// rebuild.
//
// Returns templater_domain.InterpreterPoolPort which is the pool to compile with.
// Returns uint64 which is the fresh-pool request count the build satisfies.
// Returns error when no pool is configured or a fresh pool's modules fail to load.
//
// Concurrency: State is read under a read lock. Fresh pools are built after releasing it.
func (o *InterpretedBuildOrchestrator) interpreterPoolForBuild(ctx context.Context) (templater_domain.InterpreterPoolPort, uint64, error) {
	ctx, l := logger_domain.From(ctx, log)

	o.stateLock.RLock()
	pool := o.interpreterPool
	requested := o.freshPoolRequested
	freshNeeded := requested != o.freshPoolBuilt
	o.stateLock.RUnlock()

	if !freshNeeded || o.interpreterProvider == nil {
		if freshNeeded {
			l.Warn("User Go packages changed but no interpreter provider is configured; restart to pick up the changes")
		}
		if pool == nil {
			return nil, 0, errNoInterpreterPool
		}
		return pool, requested, nil
	}

	l.Internal("[JIT-BUILD] User Go packages changed, building a fresh interpreter pool")
	fresh := o.interpreterProvider.NewInterpreterPool()
	if err := fresh.LoadModules(ctx); err != nil {
		return nil, 0, fmt.Errorf("loading modules into a fresh interpreter pool: %w", err)
	}
	return fresh, requested, nil
}

// interpretArtefacts compiles all artefacts as a single program and builds the program
// cache.
//
// Takes pool (templater_domain.InterpreterPoolPort) which provides the interpreter.
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides the artefacts
// to compile in dependency order.
// Takes manifest (*generator_dto.Manifest) which contains the build manifest.
//
// Returns map[string]*templater_adapters.PageEntry which maps relative paths to their
// linked page entries.
// Returns error when the interpreter cannot be obtained from the pool, or when
// compilation or linking fails.
func (o *InterpretedBuildOrchestrator) interpretArtefacts(
	ctx context.Context,
	pool templater_domain.InterpreterPoolPort,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
) (map[string]*templater_adapters.PageEntry, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Stage 2/2: Compiling all artefacts in a fresh interpreter...")

	interpreter, err := getInterpreterFromPool(pool)
	if err != nil {
		return nil, fmt.Errorf("getting interpreter from pool: %w", err)
	}

	packages, components := o.collectArtefactSources(ctx, sortedArtefacts)
	if len(packages) == 0 {
		return make(map[string]*templater_adapters.PageEntry), nil
	}

	o.discoverUserPackages(ctx, packages, interpreter)

	l.Internal("[JIT-BUILD] Compiling all packages in batch",
		logger_domain.Int("package_count", len(packages)))

	if err := o.compileAndExecute(ctx, interpreter, packages); err != nil {
		return nil, fmt.Errorf("batch compilation failed: %w", err)
	}

	l.Internal("[JIT-BUILD] Batch compilation complete, linking functions from registry")

	return o.linkAllArtefacts(ctx, components, manifest)
}

// compileAndExecute compiles packages as one program in the interpreter and runs their
// init functions, recording the interpreted-mode compilation metrics.
//
// Takes interpreter (templater_domain.InterpreterPort) which compiles and runs the
// program.
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
//
// Returns error when compilation or init execution fails.
func (o *InterpretedBuildOrchestrator) compileAndExecute(
	ctx context.Context,
	interpreter templater_domain.InterpreterPort,
	packages map[string]map[string]string,
) error {
	templater_adapters.InterpretedManifestRunnerCompilationCount.Add(ctx, 1)
	startTime := o.clock.Now()
	defer func() {
		elapsed := o.clock.Now().Sub(startTime)
		templater_adapters.InterpretedManifestRunnerCompilationDuration.Record(ctx, float64(elapsed)/float64(time.Millisecond))
	}()

	if err := interpreter.CompileAndExecute(ctx, o.moduleName, packages); err != nil {
		templater_adapters.InterpretedManifestRunnerCompilationErrorCount.Add(ctx, 1)
		return err
	}
	return nil
}

// collectArtefactSources collects generated source code from all artefacts into the
// format expected by CompileProgram.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which are the artefacts
// whose source code is collected.
//
// Returns map[string]map[string]string mapping relative package paths to
// filename-to-source maps.
// Returns map[string]*annotator_dto.VirtualComponent mapping relative paths to their
// virtual components for later linking.
func (o *InterpretedBuildOrchestrator) collectArtefactSources(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) (map[string]map[string]string, map[string]*annotator_dto.VirtualComponent) {
	_, l := logger_domain.From(ctx, log)

	packages := make(map[string]map[string]string, len(sortedArtefacts))
	components := make(map[string]*annotator_dto.VirtualComponent, len(sortedArtefacts))

	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			l.Error("Failed to compute relative path for batch compilation",
				logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
				logger_domain.Error(err))
			continue
		}
		relativePath = filepath.ToSlash(relativePath)

		pkgRelPath := strings.TrimPrefix(component.CanonicalGoPackagePath, o.moduleName+"/")

		packages[pkgRelPath] = map[string]string{
			"generated.go": string(artefact.Content),
		}
		components[relativePath] = component
	}

	return packages, components
}

// discoverUserPackages follows local imports transitively, skipping packages already
// registered with the interpreter.
//
// A user package the server binary also links is warned about when it declares exported
// variables, as the host and the interpreter hold separate copies of them.
//
// Takes packages (map[string]map[string]string) which is the mutable map to populate with
// discovered package sources.
// Takes interpreter (templater_domain.InterpreterPort) which checks whether a package is
// already registered.
func (o *InterpretedBuildOrchestrator) discoverUserPackages(
	ctx context.Context,
	packages map[string]map[string]string,
	interpreter templater_domain.InterpreterPort,
) {
	ctx, l := logger_domain.From(ctx, log)

	var sandbox safedisk.Sandbox
	var sandboxErr error
	if o.sandboxFactory != nil {
		sandbox, sandboxErr = o.sandboxFactory.Create("interp-project-read", o.projectRoot, safedisk.ModeReadOnly)
	} else {
		sandbox, sandboxErr = safedisk.NewSandbox(o.projectRoot, safedisk.ModeReadOnly)
	}
	if sandboxErr != nil {
		l.Error("Failed to create sandbox for user package discovery", logger_domain.Error(sandboxErr))
		return
	}
	defer func() { _ = sandbox.Close() }()

	modulePrefix := o.moduleName + "/"
	generated := maps.Clone(packages)

	pending := o.collectLocalImports(packages, modulePrefix)

	for len(pending) > 0 {
		var nextPending []string

		for _, importPath := range pending {
			discovered := o.resolveImportedPackage(
				ctx, importPath, modulePrefix, packages, sandbox, interpreter,
			)
			nextPending = append(nextPending, discovered...)
		}

		pending = nextPending
	}

	userPackages := make(map[string]map[string]string)
	for relativeDirectory, files := range packages {
		if _, isGenerated := generated[relativeDirectory]; !isGenerated {
			userPackages[relativeDirectory] = files
		}
	}
	o.warnSeparatePackageState(ctx, sandbox, userPackages, hostMainPackagePath())
}

// resolveImportedPackage attempts to resolve a single local import path into a user
// package.
//
// Takes importPath (string) which is the fully qualified Go import path to resolve.
// Takes modulePrefix (string) which is the module name followed by a slash, used to
// identify local imports.
// Takes packages (map[string]map[string]string) which is the mutable map to populate with
// discovered sources.
// Takes sandbox (safedisk.Sandbox) which provides safe filesystem access for reading user
// source files.
// Takes interpreter (templater_domain.InterpreterPort) which checks whether a package is
// already registered.
//
// Returns []string containing any transitive local import paths discovered in the
// resolved package.
func (o *InterpretedBuildOrchestrator) resolveImportedPackage(
	ctx context.Context,
	importPath string,
	modulePrefix string,
	packages map[string]map[string]string,
	sandbox safedisk.Sandbox,
	interpreter templater_domain.InterpreterPort,
) []string {
	ctx, l := logger_domain.From(ctx, log)
	if interpreter.HasRegisteredPackage(importPath) {
		return nil
	}

	relativeDirectory := strings.TrimPrefix(importPath, modulePrefix)

	if _, exists := packages[relativeDirectory]; exists {
		return nil
	}

	goFiles := o.readUserGoFiles(ctx, sandbox, relativeDirectory, interpreter)
	if len(goFiles) == 0 {
		return nil
	}

	packages[relativeDirectory] = goFiles
	l.Internal("[JIT-BUILD] Discovered user package",
		logger_domain.String(fieldPackagePath, relativeDirectory),
		logger_domain.Int("file_count", len(goFiles)))

	var transitive []string
	for _, source := range goFiles {
		transitive = append(transitive, parseLocalImportPaths(source, modulePrefix)...)
	}

	return transitive
}

// collectLocalImports scans all source files in the packages map and returns import paths
// that belong to the current module.
//
// Takes packages (map[string]map[string]string) which maps relative package paths to
// their filename-to-source maps.
// Takes modulePrefix (string) which is used to filter imports to only those belonging to
// the current module.
//
// Returns []string containing the deduplicated local import paths found across all source
// files.
func (*InterpretedBuildOrchestrator) collectLocalImports(
	packages map[string]map[string]string,
	modulePrefix string,
) []string {
	var localImports []string

	for _, sources := range packages {
		for _, source := range sources {
			localImports = append(localImports, parseLocalImportPaths(source, modulePrefix)...)
		}
	}

	return localImports
}

// readUserGoFiles reads non-test Go files accepted by CompilesFile.
//
// Takes sandbox (safedisk.Sandbox) which provides safe filesystem access scoped to the
// project root.
// Takes relDir (string) which is the relative directory path to read Go files from.
// Takes interpreter (templater_domain.InterpreterPort) which decides each file.
//
// Returns map[string]string mapping filenames to their source content, or nil when the
// directory cannot be read.
func (*InterpretedBuildOrchestrator) readUserGoFiles(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	relDir string,
	interpreter templater_domain.InterpreterPort,
) map[string]string {
	_, l := logger_domain.From(ctx, log)
	entries, readErr := sandbox.ReadDir(relDir)
	if readErr != nil {
		return nil
	}
	goFiles := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		content, readFileErr := sandbox.ReadFile(filepath.Join(relDir, e.Name()))
		if readFileErr != nil {
			l.Error("Failed to read user package file",
				logger_domain.String(fieldPath, filepath.Join(relDir, e.Name())),
				logger_domain.Error(readFileErr))
			continue
		}
		compiled, matchErr := interpreter.CompilesFile(e.Name(), content)
		if matchErr == nil && !compiled {
			l.Trace("[JIT-BUILD] Skipping user package file excluded from the interpreted build",
				logger_domain.String(fieldPath, filepath.Join(relDir, e.Name())))
			continue
		}
		goFiles[e.Name()] = string(content)
	}
	return goFiles
}

// linkAllArtefacts creates PageEntry objects for all components and links their
// registered functions from the global FunctionRegistry.
//
// Takes components (map[string]*annotator_dto.VirtualComponent) which maps relative paths
// to virtual components to link.
// Takes manifest (*generator_dto.Manifest) which provides build metadata for page entry
// creation.
//
// Returns map[string]*templater_adapters.PageEntry which maps relative paths to their
// fully linked page entries.
// Returns error when function linking fails for any component.
func (o *InterpretedBuildOrchestrator) linkAllArtefacts(
	ctx context.Context,
	components map[string]*annotator_dto.VirtualComponent,
	manifest *generator_dto.Manifest,
) (map[string]*templater_adapters.PageEntry, error) {
	progCache := make(map[string]*templater_adapters.PageEntry, len(components))

	for relativePath, component := range components {
		shortPackageName, err := extractPackageName(
			component.CanonicalGoPackagePath,
		)
		if err != nil {
			shortPackageName = component.HashedName
		}

		linkFn := func(entry *templater_adapters.PageEntry, comp *annotator_dto.VirtualComponent) error {
			return o.linkFunctionsFromRegistry(ctx, entry, comp, shortPackageName)
		}
		if err := o.populateProgCacheForComponent(ctx, manifest, component, relativePath, linkFn, progCache); err != nil {
			return nil, fmt.Errorf("linking functions from registry for %q: %w", relativePath, err)
		}
	}

	return progCache, nil
}

// buildReverseDependencyMap creates a map from component paths to the components that
// import them.
//
// Imports are resolved as the dependency sort resolves them. An ambiguous import records
// the importer under every component it could name, so a change to any of them marks it
// dirty.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides the build
// artefacts in dependency order.
//
// Returns map[string][]string which maps each component's project-relative path to the
// paths of the components that import it.
func (o *InterpretedBuildOrchestrator) buildReverseDependencyMap(
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) map[string][]string {
	resolver := newImportResolver(o.projectRoot, o.moduleName, slices.Values(sortedArtefacts))
	reverseDepsMap := make(map[string][]string)
	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		relativePath := componentRelativePath(o.projectRoot, component.Source.SourcePath)
		for _, pikoImport := range component.Source.PikoImports {
			for _, dependencyPath := range resolver.matches(pikoImport.Path) {
				if dependencyPath != relativePath {
					reverseDepsMap[dependencyPath] = append(reverseDepsMap[dependencyPath], relativePath)
				}
			}
		}
	}
	return reverseDepsMap
}

// rebuildReverseDependencyMapFromState rebuilds the reverse dependency map from the
// current artefact state.
//
// Callers must hold stateLock.
func (o *InterpretedBuildOrchestrator) rebuildReverseDependencyMapFromState() {
	allArtefacts := make([]*generator_dto.GeneratedArtefact, 0, len(o.artefactByPackagePath))
	for _, artefact := range o.artefactByPackagePath {
		allArtefacts = append(allArtefacts, artefact)
	}
	o.reverseDepsMap = o.buildReverseDependencyMap(allArtefacts)
}

// buildArtefactLookupMap builds a map from canonical package paths to artefacts.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides the artefacts
// to index by their main component's package path.
//
// Returns map[string]*generator_dto.GeneratedArtefact which maps canonical package paths
// to their matching artefacts.
func (*InterpretedBuildOrchestrator) buildArtefactLookupMap(
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) map[string]*generator_dto.GeneratedArtefact {
	artefactByPackagePath := make(map[string]*generator_dto.GeneratedArtefact)
	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component != nil {
			artefactByPackagePath[component.CanonicalGoPackagePath] = artefact
		}
	}
	return artefactByPackagePath
}

// updateOrchestratorState updates the orchestrator's internal state after a build.
//
// Takes state (orchestratorBuildState) which holds the products of the completed build.
//
// Concurrency: The stateLock mutex protects field updates.
func (o *InterpretedBuildOrchestrator) updateOrchestratorState(state orchestratorBuildState) {
	o.stateLock.Lock()
	defer o.stateLock.Unlock()
	o.progCache = state.progCache
	o.cachedManifest = state.manifest
	o.reverseDepsMap = state.reverseDepsMap
	o.artefactByPackagePath = state.artefactByPackagePath
	o.dirtyCodeCache = make(map[string][]byte)
	o.interpreterPool = state.pool
	o.freshPoolBuilt = max(o.freshPoolBuilt, state.poolGeneration)
}

// extractMarkDirtyArtefacts gets artefacts from the annotation result for the MarkDirty
// operation.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the annotation result
// with its generated artefacts.
//
// Returns []*generator_dto.GeneratedArtefact which holds the extracted artefacts, or nil
// if there are none.
// Returns error when FinalGeneratedArtefacts has an unexpected type.
func (*InterpretedBuildOrchestrator) extractMarkDirtyArtefacts(
	ctx context.Context,
	result *annotator_dto.ProjectAnnotationResult,
) ([]*generator_dto.GeneratedArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	if result.FinalGeneratedArtefacts == nil {
		l.Warn("[JIT-MARK-DIRTY] No artefacts to mark dirty")
		return nil, nil
	}

	artefacts, ok := result.FinalGeneratedArtefacts.([]*generator_dto.GeneratedArtefact)
	if !ok {
		l.Error("[JIT-MARK-DIRTY] CRITICAL: FinalGeneratedArtefacts has wrong type")
		return nil, errors.New("finalGeneratedArtefacts type assertion failed")
	}

	if len(artefacts) == 0 {
		l.Warn("[JIT-MARK-DIRTY] No artefacts to mark dirty")
		return nil, nil
	}

	l.Internal("[JIT-MARK-DIRTY] Processing artefacts", logger_domain.Int("artefact_count", len(artefacts)))
	return artefacts, nil
}

// updateArtefactLookup registers the artefacts in the package-path lookup map.
//
// Callers must hold stateLock.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the generated
// artefacts to register.
func (o *InterpretedBuildOrchestrator) updateArtefactLookup(artefacts []*generator_dto.GeneratedArtefact) {
	for _, artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}
		o.artefactByPackagePath[component.CanonicalGoPackagePath] = artefact
	}
}

// markDirectlyChangedComponents marks components as dirty based on generated code.
//
// Callers must hold stateLock.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the generated code
// to process.
//
// Returns directlyChanged (map[string]bool) which tracks paths changed directly by this
// operation.
// Returns allDirty (map[string]bool) which tracks all paths marked as dirty.
func (o *InterpretedBuildOrchestrator) markDirectlyChangedComponents(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) (directlyChanged, allDirty map[string]bool) {
	ctx, l := logger_domain.From(ctx, log)
	directlyChanged = make(map[string]bool)
	allDirty = make(map[string]bool)

	for _, artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			l.Error("Failed to compute relative path",
				logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
				logger_domain.Error(err))
			continue
		}
		relativePath = filepath.ToSlash(relativePath)

		o.dirtyCodeCache[relativePath] = artefact.Content
		directlyChanged[relativePath] = true
		allDirty[relativePath] = true

		l.Trace("[JIT-MARK-DIRTY] Marked component dirty",
			logger_domain.String(fieldPath, relativePath),
			logger_domain.Int("code_size", len(artefact.Content)))
	}

	return directlyChanged, allDirty
}

// propagateDirtyFlags marks all dependents as dirty using the reverse dependency map.
//
// Callers must hold stateLock.
//
// Takes directlyChanged (map[string]bool) which contains paths that were changed and need
// their dependents marked dirty.
// Takes allDirty (map[string]bool) which collects all paths marked dirty, including those
// affected through other dependents.
func (o *InterpretedBuildOrchestrator) propagateDirtyFlags(
	ctx context.Context,
	directlyChanged, allDirty map[string]bool,
) {
	ctx, l := logger_domain.From(ctx, log)
	queue := slices.Collect(maps.Keys(directlyChanged))

	for len(queue) > 0 {
		currentPath := queue[0]
		queue = queue[1:]

		for _, dependent := range o.reverseDepsMap[currentPath] {
			if allDirty[dependent] {
				continue
			}

			allDirty[dependent] = true
			queue = append(queue, dependent)

			o.addDependentToDirtyCache(ctx, dependent, currentPath)

			l.Trace("[JIT-MARK-DIRTY] Propagated dirty flag to dependent",
				logger_domain.String("dependent", dependent),
				logger_domain.String("changed_partial", currentPath))
		}
	}
}

// addDependentToDirtyCache finds the dependent's artefact code and adds it to the dirty
// cache for later recompilation.
//
// Callers must hold stateLock.
//
// Takes dependent (string) which is the relative path of the file to add.
// Takes changedComponent (string) which identifies the component that changed.
func (o *InterpretedBuildOrchestrator) addDependentToDirtyCache(
	ctx context.Context,
	dependent, changedComponent string,
) {
	ctx, l := logger_domain.From(ctx, log)
	for _, artefact := range o.artefactByPackagePath {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		artefactRelativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			continue
		}
		artefactRelativePath = filepath.ToSlash(artefactRelativePath)

		if artefactRelativePath == dependent {
			o.dirtyCodeCache[dependent] = artefact.Content
			l.Trace("[JIT-MARK-DIRTY] Added dependent to dirty cache for recompilation",
				logger_domain.String("dependent", dependent),
				logger_domain.String("changed_component", changedComponent),
				logger_domain.Int("code_size", len(artefact.Content)))
			return
		}
	}
}

// mergeManifest merges entries from newManifest into the cached manifest without removing
// existing entries.
//
// Callers must hold stateLock.
//
// Takes newManifest (*generator_dto.Manifest) which contains the entries to merge.
func (o *InterpretedBuildOrchestrator) mergeManifest(newManifest *generator_dto.Manifest) {
	if o.cachedManifest == nil {
		o.cachedManifest = newManifest
		return
	}

	maps.Copy(o.cachedManifest.Pages, newManifest.Pages)
	maps.Copy(o.cachedManifest.Partials, newManifest.Partials)
	maps.Copy(o.cachedManifest.Emails, newManifest.Emails)
	if o.cachedManifest.Pdfs == nil {
		o.cachedManifest.Pdfs = make(map[string]generator_dto.ManifestPdfEntry, len(newManifest.Pdfs))
	}
	maps.Copy(o.cachedManifest.Pdfs, newManifest.Pdfs)
	maps.Copy(o.cachedManifest.ErrorPages, newManifest.ErrorPages)
	if o.cachedManifest.PartialScripts == nil {
		o.cachedManifest.PartialScripts = make(map[string]string, len(newManifest.PartialScripts))
	}
	maps.Copy(o.cachedManifest.PartialScripts, newManifest.PartialScripts)
}

// getInterpreterFromPool retrieves a pre-warmed interpreter from a pool.
//
// Takes pool (templater_domain.InterpreterPoolPort) which provides the interpreter.
//
// Returns templater_domain.InterpreterPort which is a ready-to-use interpreter instance.
// Returns error when the pool cannot provide an interpreter.
func getInterpreterFromPool(pool templater_domain.InterpreterPoolPort) (templater_domain.InterpreterPort, error) {
	interpreter, err := pool.Get()
	if err != nil {
		return nil, fmt.Errorf("retrieving interpreter from pool: %w", err)
	}
	return interpreter, nil
}

// parseLocalImportPaths extracts import paths from Go source code that match the given
// module prefix.
//
// Takes source (string) which is the Go source code to parse.
// Takes modulePrefix (string) which filters imports to only those belonging to the
// current module.
//
// Returns []string containing the matching import paths.
func parseLocalImportPaths(source string, modulePrefix string) []string {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "", source, parser.ImportsOnly)
	if err != nil {
		return nil
	}

	var result []string

	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if strings.HasPrefix(importPath, modulePrefix) {
			result = append(result, importPath)
		}
	}

	return result
}
