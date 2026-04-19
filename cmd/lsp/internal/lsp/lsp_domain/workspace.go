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

package lsp_domain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// workspaceShutdownTimeout caps how long Close waits for tracked workspace
// goroutines to finish so a wedged goroutine cannot block server shutdown.
const workspaceShutdownTimeout = 30 * time.Second

// analysisCompleteParams defines the payload for the custom
// piko/analysisComplete notification. This notification signals to LSP clients
// that analysis for a document has finished, regardless of outcome.
type analysisCompleteParams struct {
	// URI is the document URI that was analysed.
	URI protocol.DocumentURI `json:"uri"`
}

// workspace manages open documents and analysis results for the LSP server.
// It implements WorkspacePort and keeps an in-memory cache of semantic
// analysis results for .pk files.
type workspace struct {
	// coordinator manages analysis across the whole project.
	coordinator coordinator_domain.CoordinatorService

	// moduleManager caches module resolvers and entry points for each module.
	moduleManager *ModuleContextManager

	// client sends diagnostics and notifications to the connected LSP client.
	client protocol.Client

	// conn is the JSON-RPC connection for sending notifications to the client.
	conn jsonrpc2.Conn

	// documents maps document URIs to their state for tracking open files.
	documents map[protocol.DocumentURI]*document

	// typeInspectorManager manages type inspection and provides access to the
	// TypeQuerier.
	typeInspectorManager *inspector_domain.TypeBuilder

	// docCache stores the content of open documents for quick access.
	docCache *DocumentCache

	// actionProviders maps provider names to their action info providers.
	actionProviders map[string]annotator_domain.ActionInfoProvider

	// cancelFuncs maps document URIs to their cancel functions. When a document
	// is closed or re-analysed, the cancel function is called to stop any running
	// analysis for that document.
	cancelFuncs map[protocol.DocumentURI]context.CancelCauseFunc

	// analysisDone maps document URIs to channels that close when analysis
	// finishes.
	analysisDone map[protocol.DocumentURI]chan struct{}

	// cachedProjectResult holds the last full project annotation result.
	// Used as the base for merging targeted rebuild results so that scoped
	// rebuilds only re-annotate affected components.
	cachedProjectResult *annotator_dto.ProjectAnnotationResult

	// reverseDependencyMap maps the project-relative path of an imported
	// component to the project-relative paths of components that depend on it.
	// Rebuilt after every successful analysis via
	// annotator_dto.BuildReverseDependencyMapFromGraph.
	reverseDependencyMap map[string][]string

	// cachedModuleCtx is the module context from the last successful analysis.
	// Stored so scoped rebuilds can reuse it without rediscovery.
	cachedModuleCtx *ModuleContext

	// rootURI is the base path for all files in the workspace folder.
	rootURI protocol.DocumentURI

	// goroutineWG tracks every goroutine spawned by the workspace so Close
	// can wait for them to drain. Without this, server shutdown can race
	// with diagnostic publishes, leaking goroutines under goleak.
	goroutineWG sync.WaitGroup

	// mu guards access to the workspace fields during concurrent operations.
	mu sync.RWMutex

	// hasInitialBuild tracks whether the first full build has completed.
	// Before this is true, all analysis uses the full entry point set.
	hasInitialBuild bool
}

// Close waits for every tracked workspace goroutine to finish.
//
// Has an upper bound so a wedged goroutine cannot block shutdown. Safe to
// call multiple times. Caller is expected to have stopped issuing fresh
// work (e.g. by cancelling the server context) before calling Close.
//
// Takes ctx (context.Context) used for diagnostic logging.
//
// Returns error when the drain timed out; nil on a clean drain.
func (w *workspace) Close(ctx context.Context) error {
	if w == nil {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	done := make(chan struct{})
	go func() {
		defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "lsp.workspace.closeWait")
		w.goroutineWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		l.Debug("LSP workspace goroutines drained")
		return nil
	case <-time.After(workspaceShutdownTimeout):
		l.Warn("LSP workspace goroutines did not drain within timeout",
			logger_domain.String("timeout", workspaceShutdownTimeout.String()))
		return fmt.Errorf("workspace shutdown timed out after %s", workspaceShutdownTimeout)
	}
}

// workspaceDeps bundles dependencies for creating a [workspace].
type workspaceDeps struct {
	// Coordinator manages document coordination across the workspace.
	Coordinator coordinator_domain.CoordinatorService

	// TypeInspectorManager builds and stores type inspectors for semantic
	// analysis.
	TypeInspectorManager *inspector_domain.TypeBuilder

	// ModuleManager provides resolvers and caches entry points for each module.
	ModuleManager *ModuleContextManager

	// Client provides the LSP client interface for sending notifications and
	// requests.
	Client protocol.Client

	// Conn is the JSON-RPC connection used to send messages to the client.
	Conn jsonrpc2.Conn

	// DocCache holds parsed documents for quick lookup by the workspace.
	DocCache *DocumentCache
}

// GetDocument retrieves a document from the workspace cache.
//
// Takes uri (protocol.DocumentURI) which identifies the document to retrieve.
//
// Returns *document which is the cached document if found.
// Returns bool which indicates whether the document exists in the cache.
//
// Safe for concurrent use.
func (w *workspace) GetDocument(uri protocol.DocumentURI) (*document, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	document, exists := w.documents[uri]
	return document, exists
}

// UpdateDocument updates the content of a document and marks it as dirty.
//
// Takes uri (protocol.DocumentURI) which identifies the document to update.
// Takes content ([]byte) which provides the new document content.
//
// Safe for concurrent use; access is protected by a mutex.
func (w *workspace) UpdateDocument(uri protocol.DocumentURI, content []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.docCache.Set(uri, content)

	if existing, exists := w.documents[uri]; exists {
		existing.dirty = true
	} else {
		w.documents[uri] = &document{
			URI:     uri,
			Content: content,
			dirty:   true,
		}
	}
}

