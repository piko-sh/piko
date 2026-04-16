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

// This file contains orchestrator and capability service related container
// methods.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/compiler/compiler_adapters"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/cssinliner"
	"piko.sh/piko/internal/esbuild/compat"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	orchestrator_otter "piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	orchestrator_querier_adapter "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_adapter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/video/video_domain"
	"piko.sh/piko/wdk/safedisk"
)

// GetCapabilityService returns the capability detection service, creating it
// if necessary.
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

// createDefaultCapabilityService sets up the default capability service with
// image transformation, video transcoding, and optional component compilation
// support.
func (c *Container) createDefaultCapabilityService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default CapabilityService...")
	serverConfig := c.config.ServerConfig

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

// createCapabilityWithCompiler sets up the compiler and creates the capability
// service with compilation support.
//
// Takes imageService (any) which is the image service to register.
// Takes videoService (any) which is the video service to register.
// Takes baseDir (string) which is the project root directory for CSS import
// resolution.
// Takes componentsDir (string) which is the path to the local components
// directory.
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

// createCSSPreProcessor creates a CSS pre-processor that resolves @import
// statements in component style blocks.
//
// Takes resolver (resolver_domain.ResolverPort) which resolves CSS import
// paths including @/ aliases.
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
	})
	return compiler_adapters.NewCSSPreProcessor(processor, fsReader, resolver.GetModuleName(), baseDir), nil
}

// hasExternalModuleComponents returns true if any external component
// definition carries a ModulePath, indicating it needs module resolution.
//
// Returns bool which is true when at least one external component has a
// non-empty ModulePath.
func (c *Container) hasExternalModuleComponents() bool {
	for _, definition := range c.externalComponents {
		if definition.ModulePath != "" {
			return true
		}
	}
	return false
}

// GetOrchestratorService returns the asset orchestration service, creating
// it if necessary.
//
// Returns orchestrator_domain.OrchestratorService which provides asset
// orchestration capabilities.
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
// This must be called after GetOrchestratorService() as the bridge is created
// during orchestrator initialisation.
//
// Returns *orchestrator_adapters.ArtefactWorkflowBridge which provides
// integration between artefacts and the workflow system.
func (c *Container) GetArtefactBridge() *orchestrator_adapters.ArtefactWorkflowBridge {
	_, _ = c.GetOrchestratorService()
	return c.artefactBridge
}

// createDefaultOrchestratorService sets up the default orchestrator service
// and its supporting parts.
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

// createOrchestratorServiceCore creates the orchestrator service and sets up
// its executor.
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

	orcService, err := orchestrator.NewService(c.GetAppContext(), orchestrator.Config{TaskStore: taskStore, EventBus: c.GetEventBus()})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create orchestrator service: %w", err)
	}

	compilerExecutor := orchestrator_adapters.NewCompilerExecutor(registryService, capabilityService)
	if err := orcService.RegisterExecutor(c.GetAppContext(), orchestrator_adapters.ExecutorNameArtefactCompiler, compilerExecutor); err != nil {
		return nil, nil, fmt.Errorf("failed to register compiler executor: %w", err)
	}

	gcExecutor := orchestrator_adapters.NewGCExecutor(registryService, orcService)
	if err := orcService.RegisterExecutor(c.GetAppContext(), orchestrator_adapters.ExecutorNameBlobGC, gcExecutor); err != nil {
		return nil, nil, fmt.Errorf("failed to register GC executor: %w", err)
	}

	return orcService, registryService, nil
}

// setupOrchestratorBridge creates and starts the artefact workflow bridge.
//
// Takes orcService (orchestrator.Service) which provides orchestration
// capabilities.
// Takes registryService (registry_domain.RegistryService) which provides registry
// access.
//
// Returns *orchestrator_adapters.ArtefactWorkflowBridge which is the
// configured bridge ready for use.
// Returns error when the bridge listener fails to start.
//
// Spawns a goroutine to wait for bridge events until the context is cancelled.
func (c *Container) setupOrchestratorBridge(orcService orchestrator.Service, registryService registry_domain.RegistryService) (*orchestrator_adapters.ArtefactWorkflowBridge, error) {
	eventBus := c.GetEventBus()
	bridge := orchestrator_adapters.NewArtefactWorkflowBridge(registryService, orcService, orcService.GetTaskDispatcher(), eventBus)

	wait, err := bridge.StartListening(c.GetAppContext(), eventBus)
	if err != nil {
		return nil, fmt.Errorf("failed to start bridge listener: %w", err)
	}
	go wait()
	return bridge, nil
}

// startOrchestratorBackground starts the orchestrator service in a background
// goroutine.
//
// Takes orcService (orchestrator.Service) which is the orchestrator to start.
//
// Safe for concurrent use. The spawned goroutine runs until shutdown.
func (c *Container) startOrchestratorBackground(orcService orchestrator.Service) {
	go orcService.Run(c.GetAppContext())
	shutdown.Register(c.GetAppContext(), "Orchestrator", func(_ context.Context) error { orcService.Stop(); return nil })
}

