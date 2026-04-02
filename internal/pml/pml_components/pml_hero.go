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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// Hero implements the pml_domain.Component interface for the <pml-hero> tag. It
// renders a full-width banner, often with a background image, that can contain
// other content components overlaid on top.
type Hero struct {
	BaseComponent
}

var _ pml_domain.Component = (*Hero)(nil)

// NewHero creates a new Hero component instance that renders a
// full-width banner with background image support.
//
// Returns *Hero which is ready to render a hero banner section.
func NewHero() *Hero {
	return &Hero{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML custom element tag name for this component.
//
// Returns string which is the tag name used in HTML markup.
func (*Hero) TagName() string {
	return "pml-hero"
}

// AllowedParents returns the list of valid parent components for this
// component.
//
// Returns []string which contains the allowed parent component names.
func (*Hero) AllowedParents() []string {
	return []string{"pml-body", "pml-container"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute names mapped to their type definitions.
func (*Hero) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrMode:                     NewEnumAttributeDefinition([]string{ValueFluidHeight, ValueFixedHeight}),
		AttrHeight:                   NewAttributeDefinition(pml_domain.TypeUnit),
		AttrBackgroundURL:            NewAttributeDefinition(pml_domain.TypeString),
		AttrBackgroundWidth:          NewAttributeDefinition(pml_domain.TypeUnit),
		AttrBackgroundHeight:         NewAttributeDefinition(pml_domain.TypeUnit),
		CSSBackgroundPosition:        NewAttributeDefinition(pml_domain.TypeString),
		CSSBackgroundSize:            NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRadius:             NewAttributeDefinition(pml_domain.TypeString),
		AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
		AttrInnerBackgroundColor:     NewAttributeDefinition(pml_domain.TypeColor),
		AttrInnerPadding:             NewAttributeDefinition(pml_domain.TypeUnit),
		AttrInnerPadding + "-top":    NewAttributeDefinition(pml_domain.TypeUnit),
		AttrInnerPadding + "-left":   NewAttributeDefinition(pml_domain.TypeUnit),
		AttrInnerPadding + "-right":  NewAttributeDefinition(pml_domain.TypeUnit),
		AttrInnerPadding + "-bottom": NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPadding:                  NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingTop:               NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingBottom:            NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingLeft:              NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingRight:             NewAttributeDefinition(pml_domain.TypeUnit),
		CSSBackgroundColor:           NewAttributeDefinition(pml_domain.TypeColor),
		CSSVerticalAlign:             NewEnumAttributeDefinition([]string{ValueTop, ValueMiddle, ValueBottom}),
		AttrAlign:                    NewEnumAttributeDefinition([]string{ValueLeft, ValueCentre, ValueRight}),
	}
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which contains the default attribute key-value
// pairs. Returns an empty map as heroes have no default attributes.
func (*Hero) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which maps CSS properties to their target
// elements within the hero's complex structure.
func (*Hero) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: "mode", Target: "config"},
		{Property: "height", Target: "container"},
		{Property: AttrBackgroundURL, Target: TargetBackground},
		{Property: AttrBackgroundWidth, Target: TargetBackground},
		{Property: AttrBackgroundHeight, Target: TargetBackground},
		{Property: CSSBackgroundPosition, Target: TargetBackground},
		{Property: CSSBackgroundSize, Target: TargetBackground},
		{Property: CSSBackgroundColor, Target: TargetBackground},
		{Property: AttrBorderRadius, Target: TargetContainer},
		{Property: AttrContainerBackgroundColor, Target: "outer"},
		{Property: AttrInnerBackgroundColor, Target: "inner"},
		{Property: AttrInnerPadding, Target: "inner"},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: CSSVerticalAlign, Target: TargetContainer},
		{Property: AttrAlign, Target: TargetContainer},
	}
}