// RemoveDocument removes a document from the workspace and clears its
// diagnostics. Called when a file is closed in the editor.
//
// Takes uri (protocol.DocumentURI) which identifies the document to remove.
//
// Safe for concurrent use. Stops any running analysis for the
// document before removing it. Clears diagnostics in a separate goroutine.
func (w *workspace) RemoveDocument(ctx context.Context, uri protocol.DocumentURI) {
	ctx, l := logger_domain.From(ctx, log)

	w.mu.Lock()

	if cancelFunc, exists := w.cancelFuncs[uri]; exists {
		l.Debug("Cancelling in-flight analysis for closed document", logger_domain.String(keyURI, uri.Filename()))
		cancelFunc(fmt.Errorf("document %s removed from workspace", uri))
		delete(w.cancelFuncs, uri)
	}

	if doneChan, exists := w.analysisDone[uri]; exists {
		select {
		case <-doneChan:
		default:
			close(doneChan)
		}
		delete(w.analysisDone, uri)
	}

	delete(w.documents, uri)

	w.docCache.Delete(uri)

	client := w.client
	w.mu.Unlock()

	if client != nil {
		detached := context.WithoutCancel(ctx)
		w.spawnTracked(detached, "lsp.workspace.removeDocument.publishDiagnostics", func() {
			_ = client.PublishDiagnostics(detached, &protocol.PublishDiagnosticsParams{
				URI:         uri,
				Diagnostics: []protocol.Diagnostic{},
			})
		})
	}

	l.Debug("Document removed from workspace", logger_domain.String(keyURI, uri.Filename()))
}

// RunAnalysisForURI orchestrates document analysis for the given URI. It
// checks if the document is dirty, runs coordinator analysis if needed,
// creates a new document with the result, updates internal state, and
// triggers diagnostic publishing.
//
// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
//
// Returns *document which contains the analysis result for the URI.
// Returns error when analysis fails or is cancelled.
//
// Safe for concurrent use. Uses mutex protection when updating the document
// cache. Analysis may be cancelled during rapid edits.
func (w *workspace) RunAnalysisForURI(ctx context.Context, uri protocol.DocumentURI) (*document, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Running analysis for URI", logger_domain.String(keyURI, uri.Filename()))

	if document := w.getCachedCleanDocument(ctx, uri); document != nil {
		return document, nil
	}

	l.Debug("Document is dirty, running analysis", logger_domain.String(keyURI, uri.Filename()))

	analysisCtx, doneChan := w.setupAnalysisContext(ctx, uri)

	defer w.cleanupAnalysisContext(ctx, uri, doneChan)

	moduleCtx, entryPoints, err := w.prepareAnalysisInputs(analysisCtx, uri)
	if err != nil {
		return nil, fmt.Errorf("preparing analysis inputs for %s: %w", uri.Filename(), err)
	}

	w.mu.RLock()
	hasCache := w.hasInitialBuild && w.cachedProjectResult != nil
	w.mu.RUnlock()

	if hasCache {
		doc, scopedErr := w.runScopedAnalysis(ctx, analysisCtx, uri, moduleCtx, entryPoints)
		if scopedErr == nil && doc != nil {
			return doc, nil
		}

		if scopedErr != nil {
			if errors.Is(scopedErr, context.Canceled) || errors.Is(scopedErr, context.DeadlineExceeded) {
				return nil, scopedErr
			}
			l.Debug("Scoped analysis failed, falling back to full build",
				logger_domain.Error(scopedErr),
				logger_domain.String(keyURI, uri.Filename()))
		}
	}

	return w.runFullAnalysis(ctx, analysisCtx, uri, moduleCtx, entryPoints)
}

// GetDocumentForCompletion returns a document suitable for completion
// requests by waiting for any in-flight analysis to complete rather
// than cancelling it, ensuring fresh results even during rapid edits.
//
// Takes uri (protocol.DocumentURI) which identifies the document to
// retrieve for completion.
//
// Returns *document which contains the analysis result for the URI.
// Returns error when analysis fails or is cancelled.
//
// Safe for concurrent use.
func (w *workspace) GetDocumentForCompletion(ctx context.Context, uri protocol.DocumentURI) (*document, error) {
	ctx, l := logger_domain.From(ctx, log)

	w.mu.RLock()
	doneChan, hasInFlight := w.analysisDone[uri]
	w.mu.RUnlock()

	if hasInFlight {
		l.Debug("Completion: waiting for in-flight analysis", logger_domain.String(keyURI, uri.Filename()))
		select {
		case <-doneChan:
			l.Debug("Completion: in-flight analysis completed", logger_domain.String(keyURI, uri.Filename()))
		case <-ctx.Done():
			l.Debug("Completion: context cancelled while waiting", logger_domain.String(keyURI, uri.Filename()))
		}

		w.mu.RLock()
		document, exists := w.documents[uri]
		isDirty := !exists || document.dirty
		w.mu.RUnlock()

		if !isDirty && document != nil {
			return document, nil
		}
	}

	return w.RunAnalysisForURI(ctx, uri)
}

// symbolTarget holds details of a symbol to search for in the codebase.
type symbolTarget struct {
	// sourcePath is the file path where the symbol is defined.
	sourcePath string

	// name is the identifier of the target symbol.
	name string

	// defLocation is the line and column where the symbol is defined.
	defLocation ast_domain.Location
}

// FindAllReferences searches for all references to a symbol across all open
// documents in the workspace. This implements workspace-wide reference search
// as described in Phase 3 of the LSP enhancement plan.
//
// Takes uri (protocol.DocumentURI) which identifies the document containing
// the symbol.
// Takes position (protocol.Position) which specifies the position of the symbol.
//
// Returns []protocol.Location which contains all reference locations found.
// Returns error when the symbol lookup fails.
func (w *workspace) FindAllReferences(ctx context.Context, uri protocol.DocumentURI, position protocol.Position) ([]protocol.Location, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Debug("Finding all references across workspace", logger_domain.String(keyURI, uri.Filename()))

	target, err := w.identifyTargetSymbol(ctx, uri, position)
	if err != nil || target == nil {
		return []protocol.Location{}, nil
	}

	l.Debug("Target symbol identified",
		logger_domain.String("name", target.name),
		logger_domain.String("sourcePath", target.sourcePath),
		logger_domain.Int("defLine", target.defLocation.Line),
		logger_domain.Int("defCol", target.defLocation.Column))

	allLocations := w.searchAllDocuments(target)

	l.Debug("Workspace reference search complete",
		logger_domain.Int("totalReferences", len(allLocations)))

	return allLocations, nil
}

// spawnTracked starts a goroutine that increments the workspace WaitGroup,
// wraps operation in goroutine.RecoverPanic so a single panic cannot crash
// the LSP server, and decrements the WaitGroup on exit.
//
// Takes ctx (context.Context) used by RecoverPanic for OTel attribution; the
// caller is expected to detach from cancellable contexts using
// context.WithoutCancel when the goroutine should outlive the request.
// Takes component (string) which identifies the goroutine in panic logs (use
// the form "lsp.workspace.<name>").
// Takes operation (func()) which is the function body to run.
func (w *workspace) spawnTracked(ctx context.Context, component string, operation func()) {
	w.goroutineWG.Go(func() {
		defer goroutine.RecoverPanic(ctx, component)
		operation()
	})
}

