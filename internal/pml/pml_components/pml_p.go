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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// Paragraph implements the pml_domain.Component interface for the <pml-p> tag.
// It is the primary component for displaying text and raw HTML content.
type Paragraph struct {
	BaseComponent
}

var _ pml_domain.Component = (*Paragraph)(nil)

const (
	// defaultParagraphAlign is the default text alignment for paragraphs.
	defaultParagraphAlign = "left"

	// defaultParagraphColor is the default text colour for paragraphs.
	defaultParagraphColor = "#000000"

	// defaultParagraphFontFamily is the default font stack for paragraph elements.
	defaultParagraphFontFamily = "Ubuntu, Helvetica, Arial, sans-serif"

	// defaultParagraphFontSize is the default font size for paragraph elements.
	defaultParagraphFontSize = "13px"

	// defaultParagraphLineHeight is the default line height for paragraph text.
	defaultParagraphLineHeight = "1"

	// defaultParagraphPadding is the default padding for paragraph elements.
	defaultParagraphPadding = "10px 25px"
)

// NewParagraph creates a new paragraph component for PML markup.
//
// Returns *Paragraph which is the initialised component ready for use.
func NewParagraph() *Paragraph {
	return &Paragraph{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML tag name for the paragraph element.
//
// Returns string which is the tag name used for this component.
func (*Paragraph) TagName() string {
	return "pml-p"
}

// IsEndingTag returns true because the transformer should not recurse into
// Paragraph's children, which are treated as raw HTML content (e.g., <b>, <i>,
// <a> tags).
//
// Returns bool which is always true for Paragraph elements.
func (*Paragraph) IsEndingTag() bool {
	return true
}

// AllowedParents returns the tag names that can contain a pml-p element.
//
// Returns []string which lists valid parent tags such as pml-col and pml-hero.
func (*Paragraph) AllowedParents() []string {
	return []string{"pml-col", "pml-hero"}
}

// AllowedAttributes defines all valid attributes for the component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute names mapped to their type definitions.
func (*Paragraph) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrAlign:                    NewEnumAttributeDefinition([]string{"left", "right", ValueCentre, "justify"}),
		AttrColour:                   NewAttributeDefinition(pml_domain.TypeColor),
		CSSBackgroundColor:           NewAttributeDefinition(pml_domain.TypeColor),
		CSSFontFamily:                NewAttributeDefinition(pml_domain.TypeString),
		CSSFontSize:                  NewAttributeDefinition(pml_domain.TypeUnit),
		CSSFontStyle:                 NewAttributeDefinition(pml_domain.TypeString),
		CSSFontWeight:                NewAttributeDefinition(pml_domain.TypeString),
		CSSLineHeight:                NewAttributeDefinition(pml_domain.TypeUnit),
		"letter-spacing":             NewAttributeDefinition(pml_domain.TypeUnit),
		AttrHeight:                   NewAttributeDefinition(pml_domain.TypeUnit),
		CSSTextDecoration:            NewAttributeDefinition(pml_domain.TypeString),
		CSSTextTransform:             NewAttributeDefinition(pml_domain.TypeString),
		AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
		AttrBorder:                   NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderLeft:               NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRight:              NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderTop:                NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderBottom:             NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRadius:             NewAttributeDefinition(pml_domain.TypeUnit),
		CSSWhiteSpace:                NewAttributeDefinition(pml_domain.TypeString),
		AttrPadding:                  NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-top":                NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-bottom":             NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-left":               NewAttributeDefinition(pml_domain.TypeUnit),
		"padding-right":              NewAttributeDefinition(pml_domain.TypeUnit),
	}
}

// DefaultAttributes returns the default attribute values for pml-p.
//
// Returns map[string]string which contains the default styling attributes.
func (*Paragraph) DefaultAttributes() map[string]string {
	return map[string]string{
		AttrAlign:     defaultParagraphAlign,
		AttrColour:    defaultParagraphColor,
		CSSFontFamily: defaultParagraphFontFamily,
		CSSFontSize:   defaultParagraphFontSize,
		CSSLineHeight: defaultParagraphLineHeight,
		AttrPadding:   defaultParagraphPadding,
	}
}

