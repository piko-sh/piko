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

package provider_grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

const (
	// defaultDialTimeout is the timeout for setting up the gRPC connection.
	defaultDialTimeout = 5 * time.Second

	// defaultRefreshInterval is the default interval between data refreshes.
	defaultRefreshInterval = 2 * time.Second
)

// Config holds the settings for the gRPC provider.
type Config struct {
	// TransportCredentials holds TLS credentials for the gRPC connection.
	// When nil, insecure credentials are used.
	TransportCredentials credentials.TransportCredentials

	// Address is the gRPC server address (e.g., "localhost:9091").
	Address string

	// DialTimeout is the maximum time allowed to establish the connection.
	DialTimeout time.Duration

	// RefreshInterval is how often data is refreshed; 0 means no automatic refresh.
	RefreshInterval time.Duration
}

// Option configures the provider settings.
type Option func(*Config)

// Connection holds a shared gRPC connection and its service clients.
// It implements io.Closer for resource cleanup.
type Connection struct {
	// conn is the underlying gRPC client connection; nil until connected.
	conn *grpc.ClientConn

	// healthClient provides access to the gRPC health check service.
	healthClient pb.HealthServiceClient

	// metricsClient provides gRPC access to system stats and traces.
	metricsClient pb.MetricsServiceClient

	// orchestratorClient provides access to the orchestrator service for
	// fetching task summaries, recent tasks, and workflow summaries.
	orchestratorClient pb.OrchestratorInspectorServiceClient

	// registryClient queries the registry for artefact and variant summaries.
	registryClient pb.RegistryInspectorServiceClient

	// dispatcherClient provides access to the dispatcher inspector service
	// for querying dead letter queues and dispatcher statistics.
	dispatcherClient pb.DispatcherInspectorServiceClient

	// rateLimiterClient provides access to the rate limiter inspector service
	// for querying rate limiter status and counters.
	rateLimiterClient pb.RateLimiterInspectorServiceClient

	// providerInfoClient provides access to the provider info service for
	// querying registered providers across hexagons.
	providerInfoClient pb.ProviderInfoServiceClient
}

// NewConnection creates a new gRPC connection to the monitoring server.
//
// Takes address (string) which specifies the gRPC server address.
// Takes opts (...Option) which provide optional settings.
//
// Returns *Connection which holds the gRPC clients for health, metrics,
// orchestrator, and registry services.
// Returns error when the connection cannot be set up or the health check fails.
func NewConnection(address string, opts ...Option) (*Connection, error) {
	config := Config{
		Address:         address,
		DialTimeout:     defaultDialTimeout,
		RefreshInterval: defaultRefreshInterval,
	}

	for _, opt := range opts {
		opt(&config)
	}

	transportCreds := insecure.NewCredentials()
	if config.TransportCredentials != nil {
		transportCreds = config.TransportCredentials
	}

	conn, err := grpc.NewClient(config.Address,
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.DefaultConfig,
		}),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating gRPC client for %s: %w", config.Address, err)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), config.DialTimeout,
		fmt.Errorf("gRPC dial exceeded %s timeout", config.DialTimeout))
	defer cancel()

	healthClient := pb.NewHealthServiceClient(conn)
	metricsClient := pb.NewMetricsServiceClient(conn)
	orchestratorClient := pb.NewOrchestratorInspectorServiceClient(conn)
	registryClient := pb.NewRegistryInspectorServiceClient(conn)
	dispatcherClient := pb.NewDispatcherInspectorServiceClient(conn)
	rateLimiterClient := pb.NewRateLimiterInspectorServiceClient(conn)
	providerInfoClient := pb.NewProviderInfoServiceClient(conn)

	if _, err := healthClient.GetHealth(ctx, &pb.GetHealthRequest{}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connecting to gRPC server at %s: %w", config.Address, err)
	}

	return &Connection{
		conn:               conn,
		healthClient:       healthClient,
		metricsClient:      metricsClient,
		orchestratorClient: orchestratorClient,
		registryClient:     registryClient,
		dispatcherClient:   dispatcherClient,
		rateLimiterClient:  rateLimiterClient,
		providerInfoClient: providerInfoClient,
	}, nil
}

// HealthClient returns the gRPC health service client.
//
// Returns pb.HealthServiceClient which provides health checking operations.
func (c *Connection) HealthClient() pb.HealthServiceClient { return c.healthClient }

