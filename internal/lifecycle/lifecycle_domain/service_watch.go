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

package lifecycle_domain

// This file contains file-watching, event handling, entry-point bookkeeping, and asset
// change processing for the lifecycle service.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
)

// watchLoop processes file change events from the watcher channel until the context is
// cancelled or the stop signal is received.
//
// Takes events (<-chan lifecycle_dto.FileEvent) which provides file change events from
// the file watcher.
func (ls *lifecycleService) watchLoop(ctx context.Context, events <-chan lifecycle_dto.FileEvent) {
	ctx, span, l := log.Span(ctx, "watchLoop")
	defer span.End()
	defer goroutine.RecoverPanic(ctx, "lifecycle.watchLoop")

	l.Internal("File watch loop started.")
	defer l.Internal("File watch loop finished.")

	for {
		select {
		case <-ctx.Done():
			l.Trace("Context done, exiting watch loop.")
			return
		case <-ls.stopChan:
			l.Trace("Stop signal received, exiting watch loop.")
			return
		case event, ok := <-events:
			if !ok {
				l.Trace("Events channel closed, exiting watch loop.")
				return
			}
			l.Trace("Received file event", logger_domain.String(fieldPath, event.Path))
			ls.handleFileEvent(ctx, event, false)
		}
	}
}

// fileEventContext holds the data needed to process a file change event.
type fileEventContext struct {
	// ctx is the request-scoped context for cancellation and deadlines.
	ctx context.Context

	// relPath is the file path relative to the project root.
	relPath string

	// artefactID is the unique identifier for the asset artefact in the registry.
	artefactID string

	// event is the file event that triggered this context.
	event lifecycle_dto.FileEvent
}

// handleFileEvent processes a file change event.
//
// Takes event (lifecycle_dto.FileEvent) which describes the file system event.
// Takes initialSeed (bool) which indicates whether this is part of the initial scan.
func (ls *lifecycleService) handleFileEvent(ctx context.Context, event lifecycle_dto.FileEvent, initialSeed bool) {
	ctx, span, _ := log.Span(ctx, "handleFileEvent",
		logger_domain.String(fieldPath, event.Path),
		logger_domain.Bool("initialSeed", initialSeed),
	)
	defer span.End()

	if !initialSeed && ls.maybeHandleStyleDependencyChange(ctx, event) {
		return
	}

	fec, ok := ls.buildFileEventContext(ctx, event)
	if !ok {
		return
	}

	if isCoreSourceFile(fec.relPath, &ls.pathsConfig) {
		ls.handleCoreSourceChange(fec, initialSeed)
		return
	}

	ls.handleAssetChange(fec)
}

// maybeHandleStyleDependencyChange checks whether the changed file is an external
// stylesheet imported by one or more components via CSS @import. If so it triggers a
// rebuild of the importing components (a targeted rebuild in dev-i, otherwise a full
// rebuild) and returns true.
//
// Takes event (lifecycle_dto.FileEvent) which describes the file system event.
//
// Returns bool which is true when the event was handled as a stylesheet dependency
// change.
//
// Concurrency: read-locks styleDepsMu via importersOfStyle and may launch a rebuild
// goroutine tracked by rebuildWG.
func (ls *lifecycleService) maybeHandleStyleDependencyChange(ctx context.Context, event lifecycle_dto.FileEvent) bool {
	if !strings.HasSuffix(strings.ToLower(event.Path), ".css") {
		return false
	}
	importers := ls.importersOfStyle(event.Path)
	if len(importers) == 0 {
		return false
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Imported stylesheet changed, rebuilding importing components",
		logger_domain.String(fieldPath, event.Path),
		logger_domain.Int("importer_count", len(importers)))

	if ls.interpretedOrchestrator != nil && ls.interpretedOrchestrator.IsInitialised() && ls.coordinatorService != nil {
		cssRelPath := event.Path
		if rel, err := ls.fs.Rel(ls.pathsConfig.BaseDir, event.Path); err == nil {
			cssRelPath = filepath.ToSlash(rel)
		}
		ls.rebuildWG.Add(1)
		go ls.executeStyleDependencyRebuild(ctx, cssRelPath, importers)
		return true
	}

	if ls.coordinatorService != nil {
		ls.RequestRebuild(ctx, fmt.Sprintf("style-change:%s", event.Path))
		return true
	}

	return false
}

