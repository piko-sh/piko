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

// This file contains coordinator, annotator, generator, and resolver container
// methods.

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/collection/collection_adapters"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/esbuild/compat"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_adapters/driven_code_emitter_go_literal"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/logger/logger_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/safedisk"
)

// GetAnnotatorService returns the code annotation service, creating it if
// necessary.
//
// Returns annotator_domain.AnnotatorPort which provides code annotation
// functionality.
// Returns error when service creation fails.
func (c *Container) GetAnnotatorService() (annotator_domain.AnnotatorPort, error) {
	c.annotatorOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.annotatorServiceOverride != nil {
			l.Internal("Using provided AnnotatorService override.")
			c.annotatorService = c.annotatorServiceOverride
			return
		}
		c.createDefaultAnnotatorService()
	})
	return c.annotatorService, c.annotatorErr
}

// createDefaultAnnotatorService sets up the default annotator service.
//
// Stores any error in c.annotatorErr rather than returning it.
func (c *Container) createDefaultAnnotatorService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default AnnotatorService...")
	baseDir := deref(c.config.ServerConfig.Paths.BaseDir, ".")

	resolver, err := c.GetResolver()
	if err != nil {
		c.annotatorErr = err
		return
	}

	fsReader, err := c.createAnnotatorFSReader(baseDir)
	if err != nil {
		c.annotatorErr = err
		return
	}

	c.annotatorService, c.annotatorErr = c.createAnnotatorServiceInstance(resolver, fsReader)
}

// createAnnotatorFSReader creates the file system reader for the annotator
// service.
//
// Takes baseDir (string) which provides the base folder path for the source
// sandbox.
//
// Returns annotator_domain.FSReaderPort which provides file system read access
// for the annotator.
// Returns error when the source sandbox cannot be created.
func (c *Container) createAnnotatorFSReader(baseDir string) (annotator_domain.FSReaderPort, error) {
	if c.coordinatorFSReaderOverride != nil {
		return c.coordinatorFSReaderOverride, nil
	}
	sourceSandbox, err := c.createSandbox("annotator-source", baseDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating annotator source sandbox: %w", err)
	}
	return generator_adapters.NewFSReader(sourceSandbox), nil
}

// createAnnotatorServiceInstance creates the annotator service with all its
// dependencies.
//
// Takes resolver (resolver_domain.ResolverPort) which resolves file paths.
// Takes fsReader (annotator_domain.FSReaderPort) which reads files from disk.
//
// Returns annotator_domain.AnnotatorPort which is the configured annotator.
// Returns error when a required dependency cannot be obtained.
func (c *Container) createAnnotatorServiceInstance(
	resolver resolver_domain.ResolverPort,
	fsReader annotator_domain.FSReaderPort,
) (annotator_domain.AnnotatorPort, error) {
	cssProcessor := annotator_domain.NewCSSProcessor(esbuildconfig.LoaderLocalCSS, &esbuildconfig.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	typeInspectorManager, err := c.GetTypeInspectorManager()
	if err != nil {
		return nil, fmt.Errorf("getting type inspector manager for annotator: %w", err)
	}

	_, l := logger_domain.From(c.GetAppContext(), log)

	collectionService, err := c.GetCollectionService()
	if err != nil {
		l.Warn("Collection service not available, GetCollection() calls will fail",
			logger_domain.Error(err))
		collectionService = nil
	}

	compilationLogLevel := logger_dto.ParseLogLevel(config.CompilerDefaultLogLevel, slog.LevelWarn)

	enableDebugLogFiles := config.CompilerEnableDebugLogFiles
	if c.compilerDebugLogsEnabled != nil {
		enableDebugLogFiles = *c.compilerDebugLogsEnabled
	}

	return annotator_domain.NewAnnotatorService(c.GetAppContext(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(typeInspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         NewAnnotatorPathsConfig(&c.config.ServerConfig),
		AssetsConfig:        c.GetAssetsConfig(),
		Cache:               annotator_adapters.NewComponentCache(),
		CompilationLogLevel: compilationLogLevel,
		EnableDebugLogFiles: enableDebugLogFiles,
		DebugLogDir:         config.CompilerDebugLogDir,
		CollectionService:   collectionService,
		ComponentRegistry:   c.GetComponentRegistry(),
	})
}

// GetCoordinatorService returns the build coordination service, creating it
// if necessary.
//
// Returns coordinator_domain.CoordinatorService which provides build
// coordination.
// Returns error when the service cannot be created.
func (c *Container) GetCoordinatorService() (coordinator_domain.CoordinatorService, error) {
	c.coordinatorOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.coordinatorServiceOverride != nil {
			l.Internal("Using provided CoordinatorService override.")
			c.coordinatorService = c.coordinatorServiceOverride
			return
		}
		c.createDefaultCoordinatorService()
	})
	return c.coordinatorService, c.coordinatorErr
}

