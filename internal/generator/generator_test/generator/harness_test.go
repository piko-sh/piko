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

package generator_test

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
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/collection/collection_adapters/driver_markdown"
	collection_adapters_driver_registry "piko.sh/piko/internal/collection/collection_adapters/driver_registry"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/component/component_adapters"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	generator_adapters_driven_code_emitter_go_literal "piko.sh/piko/internal/generator/generator_adapters/driven_code_emitter_go_literal"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/markdown/markdown_testparser"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/typegen/typegen_adapters"
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

type TopLevelTestSpec struct {
	Description   string `json:"description"`
	PackageName   string `json:"packageName,omitempty"`
	ErrorContains string `json:"errorContains,omitempty"`
	IsPage        bool   `json:"isPage"`
	ShouldError   bool   `json:"shouldError,omitempty"`
}

func runTestCase(t *testing.T, tc testCase) {
	spec := loadTestSpec(t, tc)

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	localResolver := resolver_adapters.NewLocalModuleResolver(absSrcDir)
	cacheResolver, err := resolver_adapters.NewGoModuleCacheResolverWithWorkingDir(absSrcDir)
	require.NoError(t, err, "failed to create go module cache resolver")
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)
	err = resolver.DetectLocalModule(context.Background())
	require.NoError(t, err)

	fsReader := &realFSReader{}
	fsWriter := &generator_domain.MockFSWriter{}
	testSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadOnly)
	manifestEmitter := generator_adapters.NewJSONManifestEmitter(testSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)
	annotatorPaths := annotator_domain.AnnotatorPathsConfig{
		PagesSourceDir:    "pages",
		PartialsSourceDir: "partials",
		EmailsSourceDir:   "emails",
		PartialServePath:  "/_piko/partial",
	}

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absSrcDir, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)
	annotatorComponentCache := annotator_adapters.NewComponentCache()

	providerRegistry := collection_adapters_driver_registry.NewMemoryRegistry()

	markdownParser := markdown_testparser.NewParser()
	markdownService := markdown_domain.NewMarkdownService(markdownParser, nil)
	markdownSandbox, _ := safedisk.NewNoOpSandbox(absSrcDir, safedisk.ModeReadOnly)
	markdownProvider := driver_markdown.NewMarkdownProvider("markdown", markdownSandbox, markdownService, nil, nil)
	_ = providerRegistry.Register(markdownProvider)

	collectionService := collection_domain.NewCollectionService(context.Background(), providerRegistry,
		collection_domain.WithDefaultSandbox(markdownSandbox),
	)

	componentRegistry := discoverTestComponents(t, absSrcDir, "components")

	annotatorService, _ := annotator_domain.NewAnnotatorService(context.Background(), &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         annotatorPaths,
		Cache:               annotatorComponentCache,
		CompilationLogLevel: slog.LevelInfo,
		CollectionService:   collectionService,
		ComponentRegistry:   componentRegistry,
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

	jsTempDir, err := os.MkdirTemp("", "piko-js-test-*")
	require.NoError(t, err, "Failed to create temp directory for JS")
	defer os.RemoveAll(jsTempDir)
	jsEmitter := generator_adapters.NewDiskPKJSEmitter(jsTempDir, false)

	actionGenerator := &mockActionGenerator{}

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
		PKJSEmitter:        jsEmitter,
		ActionGenerator:    actionGenerator,
		SEOService:         nil,
	})
	require.NoError(t, err)

	moduleName := resolver.GetModuleName()
	require.NotEmpty(t, moduleName, "Resolver failed to detect a module name from go.mod")
	entryPoints := discoverEntryPoints(t, resolver, "pages", "partials")
	require.NotEmpty(t, entryPoints, "Test discovery failed: no entry points found in src directory")

	allArtefacts, manifest, err := generatorService.GenerateProject(context.Background(), entryPoints)

	if spec.ShouldError {
		require.Error(t, err, "Expected generator.GenerateProject to fail, but it succeeded for: %s", tc.Name)
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains, "The error message did not contain the expected text")
		}
		return
	}
	require.NoError(t, err, "generator.GenerateProject failed unexpectedly for test case: %s", tc.Name)
	require.NotNil(t, manifest, "Manifest should not be nil on success")

	verifyGeneratedCodeBuilds(t, tc, allArtefacts, absSrcDir, actionGenerator)

	var entryPointArtefact *generator_dto.GeneratedArtefact

	for _, artefact := range allArtefacts {
		vc := artefact.Component
		if vc == nil {
			continue
		}

		formattedBytes, fmtErr := format.Source(artefact.Content)
		require.NoError(t, fmtErr, "Failed to gofmt generated code for %s", vc.HashedName)

		goldenPath := getGoldenGoPath(t, tc.Path, vc.HashedName, vc.IsPage)
		assertGoldenFile(t, goldenPath, formattedBytes, "Generated Go code for %s", vc.HashedName)

		if vc.IsPage && entryPointArtefact == nil {
			entryPointArtefact = artefact
		}
	}

	require.NotNil(t, entryPointArtefact, "Could not find the artefact for the main entry point")
	goldenASTPath := filepath.Join(tc.Path, "golden", "golden_ast.go")
	astFileContent := ast_domain.SerialiseASTToGoFileContent(ast_domain.SanitiseForEncoding(entryPointArtefact.Result.AnnotatedAST, absSrcDir), spec.PackageName)
	astFormattedBytes, fmtErr := format.Source([]byte(astFileContent))
	require.NoError(t, fmtErr)
	assertGoldenFile(t, goldenASTPath, astFormattedBytes, "Serialised AST for %s", tc.Name)

	goldenManifestPath := filepath.Join(tc.Path, "golden", "golden-manifest.json")
	manifestBytes, jsonErr := json.ConfigStd.MarshalIndent(manifest, "", "  ")
	require.NoError(t, jsonErr, "Failed to marshal manifest to JSON")
	assertGoldenFileJSON(t, goldenManifestPath, manifestBytes, "Generated manifest for %s", tc.Name)

	writtenJSFiles := jsEmitter.GetAllWrittenFiles()
	if len(writtenJSFiles) > 0 {
		for artefactID, tempPath := range writtenJSFiles {
			jsContent, readErr := os.ReadFile(tempPath)
			require.NoError(t, readErr, "Failed to read generated JS file %s", tempPath)

			goldenJSPath := filepath.Join(tc.Path, "golden", artefactID)
			assertGoldenFile(t, goldenJSPath, jsContent, "Generated JS for %s", artefactID)
		}
	}

	if len(actionGenerator.RegistryCode) > 0 {
		registryFormatted, fmtErr := format.Source(actionGenerator.RegistryCode)
		if fmtErr != nil {
			t.Logf("  - Warning: gofmt failed for registry.go: %v", fmtErr)
			registryFormatted = actionGenerator.RegistryCode
		}
		goldenRegistryPath := filepath.Join(tc.Path, "golden", "actions", "registry.go")
		assertGoldenFile(t, goldenRegistryPath, registryFormatted, "Generated action registry")

		wrappersFormatted, fmtErr := format.Source(actionGenerator.WrapperCode)
		if fmtErr != nil {
			t.Logf("  - Warning: gofmt failed for wrappers.go: %v", fmtErr)
			wrappersFormatted = actionGenerator.WrapperCode
		}
		goldenWrappersPath := filepath.Join(tc.Path, "golden", "actions", "wrappers.go")
		assertGoldenFile(t, goldenWrappersPath, wrappersFormatted, "Generated action wrappers")

		if len(actionGenerator.TypeScriptCode) > 0 {
			goldenTSPath := filepath.Join(tc.Path, "golden", "dist", "ts", "actions.gen.ts")
			assertGoldenFile(t, goldenTSPath, actionGenerator.TypeScriptCode, "Generated TypeScript actions")
		}
	}
}

