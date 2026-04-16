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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/component/component_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// themeArtefactPath is the artefact path for the generated theme CSS file.
	themeArtefactPath = "theme.css"

	// fileEventChannelBuffer is the buffer size for the file event channel during
	// asset seeding.
	fileEventChannelBuffer = 1000

	// gcDelayAfterStartup is the delay before forcing garbage collection after
	// initial tasks.
	gcDelayAfterStartup = 30 * time.Second

	// concurrencyMultiplier is the multiplier for NumCPU to determine worker pool
	// size.
	concurrencyMultiplier = 4

	// initialTaskErrChanBuffer is the buffer size for the error channel during
	// initial task execution.
	initialTaskErrChanBuffer = 3

	// fieldPath is the logging field name for file paths.
	fieldPath = "path"
)

// lifecycleService is the core domain service for build-to-runtime lifecycle
// management. It implements LifecycleService, handling file watching, build
// notifications, asset pipeline orchestration, and router hot-reload.
type lifecycleService struct {
	// gcTimer holds the post-startup GC timer so it can be stopped during
	// shutdown.
	gcTimer clock.Timer

	// componentRegistry holds the PKC component registry for deterministic tag
	// lookup.
	componentRegistry component_domain.ComponentRegistry

	// templaterService swaps the template runner for interpreted mode.
	templaterService TemplaterRunnerSwapper

	// coordinatorService handles build coordination; nil disables rebuild
	// requests.
	coordinatorService coordinator_domain.CoordinatorService

	// resolver provides module path lookup; nil when path lookup is not available.
	resolver resolver_domain.ResolverPort

	// renderRegistryPort provides access to the render registry for clearing
	// SVG and component caches; nil disables cache clearing.
	renderRegistryPort render_domain.RegistryPort

	// renderer builds CSS and other rendered assets; nil if rendering is disabled.
	renderer render_domain.RenderService

	// fs provides file system operations for joining paths.
	fs FileSystem

	// clock provides time functions; replaced during testing.
	clock clock.Clock

	// routerManager notifies the router to reload routes after manifest changes.
	routerManager RouterReloadNotifier

	// captchaService provides access to captcha providers for seeding init
	// scripts. Nil when captcha is not configured.
	captchaService captcha_domain.CaptchaServicePort

	// interpretedOrchestrator manages builds and runner lifecycle in interpreted
	// mode.
	interpretedOrchestrator InterpretedBuildOrchestrator

	// registryService handles storage and retrieval of artefacts.
	registryService registry_domain.RegistryService

	// watcherAdapter monitors file system changes; nil disables watching.
	watcherAdapter FileSystemWatcher

	// assetPipeline processes asset manifests from build results.
	assetPipeline AssetPipelinePort

	// buildCacheInvalidator clears the JIT build cache when core source files
	// change; nil means cache clearing is disabled.
	buildCacheInvalidator BuildCacheInvalidator

	// devEventNotifier broadcasts build-complete events to connected browsers
	// via SSE. Nil in production mode.
	devEventNotifier DevEventNotifier

	// unsubscribe cancels the build notification subscription; nil if not
	// subscribed.
	unsubscribe func()

	// stopChan signals goroutines to stop; closed by Stop.
	stopChan chan struct{}

	// pathsConfig holds the resolved source directory paths for this lifecycle
	// instance. All fields are value types; pointer-to-value conversion is
	// performed in the bootstrap layer.
	pathsConfig LifecyclePathsConfig

	// entryPoints holds the discovered package entry points; protected by mu.
	entryPoints []annotator_dto.EntryPoint

	// externalComponents holds component definitions from WithComponents() that
	// need module resolution at build time.
	externalComponents []component_dto.ComponentDefinition

	// configProvider holds the server and website settings.
	configProvider config.Provider

	// mu guards access to entryPoints for safe concurrent reads and writes.
	mu sync.RWMutex

	// stopOnce guards single execution of Stop.
	stopOnce sync.Once
}

