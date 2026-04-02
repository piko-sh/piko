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
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// Row represents a row container in PikoML for the <pml-row> tag.
// It implements pml_domain.Component to create table structure and handle
// full-width backgrounds with Outlook VML fallbacks.
type Row struct {
	BaseComponent
}

var _ pml_domain.Component = (*Row)(nil)

const (
	// bgSizeAuto is the default value for VML background size.
	bgSizeAuto = "auto"
)

// NewSection creates a new Row component instance. A Row is the main row
// container in PikoML for building table structures.
//
// Returns *Row which is an empty row container ready for use.
func NewSection() *Row {
	return &Row{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML custom element tag name for this component.
//
// Returns string which is the custom element tag name used in HTML.
func (*Row) TagName() string {
	return "pml-row"
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent component names.
func (*Row) AllowedParents() []string {
	return []string{"pml-body", "pml-container"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute names mapped to their definitions.
func (*Row) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		CSSBackgroundColor:      NewAttributeDefinition(pml_domain.TypeColor),
		AttrBackgroundURL:       NewAttributeDefinition(pml_domain.TypeString),
		CSSBackgroundRepeat:     NewEnumAttributeDefinition([]string{ValueRepeat, ValueNoRepeat}),
		CSSBackgroundSize:       NewAttributeDefinition(pml_domain.TypeString),
		CSSBackgroundPosition:   NewAttributeDefinition(pml_domain.TypeString),
		AttrBackgroundPositionX: NewAttributeDefinition(pml_domain.TypeString),
		AttrBackgroundPositionY: NewAttributeDefinition(pml_domain.TypeString),
		AttrBorder:              NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderBottom:        NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderLeft:          NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRadius:        NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRight:         NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderTop:           NewAttributeDefinition(pml_domain.TypeString),
		AttrCSSClass:            NewAttributeDefinition(pml_domain.TypeString),
		CSSFontFamily:           NewAttributeDefinition(pml_domain.TypeString),
		AttrDirection:           NewEnumAttributeDefinition([]string{ValueLTR, ValueRTL}),
		AttrFullWidth:           NewEnumAttributeDefinition([]string{ValueFullWidth}),
		AttrPadding:             NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingTop:          NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingBottom:       NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingLeft:         NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingRight:        NewAttributeDefinition(pml_domain.TypeUnit),
		AttrStackChildren:       NewEnumAttributeDefinition([]string{ValueTrue, ValueFalse}),
		CSSTextAlign:            NewEnumAttributeDefinition([]string{ValueLeft, ValueCentre, ValueRight}),
		AttrTextPadding:         NewAttributeDefinition(pml_domain.TypeUnit),
	}
}

// DefaultAttributes returns the default attribute values for pml-row.
// These provide sensible built-in defaults for the row component.
//
// Returns map[string]string which contains the default attribute values.
func (*Row) DefaultAttributes() map[string]string {
	return map[string]string{
		CSSBackgroundRepeat:   defaultRowBackgroundRepeat,
		CSSBackgroundSize:     defaultRowBackgroundSize,
		CSSBackgroundPosition: defaultRowBackgroundPosition,
		AttrDirection:         defaultRowDirection,
		AttrPadding:           defaultRowPadding,
		CSSTextAlign:          defaultRowTextAlign,
		AttrTextPadding:       defaultRowTextPadding,
	}
}

// GetStyleTargets returns the style targets for this row.
//
// Returns []pml_domain.StyleTarget which lists the CSS properties and their
// target elements that can be styled on a row.
func (*Row) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: CSSBackgroundColor, Target: TargetContainer},
		{Property: AttrBackgroundURL, Target: TargetContainer},
		{Property: CSSBackgroundRepeat, Target: TargetContainer},
		{Property: CSSBackgroundSize, Target: TargetContainer},
		{Property: CSSBackgroundPosition, Target: TargetContainer},
		{Property: AttrBorder, Target: TargetContainer},
		{Property: AttrBorderRadius, Target: TargetContainer},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: CSSTextAlign, Target: TargetContainer},
	}
}

// Transform converts the <pml-row> node into its final HTML table structure.
//
// Takes node (*ast_domain.TemplateNode) which is the row node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation state and settings.
//
// Returns *ast_domain.TemplateNode which is the transformed HTML table
// structure.
// Returns []*pml_domain.Error which contains any diagnostics from the
// transformation.
func (c *Row) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager
	isFullWidth, _ := styles.Get(AttrFullWidth)
	_, hasBackgroundURL := styles.Get(AttrBackgroundURL)

	containerWidth := ctx.ContainerWidth
	if containerWidth == 0 {
		containerWidth = defaultContainerWidth
	}

	_, rightPadding, _, leftPadding := expandPadding(styles)
	horizontalPadding := float64(mustParsePixels(rightPadding) + mustParsePixels(leftPadding))
	availableWidthForChildren := containerWidth - horizontalPadding
	if availableWidthForChildren < 0 {
		availableWidthForChildren = containerWidth
	}

	childCtx := ctx.CloneForChild(nil, nil, node, c)
	childCtx.ContainerWidth = availableWidthForChildren
	childCtx.SiblingCount = len(node.Children)

	if styles.IsExplicit(CSSTextAlign) {
		childCtx.InheritedTextAlign = mustGetStyle(styles, CSSTextAlign)
	}

	transformedChildren := node.Children

	var rootNode *ast_domain.TemplateNode
	if isFullWidth == ValueFullWidth {
		rootNode = c.renderFullWidth(styles, transformedChildren, childCtx, containerWidth)
	} else {
		rootNode = c.renderBoxed(styles, transformedChildren, childCtx, containerWidth)
	}

	if hasBackgroundURL {
		vmlWidth := containerWidth
		if isFullWidth == ValueFullWidth {
			vmlWidth = WidthDesktopMax
		}
		rootNode = c.renderVML(styles, rootNode, vmlWidth)
	}

	transferPikoDirectives(node, rootNode)
	return rootNode, ctx.Diagnostics()
}

