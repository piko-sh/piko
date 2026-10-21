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

// This file contains orchestrator and capability service related container methods.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/compiler/compiler_adapters"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/esbuild/compat"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	orchestrator_querier_postgres "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres"
	orchestrator_querier_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/safedisk"
)

const (
	// buildMaxRetryBackoff caps a failed task's retry delay during a build.
	buildMaxRetryBackoff = 2 * time.Second

	// schemaCheckTimeout bounds the startup probe of the orchestrator database's schema.
	schemaCheckTimeout = 10 * time.Second

	// gcHintsBatchSize is the number of GC hints each blob GC hints task processes.
	gcHintsBatchSize = 100

	// gcHintsRescheduleSeconds is the delay, in seconds, before the blob GC hints task runs
	// again.
	gcHintsRescheduleSeconds = 30

	// gcHintsInitialDelay is how long after startup the first blob GC hints task runs.
	gcHintsInitialDelay = 10 * time.Second
)

var (
	// errOrchestratorTopology is returned when the orchestrator's task store and event bus
	// cannot work together.
	errOrchestratorTopology = errors.New("orchestrator topology is not supported")
)

// GetCapabilityService returns the capability detection service, creating it if
// necessary.
//
// Returns capabilities.Service which is the capability detection service.
// Returns error when the service could not be created.
func (c *Container) GetCapabilityService() (capabilities.Service, error) {
	c.capabilityOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.capabilityServiceOverride != nil {
			l.Internal("Using provided CapabilityService override.")
			c.capabilityService = c.capabilityServiceOverride
			return
		}
		c.createDefaultCapabilityService()
	})
	return c.capabilityService, c.capabilityErr
}

// createDefaultCapabilityService sets up the default capability service with image
// transformation, video transcoding, and optional component compilation support.
func (c *Container) createDefaultCapabilityService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default CapabilityService...")
	serverConfig := c.serverConfig

	imageService, err := c.GetImageService()
	if err != nil {
		c.capabilityErr = fmt.Errorf("failed to get image service: %w", err)
		return
	}

	videoService, err := c.GetVideoService()
	if err != nil {
		c.capabilityErr = fmt.Errorf("failed to get video service: %w", err)
		return
	}

	componentsDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), deref(serverConfig.Paths.ComponentsSourceDir, "components"))
	localDirExists := true
	if _, statErr := os.Stat(componentsDir); os.IsNotExist(statErr) {
		localDirExists = false
	}

	if !localDirExists && !c.hasExternalModuleComponents() {
		l.Internal("No components directory or external module components, skipping compiler setup",
			logger_domain.String("path", componentsDir))
		c.capabilityService, c.capabilityErr = capabilities.NewServiceWithBuiltins(
			capabilities.WithImageProvider(imageService),
			capabilities.WithVideoProvider(videoService),
		)
		return
	}

	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	c.capabilityService, c.capabilityErr = c.createCapabilityWithCompiler(imageService, videoService, baseDir, componentsDir, localDirExists)
}

// createCapabilityWithCompiler sets up the compiler and creates the capability service
// with compilation support.
//
// Takes imageService (any) which is the image service to register.
// Takes videoService (any) which is the video service to register.
// Takes baseDir (string) which is the project root directory for CSS import resolution.
// Takes componentsDir (string) which is the path to the local components directory.
// Takes localDirExists (bool) which indicates whether the directory exists.
//
// Returns capabilities.Service which is the configured capability service.
// Returns error when resolver or sandbox creation fails.
func (c *Container) createCapabilityWithCompiler(
	imageService image_domain.Service,
	videoService video_domain.Service,
	baseDir string,
	componentsDir string,
	localDirExists bool,
) (capabilities.Service, error) {
	resolver, err := c.GetResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to get resolver for compiler: %w", err)
	}
	moduleName := resolver.GetModuleName()

	var inputReader compiler_domain.InputReaderPort
	if localDirExists {
		sourceSandbox, sandboxErr := c.createSandbox("capability-source", componentsDir, safedisk.ModeReadOnly)
		if sandboxErr != nil {
			return nil, sandboxErr
		}
		inputReader = compiler_adapters.NewDiskInputReader(sourceSandbox)
	} else {
		inputReader = compiler_adapters.NewMemoryInputReader()
	}

	compilerOpts := []compiler_domain.OrchestratorOption{
		compiler_domain.WithOrchestratorModuleName(moduleName),
	}

	if cssPreProcessor, preErr := c.createCSSPreProcessor(resolver, baseDir); preErr == nil {
		compilerOpts = append(compilerOpts, compiler_domain.WithOrchestratorCSSPreProcessor(cssPreProcessor))
	}

	compiler := compiler_domain.NewCompilerOrchestrator(
		inputReader,
		[]compiler_domain.TransformationPort{},
		compilerOpts...,
	)

	return capabilities.NewServiceWithBuiltins(
		capabilities.WithCompiler(compiler),
		capabilities.WithImageProvider(imageService),
		capabilities.WithVideoProvider(videoService),
	)
}

