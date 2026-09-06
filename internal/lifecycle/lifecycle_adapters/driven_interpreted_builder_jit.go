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
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
)

// executeJITCompilation performs the actual JIT compilation work.
//
// Takes relPath (string) which specifies the path of the component to compile.
//
// Returns error when prerequisites are invalid, dependency collection fails, no
// interpreter is available, or component re-evaluation fails.
//
// Not safe for concurrent use. Acquires stateLock internally and releases it before
// returning.
func (o *InterpretedBuildOrchestrator) executeJITCompilation(
	ctx context.Context,
	relPath string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	o.stateLock.Lock()

	if !o.isComponentDirty(relPath) {
		o.stateLock.Unlock()
		l.Trace("[JIT-COMPILE] Component is clean, skipping compilation",
			logger_domain.String(fieldPath, relPath))
		return nil
	}

	l.Internal("[JIT-COMPILE] ========== Compiling Component On-Demand ==========",
		logger_domain.String(fieldPath, relPath))

	if err := o.validateJITPrerequisites(ctx); err != nil {
		o.stateLock.Unlock()
		return fmt.Errorf("validating JIT prerequisites: %w", err)
	}

	sortedArtefacts, err := o.collectAndSortDependencies(ctx, relPath)
	if err != nil {
		o.stateLock.Unlock()
		return fmt.Errorf("collecting and sorting dependencies for %q: %w", relPath, err)
	}

	jitInterpreter, err := o.getJITInterpreter(ctx)
	if err != nil {
		o.stateLock.Unlock()
		return fmt.Errorf("getting JIT interpreter: %w", err)
	}

	if err := o.reevaluateComponentsBatch(ctx, sortedArtefacts, jitInterpreter); err != nil {
		o.stateLock.Unlock()
		return fmt.Errorf("batch re-evaluating components for %q: %w", relPath, err)
	}

	l.Internal("[JIT-COMPILE] ========== Compilation Complete ==========",
		logger_domain.String(fieldPath, relPath),
		logger_domain.Int("compiled_count", len(sortedArtefacts)),
		logger_domain.Int("remaining_dirty", len(o.dirtyCodeCache)))

	o.stateLock.Unlock()
	return nil
}

// isComponentDirty checks if a component is in the dirty cache.
//
// Takes relPath (string) which specifies the relative path of the component.
//
// Returns bool which is true if the component is marked as dirty.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) isComponentDirty(relPath string) bool {
	_, isDirty := o.dirtyCodeCache[relPath]
	return isDirty
}

// validateJITPrerequisites checks that an initial build has populated the manifest. Must
// be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
//
// Returns error when the cached manifest is nil.
func (o *InterpretedBuildOrchestrator) validateJITPrerequisites(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	if o.cachedManifest == nil {
		l.Error("[JIT-COMPILE] No cached manifest available")
		return errors.New("no cached manifest available for JIT compilation")
	}
	return nil
}

// collectAndSortDependencies collects dependencies and sorts them topologically. Must be
// called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes relPath (string) which specifies the path to the target component.
//
// Returns []*generator_dto.GeneratedArtefact which contains the sorted dependencies ready
// for compilation.
// Returns error when dependency collection or topological sorting fails.
func (o *InterpretedBuildOrchestrator) collectAndSortDependencies(
	ctx context.Context,
	relPath string,
) ([]*generator_dto.GeneratedArtefact, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-COMPILE] Step 1: Collecting target component and all dependencies...")
	artefactsToCompile, err := o.collectDependencies(relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to collect dependencies: %w", err)
	}
	l.Internal("[JIT-COMPILE] Collected components to compile",
		logger_domain.Int("count", len(artefactsToCompile)))

	l.Internal("[JIT-COMPILE] Step 2: Topologically sorting artefacts...")
	sortedArtefacts, err := o.topologicallySortArtefacts(artefactsToCompile)
	if err != nil {
		return nil, fmt.Errorf("failed to topologically sort artefacts: %w", err)
	}

	return sortedArtefacts, nil
}