// renderFullWidth creates the layout for a section where the background
// spans the full width of the viewport.
//
// Takes styles (*pml_domain.StyleManager) which provides the styling rules.
// Takes children ([]*ast_domain.TemplateNode) which contains the row content.
// Takes ctx (*pml_domain.TransformationContext) which provides layout context
// with reduced ContainerWidth for column sizing.
// Takes outerContainerWidth (float64) which is the original parent container
// width used for the wrapper div max-width and Outlook conditional table,
// before any padding reduction.
//
// Returns *ast_domain.TemplateNode which is the outer table wrapping the
// full-width row.
func (c *Row) renderFullWidth(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext, outerContainerWidth float64) *ast_domain.TemplateNode {
	containerWidth := outerContainerWidth
	direction := getStyleWithDefault(styles, AttrDirection, defaultRowDirection)
	_, hasBackgroundURL := styles.Get(AttrBackgroundURL)

	outerTableStyles, innerTableStyles, divStyles, tdStyles := buildFullWidthRowStyles(styles, containerWidth, direction, hasBackgroundURL)

	columnsWithWrappers := c.getRowColumnsWithWrappers(children, styles, ctx)

	innerTable := createRowInnerTable(innerTableStyles, tdStyles, direction, columnsWithWrappers)

	wrapperDiv := createRowWrapperDiv(innerTable, divStyles, hasBackgroundURL)

	outlookInnerStart, outlookInnerEnd := buildOutlookConditionalNodes(styles, containerWidth)

	outerTable := createFullWidthOuterTable(outerTableStyles, styles, outlookInnerStart, wrapperDiv, outlookInnerEnd)

	return outerTable
}

// renderBoxed creates the structure for a standard "boxed" section.
//
// Takes styles (*pml_domain.StyleManager) which provides style configuration.
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes
// to render.
// Takes ctx (*pml_domain.TransformationContext) which provides the rendering
// context with reduced ContainerWidth for column sizing.
// Takes outerContainerWidth (float64) which is the original parent container
// width used for the wrapper div max-width and Outlook conditional table,
// before any padding reduction.
//
// Returns *ast_domain.TemplateNode which is a fragment containing the boxed
// row with Outlook conditional comments.
func (c *Row) renderBoxed(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext, outerContainerWidth float64) *ast_domain.TemplateNode {
	containerWidth := outerContainerWidth
	direction := getStyleWithDefault(styles, AttrDirection, defaultRowDirection)
	bgURL, hasBackgroundURL := styles.Get(AttrBackgroundURL)

	tableStyles, divStyles, tdStyles := buildBoxedRowStyles(styles, containerWidth, direction, hasBackgroundURL)

	columnsWithWrappers := c.getRowColumnsWithWrappers(children, styles, ctx)

	table := createBoxedRowTable(tableStyles, tdStyles, direction, bgURL, hasBackgroundURL, columnsWithWrappers)

	wrapperDiv := createBoxedRowWrapperDiv(table, divStyles, styles, hasBackgroundURL)

	outlookTableStart, outlookTableEnd := buildOutlookConditionalNodes(styles, containerWidth)

	return NewFragmentNode([]*ast_domain.TemplateNode{outlookTableStart, wrapperDiv, outlookTableEnd})
}

// renderWrappedChildren iterates through the children, which should be
// transformed pml-col nodes, and wraps each one in the necessary Outlook
// conditional td structure. It also wraps the entire column group in an
// Outlook table and tr structure for proper layout.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// wrap.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped children with
// Outlook conditional comments.
func (*Row) renderWrappedChildren(children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	wrappedChildren := make([]*ast_domain.TemplateNode, 0, len(children)*2+2)

	outlookTableStart := NewRawHTMLNode(`<!--[if mso | IE]><table role="presentation" border="0" cellpadding="0" cellspacing="0"><tr><![endif]-->`)
	wrappedChildren = append(wrappedChildren, outlookTableStart)

	for _, child := range children {
		if shouldWrapChild(child) {
			wrappedChildren = append(wrappedChildren, wrapChildWithOutlookTD(child, ctx)...)
		} else {
			wrappedChildren = append(wrappedChildren, child)
		}
	}

	outlookTableEnd := NewRawHTMLNode(`<!--[if mso | IE]></tr></table><![endif]-->`)
	wrappedChildren = append(wrappedChildren, outlookTableEnd)

	return wrappedChildren
}

// renderStackedChildren wraps each child section in its own tr and td elements
// for vertical stacking.
//
// This is the behaviour of pml-container. Each child section gets a
// full-width tr instead of being side-by-side in one tr.
//
// Each child is wrapped in Outlook-conditional tr and td elements, there is
// no shared tr wrapper (unlike renderWrappedChildren for columns), and
// children stack vertically instead of horizontally.
//
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to
// wrap for vertical stacking.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context including container width.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped children
// ready for vertical stacking.
func (*Row) renderStackedChildren(children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	wrappedChildren := make([]*ast_domain.TemplateNode, 0, len(children)*2)

	for _, child := range children {
		if shouldWrapChild(child) {
			wrappedChildren = append(wrappedChildren, wrapChildWithOutlookTR(child, ctx)...)
		} else {
			wrappedChildren = append(wrappedChildren, child)
		}
	}

	return wrappedChildren
}