// createDefaultCoordinatorService sets up the coordinator service with its
// default components.
func (c *Container) createDefaultCoordinatorService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default CoordinatorService...")
	annotator, err := c.GetAnnotatorService()
	if err != nil {
		c.coordinatorErr = err
		return
	}

	fsReader, err := c.createCoordinatorFSReader()
	if err != nil {
		c.coordinatorErr = err
		return
	}

	resolver, err := c.GetResolver()
	if err != nil {
		c.coordinatorErr = err
		return
	}

	buildCache, introspectionCache, err := c.createCoordinatorCaches()
	if err != nil {
		c.coordinatorErr = err
		return
	}

	c.coordinatorService = coordinator_domain.NewService(
		c.GetAppContext(),
		annotator,
		buildCache,
		introspectionCache,
		fsReader,
		resolver,
		c.getCoordinatorOptions()...,
	)

	shutdown.Register(c.GetAppContext(), "CoordinatorService", func(ctx context.Context) error {
		c.coordinatorService.Shutdown(ctx)
		if closer, ok := buildCache.(interface{ Close() }); ok {
			closer.Close()
		}
		if closer, ok := introspectionCache.(interface{ Close() }); ok {
			closer.Close()
		}
		return nil
	})
}

// createCoordinatorCaches creates the build result and introspection caches,
// using overrides when provided and falling back to defaults otherwise.
//
// Returns coordinator_domain.BuildResultCachePort which is the build result
// cache.
// Returns coordinator_domain.IntrospectionCachePort which is the introspection
// cache.
// Returns error when a cache cannot be created.
func (c *Container) createCoordinatorCaches() (coordinator_domain.BuildResultCachePort, coordinator_domain.IntrospectionCachePort, error) {
	cacheService, err := c.GetCacheService()
	if err != nil {
		return nil, nil, fmt.Errorf("creating coordinator cache service: %w", err)
	}

	var buildCache coordinator_domain.BuildResultCachePort
	if c.coordinatorCacheOverride != nil {
		buildCache = c.coordinatorCacheOverride
	} else {
		buildCache, err = coordinator_adapters.NewBuildResultCache(c.GetAppContext(), cacheService)
		if err != nil {
			return nil, nil, fmt.Errorf("creating coordinator build result cache: %w", err)
		}
	}

	var introspectionCache coordinator_domain.IntrospectionCachePort
	if c.introspectionCacheOverride != nil {
		introspectionCache = c.introspectionCacheOverride
	} else {
		introspectionCache, err = coordinator_adapters.NewIntrospectionCache(c.GetAppContext(), cacheService)
		if err != nil {
			return nil, nil, fmt.Errorf("creating coordinator introspection cache: %w", err)
		}
	}

	return buildCache, introspectionCache, nil
}

// createCoordinatorFSReader creates the file system reader for the
// coordinator service.
//
// Returns annotator_domain.FSReaderPort which provides file system access.
// Returns error when the sandbox cannot be created.
func (c *Container) createCoordinatorFSReader() (annotator_domain.FSReaderPort, error) {
	if c.coordinatorFSReaderOverride != nil {
		return c.coordinatorFSReaderOverride, nil
	}
	serverConfig := c.config.ServerConfig
	sourceSandbox, err := c.createSandbox("coordinator-source", deref(serverConfig.Paths.BaseDir, "."), safedisk.ModeReadOnly)
	if err != nil {
		return nil, fmt.Errorf("creating coordinator source sandbox: %w", err)
	}
	return generator_adapters.NewFSReader(sourceSandbox), nil
}

