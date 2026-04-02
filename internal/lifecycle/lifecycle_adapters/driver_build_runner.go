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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// themeArtefactPath is the artefact path for the generated theme CSS file.
	themeArtefactPath = "theme.css"

	// fieldError is the logging field name for error messages.
	fieldError = "error"

	// pollInterval is the delay between checks when waiting for background tasks.
	pollInterval = 50 * time.Millisecond

	// defaultTimeout is the longest time to wait for the dispatcher to become idle.
	defaultTimeout = 5 * time.Minute

	// pipelineFlushTimeout is the longest time to wait for the bridge to
	// process all artefact events published by the registry (Phase 1). If the
	// bridge has not caught up within this window, it likely indicates a lost
	// event or infrastructure failure.
	pipelineFlushTimeout = 2 * time.Minute

	// localDiskCacheSource is the origin label for artefacts loaded from disk.
	localDiskCacheSource = "local_disk_cache"

	// gcHintDrainBatchSize is the number of GC hints to pop per iteration when
	// draining hints after a build.
	gcHintDrainBatchSize = 100
)

// buildService implements lifecycle_domain.BuilderAdapter to manage the build
// process using configuration, registry, and orchestrator services.
type buildService struct {
	// registryService handles storing and fetching build artefacts.
	registryService registry_domain.RegistryService

	// orchestratorService provides access to the task dispatcher for coordination.
	orchestratorService orchestrator_domain.OrchestratorService

	// bridge provides artefact event processing through the lifecycle bridge.
	bridge lifecycle_domain.BridgeWithCounter

	// eventBus publishes build events for coordination.
	eventBus orchestrator_domain.EventBus

	// renderer builds CSS and other theme assets.
	renderer render_domain.RenderService

	// resolver provides module name lookup for building artefact IDs.
	resolver resolver_domain.ResolverPort

	// sandboxFactory creates sandboxes for filesystem access within the build
	// service.
	sandboxFactory safedisk.Factory

	// externalSandboxFactory creates sandboxes scoped to resolved external
	// module directories. It is built lazily by seedExternalComponentFiles
	// from the module roots discovered by the resolver, so external .pkc
	// and asset files get a real sandbox rather than falling back to NoOp.
	externalSandboxFactory safedisk.Factory

	// configProvider gives access to website and server settings.
	configProvider *config.Provider

	// pathsConfig holds the resolved path settings for file system operations.
	pathsConfig lifecycle_domain.LifecyclePathsConfig

	// externalComponents holds component definitions needing module resolution.
	externalComponents []component_dto.ComponentDefinition
}

// RunBuild executes a full production build, processing all assets and
// waiting for completion.
//
// Returns *lifecycle_domain.BuildResult which holds task counts and failure
// details. This is nil when an infrastructure error prevents the build from
// completing.
// Returns error when an infrastructure failure prevents the build from
// completing (context cancelled, flush timeout, theme build failure).
func (bs *buildService) RunBuild(ctx context.Context) (*lifecycle_domain.BuildResult, error) {
	ctx, span, l := log.Span(ctx, "runBuild")
	defer span.End()

	startTime := time.Now()

	dispatcher := bs.orchestratorService.GetTaskDispatcher()
	dispatcher.SetBuildTag(uuid.NewString())
	startStats := dispatcher.Stats()

	l.Internal("Starting full build process...")

	bs.loadWebsiteConfig(ctx)

	if err := bs.buildThemeArtefact(ctx); err != nil {
		return nil, fmt.Errorf("building theme artefact: %w", err)
	}

	fileCount, err := bs.processFilesInPipeline(ctx)
	if err != nil {
		return nil, fmt.Errorf("processing files in pipeline: %w", err)
	}

	if err := bs.seedExternalComponentFiles(ctx); err != nil {
		return nil, fmt.Errorf("seeding external component files: %w", err)
	}

	l.Internal("All assets have been seeded into the registry.",
		logger_domain.Int64("fileCount", fileCount))

	l.Internal("Waiting for all asset processing tasks to complete...")
	if err := bs.waitUntilIdle(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			cause := context.Cause(ctx)
			if cause == nil {
				cause = err
			}
			l.Warn("Build context cancelled while waiting for tasks to complete.",
				logger_domain.Error(err),
				logger_domain.String("cause", cause.Error()))
			return nil, fmt.Errorf("waiting for build tasks to complete: %w", err)
		}
		return nil, fmt.Errorf("asset processing failed while waiting for completion: %w", err)
	}

	bs.drainGCHints(ctx)

	result := bs.buildResult(ctx, startTime, startStats)
	if result.HasFailures() {
		span.SetStatus(codes.Error, "Build completed with task failures")
	} else {
		span.SetStatus(codes.Ok, "Build completed successfully")
	}
	return result, nil
}

// loadWebsiteConfig loads the website settings file.
// If loading fails, some features may not work, but this is not a fatal error.
func (bs *buildService) loadWebsiteConfig(ctx context.Context) {
	_, configSpan, configLog := log.Span(ctx, "loadWebsiteConfig")
	defer configSpan.End()

	if err := bs.configProvider.LoadWebsiteConfig(); err != nil {
		configLog.Warn("Site config.json not loaded, some features may be disabled", logger_domain.String(fieldError, err.Error()))
		configSpan.RecordError(err)
		configSpan.SetStatus(codes.Error, "Failed to load website config")
		return
	}
	configSpan.SetStatus(codes.Ok, "Website config loaded")
}

