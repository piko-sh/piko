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
	"runtime"
	"slices"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// debugFilePermission is the permission mode for debug temp files.
	// Only the owner has read and write access.
	debugFilePermission = 0600

	// fieldAbsolutePath is the log field name for absolute file paths.
	fieldAbsolutePath = "absolutePath"

	// fieldPath is the log field name for recording file paths.
	fieldPath = "path"

	// fieldPackagePath is the log field name for Go package paths.
	fieldPackagePath = "pkg_path"

	// linkedFunctionCount is the number of functions linked to each component.
	linkedFunctionCount = 4
)

// InterpretedBuildOrchestrator handles the conversion of build artefacts into a
// runnable InterpretedManifestRunner. It implements
// InterpretedBuildOrchestrator and JITCompiler interfaces.
//
// This orchestrator uses a pre-warmed interpreter pool for performance. Instead
// of creating fresh interpreters with New()+Use(), it clones from a golden
// interpreter with pre-loaded stdlib symbols, then resets and returns after
// use.
//
// On file save (via MarkDirty), it only marks components as dirty without
// recompiling. On HTTP request (via JITCompile), it compiles dirty components
// just-in-time. This improves hot-reload performance by eliminating wasted
// compilation work.
type InterpretedBuildOrchestrator struct {
	// compileGroup stops repeated JIT compilations for the same path.
	compileGroup singleflight.Group

	// registryService provides access to the template registry for VFS operations.
	registryService registry_domain.RegistryService

	// i18nService provides translation and localisation for manifest runners.
	i18nService i18n_domain.Service

	// cachedManifest holds the manifest from the last build for JIT compilation.
	cachedManifest *generator_dto.Manifest

	// interpSemaphore limits how many interpreter runs can happen at the same
	// time.
	interpSemaphore chan struct{}

	// vfsAdapter is the virtual file system adapter for resolving imports.
	vfsAdapter *templater_adapters.RegistryVFSAdapter

	// progCache maps relative paths to compiled page entries.
	progCache map[string]*templater_adapters.PageEntry

	// dirtyCodeCache maps relative paths to their updated source code awaiting JIT
	// compilation.
	dirtyCodeCache map[string][]byte

	// reverseDepsMap maps component paths to the list of components that depend on
	// them.
	reverseDepsMap map[string][]string

	// interpreterPool holds reusable interpreters for template processing.
	interpreterPool templater_domain.InterpreterPoolPort

	// artefactByPackagePath maps Go package paths to their generated artefacts.
	artefactByPackagePath map[string]*generator_dto.GeneratedArtefact

	// sandboxFactory creates sandboxes for filesystem access within the
	// orchestrator.
	sandboxFactory safedisk.Factory

	// pathsConfig holds the resolved path settings for the generator.
	pathsConfig generator_domain.GeneratorPathsConfig

	// i18nDefaultLocale is the default locale for internationalisation.
	i18nDefaultLocale string

	// projectRoot is the absolute path to the project root folder.
	projectRoot string

	// moduleName is the Go module path used to resolve imports.
	moduleName string

	// stateLock guards access to orchestrator state fields for safe concurrent
	// use.
	stateLock sync.RWMutex

	// interpLock guards interpreter access during code interpretation and linking.
	interpLock sync.Mutex
}

// InterpretedBuildOrchestratorDeps holds the dependencies required to
// construct an InterpretedBuildOrchestrator.
type InterpretedBuildOrchestratorDeps struct {
	// InterpreterPool provides pooled interpreters for template execution.
	InterpreterPool templater_domain.InterpreterPoolPort

	// RegistryService provides access to the component registry.
	RegistryService registry_domain.RegistryService

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
}

// NewInterpretedBuildOrchestrator creates a new orchestrator for building
// interpreted runners.
//
// Takes deps (InterpretedBuildOrchestratorDeps) which provides all required
// dependencies for the orchestrator.
//
// Returns *InterpretedBuildOrchestrator which is ready for use with a
// concurrency limit based on runtime.NumCPU.
func NewInterpretedBuildOrchestrator(
	deps InterpretedBuildOrchestratorDeps,
) *InterpretedBuildOrchestrator {
	cpuCount := max(1, runtime.NumCPU())

	return &InterpretedBuildOrchestrator{
		compileGroup:          singleflight.Group{},
		registryService:       deps.RegistryService,
		i18nService:           deps.I18nService,
		cachedManifest:        nil,
		interpSemaphore:       make(chan struct{}, cpuCount),
		vfsAdapter:            nil,
		progCache:             make(map[string]*templater_adapters.PageEntry),
		dirtyCodeCache:        make(map[string][]byte),
		reverseDepsMap:        make(map[string][]string),
		interpreterPool:       deps.InterpreterPool,
		artefactByPackagePath: make(map[string]*generator_dto.GeneratedArtefact),
		sandboxFactory:        deps.SandboxFactory,
		pathsConfig:           deps.PathsConfig,
		i18nDefaultLocale:     deps.I18nDefaultLocale,
		projectRoot:           deps.ProjectRoot,
		moduleName:            deps.ModuleName,
		stateLock:             sync.RWMutex{},
		interpLock:            sync.Mutex{},
	}
}