// getJITInterpreter gets an interpreter from the pool for JIT compilation. Must be called
// with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
//
// Returns templater_domain.InterpreterPort which shares the symbol registry populated by
// the initial build.
// Returns error when the interpreter pool cannot provide an interpreter.
func (o *InterpretedBuildOrchestrator) getJITInterpreter(ctx context.Context) (templater_domain.InterpreterPort, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-COMPILE] Step 3: Getting interpreter from pool...")
	if o.interpreterPool == nil {
		return nil, errNoInterpreterPool
	}
	jitInterpreter, err := getInterpreterFromPool(o.interpreterPool)
	if err != nil {
		return nil, fmt.Errorf("getting interpreter from pool for JIT compilation: %w", err)
	}
	l.Internal("[JIT-COMPILE] Pre-warmed interpreter retrieved")

	return jitInterpreter, nil
}

// reevaluateComponentsBatch re-evaluates all sorted artefacts using batch compilation.
//
// A dirty entry is cleared only when it still holds the code that was compiled, so a save
// stored by MarkDirty while the lock was released stays dirty for the next compile.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which contains the artefacts
// to compile in dependency order.
// Takes interpreter (templater_domain.InterpreterPort) which handles compilation and
// execution.
//
// Returns error when batch compilation or linking fails.
//
// Not safe for concurrent use. Must be called with stateLock held. Releases and
// reacquires the lock during discovery and compilation.
func (o *InterpretedBuildOrchestrator) reevaluateComponentsBatch(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	interpreter templater_domain.InterpreterPort,
) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("[JIT-COMPILE] Step 4: Batch re-evaluating all components...")

	packages, compiledDirtyCode := o.collectJITSources(ctx, sortedArtefacts)

	o.stateLock.Unlock()
	o.discoverUserPackages(ctx, packages, interpreter)
	err := o.compileAndExecute(ctx, interpreter, packages)
	o.stateLock.Lock()

	if err != nil {
		return fmt.Errorf("batch JIT compilation failed: %w", err)
	}

	if linkErr := o.linkBatchArtefacts(ctx, sortedArtefacts); linkErr != nil {
		return linkErr
	}

	o.clearCompiledDirtyCode(compiledDirtyCode)

	return nil
}

// collectJITSources gathers the source of every artefact in a JIT batch, preferring
// pending dirty code over the artefact's last built content.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which are the artefacts in
// the batch.
//
// Returns map[string]map[string]string which maps relative package paths to
// filename-to-source maps.
// Returns map[string][]byte which maps each dirty component's relative path to the exact
// code compiled for it.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) collectJITSources(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) (map[string]map[string]string, map[string][]byte) {
	packages := make(map[string]map[string]string, len(sortedArtefacts))
	compiledDirtyCode := make(map[string][]byte)

	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}

		componentRelativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if err != nil {
			continue
		}
		componentRelativePath = filepath.ToSlash(componentRelativePath)

		code, isDirty := o.getComponentCode(ctx, artefact, componentRelativePath, component)
		pkgRelPath := strings.TrimPrefix(component.CanonicalGoPackagePath, o.moduleName+"/")
		packages[pkgRelPath] = map[string]string{"generated.go": string(code)}

		if isDirty {
			compiledDirtyCode[componentRelativePath] = code
		}
	}

	return packages, compiledDirtyCode
}

// clearCompiledDirtyCode removes dirty entries whose pending code is exactly the code
// that was compiled. An entry replaced while the compile ran no longer matches and stays
// dirty.
//
// Takes compiledDirtyCode (map[string][]byte) which maps relative paths to the code
// compiled for them.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) clearCompiledDirtyCode(compiledDirtyCode map[string][]byte) {
	for relativePath, compiled := range compiledDirtyCode {
		if pending, ok := o.dirtyCodeCache[relativePath]; ok && bytes.Equal(pending, compiled) {
			delete(o.dirtyCodeCache, relativePath)
		}
	}
}

