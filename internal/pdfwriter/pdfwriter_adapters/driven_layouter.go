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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

const (
	// autoHeightSentinel is a very tall page height (~350 metres) used when AutoHeight is
	// enabled. Content flows into this space without triggering pagination.
	autoHeightSentinel = 1e6

	// defaultRootFontSize is the root font size used when no explicit default is configured
	// in the layout config.
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

// NewLayouterAdapter creates a new layouter adapter with the given font metrics and image
// resolver implementations.
//
// Takes fontMetrics (layouter_domain.FontMetricsPort) which provides text measurement.
// Takes imageResolver (layouter_domain.ImageResolverPort) which provides image
// dimensions.
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

// Layout resolves CSS, builds the box tree, performs layout, and returns the result
// containing the positioned box tree.
//
// Takes tree (*ast_domain.TemplateAST) which is the template AST to lay out.
// Takes styling (string) which is the CSS from the template's style block.
// Takes config (layouter_dto.LayoutConfig) which specifies page dimensions, font settings
// and layout limits.
//
// Returns result (*layouter_dto.LayoutResult) which contains the positioned box tree and
// fragment tree.
// Returns err (error) when CSS resolution, box tree construction, or layout fails, when a
// layout limit is breached, or when a panic is recovered.
func (adapter *LayouterAdapter) Layout(
	ctx context.Context,
	tree *ast_domain.TemplateAST,
	styling string,
	config layouter_dto.LayoutConfig,
) (result *layouter_dto.LayoutResult, err error) {
	defer func() { pdfwriter_domain.StorePanicAsError(ctx, "layout", recover(), &err) }()

	rootFontSize := config.DefaultFontSize
	if rootFontSize <= 0 {
		rootFontSize = defaultRootFontSize
	}

	layoutHeight := config.Page.Height
	if config.Page.AutoHeight {
		layoutHeight = autoHeightSentinel
	}

	contentWidth := config.Page.ContentAreaWidth()
	limits := layouter_domain.NewLimitTracker(config.Limits)

	if err := expandRawHTMLNodes(ctx, tree, limits.Limits()); err != nil {
		return nil, err
	}

	cssAdapter := layouter_adapters.NewCSSResolutionAdapter(rootFontSize)
	cssAdapter.SetViewportDimensions(contentWidth, layoutHeight)
	cssAdapter.SetLimitTracker(limits)

	styleMap, pseudoStyleMap, err := cssAdapter.ResolveStyles(ctx, tree, styling, config.Stylesheets)
	if err != nil {
		return nil, fmt.Errorf("CSS resolution failed: %w", err)
	}
	if err := cssStageError(ctx, limits); err != nil {
		return nil, err
	}

	rootBox, err := layouter_domain.BuildBoxTree(ctx, layouter_domain.BoxTreeInput{
		Tree:           tree,
		Styles:         styleMap,
		PseudoStyles:   pseudoStyleMap,
		ImageResolver:  adapter.imageResolver,
		Limits:         limits,
		ViewportWidth:  contentWidth,
		ViewportHeight: layoutHeight,
	})
	if err != nil {
		return nil, fmt.Errorf("box tree construction failed: %w", err)
	}

	fragment, err := layouter_domain.LayoutBoxTree(ctx, rootBox, adapter.fontMetrics, limits)
	if err != nil {
		return nil, fmt.Errorf("box tree layout failed: %w", err)
	}

	pages, err := buildPages(ctx, rootBox, config, limits)
	if err != nil {
		return nil, fmt.Errorf("pagination failed: %w", err)
	}

	return &layouter_dto.LayoutResult{
		RootBox:         rootBox,
		RootFragment:    fragment,
		Pages:           pages,
		DiagnosticCount: 0,
	}, nil
}

// rawHTMLExpander materialises raw HTML into template nodes while enforcing the raw HTML
// size, nesting depth and node count limits across the whole document.
type rawHTMLExpander struct {
	// err holds the first limit breach or cancellation, after which expansion stops.
	err error

	// limits holds the resolved layout limits.
	limits layouter_dto.LayoutLimits

	// rawBytes holds the total size of raw HTML parsed so far.
	rawBytes int

	// nodeCount holds the number of document nodes visited so far.
	nodeCount int
}

// expandList returns the node list with any raw HTML expanded in place, recursing into
// children.
//
// It checks ctx and the limits before every list so neither a deeply nested template nor
// hostile raw HTML can perform unbounded work; once expansion stops the remaining nodes
// are returned unexpanded and the error is reported by the caller.
//
// Takes nodes ([]*ast_domain.TemplateNode) which is the node list to expand.
// Takes depth (int) which is the nesting depth of the nodes in the list.
//
// Returns []*ast_domain.TemplateNode which is the node list with raw HTML expanded.
func (e *rawHTMLExpander) expandList(ctx context.Context, nodes []*ast_domain.TemplateNode, depth int) []*ast_domain.TemplateNode {
	if !e.canExpand(ctx, depth) {
		return nodes
	}
	out := make([]*ast_domain.TemplateNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if !e.countNode() {
			return nodes
		}
		switch {
		case node.NodeType == ast_domain.NodeRawHTML && node.InnerHTML != "":
			out = append(out, e.parseFragment(ctx, node.InnerHTML, depth)...)
			continue
		case node.NodeType == ast_domain.NodeElement && node.InnerHTML != "" && len(node.Children) == 0:
			node.Children = e.parseFragment(ctx, node.InnerHTML, depth+1)
			node.InnerHTML = ""
		default:
			node.Children = e.expandList(ctx, node.Children, depth+1)
		}
		out = append(out, node)
	}
	return out
}

// canExpand reports whether expansion may continue into a list at the given depth,
// recording cancellation or a nesting breach.
//
// Takes depth (int) which is the nesting depth of the list about to be expanded.
//
// Returns bool which is false once expansion must stop.
func (e *rawHTMLExpander) canExpand(ctx context.Context, depth int) bool {
	if e.err != nil {
		return false
	}
	if err := ctx.Err(); err != nil {
		e.err = fmt.Errorf("raw HTML expansion cancelled: %w", err)
		return false
	}
	if depth > e.limits.MaxNestingDepth {
		e.err = fmt.Errorf("document nests deeper than %d levels: %w",
			e.limits.MaxNestingDepth, layouter_dto.ErrNestingTooDeep)
		return false
	}
	return true
}

// countNode counts one more document node against the node limit.
//
// Returns bool which is false when the document holds too many nodes.
func (e *rawHTMLExpander) countNode() bool {
	e.nodeCount++
	if e.nodeCount > e.limits.MaxBoxNodes {
		e.err = fmt.Errorf("document holds more than %d nodes: %w",
			e.limits.MaxBoxNodes, layouter_dto.ErrTooManyBoxes)
		return false
	}
	return true
}

// parseFragment parses a raw HTML string into template nodes using Piko's own AST parser,
// then recursively expands any nested raw HTML in the result.
//
// The fragment's size, and an estimate of how many elements it would create, are checked
// against the remaining budgets before it is parsed, so a small fragment of many tiny
// elements cannot make the parser allocate far beyond the node limit. On parse failure it
// returns nil (the content is dropped rather than failing the render, matching the
// behaviour for unparseable raw HTML).
//
// Takes raw (string) which is the raw HTML fragment to parse.
// Takes depth (int) which is the nesting depth the parsed nodes are placed at.
//
// Returns []*ast_domain.TemplateNode which holds the parsed and expanded nodes, or nil on
// parse failure or when a limit is breached.
func (e *rawHTMLExpander) parseFragment(ctx context.Context, raw string, depth int) []*ast_domain.TemplateNode {
	if e.err != nil {
		return nil
	}
	if len(raw) > e.limits.MaxRawHTMLBytes-e.rawBytes {
		e.err = fmt.Errorf("raw HTML totals more than %d bytes: %w",
			e.limits.MaxRawHTMLBytes, layouter_dto.ErrRawHTMLTooLarge)
		return nil
	}
	e.rawBytes += len(raw)
	if e.nodeCount+estimateElementCount(raw) > e.limits.MaxBoxNodes {
		e.err = fmt.Errorf("raw HTML holds more than %d elements: %w",
			e.limits.MaxBoxNodes, layouter_dto.ErrTooManyBoxes)
		return nil
	}

	parsed, err := ast_domain.Parse(ctx, raw, "<p-html>", nil)
	if err != nil || parsed == nil {
		return nil
	}
	return e.expandList(ctx, parsed.RootNodes, depth)
}

// expandRawHTMLNodes walks the template AST and materialises raw HTML into real child
// nodes so the box builder can lay them out.
//
// Takes tree (*ast_domain.TemplateAST) which is the template AST to expand in place.
// Takes limits (layouter_dto.LayoutLimits) which bounds the raw HTML size, nesting depth
// and node count; zero fields use the defaults.
//
// Returns error when ctx is cancelled or the document breaches a limit.
func expandRawHTMLNodes(ctx context.Context, tree *ast_domain.TemplateAST, limits layouter_dto.LayoutLimits) error {
	if tree == nil {
		return nil
	}
	expander := &rawHTMLExpander{
		err:       nil,
		limits:    limits.Resolved(),
		rawBytes:  0,
		nodeCount: 0,
	}
	tree.RootNodes = expander.expandList(ctx, tree.RootNodes, 0)
	return expander.err
}

// cssStageError reports a cancelled context or breached layout limit that stopped style
// resolution early.
//
// Takes limits (*layouter_domain.LimitTracker) which holds any recorded limit breach.
//
// Returns error which wraps the cancellation or breach, or nil when resolution completed.
func cssStageError(ctx context.Context, limits *layouter_domain.LimitTracker) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("CSS resolution cancelled: %w", err)
	}
	if err := limits.Err(); err != nil {
		return fmt.Errorf("CSS resolution failed: %w", err)
	}
	return nil
}