// BuildRunner creates a new InterpretedManifestRunner from build artefacts.
// This method orchestrates the entire JIT compilation pipeline: creates a VFS
// adapter, sorts artefacts topologically, creates a fresh interpreter,
// interprets all artefacts in dependency order, creates a PageEntry cache, and
// returns a new runner with the populated cache.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the
// annotated project artefacts to compile.
//
// Returns templater_domain.ManifestRunnerPort which is the configured runner
// ready for template execution.
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

	vfsAdapter, err := o.createVFSAdapter(ctx, artefacts)
	if err != nil {
		return nil, fmt.Errorf("creating VFS adapter: %w", err)
	}

	l.Internal("[JIT-BUILD] Stage 3/4: Topologically sorting artefacts...")
	sortedArtefacts, err := o.topologicallySortArtefacts(artefacts)
	if err != nil {
		return nil, fmt.Errorf("sorting artefacts topologically: %w", err)
	}
	l.Internal("[JIT-BUILD] Artefacts sorted", logger_domain.Int("count", len(sortedArtefacts)))

	progCache, err := o.interpretArtefacts(ctx, sortedArtefacts, manifest, vfsAdapter)
	if err != nil {
		return nil, fmt.Errorf("interpreting artefacts: %w", err)
	}

	reverseDepsMap := o.buildReverseDependencyMap(sortedArtefacts)
	artefactByPackagePath := o.buildArtefactLookupMap(sortedArtefacts)

	o.updateOrchestratorState(vfsAdapter, progCache, manifest, reverseDepsMap, artefactByPackagePath)

	l.Internal("[JIT-BUILD] ========== Interpreted Runner Build Complete ==========",
		logger_domain.Int("cached_entries", len(progCache)))

	return templater_adapters.NewInterpretedManifestRunner(o.i18nService, progCache, o, o.getDefaultLocale()), nil
}

// MarkDirty is the fast-path method called on file save.
//
// It marks components as dirty without recompiling them, enabling sub-second
// hot-reload feedback. The method stores new Go code for each changed
// component in dirtyCodeCache, propagates dirty flags to all dependent
// components using reverseDepsMap, and returns immediately (~10-50ms) without
// compilation. Actual compilation happens later via JITCompile when a
// page is requested.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// annotation results for changed files.
//
// Returns error when artefact extraction or manifest building fails.
//
// Safe for concurrent use; protects shared state with stateLock.
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

	manifest, err := o.buildManifest(ctx, artefacts)
	if err != nil {
		return fmt.Errorf("building manifest for mark-dirty: %w", err)
	}

	o.stateLock.Lock()
	defer o.stateLock.Unlock()

	o.cachedManifest = manifest

	newPathMap := o.updateArtefactLookupAndVFS(ctx, artefacts)
	o.updateVFSAdapterIfNeeded(ctx, artefacts, newPathMap)

	directlyChanged, allDirty := o.markDirectlyChangedComponents(ctx, artefacts)
	o.propagateDirtyFlags(ctx, directlyChanged, allDirty)

	l.Internal("[JIT-MARK-DIRTY] ========== Dirty Marking Complete ==========",
		logger_domain.Int("directly_changed", len(directlyChanged)),
		logger_domain.Int("total_dirty", len(allDirty)),
		logger_domain.Int("dirty_code_stored", len(o.dirtyCodeCache)))

	return nil
}

// IsInitialised returns true if the orchestrator has completed an initial
// full build.
//
// This is used by the daemon service to distinguish between the initial build
// (which requires BuildRunner) and subsequent incremental builds (which use
// MarkDirty).
//
// Returns bool which is true when the interpreter pool exists and the program
// cache is populated.
//
// Safe for concurrent use. Uses a read lock to access internal state.
func (o *InterpretedBuildOrchestrator) IsInitialised() bool {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	return o.interpreterPool != nil && len(o.progCache) > 0
}

// GetCachedEntry retrieves a compiled page entry from the cache.
// This method is part of the JITCompiler interface used by
// InterpretedManifestRunner.
//
// Takes relPath (string) which specifies the relative path to look up.
//
// Returns *templater_adapters.PageEntry which is the cached entry if found.
// Returns bool which indicates whether the entry was present in the cache.
//
// Safe for concurrent use; protected by a read lock.
func (o *InterpretedBuildOrchestrator) GetCachedEntry(relPath string) (*templater_adapters.PageEntry, bool) {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	entry, found := o.progCache[relPath]
	return entry, found
}

