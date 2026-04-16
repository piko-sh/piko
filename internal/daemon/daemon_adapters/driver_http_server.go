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
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// serverPurpose identifies the role of an HTTP server for logging.
type serverPurpose string

const (
	// serverPurposeMain is the purpose label for the main application server.
	serverPurposeMain serverPurpose = "main"

	// serverPurposeHealth marks a server as a health check endpoint.
	serverPurposeHealth serverPurpose = "health"

	// ServerPurposeMain is the exported form of serverPurposeMain for use by
	// the bootstrap layer when creating adapters via the TLS factory.
	ServerPurposeMain = serverPurposeMain

	// ServerPurposeHealth is the exported form of serverPurposeHealth.
	ServerPurposeHealth = serverPurposeHealth
)

// driverHTTPServerAdapter implements the ServerAdapter interface using the
// standard Go http.Server for production HTTP serving.
type driverHTTPServerAdapter struct {
	// server holds the HTTP server instance created during ListenAndServe.
	server *http.Server

	// tlsConfig holds optional TLS configuration; nil means plain HTTP.
	tlsConfig *TLSAdapterConfig

	// onBound is an optional callback invoked after the server successfully
	// binds to a port, receiving the resolved listen address.
	onBound func(address string)

	// purpose indicates whether this server handles health probes or main traffic.
	purpose serverPurpose
}

var _ daemon_domain.ServerAdapter = (*driverHTTPServerAdapter)(nil)

// ListenAndServe starts the HTTP server listening on the specified address.
// It blocks until the server shuts down or encounters a fatal error.
//
// Takes address (string) which specifies the TCP address to listen on.
// Takes handler (http.Handler) which handles incoming HTTP requests.
//
// Returns error when the address cannot be bound or the server fails.
func (a *driverHTTPServerAdapter) ListenAndServe(
	address string,
	handler http.Handler,
) error {
	ctx, span, l := log.Span(context.Background(), "driverHTTPServerAdapter.ListenAndServe",
		logger_domain.String("address", address),
	)
	defer span.End()

	l.Internal("Configuring HTTP server")
	a.server = &http.Server{
		Addr:     address,
		Handler:  handler,
		ErrorLog: stdlog.New(&httpServerErrorWriter{}, "", 0),

		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	a.recordServerSpanAttributes(span)

	listener, err := a.createListener(address, l)
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	a.logServerReady(l, address)

	if a.onBound != nil {
		a.onBound(address)
	}

	startTime := time.Now()
	err = a.server.Serve(listener)
	duration := time.Since(startTime)

	serverStartupDuration.Record(ctx, float64(duration.Milliseconds()))
	span.SetAttributes(attribute.Int64("durationMs", duration.Milliseconds()))
	recordServerCompletion(ctx, span, err)
	if err != nil {
		return fmt.Errorf("serving HTTP: %w", err)
	}
	return nil
}

// Shutdown stops the HTTP server gracefully, allowing in-flight requests to
// complete before returning.
//
// Returns error when the server fails to shut down within the context deadline.
func (a *driverHTTPServerAdapter) Shutdown(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "driverHTTPServerAdapter.Shutdown")
	defer span.End()

	if a.server == nil {
		l.Internal("No server instance to shutdown")
		span.SetStatus(codes.Ok, "No server instance to shutdown")
		return nil
	}

	l.Internal("Shutting down HTTP server")

	startTime := time.Now()
	err := a.server.Shutdown(ctx)
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

// SetOnBound registers a callback invoked after the server binds successfully.
//
// Takes fn (func(address string)) which is the callback receiving the resolved
// listen address.
func (a *driverHTTPServerAdapter) SetOnBound(fn func(address string)) {
	a.onBound = fn
}

// recordServerSpanAttributes adds server configuration attributes to the
// trace span.
//
// Takes span (trace.Span) which receives the configuration attributes.
func (a *driverHTTPServerAdapter) recordServerSpanAttributes(span trace.Span) {
	span.SetAttributes(
		attribute.Int64("readTimeoutMs", a.server.ReadTimeout.Milliseconds()),
		attribute.Int64("writeTimeoutMs", a.server.WriteTimeout.Milliseconds()),
		attribute.Int64("idleTimeoutMs", a.server.IdleTimeout.Milliseconds()),
		attribute.Int64("readHeaderTimeoutMs", a.server.ReadHeaderTimeout.Milliseconds()),
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
			NextProtos:     a.tlsConfig.NextProtos,
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

// NewDriverHTTPServerAdapter creates a new HTTP server adapter for the main
// server.
//
// Returns daemon_domain.ServerAdapter which is the configured adapter ready
// for use.
func NewDriverHTTPServerAdapter() daemon_domain.ServerAdapter {
	return &driverHTTPServerAdapter{purpose: serverPurposeMain}
}

// NewDriverHTTPServerAdapterWithTLS creates a TLS-enabled HTTP server adapter
// for the main server. The provided config controls TLS certificate loading,
// client auth, and protocol negotiation.
//
// Takes config (TLSAdapterConfig) which provides the TLS settings.
//
// Returns daemon_domain.ServerAdapter which is the configured TLS adapter
// ready for use.
func NewDriverHTTPServerAdapterWithTLS(config TLSAdapterConfig) daemon_domain.ServerAdapter {
	return &driverHTTPServerAdapter{purpose: serverPurposeMain, tlsConfig: &config}
}

// NewHealthServerAdapter creates a server adapter for the health probe server.
//
// Returns daemon_domain.ServerAdapter which provides the health probe server
// adapter ready for use.
func NewHealthServerAdapter() daemon_domain.ServerAdapter {
	return &driverHTTPServerAdapter{purpose: serverPurposeHealth}
}

// NewHealthServerAdapterWithTLS creates a TLS-enabled server adapter for the
// health probe server.
//
// Takes config (TLSAdapterConfig) which provides the TLS settings.
//
// Returns daemon_domain.ServerAdapter which provides the TLS-enabled health
// probe adapter ready for use.
func NewHealthServerAdapterWithTLS(config TLSAdapterConfig) daemon_domain.ServerAdapter {
	return &driverHTTPServerAdapter{purpose: serverPurposeHealth, tlsConfig: &config}
}

// recordServerCompletion records metrics and span status based on the server
// completion result.
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

// formatServerURL converts an address to a full URL with the appropriate
// scheme based on whether TLS is enabled.
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

// httpServerErrorWriter adapts Go's http.Server error output to Piko's
// structured logger. Messages that are expected noise (such as TLS handshake
// errors from plain-HTTP clients) are logged at Internal level; everything
// else is logged as a warning.
type httpServerErrorWriter struct{}

// Write logs the given bytes as a structured message and returns the number
// of bytes consumed.
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