// GetStyleTargets returns the style target mappings for paragraph attributes.
//
// In this model, all typography styles apply to the main content container
// (div). Padding and background are also defined as container styles, which
// signals to the parent pml-col that it should read these attributes and
// apply them to the wrapping td element.
//
// Returns []pml_domain.StyleTarget which lists the attribute-to-target
// mappings for rendering.
func (*Paragraph) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: AttrAlign, Target: TargetContainer},
		{Property: AttrColour, Target: TargetContainer},
		{Property: CSSBackgroundColor, Target: TargetContainer},
		{Property: CSSFontFamily, Target: TargetContainer},
		{Property: CSSFontSize, Target: TargetContainer},
		{Property: CSSFontStyle, Target: TargetContainer},
		{Property: CSSFontWeight, Target: TargetContainer},
		{Property: CSSLineHeight, Target: TargetContainer},
		{Property: "letter-spacing", Target: TargetContainer},
		{Property: AttrHeight, Target: TargetContainer},
		{Property: CSSTextDecoration, Target: TargetContainer},
		{Property: CSSTextTransform, Target: TargetContainer},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: AttrContainerBackgroundColor, Target: TargetContainer},
		{Property: AttrBorder, Target: TargetContainer},
		{Property: AttrBorderLeft, Target: TargetContainer},
		{Property: AttrBorderRight, Target: TargetContainer},
		{Property: AttrBorderTop, Target: TargetContainer},
		{Property: AttrBorderBottom, Target: TargetContainer},
		{Property: AttrBorderRadius, Target: TargetContainer},
		{Property: CSSWhiteSpace, Target: TargetContainer},
	}
}

// Transform converts a pml-p node into its final, email-safe HTML structure.
//
// It renders a single div element with typography styles. It does not handle
// padding; that is the responsibility of the parent pml-col component. When
// a height style is present, the output is wrapped in an Outlook-compatible
// height wrapper.
//
// Takes node (*ast_domain.TemplateNode) which is the pml-p node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides styles and
// diagnostics.
//
// Returns *ast_domain.TemplateNode which is the transformed div or wrapper.
// Returns []*pml_domain.Error which contains any diagnostics from the context.
func (*Paragraph) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	divStyles := map[string]string{
		CSSFontSize:   defaultParagraphFontSize,
		CSSLineHeight: defaultParagraphLineHeight,
		CSSColor:      defaultParagraphColor,
	}

	if styles.IsExplicit(CSSFontFamily) {
		copyStyle(styles, divStyles, CSSFontFamily)
	}
	copyStyle(styles, divStyles, CSSFontSize)
	copyStyle(styles, divStyles, CSSFontStyle)
	copyStyle(styles, divStyles, CSSFontWeight)
	copyStyle(styles, divStyles, "letter-spacing")
	copyStyle(styles, divStyles, CSSLineHeight)
	if styles.IsExplicit(AttrAlign) {
		copyStyle(styles, divStyles, AttrAlign, CSSTextAlign)
	}
	copyStyle(styles, divStyles, CSSTextDecoration)
	copyStyle(styles, divStyles, CSSTextTransform)
	copyStyle(styles, divStyles, CSSColor)
	copyStyle(styles, divStyles, CSSHeight)
	copyStyle(styles, divStyles, CSSBackgroundColor)
	copyBorderStyle(styles, divStyles, AttrBorder, CSSBorder)
	copyBorderStyle(styles, divStyles, AttrBorderLeft, CSSBorderLeft)
	copyBorderStyle(styles, divStyles, AttrBorderRight, CSSBorderRight)
	copyBorderStyle(styles, divStyles, AttrBorderTop, CSSBorderTop)
	copyBorderStyle(styles, divStyles, AttrBorderBottom, CSSBorderBottom)
	copyStyle(styles, divStyles, AttrBorderRadius)
	copyStyle(styles, divStyles, CSSWhiteSpace)

	divAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
		NewHTMLAttribute("data-pml-padding", mustGetStyle(styles, AttrPadding)),
	}

	divAttrs = buildParagraphDataAttributes(styles, divAttrs)

	processedChildren := processInlineChildren(node.Children)
	divNode := NewElementNode(ElementDiv, divAttrs, processedChildren)

	if height, ok := styles.Get(CSSHeight); ok && height != "" {
		outlookWrapper := renderOutlookParagraphHeightWrapper(height, divNode)
		transferPikoDirectives(node, outlookWrapper)
		return outlookWrapper, ctx.Diagnostics()
	}

	transferPikoDirectives(node, divNode)
	return divNode, ctx.Diagnostics()
}