// Transform converts the <pml-hero> node into an email-safe HTML structure.
//
// This component is one of the most complex due to its need to overlay content
// on a responsive background image that works even in Outlook.
//
// The transformation uses several email hacks to ensure cross-client
// compatibility:
//
//  1. VML (Vector Markup Language) Background for Outlook:
//     Outlook on Windows does not support CSS background-image. To solve this,
//     a VML shape is generated inside Outlook-only conditional comments.
//     - <v:rect>: A full-width rectangle is created.
//     - <v:fill>: Uses background-url as its source, handles
//     background-position
//     and background-size (via aspect="atleast" for cover).
//     - <v:textbox>: All hero content is rendered inside this textbox, creating
//     the illusion of HTML content overlaid on a background image.
//
//  2. CSS Background for Modern Clients:
//     For Gmail, Apple Mail, and others, a div with CSS background-image,
//     background-position, background-size, and background-repeat is used.
//
//  3. Dual-Mode Height Calculation:
//     - mode=ValueFixedHeight: The height attribute is strictly enforced.
//     In Outlook, height is set on the VML <v:rect>. In modern clients, it is
//     set as height and line-height on a <td> wrapper.
//     - mode=ValueFluidHeight (Default): Uses the "padding hack". The aspect
//     ratio of the background image sets a percentage-based padding-bottom on a
//     spacer <td>, making height scale proportionally with width.
//
//  4. Multi-Layered Table Structure:
//     Nested tables orchestrate these behaviours: an outermost Outlook
//     container table, the VML structure (Outlook only), a modern div container
//     (hidden from Outlook), and an inner content table.
//
//  5. Piko Directive Preservation:
//     All p-* directives are transferred to the outermost element, ensuring
//     entire hero sections can be rendered dynamically.
//
// Takes node (*ast_domain.TemplateNode) which is the <pml-hero> node to
// transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation state and style manager.
//
// Returns *ast_domain.TemplateNode which is the transformed HTML structure.
// Returns []*pml_domain.Error which contains any validation errors encountered.
func (c *Hero) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager
	mode := getStyleWithDefault(styles, AttrMode, ValueFixedHeight)

	if !c.validateMandatoryAttributes(styles, ctx, node) {
		return NewFragmentNode(nil), ctx.Diagnostics()
	}

	containerWidth := calculateContainerWidth(ctx)
	childContainerWidth := calculateChildContainerWidth(styles, containerWidth)

	childCtx := ctx.CloneForChild(nil, nil, node, c)
	childCtx.ContainerWidth = childContainerWidth
	transformedChildren := node.Children

	vmlContent := c.renderVML(styles, transformedChildren, childCtx, mode, containerWidth)
	modernContent := c.renderModern(styles, transformedChildren, childCtx, mode, containerWidth)

	rootNode := NewFragmentNode([]*ast_domain.TemplateNode{
		vmlContent,
		NewRawHTMLNode("<!--[if !mso | IE]><!-->"),
		modernContent,
		NewRawHTMLNode("<!--<![endif]-->"),
	})

	transferPikoDirectives(node, rootNode)
	return rootNode, ctx.Diagnostics()
}

// validateMandatoryAttributes checks if all required background attributes
// are present.
//
// Takes styles (*pml_domain.StyleManager) which provides access to the
// element's style attributes.
// Takes ctx (*pml_domain.TransformationContext) which receives diagnostics
// when validation fails.
// Takes node (*ast_domain.TemplateNode) which provides location information
// for error reporting.
//
// Returns bool which is true when background-url, background-width, and
// background-height are all present.
func (c *Hero) validateMandatoryAttributes(styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext, node *ast_domain.TemplateNode) bool {
	_, bgURLExists := styles.Get(AttrBackgroundURL)
	_, bgWidthExists := styles.Get(AttrBackgroundWidth)
	_, bgHeightExists := styles.Get(AttrBackgroundHeight)

	if !bgURLExists || !bgWidthExists || !bgHeightExists {
		ctx.AddDiagnostic(
			"`background-url`, `background-width`, and `background-height` are mandatory attributes for <pml-hero>.",
			c.TagName(),
			pml_domain.SeverityError,
			node.Location,
		)
		return false
	}
	return true
}