// LifecycleServiceDeps contains all dependencies needed to create a
// LifecycleService.
type LifecycleServiceDeps struct {
	// WatcherAdapter watches the file system for changes.
	WatcherAdapter FileSystemWatcher

	// Renderer provides document rendering services.
	Renderer render_domain.RenderService

	// RegistryService provides access to the service registry.
	RegistryService registry_domain.RegistryService

	// Clock provides time functions; nil uses the real system clock.
	Clock clock.Clock

	// Resolver provides module path and directory resolution for
	// lifecycle operations.
	Resolver resolver_domain.ResolverPort

	// RenderRegistryPort gives access to the render registry.
	RenderRegistryPort render_domain.RegistryPort

	// RouterManager notifies the router to reload after lifecycle changes.
	RouterManager RouterReloadNotifier

	// TemplaterService runs templates and allows them to be swapped at runtime.
	TemplaterService TemplaterRunnerSwapper

	// AssetPipeline handles asset processing and bundling.
	AssetPipeline AssetPipelinePort

	// InterpretedOrchestrator manages the build process for interpreted assets.
	InterpretedOrchestrator InterpretedBuildOrchestrator

	// CoordinatorService manages the ordering and timing of lifecycle operations.
	CoordinatorService coordinator_domain.CoordinatorService

	// BuildCacheInvalidator clears stored build files when content changes.
	BuildCacheInvalidator BuildCacheInvalidator

	// DevEventNotifier broadcasts build-complete events to connected browsers.
	// Nil in production mode.
	DevEventNotifier DevEventNotifier

	// FileSystem provides file system operations; nil uses the OS file system.
	FileSystem FileSystem

	// ComponentRegistry holds the PKC component registry for auto-discovery.
	// If nil, component auto-discovery is disabled.
	ComponentRegistry component_domain.ComponentRegistry

	// CaptchaService provides access to captcha providers for seeding init
	// scripts into the registry. Nil when captcha is not configured.
	CaptchaService captcha_domain.CaptchaServicePort

	// PathsConfig holds the resolved source directory paths. All fields are
	// value types; the bootstrap layer converts pointer config fields before
	// passing this in.
	PathsConfig LifecyclePathsConfig

	// ExternalComponents holds component definitions from WithComponents() that
	// require module resolution. The lifecycle service resolves their ModulePath
	// to disk directories and discovers .pkc files there.
	ExternalComponents []component_dto.ComponentDefinition

	// ConfigProvider supplies configuration settings for the lifecycle service.
	ConfigProvider config.Provider
}

// Start begins the lifecycle management: file watching and build
// notification handling.
//
// Returns error when initial entry point discovery fails or the file watcher
// cannot be started.
//
// Safe for concurrent use. Spawns background goroutines for the watch loop
// and build notification handling that run until Stop is called.
func (ls *lifecycleService) Start(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "LifecycleService.Start")
	defer span.End()

	l.Internal("Starting lifecycle service...")

	if ls.componentRegistry != nil {
		if err := ls.discoverAndRegisterComponents(ctx); err != nil {
			l.Warn("Failed to discover local components", logger_domain.Error(err))
		}
	}

	ls.mu.Lock()
	entrypoints, err := ls.discoverInitialEntryPoints(ctx)
	if err != nil {
		ls.mu.Unlock()
		return fmt.Errorf("discovering initial entry points: %w", err)
	}
	ls.entryPoints = entrypoints
	ls.mu.Unlock()
	l.Internal("Initial entry points discovered", logger_domain.Int("count", len(entrypoints)))

	if ls.watcherAdapter != nil {
		staticDirs := ls.getStaticWatchDirs()
		events, err := ls.watcherAdapter.Watch(ctx, staticDirs, nil)
		if err != nil {
			return fmt.Errorf("failed to start file watcher: %w", err)
		}

		go ls.watchLoop(ctx, events)
		l.Internal("File watcher started", logger_domain.Int("watch_dirs", len(staticDirs)))
	}

	if ls.coordinatorService != nil {
		notifications, unsubscribe := ls.coordinatorService.Subscribe("lifecycle-service")
		ls.unsubscribe = unsubscribe
		go ls.handleBuildNotifications(ctx, notifications)
		l.Internal("Subscribed to build notifications")
	}

	span.SetStatus(codes.Ok, "Lifecycle service started")
	return nil
}

