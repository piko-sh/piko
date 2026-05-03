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

//go:build bench

package generator_test_bench

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	generator_adapters_driven_code_emitter_go_literal "piko.sh/piko/internal/generator/generator_adapters/driven_code_emitter_go_literal"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

func newTestCacheService() cache_domain.Service {
	return cache_domain.NewService("")
}

const benchmarkCacheKey = "benchmark-cache"

type staticKeyGenerator struct{}

func (*staticKeyGenerator) Generate(_ context.Context, _ inspector_dto.Config, _ map[string][]byte, _ map[string]string) (string, error) {
	return benchmarkCacheKey, nil
}

type testCase struct {
	Name string
	Path string
}

type cacheBundle struct {
	inspectorProvider  inspector_domain.TypeDataProvider
	annotatorCache     annotator_domain.ComponentCachePort
	coordinatorCache   coordinator_domain.BuildResultCachePort
	introspectionCache coordinator_domain.IntrospectionCachePort
}

type serviceStack struct {
	generatorService  generator_domain.GeneratorService
	allSourceContents map[string][]byte
	entryPoints       []annotator_dto.EntryPoint
}

func createServiceStack(
	tb testing.TB,
	tc testCase,
	caches *cacheBundle,
) (*serviceStack, *cacheBundle) {
	tb.Helper()
	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(tb, err)

	localResolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	cacheResolver, err := resolver_adapters.NewGoModuleCacheResolverWithWorkingDir(absSrcDir)
	require.NoError(tb, err, "failed to create go module cache resolver")
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(tb, err)

	fsReader := &realFSReader{}
	fsWriter := &generator_domain.MockFSWriter{}
	testSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadWrite)
	tb.Cleanup(func() { testSandbox.Close() })
	manifestEmitter := generator_adapters.NewJSONManifestEmitter(testSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)
	serverConfig := &bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:           &absSrcDir,
			PartialsSourceDir: new("partials"),
			PagesSourceDir:    new("pages"),
		},
	}

	var inspectorProvider inspector_domain.TypeDataProvider
	var annotatorCache annotator_domain.ComponentCachePort
	var coordinatorCache coordinator_domain.BuildResultCachePort
	var introspectionCache coordinator_domain.IntrospectionCachePort

	if caches != nil && caches.inspectorProvider != nil {
		inspectorProvider = caches.inspectorProvider
	} else {
		inspectorProvider = inspector_adapters.NewInMemoryProvider(nil)
	}

	if caches != nil && caches.annotatorCache != nil {
		annotatorCache = caches.annotatorCache
	} else {
		annotatorCache = annotator_adapters.NewComponentCache()
	}

	cacheService := newTestCacheService()

	if caches != nil && caches.coordinatorCache != nil {
		coordinatorCache = caches.coordinatorCache
	} else {
		var cacheErr error
		coordinatorCache, cacheErr = coordinator_adapters.NewBuildResultCache(context.Background(), cacheService)
		require.NoError(tb, cacheErr)
	}

	if caches != nil && caches.introspectionCache != nil {
		introspectionCache = caches.introspectionCache
	} else {
		var cacheErr error
		introspectionCache, cacheErr = coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
		require.NoError(tb, cacheErr)
	}

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspectorProvider),
	)

	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         bootstrap.NewAnnotatorPathsConfig(serverConfig),
		Cache:               annotatorCache,
		CompilationLogLevel: 0,
		CollectionService:   nil,
	})

	coordinatorService := coordinator_domain.NewService(
		context.Background(), annotatorService, coordinatorCache, introspectionCache, fsReader, resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)
	tb.Cleanup(func() { coordinatorService.Shutdown(context.Background()) })

	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	registerEmitter := generator_adapters.NewRegisterEmitter(fsWriter)
	generatorService, err := generator_domain.NewGeneratorService(context.Background(), bootstrap.NewGeneratorPathsConfig(serverConfig), "en", generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		ManifestEmitter:    manifestEmitter,
		Coordinator:        coordinatorService,
		Resolver:           resolver,
		RegisterEmitter:    registerEmitter,
		CodeEmitterFactory: codeEmitterFactory,
		SEOService:         nil,
	})
	require.NoError(tb, err)

	entryPoints := discoverEntryPoints(tb, resolver, *serverConfig)
	require.NotEmpty(tb, entryPoints, "Benchmark discovery failed: no entry points found in src directory")

	allSourceContents := make(map[string][]byte)
	_ = filepath.WalkDir(absSrcDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			content, _ := os.ReadFile(path)
			allSourceContents[path] = content
		}
		return nil
	})

	createdCaches := &cacheBundle{
		inspectorProvider:  inspectorProvider,
		annotatorCache:     annotatorCache,
		coordinatorCache:   coordinatorCache,
		introspectionCache: introspectionCache,
	}

	return &serviceStack{
		generatorService:  generatorService,
		entryPoints:       entryPoints,
		allSourceContents: allSourceContents,
	}, createdCaches
}