// renderVML builds VML markup for Outlook background image support.
//
// Takes styles (*pml_domain.StyleManager) which provides the background styles.
// Takes content (*ast_domain.TemplateNode) which is the content to wrap with
// VML.
// Takes containerWidth (float64) which sets the width for the VML rectangle.
//
// Returns *ast_domain.TemplateNode which wraps the content with VML start and
// end markers.
func (c *Row) renderVML(styles *pml_domain.StyleManager, content *ast_domain.TemplateNode, containerWidth float64) *ast_domain.TemplateNode {
	bgURL := mustGetStyle(styles, "background-url")
	bgColor := getStyleWithDefault(styles, "background-color", "#ffffff")
	bgRepeat := getStyleWithDefault(styles, "background-repeat", "repeat")
	bgSize := getStyleWithDefault(styles, "background-size", bgSizeAuto)
	isFullWidth := mustGetStyle(styles, "full-width") == "full-width"

	posX, posY := c.parseBackgroundPosition(styles)
	vmlOriginX, vmlPosX := c.getVMLPosition(posX, bgRepeat == "repeat")
	vmlOriginY, vmlPosY := c.getVMLPosition(posY, bgRepeat == "repeat")

	vmlAspect, vmlSize := calculateVMLAspectAndSize(bgSize)
	vmlType := calculateVMLType(bgRepeat, bgSize)

	if bgSize == bgSizeAuto {
		vmlOriginX, vmlPosX = ValueHalf, ValueHalf
		vmlOriginY, vmlPosY = "0", "0"
	}

	vmlRectStyle := buildRowVMLRectStyle(isFullWidth, containerWidth)
	vmlString := buildRowVMLString(vmlParams{
		rectStyle: vmlRectStyle,
		originX:   vmlOriginX,
		originY:   vmlOriginY,
		posX:      vmlPosX,
		posY:      vmlPosY,
		bgURL:     bgURL,
		bgColor:   bgColor,
		vmlType:   vmlType,
		size:      vmlSize,
		aspect:    vmlAspect,
	})

	vmlStart := NewRawHTMLNode(vmlString)
	vmlEnd := NewRawHTMLNode(`<!--[if mso | IE]></v:textbox></v:rect><![endif]-->`)

	return NewFragmentNode([]*ast_domain.TemplateNode{vmlStart, content, vmlEnd})
}

// vmlParams holds the values needed to build a VML string.
type vmlParams struct {
	// rectStyle is the VML style attribute for the rectangle element.
	rectStyle string

	// originX is the x-coordinate for the VML shape origin point.
	originX string

	// originY is the vertical start position for the VML rectangle.
	originY string

	// posX is the horizontal position for the VML rectangle.
	posX string

	// posY is the vertical position of the VML shape.
	posY string

	// bgURL is the URL for the background image of the VML rectangle.
	bgURL string

	// bgColor is the background colour for the VML rectangle.
	bgColor string

	// vmlType is the VML element type for Microsoft Office conditional markup.
	vmlType string

	// size specifies the VML element size attribute; empty means omit.
	size string

	// aspect specifies the VML aspect ratio attribute; empty means no aspect is set.
	aspect string
}

// parseBackgroundPosition parses the background position attributes.
// It first checks for override attributes (background-position-x and
// background-position-y), then falls back to parsing the unified
// background-position attribute.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes.
//
// Returns x (string) which is the horizontal position value.
// Returns y (string) which is the vertical position value.
func (*Row) parseBackgroundPosition(styles *pml_domain.StyleManager) (x, y string) {
	backgroundPosition := getStyleWithDefault(styles, "background-position", "top center")
	parts := strings.Fields(backgroundPosition)

	if len(parts) == 1 {
		value := parts[0]
		if value == "top" || value == "bottom" {
			x, y = "center", value
		} else {
			x, y = value, ValueCentre
		}
	} else if len(parts) == 2 {
		val1, val2 := parts[0], parts[1]
		if (val1 == ValueTop || val1 == ValueBottom) || (val1 == ValueCentre && (val2 == ValueLeft || val2 == ValueRight)) {
			x, y = val2, val1
		} else {
			x, y = val1, val2
		}
	} else {
		x, y = ValueCentre, ValueTop
	}

	if posX, ok := styles.Get(AttrBackgroundPositionX); ok && posX != "" {
		x = posX
	}
	if posY, ok := styles.Get(AttrBackgroundPositionY); ok && posY != "" {
		y = posY
	}

	return x, y
}

