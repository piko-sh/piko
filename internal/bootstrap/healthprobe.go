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

package bootstrap

// This file provides integration of the healthprobe system into the Piko build
// pipeline.

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_adapters"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
)

// probeGetter is a function that returns a service that may implement
// healthprobe_domain.Probe.
type probeGetter func(c *Container) (any, error)

// probeRegistration holds a named health probe getter for service registration.
type probeRegistration struct {
	// getter retrieves the service instance to probe from the container.
	getter probeGetter

	// name is the service name used for logging.
	name string
}

// defaultCheckTimeoutSeconds is the default health probe check timeout.
const defaultCheckTimeoutSeconds = 5

// serviceProbes lists all services that may implement health probes.
var serviceProbes = []probeRegistration{
	{name: "RegistryService", getter: func(c *Container) (any, error) { return c.GetRegistryService() }},
	{name: "CollectionService", getter: func(c *Container) (any, error) { return c.GetCollectionService() }},
	{name: "OrchestratorService", getter: func(c *Container) (any, error) { return c.GetOrchestratorService() }},
	{name: "EventsProvider", getter: func(c *Container) (any, error) { return c.GetEventsProvider() }},
	{name: "StorageService", getter: func(c *Container) (any, error) { return c.GetStorageService() }},
	{name: "EmailService", getter: func(c *Container) (any, error) { return c.GetEmailService() }},
	{name: "CryptoService", getter: func(c *Container) (any, error) { return c.GetCryptoService() }},
	{name: "CacheService", getter: func(c *Container) (any, error) { return c.GetCacheService() }},
	{name: "SEOService", getter: func(c *Container) (any, error) { return c.GetSEOService() }},
	{name: "ImageService", getter: func(c *Container) (any, error) { return c.GetImageService() }},
	{name: "VideoService", getter: func(c *Container) (any, error) { return c.GetVideoService() }},
	{name: "LLMService", getter: func(c *Container) (any, error) { return c.GetLLMService() }},
	{name: "DatabaseService", getter: func(c *Container) (any, error) { return c.GetDatabaseService() }},
	{name: "CaptchaService", getter: func(c *Container) (any, error) { return c.GetCaptchaService() }},
	{name: "SpamDetectService", getter: func(c *Container) (any, error) { return c.GetSpamDetectService() }},
}

// createHealthProbeService creates and sets up the health probe service with
// health check probes from all services. Call this once during bootstrap.
//
// The health probe service:
//   - Runs liveness checks to see if the application is running.
//   - Runs readiness checks to see if the application can serve traffic.
//   - Gathers health status from all registered services and adapters.
//   - Provides health endpoints for tools like Kubernetes.
//
// Takes c (*Container) which provides access to services and configuration.
//
// Returns healthprobe_domain.Service which is the configured health probe
// service ready for use.
// Returns error when the service cannot be created.
func createHealthProbeService(c *Container) (healthprobe_domain.Service, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating health probe service...")

	registry := healthprobe_adapters.NewInMemoryRegistry()
	config := c.GetConfigProvider().ServerConfig
	checkTimeout := time.Duration(deref(config.HealthProbe.CheckTimeoutSeconds, defaultCheckTimeoutSeconds)) * time.Second

	registerServiceProbes(c, registry)

	registerDirectProbes(c, registry)

	registerCustomProbes(c, registry)

	healthProbeService := healthprobe_domain.NewService(registry, checkTimeout, "PikoApplication")
	l.Internal("Health probe service initialised")

	return healthProbeService, nil
}

// registerServiceProbes registers health probes from services that return
// a service and error pair.
//
// Takes c (*Container) which provides access to service instances.
// Takes registry (healthprobe_domain.Registry) which receives probe
// registrations.
func registerServiceProbes(c *Container, registry healthprobe_domain.Registry) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	for _, reg := range serviceProbes {
		service, err := reg.getter(c)
		if err != nil {
			continue
		}
		if probe, ok := service.(healthprobe_domain.Probe); ok {
			registry.Register(probe)
			l.Internal("Registered health probe", logger_domain.String("service", reg.name))
		}
	}
}

