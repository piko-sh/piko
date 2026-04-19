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

package coordinator_domain

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

// coordinatorOptions holds configuration for the coordinator service.
// Fields are ordered for optimal memory alignment.
type coordinatorOptions struct {
	// fileHashCache stores file hashes to avoid reading files that have not
	// changed.
	fileHashCache FileHashCachePort

	// codeEmitter outputs formatted code in dev-i mode; nil disables output.
	codeEmitter CodeEmitterPort

	// clientScriptEmitter transpiles and stores client-side scripts in
	// dev-i mode; nil disables per-component <script> emission.
	clientScriptEmitter ClientScriptEmitterPort

	// diagnosticOutput specifies where to write diagnostic messages; nil means
	// silent.
	diagnosticOutput DiagnosticOutputPort

	// clock provides time operations; nil defaults to RealClock.
	clock clock.Clock

	// baseDirSandbox provides file access within the project base directory.
	baseDirSandbox safedisk.Sandbox

	// sandboxFactory creates sandboxes with validated paths. When set and
	// baseDirSandbox is nil, the factory is used before falling back to
	// NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// debounceDuration is the wait time before processing changes; 0 means
	// immediate.
	debounceDuration time.Duration

	// maxBuildWaitDuration is the longest time a caller waits for a build result;
	// defaults to 30 seconds.
	maxBuildWaitDuration time.Duration

	// enableStaticHoisting controls whether static nodes are hoisted to
	// package-level variables in generated code. Defaults to true via
	// config.CompilerEnableStaticHoisting.
	enableStaticHoisting bool

	// enablePrerendering enables static HTML prerendering at generation
	// time.
	enablePrerendering bool

	// stripHTMLComments removes HTML comments from generated output.
	stripHTMLComments bool

	// enableDwarfLineDirectives enables valid DWARF //line directives
	// in generated code.
	enableDwarfLineDirectives bool
}

// CoordinatorOption is a functional option for configuring the coordinator
// service.
type CoordinatorOption func(*coordinatorOptions)