// Stop shuts down the lifecycle service and releases its resources.
//
// Returns error when the file watcher fails to close.
func (ls *lifecycleService) Stop(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "LifecycleService.Stop")
	defer span.End()

	var stopErr error
	ls.stopOnce.Do(func() {
		l.Internal("Stopping lifecycle service...")
		close(ls.stopChan)

		if ls.gcTimer != nil {
			l.Internal("Stopping post-startup GC timer")
			ls.gcTimer.Stop()
		}

		if ls.unsubscribe != nil {
			ls.unsubscribe()
		}

		if ls.watcherAdapter != nil {
			if err := ls.watcherAdapter.Close(); err != nil {
				stopErr = fmt.Errorf("failed to close file watcher: %w", err)
			}
		}
	})

	return stopErr
}

// RunInitialTasks runs one-time startup tasks such as asset seeding and
// configuration loading.
//
// Returns error when any critical task fails during startup.
//
// Spawns goroutines for each startup task and waits for them to complete.
// Schedules a garbage collection after the initial build period.
func (ls *lifecycleService) RunInitialTasks(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "RunInitialTasks")
	defer span.End()

	l.Internal("Running initial tasks...")

	var wg sync.WaitGroup
	errChan := make(chan error, initialTaskErrChanBuffer)

	workerLimit := runtime.NumCPU() * concurrencyMultiplier
	limiter := make(chan struct{}, workerLimit)
	l.Internal("Concurrency limit set for initial tasks", logger_domain.Int("workers", workerLimit))
	span.SetAttributes(attribute.Int("workerLimit", workerLimit))

	wg.Go(func() {
		if err := ls.seedThemeArtefact(ctx); err != nil {
			errChan <- fmt.Errorf("failed to seed theme artefact: %w", err)
		}
	})

	wg.Go(func() {
		if err := ls.seedCaptchaInitScripts(ctx); err != nil {
			errChan <- fmt.Errorf("failed to seed captcha init scripts: %w", err)
		}
	})

	wg.Go(func() {
		if err := ls.configProvider.LoadWebsiteConfig(); err != nil {
			l.Warn("Site config.json not loaded, some features may be disabled", logger_domain.Error(err))
		}
	})

	wg.Go(func() {
		ls.seedAllAssets(ctx, limiter)
	})

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "A parallel initial task failed")
			return fmt.Errorf("running initial tasks: %w", err)
		}
	}

	ls.gcTimer = ls.clock.AfterFunc(gcDelayAfterStartup, func() {
		l.Internal("Initial build period has passed. Forcing GC and attempting to release OS memory.")
		runtime.GC()
		debug.FreeOSMemory()
	})

	l.Internal("All initial tasks have been launched concurrently.")
	span.SetStatus(codes.Ok, "Initial tasks completed successfully")
	return nil
}

// GetEntryPoints returns the current set of discovered entry points.
//
// Returns []annotator_dto.EntryPoint which is a copy of the internal entry
// points slice.
//
// Safe for concurrent use. Returns a defensive copy to prevent data races.
func (ls *lifecycleService) GetEntryPoints() []annotator_dto.EntryPoint {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	result := make([]annotator_dto.EntryPoint, len(ls.entryPoints))
	copy(result, ls.entryPoints)
	return result
}

// RequestRebuild triggers a rebuild via the coordinator.
//
// Takes causationID (string) which identifies the cause of the rebuild request.
//
// Safe for concurrent use. Acquires a read lock to access entry points and
// action providers before delegating to the coordinator service.
func (ls *lifecycleService) RequestRebuild(ctx context.Context, causationID string) {
	if ls.coordinatorService == nil {
		return
	}

	ls.mu.RLock()
	currentEntryPoints := ls.entryPoints
	ls.mu.RUnlock()

	ls.coordinatorService.RequestRebuild(ctx, currentEntryPoints,
		coordinator_domain.WithCausationID(causationID))
}