// getVMLPosition calculates VML position values from a CSS position string.
// This handles the VML position calculation for Outlook backgrounds.
//
// Takes pos (string) which is the CSS position value (percentage or keyword).
// Takes isRepeat (bool) which indicates if the background repeats.
//
// Returns origin (string) which is the VML origin coordinate.
// Returns position (string) which is the VML position coordinate.
func (*Row) getVMLPosition(pos string, isRepeat bool) (origin, position string) {
	isPercentage := strings.HasSuffix(pos, FormatPercent)

	if isPercentage {
		valString := strings.TrimSuffix(pos, FormatPercent)
		parsedValue, err := strconv.ParseFloat(valString, 64)
		if err != nil {
			return ValueHalf, ValueHalf
		}
		decimal := parsedValue / PercentFull

		if isRepeat {
			return fmt.Sprintf(FormatFloatCompact, decimal), fmt.Sprintf(FormatFloatCompact, decimal)
		}
		vmlPos := -PercentHalf + decimal
		return fmt.Sprintf(FormatFloatCompact, vmlPos), fmt.Sprintf(FormatFloatCompact, vmlPos)
	}

	switch pos {
	case ValueLeft, ValueTop:
		if isRepeat {
			return ValueZero, ValueZero
		}
		return ValueZero, "-0.5"
	case "center":
		if isRepeat {
			return ValueHalf, ValueHalf
		}
		return "-0.5", "-0.5"
	case "right", "bottom":
		if isRepeat {
			return "1", "1"
		}
		return "-1", "-1"
	}

	return ValueHalf, ValueHalf
}

// getRowColumnsWithWrappers returns the child columns wrapped in Outlook
// conditional structures. Uses stack-children flag to determine horizontal or
// vertical layout.
//
// Takes children ([]*ast_domain.TemplateNode) which are the column nodes to
// wrap.
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes.
// Takes ctx (*pml_domain.TransformationContext) which holds the current
// transformation state.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped column nodes.
func (c *Row) getRowColumnsWithWrappers(children []*ast_domain.TemplateNode, styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	stackChildren, _ := styles.Get(AttrStackChildren)
	if stackChildren == ValueTrue {
		return c.renderStackedChildren(children, ctx)
	}
	return c.renderWrappedChildren(children, ctx)
}

// shouldWrapChild checks if a child node should be wrapped with Outlook TD
// tags. It matches both untransformed pml-* elements and transformed column
// divs that carry data-pml-outlook-width attributes.
//
// Takes child (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the child is a wrappable element.
func shouldWrapChild(child *ast_domain.TemplateNode) bool {
	if child.NodeType != ast_domain.NodeElement {
		return false
	}
	if strings.HasPrefix(child.TagName, "pml-") {
		return true
	}
	for i := range child.Attributes {
		if child.Attributes[i].Name == "data-pml-outlook-width" {
			return true
		}
	}
	return false
}

// wrapChildWithOutlookTD wraps a child node with Outlook conditional TD tags.
// It reads width and vertical-align from data-pml-outlook-* attributes set by
// the column component, falling back to StyleManager lookup for untransformed
// nodes.
//
// Takes child (*ast_domain.TemplateNode) which is the node to wrap.
// Takes ctx (*pml_domain.TransformationContext) which provides the registry and
// settings for style resolution.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped child with
// Outlook-specific opening and closing conditional comments.
func wrapChildWithOutlookTD(child *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	var widthPx float64
	verticalAlign := ValueTop

	outlookWidth := getNodeDataAttr(child, "data-pml-outlook-width")
	outlookVAlign := getNodeDataAttr(child, "data-pml-outlook-vertical-align")

	if outlookWidth != "" {
		widthPx = float64(mustParsePixels(outlookWidth))
		if outlookVAlign != "" {
			verticalAlign = outlookVAlign
		}
		removeNodeDataAttr(child, "data-pml-outlook-width")
		removeNodeDataAttr(child, "data-pml-outlook-vertical-align")
	} else if ctx.Registry != nil {
		childComp, _ := ctx.Registry.Get(child.TagName)
		childStyleManager := pml_domain.NewStyleManager(child, childComp, ctx.Config)
		widthPx = calculateChildWidth(childStyleManager, ctx)
		verticalAlign = getStyleWithDefault(childStyleManager, CSSVerticalAlign, ValueTop)
	} else {
		widthPx = calculateEqualDistributionWidth(ctx)
	}

	startTag := buildOutlookTDTag("", "", verticalAlign, widthPx)
	startComment := NewRawHTMLNode(startTag)
	endComment := NewRawHTMLNode(`<!--[if mso | IE]></td><![endif]-->`)

	return []*ast_domain.TemplateNode{startComment, child, endComment}
}

// calculateChildWidth returns the width in pixels for a child element.
//
// Takes styleManager (*pml_domain.StyleManager) which provides access to the
// element's style attributes.
// Takes ctx (*pml_domain.TransformationContext) which holds the current
// transformation state including parent dimensions.
//
// Returns float64 which is the calculated width in pixels.
func calculateChildWidth(styleManager *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) float64 {
	widthAttr, hasWidth := styleManager.Get("width")
	if !hasWidth {
		return calculateEqualDistributionWidth(ctx)
	}

	if strings.HasSuffix(widthAttr, "%") {
		return calculatePercentageWidth(widthAttr, ctx)
	}

	return float64(mustParsePixels(widthAttr))
}

// calculateEqualDistributionWidth works out the width for a child element when
// no explicit width is set.
//
// Takes ctx (*pml_domain.TransformationContext) which provides the container
// width and sibling count for distribution.
//
// Returns float64 which is the container width divided equally among siblings,
// or the full container width if there are no siblings.
func calculateEqualDistributionWidth(ctx *pml_domain.TransformationContext) float64 {
	if ctx.SiblingCount > 0 {
		return ctx.ContainerWidth / float64(ctx.SiblingCount)
	}
	return ctx.ContainerWidth
}