// getCoordinatorOptions returns the functional options for the coordinator
// service.
//
// Returns []coordinator_domain.CoordinatorOption which contains the configured
// options including any overrides for file hash cache, code emitter, and
// diagnostic output.
func (c *Container) getCoordinatorOptions() []coordinator_domain.CoordinatorOption {
	var opts []coordinator_domain.CoordinatorOption

	if c.coordinatorFileHashCacheOverride != nil {
		opts = append(opts, coordinator_domain.WithFileHashCache(c.coordinatorFileHashCacheOverride))
	}
	if c.coordinatorCodeEmitterOverride != nil {
		opts = append(opts, coordinator_domain.WithCodeEmitter(c.coordinatorCodeEmitterOverride))
	}

	serverConfig := c.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	baseDirSandbox, err := c.createSandbox("coordinator-basedir", baseDir, safedisk.ModeReadOnly)
	if err == nil {
		opts = append(opts, coordinator_domain.WithBaseDirSandbox(baseDirSandbox))
	}

	diagnosticOutput := c.coordinatorDiagnosticOutputOverride
	if diagnosticOutput == nil {
		diagnosticOutput = coordinator_adapters.NewCLIDiagnosticOutput()
	}
	opts = append(opts,
		coordinator_domain.WithDiagnosticOutput(diagnosticOutput),
		coordinator_domain.WithStaticHoisting(config.CompilerEnableStaticHoisting),
		coordinator_domain.WithPrerendering(c.experimentalPrerendering),
		coordinator_domain.WithStripHTMLComments(c.experimentalCommentStripping),
		coordinator_domain.WithDwarfLineDirectives(c.experimentalDwarfLineDirectives),
	)

	return opts
}

// GetCoordinatorCache returns the build result cache for the coordinator.
//
// Returns coordinator_domain.BuildResultCachePort which provides the cache
// instance, initialised lazily on first call.
// Returns error when the cache cannot be created.
func (c *Container) GetCoordinatorCache() (coordinator_domain.BuildResultCachePort, error) {
	c.coordinatorCacheOnce.Do(func() {
		if c.coordinatorCacheOverride != nil {
			c.coordinatorCache = c.coordinatorCacheOverride
			return
		}
		cacheService, err := c.GetCacheService()
		if err != nil {
			c.coordinatorCacheErr = fmt.Errorf("creating coordinator cache service: %w", err)
			return
		}
		c.coordinatorCache, c.coordinatorCacheErr = coordinator_adapters.NewBuildResultCache(c.GetAppContext(), cacheService)
	})
	return c.coordinatorCache, c.coordinatorCacheErr
}

// GetGeneratorService returns the code generation service, creating it if
// necessary.
//
// Returns generator_domain.GeneratorService which provides code generation.
// Returns error when the service cannot be created.
func (c *Container) GetGeneratorService() (generator_domain.GeneratorService, error) {
	c.generatorOnce.Do(func() {
		if c.generatorServiceOverride != nil {
			c.generatorService = c.generatorServiceOverride
			return
		}
		c.createDefaultGeneratorService()
	})
	return c.generatorService, c.generatorErr
}

// GetCodeEmitter returns a code emitter for generating Go code from
// annotations. This is primarily used in dev-i mode to provide emitted
// artefacts to the coordinator.
//
// Returns coordinator_domain.CodeEmitterPort which generates Go code.
// Returns error when the emitter cannot be created.
func (c *Container) GetCodeEmitter() (coordinator_domain.CodeEmitterPort, error) {
	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil,
		render_domain.WithStripHTMLComments(c.experimentalCommentStripping),
	)
	factory := driven_code_emitter_go_literal.NewEmitterFactory(c.GetAppContext(), prerenderer)
	emitter := factory.NewEmitter()
	return emitter, nil
}

// SetIntrospectionCacheOverride sets the Tier 1 introspection cache override,
// which must be called before GetCoordinatorService to take effect and replaces
// the default introspection cache created during coordinator service
// initialisation.
//
// Takes cache (IntrospectionCachePort) which provides the cache to use.
func (c *Container) SetIntrospectionCacheOverride(cache coordinator_domain.IntrospectionCachePort) {
	c.introspectionCacheOverride = cache
}

