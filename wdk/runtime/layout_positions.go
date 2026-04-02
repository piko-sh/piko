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

package runtime

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// defaultPointsPerPixel is the default CSS points-per-pixel ratio.
	defaultPointsPerPixel = 0.75

	// defaultRootFontSize is the default root font size in points.
	defaultRootFontSize = 12.0
)

// LayoutPositionRect represents a positioned element's bounding box in pixels.
type LayoutPositionRect struct {
	// TextRects holds per-line bounding boxes for text content within this
	// element. Empty for elements that contain no text runs.
	TextRects []LayoutPositionRect `json:"textRects,omitempty"`

	// X holds the horizontal position of the element in pixels.
	X float64 `json:"x"`

	// Y holds the vertical position of the element in pixels.
	Y float64 `json:"y"`

	// Width holds the element width in pixels.
	Width float64 `json:"width"`

	// Height holds the element height in pixels.
	Height float64 `json:"height"`

	// PageIndex is the zero-based page index. Only set when pagination is enabled.
	PageIndex int `json:"pageIndex,omitempty"`
}

// LayoutPositionConfig configures the layout position extraction pipeline.
type LayoutPositionConfig struct {
	// ManifestPath is the path to the compiled manifest file (e.g. dist/manifest.bin).
	ManifestPath string

	// RequestPath is the URL route to look up in the manifest (e.g. "/main").
	RequestPath string

	// AttributeName is the HTML attribute used to identify elements (e.g. "data-layout-id").
	AttributeName string

	// FontData is the raw TTF or OTF font bytes used for text measurement.
	FontData []byte

	// ExtraStylesheets holds additional CSS stylesheets (e.g. CSS reset) applied
	// before the page's own styling during CSS resolution.
	ExtraStylesheets []string

	// PointsPerPixel is the CSS points-per-pixel ratio. Defaults to 0.75 if zero.
	PointsPerPixel float64

	// RootFontSize is the root font size in points. Defaults to 12.0 if zero.
	RootFontSize float64

	// PageWidthPx is the page width in CSS pixels. Only used when Paginate is true.
	PageWidthPx float64

	// PageHeightPx is the page height in CSS pixels. Only used when Paginate is true.
	PageHeightPx float64

	// PageMarginPx is the uniform page margin in CSS pixels. Only used when Paginate is true.
	PageMarginPx float64

	// ViewportWidth is the viewport width in pixels.
	ViewportWidth int

	// ViewportHeight is the viewport height in pixels.
	ViewportHeight int

	// Paginate enables pagination after layout. When true, positions are
	// emitted as page-relative coordinates and include a PageIndex.
	Paginate bool
}

// ExtractLayoutPositions loads a compiled manifest, gets the post-codegen AST
// for the specified page, runs the CSS resolution and layout pipeline, and
// returns element positions keyed by the specified attribute value.
//
// The compiled dist/ package must have been imported (so that init() has
// registered the BuildAST functions) before calling this function.
//
// Takes config (LayoutPositionConfig) which specifies the manifest, page,
// viewport, font, and attribute settings.
//
// Returns map[string]LayoutPositionRect which maps attribute values to
// positioned bounding boxes in pixel coordinates.
// Returns error when any step of the pipeline fails.
func ExtractLayoutPositions(config LayoutPositionConfig) (map[string]LayoutPositionRect, error) {
	ctx := context.Background()

	tree, styling, err := loadLayoutAST(ctx, config)
	if err != nil {
		return nil, err
	}

	pointsPerPixel := config.PointsPerPixel
	if pointsPerPixel == 0 {
		pointsPerPixel = defaultPointsPerPixel
	}

	rootFontSize := config.RootFontSize
	if rootFontSize == 0 {
		rootFontSize = defaultRootFontSize
	}

	rootBox, fontMetrics, err := buildLayoutBoxTree(ctx, config, tree, styling, pointsPerPixel, rootFontSize)
	if err != nil {
		return nil, err
	}

	_ = layouter_domain.LayoutBoxTree(ctx, rootBox, fontMetrics)

	var pageGeometry layouter_domain.PageGeometry
	if config.Paginate {
		pageGeometry = layouter_domain.UniformPageGeometry(
			(config.PageHeightPx - 2*config.PageMarginPx) * pointsPerPixel,
		)
		_ = layouter_domain.Paginate(ctx, rootBox, pageGeometry)
	}

	positions := make(map[string]LayoutPositionRect)
	collectLayoutPositions(rootBox, config.AttributeName, pointsPerPixel, config.Paginate, pageGeometry, positions)

	return positions, nil
}

