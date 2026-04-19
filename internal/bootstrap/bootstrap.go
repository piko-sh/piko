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
	"os"

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

// ConfigAndContainer creates the DI container and resolves all application
// configuration as the first phase of setup.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes deps (*Dependencies) which provides core dependencies (currently
// just AppRouter; nil is rejected).
// Takes opts (...Option) which allows changes to container behaviour.
//
// Returns *Container which is the fully configured container ready to build
// services.
// Returns error when deps.AppRouter is nil, or when the server
// configuration cannot be resolved.
func ConfigAndContainer(ctx context.Context, deps *Dependencies, opts ...Option) (*Container, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Bootstrap Phase 1: Initialising container and resolving configuration...")

	if deps.AppRouter == nil {
		err := errors.New("appRouter dependency cannot be nil")
		l.Error("Bootstrap validation failed", logger_domain.Error(err))
		return nil, fmt.Errorf("validating bootstrap dependencies: %w", err)
	}

	container := NewContainer(opts...)

	container.applyAutoMemoryLimit(ctx)

	l.Internal("Validating provider configuration...")
	if err := container.ValidateProviderConfiguration(); err != nil {
		l.Error("Provider configuration validation failed", logger_domain.Error(err))
		return nil, fmt.Errorf("validating provider configuration: %w", err)
	}

	l.Internal("Resolving server configuration from programmatic options...")
	loadCtx, err := resolveServerConfig(&container.serverConfig, container.configServerOverrides, container.configResolvers)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve server configuration: %w", err)
	}
	summary, _ := config_domain.Summarise(loadCtx)
	l.Internal(summary)

	if container.websiteConfigOverride != nil {
		l.Internal("Using programmatic website configuration (WithWebsiteConfig)")
		container.websiteConfig = *container.websiteConfigOverride
	} else {
		l.Internal("No website configuration supplied; using defaults. Pass WithWebsiteConfig to set theme, fonts, locales, and favicons.")
		container.websiteConfig = config.WebsiteConfig{}
	}

	resolveFaviconSources(container)

	l.Internal("Initialising logger from configuration...")
	otelOpts := buildOtelSetupOptions(ctx, container)
	_, loggerShutdown, err := logger.Initialise(ctx, container.serverConfig.Logger, NewOtelSetupConfig(&container.serverConfig), otelOpts)
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

// resolveServerConfig populates the supplied target ServerConfig from
// programmatic overrides and resolver placeholders.
//
// piko itself reads no files, env vars, or CLI flags. Only the Defaults,
// Programmatic, Resolvers, ProgrammaticOverrides and Validation passes
// from config_domain run here.
//
// Takes target (*ServerConfig) which is populated in place.
// Takes overrides (*ServerConfig) which carries the user's With*
// option values. May be nil.
// Takes resolvers ([]config_domain.Resolver) which expand placeholder
// strings such as "aws-secret:my/key" inside the override values.
//
// Returns *config_domain.LoadContext which carries source tracking for
// the summary.
// Returns error when merging or path validation fails.
func resolveServerConfig(target *ServerConfig, overrides *ServerConfig, resolvers []config_domain.Resolver) (*config_domain.LoadContext, error) {
	opts := config_domain.LoaderOptions{
		ProgrammaticOverrides: overrides,
		Resolvers:             resolvers,
		UseGlobalResolvers:    true,
		PassOrder: []config_domain.Pass{
			config_domain.PassDefaults,
			config_domain.PassProgrammatic,
			config_domain.PassResolvers,
			config_domain.PassProgrammaticOverrides,
			config_domain.PassValidation,
		},
	}

	ctx, err := config_domain.Load(context.Background(), target, opts)
	if err != nil {
		return nil, fmt.Errorf("resolving server configuration: %w", err)
	}

	if err := validateBaseDir(target); err != nil {
		return nil, fmt.Errorf("validating server paths: %w", err)
	}
	return ctx, nil
}

// validateBaseDir checks that the configured base directory exists and is a
// directory. Defaulted to "." when unset, so users without WithBaseDir get
// the current working directory.
//
// Takes sc (*ServerConfig) which is the resolved server configuration.
//
// Returns error when the base directory is missing, inaccessible, or not a
// directory.
func validateBaseDir(sc *ServerConfig) error {
	baseDir := "."
	if sc.Paths.BaseDir != nil {
		baseDir = *sc.Paths.BaseDir
	}
	info, err := os.Stat(baseDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("website base directory not found: %s", baseDir)
		}
		return fmt.Errorf("cannot access website base directory %s: %w", baseDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("baseDir is not a directory: %s", baseDir)
	}
	return nil
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