// calculatePercentageWidth converts a percentage width string to an absolute
// width value.
//
// Takes widthAttr (string) which contains the width as a percentage string.
// Takes ctx (*pml_domain.TransformationContext) which provides the container
// width and sibling count for the calculation.
//
// Returns float64 which is the calculated width in absolute units. When parsing
// fails, the width is split equally among siblings.
func calculatePercentageWidth(widthAttr string, ctx *pml_domain.TransformationContext) float64 {
	var percent float64
	if _, err := fmt.Sscanf(strings.TrimSuffix(widthAttr, FormatPercent), "%f", &percent); err != nil {
		percent = PercentFull / float64(ctx.SiblingCount)
	}
	return ctx.ContainerWidth * percent / PercentFull
}

// buildOutlookTDTag builds the opening TD tag for Outlook conditional comments.
//
// Takes align (string) which sets the horizontal alignment.
// Takes cssClass (string) which sets the CSS class prefix.
// Takes verticalAlign (string) which sets the vertical alignment.
// Takes widthPx (float64) which sets the column width in pixels.
//
// Returns string which is the complete Outlook TD opening tag.
func buildOutlookTDTag(align, cssClass, verticalAlign string, widthPx float64) string {
	startTag := `<!--[if mso | IE]><td`
	if align != "" {
		startTag += fmt.Sprintf(` align=%q`, align)
	}
	if cssClass != "" {
		startTag += fmt.Sprintf(` class="%s-outlook"`, cssClass)
	}
	startTag += fmt.Sprintf(` style="vertical-align:%s;width:%.0fpx;" ><![endif]-->`, verticalAlign, widthPx)
	return startTag
}

// wrapChildWithOutlookTR wraps a child node with Outlook-specific TR/TD tags
// for vertical stacking in email layouts.
//
// Takes child (*ast_domain.TemplateNode) which is the node to wrap.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context including the registry and settings.
//
// Returns []*ast_domain.TemplateNode which contains the wrapped nodes in order:
// start comment, child, and end comment.
func wrapChildWithOutlookTR(child *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	childComp, _ := ctx.Registry.Get(child.TagName)
	childStyleManager := pml_domain.NewStyleManager(child, childComp, ctx.Config)

	align := getStyleWithDefault(childStyleManager, "align", "center")
	cssClass := getStyleWithDefault(childStyleManager, "css-class", "")

	startTag := buildOutlookTRTDTag(align, cssClass, ctx.ContainerWidth)
	startComment := NewRawHTMLNode(startTag)
	endComment := NewRawHTMLNode(`<!--[if mso | IE]></td></tr><![endif]-->`)

	return []*ast_domain.TemplateNode{startComment, child, endComment}
}

// buildOutlookTRTDTag builds the opening TR/TD tag wrapped in an Outlook
// conditional comment for email stacking layouts.
//
// Takes align (string) which sets the cell alignment attribute.
// Takes cssClass (string) which sets the CSS class prefix for Outlook.
// Takes containerWidth (float64) which sets the cell width in pixels.
//
// Returns string which is the conditional comment tag for Outlook rendering.
func buildOutlookTRTDTag(align, cssClass string, containerWidth float64) string {
	startTag := `<!--[if mso | IE]><tr><td`
	if align != "" {
		startTag += fmt.Sprintf(` align=%q`, align)
	}
	if cssClass != "" {
		startTag += fmt.Sprintf(` class="%s-outlook"`, cssClass)
	}
	startTag += fmt.Sprintf(` width="%.0f"><![endif]-->`, containerWidth)
	return startTag
}

// calculateVMLAspectAndSize works out VML aspect and size attributes from a
// CSS background-size value.
//
// Takes bgSize (string) which specifies the CSS background-size value.
//
// Returns vmlAspect (string) which is the VML aspect attribute value.
// Returns vmlSize (string) which is the VML size attribute value.
func calculateVMLAspectAndSize(bgSize string) (vmlAspect, vmlSize string) {
	if bgSize == "cover" || bgSize == "contain" {
		vmlSize = "1,1"
		if bgSize == "cover" {
			vmlAspect = "atleast"
		} else {
			vmlAspect = "atmost"
		}
	} else if bgSize != bgSizeAuto && bgSize != "" {
		parts := strings.Fields(bgSize)
		if len(parts) == 1 {
			vmlSize = parts[0]
			vmlAspect = "atmost"
		} else {
			vmlSize = strings.Join(parts, ",")
		}
	}
	return vmlAspect, vmlSize
}

// calculateVMLType determines the VML fill type based on repeat and size
// settings.
//
// Takes bgRepeat (string) which specifies the background repeat mode.
// Takes bgSize (string) which specifies the background size setting.
//
// Returns string which is the VML fill type: "tile" or "frame".
func calculateVMLType(bgRepeat, bgSize string) string {
	if bgRepeat == "repeat" || bgSize == bgSizeAuto {
		return "tile"
	}
	return "frame"
}

// buildRowVMLRectStyle builds the VML rect style string for row backgrounds.
//
// Takes isFullWidth (bool) which indicates whether the row spans the full
// width.
// Takes containerWidth (float64) which specifies the width in pixels when not
// full width.
//
// Returns string which is the VML style attribute value.
func buildRowVMLRectStyle(isFullWidth bool, containerWidth float64) string {
	if isFullWidth {
		return "mso-width-percent:1000;"
	}
	return fmt.Sprintf("width:%.0fpx;", containerWidth)
}

