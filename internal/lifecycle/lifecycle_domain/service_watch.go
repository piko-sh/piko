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

// This file contains file-watching, event handling, entry-point bookkeeping,
// and asset change processing for the lifecycle service.

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
)

// watchLoop processes file change events from the watcher channel until
// the context is cancelled or the stop signal is received.
//
// Takes events (<-chan lifecycle_dto.FileEvent) which provides file change
// events from the file watcher.
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
// Takes initialSeed (bool) which indicates whether this is part of the initial
// scan.
func (ls *lifecycleService) handleFileEvent(ctx context.Context, event lifecycle_dto.FileEvent, initialSeed bool) {
	ctx, span, _ := log.Span(ctx, "handleFileEvent",
		logger_domain.String(fieldPath, event.Path),
		logger_domain.Bool("initialSeed", initialSeed),
	)
	defer span.End()

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
// In interpreted mode with an initialised orchestrator, this triggers a
// targeted rebuild that only re-annotates and regenerates the changed
// component and its transitive dependents. Otherwise it falls back to a full
// coordinator rebuild.
//
// Takes fec (fileEventContext) which provides the file event details and
// logging context.
// Takes initialSeed (bool) which indicates whether this is the initial file
// discovery phase.
//
// Safe for concurrent use. Targeted rebuilds are spawned in a separate
// goroutine.
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

	if ls.interpretedOrchestrator != nil && ls.interpretedOrchestrator.IsInitialised() && ls.coordinatorService != nil {
		go ls.executeTargetedRebuild(ctx, fec.relPath)
		return
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

// handleAssetChange processes changes to asset files.
//
// Takes fec (fileEventContext) which provides the file event details and
// logging context.
func (ls *lifecycleService) handleAssetChange(fec fileEventContext) {
	_, l := logger_domain.From(fec.ctx, log)

	l.Trace("Asset file changed, processing via orchestrator pipeline.")
	ls.clearSvgCacheIfNeeded(fec)

	switch fec.event.Type {
	case lifecycle_dto.FileEventTypeCreate, lifecycle_dto.FileEventTypeWrite:
		ls.upsertAssetArtefact(fec)
	case lifecycle_dto.FileEventTypeRemove, lifecycle_dto.FileEventTypeRename:
		ls.deleteAssetArtefact(fec)
	default:
		l.Warn("Unknown file event type for asset change",
			logger_domain.Int("event_type", int(fec.event.Type)),
			logger_domain.String("path", fec.event.Path))
	}
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

// clearComponentCacheIfNeeded clears the component cache for PKC files. The
// component tag name is taken from the file name without the extension.
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
// Takes fec (fileEventContext) which provides the file event details and
// context for the operation.
func (ls *lifecycleService) upsertAssetArtefact(fec fileEventContext) {
	ctx, l := logger_domain.From(fec.ctx, log)

	file, err := ls.fs.Open(fec.event.Path)
	if err != nil {
		l.Error("Failed to read updated asset file", logger_domain.Error(err))
		return
	}
	defer func() { _ = file.Close() }()

	profiles := GetProfilesForFile(fec.artefactID, nil)
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

// componentType represents the kind of component file, such as page, partial,
// or email template.
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
// Takes event (lifecycle_dto.FileEvent) which specifies the file system event
// that happened.
// Takes relPath (string) which is the path to the affected file, relative to
// the project root.
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
// Takes relPath (string) which is the relative path to check against configured
// source directories.
//
// Returns componentType which indicates whether the path matches a page,
// partial, or email source directory.
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
// Takes entryPointPath (string) which specifies the path of the entry point
// to remove.
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

// executeTargetedRebuild performs an incremental rebuild for a single changed
// file by only re-annotating and regenerating the changed component and its
// transitive dependents.
//
// It queries the orchestrator's reverse dependency map to find affected
// components, filters the entry point list to only those components, runs a
// synchronous coordinator build with the targeted subset, then merges the
// result and proactively JIT-compiles all dirty components.
//
// Falls back to a full coordinator rebuild if the targeted build fails.
//
// Takes relPath (string) which is the project-relative path of the changed
// file (e.g. "pages/login.pk").
//
// Designed to run in a goroutine so the watch loop is not blocked.
func (ls *lifecycleService) executeTargetedRebuild(ctx context.Context, relPath string) {
	defer goroutine.RecoverPanic(ctx, "lifecycle.executeTargetedRebuild")
	ctx, l := logger_domain.From(ctx, log)

	affected := ls.interpretedOrchestrator.GetAffectedComponents(relPath)

	allPaths := make([]string, 0, 1+len(affected))
	allPaths = append(allPaths, relPath)
	allPaths = append(allPaths, affected...)

	targetedEntryPoints := ls.filterEntryPointsByPaths(allPaths)

	if len(targetedEntryPoints) == 0 {
		l.Warn("No entry points found for targeted rebuild, falling back to full rebuild",
			logger_domain.String(fieldPath, relPath))
		ls.RequestRebuild(ctx, fmt.Sprintf("fallback-file-change:%s", relPath))
		return
	}

	l.Internal("Starting targeted rebuild",
		logger_domain.String("changed", relPath),
		logger_domain.Int("affected_count", len(targetedEntryPoints)))

	result, err := ls.coordinatorService.GetOrBuildProject(ctx, targetedEntryPoints,
		coordinator_domain.WithCausationID(fmt.Sprintf("targeted:%s", relPath)))
	if err != nil {
		l.Error("Targeted rebuild failed, falling back to full rebuild",
			logger_domain.String(fieldPath, relPath),
			logger_domain.Error(err))
		ls.RequestRebuild(ctx, fmt.Sprintf("fallback-file-change:%s", relPath))
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

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, allPaths)
	}

	l.Internal("Targeted rebuild complete",
		logger_domain.String("changed", relPath))
}

// filterEntryPointsByPaths returns the subset of the lifecycle service's entry
// points whose paths match any of the given relative paths. The relative paths
// are prefixed with the module name before comparison.
//
// Takes relPaths ([]string) which contains project-relative paths to match
// (e.g. "pages/login.pk").
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
