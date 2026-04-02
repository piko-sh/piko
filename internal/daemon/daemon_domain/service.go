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

package daemon_domain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/seo/seo_adapters"
)

// DaemonServiceDeps contains all dependencies needed to create a DaemonService.
// This struct-based dependency injection pattern improves testability by making
// dependencies explicit and easier to mock.
//
// Build-triggered operations (asset manifest, interpreted mode, routing)
// are handled by the lifecycle service. The daemon focuses on HTTP serving,
// signal handling, and SEO artefact generation.
type DaemonServiceDeps struct {
	// OrchestratorService handles task scheduling and coordination.
	OrchestratorService orchestrator_domain.OrchestratorService

	// SignalNotifier is optional - if nil, defaults to osSignalNotifier.
	// Inject a mock to control shutdown timing in tests without real OS signals.
	SignalNotifier SignalNotifier

	// HealthServer is the server for health checks; nil disables health checks.
	HealthServer ServerAdapter

	// TLSRedirectServer is the plain HTTP server that redirects to HTTPS;
	// nil disables the redirect.
	TLSRedirectServer ServerAdapter

	// HealthRouter serves health check endpoint requests.
	HealthRouter http.Handler

	// CoordinatorService manages coordination tasks for the daemon.
	CoordinatorService coordinator_domain.CoordinatorService

	// DrainSignaller marks readiness probes as unhealthy during shutdown.
	// When nil, no drain signal is sent before shutting down servers.
	DrainSignaller DrainSignaller

	// Server is the HTTP server adapter for handling requests.
	Server ServerAdapter

	// FinalRouter is the fully configured HTTP handler for the daemon.
	FinalRouter http.Handler

	// SEOService provides search engine optimisation operations.
	SEOService SEOServicePort

	// OnServerBound is an optional callback invoked after the main HTTP server
	// successfully binds to a port. The callback receives the resolved listen
	// address (e.g. ":8081"). Used by the startup banner to display the actual
	// port when auto-next-port is enabled.
	OnServerBound func(address string)

	// WatchMode points to the build config's WatchMode flag, allowing RunProd
	// to disable file watching. Nil is safe and means no mutation occurs.
	WatchMode *bool

	// OnHealthBound is an optional callback invoked after the health server
	// successfully binds to a port. The callback receives the resolved listen
	// address (e.g. "127.0.0.1:9092").
	OnHealthBound func(address string)

	// DaemonConfig holds the resolved network and health probe values needed
	// by the daemon lifecycle. All fields are value types; pointer-to-value
	// conversion is performed in the bootstrap layer.
	DaemonConfig DaemonConfig
}

// daemonService is the core domain service for the Piko runtime daemon.
// It implements DaemonService and manages the application lifecycle, including
// starting and stopping the HTTP server and handling graceful shutdown.
//
// Build-triggered operations (asset manifest processing, interpreted mode
// handling, route reloading) are handled by the lifecycle service. The daemon
// only processes SEO artefacts from build notifications.
type daemonService struct {
	// seoCtx is the context for SEO operations; cancelled during shutdown.
	seoCtx context.Context

	// coordinatorService handles build coordination and notification subscriptions.
	coordinatorService coordinator_domain.CoordinatorService

	// healthRouter is the HTTP handler for health check endpoints.
	healthRouter http.Handler

	// healthServer is the HTTP server adapter for health check endpoints.
	healthServer ServerAdapter

	// orchestratorService manages task orchestration; nil until the daemon starts.
	orchestratorService orchestrator_domain.OrchestratorService

	// drainSignaller marks readiness probes as unhealthy during shutdown;
	// nil when health probes are disabled.
	drainSignaller DrainSignaller

	// signalNotifier handles OS signal listening for graceful shutdown.
	signalNotifier SignalNotifier

	// serverAdapter handles HTTP server operations for the main server.
	serverAdapter ServerAdapter

	// tlsRedirectServer handles HTTP-to-HTTPS redirect; nil when disabled.
	tlsRedirectServer ServerAdapter

	// seoService generates SEO output for projects; nil disables SEO processing.
	seoService SEOServicePort

	// finalRouter is the HTTP handler that processes requests after tracing is set up.
	finalRouter http.Handler

	// stopChan signals the service to begin shutdown when Stop is called.
	stopChan chan struct{}

	// seoCancel stops SEO work when the service shuts down.
	seoCancel context.CancelCauseFunc

	// watchMode points to the build config's WatchMode flag.
	watchMode *bool

	// daemonConfig holds resolved network and health probe values as plain types.
	daemonConfig DaemonConfig

	// seoWg tracks SEO generation goroutines that are still running.
	seoWg sync.WaitGroup

	// stopOnce guards single closure of the stop channel.
	stopOnce sync.Once
}