// GetAllCachedKeys returns all keys in the prog cache.
// This method is part of the JITCompiler interface used by
// InterpretedManifestRunner.
//
// Returns []string which contains all cached program keys.
//
// Safe for concurrent use; acquires a read lock on the state.
func (o *InterpretedBuildOrchestrator) GetAllCachedKeys() []string {
	o.stateLock.RLock()
	defer o.stateLock.RUnlock()
	return slices.Collect(maps.Keys(o.progCache))
}

// JITCompile performs on-demand compilation when a dirty page is requested.
// It compiles only the specific requested component and its dependencies if
// needed.
//
// The method:
//  1. Checks if the component is dirty (has code in dirtyCodeCache)
//  2. If dirty, uses the long-lived interpreter to JIT-compile the new code
//  3. Updates the PageEntry in progCache with the new function pointers
//  4. Removes the component from dirtyCodeCache
//  5. Returns immediately after compiling just ONE component
//
// This means only visited pages pay the compilation cost.
//
// Takes relPath (string) which specifies the relative path of the component
// to compile.
//
// Returns error when compilation fails.
func (o *InterpretedBuildOrchestrator) JITCompile(
	ctx context.Context,
	relPath string,
) error {
	ctx, span, _ := log.Span(ctx, "InterpretedBuildOrchestrator.JITCompile",
		logger_domain.String(fieldPath, relPath))
	defer span.End()

	_, err, _ := o.compileGroup.Do(relPath, func() (any, error) {
		return nil, o.executeJITCompilation(ctx, relPath)
	})
	if err != nil {
		return fmt.Errorf("JIT compiling %q: %w", relPath, err)
	}
	return nil
}

// GetAffectedComponents returns all component paths that transitively depend
// on the given component. It performs a BFS traversal of the reverse
// dependency map starting from relPath.
//
// Takes relPath (string) which is the relative path of the changed component.
//
// Returns []string which contains the relative paths of all transitively
// dependent components, not including relPath itself.
//
// Safe for concurrent use; acquires a read lock on the state.
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

// MarkComponentsDirty marks changed components for recompilation, merging the
// partial build result into the existing manifest rather than replacing it.
// This is used by targeted rebuilds where the result only contains a subset
// of components.
//
// The method is identical to MarkDirty except that it merges the new manifest
// entries into the cached manifest instead of replacing it, preserving entries
// for components not included in the targeted build.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// annotation results for changed files.
//
// Returns error when artefact extraction or manifest building fails.
//
// Safe for concurrent use; protects shared state with stateLock.
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

	manifest, err := o.buildManifest(ctx, artefacts)
	if err != nil {
		return fmt.Errorf("building manifest for targeted mark-dirty: %w", err)
	}

	o.stateLock.Lock()
	defer o.stateLock.Unlock()

	o.mergeManifest(manifest)

	newPathMap := o.updateArtefactLookupAndVFS(ctx, artefacts)
	o.updateVFSAdapterIfNeeded(ctx, artefacts, newPathMap)

	o.rebuildReverseDependencyMapFromState()

	directlyChanged, allDirty := o.markDirectlyChangedComponents(ctx, artefacts)
	o.propagateDirtyFlags(ctx, directlyChanged, allDirty)

	l.Internal("[JIT-MARK-DIRTY] ========== Targeted Dirty Marking Complete ==========",
		logger_domain.Int("directly_changed", len(directlyChanged)),
		logger_domain.Int("total_dirty", len(allDirty)),
		logger_domain.Int("dirty_code_stored", len(o.dirtyCodeCache)))

	return nil
}