// buildThemeArtefact creates and stores the theme.css file.
//
// Returns error when building the CSS fails or storing the file fails.
func (bs *buildService) buildThemeArtefact(ctx context.Context) error {
	themeCtx, themeSpan, themeLog := log.Span(ctx, "generateThemeArtefact")
	defer themeSpan.End()

	cssBytes, err := bs.renderer.BuildThemeCSS(themeCtx, &bs.configProvider.WebsiteConfig)
	if err != nil {
		themeLog.Error("Failed to build theme CSS", logger_domain.Error(err))
		themeSpan.RecordError(err)
		return fmt.Errorf("failed to build theme CSS: %w", err)
	}

	desiredProfiles := lifecycle_domain.GetProfilesForFile(themeArtefactPath, nil)
	_, err = bs.registryService.UpsertArtefact(themeCtx, themeArtefactPath, themeArtefactPath, bytes.NewReader(cssBytes), localDiskCacheSource, desiredProfiles)
	if err != nil {
		themeLog.Error("Failed to upsert theme artefact", logger_domain.Error(err))
		themeSpan.RecordError(err)
		return fmt.Errorf("failed to upsert theme artefact: %w", err)
	}

	themeLog.Trace("Successfully generated and seeded theme.css artefact.")
	themeSpan.SetStatus(codes.Ok, "Theme artefact seeded")
	return nil
}

// processFilesInPipeline walks directories and processes files using a worker
// pool.
//
// Returns int64 which is the total number of files processed.
// Returns error when the directory walk or file processing fails.
func (bs *buildService) processFilesInPipeline(ctx context.Context) (int64, error) {
	const numWorkers = 2
	fileEvents := make(chan lifecycle_dto.FileEvent, numWorkers*2)
	var fileCount atomic.Int64
	var wg sync.WaitGroup

	ctx, walkSpan, _ := log.Span(ctx, "walkAndProcessFiles",
		logger_domain.String("operation", "pipelinedProcessing"),
		logger_domain.Int("workerCount", numWorkers),
	)
	defer walkSpan.End()

	bs.startFileProcessingWorkers(ctx, numWorkers, fileEvents, &wg)

	if err := bs.walkAndStreamFiles(ctx, fileEvents, &fileCount); err != nil {
		close(fileEvents)
		wg.Wait()
		return 0, fmt.Errorf("walking and streaming files: %w", err)
	}

	close(fileEvents)

	wg.Wait()

	walkSpan.SetAttributes(attribute.Int64("fileCount", fileCount.Load()))
	return fileCount.Load(), nil
}

// startFileProcessingWorkers starts the specified number of worker
// goroutines to process file events.
//
// Takes numWorkers (int) which specifies how many goroutines to spawn.
// Takes fileEvents (<-chan lifecycle_dto.FileEvent) which provides the events
// to process.
// Takes wg (*sync.WaitGroup) which tracks worker completion.
func (bs *buildService) startFileProcessingWorkers(
	ctx context.Context,
	numWorkers int,
	fileEvents <-chan lifecycle_dto.FileEvent,
	wg *sync.WaitGroup,
) {
	for i := range numWorkers {
		wg.Go(func() {
			for event := range fileEvents {
				select {
				case <-ctx.Done():
					return
				default:
					bs.handleFileEvent(ctx, event)
				}
			}
		})
		_ = i
	}
}

// walkAndStreamFiles walks directories and streams file events to the channel.
//
// Takes fileEvents (chan<- lifecycle_dto.FileEvent) which receives file events
// for each file found during the walk.
// Takes fileCount (*atomic.Int64) which is incremented for each file streamed.
//
// Returns error when a directory walk fails.
func (bs *buildService) walkAndStreamFiles(
	ctx context.Context,
	fileEvents chan<- lifecycle_dto.FileEvent,
	fileCount *atomic.Int64,
) error {
	ctx, walkLog := logger_domain.From(ctx, log)

	dirs := bs.getDirectories()
	for _, directory := range dirs {
		walkLog.Trace("Walking directory", logger_domain.String(fieldDir, directory))

		err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				walkLog.Trace("Error walking directory", logger_domain.String("path", path), logger_domain.String(fieldError, err.Error()))
				return nil
			}
			if d.IsDir() {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case fileEvents <- lifecycle_dto.FileEvent{Path: path, Type: lifecycle_dto.FileEventTypeCreate}:
				fileCount.Add(1)
			}
			return nil
		})
		if err != nil {
			walkLog.Error("Failed to walk directory", logger_domain.String(fieldDir, directory), logger_domain.Error(err))
			return fmt.Errorf("walking directory %q: %w", directory, err)
		}
	}
	return nil
}

// waitUntilIdle waits until all work is complete using state polling.
//
// This uses a two-phase approach to avoid race conditions:
//
// Phase 1 (Pipeline Flush): Wait until the bridge has processed all artefact
// events published by the registry. This means all tasks have been sent.
//
// Phase 2 (Work Completion): Poll the dispatcher until it reports idle state:
// all tasks completed, all queues empty, no pending retries.
//
// Returns error when the pipeline flush or dispatcher idle wait fails.
func (bs *buildService) waitUntilIdle(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "buildService.waitUntilIdle")
	defer span.End()

	startTime := time.Now()

	l.Internal("Phase 1: Waiting for pipeline flush (bridge to process all artefact events)")
	if err := bs.waitForPipelineFlush(ctx, startTime); err != nil {
		return fmt.Errorf("waiting for pipeline flush: %w", err)
	}

	l.Internal("Phase 2: Waiting for dispatcher to be idle")
	if err := bs.waitForDispatcherIdle(ctx, startTime, span); err != nil {
		return fmt.Errorf("waiting for dispatcher to become idle: %w", err)
	}

	span.SetStatus(codes.Ok, "Build completed")
	return nil
}