// getAssetSourceDirs returns the full paths to all source directories.
//
// Returns []string which contains paths to configured source directories,
// including assets, pages, components, partials, and i18n.
func (ls *lifecycleService) getAssetSourceDirs() []string {
	var dirs []string
	paths := &ls.pathsConfig

	if paths.AssetsSourceDir != "" {
		dirs = append(dirs, ls.fs.Join(paths.BaseDir, paths.AssetsSourceDir))
	}
	if paths.PagesSourceDir != "" {
		dirs = append(dirs, ls.fs.Join(paths.BaseDir, paths.PagesSourceDir))
	}
	if paths.ComponentsSourceDir != "" {
		dirs = append(dirs, ls.fs.Join(paths.BaseDir, paths.ComponentsSourceDir))
	}
	if paths.PartialsSourceDir != "" {
		dirs = append(dirs, ls.fs.Join(paths.BaseDir, paths.PartialsSourceDir))
	}
	if paths.I18nSourceDir != "" {
		dirs = append(dirs, ls.fs.Join(paths.BaseDir, paths.I18nSourceDir))
	}

	return dirs
}

// seedAllAssets finds all asset files and adds them to the registry.
//
// Takes limiter (chan struct{}) which controls how many files are processed at
// once.
func (ls *lifecycleService) seedAllAssets(ctx context.Context, limiter chan struct{}) {
	ctx, span, l := log.Span(ctx, "seedAllAssets")
	defer span.End()

	assetDirs := ls.getAssetSourceDirs()
	l.Internal("Identified asset source directories for initial seeding", logger_domain.Field("dirs", assetDirs))

	allFilesToSeed := ls.discoverAssetFiles(ctx, assetDirs)

	l.Internal("Discovered initial files to seed into registry.", logger_domain.Int("file_count", len(allFilesToSeed)))
	span.SetAttributes(attribute.Int("file_count", len(allFilesToSeed)))

	ls.processFilesWithLimiter(ctx, allFilesToSeed, limiter)

	ls.seedExternalComponentFiles(ctx, limiter)

	l.Internal("Initial asset seeding complete.")
	span.SetStatus(codes.Ok, "Asset seeding complete")
}

// discoverAssetFiles walks all asset directories and collects relevant files.
//
// Takes ctx (context.Context) which allows cancellation of file discovery.
// Takes assetDirs ([]string) which lists directories to scan for assets.
//
// Returns []lifecycle_dto.FileEvent which contains all discovered files.
//
// Spawns one goroutine per directory; blocks until all walks complete.
func (ls *lifecycleService) discoverAssetFiles(ctx context.Context, assetDirs []string) []lifecycle_dto.FileEvent {
	fileChan := make(chan lifecycle_dto.FileEvent, fileEventChannelBuffer)
	var walkWg sync.WaitGroup
	walkWg.Add(len(assetDirs))

	for _, directory := range assetDirs {
		go ls.walkAssetDir(ctx, directory, fileChan, &walkWg)
	}

	go func() {
		walkWg.Wait()
		close(fileChan)
	}()

	allFiles := make([]lifecycle_dto.FileEvent, 0, fileEventChannelBuffer)
	for fileEvent := range fileChan {
		select {
		case <-ctx.Done():
			return allFiles
		default:
		}
		allFiles = append(allFiles, fileEvent)
	}
	return allFiles
}

// walkAssetDir walks a single directory and sends matching files to the
// channel.
//
// Takes ctx (context.Context) which carries the logger and cancellation.
// Takes directory (string) which specifies the directory path to walk.
// Takes fileChan (chan<- lifecycle_dto.FileEvent) which receives file events.
// Takes wg (*sync.WaitGroup) which tracks when this walk completes.
func (ls *lifecycleService) walkAssetDir(ctx context.Context, directory string, fileChan chan<- lifecycle_dto.FileEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	paths := &ls.pathsConfig

	_ = ls.fs.WalkDir(directory, func(path string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() {
			return nil
		}
		return ls.processWalkedFile(ctx, path, paths, fileChan)
	})
}