// ProactiveRecompile JIT-compiles all components currently in the dirty code
// cache. This runs compilation eagerly rather than waiting for an HTTP request
// to trigger it.
//
// Compilation errors for individual components are logged but do not stop the
// batch; all dirty components are attempted.
//
// Returns error only when a systemic failure prevents all compilation.
//
// Safe for concurrent use; reads dirtyCodeCache keys under a read lock, then
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
		if err := o.JITCompile(ctx, relPath); err != nil {
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

// isEmptyVirtualModule checks if the virtual module has no components.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// module to check.
//
// Returns bool which is true when the virtual module is nil or has no
// components.
func (*InterpretedBuildOrchestrator) isEmptyVirtualModule(result *annotator_dto.ProjectAnnotationResult) bool {
	return result.VirtualModule == nil || len(result.VirtualModule.ComponentsByHash) == 0
}

// createEmptyRunner creates a runner with no cached entries.
//
// Returns templater_domain.ManifestRunnerPort which is a runner with an empty
// page cache and no manifest.
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
// Returns string which is the configured default locale, or "en" if none is
// set.
func (o *InterpretedBuildOrchestrator) getDefaultLocale() string {
	if o.i18nDefaultLocale != "" {
		return o.i18nDefaultLocale
	}
	return "en"
}

// extractArtefacts gets and checks the artefacts from a build result.
//
// Takes ctx (context.Context) which carries the logger.
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the build
// output with the generated artefacts.
//
// Returns []*generator_dto.GeneratedArtefact which holds the checked artefacts
// ready for use.
// Returns error when FinalGeneratedArtefacts is nil, has the wrong type, or
// is empty.
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
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// generated items to include in the manifest.
//
// Returns *generator_dto.Manifest which contains the organised pages, partials,
// and emails.
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

// createVFSAdapter creates and configures the VFS adapter for import
// resolution.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// generated code artefacts to include in the virtual filesystem.
//
// Returns *templater_adapters.RegistryVFSAdapter which provides the configured
// virtual filesystem for the interpreter.
// Returns error when the path resolution fails or the adapter cannot be
// created.
func (o *InterpretedBuildOrchestrator) createVFSAdapter(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) (*templater_adapters.RegistryVFSAdapter, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Stage 1/4: Creating VFS adapter...")
	virtualGoPath := filepath.Join(o.projectRoot, ".piko-gopath")
	virtualGoPath, err := filepath.Abs(virtualGoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for virtual GOPATH: %w", err)
	}

	virtualGoRoot := getGOROOT()
	var projectSandbox safedisk.Sandbox
	if o.sandboxFactory != nil {
		projectSandbox, err = o.sandboxFactory.Create("interp-project-source", o.projectRoot, safedisk.ModeReadOnly)
	} else {
		projectSandbox, err = safedisk.NewSandbox(o.projectRoot, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create project sandbox: %w", err)
	}
	vfsAdapter, err := templater_adapters.NewRegistryVFSAdapter(
		ctx,
		o.registryService,
		virtualGoPath, virtualGoRoot,
		o.moduleName, projectSandbox,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VFS adapter: %w", err)
	}

	l.Internal("[JIT-BUILD] Stage 2/4: Building VFS map...")
	newPathMap := o.buildVFSPathMap(ctx, artefacts)
	l.Internal("[JIT-BUILD] VFS map built", logger_domain.Int("total_mappings", len(newPathMap)))

	vfsAdapter.UpdateMap(newPathMap)
	if err := vfsAdapter.UpdateFreshArtefacts(artefacts, o.projectRoot); err != nil {
		return nil, fmt.Errorf("failed to update VFS fresh artefacts cache: %w", err)
	}

	return vfsAdapter, nil
}

// buildVFSPathMap builds a map from canonical package paths to relative
// artefact paths.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// generated artefacts to map.
//
// Returns map[string]string which maps canonical Go package paths to their
// relative file paths from the project root.
func (o *InterpretedBuildOrchestrator) buildVFSPathMap(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) map[string]string {
	ctx, l := logger_domain.From(ctx, log)
	newPathMap := make(map[string]string)
	for _, artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			l.Error("Failed to compute relative path for VFS map",
				logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
				logger_domain.String("projectRoot", o.projectRoot),
				logger_domain.Error(err))
			continue
		}
		relativePath = filepath.ToSlash(relativePath)

		newPathMap[component.CanonicalGoPackagePath] = relativePath
		l.Trace("[JIT-BUILD] Added VFS mapping",
			logger_domain.String("canonical_path", component.CanonicalGoPackagePath),
			logger_domain.String("artefact_id", relativePath))
	}
	return newPathMap
}

// interpretArtefacts interprets all artefacts and builds the program cache.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides the
// artefacts to interpret in dependency order.
// Takes manifest (*generator_dto.Manifest) which contains the build manifest.
// Takes vfsAdapter (*templater_adapters.RegistryVFSAdapter) which provides the
// virtual filesystem for source code access.
//
// Returns map[string]*templater_adapters.PageEntry which maps relative paths to
// their interpreted page entries.
// Returns error when the interpreter cannot be obtained from the pool or when
// any artefact fails to interpret.
func (o *InterpretedBuildOrchestrator) interpretArtefacts(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
	vfsAdapter *templater_adapters.RegistryVFSAdapter,
) (map[string]*templater_adapters.PageEntry, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Stage 4/4: Getting interpreter from pool and interpreting artefacts...")

	freshInterpreter, err := o.getInterpreterFromPool()
	if err != nil {
		return nil, fmt.Errorf("getting interpreter from pool: %w", err)
	}

	if batchInterp, ok := freshInterpreter.(templater_domain.BatchInterpreterPort); ok {
		return o.interpretArtefactsBatch(ctx, batchInterp, sortedArtefacts, manifest)
	}

	defer o.returnInterpreterToPool(ctx, freshInterpreter)
	return o.interpretArtefactsIncremental(ctx, freshInterpreter, sortedArtefacts, manifest, vfsAdapter)
}