// fallbackSignalNotifier implements SignalNotifier using signal.NotifyContext.
// It is used when no notifier is provided, to avoid a circular dependency.
type fallbackSignalNotifier struct{}

// NotifyContext returns a context that is cancelled on SIGINT or SIGTERM.
//
// Returns context.Context which is cancelled when a signal is received.
// Returns context.CancelFunc which stops listening for signals when called.
func (*fallbackSignalNotifier) NotifyContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}

// RunDev starts the daemon in development mode.
// It launches the server, file watcher, and asset orchestrator at the same
// time.
//
// Returns error when setup fails or a fatal error occurs during operation.
func (ds *daemonService) RunDev(parentCtx context.Context) error {
	ctx, span, l := log.Span(parentCtx, "DaemonService.RunDev")
	defer span.End()

	l.Internal("Daemon service is starting up in Development Mode...")
	checkHostConfiguration(ctx)
	fileEventsProcessed.Add(ctx, 1)

	signalCtx, cancelSignal := ds.signalNotifier.NotifyContext(ctx)
	defer cancelSignal()
	l.Internal("Setting up signal handlers for graceful shutdown")

	serverErrChan := ds.launchDaemonProcess(signalCtx)
	unsubscribe := ds.subscribeToCoordinator(signalCtx)
	defer unsubscribe()

	return ds.awaitShutdownDev(signalCtx, span, serverErrChan)
}

// RunProd starts the daemon in production mode.
// It does not start the file watcher and assumes all artefacts are pre-built.
//
// Returns error when the daemon main process fails.
func (ds *daemonService) RunProd(parentCtx context.Context) error {
	ctx, span, l := log.Span(parentCtx, "DaemonService.RunProd")
	defer span.End()

	l.Internal("Daemon service is starting up in Production Mode...")

	checkHostConfiguration(ctx)

	fileEventsProcessed.Add(ctx, 1)

	signalCtx, cancel := ds.signalNotifier.NotifyContext(ctx)
	defer cancel()
	l.Internal("Setting up signal handlers for graceful shutdown")

	if ds.watchMode != nil {
		*ds.watchMode = false
	}

	l.Internal("Delegating to main daemon process")
	err := ds.runDaemonMain(signalCtx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.ReportError(span, err, "Daemon main process failed")
	}

	if err != nil {
		return fmt.Errorf("running daemon main process: %w", err)
	}
	return nil
}

// Stop shuts down the daemon service in an orderly way.
//
// Returns error when the shutdown process fails.
func (ds *daemonService) Stop(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "DaemonService.Stop")
	defer span.End()

	l.Internal("Stop requested, initiating graceful shutdown...")

	l.Internal("Closing stop channel to signal shutdown")
	ds.stopOnce.Do(func() {
		close(ds.stopChan)
	})

	l.Internal("Cancelling SEO artefact generation...")
	ds.seoCancel(errors.New("SEO generation stopped"))

	l.Internal("Waiting for SEO goroutines to complete...")
	ds.seoWg.Wait()

	l.Internal("Stopping orchestrator service")
	if ds.orchestratorService != nil {
		ds.orchestratorService.Stop()
	}

	l.Internal("Delegating to shutdown process")
	err := ds.shutdown(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Shutdown process failed")
		return fmt.Errorf("shutting down daemon services: %w", err)
	}
	span.SetStatus(codes.Ok, "Shutdown completed successfully")
	return nil
}

