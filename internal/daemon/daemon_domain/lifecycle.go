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
	"net"
	"net/http"
	"strconv"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/netutil"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// mapCarrierPoolSize is the starting capacity for pooled MapCarrier instances.
	mapCarrierPoolSize = 16

	// maxPortRetries is the maximum number of ports to try when auto-selecting.
	maxPortRetries = 100

	// keyPort is the logging key for port numbers.
	keyPort = "port"

	// keyAddress is the logging key for server bind addresses.
	keyAddress = "address"

	// keyBindAddress is the logging key for network bind addresses.
	keyBindAddress = "bind_address"
)

// serverKind identifies the type of server for differentiated logging and telemetry.
type serverKind uint8

const (
	// serverKindMain is the main HTTP server that handles client requests.
	serverKindMain serverKind = iota

	// serverKindHealth identifies the health check server in logging and errors.
	serverKindHealth

	// serverKindTLSRedirect identifies the HTTP-to-HTTPS redirect server.
	serverKindTLSRedirect
)

var (
	// serverKindLabels maps each server kind to its display text for logs and errors.
	serverKindLabels = [3]struct {
		name         string
		shutdownOK   string
		shutdownFail string
		portInUse    string
		eventName    string
	}{
		serverKindMain:   {"HTTP server", "Server started and shut down gracefully", "Unexpected server error", "Port in use", "PortInUse"},
		serverKindHealth: {"Health check server", "Health server started and shut down gracefully", "Unexpected health server error", "Health server port in use", "HealthPortInUse"},
		serverKindTLSRedirect: {
			"TLS redirect server", "TLS redirect server started and shut down gracefully",
			"Unexpected TLS redirect server error", "TLS redirect server port in use", "TLSRedirectPortInUse",
		},
	}

	// mapCarrierPool provides reusable MapCarrier instances to avoid per-request map
	// allocation during trace context extraction.
	mapCarrierPool = sync.Pool{
		New: func() any {
			return make(propagation.MapCarrier, mapCarrierPoolSize)
		},
	}

	// errContinueRetry is a sentinel error indicating the port loop should continue.
	errContinueRetry = errors.New("continue retry")
)

// runDaemonMain is the main loop for the daemon. It starts the HTTP servers and waits for
// a shutdown signal.
//
// Returns error when the server fails or shutdown stops early.
func (ds *daemonService) runDaemonMain(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "runDaemonMain")
	defer span.End()

	l.Internal("Daemon main process starting...")

	servers := ds.startHTTPServers(ctx)

	return ds.waitForShutdown(ctx, span, servers)
}

var (
	// errServersStopped is the cause given when the servers were released without any
	// failure.
	errServersStopped = errors.New("servers stopped")
)

// runningServers reports on the servers a daemon launched.
type runningServers struct {
	// failed carries the first fatal server failure.
	failed chan error

	// stopped closes once every server goroutine has returned.
	stopped chan struct{}
}

// startHTTPServers launches the main and health servers.
//
// Returns runningServers which reports failure and completion.
//
// Concurrency: starts one goroutine per server, tracked by a WaitGroup; a further
// goroutine closes the stopped channel once they have all returned. That last one is
// deliberately not tracked by the WaitGroup it waits on, or it would be waiting for
// itself.
func (ds *daemonService) startHTTPServers(ctx context.Context) runningServers {
	servers := runningServers{
		failed:  make(chan error, 1),
		stopped: make(chan struct{}),
	}

	serveCtx, abortServers := context.WithCancelCause(context.WithoutCancel(ctx))

	var running sync.WaitGroup

	healthSettled := make(chan struct{})

	if ds.shouldStartHealthServer() {
		ds.launchAuxiliary(ctx, &running, "daemon.startHealthServer", func() error {
			return ds.startHealthServer(serveCtx)
		}, healthSettled)
	} else {
		close(healthSettled)
	}

	if ds.shouldStartTLSRedirect() {
		ds.launchAuxiliary(ctx, &running, "daemon.startTLSRedirectServer", func() error {
			return ds.startTLSRedirectServer(serveCtx)
		}, nil)
	}

	ds.launch(ctx, &running, servers, abortServers, "daemon.startMainServer", func() error {
		ds.waitForHealthServer(serveCtx, healthSettled)

		return ds.startMainServer(serveCtx)
	})

	go func() {
		defer goroutine.RecoverPanic(ctx, "daemon.serversStoppedWatcher")

		running.Wait()
		abortServers(errServersStopped)
		close(servers.stopped)
	}()

	return servers
}