// buildRowVMLString builds the full VML string for row backgrounds.
//
// Takes params (vmlParams) which holds the VML settings.
//
// Returns string which is the VML markup for Microsoft Outlook.
func buildRowVMLString(params vmlParams) string {
	vmlString := fmt.Sprintf(`<!--[if mso | IE]><v:rect xmlns:v="urn:schemas-microsoft-com:vml" fill="true" stroke="false" style=%q><v:fill origin=%q position=%q src=%q color=%q type=%q`,
		params.rectStyle, fmt.Sprintf("%s, %s", params.originX, params.originY), fmt.Sprintf("%s, %s", params.posX, params.posY), params.bgURL, params.bgColor, params.vmlType)

	if params.size != "" {
		vmlString += fmt.Sprintf(` size=%q`, params.size)
	}
	if params.aspect != "" {
		vmlString += fmt.Sprintf(` aspect=%q`, params.aspect)
	}
	vmlString += ` /><v:textbox style="mso-fit-shape-to-text:true" inset="0,0,0,0"><![endif]-->`

	return vmlString
}

// getBackgroundShorthand builds the CSS background shorthand property for
// modern email clients. Both the shorthand and separate properties are emitted
// because Yahoo does not support shorthand, but other modern clients work
// better with it.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values.
//
// Returns string which is the CSS background shorthand value.
func getBackgroundShorthand(styles *pml_domain.StyleManager) string {
	bgURL, hasBgURL := styles.Get(AttrBackgroundURL)
	bgColor := getStyleWithDefault(styles, "background-color", "")

	if !hasBgURL {
		return bgColor
	}

	bgRepeat := getStyleWithDefault(styles, "background-repeat", "repeat")
	bgSize := getStyleWithDefault(styles, "background-size", ValueAuto)
	posX, posY := (&Row{BaseComponent: BaseComponent{}}).parseBackgroundPosition(styles)
	backgroundPosition := posX + ValueSpace + posY

	parts := []string{}
	if bgColor != "" {
		parts = append(parts, bgColor)
	}
	parts = append(parts,
		fmt.Sprintf("url('%s')", bgURL),
		backgroundPosition,
		fmt.Sprintf("/ %s", bgSize),
		bgRepeat,
	)

	return strings.Join(parts, ValueSpace)
}

// buildFullWidthRowStyles builds all style maps needed to render a full-width
// row.
//
// Takes styles (*pml_domain.StyleManager) which provides the source styles.
// Takes containerWidth (float64) which sets the maximum width in pixels.
// Takes direction (string) which sets the text direction for the row.
// Takes hasBackgroundURL (bool) which controls whether background image styles
// are included.
//
// Returns outerTableStyles (map[string]string) which holds styles for the
// outer table wrapper, including background settings.
// Returns innerTableStyles (map[string]string) which holds styles for the
// inner table, including border collapse settings.
// Returns divStyles (map[string]string) which holds styles for the div
// wrapper, including max-width and overflow settings.
// Returns tdStyles (map[string]string) which holds styles for the table cell,
// including padding and alignment.
func buildFullWidthRowStyles(
	styles *pml_domain.StyleManager,
	containerWidth float64,
	direction string,
	hasBackgroundURL bool,
) (outerTableStyles, innerTableStyles, divStyles, tdStyles map[string]string) {
	outerTableStyles = map[string]string{CSSWidth: Value100}
	if hasBackgroundURL {
		outerTableStyles[CSSBackground] = getBackgroundShorthand(styles)
	}
	copyStyle(styles, outerTableStyles, CSSBackgroundColor)
	copyStyle(styles, outerTableStyles, CSSBackgroundRepeat)
	copyStyle(styles, outerTableStyles, CSSBackgroundSize)
	copyStyle(styles, outerTableStyles, CSSBackgroundPosition)

	innerTableStyles = map[string]string{CSSWidth: Value100}
	if borderRadius, ok := styles.Get(AttrBorderRadius); ok && borderRadius != "" {
		innerTableStyles[CSSBorderCollapse] = ValueSeparate
	}

	divStyles = map[string]string{
		CSSMargin:   ValueMarginAuto,
		CSSMaxWidth: fmt.Sprintf("%.0fpx", containerWidth),
	}
	if borderRadius, ok := styles.Get(AttrBorderRadius); ok && borderRadius != "" {
		divStyles[CSSOverflow] = ValueHidden
		divStyles[AttrBorderRadius] = borderRadius
	}

	tdStyles = buildRowTdStyles(styles, direction)

	return outerTableStyles, innerTableStyles, divStyles, tdStyles
}

// buildBoxedRowStyles builds all style maps for boxed row rendering.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values.
// Takes containerWidth (float64) which sets the maximum width in pixels.
// Takes direction (string) which sets the text direction for the row.
// Takes hasBackgroundURL (bool) which indicates if a background image is set.
//
// Returns tableStyles (map[string]string) which holds the table element styles
// including background settings.
// Returns divStyles (map[string]string) which holds the wrapper div styles
// including max-width and border radius overflow handling.
// Returns tdStyles (map[string]string) which holds the table cell styles.
func buildBoxedRowStyles(
	styles *pml_domain.StyleManager,
	containerWidth float64,
	direction string,
	hasBackgroundURL bool,
) (tableStyles, divStyles, tdStyles map[string]string) {
	tableStyles = map[string]string{CSSWidth: Value100}
	if hasBackgroundURL {
		tableStyles[CSSBackground] = getBackgroundShorthand(styles)
	}
	copyStyle(styles, tableStyles, CSSBackgroundColor)
	copyStyle(styles, tableStyles, CSSBackgroundRepeat)
	copyStyle(styles, tableStyles, CSSBackgroundSize)
	copyStyle(styles, tableStyles, CSSBackgroundPosition)

	if borderRadius, ok := styles.Get(AttrBorderRadius); ok && borderRadius != "" {
		tableStyles[CSSBorderCollapse] = ValueSeparate
	}

	divStyles = map[string]string{
		CSSMargin:   ValueMarginAuto,
		CSSMaxWidth: fmt.Sprintf("%.0fpx", containerWidth),
	}
	if borderRadius, ok := styles.Get(AttrBorderRadius); ok && borderRadius != "" {
		divStyles[CSSOverflow] = ValueHidden
		divStyles[AttrBorderRadius] = borderRadius
	}

	tdStyles = buildRowTdStyles(styles, direction)

	return tableStyles, divStyles, tdStyles
}