// GetHandler returns the HTTP handler for serving requests.
// Use it to test without starting the full server.
//
// Returns http.Handler which processes incoming HTTP requests.
func (ds *daemonService) GetHandler() http.Handler {
	return ds.finalRouter
}

// launchDaemonProcess starts the main daemon process in a
// background task, running the main loop and closing the error
// channel on exit.
//
// Returns chan error which receives any fatal errors from the
// daemon process and is closed when the process exits.
//
// Concurrent goroutine is spawned to run the main daemon loop
// in the background.
func (ds *daemonService) launchDaemonProcess(ctx context.Context) chan error {
	ctx, l := logger_domain.From(ctx, log)
	serverErrChan := make(chan error, 1)

	go func() {
		defer close(serverErrChan)
		defer goroutine.RecoverPanicToChannel(ctx, "daemon.launchDaemonProcess", serverErrChan)
		l.Internal("Launching main daemon process in background...")
		err := ds.runDaemonMain(ctx)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("Daemon main process failed", logger_domain.Error(err))
			serverErrChan <- err
		}
	}()

	l.Internal("HTTP server is starting...")
	return serverErrChan
}

// subscribeToCoordinator subscribes to build notifications if a coordinator
// is available.
//
// Returns func() which unsubscribes from notifications. Returns a no-op
// function when no coordinator is set up.
//
// Spawns a goroutine to handle incoming build notifications. The goroutine
// runs until the context is cancelled.
func (ds *daemonService) subscribeToCoordinator(ctx context.Context) func() {
	if ds.coordinatorService == nil {
		return func() {}
	}

	buildNotifications, unsubscribe := ds.coordinatorService.Subscribe("daemon-watcher")
	go ds.handleBuildNotifications(ctx, buildNotifications)
	return unsubscribe
}

// awaitShutdownDev waits for a shutdown signal or server error in dev mode.
//
// Takes span (trace.Span) which records the shutdown status.
// Takes serverErrChan (chan error) which receives server errors.
//
// Returns error when the server fails or when shutdown fails.
func (ds *daemonService) awaitShutdownDev(
	signalCtx context.Context,
	span trace.Span,
	serverErrChan chan error,
) error {
	signalCtx, l := logger_domain.From(signalCtx, log)
	select {
	case <-signalCtx.Done():
		cause := context.Cause(signalCtx)
		causeMessage := "unknown"
		if cause != nil {
			causeMessage = cause.Error()
		}
		l.Notice("Shutdown signal received, initiating graceful shutdown.",
			logger_domain.String("cause", causeMessage),
		)
		span.SetStatus(codes.Ok, "Shutdown signal received")

	case err := <-serverErrChan:

		if err == nil && signalCtx.Err() != nil {
			l.Notice("Shutdown signal received, initiating graceful shutdown.",
				logger_domain.String("cause", signalCtx.Err().Error()),
			)
			span.SetStatus(codes.Ok, "Shutdown signal received")
			break
		}
		if err == nil {
			err = errors.New("server exited without error")
		}
		l.Error("HTTP server stopped unexpectedly", logger_domain.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP server failed")
		return fmt.Errorf("HTTP server stopped unexpectedly: %w", err)
	}

	l.Internal("Starting graceful shutdown of services...")
	shutdownCtx, cancelShutdown := context.WithTimeoutCause(context.WithoutCancel(signalCtx), 10*time.Second,
		errors.New("graceful shutdown exceeded 10s timeout"))
	defer cancelShutdown()

	if err := ds.shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down daemon services: %w", err)
	}
	return nil
}