// launch runs a server whose failure brings the daemon down.
//
// Takes running (*sync.WaitGroup) which tracks the server goroutine.
// Takes servers (runningServers) which receives the failure.
// Takes abort (context.CancelCauseFunc) which stops the surviving servers.
// Takes component (string) which names the server in logs and panic reports.
// Takes run (func() error) which runs the server.
func (ds *daemonService) launch(
	ctx context.Context,
	running *sync.WaitGroup,
	servers runningServers,
	abort context.CancelCauseFunc,
	component string,
	run func() error,
) {
	running.Go(func() {
		if err := goroutine.SafeCall(ctx, component, run); err != nil {
			ds.reportServerFailure(ctx, servers.failed, abort, err)
		}
	})
}

// launchAuxiliary runs a server whose failure the daemon survives.
//
// Takes running (*sync.WaitGroup) which tracks the server goroutine.
// Takes component (string) which names the server in logs and panic reports.
// Takes run (func() error) which runs the server.
// Takes settled (chan struct{}) which is closed when the server exits, and may be nil.
func (*daemonService) launchAuxiliary(
	ctx context.Context,
	running *sync.WaitGroup,
	component string,
	run func() error,
	settled chan struct{},
) {
	running.Go(func() {
		if settled != nil {
			defer close(settled)
		}

		if err := goroutine.SafeCall(ctx, component, run); err != nil {
			_, l := logger_domain.From(ctx, log)
			l.Error("Auxiliary server stopped; the application continues to serve without it",
				logger_domain.String(logger_domain.FieldStrComponent, component),
				logger_domain.Error(err),
			)
		}
	})
}

// reportServerFailure records the first fatal server failure and stops the rest.
//
// Takes failed (chan error) which receives the error, and keeps only the first.
// Takes abort (context.CancelCauseFunc) which stops the surviving servers.
// Takes err (error) which is the failure.
func (ds *daemonService) reportServerFailure(
	ctx context.Context,
	failed chan error,
	abort context.CancelCauseFunc,
	err error,
) {
	abort(err)
	ds.signalDrain(ctx)

	select {
	case failed <- err:
	default:
	}
}

// waitForHealthServer holds the main server until the health probe has bound or given up.
//
// Takes settled (chan struct{}) which closes when the health server exits.
func (ds *daemonService) waitForHealthServer(ctx context.Context, settled <-chan struct{}) {
	if !ds.shouldStartHealthServer() || ds.healthServer == nil {
		return
	}

	select {
	case <-ds.healthServer.Bound():
	case <-settled:
	case <-ctx.Done():
	}
}

// startMainServer sets up and starts the main HTTP server.
//
// HTTP/2 is negotiated by the server adapter: over ALPN when TLS is enabled, and over
// cleartext prior-knowledge h2c when it is not.
//
// Returns error when the server fails to start.
func (ds *daemonService) startMainServer(ctx context.Context) error {
	return ds.startServer(ctx, ds.createTracingHandler())
}

// createTracingHandler wraps the final router to extract distributed trace context from
// incoming request headers and initialise the per-request PikoRequestCtx carrier. The
// carrier is acquired from a pool here and released after ServeHTTP returns, so every
// downstream handler can mutate it via the context pointer without additional
// context.WithValue calls.
//
// Returns http.Handler which extracts trace context from request headers, allowing trace
// propagation from upstream services.
func (ds *daemonService) createTracingHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pctx := daemon_dto.AcquirePikoRequestCtx()
		pctx.OtelExtracted = true
		pctx.DevelopmentMode = ds.daemonConfig.DevelopmentMode

		reqCtx := extractTraceContext(r)
		reqCtx = daemon_dto.WithPikoRequestCtx(reqCtx, pctx)

		ds.finalRouter.ServeHTTP(w, r.WithContext(reqCtx))

		daemon_dto.ReleasePikoRequestCtx(pctx)
	})
}