// processWalkedFile checks if a file is relevant and sends it to the channel.
//
// Takes ctx (context.Context) which carries the logger and cancellation.
// Takes path (string) which is the absolute path to the file.
// Takes paths (*LifecyclePathsConfig) which provides include and exclude rules.
// Takes fileChan (chan<- lifecycle_dto.FileEvent) which receives file events.
//
// Returns error when processing fails, or nil on success or when skipped.
func (ls *lifecycleService) processWalkedFile(ctx context.Context, path string, paths *LifecyclePathsConfig, fileChan chan<- lifecycle_dto.FileEvent) error {
	ctx, l := logger_domain.From(ctx, log)
	relPath, relErr := ls.fs.Rel(paths.BaseDir, path)
	if relErr != nil {
		l.Warn("Could not determine relative path during initial seed",
			logger_domain.String(fieldPath, path), logger_domain.Error(relErr))
		return nil
	}
	relPath = filepath.ToSlash(relPath)

	if !isRelevantFileForProcessing(relPath, paths) {
		l.Trace("Skipping initial seed of irrelevant file", logger_domain.String(fieldPath, relPath))
		return nil
	}

	fileChan <- lifecycle_dto.FileEvent{Path: path, Type: lifecycle_dto.FileEventTypeCreate}
	return nil
}

// processFilesWithLimiter processes files simultaneously with a
// semaphore limiter, spawning one task per file.
//
// Takes files ([]lifecycle_dto.FileEvent) which contains the file
// events to process.
// Takes limiter (chan struct{}) which controls throughput by
// acting as a semaphore.
//
// Concurrent goroutines are spawned per file, bounded by the
// semaphore limiter.
func (ls *lifecycleService) processFilesWithLimiter(ctx context.Context, files []lifecycle_dto.FileEvent, limiter chan struct{}) {
	var seedWg sync.WaitGroup
	for _, fileEvent := range files {
		select {
		case <-ctx.Done():
			seedWg.Wait()
			return
		default:
		}
		seedWg.Add(1)
		limiter <- struct{}{}
		go func(event lifecycle_dto.FileEvent) {
			defer seedWg.Done()
			defer goroutine.RecoverPanic(ctx, "lifecycle.processFile")
			defer func() { <-limiter }()
			ls.handleFileEvent(ctx, event, true)
		}(fileEvent)
	}
	seedWg.Wait()
}

// seedThemeArtefact creates the theme.css file and saves it to the registry.
//
// Returns error when building the CSS fails or the artefact cannot be saved.
func (ls *lifecycleService) seedThemeArtefact(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "seedThemeArtefact")
	defer span.End()

	l.Internal("Seeding theme.css artefact into registry...")

	if ls.renderer == nil {
		l.Internal("No renderer available, skipping theme artefact seeding")
		return nil
	}

	cssBytes, err := ls.renderer.BuildThemeCSS(ctx, &ls.configProvider.WebsiteConfig)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to build theme CSS: %w", err)
	}

	desiredProfiles := GetProfilesForFile(themeArtefactPath, nil)

	_, err = ls.registryService.UpsertArtefact(ctx, themeArtefactPath, themeArtefactPath, bytes.NewReader(cssBytes), "local_disk_cache", desiredProfiles)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create theme artefact with source: %w", err)
	}

	l.Internal("Successfully seeded theme.css artefact. Orchestrator will process it.")

	return nil
}