// interpretArtefactsBatch compiles all artefacts as a single program
// using the batch compilation path.
//
// This collects all generated source code, calls CompileAndExecute to
// compile and run init functions, then links the registered functions
// to page entries.
//
// Takes ctx (context.Context) which carries the logger and deadline.
// Takes batchInterp (templater_domain.BatchInterpreterPort) which
// executes the batch compilation.
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which
// are the artefacts to compile, in dependency order.
//
// Returns map[string]*PageEntry which maps relative paths
// to their linked page entries.
// Returns error when batch compilation or linking fails.
func (o *InterpretedBuildOrchestrator) interpretArtefactsBatch(
	ctx context.Context,
	batchInterp templater_domain.BatchInterpreterPort,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
) (map[string]*templater_adapters.PageEntry, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Using batch compilation path")

	packages, components := o.collectArtefactSources(ctx, sortedArtefacts)
	if len(packages) == 0 {
		return make(map[string]*templater_adapters.PageEntry), nil
	}

	o.discoverUserPackages(ctx, packages, batchInterp)

	l.Internal("[JIT-BUILD] Compiling all packages in batch",
		logger_domain.Int("package_count", len(packages)))

	if err := batchInterp.CompileAndExecute(ctx, o.moduleName, packages); err != nil {
		return nil, fmt.Errorf("batch compilation failed: %w", err)
	}

	l.Internal("[JIT-BUILD] Batch compilation complete, linking functions from registry")

	return o.linkAllArtefacts(ctx, components, manifest)
}

// collectArtefactSources collects generated source code from all
// artefacts into the format expected by CompileProgram.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which
// are the artefacts whose source code is collected.
//
// Returns map[string]map[string]string mapping relative package
// paths to filename-to-source maps.
// Returns map[string]*annotator_dto.VirtualComponent mapping
// relative paths to their virtual components for later linking.
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

// discoverUserPackages finds user-written Go packages that are
// actually imported by the generated code and adds them to the
// packages map for batch compilation.
//
// Only packages reachable through the import graph are included;
// unrelated project packages are ignored. Discovery is
// import-driven: import statements are parsed from the generated
// sources, filtered to local imports (those prefixed with the
// module name), and resolved from disk. Newly discovered packages
// are scanned for their own local imports, repeating until all
// transitive dependencies are found. Packages that are already
// available in the symbol registry are skipped because their types
// are pre-registered and do not need source compilation.
//
// Takes packages (map[string]map[string]string) which is the
// mutable map to populate with discovered package sources.
// Takes batchInterpreter (templater_domain.BatchInterpreterPort)
// which checks whether a package is already registered.
func (o *InterpretedBuildOrchestrator) discoverUserPackages(
	ctx context.Context,
	packages map[string]map[string]string,
	batchInterpreter templater_domain.BatchInterpreterPort,
) {
	_, l := logger_domain.From(ctx, log)

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

	pending := o.collectLocalImports(packages, modulePrefix)

	for len(pending) > 0 {
		var nextPending []string

		for _, importPath := range pending {
			discovered := o.resolveImportedPackage(
				importPath, modulePrefix, packages, sandbox, batchInterpreter, l,
			)
			nextPending = append(nextPending, discovered...)
		}

		pending = nextPending
	}
}

// resolveImportedPackage attempts to resolve a single local import
// path into a user package.
//
// When the package is found and not already known, its source files
// are added to packages and any transitive local imports are
// returned for further processing.
//
// Takes importPath (string) which is the fully qualified Go import
// path to resolve.
// Takes modulePrefix (string) which is the module name followed by
// a slash, used to identify local imports.
// Takes packages (map[string]map[string]string) which is the
// mutable map to populate with discovered sources.
// Takes sandbox (safedisk.Sandbox) which provides safe filesystem
// access for reading user source files.
// Takes batchInterpreter (templater_domain.BatchInterpreterPort)
// which checks whether a package is already registered.
// Takes l (logger_domain.Logger) which logs discovery progress.
//
// Returns []string containing any transitive local import paths
// discovered in the resolved package.
func (o *InterpretedBuildOrchestrator) resolveImportedPackage(
	importPath string,
	modulePrefix string,
	packages map[string]map[string]string,
	sandbox safedisk.Sandbox,
	batchInterpreter templater_domain.BatchInterpreterPort,
	l logger_domain.Logger,
) []string {
	if batchInterpreter.HasRegisteredPackage(importPath) {
		return nil
	}

	relativeDirectory := strings.TrimPrefix(importPath, modulePrefix)

	if _, exists := packages[relativeDirectory]; exists {
		return nil
	}

	goFiles := o.readUserGoFiles(sandbox, relativeDirectory, l)
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

// collectLocalImports scans all source files in the packages map
// and returns import paths that belong to the current module.
//
// Takes packages (map[string]map[string]string) which maps
// relative package paths to their filename-to-source maps.
// Takes modulePrefix (string) which is used to filter imports to
// only those belonging to the current module.
//
// Returns []string containing the deduplicated local import paths
// found across all source files.
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

// parseLocalImportPaths extracts import paths from Go source code
// that match the given module prefix.
//
// Uses go/parser with ImportsOnly for efficiency, since only the
// import block is parsed, not function bodies. All import styles
// (standard, aliased, blank, dot) are handled because the path is
// always extracted from importSpec.Path.Value.
//
// Takes source (string) which is the Go source code to parse.
// Takes modulePrefix (string) which filters imports to only those
// belonging to the current module.
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

// readUserGoFiles reads all non-test .go files from a directory
// using the provided sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides safe filesystem
// access scoped to the project root.
// Takes relDir (string) which is the relative directory path to
// read Go files from.
// Takes l (logger_domain.Logger) which logs read errors.
//
// Returns map[string]string mapping filenames to their source
// content, or nil when the directory cannot be read.
func (*InterpretedBuildOrchestrator) readUserGoFiles(
	sandbox safedisk.Sandbox,
	relDir string,
	l logger_domain.Logger,
) map[string]string {
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
		goFiles[e.Name()] = string(content)
	}
	return goFiles
}

