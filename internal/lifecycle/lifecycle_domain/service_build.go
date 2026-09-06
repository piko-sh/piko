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

// This file contains build notification handling, interpreted-mode runner management, and
// route reloading for the lifecycle service.

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/goroutine"
)

// handleBuildNotifications listens for build events from the coordinator.
//
// Takes notifications (<-chan coordinator_domain.BuildNotification) which provides the
// stream of build events to process.
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
// Takes notification (coordinator_domain.BuildNotification) which holds the build result
// to process.
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

// updateWatchedFilesFromBuild updates the file watcher with asset paths from the build
// result. This means new assets are watched for hot-reload.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the asset manifest
// from which to extract file paths.
func (ls *lifecycleService) updateWatchedFilesFromBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	ls.updateStyleDepsFromBuild(result)

	if ls.watcherAdapter == nil {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	var watched []string
	if result.FinalAssetManifest != nil {
		watched = ls.extractAssetPathsFromManifest(result)
	}

	watched = append(watched, ls.allWatchedStyleFiles()...)
	if len(watched) == 0 {
		return
	}

	if err := ls.watcherAdapter.UpdateWatchedFiles(ctx, watched); err != nil {
		l.Error("Failed to update watched files after build", logger_domain.Error(err))
	}
}

// updateStyleDepsFromBuild merges the CSS @import dependencies reported by each annotated
// component in result into componentStyleDeps. Each build updates only the components it
// annotated (so targeted rebuilds do not erase other components' entries), and components
// that no longer import any stylesheet are dropped.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the per-component
// annotation results and the virtual module for path resolution.
//
// Concurrency: acquires styleDepsMu while merging the build's style dependencies.
func (ls *lifecycleService) updateStyleDepsFromBuild(result *annotator_dto.ProjectAnnotationResult) {
	if result == nil || result.VirtualModule == nil {
		return
	}

	ls.styleDepsMu.Lock()
	defer ls.styleDepsMu.Unlock()
	if ls.componentStyleDeps == nil {
		ls.componentStyleDeps = make(map[string][]string)
	}

	for hashedName, ar := range result.ComponentResults {
		if ar == nil {
			continue
		}
		relPath := ls.componentRelPath(hashedName, result.VirtualModule)
		if relPath == "" {
			continue
		}
		styleRelPaths := make([]string, 0, len(ar.ImportedStylePaths))
		for _, p := range ar.ImportedStylePaths {
			if rel := ls.projectRelPath(p); rel != "" {
				styleRelPaths = append(styleRelPaths, rel)
			}
		}
		if len(styleRelPaths) == 0 {
			delete(ls.componentStyleDeps, relPath)
			continue
		}
		ls.componentStyleDeps[relPath] = styleRelPaths
	}
}

// removeComponentStyleDeps drops the CSS @import dependency entry for a component that is
// no longer present, so its stylesheet watches do not linger after the component is
// removed or renamed away.
//
// Takes relPath (string) which is the component's project-relative source path (e.g.
// "pages/old.pk").
//
// Concurrency: acquires styleDepsMu while deleting the entry.
func (ls *lifecycleService) removeComponentStyleDeps(relPath string) {
	ls.styleDepsMu.Lock()
	defer ls.styleDepsMu.Unlock()
	delete(ls.componentStyleDeps, relPath)
}

// projectRelPath converts an absolute path to a slash-separated path relative to the
// project root, matching the form the file watcher reports (and the form asset paths
// use). The project root is the configured base directory resolved to an absolute path.
//
// Takes absPath (string) which is the absolute path to relativise.
//
// Returns string which is the slash-separated project-relative path, or "" if it lies
// outside the project root or cannot be relativised.
func (ls *lifecycleService) projectRelPath(absPath string) string {
	base := ls.pathsConfig.BaseDir
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}
	rel, err := ls.fs.Rel(base, absPath)
	if err != nil || !filepath.IsLocal(rel) {
		return ""
	}
	return filepath.ToSlash(rel)
}

// componentRelPath resolves a component's project-relative source path (e.g.
// "pages/index.pk") from its hashed name using the virtual module.
//
// Takes hashedName (string) which identifies the component.
// Takes vm (*annotator_dto.VirtualModule) which maps hashed names to components.
//
// Returns string which is the slash-separated project-relative path, or "" if unresolved.
func (ls *lifecycleService) componentRelPath(hashedName string, vm *annotator_dto.VirtualModule) string {
	vc, ok := vm.ComponentsByHash[hashedName]
	if !ok || vc == nil || vc.Source == nil {
		return ""
	}
	return ls.projectRelPath(vc.Source.SourcePath)
}