func discoverEntryPoints(t *testing.T, resolver resolver_domain.ResolverPort, pagesDir, partialsDir string) []annotator_dto.EntryPoint {
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

				entryPoints = append(entryPoints, annotator_dto.EntryPoint{
					Path:     pikoPath,
					IsPage:   isPotentiallyPage,
					IsPublic: isPotentiallyPage,
				})
			}
			return nil
		})
	}

	discover(pagesDir, true)
	discover(partialsDir, false)

	return entryPoints
}

func loadTestSpec(t *testing.T, tc testCase) TopLevelTestSpec {
	t.Helper()
	var spec TopLevelTestSpec
	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	if os.IsNotExist(err) {
		return TopLevelTestSpec{
			Description: "Default test case",
			PackageName: "default_test_pkg",
			IsPage:      true,
		}
	}
	require.NoError(t, err, "Failed to read testspec.json for %s", tc.Name)
	err = json.Unmarshal(specBytes, &spec)
	require.NoError(t, err, "Failed to parse testspec.json for %s", tc.Name)
	if spec.PackageName == "" {
		spec.PackageName = "default_test_pkg"
	}
	return spec
}

func verifyGeneratedCodeBuilds(t *testing.T, tc testCase, artefacts []*generator_dto.GeneratedArtefact, srcDir string, actionGen *mockActionGenerator) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "piko-gen-test-"+tc.Name+"-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	srcModPath := filepath.Join(tc.Path, "src", "go.mod")
	goModBytes, err := os.ReadFile(srcModPath)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)

	goModContent := string(goModBytes)
	goModContent = fixReplaceDirectives(goModContent, filepath.Join(tc.Path, "src"))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644))

	srcSumPath := filepath.Join(tc.Path, "src", "go.sum")
	if goSumBytes, err := os.ReadFile(srcSumPath); err == nil {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "go.sum"), goSumBytes, 0644))
	}

	for _, artefact := range artefacts {
		if artefact.SuggestedPath == "" {
			continue
		}

		relPath, err := filepath.Rel(srcDir, artefact.SuggestedPath)
		if err != nil {

			relPath = filepath.Base(artefact.SuggestedPath)
		}
		destPath := filepath.Join(tempDir, relPath)

		formattedBytes, fmtErr := format.Source(artefact.Content)
		if fmtErr != nil {
			t.Logf("  - Warning: gofmt failed for %s: %v", relPath, fmtErr)
			formattedBytes = artefact.Content
		}

		require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0755))
		require.NoError(t, os.WriteFile(destPath, formattedBytes, 0644))
	}

	srcPackageDir := filepath.Join(tc.Path, "src", "pkg")
	if _, err := os.Stat(srcPackageDir); err == nil {
		tempPackageDir := filepath.Join(tempDir, "pkg")
		require.NoError(t, copyDir(srcPackageDir, tempPackageDir), "Failed to copy src/pkg")
	}

	srcActionsDir := filepath.Join(tc.Path, "src", "actions")
	if _, err := os.Stat(srcActionsDir); err == nil {
		tempActionsDir := filepath.Join(tempDir, "actions")
		require.NoError(t, copyDir(srcActionsDir, tempActionsDir), "Failed to copy src/actions")
	}

	if actionGen != nil && len(actionGen.RegistryCode) > 0 {
		distActionsDir := filepath.Join(tempDir, "dist", "actions")
		require.NoError(t, os.MkdirAll(distActionsDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(distActionsDir, "registry.go"), actionGen.RegistryCode, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(distActionsDir, "wrappers.go"), actionGen.WrapperCode, 0644))
	}

	goWorkOffEnv := append(os.Environ(), "GOWORK=off")

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tempDir
	tidyCmd.Env = goWorkOffEnv
	tidyOutput, err := tidyCmd.CombinedOutput()
	require.NoError(t, err, "go mod tidy failed:\n%s", string(tidyOutput))

	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = tempDir
	buildCmd.Env = goWorkOffEnv
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Generated code failed to build:\n%s", string(buildOutput))

	vetCmd := exec.Command("go", "vet", "./...")
	vetCmd.Dir = tempDir
	vetCmd.Env = goWorkOffEnv
	vetOutput, err := vetCmd.CombinedOutput()
	if err != nil {
		t.Logf("go vet reported issues:\n%s", string(vetOutput))
	}
}