// buildRowTdStyles constructs the TD style map shared by both full-width and
// boxed modes.
//
// Takes styles (*pml_domain.StyleManager) which provides the source styles to
// copy from.
// Takes direction (string) which specifies the text direction if not empty.
//
// Returns map[string]string which contains the CSS properties for the TD
// element.
func buildRowTdStyles(styles *pml_domain.StyleManager, direction string) map[string]string {
	tdStyles := map[string]string{
		CSSLineHeight:        ValueZeroPx,
		CSSFontSize:          ValueZeroPx,
		CSSMsoLineHeightRule: ValueExactly,
	}

	if direction != "" {
		tdStyles[CSSDirection] = direction
	}

	copyStyle(styles, tdStyles, AttrBorder)
	copyStyle(styles, tdStyles, CSSBorderTop)
	copyStyle(styles, tdStyles, CSSBorderRight)
	copyStyle(styles, tdStyles, CSSBorderBottom)
	copyStyle(styles, tdStyles, CSSBorderLeft)
	copyStyle(styles, tdStyles, AttrBorderRadius)
	copyStyle(styles, tdStyles, CSSTextAlign)
	copyStyle(styles, tdStyles, CSSFontFamily)

	top, right, bottom, left := expandPadding(styles)
	tdStyles[CSSPadding] = fmt.Sprintf("%s %s %s %s", coalesce(top, ValueZeroPx), coalesce(right, ValueZeroPx), coalesce(bottom, ValueZeroPx), coalesce(left, ValueZeroPx))

	return tdStyles
}

// createRowInnerTable builds the inner table structure for a row layout,
// producing a nested table > tbody > tr > td hierarchy.
//
// Takes tableStyles (map[string]string) which sets CSS styles for the table.
// Takes tdStyles (map[string]string) which sets CSS styles for the td cell.
// Takes direction (string) which sets the text direction if not empty.
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes to nest.
//
// Returns *ast_domain.TemplateNode which is the complete table structure.
func createRowInnerTable(tableStyles, tdStyles map[string]string, direction string, children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	tableAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, ValueCentre),
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
	}

	if direction != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrDir, direction))
	}
	if len(tableStyles) > 0 {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrStyle, mapToStyleString(tableStyles)))
	}

	td := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(tdStyles)),
	}, children)
	tr := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{td})
	tbody := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{tr})

	return NewElementNode(ElementTable, sortHTMLAttributes(tableAttrs), []*ast_domain.TemplateNode{tbody})
}

// createRowWrapperDiv creates a wrapper div element around the given content.
//
// When hasBackgroundURL is true, the content is first wrapped in an inner div
// with line-height and font-size set to zero. This inner div supports
// background images.
//
// Takes content (*ast_domain.TemplateNode) which is the content to wrap.
// Takes divStyles (map[string]string) which specifies the styles for the
// outer wrapper div.
// Takes hasBackgroundURL (bool) which indicates whether to add an inner div
// for background image support.
//
// Returns *ast_domain.TemplateNode which is the wrapper div element.
func createRowWrapperDiv(content *ast_domain.TemplateNode, divStyles map[string]string, hasBackgroundURL bool) *ast_domain.TemplateNode {
	var children []*ast_domain.TemplateNode

	if hasBackgroundURL {
		innerDivStyles := map[string]string{
			CSSLineHeight: ValueZero,
			CSSFontSize:   ValueZero,
		}
		innerDiv := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
			NewHTMLAttribute(AttrStyle, mapToStyleString(innerDivStyles)),
		}, []*ast_domain.TemplateNode{content})
		children = []*ast_domain.TemplateNode{innerDiv}
	} else {
		children = []*ast_domain.TemplateNode{content}
	}

	return NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
	}, children)
}

// buildOutlookConditionalNodes creates the Outlook conditional comment nodes.
//
// Takes styles (*pml_domain.StyleManager) which provides CSS class and
// background colour values for the Outlook table.
// Takes containerWidth (float64) which sets the fixed width for the table.
//
// Returns startNode (*ast_domain.TemplateNode) which opens the Outlook
// conditional table structure.
// Returns endNode (*ast_domain.TemplateNode) which closes the conditional
// table structure.
func buildOutlookConditionalNodes(
	styles *pml_domain.StyleManager,
	containerWidth float64,
) (startNode, endNode *ast_domain.TemplateNode) {
	outlookHTML := `<!--[if mso | IE]><table align="center" border="0" cellpadding="0" cellspacing="0"`

	if cssClass, ok := styles.Get(AttrCSSClass); ok && cssClass != "" {
		outlookHTML += fmt.Sprintf(` class="%s-outlook"`, cssClass)
	}
	if bgColor, ok := styles.Get(CSSBackgroundColor); ok && bgColor != "" {
		outlookHTML += fmt.Sprintf(` bgcolor=%q`, bgColor)
	}

	outlookHTML += fmt.Sprintf(
		` style="width:%.0fpx;" width="%.0f" role="presentation">`+
			`<tr><td style="line-height:0px;font-size:0px;mso-line-height-rule:exactly;"><![endif]-->`,
		containerWidth, containerWidth)

	startNode = NewRawHTMLNode(outlookHTML)
	endNode = NewRawHTMLNode(`<!--[if mso | IE]></td></tr></table><![endif]-->`)
	return startNode, endNode
}