// createCSSPreProcessor creates a CSS pre-processor that resolves @import statements in
// component style blocks.
//
// Takes resolver (resolver_domain.ResolverPort) which resolves CSS import paths including
// @/ aliases.
// Takes baseDir (string) which is the root directory for the sandbox.
//
// Returns compiler_domain.CSSPreProcessorPort which resolves CSS imports.
// Returns error when sandbox creation fails.
func (c *Container) createCSSPreProcessor(resolver resolver_domain.ResolverPort, baseDir string) (compiler_domain.CSSPreProcessorPort, error) {
	sourceSandbox, err := c.createSandbox("compiler-css-source", baseDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating CSS pre-processor sandbox: %w", err)
	}
	fsReader := generator_adapters.NewFSReader(sourceSandbox)
	processor := cssinliner.NewProcessor(cssinliner.ProcessorConfig{
		Resolver: resolver,
		Loader:   esbuildconfig.LoaderLocalCSS,
		Options: &esbuildconfig.Options{
			UnsupportedCSSFeatures: compat.Nesting,
		},
		Limits: cssinliner.Limits{
			MaxDepth:      deref(c.serverConfig.Build.CSSImportMaxDepth, cssinliner.DefaultMaxImportDepth),
			MaxTotalBytes: deref(c.serverConfig.Build.CSSImportMaxBytes, cssinliner.DefaultMaxInlinedBytes),
		},
		DiagnosticCode:       "",
		ImportDiagnosticCode: "",
	})
	return compiler_adapters.NewCSSPreProcessor(processor, fsReader, resolver.GetModuleName(), baseDir), nil
}

// hasExternalModuleComponents returns true if any external component definition carries a
// ModulePath, indicating it needs module resolution.
//
// Returns bool which is true when at least one external component has a non-empty
// ModulePath.
func (c *Container) hasExternalModuleComponents() bool {
	for _, definition := range c.externalComponents {
		if definition.ModulePath != "" {
			return true
		}
	}
	return false
}

// GetOrchestratorService returns the asset orchestration service, creating it if
// necessary.
//
// Returns orchestrator_domain.OrchestratorService which provides asset orchestration
// capabilities.
// Returns error when the service could not be created.
func (c *Container) GetOrchestratorService() (orchestrator_domain.OrchestratorService, error) {
	c.orchestratorOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.orchestratorServiceOverride != nil {
			l.Internal("Using provided OrchestratorService override.")
			c.orchestratorService = c.orchestratorServiceOverride
			return
		}
		c.createDefaultOrchestratorService()
	})
	return c.orchestratorService, c.orchestratorErr
}

// GetArtefactBridge returns the ArtefactWorkflowBridge.
//
// This must be called after GetOrchestratorService() as the bridge is created during
// orchestrator initialisation.
//
// Returns *orchestrator_adapters.ArtefactWorkflowBridge which provides integration
// between artefacts and the workflow system.
func (c *Container) GetArtefactBridge() *orchestrator_adapters.ArtefactWorkflowBridge {
	_, _ = c.GetOrchestratorService()
	return c.artefactBridge
}

// startupArtefactReconciler returns what dispatches, when a server starts, the desired
// profiles an earlier run left unbuilt. It resolves the artefact bridge only when that
// check runs, so creating a lifecycle service starts no orchestrator; see
// startupReconciler.
//
// Returns lifecycle_domain.ArtefactReconciler which defers to the artefact bridge, or nil
// when this process cannot write the registry's blobs, so nothing could be rebuilt.
func (c *Container) startupArtefactReconciler() lifecycle_domain.ArtefactReconciler {
	if c.registryBlobsReadOnly() {
		return nil
	}
	return startupReconciler{container: c}
}