// MetricsClient returns the gRPC metrics service client.
//
// Returns pb.MetricsServiceClient which provides access to the metrics API.
func (c *Connection) MetricsClient() pb.MetricsServiceClient { return c.metricsClient }

// OrchestratorClient returns the gRPC orchestrator inspector service client.
//
// Returns pb.OrchestratorInspectorServiceClient which provides access to the
// orchestrator inspection API.
func (c *Connection) OrchestratorClient() pb.OrchestratorInspectorServiceClient {
	return c.orchestratorClient
}

// RegistryClient returns the gRPC registry inspector service client.
//
// Returns pb.RegistryInspectorServiceClient which provides access to the
// registry inspection API.
func (c *Connection) RegistryClient() pb.RegistryInspectorServiceClient { return c.registryClient }

// DispatcherClient returns the gRPC dispatcher inspector service client.
//
// Returns pb.DispatcherInspectorServiceClient which provides access to the
// dispatcher inspection API.
func (c *Connection) DispatcherClient() pb.DispatcherInspectorServiceClient {
	return c.dispatcherClient
}

// RateLimiterClient returns the gRPC rate limiter inspector service client.
//
// Returns pb.RateLimiterInspectorServiceClient which provides access to the
// rate limiter inspection API.
func (c *Connection) RateLimiterClient() pb.RateLimiterInspectorServiceClient {
	return c.rateLimiterClient
}

// ProviderInfoClient returns the gRPC provider info service client.
//
// Returns pb.ProviderInfoServiceClient which provides access to the
// provider info inspection API.
func (c *Connection) ProviderInfoClient() pb.ProviderInfoServiceClient {
	return c.providerInfoClient
}

// Close closes the gRPC connection.
//
// Returns error when the underlying connection fails to close.
func (c *Connection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// WithDialTimeout sets the dial timeout for establishing connections.
//
// Takes d (time.Duration) which specifies the maximum time to wait for a
// connection to be established.
//
// Returns Option which configures the dial timeout on the Config.
func WithDialTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.DialTimeout = d
	}
}

// WithRefreshInterval sets the refresh interval for the provider.
//
// Takes d (time.Duration) which specifies the interval between refreshes.
//
// Returns Option which configures the refresh interval on the Config.
func WithRefreshInterval(d time.Duration) Option {
	return func(c *Config) {
		c.RefreshInterval = d
	}
}

// WithTransportCredentials sets TLS transport credentials for the gRPC
// connection. When not set, insecure credentials are used.
//
// Takes creds (credentials.TransportCredentials) which provides TLS settings.
//
// Returns Option which configures transport credentials on the Config.
func WithTransportCredentials(creds credentials.TransportCredentials) Option {
	return func(c *Config) {
		c.TransportCredentials = creds
	}
}

// NewProviders creates all TUI providers using a shared gRPC connection.
//
// Takes address (string) which is the gRPC server address.
// Takes opts (...Option) for configuration.
//
// Returns *tui_domain.Providers which contains all the configured providers.
// Returns error when the connection cannot be established.
func NewProviders(address string, opts ...Option) (*tui_domain.Providers, error) {
	config := Config{
		Address:         address,
		DialTimeout:     defaultDialTimeout,
		RefreshInterval: defaultRefreshInterval,
	}

	for _, opt := range opts {
		opt(&config)
	}

	conn, err := NewConnection(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("establishing gRPC connection to %s: %w", address, err)
	}

	_, l := logger.From(context.Background(), log)
	l.Debug("Connected to gRPC monitoring server", logger.String("address", address))

	return &tui_domain.Providers{
		Resources: []tui_domain.ResourceProvider{
			NewResourceProvider(conn, config.RefreshInterval),
		},
		Metrics: []tui_domain.MetricsProvider{
			NewMetricsProvider(conn, config.RefreshInterval),
		},
		Traces: []tui_domain.TracesProvider{
			NewTracesProvider(conn, config.RefreshInterval),
		},
		Health: []tui_domain.HealthProvider{
			NewHealthProvider(conn, config.RefreshInterval),
		},
		System: []tui_domain.SystemProvider{
			NewSystemProvider(conn, config.RefreshInterval),
		},
		FDs: []tui_domain.FDsProvider{
			NewFDsProvider(conn, config.RefreshInterval),
		},
		Panels: nil,
	}, nil
}