// shouldStartHealthServer checks if the health server should be started.
//
// Returns bool which is true when the health server and router are configured and health
// probes are enabled.
func (ds *daemonService) shouldStartHealthServer() bool {
	return ds.healthServer != nil &&
		ds.healthRouter != nil &&
		ds.daemonConfig.HealthEnabled
}

// waitForShutdown blocks until a shutdown signal or error is received.
//
// Takes span (trace.Span) which records the shutdown status.
// Takes servers (runningServers) which carries the failure and stop signals.
//
// Returns error when a server reported a fatal failure.
func (ds *daemonService) waitForShutdown(
	ctx context.Context,
	span trace.Span,
	servers runningServers,
) error {
	ctx, l := logger_domain.From(ctx, log)
	select {
	case <-ctx.Done():
		l.Internal("Shutdown signal received (context cancelled or OS signal).")
		span.SetStatus(codes.Ok, "Shutdown signal received")

		return nil

	case err := <-servers.failed:
		return ds.handleServerError(ctx, span, err)

	case <-servers.stopped:
		return ds.handleServersStopped(ctx, span, servers)

	case <-ds.stopChan:
		l.Internal("Stop() called, initiating shutdown.")
		span.SetStatus(codes.Ok, "Stop called")

		return nil
	}
}

// handleServersStopped decides what it means for every server to have returned.
//
// Takes span (trace.Span) which records the outcome.
// Takes servers (runningServers) which may still hold a failure.
//
// Returns error when a server failed, or nil for a clean stop.
func (ds *daemonService) handleServersStopped(
	ctx context.Context,
	span trace.Span,
	servers runningServers,
) error {
	select {
	case err := <-servers.failed:
		return ds.handleServerError(ctx, span, err)
	default:
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("All servers stopped.")
	span.SetStatus(codes.Ok, "Servers stopped")

	return nil
}

// handleServerError processes errors from the HTTP server.
//
// Takes span (trace.Span) which records the error for tracing.
// Takes err (error) which is the error to handle.
//
// Returns error when the error is not nil and is not a normal server shutdown
// (http.ErrServerClosed).
func (*daemonService) handleServerError(ctx context.Context, span trace.Span, err error) error {
	ctx, l := logger_domain.From(ctx, log)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	l.Error("HTTP server stopped unexpectedly", logger_domain.Error(err))
	span.RecordError(err)
	span.SetStatus(codes.Error, "HTTP server stopped unexpectedly")
	return fmt.Errorf("HTTP server stopped unexpectedly: %w", err)
}

// startServer starts the HTTP server on the configured port. If the port is in use and
// AutoNextPort is enabled, it tries the next available port.
//
// Takes handler (http.Handler) which handles incoming HTTP requests.
//
// Returns error when the port setting is invalid or the server cannot start.
func (ds *daemonService) startServer(ctx context.Context, handler http.Handler) error {
	ctx, span, l := log.Span(ctx, "startServer")
	defer span.End()

	initialPort, err := strconv.Atoi(ds.daemonConfig.NetworkPort)
	if err != nil {
		l.Error("Invalid initial port in configuration", logger_domain.Error(err), logger_domain.String(keyPort, ds.daemonConfig.NetworkPort))
		return fmt.Errorf("invalid port in config: %w", err)
	}

	autoNextPort := ds.daemonConfig.NetworkAutoNextPort
	span.SetAttributes(attribute.Bool("autoNextPort", autoNextPort))
	ds.logServerStartupConfig(ctx, initialPort, autoNextPort)

	return ds.tryStartOnPort(ctx, span, handler, initialPort, autoNextPort)
}

// logServerStartupConfig logs the server startup settings.
//
// Takes initialPort (int) which is the port to bind to first.
// Takes autoNextPort (bool) which enables trying the next port if the first is in use.
func (*daemonService) logServerStartupConfig(ctx context.Context, initialPort int, autoNextPort bool) {
	ctx, l := logger_domain.From(ctx, log)
	if autoNextPort {
		l.Internal("Starting HTTP server with auto-port selection",
			logger_domain.Int("initial_port", initialPort),
			logger_domain.Int("max_retries", maxPortRetries))
	} else {
		l.Internal("Starting HTTP server on fixed port", logger_domain.Int(keyPort, initialPort))
	}
}

// tryStartOnPort attempts to start the server on the specified port, retrying on
// different ports if configured.
//
// Takes span (trace.Span) which records tracing information.
// Takes handler (http.Handler) which serves incoming HTTP requests.
// Takes initialPort (int) which specifies the first port to try.
// Takes autoNextPort (bool) which enables trying subsequent ports on failure.
//
// Returns error when no available port is found after all retries.
func (ds *daemonService) tryStartOnPort(ctx context.Context, span trace.Span, handler http.Handler, initialPort int, autoNextPort bool) error {
	ctx, l := logger_domain.From(ctx, log)
	for i := range maxPortRetries {
		currentPort := initialPort + i
		addr := net.JoinHostPort("", strconv.Itoa(currentPort))

		if i > 0 {
			l.Notice("HTTP server using alternative port",
				logger_domain.Int(keyPort, currentPort),
				logger_domain.String(keyAddress, addr))
		} else {
			l.Internal("Binding HTTP server to port",
				logger_domain.Int(keyPort, currentPort),
				logger_domain.String(keyAddress, addr))
		}

		listenErr := ds.serverAdapter.ListenAndServe(ctx, addr, handler)
		result := ds.handleServerListenResult(ctx, span, listenErr, addr, autoNextPort, i, serverKindMain)

		if result == nil || !errors.Is(result, errContinueRetry) {
			return result
		}
	}

	return ds.reportNoPortAvailable(ctx, span, initialPort)
}

// handleServerListenResult checks the result of a server listen attempt and takes the
// correct action. It uses serverKind to set the right log messages and span attributes.
//
// Takes span (trace.Span) which records the operation status and errors.
// Takes listenErr (error) which is the error from the listen attempt.
// Takes addr (string) which is the address the server tried to listen on.
// Takes autoNextPort (bool) which enables trying the next port on conflict.
// Takes attempt (int) which tracks the current retry attempt number.
// Takes kind (serverKind) which identifies the server type for logging.
//
// Returns error when the listen attempt failed with an unexpected error.
func (ds *daemonService) handleServerListenResult(ctx context.Context, span trace.Span, listenErr error, addr string, autoNextPort bool, attempt int, kind serverKind) error {
	ctx, l := logger_domain.From(ctx, log)
	labels := serverKindLabels[kind]

	if listenErr == nil || errors.Is(listenErr, http.ErrServerClosed) {
		l.Internal(labels.name+" shut down gracefully.", logger_domain.String(keyAddress, addr))
		span.SetStatus(codes.Ok, labels.shutdownOK)
		return nil
	}

	if netutil.IsPortInUseError(listenErr) {
		return ds.handleServerPortInUse(ctx, span, listenErr, addr, autoNextPort, attempt, kind)
	}

	l.Error("Failed to start "+labels.name+" with an unexpected error",
		logger_domain.String(keyAddress, addr),
		logger_domain.Error(listenErr))
	span.RecordError(listenErr)
	span.SetStatus(codes.Error, labels.shutdownFail)
	return fmt.Errorf("starting %s on %s: %w", labels.name, addr, listenErr)
}

// handleServerPortInUse handles the case when a server port is already in use.
//
// Takes span (trace.Span) which records tracing events and errors.
// Takes listenErr (error) which is the original port binding error.
// Takes addr (string) which is the address that failed to bind.
// Takes autoNextPort (bool) which indicates whether to try the next port.
// Takes attempt (int) which is the current retry attempt number.
// Takes kind (serverKind) which indicates the type of server.
//
// Returns error when autoNextPort is false, returning the original error.
// Returns errContinueRetry when autoNextPort is true to signal a retry.
func (*daemonService) handleServerPortInUse(ctx context.Context, span trace.Span, listenErr error, addr string, autoNextPort bool, attempt int, kind serverKind) error {
	ctx, l := logger_domain.From(ctx, log)
	labels := serverKindLabels[kind]

	if !autoNextPort {
		l.Error(labels.name+" port is already in use and AutoNextPort is disabled.",
			logger_domain.String(keyAddress, addr),
			logger_domain.Error(listenErr))
		span.RecordError(listenErr)
		span.SetStatus(codes.Error, labels.portInUse)
		return fmt.Errorf("%s port %s already in use: %w", labels.name, addr, listenErr)
	}

	l.Warn(labels.name+" port in use, trying next available port...",
		logger_domain.String(keyAddress, addr),
		logger_domain.Int("attempt", attempt+1))
	span.AddEvent(labels.eventName, trace.WithAttributes(attribute.String(keyAddress, addr)))
	return errContinueRetry
}

// reportNoPortAvailable reports that no free port could be found.
//
// Takes span (trace.Span) which records the error for tracing.
// Takes initialPort (int) which is the first port that was tried.
//
// Returns error when no port is free after all retries.
func (*daemonService) reportNoPortAvailable(ctx context.Context, span trace.Span, initialPort int) error {
	ctx, l := logger_domain.From(ctx, log)
	finalErr := fmt.Errorf("could not find an available port after %d retries, starting from %d", maxPortRetries, initialPort)
	l.Error(finalErr.Error())
	span.RecordError(finalErr)
	span.SetStatus(codes.Error, "No available port found")
	return finalErr
}

// startHealthServer starts the health check HTTP server on the configured port and bind
// address. If AutoNextPort is enabled, it tries the next ports when the configured port
// is already in use.
//
// Returns error when the port setting is invalid or the server cannot start.
func (ds *daemonService) startHealthServer(ctx context.Context) error {
	ctx, span, _ := log.Span(ctx, "startHealthServer")
	defer span.End()

	config := ds.daemonConfig

	initialPort, err := strconv.Atoi(config.HealthPort)
	if err != nil {
		return fmt.Errorf("invalid health probe port %q in config: %w", config.HealthPort, err)
	}

	bindAddress := normaliseBindAddress(config.HealthBindAddress)
	if err := validateBindAddress(bindAddress, "WithHealthProbePort"); err != nil {
		return fmt.Errorf("invalid health probe bind address %q: %w", config.HealthBindAddress, err)
	}

	ds.logHealthServerStartupConfig(ctx, bindAddress, initialPort, config.HealthAutoNextPort)

	return ds.tryStartHealthServerOnPort(ctx, span, bindAddress, initialPort, config.HealthAutoNextPort, config.HealthLivePath, config.HealthReadyPath)
}

// logHealthServerStartupConfig writes the health server settings to the log.
//
// Takes bindAddr (string) which specifies the network address to bind to.
// Takes initialPort (int) which specifies the starting port number.
// Takes autoNextPort (bool) which enables automatic port selection if the first port is
// already in use.
func (*daemonService) logHealthServerStartupConfig(ctx context.Context, bindAddr string, initialPort int, autoNextPort bool) {
	ctx, l := logger_domain.From(ctx, log)
	if autoNextPort {
		l.Internal("Starting health check server with auto-port selection",
			logger_domain.String(keyBindAddress, bindAddr),
			logger_domain.Int("initial_port", initialPort),
			logger_domain.Int("max_retries", maxPortRetries))
	} else {
		l.Internal("Starting health check server on fixed port",
			logger_domain.String(keyBindAddress, bindAddr),
			logger_domain.Int(keyPort, initialPort))
	}
}

// tryStartHealthServerOnPort tries to start the health server on the given port,
// advancing to the next port if needed.
//
// Takes span (trace.Span) which records tracing data.
// Takes bindAddr (string) which is the address to bind to.
// Takes initialPort (int) which is the first port to try.
// Takes autoNextPort (bool) which allows trying the next port on failure.
// Takes livePath (string) which is the path for liveness checks.
// Takes readyPath (string) which is the path for readiness checks.
//
// Returns error when no port can be used or the server fails to start.
func (ds *daemonService) tryStartHealthServerOnPort(ctx context.Context, span trace.Span, bindAddr string, initialPort int, autoNextPort bool, livePath, readyPath string) error {
	ctx, l := logger_domain.From(ctx, log)
	for i := range maxPortRetries {
		currentPort := initialPort + i
		addr := net.JoinHostPort(bindAddr, strconv.Itoa(currentPort))

		if i > 0 {
			l.Notice("Health check server using alternative port",
				logger_domain.Int(keyPort, currentPort),
				logger_domain.String(keyAddress, addr),
				logger_domain.String("live_path", livePath),
				logger_domain.String("ready_path", readyPath))
		} else {
			l.Internal("Binding health check server to port",
				logger_domain.Int(keyPort, currentPort),
				logger_domain.String(keyAddress, addr),
				logger_domain.String("live_path", livePath),
				logger_domain.String("ready_path", readyPath))
		}

		listenErr := ds.healthServer.ListenAndServe(ctx, addr, ds.healthRouter)
		result := ds.handleServerListenResult(ctx, span, listenErr, addr, autoNextPort, i, serverKindHealth)

		if result == nil || !errors.Is(result, errContinueRetry) {
			return result
		}
	}

	return ds.reportNoHealthPortAvailable(ctx, span, initialPort)
}

// reportNoHealthPortAvailable reports that no port is available for the health server
// after trying all allowed ports.
//
// Takes span (trace.Span) which records the error for tracing.
// Takes initialPort (int) which is the first port that was tried.
//
// Returns error when no port could be found after the maximum number of tries.
func (*daemonService) reportNoHealthPortAvailable(ctx context.Context, span trace.Span, initialPort int) error {
	ctx, l := logger_domain.From(ctx, log)
	finalErr := fmt.Errorf("could not find an available port for health server after %d retries, starting from %d", maxPortRetries, initialPort)
	l.Error(finalErr.Error())
	span.RecordError(finalErr)
	span.SetStatus(codes.Error, "No available health server port found")
	return finalErr
}

// shutdown stops the daemon servers in the correct order for graceful rolling deploys.
//
// Returns error when the main or health server fails to shut down cleanly.
func (ds *daemonService) shutdown(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "shutdown")
	defer span.End()

	l.Internal("Daemon shutting down servers.")

	ds.signalDrain(ctx)
	ds.waitDrainDelay(ctx, span)

	mainErr := ds.shutdownMainServer(ctx, span)
	healthErr := ds.shutdownHealthServer(ctx, span)
	ds.shutdownTLSRedirectServer(ctx, span)

	if mainErr != nil {
		span.SetStatus(codes.Error, "Error shutting down main server")
		return fmt.Errorf("shutting down main server: %w", mainErr)
	}
	if healthErr != nil {
		span.SetStatus(codes.Error, "Error shutting down health server")
		return fmt.Errorf("shutting down health server: %w", healthErr)
	}

	l.Internal("Daemon runner services shut down cleanly.")
	span.SetStatus(codes.Ok, "Daemon runner services shut down cleanly")
	return nil
}