// renderVML creates the VML (Vector Markup Language) structure for
// Outlook, generating a v:rect with v:fill for the background image
// and rendering content inside a v:textbox where height handling
// differs based on the mode attribute.
//
// Takes styles (*pml_domain.StyleManager) which provides style values.
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to wrap.
// Takes mode (string) which determines the output structure.
// Takes containerWidth (float64) which specifies the container width.
//
// Returns *ast_domain.TemplateNode which is the VML fragment for Outlook.
func (c *Hero) renderVML(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, _ *pml_domain.TransformationContext, mode string, containerWidth float64) *ast_domain.TemplateNode {
	vmlAttrs := extractVMLAttributes(styles, mode, containerWidth)
	contentTable := c.renderContentTable(styles, children, mode, containerWidth)

	vmlString := buildVMLStructure(vmlAttrs)
	vmlEnd := buildVMLEnd()

	fragmentChildren := buildVMLFragment(vmlAttrs, vmlString, vmlEnd, contentTable)
	return NewFragmentNode(fragmentChildren)
}

// vmlAttributes holds the settings needed to render VML elements.
type vmlAttributes struct {
	// bgURL is the URL of the background image for VML rendering.
	bgURL string

	// bgColor is the background colour for the VML container.
	bgColor string

	// bgPosition specifies where to place the background image.
	bgPosition string

	// height specifies the height attribute for the VML element.
	height string

	// align specifies the horizontal alignment for the VML container.
	align string

	// containerBgColor is the background colour for the outer container table.
	containerBgColor string

	// vmlOrigin is the starting point for placing the VML element.
	vmlOrigin string

	// vmlPosition is the VML position value for placing elements in the layout.
	vmlPosition string

	// vmlStyle is the CSS style string for the VML textbox element.
	vmlStyle string

	// containerWidth is the width of the container in pixels.
	containerWidth float64
}

// renderContentTable creates the table structure that wraps the hero's
// children.
//
// This table is used inside both the VML v:textbox (for Outlook) and the modern
// div. The structure differs based on the mode attribute.
//
// Takes styles (*pml_domain.StyleManager) which provides style settings.
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to wrap.
// Takes mode (string) which determines the output structure.
// Takes containerWidth (float64) which specifies the container width.
//
// Returns *ast_domain.TemplateNode which is the constructed content table.
func (c *Hero) renderContentTable(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, mode string, containerWidth float64) *ast_domain.TemplateNode {
	verticalAlign := getStyleWithDefault(styles, "vertical-align", "top")
	top, right, bottom, left := expandPadding(styles)

	tdStyles := buildContentTDStyles(styles, mode, verticalAlign, top, right, bottom, left)
	paddingBottomPercent := calculateFluidHeightPadding(styles, mode)
	wrappedChildren := c.renderInnerContent(styles, children, containerWidth)

	contentTable := buildContentTableStructure(tdStyles, wrappedChildren)
	addSpacerRowIfNeeded(contentTable, mode, paddingBottomPercent)

	return contentTable
}

// renderInnerContent wraps the children with inner padding and background
// colour.
//
// Takes styles (*pml_domain.StyleManager) which provides style values.
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to wrap.
// Takes containerWidth (float64) which specifies the container width.
//
// Returns *ast_domain.TemplateNode which contains the wrapped content with
// Outlook conditional wrappers.
func (*Hero) renderInnerContent(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, containerWidth float64) *ast_domain.TemplateNode {
	innerBgColor := mustGetStyle(styles, "inner-background-color")
	innerTop, innerRight, innerBottom, innerLeft := expandInnerPadding(styles)
	align := getStyleWithDefault(styles, AttrAlign, "center")

	childrenTable := buildInnerChildrenTable(children, innerBgColor)
	innerDiv := buildInnerContentDiv(align, innerBgColor, childrenTable)
	outlookWrapper := buildOutlookInnerWrapper(align, containerWidth, innerBgColor, innerTop, innerRight, innerBottom, innerLeft)

	return NewFragmentNode([]*ast_domain.TemplateNode{
		NewRawHTMLNode(outlookWrapper.start),
		innerDiv,
		NewRawHTMLNode(outlookWrapper.end),
	})
}

// outlookWrapperHTML holds the opening and closing HTML for Outlook wrapper.
type outlookWrapperHTML struct {
	// start is the opening HTML comment for Outlook conditional rendering.
	start string

	// end is the closing Outlook conditional comment HTML.
	end string
}