// waitForPipelineFlush waits until the bridge has processed all artefact
// events sent by the registry.
//
// Takes startTime (time.Time) which marks when the build started for timeout
// checks.
//
// Returns error when the context is cancelled or the flush times out.
func (bs *buildService) waitForPipelineFlush(ctx context.Context, startTime time.Time) error {
	ctx, l := logger_domain.From(ctx, log)
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for pipeline flush: %w", err)
		}

		if time.Since(startTime) > pipelineFlushTimeout {
			published := bs.registryService.ArtefactEventsPublished()
			processed := bs.bridge.ArtefactEventsProcessed()
			return fmt.Errorf("pipeline flush timeout after %v: published=%d processed=%d remaining=%d",
				pipelineFlushTimeout, published, processed, published-processed)
		}

		published := bs.registryService.ArtefactEventsPublished()
		processed := bs.bridge.ArtefactEventsProcessed()

		if published == 0 {
			l.Internal("No artefact events published, skipping pipeline flush")
			return nil
		}

		if processed >= published {
			l.Internal("Pipeline flushed",
				logger_domain.Int64("eventsPublished", published),
				logger_domain.Int64("eventsProcessed", processed))
			return nil
		}

		l.Internal("Waiting for pipeline flush",
			logger_domain.Int64("eventsPublished", published),
			logger_domain.Int64("eventsProcessed", processed),
			logger_domain.Int64("remaining", published-processed))

		time.Sleep(pollInterval)
	}
}

// waitForDispatcherIdle polls the dispatcher until it reports idle state.
//
// Takes startTime (time.Time) which marks when the build started for timeout
// checks.
// Takes span (trace.Span) which receives tracing events for observability.
//
// Returns error when the context is cancelled or the build times out.
func (bs *buildService) waitForDispatcherIdle(
	ctx context.Context,
	startTime time.Time,
	span trace.Span,
) error {
	ctx, _ = logger_domain.From(ctx, log)

	dispatcher := bs.orchestratorService.GetTaskDispatcher()
	lastLogTime := time.Now()

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("waiting for dispatcher idle: %w", err)
		}

		if err := bs.checkBuildTimeout(ctx, startTime, dispatcher, span); err != nil {
			return fmt.Errorf("checking build timeout: %w", err)
		}

		if isBuildIdle(dispatcher) {
			bs.logDispatcherIdle(ctx, startTime, dispatcher)
			return nil
		}

		bs.logDispatcherProgress(ctx, &lastLogTime, dispatcher)
		time.Sleep(pollInterval)
	}
}

// isBuildIdle checks if the dispatcher has finished all build work.
// Unlike dispatcher.IsIdle(), this ignores delayed tasks (e.g. GC tasks
// scheduled for the future) which should not block build completion.
func isBuildIdle(dispatcher orchestrator_domain.TaskDispatcher) bool {
	stats := dispatcher.Stats()
	allDone := stats.TasksDispatched <= stats.TasksCompleted+stats.TasksFailed+stats.TasksRetried
	return allDone && stats.ActiveWorkers == 0
}

// checkBuildTimeout checks if the build has gone over the time limit.
//
// Takes ctx (context.Context) which carries the logger.
// Takes startTime (time.Time) which is when the build started.
// Takes dispatcher (orchestrator_domain.TaskDispatcher) which gives task queue
// details for error reporting.
// Takes span (trace.Span) which is the tracing span for error reporting.
//
// Returns error when the time since startTime is more than the default timeout.
func (*buildService) checkBuildTimeout(
	ctx context.Context,
	startTime time.Time,
	dispatcher orchestrator_domain.TaskDispatcher,
	span trace.Span,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if time.Since(startTime) <= defaultTimeout {
		return nil
	}

	stats := dispatcher.Stats()
	err := fmt.Errorf("timeout waiting for dispatcher to become idle after %v", defaultTimeout)
	l.ReportError(span, err, "Build timeout",
		logger_domain.Int64("tasksDispatched", stats.TasksDispatched),
		logger_domain.Int64("tasksCompleted", stats.TasksCompleted),
		logger_domain.Int64("tasksFailed", stats.TasksFailed),
		logger_domain.Int("highQueueLen", stats.HighQueueLen),
		logger_domain.Int("normalQueueLen", stats.NormalQueueLen),
		logger_domain.Int("lowQueueLen", stats.LowQueueLen))
	return fmt.Errorf("build timeout: %w", err)
}

// logDispatcherIdle logs that the dispatcher has become idle.
//
// Takes ctx (context.Context) which carries the logger.
// Takes startTime (time.Time) which marks when the build started.
// Takes dispatcher (orchestrator_domain.TaskDispatcher) which provides stats.
func (*buildService) logDispatcherIdle(
	ctx context.Context,
	startTime time.Time,
	dispatcher orchestrator_domain.TaskDispatcher,
) {
	ctx, l := logger_domain.From(ctx, log)
	stats := dispatcher.Stats()
	duration := time.Since(startTime)
	l.Internal("Dispatcher is idle, build complete",
		logger_domain.Duration("waitDuration", duration),
		logger_domain.Int64("tasksDispatched", stats.TasksDispatched),
		logger_domain.Int64("tasksCompleted", stats.TasksCompleted),
		logger_domain.Int64("tasksFailed", stats.TasksFailed))
}

// buildResult collects dispatcher statistics and any failure details into a
// BuildResult struct for the caller to inspect and format.
//
// Takes ctx (context.Context) which is passed to the dispatcher for querying
// failed tasks.
// Takes buildStartTime (time.Time) which marks when the build started.
// Takes startStats (orchestrator_domain.DispatcherStats) which holds the
// counter snapshot taken at the beginning of the build so that only the
// delta for this run is reported.
//
// Returns *lifecycle_domain.BuildResult which holds the build outcome.
func (bs *buildService) buildResult(
	ctx context.Context,
	buildStartTime time.Time,
	startStats orchestrator_domain.DispatcherStats,
) *lifecycle_domain.BuildResult {
	ctx, l := logger_domain.From(ctx, log)
	dispatcher := bs.orchestratorService.GetTaskDispatcher()
	endStats := dispatcher.Stats()

	result := &lifecycle_domain.BuildResult{
		TotalDispatched:  endStats.TasksDispatched - startStats.TasksDispatched,
		TotalCompleted:   endStats.TasksCompleted - startStats.TasksCompleted,
		TotalFailed:      endStats.TasksFailed - startStats.TasksFailed,
		TotalFatalFailed: endStats.TasksFatalFailed - startStats.TasksFatalFailed,
		TotalRetried:     endStats.TasksRetried - startStats.TasksRetried,
		Duration:         time.Since(buildStartTime),
	}

	failed := endStats.TasksFailed - startStats.TasksFailed
	if failed > 0 {
		failures, err := dispatcher.FailedTasks(ctx)
		if err != nil {
			l.Warn("Failed to retrieve task failure details", logger_domain.Error(err))
		} else {
			result.Failures = make([]lifecycle_domain.BuildFailure, len(failures))
			for i, f := range failures {
				result.Failures[i] = lifecycle_domain.BuildFailure{
					ArtefactID: f.WorkflowID,
					Executor:   f.Executor,
					Error:      f.LastError,
					Attempt:    f.Attempt,
					IsFatal:    f.IsFatal,
				}
			}
		}
	}

	return result
}