// allWatchedStyleFiles returns the de-duplicated union of every component's imported
// stylesheet paths, for registration with the file watcher.
//
// The paths are joined with the base directory so they take the same form as the asset
// paths from extractAssetPathsFromManifest (project-relative when BaseDir is ".",
// absolute otherwise), keeping one consistent representation in the watch set.
//
// Returns []string which contains the stylesheet paths to watch (nil if none).
//
// Concurrency: read-locks styleDepsMu while building the union.
func (ls *lifecycleService) allWatchedStyleFiles() []string {
	ls.styleDepsMu.RLock()
	defer ls.styleDepsMu.RUnlock()
	if len(ls.componentStyleDeps) == 0 {
		return nil
	}
	baseDir := ls.pathsConfig.BaseDir
	seen := make(map[string]struct{})
	var out []string
	for _, paths := range ls.componentStyleDeps {
		for _, rel := range paths {
			full := ls.fs.Join(baseDir, rel)
			if _, ok := seen[full]; ok {
				continue
			}
			seen[full] = struct{}{}
			out = append(out, full)
		}
	}
	return out
}

// normaliseStylePath converts a watcher-reported path into the canonical slash-separated
// project-relative form used as the componentStyleDeps key and value.
//
// The watcher may report an absolute path (with an absolute base directory) or a
// project-relative one (the default, where BaseDir is "."); both reduce to the same
// project-relative form so lookups match regardless of configuration.
//
// Takes p (string) which is the path to normalise.
//
// Returns string which is the slash-separated project-relative path, or "" if an absolute
// path lies outside the project root.
func (ls *lifecycleService) normaliseStylePath(p string) string {
	if filepath.IsAbs(p) {
		return ls.projectRelPath(p)
	}
	return filepath.ToSlash(filepath.Clean(p))
}

// importersOfStyle returns the project-relative paths of components that import the
// stylesheet at cssPath via CSS @import.
//
// Takes cssPath (string) which is the path of the changed stylesheet as reported by the
// file watcher; it may be absolute or project-relative and is normalised before lookup.
//
// Returns []string which contains the importing components' relative paths (nil if none).
//
// Concurrency: read-locks styleDepsMu while scanning for importers.
func (ls *lifecycleService) importersOfStyle(cssPath string) []string {
	target := ls.normaliseStylePath(cssPath)
	if target == "" {
		return nil
	}

	ls.styleDepsMu.RLock()
	defer ls.styleDepsMu.RUnlock()

	var importers []string
	for relPath, paths := range ls.componentStyleDeps {
		if slices.Contains(paths, target) {
			importers = append(importers, relPath)
		}
	}
	return importers
}

// extractAssetPathsFromManifest converts asset manifest entries to absolute file paths.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the asset manifest
// entries to convert.
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
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the asset manifest
// to process.
func (ls *lifecycleService) processAssetManifest(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	if ls.assetPipeline == nil || len(result.FinalAssetManifest) == 0 {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	if err := ls.assetPipeline.ProcessBuildResult(ctx, result); err != nil {
		l.Error("Failed to process asset manifest", logger_domain.Error(err))
	}
}

// handleInterpretedBuild processes build results when running in interpreted mode.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the annotation
// results to process.
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
// Takes result (*annotator_dto.ProjectAnnotationResult) which holds the annotation data
// used to build the runner.
func (ls *lifecycleService) handleInitialBuild(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) {
	ctx, l := logger_domain.From(ctx, log)

	l.Internal("Initial build: creating interpreted runner...")

	newRunner, err := ls.interpretedOrchestrator.BuildRunner(ctx, result)
	if err != nil {
		l.Error("Failed to build initial interpreted runner", logger_domain.Error(err))
		return
	}

	ls.templaterService.SetRunner(newRunner)
	ls.setCurrentRunner(newRunner)
	l.Internal("Initial interpreted runner successfully created")
	ls.reloadRoutesIfNeeded(ctx, newRunner)

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, nil)
	}
}

// handleIncrementalBuild marks components dirty and proactively JIT-compiles them rather
// than waiting for an HTTP request. It then reloads the router so pages created after
// server start become routable without a restart.
//
// The interpreted runner created at the initial build is orchestrator-backed: a newly
// created page is compiled into the orchestrator's program cache by ProactiveRecompile,
// so reloading routes against the existing runner picks up the new page's key. No new
// runner is needed.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the components to
// mark as dirty.
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

	if runner := ls.currentRunnerSnapshot(); ls.routerManager != nil && runner != nil {
		ls.reloadRoutesIfNeeded(ctx, runner)
	}

	if ls.devEventNotifier != nil {
		ls.devEventNotifier.NotifyRebuildComplete(ctx, nil)
	}
}