// buildParagraphDataAttributes appends optional data-pml attributes to the
// given attribute slice. These attributes communicate alignment, per-side
// padding overrides, and container background colour to the parent pml-col
// component.
//
// Takes styles (*pml_domain.StyleManager) which provides the resolved style
// values.
// Takes divAttrs ([]ast_domain.HTMLAttribute) which is the base attribute
// slice to extend.
//
// Returns []ast_domain.HTMLAttribute which is the extended attribute slice.
func buildParagraphDataAttributes(styles *pml_domain.StyleManager, divAttrs []ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	if styles.IsExplicit(AttrAlign) {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-align", mustGetStyle(styles, AttrAlign)))
	}

	if paddingTop := mustGetStyle(styles, "padding-top"); paddingTop != "" {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-padding-top", paddingTop))
	}
	if paddingRight := mustGetStyle(styles, "padding-right"); paddingRight != "" {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-padding-right", paddingRight))
	}
	if paddingBottom := mustGetStyle(styles, "padding-bottom"); paddingBottom != "" {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-padding-bottom", paddingBottom))
	}
	if paddingLeft := mustGetStyle(styles, "padding-left"); paddingLeft != "" {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-padding-left", paddingLeft))
	}

	if containerBg := mustGetStyle(styles, AttrContainerBackgroundColor); containerBg != "" {
		divAttrs = append(divAttrs, NewHTMLAttribute("data-pml-container-background-color", containerBg))
	}

	return divAttrs
}

// renderOutlookParagraphHeightWrapper creates a conditional table structure
// to set a fixed height in Microsoft Outlook email clients.
//
// Takes height (string) which gives the height value in pixels.
// Takes contentNode (*ast_domain.TemplateNode) which is the content to wrap.
//
// Returns *ast_domain.TemplateNode which holds the wrapped content with
// Outlook-specific conditional comments.
func renderOutlookParagraphHeightWrapper(height string, contentNode *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	heightPx := mustParsePixels(height)

	startTag := fmt.Sprintf(
		`<!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><td height="%d" style="vertical-align:top;height:%s;"><![endif]-->`,
		heightPx,
		height,
	)
	endTag := `<!--[if mso | IE]></td></tr></table><![endif]-->`

	startComment := NewRawHTMLNode(startTag)
	endComment := NewRawHTMLNode(endTag)

	return NewFragmentNode([]*ast_domain.TemplateNode{startComment, contentNode, endComment})
}

// copyBorderStyle copies a border shorthand property to the destination map,
// falling back to reassembling from longhand properties when the shorthand is
// not present.
//
// Takes styles (*pml_domain.StyleManager) which provides the source values.
// Takes dest (map[string]string) which receives the border style.
// Takes srcKey (string) which is the shorthand attribute name (e.g. "border-left").
// Takes destKey (string) which is the CSS property name for the destination.
func copyBorderStyle(styles *pml_domain.StyleManager, dest map[string]string, srcKey string, destKey string) {
	if v, ok := styles.Get(srcKey); ok && v != "" {
		dest[destKey] = v
		return
	}

	widthKey := srcKey + "-width"
	styleKey := srcKey + "-style"
	colorKey := srcKey + "-color"

	width, _ := styles.Get(widthKey)
	borderStyle, _ := styles.Get(styleKey)
	color, _ := styles.Get(colorKey)

	var parts []string
	if width != "" {
		parts = append(parts, width)
	}
	if borderStyle != "" {
		parts = append(parts, borderStyle)
	}
	if color != "" {
		parts = append(parts, color)
	}

	if len(parts) > 0 {
		dest[destKey] = strings.Join(parts, " ")
	}
}

// processInlineChildren transforms PML elements within inline content to their
// HTML equivalents, letting pml-p contain basic PML elements like pml-br while
// still treating most content as raw HTML.
//
// Currently handles:
//   - pml-br -> <br> (simple line break, no height support in inline context)
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// process.
//
// Returns []*ast_domain.TemplateNode which contains the processed children with
// PML elements converted to HTML equivalents.
func processInlineChildren(children []*ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	if len(children) == 0 {
		return children
	}

	result := make([]*ast_domain.TemplateNode, 0, len(children))

	for _, child := range children {
		result = append(result, processInlineChild(child))
	}

	return result
}

// processInlineChild transforms a single child node, converting PML elements
// to their HTML equivalents and processing nested children.
//
// Takes child (*ast_domain.TemplateNode) which is the node to process.
//
// Returns *ast_domain.TemplateNode which is the processed node.
func processInlineChild(child *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	if child == nil {
		return nil
	}

	if child.NodeType == ast_domain.NodeElement && child.TagName == "pml-br" {
		brNode := NewElementNode("br", nil, nil)
		transferPikoDirectives(child, brNode)
		return brNode
	}

	if child.NodeType == ast_domain.NodeElement && len(child.Children) > 0 {
		child.Children = processInlineChildren(child.Children)
	}

	return child
}
