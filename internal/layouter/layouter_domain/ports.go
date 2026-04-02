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

package layouter_domain

import (
	"context"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
)

// StyleMap maps each TemplateNode to its resolved ComputedStyle.
type StyleMap map[*ast_domain.TemplateNode]*ComputedStyle

// PseudoType identifies a CSS pseudo-element.
type PseudoType int

const (
	// PseudoNone means no pseudo-element.
	PseudoNone PseudoType = iota

	// PseudoBefore represents the ::before pseudo-element.
	PseudoBefore

	// PseudoAfter represents the ::after pseudo-element.
	PseudoAfter
)

// PseudoStyleMap maps template nodes to their pseudo-element
// computed styles.
type PseudoStyleMap map[*ast_domain.TemplateNode]map[PseudoType]*ComputedStyle

// LayoutService is the primary driving port for the layout engine.
type LayoutService interface {
	// Layout performs the full layout pipeline: style resolution, box tree
	// construction, layout, and pagination. Returns a LayoutResult
	// containing positioned boxes assigned to pages.
	Layout(ctx context.Context, tree *ast_domain.TemplateAST, styling string, config layouter_dto.LayoutConfig) (*layouter_dto.LayoutResult, error)

	// LayoutToBoxTree performs style resolution, box tree construction, and
	// layout without pagination. Returns the root LayoutBox of the
	// positioned box tree on an infinite canvas.
	LayoutToBoxTree(ctx context.Context, tree *ast_domain.TemplateAST, styling string, config layouter_dto.LayoutConfig) (*LayoutBox, error)
}

// FontMetricsPort provides text measurement and font metric queries. It is
// consumed during the layout phase for inline layout and line breaking.
type FontMetricsPort interface {
	// MeasureText returns the width in points of the given text string
	// when rendered with the specified font, size, and text direction.
	MeasureText(font FontDescriptor, size float64, text string, direction DirectionType) float64

	// ShapeText produces positioned glyphs for the given text, applying
	// kerning and ligature substitutions with the given text direction.
	ShapeText(font FontDescriptor, size float64, text string, direction DirectionType) []GlyphPosition

	// GetMetrics returns the vertical metrics (ascent, descent, line gap)
	// for the specified font at the given size.
	GetMetrics(font FontDescriptor, size float64) FontMetrics

	// ResolveFallback returns a font descriptor for a font that contains
	// the given character, walking the fallback chain if the primary font
	// lacks coverage.
	ResolveFallback(font FontDescriptor, character rune) FontDescriptor

	// SplitGraphemeClusters segments text into grapheme clusters
	// (user-perceived characters). A grapheme cluster may span
	// multiple runes, for example emoji with ZWJ sequences or
	// base characters followed by combining marks.
	SplitGraphemeClusters(text string) []string
}

// StylesheetPort resolves CSS styles for a template AST. It handles CSS
// parsing, cascade, specificity, selector matching, and produces a StyleMap.
type StylesheetPort interface {
	// ResolveStyles resolves CSS styles for every node
	// in the AST, including pseudo-element styles.
	//
	// The styling parameter is the primary CSS string
	// (e.g. from a <style> block). Additional
	// stylesheets from the LayoutConfig are also
	// applied.
	ResolveStyles(ctx context.Context, tree *ast_domain.TemplateAST, styling string, additionalStylesheets []string) (StyleMap, PseudoStyleMap, error)
}

// ImageResolverPort provides intrinsic dimensions for replaced elements such
// as images and SVGs.
type ImageResolverPort interface {
	// GetImageDimensions returns the natural width and height in points
	// for the image at the given source path or URL.
	GetImageDimensions(ctx context.Context, source string) (width, height float64, err error)
}