// coordinatorService implements the CoordinatorService interface. It is the
// stateful, long-lived service that manages the entire build lifecycle,
// including caching, state management, debouncing, and preventing concurrent
// builds (cache stampede protection).
type coordinatorService struct {
	// lastTriggerTime records when the last build was triggered; zero means no
	// build has happened yet.
	lastTriggerTime time.Time

	// buildGroup prevents duplicate build requests with the same input hash.
	buildGroup singleflight.Group

	// clock provides time functions for debounce timing.
	clock clock.Clock

	// resolver provides path resolution and base directory access.
	resolver resolver_domain.ResolverPort

	// fsReader reads source files from the file system.
	fsReader annotator_domain.FSReaderPort

	// annotator handles code annotation and runs introspection phases.
	annotator annotator_domain.AnnotatorPort

	// codeEmitter generates code for dev-i mode; nil in other modes.
	codeEmitter CodeEmitterPort

	// clientScriptEmitter transpiles <script lang="ts"> blocks to JS in
	// dev-i mode so the rendered page emits the per-component script
	// tags; nil disables emission and matches the non-dev-i default.
	clientScriptEmitter ClientScriptEmitterPort

	// diagnosticOutput sends diagnostics to the user; nil in LSP mode.
	diagnosticOutput DiagnosticOutputPort

	// cache stores complete annotation results for Tier 2 caching.
	cache BuildResultCachePort

	// introspectionCache stores tier 1 type introspection results.
	introspectionCache IntrospectionCachePort

	// fileHashCache stores file hashes to avoid reading unchanged files;
	// nil disables caching.
	fileHashCache FileHashCachePort

	// baseDirSandbox provides file access within the project base directory.
	baseDirSandbox safedisk.Sandbox

	// sandboxFactory creates sandboxes with validated paths. When set, the
	// factory is preferred over NewNoOpSandbox for fallback sandbox creation.
	sandboxFactory safedisk.Factory

	// baseDirSandboxPath is the path for which the cached sandbox was created.
	baseDirSandboxPath string

	// debounceTimer schedules the next rebuild; nil when no build is waiting.
	debounceTimer clock.Timer

	// lastBuildRequest stores the most recent build request for debounced
	// rebuilds.
	lastBuildRequest *coordinator_dto.BuildRequest

	// shutdown signals the coordinator to stop processing.
	shutdown chan struct{}

	// subscribers maps unique IDs to active subscribers.
	subscribers map[uint64]subscriber

	// rebuildTrigger receives build requests to handle in the build loop.
	rebuildTrigger chan *coordinator_dto.BuildRequest

	// status holds the current build state and most recent result.
	status buildStatus

	// waiters stores pending build requests, keyed by input hash.
	waiters sync.Map

	// wg tracks the build loop goroutine for graceful shutdown.
	wg sync.WaitGroup

	// invalidationEpoch is incremented by Invalidate(). Builds that started
	// before the current epoch must not write to the cache, as their results
	// are stale.
	invalidationEpoch atomic.Uint64

	// buildInFlight tracks whether a build is currently executing in the
	// build loop. Invalidate() waits for it to complete before returning.
	buildInFlight sync.WaitGroup

	// debounceDuration is the shortest time allowed between build triggers.
	debounceDuration time.Duration

	// maxBuildWaitDuration is the longest time a caller waits for a build result.
	maxBuildWaitDuration time.Duration

	// nextSubID is the next subscription ID to assign.
	nextSubID uint64

	// mu guards access to coordinator state fields.
	mu sync.RWMutex

	// subMutex guards access to the subscribers map.
	subMutex sync.RWMutex

	// debounceMutex guards access to the debounce timer.
	debounceMutex sync.Mutex

	// enableStaticHoisting controls whether static nodes are hoisted
	// to package-level variables in generated code.
	enableStaticHoisting bool

	// enablePrerendering enables static HTML prerendering at generation
	// time.
	enablePrerendering bool

	// stripHTMLComments removes HTML comments from generated output.
	stripHTMLComments bool

	// enableDwarfLineDirectives enables valid DWARF //line directives
	// in generated code.
	enableDwarfLineDirectives bool
}

// GetResult retrieves the latest successful build or triggers an initial build
// on cold start.
//
// Provides a non-blocking fast path for retrieving cached results.
// If no build is available, falls back to the blocking GetOrBuildProject.
//
// Actions are auto-discovered from the actions/ directory during annotation.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the code
// locations to analyse.
// Takes opts (...BuildOption) which configures the build behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the annotation
// data for the project.
// Returns error when the build fails or cannot be completed.
//
// Safe for concurrent use. Uses a mutex to protect the last build request
// state.
func (s *coordinatorService) GetResult(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.GetResult")
	defer span.End()

	buildOpts := &buildOptions{}
	for _, opt := range opts {
		opt(buildOpts)
	}

	s.mu.Lock()
	s.lastBuildRequest = &coordinator_dto.BuildRequest{
		CausationID:   buildOpts.CausationID,
		EntryPoints:   entryPoints,
		FaultTolerant: buildOpts.FaultTolerant,
		Resolver:      buildOpts.Resolver,
	}
	s.mu.Unlock()

	if result, ok := s.GetLastSuccessfulBuild(); ok {
		l.Trace("GetResult: Fast path successful, returning last successful build.")
		span.SetAttributes(attribute.String("result.source", "memory"))
		return result, nil
	}

	l.Trace("GetResult: No successful build available, falling back to GetOrBuildProject for initial build.")
	span.SetAttributes(attribute.String("result.source", "build"))
	return s.GetOrBuildProject(ctx, entryPoints, opts...)
}

// Shutdown performs a graceful shutdown of the coordinator service, ensuring
// the background build loop is terminated and any pending timers are stopped.
//
// Safe for concurrent use. Blocks until all background goroutines have exited.
func (s *coordinatorService) Shutdown(ctx context.Context) {
	shutdownCtx := context.WithoutCancel(ctx)
	shutdownCtx, sl := logger_domain.From(shutdownCtx, log)
	sl.Internal("Shutting down coordinator service...")
	close(s.shutdown)

	s.debounceMutex.Lock()
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}
	s.debounceMutex.Unlock()

	s.wg.Wait()

	if s.fileHashCache != nil {
		if err := s.fileHashCache.Persist(shutdownCtx); err != nil {
			sl.Error("Failed to persist file hash cache during shutdown.", logger_domain.Error(err))
		}
	}

	sl.Internal("Coordinator service shut down gracefully.")
}

