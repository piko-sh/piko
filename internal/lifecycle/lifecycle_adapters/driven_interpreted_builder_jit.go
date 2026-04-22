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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/safedisk"
)

// executeJITCompilation performs the actual JIT compilation work.
//
// Takes relPath (string) which specifies the path of the component to compile.
//
// Returns error when prerequisites are invalid, dependency collection fails,
// no interpreter is available, or component re-evaluation fails.
//
// Not safe for concurrent use. Acquires stateLock internally and releases it
// before returning.
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

	if batchInterp, ok := jitInterpreter.(templater_domain.BatchInterpreterPort); ok {
		if err := o.reevaluateComponentsBatch(ctx, sortedArtefacts, batchInterp); err != nil {
			o.stateLock.Unlock()
			return fmt.Errorf("batch re-evaluating components for %q: %w", relPath, err)
		}
	} else {
		defer o.returnInterpreterToPool(ctx, jitInterpreter)
		if err := o.reevaluateComponents(ctx, sortedArtefacts, jitInterpreter); err != nil {
			o.stateLock.Unlock()
			return fmt.Errorf("re-evaluating components for %q: %w", relPath, err)
		}
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

// validateJITPrerequisites checks that the VFS adapter and manifest are
// available. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
//
// Returns error when the VFS adapter or cached manifest is nil.
func (o *InterpretedBuildOrchestrator) validateJITPrerequisites(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	if o.vfsAdapter == nil {
		l.Error("[JIT-COMPILE] No VFS adapter available")
		return errors.New("no VFS adapter available for JIT compilation")
	}
	if o.cachedManifest == nil {
		l.Error("[JIT-COMPILE] No cached manifest available")
		return errors.New("no cached manifest available for JIT compilation")
	}
	return nil
}

// collectAndSortDependencies collects dependencies and sorts them
// topologically. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes relPath (string) which specifies the path to the target component.
//
// Returns []*generator_dto.GeneratedArtefact which contains the sorted
// dependencies ready for compilation.
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

// getJITInterpreter gets an interpreter from the pool and sets it up for JIT
// compilation. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
//
// Returns templater_domain.InterpreterPort which is set up with the VFS and
// build context.
// Returns error when the interpreter pool is empty or not available.
func (o *InterpretedBuildOrchestrator) getJITInterpreter(ctx context.Context) (templater_domain.InterpreterPort, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[JIT-COMPILE] Step 3: Getting interpreter from pool...")
	jitInterpreter, err := o.getInterpreterFromPool()
	if err != nil {
		return nil, fmt.Errorf("getting interpreter from pool for JIT compilation: %w", err)
	}

	jitInterpreter.SetBuildContext(o.vfsAdapter.GetBuildContext())
	jitInterpreter.SetSourcecodeFilesystem(o.vfsAdapter)
	l.Internal("[JIT-COMPILE] Pre-warmed interpreter retrieved and configured with VFS")

	return jitInterpreter, nil
}

// reevaluateComponents re-evaluates all sorted artefacts in order.
//
// Must be called with stateLock held. Releases and reacquires lock during
// interpretation.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which contains
// the artefacts to re-evaluate in dependency order.
// Takes interpreter (templater_domain.InterpreterPort) which handles template
// interpretation.
//
// Returns error when any component fails to re-evaluate.
func (o *InterpretedBuildOrchestrator) reevaluateComponents(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	interpreter templater_domain.InterpreterPort,
) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("[JIT-COMPILE] Step 4: Re-evaluating all components in sorted order...")

	for _, artefact := range sortedArtefacts {
		if err := o.reevaluateSingleComponent(ctx, artefact, interpreter); err != nil {
			return fmt.Errorf("re-evaluating component: %w", err)
		}
	}

	return nil
}

