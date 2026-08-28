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

package daemon_adapters

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/goroutine"
)

// serverPurpose identifies the role of an HTTP server for logging.
type serverPurpose string

const (
	// serverPurposeMain is the purpose label for the main application server.
	serverPurposeMain serverPurpose = "main"

	// serverPurposeHealth marks a server as a health check endpoint.
	serverPurposeHealth serverPurpose = "health"

	// ServerPurposeMain is the exported form of serverPurposeMain for use by the bootstrap
	// layer when creating adapters via the TLS factory.
	ServerPurposeMain = serverPurposeMain

	// ServerPurposeHealth is the exported form of serverPurposeHealth.
	ServerPurposeHealth = serverPurposeHealth

	// http2MaxConcurrentStreams is the most streams that can run at the same time on one
	// HTTP/2 connection.
	http2MaxConcurrentStreams = 250

	// http2SendPingTimeout is the duration of inactivity before sending a PING frame to
	// verify the client connection is still alive.
	http2SendPingTimeout = 30 * time.Second

	// http2PingTimeout is the duration to wait for a PING response before closing the
	// connection. Only applies when http2SendPingTimeout is non-zero.
	http2PingTimeout = 15 * time.Second

	// alpnProtocolHTTP2 is the ALPN identifier a client offers to negotiate HTTP/2 over TLS.
	alpnProtocolHTTP2 = "h2"

	// alpnProtocolHTTP11 is the ALPN identifier for HTTP/1.1.
	alpnProtocolHTTP11 = "http/1.1"
)

// driverHTTPServerAdapter implements the ServerAdapter interface using the standard Go
// http.Server for production HTTP serving.
type driverHTTPServerAdapter struct {
	// server holds the HTTP server instance created during ListenAndServe. Shutdown runs on
	// a different goroutine from the one serving, so the pointer is published atomically.
	server atomic.Pointer[http.Server]

	// tlsConfig holds optional TLS configuration; nil means plain HTTP.
	tlsConfig *TLSAdapterConfig

	// onBound is an optional callback invoked after the server successfully binds to a port,
	// receiving the resolved listen address.
	onBound func(address string)

	// boundChan closes once the listener is bound, and never closes when the bind fails.
	boundChan chan struct{}

	// purpose indicates whether this server handles health probes or main traffic.
	purpose serverPurpose

	// boundMu guards boundChan and boundClosed.
	boundMu sync.Mutex

	// boundClosed records that boundChan has been closed, so closing is idempotent. The port
	// retry loop can bind more than once.
	boundClosed bool
}

var (
	_ daemon_domain.ServerAdapter = (*driverHTTPServerAdapter)(nil)
)

// ListenAndServe starts the HTTP server listening on the specified address. It blocks
// until the server shuts down or encounters a fatal error.
//
// Takes address (string) which specifies the TCP address to listen on.
// Takes handler (http.Handler) which handles incoming HTTP requests.
//
// Returns error when the address cannot be bound or the server fails.
func (a *driverHTTPServerAdapter) ListenAndServe(
	ctx context.Context,
	address string,
	handler http.Handler,
) error {
	spanCtx, span, l := log.Span(context.WithoutCancel(ctx), "driverHTTPServerAdapter.ListenAndServe",
		logger_domain.String("address", address),
	)
	defer span.End()

	l.Internal("Configuring HTTP server")

	server := a.buildServer(spanCtx, address, handler)
	a.server.Store(server)

	a.recordServerSpanAttributes(span, server)

	listener, err := a.createListener(address, l)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	a.logServerReady(l, address)

	if a.onBound != nil {
		a.onBound(address)
	}
	a.markBound()

	stopWatching := a.closeOnContextCancel(ctx, server)
	defer stopWatching()

	startTime := time.Now()
	err = server.Serve(listener)
	duration := time.Since(startTime)

	serverStartupDuration.Record(spanCtx, float64(duration.Milliseconds()))
	span.SetAttributes(attribute.Int64("durationMs", duration.Milliseconds()))
	recordServerCompletion(spanCtx, span, err)
	if err != nil {
		return fmt.Errorf("serving HTTP: %w", err)
	}
	return nil
}

// Shutdown stops the HTTP server gracefully, allowing in-flight requests to complete
// before returning.
//
// Returns error when the server fails to shut down within the context deadline.
func (a *driverHTTPServerAdapter) Shutdown(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "driverHTTPServerAdapter.Shutdown")
	defer span.End()

	server := a.server.Load()
	if server == nil {
		l.Internal("No server instance to shutdown")
		span.SetStatus(codes.Ok, "No server instance to shutdown")
		return nil
	}

	l.Internal("Shutting down HTTP server")

	startTime := time.Now()
	err := server.Shutdown(ctx)
	duration := time.Since(startTime)

	serverShutdownDuration.Record(ctx, float64(duration.Milliseconds()))
	span.SetAttributes(attribute.Int64("durationMs", duration.Milliseconds()))

	if err != nil {
		l.Error("Error shutting down HTTP server", logger_domain.String(logger_domain.KeyError, err.Error()))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Error shutting down HTTP server")
		serverErrorCount.Add(ctx, 1)
	} else {
		l.Internal("HTTP server shutdown completed successfully")
		span.SetStatus(codes.Ok, "HTTP server shutdown completed successfully")
	}
	if err != nil {
		return fmt.Errorf("shutting down HTTP server: %w", err)
	}
	return nil
}