// runFullAnalysis executes a full project build with all entry points. It
// stores the result as the workspace-level cache for future scoped rebuilds.
//
// Takes ctx (context.Context) which is the outer context for diagnostics.
// Takes analysisCtx (context.Context) which is the cancellable analysis
// context.
// Takes uri (protocol.DocumentURI) which identifies the changed document.
// Takes moduleCtx (*ModuleContext) which provides the module root and resolver.
// Takes entryPoints ([]annotator_dto.EntryPoint) which is the full set of
// entry points.
//
// Returns *document which contains the analysis result for the URI.
// Returns error when the build fails or is cancelled.
//
// Safe for concurrent use. Acquires the workspace mutex to update the cached
// project result and documents.
func (w *workspace) runFullAnalysis(
	ctx context.Context,
	analysisCtx context.Context,
	uri protocol.DocumentURI,
	moduleCtx *ModuleContext,
	entryPoints []annotator_dto.EntryPoint,
) (*document, error) {
	_, l := logger_domain.From(ctx, log)

	l.Debug("Running full analysis (all entry points)",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.Int("entryPoints", len(entryPoints)))

	projectResult, err := w.runCoordinatorAnalysis(analysisCtx, uri, entryPoints, moduleCtx.Resolver)
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		cause := context.Cause(analysisCtx)
		if cause == nil {
			cause = err
		}
		l.Debug("Analysis cancelled or timed out for URI (expected during rapid edits)",
			logger_domain.String(keyURI, uri.Filename()),
			logger_domain.String("cause", cause.Error()))
		return nil, err
	}

	if projectResult == nil {
		return w.handleNilProjectResult(ctx, uri, moduleCtx, err)
	}

	if err != nil {
		l.Warn("Analysis completed with errors, but returning partial result for intelligence features",
			logger_domain.Error(err),
			logger_domain.String(keyURI, uri.Filename()))
	}

	w.mu.Lock()
	w.cachedProjectResult = projectResult
	w.cachedModuleCtx = moduleCtx
	w.hasInitialBuild = true
	if projectResult.VirtualModule != nil && projectResult.VirtualModule.Graph != nil {
		w.reverseDependencyMap = annotator_dto.BuildReverseDependencyMapFromGraph(
			projectResult.VirtualModule.Graph,
			moduleCtx.ModuleRoot,
		)
	}
	w.mu.Unlock()

	newDoc := w.createDocumentFromResult(ctx, uri, projectResult, moduleCtx)

	w.mu.Lock()
	w.documents[uri] = newDoc
	w.mu.Unlock()

	w.publishDiagnostics(context.WithoutCancel(ctx), uri, newDoc)

	return newDoc, nil
}

// runScopedAnalysis executes a targeted build that only re-annotates the
// changed component and its transitive dependents. It merges the partial
// result into the cached full project result and publishes diagnostics for
// all affected files.
//
// Takes ctx (context.Context) which is the outer context for diagnostics.
// Takes analysisCtx (context.Context) which is the cancellable analysis
// context.
// Takes uri (protocol.DocumentURI) which identifies the changed document.
// Takes moduleCtx (*ModuleContext) which provides the module root and resolver.
// Takes entryPoints ([]annotator_dto.EntryPoint) which is the full set of
// entry points.
//
// Returns *document which contains the analysis result for the URI, or nil if
// the scoped build could not proceed (caller should fall back to full build).
// Returns error when the build fails or is cancelled.
//
// Safe for concurrent use. Delegates to mergeAndCommitScopedResult for atomic
// cache updates.
func (w *workspace) runScopedAnalysis(
	ctx context.Context,
	analysisCtx context.Context,
	uri protocol.DocumentURI,
	moduleCtx *ModuleContext,
	entryPoints []annotator_dto.EntryPoint,
) (*document, error) {
	_, l := logger_domain.From(ctx, log)

	targetedEntryPoints, affected, err := w.resolveAffectedEntryPoints(uri, moduleCtx, entryPoints)
	if err != nil {
		return nil, err
	}

	if len(targetedEntryPoints) == 0 {
		l.Debug("No matching entry points for scoped build, falling back to full build",
			logger_domain.String(keyURI, uri.Filename()))
		return nil, nil
	}

	l.Debug("Running scoped analysis (targeted entry points)",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.Int("targetedEntryPoints", len(targetedEntryPoints)),
		logger_domain.Int("totalEntryPoints", len(entryPoints)),
		logger_domain.Int("affectedComponents", len(affected)+1))

	targetedResult, buildErr := w.runCoordinatorAnalysis(analysisCtx, uri, targetedEntryPoints, moduleCtx.Resolver)
	if buildErr != nil && (errors.Is(buildErr, context.Canceled) || errors.Is(buildErr, context.DeadlineExceeded)) {
		return nil, buildErr
	}

	if targetedResult == nil {
		l.Debug("Scoped analysis returned nil result, falling back to full build",
			logger_domain.String(keyURI, uri.Filename()))
		return nil, nil
	}

	if buildErr != nil {
		l.Warn("Scoped analysis completed with errors, returning partial result",
			logger_domain.Error(buildErr),
			logger_domain.String(keyURI, uri.Filename()))
	}

	mergedResult := w.mergeAndCommitScopedResult(ctx, uri, targetedResult, affected, moduleCtx)

	newDoc := w.createDocumentFromResult(ctx, uri, mergedResult, moduleCtx)

	w.mu.Lock()
	w.documents[uri] = newDoc
	w.mu.Unlock()

	w.publishDiagnostics(context.WithoutCancel(ctx), uri, newDoc)

	w.publishDiagnosticsForAffectedPaths(context.WithoutCancel(ctx), mergedResult, affected, moduleCtx)

	return newDoc, nil
}