// linkAllArtefacts creates PageEntry objects for all components and
// links their registered functions from the global FunctionRegistry.
//
// Takes components (map[string]*annotator_dto.VirtualComponent)
// which maps relative paths to virtual components to link.
// Takes manifest (*generator_dto.Manifest) which provides build
// metadata for page entry creation.
//
// Returns map[string]*PageEntry which maps relative paths
// to their fully linked page entries.
// Returns error when function linking fails for any
// component.
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

		entry := o.createPageEntry(ctx, manifest, component)
		if err := o.linkFunctionsFromRegistry(ctx, entry, component, shortPackageName); err != nil {
			return nil, fmt.Errorf("linking functions from registry for %q: %w", relativePath, err)
		}
		progCache[relativePath] = entry
	}

	return progCache, nil
}

// interpretArtefactsIncremental evaluates artefacts one-by-one
// using the incremental Eval() path with VFS-based import
// resolution.
//
// Takes freshInterpreter (templater_domain.InterpreterPort) which
// executes the generated code for each artefact.
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which
// are the artefacts to evaluate, in dependency order.
// Takes manifest (*generator_dto.Manifest) which provides build
// metadata for page entry creation.
// Takes vfsAdapter (*templater_adapters.RegistryVFSAdapter) which
// provides the virtual filesystem for import resolution.
//
// Returns map[string]*PageEntry which maps relative paths
// to their interpreted page entries.
// Returns error when interpretation fails for any artefact.
func (o *InterpretedBuildOrchestrator) interpretArtefactsIncremental(
	ctx context.Context,
	freshInterpreter templater_domain.InterpreterPort,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
	vfsAdapter *templater_adapters.RegistryVFSAdapter,
) (map[string]*templater_adapters.PageEntry, error) {
	ctx, l := logger_domain.From(ctx, log)

	buildCtx := vfsAdapter.GetBuildContext()
	freshInterpreter.SetBuildContext(buildCtx)
	freshInterpreter.SetSourcecodeFilesystem(vfsAdapter)
	l.Internal("[JIT-BUILD] Pre-warmed interpreter retrieved from pool and configured with VFS",
		logger_domain.String("gopath", buildCtx.GOPATH))

	progCache := make(map[string]*templater_adapters.PageEntry)
	for _, artefact := range sortedArtefacts {
		entry, relPath, err := o.interpretSingleArtefact(ctx, freshInterpreter, artefact, manifest)
		if err != nil {
			return nil, fmt.Errorf("interpreting artefact %q: %w", relPath, err)
		}
		if entry != nil {
			progCache[relPath] = entry
		}
	}

	return progCache, nil
}

// getInterpreterFromPool retrieves a pre-warmed interpreter from the pool.
//
// Returns templater_domain.InterpreterPort which is a ready-to-use interpreter
// instance.
// Returns error when the pool returns an invalid type.
func (o *InterpretedBuildOrchestrator) getInterpreterFromPool() (templater_domain.InterpreterPort, error) {
	interp, err := o.interpreterPool.Get()
	if err != nil {
		return nil, fmt.Errorf("retrieving interpreter from pool: %w", err)
	}
	return interp, nil
}

// returnInterpreterToPool resets and returns the interpreter to the pool.
//
// Takes ctx (context.Context) which carries the logger.
// Takes interpreter (templater_domain.InterpreterPort) which is the
// interpreter to reset and return to the pool.
func (o *InterpretedBuildOrchestrator) returnInterpreterToPool(ctx context.Context, interpreter templater_domain.InterpreterPort) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-BUILD] Resetting and returning interpreter to pool")
	interpreter.Reset()
	o.interpreterPool.Put(interpreter)
}

