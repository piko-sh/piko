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

package piko

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/components"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/email/email_provider_mock"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/runtime"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// RunModeDev is the development mode with hot reloading.
	RunModeDev = "dev"

	// RunModeDevInterpreted is the run mode for development with interpreted code.
	RunModeDevInterpreted = "dev-i"

	// RunModeProd is the production run mode for the SSR server.
	RunModeProd = "prod"

	// dirPermOwnerRWX is the file permission for directories (rwxr-x---).
	dirPermOwnerRWX = 0750

	// defaultComponentStartTimeout is the maximum time allowed for a lifecycle
	// component to start before the start is considered failed.
	defaultComponentStartTimeout = 30 * time.Second

	// GenerateModeManifest is the generation mode for manifest output.
	GenerateModeManifest = bootstrap.GenerateModeManifest

	// GenerateModeAll generates all build outputs including SQL, manifest, and assets.
	GenerateModeAll = bootstrap.GenerateModeAll

	// GenerateModeSQL runs the querier code generator against registered databases.
	GenerateModeSQL = bootstrap.GenerateModeSQL

	// GenerateModeAssets runs annotation to discover template-derived asset
	// requirements, then builds static assets. Code emission is skipped.
	GenerateModeAssets = bootstrap.GenerateModeAssets

	// LevelDebug is the debug log level for detailed diagnostic information.
	LevelDebug = slog.LevelDebug

	// LevelInfo is the info logging level from the standard slog package.
	LevelInfo = slog.LevelInfo

	// LevelWarn is the warning log level for conditions that may need attention.
	LevelWarn = slog.LevelWarn

	// LevelError is the error severity level for log messages.
	LevelError = slog.LevelError
)

var (
	// Version holds the current version of Piko. This can be overridden at build
	// time using: go build -ldflags "-X piko.sh/piko.Version=1.0.0".
	Version = "0.1.0-alpha"

	// log is the package-level logger for the piko package.
	log = logger_domain.GetLogger("piko")

	// WithComponents registers external PKC components with the Piko framework,
	// letting third-party component libraries provide components that can be used
	// in templates.
	//
	// Components registered via WithComponents are validated at startup:
	//   - Tag names must contain a hyphen (e.g., "my-button", not "button")
	//   - Tag names must not shadow HTML elements (e.g., "div", "span")
	//   - Tag names must not use reserved prefixes (piko:, pml-)
	//   - Duplicate tag names result in a startup error
	//
	// Takes components (...ComponentDefinition) which specifies the external
	// components to register.
	//
	// Returns Option which configures the container with the external components.
	//
	// Example:
	// server := piko.New(
	//     piko.WithComponents(
	//         piko.ComponentDefinition{
	//             TagName:    "my-button",
	//             ModulePath: "github.com/myorg/piko-components",
	//         },
	//         piko.ComponentDefinition{
	//             TagName:    "my-card",
	//             ModulePath: "github.com/myorg/piko-components",
	//         },
	//     ),
	// )
	WithComponents = bootstrap.WithComponents

	// WithSEO provides the SEO configuration for sitemap and robots.txt
	// generation. SEO is only active when this option is provided with an
	// enabled configuration and a non-empty sitemap hostname.
	WithSEO = bootstrap.WithSEO

	// WithAssets provides the asset configuration including image/video profiles,
	// screen breakpoints, and default densities for responsive images. These
	// settings are used at compile time for static asset analysis.
	WithAssets = bootstrap.WithAssets

	// WithWebsiteConfig provides the website configuration programmatically,
	// replacing the file-based config.json loading entirely. When set, the
	// config.json file is not read.
	//
	// Use it for programmatic server setup where the website settings are
	// defined in Go code rather than a JSON file.
	WithWebsiteConfig = bootstrap.WithWebsiteConfig

	// WithStandardLoader causes the type inspector to use the
	// standard golang.org/x/tools/go/packages.Load instead of the
	// faster quickpackages loader.
	//
	// This is slower but always stable, since it is maintained by
	// the Go team. Useful as a fallback when quickpackages
	// encounters issues with specific dependency configurations
	// (e.g. complex CGo setups).
	WithStandardLoader = bootstrap.WithStandardLoader
)

// SymbolExports is a type alias for registering exported symbols with the
// template interpreter. It maps package names to symbol names to their values,
// and is compatible with SymbolExports from the interpreter provider.
type SymbolExports = templater_domain.SymbolExports

// PublicConfig defines the set of configurable options that can be set
// programmatically when embedding the Piko server.
type PublicConfig struct {
	// BaseDir is the base directory path for server files; empty uses the default.
	BaseDir string

	// AssetsSourceDir is the path to the folder that holds static assets.
	AssetsSourceDir string

	// PagesSourceDir is the path to the folder containing page templates.
	PagesSourceDir string

	// Port specifies the TCP port number; 0 uses the default from the config.
	Port int

	// WatchMode enables automatic rebuilding when source files change.
	WatchMode bool
}