// startupReconciler hands a server's startup re-check to the artefact bridge, resolved when
// the check runs. It does nothing in a process running from a copy of .piko that another
// process owns, since the owner's own startup does that work.
type startupReconciler struct {
	// container provides the artefact bridge and the persistence provider.
	container *Container
}

// ReconcileArtefacts dispatches the work the given artefacts still need.
//
// Takes artefactIDs ([]string) which identifies the artefacts to re-check.
//
// Returns int which is the number of tasks dispatched.
// Returns error when an artefact could not be read or dispatched.
func (r startupReconciler) ReconcileArtefacts(ctx context.Context, artefactIDs []string) (int, error) {
	bridge := r.bridge()
	if bridge == nil {
		return 0, nil
	}
	return bridge.ReconcileArtefacts(ctx, artefactIDs)
}

// UnbuiltProfiles reports desired profiles that are not built although every profile they
// depend on is.
//
// Takes artefactIDs ([]string) which identifies the artefacts to inspect.
//
// Returns map[string][]string which maps artefact IDs to unbuilt profile names.
// Returns error when an artefact could not be read.
func (r startupReconciler) UnbuiltProfiles(ctx context.Context, artefactIDs []string) (map[string][]string, error) {
	bridge := r.bridge()
	if bridge == nil {
		return nil, nil
	}
	return bridge.UnbuiltProfiles(ctx, artefactIDs)
}

// bridge returns the artefact bridge, or nil in a secondary or when there is none.
//
// Returns *orchestrator_adapters.ArtefactWorkflowBridge which is the bridge, or nil.
func (r startupReconciler) bridge() *orchestrator_adapters.ArtefactWorkflowBridge {
	c := r.container
	if c.dbProvider != nil && c.dbProvider.IsSecondary() {
		return nil
	}
	return c.GetArtefactBridge()
}

// createDefaultOrchestratorService sets up the default orchestrator service and its
// supporting parts.
func (c *Container) createDefaultOrchestratorService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default OrchestratorService...")

	orcService, registryService, err := c.createOrchestratorServiceCore()
	if err != nil {
		c.orchestratorErr = err
		return
	}

	bridge, err := c.setupOrchestratorBridge(orcService, registryService)
	if err != nil {
		c.orchestratorErr = err
		return
	}

	c.startOrchestratorBackground(orcService)
	c.orchestratorService = orcService
	c.artefactBridge = bridge
}

