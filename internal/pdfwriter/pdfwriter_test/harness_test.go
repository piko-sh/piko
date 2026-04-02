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

package pdfwriter_test

import (
	"bytes"
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
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testSpec struct {
	Description     string  `json:"description"`
	PageWidth       float64 `json:"pageWidth"`
	PageHeight      float64 `json:"pageHeight"`
	DefaultFontSize float64 `json:"defaultFontSize"`
}

type realFSReader struct{}

func (*realFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func runPdfPaintTest(t *testing.T, testDirectory string) {
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

	fontEntries := []layouter_dto.FontEntry{
		{Family: fonts.NotoSansFamilyName, Weight: 400, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansRegularTTF},
		{Family: fonts.NotoSansFamilyName, Weight: 700, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansBoldTTF},
	}
	fontMetrics, fontMetricsError := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if fontMetricsError != nil {
		t.Fatalf("failed to create font metrics: %v", fontMetricsError)
	}
	imageResolver := &layouter_adapters.MockImageResolver{}

	layouterAdapter := pdfwriter_adapters.NewLayouterAdapter(fontMetrics, imageResolver)

	layoutConfig := layouter_dto.LayoutConfig{
		Page:            pageConfig,
		DefaultFontSize: rootFontSize,
	}

	layoutResult, err := layouterAdapter.Layout(ctx, tree, styling, layoutConfig)
	if err != nil {
		t.Fatalf("layout failed: %v", err)
	}

	painter := pdfwriter_domain.NewPdfPainter(pageConfig.Width, pageConfig.Height, fontEntries, nil)
	var buffer bytes.Buffer
	if err := painter.Paint(ctx, layoutResult, &buffer); err != nil {
		t.Fatalf("PDF paint failed: %v", err)
	}

	actual := buffer.Bytes()

	if !strings.HasPrefix(string(actual), "%PDF") {
		t.Fatal("PDF output does not start with %PDF header")
	}

	goldenDirectory := filepath.Join(testDirectory, "golden")
	goldenPath := filepath.Join(goldenDirectory, "golden.pdf")

	if *updateGoldenFiles {
		if err := os.MkdirAll(goldenDirectory, 0o755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, actual, 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		t.Logf("updated golden file: %s (%d bytes)", goldenPath, len(actual))
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file not found at %s (run with -update to generate): %v", goldenPath, err)
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf("PDF output does not match golden file (%d bytes actual vs %d bytes expected)",
			len(actual), len(expected))
	}
}