// signalDrain marks the health probe service as draining so that readiness checks return
// unhealthy before any server is stopped.
func (ds *daemonService) signalDrain(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	if ds.drainSignaller != nil {
		ds.drainSignaller.SignalDrain()
		l.Internal("Drain signal sent, readiness probes now return unhealthy.")
	}
}

// waitDrainDelay pauses for the configured drain delay, giving load balancers time to
// deregister the instance.
//
// Takes span (trace.Span) which records drain delay tracing events.
func (ds *daemonService) waitDrainDelay(ctx context.Context, span trace.Span) {
	ctx, l := logger_domain.From(ctx, log)
	delay := ds.daemonConfig.ShutdownDrainDelay
	if delay <= 0 {
		return
	}

	l.Internal("Waiting for shutdown drain delay...",
		logger_domain.Duration("delay", delay))
	span.AddEvent("DrainDelayStarted", trace.WithAttributes(
		attribute.String("delay", delay.String())))

	drainCtx, cancel := context.WithTimeoutCause(ctx, delay,
		fmt.Errorf("shutdown drain delay of %s elapsed", delay))
	defer cancel()

	<-drainCtx.Done()

	l.Internal("Shutdown drain delay completed.")
	span.AddEvent("DrainDelayCompleted")
}

// shutdownMainServer gracefully stops the main HTTP server.
//
// Takes span (trace.Span) which records shutdown errors for tracing.
//
// Returns error when the server fails to shut down cleanly.
func (ds *daemonService) shutdownMainServer(ctx context.Context, span trace.Span) error {
	ctx, l := logger_domain.From(ctx, log)
	if ds.serverAdapter == nil {
		return nil
	}
	if err := ds.serverAdapter.Shutdown(ctx); err != nil {
		l.Error("Error shutting down main server", logger_domain.Error(err))
		span.RecordError(err)
		return fmt.Errorf("shutting down main server: %w", err)
	}
	return nil
}