// buildFileEventContext creates the context needed to process a file event.
//
// Takes event (lifecycle_dto.FileEvent) which contains the file event details.
//
// Returns fileEventContext which holds the processed event context.
// Returns bool which is false if the event should be skipped.
func (ls *lifecycleService) buildFileEventContext(ctx context.Context, event lifecycle_dto.FileEvent) (fileEventContext, bool) {
	ctx, l := logger_domain.From(ctx, log)
	relPath, err := ls.fs.Rel(ls.pathsConfig.BaseDir, event.Path)
	if err != nil {
		l.Warn("Failed to compute relative path", logger_domain.Error(err))
		return fileEventContext{}, false
	}
	relPathSlash := filepath.ToSlash(relPath)

	if !isRelevantFileForProcessing(relPathSlash, &ls.pathsConfig) {
		l.Trace("Ignoring event for irrelevant file")
		return fileEventContext{}, false
	}

	moduleName := ""
	if ls.resolver != nil {
		moduleName = ls.resolver.GetModuleName()
	}

	return fileEventContext{
		ctx:        ctx,
		event:      event,
		relPath:    relPathSlash,
		artefactID: moduleName + "/" + relPathSlash,
	}, true
}

// handleCoreSourceChange processes changes to core source files (.pk, etc.).
//
// In interpreted mode with an initialised orchestrator, this triggers a targeted rebuild
// that only re-annotates and regenerates the changed component and its transitive
// dependents. Otherwise it falls back to a full coordinator rebuild.
//
// Takes fec (fileEventContext) which provides the file event details and logging context.
// Takes initialSeed (bool) which indicates whether this is the initial file discovery
// phase.
//
// Safe for concurrent use. Targeted rebuilds are spawned in a separate goroutine.
func (ls *lifecycleService) handleCoreSourceChange(fec fileEventContext, initialSeed bool) {
	ctx, l := logger_domain.From(fec.ctx, log)

	ls.updateBuildContext(ctx, fec.event, fec.relPath)

	ext := strings.ToLower(filepath.Ext(fec.relPath))
	if ext == ".pkc" && !initialSeed {
		l.Trace("PKC file changed, upserting artefact for recompilation.")
		ls.clearComponentCacheIfNeeded(fec)
		ls.upsertAssetArtefact(fec)
	}

	if initialSeed {
		return
	}

	isPageRemoval := isRemovalEvent(fec.event.Type) && strings.HasSuffix(strings.ToLower(fec.relPath), ".pk")
	isPageCreate := fec.event.Type == lifecycle_dto.FileEventTypeCreate && strings.HasSuffix(strings.ToLower(fec.relPath), ".pk")

	if ls.interpretedOrchestrator != nil && ls.interpretedOrchestrator.IsInitialised() && ls.coordinatorService != nil {
		if isPageRemoval {
			ls.handlePageRemoval(ctx, fec.relPath)
			return
		}
		ls.rebuildWG.Add(1)
		go ls.executeTargetedRebuild(ctx, fec.relPath, isPageCreate)
		return
	}

	if isPageRemoval {
		ls.removeComponentStyleDeps(fec.relPath)
	}

	if ls.buildCacheInvalidator != nil {
		l.Trace("Core source file changed, invalidating JIT build cache.")
		ls.buildCacheInvalidator.InvalidateBuildCache()
		return
	}

	if ls.coordinatorService != nil {
		l.Trace("Core source file changed, requesting rebuild.")
		ls.RequestRebuild(ctx, fmt.Sprintf("file-change:%s", fec.relPath))
	}
}