// SSRServer is the main struct for working with the Piko framework.
type SSRServer struct {
	// AppRouter is the HTTP router for web requests.
	AppRouter *chi.Mux

	// Container holds the dependency injection container for the server.
	Container *bootstrap.Container

	// config holds the server configuration provider.
	config *config.Provider

	// daemon holds the running daemon service instance; nil until Run is called.
	daemon daemon_domain.DaemonService

	// interpreterProvider supplies the interpreter pool for dev-i mode.
	interpreterProvider templater_domain.InterpreterProviderPort

	// symbols holds optional interpreter exports for registration.
	symbols templater_domain.SymbolExports

	// PreBuildHook is called after the previous daemon is stopped and caches
	// are invalidated, but before the new daemon starts building. This allows
	// tests to reset spy statistics in a race-free window.
	PreBuildHook func()

	// lifecycleComponents holds registered components for lifecycle management.
	lifecycleComponents []LifecycleComponent

	// options holds bootstrap settings passed to ConfigAndContainer.
	options []Option
}

// Configure sets configuration values to override those loaded from disk.
//
// Takes publicConfig (PublicConfig) which specifies the
// configuration values to apply.
func (s *SSRServer) Configure(publicConfig PublicConfig) {
	if publicConfig.Port != 0 {
		s.config.ServerConfig.Network.Port = new(fmt.Sprintf("%d", publicConfig.Port))
	}
	if publicConfig.BaseDir != "" {
		s.config.ServerConfig.Paths.BaseDir = &publicConfig.BaseDir
	}
	if publicConfig.PagesSourceDir != "" {
		s.config.ServerConfig.Paths.PagesSourceDir = &publicConfig.PagesSourceDir
	}
	if publicConfig.AssetsSourceDir != "" {
		s.config.ServerConfig.Paths.AssetsSourceDir = &publicConfig.AssetsSourceDir
	}
	s.config.ServerConfig.Build.WatchMode = &publicConfig.WatchMode
}

// RegisterLifecycle adds a component to be managed during server startup and
// shutdown.
//
// Components are started in the order they are added. OnStart is called before
// the HTTP server starts. During shutdown, components are stopped in reverse
// order (last added, first stopped).
//
// If the component also implements HealthProbe, it will be added to the health
// monitoring system and shown via the /health, /live, and /ready endpoints.
//
// May be called many times, but must be called before Run.
//
// Takes component (LifecycleComponent) which is the component to add.
//
// Example:
// ssr := piko.New()
// ssr.RegisterLifecycle(myDatabaseComponent)
// ssr.RegisterLifecycle(myCacheComponent)
// ssr.Run(actions, piko.RunModeDev)
func (s *SSRServer) RegisterLifecycle(component LifecycleComponent) {
	s.lifecycleComponents = append(s.lifecycleComponents, component)
	_, l := logger_domain.From(context.Background(), log)
	l.Internal("Lifecycle component registered", logger_domain.String("component", component.Name()))
}

// Setup bootstraps the configuration and creates the DI container without
// starting the server. Use it when you need access to services without
// running the daemon, such as in tests.
//
// Returns error when the container fails to initialise.
func (s *SSRServer) Setup() error {
	deps := &bootstrap.Dependencies{
		ConfigProvider: s.config,
		AppRouter:      s.AppRouter,
		SymbolProvider: s.symbols,
	}

	container, err := bootstrap.ConfigAndContainer(context.Background(), deps, s.options...)
	if err != nil {
		return fmt.Errorf("failed to initialise container: %w", err)
	}
	s.Container = container
	return nil
}

// WithInterpreterProvider sets the interpreter provider for dev-i mode.
//
// This is required when running in RunModeDevInterpreted. The provider
// creates pooled interpreters with pre-loaded symbols for efficient JIT
// compilation.
//
// Takes provider (templater_domain.InterpreterProviderPort) which provides the
// interpreter pool and symbol management.
//
// Example:
// import pikointerp "piko.sh/piko/wdk/interp/interp_provider_piko"
// server := piko.New()
// server.WithInterpreterProvider(pikointerp.NewProvider())
// server.Run(actions, piko.RunModeDevInterpreted)
func (s *SSRServer) WithInterpreterProvider(provider templater_domain.InterpreterProviderPort) {
	s.interpreterProvider = provider
}

// WithSymbols sets additional exported symbols for the interpreter.
//
// These symbols will be registered with the interpreter provider when running
// in dev-i mode, making custom types and functions available to interpreted
// template code.
//
// Takes symbols (templater_domain.SymbolExports) which provides the symbols to
// expose.
func (s *SSRServer) WithSymbols(symbols templater_domain.SymbolExports) {
	s.symbols = symbols
}