// shutdownHealthServer gracefully stops the health check HTTP server.
//
// Takes span (trace.Span) which records shutdown errors for tracing.
//
// Returns error when the health server fails to shut down cleanly.
func (ds *daemonService) shutdownHealthServer(ctx context.Context, span trace.Span) error {
	ctx, l := logger_domain.From(ctx, log)
	if ds.healthServer == nil {
		return nil
	}
	if err := ds.healthServer.Shutdown(ctx); err != nil {
		l.Error("Error shutting down health server", logger_domain.Error(err))
		span.RecordError(err)
		return fmt.Errorf("shutting down health server: %w", err)
	}
	return nil
}

// shouldStartTLSRedirect returns true when TLS redirect is configured.
//
// Returns bool which is true when a TLS redirect server and port are set.
func (ds *daemonService) shouldStartTLSRedirect() bool {
	return ds.tlsRedirectServer != nil && ds.daemonConfig.TLSRedirectHTTPPort != ""
}

// startTLSRedirectServer starts a plain HTTP listener that 301-redirects all requests to
// the HTTPS server.
//
// Returns error when the redirect server fails to start or encounters a fatal listen
// error.
func (ds *daemonService) startTLSRedirectServer(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "startTLSRedirectServer")
	defer span.End()

	port := ds.daemonConfig.TLSRedirectHTTPPort
	httpsPort := ds.daemonConfig.NetworkPort
	addr := net.JoinHostPort("", port)

	l.Internal("Starting TLS redirect server",
		logger_domain.String(keyPort, port),
		logger_domain.String("https_port", httpsPort),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		if httpsPort != "443" {
			host = net.JoinHostPort(host, httpsPort)
		}
		target := "https://" + host + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently) //nolint:gosec // target host derived from r.Host which the server validates upstream
	})

	listenErr := ds.tlsRedirectServer.ListenAndServe(ctx, addr, handler)
	result := ds.handleServerListenResult(ctx, span, listenErr, addr, false, 0, serverKindTLSRedirect)
	if result == nil || !errors.Is(result, errContinueRetry) {
		return result
	}
	return result
}

