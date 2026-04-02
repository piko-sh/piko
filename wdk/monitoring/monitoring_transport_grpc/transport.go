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

// Package monitoring_transport_grpc provides a gRPC transport implementation
// for the piko monitoring subsystem.
//
// Usage:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringTransport(monitoring_transport_grpc.Transport()),
//	)
package monitoring_transport_grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/safedisk"
)

// Option configures the gRPC transport.
type Option func(*transportConfig)

// transportConfig holds options for the gRPC monitoring transport.
type transportConfig struct {
	// grpcServerOpts holds additional gRPC server options (e.g. TLS credentials).
	grpcServerOpts []grpc.ServerOption

	// enableReflection controls whether gRPC reflection is registered.
	enableReflection bool
}

// WithReflection controls whether the gRPC reflection service is registered.
// Reflection is enabled by default.
//
// Takes enabled (bool) which controls whether reflection is active.
//
// Returns Option which configures reflection on the transport.
func WithReflection(enabled bool) Option {
	return func(c *transportConfig) {
		c.enableReflection = enabled
	}
}

// WithGRPCServerOptions appends additional gRPC server options, such as TLS
// credentials or interceptors.
//
// Takes opts (...grpc.ServerOption) which are the options to add.
//
// Returns Option which appends the gRPC options to the transport config.
func WithGRPCServerOptions(opts ...grpc.ServerOption) Option {
	return func(c *transportConfig) {
		c.grpcServerOpts = append(c.grpcServerOpts, opts...)
	}
}

// Transport returns a TransportFactory that creates a gRPC-based monitoring
// transport server. This is the entry point for bootstrap wiring.
//
// Takes opts (...Option) for gRPC-specific configuration.
//
// Returns monitoring_domain.TransportFactory which creates gRPC transport
// servers.
func Transport(opts ...Option) monitoring_domain.TransportFactory {
	tc := transportConfig{
		enableReflection: true,
	}
	for _, opt := range opts {
		opt(&tc)
	}

	return func(deps monitoring_domain.MonitoringDeps, config monitoring_domain.TransportConfig) (monitoring_domain.TransportServer, error) {
		var serverOpts []ServerOption
		serverOpts = append(serverOpts, WithAddress(config.Address))

		if config.AutoNextPort {
			serverOpts = append(serverOpts, WithAutoNextPort(true))
		}

		serverOpts = append(serverOpts, WithServerReflection(tc.enableReflection))

		if config.TLS.Enabled() {
			tlsFactory, factoryErr := safedisk.NewCLIFactory("")
			if factoryErr != nil {
				return nil, fmt.Errorf("creating sandbox factory for TLS: %w", factoryErr)
			}
			creds, cleanup, err := buildGRPCTLSCredentials(context.Background(), config.TLS, tlsFactory)
			if err != nil {
				return nil, err
			}
			serverOpts = append(serverOpts, WithServerGRPCOptions(grpc.Creds(creds)))

			_ = cleanup
		}

		if len(tc.grpcServerOpts) > 0 {
			serverOpts = append(serverOpts, WithServerGRPCOptions(tc.grpcServerOpts...))
		}

		server := NewServer(deps, defaultServiceRegistrar(), serverOpts...)
		return server, nil
	}
}