// seedCaptchaInitScripts registers captcha provider init scripts as artefacts
// in the registry. Each cloud provider's init script is a static JavaScript
// file that uses data-attribute selectors to find and initialise captcha
// widgets on the page.
//
// Returns error when an init script cannot be registered.
func (ls *lifecycleService) seedCaptchaInitScripts(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "seedCaptchaInitScripts")
	defer span.End()

	if ls.captchaService == nil || !ls.captchaService.IsEnabled() {
		l.Internal("Captcha service not configured, skipping init script seeding")
		return nil
	}

	providers := ls.captchaService.ListProviders(ctx)
	l.Internal("Seeding captcha init scripts", logger_domain.Int("provider_count", len(providers)))

	for _, providerInfo := range providers {
		provider, err := ls.captchaService.GetProviderByName(ctx, providerInfo.Name)
		if err != nil {
			l.Warn("Failed to resolve captcha provider for init script seeding",
				logger_domain.String("provider", providerInfo.Name),
				logger_domain.Error(err))
			continue
		}

		requirements := provider.RenderRequirements()
		if requirements.InitScript == nil || requirements.ServerSideToken {
			continue
		}

		scriptContent, err := requirements.InitScript()
		if err != nil {
			l.Warn("Failed to get captcha init script content",
				logger_domain.String("provider", providerInfo.Name),
				logger_domain.Error(err))
			continue
		}

		artefactID := fmt.Sprintf("captcha/init-%s.js", providerInfo.Name)
		desiredProfiles := GetProfilesForFile(artefactID, nil)

		_, err = ls.registryService.UpsertArtefact(
			ctx,
			artefactID,
			artefactID,
			bytes.NewReader([]byte(scriptContent)),
			"local_disk_cache",
			desiredProfiles,
		)
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("seeding captcha init script %q: %w", artefactID, err)
		}

		l.Internal("Seeded captcha init script",
			logger_domain.String("artefact_id", artefactID),
			logger_domain.Int("size_bytes", len(scriptContent)))
	}

	span.SetStatus(codes.Ok, "Captcha init scripts seeded")
	return nil
}

// entryPointDiscoveryConfig holds context for discovering entry points in a
// directory.
type entryPointDiscoveryConfig struct {
	// baseDir is the root directory used to calculate relative paths for entry
	// points.
	baseDir string

	// moduleName is the Go module path used to build relative import paths.
	moduleName string

	// isPage indicates whether this entry point is a page.
	isPage bool

	// isPublic indicates whether anyone can access the entry point.
	isPublic bool
}

// discoverInitialEntryPoints walks configured directories to find all .pk
// files.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry points
// from pages and partials directories.
// Returns error when no resolver is provided or the pages source directory does
// not exist.
func (ls *lifecycleService) discoverInitialEntryPoints(ctx context.Context) ([]annotator_dto.EntryPoint, error) {
	if ls.resolver == nil {
		return nil, errors.New("cannot discover entry points: no resolver provided")
	}

	paths := &ls.pathsConfig
	moduleName := ls.resolver.GetModuleName()

	pagesConfig := entryPointDiscoveryConfig{
		baseDir: paths.BaseDir, moduleName: moduleName, isPage: true, isPublic: true,
	}
	entryPoints, err := ls.discoverEntryPointsInDir(ctx, paths.PagesSourceDir, pagesConfig)
	if err != nil {
		if ls.fs.IsNotExist(err) {
			return nil, fmt.Errorf("pages source directory '%s' does not exist", paths.PagesSourceDir)
		}
		return nil, fmt.Errorf("discovering entry points in pages directory %q: %w", paths.PagesSourceDir, err)
	}

	partialsConfig := entryPointDiscoveryConfig{
		baseDir: paths.BaseDir, moduleName: moduleName, isPage: false, isPublic: false,
	}
	partials, err := ls.discoverEntryPointsInDir(ctx, paths.PartialsSourceDir, partialsConfig)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Could not walk partials source directory", logger_domain.Error(err))
	} else {
		entryPoints = append(entryPoints, partials...)
	}

	return entryPoints, nil
}