// logDispatcherProgress logs the dispatcher progress at most once per second
// to prevent excessive log output.
//
// Takes ctx (context.Context) which carries the logger.
// Takes lastLogTime (*time.Time) which tracks when the last log was written.
// Takes dispatcher (TaskDispatcher) which provides the current task counts.
func (*buildService) logDispatcherProgress(
	ctx context.Context,
	lastLogTime *time.Time,
	dispatcher orchestrator_domain.TaskDispatcher,
) {
	ctx, l := logger_domain.From(ctx, log)
	if time.Since(*lastLogTime) < time.Second {
		return
	}

	stats := dispatcher.Stats()
	l.Notice("Waiting for dispatcher to become idle",
		logger_domain.Int64("tasksDispatched", stats.TasksDispatched),
		logger_domain.Int64("tasksCompleted", stats.TasksCompleted),
		logger_domain.Int64("tasksFailed", stats.TasksFailed),
		logger_domain.Int("highQueueLen", stats.HighQueueLen),
		logger_domain.Int("normalQueueLen", stats.NormalQueueLen),
		logger_domain.Int("lowQueueLen", stats.LowQueueLen),
		logger_domain.Int("activeWorkers", int(stats.ActiveWorkers)))
	*lastLogTime = time.Now()
}

// handleFileEvent processes a file system event and updates the artefact store.
//
// Takes event (lifecycle_dto.FileEvent) which contains the file path and event
// type to process.
func (bs *buildService) handleFileEvent(ctx context.Context, event lifecycle_dto.FileEvent) {
	ctx, span, l := log.Span(ctx, "handleFileEvent",
		logger_domain.String("path", event.Path),
		logger_domain.Field("eventType", event.Type),
	)
	defer span.End()

	artefactID, relPath, err := bs.computeArtefactID(ctx, event.Path, span)
	if err != nil {
		return
	}

	l = l.With(logger_domain.String("artefactID", artefactID))
	l.Trace("Processing file")

	fileData, err := bs.openFileForProcessing(ctx, event.Path, span)
	if err != nil {
		return
	}
	defer func() { _ = fileData.Close() }()

	bs.upsertFileArtefact(ctx, artefactID, relPath, fileData, span)
}

// computeArtefactID computes the module-absolute artefact ID and relative path
// from a file path. This keeps artefact IDs matching what templates specify
// (e.g., "mymodule/lib/icons/arrow.svg").
//
// Takes ctx (context.Context) which carries the logger.
// Takes filePath (string) which is the absolute path to the file.
// Takes span (trace.Span) which records errors for tracing.
//
// Returns artefactID (string) which is the module-prefixed artefact
// ID.
// Returns relPath (string) which is the project-relative path for
// use as sourcePath.
// Returns error when the relative path cannot be computed.
func (bs *buildService) computeArtefactID(
	ctx context.Context,
	filePath string,
	span trace.Span,
) (artefactID string, relPath string, err error) {
	ctx, l := logger_domain.From(ctx, log)
	relPath, err = filepath.Rel(bs.pathsConfig.BaseDir, filePath)
	if err != nil {
		l.Warn("Failed to compute relative path", logger_domain.String(fieldError, err.Error()))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to compute relative path")
		return "", "", fmt.Errorf("computing relative path for %q: %w", filePath, err)
	}
	relPathSlash := filepath.ToSlash(relPath)
	moduleName := bs.resolver.GetModuleName()
	return moduleName + "/" + relPathSlash, relPathSlash, nil
}

// sandboxedFile wraps a file handle and its sandbox, ensuring both are closed
// when the file is closed.
type sandboxedFile struct {
	// file is the underlying file handle for read operations.
	file io.ReadCloser

	// sandbox provides safe file system operations for the sandboxed file.
	sandbox safedisk.Sandbox
}

// Read reads from the underlying file.
//
// Takes p ([]byte) which is the buffer to read into.
//
// Returns n (int) which is the number of bytes read.
// Returns err (error) when the read fails.
func (sf *sandboxedFile) Read(p []byte) (n int, err error) {
	return sf.file.Read(p)
}

// Close closes both the file and the sandbox.
//
// Returns error when closing the file or sandbox fails.
func (sf *sandboxedFile) Close() error {
	fileErr := sf.file.Close()
	sandboxErr := sf.sandbox.Close()
	if fileErr != nil {
		return fmt.Errorf("closing sandboxed file: %w", fileErr)
	}
	if sandboxErr != nil {
		return fmt.Errorf("closing sandbox: %w", sandboxErr)
	}
	return nil
}