// resolveAffectedEntryPoints determines which entry points are affected by a
// change to the given URI, returning the targeted entry points and the list of
// affected relative paths (excluding the changed file itself).
//
// Takes uri (protocol.DocumentURI) which identifies the changed document.
// Takes moduleCtx (*ModuleContext) which provides the module root.
// Takes entryPoints ([]annotator_dto.EntryPoint) which is the full entry point
// set to filter.
//
// Returns []annotator_dto.EntryPoint which contains the affected entry points.
// Returns []string which holds the relative paths of transitive dependents.
// Returns error when URI-to-path conversion fails.
//
// Safe for concurrent use. Takes a read lock on the workspace mutex.
func (w *workspace) resolveAffectedEntryPoints(
	uri protocol.DocumentURI,
	moduleCtx *ModuleContext,
	entryPoints []annotator_dto.EntryPoint,
) ([]annotator_dto.EntryPoint, []string, error) {
	changedAbsPath, err := uriToPath(uri)
	if err != nil {
		return nil, nil, fmt.Errorf("converting URI to path: %w", err)
	}

	changedRelPath, err := filepath.Rel(moduleCtx.ModuleRoot, changedAbsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("computing relative path: %w", err)
	}
	changedRelPath = filepath.ToSlash(changedRelPath)

	w.mu.RLock()
	reverseDeps := w.reverseDependencyMap
	w.mu.RUnlock()

	affected := annotator_dto.GetTransitiveDependents(reverseDeps, changedRelPath)
	allAffectedRelPaths := append([]string{changedRelPath}, affected...)

	targetedEntryPoints := annotator_dto.FilterEntryPointsByRelativePaths(
		entryPoints, allAffectedRelPaths, moduleCtx.ModuleName,
	)

	return targetedEntryPoints, affected, nil
}

// mergeAndCommitScopedResult atomically merges a targeted (partial) annotation
// result into the cached full project result and commits all workspace state
// changes (cached result, reverse dependency map, affected document
// invalidation) under a single write lock.
//
// This prevents concurrent scoped merges from losing each other's updates:
// without atomicity, goroutine A could read the cache, goroutine B could read
// the same cache, then B's write would overwrite A's merged result.
//
// Takes targetedResult (*annotator_dto.ProjectAnnotationResult) which contains
// results for the subset of components that were re-annotated.
// Takes affectedRelPaths ([]string) which are the transitive dependents to
// invalidate.
// Takes moduleCtx (*ModuleContext) which provides module root and name.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the merged result.
//
// Safe for concurrent use. Acquires the workspace write lock for the entire
// read-merge-write cycle.
func (w *workspace) mergeAndCommitScopedResult(
	_ context.Context,
	_ protocol.DocumentURI,
	targetedResult *annotator_dto.ProjectAnnotationResult,
	affectedRelPaths []string,
	moduleCtx *ModuleContext,
) *annotator_dto.ProjectAnnotationResult {
	w.mu.Lock()
	defer w.mu.Unlock()

	cached := w.cachedProjectResult
	if cached == nil {
		w.cachedProjectResult = targetedResult
		w.cachedModuleCtx = moduleCtx
		return targetedResult
	}

	mergedResult := mergeScopedAnnotationResults(cached, targetedResult)

	w.cachedProjectResult = mergedResult
	w.cachedModuleCtx = moduleCtx
	if mergedResult.VirtualModule != nil && mergedResult.VirtualModule.Graph != nil {
		w.reverseDependencyMap = annotator_dto.BuildReverseDependencyMapFromGraph(
			mergedResult.VirtualModule.Graph,
			moduleCtx.ModuleRoot,
		)
	}

	for _, relPath := range affectedRelPaths {
		absPath := filepath.Join(moduleCtx.ModuleRoot, relPath)
		affectedURI := protocol.DocumentURI("file://" + absPath)
		if doc, exists := w.documents[affectedURI]; exists {
			doc.dirty = true
		}
	}

	return mergedResult
}

// mergeScopedAnnotationResults combines a cached full project result with a
// targeted (partial) rebuild result, producing a merged result that contains
// both the unchanged components from the cache and the updated components
// from the targeted build.
//
// Takes cached (*annotator_dto.ProjectAnnotationResult) which is the full
// project result to merge into.
// Takes targetedResult (*annotator_dto.ProjectAnnotationResult) which contains
// the partial rebuild results.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the merged result.
func mergeScopedAnnotationResults(
	cached *annotator_dto.ProjectAnnotationResult,
	targetedResult *annotator_dto.ProjectAnnotationResult,
) *annotator_dto.ProjectAnnotationResult {
	targetedSourcePaths := make(map[string]bool)
	if targetedResult.VirtualModule != nil && targetedResult.VirtualModule.Graph != nil {
		for _, comp := range targetedResult.VirtualModule.Graph.Components {
			targetedSourcePaths[comp.SourcePath] = true
		}
	}

	mergedComponents := make(map[string]*annotator_dto.AnnotationResult, len(cached.ComponentResults))
	maps.Copy(mergedComponents, cached.ComponentResults)
	maps.Copy(mergedComponents, targetedResult.ComponentResults)

	mergedSourceContents := make(map[string][]byte, len(cached.AllSourceContents))
	maps.Copy(mergedSourceContents, cached.AllSourceContents)
	maps.Copy(mergedSourceContents, targetedResult.AllSourceContents)

	mergedDiagnostics := make([]*ast_domain.Diagnostic, 0, len(cached.AllDiagnostics))
	for _, d := range cached.AllDiagnostics {
		if d.SourcePath != "" && targetedSourcePaths[d.SourcePath] {
			continue
		}
		mergedDiagnostics = append(mergedDiagnostics, d)
	}
	mergedDiagnostics = append(mergedDiagnostics, targetedResult.AllDiagnostics...)

	virtualModule := cached.VirtualModule
	if targetedResult.VirtualModule != nil {
		virtualModule = targetedResult.VirtualModule
	}

	return &annotator_dto.ProjectAnnotationResult{
		ComponentResults:        mergedComponents,
		AllSourceContents:       mergedSourceContents,
		AllDiagnostics:          mergedDiagnostics,
		VirtualModule:           virtualModule,
		FinalAssetManifest:      cached.FinalAssetManifest,
		FinalGeneratedArtefacts: cached.FinalGeneratedArtefacts,
		AnnotatedComponentCount: len(targetedResult.ComponentResults),
		GeneratedArtefactCount:  targetedResult.GeneratedArtefactCount,
	}
}