// isRemovalEvent reports whether a file event type represents the disappearance of a file
// (a removal, or a rename away that the watcher could not resolve to a create).
//
// Takes eventType (lifecycle_dto.FileEventType) which is the event type to classify.
//
// Returns bool which is true for remove and rename events.
func isRemovalEvent(eventType lifecycle_dto.FileEventType) bool {
	return eventType == lifecycle_dto.FileEventTypeRemove || eventType == lifecycle_dto.FileEventTypeRename
}

// handlePageRemoval cleans up after a removed or renamed-away .pk page.
//
// It drops the component's stale CSS @import dependency entry and, in dev-i mode, reloads
// the router so the removed page's route disappears and any page created by the matching
// rename registers. The entry point itself was already dropped by updateBuildContext.
//
// Takes relPath (string) which is the project-relative path of the removed page (e.g.
// "pages/old.pk").
func (ls *lifecycleService) handlePageRemoval(ctx context.Context, relPath string) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Page removed, cleaning up dependencies and reloading routes",
		logger_domain.String(fieldPath, relPath))

	ls.removeComponentStyleDeps(relPath)

	if ls.interpretedOrchestrator != nil && ls.interpretedOrchestrator.IsInitialised() {
		ls.interpretedOrchestrator.RemoveComponent(ctx, relPath)

		if runner := ls.currentRunnerSnapshot(); ls.routerManager != nil && runner != nil {
			ls.reloadRoutesIfNeeded(ctx, runner)
		}
	}
}

// handleAssetChange processes changes to asset files.
//
// Takes fec (fileEventContext) which provides the file event details and logging context.
func (ls *lifecycleService) handleAssetChange(fec fileEventContext) {
	ctx, l := logger_domain.From(fec.ctx, log)

	l.Trace("Asset file changed, processing via orchestrator pipeline.")

	switch fec.event.Type {
	case lifecycle_dto.FileEventTypeCreate, lifecycle_dto.FileEventTypeWrite:
		ls.upsertAssetArtefact(fec)
	case lifecycle_dto.FileEventTypeRemove, lifecycle_dto.FileEventTypeRename:
		ls.deleteAssetArtefact(fec)
	default:
		l.Warn("Unknown file event type for asset change",
			logger_domain.Int("event_type", int(fec.event.Type)),
			logger_domain.String("path", fec.event.Path))
		return
	}

	ls.clearSvgCacheIfNeeded(fec)

	ls.notifyAssetReload(ctx, fec.relPath)
}

// notifyAssetReload broadcasts a build-completion event so the browser live-reloads after
// an asset is reprocessed. It is a no-op outside dev/dev-i modes where the notifier is
// nil.
//
// Takes relPath (string) which is the project-relative path of the changed asset (e.g.
// "lib/icon.svg").
func (ls *lifecycleService) notifyAssetReload(ctx context.Context, relPath string) {
	if ls.devEventNotifier == nil {
		return
	}
	ls.devEventNotifier.NotifyRebuildComplete(ctx, []string{relPath})
}

// clearSvgCacheIfNeeded clears the SVG cache when the changed file is an SVG.
//
// Takes fec (fileEventContext) which provides the file event and artefact ID.
func (ls *lifecycleService) clearSvgCacheIfNeeded(fec fileEventContext) {
	if ls.renderRegistryPort == nil {
		return
	}
	ext := strings.ToLower(filepath.Ext(fec.event.Path))
	if ext == ".svg" {
		ls.renderRegistryPort.ClearSvgCache(fec.ctx, fec.artefactID)
	}
}

// clearComponentCacheIfNeeded clears the component cache for PKC files. The component tag
// name is taken from the file name without the extension.
//
// Takes fec (fileEventContext) which provides the file event and relative path.
func (ls *lifecycleService) clearComponentCacheIfNeeded(fec fileEventContext) {
	if ls.renderRegistryPort == nil {
		return
	}
	ext := strings.ToLower(filepath.Ext(fec.event.Path))
	if ext == ".pkc" {
		ctx, l := logger_domain.From(fec.ctx, log)

		tagName := strings.TrimSuffix(filepath.Base(fec.relPath), ext)
		l.Trace("Clearing component cache for PKC file",
			logger_domain.String("tagName", tagName))
		ls.renderRegistryPort.ClearComponentCache(ctx, tagName)
	}
}