// linkBatchArtefacts links functions from the registry for each artefact after batch
// compilation.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which contains the artefacts
// to link.
//
// Returns error when function linking fails for any artefact.
//
// Must be called with stateLock held.
func (o *InterpretedBuildOrchestrator) linkBatchArtefacts(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) error {
	ctx, l := logger_domain.From(ctx, log)
	for _, artefact := range sortedArtefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}
		componentRelativePath, relErr := filepath.Rel(o.projectRoot, component.Source.SourcePath)
		if relErr != nil {
			continue
		}
		componentRelativePath = filepath.ToSlash(componentRelativePath)

		shortPackageName, nameErr := extractPackageName(string(artefact.Content))
		if nameErr != nil {
			shortPackageName = component.HashedName
		}

		linkFn := func(entry *templater_adapters.PageEntry, comp *annotator_dto.VirtualComponent) error {
			return o.linkFunctionsFromRegistry(ctx, entry, comp, shortPackageName)
		}
		if err := o.populateProgCacheForComponent(ctx, o.cachedManifest, component, componentRelativePath, linkFn, o.progCache); err != nil {
			l.Error("[JIT-COMPILE] Failed to link functions after batch compilation",
				logger_domain.String(fieldPath, componentRelativePath),
				logger_domain.Error(err))
			return fmt.Errorf("failed to link %s: %w", componentRelativePath, err)
		}
	}
	return nil
}

// getComponentCode returns the code to use for compilation and whether it is dirty. Must
// be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the clean source
// content.
// Takes componentRelativePath (string) which identifies the component's relative path.
// Takes component (*annotator_dto.VirtualComponent) which provides package metadata.
//
// Returns []byte which is the source code to compile.
// Returns bool which is true when the code is dirty, false when clean.
func (o *InterpretedBuildOrchestrator) getComponentCode(
	ctx context.Context,
	artefact *generator_dto.GeneratedArtefact,
	componentRelativePath string,
	component *annotator_dto.VirtualComponent,
) ([]byte, bool) {
	_, l := logger_domain.From(ctx, log)
	if dirtyCode, hasDirty := o.dirtyCodeCache[componentRelativePath]; hasDirty {
		l.Trace("[JIT-COMPILE] Re-evaluating DIRTY component...",
			logger_domain.String(fieldPath, componentRelativePath),
			logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))
		return dirtyCode, true
	}

	l.Trace("[JIT-COMPILE] Re-evaluating CLEAN dependency...",
		logger_domain.String(fieldPath, componentRelativePath),
		logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))
	return artefact.Content, false
}

// createPageEntry creates a PageEntry from the manifest or as a private partial.
//
// Takes ctx (context.Context) which carries the logger.
// Takes manifest (*generator_dto.Manifest) which contains page metadata.
// Takes component (*annotator_dto.VirtualComponent) which provides the source info.
//
// Returns *templater_adapters.PageEntry which is either from the manifest or a fallback
// for private partials.
func (o *InterpretedBuildOrchestrator) createPageEntry(
	ctx context.Context,
	manifest *generator_dto.Manifest,
	component *annotator_dto.VirtualComponent,
) *templater_adapters.PageEntry {
	ctx, l := logger_domain.From(ctx, log)
	relativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
	if err != nil {
		relativePath = component.Source.SourcePath
	}
	relativePath = filepath.ToSlash(relativePath)

	entry := createPageEntryFromManifest(manifest, relativePath)
	if entry == nil {
		l.Trace("Creating PageEntry for private partial from VirtualComponent",
			logger_domain.String("source_path", relativePath),
			logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))
		entry = templater_adapters.NewComponentPageEntry(component.CanonicalGoPackagePath, relativePath)
	}

	entry.SetBaseDir(o.projectRoot)
	entry.SetJSArtefactToPartialNameMap(buildJSArtefactToPartialNameMap(manifest))
	entry.InitialiseLocalStore()

	return entry
}