func createServiceStackWithFileCache(
	tb testing.TB,
	tc testCase,
	caches *cacheBundle,
) (*serviceStack, *cacheBundle) {
	tb.Helper()
	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(tb, err)

	cacheDir := filepath.Join(tc.Path, "cache")
	absCacheDir, err := filepath.Abs(cacheDir)
	require.NoError(tb, err)

	cacheFile := filepath.Join(absCacheDir, fmt.Sprintf("typedata-%s.bin", benchmarkCacheKey))
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		tb.Skipf("Cache file not found: %s. Run `go test -tags=bench -run TestGenerateCacheFiles ./internal/generator/generator_test_bench/` to generate.", cacheFile)
	}

	localResolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	cacheResolver, err := resolver_adapters.NewGoModuleCacheResolverWithWorkingDir(absSrcDir)
	require.NoError(tb, err, "failed to create go module cache resolver")
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(tb, err)

	fsReader := &realFSReader{}
	fsWriter := &generator_domain.MockFSWriter{}
	testSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadWrite)
	tb.Cleanup(func() { testSandbox.Close() })
	manifestEmitter := generator_adapters.NewJSONManifestEmitter(testSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)
	serverConfig := &bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:           &absSrcDir,
			PartialsSourceDir: new("partials"),
			PagesSourceDir:    new("pages"),
		},
	}

	var inspectorProvider inspector_domain.TypeDataProvider
	var annotatorCache annotator_domain.ComponentCachePort
	var coordinatorCache coordinator_domain.BuildResultCachePort
	var introspectionCache coordinator_domain.IntrospectionCachePort

	if caches != nil && caches.inspectorProvider != nil {
		inspectorProvider = caches.inspectorProvider
	} else {

		cacheSandbox, sandboxErr := safedisk.NewNoOpSandbox(absCacheDir, safedisk.ModeReadWrite)
		require.NoError(tb, sandboxErr, "creating cache sandbox")
		inspectorProvider = inspector_adapters.NewFlatBufferCache(cacheSandbox)
	}

	if caches != nil && caches.annotatorCache != nil {
		annotatorCache = caches.annotatorCache
	} else {
		annotatorCache = annotator_adapters.NewComponentCache()
	}

	fileCacheService := newTestCacheService()

	if caches != nil && caches.coordinatorCache != nil {
		coordinatorCache = caches.coordinatorCache
	} else {
		var cacheErr error
		coordinatorCache, cacheErr = coordinator_adapters.NewBuildResultCache(context.Background(), fileCacheService)
		require.NoError(tb, cacheErr)
	}

	if caches != nil && caches.introspectionCache != nil {
		introspectionCache = caches.introspectionCache
	} else {
		var cacheErr error
		introspectionCache, cacheErr = coordinator_adapters.NewIntrospectionCache(context.Background(), fileCacheService)
		require.NoError(tb, cacheErr)
	}

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspectorProvider),
		inspector_domain.WithBuilderCacheKeyGenerator(&staticKeyGenerator{}),
	)

	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         bootstrap.NewAnnotatorPathsConfig(serverConfig),
		Cache:               annotatorCache,
		CompilationLogLevel: 0,
		CollectionService:   nil,
	})

	coordinatorService := coordinator_domain.NewService(
		context.Background(), annotatorService, coordinatorCache, introspectionCache, fsReader, resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)
	tb.Cleanup(func() { coordinatorService.Shutdown(context.Background()) })

	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	registerEmitter := generator_adapters.NewRegisterEmitter(fsWriter)
	generatorService, err := generator_domain.NewGeneratorService(context.Background(), bootstrap.NewGeneratorPathsConfig(serverConfig), "en", generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		ManifestEmitter:    manifestEmitter,
		Coordinator:        coordinatorService,
		Resolver:           resolver,
		RegisterEmitter:    registerEmitter,
		CodeEmitterFactory: codeEmitterFactory,
		SEOService:         nil,
	})
	require.NoError(tb, err)

	entryPoints := discoverEntryPoints(tb, resolver, *serverConfig)
	require.NotEmpty(tb, entryPoints, "Benchmark discovery failed: no entry points found in src directory")

	allSourceContents := make(map[string][]byte)
	_ = filepath.WalkDir(absSrcDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			content, _ := os.ReadFile(path)
			allSourceContents[path] = content
		}
		return nil
	})

	createdCaches := &cacheBundle{
		inspectorProvider:  inspectorProvider,
		annotatorCache:     annotatorCache,
		coordinatorCache:   coordinatorCache,
		introspectionCache: introspectionCache,
	}

	return &serviceStack{
		generatorService:  generatorService,
		entryPoints:       entryPoints,
		allSourceContents: allSourceContents,
	}, createdCaches
}