// Generate produces a Piko build using the two-phase bootstrap process.
//
// For dev-i mode, this also builds the daemon infrastructure to set up routes
// on the AppRouter, allowing the server to serve requests immediately after
// generation completes. The daemon is not started; use Run to actually start
// serving HTTP requests.
//
// Takes runMode (string) which specifies the execution mode for the build.
//
// Returns error when configuration bootstrap fails, interpreter provider is
// missing for dev-i mode, or the build process fails.
func (s *SSRServer) Generate(ctx context.Context, runMode string) error {
	deps := &bootstrap.Dependencies{
		ConfigProvider: s.config,
		AppRouter:      s.AppRouter,
	}

	if runMode == RunModeDevInterpreted {
		if err := s.prepareInterpretedDeps(deps); err != nil {
			return fmt.Errorf("preparing interpreted dependencies: %w", err)
		}
	}

	container, err := s.ensureContainer(ctx, deps, runMode)
	if err != nil {
		return err
	}

	if runMode == RunModeDevInterpreted {
		if s.daemon != nil {
			stopCtx, stopCancel := context.WithTimeoutCause(ctx, 10*time.Second,
				errors.New("stopping previous daemon exceeded 10s timeout"))
			_ = s.daemon.Stop(stopCtx)
			stopCancel()
			s.daemon = nil

			if coordService, err := container.GetCoordinatorService(); err == nil {
				_ = coordService.Invalidate(ctx)
			}

			if s.PreBuildHook != nil {
				s.PreBuildHook()
			}
		}

		daemon, daemonErr := bootstrap.Daemon(ctx, runMode, container, deps)
		if daemonErr != nil {
			return fmt.Errorf("failed to build daemon for dev-i mode: %w", daemonErr)
		}
		s.daemon = daemon
		return nil
	}

	return bootstrap.BuildProject(ctx, runMode, container)
}

// Run is the high-level API for starting the Piko server.
//
// Takes runMode (string) which specifies the execution mode (prod, dev, or
// dev-interpreted).
//
// Returns error when configuration bootstrap fails, global setup fails,
// lifecycle components fail to start, daemon bootstrap fails, or the daemon
// exits with an unexpected error.
//
// Spawns a goroutine to listen for shutdown signals. The goroutine runs until
// a shutdown signal is received.
func (s *SSRServer) Run(runMode string) error {
	ctx := context.Background()
	ctx, l := logger_domain.From(ctx, log)
	l = l.With(logger_domain.String(logger_domain.FieldStrContext, "piko.Run"))

	deps, err := s.buildDependencies(runMode)
	if err != nil {
		return err
	}

	container, err := bootstrap.ConfigAndContainer(ctx, deps, s.options...)
	if err != nil {
		return fmt.Errorf("failed during configuration bootstrap: %w", err)
	}
	s.Container = container

	if err := s.installCrashOutput(ctx, container); err != nil {
		return err
	}

	isDevMode := runMode == RunModeDev || runMode == RunModeDevInterpreted
	if isDevMode {
		if container.IsDevHotreloadEnabled() {
			container.AddFrontendModule(daemon_frontend.ModuleDev, nil)
		}
		if container.IsDevWidgetEnabled() {
			container.AddExternalComponents(components.Dev()...)
			daemon_frontend.SetDevWidgetHTML("<piko-dev-widget></piko-dev-widget>")
		}
	}

	if err := performGlobalSetup(ctx, container.GetConfigProvider(), container, isDevMode); err != nil {
		return fmt.Errorf("performing global setup: %w", err)
	}

	go shutdown.ListenAndShutdown(shutdown.DefaultTimeout)
	defer func() {
		if r := recover(); r != nil {
			l.Error(fmt.Sprintf("Panic in main: %v", r))
			shutdown.Cleanup(ctx, shutdown.DefaultTimeout)
			os.Exit(1)
		}
	}()

	s.registerLifecycleHealthProbes(container)
	s.registerLifecycleShutdownHooks(ctx)

	if err := s.startLifecycleComponents(ctx); err != nil {
		l.Error("Lifecycle component startup failed, running cleanup", logger_domain.Error(err))
		shutdown.Cleanup(ctx, shutdown.DefaultTimeout)
		return fmt.Errorf("failed to start lifecycle components: %w", err)
	}

	return s.startAndRunDaemon(ctx, runMode, container, deps)
}

// Close performs graceful cleanup of all services created during Generate().
// Call this after Generate() when you do not intend to call Run(), to ensure
// background goroutines (orchestrator, coordinator, event bus, cache, etc.)
// are properly terminated.
//
// Safe to call multiple times.
func (*SSRServer) Close() {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second,
		errors.New("SSR server Close exceeded 10s timeout"))
	defer cancel()
	shutdown.Cleanup(ctx, 10*time.Second)
	shutdown.Reset()
}

// Stop initiates a graceful shutdown of the running daemon service.
//
// Returns error when the shutdown fails or times out.
func (s *SSRServer) Stop() error {
	if s.daemon == nil {
		return nil
	}
	ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second,
		errors.New("SSR server Stop exceeded 10s timeout"))
	defer cancel()
	return s.daemon.Stop(ctx)
}

// GetHandler returns the HTTP handler for serving requests, or nil if the
// daemon has not been built yet.
//
// Use it to test without starting the full server. The daemon must be built
// first via Generate or Run.
//
// Returns http.Handler which processes incoming HTTP requests.
func (s *SSRServer) GetHandler() http.Handler {
	if s.daemon == nil {
		return nil
	}
	return s.daemon.GetHandler()
}