// publishDiagnosticsForAffectedPaths publishes diagnostics for a set of
// affected component paths beyond the one that was directly edited. This
// ensures that open tabs for dependent files show up-to-date errors after
// a scoped rebuild.
//
// Takes projectResult (*annotator_dto.ProjectAnnotationResult) which contains
// the merged diagnostics.
// Takes affectedRelPaths ([]string) which are the project-relative paths of
// affected components (excluding the directly edited file).
// Takes moduleCtx (*ModuleContext) which provides the module root for path
// conversion.
func (w *workspace) publishDiagnosticsForAffectedPaths(
	ctx context.Context,
	projectResult *annotator_dto.ProjectAnnotationResult,
	affectedRelPaths []string,
	moduleCtx *ModuleContext,
) {
	ctx, l := logger_domain.From(ctx, log)

	client := w.getClient()
	if client == nil {
		return
	}

	for _, relPath := range affectedRelPaths {
		absPath := filepath.Join(moduleCtx.ModuleRoot, relPath)
		affectedURI := protocol.DocumentURI("file://" + absPath)

		lspDiagnostics := convertDiagnosticsToLSP(ctx, projectResult.AllDiagnostics, absPath)

		if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         affectedURI,
			Diagnostics: lspDiagnostics,
		}); err != nil {
			l.Error("Failed to publish diagnostics for affected file",
				logger_domain.Error(err),
				logger_domain.String(keyURI, affectedURI.Filename()))
		}
	}
}

// resetScopedAnalysisCache resets the scoped analysis state, forcing the next
// analysis to perform a full build. Called when structural changes occur (file
// creation, deletion, or rename) that may add or remove entry points.
//
// Safe for concurrent use.
func (w *workspace) resetScopedAnalysisCache() {
	if w == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.hasInitialBuild = false
	w.cachedProjectResult = nil
	w.reverseDependencyMap = make(map[string][]string)
	w.cachedModuleCtx = nil

	if w.moduleManager != nil {
		w.moduleManager.InvalidateAllEntryPoints()
	}
}

// setClient updates the LSP client used for notifications.
//
// Takes client (protocol.Client) which provides the new client instance.
//
// Safe for concurrent use. Acquires the workspace mutex.
func (w *workspace) setClient(client protocol.Client) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.client = client
}

// setConn updates the JSON-RPC connection for notifications.
//
// Takes conn (jsonrpc2.Conn) which provides the new connection.
//
// Safe for concurrent use. Acquires the workspace mutex.
func (w *workspace) setConn(conn jsonrpc2.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.conn = conn
}

// getClient returns the current LSP client for sending notifications.
//
// Returns protocol.Client which may be nil if not yet set.
//
// Safe for concurrent use. Holds a read lock while accessing the client.
func (w *workspace) getClient() protocol.Client {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.client
}

// getConn returns the current JSON-RPC connection for notifications.
//
// Returns jsonrpc2.Conn which may be nil if not yet set.
//
// Safe for concurrent use. Acquires a read lock.
func (w *workspace) getConn() jsonrpc2.Conn {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.conn
}

// identifyTargetSymbol finds the symbol at the given position and returns its
// target info.
//
// Takes uri (protocol.DocumentURI) which specifies the document location.
// Takes position (protocol.Position) which specifies the position within the
// document.
//
// Returns *symbolTarget which contains the symbol's definition location, source
// path, and name.
// Returns error when the document analysis fails.
func (w *workspace) identifyTargetSymbol(ctx context.Context, uri protocol.DocumentURI, position protocol.Position) (*symbolTarget, error) {
	ctx, l := logger_domain.From(ctx, log)

	document, err := w.RunAnalysisForURI(ctx, uri)
	if err != nil || document == nil {
		l.Debug("Failed to get document for reference search", logger_domain.Error(err))
		if err != nil {
			return nil, fmt.Errorf("analysing document for reference search: %w", err)
		}
		return nil, nil
	}

	if document.AnnotationResult == nil || document.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	targetExpr, _ := findExpressionAtPosition(ctx, document.AnnotationResult.AnnotatedAST, position, uri.Filename())
	if targetExpr == nil {
		return nil, nil
	}

	targetAnn := targetExpr.GetGoAnnotation()
	if targetAnn == nil || targetAnn.Symbol == nil {
		return nil, nil
	}

	defLocation := targetAnn.Symbol.ReferenceLocation
	if defLocation.IsSynthetic() {
		return nil, nil
	}

	var sourcePath string
	if targetAnn.OriginalSourcePath != nil {
		sourcePath = *targetAnn.OriginalSourcePath
	}

	return &symbolTarget{
		defLocation: defLocation,
		sourcePath:  sourcePath,
		name:        targetAnn.Symbol.Name,
	}, nil
}

// searchAllDocuments searches all open documents for references to the target
// symbol.
//
// Takes target (*symbolTarget) which specifies the symbol to search for.
//
// Returns []protocol.Location which contains all locations where the target
// symbol is found across documents.
//
// Safe for concurrent use. Takes a read lock while copying the document list,
// then searches each document without holding the lock.
func (w *workspace) searchAllDocuments(target *symbolTarget) []protocol.Location {
	w.mu.RLock()
	documentsToSearch := make([]*document, 0, len(w.documents))
	for _, d := range w.documents {
		documentsToSearch = append(documentsToSearch, d)
	}
	w.mu.RUnlock()

	var allLocations []protocol.Location
	for _, searchDoc := range documentsToSearch {
		locations := searchDoc.findReferencesToSymbol(target.defLocation, target.sourcePath)
		allLocations = append(allLocations, locations...)
	}

	return allLocations
}

// getCachedCleanDocument returns the cached document if it exists and is
// clean, or nil if not found or dirty.
//
// Takes uri (protocol.DocumentURI) which identifies the document to retrieve.
//
// Returns *document which is the cached document, or nil if not found or
// dirty.
//
// Safe for concurrent use. Uses a read lock to access the document cache.
func (w *workspace) getCachedCleanDocument(ctx context.Context, uri protocol.DocumentURI) *document {
	_, l := logger_domain.From(ctx, log)

	w.mu.RLock()
	document, exists := w.documents[uri]
	isDirty := !exists || document.dirty
	w.mu.RUnlock()

	if !isDirty {
		l.Debug("Document is clean, using cached result", logger_domain.String(keyURI, uri.Filename()))
		return document
	}
	return nil
}