// upsertAssetArtefact creates or updates an asset artefact in the registry.
//
// Takes fec (fileEventContext) which provides the file event details and context for the
// operation.
func (ls *lifecycleService) upsertAssetArtefact(fec fileEventContext) {
	ctx, l := logger_domain.From(fec.ctx, log)

	file, err := ls.fs.Open(fec.event.Path)
	if err != nil {
		l.Error("Failed to read updated asset file", logger_domain.Error(err))
		return
	}
	defer func() { _ = file.Close() }()

	profiles := GetProfilesForFile(fec.artefactID, ResolverModuleName(ls.resolver), nil)
	normalisedID := NormaliseAssetArtefactID(fec.artefactID)
	_, err = ls.registryService.UpsertArtefact(ctx, normalisedID, fec.relPath, file, "local_disk_cache", profiles)
	if err != nil {
		l.Error("Failed to upsert asset artefact", logger_domain.Error(err))
	}
}

// deleteAssetArtefact removes an asset artefact from the registry.
//
// Takes fec (fileEventContext) which provides the context and artefact ID.
func (ls *lifecycleService) deleteAssetArtefact(fec fileEventContext) {
	ctx, l := logger_domain.From(fec.ctx, log)

	normalisedID := NormaliseAssetArtefactID(fec.artefactID)
	err := ls.registryService.DeleteArtefact(ctx, normalisedID)
	if err != nil && !errors.Is(err, registry_domain.ErrArtefactNotFound) {
		l.Error("Failed to delete asset artefact", logger_domain.Error(err))
	}
}

// componentType represents the kind of component file, such as page, partial, or email
// template.
type componentType struct {
	// isPage indicates whether the component is a page template.
	isPage bool

	// isPartial indicates the file is in the partials source folder.
	isPartial bool

	// isEmail indicates whether the component is an email template.
	isEmail bool
}

// updateBuildContext adds or removes entry points based on file events.
//
// Takes event (lifecycle_dto.FileEvent) which specifies the file system event that
// happened.
// Takes relPath (string) which is the path to the affected file, relative to the project
// root.
//
// Safe for concurrent use; protects build context updates with a mutex.
func (ls *lifecycleService) updateBuildContext(ctx context.Context, event lifecycle_dto.FileEvent, relPath string) {
	if ls.resolver == nil || !strings.HasSuffix(strings.ToLower(relPath), ".pk") {
		return
	}

	compType := ls.determineComponentType(relPath)
	if !compType.isPage && !compType.isPartial && !compType.isEmail {
		return
	}

	entryPointPath := filepath.ToSlash(filepath.Join(ls.resolver.GetModuleName(), relPath))

	ls.mu.Lock()
	defer ls.mu.Unlock()

	switch event.Type {
	case lifecycle_dto.FileEventTypeCreate:
		ls.addEntryPointIfNotExists(ctx, entryPointPath, compType)
	case lifecycle_dto.FileEventTypeRemove, lifecycle_dto.FileEventTypeRename:
		ls.removeEntryPoint(entryPointPath)
	case lifecycle_dto.FileEventTypeWrite:
	default:
		_, l := logger_domain.From(ctx, log)
		l.Warn("Unknown file event type for component change",
			logger_domain.Int("event_type", int(event.Type)),
			logger_domain.String("path", entryPointPath))
	}
}

// determineComponentType determines the type of component based on its path.
//
// Takes relPath (string) which is the relative path to check against configured source
// directories.
//
// Returns componentType which indicates whether the path matches a page, partial, or
// email source directory.
func (ls *lifecycleService) determineComponentType(relPath string) componentType {
	paths := &ls.pathsConfig
	var ct componentType

	if paths.PagesSourceDir != "" && hasPrefix(relPath, paths.PagesSourceDir) {
		ct.isPage = true
	} else if paths.PartialsSourceDir != "" && hasPrefix(relPath, paths.PartialsSourceDir) {
		ct.isPartial = true
	} else if paths.EmailsSourceDir != "" && hasPrefix(relPath, paths.EmailsSourceDir) {
		ct.isEmail = true
	}

	return ct
}

