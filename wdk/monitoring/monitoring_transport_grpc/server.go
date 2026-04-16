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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/netutil"
)

var _ monitoring_domain.TransportServer = (*Server)(nil)

// maxPortRetries is the maximum number of port increments to try when
// AutoNextPort is enabled.
const maxPortRetries = 100

// ServerConfig holds settings for the gRPC monitoring server.
type ServerConfig struct {
	// Address is the TCP address to listen on (e.g. ":9091" or "localhost:9091").
	Address string

	// GRPCServerOpts holds additional gRPC server options (e.g. TLS credentials).
	GRPCServerOpts []grpc.ServerOption

	// AutoNextPort enables automatic port selection when the configured port is
	// already in use. The server will try up to maxPortRetries consecutive ports.
	AutoNextPort bool

	// EnableReflection controls whether the gRPC reflection service is
	// registered. Reflection exposes the full API surface, so consider
	// disabling it in production.
	EnableReflection bool
}

// ServerOption configures the monitoring server.
type ServerOption func(*ServerConfig)

// Server wraps a gRPC server for monitoring services.
type Server struct {
	// deps holds the monitoring dependencies injected into the server.
	deps monitoring_domain.MonitoringDeps

	// grpcServer is the underlying gRPC server instance.
	grpcServer *grpc.Server

	// actualAddress holds the address the server is actually listening on,
	// which may differ from config.Address when AutoNextPort is enabled.
	actualAddress atomic.Value

	// config holds the server configuration including the listen address.
	config ServerConfig
}

// serviceRegistrar is a function type that registers gRPC services on a
// server. This is package-private; external users go through Transport().
type serviceRegistrar func(server *grpc.Server, deps monitoring_domain.MonitoringDeps)

// NewServer creates a new monitoring gRPC server.
//
// Takes deps (MonitoringDeps) which provides access to inspectors and telemetry.
// Takes registrar (serviceRegistrar) which registers the gRPC services.
// Takes opts (...ServerOption) for configuration.
//
// Returns *Server ready to be started.
func NewServer(deps monitoring_domain.MonitoringDeps, registrar serviceRegistrar, opts ...ServerOption) *Server {
	config := ServerConfig{
		Address:          ":9091",
		EnableReflection: true,
	}

	for _, opt := range opts {
		opt(&config)
	}

	recoveryFunc := func(
		ctx context.Context,
		req any,
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = status.Errorf(codes.Internal, "panic recovered: %v", r)
			}
		}()
		return handler(ctx, req)
	}

	allOpts := make([]grpc.ServerOption, 0, len(config.GRPCServerOpts)+1)
	allOpts = append(allOpts, grpc.ChainUnaryInterceptor(recoveryFunc))
	allOpts = append(allOpts, config.GRPCServerOpts...)
	grpcServer := grpc.NewServer(allOpts...)

	server := &Server{
		grpcServer: grpcServer,
		config:     config,
		deps:       deps,
	}

	if registrar != nil {
		registrar(grpcServer, deps)
	}

	if config.EnableReflection {
		reflection.Register(grpcServer)
	}

	return server
}

// Start begins serving gRPC requests. When AutoNextPort is enabled, it retries
// on consecutive ports if the configured port is already in use.
//
// Returns error when the server fails to start or encounters an error while
// serving.
//
// Safe for concurrent use. The spawned goroutine runs until the context is
// cancelled or the server encounters an error.
func (s *Server) Start(ctx context.Context) error {
	listener, err := s.listen(ctx)
	if err != nil {
		return fmt.Errorf("starting gRPC monitoring server listener: %w", err)
	}

	addr := listener.Addr().String()
	s.actualAddress.Store(addr)

	ctx, l := logger_domain.From(ctx, log)
	l.Notice("Starting gRPC monitoring server",
		String("address", addr),
	)

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		defer goroutine.RecoverPanicToChannel(ctx, "monitoring.grpcServe", errCh)
		if err := s.grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		s.Stop(context.Background())
		return ctx.Err()
	case err := <-errCh:
		if err == nil {
			err = errors.New("server exited without error")
		}
		return fmt.Errorf("serving gRPC monitoring server: %w", err)
	}
}