// reevaluateComponentsBatch re-evaluates all sorted artefacts using
// batch compilation.
//
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which
// contains the artefacts to compile in dependency order.
// Takes batchInterp (templater_domain.BatchInterpreterPort) which
// handles batch compilation and execution.
//
// Returns error when batch compilation or linking fails.
//
// Not safe for concurrent use. Must be called with stateLock held.
// Releases and reacquires the lock during compilation.
func (o *InterpretedBuildOrchestrator) reevaluateComponentsBatch(
	ctx context.Context,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
	batchInterp templater_domain.BatchInterpreterPort,
) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("[JIT-COMPILE] Step 4: Batch re-evaluating all components...")

	packages := make(map[string]map[string]string, len(sortedArtefacts))
	var dirtyPaths []string

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
		packages[pkgRelPath] = map[string]string{"generated.go": code}

		if isDirty {
			dirtyPaths = append(dirtyPaths, componentRelativePath)
		}
	}

	o.stateLock.Unlock()
	err := batchInterp.CompileAndExecute(ctx, o.moduleName, packages)
	o.stateLock.Lock()

	if err != nil {
		return fmt.Errorf("batch JIT compilation failed: %w", err)
	}

	if linkErr := o.linkBatchArtefacts(ctx, l, sortedArtefacts); linkErr != nil {
		return linkErr
	}

	for _, dirtyPath := range dirtyPaths {
		delete(o.dirtyCodeCache, dirtyPath)
	}

	return nil
}

// linkBatchArtefacts links functions from the registry for each
// artefact after batch compilation.
//
// Takes l (logger_domain.Logger) which is the logger instance.
// Takes sortedArtefacts ([]*generator_dto.GeneratedArtefact) which
// contains the artefacts to link.
//
// Returns error when function linking fails for any artefact.
func (o *InterpretedBuildOrchestrator) linkBatchArtefacts(
	ctx context.Context,
	l logger_domain.Logger,
	sortedArtefacts []*generator_dto.GeneratedArtefact,
) error {
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

// reevaluateSingleComponent re-evaluates a single artefact.
//
// Takes artefact (*generator_dto.GeneratedArtefact) which specifies the
// artefact to re-evaluate.
// Takes interpreter (templater_domain.InterpreterPort) which provides the
// interpretation capability.
//
// Returns error when interpretation fails.
//
// Not safe for concurrent use. Must be called with stateLock held.
// Releases and reacquires the lock during interpretation.
func (o *InterpretedBuildOrchestrator) reevaluateSingleComponent(
	ctx context.Context,
	artefact *generator_dto.GeneratedArtefact,
	interpreter templater_domain.InterpreterPort,
) error {
	ctx, l := logger_domain.From(ctx, log)

	component, _ := generator_domain.GetMainComponent(artefact.Result)
	if component == nil {
		return nil
	}

	componentRelativePath, err := filepath.Rel(o.projectRoot, component.Source.SourcePath)
	if err != nil {
		l.Error("[JIT-COMPILE] Failed to compute relative path",
			logger_domain.String("sourcePath", component.Source.SourcePath),
			logger_domain.Error(err))
		return nil
	}
	componentRelativePath = filepath.ToSlash(componentRelativePath)

	code, isDirty := o.getComponentCode(ctx, artefact, componentRelativePath, component)

	manifestSnapshot := o.cachedManifest
	o.stateLock.Unlock()
	entries, err := o.interpretAndLink(ctx, interpreter, code, manifestSnapshot, component, componentRelativePath)
	o.stateLock.Lock()

	if err != nil {
		l.Error("[JIT-COMPILE] Failed to interpret component",
			logger_domain.String(fieldPath, componentRelativePath),
			logger_domain.Error(err))
		return fmt.Errorf("failed to interpret %s: %w", componentRelativePath, err)
	}

	maps.Copy(o.progCache, entries)

	if isDirty {
		delete(o.dirtyCodeCache, componentRelativePath)
		l.Trace("[JIT-COMPILE] Component re-compiled and removed from dirty cache",
			logger_domain.String(fieldPath, componentRelativePath))
	}

	return nil
}

// getComponentCode returns the code to use for compilation and whether it is
// dirty. Must be called with stateLock held.
//
// Takes ctx (context.Context) which carries the logger.
// Takes artefact (*generator_dto.GeneratedArtefact) which provides the clean
// source content.
// Takes componentRelativePath (string) which identifies the component's
// relative path.
// Takes component (*annotator_dto.VirtualComponent) which provides package
// metadata.
//
// Returns string which is the source code to compile.
// Returns bool which is true when the code is dirty, false when clean.
func (o *InterpretedBuildOrchestrator) getComponentCode(
	ctx context.Context,
	artefact *generator_dto.GeneratedArtefact,
	componentRelativePath string,
	component *annotator_dto.VirtualComponent,
) (string, bool) {
	ctx, l := logger_domain.From(ctx, log)
	if dirtyCode, hasDirty := o.dirtyCodeCache[componentRelativePath]; hasDirty {
		l.Trace("[JIT-COMPILE] Re-evaluating DIRTY component...",
			logger_domain.String(fieldPath, componentRelativePath),
			logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))
		return string(dirtyCode), true
	}

	l.Trace("[JIT-COMPILE] Re-evaluating CLEAN dependency...",
		logger_domain.String(fieldPath, componentRelativePath),
		logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath))
	return string(artefact.Content), false
}