// Bound reports when the listener is accepting connections.
//
// Returns <-chan struct{} which closes once the listener is bound.
//
// Concurrency: safe for concurrent use; the channel is created under boundMu.
func (a *driverHTTPServerAdapter) Bound() <-chan struct{} {
	a.boundMu.Lock()
	defer a.boundMu.Unlock()

	return a.ensureBoundChanLocked()
}

// SetOnBound registers a callback invoked after the server binds successfully.
//
// Takes fn (func(address string)) which is the callback receiving the resolved listen
// address.
func (a *driverHTTPServerAdapter) SetOnBound(fn func(address string)) {
	a.onBound = fn
}

// markBound signals that the listener is accepting connections.
//
// Concurrency: safe for concurrent use; guarded by boundMu.
func (a *driverHTTPServerAdapter) markBound() {
	a.boundMu.Lock()
	defer a.boundMu.Unlock()

	channel := a.ensureBoundChanLocked()
	if a.boundClosed {
		return
	}
	a.boundClosed = true
	close(channel)
}

// ensureBoundChanLocked returns the bound channel, creating it when Bound has not yet
// been called. The caller must hold boundMu.
//
// Returns chan struct{} which closes once the listener binds.
func (a *driverHTTPServerAdapter) ensureBoundChanLocked() chan struct{} {
	if a.boundChan == nil {
		a.boundChan = make(chan struct{})
	}

	return a.boundChan
}

// closeOnContextCancel closes the server when the context is cancelled.
//
// Takes server (*http.Server) which is closed on cancellation.
//
// Returns func() which stops the watcher and waits for it.
//
// Concurrency: starts one goroutine; the returned function waits for it to exit, so it
// cannot outlive the call.
func (*driverHTTPServerAdapter) closeOnContextCancel(ctx context.Context, server *http.Server) func() {
	stopped := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer goroutine.RecoverPanic(ctx, "daemon.serverCancelWatcher")
		defer close(done)

		select {
		case <-ctx.Done():
			_ = server.Close()
		case <-stopped:
		}
	}()

	return func() {
		close(stopped)
		<-done
	}
}

// buildServer constructs the HTTP server, including its HTTP/2 configuration.
//
// Takes address (string) which is the TCP address the server will listen on.
// Takes handler (http.Handler) which handles incoming requests.
//
// Returns *http.Server which is configured but not yet listening.
func (a *driverHTTPServerAdapter) buildServer(ctx context.Context, address string, handler http.Handler) *http.Server {
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	if a.tlsConfig == nil {
		protocols.SetUnencryptedHTTP2(true)
	}

	countErrorCtx := context.WithoutCancel(ctx)

	return &http.Server{
		Addr:     address,
		Handler:  handler,
		ErrorLog: stdlog.New(&httpServerErrorWriter{}, "", 0),

		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,

		Protocols: protocols,
		HTTP2: &http.HTTP2Config{
			MaxConcurrentStreams: http2MaxConcurrentStreams,
			SendPingTimeout:      http2SendPingTimeout,
			PingTimeout:          http2PingTimeout,
			CountError: func(errType string) {
				daemon_domain.RecordHTTP2ProtocolError(countErrorCtx, errType)
			},
		},
	}
}

// recordServerSpanAttributes adds server configuration attributes to the trace span.
//
// Takes span (trace.Span) which receives the configuration attributes.
// Takes server (*http.Server) whose resolved timeouts are reported.
func (a *driverHTTPServerAdapter) recordServerSpanAttributes(span trace.Span, server *http.Server) {
	span.SetAttributes(
		attribute.Int64("readTimeoutMs", server.ReadTimeout.Milliseconds()),
		attribute.Int64("writeTimeoutMs", server.WriteTimeout.Milliseconds()),
		attribute.Int64("idleTimeoutMs", server.IdleTimeout.Milliseconds()),
		attribute.Int64("readHeaderTimeoutMs", server.ReadHeaderTimeout.Milliseconds()),
		attribute.Bool("tls.enabled", a.tlsConfig != nil),
	)
	if a.tlsConfig != nil {
		span.SetAttributes(
			attribute.String("tls.min_version", formatTLSVersion(a.tlsConfig.MinVersion)),
		)
	}
}