// Stop shuts down the gRPC server gracefully with a 10-second timeout. If the
// graceful shutdown does not complete within the timeout, the server is
// forcefully stopped to prevent indefinite blocking.
//
// Takes ctx (context.Context) for logging context propagation.
//
// Safe for concurrent use. The spawned goroutine runs until GracefulStop
// completes or the timeout expires.
func (s *Server) Stop(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	l.Notice("Stopping gRPC monitoring server")

	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		l.Notice("gRPC monitoring server stopped gracefully")
	case <-time.After(10 * time.Second):
		l.Warn("gRPC monitoring server graceful stop timed out, forcing stop")
		s.grpcServer.Stop()
	}
}

// Address returns the address the server is listening on.
//
// Returns string which is the network address the server listens on.
func (s *Server) Address() string {
	if addr, ok := s.actualAddress.Load().(string); ok && addr != "" {
		return addr
	}
	return s.config.Address
}

// listen opens a TCP listener on the configured address.
//
// Returns net.Listener which is the bound TCP listener.
// Returns error when the address cannot be bound.
func (s *Server) listen(ctx context.Context) (net.Listener, error) {
	_, l := logger_domain.From(ctx, log)
	if !s.config.AutoNextPort {
		listener, err := net.Listen("tcp", s.config.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on %s: %w", s.config.Address, err)
		}
		return listener, nil
	}

	host, portString, err := splitHostPort(s.config.Address)
	if err != nil {
		return nil, fmt.Errorf("invalid monitoring address %q: %w", s.config.Address, err)
	}

	initialPort, err := strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("invalid port in monitoring address %q: %w", s.config.Address, err)
	}

	l.Internal("Starting gRPC monitoring server with auto-port selection",
		Int("initial_port", initialPort),
		Int("max_retries", maxPortRetries))

	for i := range maxPortRetries {
		currentPort := initialPort + i
		addr := net.JoinHostPort(host, strconv.Itoa(currentPort))

		listener, listenErr := net.Listen("tcp", addr)
		if listenErr == nil {
			if i > 0 {
				l.Notice("gRPC monitoring server using alternative port",
					Int("port", currentPort),
					String("address", addr))
			}
			return listener, nil
		}

		if !netutil.IsPortInUseError(listenErr) {
			return nil, fmt.Errorf("failed to listen on %s: %w", addr, listenErr)
		}

		l.Warn("Monitoring port in use, trying next port...",
			String("address", addr),
			Int("attempt", i+1))
	}

	return nil, fmt.Errorf("could not find an available port after %d retries, starting from %d", maxPortRetries, initialPort)
}

// WithAddress sets the server listen address.
//
// Takes addr (string) which specifies the address to listen on.
//
// Returns ServerOption which configures the server with the given address.
func WithAddress(addr string) ServerOption {
	return func(c *ServerConfig) {
		c.Address = addr
	}
}

// WithServerGRPCOptions appends additional gRPC server options, such as TLS credentials.
//
// Takes opts (...grpc.ServerOption) which are the options to add.
//
// Returns ServerOption which appends the gRPC options to the server config.
func WithServerGRPCOptions(opts ...grpc.ServerOption) ServerOption {
	return func(c *ServerConfig) {
		c.GRPCServerOpts = append(c.GRPCServerOpts, opts...)
	}
}

// WithAutoNextPort enables automatic port selection. When the configured port
// is already in use, the server tries the next port, up to maxPortRetries
// attempts.
//
// Takes enabled (bool) which controls whether auto-port selection is active.
//
// Returns ServerOption which configures auto-port selection on the server.
func WithAutoNextPort(enabled bool) ServerOption {
	return func(c *ServerConfig) {
		c.AutoNextPort = enabled
	}
}

// WithServerReflection controls whether the gRPC reflection service is
// registered. Reflection is enabled by default.
//
// Takes enabled (bool) which controls whether reflection is active.
//
// Returns ServerOption which configures reflection on the server.
func WithServerReflection(enabled bool) ServerOption {
	return func(c *ServerConfig) {
		c.EnableReflection = enabled
	}
}

// splitHostPort splits an address into host and port parts.
//
// Takes address (string) which is the "host:port" or ":port" address.
//
// Returns string which is the host portion.
// Returns string which is the port portion.
// Returns error when the address cannot be parsed.
func splitHostPort(address string) (host string, port string, err error) {
	host, port, err = net.SplitHostPort(address)
	if err != nil {
		return "", "", err
	}
	return host, port, nil
}