// interpretAndLink evaluates Go code in the interpreter and retrieves the
// registered function pointers from the global registry.
//
// The "Two Worlds Problem" is solved by the symbol provider, which exposes
// the real templater_domain.RegisterASTFunc (and other registration functions)
// to the interpreter. When the interpreted code's init function calls
// RegisterASTFunc, it writes to the actual global registry in the main
// program's memory space.
//
// The method:
//  1. Compiles the code using interpreter.Eval, which runs init and registers
//     functions.
//  2. Retrieves function pointers from the global registry using
//     Get*Func(packagePath).
//  3. Injects them into the PageEntry via setters.
//
// Takes interpreter (templater_domain.InterpreterPort) which executes the Go
// code.
// Takes code (string) which contains the generated Go source to evaluate.
// Takes manifest (*generator_dto.Manifest) which provides page metadata.
// Takes component (*annotator_dto.VirtualComponent) which describes the
// component.
//
// Returns map[string]*templater_adapters.PageEntry which is keyed by
// manifest route, each entry carrying the linked render / build
// functions for one virtual page instance.
// Returns error when code evaluation or function linking fails.
//
// Not safe for concurrent use. Uses a semaphore and mutex to control
// interpreter access.
func (o *InterpretedBuildOrchestrator) interpretAndLink(
	ctx context.Context,
	interpreter templater_domain.InterpreterPort,
	code string,
	manifest *generator_dto.Manifest,
	component *annotator_dto.VirtualComponent,
	componentRelativePath string,
) (map[string]*templater_adapters.PageEntry, error) {
	templater_adapters.InterpretedManifestRunnerCompilationCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		templater_adapters.InterpretedManifestRunnerCompilationDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	ctx, span, l := log.Span(ctx, "interpretAndLink",
		logger_domain.String(fieldPackagePath, component.CanonicalGoPackagePath),
		logger_domain.String("hashed_name", component.HashedName))
	defer span.End()

	o.interpSemaphore <- struct{}{}
	defer func() { <-o.interpSemaphore }()

	o.interpLock.Lock()
	defer o.interpLock.Unlock()

	shortPackageName, err := extractPackageName(code)
	if err != nil {
		l.Error("No package declaration found in generated code")
		return nil, fmt.Errorf("extracting package name for %q: %w", component.CanonicalGoPackagePath, err)
	}

	if err := o.evaluateCodeInInterpreter(ctx, interpreter, code, component, shortPackageName); err != nil {
		return nil, fmt.Errorf("evaluating code in interpreter for %q: %w", component.CanonicalGoPackagePath, err)
	}

	entries := make(map[string]*templater_adapters.PageEntry)
	linkFn := func(entry *templater_adapters.PageEntry, comp *annotator_dto.VirtualComponent) error {
		return o.linkFunctionsFromRegistry(ctx, entry, comp, shortPackageName)
	}
	if err := o.populateProgCacheForComponent(ctx, manifest, component, componentRelativePath, linkFn, entries); err != nil {
		return nil, fmt.Errorf("linking functions from registry for %q: %w", component.CanonicalGoPackagePath, err)
	}
	return entries, nil
}