// SetCoordinatorCacheOverride sets the Tier 2 build result cache override,
// which must be called before GetCoordinatorService to take effect and replaces
// the default build result cache created during coordinator service
// initialisation.
//
// Takes cache (BuildResultCachePort) which provides the cache to use.
func (c *Container) SetCoordinatorCacheOverride(cache coordinator_domain.BuildResultCachePort) {
	c.coordinatorCacheOverride = cache
}

// SetCoordinatorCodeEmitterOverride sets the code emitter override for the
// coordinator. This must be called before GetCoordinatorService to take
// effect.
//
// Takes emitter (CodeEmitterPort) which provides the code emission behaviour.
func (c *Container) SetCoordinatorCodeEmitterOverride(emitter coordinator_domain.CodeEmitterPort) {
	c.coordinatorCodeEmitterOverride = emitter
}

// SetCoordinatorDiagnosticOutputOverride sets the diagnostic output strategy
// for the coordinator.
//
// This must be called before GetCoordinatorService() to take effect. Use
// SilentDiagnosticOutput for LSP contexts to prevent stderr pollution.
//
// Takes output (DiagnosticOutputPort) which specifies the output strategy.
func (c *Container) SetCoordinatorDiagnosticOutputOverride(output coordinator_domain.DiagnosticOutputPort) {
	c.coordinatorDiagnosticOutputOverride = output
}

// SetFSReaderOverride sets the file system reader for the coordinator.
//
// This must be called before GetCoordinatorService() to take effect. Use
// lspFSReader for LSP contexts to prioritise in-memory content for open
// documents.
//
// Takes fsReader (annotator_domain.FSReaderPort) which provides the file
// system reader to use for document access.
func (c *Container) SetFSReaderOverride(fsReader annotator_domain.FSReaderPort) {
	c.coordinatorFSReaderOverride = fsReader
}

// SetResolverOverride sets a custom resolver for module path resolution.
//
// Takes resolver (resolver_domain.ResolverPort) which specifies the resolver
// to use instead of the default.
//
// Must be called before GetResolver() to take effect. Use
// InMemoryModuleResolver for tests or contexts without go.mod.
func (c *Container) SetResolverOverride(resolver resolver_domain.ResolverPort) {
	c.resolverOverride = resolver
}

// GetResolver returns the module resolution service, creating it if necessary.
//
// Returns resolver_domain.ResolverPort which provides module path resolution.
// Returns error when the Go module cannot be detected.
func (c *Container) GetResolver() (resolver_domain.ResolverPort, error) {
	c.resolverOnce.Do(func() {
		if c.resolverOverride != nil {
			c.resolver = c.resolverOverride
			return
		}
		localResolver := resolver_adapters.NewLocalModuleResolver(deref(c.config.ServerConfig.Paths.BaseDir, "."))
		cacheResolver := resolver_adapters.NewGoModuleCacheResolver()
		resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
		if err := resolver.DetectLocalModule(c.GetAppContext()); err != nil {
			c.resolverErr = fmt.Errorf("could not detect Go module: %w", err)
			return
		}
		c.resolver = resolver
	})
	return c.resolver, c.resolverErr
}

// createDefaultGeneratorService sets up the generator service with default
// settings.
func (c *Container) createDefaultGeneratorService() {
	serverConfig := &c.config.ServerConfig

	sandboxes, err := c.createGeneratorSandboxes(serverConfig)
	if err != nil {
		c.generatorErr = err
		return
	}

	ports, err := c.createGeneratorPorts(serverConfig, sandboxes)
	if err != nil {
		c.generatorErr = err
		return
	}

	c.generatorService, c.generatorErr = generator_domain.NewGeneratorService(c.GetAppContext(), NewGeneratorPathsConfig(serverConfig), deref(serverConfig.I18nDefaultLocale, "en"), ports,
		generator_domain.WithPrerendering(c.experimentalPrerendering),
		generator_domain.WithStripHTMLComments(c.experimentalCommentStripping),
		generator_domain.WithDwarfLineDirectives(c.experimentalDwarfLineDirectives),
	)
}