// addEntryPointIfNotExists adds an entry point if it is not already present.
//
// Takes entryPointPath (string) which specifies the path of the entry point.
// Takes compType (componentType) which describes the component type.
//
// The caller must hold ls.mu.
func (ls *lifecycleService) addEntryPointIfNotExists(ctx context.Context, entryPointPath string, compType componentType) {
	for _, ep := range ls.entryPoints {
		if ep.Path == entryPointPath {
			return
		}
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("New component file created, adding to entry points.",
		logger_domain.String(fieldPath, entryPointPath))
	ls.entryPoints = append(ls.entryPoints, annotator_dto.EntryPoint{
		Path:              entryPointPath,
		IsPage:            compType.isPage,
		IsEmail:           compType.isEmail,
		IsPublic:          compType.isPage,
		VirtualPageSource: nil,
	})
}

// removeEntryPoint removes an entry point by its path.
//
// Takes entryPointPath (string) which specifies the path of the entry point to remove.
//
// Must be called with ls.mu held.
func (ls *lifecycleService) removeEntryPoint(entryPointPath string) {
	newEntryPoints := make([]annotator_dto.EntryPoint, 0, len(ls.entryPoints))
	for _, ep := range ls.entryPoints {
		if ep.Path != entryPointPath {
			newEntryPoints = append(newEntryPoints, ep)
		}
	}
	ls.entryPoints = newEntryPoints
}

// executeTargetedRebuild performs an incremental rebuild for a single changed file by
// only re-annotating and regenerating the changed component and its transitive
// dependents.
//
// It queries the orchestrator's reverse dependency map to find affected components,
// filters the entry point list to only those components, runs a synchronous coordinator
// build with the targeted subset, then merges the result and proactively JIT-compiles all
// dirty components.
//
// Falls back to a full coordinator rebuild if the targeted build fails.
//
// Takes relPath (string) which is the project-relative path of the changed file (e.g.
// "pages/login.pk").
// Takes reloadRoutes (bool) which requests a router reload after recompilation so a newly
// created page becomes routable without a restart.
//
// Designed to run in a goroutine so the watch loop is not blocked.
//
// The caller must Add(1) on rebuildWG before launching this goroutine, and the rebuild
// calls Done() on the WaitGroup on exit.
func (ls *lifecycleService) executeTargetedRebuild(ctx context.Context, relPath string, reloadRoutes bool) {
	defer ls.rebuildWG.Done()
	defer goroutine.RecoverPanic(ctx, "lifecycle.executeTargetedRebuild")

	affected := ls.interpretedOrchestrator.GetAffectedComponents(relPath)

	allPaths := make([]string, 0, 1+len(affected))
	allPaths = append(allPaths, relPath)
	allPaths = append(allPaths, affected...)

	ls.runTargetedRebuild(ctx, relPath, allPaths, reloadRoutes)
}

// executeStyleDependencyRebuild rebuilds the components that import a changed external
// stylesheet, plus their transitive dependents. The coordinator's input hash now folds in
// imported-stylesheet contents, so the changed CSS makes the rebuild miss the cache and
// re-inline the new styles.
//
// Takes cssRelPath (string) which is the project-relative path of the changed stylesheet,
// used for logging and build causation.
// Takes importers ([]string) which are the project-relative paths of components that
// import the stylesheet.
//
// Designed to run in a goroutine; the caller must Add(1) on rebuildWG before launching
// it.
func (ls *lifecycleService) executeStyleDependencyRebuild(ctx context.Context, cssRelPath string, importers []string) {
	defer ls.rebuildWG.Done()
	defer goroutine.RecoverPanic(ctx, "lifecycle.executeStyleDependencyRebuild")

	seen := make(map[string]bool, len(importers))
	var allPaths []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		allPaths = append(allPaths, p)
	}
	for _, importer := range importers {
		add(importer)
		for _, dep := range ls.interpretedOrchestrator.GetAffectedComponents(importer) {
			add(dep)
		}
	}

	ls.runTargetedRebuild(ctx, cssRelPath, allPaths, false)
}