// ScheduleGCTasks seeds the GC task queue with the first hint processing and
// orphan scan tasks. Deduplication keys prevent duplicate scheduling if the
// system restarts.
//
// Safe to call multiple times; panics from a stopped orchestrator (e.g. after
// daemon restart in dev-interpreted mode) are recovered and silently ignored.
func (c *Container) ScheduleGCTasks() {
	defer func() {
		if r := recover(); r != nil {
			_, l := logger_domain.From(c.GetAppContext(), log)
			l.Trace("ScheduleGCTasks recovered from panic (orchestrator likely stopped)",
				logger_domain.Field("panic", r))
		}
	}()

	ctx := c.GetAppContext()
	orcService := c.orchestratorService
	if orcService == nil {
		return
	}

	hintsTask := orchestrator_domain.NewTask(orchestrator_adapters.ExecutorNameBlobGC, map[string]any{
		"mode":               "hints",
		"batch_size":         100,
		"reschedule_seconds": 30,
	})
	hintsTask.DeduplicationKey = "blob.gc.hints"
	hintsTask.Config.Priority = orchestrator_domain.PriorityLow
	_, _ = orcService.Schedule(ctx, hintsTask, time.Now().Add(10*time.Second))

	orphansTask := orchestrator_domain.NewTask(orchestrator_adapters.ExecutorNameBlobGC, map[string]any{
		"mode":               "orphans",
		"batch_size":         100,
		"reschedule_seconds": 3600,
	})
	orphansTask.DeduplicationKey = "blob.gc.orphans"
	orphansTask.Config.Priority = orchestrator_domain.PriorityLow
	_, _ = orcService.Schedule(ctx, orphansTask, time.Now().Add(60*time.Second))
}

// createOrchestratorTaskStore creates the task store, using the querier-based
// DAL adapter when a DatabaseNameOrchestrator database is registered, or
// falling back to the default otter in-memory backend.
//
// Returns orchestrator_domain.TaskStore which is the configured task store.
// Returns error when the factory or database connection fails.
func (c *Container) createOrchestratorTaskStore() (orchestrator_domain.TaskStore, error) {
	if c.dbRegistrations != nil {
		if _, registered := c.dbRegistrations[DatabaseNameOrchestrator]; registered {
			return c.createQuerierOrchestratorDAL()
		}
	}

	return c.createProviderOrchestratorDAL()
}

// createQuerierOrchestratorDAL creates an orchestrator DAL from a
// querier-managed database connection registered via
// AddDatabase(DatabaseNameOrchestrator, ...).
//
// Returns orchestrator_domain.TaskStore which is the querier-backed task store.
// Returns error when the database connection cannot be obtained.
func (c *Container) createQuerierOrchestratorDAL() (orchestrator_domain.TaskStore, error) {
	if err := c.runMigrationsIfConfigured(DatabaseNameOrchestrator); err != nil {
		return nil, fmt.Errorf("failed to migrate orchestrator database: %w", err)
	}

	database, err := c.GetDatabaseConnection(DatabaseNameOrchestrator)
	if err != nil {
		return nil, fmt.Errorf("failed to get orchestrator database connection: %w", err)
	}

	dal := orchestrator_querier_adapter.New(database)

	if inspector, ok := dal.(orchestrator_domain.OrchestratorInspector); ok {
		c.orchestratorInspector = inspector
	}

	return dal, nil
}

// createProviderOrchestratorDAL creates an orchestrator DAL from either a
// user-provided cache provider or the default otter in-memory backend with WAL
// persistence.
//
// Returns orchestrator_domain.TaskStore which is the cache-backed task store.
// Returns error when the DAL cannot be created or does not implement
// TaskStore.
func (c *Container) createProviderOrchestratorDAL() (orchestrator_domain.TaskStore, error) {
	dalAny, err := c.createOrchestratorDALInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator DAL: %w", err)
	}

	dal, ok := dalAny.(orchestrator_domain.TaskStore)
	if !ok {
		return nil, errors.New("orchestrator DAL does not implement TaskStore")
	}

	if inspector, ok := dalAny.(orchestrator_domain.OrchestratorInspector); ok {
		c.orchestratorInspector = inspector
	}

	return dal, nil
}

// createOrchestratorDALInstance returns the orchestrator DAL from either the
// cache override or the default otter backend.
//
// Returns any which is the DAL instance (expected to implement TaskStore).
// Returns error when the DAL cannot be created.
func (c *Container) createOrchestratorDALInstance() (any, error) {
	if c.orchestratorCacheOverride != nil {
		return orchestrator_otter.NewOtterDAL(
			orchestrator_otter.Config{},
			orchestrator_otter.WithCache(c.orchestratorCacheOverride),
		)
	}
	return c.createOtterOrchestratorDAL()
}