// renderModern creates the AST for modern, CSS-background-supporting clients.
// This builds a div with CSS background styles and the content table inside.
//
// Takes styles (*pml_domain.StyleManager) which provides style values.
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to wrap.
// Takes mode (string) which determines the output structure.
// Takes containerWidth (float64) which specifies the container width.
//
// Returns *ast_domain.TemplateNode which is the modern div element with
// CSS background styles.
func (c *Hero) renderModern(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, _ *pml_domain.TransformationContext, mode string, containerWidth float64) *ast_domain.TemplateNode {
	bgURL := mustGetStyle(styles, AttrBackgroundURL)
	bgColor := getStyleWithDefault(styles, CSSBackgroundColor, "#ffffff")
	bgPosition := getStyleWithDefault(styles, CSSBackgroundPosition, "center center")
	bgSize := getStyleWithDefault(styles, "background-size", "cover")
	align := getStyleWithDefault(styles, AttrAlign, "center")

	divStyles := buildModernDivStyles(styles, bgURL, bgColor, bgPosition, bgSize, containerWidth)
	contentTable := c.renderContentTable(styles, children, mode, containerWidth)
	heroDiv := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, align),
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
	}, []*ast_domain.TemplateNode{contentTable})

	return wrapWithContainerBgIfNeeded(styles, heroDiv)
}

// calculateContainerWidth returns the container width, using the mobile
// breakpoint as the default if the width is not set.
//
// Takes ctx (*pml_domain.TransformationContext) which provides the current
// transformation state including container dimensions.
//
// Returns float64 which is the container width, or WidthMobileBreakpoint if
// the width is zero.
func calculateContainerWidth(ctx *pml_domain.TransformationContext) float64 {
	if ctx.ContainerWidth == 0 {
		return WidthMobileBreakpoint
	}
	return ctx.ContainerWidth
}

// calculateChildContainerWidth works out the width available for child
// elements by removing padding from the total width.
//
// Takes styles (*pml_domain.StyleManager) which holds the padding values.
// Takes containerWidth (float64) which is the total container width in pixels.
//
// Returns float64 which is the width available for child elements. If the
// padding is larger than the container width, returns the original width.
func calculateChildContainerWidth(styles *pml_domain.StyleManager, containerWidth float64) float64 {
	_, right, _, left := expandPadding(styles)
	paddingHorizontal := float64(mustParsePixels(right) + mustParsePixels(left))

	_, innerRight, _, innerLeft := expandInnerPadding(styles)
	innerPaddingHorizontal := float64(mustParsePixels(innerRight) + mustParsePixels(innerLeft))

	childContainerWidth := containerWidth - paddingHorizontal - innerPaddingHorizontal
	if childContainerWidth < 0 {
		return containerWidth
	}
	return childContainerWidth
}

// extractVMLAttributes gathers and processes all VML-related style values.
//
// Takes styles (*pml_domain.StyleManager) which provides CSS style values.
// Takes mode (string) which specifies the VML rendering mode.
// Takes containerWidth (float64) which sets the container width in pixels.
//
// Returns vmlAttributes which contains the processed VML settings.
func extractVMLAttributes(styles *pml_domain.StyleManager, mode string, containerWidth float64) vmlAttributes {
	bgPosition := getStyleWithDefault(styles, CSSBackgroundPosition, "center center")
	height := mustGetStyle(styles, "height")
	vmlOrigin, vmlPosition := parseVMLBackgroundPosition(bgPosition)

	return vmlAttributes{
		bgURL:            mustGetStyle(styles, AttrBackgroundURL),
		bgColor:          getStyleWithDefault(styles, CSSBackgroundColor, "#ffffff"),
		bgPosition:       bgPosition,
		height:           height,
		align:            getStyleWithDefault(styles, AttrAlign, "center"),
		containerBgColor: mustGetStyle(styles, "container-background-color"),
		containerWidth:   containerWidth,
		vmlOrigin:        vmlOrigin,
		vmlPosition:      vmlPosition,
		vmlStyle:         buildHeroVMLRectStyle(mode, height, containerWidth),
	}
}

