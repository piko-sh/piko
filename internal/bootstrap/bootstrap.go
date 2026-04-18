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

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/logger"
)

const (
	// runModeDev is the run mode value for development.
	runModeDev = "dev"

	// runModeDevInterpreted is the run mode for development with interpreted code.
	runModeDevInterpreted = "dev-i"

	// runModeProd is the run mode for production environments.
	runModeProd = "prod"

	// manifestFormatJSON is the format name for JSON manifest files.
	manifestFormatJSON = "json"

	// manifestFormatFlatbuffers is the FlatBuffers manifest format option.
	manifestFormatFlatbuffers = "flatbuffers"

	// manifestFilenameJSON is the file name for JSON format manifest files.
	manifestFilenameJSON = "manifest.json"

	// manifestFilenameBinary is the filename for the FlatBuffers manifest.
	manifestFilenameBinary = "manifest.bin"

	// distDirName is the folder name where build output files are stored.
	distDirName = "dist"

	// logKeyRunMode is the structured logging key for the daemon run mode.
	logKeyRunMode = "runMode"
)

// daemonBuilderFunc defines a function that builds a daemon service for a
// given run mode. It is used as a strategy in the dispatch table pattern.
type daemonBuilderFunc func(ctx context.Context, c *Container, deps *Dependencies) (daemon_domain.DaemonService, error)

// builders maps run mode strings to their builder functions using the Strategy
// pattern. New run modes are added by inserting a single entry.
var builders = map[string]daemonBuilderFunc{
	runModeProd:           buildProdDaemon,
	runModeDev:            buildDevDaemon,
	runModeDevInterpreted: buildDevInterpretedDaemon,
}

// ConfigAndContainer creates the DI container and loads all application
// configuration as the first phase of setup.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes deps (*Dependencies) which provides core dependencies including
// ConfigProvider and AppRouter.
// Takes opts (...Option) which allows changes to container behaviour.
//
// Returns *Container which is the fully configured container ready to build
// services.
// Returns error when deps.ConfigProvider or deps.AppRouter is nil, or when
// the server configuration cannot be loaded.
func ConfigAndContainer(ctx context.Context, deps *Dependencies, opts ...Option) (*Container, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Bootstrap Phase 1: Initialising container and loading configuration...")

	if deps.ConfigProvider == nil || deps.AppRouter == nil {
		err := errors.New("configProvider and appRouter dependencies cannot be nil")
		l.Error("Bootstrap validation failed", logger_domain.Error(err))
		return nil, fmt.Errorf("validating bootstrap dependencies: %w", err)
	}

	container := NewContainer(deps.ConfigProvider, opts...)

	container.applyAutoMemoryLimit(ctx)

	l.Internal("Validating provider configuration...")
	if err := container.ValidateProviderConfiguration(); err != nil {
		l.Error("Provider configuration validation failed", logger_domain.Error(err))
		return nil, fmt.Errorf("validating provider configuration: %w", err)
	}

	l.Internal("Loading and resolving server configuration (piko.yaml)...")
	loadCtx, err := container.config.LoadConfig(container.configServerDefaults, container.configServerOverrides, container.configResolvers...)
	if err != nil {
		return nil, fmt.Errorf("failed to load server configuration: %w", err)
	}
	summary, _ := config_domain.Summarise(loadCtx)
	l.Internal(summary)

	if container.websiteConfigOverride != nil {
		l.Internal("Using programmatic website configuration (WithWebsiteConfig)")
		container.config.WebsiteConfig = *container.websiteConfigOverride
	} else {
		l.Internal("Loading website configuration (config.json)...")
		if err := container.config.LoadWebsiteConfig(); err != nil {
			l.Warn("Could not load website config, using defaults.", logger_domain.Error(err))
			container.config.WebsiteConfig = config.WebsiteConfig{}
		} else {
			l.Internal("Website configuration loaded successfully")
		}
	}

	resolveFaviconSources(container)

	l.Internal("Initialising logger from configuration...")
	otelOpts := buildOtelSetupOptions(ctx, container)
	_, loggerShutdown, err := logger.Initialise(ctx, container.config.ServerConfig.Logger, NewOtelSetupConfig(&container.config.ServerConfig), otelOpts)
	if err != nil {
		l.Error("Failed to initialise logger from configuration, continuing with default logger", logger_domain.Error(err))
	} else {
		l.Internal("Logger initialised from configuration")
		if loggerShutdown != nil {
			shutdown.Register(ctx, "Logger", loggerShutdown)
		}
	}

	initialiseGlobalServices(container)

	return container, nil
}

// Daemon performs the second phase of initialisation. It takes a fully
// configured container and assembles the final, runnable daemon service.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes runMode (string) which specifies the operating mode for builder
// selection.
// Takes container (*Container) which provides the fully configured container.
// Takes deps (*Dependencies) which supplies the required dependencies.
//
// Returns daemon_domain.DaemonService which is the assembled daemon ready to
// run.
// Returns error when the run mode is unknown or daemon assembly fails.
func Daemon(ctx context.Context, runMode string, container *Container, deps *Dependencies) (daemon_domain.DaemonService, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Bootstrap Phase 2: Assembling daemon services...", logger_domain.String(logKeyRunMode, runMode))

	builder, ok := builders[runMode]
	if !ok {
		err := fmt.Errorf("invalid or unknown run mode: %q", runMode)
		l.Error("Could not find a builder for the run mode", logger_domain.Error(err), logger_domain.String(logKeyRunMode, runMode))
		return nil, fmt.Errorf("selecting daemon builder: %w", err)
	}

	l.Internal("Selected builder, assembling daemon...", logger_domain.String(logKeyRunMode, runMode))
	daemon, err := builder(ctx, container, deps)
	if err != nil {
		l.Error("Daemon assembly failed", logger_domain.Error(err), logger_domain.String(logKeyRunMode, runMode))
		return nil, fmt.Errorf("failed to build daemon for mode %s: %w", runMode, err)
	}

	l.Notice("Daemon successfully bootstrapped and ready to run.")
	return daemon, nil
}

// buildOtelSetupOptions creates OtelSetupOptions from the container's OTEL
// components. This includes the monitoring service's span processor and metrics
// reader, and the metrics exporter's reader if configured.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes container (*Container) which provides access to OTEL components.
//
// Returns *driver_handlers.OtelSetupOptions with span processors and metric
// readers,
// or nil if no components are configured.
func buildOtelSetupOptions(ctx context.Context, container *Container) *driver_handlers.OtelSetupOptions {
	_, l := logger_domain.From(ctx, log)
	metricsExporter := container.GetMetricsExporter()
	monitoringService := container.GetMonitoringService()

	if metricsExporter == nil && monitoringService == nil {
		return nil
	}

	var opts driver_handlers.OtelSetupOptions

	if monitoringService != nil {
		opts.AdditionalSpanProcessors = append(opts.AdditionalSpanProcessors, monitoringService.SpanProcessor())
		opts.AdditionalMetricReaders = append(opts.AdditionalMetricReaders, monitoringService.MetricsReader())
		l.Internal("Monitoring service OTEL components registered")
	}

	if metricsExporter != nil {
		opts.AdditionalMetricReaders = append(opts.AdditionalMetricReaders, metricsExporter.Reader())
		l.Internal("Metrics exporter reader registered")
	}

	return &opts
}
