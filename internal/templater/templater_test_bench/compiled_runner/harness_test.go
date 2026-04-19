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

package compiled_bench_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"piko.sh/piko/internal/json"
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

var runBenchmarks = flag.Bool("run-bench", false, "Run the compiled benchmark suite")

type TemplaterBenchSpec struct {
	Description string `json:"description"`
	TargetPage  string `json:"targetPage"`
}

var mainTestGoTemplate = template.Must(template.ParseFiles("main_test.go.tmpl"))

func runBenchmarkCase(b *testing.B, bc benchmarkCase) {

	spec := loadBenchSpec(b, bc)
	tempDir := b.TempDir()

	srcDir := filepath.Join(bc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(b, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(b, err)
	fsReader := &realFSReader{}
	benchSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadWrite)
	fsWriter := generator_adapters.NewFSWriter(benchSandbox)
	manifestEmitter := generator_adapters.NewJSONManifestEmitter(benchSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(esbuildconfig.LoaderCSS, &esbuildconfig.Options{MinifyWhitespace: true}, resolver)
	baseDir := resolver.GetBaseDir()
	serverConfig := bootstrap.ServerConfig{
		Paths: config.PathsConfig{
			BaseDir:           &baseDir,
			PagesSourceDir:    new("pages"),
			PartialsSourceDir: new("partials"),
		},
	}
	inspectorManager := inspector_domain.NewTypeBuilder(inspector_dto.Config{BaseDir: *serverConfig.Paths.BaseDir, ModuleName: resolver.GetModuleName()}, inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)))
	annotatorComponentCache := annotator_adapters.NewComponentCache()
	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         bootstrap.NewAnnotatorPathsConfig(&serverConfig),
		Cache:               annotatorComponentCache,
		CompilationLogLevel: 0,
		CollectionService:   nil,
	})
	cacheService := newTestCacheService()
	coordinatorCache, err := coordinator_adapters.NewBuildResultCache(context.Background(), cacheService)
	require.NoError(b, err)
	introspectionCache, err := coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
	require.NoError(b, err)
	coordinatorService := coordinator_domain.NewService(
		context.Background(), annotatorService, coordinatorCache, introspectionCache, fsReader, resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
	)
	b.Cleanup(func() { coordinatorService.Shutdown(context.Background()) })
	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	registerEmitter := generator_adapters.NewRegisterEmitter(fsWriter)
	generatorService, err := generator_domain.NewGeneratorService(context.Background(), bootstrap.NewGeneratorPathsConfig(&serverConfig), "en", generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		ManifestEmitter:    manifestEmitter,
		Coordinator:        coordinatorService,
		Resolver:           resolver,
		RegisterEmitter:    registerEmitter,
		CodeEmitterFactory: codeEmitterFactory,
		SEOService:         nil,
	})
	require.NoError(b, err)
	entryPoints := discoverEntryPoints(b, resolver, serverConfig)
	require.NotEmpty(b, entryPoints)

	allArtefacts, manifest, err := generatorService.GenerateProject(context.Background(), entryPoints)
	require.NoError(b, err)

	projectRoot := resolver.GetBaseDir()
	for _, artefact := range allArtefacts {
		relPath, err := filepath.Rel(projectRoot, artefact.SuggestedPath)
		require.NoError(b, err)
		outputPath := filepath.Join(tempDir, relPath)
		require.NoError(b, os.MkdirAll(filepath.Dir(outputPath), 0755))
		require.NoError(b, fsWriter.WriteFile(context.Background(), outputPath, artefact.Content))
	}
	require.NoError(b, manifestEmitter.EmitCode(context.Background(), manifest, filepath.Join(tempDir, "dist", "manifest.json")))

	moduleName := resolver.GetModuleName()
	pikoProjectRoot, err := filepath.Abs(filepath.Join(srcDir, "../../../../../../../"))
	require.NoError(b, err)
	goModContent := mustReadFile(b, filepath.Join(srcDir, "go.mod"))
	goModString := strings.ReplaceAll(string(goModContent), "replace piko.sh/piko => ../../../../../../../", fmt.Sprintf("replace piko.sh/piko => %s", pikoProjectRoot))
	require.NoError(b, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModString), 0644))
	if goSumContent, err := os.ReadFile(filepath.Join(srcDir, "go.sum")); err == nil {
		require.NoError(b, os.WriteFile(filepath.Join(tempDir, "go.sum"), goSumContent, 0644))
	}
	goWorkContent := fmt.Sprintf("go 1.25.1\n\nuse (\n\t.\n\t%s\n)\n", pikoProjectRoot)
	require.NoError(b, os.WriteFile(filepath.Join(tempDir, "go.work"), []byte(goWorkContent), 0644))

	mainTestGoPath := filepath.Join(tempDir, "main_test.go")
	mainTestGoFile, err := os.Create(mainTestGoPath)
	require.NoError(b, err)
	err = mainTestGoTemplate.Execute(mainTestGoFile, map[string]string{"ModuleName": moduleName})
	require.NoError(b, err)
	mainTestGoFile.Close()

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	tidyOutput, err := tidyCmd.CombinedOutput()
	require.NoError(b, err, "Failed to `go mod tidy` in temp directory.\nOutput:\n%s", string(tidyOutput))

	benchArgs := []string{"test", "-tags", "bench", "-bench=.", "-benchmem"}

	profileType := os.Getenv("PIKO_PROFILE_TYPE")
	profileOutputDir := os.Getenv("PIKO_PROFILE_OUTPUT_DIR")
	var profileOutPath string

	if profileType != "" && profileOutputDir != "" {

		profileOutPath = filepath.Join(tempDir, fmt.Sprintf("%s.out", profileType))
		benchArgs = append(benchArgs, fmt.Sprintf("-%sprofile=%s", profileType, profileOutPath))
	}

	benchCmd := exec.Command("go", benchArgs...)
	benchCmd.Dir = tempDir
	benchCmd.Env = append(os.Environ(), "GOWORK=off", fmt.Sprintf("PIKO_BENCH_TARGET_PAGE=%s", spec.TargetPage))

	benchCmd.Stdout = os.Stdout
	benchCmd.Stderr = os.Stderr

	err = benchCmd.Run()
	require.NoError(b, err, "Benchmark execution failed")

	if profileType != "" && profileOutputDir != "" && profileOutPath != "" {
		if _, err := os.Stat(profileOutPath); err == nil {

			if err := os.MkdirAll(profileOutputDir, 0755); err == nil {
				destPath := filepath.Join(profileOutputDir, fmt.Sprintf("%s_%s.out", bc.Name, profileType))
				profileData, err := os.ReadFile(profileOutPath)
				if err == nil {
					_ = os.WriteFile(destPath, profileData, 0644)
				}
			}
		}
	}
}

type benchmarkCase struct {
	Name string
	Path string
}

func loadBenchSpec(t testing.TB, tc benchmarkCase) TemplaterBenchSpec {
	t.Helper()
	var spec TemplaterBenchSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	return spec
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

		filepath.WalkDir(sourceRoot, func(absPath string, d os.DirEntry, walkErr error) error {
			require.NoError(t, walkErr)
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
				relPathToBase, _ := filepath.Rel(baseDir, absPath)
				pikoPath := filepath.ToSlash(filepath.Join(moduleName, relPathToBase))
				entryPoints = append(entryPoints, annotator_dto.EntryPoint{Path: pikoPath, IsPage: isPotentiallyPage, IsPublic: isPotentiallyPage})
			}
			return nil
		})
	}

	discover(*serverConfig.Paths.PagesSourceDir, true)
	discover(*serverConfig.Paths.PartialsSourceDir, false)
	return entryPoints
}

func mustReadFile(t testing.TB, path string) []byte {
	t.Helper()
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	return bytes
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