// interpretSingleArtefact interprets a single artefact and returns its
// PageEntry.
//
// Takes interpreter (templater_domain.InterpreterPort) which executes the
// generated code.
// Takes artefact (*generator_dto.GeneratedArtefact) which contains the
// generated content to interpret.
// Takes manifest (*generator_dto.Manifest) which provides build metadata.
//
// Returns *templater_adapters.PageEntry which is the interpreted page entry,
// or nil if no main component exists.
// Returns string which is the relative path to the artefact.
// Returns error when interpretation or linking fails.
func (o *InterpretedBuildOrchestrator) interpretSingleArtefact(
	ctx context.Context,
	interpreter templater_domain.InterpreterPort,
	artefact *generator_dto.GeneratedArtefact,
	manifest *generator_dto.Manifest,
) (*templater_adapters.PageEntry, string, error) {
	ctx, l := logger_domain.From(ctx, log)
	component, _ := generator_domain.GetMainComponent(artefact.Result)
	if component == nil {
		return nil, "", nil
	}

	relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
	if err != nil {
		l.Error("Failed to compute relative path",
			logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
			logger_domain.String("projectRoot", o.projectRoot),
			logger_domain.Error(err))
		return nil, "", nil
	}
	relativePath = filepath.ToSlash(relativePath)

	pikoPath := filepath.ToSlash(filepath.Join(o.moduleName, relativePath))
	l.Trace("[JIT-INTERP] Interpreting and linking artefact",
		logger_domain.String("artefact_id", relativePath),
		logger_domain.String("piko_path", pikoPath),
		logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))

	entry, err := o.interpretAndLink(ctx, interpreter, string(artefact.Content), manifest, component)
	if err != nil {
		return nil, "", fmt.Errorf("failed to interpret and link %s: %w", pikoPath, err)
	}

	l.Trace("[JIT-BUILD] Successfully cached entry",
		logger_domain.String("cache_key", relativePath),
		logger_domain.String("piko_path", pikoPath),
		logger_domain.String("fs_path", component.Source.SourcePath))

	return entry, relativePath, nil
}

// buildReverseDependencyMap creates a map from import paths to the components
// that depend on them.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides the
// build artefacts in dependency order.
//
// Returns map[string][]string which maps each import path to the list of
// component paths that depend on it.
func (o *InterpretedBuildOrchestrator) buildReverseDependencyMap(
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) map[string][]string {
	reverseDepsMap := make(map[string][]string)
	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			continue
		}
		relativePath = filepath.ToSlash(relativePath)

		for _, pikoImport := range component.Source.PikoImports {
			importRelativePath := o.extractImportRelativePath(pikoImport.Path)
			reverseDepsMap[importRelativePath] = append(reverseDepsMap[importRelativePath], relativePath)
		}
	}
	return reverseDepsMap
}

// rebuildReverseDependencyMapFromState rebuilds the reverse dependency map
// from the current artefact state. Called after targeted builds to ensure
// imports added or removed during the build are reflected in future
// GetAffectedComponents lookups.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) rebuildReverseDependencyMapFromState() {
	allArtefacts := make([]*generator_dto.GeneratedArtefact, 0, len(o.artefactByPackagePath))
	for _, artefact := range o.artefactByPackagePath {
		allArtefacts = append(allArtefacts, artefact)
	}
	o.reverseDepsMap = o.buildReverseDependencyMap(allArtefacts)
}

// extractImportRelativePath gets the relative path from a piko import path.
//
// Takes importPath (string) which is the full import path to process.
//
// Returns string which is the part after the first slash, or the original
// path if no slash is found.
func (*InterpretedBuildOrchestrator) extractImportRelativePath(importPath string) string {
	parts := strings.SplitN(importPath, "/", 2)
	if len(parts) > 1 {
		return filepath.ToSlash(parts[1])
	}
	return filepath.ToSlash(importPath)
}

// buildArtefactLookupMap builds a map from canonical package paths to
// artefacts.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which provides
// the artefacts to index by their main component's package path.
//
// Returns map[string]*generator_dto.GeneratedArtefact which maps canonical
// package paths to their matching artefacts.
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