// prepareInterpretedDeps configures interpreter-specific dependencies for
// dev-i mode.
//
// Takes deps (*bootstrap.Dependencies) which receives the interpreter pool
// and symbol provider.
//
// Returns error when no interpreter provider has been set.
func (s *SSRServer) prepareInterpretedDeps(deps *bootstrap.Dependencies) error {
	if s.interpreterProvider == nil {
		return errors.New("dev-i mode requires an interpreter provider; call WithInterpreterProvider() first")
	}
	if s.symbols != nil {
		s.interpreterProvider.RegisterSymbols(s.symbols)
	}
	deps.SymbolProvider = s.interpreterProvider.NewSymbolProvider()
	deps.InterpreterPool = s.interpreterProvider.NewInterpreterPool(deps.SymbolProvider.(templater_domain.SymbolProviderPort))
	return nil
}

// ensureContainer creates and initialises the bootstrap Container if one has
// not already been assigned. Dev mode front-end modules and global setup are
// applied on first creation.
//
// Takes deps (*bootstrap.Dependencies) which provides the application
// dependencies.
// Takes runMode (string) which specifies the execution mode (prod, dev, or
// dev-interpreted).
//
// Returns *bootstrap.Container which is the initialised container.
// Returns error when configuration bootstrap or global setup fails.
func (s *SSRServer) ensureContainer(ctx context.Context, deps *bootstrap.Dependencies, runMode string) (*bootstrap.Container, error) {
	if s.Container != nil {
		return s.Container, nil
	}

	container, err := bootstrap.ConfigAndContainer(ctx, deps, s.options...)
	if err != nil {
		return nil, fmt.Errorf("failed during configuration bootstrap for generate command: %w", err)
	}
	s.Container = container

	isDevMode := runMode == RunModeDev || runMode == RunModeDevInterpreted
	if isDevMode {
		if container.IsDevHotreloadEnabled() {
			container.AddFrontendModule(daemon_frontend.ModuleDev, nil)
		}
		if container.IsDevWidgetEnabled() {
			container.AddExternalComponents(components.Dev()...)
			daemon_frontend.SetDevWidgetHTML("<piko-dev-widget></piko-dev-widget>")
		}
	}

	if err := performGlobalSetup(ctx, container.GetConfigProvider(), container, isDevMode); err != nil {
		return nil, fmt.Errorf("performing global setup: %w", err)
	}

	return container, nil
}

// installCrashOutput wires the runtime crash mirror into the running
// process.
//
// Called from Run as early as possible, before any application goroutines
// are spawned, so the runtime can mirror unrecovered panics and fatal
// errors to the configured file even when stderr is lost (container kill,
// etc.). The closeFn returned by InstallCrashOutput is registered with the
// shutdown pipeline so it runs at graceful shutdown alongside other
// lifecycle cleanup.
//
// Takes container (*bootstrap.Container) which provides the configured
// crash-output path and traceback level.
//
// Returns error wrapping the bootstrap failure when the configured
// traceback level is invalid; file-open failures are best-effort and do
// not propagate.
func (*SSRServer) installCrashOutput(ctx context.Context, container *bootstrap.Container) error {
	crashOutputClose, err := bootstrap.InstallCrashOutput(ctx, container)
	if err != nil {
		return fmt.Errorf("installing crash output: %w", err)
	}
	if crashOutputClose != nil {
		shutdown.Register(ctx, "CrashOutput", func(_ context.Context) error {
			crashOutputClose()
			return nil
		})
	}
	return nil
}

// buildDependencies creates the bootstrap dependencies struct from the
// SSRServer's configured providers.
//
// Takes runMode (string) which specifies the server execution mode
// (RunModeDev, RunModeDevInterpreted, or RunModeProd).
//
// Returns *bootstrap.Dependencies which contains the configured dependencies.
// Returns error when dev-i mode is requested but no interpreter provider is
// set.
func (s *SSRServer) buildDependencies(runMode string) (*bootstrap.Dependencies, error) {
	deps := &bootstrap.Dependencies{
		ConfigProvider: s.config,
		AppRouter:      s.AppRouter,
	}

	if runMode == RunModeDevInterpreted {
		if s.interpreterProvider == nil {
			return nil, errors.New("dev-i mode requires an interpreter provider; call WithInterpreterProvider() first")
		}
		if s.symbols != nil {
			s.interpreterProvider.RegisterSymbols(s.symbols)
		}
		symbolProvider := s.interpreterProvider.NewSymbolProvider()
		deps.SymbolProvider = symbolProvider
		deps.InterpreterPool = s.interpreterProvider.NewInterpreterPool(symbolProvider)
	}

	return deps, nil
}

