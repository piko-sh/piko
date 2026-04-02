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

// This file contains build notification handling, interpreted-mode runner
// management, and route reloading for the lifecycle service.

import (
	"context"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
)

// handleBuildNotifications listens for build events from the coordinator.
//
// Takes notifications (<-chan coordinator_domain.BuildNotification) which
// provides the stream of build events to process.
func (ls *lifecycleService) handleBuildNotifications(ctx context.Context, notifications <-chan coordinator_domain.BuildNotification) {
	defer goroutine.RecoverPanic(ctx, "lifecycle.handleBuildNotifications")
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting to listen for build notifications...")

	for {
		select {
		case <-ctx.Done():
			l.Trace("Stopping build notification listener.")
			return
		case <-ls.stopChan:
			l.Trace("Stop signal received, exiting notification handler.")
			return
		case notification, ok := <-notifications:
			if !ok {
				l.Warn("Build notification channel was closed.")
				return
			}
			ls.processBuildNotification(ctx, notification)
		}
	}
}

// processBuildNotification handles a single build notification.
//
// Takes notification (coordinator_domain.BuildNotification) which holds the
// build result to process.
func (ls *lifecycleService) processBuildNotification(ctx context.Context, notification coordinator_domain.BuildNotification) {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("Received new build result", logger_domain.String("causationID", notification.CausationID))

	if notification.Result == nil {
		l.Warn("Build notification contained no result")
		return
	}

	ls.processAssetManifest(ctx, notification.Result)

	if !strings.HasPrefix(notification.CausationID, "targeted:") {
		ls.handleInterpretedBuild(ctx, notification.Result)
	}

	ls.updateWatchedFilesFromBuild(ctx, notification.Result)
}

// updateWatchedFilesFromBuild updates the file watcher with asset paths from
// the build result. This means new assets are watched for hot-reload.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// asset manifest from which to extract file paths.
func (ls *lifecycleService) updateWatchedFilesFromBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	if ls.watcherAdapter == nil || result.FinalAssetManifest == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	assetFiles := ls.extractAssetPathsFromManifest(result)
	if err := ls.watcherAdapter.UpdateWatchedFiles(ctx, assetFiles); err != nil {
		l.Error("Failed to update watched files after build", logger_domain.Error(err))
	}
}

// extractAssetPathsFromManifest converts asset manifest entries to absolute
// file paths.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// asset manifest entries to convert.
//
// Returns []string which contains the absolute file paths for each asset.
func (ls *lifecycleService) extractAssetPathsFromManifest(result *annotator_dto.ProjectAnnotationResult) []string {
	baseDir := ls.pathsConfig.BaseDir

	var modulePrefix string
	if ls.resolver != nil {
		modulePrefix = ls.resolver.GetModuleName() + "/"
	}

	assetFiles := make([]string, 0, len(result.FinalAssetManifest))
	for _, asset := range result.FinalAssetManifest {
		relPath := asset.SourcePath
		if modulePrefix != "" {
			relPath = strings.TrimPrefix(asset.SourcePath, modulePrefix)
		}
		assetFiles = append(assetFiles, ls.fs.Join(baseDir, relPath))
	}

	return assetFiles
}

// processAssetManifest processes the asset manifest from a build result.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// asset manifest to process.
func (ls *lifecycleService) processAssetManifest(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	if ls.assetPipeline == nil || len(result.FinalAssetManifest) == 0 {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	if err := ls.assetPipeline.ProcessBuildResult(ctx, result); err != nil {
		l.Error("Failed to process asset manifest", logger_domain.Error(err))
	}
}

// handleInterpretedBuild processes build results when running in interpreted
// mode.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// annotation results to process.
func (ls *lifecycleService) handleInterpretedBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	if ls.interpretedOrchestrator == nil || ls.templaterService == nil {
		return
	}

	if !ls.interpretedOrchestrator.IsInitialised() {
		ls.handleInitialBuild(ctx, result)
		return
	}

	ls.handleIncrementalBuild(ctx, result)
}

// handleInitialBuild creates the initial interpreted runner.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the
// annotation data used to build the runner.
func (ls *lifecycleService) handleInitialBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("Initial build: creating interpreted runner...")

	newRunner, err := ls.interpretedOrchestrator.BuildRunner(ctx, result)
	if err != nil {
		l.Error("Failed to build initial interpreted runner", logger_domain.Error(err))
		return
	}

	ls.templaterService.SetRunner(newRunner)
	l.Internal("Initial interpreted runner successfully created")
	ls.reloadRoutesIfNeeded(ctx, newRunner)

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, nil)
	}
}