// openFileForProcessing opens a file and returns the file handle.
//
// Takes filePath (string) which specifies the path to the file to open.
// Takes parentSpan (trace.Span) which receives error status on failure.
//
// Returns io.ReadCloser which provides access to the file contents.
// Returns error when the sandbox cannot be created or the file cannot be
// opened.
func (bs *buildService) openFileForProcessing(
	ctx context.Context,
	filePath string,
	parentSpan trace.Span,
) (io.ReadCloser, error) {
	_, readSpan, readLog := log.Span(ctx, "readFile",
		logger_domain.String("path", filePath),
	)
	defer readSpan.End()

	parentDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	var sandbox safedisk.Sandbox
	var err error
	if bs.sandboxFactory != nil {
		sandbox, err = bs.sandboxFactory.Create("build-read-file", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		readLog.Error("Failed to create sandbox for file", logger_domain.String(fieldError, err.Error()))
		readSpan.RecordError(err)
		readSpan.SetStatus(codes.Error, "Failed to create sandbox")
		parentSpan.RecordError(err)
		parentSpan.SetStatus(codes.Error, "File event processing failed")
		return nil, fmt.Errorf("creating sandbox for %q: %w", filePath, err)
	}

	fileData, err := sandbox.Open(fileName)
	if err != nil {
		_ = sandbox.Close()
		readLog.Error("Failed to read updated file", logger_domain.String(fieldError, err.Error()))
		readSpan.RecordError(err)
		readSpan.SetStatus(codes.Error, "Failed to read file")
		parentSpan.RecordError(err)
		parentSpan.SetStatus(codes.Error, "File event processing failed")
		return nil, fmt.Errorf("opening file %q: %w", filePath, err)
	}

	readSpan.SetStatus(codes.Ok, "File read successfully")
	return &sandboxedFile{file: fileData, sandbox: sandbox}, nil
}

// upsertFileArtefact upserts the file artefact into the registry.
//
// Takes artefactID (string) which identifies the artefact to upsert.
// Takes sourcePath (string) which is the project-relative path for storage key
// generation.
// Takes fileData (io.ReadCloser) which provides the artefact content.
// Takes parentSpan (trace.Span) which receives error status on failure.
func (bs *buildService) upsertFileArtefact(
	ctx context.Context,
	artefactID string,
	sourcePath string,
	fileData io.ReadCloser,
	parentSpan trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)
	profiles := lifecycle_domain.GetProfilesForFile(artefactID, nil)
	l.Trace("Upserting artefact with profiles", logger_domain.Int("profileCount", len(profiles)))

	upsertCtx, upsertSpan, upsertLog := log.Span(ctx, "upsertArtefact",
		logger_domain.String("artefactID", artefactID),
		logger_domain.Int("profileCount", len(profiles)),
	)
	defer upsertSpan.End()

	meta, err := bs.registryService.UpsertArtefact(upsertCtx, artefactID, sourcePath, fileData, localDiskCacheSource, profiles)
	if err != nil {
		upsertLog.Error("Failed to upsert source artefact", logger_domain.String(fieldError, err.Error()))
		upsertSpan.RecordError(err)
		upsertSpan.SetStatus(codes.Error, "Failed to upsert artefact")
		parentSpan.RecordError(err)
		parentSpan.SetStatus(codes.Error, "File event processing failed")
		return
	}

	if meta != nil {
		for i := range meta.ActualVariants {
			if meta.ActualVariants[i].SRIHash != "" {
				daemon_frontend.SetSRIHash(artefactID, meta.ActualVariants[i].SRIHash)
				break
			}
		}
	}

	upsertLog.Trace("Successfully upserted artefact")
	upsertSpan.SetStatus(codes.Ok, "Artefact upserted successfully")
}

// getDirectories returns the list of source directories to watch for changes.
//
// Returns []string which contains the full paths for assets and components
// directories based on the current settings.
func (bs *buildService) getDirectories() []string {
	_, span, l := log.Span(context.Background(), "getDirectories")
	defer span.End()

	var dirs []string

	l.Internal("Determining directories from configuration")

	if bs.pathsConfig.AssetsSourceDir != "" {
		path := filepath.Join(bs.pathsConfig.BaseDir, bs.pathsConfig.AssetsSourceDir)
		dirs = append(dirs, path)
		l.Trace("Added assets directory to list", logger_domain.String(fieldDir, path))
		span.SetAttributes(attribute.String("assetsDir", path))
	}

	if bs.pathsConfig.ComponentsSourceDir != "" {
		path := filepath.Join(bs.pathsConfig.BaseDir, bs.pathsConfig.ComponentsSourceDir)
		dirs = append(dirs, path)
		l.Trace("Added components directory to list", logger_domain.String(fieldDir, path))
		span.SetAttributes(attribute.String("componentsDir", path))
	}

	l.Internal("Directories determined", logger_domain.Int("dirCount", len(dirs)))
	span.SetAttributes(attribute.Int("dirCount", len(dirs)))
	return dirs
}

// seedExternalComponentFiles resolves external component module directories
// and seeds their .pkc files into the registry with module-path-based artefact
// IDs. This mirrors the lifecycle service's seeding but uses the build
// service's direct file access instead of the filesystem abstraction.
//
// Returns error when directory resolution fails fatally.
func (bs *buildService) seedExternalComponentFiles(ctx context.Context) error {
	dirMap := bs.resolveExternalComponentDirs(ctx)
	assetDirMap := bs.resolveExternalAssetDirs(ctx)

	if err := bs.buildExternalSandboxFactory(ctx, dirMap, assetDirMap); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to create sandbox factory for external modules, external files will be skipped",
			logger_domain.Error(err))
		return nil
	}

	for absDir, modulePath := range dirMap {
		if err := bs.walkAndSeedExternalDir(ctx, absDir, modulePath); err != nil {
			return fmt.Errorf("seeding external component directory %q: %w", absDir, err)
		}
	}

	for absDir, artefactPrefix := range assetDirMap {
		if err := bs.walkAndSeedExternalAssetDir(ctx, absDir, artefactPrefix); err != nil {
			return fmt.Errorf("seeding external asset directory %q: %w", absDir, err)
		}
	}

	return nil
}