// discoverAndRegisterComponents walks the components directory and registers
// all .pkc files in the component registry for deterministic tag lookup.
//
// Component tag names are derived from the filename (without the .pkc
// extension). For example, "my-button.pkc" registers as tag name "my-button".
//
// Returns error when the directory cannot be walked.
func (ls *lifecycleService) discoverAndRegisterComponents(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "LifecycleService.discoverAndRegisterComponents")
	defer span.End()

	var totalRegistered int
	var totalErrors []string

	componentsDir := ls.pathsConfig.ComponentsSourceDir
	if componentsDir != "" {
		absDir := ls.fs.Join(ls.pathsConfig.BaseDir, componentsDir)
		registered, regErrors, err := ls.walkAndRegisterLocalComponents(ctx, absDir, ls.pathsConfig.BaseDir)
		if err != nil && !ls.fs.IsNotExist(err) {
			return fmt.Errorf("failed to walk components directory: %w", err)
		}
		totalRegistered += registered
		totalErrors = append(totalErrors, regErrors...)
	}

	externalDirs := ls.resolveExternalComponentDirs(ctx)
	for absDir := range externalDirs {
		registered, regErrors, err := ls.walkAndRegisterLocalComponents(ctx, absDir, absDir)
		if err != nil && !ls.fs.IsNotExist(err) {
			l.Warn("Failed to walk external component directory",
				logger_domain.String("dir", absDir),
				logger_domain.Error(err))
		}
		totalRegistered += registered
		totalErrors = append(totalErrors, regErrors...)
	}

	l.Internal("Component discovery complete",
		logger_domain.Int("registered", totalRegistered),
		logger_domain.Int("errors", len(totalErrors)),
	)

	return nil
}

// walkAndRegisterLocalComponents walks the directory and registers .pkc files.
//
// Takes ctx (context.Context) which carries the logger.
// Takes absDir (string) which is the absolute path to walk.
// Takes baseDir (string) which is the base for computing relative paths.
//
// Returns int which is the count of successfully registered components.
// Returns []string which contains error messages for failed registrations.
// Returns error when the directory walk itself fails.
func (ls *lifecycleService) walkAndRegisterLocalComponents(ctx context.Context, absDir, baseDir string) (int, []string, error) {
	ctx, l := logger_domain.From(ctx, log)
	var registered int
	var regErrors []string

	err := ls.fs.WalkDir(absDir, func(absPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if ls.fs.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pkc") {
			return nil
		}

		tagName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		relPath, _ := ls.fs.Rel(baseDir, absPath)

		definition := component_dto.ComponentDefinition{
			TagName:    tagName,
			SourcePath: relPath,
			IsExternal: false,
		}

		if regErr := ls.componentRegistry.Register(definition); regErr != nil {
			regErrors = append(regErrors, fmt.Sprintf("%s: %v", tagName, regErr))
			l.Warn("Failed to register component",
				logger_domain.String("tag_name", tagName),
				logger_domain.String(fieldPath, relPath),
				logger_domain.Error(regErr),
			)
		} else {
			registered++
			l.Internal("Registered local component",
				logger_domain.String("tag_name", tagName),
				logger_domain.String(fieldPath, relPath),
			)
		}
		return nil
	})

	return registered, regErrors, err
}

// discoverEntryPointsInDir walks a single directory to find .pk files.
//
// Takes directory (string) which is the relative path to search within the base
// directory.
// Takes discoveryConfig (entryPointDiscoveryConfig) which
// provides discovery settings including the base directory
// path.
//
// Returns []annotator_dto.EntryPoint which contains all discovered entry
// points from matching files.
// Returns error when the directory walk fails.
func (ls *lifecycleService) discoverEntryPointsInDir(ctx context.Context, directory string, discoveryConfig entryPointDiscoveryConfig) ([]annotator_dto.EntryPoint, error) {
	if directory == "" {
		return nil, nil
	}

	var entryPoints []annotator_dto.EntryPoint
	absDir := ls.fs.Join(discoveryConfig.baseDir, directory)

	if _, err := ls.fs.Stat(absDir); ls.fs.IsNotExist(err) {
		return nil, nil
	}

	err := ls.fs.WalkDir(absDir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Error during initial directory walk", logger_domain.Error(err))
			return nil
		}
		if ep := ls.tryCreateEntryPoint(absPath, d, discoveryConfig); ep != nil {
			entryPoints = append(entryPoints, *ep)
		}
		return nil
	})
	if err != nil {
		return entryPoints, fmt.Errorf("walking directory %q for entry points: %w", absDir, err)
	}

	return entryPoints, nil
}

