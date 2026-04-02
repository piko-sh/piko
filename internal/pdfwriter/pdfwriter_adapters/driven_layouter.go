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

package pdfwriter_adapters

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

const (
	// autoHeightSentinel is a very tall page height (~350 metres) used
	// when AutoHeight is enabled. Content flows into this space without
	// triggering pagination.
	autoHeightSentinel = 1e6

	// defaultRootFontSize is the root font size used when no explicit
	// default is configured in the layout config.
	defaultRootFontSize = 12.0
)

// LayouterAdapter wraps the layouter's domain functions behind the
// pdfwriter_domain.LayoutPort interface.
type LayouterAdapter struct {
	// fontMetrics provides text measurement during layout.
	fontMetrics layouter_domain.FontMetricsPort

	// imageResolver provides intrinsic dimensions for replaced elements.
	imageResolver layouter_domain.ImageResolverPort
}

// NewLayouterAdapter creates a new layouter adapter with the given
// font metrics and image resolver implementations.
//
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides
// text measurement.
// Takes imageResolver (layouter_domain.ImageResolverPort) which
// provides image dimensions.
//
// Returns *LayouterAdapter which implements pdfwriter_domain.LayoutPort.
func NewLayouterAdapter(
	fontMetrics layouter_domain.FontMetricsPort,
	imageResolver layouter_domain.ImageResolverPort,
) *LayouterAdapter {
	return &LayouterAdapter{
		fontMetrics:   fontMetrics,
		imageResolver: imageResolver,
	}
}

// Layout resolves CSS, builds the box tree, performs layout, and returns
// the result containing the positioned box tree.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes tree (*ast_domain.TemplateAST) which is the template AST to
// lay out.
// Takes styling (string) which is the CSS from the template's style
// block.
// Takes config (layouter_dto.LayoutConfig) which specifies page
// dimensions and font settings.
//
// Returns *layouter_dto.LayoutResult which contains the positioned
// box tree and fragment tree.
// Returns error when CSS resolution, box tree construction, or layout
// fails.
func (adapter *LayouterAdapter) Layout(
	ctx context.Context,
	tree *ast_domain.TemplateAST,
	styling string,
	config layouter_dto.LayoutConfig,
) (*layouter_dto.LayoutResult, error) {
	rootFontSize := config.DefaultFontSize
	if rootFontSize <= 0 {
		rootFontSize = defaultRootFontSize
	}

	layoutHeight := config.Page.Height
	if config.Page.AutoHeight {
		layoutHeight = autoHeightSentinel
	}

	cssAdapter := layouter_adapters.NewCSSResolutionAdapter(rootFontSize)
	cssAdapter.SetViewportDimensions(config.Page.Width, layoutHeight)

	styleMap, pseudoStyleMap, err := cssAdapter.ResolveStyles(ctx, tree, styling, config.Stylesheets)
	if err != nil {
		return nil, fmt.Errorf("CSS resolution failed: %w", err)
	}

	rootBox, err := layouter_domain.BuildBoxTree(
		ctx, tree, styleMap, pseudoStyleMap,
		adapter.imageResolver, config.Page.Width, layoutHeight,
	)
	if err != nil {
		return nil, fmt.Errorf("box tree construction failed: %w", err)
	}

	fragment := layouter_domain.LayoutBoxTree(ctx, rootBox, adapter.fontMetrics)
	pages := buildPages(ctx, rootBox, config)

	return &layouter_dto.LayoutResult{
		RootBox:      rootBox,
		RootFragment: fragment,
		Pages:        pages,
	}, nil
}

// buildPages produces the page output list. For auto-height layouts a
// single page is measured from the content extent; otherwise the box
// tree is paginated uniformly.
//
// Takes rootBox (*layouter_domain.LayoutBox) which is the laid-out box tree.
// Takes config (layouter_dto.LayoutConfig) which specifies page dimensions
// and auto-height settings.
//
// Returns []layouter_dto.PageOutput which holds the generated page list.
func buildPages(
	ctx context.Context,
	rootBox *layouter_domain.LayoutBox,
	config layouter_dto.LayoutConfig,
) []layouter_dto.PageOutput {
	if config.Page.AutoHeight {
		extent := layouter_domain.MeasureContentExtent(rootBox)
		measuredHeight := extent + config.Page.MarginBottom
		return []layouter_dto.PageOutput{{
			Index:  0,
			Width:  config.Page.Width,
			Height: measuredHeight,
		}}
	}

	maxPage := layouter_domain.Paginate(ctx, rootBox, layouter_domain.UniformPageGeometry(config.Page.Height))

	pages := make([]layouter_dto.PageOutput, maxPage+1)
	for i := range pages {
		pages[i] = layouter_dto.PageOutput{
			Index:  i,
			Width:  config.Page.Width,
			Height: config.Page.Height,
		}
	}

	return pages
}