// startAndRunDaemon bootstraps and runs the daemon service.
//
// Takes runMode (string) which specifies the execution mode (prod, dev, or
// dev-interpreted).
// Takes container (*bootstrap.Container) which provides the application
// container.
// Takes deps (*bootstrap.Dependencies) which provides the application
// dependencies.
//
// Returns error when bootstrapping fails, the run mode is invalid, or the
// daemon exits unexpectedly.
func (s *SSRServer) startAndRunDaemon(ctx context.Context, runMode string, container *bootstrap.Container, deps *bootstrap.Dependencies) error {
	ctx, l := logger_domain.From(ctx, log)

	daemonConfig := bootstrap.NewDaemonConfig(&s.config.ServerConfig)
	daemonConfig.IAmACatPerson = container.IsIAmACatPerson()
	bannerInfo := daemon_domain.BuildStartupBannerInfo(daemonConfig, runMode, Version)
	bannerEnabled := container.IsStartupBannerEnabled()

	if profilingConfig := container.GetProfilingConfig(); profilingConfig != nil {
		addr := profiler.ServerAddress(*profilingConfig)
		bannerInfo.ProfilingURL = "http://" + addr + profiler.BasePath + "/debug/pprof/"
		bannerInfo.ProfilingExposed = profilingConfig.BindAddress == "0.0.0.0"
	}

	var healthAddress atomic.Value

	container.SetOnHealthBound(func(address string) {
		healthAddress.Store(address)
	})

	container.SetOnServerBound(func(address string) {
		bannerInfo.ServerURL = fmt.Sprintf("http://localhost%s", address)
		bannerInfo.AutoPort = false

		if resolved, ok := healthAddress.Load().(string); ok && resolved != "" {
			bannerInfo.HealthProbeURL = fmt.Sprintf("http://%s", resolved)
			bannerInfo.LivePath = daemonConfig.HealthLivePath
			bannerInfo.ReadyPath = daemonConfig.HealthReadyPath
		}

		if monitoringService := container.GetMonitoringService(); monitoringService != nil {
			bannerInfo.MonitoringURL = monitoringService.Address()
			bannerInfo.MonitoringExposed = strings.HasPrefix(monitoringService.Address(), "0.0.0.0")
		}

		daemon_domain.PrintStartupBanner(ctx, bannerEnabled, bannerInfo)
	})

	daemonService, err := bootstrap.Daemon(ctx, runMode, container, deps)
	if err != nil {
		return fmt.Errorf("failed to bootstrap daemon: %w", err)
	}
	s.daemon = daemonService

	return s.runDaemonForMode(ctx, runMode, l)
}

// runDaemonForMode selects the appropriate daemon run method based on the
// mode, executes it, and interprets the result. A normal server-closed error
// is treated as a graceful stop.
//
// Takes runMode (string) which selects between prod and dev execution paths.
// Takes l (logger_domain.Logger) which is the logger for reporting the outcome.
//
// Returns error when the run mode is invalid or the daemon exits unexpectedly.
func (s *SSRServer) runDaemonForMode(ctx context.Context, runMode string, l logger_domain.Logger) error {
	var runErr error
	switch runMode {
	case RunModeProd:
		runErr = s.daemon.RunProd(ctx)
	case RunModeDev, RunModeDevInterpreted:
		runErr = s.daemon.RunDev(ctx)
	default:
		return fmt.Errorf("invalid run mode: %s", runMode)
	}

	if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
		l.Error("Daemon exited with an unexpected error", logger_domain.Error(runErr))
		return runErr
	}

	l.Notice("Daemon stopped gracefully.")
	return nil
}

// startLifecycleComponents calls OnStart on all registered lifecycle
// components in the order they were added.
//
// Returns error when any component fails to start within the 30-second timeout.
func (s *SSRServer) startLifecycleComponents(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	if len(s.lifecycleComponents) == 0 {
		return nil
	}

	l.Internal("Starting lifecycle components...", logger_domain.Int("count", len(s.lifecycleComponents)))

	for _, component := range s.lifecycleComponents {
		componentLog := l.With(logger_domain.String("component", component.Name()))
		componentLog.Internal("Starting lifecycle component...")

		timeout := defaultComponentStartTimeout
		if t, ok := component.(LifecycleStartTimeout); ok {
			timeout = t.StartTimeout()
		}

		startCtx, cancel := context.WithTimeoutCause(ctx, timeout,
			fmt.Errorf("component start exceeded %s timeout", timeout))
		err := component.OnStart(startCtx)
		cancel()

		if err != nil {
			componentLog.Error("Failed to start lifecycle component", logger_domain.Error(err))
			return fmt.Errorf("lifecycle component %q failed to start: %w", component.Name(), err)
		}

		componentLog.Internal("Lifecycle component started successfully")
	}

	l.Internal("All lifecycle components started successfully")
	return nil
}

// registerLifecycleShutdownHooks registers shutdown hooks with the shutdown
// system. Components are stopped in reverse order (LIFO), so the last
// component added is stopped first.
func (s *SSRServer) registerLifecycleShutdownHooks(ctx context.Context) {
	if len(s.lifecycleComponents) == 0 {
		return
	}

	shutdownCtx := context.WithoutCancel(ctx)

	for i := len(s.lifecycleComponents) - 1; i >= 0; i-- {
		component := s.lifecycleComponents[i]

		shutdown.Register(shutdownCtx, fmt.Sprintf("LifecycleComponent:%s", component.Name()), func(ctx context.Context) error {
			ctx, componentLog := logger_domain.From(ctx, log)
			componentLog = componentLog.With(logger_domain.String("component", component.Name()))
			componentLog.Internal("Stopping lifecycle component...")

			if err := component.OnStop(ctx); err != nil {
				componentLog.Error("Error stopping lifecycle component", logger_domain.Error(err))
				return err
			}

			componentLog.Internal("Lifecycle component stopped successfully")
			return nil
		})
	}

	_, sl := logger_domain.From(s.Container.GetAppContext(), log)
	sl.Internal("Registered shutdown hooks for lifecycle components", logger_domain.Int("count", len(s.lifecycleComponents)))
}