// createOrchestratorServiceCore creates the orchestrator service and sets up its
// executor.
//
// Returns orchestrator.Service which is the configured orchestrator.
// Returns registry_domain.RegistryService which provides registry access.
// Returns error when any dependency cannot be fetched or set up.
func (c *Container) createOrchestratorServiceCore() (orchestrator.Service, registry_domain.RegistryService, error) {
	registryService, err := c.GetRegistryService()
	if err != nil {
		return nil, nil, fmt.Errorf("getting registry service for orchestrator: %w", err)
	}
	capabilityService, err := c.GetCapabilityService()
	if err != nil {
		return nil, nil, fmt.Errorf("getting capability service for orchestrator: %w", err)
	}
	taskStore, err := c.createOrchestratorTaskStore()
	if err != nil {
		return nil, nil, fmt.Errorf("creating orchestrator task store: %w", err)
	}

	eventBus, err := c.GetEventBus()
	if err != nil {
		return nil, nil, fmt.Errorf("getting event bus for orchestrator: %w", err)
	}

	dispatcherConfig := c.orchestratorDispatcherConfig()
	dispatcherConfig.Transport, err = resolveOrchestratorTopology(c.GetAppContext(), taskStore, eventBus)
	if err != nil {
		return nil, nil, err
	}
	orcService, err := orchestrator.NewService(c.GetAppContext(), orchestrator.Config{
		TaskStore:          taskStore,
		EventBus:           eventBus,
		Dispatcher:         &dispatcherConfig,
		WorkerCount:        0,
		SchedulerInterval:  0,
		DispatcherInterval: 0,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create orchestrator service: %w", err)
	}

	compilerExecutor := orchestrator_adapters.NewCompilerExecutor(registryService, capabilityService,
		orchestrator_adapters.WithEmptyOutputs(c.compileEmptyOutputs()))
	if err := orcService.RegisterExecutor(c.GetAppContext(), orchestrator_adapters.ExecutorNameArtefactCompiler, compilerExecutor); err != nil {
		return nil, nil, fmt.Errorf("failed to register compiler executor: %w", err)
	}

	gcExecutor := orchestrator_adapters.NewGCExecutor(registryService, orcService)
	if err := orcService.RegisterExecutor(c.GetAppContext(), orchestrator_adapters.ExecutorNameBlobGC, gcExecutor); err != nil {
		return nil, nil, fmt.Errorf("failed to register GC executor: %w", err)
	}

	return orcService, registryService, nil
}

// resolveOrchestratorTopology selects and validates task-store and event-bus locality.
//
// A distributed bus requires a shared store. A local bus with a shared store is allowed
// with a warning because replicas share work only through store sweeps.
//
// Takes taskStore (orchestrator_domain.TaskStore) which holds the tasks.
// Takes eventBus (orchestrator_domain.EventBus) which carries the orchestrator's events.
//
// Returns orchestrator_domain.TransportMode which is the transport to use.
// Returns error when the store and bus cannot work together.
func resolveOrchestratorTopology(
	ctx context.Context,
	taskStore orchestrator_domain.TaskStore,
	eventBus orchestrator_domain.EventBus,
) (orchestrator_domain.TransportMode, error) {
	_, l := logger_domain.From(ctx, log)
	localStore := false
	if local, ok := taskStore.(interface{ LocalStore() bool }); ok {
		localStore = local.LocalStore()
	}
	distributedBus := false
	if locality, ok := eventBus.(orchestrator_domain.NodeLocality); ok {
		distributedBus = !locality.NodeLocal()
	}

	storeKind, busKind := "shared database", "in-process"
	if localStore {
		storeKind = "in-process"
	}
	if distributedBus {
		busKind = "distributed"
	}

	switch {
	case distributedBus && localStore:
		return orchestrator_domain.TransportAuto, fmt.Errorf("%w: the event bus is distributed but the task store is "+
			"in-process, so other nodes would be handed tasks they cannot find; register an orchestrator database "+
			"with AddDatabase(%q) to share tasks between replicas, or use an in-process events provider",
			errOrchestratorTopology, DatabaseNameOrchestrator)
	case distributedBus:
		l.Internal("Orchestrator is clustered",
			logger_domain.String("store", storeKind),
			logger_domain.String("tasks", orchestrator_domain.TransportBus.String()),
			logger_domain.String("events", busKind))
		return orchestrator_domain.TransportBus, nil
	case !localStore:
		l.Warn("Orchestrator shares a task database but its event bus is in-process; replicas share work only "+
			"through the store's periodic sweep. Configure a distributed events provider for prompt sharing.",
			logger_domain.String("store", storeKind),
			logger_domain.String("events", busKind))
	}
	l.Internal("Orchestrator is standalone",
		logger_domain.String("store", storeKind),
		logger_domain.String("tasks", orchestrator_domain.TransportLocal.String()),
		logger_domain.String("events", busKind))
	return orchestrator_domain.TransportLocal, nil
}

// orchestratorDispatcherConfig selects dispatcher settings for the container.
//
// Builds cap retry delays and allocate workers per CPU. Processes using a read-only
// persistence copy do not recover the owner's tasks.
//
// Returns orchestrator_domain.DispatcherConfig which holds the settings.
func (c *Container) orchestratorDispatcherConfig() orchestrator_domain.DispatcherConfig {
	config := orchestrator_domain.DefaultDispatcherConfig()
	if c.dbProvider != nil && c.dbProvider.IsSecondary() {
		config.SkipRecovery = true
	}
	if c.isBuildTime {
		config.MaxRetryBackoff = buildMaxRetryBackoff
		workers := runtime.GOMAXPROCS(0)
		config.WatermillHighHandlers = max(config.WatermillHighHandlers, workers)
		config.WatermillNormalHandlers = max(config.WatermillNormalHandlers, workers)
		config.WatermillLowHandlers = max(config.WatermillLowHandlers, workers/2)
	}
	return config
}

// setupOrchestratorBridge creates and starts the artefact workflow bridge.
//
// Takes orcService (orchestrator.Service) which provides orchestration capabilities.
// Takes registryService (registry_domain.RegistryService) which provides registry access.
//
// Returns *orchestrator_adapters.ArtefactWorkflowBridge which is the configured bridge
// ready for use.
// Returns error when the bridge listener fails to start.
//
// Spawns a goroutine to wait for bridge events until the context is cancelled.
func (c *Container) setupOrchestratorBridge(orcService orchestrator.Service, registryService registry_domain.RegistryService) (*orchestrator_adapters.ArtefactWorkflowBridge, error) {
	eventBus, err := c.GetEventBus()
	if err != nil {
		return nil, fmt.Errorf("getting event bus for orchestrator bridge: %w", err)
	}
	bridge := orchestrator_adapters.NewArtefactWorkflowBridge(registryService, orcService, orcService.GetTaskDispatcher(), eventBus,
		orchestrator_adapters.WithBridgeEmptyOutputs(c.compileEmptyOutputs()))

	wait, err := bridge.StartListening(c.GetAppContext(), eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to start bridge listener: %w", err)
	}
	go wait()
	return bridge, nil
}

// compileEmptyOutputs returns the shared record of profiles compiled without output.
//
// The record is initialised once and shared across callers.
//
// Returns *orchestrator_adapters.EmptyOutputs which is the shared record.
func (c *Container) compileEmptyOutputs() *orchestrator_adapters.EmptyOutputs {
	c.emptyOutputsOnce.Do(func() {
		c.emptyOutputs = orchestrator_adapters.NewEmptyOutputs()
	})
	return c.emptyOutputs
}

// startOrchestratorBackground starts the orchestrator service in the background.
//
// Takes orcService (orchestrator.Service) which is the orchestrator to start.
//
// Concurrency: safe for concurrent use; the spawned goroutine runs until shutdown.
func (c *Container) startOrchestratorBackground(orcService orchestrator.Service) {
	go orcService.Run(c.GetAppContext())
	shutdown.Register(c.GetAppContext(), "Orchestrator", func(_ context.Context) error { orcService.Stop(); return nil })
}

// ScheduleGCTasks seeds the GC task queue with the first hint processing task. The
// deduplication key prevents duplicate scheduling if the system restarts.
//
// Safe to call multiple times. An orchestrator that is already shutting down (for example
// when the daemon stops during an interpreted-mode rebuild) refuses the task, which is
// expected and only traced; any other scheduling failure is logged as a warning.
func (c *Container) ScheduleGCTasks() {
	ctx, l := logger_domain.From(c.GetAppContext(), log)
	orcService := c.orchestratorService
	if orcService == nil {
		return
	}

	hints := orchestrator_domain.RecurringTask{
		Payload: map[string]any{
			"mode":               "hints",
			"batch_size":         gcHintsBatchSize,
			"reschedule_seconds": gcHintsRescheduleSeconds,
		},
		Key:        "blob.gc.hints",
		Executor:   orchestrator_adapters.ExecutorNameBlobGC,
		Delay:      gcHintsInitialDelay,
		Priority:   orchestrator_domain.PriorityLow,
		Background: true,
	}
	if recurring, ok := orcService.GetTaskDispatcher().(orchestrator_domain.RecurringRegistry); ok {
		if err := recurring.RegisterRecurring(ctx, hints); err != nil && !errors.Is(err, orchestrator_domain.ErrDispatcherStopped) {
			l.Warn("Failed to schedule the GC hints task", logger_domain.Error(err))
		}
		return
	}

	hintsTask := orchestrator_domain.NewTask(hints.Executor, hints.Payload)
	hintsTask.DeduplicationKey = hints.Key
	hintsTask.Config.Priority = hints.Priority
	hintsTask.Background = hints.Background
	_, err := orcService.Schedule(ctx, hintsTask, time.Now().Add(hints.Delay))
	switch {
	case err == nil, errors.Is(err, orchestrator_domain.ErrDuplicateTask):
	case errors.Is(err, orchestrator_domain.ErrOrchestratorShuttingDown):
		l.Trace("Skipped GC task scheduling because the orchestrator is shutting down")
	default:
		l.Warn("Failed to schedule the GC hints task", logger_domain.Error(err))
	}
}

// createOrchestratorTaskStore creates the task store, using the querier-based DAL adapter
// when a DatabaseNameOrchestrator database is registered, or falling back to the default
// otter in-memory backend.
//
// Returns orchestrator_domain.TaskStore which is the configured task store.
// Returns error when the factory or database connection fails.
func (c *Container) createOrchestratorTaskStore() (orchestrator_domain.TaskStore, error) {
	if c.isBuildTime {
		return c.createProviderOrchestratorDAL()
	}

	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameOrchestrator]; registered {
			return c.createQuerierOrchestratorDAL()
		}
	}

	return c.createProviderOrchestratorDAL()
}

