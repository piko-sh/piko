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

package generator_project_test

import (
	"context"
	"flag"
	"go/format"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
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
	"piko.sh/piko/wdk/safedisk"
)

func newTestCacheService() cache_domain.Service {
	return cache_domain.NewService("")
}

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type ProjectTestSpec struct {
	Description   string                     `json:"description"`
	ErrorContains string                     `json:"errorContains,omitempty"`
	EntryPoints   []annotator_dto.EntryPoint `json:"entryPoints"`
	ShouldError   bool                       `json:"shouldError,omitempty"`
}

func runProjectTestCase(t *testing.T, tc testCase) {
	var spec ProjectTestSpec
	specBytes, err := os.ReadFile(filepath.Join(tc.Path, "testspec.json"))
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	require.NotEmpty(t, spec.EntryPoints, "testspec.json must define at least one entry point")

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)
	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Resolver failed to detect a module name from go.mod")

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
	cssProcessor := annotator_domain.NewCSSProcessor(esbuildconfig.LoaderCSS, &esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true}, resolver)
	annotatorPaths := annotator_domain.AnnotatorPathsConfig{
		PagesSourceDir:    "pages",
		PartialsSourceDir: "partials",
	}
	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
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
		EnableDebugLogFiles: true,
		DebugLogDir:         "tmp/logs",
	})
	cacheService := newTestCacheService()
	coordinatorCache, err := coordinator_adapters.NewBuildResultCache(context.Background(), cacheService)
	require.NoError(t, err)
	introspectionCache, err := coordinator_adapters.NewIntrospectionCache(context.Background(), cacheService)
	require.NoError(t, err)
	coordinatorService := coordinator_domain.NewService(
		context.Background(),
		annotatorService, coordinatorCache, introspectionCache, fsReader, resolver,
		coordinator_domain.WithDiagnosticOutput(coordinator_adapters.NewSilentDiagnosticOutput()),
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
		require.Error(t, err, "Expected an error during project generation, but got none.")
		if spec.ErrorContains != "" {
			require.Contains(t, err.Error(), spec.ErrorContains, "Error message did not contain expected text.")
		}
		return
	}
	require.NoError(t, err, "Failed to generate all artefacts for test case: %s", tc.Name)

	require.NotNil(t, manifest, "Manifest should not be nil on a successful project build")

	require.NotEmpty(t, allArtefacts, "Generation succeeded but produced no artefacts.")
	virtualModule := allArtefacts[0].Result.VirtualModule
	require.NotNil(t, virtualModule, "VirtualModule should not be nil in the annotation result.")

	allGeneratedGoFiles := make(map[string][]byte)
	for _, artefact := range allArtefacts {
		allGeneratedGoFiles[artefact.SuggestedPath] = artefact.Content
	}

	goldenDir := filepath.Join(tc.Path, "golden")

	sortedHashes := make([]string, 0, len(virtualModule.ComponentsByHash))
	for hash := range virtualModule.ComponentsByHash {
		sortedHashes = append(sortedHashes, hash)
	}
	sort.Strings(sortedHashes)

	for _, hashName := range sortedHashes {
		vc := virtualModule.ComponentsByHash[hashName]
		virtualPath := vc.VirtualGoFilePath
		content, ok := allGeneratedGoFiles[virtualPath]
		require.True(t, ok, "No generated content found for virtual path: %s", virtualPath)

		relPath, err := filepath.Rel(absSrcDir, virtualPath)
		require.NoError(t, err, "Failed to make VirtualGoFilePath relative: %s", virtualPath)
		goldenPath := filepath.Join(goldenDir, relPath)

		formattedBytes, fmtErr := format.Source(content)
		require.NoError(t, fmtErr, "Failed to gofmt generated code for %s", relPath)

		if *updateGoldenFiles {
			require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
			require.NoError(t, os.WriteFile(goldenPath, formattedBytes, 0644))
		}

		expectedBytes, readErr := os.ReadFile(goldenPath)
		require.NoError(t, readErr, "Failed to read golden file for %s. Run with -update.", relPath)
		assert.Equal(t, string(expectedBytes), string(formattedBytes), "Generated code for %s does not match golden file. Run with -update if intentional.", relPath)
	}

	manifestBytes, err := json.ConfigStd.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	goldenManifestPath := filepath.Join(tc.Path, "golden_manifest.json")
	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenManifestPath, manifestBytes, 0644))
	}

	expectedManifestBytes, err := os.ReadFile(goldenManifestPath)
	require.NoError(t, err, "Failed to read golden_manifest.json. Run with -update.")
	assert.JSONEq(t, string(expectedManifestBytes), string(manifestBytes), "Generated manifest does not match golden file.")

	for virtualPath, content := range allGeneratedGoFiles {
		relPath, err := filepath.Rel(absSrcDir, virtualPath)
		require.NoError(t, err, "Failed to make VirtualGoFilePath relative: %s", virtualPath)
		diskPath := filepath.Join(goldenDir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(diskPath), 0755))
		require.NoError(t, os.WriteFile(diskPath, content, 0644))
	}
	verifyGoldenBuild(t, tc.Path)
}

func verifyGoldenBuild(t *testing.T, testCasePath string) {
	srcModPath := filepath.Join(testCasePath, "src", "go.mod")
	srcSumPath := filepath.Join(testCasePath, "src", "go.sum")
	goldenDir := filepath.Join(testCasePath, "golden")
	goldenModPath := filepath.Join(goldenDir, "go.mod")
	goldenSumPath := filepath.Join(goldenDir, "go.sum")
	srcPackageDir := filepath.Join(testCasePath, "src", "pkg")
	goldenPackageDir := filepath.Join(goldenDir, "pkg")

	goModBytes, err := os.ReadFile(srcModPath)
	if os.IsNotExist(err) {
		t.Logf("  - SKIPPING build verification: no go.mod found in src directory.")
		return
	}
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(goldenModPath, goModBytes, 0644))
	if goSumBytes, err := os.ReadFile(srcSumPath); err == nil {
		require.NoError(t, os.WriteFile(goldenSumPath, goSumBytes, 0644))
	}

	if _, err := os.Stat(srcPackageDir); err == nil {
		err = copyDir(srcPackageDir, goldenPackageDir)
		require.NoError(t, err, "Failed to copy src/pkg to golden/pkg")
	} else if !os.IsNotExist(err) {
		require.NoError(t, err, "Error checking for src/pkg directory")
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = goldenDir
	tidyCmd.Env = append(os.Environ(), "GOWORK=off")
	tidyOutput, err := tidyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to tidy.\nBuild Output:\n%s", string(tidyOutput))
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer destFile.Close()
		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