// registerLifecycleHealthProbes registers health probes for lifecycle
// components. If a component implements HealthProbe, it adds the component to
// the health monitoring system.
//
// Takes container (*bootstrap.Container) which provides the health probe
// registration system.
func (s *SSRServer) registerLifecycleHealthProbes(container *bootstrap.Container) {
	if len(s.lifecycleComponents) == 0 {
		return
	}

	_, l := logger_domain.From(s.Container.GetAppContext(), log)
	probeCount := 0
	for _, component := range s.lifecycleComponents {
		if probe, ok := component.(LifecycleHealthProbe); ok {
			adapter := &lifecycleProbeAdapter{
				component: component,
				probe:     probe,
			}

			container.AddCustomHealthProbe(adapter)
			probeCount++

			l.Internal("Registered health probe for lifecycle component", logger_domain.String("component", component.Name()))
		}
	}

	if probeCount > 0 {
		l.Internal("Registered health probes for lifecycle components", logger_domain.Int("count", probeCount))
	}
}

// Option is a functional option used to customise the Piko server during
// initialisation.
type Option = bootstrap.Option

// Container is the Dependency Injection container that holds all application
// services. It is exposed for advanced use cases where direct access to a
// service is needed.
type Container = bootstrap.Container

// ConfigResolver enables integration with secret managers such as AWS Secrets
// Manager or HashiCorp Vault for custom configuration value resolution.
type ConfigResolver = config_domain.Resolver

// ComponentDefinition describes a PKC component that can be used in templates.
//
// Tag names must contain a hyphen (per Web Components specification) and must
// not shadow HTML element names or use reserved prefixes (piko:, pml-).
type ComponentDefinition = component_dto.ComponentDefinition

// ServerConfig is the root configuration object for the Piko server.
// Pass it to WithServerConfigDefaults to set defaults programmatically.
type ServerConfig = config.ServerConfig

// PathsConfig holds file system and URL path settings for the project.
type PathsConfig = config.PathsConfig

// NetworkConfig holds the network configuration for the server, including
// port, public domain, HTTPS, and auto-port selection.
type NetworkConfig = config.NetworkConfig

// BuildModeConfig holds settings for the build and watch process.
type BuildModeConfig = config.BuildModeConfig

// DatabaseConfig holds database connection configuration.
type DatabaseConfig = config.DatabaseConfig

// PostgresDatabaseConfig holds PostgreSQL connection settings.
type PostgresDatabaseConfig = config.PostgresDatabaseConfig

// D1DatabaseConfig holds Cloudflare D1-specific settings.
type D1DatabaseConfig = config.D1DatabaseConfig

// OtlpConfig holds configuration for OpenTelemetry Protocol exporting.
type OtlpConfig = config.OtlpConfig

// OtlpTLSConfig holds TLS settings for the OTLP exporter connection.
type OtlpTLSConfig = config.OtlpTLSConfig

// LoggerConfig holds settings for the application logger.
type LoggerConfig = logger_dto.Config

// HealthProbeConfig holds configuration for the health check server.
type HealthProbeConfig = config.HealthProbeConfig

// StorageConfig holds settings for the storage service.
type StorageConfig = config.StorageConfig

// StoragePresignConfig configures presigned URL support for storage operations.
type StoragePresignConfig = config.StoragePresignConfig

// SecurityConfig holds security-related settings including headers, rate
// limiting, cookies, and encryption.
type SecurityConfig = config.SecurityConfig

// SecurityHeadersConfig configures HTTP security headers following OWASP
// best practices.
type SecurityHeadersConfig = config.SecurityHeadersConfig

// CookieSecurityConfig holds secure defaults for HTTP cookies.
type CookieSecurityConfig = config.CookieSecurityConfig

// RateLimitConfig configures request rate limiting with trusted proxy support.
type RateLimitConfig = config.RateLimitConfig

// RateLimitTierConfig configures rate limits for a tier (global or actions).
type RateLimitTierConfig = config.RateLimitTierConfig

// SandboxConfig configures filesystem sandboxing for Piko internals.
type SandboxConfig = config.SandboxConfig

// ReportingConfig configures the Reporting-Endpoints HTTP header.
type ReportingConfig = config.ReportingConfig

// AWSKMSConfig holds settings for AWS Key Management Service.
type AWSKMSConfig = config.AWSKMSConfig

// GCPKMSConfig holds settings for Google Cloud Key Management Service.
type GCPKMSConfig = config.GCPKMSConfig

// SEOConfig holds settings for SEO artefact generation including sitemap.xml
// and robots.txt. Use WithSEO to provide this configuration.
type SEOConfig = config.SEOConfig

// SitemapConfig holds settings for sitemap.xml generation.
type SitemapConfig = config.SitemapConfig

// SitemapEntryDefaults provides default values for sitemap entries.
type SitemapEntryDefaults = config.SitemapEntryDefaults

// SitemapChunkConfig defines a named sitemap chunk with its own sources.
type SitemapChunkConfig = config.SitemapChunkConfig