// RequestRebuild starts a build without blocking the caller.
//
// Actions are found from the actions/ directory during annotation.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the code
// locations to start the build from.
// Takes opts (...BuildOption) which sets optional build behaviour.
//
// Safe for concurrent use. Uses debouncing to combine rapid requests into
// a single build. The build runs in a separate goroutine after the debounce
// period.
func (s *coordinatorService) RequestRebuild(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	opts ...BuildOption,
) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.RequestRebuild")
	defer span.End()
	l.Trace("Rebuild requested.")

	buildOpts := &buildOptions{}
	for _, opt := range opts {
		opt(buildOpts)
	}

	s.mu.Lock()
	s.lastBuildRequest = &coordinator_dto.BuildRequest{
		CausationID:   buildOpts.CausationID,
		EntryPoints:   entryPoints,
		Resolver:      buildOpts.Resolver,
		FaultTolerant: buildOpts.FaultTolerant,
	}
	s.status.State = stateBuilding
	requestToBuild := s.lastBuildRequest
	s.mu.Unlock()

	s.debounceMutex.Lock()
	defer s.debounceMutex.Unlock()

	now := s.clock.Now()
	if s.lastTriggerTime.IsZero() || now.Sub(s.lastTriggerTime) > s.debounceDuration {
		l.Trace("Debounce cooldown elapsed (or first trigger). Triggering immediate build.")
		s.lastTriggerTime = now
		s.triggerBuild(ctx, requestToBuild)
		return
	}

	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}

	l.Trace("Within debounce window. Scheduling trailing-edge rebuild.")
	s.debounceTimer = s.clock.AfterFunc(s.debounceDuration, func() {
		_, dl := logger_domain.From(ctx, log)
		dl.Trace("Trailing-edge debounce timer fired.")
		s.lastTriggerTime = s.clock.Now()
		s.triggerBuild(ctx, requestToBuild)
	})
}

// GetStatus returns the current build status.
//
// Returns buildStatus which shows the current state of the build.
//
// Safe for concurrent use.
func (s *coordinatorService) GetStatus() buildStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// GetLastSuccessfulBuild returns the most recent successfully completed build
// result.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the build
// result, or nil if no successful build exists.
// Returns bool which indicates whether a successful build result was found.
//
// Safe for concurrent use. Uses a read lock to protect access to the status.
func (s *coordinatorService) GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.status.Result != nil {
		return s.status.Result, true
	}
	return nil, false
}

// Invalidate clears the annotation cache (Tier 2) and in-memory state.
//
// This does not clear Tier 1 (introspection cache) because Tier 1 should only
// be cleared when script blocks or .go files change. The build executor finds
// these changes by comparing introspection hashes.
//
// Clearing Tier 1 on every file change would break the two-tier caching
// system, which allows fast template-only rebuilds.
//
// The hash-based cache checking works as follows:
//   - Uses Tier 1 fast path when only template, style, or i18n files changed
//     (5-10x faster).
//   - Starts full rebuild when script blocks changed (automatic Tier 1
//     clearing through hash mismatch).
//
// Returns error when the annotation cache cannot be cleared.
//
// Safe for concurrent use. Acquires a lock when clearing in-memory state.
func (s *coordinatorService) Invalidate(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.Invalidate")
	defer span.End()

	l.Internal("Invalidating Tier 2 (annotation cache) only. Tier 1 (introspection) preserved for fast-path rebuilds.")

	s.invalidationEpoch.Add(1)

	if err := s.cache.Clear(ctx); err != nil {
		l.ReportError(span, err, "Failed to clear annotation cache")
		cacheErrorCount.Add(ctx, 1)
		return fmt.Errorf("clearing annotation cache: %w", err)
	}
	l.Internal("Annotation cache (Tier 2) cleared.")

	s.mu.Lock()
	s.status.Result = nil
	s.mu.Unlock()

	s.drainRebuildTrigger()

	s.buildInFlight.Wait()

	l.Internal("In-memory status result cleared. Full invalidation complete.")
	return nil
}