// handleBuildNotifications listens for build notifications and processes them.
//
// Takes notifications (<-chan coordinator_domain.BuildNotification) which
// yields build events from the coordinator.
//
// Blocks until the context is cancelled or the channel is closed.
func (ds *daemonService) handleBuildNotifications(ctx context.Context, notifications <-chan coordinator_domain.BuildNotification) {
	defer goroutine.RecoverPanic(ctx, "daemon.handleBuildNotifications")
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting to listen for build notifications from coordinator...")

	for {
		select {
		case <-ctx.Done():
			l.Internal("Stopping build notification listener.")
			return
		case notification, ok := <-notifications:
			if !ok {
				l.Warn("Build notification channel was closed.")
				return
			}
			ds.processNotification(ctx, &notification)
		}
	}
}

// processNotification handles a single build notification.
//
// The daemon only processes SEO artefacts from build notifications. Other
// updates triggered by builds (such as asset manifest, interpreted mode, and
// route reloading) are handled by the lifecycle service.
//
// Takes notification (*coordinator_domain.BuildNotification) which contains
// the build result to process.
func (ds *daemonService) processNotification(ctx context.Context, notification *coordinator_domain.BuildNotification) {
	_, l := logger_domain.From(ctx, log)
	l.Trace("Received new build result", logger_domain.String("causationID", notification.CausationID))

	if notification.Result == nil {
		l.Warn("Build notification contained no result")
		return
	}

	ds.processSEOArtefacts(notification.Result)
}

// processSEOArtefacts regenerates SEO artefacts in the background.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the
// annotation data to turn into SEO artefacts.
func (ds *daemonService) processSEOArtefacts(result *annotator_dto.ProjectAnnotationResult) {
	if ds.seoService == nil {
		return
	}

	ctx, l := logger_domain.From(ds.seoCtx, log)
	l.Trace("Regenerating SEO artefacts after build...")

	ds.seoWg.Go(func() {
		translator := seo_adapters.NewProjectViewTranslator()
		projectView := translator.Translate(result)
		if err := ds.seoService.GenerateArtefacts(ctx, projectView); err != nil {
			if !errors.Is(err, context.Canceled) {
				l.Error("Failed to regenerate SEO artefacts", logger_domain.Error(err))
			}
			return
		}
		l.Trace("SEO artefacts regenerated successfully.")
	})
}

// NewService creates a configured daemon service using dependency injection.
// All dependencies are passed through the deps struct, which makes it easy
// to test by allowing dependencies to be replaced.
//
// Takes deps (*DaemonServiceDeps) which provides all service dependencies.
//
// Returns DaemonService which is the configured daemon service ready for use.
func NewService(ctx context.Context, deps *DaemonServiceDeps) DaemonService {
	signalNotifier := deps.SignalNotifier
	if signalNotifier == nil {
		signalNotifier = &fallbackSignalNotifier{}
	}

	if deps.OnServerBound != nil && deps.Server != nil {
		deps.Server.SetOnBound(deps.OnServerBound)
	}
	if deps.OnHealthBound != nil && deps.HealthServer != nil {
		deps.HealthServer.SetOnBound(deps.OnHealthBound)
	}

	seoCtx, seoCancel := context.WithCancelCause(ctx)

	return &daemonService{
		daemonConfig:        deps.DaemonConfig,
		watchMode:           deps.WatchMode,
		coordinatorService:  deps.CoordinatorService,
		drainSignaller:      deps.DrainSignaller,
		finalRouter:         deps.FinalRouter,
		healthServer:        deps.HealthServer,
		healthRouter:        deps.HealthRouter,
		orchestratorService: deps.OrchestratorService,
		serverAdapter:       deps.Server,
		tlsRedirectServer:   deps.TLSRedirectServer,
		signalNotifier:      signalNotifier,
		seoService:          deps.SEOService,
		stopChan:            make(chan struct{}),
		stopOnce:            sync.Once{},
		seoWg:               sync.WaitGroup{},
		seoCtx:              seoCtx,
		seoCancel:           seoCancel,
	}
}