// createListener creates a TCP listener and optionally wraps it with TLS.
//
// Takes address (string) which is the TCP address to bind to.
// Takes l (logger_domain.Logger) which provides structured logging.
//
// Returns net.Listener which is the bound listener.
// Returns error when binding fails.
func (a *driverHTTPServerAdapter) createListener(address string, l logger_domain.Logger) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		l.Internal("Failed to bind to address", logger_domain.String("address", address), logger_domain.Error(err))
		return nil, fmt.Errorf("binding to address %s: %w", address, err)
	}

	if a.tlsConfig != nil {
		tlsConfig := &tls.Config{
			GetCertificate: a.tlsConfig.GetCertificate,
			ClientAuth:     a.tlsConfig.ClientAuth,
			ClientCAs:      a.tlsConfig.ClientCAs,
			MinVersion:     max(a.tlsConfig.MinVersion, tls.VersionTLS12),
			NextProtos:     alpnProtocols(a.tlsConfig.NextProtos),
		}
		listener = tls.NewListener(listener, tlsConfig)
		l.Internal("TLS enabled on listener",
			logger_domain.String("address", address),
			logger_domain.String("min_version", formatTLSVersion(a.tlsConfig.MinVersion)),
		)
	}

	return listener, nil
}

// logServerReady logs the appropriate ready message based on server purpose.
//
// Takes l (logger_domain.Logger) which provides structured logging.
// Takes address (string) which is the server address.
func (a *driverHTTPServerAdapter) logServerReady(l logger_domain.Logger, address string) {
	url := formatServerURL(address, a.tlsConfig != nil)
	if a.purpose == serverPurposeHealth {
		l.Internal("Health probe ready", logger_domain.String("url", url))
	} else {
		l.Internal("Server ready", logger_domain.String("url", url))
	}
}

// NewDriverHTTPServerAdapter creates a new HTTP server adapter for the main server.
//
// Returns daemon_domain.ServerAdapter which is the configured adapter ready for use.
func NewDriverHTTPServerAdapter() daemon_domain.ServerAdapter {
	return &driverHTTPServerAdapter{purpose: serverPurposeMain}
}

// alpnProtocols resolves the ALPN list offered by a TLS listener.
//
// The server declares HTTP/2 in its Protocols set, but ALPN is negotiated by the
// listener's tls.Config, so an empty list would leave the server advertising a protocol
// no client can select. Defaulting here keeps the two in step.
//
// Takes configured ([]string) which is the caller's ALPN preference order, possibly
// empty.
//
// Returns []string which is the configured list, or the HTTP/2-then-HTTP/1.1 default.
func alpnProtocols(configured []string) []string {
	if len(configured) > 0 {
		return configured
	}
	return []string{alpnProtocolHTTP2, alpnProtocolHTTP11}
}

// recordServerCompletion records metrics and span status based on the server completion
// result.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes span (trace.Span) which receives the status and any error details.
// Takes err (error) which indicates the server completion state.
func recordServerCompletion(ctx context.Context, span trace.Span, err error) {
	ctx, l := logger_domain.From(ctx, log)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		l.Error("HTTP server failed", logger_domain.String(logger_domain.KeyError, err.Error()))
		span.RecordError(err)
		span.SetStatus(codes.Error, "HTTP server failed")
		serverErrorCount.Add(ctx, 1)
		return
	}
	if errors.Is(err, http.ErrServerClosed) {
		span.SetStatus(codes.Ok, "HTTP server closed gracefully")
		return
	}
	span.SetStatus(codes.Ok, "HTTP server started successfully")
}

// formatServerURL converts an address to a full URL with the appropriate scheme based on
// whether TLS is enabled.
//
// Takes address (string) which is the host:port or just :port to format.
// Takes isTLS (bool) which selects https:// when true, http:// when false.
//
// Returns string which is the full URL with the correct scheme.
func formatServerURL(address string, isTLS bool) string {
	scheme := "http"
	if isTLS {
		scheme = "https"
	}
	if len(address) > 0 && address[0] == ':' {
		return scheme + "://localhost" + address
	}
	return scheme + "://" + address
}

// httpServerErrorWriter adapts Go's http.Server error output to Piko's structured logger.
// Messages that are expected noise (such as TLS handshake errors from plain-HTTP clients)
// are logged at Internal level; everything else is logged as a warning.
type httpServerErrorWriter struct{}

// Write logs the given bytes as a structured message and returns the number of bytes
// consumed.
//
// Takes p ([]byte) which contains the error message from the HTTP server.
//
// Returns int which is the number of bytes written.
// Returns error which is always nil.
func (*httpServerErrorWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if strings.Contains(msg, "TLS handshake error") {
		log.Internal(msg)
	} else {
		log.Warn(msg)
	}
	return len(p), nil
}

// formatTLSVersion returns a human-readable string for a TLS version constant.
//
// Takes version (uint16) which is the TLS version constant.
//
// Returns string which is the version label (e.g. "TLS 1.3").
func formatTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	default:
		return "unknown"
	}
}