// loadLayoutAST loads the manifest and retrieves the AST and styling for
// the configured page.
//
// Takes config (LayoutPositionConfig) which specifies the manifest path and
// request path.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns string which is the CSS styling for the page.
// Returns error when the manifest cannot be loaded or the page is not found.
func loadLayoutAST(ctx context.Context, config LayoutPositionConfig) (*ast_domain.TemplateAST, string, error) {
	provider := generator_adapters.NewFlatBufferManifestProvider(config.ManifestPath)
	store, err := templater_adapters.NewManifestStore(ctx, provider)
	if err != nil {
		return nil, "", fmt.Errorf("loading manifest: %w", err)
	}

	entry, found := store.GetPageEntry(config.RequestPath)
	if !found {
		entry, found = findPageEntryByRoute(store, config.RequestPath)
		if !found {
			return nil, "", fmt.Errorf("page entry not found for path %q (available keys: %v)", config.RequestPath, store.GetKeys())
		}
	}

	requestData := templater_dto.NewRequestDataBuilder().
		WithContext(ctx).
		Build()
	tree, _ := entry.GetASTRoot(requestData)
	styling := entry.GetStyling()

	return tree, styling, nil
}

// buildLayoutBoxTree creates font metrics, resolves CSS, and builds the box
// tree for layout position extraction.
//
// Takes config (LayoutPositionConfig) which provides font data and viewport
// dimensions.
// Takes tree (*ast_domain.TemplateAST) which is the parsed template AST.
// Takes styling (string) which is the CSS stylesheet to resolve.
// Takes pointsPerPixel (float64) which is the CSS points-per-pixel ratio.
// Takes rootFontSize (float64) which is the root font size in points.
//
// Returns *layouter_domain.LayoutBox which is the root of the box tree.
// Returns *layouter_adapters.GoTextFontMetrics which provides font measurement.
// Returns error when font creation, style resolution, or box tree building
// fails.
func buildLayoutBoxTree(
	ctx context.Context,
	config LayoutPositionConfig,
	tree *ast_domain.TemplateAST,
	styling string,
	pointsPerPixel, rootFontSize float64,
) (*layouter_domain.LayoutBox, *layouter_adapters.GoTextFontMetrics, error) {
	fontMetrics, err := layouter_adapters.NewGoTextFontMetrics([]layouter_dto.FontEntry{
		{
			Family: "NotoSans",
			Weight: 400,
			Style:  int(layouter_domain.FontStyleNormal),
			Data:   config.FontData,
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating font metrics: %w", err)
	}

	viewportWidthPoints := float64(config.ViewportWidth) * pointsPerPixel
	viewportHeightPoints := float64(config.ViewportHeight) * pointsPerPixel

	cssAdapter := layouter_adapters.NewCSSResolutionAdapter(rootFontSize)
	cssAdapter.SetViewportDimensions(viewportWidthPoints, viewportHeightPoints)
	styleMap, pseudoStyleMap, err := cssAdapter.ResolveStyles(ctx, tree, styling, config.ExtraStylesheets)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving styles: %w", err)
	}

	imageResolver := &layouter_adapters.MockImageResolver{}

	rootBox, err := layouter_domain.BuildBoxTree(
		ctx,
		tree,
		styleMap,
		pseudoStyleMap,
		imageResolver,
		viewportWidthPoints,
		viewportHeightPoints,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("building box tree: %w", err)
	}

	return rootBox, fontMetrics, nil
}

// collectLayoutPositions walks the box tree recursively and records a
// LayoutPositionRect for each element that carries the named attribute.
//
// Takes box (*layouter_domain.LayoutBox) which is the current box to inspect.
// Takes attributeName (string) which is the HTML attribute to match.
// Takes pointsPerPixel (float64) which converts points to pixels.
// Takes paginate (bool) which enables page-relative coordinate adjustment.
// Takes pageGeometry (layouter_domain.PageGeometry) which provides page
// dimensions when paginate is true.
// Takes positions (map[string]LayoutPositionRect) which accumulates results.
func collectLayoutPositions(
	box *layouter_domain.LayoutBox,
	attributeName string,
	pointsPerPixel float64,
	paginate bool,
	pageGeometry layouter_domain.PageGeometry,
	positions map[string]LayoutPositionRect,
) {
	if box.SourceNode != nil {
		collectMatchingAttribute(box, attributeName, pointsPerPixel, paginate, pageGeometry, positions)
	}

	for _, child := range box.Children {
		collectLayoutPositions(child, attributeName, pointsPerPixel, paginate, pageGeometry, positions)
	}
}

// collectMatchingAttribute searches a box's source node attributes for a
// matching attribute and, if found, records a LayoutPositionRect in positions.
//
// Takes box (*layouter_domain.LayoutBox) which is the box to inspect.
// Takes attributeName (string) which is the attribute name to match.
// Takes pointsPerPixel (float64) which converts points to pixels.
// Takes paginate (bool) which enables page-relative coordinate adjustment.
// Takes pageGeometry (layouter_domain.PageGeometry) which provides page
// dimensions when paginate is true.
// Takes positions (map[string]LayoutPositionRect) which accumulates results.
func collectMatchingAttribute(
	box *layouter_domain.LayoutBox,
	attributeName string,
	pointsPerPixel float64,
	paginate bool,
	pageGeometry layouter_domain.PageGeometry,
	positions map[string]LayoutPositionRect,
) {
	for i := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[i].Name != attributeName {
			continue
		}
		yPt := box.BorderBoxY()
		if paginate && pageGeometry.DefaultHeight > 0 {
			yPt = yPt + box.PageYOffset - pageGeometry.PageStart(box.PageIndex)
		}

		rect := LayoutPositionRect{
			X:      box.BorderBoxX() / pointsPerPixel,
			Y:      yPt / pointsPerPixel,
			Width:  box.BorderBoxWidth() / pointsPerPixel,
			Height: box.BorderBoxHeight() / pointsPerPixel,
		}
		if paginate {
			rect.PageIndex = box.PageIndex
		}
		rect.TextRects = collectTextRunRects(box, pointsPerPixel)
		positions[box.SourceNode.Attributes[i].Value] = rect
		return
	}
}