// setupAnalysisContext cancels any previous analysis and creates a new
// cancellable context.
//
// Takes ctx (context.Context) which is the parent context for the new
// analysis.
// Takes uri (protocol.DocumentURI) which identifies the document being
// analysed.
//
// Returns context.Context which is the new cancellable analysis context.
// Returns chan struct{} which signals completion when closed.
//
// Safe for concurrent use. Acquires the workspace mutex.
func (w *workspace) setupAnalysisContext(ctx context.Context, uri protocol.DocumentURI) (context.Context, chan struct{}) {
	ctx, l := logger_domain.From(ctx, log)

	w.mu.Lock()
	defer w.mu.Unlock()

	if cancelFunc, exists := w.cancelFuncs[uri]; exists {
		l.Debug("Cancelling previous analysis for URI", logger_domain.String(keyURI, uri.Filename()))
		cancelFunc(errors.New("analysis superseded by new document version"))
		delete(w.cancelFuncs, uri)
	}

	if doneChan, exists := w.analysisDone[uri]; exists {
		select {
		case <-doneChan:
		default:
			close(doneChan)
		}
	}

	analysisCtx, cancel := context.WithCancelCause(ctx)
	w.cancelFuncs[uri] = cancel

	doneChan := make(chan struct{})
	w.analysisDone[uri] = doneChan

	return analysisCtx, doneChan
}

// cleanupAnalysisContext removes the cancel function and signals completion.
//
// Takes uri (protocol.DocumentURI) which identifies the document to clean up.
// Takes doneChan (chan struct{}) which is the channel to close when done.
//
// Safe for concurrent use. Acquires the workspace mutex to modify internal
// maps before signalling completion.
func (w *workspace) cleanupAnalysisContext(ctx context.Context, uri protocol.DocumentURI, doneChan chan struct{}) {
	w.mu.Lock()
	delete(w.cancelFuncs, uri)
	if existingDoneChan, exists := w.analysisDone[uri]; exists && existingDoneChan == doneChan {
		close(doneChan)
		delete(w.analysisDone, uri)
	}
	w.mu.Unlock()

	w.signalAnalysisComplete(context.WithoutCancel(ctx), uri)
}

// signalAnalysisComplete sends the piko/analysisComplete notification.
//
// Takes uri (protocol.DocumentURI) which identifies the document that was
// analysed.
func (w *workspace) signalAnalysisComplete(ctx context.Context, uri protocol.DocumentURI) {
	ctx, l := logger_domain.From(ctx, log)

	conn := w.getConn()
	if conn == nil {
		l.Warn("Cannot send piko/analysisComplete notification: conn is nil",
			logger_domain.String(keyURI, uri.Filename()))
		return
	}

	l.Debug("Sending piko/analysisComplete notification", logger_domain.String(keyURI, uri.Filename()))
	if err := conn.Notify(ctx, "piko/analysisComplete", &analysisCompleteParams{URI: uri}); err != nil {
		l.Error("Failed to send piko/analysisComplete notification",
			logger_domain.Error(err),
			logger_domain.String(keyURI, uri.Filename()))
	}
}

// prepareAnalysisInputs gets the module context for the file and discovers its
// entry points.
//
// Takes uri (protocol.DocumentURI) which identifies the file being analysed.
//
// Returns *ModuleContext which contains the resolver for the file's module.
// Returns []annotator_dto.EntryPoint which contains the entry points for this
// module.
// Returns error when module detection or entry point discovery fails.
func (w *workspace) prepareAnalysisInputs(
	ctx context.Context,
	uri protocol.DocumentURI,
) (moduleCtx *ModuleContext, entryPoints []annotator_dto.EntryPoint, err error) {
	ctx, l := logger_domain.From(ctx, log)

	filePath, err := uriToPath(uri)
	if err != nil {
		err = fmt.Errorf("invalid file URI: %w", err)
		w.publishErrorDiagnosticAsync(ctx, uri, err.Error(), "lsp.workspace.prepareAnalysisInputs.uriToPath")
		return nil, nil, err
	}

	moduleCtx, err = w.moduleManager.GetContextForFile(ctx, filePath)
	if err != nil {
		err = fmt.Errorf("failed to get module context: %w", err)
		w.publishErrorDiagnosticAsync(ctx, uri, err.Error(), "lsp.workspace.prepareAnalysisInputs.moduleContext")
		return nil, nil, err
	}

	entryPoints, err = moduleCtx.GetEntryPoints(ctx)
	if err != nil {
		err = fmt.Errorf("failed to discover project entry points: %w", err)
		w.publishErrorDiagnosticAsync(ctx, uri, err.Error(), "lsp.workspace.prepareAnalysisInputs.entryPoints")
		return nil, nil, err
	}

	if len(entryPoints) == 0 {
		l.Warn("No entry points (.pk files in pages/emails) found in module",
			logger_domain.String("moduleRoot", moduleCtx.ModuleRoot))
	}

	return moduleCtx, entryPoints, nil
}

// runCoordinatorAnalysis runs the coordinator to build the project.
//
// Takes uri (protocol.DocumentURI) which identifies the document to analyse.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the starting
// points for analysis.
// Takes resolver (ResolverPort) which provides path resolution for this build.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the analysis
// results for the project.
// Returns error when the coordinator fails to build the project.
func (w *workspace) runCoordinatorAnalysis(
	ctx context.Context,
	uri protocol.DocumentURI,
	entryPoints []annotator_dto.EntryPoint,
	resolver resolver_domain.ResolverPort,
) (*annotator_dto.ProjectAnnotationResult, error) {
	causationID := fmt.Sprintf("workspace-analysis-%s", uri)

	return w.coordinator.GetOrBuildProject(
		ctx,
		entryPoints,
		coordinator_domain.WithCausationID(causationID),
		coordinator_domain.WithFaultTolerance(),
		coordinator_domain.WithResolver(resolver),
	)
}

// copyActionProviders returns a copy of the action providers map.
//
// Returns map[string]annotator_domain.ActionInfoProvider which is a shallow
// copy of the current action providers.
//
// Safe for concurrent use. Acquires a read lock.
func (w *workspace) copyActionProviders() map[string]annotator_domain.ActionInfoProvider {
	w.mu.RLock()
	defer w.mu.RUnlock()
	providers := make(map[string]annotator_domain.ActionInfoProvider, len(w.actionProviders))
	maps.Copy(providers, w.actionProviders)
	return providers
}