// drainRebuildTrigger removes any pending rebuild requests from the channel
// so that stale triggers from a previous daemon do not interfere with the
// next build cycle.
func (s *coordinatorService) drainRebuildTrigger() {
	for {
		select {
		case <-s.rebuildTrigger:
		default:
			return
		}
	}
}

// initialisePostCreation handles post-creation initialisation: loading the file
// hash cache and creating the sandbox.
func (s *coordinatorService) initialisePostCreation(ctx context.Context) {
	ctx, il := logger_domain.From(ctx, log)
	if s.fileHashCache != nil {
		if err := s.fileHashCache.Load(ctx); err != nil {
			il.Warn("Failed to load file hash cache, performance may be degraded.", logger_domain.Error(err))
		}
	}

	if s.baseDirSandbox == nil {
		baseDir := s.resolver.GetBaseDir()
		if baseDir != "" {
			var sandbox safedisk.Sandbox
			var err error
			if s.sandboxFactory != nil {
				sandbox, err = s.sandboxFactory.Create("coordinator base dir", baseDir, safedisk.ModeReadOnly)
			} else {
				sandbox, err = safedisk.NewNoOpSandbox(baseDir, safedisk.ModeReadOnly)
			}
			if err != nil {
				il.Warn("Failed to create base directory sandbox, file operations may use direct OS calls.",
					logger_domain.Error(err),
					logger_domain.String("base_dir", baseDir))
			} else {
				s.baseDirSandbox = sandbox
				s.baseDirSandboxPath = baseDir
			}
		}
	} else {
		s.baseDirSandboxPath = s.resolver.GetBaseDir()
	}
}

// WithDebounceDuration sets the debounce duration for build requests.
//
// Takes d (time.Duration) which sets how long to wait before processing a
// build request. Values of zero or less are ignored.
//
// Returns CoordinatorOption which configures the coordinator's debounce
// behaviour.
func WithDebounceDuration(d time.Duration) CoordinatorOption {
	return func(o *coordinatorOptions) {
		if d > 0 {
			o.debounceDuration = d
		}
	}
}

// WithMaxBuildWaitDuration sets the maximum time a caller waits for a build
// result before timing out, defaulting to 30 seconds and worth increasing
// for integration tests under heavy system load.
//
// Takes d (time.Duration) which sets the maximum wait time. Values of zero or
// less are ignored.
//
// Returns CoordinatorOption which configures the coordinator's build wait
// timeout.
func WithMaxBuildWaitDuration(d time.Duration) CoordinatorOption {
	return func(o *coordinatorOptions) {
		if d > 0 {
			o.maxBuildWaitDuration = d
		}
	}
}

// WithFileHashCache sets the file hash cache for read optimisation.
// If not provided, all files will be read in full during hash calculation.
//
// Takes cache (FileHashCachePort) which provides the cache for file hash
// lookups.
//
// Returns CoordinatorOption which configures the coordinator with the cache.
func WithFileHashCache(cache FileHashCachePort) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.fileHashCache = cache
	}
}

// WithCodeEmitter sets the code emitter for dev-i mode. This is only needed
// when the coordinator must create fully-emitted artefacts for the interpreted
// runner.
//
// Takes emitter (CodeEmitterPort) which handles code emission for artefacts.
//
// Returns CoordinatorOption which configures the coordinator with the emitter.
func WithCodeEmitter(emitter CodeEmitterPort) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.codeEmitter = emitter
	}
}