// shutdownTLSRedirectServer gracefully stops the TLS redirect server.
//
// Takes span (trace.Span) which records any shutdown errors for tracing.
func (ds *daemonService) shutdownTLSRedirectServer(ctx context.Context, span trace.Span) {
	if ds.tlsRedirectServer == nil {
		return
	}
	if err := ds.tlsRedirectServer.Shutdown(ctx); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("Error shutting down TLS redirect server", logger_domain.Error(err))
		span.RecordError(err)
	}
}

// extractTraceContext gets trace context from incoming request headers.
//
// Uses a pooled MapCarrier to avoid creating new objects for each request. The
// OTel-extracted marker is set on PikoRequestCtx by the caller, not here.
//
// Takes r (*http.Request) which provides the headers containing trace data.
//
// Returns context.Context which holds the extracted trace information.
func extractTraceContext(r *http.Request) context.Context {
	reqCtx := r.Context()

	carrier, ok := mapCarrierPool.Get().(propagation.MapCarrier)
	if !ok {
		carrier = make(propagation.MapCarrier, mapCarrierPoolSize)
	}
	clear(carrier)

	for k, v := range r.Header {
		if len(v) > 0 {
			carrier.Set(k, v[0])
		}
	}

	result := otel.GetTextMapPropagator().Extract(reqCtx, carrier)
	mapCarrierPool.Put(carrier)

	return result
}