// evaluateCodeInInterpreter evaluates the Go code in the interpreter.
//
// Takes interpreter (templater_domain.InterpreterPort) which executes the Go
// code.
// Takes code (string) which contains the Go source code to evaluate.
// Takes component (*annotator_dto.VirtualComponent) which provides package
// metadata.
// Takes shortPackageName (string) which specifies the alias for package
// registration.
//
// Returns error when evaluation fails.
func (o *InterpretedBuildOrchestrator) evaluateCodeInInterpreter(
	ctx context.Context,
	interpreter templater_domain.InterpreterPort,
	code string,
	component *annotator_dto.VirtualComponent,
	shortPackageName string,
) error {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("Evaluating code in interpreter", logger_domain.String("short_pkg_name", shortPackageName))

	o.writeDebugFile(ctx, code, component)
	o.logCodeCharacteristics(ctx, code)

	evalResult, err := interpreter.Eval(ctx, code)
	if err != nil {
		templater_adapters.InterpretedManifestRunnerCompilationErrorCount.Add(ctx, 1)
		l.Error("Interpreter evaluation failed", logger_domain.Error(err))
		return fmt.Errorf("interpreter eval failed for %s: %w", component.CanonicalGoPackagePath, err)
	}
	l.Trace("Code evaluated successfully", logger_domain.String("eval_result_type", fmt.Sprintf("%T", evalResult)))

	if err := interpreter.RegisterPackageAlias(component.CanonicalGoPackagePath, shortPackageName); err != nil {
		l.Warn("Failed to register package alias",
			logger_domain.String("short_pkg_name", shortPackageName),
			logger_domain.Error(err))
	}

	return nil
}

// writeDebugFile writes the generated code to a temporary file for debugging.
//
// Takes ctx (context.Context) which carries the logger.
// Takes code (string) which is the generated Go code to write.
// Takes component (*annotator_dto.VirtualComponent) which identifies the
// component for naming the debug file.
func (o *InterpretedBuildOrchestrator) writeDebugFile(
	ctx context.Context,
	code string,
	component *annotator_dto.VirtualComponent,
) {
	ctx, l := logger_domain.From(ctx, log)
	tmpFileName := fmt.Sprintf("piko_debug_%s.go", component.HashedName)
	tmpFilePath := filepath.Join(os.TempDir(), tmpFileName)

	var tempSandbox safedisk.Sandbox
	var sandboxErr error
	if o.sandboxFactory != nil {
		tempSandbox, sandboxErr = o.sandboxFactory.Create("interp-jit-debug", os.TempDir(), safedisk.ModeReadWrite)
	} else {
		tempSandbox, sandboxErr = safedisk.NewSandbox(os.TempDir(), safedisk.ModeReadWrite)
	}
	if sandboxErr != nil {
		l.Warn("Failed to create sandbox for debug file", logger_domain.Error(sandboxErr))
		return
	}
	defer func() { _ = tempSandbox.Close() }()

	if err := tempSandbox.WriteFile(tmpFileName, []byte(code), debugFilePermission); err != nil {
		l.Warn("Failed to write debug file", logger_domain.Error(err))
	} else {
		l.Trace("[DEBUG] Wrote generated code to temp file", logger_domain.String(fieldPath, tmpFilePath))
	}
}