// buildHeroVMLRectStyle creates the style string for the VML v:rect element
// in hero components.
//
// Takes mode (string) which sets the height mode for the hero.
// Takes height (string) which sets the fixed height value when used.
// Takes containerWidth (float64) which sets the width in pixels.
//
// Returns string which contains the VML style attributes.
func buildHeroVMLRectStyle(mode, height string, containerWidth float64) string {
	vmlStyle := fmt.Sprintf("width:%.0fpx;", containerWidth)
	if mode == ValueFixedHeight && height != "" {
		vmlStyle += fmt.Sprintf("height:%s;", height)
	}
	return vmlStyle
}

// buildVMLStructure creates the opening VML structure markup for Microsoft
// Outlook email client support.
//
// Takes attrs (vmlAttributes) which specifies the VML styling and positioning.
//
// Returns string which contains the conditional VML markup for Outlook.
func buildVMLStructure(attrs vmlAttributes) string {
	return fmt.Sprintf(`<!--[if mso | IE]>
<table align="%s" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:%.0fpx;" width="%.0f">
  <tr>
    <td style="line-height:0;font-size:0;mso-line-height-rule:exactly;">
      <v:rect xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" fill="true" stroke="false" style="%s">
        <v:fill origin="%s" position="%s" src="%s" color="%s" type="frame" />
        <w:anchorlock/>
        <v:textbox style="mso-fit-shape-to-text:true" inset="0,0,0,0">
<![endif]-->`, attrs.align, attrs.containerWidth, attrs.containerWidth, attrs.vmlStyle,
		attrs.vmlOrigin, attrs.vmlPosition, attrs.bgURL, attrs.bgColor)
}

// buildVMLEnd creates the closing VML tags for Microsoft Outlook emails.
//
// Returns string which contains the VML closing markup.
func buildVMLEnd() string {
	return `<!--[if mso | IE]>
        </v:textbox>
      </v:rect>
    </td>
  </tr>
</table>
<![endif]-->`
}

// buildVMLFragment builds the full VML fragment with an optional outer table
// wrapper.
//
// Takes attrs (vmlAttributes) which specifies the VML styling attributes.
// Takes vmlString (string) which contains the opening VML markup.
// Takes vmlEnd (string) which contains the closing VML markup.
// Takes contentTable (*ast_domain.TemplateNode) which provides the inner
// content to wrap.
//
// Returns []*ast_domain.TemplateNode which contains the built fragment nodes
// ready for rendering.
func buildVMLFragment(attrs vmlAttributes, vmlString, vmlEnd string, contentTable *ast_domain.TemplateNode) []*ast_domain.TemplateNode {
	var fragmentChildren []*ast_domain.TemplateNode

	if attrs.containerBgColor != "" {
		outerTableStart := buildOuterTableStart(attrs)
		fragmentChildren = append(fragmentChildren, NewRawHTMLNode(outerTableStart))
	}

	fragmentChildren = append(fragmentChildren,
		NewRawHTMLNode(vmlString),
		contentTable,
		NewRawHTMLNode(vmlEnd),
	)

	if attrs.containerBgColor != "" {
		outerTableEnd := buildOuterTableEnd()
		fragmentChildren = append(fragmentChildren, NewRawHTMLNode(outerTableEnd))
	}

	return fragmentChildren
}

// buildOuterTableStart creates the opening outer table wrapper for container
// background colour in Microsoft Outlook.
//
// Takes attrs (vmlAttributes) which provides the alignment, width, and
// background colour settings for the table.
//
// Returns string which contains the VML conditional comment markup for
// Outlook support.
func buildOuterTableStart(attrs vmlAttributes) string {
	return fmt.Sprintf(`<!--[if mso | IE]>
<table align="%s" border="0" cellpadding="0" cellspacing="0" role="presentation" style="width:%.0fpx;background-color:%s;" width="%.0f">
  <tr>
    <td style="line-height:0;font-size:0;mso-line-height-rule:exactly;">
<![endif]-->`, attrs.align, attrs.containerWidth, attrs.containerBgColor, attrs.containerWidth)
}

// buildOuterTableEnd creates the closing outer table wrapper.
//
// Returns string which contains the VML closing tags for Outlook support.
func buildOuterTableEnd() string {
	return `<!--[if mso | IE]>
    </td>
  </tr>
</table>
<![endif]-->`
}