// generatorSandboxes holds the sandboxes for the generator service.
type generatorSandboxes struct {
	// source is the sandbox for reading source files.
	source safedisk.Sandbox

	// base is the read-write sandbox rooted at the project base directory,
	// used by the generator for creating action stubs and reading TypeScript sources.
	base safedisk.Sandbox

	// output is the sandbox folder where generated files are written.
	output safedisk.Sandbox
}

// createGeneratorSandboxes creates the source and output sandboxes for the
// generator.
//
// Takes serverConfig (*config.ServerConfig) which provides the paths
// configuration.
//
// Returns generatorSandboxes which contains the source and output sandboxes.
// Returns error when either sandbox cannot be created.
func (c *Container) createGeneratorSandboxes(serverConfig *config.ServerConfig) (generatorSandboxes, error) {
	sourceSandbox, err := c.createSandbox("generator-source", deref(serverConfig.Paths.BaseDir, "."), safedisk.ModeReadOnly)
	if err != nil {
		return generatorSandboxes{}, fmt.Errorf("creating generator source sandbox: %w", err)
	}
	baseSandbox, err := c.createSandbox("generator-base", deref(serverConfig.Paths.BaseDir, "."), safedisk.ModeReadWrite)
	if err != nil {
		return generatorSandboxes{}, fmt.Errorf("creating generator base sandbox: %w", err)
	}
	outputDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), "dist")
	outputSandbox, err := c.createSandbox("generator-output", outputDir, safedisk.ModeReadWrite)
	if err != nil {
		return generatorSandboxes{}, fmt.Errorf("creating generator output sandbox: %w", err)
	}
	return generatorSandboxes{source: sourceSandbox, base: baseSandbox, output: outputSandbox}, nil
}

// createGeneratorPorts creates all the ports for the generator service.
//
// Takes serverConfig (*config.ServerConfig) which provides server settings.
// Takes sandboxes (generatorSandboxes) which specifies input and output paths.
//
// Returns generator_domain.GeneratorPorts which contains all configured ports.
// Returns error when any port dependency fails to initialise.
func (c *Container) createGeneratorPorts(serverConfig *config.ServerConfig, sandboxes generatorSandboxes) (generator_domain.GeneratorPorts, error) {
	fsWriter := generator_adapters.NewFSWriter(sandboxes.output)

	coordinator, err := c.GetCoordinatorService()
	if err != nil {
		return generator_domain.GeneratorPorts{}, fmt.Errorf("getting coordinator service for generator: %w", err)
	}
	resolver, err := c.GetResolver()
	if err != nil {
		return generator_domain.GeneratorPorts{}, fmt.Errorf("getting resolver for generator: %w", err)
	}
	manifestEmitter, err := createManifestEmitterFromConfig(sandboxes.output)
	if err != nil {
		return generator_domain.GeneratorPorts{}, fmt.Errorf("creating manifest emitter for generator: %w", err)
	}

	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil,
		render_domain.WithStripHTMLComments(c.experimentalCommentStripping),
	)

	return generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		BaseSandbox:        sandboxes.base,
		ManifestEmitter:    manifestEmitter,
		Coordinator:        coordinator,
		Resolver:           resolver,
		RegisterEmitter:    generator_adapters.NewRegisterEmitter(fsWriter),
		CodeEmitterFactory: driven_code_emitter_go_literal.NewEmitterFactory(c.GetAppContext(), prerenderer),
		CollectionEmitter:  c.createCollectionEmitter(fsWriter, sandboxes.output, resolver),
		SearchIndexEmitter: c.createSearchIndexEmitter(fsWriter, sandboxes.output, resolver),
		PKJSEmitter:        c.createPKJSEmitter(),
		I18nEmitter: generator_adapters.NewDrivenI18nEmitter(generator_adapters.I18nEmitterConfig{
			BaseDir:       deref(serverConfig.Paths.BaseDir, "."),
			I18nSourceDir: deref(serverConfig.Paths.I18nSourceDir, "locales"),
			DefaultLocale: deref(serverConfig.I18nDefaultLocale, "en"),
		}, sandboxes.source, sandboxes.output),
		ActionGenerator: generator_adapters.NewActionGeneratorAdapter(generator_adapters.WithActionSandbox(sandboxes.output)),
		SEOService:      c.getSEOServiceOptional(),
	}, nil
}