// linkFunctionsFromRegistry attaches every function the component registered to the page
// entry, through the same linker the compiled manifest store uses so the interpreted
// build cannot miss a function kind (auth policy, preview and so on).
//
// Takes ctx (context.Context) which carries the logger.
// Takes entry (*templater_adapters.PageEntry) which receives the linked function
// pointers.
// Takes component (*annotator_dto.VirtualComponent) which provides the package path used
// to look up functions in the registry.
// Takes shortPackageName (string) which identifies the package in error messages.
//
// Returns error when the BuildAST function is not found in the global registry.
func (*InterpretedBuildOrchestrator) linkFunctionsFromRegistry(
	ctx context.Context,
	entry *templater_adapters.PageEntry,
	component *annotator_dto.VirtualComponent,
	shortPackageName string,
) error {
	ctx, l := logger_domain.From(ctx, log)
	registryKey := component.CanonicalGoPackagePath

	l.Trace("Extracting BuildAST function from global registry",
		logger_domain.String("canonical_path", component.CanonicalGoPackagePath),
		logger_domain.String("registry_key", registryKey))

	_, ok := templater_domain.GetASTFunc(registryKey)
	if !ok {
		l.Error("BuildAST function NOT FOUND in global registry after evaluation",
			logger_domain.String("registry_key", registryKey),
			logger_domain.String("canonical_path", component.CanonicalGoPackagePath),
			logger_domain.String("short_pkg_name", shortPackageName))
		return fmt.Errorf("BuildAST not found in global registry for %s (canonical: %s, short name: %s)",
			registryKey, component.CanonicalGoPackagePath, shortPackageName)
	}

	entry.LinkFuncsFor(registryKey)

	l.Trace("Component fully linked",
		logger_domain.String("source_path", component.Source.SourcePath))

	return nil
}

// populateProgCacheForComponent writes one or more PageEntry records into target for a
// component that has just been interpreted.
//
// For collection-backed components the .pk file itself has no route; the manifest holds a
// separate entry per virtual instance (for example one per markdown post), each with its
// own concrete route. Without this expansion the dev-i runner only registers a single
// entry keyed by the .pk file's relative path, loses the per-instance routes, and leaves
// /blog/post-slug unreachable in interpreted mode.
//
// Takes ctx (context.Context) which carries the logger.
// Takes manifest (*generator_dto.Manifest) which holds the per-instance manifest entries.
// Takes component (*annotator_dto.VirtualComponent) whose instances drive the expansion.
// Takes componentRelativePath (string) which is the .pk file's path relative to the
// project root, used when the component has no instances.
// Takes linkFn which performs the function-pointer wiring. Called once per emitted entry
// so that instance-specific caches (like the registered AST function) end up attached to
// each entry.
// Takes target (map[string]*templater_adapters.PageEntry) which receives the produced
// entries.
//
// Returns error when any instance's link step fails.
func (o *InterpretedBuildOrchestrator) populateProgCacheForComponent(
	ctx context.Context,
	manifest *generator_dto.Manifest,
	component *annotator_dto.VirtualComponent,
	componentRelativePath string,
	linkFn func(entry *templater_adapters.PageEntry, component *annotator_dto.VirtualComponent) error,
	target map[string]*templater_adapters.PageEntry,
) error {
	if len(component.VirtualInstances) == 0 || isCollectionBacked(component) {
		entry := o.createPageEntry(ctx, manifest, component)
		if err := linkFn(entry, component); err != nil {
			return err
		}
		target[componentRelativePath] = entry
		return nil
	}

	staged := make(map[string]*templater_adapters.PageEntry, len(component.VirtualInstances))
	for _, instance := range component.VirtualInstances {
		manifestKey := instance.ManifestKey
		if manifestKey == "" {
			continue
		}
		entry := o.createInstancePageEntry(manifest, component, instance)
		if err := linkFn(entry, component); err != nil {
			return err
		}
		staged[manifestKey] = entry
	}
	maps.Copy(target, staged)
	return nil
}