// registerDirectProbes adds health probes from services that do not return
// errors during setup.
//
// Takes c (*Container) which provides access to the service instances.
// Takes registry (healthprobe_domain.Registry) which receives the probe
// registrations.
func registerDirectProbes(c *Container, registry healthprobe_domain.Registry) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	if probe, ok := c.GetRenderer().(healthprobe_domain.Probe); ok {
		registry.Register(probe)
		l.Internal("Registered RenderService health probe")
	}

	if probe, ok := c.GetCSRFService().(healthprobe_domain.Probe); ok {
		registry.Register(probe)
		l.Internal("Registered CSRFService health probe")
	}
}

// registerCustomProbes adds custom health probes from the container to the
// registry.
//
// Takes c (*Container) which holds the custom health probes to register.
// Takes registry (healthprobe_domain.Registry) which receives the probe
// registrations.
func registerCustomProbes(c *Container, registry healthprobe_domain.Registry) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	for _, customProbe := range c.customHealthProbes {
		registry.Register(customProbe)
		l.Internal("Registered custom health probe", logger_domain.String("probe_name", customProbe.Name()))
	}

	if len(c.customHealthProbes) > 0 {
		l.Internal("Registered custom health probes", logger_domain.Int("count", len(c.customHealthProbes)))
	}
}

// setupHealthProbeServer creates the health probe HTTP server and router.
//
// Takes c (*Container) which provides access to settings and services.
//
// Returns daemon_domain.ServerAdapter which is the configured health server.
// Returns http.Handler which is the router with health check paths added.
// Returns daemon_domain.DrainSignaller which signals drain to the health
// service during shutdown; nil when health probes are disabled.
// Returns error when the health probe service cannot be obtained.
func setupHealthProbeServer(
	c *Container,
) (daemon_domain.ServerAdapter, http.Handler, daemon_domain.DrainSignaller, error) {
	config := c.GetConfigProvider().ServerConfig

	if !deref(config.HealthProbe.Enabled, true) {
		return nil, nil, nil, nil
	}

	healthProbeService, err := c.GetHealthProbeService()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get health probe service: %w", err)
	}

	healthHandler := healthprobe_adapters.NewHTTPHandlerAdapter(healthProbeService)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	livePath := deref(config.HealthProbe.LivePath, "/live")
	readyPath := deref(config.HealthProbe.ReadyPath, "/ready")

	r.Method(http.MethodGet, livePath, http.HandlerFunc(healthHandler.ServeLiveness))
	r.Method(http.MethodGet, readyPath, http.HandlerFunc(healthHandler.ServeReadiness))

	_, l := logger_domain.From(c.GetAppContext(), log)

	if deref(config.HealthProbe.MetricsEnabled, true) {
		metricsExporter := c.GetMetricsExporter()
		if metricsExporter != nil {
			metricsPath := deref(config.HealthProbe.MetricsPath, "/metrics")
			r.Method(http.MethodGet, metricsPath, metricsExporter.Handler())
			l.Internal("Metrics endpoint configured",
				logger_domain.String("metrics_path", metricsPath))
		}
	}

	daemonConfig := NewDaemonConfig(&config)
	healthServer, tlsCleanup, err := daemon_adapters.NewServerAdapterFromTLSConfig(
		c.GetAppContext(),
		daemonConfig.HealthTLS,
		daemon_adapters.ServerPurposeHealth,
		c.createSandbox,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("initialising health server TLS: %w", err)
	}
	if tlsCleanup != nil {
		shutdown.Register(c.GetAppContext(), "HealthTLSCertificateLoader", func(_ context.Context) error {
			return tlsCleanup()
		})
	}

	l.Internal("Health check server configured",
		logger_domain.String("live_path", livePath),
		logger_domain.String("ready_path", readyPath))

	var drainSignaller daemon_domain.DrainSignaller
	if ds, ok := healthProbeService.(daemon_domain.DrainSignaller); ok {
		drainSignaller = ds
	}

	return healthServer, r, drainSignaller, nil
}

// newTLSRedirectServerIfConfigured returns a new HTTP server adapter when the
// daemon config enables TLS redirect, or nil otherwise.
//
// Takes daemonConfig (daemon_domain.DaemonConfig) which provides the TLS
// redirect settings to check.
//
// Returns daemon_domain.ServerAdapter which is the redirect server, or nil
// when TLS redirect is not enabled.
func newTLSRedirectServerIfConfigured(daemonConfig daemon_domain.DaemonConfig) daemon_domain.ServerAdapter {
	if daemonConfig.TLSRedirectHTTPPort != "" {
		return daemon_adapters.NewDriverHTTPServerAdapter()
	}
	return nil
}