// RobotsConfig holds settings for robots.txt generation.
type RobotsConfig = config.RobotsConfig

// RobotsRuleGroup holds a set of rules for one or more user agents.
type RobotsRuleGroup = config.RobotsRuleGroup

// AssetsConfig holds settings for image and video asset processing including
// transformation profiles, screen breakpoints, and default densities.
type AssetsConfig = config.AssetsConfig

// ImageAssetsConfig holds image-specific asset configuration.
type ImageAssetsConfig = config.ImageAssetsConfig

// VideoAssetsConfig holds video-specific asset configuration.
type VideoAssetsConfig = config.VideoAssetsConfig

// AssetTransformationStep represents a single transformation to apply to an
// asset.
type AssetTransformationStep = config.AssetTransformationStep

// WebsiteConfig defines the user-facing properties of the website being
// served, including theme, favicons, fonts, and i18n settings.
type WebsiteConfig = config.WebsiteConfig

// FaviconDefinition describes a single favicon link element for a website.
type FaviconDefinition = config.FaviconDefinition

// FontDefinition defines a font to be loaded by the website.
type FontDefinition = config.FontDefinition

// Translation represents a translatable string returned by r.T() and r.LT().
// It supports variables and pluralisation, and implements fmt.Stringer.
type Translation = i18n_domain.Translation

// I18nConfig contains the internationalisation configuration for a website. It
// defines the supported locales, default locale, and URL strategy for i18n
// routing.
//
// Use I18nConfig when calling GenerateLocaleHead to generate SEO metadata for
// your pages.
type I18nConfig struct {
	// DefaultLocale is the fallback locale used when a translation is missing or
	// the user's preferred locale is not supported. Example: "en".
	DefaultLocale string `json:"defaultLocale"`

	// Strategy defines how locale information is represented
	// in URLs.
	//
	// Supported values:
	//   - "prefix": All routes are prefixed with the locale (e.g., /en/about,
	//     /fr/about)
	//   - "prefix_except_default": Only non-default locales get a prefix (e.g.,
	//     /about, /fr/about)
	//   - "domain": Different domains for different locales (e.g., example.com,
	//     example.fr)
	Strategy string `json:"strategy"`

	// Locales is the list of all supported locales for the website, such as
	// []string{"en", "fr", "de", "es"}. An empty slice disables i18n features.
	Locales []string `json:"locales"`
}

// New creates and sets up a new SSRServer instance.
//
// Takes opts (...bootstrap.Option) which configures the server behaviour.
//
// Returns *SSRServer which is ready for use with default router and config.
func New(opts ...bootstrap.Option) *SSRServer {
	return &SSRServer{
		AppRouter: chi.NewRouter(),
		config:    config.NewConfigProvider(),
		options:   opts,
	}
}

// GenerateLocaleHead generates internationalisation SEO metadata for a page.
//
// Designed to be called from within a component's Render function to populate
// the Metadata.Language, Metadata.CanonicalUrl, and Metadata.AlternateLinks
// fields.
//
// Takes r (*RequestData) which provides the current request data.
// Takes i18nConfig (I18nConfig) which defines locales and URL strategy.
// Takes pagePath (string) which is the page's URL path (e.g. "/about").
// Takes supportedLocalesOverride ([]string) which optionally specifies locales
// to use instead of the full config. Pass nil or empty slice to use all
// locales from the config.
//
// Returns locale (string) which is the current request's locale from r.Locale.
// Returns canonicalURL (string) which is the canonical URL for this page.
// Returns alternateLinks ([]map[string]string) which contains hreflang
// alternate links for SEO.
func GenerateLocaleHead(
	r *RequestData,
	i18nConfig I18nConfig,
	pagePath string,
	supportedLocalesOverride []string,
) (locale string, canonicalURL string, alternateLinks []map[string]string) {
	return runtime.GenerateLocaleHead(r, runtime.I18nConfig{
		DefaultLocale: i18nConfig.DefaultLocale,
		Strategy:      i18nConfig.Strategy,
		Locales:       i18nConfig.Locales,
	}, pagePath, supportedLocalesOverride)
}

// RunHeadless bootstraps Piko's global services for headless use
// cases such as CLI tools, import scripts, background workers, and
// microservices.
//
// This initialises the framework's service container (image
// processing, storage, cache, persistence) without starting an HTTP
// server, loading configuration files, or setting up frontend assets.
// Call this before any code that uses framework services via global
// access functions (e.g., media.GetImageDimensions,
// storage.GetDefaultService).
//
// Takes opts (...Option) which configure providers, identical to New.
//
// Returns *bootstrap.Container which provides direct service access
// if needed.
// Returns error when provider configuration is invalid.
//
// Example:
//
//	container, err := piko.RunHeadless(
//		persistence.WithDriver(sqlite.New(sqlite.Config{})),
//		piko.WithImageProvider("imaging", imagingProvider),
//		piko.WithStorageProvider("media", diskProvider),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Framework services are now available globally
//	width, height, err := media.GetImageDimensions(ctx, reader)
func RunHeadless(opts ...Option) (*bootstrap.Container, error) {
	return bootstrap.InitialiseHeadless(opts...)
}