// createInstancePageEntry builds a per-instance PageEntry using the manifest's
// pre-computed entry for the instance's ManifestKey, which carries the concrete
// RoutePatterns. Falls back to a minimal entry when the manifest lookup misses so a
// broken early-JIT state still produces a usable progCache.
//
// Takes manifest (*generator_dto.Manifest) which supplies the per- instance page data.
// Takes component (*annotator_dto.VirtualComponent) the instance belongs to; provides the
// fallback PackagePath.
// Takes instance (annotator_dto.VirtualPageInstance) providing the manifest key used for
// lookup.
//
// Returns the prepared entry before function-pointer linking.
func (o *InterpretedBuildOrchestrator) createInstancePageEntry(
	manifest *generator_dto.Manifest,
	component *annotator_dto.VirtualComponent,
	instance annotator_dto.VirtualPageInstance,
) *templater_adapters.PageEntry {
	entry := createPageEntryFromManifest(manifest, instance.ManifestKey)
	if entry == nil {
		entry = templater_adapters.NewComponentPageEntry(component.CanonicalGoPackagePath, instance.ManifestKey)
	}
	entry.SetBaseDir(o.projectRoot)
	entry.SetJSArtefactToPartialNameMap(buildJSArtefactToPartialNameMap(manifest))
	entry.InitialiseLocalStore()
	return entry
}

// buildJSArtefactToPartialNameMap iterates the manifest's partials and builds the mapping
// from JS artefact IDs to friendly partial names.
//
// This mirrors the logic in processPartials (driven_manifest_store.go) for the compiled
// path.
//
// Takes manifest (*generator_dto.Manifest) which provides the partials to iterate.
//
// Returns map[string]string which maps JS artefact IDs to partial names.
func buildJSArtefactToPartialNameMap(manifest *generator_dto.Manifest) map[string]string {
	m := make(map[string]string, len(manifest.Partials))
	for _, partial := range manifest.Partials {
		if partial.JSArtefactID != "" {
			m[partial.JSArtefactID] = partial.PartialName
		}
	}
	return m
}

// extractPackageName finds the package name from Go source code.
//
// Takes code (string) which contains the Go source code to parse.
//
// Returns string which is the package name found in the code.
// Returns error when the code has no package declaration.
func extractPackageName(code string) (string, error) {
	for line := range strings.SplitSeq(code, "\n") {
		trimmed := strings.TrimSpace(line)
		packageName, found := strings.CutPrefix(trimmed, "package ")
		if !found {
			continue
		}

		packageName = strings.TrimSpace(packageName)
		if index := strings.Index(packageName, "//"); index != -1 {
			packageName = strings.TrimSpace(packageName[:index])
		}
		return packageName, nil
	}
	return "", errors.New("invalid generated code: missing package declaration")
}

// isCollectionBacked reports whether the component declares a p-collection consumer.
// These pages register a single dynamic route instead of per-item routes; the runtime
// resolves items by slug.
//
// Takes component (*annotator_dto.VirtualComponent) which is the component descriptor to
// inspect.
//
// Returns bool which is true when the component consumes a p-collection.
func isCollectionBacked(component *annotator_dto.VirtualComponent) bool {
	return component != nil && component.Source != nil && component.Source.HasCollection
}

// createPageEntryFromManifest builds a PageEntry from manifest data by looking up the
// source path in the pages, partials, emails, PDFs, or error pages maps, using the same
// constructors as the compiled manifest store. Error page entries also carry their
// routing metadata so the interpreted store can dispatch by status code.
//
// Takes manifest (*generator_dto.Manifest) which holds the page, partial, email, PDF and
// error page data to search.
// Takes sourcePath (string) which identifies the entry to find.
//
// Returns *templater_adapters.PageEntry which wraps the found manifest data, or nil when
// the source path is not found in any map.
func createPageEntryFromManifest(manifest *generator_dto.Manifest, sourcePath string) *templater_adapters.PageEntry {
	if pageData, ok := manifest.Pages[sourcePath]; ok {
		return templater_adapters.NewPageEntry(pageData)
	}
	if partialData, ok := manifest.Partials[sourcePath]; ok {
		return templater_adapters.NewPartialPageEntry(partialData)
	}
	if emailData, ok := manifest.Emails[sourcePath]; ok {
		return templater_adapters.NewEmailPageEntry(emailData)
	}
	if pdfData, ok := manifest.Pdfs[sourcePath]; ok {
		return templater_adapters.NewPdfPageEntry(pdfData)
	}
	if errorData, ok := manifest.ErrorPages[sourcePath]; ok {
		entry := templater_adapters.NewErrorPageEntry(errorData)
		entry.ErrorDispatch = templater_adapters.NewErrorPageDispatch(errorData)
		return entry
	}
	return nil
}
