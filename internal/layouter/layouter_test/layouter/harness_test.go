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

package layouter_test

import (
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/resolver/resolver_adapters"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testSpec struct {
	Description     string  `json:"description"`
	PageWidth       float64 `json:"pageWidth"`
	PageHeight      float64 `json:"pageHeight"`
	DefaultFontSize float64 `json:"defaultFontSize"`
	ShouldError     bool    `json:"shouldError"`
}

type realFSReader struct{}

func (*realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func runLayoutTest(t *testing.T, testDirectory string) {
	t.Helper()
	ctx := context.Background()

	specPath := filepath.Join(testDirectory, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read testspec.json: %v", err)
	}

	var spec testSpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("failed to parse testspec.json: %v", err)
	}

	srcDirectory := filepath.Join(testDirectory, "src")
	absoluteSrcDirectory, err := filepath.Abs(srcDirectory)
	if err != nil {
		t.Fatalf("failed to resolve absolute path: %v", err)
	}

	resolver := resolver_adapters.NewLocalModuleResolver(absoluteSrcDirectory)
	if err := resolver.DetectLocalModule(ctx); err != nil {
		t.Fatalf("failed to detect local module: %v", err)
	}

	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderCSS,
		&esbuildconfig.Options{MinifyWhitespace: true, MinifySyntax: true},
		resolver,
	)

	inspectorManager := inspector_domain.NewTypeBuilder(
		inspector_dto.Config{BaseDir: absoluteSrcDirectory, ModuleName: resolver.GetModuleName()},
		inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)),
	)

	cache := annotator_adapters.NewComponentCache()
	annotatorService, err := annotator_domain.NewAnnotatorService(ctx, &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            &realFSReader{},
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(inspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         annotator_domain.AnnotatorPathsConfig{},
		Cache:               cache,
		CompilationLogLevel: slog.LevelInfo,
	})
	if err != nil {
		t.Fatalf("failed to create annotator service: %v", err)
	}

	moduleName := resolver.GetModuleName()
	entryPointModulePath := filepath.ToSlash(filepath.Join(moduleName, "main.pk"))

	annotationResult, _, annotateError := annotatorService.Annotate(ctx, entryPointModulePath, true)
	if annotateError != nil {
		if spec.ShouldError {
			return
		}
		t.Fatalf("annotation failed: %v", annotateError)
	}

	tree := annotationResult.AnnotatedAST
	styling := annotationResult.StyleBlock

	pageConfig := layouter_dto.PageA4
	if spec.PageWidth > 0 {
		pageConfig.Width = spec.PageWidth
	}
	if spec.PageHeight > 0 {
		pageConfig.Height = spec.PageHeight
	}

	rootFontSize := 12.0
	if spec.DefaultFontSize > 0 {
		rootFontSize = spec.DefaultFontSize
	}

	cssAdapter := layouter_adapters.NewCSSResolutionAdapter(rootFontSize)
	cssAdapter.SetViewportDimensions(pageConfig.Width, pageConfig.Height)
	styleMap, pseudoStyleMap, err := cssAdapter.ResolveStyles(ctx, tree, styling, nil)
	if err != nil {
		if spec.ShouldError {
			return
		}
		t.Fatalf("style resolution failed: %v", err)
	}

	fontMetrics := &layouter_adapters.MockFontMetrics{}
	imageResolver := &layouter_adapters.MockImageResolver{}

	rootBox, err := layouter_domain.BuildBoxTree(
		ctx,
		tree,
		styleMap,
		pseudoStyleMap,
		imageResolver,
		pageConfig.Width,
		pageConfig.Height,
	)
	if err != nil {
		if spec.ShouldError {
			return
		}
		t.Fatalf("box tree construction failed: %v", err)
	}

	_ = layouter_domain.LayoutBoxTree(context.Background(), rootBox, fontMetrics)

	actual := layouter_domain.SerialiseLayoutBoxToGoFileContent(rootBox, "test")

	goldenDirectory := filepath.Join(testDirectory, "golden")
	goldenPath := filepath.Join(goldenDirectory, "golden.go")

	if *updateGoldenFiles {
		if err := os.MkdirAll(goldenDirectory, 0o755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expectedBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file not found at %s (run with -update to generate): %v", goldenPath, err)
	}

	expected := string(expectedBytes)
	if normalise(actual) != normalise(expected) {
		t.Errorf("layout output does not match golden file\n\n--- EXPECTED ---\n%s\n--- ACTUAL ---\n%s",
			expected, actual)
	}
}

func normalise(text string) string {
	return strings.TrimSpace(text) + "\n"
}