// createQuerierOrchestratorDAL creates an orchestrator DAL from a querier-managed
// database connection registered via AddDatabase(DatabaseNameOrchestrator, ...).
//
// Returns orchestrator_domain.TaskStore which is the querier-backed task store.
// Returns error when the database connection cannot be obtained.
func (c *Container) createQuerierOrchestratorDAL() (orchestrator_domain.TaskStore, error) {
	database, observed, driver, err := c.resolveQuerierDatabase(DatabaseNameOrchestrator, "orchestrator")
	if err != nil {
		return nil, err
	}

	if err := checkOrchestratorSchema(database, isPostgresDriver(driver)); err != nil {
		return nil, err
	}

	if isPostgresDriver(driver) {
		dal := orchestrator_querier_postgres.NewObserved(database, observed)
		if inspector, ok := dal.(orchestrator_domain.OrchestratorInspector); ok {
			c.orchestratorInspector = inspector
		}
		return dal, nil
	}

	dal := orchestrator_querier_sqlite.NewObserved(database, observed)
	if inspector, ok := dal.(orchestrator_domain.OrchestratorInspector); ok {
		c.orchestratorInspector = inspector
	}

	return dal, nil
}

// checkOrchestratorSchema verifies the orchestrator database has every migration applied,
// so an outdated schema fails at startup with its remedy rather than on the first task.
//
// Takes database (*sql.DB) which is the orchestrator database.
// Takes postgres (bool) which selects the Postgres schema over SQLite's.
//
// Returns error which wraps orchestrator_dal.ErrSchemaOutdated when a migration is
// missing.
func checkOrchestratorSchema(database *sql.DB, postgres bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), schemaCheckTimeout)
	defer cancel()
	if postgres {
		return orchestrator_querier_postgres.CheckSchema(ctx, database)
	}
	return orchestrator_querier_sqlite.CheckSchema(ctx, database)
}