// buildExternalSandboxFactory creates a sandbox factory scoped to the resolved
// external module directories. This gives external component and asset files
// a proper sandbox backed by os.Root, without needing the project's sandbox
// factory (which is scoped to the project's source directory).
//
// Takes componentDirs (map[string]string) which maps absolute directory paths
// to their external component module identifiers.
// Takes assetDirs (map[string]string) which maps absolute directory paths to
// their external asset module identifiers.
//
// Returns error when the sandbox factory cannot be created.
func (bs *buildService) buildExternalSandboxFactory(
	ctx context.Context,
	componentDirs map[string]string,
	assetDirs map[string]string,
) error {
	if len(componentDirs) == 0 && len(assetDirs) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var allowedPaths []string
	for absDir := range componentDirs {
		if _, ok := seen[absDir]; !ok {
			seen[absDir] = struct{}{}
			allowedPaths = append(allowedPaths, absDir)
		}
	}
	for absDir := range assetDirs {
		if _, ok := seen[absDir]; !ok {
			seen[absDir] = struct{}{}
			allowedPaths = append(allowedPaths, absDir)
		}
	}

	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      true,
		AllowedPaths: allowedPaths,
	})
	if err != nil {
		return fmt.Errorf("creating external sandbox factory: %w", err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Created sandbox factory for external modules",
		logger_domain.Int("allowed_paths", len(allowedPaths)))

	bs.externalSandboxFactory = factory
	return nil
}