// tryCreateEntryPoint checks if a directory entry is a valid .pk file and
// creates an entry point.
//
// Takes absPath (string) which is the absolute path to the file.
// Takes d (fs.DirEntry) which is the directory entry to check.
// Takes discoveryConfig (entryPointDiscoveryConfig) which
// provides discovery settings.
//
// Returns *annotator_dto.EntryPoint which is the created entry point, or nil
// if the entry is a directory or not a valid .pk file.
func (ls *lifecycleService) tryCreateEntryPoint(absPath string, d fs.DirEntry, discoveryConfig entryPointDiscoveryConfig) *annotator_dto.EntryPoint {
	if d.IsDir() || !isValidPKFile(d.Name()) {
		return nil
	}

	relPath, err := ls.fs.Rel(discoveryConfig.baseDir, absPath)
	if err != nil {
		return nil
	}
	pikoPath := filepath.ToSlash(ls.fs.Join(discoveryConfig.moduleName, relPath))

	return &annotator_dto.EntryPoint{
		Path:              pikoPath,
		IsPage:            discoveryConfig.isPage,
		IsPublic:          discoveryConfig.isPublic,
		IsEmail:           false,
		VirtualPageSource: nil,
	}
}

// getStaticWatchDirs finds the folders to watch for file changes.
//
// Returns []string which holds full paths to source folders, or nil if no
// resolver is set.
func (ls *lifecycleService) getStaticWatchDirs() []string {
	if ls.resolver == nil {
		return nil
	}

	paths := &ls.pathsConfig
	dirSet := make(map[string]struct{})

	addDir := func(dirPath string) {
		if dirPath != "" {
			absPath := dirPath
			if !filepath.IsAbs(dirPath) {
				absPath = ls.fs.Join(paths.BaseDir, dirPath)
			}
			dirSet[absPath] = struct{}{}
		}
	}

	addDir(paths.PagesSourceDir)
	addDir(paths.PartialsSourceDir)
	addDir(paths.ComponentsSourceDir)
	addDir(paths.I18nSourceDir)
	addDir("actions")
	addDir("pkg")
	addDir("cmd")
	addDir("internal")

	dirs := make([]string, 0, len(dirSet))
	for directory := range dirSet {
		dirs = append(dirs, directory)
	}

	return dirs
}

// NewLifecycleService creates a new lifecycle service with the provided
// dependencies.
//
// Takes deps (*LifecycleServiceDeps) which provides all required service
// dependencies. If FileSystem is nil, defaults to OS filesystem. If Clock is
// nil, defaults to real clock.
//
// Returns LifecycleService which is ready to manage lifecycle operations.
func NewLifecycleService(deps *LifecycleServiceDeps) LifecycleService {
	fileSystem := deps.FileSystem
	if fileSystem == nil {
		fileSystem = newOSFileSystem()
	}

	clk := deps.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	return &lifecycleService{
		pathsConfig:             deps.PathsConfig,
		configProvider:          deps.ConfigProvider,
		watcherAdapter:          deps.WatcherAdapter,
		registryService:         deps.RegistryService,
		coordinatorService:      deps.CoordinatorService,
		resolver:                deps.Resolver,
		renderRegistryPort:      deps.RenderRegistryPort,
		renderer:                deps.Renderer,
		fs:                      fileSystem,
		clock:                   clk,
		assetPipeline:           deps.AssetPipeline,
		interpretedOrchestrator: deps.InterpretedOrchestrator,
		templaterService:        deps.TemplaterService,
		routerManager:           deps.RouterManager,
		buildCacheInvalidator:   deps.BuildCacheInvalidator,
		devEventNotifier:        deps.DevEventNotifier,
		componentRegistry:       deps.ComponentRegistry,
		externalComponents:      deps.ExternalComponents,
		captchaService:          deps.CaptchaService,
		stopChan:                make(chan struct{}),
		unsubscribe:             nil,
		entryPoints:             nil,
		mu:                      sync.RWMutex{},
		stopOnce:                sync.Once{},
	}
}

// isValidPKFile checks if a filename is a valid PK template file.
//
// Takes name (string) which is the filename to check.
//
// Returns bool which is true if the file has a .pk extension and does not
// start with an underscore.
func isValidPKFile(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".pk") && !strings.HasPrefix(name, "_")
}