// logCodeCharacteristics logs which registration calls are in the code.
//
// Takes ctx (context.Context) which carries the logger.
// Takes code (string) which is the source code to check.
func (*InterpretedBuildOrchestrator) logCodeCharacteristics(ctx context.Context, code string) {
	ctx, l := logger_domain.From(ctx, log)
	hasInit := strings.Contains(code, "func init()")
	hasRegister := strings.Contains(code, "RegisterASTFunc")
	l.Trace("[DEBUG] Code has init and register",
		logger_domain.Bool("has_init", hasInit),
		logger_domain.Bool("has_register", hasRegister))
}

// createPageEntry creates a PageEntry from the manifest or as a private
// partial.
//
// Takes ctx (context.Context) which carries the logger.
// Takes manifest (*generator_dto.Manifest) which contains page metadata.
// Takes component (*annotator_dto.VirtualComponent) which provides the source
// info.
//
// Returns *templater_adapters.PageEntry which is either from the manifest or a
// fallback for private partials.
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
		//nolint:exhaustruct // partial init
		entry = &templater_adapters.PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:        component.CanonicalGoPackagePath,
				OriginalSourcePath: relativePath,
			},
		}
	}

	entry.SetBaseDir(o.projectRoot)
	entry.SetJSArtefactToPartialNameMap(buildJSArtefactToPartialNameMap(manifest))
	entry.InitialiseLocalStore()

	return entry
}

// buildJSArtefactToPartialNameMap iterates the manifest's partials and builds
// the mapping from JS artefact IDs to friendly partial names.
//
// This mirrors the logic in processPartials (driven_manifest_store.go) for
// the compiled path.
//
// Takes manifest (*generator_dto.Manifest) which provides the partials to
// iterate.
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

// linkFunctionsFromRegistry gets function pointers from the global registry
// and attaches them to the page entry.
//
// Takes ctx (context.Context) which carries the logger.
// Takes entry (*templater_adapters.PageEntry) which receives the linked
// function pointers.
// Takes component (*annotator_dto.VirtualComponent) which provides the package
// path used to look up functions in the registry.
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

	astFunc, ok := templater_domain.GetASTFunc(registryKey)
	if !ok {
		l.Error("BuildAST function NOT FOUND in global registry after evaluation",
			logger_domain.String("registry_key", registryKey),
			logger_domain.String("canonical_path", component.CanonicalGoPackagePath),
			logger_domain.String("short_pkg_name", shortPackageName))
		return fmt.Errorf("BuildAST not found in global registry for %s (canonical: %s, short name: %s)",
			registryKey, component.CanonicalGoPackagePath, shortPackageName)
	}

	entry.SetASTFunc(astFunc)
	entry.SetCachePolicyFunc(templater_domain.GetCachePolicyFunc(registryKey))
	entry.SetMiddlewareFunc(templater_domain.GetMiddlewareFunc(registryKey))
	entry.SetSupportedLocalesFunc(templater_domain.GetSupportedLocalesFunc(registryKey))

	entry.InitialiseCachedMetadata()

	l.Trace("Component fully linked",
		logger_domain.String("source_path", component.Source.SourcePath),
		logger_domain.Int("functions_linked", linkedFunctionCount))

	return nil
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