// resolveExternalComponentDirs returns a map of absolute directory paths to
// their module paths by resolving each unique ModulePath in the external
// component definitions.
//
// Returns map[string]string where keys are absolute directories and values are
// the original module paths.
func (bs *buildService) resolveExternalComponentDirs(ctx context.Context) map[string]string {
	if len(bs.externalComponents) == 0 || bs.resolver == nil {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	seen := make(map[string]struct{}, len(bs.externalComponents))
	var modulePaths []string
	for _, definition := range bs.externalComponents {
		if definition.ModulePath == "" {
			continue
		}
		if _, ok := seen[definition.ModulePath]; !ok {
			seen[definition.ModulePath] = struct{}{}
			modulePaths = append(modulePaths, definition.ModulePath)
		}
	}

	result := make(map[string]string, len(modulePaths))
	for _, mp := range modulePaths {
		moduleBase, subpath, err := bs.resolver.FindModuleBoundary(ctx, mp)
		if err != nil {
			l.Warn("Failed to find module boundary for external component",
				logger_domain.String("module_path", mp),
				logger_domain.Error(err))
			continue
		}

		moduleDir, err := bs.resolver.GetModuleDir(ctx, moduleBase)
		if err != nil {
			l.Warn("Failed to resolve module directory for external component",
				logger_domain.String("module_base", moduleBase),
				logger_domain.Error(err))
			continue
		}

		absDir := filepath.Join(moduleDir, subpath)
		result[absDir] = mp
		l.Internal("Resolved external component directory",
			logger_domain.String("module_path", mp),
			logger_domain.String("resolved_dir", absDir))
	}

	return result
}

// walkSeedParams groups the parameters that differ between directory-seeding
// call sites, keeping the shared walkAndSeedDirectory method's signature short.
type walkSeedParams struct {
	// spanName is the OpenTelemetry span identifier.
	spanName string

	// labelKey is the structured-logging field key for the second span attribute.
	labelKey string

	// labelValue is the structured-logging field value for the second span attribute.
	labelValue string

	// walkFunction is the callback passed to filepath.WalkDir.
	walkFunction fs.WalkDirFunc

	// warnMessage is logged when the directory walk encounters an error.
	warnMessage string

	// statusMessage is recorded on the span when seeding completes.
	statusMessage string
}

// walkAndSeedDirectory walks a single directory and seeds its entries into the
// registry, using the callback and labelling provided by the given parameters.
//
// Takes absDir (string) which is the absolute path to walk.
// Takes parameters (walkSeedParams) which configures span names, labels, walk
// callback, and log messages.
//
// Returns error when the directory walk itself fails fatally.
func (*buildService) walkAndSeedDirectory(
	ctx context.Context,
	absDir string,
	parameters walkSeedParams,
) error {
	_, span, l := log.Span(ctx, parameters.spanName,
		logger_domain.String(fieldDir, absDir),
		logger_domain.String(parameters.labelKey, parameters.labelValue),
	)
	defer span.End()

	walkErr := filepath.WalkDir(absDir, parameters.walkFunction)

	if walkErr != nil {
		l.Warn(parameters.warnMessage,
			logger_domain.String(fieldDir, absDir),
			logger_domain.Error(walkErr))
	}

	span.SetStatus(codes.Ok, parameters.statusMessage)
	return nil
}

// walkAndSeedExternalDir walks a single external directory and seeds .pkc
// files into the registry with artefact IDs prefixed by the module path.
//
// Takes absDir (string) which is the absolute path to walk.
// Takes modulePath (string) which provides the artefact ID prefix.
//
// Returns error when the directory walk itself fails fatally.
func (bs *buildService) walkAndSeedExternalDir(
	ctx context.Context,
	absDir, modulePath string,
) error {
	return bs.walkAndSeedDirectory(ctx, absDir, walkSeedParams{
		spanName:   "seedExternalComponentDir",
		labelKey:   "module_path",
		labelValue: modulePath,
		walkFunction: func(absPath string, entry os.DirEntry, err error) error {
			return bs.seedExternalComponentEntry(ctx, absDir, modulePath, absPath, entry, err)
		},
		warnMessage:   "Failed to walk external component directory",
		statusMessage: "External component directory seeded",
	})
}

// seedExternalComponentEntry processes a single directory entry during
// an external component directory walk, seeding .pkc files into the
// registry.
//
// Takes absDir (string) which is the root directory being walked.
// Takes modulePath (string) which provides the artefact ID prefix.
// Takes absPath (string) which is the absolute path of the current entry.
// Takes entry (os.DirEntry) which holds details about the directory entry.
// Takes walkError (error) which is any error encountered during the walk.
//
// Returns error which is always nil; errors are logged and the walk continues.
func (bs *buildService) seedExternalComponentEntry(
	ctx context.Context,
	absDir, modulePath, absPath string,
	entry os.DirEntry,
	walkError error,
) error {
	if walkError != nil {
		return nil
	}
	if entry.IsDir() || filepath.Ext(entry.Name()) != ".pkc" {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	relPath, relErr := filepath.Rel(absDir, absPath)
	if relErr != nil {
		return nil
	}
	relPathSlash := filepath.ToSlash(relPath)
	artefactID := modulePath + "/" + relPathSlash

	parentDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	sandbox, sandboxErr := bs.createExternalSandbox("build-external-component", parentDir)
	if sandboxErr != nil {
		l.Error("Failed to create sandbox for external component file",
			logger_domain.String(fieldPath, absPath),
			logger_domain.Error(sandboxErr))
		return nil
	}

	file, openErr := sandbox.Open(fileName)
	if openErr != nil {
		_ = sandbox.Close()
		l.Error("Failed to read external component file",
			logger_domain.String(fieldPath, absPath),
			logger_domain.Error(openErr))
		return nil
	}
	defer func() { _ = file.Close(); _ = sandbox.Close() }()

	profiles := lifecycle_domain.GetProfilesForFile(artefactID, nil)
	if _, upsertErr := bs.registryService.UpsertArtefact(ctx, artefactID, relPathSlash, file, localDiskCacheSource, profiles); upsertErr != nil {
		l.Error("Failed to seed external component artefact",
			logger_domain.String("artefact_id", artefactID),
			logger_domain.Error(upsertErr))
	}

	return nil
}

// externalAssetKey identifies a unique (moduleBase, assetPath) pair for
// external asset directory deduplication.
type externalAssetKey struct {
	// moduleBase is the Go module path prefix for the external component.
	moduleBase string

	// assetPath is the relative path to the asset directory within the module.
	assetPath string
}

// resolveExternalAssetDirs collects unique (moduleBase, assetPath) pairs from
// external component definitions and resolves them to absolute directories.
//
// Returns map[string]string which maps absolute asset directory paths to their
// artefact ID prefixes (e.g. "piko.sh/piko/lib/icons").
func (bs *buildService) resolveExternalAssetDirs(ctx context.Context) map[string]string {
	if len(bs.externalComponents) == 0 || bs.resolver == nil {
		return nil
	}

	ctx, l := logger_domain.From(ctx, log)

	pairs := bs.collectExternalAssetPairs(ctx)
	return bs.resolveAssetPairsToDirectories(ctx, l, pairs)
}

// collectExternalAssetPairs builds a deduplicated list of (moduleBase,
// assetPath) pairs from the external component definitions.
//
// Returns []externalAssetKey which contains the unique pairs.
func (bs *buildService) collectExternalAssetPairs(ctx context.Context) []externalAssetKey {
	seen := make(map[externalAssetKey]struct{})
	var pairs []externalAssetKey

	for _, definition := range bs.externalComponents {
		if definition.ModulePath == "" || len(definition.AssetPaths) == 0 {
			continue
		}
		moduleBase, _, err := bs.resolver.FindModuleBoundary(ctx, definition.ModulePath)
		if err != nil {
			continue
		}
		for _, ap := range definition.AssetPaths {
			key := externalAssetKey{moduleBase: moduleBase, assetPath: ap}
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				pairs = append(pairs, key)
			}
		}
	}

	return pairs
}

// resolveAssetPairsToDirectories resolves each (moduleBase, assetPath) pair to
// an absolute directory and its artefact ID prefix.
//
// Takes l (logger_domain.Logger) which provides structured logging.
// Takes pairs ([]externalAssetKey) which contains the pairs to resolve.
//
// Returns map[string]string which maps absolute directories to artefact
// prefixes.
func (bs *buildService) resolveAssetPairsToDirectories(ctx context.Context, l logger_domain.Logger, pairs []externalAssetKey) map[string]string {
	result := make(map[string]string, len(pairs))
	for _, p := range pairs {
		moduleDir, err := bs.resolver.GetModuleDir(ctx, p.moduleBase)
		if err != nil {
			l.Warn("Failed to resolve module directory for external asset",
				logger_domain.String("module_base", p.moduleBase),
				logger_domain.Error(err))
			continue
		}
		absDir := filepath.Join(moduleDir, p.assetPath)
		artefactPrefix := p.moduleBase + "/" + filepath.ToSlash(p.assetPath)
		result[absDir] = artefactPrefix
		l.Internal("Resolved external asset directory",
			logger_domain.String("module_base", p.moduleBase),
			logger_domain.String("asset_path", p.assetPath),
			logger_domain.String("resolved_dir", absDir))
	}
	return result
}

// walkAndSeedExternalAssetDir walks a single external asset directory and seeds
// all files into the registry with artefact IDs prefixed by artefactPrefix.
//
// Takes absDir (string) which is the absolute path to walk.
// Takes artefactPrefix (string) which provides the artefact ID prefix
// (e.g. "piko.sh/piko/lib/icons").
//
// Returns error when the directory walk itself fails fatally.
func (bs *buildService) walkAndSeedExternalAssetDir(
	ctx context.Context,
	absDir, artefactPrefix string,
) error {
	return bs.walkAndSeedDirectory(ctx, absDir, walkSeedParams{
		spanName:   "seedExternalAssetDir",
		labelKey:   "artefact_prefix",
		labelValue: artefactPrefix,
		walkFunction: func(absPath string, entry os.DirEntry, err error) error {
			return bs.seedExternalAssetEntry(ctx, absDir, artefactPrefix, absPath, entry, err)
		},
		warnMessage:   "Failed to walk external asset directory",
		statusMessage: "External asset directory seeded",
	})
}

// seedExternalAssetEntry processes a single directory entry during an external
// asset directory walk, seeding files into the registry.
//
// Takes absDir (string) which is the root directory being walked.
// Takes artefactPrefix (string) which provides the artefact ID prefix.
// Takes absPath (string) which is the absolute path of the current entry.
// Takes entry (os.DirEntry) which holds details about the directory entry.
// Takes walkError (error) which is any error encountered during the walk.
//
// Returns error which is always nil; errors are logged and the walk continues.
func (bs *buildService) seedExternalAssetEntry(
	ctx context.Context,
	absDir, artefactPrefix, absPath string,
	entry os.DirEntry,
	walkError error,
) error {
	if walkError != nil || entry.IsDir() {
		return nil
	}

	_, l := logger_domain.From(ctx, log)

	relPath, relErr := filepath.Rel(absDir, absPath)
	if relErr != nil {
		return nil
	}
	relPathSlash := filepath.ToSlash(relPath)
	artefactID := artefactPrefix + "/" + relPathSlash

	parentDir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	sandbox, sandboxErr := bs.createExternalSandbox("build-external-asset", parentDir)
	if sandboxErr != nil {
		l.Error("Failed to create sandbox for external asset file",
			logger_domain.String(fieldPath, absPath),
			logger_domain.Error(sandboxErr))
		return nil
	}

	file, openErr := sandbox.Open(fileName)
	if openErr != nil {
		_ = sandbox.Close()
		l.Error("Failed to read external asset file",
			logger_domain.String(fieldPath, absPath),
			logger_domain.Error(openErr))
		return nil
	}
	defer func() { _ = file.Close(); _ = sandbox.Close() }()

	profiles := lifecycle_domain.GetProfilesForFile(artefactID, nil)
	if _, upsertErr := bs.registryService.UpsertArtefact(ctx, artefactID, relPathSlash, file, localDiskCacheSource, profiles); upsertErr != nil {
		l.Error("Failed to seed external asset artefact",
			logger_domain.String("artefact_id", artefactID),
			logger_domain.Error(upsertErr))
	}

	return nil
}

// createExternalSandbox creates a read-only sandbox for an external module
// path using the externalSandboxFactory. Falls back to NoOpSandbox when no
// external factory is available.
//
// Takes purpose (string) which describes the sandbox's intended use.
// Takes path (string) which is the directory to sandbox.
//
// Returns safedisk.Sandbox which is the read-only sandbox scoped to the given
// path.
// Returns error when sandbox creation fails.
func (bs *buildService) createExternalSandbox(purpose, path string) (safedisk.Sandbox, error) {
	if bs.externalSandboxFactory != nil {
		return bs.externalSandboxFactory.Create(purpose, path, safedisk.ModeReadOnly)
	}
	return safedisk.NewNoOpSandbox(path, safedisk.ModeReadOnly)
}

// NewBuildService creates a new build service for running full builds.
//
// Takes configProvider (*config.Provider) which provides configuration settings.
// Takes pathsConfig (lifecycle_domain.LifecyclePathsConfig) which holds the
// resolved path settings for file system operations.
// Takes registry (registry_domain.RegistryService) which manages component
// registration.
// Takes orchestratorService (orchestrator_domain.OrchestratorService) which
// coordinates build operations.
// Takes bridge (lifecycle_domain.BridgeWithCounter) which handles lifecycle
// transitions.
// Takes eventBus (orchestrator_domain.EventBus) which dispatches build events.
// Takes renderer (render_domain.RenderService) which handles output rendering.
// Takes resolver (resolver_domain.ResolverPort) which resolves dependencies.
// Takes externalComponents ([]component_dto.ComponentDefinition) which holds
// component definitions needing module resolution.
// Takes sandboxFactory (safedisk.Factory) which creates sandboxes for
// filesystem access.
//
// Returns lifecycle_domain.BuilderAdapter which is the configured build service
// ready for use.
//
//nolint:revive // DI constructor
func NewBuildService(
	configProvider *config.Provider,
	pathsConfig lifecycle_domain.LifecyclePathsConfig,
	registry registry_domain.RegistryService,
	orchestratorService orchestrator_domain.OrchestratorService,
	bridge lifecycle_domain.BridgeWithCounter,
	eventBus orchestrator_domain.EventBus,
	renderer render_domain.RenderService,
	resolver resolver_domain.ResolverPort,
	externalComponents []component_dto.ComponentDefinition,
	sandboxFactory safedisk.Factory,
) lifecycle_domain.BuilderAdapter {
	return &buildService{
		configProvider:      configProvider,
		pathsConfig:         pathsConfig,
		registryService:     registry,
		orchestratorService: orchestratorService,
		bridge:              bridge,
		eventBus:            eventBus,
		renderer:            renderer,
		resolver:            resolver,
		externalComponents:  externalComponents,
		sandboxFactory:      sandboxFactory,
	}
}

// drainGCHints synchronously pops all queued GC hints and deletes the
// corresponding blobs. Called after the build dispatcher is idle so that
// blobs from replaced variants are cleaned up before the build exits.
func (bs *buildService) drainGCHints(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	totalDeleted := 0

	for {
		hints, err := bs.registryService.PopGCHints(ctx, gcHintDrainBatchSize)
		if err != nil {
			l.Warn("Failed to pop GC hints during build cleanup", logger_domain.Error(err))
			break
		}
		if len(hints) == 0 {
			break
		}

		for _, hint := range hints {
			store, storeErr := bs.registryService.GetBlobStore(hint.BackendID)
			if storeErr != nil {
				continue
			}
			if deleteErr := store.Delete(ctx, hint.StorageKey); deleteErr != nil {
				continue
			}
			totalDeleted++
		}
	}

	if totalDeleted > 0 {
		l.Internal("Cleaned up obsolete blobs after build",
			logger_domain.Int("deleted_count", totalDeleted))
	}
}