func runGeneration(tb testing.TB, stack *serviceStack) {
	tb.Helper()
	_, _, err := stack.generatorService.GenerateProject(context.Background(), stack.entryPoints)
	if err != nil {
		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](err); ok {
			formattedDiags := annotator_domain.FormatAllDiagnostics(semanticErr.Diagnostics, stack.allSourceContents)
			tb.Fatalf("GenerateProject failed with semantic errors:\n%s", formattedDiags)
		} else {
			tb.Fatalf("GenerateProject failed with a fatal error: %v", err)
		}
	}
}

func BenchmarkGeneratorService_Cold(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {

				stack, _ := createServiceStack(b, tc, nil)
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_WarmInspector(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {

			b.StopTimer()

			initialStack, warmCaches := createServiceStack(b, tc, nil)
			runGeneration(b, initialStack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				stack, _ := createServiceStack(b, tc, &cacheBundle{
					inspectorProvider: warmCaches.inspectorProvider,
				})
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_WarmAnnotator(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			initialStack, warmCaches := createServiceStack(b, tc, nil)
			runGeneration(b, initialStack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				stack, _ := createServiceStack(b, tc, &cacheBundle{
					annotatorCache: warmCaches.annotatorCache,
				})
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_WarmCoordinator(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			initialStack, warmCaches := createServiceStack(b, tc, nil)
			runGeneration(b, initialStack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				stack, _ := createServiceStack(b, tc, &cacheBundle{
					coordinatorCache:   warmCaches.coordinatorCache,
					introspectionCache: warmCaches.introspectionCache,
				})
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_FullyWarm(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			initialStack, warmCaches := createServiceStack(b, tc, nil)
			runGeneration(b, initialStack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				stack, _ := createServiceStack(b, tc, warmCaches)
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_SteadyState(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			stack, _ := createServiceStack(b, tc, nil)

			runGeneration(b, stack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				runGeneration(b, stack)
			}
		})
	}
}

func discoverEntryPoints(t testing.TB, resolver resolver_domain.ResolverPort, serverConfig bootstrap.ServerConfig) []annotator_dto.EntryPoint {
	t.Helper()
	var entryPoints []annotator_dto.EntryPoint
	moduleName := resolver.GetModuleName()
	baseDir := resolver.GetBaseDir()

	discover := func(sourceDir string, isPotentiallyPage bool) {
		sourceRoot := filepath.Join(baseDir, sourceDir)
		if _, err := os.Stat(sourceRoot); os.IsNotExist(err) {
			return
		}

		_ = filepath.WalkDir(sourceRoot, func(absPath string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
				relPathToBase, _ := filepath.Rel(baseDir, absPath)
				pikoPath := filepath.ToSlash(filepath.Join(moduleName, relPathToBase))

				entryPoints = append(entryPoints, annotator_dto.EntryPoint{
					Path:     pikoPath,
					IsPage:   isPotentiallyPage,
					IsPublic: isPotentiallyPage,
				})
			}
			return nil
		})
	}

	discover(*serverConfig.Paths.PagesSourceDir, true)
	discover(*serverConfig.Paths.PartialsSourceDir, false)
	return entryPoints
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func BenchmarkGeneratorService_FileCacheWarmInspector(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {

				stack, _ := createServiceStackWithFileCache(b, tc, nil)
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_FileCacheFullyWarm(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			initialStack, warmCaches := createServiceStackWithFileCache(b, tc, nil)
			runGeneration(b, initialStack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {

				stack, _ := createServiceStackWithFileCache(b, tc, warmCaches)
				runGeneration(b, stack)
			}
		})
	}
}

func BenchmarkGeneratorService_FileCacheSteadyState(b *testing.B) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(b, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		b.Run(tc.Name, func(b *testing.B) {
			b.StopTimer()

			stack, _ := createServiceStackWithFileCache(b, tc, nil)
			runGeneration(b, stack)

			b.ReportAllocs()
			b.StartTimer()

			for b.Loop() {
				runGeneration(b, stack)
			}
		})
	}
}

func TestProfile_ColdGeneration(t *testing.T) {
	tc := testCase{
		Name: "03_large_complex_dashboard",
		Path: filepath.Join("./testdata", "03_large_complex_dashboard"),
	}

	if _, err := os.Stat(tc.Path); os.IsNotExist(err) {
		t.Fatalf("Test fixture not found: %s", tc.Path)
	}

	stack, _ := createServiceStack(t, tc, nil)
	runGeneration(t, stack)
}

func TestGenerateCacheFiles(t *testing.T) {
	testdataRoot := "./testdata"
	entries, err := os.ReadDir(testdataRoot)
	require.NoError(t, err, "Failed to read testdata directory")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
		}

		t.Run(tc.Name, func(t *testing.T) {
			generateCacheForTestCase(t, tc)
		})
	}
}

func generateCacheForTestCase(t *testing.T, tc testCase) {
	t.Helper()
	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	cacheDir := filepath.Join(tc.Path, "cache")
	absCacheDir, err := filepath.Abs(cacheDir)
	require.NoError(t, err)

	err = os.MkdirAll(absCacheDir, 0755)
	require.NoError(t, err, "Failed to create cache directory")

	localResolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	cacheResolver, err := resolver_adapters.NewGoModuleCacheResolverWithWorkingDir(absSrcDir)
	require.NoError(t, err, "failed to create go module cache resolver")
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)

	cacheSandbox, err := safedisk.NewNoOpSandbox(absCacheDir, safedisk.ModeReadWrite)
	require.NoError(t, err, "creating cache sandbox")
	inspectorProvider := inspector_adapters.NewFlatBufferCache(cacheSandbox)

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspectorProvider),
		inspector_domain.WithBuilderCacheKeyGenerator(&staticKeyGenerator{}),
	)

	fsReader := &realFSReader{}
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)
	serverConfig := &bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:           &absSrcDir,
			PartialsSourceDir: new("partials"),
			PagesSourceDir:    new("pages"),
		},
	}

	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         bootstrap.NewAnnotatorPathsConfig(serverConfig),
		Cache:               annotator_adapters.NewComponentCache(),
		CompilationLogLevel: 0,
		CollectionService:   nil,
	})

	genCacheService := newTestCacheService()
	genCoordinatorCache, err := coordinator_adapters.NewBuildResultCache(context.Background(), genCacheService)
	require.NoError(t, err)
	genIntrospectionCache, err := coordinator_adapters.NewIntrospectionCache(context.Background(), genCacheService)
	require.NoError(t, err)
	coordinatorService := coordinator_domain.NewService(
		context.Background(),
		annotatorService,
		genCoordinatorCache,
		genIntrospectionCache,
		fsReader,
		resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)
	defer coordinatorService.Shutdown(context.Background())

	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	persistSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadWrite)
	defer persistSandbox.Close()
	generatorService, err := generator_domain.NewGeneratorService(context.Background(), bootstrap.NewGeneratorPathsConfig(serverConfig), "en", generator_domain.GeneratorPorts{
		FSWriter:           &generator_domain.MockFSWriter{},
		ManifestEmitter:    generator_adapters.NewJSONManifestEmitter(persistSandbox),
		Coordinator:        coordinatorService,
		Resolver:           resolver,
		RegisterEmitter:    generator_adapters.NewRegisterEmitter(&generator_domain.MockFSWriter{}),
		CodeEmitterFactory: codeEmitterFactory,
		SEOService:         nil,
	})
	require.NoError(t, err)

	entryPoints := discoverEntryPoints(t, resolver, *serverConfig)
	require.NotEmpty(t, entryPoints, "No entry points found")

	_, _, err = generatorService.GenerateProject(context.Background(), entryPoints)
	if err != nil {
		if semanticErr, ok := errors.AsType[*annotator_domain.SemanticError](err); ok {
			allSourceContents := make(map[string][]byte)
			_ = filepath.WalkDir(absSrcDir, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					content, _ := os.ReadFile(path)
					allSourceContents[path] = content
				}
				return nil
			})
			formattedDiags := annotator_domain.FormatAllDiagnostics(semanticErr.Diagnostics, allSourceContents)
			t.Fatalf("GenerateProject failed with semantic errors:\n%s", formattedDiags)
		} else {
			t.Fatalf("GenerateProject failed: %v", err)
		}
	}

	cacheFile := filepath.Join(absCacheDir, fmt.Sprintf("typedata-%s.bin", benchmarkCacheKey))
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatalf("Cache file was not created: %s", cacheFile)
	}

	t.Logf("Successfully generated cache file: %s", cacheFile)
}
