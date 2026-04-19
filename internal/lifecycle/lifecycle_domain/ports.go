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

import (
	"context"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

// LifecycleService is the primary driving port for the lifecycle hexagon.
// It handles the build-to-runtime bridge: file watching, build notifications,
// asset pipeline orchestration, and router hot-reload coordination.
type LifecycleService interface {
	// Start begins lifecycle management: file watching and build notification
	// handling. It runs in the background and returns straight away after starting
	// goroutines.
	//
	// Returns error when the lifecycle manager cannot be started.
	Start(ctx context.Context) error

	// Stop shuts down file watchers and cleans up resources.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Stop(ctx context.Context) error

	// RunInitialTasks runs one-time startup tasks such as asset seeding and
	// config loading. Call this after Start to perform the first asset discovery.
	//
	// Returns error when a startup task fails.
	RunInitialTasks(ctx context.Context) error

	// GetEntryPoints returns the current set of discovered entry points.
	// The coordinator uses this to know what to build.
	//
	// Returns []annotator_dto.EntryPoint which contains the entry points found.
	GetEntryPoints() []annotator_dto.EntryPoint

	// RequestRebuild triggers a rebuild through the coordinator.
	//
	// Takes causationID (string) which identifies the request for tracing.
	RequestRebuild(ctx context.Context, causationID string)
}

// FileSystemWatcher provides file system event detection for source files and
// assets. It implements lifecycle_domain.FileSystemWatcher and io.Closer.
type FileSystemWatcher interface {
	// Watch begins watching the specified directories for file changes.
	//
	// Takes recursiveDirs ([]string) which lists directories to watch recursively,
	// including all subdirectories.
	// Takes nonRecursiveDirs ([]string) which lists directories to watch at the
	// top level only.
	//
	// Returns <-chan lifecycle_dto.FileEvent which yields file change events as
	// they occur.
	// Returns error when the watcher cannot be started.
	Watch(ctx context.Context, recursiveDirs []string, nonRecursiveDirs []string) (<-chan lifecycle_dto.FileEvent, error)

	// UpdateWatchedFiles adds or removes files from the watch list.
	// Used to track asset files found during builds.
	//
	// Takes files ([]string) which lists the file paths to watch.
	//
	// Returns error when the watch list cannot be updated.
	UpdateWatchedFiles(ctx context.Context, files []string) error

	// Close stops all file watching and releases resources.
	//
	// Returns error when closing fails.
	Close() error
}

// RouterReloadNotifier is a driven port that the lifecycle uses to tell the
// daemon to reload routes after a build finishes. This is typically provided
// by the daemon's RouterManager.
type RouterReloadNotifier interface {
	// ReloadRoutes refreshes the routing configuration from the manifest store.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes store (ManifestStoreView) which provides access to route manifests.
	//
	// Returns error when the routes cannot be reloaded.
	ReloadRoutes(ctx context.Context, store templater_domain.ManifestStoreView) error
}

// InterpretedBuildOrchestrator handles JIT compilation for interpreted mode,
// converting build artefacts into a runnable InterpretedManifestRunner. This is
// only used in dev-i mode and should be nil in compiled or production modes.
type InterpretedBuildOrchestrator interface {
	// BuildRunner creates a new manifest runner from scratch.
	// This is called on the first build or when a full rebuild is needed.
	//
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// parsed project data.
	//
	// Returns templater_domain.ManifestRunnerPort which is the new runner.
	// Returns error when the build fails.
	BuildRunner(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error)

	// MarkDirty marks changed components for recompilation on their next access.
	//
	// This replaces the cached manifest with the one from the result. Use
	// MarkComponentsDirty for targeted rebuilds where only a subset of
	// components is present in the result.
	//
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// changed components to mark.
	//
	// Returns error when marking fails.
	MarkDirty(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error

	// MarkComponentsDirty marks changed components for recompilation, merging
	// the partial result into the existing manifest rather than replacing it.
	// This is used by targeted rebuilds where the result only contains the
	// changed component and its dependents.
	//
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// changed components to mark.
	//
	// Returns error when marking fails.
	MarkComponentsDirty(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error

	// IsInitialised reports whether the initial runner has been created. This is
	// used to tell the first build apart from later updates.
	//
	// Returns bool which is true if the runner has been created, false otherwise.
	IsInitialised() bool

	// GetAffectedComponents returns all component paths that transitively
	// depend on the given component, found by BFS through the reverse
	// dependency map. Used to determine which components need
	// rebuilding when a partial changes.
	//
	// Takes relPath (string) which is the relative path of the changed
	// component.
	//
	// Returns []string which contains the relative paths of all affected
	// components.
	GetAffectedComponents(relPath string) []string

	// ProactiveRecompile JIT-compiles all components currently in the dirty
	// code cache. This runs compilation eagerly rather than waiting for an
	// HTTP request to trigger it.
	//
	// Returns error when compilation fails for any component.
	ProactiveRecompile(ctx context.Context) error
}

// TemplaterRunnerSwapper allows lifecycle to update the templater's manifest
// runner. Used when a new runner is created after a build in interpreted mode.
type TemplaterRunnerSwapper interface {
	// SetRunner assigns the manifest runner used to execute template operations.
	//
	// Takes runner (ManifestRunnerPort) which handles manifest execution.
	SetRunner(runner templater_domain.ManifestRunnerPort)
}

// BuildCacheInvalidator triggers a rebuild in interpreted mode.
// The manifest runner implements BuildCacheInvalidator when it supports cache
// invalidation.
type BuildCacheInvalidator interface {
	// InvalidateBuildCache clears any stored build outputs held by the runner.
	InvalidateBuildCache()
}

// AssetPipelinePort processes build results to create transformation profiles.
// It turns asset requirements from the build manifest into instructions for
// the registry.
type AssetPipelinePort interface {
	// ProcessBuildResult processes a project annotation result from the build.
	//
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// build annotations to process.
	//
	// Returns error when processing fails.
	ProcessBuildResult(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error
}

// RenderRegistryCachePort provides cache clearing for render registries.
// It is used by lifecycle to clear caches when assets such as SVG files change.
type RenderRegistryCachePort interface {
	// ClearSvgCache removes the cached SVG data for the given identifier.
	//
	// Takes svgID (string) which identifies the SVG to remove from the cache.
	ClearSvgCache(ctx context.Context, svgID string)
}

// BuilderAdapter defines an interface for running a full build process.
type BuilderAdapter interface {
	// RunBuild runs the build process.
	//
	// Returns *BuildResult which holds task counts and any failure details.
	// Returns error when an infrastructure failure prevents the build from
	// completing (context cancelled, flush timeout).
	RunBuild(ctx context.Context) (*BuildResult, error)
}

// BridgeWithCounter provides access to the count of processed artefact events.
// It is used to detect when a pipeline flush has finished.
type BridgeWithCounter interface {
	// ArtefactEventsProcessed returns the count of artefact events processed.
	//
	// Returns int64 which is the total number of artefact events handled.
	ArtefactEventsProcessed() int64
}

// DevEventNotifier sends browser-visible notifications when dev builds
// complete. It is only wired in dev/dev-i modes; nil in production.
type DevEventNotifier interface {
	// NotifyRebuildComplete broadcasts a build-completion event to connected
	// browsers.
	//
	// The affected paths are project-relative component paths
	// (e.g. "pages/login.pk") which the implementation converts to URL
	// route patterns.
	//
	// Takes ctx (context.Context) which carries logging context.
	// Takes affectedPaths ([]string) which lists the components that were
	// rebuilt.
	NotifyRebuildComplete(ctx context.Context, affectedPaths []string)
}