// WithClientScriptEmitter sets the client script emitter for dev-i mode.
//
// The emitter transpiles each component's <script lang="ts"> block and
// registers the resulting artefact ID so the renderer can emit
// per-component script tags. When unset, interpreted pages skip
// client-side script emission and client-side behaviour will not run.
//
// Takes emitter (ClientScriptEmitterPort) which handles JS emission.
//
// Returns CoordinatorOption which configures the coordinator with the
// emitter.
func WithClientScriptEmitter(emitter ClientScriptEmitterPort) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.clientScriptEmitter = emitter
	}
}

// WithDiagnosticOutput sets how diagnostic messages are shown to users.
// Command-line tools use rich ANSI-formatted output, while LSP mode stays
// silent.
//
// Takes output (DiagnosticOutputPort) which specifies how to display messages.
//
// Returns CoordinatorOption which sets the coordinator's diagnostic output.
func WithDiagnosticOutput(output DiagnosticOutputPort) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.diagnosticOutput = output
	}
}

// WithBaseDirSandbox sets a custom sandbox for the project base directory.
// Use it with mock sandboxes for testing filesystem operations.
//
// If not provided, a real sandbox is created using safedisk.NewNoOpSandbox
// during service setup.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access within
// the project base directory.
//
// Returns CoordinatorOption which sets up the coordinator with the given
// sandbox.
func WithBaseDirSandbox(sandbox safedisk.Sandbox) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.baseDirSandbox = sandbox
	}
}

// WithSandboxFactory sets the sandbox factory used for fallback sandbox
// creation. When baseDirSandbox is nil, the factory is tried before falling
// back to NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns CoordinatorOption which configures the coordinator with the factory.
func WithSandboxFactory(factory safedisk.Factory) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.sandboxFactory = factory
	}
}

// WithStaticHoisting controls whether static nodes are hoisted to
// package-level variables in generated code.
//
// Takes enabled (bool) which enables or disables static hoisting.
//
// Returns CoordinatorOption which configures static hoisting.
func WithStaticHoisting(enabled bool) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.enableStaticHoisting = enabled
	}
}

// WithPrerendering controls whether static HTML is prerendered at
// generation time.
//
// Takes enabled (bool) which enables or disables prerendering.
//
// Returns CoordinatorOption which configures prerendering.
func WithPrerendering(enabled bool) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.enablePrerendering = enabled
	}
}

// WithStripHTMLComments controls whether HTML comments are omitted
// from generated output.
//
// Takes enabled (bool) which enables or disables comment stripping.
//
// Returns CoordinatorOption which configures comment stripping.
func WithStripHTMLComments(enabled bool) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.stripHTMLComments = enabled
	}
}

// WithDwarfLineDirectives controls whether the code generator emits
// valid DWARF //line directives.
//
// Takes enabled (bool) which enables or disables DWARF directives.
//
// Returns CoordinatorOption which configures DWARF line directives.
func WithDwarfLineDirectives(enabled bool) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.enableDwarfLineDirectives = enabled
	}
}

// NewService creates a new coordinator service. It requires its core
// dependencies as interfaces, adhering to the hexagonal architecture.
//
// Optional dependencies (file hash cache, code emitter, diagnostic output,
// context) should be provided via functional options: WithFileHashCache,
// WithCodeEmitter, WithDiagnosticOutput, WithContext.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation throughout the service lifetime.
// Takes annotator (AnnotatorPort) which provides code annotation capabilities.
// Takes cache (BuildResultCachePort) which stores build results.
// Takes introspectionCache (IntrospectionCachePort) which caches introspection
// data.
// Takes fsReader (FSReaderPort) which reads from the file system.
// Takes resolver (ResolverPort) which resolves package dependencies.
// Takes opts (CoordinatorOption) which configures optional behaviour.
//
// Returns CoordinatorService which is ready for use.
//
// Spawns a background goroutine that runs the build loop until the service
// is stopped.
func NewService(
	ctx context.Context,
	annotator annotator_domain.AnnotatorPort,
	cache BuildResultCachePort,
	introspectionCache IntrospectionCachePort,
	fsReader annotator_domain.FSReaderPort,
	resolver resolver_domain.ResolverPort,
	opts ...CoordinatorOption,
) CoordinatorService {
	options := applyCoordinatorOptions(opts...)
	service := newCoordinatorService(annotator, cache, introspectionCache, fsReader, resolver, options)
	service.initialisePostCreation(ctx)
	service.wg.Add(1)
	go service.buildLoop(ctx)
	return service
}