// reloadRoutesIfNeeded reloads routes if a router manager is set.
//
// Takes ctx (context.Context) which carries the logger.
// Takes newRunner (templater_domain.ManifestRunnerPort) which provides the manifest data
// for route creation.
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

// SetCurrentRunner records the interpreted runner from outside the build-notification
// flow.
//
// In dev-i the initial runner is built by the bootstrap builder, not by
// handleInitialBuild, so the builder calls this after the initial build to seed the
// runner snapshot that file-event route reloads depend on.
//
// Takes runner (templater_domain.ManifestRunnerPort) which is the runner to store.
func (ls *lifecycleService) SetCurrentRunner(runner templater_domain.ManifestRunnerPort) {
	ls.setCurrentRunner(runner)
}

// setCurrentRunner stores the interpreted runner so the watch goroutine can read it
// without racing the build-notification goroutine.
//
// Takes runner (templater_domain.ManifestRunnerPort) which is the runner to store.
//
// Concurrency: write-locks mu.
func (ls *lifecycleService) setCurrentRunner(runner templater_domain.ManifestRunnerPort) {
	ls.mu.Lock()
	ls.currentRunner = runner
	ls.mu.Unlock()
}

// currentRunnerSnapshot returns the current interpreted runner.
//
// Callers reload routes with the returned value outside the lock, so mu is never held
// across the router I/O.
//
// Returns templater_domain.ManifestRunnerPort which is the current runner, or nil when no
// initial build has completed.
//
// Concurrency: read-locks mu.
func (ls *lifecycleService) currentRunnerSnapshot() templater_domain.ManifestRunnerPort {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.currentRunner
}

// interpretedRunnerView defines the interface for an interpreted runner that provides
// page entry information for route registration.
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

	// FindErrorPage looks up the most specific custom error page for the given status code
	// and request path, so reloaded routers continue to resolve user-provided error pages
	// (e.g. !404.pk) after a hot reload.
	//
	// Takes statusCode (int) which is the HTTP status code to match.
	// Takes requestPath (string) which is the URL path being requested.
	//
	// Returns templater_domain.PageEntryView which is the matching error page entry.
	// Returns bool which indicates whether a matching error page was found.
	FindErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool)

	// ListPreviewEntries returns all entries that expose a Preview function.
	//
	// Returns []templater_domain.PreviewCatalogueEntry which contains the preview entries.
	ListPreviewEntries() []templater_domain.PreviewCatalogueEntry
}

// interpretedManifestStoreViewAdapter implements ManifestStoreView by wrapping an
// interpretedRunnerView for router registration.
type interpretedManifestStoreViewAdapter struct {
	// runner provides access to page keys and entries from the interpreted manifest.
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

// FindErrorPage resolves the most specific custom error page by delegating to the runner.
//
// This keeps user-provided error pages (e.g. !404.pk) working after a hot reload rebuilds
// the router.
//
// Takes statusCode (int) which is the HTTP status code to match.
// Takes requestPath (string) which is the request path being served.
//
// Returns templater_domain.PageEntryView which is the matching error page entry.
// Returns bool which indicates whether a matching error page was found.
func (a *interpretedManifestStoreViewAdapter) FindErrorPage(statusCode int, requestPath string) (templater_domain.PageEntryView, bool) {
	return a.runner.FindErrorPage(statusCode, requestPath)
}

// ListPreviewEntries returns all preview entries from the wrapped interpreted runner.
//
// Returns []templater_domain.PreviewCatalogueEntry which contains the preview entries.
func (a *interpretedManifestStoreViewAdapter) ListPreviewEntries() []templater_domain.PreviewCatalogueEntry {
	return a.runner.ListPreviewEntries()
}

// newInterpretedManifestStoreView creates a store view adapter for an interpreted runner.
//
// Takes runner (templater_domain.ManifestRunnerPort) which provides the runner to wrap.
//
// Returns templater_domain.ManifestStoreView which wraps the runner for store access.
//
// Panics if the runner does not implement interpretedRunnerView.
func newInterpretedManifestStoreView(runner templater_domain.ManifestRunnerPort) templater_domain.ManifestStoreView {
	if interpretedRunner, ok := runner.(interpretedRunnerView); ok {
		return &interpretedManifestStoreViewAdapter{runner: interpretedRunner}
	}
	panic("newInterpretedManifestStoreView called with non-interpreted runner")
}