// createFullWidthOuterTable creates the outer table structure for full-width
// mode.
//
// Takes outerTableStyles (map[string]string) which provides inline CSS styles
// for the outer table element.
// Takes styles (*pml_domain.StyleManager) which manages CSS class lookups.
// Takes outlookStart (*ast_domain.TemplateNode) which contains the Outlook
// conditional comment opening.
// Takes wrapperDiv (*ast_domain.TemplateNode) which holds the inner content
// wrapper.
// Takes outlookEnd (*ast_domain.TemplateNode) which contains the Outlook
// conditional comment closing.
//
// Returns *ast_domain.TemplateNode which is the complete outer table element
// with nested tbody, tr, and td structure.
func createFullWidthOuterTable(outerTableStyles map[string]string, styles *pml_domain.StyleManager, outlookStart, wrapperDiv, outlookEnd *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	outerTableAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, ValueCentre),
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(CSSWidth, Value100),
	}

	if cssClass, ok := styles.Get(AttrCSSClass); ok && cssClass != "" {
		outerTableAttrs = append(outerTableAttrs, NewHTMLAttribute(AttrClass, cssClass))
	}
	if len(outerTableStyles) > 0 {
		outerTableAttrs = append(outerTableAttrs, NewHTMLAttribute(AttrStyle, mapToStyleString(outerTableStyles)))
	}

	outerTd := NewElementNode(ElementTd, nil, []*ast_domain.TemplateNode{outlookStart, wrapperDiv, outlookEnd})
	outerTr := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{outerTd})
	outerTbody := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{outerTr})

	return NewElementNode(ElementTable, sortHTMLAttributes(outerTableAttrs), []*ast_domain.TemplateNode{outerTbody})
}

// createBoxedRowTable creates the main table for boxed mode with an optional
// background attribute.
//
// Takes tableStyles (map[string]string) which specifies CSS styles for the
// table element.
// Takes tdStyles (map[string]string) which specifies CSS styles for the cell.
// Takes direction (string) which sets the text direction attribute if not
// empty.
// Takes bgURL (string) which provides the background image URL.
// Takes hasBackgroundURL (bool) which indicates whether to add the background
// attribute.
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes
// to place inside the table cell.
//
// Returns *ast_domain.TemplateNode which is the constructed table element.
func createBoxedRowTable(tableStyles, tdStyles map[string]string, direction, bgURL string, hasBackgroundURL bool, children []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	tableAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, ValueCentre),
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
	}

	if hasBackgroundURL {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrBackground, bgURL))
	}
	if direction != "" {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrDir, direction))
	}
	if len(tableStyles) > 0 {
		tableAttrs = append(tableAttrs, NewHTMLAttribute(AttrStyle, mapToStyleString(tableStyles)))
	}

	td := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(tdStyles)),
	}, children)
	tr := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{td})
	tbody := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{tr})

	return NewElementNode(ElementTable, sortHTMLAttributes(tableAttrs), []*ast_domain.TemplateNode{tbody})
}

// createBoxedRowWrapperDiv creates the wrapper div for boxed mode with an
// optional CSS class.
//
// When a background image is used, it wraps the table in an inner div with
// styles that reset line height and font size.
//
// Takes table (*ast_domain.TemplateNode) which is the table element to wrap.
// Takes divStyles (map[string]string) which sets the wrapper div styles.
// Takes styles (*pml_domain.StyleManager) which gives access to CSS classes.
// Takes hasBackgroundURL (bool) which shows if a background image is used.
//
// Returns *ast_domain.TemplateNode which is the configured wrapper div element.
func createBoxedRowWrapperDiv(table *ast_domain.TemplateNode, divStyles map[string]string, styles *pml_domain.StyleManager, hasBackgroundURL bool) *ast_domain.TemplateNode {
	var children []*ast_domain.TemplateNode

	if hasBackgroundURL {
		innerDivStyles := map[string]string{
			CSSLineHeight: ValueZero,
			CSSFontSize:   ValueZero,
		}
		innerDiv := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
			NewHTMLAttribute(AttrStyle, mapToStyleString(innerDivStyles)),
		}, []*ast_domain.TemplateNode{table})
		children = []*ast_domain.TemplateNode{innerDiv}
	} else {
		children = []*ast_domain.TemplateNode{table}
	}

	wrapperAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
	}

	if cssClass, ok := styles.Get(AttrCSSClass); ok && cssClass != "" {
		wrapperAttrs = append([]ast_domain.HTMLAttribute{NewHTMLAttribute(AttrClass, cssClass)}, wrapperAttrs...)
	}

	return NewElementNode(ElementDiv, sortHTMLAttributes(wrapperAttrs), children)
}