// withClock sets a custom clock for time operations. This is mainly used for
// testing to make debounce logic deterministic.
//
// Takes c (clock.Clock) which provides the clock implementation to use.
//
// Returns CoordinatorOption which configures the coordinator with the clock.
func withClock(c clock.Clock) CoordinatorOption {
	return func(o *coordinatorOptions) {
		o.clock = c
	}
}

// applyCoordinatorOptions applies the given functional options and returns the
// settings.
//
// Takes opts (...CoordinatorOption) which specifies the options to apply.
//
// Returns coordinatorOptions which holds the settings with defaults applied
// for any values not set.
func applyCoordinatorOptions(opts ...CoordinatorOption) coordinatorOptions {
	options := coordinatorOptions{
		fileHashCache:        nil,
		codeEmitter:          nil,
		clientScriptEmitter:  nil,
		diagnosticOutput:     nil,
		clock:                nil,
		baseDirSandbox:       nil,
		debounceDuration:     defaultDebounceDuration,
		maxBuildWaitDuration: defaultMaxBuildWaitDuration,
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.clock == nil {
		options.clock = clock.RealClock()
	}
	return options
}

// newCoordinatorService creates a new coordinator service with all fields
// initialised.
//
// Takes annotator (annotator_domain.AnnotatorPort) which provides code
// annotation capabilities.
// Takes cache (BuildResultCachePort) which stores build results.
// Takes introspectionCache (IntrospectionCachePort) which caches introspection
// data.
// Takes fsReader (annotator_domain.FSReaderPort) which reads from the file
// system.
// Takes resolver (resolver_domain.ResolverPort) which resolves package paths.
// Takes options (coordinatorOptions) which configures service behaviour.
//
// Returns *coordinatorService which is ready for use with all internal state
// initialised.
func newCoordinatorService(
	annotator annotator_domain.AnnotatorPort,
	cache BuildResultCachePort,
	introspectionCache IntrospectionCachePort,
	fsReader annotator_domain.FSReaderPort,
	resolver resolver_domain.ResolverPort,
	options coordinatorOptions,
) *coordinatorService {
	return &coordinatorService{
		lastTriggerTime:           time.Time{},
		buildGroup:                singleflight.Group{},
		clock:                     options.clock,
		resolver:                  resolver,
		fsReader:                  fsReader,
		annotator:                 annotator,
		codeEmitter:               options.codeEmitter,
		clientScriptEmitter:       options.clientScriptEmitter,
		diagnosticOutput:          options.diagnosticOutput,
		cache:                     cache,
		introspectionCache:        introspectionCache,
		fileHashCache:             options.fileHashCache,
		baseDirSandbox:            options.baseDirSandbox,
		sandboxFactory:            options.sandboxFactory,
		baseDirSandboxPath:        "",
		debounceTimer:             nil,
		lastBuildRequest:          nil,
		shutdown:                  make(chan struct{}),
		subscribers:               make(map[uint64]subscriber),
		rebuildTrigger:            make(chan *coordinator_dto.BuildRequest, 1),
		status:                    buildStatus{State: stateIdle, LastBuildTime: time.Time{}, LastBuildError: nil, Result: nil},
		waiters:                   sync.Map{},
		wg:                        sync.WaitGroup{},
		debounceDuration:          options.debounceDuration,
		maxBuildWaitDuration:      options.maxBuildWaitDuration,
		nextSubID:                 0,
		mu:                        sync.RWMutex{},
		subMutex:                  sync.RWMutex{},
		debounceMutex:             sync.Mutex{},
		enableStaticHoisting:      options.enableStaticHoisting,
		enablePrerendering:        options.enablePrerendering,
		stripHTMLComments:         options.stripHTMLComments,
		enableDwarfLineDirectives: options.enableDwarfLineDirectives,
	}
}