// InitialiseForTesting initialises Piko's global services with minimal
// dependencies suitable for unit and integration tests.
//
// Creates a fully mocked Piko environment with:
//   - In-memory cache provider (no Redis/external cache required)
//   - In-memory storage provider (no S3/disk writes)
//   - In-memory registry (no metadata.db SQLite file created)
//   - Mock email provider (no actual emails sent)
//   - Mock email template service
//   - Mock image transformer (no actual image processing)
//   - In-memory event bus
//
// This keeps tests:
//   - Fast (no I/O operations)
//   - Isolated (no shared state between tests)
//   - Portable (no external dependencies)
//   - Clean (no persistent state or files created)
//
// Returns *bootstrap.Container which can be used to access services or for
// cleanup.
//
// Example usage:
//
//	func TestMain(m *testing.M) {
//	    piko.InitialiseForTesting()
//	    os.Exit(m.Run())
//	}
//
// This initialisation is lightweight and does not start the full daemon.
// It provides a complete mocked Piko environment suitable for
// testing. Tests should defer cleanup of any resources if needed.
func InitialiseForTesting() *bootstrap.Container {
	mockProvider := email_provider_mock.NewMockEmailProvider()

	return bootstrap.InitialiseForTesting(mockProvider, email_dto.EmailNameDefault)
}

// performGlobalSetup sets up global services and registers custom modules.
//
// Takes configProvider (*config.Provider) which provides server settings.
// Takes container (*bootstrap.Container) which holds custom frontend modules.
//
// Returns error when logger setup fails, directory creation fails, or module
// registration fails.
func performGlobalSetup(ctx context.Context, configProvider *config.Provider, container *bootstrap.Container, devMode bool) error {
	ctx, l := logger_domain.From(ctx, log)

	if err := logger.Apply(bootstrap.NewOtelSetupConfig(&configProvider.ServerConfig)); err != nil {
		return fmt.Errorf("failed to apply logger configuration: %w", err)
	}
	if loggerShutdown := logger.GetShutdownFunc(); loggerShutdown != nil {
		shutdown.Register(ctx, "Logger", loggerShutdown)
	}

	baseDir := "."
	if configProvider.ServerConfig.Paths.BaseDir != nil {
		baseDir = *configProvider.ServerConfig.Paths.BaseDir
	}
	if err := ensurePikoInternalDir(baseDir, config.PikoInternalPath); err != nil {
		return fmt.Errorf("ensuring piko internal directory: %w", err)
	}

	if err := daemon_frontend.InitAssetStore(ctx); err != nil {
		return fmt.Errorf("failed to initialise embedded asset store: %w", err)
	}

	if container != nil {
		if err := setupFrontendModules(ctx, l, container, devMode); err != nil {
			return err
		}
	}

	return nil
}

// setupFrontendModules configures frontend SRI, custom modules, and module HTML
// from the container.
//
// Takes l (logger_domain.Logger) which logs diagnostic messages about module setup.
// Takes container (*bootstrap.Container) which provides frontend module configuration.
//
// Returns error when a custom frontend module fails to register.
func setupFrontendModules(ctx context.Context, l logger_domain.Logger, container *bootstrap.Container, devMode bool) error {
	if devMode {
		daemon_frontend.SetSRIEnabled(false)
	} else {
		daemon_frontend.SetSRIEnabled(container.IsSRIEnabled())
	}
	for name, module := range container.GetCustomFrontendModules() {
		if err := daemon_frontend.RegisterCustomModule(ctx, name, module.Content, module.ETag); err != nil {
			return fmt.Errorf("failed to register custom frontend module %q: %w", name, err)
		}
	}

	preloadHTML, scriptHTML, configHTML := daemon_frontend.GenerateModuleHTML(
		container.GetFrontendModules(),
		container.GetCustomFrontendModules(),
	)
	daemon_frontend.SetModuleHTML(preloadHTML, scriptHTML, configHTML)

	if preloadHTML != "" || scriptHTML != "" {
		l.Internal("Frontend modules configured",
			logger_domain.Int("builtin_count", len(container.GetFrontendModules())),
			logger_domain.Int("custom_count", len(container.GetCustomFrontendModules())))
	}

	return nil
}

// ensurePikoInternalDir creates the .piko internal directory using a sandboxed
// filesystem operation.
//
// Takes baseDir (string) which is the project root directory.
// Takes internalPath (string) which is the relative path for the
// internal directory.
//
// Returns error when sandbox creation or directory creation fails.
func ensurePikoInternalDir(baseDir, internalPath string) error {
	factory, factoryErr := safedisk.NewCLIFactory(baseDir)
	if factoryErr != nil {
		return fmt.Errorf("could not create sandbox factory for .piko directory: %w", factoryErr)
	}

	sandbox, err := factory.Create("piko-internal-dir", baseDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("could not create sandbox for .piko directory: %w", err)
	}
	defer func() { _ = sandbox.Close() }()

	if err := sandbox.MkdirAll(internalPath, dirPermOwnerRWX); err != nil {
		return fmt.Errorf("could not create .piko directory: %w", err)
	}
	return nil
}