func assertGoldenFile(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()
	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, actualBytes, 0644))
	}
	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	if !bytes.Equal(expectedBytes, actualBytes) {
		t.Logf("--- EXPECTED (%s) ---\n%s\n--- ACTUAL (%s) ---\n%s\n", filepath.Base(goldenPath), string(expectedBytes), filepath.Base(goldenPath), string(actualBytes))
		assert.Fail(t, fmt.Sprintf("Golden file mismatch: %s. Run with -update if this change is intentional.", goldenPath), msgAndArgs...)
	}
}

func assertGoldenFileJSON(t *testing.T, goldenPath string, actualBytes []byte, msgAndArgs ...any) {
	t.Helper()
	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0755))
		require.NoError(t, os.WriteFile(goldenPath, actualBytes, 0644))
	}
	expectedBytes, readErr := os.ReadFile(goldenPath)
	require.NoError(t, readErr, "Failed to read golden file %s. Run with -update flag to create it.", goldenPath)

	assert.JSONEq(t, string(expectedBytes), string(actualBytes), msgAndArgs...)
}

func fixReplaceDirectives(goModContent string, srcDir string) string {

	absSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return goModContent
	}

	lines := strings.Split(goModContent, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "replace ") && strings.Contains(trimmed, " => ") {
			parts := strings.SplitN(trimmed, " => ", 2)
			if len(parts) == 2 {
				targetPath := strings.TrimSpace(parts[1])

				if strings.HasPrefix(targetPath, ".") || strings.HasPrefix(targetPath, "..") {
					absPath := filepath.Join(absSrcDir, targetPath)
					absPath = filepath.Clean(absPath)
					line = parts[0] + " => " + absPath
				}
			}
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
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

func getGoldenGoPath(t *testing.T, testCasePath, hashedName string, isPage bool) string {
	t.Helper()
	directory := "partials"
	if isPage {
		directory = "pages"
	}
	return filepath.Join(testCasePath, "golden", directory, hashedName, "golden.go")
}

func discoverTestComponents(t *testing.T, baseDir, componentsDir string) annotator_domain.ComponentRegistryPort {
	t.Helper()

	if componentsDir == "" {
		return nil
	}

	absDir := filepath.Join(baseDir, componentsDir)
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return nil
	}

	registry := component_adapters.NewInMemoryRegistry()
	var registered int

	walkErr := filepath.WalkDir(absDir, func(absPath string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if !strings.HasSuffix(strings.ToLower(d.Name()), ".pkc") {
			return nil
		}

		tagName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		relPath, _ := filepath.Rel(baseDir, absPath)

		definition := component_dto.ComponentDefinition{
			TagName:    tagName,
			SourcePath: relPath,
			IsExternal: false,
		}

		if regErr := registry.Register(definition); regErr != nil {
			t.Logf("failed to register component %s: %v", tagName, regErr)
		} else {
			registered++
		}

		return nil
	})

	if walkErr != nil {
		return nil
	}

	if registered > 0 {
		return registry
	}

	return nil
}

type realFSReader struct{}

func (r *realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

type mockActionGenerator struct {
	ModuleName     string
	RegistryCode   []byte
	WrapperCode    []byte
	TypeScriptCode []byte
}

func (m *mockActionGenerator) GenerateActions(
	ctx context.Context,
	manifest *annotator_dto.ActionManifest,
	moduleName string,
	outputDir string,
) error {
	if manifest == nil || len(manifest.Actions) == 0 {
		return nil
	}

	m.ModuleName = moduleName

	specs := generator_adapters.ConvertManifestToSpecs(manifest)
	if len(specs) == 0 {
		return nil
	}

	registryEmitter := generator_adapters.NewActionRegistryEmitter()
	wrapperEmitter := generator_adapters.NewActionWrapperEmitter()
	tsEmitter := typegen_adapters.NewActionTypeScriptEmitter()

	var err error

	m.RegistryCode, err = registryEmitter.EmitRegistry(ctx, specs)
	if err != nil {
		return err
	}

	m.WrapperCode, err = wrapperEmitter.EmitWrappers(ctx, specs)
	if err != nil {
		return err
	}

	m.TypeScriptCode, err = tsEmitter.EmitTypeScript(ctx, specs)
	if err != nil {
		return err
	}

	return nil
}