// handleNilProjectResult creates a fallback document when the project result
// is nil.
//
// Takes uri (protocol.DocumentURI) which identifies the document that failed.
// Takes moduleCtx (*ModuleContext) which provides the resolver for the
// document's module.
// Takes analysisErr (error) which is the analysis error, if any.
//
// Returns *document which is a fallback document with cached content.
// Returns error which is always nil.
//
// Safe for concurrent use. Uses a mutex when updating the document cache.
// Publishes error diagnostics in a separate goroutine for non-cancellation
// errors.
func (w *workspace) handleNilProjectResult(
	ctx context.Context,
	uri protocol.DocumentURI,
	moduleCtx *ModuleContext,
	analysisErr error,
) (*document, error) {
	ctx, l := logger_domain.From(ctx, log)

	if analysisErr != nil {
		if errors.Is(analysisErr, context.Canceled) || errors.Is(analysisErr, context.DeadlineExceeded) {
			l.Debug("Analysis cancelled or timed out with no partial result (expected during rapid edits)",
				logger_domain.String(keyURI, uri.Filename()),
				logger_domain.Error(analysisErr))
		} else {
			l.Error("Analysis failed with no partial result",
				logger_domain.Error(analysisErr),
				logger_domain.String(keyURI, uri.Filename()),
				logger_domain.String("error_type", fmt.Sprintf("%T", analysisErr)))
			w.publishErrorDiagnosticAsync(ctx, uri, fmt.Sprintf("Analysis failed: %s", analysisErr.Error()), "lsp.workspace.handleNilProjectResult.analysisFailed")
		}
	} else {
		l.Error("Analysis produced no result", logger_domain.String(keyURI, uri.Filename()))
		w.publishErrorDiagnosticAsync(ctx, uri, "Analysis completed but produced no result", "lsp.workspace.handleNilProjectResult.noResult")
	}

	content, _ := w.docCache.Get(uri)
	errorDoc := &document{
		URI:      uri,
		Content:  content,
		Resolver: moduleCtx.Resolver,
		dirty:    false,
	}

	w.mu.Lock()
	w.documents[uri] = errorDoc
	w.mu.Unlock()

	return errorDoc, nil
}

// createDocumentFromResult builds a document from the project result.
//
// Takes uri (protocol.DocumentURI) which identifies the document to create.
// Takes projectResult (*annotator_dto.ProjectAnnotationResult) which provides
// the annotation data for the project.
// Takes moduleCtx (*ModuleContext) which provides the resolver for the
// document's module.
//
// Returns *document which contains the built document with its content,
// annotations, and analysis data.
func (w *workspace) createDocumentFromResult(
	ctx context.Context,
	uri protocol.DocumentURI,
	projectResult *annotator_dto.ProjectAnnotationResult,
	moduleCtx *ModuleContext,
) *document {
	content, _ := w.docCache.Get(uri)
	filePath, _ := uriToPath(uri)
	annotationResult := w.extractAnnotationResultForURI(projectResult, filePath)

	w.logAnnotationResultStatus(ctx, uri, filePath, projectResult, annotationResult)

	analysisMap := w.extractTypedAnalysisMap(ctx, uri, annotationResult)
	typeQuerier := w.getTypeQuerier(ctx)

	return &document{
		URI:              uri,
		Content:          content,
		ProjectResult:    projectResult,
		AnnotationResult: annotationResult,
		AnalysisMap:      analysisMap,
		TypeInspector:    typeQuerier,
		Resolver:         moduleCtx.Resolver,
		dirty:            false,
	}
}

// logAnnotationResultStatus logs debug details about the annotation
// extraction process.
//
// Takes uri (protocol.DocumentURI) which identifies the document being
// processed.
// Takes filePath (string) which provides the file path for logging.
// Takes projectResult (*annotator_dto.ProjectAnnotationResult) which holds
// the project-level annotation data.
// Takes annotationResult (*annotator_dto.AnnotationResult) which holds the
// extracted annotation, or nil if extraction failed.
func (*workspace) logAnnotationResultStatus(
	ctx context.Context,
	uri protocol.DocumentURI,
	filePath string,
	projectResult *annotator_dto.ProjectAnnotationResult,
	annotationResult *annotator_dto.AnnotationResult,
) {
	_, l := logger_domain.From(ctx, log)

	if annotationResult == nil {
		l.Warn("extractAnnotationResultForURI returned nil",
			logger_domain.String(keyURI, uri.Filename()),
			logger_domain.String("filePath", filePath),
			logger_domain.Bool("has_virtual_module", projectResult.VirtualModule != nil),
			logger_domain.Bool("has_graph", projectResult.VirtualModule != nil && projectResult.VirtualModule.Graph != nil))

		if projectResult.VirtualModule != nil && projectResult.VirtualModule.Graph != nil {
			l.Debug("Available paths in PathToHashedName:",
				logger_domain.Int("count", len(projectResult.VirtualModule.Graph.PathToHashedName)))
			for modulePath := range projectResult.VirtualModule.Graph.PathToHashedName {
				l.Debug("  - Available path:", logger_domain.String("path", modulePath))
			}
		}
	} else {
		l.Debug("Successfully extracted annotationResult",
			logger_domain.String(keyURI, uri.Filename()),
			logger_domain.Bool("has_annotated_ast", annotationResult.AnnotatedAST != nil))
	}
}

// extractTypedAnalysisMap extracts and type-asserts the analysis map from the
// annotation result.
//
// Takes uri (protocol.DocumentURI) which identifies the document for logging.
// Takes annotationResult (*annotator_dto.AnnotationResult) which contains the
// analysis map to extract.
//
// Returns map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext which
// maps template nodes to their analysis contexts, or nil if the result is nil
// or type assertion fails.
func (*workspace) extractTypedAnalysisMap(
	ctx context.Context,
	uri protocol.DocumentURI,
	annotationResult *annotator_dto.AnnotationResult,
) map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext {
	_, l := logger_domain.From(ctx, log)

	if annotationResult == nil || annotationResult.AnalysisMap == nil {
		return nil
	}

	typedMap, ok := annotationResult.AnalysisMap.(map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext)
	if !ok {
		l.Warn("AnalysisMap type assertion failed", logger_domain.String(keyURI, uri.Filename()))
		return nil
	}

	l.Debug("AnalysisMap extracted successfully",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.Int("entries", len(typedMap)))

	return typedMap
}

// getTypeQuerier retrieves the TypeQuerier from the TypeInspectorManager.
//
// Returns TypeInspectorPort which provides type lookup services, or nil when
// the manager is unavailable or the querier cannot be obtained.
func (w *workspace) getTypeQuerier(ctx context.Context) TypeInspectorPort {
	_, l := logger_domain.From(ctx, log)

	if w.typeInspectorManager == nil {
		return nil
	}

	typeQuerier, ok := w.typeInspectorManager.GetQuerier()
	if !ok {
		l.Warn("Failed to get TypeQuerier from TypeInspectorManager")
		return nil
	}

	return typeQuerier
}