// buildContentTDStyles creates a style map for the content table's TD element.
//
// Takes styles (*pml_domain.StyleManager) which gives access to style values.
// Takes mode (string) which sets the layout mode (for example, fixed height).
// Takes verticalAlign (string) which sets vertical alignment of content.
// Takes top (string) which sets the top padding value.
// Takes right (string) which sets the right padding value.
// Takes bottom (string) which sets the bottom padding value.
// Takes left (string) which sets the left padding value.
//
// Returns map[string]string which holds the CSS properties for the TD element.
func buildContentTDStyles(styles *pml_domain.StyleManager, mode, verticalAlign, top, right, bottom, left string) map[string]string {
	tdStyles := map[string]string{
		"padding":           fmt.Sprintf("%s %s %s %s", top, right, bottom, left),
		"vertical-align":    verticalAlign,
		"background-repeat": "no-repeat",
	}

	if bgAttr := mustGetStyle(styles, AttrBackgroundURL); bgAttr != "" {
		tdStyles["background"] = bgAttr
	}

	if bgPosition := mustGetStyle(styles, CSSBackgroundPosition); bgPosition != "" {
		tdStyles[CSSBackgroundPosition] = bgPosition
	}

	if borderRadius := mustGetStyle(styles, AttrBorderRadius); borderRadius != "" {
		tdStyles[AttrBorderRadius] = borderRadius
	}

	if mode == ValueFixedHeight {
		if height, ok := styles.Get(AttrHeight); ok && height != "" {
			adjustedHeight := calculateAdjustedHeight(height, top, bottom)
			if adjustedHeight > 0 {
				tdStyles["height"] = fmt.Sprintf("%dpx", adjustedHeight)
			}
		}
	}

	return tdStyles
}

// calculateAdjustedHeight subtracts padding from height for proper sizing in
// fixed-height mode.
//
// Takes height (string) which specifies the total height in pixels.
// Takes top (string) which specifies the top padding in pixels.
// Takes bottom (string) which specifies the bottom padding in pixels.
//
// Returns int which is the adjusted height after subtracting padding.
func calculateAdjustedHeight(height, top, bottom string) int {
	heightPx := mustParsePixels(height)
	topPx := mustParsePixels(top)
	bottomPx := mustParsePixels(bottom)
	return heightPx - topPx - bottomPx
}

// calculateFluidHeightPadding works out the padding-bottom percentage for
// fluid-height mode.
//
// Takes styles (*pml_domain.StyleManager) which provides access to background
// width and height values.
// Takes mode (string) which specifies the height mode to check.
//
// Returns string which is the percentage padding (e.g. "56.25%") or empty if
// the mode is not fluid-height or background sizes are missing.
func calculateFluidHeightPadding(styles *pml_domain.StyleManager, mode string) string {
	if mode != ValueFluidHeight {
		return ""
	}

	bgWidth := mustGetStyle(styles, "background-width")
	bgHeight := mustGetStyle(styles, "background-height")

	if bgWidth == "" || bgHeight == "" {
		return ""
	}

	widthPx := float64(mustParsePixels(bgWidth))
	heightPx := float64(mustParsePixels(bgHeight))

	if widthPx > 0 {
		aspectRatio := (heightPx / widthPx) * PercentFull
		return fmt.Sprintf("%.2f%%", aspectRatio)
	}

	return ""
}

// buildContentTableStructure creates a table with tbody, tr, and td elements
// to wrap content.
//
// Takes tdStyles (map[string]string) which provides CSS styles for the td
// element.
// Takes wrappedChildren (*ast_domain.TemplateNode) which contains the content
// to place inside the table cell.
//
// Returns *ast_domain.TemplateNode which is the complete table structure with
// the content wrapped inside.
func buildContentTableStructure(tdStyles map[string]string, wrappedChildren *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	contentTd := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(tdStyles)),
	}, []*ast_domain.TemplateNode{wrappedChildren})

	contentTr := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{contentTd})
	contentTbody := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{contentTr})

	return NewElementNode(ElementTable, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrWidth, Value100),
	}, []*ast_domain.TemplateNode{contentTbody})
}

