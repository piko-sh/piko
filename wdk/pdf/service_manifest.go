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

package pdf

import (
	"context"
	"fmt"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_svgwriter"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_dto"
)

// serviceConfig accumulates options for NewServiceFromManifest.
type serviceConfig struct {
	// extraFontEntries holds additional font entries beyond the defaults.
	extraFontEntries []layouter_dto.FontEntry

	// excludeDefaultBold prevents registration of the default NotoSans-Bold font.
	excludeDefaultBold bool

	// svgVectorRendering enables native SVG-to-PDF vector rendering.
	svgVectorRendering bool
}

// ServiceOption configures a service created by NewServiceFromManifest.
type ServiceOption func(*serviceConfig)

// WithFont registers an additional font for PDF rendering. NotoSans is
// always included as the default; use this option to add extra font
// families, weights, or styles.
//
// Takes family (string) which is the CSS font-family name.
// Takes weight (int) which is the CSS font-weight value (100-900).
// Takes style (int) which is the font style variant (0 = normal, 1 = italic).
// Takes data ([]byte) which is the raw TTF or OTF font bytes.
//
// Returns ServiceOption which adds the font entry.
func WithFont(family string, weight int, style int, data []byte) ServiceOption {
	return func(c *serviceConfig) {
		c.extraFontEntries = append(c.extraFontEntries, layouter_dto.FontEntry{
			Family: family,
			Weight: weight,
			Style:  style,
			Data:   data,
		})
	}
}

// WithVariableFont registers an OpenType variable font for
// PDF rendering, where a single file covers a continuous
// weight range and replaces multiple static font files.
//
// Takes family (string) which is the CSS font-family name.
// Takes weightMin (int) which is the minimum weight (e.g. 100).
// Takes weightMax (int) which is the maximum weight (e.g. 900).
// Takes style (int) which is the font style variant (0 = normal, 1 = italic).
// Takes data ([]byte) which is the raw variable TTF font bytes.
//
// Returns ServiceOption which adds the variable font entry.
func WithVariableFont(family string, weightMin, weightMax int, style int, data []byte) ServiceOption {
	return func(c *serviceConfig) {
		c.extraFontEntries = append(c.extraFontEntries, layouter_dto.FontEntry{
			Family:     family,
			Style:      style,
			Data:       data,
			IsVariable: true,
			WeightMin:  weightMin,
			WeightMax:  weightMax,
		})
	}
}

// WithExcludeDefaultBold prevents the default NotoSans-Bold font from being
// registered, forcing the PDF painter to synthesise bold via fill+stroke.
//
// Returns ServiceOption which disables the default bold font.
func WithExcludeDefaultBold() ServiceOption {
	return func(c *serviceConfig) {
		c.excludeDefaultBold = true
	}
}

// WithSVGVectorRendering enables native SVG-to-PDF vector rendering. SVG
// images embedded as data URIs will be rendered as crisp vector paths
// instead of rasterised images.
//
// Returns ServiceOption which enables vector SVG rendering.
func WithSVGVectorRendering() ServiceOption {
	return func(c *serviceConfig) {
		c.svgVectorRendering = true
	}
}

// NewServiceFromManifest creates a PDF writer service from a compiled
// manifest file. This is intended for tests and CLI tools that need to
// render PDFs without the full daemon bootstrap.
//
// NotoSans regular (400) is always registered. NotoSans bold (700) is
// registered unless WithExcludeDefaultBold is used. Additional fonts can
// be added via WithFont and WithVariableFont.
//
// Takes manifestPath (string) which is the path to the compiled manifest
// file (e.g. "dist/manifest.bin").
// Takes opts (...ServiceOption) which configure fonts and rendering.
//
// Returns Service which is the configured PDF writer service.
// Returns error when the manifest cannot be loaded or font metrics
// cannot be created.
func NewServiceFromManifest(ctx context.Context, manifestPath string, opts ...ServiceOption) (Service, error) {
	config := &serviceConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fontEntries := make([]layouter_dto.FontEntry, 0, 2+len(config.extraFontEntries))
	fontEntries = append(fontEntries, layouter_dto.FontEntry{
		Family: fonts.NotoSansFamilyName,
		Weight: 400,
		Style:  int(layouter_domain.FontStyleNormal),
		Data:   fonts.NotoSansRegularTTF,
	})
	if !config.excludeDefaultBold {
		fontEntries = append(fontEntries, layouter_dto.FontEntry{
			Family: fonts.NotoSansFamilyName,
			Weight: 700,
			Style:  int(layouter_domain.FontStyleNormal),
			Data:   fonts.NotoSansBoldTTF,
		})
	}
	fontEntries = append(fontEntries, config.extraFontEntries...)

	fontMetrics, err := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if err != nil {
		return nil, fmt.Errorf("pdf: creating font metrics: %w", err)
	}

	var imageResolver layouter_domain.ImageResolverPort = &layouter_adapters.MockImageResolver{}
	if config.svgVectorRendering {
		svgData := driven_svgwriter.NewDataURISVGDataAdapter()
		imageResolver = driven_svgwriter.NewSVGImageResolver(imageResolver, svgData)
	}

	layouter := pdfwriter_adapters.NewLayouterAdapter(fontMetrics, imageResolver)

	imageData := pdfwriter_adapters.NewDataURIImageDataAdapter(nil)

	provider := generator_adapters.NewFlatBufferManifestProvider(manifestPath)
	store, err := templater_adapters.NewManifestStore(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("pdf: loading manifest from %q: %w", manifestPath, err)
	}

	templateRunner := &manifestTemplateRunner{store: store}

	return pdfwriter_domain.NewPdfWriterService(
		templateRunner,
		layouter,
		fontEntries,
		imageData,
		fontMetrics,
	), nil
}

// manifestTemplateRunner adapts a ManifestStore to implement
// pdfwriter_domain.TemplateRunnerPort for standalone rendering.
type manifestTemplateRunner struct {
	// store holds the loaded manifest for page entry lookups.
	store *templater_adapters.ManifestStore
}

// RunPdfWithProps loads the page entry from the manifest store and returns
// the AST tree and styling.
//
// Takes templatePath (string) which is the page path to look up.
// Takes props (any) which holds optional component props for rendering.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns string which is the CSS styling for the page.
// Returns error when the page entry is not found.
func (r *manifestTemplateRunner) RunPdfWithProps(
	ctx context.Context,
	templatePath string,
	_ *http.Request,
	props any,
) (*ast_domain.TemplateAST, string, error) {
	entry, found := r.store.GetPageEntry(templatePath)
	if !found {
		return nil, "", fmt.Errorf("PDF entry not found for path %q (available keys: %v)",
			templatePath, r.store.GetKeys())
	}

	requestData := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		Build()

	var tree *ast_domain.TemplateAST
	if props != nil {
		tree, _ = entry.GetASTRootWithProps(requestData, props)
	} else {
		tree, _ = entry.GetASTRoot(requestData)
	}
	styling := entry.GetStyling()

	return tree, styling, nil
}
