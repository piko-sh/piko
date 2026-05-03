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
	"fmt"
	"path/filepath"

	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_adapters"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_adapters"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/typegen/typegen_adapters"
	"piko.sh/piko/wdk/safedisk"
)

// runInitialTasksInBackground spawns a goroutine that runs the lifecycle
// service's one-time startup tasks (theme seeding, configuration loading,
// asset discovery). Errors are logged unless the context has already been
// cancelled.
//
// Takes appCtx (context.Context) which controls the lifetime of the goroutine.
// Takes l (logger_domain.Logger) which logs errors from the background tasks.
// Takes lifecycleService (lifecycle_domain.LifecycleService) which provides the
// initial tasks to run.
func runInitialTasksInBackground(appCtx context.Context, l logger_domain.Logger, lifecycleService lifecycle_domain.LifecycleService) {
	go func() {
		if err := lifecycleService.RunInitialTasks(appCtx); err != nil {
			if appCtx.Err() == nil {
				l.Error("Initial tasks failed", logger_domain.Error(err))
			}
		}
	}()
}

// wireMonitoringInspectors connects the orchestrator and registry inspectors
// to the monitoring service if it is enabled, then starts the service.
//
// Takes c (*Container) which provides access to the monitoring and inspector
// services.
// Takes renderRegistry (render_domain.RegistryPort) which may implement the
// render cache stats provider interface.
func wireMonitoringInspectors(c *Container, renderRegistry render_domain.RegistryPort) {
	_, l := logger_domain.From(c.GetAppContext(), log)

	monitoringService := c.GetMonitoringService()
	if monitoringService != nil {
		monitoringService.SetInspectors(
			c.GetOrchestratorInspector(),
			c.GetRegistryInspector(),
			c.GetMonitoringHealthProbeService(),
			c.GetDispatcherInspector(),
			c.GetRateLimiterInspector(),
		)

		if inspector := createProviderInfoAggregator(c); inspector != nil {
			monitoringService.SetProviderInfoInspector(inspector)
		}

		if cacheStats, ok := renderRegistry.(monitoring_domain.RenderCacheStatsProvider); ok {
			monitoringService.SetRenderCacheStatsProvider(cacheStats)
		}

		l.Internal("Wired monitoring inspectors")

		c.StartMonitoringService()
	}

	c.StartProfilingServer()
}

// buildDaemonWithWatcher is a common helper for building a daemon that requires
// a filesystem watcher, typically for development modes.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes modeName (string) which describes the mode for error messages.
// Takes sandboxFactory (safedisk.Factory) which creates sandboxes for
// filesystem access within the watcher.
// Takes buildDeps (func) which creates the daemon's dependency struct.
//
// Returns daemon_domain.DaemonService which is the initialised daemon.
// Returns error when the watcher fails to start or dependencies cannot be built.
func buildDaemonWithWatcher(
	ctx context.Context,
	modeName string,
	sandboxFactory safedisk.Factory,
	buildDeps func(lifecycle_domain.FileSystemWatcher) (*daemon_domain.DaemonServiceDeps, error),
) (daemon_domain.DaemonService, error) {
	fsWatcher, err := lifecycle_adapters.NewFSNotifyWatcher(sandboxFactory)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise file system watcher for %s: %w", modeName, err)
	}

	_, l := logger_domain.From(ctx, log)

	var success bool
	defer func() {
		if !success {
			if closeErr := fsWatcher.Close(); closeErr != nil {
				l.Warn("Failed to close file watcher during cleanup", logger_domain.Error(closeErr))
			}
		}
	}()

	deps, err := buildDeps(fsWatcher)
	if err != nil {
		return nil, fmt.Errorf("building %s daemon dependencies: %w", modeName, err)
	}

	success = true
	return daemon_domain.NewService(ctx, deps), nil
}

// wireDevAPIOptionalDeps attaches optional monitoring providers to the dev API
// handler and event broadcaster. Each provider is best-effort; failures are
// logged and the handler gracefully degrades to returning {"available": false}
// for that endpoint.
//
// Takes ctx (context.Context) which carries logging context.
// Takes c (*Container) which provides the dependency injection container.
// Takes devAPIHandler (*daemon_adapters.DevAPIHandler) which is the handler to
// configure.
// Takes devEventBroadcaster (*daemon_adapters.DevEventBroadcaster) which is
// the broadcaster to configure.
func wireDevAPIOptionalDeps(
	ctx context.Context,
	c *Container,
	devAPIHandler *daemon_adapters.DevAPIHandler,
	devEventBroadcaster *daemon_adapters.DevEventBroadcaster,
) {
	_, l := logger_domain.From(ctx, log)

	if healthService, err := c.GetHealthProbeService(); err == nil {
		adapter := monitoring_adapters.NewHealthProbeAdapter(healthService)
		devAPIHandler.SetHealthProbeService(adapter)
		devEventBroadcaster.SetHealthProbeService(adapter)
	} else {
		l.Trace("Health probe service not available for dev widget", logger_domain.Error(err))
	}

	resourceCollector := monitoring_domain.NewResourceCollector()
	resourceCollector.Start(c.GetAppContext())
	devAPIHandler.SetResourceProvider(resourceCollector)
	devEventBroadcaster.SetResourceProvider(resourceCollector)

	if inspector := createProviderInfoAggregator(c); inspector != nil {
		devAPIHandler.SetProviderInfoInspector(inspector)
		devEventBroadcaster.SetProviderInfoInspector(inspector)
	}
}

// ensureTypeDefinitions writes TypeScript type definitions to dist/ts/ for IDE
// integration, enabling autocomplete for the piko namespace and server-side
// actions. Errors are logged but do not fail the build.
//
// Takes ctx (context.Context) which carries logging context.
// Takes c (*Container) which provides the application context and
// server configuration.
func ensureTypeDefinitions(ctx context.Context, c *Container) {
	_, l := logger_domain.From(ctx, log)
	appCtx := c.GetAppContext()
	typegenService := typegen_adapters.NewTypeDefinitionService()
	baseDir := deref(c.serverConfig.Paths.BaseDir, ".")
	distTsDir := filepath.Join(baseDir, "dist", "ts")

	if err := typegenService.EnsureTypeDefinitions(appCtx, distTsDir); err != nil {
		l.Warn("Failed to write TypeScript type definitions",
			logger_domain.Error(err),
			logger_domain.String("dist_ts_dir", distTsDir),
		)
		return
	}

	l.Internal("TypeScript type definitions written to dist/ts/")
}
