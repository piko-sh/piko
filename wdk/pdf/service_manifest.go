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
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync/atomic"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_svgwriter"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultRegularFontWeight is the CSS font-weight of the default NotoSans regular face.
	defaultRegularFontWeight = 400

	// defaultBoldFontWeight is the CSS font-weight of the default NotoSans bold face.
	defaultBoldFontWeight = 700
)

var (
	// ErrManifestNotLoaded is returned when a service created by NewServiceFromManifest
	// renders before its manifest has been loaded with Load.
	ErrManifestNotLoaded = errors.New("pdf: manifest not loaded; call Load before rendering")
)

// serviceConfig accumulates options for NewServiceFromManifest.
type serviceConfig struct {
	// extraFontEntries holds additional font entries beyond the defaults.
	extraFontEntries []layouter_dto.FontEntry

	// layoutLimits bounds the work of every layout the service performs.
	layoutLimits layouter_dto.LayoutLimits

	// maxImagePixels caps the pixel area of any embedded image. Zero uses the default.
	maxImagePixels int

	// excludeDefaultBold prevents registration of the default NotoSans-Bold font.
	excludeDefaultBold bool

	// svgVectorRendering enables SVG sizing for replaced elements (the SVG image resolver).
	// Inline <svg> painting is always enabled via the SVG renderer and is not gated by this
	// flag.
	svgVectorRendering bool
}

// ServiceOption configures a service created by NewServiceFromManifest.
type ServiceOption func(*serviceConfig)

// ManifestService is a PDF writer service that renders templates from a compiled
// manifest. It is created by NewServiceFromManifest and must be loaded with Load before
// rendering; rendering earlier fails with ErrManifestNotLoaded.
type ManifestService struct {
	Service

	// runner resolves PDF templates from the loaded manifest.
	runner *manifestTemplateRunner

	// manifestPath is the path to the compiled manifest file.
	manifestPath string
}

// NewServiceFromManifest creates a PDF writer service for a compiled manifest file.
//
// This is intended for tests and CLI tools that need to render PDFs without the full
// daemon bootstrap. Creation performs no I/O. Call Load to read the manifest before
// rendering.
//
// NotoSans regular (400) is always registered. NotoSans bold (700) is registered unless
// WithExcludeDefaultBold is used. Additional fonts can be added via WithFont and
// WithVariableFont.
//
// Takes manifestPath (string) which is the path to the compiled manifest file (e.g.
// "dist/manifest.bin").
// Takes opts (...ServiceOption) which configure fonts, rendering and limits.
//
// Returns *ManifestService which is the configured PDF writer service.
// Returns error when the manifest path is empty or font metrics cannot be created.
func NewServiceFromManifest(manifestPath string, opts ...ServiceOption) (*ManifestService, error) {
	if manifestPath == "" {
		return nil, errors.New("pdf: manifest path must not be empty")
	}

	config := &serviceConfig{}
	for _, opt := range opts {
		opt(config)
	}

	fontEntries := buildFontEntries(config)

	fontMetrics, err := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if err != nil {
		return nil, fmt.Errorf("pdf: creating font metrics: %w", err)
	}

	svgData := driven_svgwriter.NewDataURISVGDataAdapter()

	var imageResolver layouter_domain.ImageResolverPort = &layouter_adapters.MockImageResolver{}
	if config.svgVectorRendering {
		imageResolver = driven_svgwriter.NewSVGImageResolver(imageResolver, svgData)
	}

	runner := &manifestTemplateRunner{}

	return &ManifestService{
		Service: pdfwriter_domain.NewPdfWriterService(
			runner,
			pdfwriter_adapters.NewLayouterAdapter(fontMetrics, imageResolver),
			fontEntries,
			pdfwriter_adapters.NewDataURIImageDataAdapter(nil),
			fontMetrics,
			pdfwriter_domain.WithSVGRenderer(driven_svgwriter.New(), svgData),
			pdfwriter_domain.WithLayoutLimits(config.layoutLimits),
			pdfwriter_domain.WithMaxImagePixels(config.maxImagePixels),
		),
		runner:       runner,
		manifestPath: manifestPath,
	}, nil
}

// Load reads the compiled manifest through a read-only sandbox rooted at the manifest's
// directory and makes its PDF templates available for rendering. Calling Load again
// replaces the loaded manifest, for example after a rebuild.
//
// Returns error when the manifest directory cannot be opened or the manifest cannot be
// read or parsed.
func (s *ManifestService) Load(ctx context.Context) (err error) {
	sandbox, err := safedisk.NewSandbox(filepath.Dir(s.manifestPath), safedisk.ModeReadOnly)
	if err != nil {
		return fmt.Errorf("pdf: opening manifest directory for %q: %w", s.manifestPath, err)
	}
	defer func() {
		if closeErr := sandbox.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("pdf: closing manifest sandbox: %w", closeErr))
		}
	}()

	provider := generator_adapters.NewFlatBufferManifestProvider(
		s.manifestPath, generator_adapters.WithFlatBufferManifestSandbox(sandbox),
	)
	if err := s.runner.load(ctx, provider); err != nil {
		return fmt.Errorf("pdf: loading manifest from %q: %w", s.manifestPath, err)
	}
	return nil
}

// manifestTemplateRunner adapts a ManifestStore to implement
// pdfwriter_domain.TemplateRunnerPort for standalone rendering.
type manifestTemplateRunner struct {
	// store holds the loaded manifest for page entry lookups, or nil before loading.
	store atomic.Pointer[templater_adapters.ManifestStore]
}