// addSpacerRowIfNeeded adds a spacer row to the table when in fluid-height
// mode.
//
// Takes contentTable (*ast_domain.TemplateNode) which is the table to modify.
// Takes mode (string) which sets the layout mode.
// Takes paddingBottomPercent (string) which sets the spacer height as a
// percentage.
func addSpacerRowIfNeeded(contentTable *ast_domain.TemplateNode, mode, paddingBottomPercent string) {
	if mode != ValueFluidHeight || paddingBottomPercent == "" {
		return
	}

	spacerTD := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, fmt.Sprintf("padding-bottom:%s;", paddingBottomPercent)),
	}, nil)

	spacerRow := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{spacerTD})

	tbody := contentTable.Children[0]
	tbody.Children = append([]*ast_domain.TemplateNode{spacerRow}, tbody.Children...)
}

// buildInnerChildrenTable creates the table structure that holds the actual
// children.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes
// to wrap in a table.
//
// Returns *ast_domain.TemplateNode which is the table element containing
// the children.
func buildInnerChildrenTable(children []*ast_domain.TemplateNode, _ string) *ast_domain.TemplateNode {
	innerTableStyle := map[string]string{
		"width":  "100%",
		"margin": "0px",
	}

	childrenTbody := NewElementNode(ElementTbody, nil, children)
	return NewElementNode(ElementTable, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrStyle, mapToStyleString(innerTableStyle)),
	}, []*ast_domain.TemplateNode{childrenTbody})
}

// buildInnerContentDiv creates the div wrapper for the inner content.
//
// Takes align (string) which sets the horizontal alignment of the content.
// Takes innerBgColor (string) which sets the background colour of the div.
// Takes childrenTable (*ast_domain.TemplateNode) which contains the nested
// content to wrap.
//
// Returns *ast_domain.TemplateNode which is the configured div element.
func buildInnerContentDiv(align, innerBgColor string, childrenTable *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	innerDivStyle := map[string]string{
		"margin": "0px auto",
	}
	if innerBgColor != "" {
		innerDivStyle[CSSBackgroundColor] = innerBgColor
	}

	return NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, align),
		NewHTMLAttribute(AttrClass, "pml-hero-content"),
		NewHTMLAttribute(AttrStyle, mapToStyleString(innerDivStyle)),
	}, []*ast_domain.TemplateNode{childrenTable})
}

// buildOutlookInnerWrapper creates the Outlook-specific wrapper HTML.
//
// Takes align (string) which specifies the table alignment.
// Takes containerWidth (float64) which sets the table width in pixels.
// Takes innerBgColor (string) which sets the background colour of the cell.
// Takes innerTop (string) which sets the top padding value.
// Takes innerRight (string) which sets the right padding value.
// Takes innerBottom (string) which sets the bottom padding value.
// Takes innerLeft (string) which sets the left padding value.
//
// Returns outlookWrapperHTML which contains the start and end conditional
// comment blocks for Outlook rendering.
func buildOutlookInnerWrapper(align string, containerWidth float64, innerBgColor, innerTop, innerRight, innerBottom, innerLeft string) outlookWrapperHTML {
	outlookInnerTdStyle := map[string]string{}
	if innerBgColor != "" {
		outlookInnerTdStyle[CSSBackgroundColor] = innerBgColor
	}
	if innerTop != "" || innerRight != "" || innerBottom != "" || innerLeft != "" {
		outlookInnerTdStyle["padding"] = fmt.Sprintf("%s %s %s %s", innerTop, innerRight, innerBottom, innerLeft)
	}

	start := fmt.Sprintf(`<!--[if mso | IE]>
<table align="%s" border="0" cellpadding="0" cellspacing="0" style="width:%.0fpx;" width="%.0f">
  <tr>
    <td style="%s">
<![endif]-->`, align, containerWidth, containerWidth, mapToStyleString(outlookInnerTdStyle))

	end := `<!--[if mso | IE]>
    </td>
  </tr>
</table>
<![endif]-->`

	return outlookWrapperHTML{start: start, end: end}
}

