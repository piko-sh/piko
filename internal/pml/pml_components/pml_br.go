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

package pml_components

import (
	"fmt"
	"slices"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// LineBreak represents the <pml-br> tag and implements Component.
// It adds a block of empty vertical space with a set height.
type LineBreak struct {
	BaseComponent
}

var _ pml_domain.Component = (*LineBreak)(nil)

// defaultBreakHeight is the default height for line breaks.
const defaultBreakHeight = "20px"

// NewLineBreak creates a new LineBreak component.
//
// A LineBreak renders a block of empty vertical space with a set height.
//
// Returns *LineBreak which is the new component ready for configuration.
func NewLineBreak() *LineBreak {
	return &LineBreak{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the PML tag identifier for line breaks.
func (*LineBreak) TagName() string {
	return "pml-br"
}

// IsEndingTag returns whether this element is self-closing.
//
// Returns bool which is always true because LineBreak is a void element
// that cannot have children.
func (*LineBreak) IsEndingTag() bool {
	return true
}

// AllowedParents returns an empty slice, allowing pml-br in any context. This
// follows PML's philosophy of being flexible about parent contexts.
//
// The component adapts its output based on context:
//   - In pml-col/pml-hero: block-level spacer with height.
//   - In pml-p/pml-li: simple <br> tag for inline line breaks.
//
// Returns []string which is empty to allow all parent contexts.
func (*LineBreak) AllowedParents() []string {
	return []string{}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute names and their type definitions.
func (*LineBreak) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrHeight:                   NewAttributeDefinition(pml_domain.TypeUnit),
		AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
		AttrPadding:                  NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-top":                NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-bottom":             NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-left":               NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-right":              NewAttributeDefinition(pml_domain.TypeUnit),
	}
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as line breaks have no
// default attributes.
func (*LineBreak) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which contains the height and padding
// properties that can be styled on this line break.
func (*LineBreak) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: AttrHeight, Target: TargetContainer},
		{Property: AttrPadding, Target: TargetContainer},
	}
}

// Transform converts a pml-br node into email-safe HTML.
//
// The output adapts based on context:
//   - With explicit height: block-level spacer (Outlook-compatible)
//   - In block context (pml-col, pml-hero): block-level spacer with default 20px
//   - In inline context (pml-p, pml-li, pml-button): simple <br> tag
//
// This context-aware behaviour follows PML's philosophy of being flexible,
// allowing pml-br to be used naturally in both layout and text contexts.
//
// Takes node (*ast_domain.TemplateNode) which is the pml-br element to
// transform.
// Takes ctx (*pml_domain.TransformationContext) which provides style
// management, parent context, and diagnostics collection.
//
// Returns *ast_domain.TemplateNode which is either a simple <br> element
// or a fragment containing Outlook table and modern div structure.
// Returns []*pml_domain.Error which contains any diagnostics collected
// during transformation.
func (*LineBreak) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager
	height, hasExplicitHeight := styles.Get(CSSHeight)

	if hasExplicitHeight {
		return renderBlockSpacer(node, ctx, height)
	}

	if isInlineContext(ctx) {
		return renderInlineBr(node, ctx)
	}

	return renderBlockSpacer(node, ctx, defaultBreakHeight)
}

// isInlineContext checks if the component is inside an inline text container.
//
// Takes ctx (*pml_domain.TransformationContext) which provides the current
// transformation state including parent component information.
//
// Returns bool which is true when the parent component treats children as raw
// inline content.
func isInlineContext(ctx *pml_domain.TransformationContext) bool {
	if ctx.ParentComponent == nil {
		return false
	}

	parentTag := ctx.ParentComponent.TagName()

	inlineParents := []string{
		"pml-p",
		"pml-li",
		"pml-button",
	}

	return slices.Contains(inlineParents, parentTag)
}

// renderBlockSpacer creates the Outlook-compatible vertical spacer.
//
// The transformation creates reliable vertical space that email clients will
// not collapse. A simple empty div with height is unreliable, especially in
// Outlook, so this uses proven techniques for cross-client compatibility.
//
// The implementation uses the "Div with Zero Font Size" technique. The core
// spacer is a div styled with height, line-height, and font-size: 0px. The
// zero font size tells email clients the element contains no renderable text,
// preventing unwanted vertical space from font metrics. A hair space entity
// (&#8202;) inside the div prevents it from being treated as empty, which some
// clients would collapse or ignore.
//
// For Outlook on Windows, a table-based approach provides reliable spacing.
// The transformation generates a td with a height attribute wrapped in
// Outlook conditional comments. A non-breaking space prevents cell collapse.
//
// Takes node (*ast_domain.TemplateNode) which is the source node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides transformation
// state and diagnostics collection.
// Takes height (string) which specifies the vertical space size as a CSS value.
//
// Returns *ast_domain.TemplateNode which is a fragment containing the spacer
// elements.
// Returns []*pml_domain.Error which contains any diagnostics collected during
// transformation.
func renderBlockSpacer(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext, height string) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	divStyles := map[string]string{
		CSSHeight:     height,
		CSSLineHeight: height,
		CSSFontSize:   ValueZeroPx,
	}

	hairSpaceNode := NewSimpleTextNode("&#8202;")
	divNode := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
	}, []*ast_domain.TemplateNode{hairSpaceNode})

	outlookTable := renderOutlookBreakTable(height)

	containerFragment := NewFragmentNode([]*ast_domain.TemplateNode{
		outlookTable,
		divNode,
	})

	transferPikoDirectives(node, containerFragment)

	return containerFragment, ctx.Diagnostics()
}

// renderInlineBr creates a simple <br> element for inline contexts.
// This is used when pml-br appears inside text containers like pml-p or pml-li.
//
// Takes node (*ast_domain.TemplateNode) which is the source node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the transformation
// state.
//
// Returns *ast_domain.TemplateNode which is the new br element with transferred
// directives.
// Returns []*pml_domain.Error which contains any diagnostics from the context.
func renderInlineBr(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	brNode := NewElementNode("br", nil, nil)
	transferPikoDirectives(node, brNode)

	return brNode, ctx.Diagnostics()
}

// renderOutlookBreakTable generates the Outlook-specific table structure for
// reliable vertical spacing in email clients.
//
// This creates a table with a fixed-height td, wrapped in conditional comments
// that only target Outlook and IE. The height attribute on the td is the most
// reliable way to create vertical space in Outlook's rendering engine.
//
// Takes height (string) which specifies the spacing height in pixel format.
//
// Returns *ast_domain.TemplateNode which contains the conditional HTML comment
// with the table structure.
func renderOutlookBreakTable(height string) *ast_domain.TemplateNode {
	heightPx := mustParsePixels(height)

	tableHTML := fmt.Sprintf(
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td style=%q height="%d">&nbsp;</td></tr></table>`,
		fmt.Sprintf("height:%s;", height),
		heightPx,
	)

	return NewRawHTMLNode(fmt.Sprintf("<!--[if mso | IE]>%s<![endif]-->", tableHTML))
}