// runTargetedRebuild filters the entry points to allPaths, runs a synchronous coordinator
// build for that subset, marks the rebuilt components dirty, and proactively recompiles.
// It falls back to a full rebuild when no entry points match or the build fails.
//
// Takes changedLabel (string) which labels the rebuild for logging and build causation.
// Takes allPaths ([]string) which are the project-relative paths to rebuild.
// Takes reloadRoutes (bool) which requests a router reload after recompilation so a newly
// created page becomes routable without a restart.
func (ls *lifecycleService) runTargetedRebuild(ctx context.Context, changedLabel string, allPaths []string, reloadRoutes bool) {
	ctx, l := logger_domain.From(ctx, log)

	targetedEntryPoints := ls.filterEntryPointsByPaths(allPaths)

	if len(targetedEntryPoints) == 0 {
		l.Warn("No entry points found for targeted rebuild, falling back to full rebuild",
			logger_domain.String(fieldPath, changedLabel))
		ls.RequestRebuild(ctx, fmt.Sprintf("fallback-file-change:%s", changedLabel))
		return
	}

	l.Internal("Starting targeted rebuild",
		logger_domain.String("changed", changedLabel),
		logger_domain.Int("affected_count", len(targetedEntryPoints)))

	result, err := ls.coordinatorService.GetOrBuildProject(ctx, targetedEntryPoints,
		coordinator_domain.WithCausationID(fmt.Sprintf("targeted:%s", changedLabel)))
	if err != nil {
		l.Error("Targeted rebuild failed, falling back to full rebuild",
			logger_domain.String(fieldPath, changedLabel),
			logger_domain.Error(err))
		ls.RequestRebuild(ctx, fmt.Sprintf("fallback-file-change:%s", changedLabel))
		return
	}

	if err := ls.interpretedOrchestrator.MarkComponentsDirty(ctx, result); err != nil {
		l.Error("Failed to mark components dirty after targeted rebuild",
			logger_domain.Error(err))
		return
	}

	if err := ls.interpretedOrchestrator.ProactiveRecompile(ctx); err != nil {
		l.Error("Proactive recompile failed after targeted rebuild",
			logger_domain.Error(err))
	}

	if reloadRoutes {
		if runner := ls.currentRunnerSnapshot(); ls.routerManager != nil && runner != nil {
			ls.reloadRoutesIfNeeded(ctx, runner)
		}
	}

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, allPaths)
	}

	l.Internal("Targeted rebuild complete",
		logger_domain.String("changed", changedLabel))
}

// filterEntryPointsByPaths returns the subset of the lifecycle service's entry points
// whose paths match any of the given relative paths. The relative paths are prefixed with
// the module name before comparison.
//
// Takes relPaths ([]string) which contains project-relative paths to match (e.g.
// "pages/login.pk").
//
// Returns []annotator_dto.EntryPoint which contains the matching entry points.
//
// Safe for concurrent use; acquires a read lock on ls.mu.
func (ls *lifecycleService) filterEntryPointsByPaths(relPaths []string) []annotator_dto.EntryPoint {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	pathSet := make(map[string]bool, len(relPaths))
	var modulePrefix string
	if ls.resolver != nil {
		modulePrefix = ls.resolver.GetModuleName() + "/"
	}
	for _, p := range relPaths {
		pathSet[modulePrefix+p] = true
	}

	var filtered []annotator_dto.EntryPoint
	for _, ep := range ls.entryPoints {
		if pathSet[ep.Path] {
			filtered = append(filtered, ep)
		}
	}
	return filtered
}