// handleIncrementalBuild marks components dirty and proactively JIT-compiles
// them rather than waiting for an HTTP request.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
// components to mark as dirty.
func (ls *lifecycleService) handleIncrementalBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("Incremental build: marking components dirty...")

	if err := ls.interpretedOrchestrator.MarkDirty(ctx, result); err != nil {
		l.Error("Failed to mark components dirty", logger_domain.Error(err))
		return
	}

	l.Internal("Components marked dirty, starting proactive compilation...")

	if err := ls.interpretedOrchestrator.ProactiveRecompile(ctx); err != nil {
		l.Error("Proactive recompile failed", logger_domain.Error(err))
	}

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, nil)
	}
}

// reloadRoutesIfNeeded reloads routes if a router manager is set.
//
// Takes ctx (context.Context) which carries the logger.
// Takes newRunner (templater_domain.ManifestRunnerPort) which provides the
// manifest data for route creation.
func (ls *lifecycleService) reloadRoutesIfNeeded(ctx context.Context, newRunner templater_domain.ManifestRunnerPort) {
	if ls.routerManager == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	manifestStore := newInterpretedManifestStoreView(newRunner)
	if err := ls.routerManager.ReloadRoutes(ctx, manifestStore); err != nil {
		l.Error("Failed to load routes after initial build", logger_domain.Error(err))
		return
	}

	l.Internal("Routes successfully loaded")
}

// interpretedRunnerView defines the interface for an interpreted runner that
// provides page entry information for route registration.
type interpretedRunnerView interface {
	// GetKeys returns all keys stored in the collection.
	//
	// Returns []string which contains the keys in no particular order.
	GetKeys() []string

	// GetPageEntryByPath retrieves a page entry by its path.
	//
	// Takes path (string) which is the path to look up.
	//
	// Returns templater_domain.PageEntryView which is the page entry if found.
	// Returns bool which indicates whether the entry was found.
	GetPageEntryByPath(path string) (templater_domain.PageEntryView, bool)
}

// interpretedManifestStoreViewAdapter implements ManifestStoreView by
// wrapping an interpretedRunnerView for router registration.
type interpretedManifestStoreViewAdapter struct {
	// runner provides access to page keys and entries from the interpreted
	// manifest.
	runner interpretedRunnerView
}

// GetKeys returns all page keys from the interpreted runner.
//
// Returns []string which contains all available page keys.
func (a *interpretedManifestStoreViewAdapter) GetKeys() []string {
	return a.runner.GetKeys()
}

// GetPageEntry retrieves a page entry by its path from the interpreted runner.
//
// Takes path (string) which specifies the path to look up.
//
// Returns templater_domain.PageEntryView which contains the page entry data.
// Returns bool which indicates whether the entry was found.
func (a *interpretedManifestStoreViewAdapter) GetPageEntry(path string) (templater_domain.PageEntryView, bool) {
	return a.runner.GetPageEntryByPath(path)
}

// FindErrorPage is not supported in interpreted mode, where error
// pages require the compiled manifest store, and always returns
// (nil, false).
//
// Takes statusCode (int) which is the HTTP status code (unused).
// Takes requestPath (string) which is the request path (unused).
//
// Returns (nil, false) always.
func (*interpretedManifestStoreViewAdapter) FindErrorPage(_ int, _ string) (templater_domain.PageEntryView, bool) {
	return nil, false
}

// GetCollectionFallbackRoutes is not supported in interpreted mode. Static
// collection expansion only happens during a compiled build.
//
// Returns nil always.
func (*interpretedManifestStoreViewAdapter) GetCollectionFallbackRoutes() []templater_domain.CollectionFallbackRouteView {
	return nil
}

// ListPreviewEntries is not supported in this adapter. The interpreted mode
// preview support is provided by the InterpretedManifestStoreView instead.
//
// Returns nil always.
func (*interpretedManifestStoreViewAdapter) ListPreviewEntries() []templater_domain.PreviewCatalogueEntry {
	return nil
}

// newInterpretedManifestStoreView creates a store view adapter for an
// interpreted runner.
//
// Takes runner (templater_domain.ManifestRunnerPort) which provides the runner
// to wrap.
//
// Returns templater_domain.ManifestStoreView which wraps the runner for store
// access.
//
// Panics if the runner does not implement interpretedRunnerView.
func newInterpretedManifestStoreView(runner templater_domain.ManifestRunnerPort) templater_domain.ManifestStoreView {
	if interpretedRunner, ok := runner.(interpretedRunnerView); ok {
		return &interpretedManifestStoreViewAdapter{runner: interpretedRunner}
	}
	panic("newInterpretedManifestStoreView called with non-interpreted runner")
}