// expandInnerPadding works like expandPadding but for inner-padding values.
//
// Takes sm (*pml_domain.StyleManager) which provides access to style values.
//
// Returns top (string) which is the inner-padding value for the top edge.
// Returns right (string) which is the inner-padding value for the right edge.
// Returns bottom (string) which is the inner-padding value for the bottom edge.
// Returns left (string) which is the inner-padding value for the left edge.
func expandInnerPadding(sm *pml_domain.StyleManager) (top, right, bottom, left string) {
	if basePadding, ok := sm.Get("inner-padding"); ok {
		top, right, bottom, left = parsePadding(basePadding)
	}

	if value, ok := sm.Get("inner-padding-top"); ok {
		top = value
	}
	if value, ok := sm.Get("inner-padding-right"); ok {
		right = value
	}
	if value, ok := sm.Get("inner-padding-bottom"); ok {
		bottom = value
	}
	if value, ok := sm.Get("inner-padding-left"); ok {
		left = value
	}

	return top, right, bottom, left
}

// parseVMLBackgroundPosition converts a CSS background-position value to VML
// origin and position values.
//
// Takes bgPosition (string) which specifies the CSS background-position value.
//
// Returns origin (string) which is the VML origin value.
// Returns position (string) which is the VML position value.
func parseVMLBackgroundPosition(bgPosition string) (origin, position string) {
	positionMap := map[string]struct{ origin, position string }{
		"top":           {origin: VMLPositionTopCentre, position: VMLPositionTopCentre},
		"top center":    {origin: VMLPositionTopCentre, position: VMLPositionTopCentre},
		"bottom":        {origin: VMLPositionBottomCentre, position: VMLPositionBottomCentre},
		"bottom center": {origin: VMLPositionBottomCentre, position: VMLPositionBottomCentre},
		"left":          {origin: VMLPositionLeftCentre, position: VMLPositionLeftCentre},
		"left center":   {origin: VMLPositionLeftCentre, position: VMLPositionLeftCentre},
		"right":         {origin: VMLPositionRightCentre, position: VMLPositionRightCentre},
		"right center":  {origin: VMLPositionRightCentre, position: VMLPositionRightCentre},
	}

	if vmlPos, ok := positionMap[bgPosition]; ok {
		return vmlPos.origin, vmlPos.position
	}

	return VMLPositionCentreCentre, VMLPositionCentreCentre
}

// buildModernDivStyles creates the style map for the modern hero div.
//
// Takes styles (*pml_domain.StyleManager) which provides style overrides.
// Takes bgURL (string) which is the background image URL.
// Takes bgColor (string) which is the background colour.
// Takes bgPosition (string) which is the background position.
// Takes bgSize (string) which is the background size.
// Takes containerWidth (float64) which is the maximum container width.
//
// Returns map[string]string which holds the CSS property and value pairs.
func buildModernDivStyles(styles *pml_domain.StyleManager, bgURL, bgColor, bgPosition, bgSize string, containerWidth float64) map[string]string {
	divStyles := map[string]string{
		"background-image":    fmt.Sprintf("url(%s)", bgURL),
		CSSBackgroundPosition: bgPosition,
		"background-size":     bgSize,
		"background-repeat":   "no-repeat",
		CSSBackgroundColor:    bgColor,
		"margin":              "0px auto",
		"max-width":           fmt.Sprintf("%.0fpx", containerWidth),
	}

	if borderRadius, ok := styles.Get(AttrBorderRadius); ok && borderRadius != "" {
		divStyles[AttrBorderRadius] = borderRadius
		divStyles["overflow"] = "hidden"
	}

	return divStyles
}

// wrapWithContainerBgIfNeeded wraps the hero div with an outer div if a
// container background colour is set.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// values.
// Takes heroDiv (*ast_domain.TemplateNode) which is the element to wrap.
//
// Returns *ast_domain.TemplateNode which is the original heroDiv if no
// background colour is set, or a new wrapper div containing it.
func wrapWithContainerBgIfNeeded(styles *pml_domain.StyleManager, heroDiv *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	containerBgColor := mustGetStyle(styles, "container-background-color")
	if containerBgColor == "" {
		return heroDiv
	}

	outerDivStyles := map[string]string{
		CSSBackgroundColor: containerBgColor,
	}

	return NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(outerDivStyles)),
	}, []*ast_domain.TemplateNode{heroDiv})
}
