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

// Package pikotest_test provides an integration test suite that validates the
// pikotest testing framework works end-to-end. Each test scenario has a PK
// source file, generated Go code, and colocated _pikotest_test.go files.
//
// Phase 1 verifies golden file generation (same as pikotest_meta).
// Phase 2 runs `go test` as a subprocess to validate that the colocated
// tests actually pass or fail as expected.
//
// Run with: go test ./internal/pikotest/pikotest_test/...
// Update golden files: go test ./internal/pikotest/pikotest_test/... -update
package pikotest_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
	"piko.sh/piko/internal/testutil/leakcheck"
	"piko.sh/piko/wdk/safedisk"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files by regenerating from PK sources")

type TestSpec struct {
	Description             string                     `json:"description"`
	ExpectedFailureContains string                     `json:"expectedFailureContains,omitempty"`
	EntryPoints             []annotator_dto.EntryPoint `json:"entryPoints"`
	ShouldError             bool                       `json:"shouldError,omitempty"`
	ShouldFail              bool                       `json:"shouldFail,omitempty"`
}

type testCase struct {
	Name string
	Path string
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func TestPikotestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pikotest integration tests in short mode")
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

		tc := testCase{
			Name: entry.Name(),
			Path: filepath.Join(testdataRoot, entry.Name()),
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

			var spec TestSpec
			specBytes, readErr := os.ReadFile(specPath)
			require.NoError(t, readErr, "Failed to read testspec.json")
			require.NoError(t, json.Unmarshal(specBytes, &spec), "Failed to parse testspec.json")

			runGoldenFileTest(t, tc, spec)

			runSubprocessTest(t, tc, spec)
		})
	}
}

func runGoldenFileTest(t *testing.T, tc testCase, spec TestSpec) {
	t.Helper()

	srcDir := filepath.Join(tc.Path, "src")

	genDir := t.TempDir()
	require.NoError(t, copyDir(srcDir, genDir), "Failed to copy src to generation directory")
	rewriteGoModReplace(t, genDir)

	absGenDir, err := filepath.Abs(genDir)
	require.NoError(t, err)

	resolver := resolver_adapters.NewLocalModuleResolver(absGenDir)
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
	testSandbox, _ := safedisk.NewNoOpSandbox(absGenDir, safedisk.ModeReadWrite)
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
		inspector_dto.Config{BaseDir: absGenDir, ModuleName: moduleName},
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
	)
	defer coordinatorService.Shutdown(context.Background())
	prerenderer := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	codeEmitterFactory := generator_adapters_driven_code_emitter_go_literal.NewEmitterFactory(context.Background(), prerenderer)
	registerEmitter := generator_adapters.NewRegisterEmitter(fsWriter)
	generatorPaths := generator_domain.GeneratorPathsConfig{
		BaseDir:        absGenDir,
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

	allArtefacts, manifest, genErr := generatorService.GenerateProject(context.Background(), pikoEntryPoints)
	if spec.ShouldError {
		require.Error(t, genErr, "Expected error during generation")
		return
	}
	require.NoError(t, genErr, "Generation failed")
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

		relPath, relErr := filepath.Rel(absGenDir, virtualPath)
		require.NoError(t, relErr)
		goldenPath := filepath.Join(goldenDir, relPath)

		formattedBytes, fmtErr := format.Source(content)
		require.NoError(t, fmtErr, "Failed to gofmt: %s", relPath)

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

	manifestBytes, err := json.ConfigStd.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)

	goldenManifestPath := filepath.Join(tc.Path, "golden_manifest.json")
	if *updateGoldenFiles {
		require.NoError(t, os.WriteFile(goldenManifestPath, manifestBytes, 0644))
		t.Logf("Updated golden manifest: %s", goldenManifestPath)
	} else {
		expectedManifestBytes, readErr := os.ReadFile(goldenManifestPath)
		if !os.IsNotExist(readErr) {
			require.NoError(t, readErr)
			assert.JSONEq(t, string(expectedManifestBytes), string(manifestBytes),
				"Manifest mismatch. Run with -update to fix.")
		}
	}
}

func runSubprocessTest(t *testing.T, tc testCase, spec TestSpec) {
	t.Helper()

	srcDir := filepath.Join(tc.Path, "src")
	goldenDir := filepath.Join(tc.Path, "golden")

	hasTestFiles := false
	_ = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), "_pikotest_test.go") {
			hasTestFiles = true
			return filepath.SkipAll
		}
		return nil
	})

	if !hasTestFiles {
		t.Log("No _pikotest_test.go files found, skipping subprocess test")
		return
	}

	tmpDir := t.TempDir()
	require.NoError(t, copyDir(srcDir, tmpDir), "Failed to copy src to temp directory")

	if _, err := os.Stat(goldenDir); err == nil {
		require.NoError(t, copyDir(goldenDir, tmpDir), "Failed to copy golden files to temp directory")
	}

	rewriteGoModReplace(t, tmpDir)

	command := exec.Command("go", "test", "-v", "-count=1", "./...")
	command.Dir = tmpDir
	command.Env = append(os.Environ(), "GOWORK=off")

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	combined := stdout.String() + stderr.String()

	if spec.ShouldFail {
		require.Error(t, err,
			"Expected test to fail but it passed.\nStdout:\n%s\nStderr:\n%s",
			stdout.String(), stderr.String())

		if spec.ExpectedFailureContains != "" {
			require.True(t,
				strings.Contains(combined, spec.ExpectedFailureContains),
				"Expected output to contain %q.\nCombined output:\n%s",
				spec.ExpectedFailureContains, combined)
		}
	} else {
		require.NoError(t, err,
			"Test failed unexpectedly.\nStdout:\n%s\nStderr:\n%s",
			stdout.String(), stderr.String())
	}
}

func rewriteGoModReplace(t *testing.T, directory string) {
	t.Helper()

	repoRoot, err := findRepoRoot()
	require.NoError(t, err, "Failed to find repo root")

	goModPath := filepath.Join(directory, "go.mod")
	content, err := os.ReadFile(goModPath)
	require.NoError(t, err, "Failed to read go.mod in temp directory")

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "replace piko.sh/piko =>") {
			lines[i] = "replace piko.sh/piko => " + repoRoot
		}
	}

	require.NoError(t, os.WriteFile(goModPath, []byte(strings.Join(lines, "\n")), 0644))
}

func findRepoRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(directory, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			if strings.Contains(string(data), "module piko.sh/piko") {
				return directory, nil
			}
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("could not find repo root (go.mod with module piko.sh/piko)")
		}
		directory = parent
	}
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
		defer func() { _ = srcFile.Close() }()
		destFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer func() { _ = destFile.Close() }()
		_, err = io.Copy(destFile, srcFile)
		return err
	})
}

func TestMain(m *testing.M) {
	flag.Parse()

	fmt.Println("=== Pikotest Integration Suite ===")
	if *updateGoldenFiles {
		fmt.Println("Mode: Updating golden files")
	} else {
		fmt.Println("Mode: Verifying golden files")
	}

	leakcheck.VerifyTestMain(m)
}