// createCollectionEmitter creates the collection emitter for the generator.
//
// Takes fsWriter (generator_domain.FSWriterPort) which handles writing files.
// Takes outputSandbox (safedisk.Sandbox) which provides a safe output folder.
// Takes resolver (resolver_domain.ResolverPort) which resolves module names.
//
// Returns generator_domain.CollectionEmitterPort which writes collection data
// to files.
func (*Container) createCollectionEmitter(fsWriter generator_domain.FSWriterPort, outputSandbox safedisk.Sandbox, resolver resolver_domain.ResolverPort) generator_domain.CollectionEmitterPort {
	return generator_adapters.NewDrivenCollectionEmitter(
		collection_adapters.NewFlatBufferEncoder(),
		fsWriter,
		outputSandbox,
		resolver.GetModuleName(),
	)
}

// createSearchIndexEmitter creates the search index emitter for the generator.
//
// Takes fsWriter (generator_domain.FSWriterPort) which handles file system
// writes.
// Takes outputSandbox (safedisk.Sandbox) which restricts output to a safe
// directory.
// Takes resolver (resolver_domain.ResolverPort) which provides the module name.
//
// Returns generator_domain.SearchIndexEmitterPort which emits search index
// data.
func (*Container) createSearchIndexEmitter(
	fsWriter generator_domain.FSWriterPort,
	outputSandbox safedisk.Sandbox,
	resolver resolver_domain.ResolverPort,
) generator_domain.SearchIndexEmitterPort {
	return generator_adapters.NewDrivenSearchIndexEmitter(
		search_domain.NewIndexBuilder(),
		fsWriter,
		outputSandbox,
		resolver.GetModuleName(),
		config.ManifestFormat,
	)
}

// createPKJSEmitter creates the PK JavaScript emitter for the generator.
//
// Returns generator_domain.PKJSEmitterPort which outputs JavaScript for PK
// client scripts.
func (c *Container) createPKJSEmitter() generator_domain.PKJSEmitterPort {
	_, l := logger_domain.From(c.GetAppContext(), log)
	registryService, err := c.GetRegistryService()
	if err != nil {
		l.Internal("Failed to get registry service for PK JS emitter, client scripts will be disabled", logger_domain.Error(err))
		registryService = nil
	}
	return generator_adapters.NewPKJSEmitter(registryService)
}

// getSEOServiceOptional returns the SEO service, or nil if not available.
//
// Returns seo_domain.SEOService which is the SEO service, or nil if setup
// fails.
func (c *Container) getSEOServiceOptional() seo_domain.SEOService {
	_, l := logger_domain.From(c.GetAppContext(), log)
	seoService, err := c.GetSEOService()
	if err != nil {
		l.Internal("Failed to initialise SEO service for generator, continuing without SEO features", logger_domain.Error(err))
		return nil
	}
	return seoService
}

// GetTypeInspectorManager returns the type inspection manager, creating it
// if needed.
//
// Returns *inspector_domain.TypeBuilder which provides type inspection.
// Returns error when the manager cannot be created.
func (c *Container) GetTypeInspectorManager() (*inspector_domain.TypeBuilder, error) {
	c.typeInspectorBuilderOnce.Do(func() {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if c.typeInspectorBuilderOverride != nil {
			l.Internal("Using provided TypeInspectorManager override.")
			c.typeInspectorBuilder = c.typeInspectorBuilderOverride
			return
		}
		c.createDefaultTypeInspectorManager()
	})
	return c.typeInspectorBuilder, c.typeInspectorBuilderErr
}

