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

// Package pikotest_meta provides a meta-test suite that verifies the pikotest
// testing framework works correctly. Each test scenario has a PK source file
// and pre-generated golden Go code. The tests verify that pikotest assertions
// behave as expected when testing these components.
//
// Run with: go test ./internal/integration/pikotest_meta/...
// Update golden files: go test ./internal/integration/pikotest_meta/... -update
package pikotest_meta_test

import (
	"context"
	"flag"
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/json"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/cache/cache_domain"
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
	"piko.sh/piko/internal/testutil/leakcheck"
	"piko.sh/piko/wdk/safedisk"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files by regenerating from PK sources")

type TestSpec struct {
	Description string                     `json:"description"`
	EntryPoints []annotator_dto.EntryPoint `json:"entryPoints"`
	ShouldError bool                       `json:"shouldError,omitempty"`
}

type testCase struct {
	Name string
	Path string
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func TestPikotestMeta_GoldenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pikotest meta golden file tests in short mode")
	}

	testdataRoot := "./testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Skipf("No testdata directory found: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		tc := testCase{
			Name: testCaseName,
			Path: filepath.Join(testdataRoot, testCaseName),
		}

		t.Run(tc.Name, func(t *testing.T) {
			srcPath := filepath.Join(tc.Path, "src")
			specPath := filepath.Join(tc.Path, "testspec.json")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'src' directory", tc.Name)
				return
			}
			if _, err := os.Stat(specPath); os.IsNotExist(err) {
				t.Skipf("Skipping test case '%s': missing 'testspec.json' file", tc.Name)
				return
			}

			runGoldenFileTest(t, tc)
		})
	}
}

func runGoldenFileTest(t *testing.T, tc testCase) {
	t.Helper()

	var spec TestSpec
	specBytes, err := os.ReadFile(filepath.Join(tc.Path, "testspec.json"))
	require.NoError(t, err, "Failed to read testspec.json")
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json")

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)
	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Failed to detect module name")

	pikoEntryPoints := make([]annotator_dto.EntryPoint, len(spec.EntryPoints))
	for i, ep := range spec.EntryPoints {
		pikoPath := filepath.ToSlash(filepath.Join(moduleName, ep.Path))
		pikoEntryPoints[i] = annotator_dto.EntryPoint{
			Path:     pikoPath,
			IsPage:   ep.IsPage,
			IsPublic: true,
		}
	}

	fsReader := &realFSReader{}
	testSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadWrite)
	fsWriter := generator_adapters.NewFSWriter(testSandbox)
	manifestEmitter := generator_adapters.NewJSONManifestEmitter(testSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)
	annotatorPaths := annotator_domain.AnnotatorPathsConfig{
		PagesSourceDir:    "pages",
		PartialsSourceDir: "partials",
	}
	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: moduleName},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)
	annotatorComponentCache := annotator_adapters.NewComponentCache()
	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         annotatorPaths,
		Cache:               annotatorComponentCache,
		CompilationLogLevel: slog.LevelInfo,
		CollectionService:   nil,
	})
	cacheService := cache_domain.NewService("")
	coordinatorCache, err := coordinator_adapters.NewBuildResultCache(context.Background(), cacheService)
	require.NoError(t, err)
	introspectionCache, err := coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
	require.NoError(t, err)
	coordinatorService := coordinator_domain.NewService(
		context.Background(),
		annotatorService,
		coordinatorCache,
		introspectionCache,
		fsReader,
		resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
		coordinator_domain.WithMaxBuildWaitDuration(5*time.Minute),
	)
	defer coordinatorService.Shutdown(context.Background())
	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	registerEmitter := generator_adapters.NewRegisterEmitter(fsWriter)
	generatorPaths := generator_domain.GeneratorPathsConfig{
		BaseDir:        absSrcDir,
		PagesSourceDir: "pages",
	}
	generatorService, err := generator_domain.NewGeneratorService(context.Background(), generatorPaths, "", generator_domain.GeneratorPorts{
		FSWriter:           fsWriter,
		ManifestEmitter:    manifestEmitter,
		Coordinator:        coordinatorService,
		Resolver:           resolver,
		RegisterEmitter:    registerEmitter,
		CodeEmitterFactory: codeEmitterFactory,
		SEOService:         nil,
	})
	require.NoError(t, err)

	allArtefacts, manifest, err := generatorService.GenerateProject(context.Background(), pikoEntryPoints)
	if spec.ShouldError {
		require.Error(t, err, "Expected error during generation")
		return
	}
	require.NoError(t, err, "Generation failed")
	require.NotNil(t, manifest)
	require.NotEmpty(t, allArtefacts)

	allGeneratedGoFiles := make(map[string][]byte)
	for _, artefact := range allArtefacts {
		allGeneratedGoFiles[artefact.SuggestedPath] = artefact.Content
	}

	goldenDir := filepath.Join(tc.Path, "golden")

	virtualModule := allArtefacts[0].Result.VirtualModule
	require.NotNil(t, virtualModule)

	sortedHashes := make([]string, 0, len(virtualModule.ComponentsByHash))
	for hash := range virtualModule.ComponentsByHash {
		sortedHashes = append(sortedHashes, hash)
	}
	sort.Strings(sortedHashes)

	for _, hashName := range sortedHashes {
		vc := virtualModule.ComponentsByHash[hashName]
		virtualPath := vc.VirtualGoFilePath
		content, ok := allGeneratedGoFiles[virtualPath]
		require.True(t, ok, "No generated content for: %s", virtualPath)

		relPath, err := filepath.Rel(absSrcDir, virtualPath)
		require.NoError(t, err)
		goldenPath := filepath.Join(goldenDir, relPath)

		formattedBytes, fmtErr := format.Source(content)
		require.NoError(t, fmtErr, "Failed to gofmt: %s", relPath)

		srcPath := virtualPath
		require.NoError(t, os.MkdirAll(filepath.Dir(srcPath), 0755))
		require.NoError(t, os.WriteFile(srcPath, formattedBytes, 0644))

		if *updateGoldenFiles {
			require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
			require.NoError(t, os.WriteFile(goldenPath, formattedBytes, 0644))
			t.Logf("Updated golden file: %s", goldenPath)
		} else {
			expectedBytes, readErr := os.ReadFile(goldenPath)
			if os.IsNotExist(readErr) {
				t.Fatalf("Golden file not found: %s\nRun with -update to create", goldenPath)
			}
			require.NoError(t, readErr)
			assert.Equal(t, string(expectedBytes), string(formattedBytes),
				"Golden mismatch for %s. Run with -update to fix.", relPath)
		}
	}

	manifestBytes, err := json.StdConfig().MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	goldenManifestPath := filepath.Join(tc.Path, "golden_manifest.json")
	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenManifestPath, manifestBytes, 0644))
		t.Logf("Updated golden manifest: %s", goldenManifestPath)
	} else {
		expectedManifestBytes, err := os.ReadFile(goldenManifestPath)
		if !os.IsNotExist(err) {
			require.NoError(t, err)
			assert.JSONEq(t, string(expectedManifestBytes), string(manifestBytes),
				"Manifest mismatch. Run with -update to fix.")
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	fmt.Println("=== Pikotest Meta-Test Suite ===")
	if *updateGoldenFiles {
		fmt.Println("Mode: Updating golden files")
	} else {
		fmt.Println("Mode: Verifying golden files")
	}

	leakcheck.VerifyTestMain(m)
}