// updateOrchestratorState updates the orchestrator's internal state after a
// build.
//
// Takes vfsAdapter (*templater_adapters.RegistryVFSAdapter) which provides
// virtual filesystem access for templates.
// Takes progCache (map[string]*templater_adapters.PageEntry) which contains
// cached page entries by path.
// Takes manifest (*generator_dto.Manifest) which holds the build manifest.
// Takes reverseDepsMap (map[string][]string) which maps paths to their
// dependents.
// Takes artefactByPackagePath (map[string]*generator_dto.GeneratedArtefact) which
// maps package paths to generated artefacts.
//
// Safe for concurrent use; acquires stateLock before updating fields.
func (o *InterpretedBuildOrchestrator) updateOrchestratorState(
	vfsAdapter *templater_adapters.RegistryVFSAdapter,
	progCache map[string]*templater_adapters.PageEntry,
	manifest *generator_dto.Manifest,
	reverseDepsMap map[string][]string,
	artefactByPackagePath map[string]*generator_dto.GeneratedArtefact,
) {
	o.stateLock.Lock()
	defer o.stateLock.Unlock()
	o.vfsAdapter = vfsAdapter
	o.progCache = progCache
	o.cachedManifest = manifest
	o.reverseDepsMap = reverseDepsMap
	o.artefactByPackagePath = artefactByPackagePath
	o.dirtyCodeCache = make(map[string][]byte)
}

// extractMarkDirtyArtefacts gets artefacts from the annotation result for the
// MarkDirty operation.
//
// Takes ctx (context.Context) which carries the logger.
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the
// annotation result with its generated artefacts.
//
// Returns []*generator_dto.GeneratedArtefact which holds the extracted
// artefacts, or nil if there are none.
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

// updateArtefactLookupAndVFS updates the artefact lookup map and builds
// the VFS path map. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// generated artefacts to register.
//
// Returns map[string]string which maps canonical package paths to relative
// file paths.
func (o *InterpretedBuildOrchestrator) updateArtefactLookupAndVFS(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
) map[string]string {
	ctx, l := logger_domain.From(ctx, log)
	newPathMap := make(map[string]string)
	for _, artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		o.artefactByPackagePath[component.CanonicalGoPackagePath] = artefact

		relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			l.Error("[JIT-MARK-DIRTY] Failed to compute relative path for VFS map",
				logger_domain.String(fieldAbsolutePath, component.Source.SourcePath),
				logger_domain.String("projectRoot", o.projectRoot),
				logger_domain.Error(err))
			continue
		}
		relativePath = filepath.ToSlash(relativePath)
		newPathMap[component.CanonicalGoPackagePath] = relativePath
	}
	return newPathMap
}

// updateVFSAdapterIfNeeded updates the VFS adapter with new path mappings
// and artefacts. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// newly created artefacts to add to the VFS cache.
// Takes newPathMap (map[string]string) which provides the path mappings to
// update in the VFS adapter.
func (o *InterpretedBuildOrchestrator) updateVFSAdapterIfNeeded(
	ctx context.Context,
	artefacts []*generator_dto.GeneratedArtefact,
	newPathMap map[string]string,
) {
	ctx, l := logger_domain.From(ctx, log)
	if o.vfsAdapter == nil {
		return
	}

	o.vfsAdapter.UpdateMap(newPathMap)
	l.Internal("[JIT-MARK-DIRTY] Updated VFS path mappings",
		logger_domain.Int("new_mappings", len(newPathMap)))

	if err := o.vfsAdapter.UpdateFreshArtefacts(artefacts, o.projectRoot); err != nil {
		l.Warn("[JIT-MARK-DIRTY] Failed to update VFS fresh artefacts cache",
			logger_domain.Error(err))
	} else {
		l.Internal("[JIT-MARK-DIRTY] VFS adapter updated with new artefacts",
			logger_domain.Int("artefact_count", len(artefacts)))
	}
}

// markDirectlyChangedComponents marks components as dirty based on generated
// code. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// generated code to process.
//
// Returns directlyChanged (map[string]bool) which tracks paths changed
// directly by this operation.
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

// propagateDirtyFlags marks all dependents as dirty using the reverse
// dependency map.
//
// Takes ctx (context.Context) which carries the logger.
// Takes directlyChanged (map[string]bool) which contains paths that were
// changed and need their dependents marked dirty.
// Takes allDirty (map[string]bool) which collects all paths marked dirty,
// including those affected through other dependents.
//
// Must be called with stateLock held.
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

// addDependentToDirtyCache finds the dependent's artefact code and adds it to
// the dirty cache for later recompilation.
//
// Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
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

// mergeManifest merges entries from newManifest into the cached manifest
// without removing existing entries. This preserves manifest data for
// components not included in a targeted build.
//
// Takes newManifest (*generator_dto.Manifest) which contains the entries to
// merge.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) mergeManifest(newManifest *generator_dto.Manifest) {
	if o.cachedManifest == nil {
		o.cachedManifest = newManifest
		return
	}

	maps.Copy(o.cachedManifest.Pages, newManifest.Pages)
	maps.Copy(o.cachedManifest.Partials, newManifest.Partials)
	maps.Copy(o.cachedManifest.Emails, newManifest.Emails)
	maps.Copy(o.cachedManifest.ErrorPages, newManifest.ErrorPages)
}