// createProviderOrchestratorDAL creates an orchestrator DAL from the default otter
// in-memory backend with WAL persistence.
//
// Returns orchestrator_domain.TaskStore which is the otter-backed task store.
// Returns error when the otter DAL cannot be created or does not implement TaskStore.
func (c *Container) createProviderOrchestratorDAL() (orchestrator_domain.TaskStore, error) {
	if c.embeddedPikoFS != nil {
		return c.createEmbeddedOrchestratorDAL()
	}

	dalAny, err := c.createOtterOrchestratorDAL()
	if err != nil {
		return nil, fmt.Errorf("failed to create otter orchestrator DAL: %w", err)
	}

	dal, ok := dalAny.(orchestrator_domain.TaskStore)
	if !ok {
		return nil, errors.New("otter orchestrator DAL does not implement TaskStore")
	}

	if inspector, ok := dalAny.(orchestrator_domain.OrchestratorInspector); ok {
		c.orchestratorInspector = inspector
	}

	return dal, nil
}

// createEmbeddedOrchestratorDAL creates a fresh in-memory otter task store for a
// single-binary deploy.
//
// Returns orchestrator_domain.TaskStore which is the fresh in-memory task store.
// Returns error when the DAL cannot be created or does not implement TaskStore.
func (c *Container) createEmbeddedOrchestratorDAL() (orchestrator_domain.TaskStore, error) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating a fresh in-memory orchestrator DAL for embedded mode (task state is never embedded)")

	dalAny, otterErr := orchestrator_otter.NewOtterDAL(orchestrator_otter.Config{})
	if otterErr != nil {
		return nil, fmt.Errorf("creating embedded orchestrator DAL: %w", otterErr)
	}

	dal, ok := dalAny.(orchestrator_domain.TaskStore)
	if !ok {
		return nil, errors.New("otter orchestrator DAL does not implement TaskStore")
	}

	if rebuilder, ok := dalAny.(interface{ RebuildIndexes(context.Context) }); ok {
		rebuilder.RebuildIndexes(c.GetAppContext())
	}

	if inspector, isInspector := dalAny.(orchestrator_domain.OrchestratorInspector); isInspector {
		c.orchestratorInspector = inspector
	}

	return dal, nil
}