// buildPages produces the page output list. For auto-height layouts a single page is
// measured from the content extent; otherwise the box tree is paginated uniformly.
//
// Takes rootBox (*layouter_domain.LayoutBox) which is the laid-out box tree.
// Takes config (layouter_dto.LayoutConfig) which specifies page dimensions and
// auto-height settings.
// Takes limits (*layouter_domain.LimitTracker) which enforces the page and box limits.
//
// Returns []layouter_dto.PageOutput which holds the generated page list.
// Returns error when pagination is cancelled or breaches a limit.
func buildPages(
	ctx context.Context,
	rootBox *layouter_domain.LayoutBox,
	config layouter_dto.LayoutConfig,
	limits *layouter_domain.LimitTracker,
) ([]layouter_dto.PageOutput, error) {
	if config.Page.AutoHeight {
		extent := layouter_domain.MeasureContentExtent(rootBox)
		measuredHeight := extent + config.Page.MarginTop + config.Page.MarginBottom
		return []layouter_dto.PageOutput{{
			Index:  0,
			Width:  config.Page.Width,
			Height: measuredHeight,
		}}, nil
	}

	maxPage, err := layouter_domain.Paginate(
		ctx, rootBox, layouter_domain.UniformPageGeometry(config.Page.ContentAreaHeight()), limits,
	)
	if err != nil {
		return nil, err
	}

	pages := make([]layouter_dto.PageOutput, maxPage+1)
	for i := range pages {
		pages[i] = layouter_dto.PageOutput{
			Index:  i,
			Width:  config.Page.Width,
			Height: config.Page.Height,
		}
	}

	return pages, nil
}

// estimateElementCount estimates how many elements parsing raw HTML would create by
// counting the opening angle brackets that do not start an end tag. Text containing a
// literal "<" can only raise the estimate, so the estimate never lets an oversized
// fragment through.
//
// Takes raw (string) which is the raw HTML fragment.
//
// Returns int which is the estimated number of elements.
func estimateElementCount(raw string) int {
	return strings.Count(raw, "<") - strings.Count(raw, "</")
}