// populateProgCacheForComponent writes one or more PageEntry records
// into target for a component that has just been interpreted.
//
// For collection-backed components the .pk file itself has no route;
// the manifest holds a separate entry per virtual instance (for
// example one per markdown post), each with its own concrete route.
// Without this expansion the dev-i runner only registers a single
// entry keyed by the .pk file's relative path, loses the per-instance
// routes, and leaves /blog/post-slug unreachable in interpreted
// mode.
//
// Takes ctx (context.Context) which carries the logger.
// Takes manifest (*generator_dto.Manifest) which holds the per-instance
// manifest entries.
// Takes component (*annotator_dto.VirtualComponent) whose instances
// drive the expansion.
// Takes componentRelativePath (string) which is the .pk file's path
// relative to the project root, used when the component has no
// instances.
// Takes linkFn which performs the function-pointer wiring. Called
// once per emitted entry so that instance-specific caches (like the
// registered AST function) end up attached to each entry.
// Takes target (map[string]*templater_adapters.PageEntry) which
// receives the produced entries.
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
	if len(component.VirtualInstances) == 0 {
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

// createInstancePageEntry builds a per-instance PageEntry using the
// manifest's pre-computed entry for the instance's ManifestKey, which
// carries the concrete RoutePatterns. Falls back to a minimal entry
// when the manifest lookup misses so a broken early-JIT state still
// produces a usable progCache.
//
// Takes manifest (*generator_dto.Manifest) which supplies the per-
// instance page data.
// Takes component (*annotator_dto.VirtualComponent) the instance
// belongs to; provides the fallback PackagePath.
// Takes instance (annotator_dto.VirtualPageInstance) providing the
// manifest key used for lookup.
//
// Returns the prepared entry before function-pointer linking.
func (o *InterpretedBuildOrchestrator) createInstancePageEntry(
	manifest *generator_dto.Manifest,
	component *annotator_dto.VirtualComponent,
	instance annotator_dto.VirtualPageInstance,
) *templater_adapters.PageEntry {
	entry := createPageEntryFromManifest(manifest, instance.ManifestKey)
	if entry == nil {
		//nolint:exhaustruct // partial init for missing manifest data
		entry = &templater_adapters.PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:        component.CanonicalGoPackagePath,
				OriginalSourcePath: instance.ManifestKey,
			},
		}
	}
	entry.SetBaseDir(o.projectRoot)
	entry.SetJSArtefactToPartialNameMap(buildJSArtefactToPartialNameMap(manifest))
	entry.InitialiseLocalStore()
	return entry
}

// createPageEntryFromManifest builds a PageEntry from manifest data by looking
// up the source path in the pages, partials, or emails maps.
//
// data, others use zero values.
//
// Takes manifest (*generator_dto.Manifest) which holds the page, partial, and
// email data to search.
// Takes sourcePath (string) which identifies the entry to find.
//
// Returns *templater_adapters.PageEntry which wraps the found manifest data,
// or nil when the source path is not found in any map.
//
//nolint:exhaustruct // partial init
func createPageEntryFromManifest(manifest *generator_dto.Manifest, sourcePath string) *templater_adapters.PageEntry {
	if pageData, ok := manifest.Pages[sourcePath]; ok {
		return &templater_adapters.PageEntry{ManifestPageEntry: pageData}
	}
	if partialData, ok := manifest.Partials[sourcePath]; ok {
		return &templater_adapters.PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:        partialData.PackagePath,
				OriginalSourcePath: partialData.OriginalSourcePath,
				RoutePatterns:      map[string]string{"": partialData.PartialSrc},
				StyleBlock:         partialData.StyleBlock,
			},
		}
	}
	if emailData, ok := manifest.Emails[sourcePath]; ok {
		return &templater_adapters.PageEntry{
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				PackagePath:         emailData.PackagePath,
				OriginalSourcePath:  emailData.OriginalSourcePath,
				StyleBlock:          emailData.StyleBlock,
				HasSupportedLocales: emailData.HasSupportedLocales,
				LocalTranslations:   emailData.LocalTranslations,
			},
		}
	}
	return nil
}

// getGOROOT finds the Go root directory using a fallback approach.
//
// It first checks the GOROOT environment variable. If that is not set,
// it runs `go env GOROOT` to get the value. If Go is not in PATH, it
// falls back to the runtime.GOROOT function.
//
// Returns string which is the path to the Go root directory.
func getGOROOT() string {
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		return goroot
	}

	command := exec.Command("go", "env", "GOROOT")
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err == nil {
		if result := strings.TrimSpace(out.String()); result != "" {
			return result
		}
	}

	return runtime.GOROOT() //nolint:staticcheck // fallback without go in PATH
}