// collectTextRunRects walks the children of box and returns a
// LayoutPositionRect for each BoxTextRun descendant, representing per-line
// text bounding boxes. The walk stops at children that have their own
// data-layout-id (those are tracked independently).
//
// Takes box (*layouter_domain.LayoutBox) which is the parent box to walk.
// Takes pointsPerPixel (float64) which converts points to pixels.
//
// Returns []LayoutPositionRect which holds per-line text bounding boxes.
func collectTextRunRects(box *layouter_domain.LayoutBox, pointsPerPixel float64) []LayoutPositionRect {
	var rects []LayoutPositionRect
	collectTextRunRectsRecursive(box, pointsPerPixel, &rects)
	return rects
}

// collectTextRunRectsRecursive recursively collects text run
// bounding boxes from box children, appending results to
// rects.
//
// Takes box (*layouter_domain.LayoutBox) which is the parent
// box to walk.
// Takes pointsPerPixel (float64) which converts points to
// pixels.
// Takes rects (*[]LayoutPositionRect) which accumulates the
// per-line text bounding boxes.
func collectTextRunRectsRecursive(box *layouter_domain.LayoutBox, pointsPerPixel float64, rects *[]LayoutPositionRect) {
	for _, child := range box.Children {
		if child.Type == layouter_domain.BoxListMarker {
			continue
		}
		if child.Type == layouter_domain.BoxTextRun && !child.IsListMarker {
			*rects = append(*rects, LayoutPositionRect{
				X:      child.ContentX / pointsPerPixel,
				Y:      child.ContentY / pointsPerPixel,
				Width:  child.ContentWidth / pointsPerPixel,
				Height: child.ContentHeight / pointsPerPixel,
			})
			continue
		}

		if hasLayoutIDAttribute(child) {
			continue
		}
		collectTextRunRectsRecursive(child, pointsPerPixel, rects)
	}
}

// hasLayoutIDAttribute reports whether a box's source node carries a
// "data-layout-id" attribute, indicating it is tracked independently.
//
// Takes box (*layouter_domain.LayoutBox) which is the box to inspect.
//
// Returns bool which is true when the box has a data-layout-id attribute.
func hasLayoutIDAttribute(box *layouter_domain.LayoutBox) bool {
	if box.SourceNode == nil {
		return false
	}
	for j := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[j].Name == "data-layout-id" {
			return true
		}
	}
	return false
}

// findPageEntryByRoute converts a request URL such as
// "/main" to the manifest page key format and looks it up
// in the store.
//
// Takes store (templater_domain.ManifestStoreView) which provides page lookups.
// Takes requestPath (string) which is the URL path to convert.
//
// Returns templater_domain.PageEntryView which is the matched page entry.
// Returns bool which is true when the page was found.
func findPageEntryByRoute(store templater_domain.ManifestStoreView, requestPath string) (templater_domain.PageEntryView, bool) {
	pageName := strings.TrimPrefix(requestPath, "/")
	if pageName == "" {
		pageName = "index"
	}
	pageKey := "pages/" + pageName + ".pk"
	return store.GetPageEntry(pageKey)
}