// createDefaultTypeInspectorManager sets up the type inspector builder with
// default settings.
func (c *Container) createDefaultTypeInspectorManager() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default TypeInspectorManager...")
	serverConfig := c.config.ServerConfig
	cacheDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), ".piko", "cache", "types")

	cacheSandbox, err := c.createSandbox("type-inspector-cache", cacheDir, safedisk.ModeReadWrite)
	if err != nil {
		c.typeInspectorBuilderErr = fmt.Errorf("creating cache sandbox: %w", err)
		return
	}

	var provider inspector_domain.TypeDataProvider
	if c.typeDataProvider != nil {
		provider = c.typeDataProvider(cacheSandbox)
	} else {
		var err error
		provider, err = createTypeDataProviderFromConfig(cacheSandbox)
		if err != nil {
			c.typeInspectorBuilderErr = fmt.Errorf("creating type data provider: %w", err)
			return
		}
	}

	localResolver := resolver_adapters.NewLocalModuleResolver(deref(serverConfig.Paths.BaseDir, "."))
	cacheResolver := resolver_adapters.NewGoModuleCacheResolver()
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
	if err := resolver.DetectLocalModule(c.GetAppContext()); err != nil {
		c.typeInspectorBuilderErr = fmt.Errorf("could not detect Go module for type inspector: %w", err)
		return
	}
	c.typeInspectorBuilder = inspector_domain.NewTypeBuilder(
		inspector_dto.Config{
			BaseDir:           deref(serverConfig.Paths.BaseDir, "."),
			ModuleName:        resolver.GetModuleName(),
			BuildFlags:        inspector_dto.AnalysisBuildFlags,
			UseStandardLoader: c.useStandardLoader,
		},
		inspector_domain.WithProvider(provider),
	)
}

// GetCollectionService returns the collection service, initialising it if
// needed. The collection service is responsible for processing GetCollection()
// calls detected by the type resolver during template annotation.
//
// Returns collection_domain.CollectionService which handles collection
// processing for template annotation.
// Returns error when the service fails to initialise.
func (c *Container) GetCollectionService() (collection_domain.CollectionService, error) {
	c.collectionServiceOnce.Do(func() {
		c.createDefaultCollectionService()
	})
	return c.collectionService, c.collectionServiceErr
}

// createDefaultCollectionService sets up the collection service with default
// settings and stores any error in the container.
func (c *Container) createDefaultCollectionService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	l.Internal("Creating default CollectionService...")
	service, err := createCollectionService(c)
	if err != nil {
		c.collectionServiceErr = err
		l.Error("Failed to create collection service", logger_domain.Error(err))
		return
	}
	c.collectionService = service
	initHybridCache(c)
}

// GetSearchService returns the search service, initialising it if needed.
// The search service provides full-text search functionality for static
// collections using pre-built FlatBuffer search indexes.
//
// Returns collection_domain.SearchServicePort which is the search service.
// Returns error when the search service cannot be initialised.
func (c *Container) GetSearchService() (collection_domain.SearchServicePort, error) {
	c.searchServiceOnce.Do(func() {
		c.createDefaultSearchService()
	})
	return c.searchService, c.searchServiceErr
}

// createDefaultSearchService sets up the search service with default settings,
// or uses an override if one has been provided.
func (c *Container) createDefaultSearchService() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	if c.searchServiceOverride != nil {
		c.searchService = c.searchServiceOverride
		l.Internal("Using overridden SearchService")
		return
	}

	l.Internal("Creating default SearchService...")
	c.searchService = collection_domain.NewSearchService()
}

// SetSearchService sets a custom search service implementation.
// This is typically used for testing with mock implementations.
//
// Takes service (SearchServicePort) which is the search service to use.
func (c *Container) SetSearchService(service collection_domain.SearchServicePort) {
	c.searchServiceOverride = service
}

// createManifestEmitterFromConfig creates a FlatBuffer manifest emitter.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
//
// Returns generator_domain.ManifestEmitterPort which is the configured emitter.
// Returns error when creation fails.
func createManifestEmitterFromConfig(sandbox safedisk.Sandbox) (generator_domain.ManifestEmitterPort, error) {
	return generator_adapters.NewFlatBufferManifestEmitter(sandbox), nil
}

// createTypeDataProviderFromConfig creates a FlatBuffer type data provider.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
//
// Returns inspector_domain.TypeDataProvider which is the configured provider.
// Returns error when creation fails.
func createTypeDataProviderFromConfig(sandbox safedisk.Sandbox) (inspector_domain.TypeDataProvider, error) {
	return inspector_adapters.NewFlatBufferCache(sandbox), nil
}