// extractAnnotationResultForURI extracts the annotation result for a file from
// the project result using the VirtualModule mapping.
//
// Takes projectResult (*annotator_dto.ProjectAnnotationResult) which contains
// the full project annotation data including the VirtualModule graph.
// Takes absPath (string) which is the absolute file path used as a key in
// PathToHashedName.
//
// Returns *annotator_dto.AnnotationResult which is the annotation result for
// the file, or nil if the file is not found in the project.
func (*workspace) extractAnnotationResultForURI(
	projectResult *annotator_dto.ProjectAnnotationResult,
	absPath string,
) *annotator_dto.AnnotationResult {
	if projectResult.VirtualModule == nil || projectResult.VirtualModule.Graph == nil {
		return nil
	}

	hashedName, ok := projectResult.VirtualModule.Graph.PathToHashedName[absPath]
	if !ok {
		return nil
	}

	compResult, ok := projectResult.ComponentResults[hashedName]
	if !ok {
		return nil
	}

	return compResult
}

// publishDiagnostics sends diagnostics for a document to the LSP client.
//
// Takes uri (protocol.DocumentURI) which identifies the document.
// Takes document (*document) which contains the project result with diagnostics.
func (w *workspace) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, document *document) {
	ctx, l := logger_domain.From(ctx, log)

	l.Info("publishDiagnostics: Starting",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.Bool("hasProjectResult", document.ProjectResult != nil))

	client := w.getClient()
	if client == nil {
		l.Warn("publishDiagnostics: client is nil, skipping",
			logger_domain.String(keyURI, uri.Filename()))
		return
	}

	if document.ProjectResult == nil {
		l.Info("publishDiagnostics: No project result, clearing diagnostics")
		_ = client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []protocol.Diagnostic{},
		})
		return
	}

	filePath, _ := uriToPath(uri)

	lspDiagnostics := convertDiagnosticsToLSP(ctx, document.ProjectResult.AllDiagnostics, filePath)

	l.Info("publishDiagnostics: Publishing diagnostics to client",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.Int("totalDiagnostics", len(document.ProjectResult.AllDiagnostics)),
		logger_domain.Int("filteredDiagnostics", len(lspDiagnostics)))

	if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: lspDiagnostics,
	}); err != nil {
		l.Error("publishDiagnostics: Error publishing", logger_domain.Error(err))
	} else {
		l.Info("publishDiagnostics: Successfully published")
	}
}

// publishErrorDiagnosticAsync schedules publishErrorDiagnostic on a tracked
// goroutine using a context that survives the caller's cancellation. Replaces
// bare `go w.publishErrorDiagnostic(...)` so that workspace shutdown can drain
// in-flight publishes via Close.
//
// Takes ctx (context.Context) which is detached via context.WithoutCancel so
// the publish completes even after the caller's request finishes.
// Takes uri (protocol.DocumentURI) which identifies the document to report
// the error against.
// Takes message (string) which describes the error to show to the user.
// Takes component (string) which identifies the call site for panic
// attribution (use the form "lsp.workspace.<context>").
func (w *workspace) publishErrorDiagnosticAsync(ctx context.Context, uri protocol.DocumentURI, message, component string) {
	detached := context.WithoutCancel(ctx)
	w.spawnTracked(detached, component, func() {
		w.publishErrorDiagnostic(detached, uri, message)
	})
}

// publishErrorDiagnostic sends an error message to the user through the LSP.
// This is used when the coordinator or other key parts fail, so the user sees
// helpful feedback instead of no response.
//
// Takes uri (protocol.DocumentURI) which identifies the document to report
// the error against.
// Takes message (string) which describes the error to show to the user.
func (w *workspace) publishErrorDiagnostic(ctx context.Context, uri protocol.DocumentURI, message string) {
	ctx, l := logger_domain.From(ctx, log)

	l.Info("publishErrorDiagnostic: Publishing error diagnostic",
		logger_domain.String(keyURI, uri.Filename()),
		logger_domain.String("message", message))

	client := w.getClient()
	if client == nil {
		l.Warn("publishErrorDiagnostic: client is nil, skipping",
			logger_domain.String(keyURI, uri.Filename()))
		return
	}

	errorDiagnostic := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 1},
		},
		Severity: protocol.DiagnosticSeverityError,
		Source:   "piko-lsp",
		Message:  message,
	}

	if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{errorDiagnostic},
	}); err != nil {
		l.Error("publishErrorDiagnostic: Error publishing", logger_domain.Error(err))
	} else {
		l.Info("publishErrorDiagnostic: Successfully published error diagnostic")
	}
}

// sourceDirInfo describes a source directory for entry point discovery.
type sourceDirInfo struct {
	// Dir is the path relative to the project root; empty means not set.
	Dir string

	// IsPage indicates whether this directory contains a page entry point.
	IsPage bool

	// IsEmail indicates whether this directory contains email templates.
	IsEmail bool

	// IsPublic indicates whether the entry point serves public assets.
	IsPublic bool
}

// newWorkspace creates a new workspace instance.
//
// Takes deps (workspaceDeps) which provides the required dependencies.
// Takes rootURI (protocol.DocumentURI) which specifies the workspace root.
//
// Returns *workspace which is the set up workspace ready for use.
func newWorkspace(deps workspaceDeps, rootURI protocol.DocumentURI) *workspace {
	return &workspace{
		documents:            make(map[protocol.DocumentURI]*document),
		coordinator:          deps.Coordinator,
		typeInspectorManager: deps.TypeInspectorManager,
		moduleManager:        deps.ModuleManager,
		client:               deps.Client,
		conn:                 deps.Conn,
		rootURI:              rootURI,
		docCache:             deps.DocCache,
		actionProviders:      make(map[string]annotator_domain.ActionInfoProvider),
		cancelFuncs:          make(map[protocol.DocumentURI]context.CancelCauseFunc),
		analysisDone:         make(map[protocol.DocumentURI]chan struct{}),
		reverseDependencyMap: make(map[string][]string),
	}
}

// isValidEntryPointFile checks if a directory entry is a valid .pk or .pkc
// entry point file.
//
// Takes d (fs.DirEntry) which is the directory entry to check.
//
// Returns bool which is true if the entry is a non-private .pk file.
func isValidEntryPointFile(d fs.DirEntry) bool {
	if d.IsDir() {
		return false
	}
	if !strings.HasSuffix(d.Name(), ".pk") && !strings.HasSuffix(d.Name(), pkcFileExtension) {
		return false
	}
	if strings.HasPrefix(d.Name(), "_") {
		return false
	}
	return true
}