// RunPdfWithProps loads the page entry from the manifest store and returns the AST tree
// and styling.
//
// Takes templatePath (string) which is the page path to look up.
// Takes props (any) which holds optional component props for rendering.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns string which is the CSS styling for the page.
// Returns error when the manifest is not loaded, the page entry is not found, or the
// template reports a render error.
func (r *manifestTemplateRunner) RunPdfWithProps(
	ctx context.Context,
	templatePath string,
	_ *http.Request,
	props any,
) (*ast_domain.TemplateAST, string, error) {
	store := r.store.Load()
	if store == nil {
		return nil, "", ErrManifestNotLoaded
	}
	entry, found := store.GetPageEntry(templatePath)
	if !found {
		return nil, "", fmt.Errorf("PDF entry not found for path %q (available keys: %v)",
			templatePath, store.GetKeys())
	}

	requestData := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		Build()
	defer requestData.Release()

	tree, metadata := entry.GetASTRootWithProps(requestData, props)
	if metadata.RenderError != nil {
		return nil, "", fmt.Errorf("rendering PDF template %q: %w", templatePath, metadata.RenderError)
	}

	return tree, entry.GetStyling(), nil
}

// load reads the manifest from provider and makes it the runner's current store.
//
// Takes provider (generator_domain.ManifestProviderPort) which supplies the manifest.
//
// Returns error when the manifest cannot be loaded.
func (r *manifestTemplateRunner) load(ctx context.Context, provider generator_domain.ManifestProviderPort) error {
	store, err := templater_adapters.NewManifestStore(ctx, provider)
	if err != nil {
		return err
	}
	r.store.Store(store)
	return nil
}

// WithFont registers an additional font for PDF rendering. NotoSans is always included as
// the default; use this option to add extra font families, weights, or styles.
//
// Takes family (string) which is the CSS font-family name.
// Takes weight (int) which is the CSS font-weight value (100-900).
// Takes style (int) which is the font style variant (0 = normal, 1 = italic).
// Takes data ([]byte) which is the raw TTF or OTF font bytes.
//
// Returns ServiceOption which adds the font entry.
func WithFont(family string, weight int, style int, data []byte) ServiceOption {
	return func(c *serviceConfig) {
		c.extraFontEntries = append(c.extraFontEntries, newStaticFontEntry(family, weight, style, data))
	}
}

// WithVariableFont registers an OpenType variable font for PDF rendering, where a single
// file covers a continuous weight range and replaces multiple static font files.
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
			Weight:     0,
		})
	}
}

// WithExcludeDefaultBold prevents the default NotoSans-Bold font from being registered,
// forcing the PDF painter to synthesise bold via fill+stroke.
//
// Returns ServiceOption which disables the default bold font.
func WithExcludeDefaultBold() ServiceOption {
	return func(c *serviceConfig) {
		c.excludeDefaultBold = true
	}
}

// WithSVGVectorRendering enables native SVG-to-PDF vector rendering. SVG images embedded
// as data URIs will be rendered as crisp vector paths instead of rasterised images.
//
// Returns ServiceOption which enables vector SVG rendering.
func WithSVGVectorRendering() ServiceOption {
	return func(c *serviceConfig) {
		c.svgVectorRendering = true
	}
}

// WithLayoutLimits bounds the work of every layout the service performs, such as the raw
// HTML size, nesting depth, box count, table and grid sizes and page count. Unset fields
// keep their built-in defaults, and a render can override individual fields with
// RenderBuilder.WithLayoutLimits.
//
// Takes limits (LayoutLimits) which holds the service-wide limits.
//
// Returns ServiceOption which applies the limits.
func WithLayoutLimits(limits LayoutLimits) ServiceOption {
	return func(c *serviceConfig) {
		c.layoutLimits = limits
	}
}

// WithMaxImagePixels caps the pixel area (width times height) of any image the service's
// renders embed. Non-positive values keep the built-in default.
//
// Takes pixels (int) which is the maximum pixel area of one image.
//
// Returns ServiceOption which applies the cap.
func WithMaxImagePixels(pixels int) ServiceOption {
	return func(c *serviceConfig) {
		c.maxImagePixels = pixels
	}
}

// buildFontEntries assembles font registrations for a service by selecting the default
// NotoSans regular face, the default NotoSans bold face unless excluded, then any extra
// fonts.
//
// Takes config (*serviceConfig) which holds the accumulated service options.
//
// Returns []layouter_dto.FontEntry which lists the fonts to register, defaults first.
func buildFontEntries(config *serviceConfig) []layouter_dto.FontEntry {
	fontEntries := make([]layouter_dto.FontEntry, 0, 2+len(config.extraFontEntries))
	fontEntries = append(fontEntries, newStaticFontEntry(
		fonts.NotoSansFamilyName, defaultRegularFontWeight, int(layouter_domain.FontStyleNormal), fonts.NotoSansRegularTTF,
	))
	if !config.excludeDefaultBold {
		fontEntries = append(fontEntries, newStaticFontEntry(
			fonts.NotoSansFamilyName, defaultBoldFontWeight, int(layouter_domain.FontStyleNormal), fonts.NotoSansBoldTTF,
		))
	}
	return append(fontEntries, config.extraFontEntries...)
}

// newStaticFontEntry creates a font registration for a static (non-variable) font file
// covering a single weight.
//
// Takes family (string) which is the CSS font-family name.
// Takes weight (int) which is the CSS font-weight value (100-900).
// Takes style (int) which is the font style variant (0 = normal, 1 = italic).
// Takes data ([]byte) which is the raw TTF or OTF font bytes.
//
// Returns layouter_dto.FontEntry which describes the static font.
func newStaticFontEntry(family string, weight int, style int, data []byte) layouter_dto.FontEntry {
	return layouter_dto.FontEntry{
		Family:     family,
		Weight:     weight,
		Style:      style,
		Data:       data,
		WeightMin:  0,
		WeightMax:  0,
		IsVariable: false,
	}
}
