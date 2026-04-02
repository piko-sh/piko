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

// Column implements the pml_domain.Component interface for the <pml-col> tag.
// It is the main content container within a section, handling responsive
// stacking and padding for child content.
type Column struct {
	BaseComponent
}

var _ pml_domain.Component = (*Column)(nil)

const (
	// defaultColumnAlign is the default text alignment for column content.
	defaultColumnAlign = "left"

	// defaultColumnDirection is the default text direction for columns.
	defaultColumnDirection = "ltr"

	// defaultColumnVerticalAlign is the default vertical alignment for columns.
	defaultColumnVerticalAlign = "top"

	// defaultColumnPadding is the default padding value for columns.
	defaultColumnPadding = "0"
)

// NewColumn creates a new Column component instance. A Column is the main
// content container within a row and handles responsive stacking.
//
// Returns *Column which is ready to be configured and added to a row.
func NewColumn() *Column {
	return &Column{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML tag name for this column element.
//
// Returns string which is the tag name used when rendering this column.
func (*Column) TagName() string {
	return "pml-col"
}

// AllowedParents returns the list of valid parent components for this
// component.
//
// Returns []string which contains the allowed parent component names.
func (*Column) AllowedParents() []string {
	return []string{"pml-row", "pml-no-stack"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute names and their type definitions.
func (*Column) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		CSSBackgroundColor:       NewAttributeDefinition(pml_domain.TypeColor),
		AttrBorder:               NewAttributeDefinition(pml_domain.TypeString),
		AttrBorderRadius:         NewAttributeDefinition(pml_domain.TypeUnit),
		AttrInnerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
		AttrPadding:              NewAttributeDefinition(pml_domain.TypeUnit),
		CSSVerticalAlign:         NewEnumAttributeDefinition([]string{ValueTop, ValueMiddle, ValueBottom}),
		AttrWidth:                NewAttributeDefinition(pml_domain.TypeUnit),
		AttrDirection:            NewEnumAttributeDefinition([]string{ValueLTR, ValueRTL}),
		AttrInnerBorder:          NewAttributeDefinition(pml_domain.TypeString),
		AttrInnerBorderRadius:    NewAttributeDefinition(pml_domain.TypeUnit),
		AttrCSSClass:             NewAttributeDefinition(pml_domain.TypeString),
	}
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as columns have no default
// attributes.
func (*Column) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which contains the supported style
// properties and their target elements.
func (*Column) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: CSSBackgroundColor, Target: TargetContainer},
		{Property: AttrBorder, Target: TargetContainer},
		{Property: AttrBorderRadius, Target: TargetContainer},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: CSSVerticalAlign, Target: TargetContainer},
		{Property: AttrWidth, Target: TargetContainer},
	}
}

// Transform renders a column as a modern div wrapper with responsive styles.
// The parent pml-row handles the Outlook conditional td structure that this
// div will be placed inside.
//
// Takes node (*ast_domain.TemplateNode) which is the column node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides style and media
// query context.
//
// Returns *ast_domain.TemplateNode which is the rendered div element with
// responsive width styles.
// Returns []*pml_domain.Error which contains any diagnostics from the context.
func (c *Column) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	widthPercent, _, cssClassName := c.calculateWidth(styles, ctx)

	divStyles := c.getDivStyles(styles, ctx)

	classValue := cssClassName
	if !ctx.IsInsideGroup {
		classValue += " " + ClassColGroup
	}
	if value, ok := styles.Get(AttrCSSClass); ok {
		classValue += " " + value
	}

	var columnContent *ast_domain.TemplateNode
	if c.hasGutter(styles) {
		columnContent = c.renderGutter(styles, node.Children, ctx)
	} else {
		columnContent = c.renderColumn(styles, node.Children, ctx)
	}

	if !ctx.IsInsideGroup && ctx.MediaQueryCollector != nil {
		ctx.MediaQueryCollector.RegisterClass(cssClassName, "width: 100% !important;")
	}

	divStyles[CSSWidth] = fmt.Sprintf("%.2f%%", widthPercent)

	_, widthPixels, _ := c.calculateWidth(styles, ctx)
	verticalAlign := getStyleWithDefault(styles, CSSVerticalAlign, defaultColumnVerticalAlign)

	modernDiv := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrClass, classValue),
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
		NewHTMLAttribute("data-pml-outlook-width", fmt.Sprintf("%.0f", widthPixels)),
		NewHTMLAttribute("data-pml-outlook-vertical-align", verticalAlign),
	}, []*ast_domain.TemplateNode{columnContent})

	transferPikoDirectives(node, modernDiv)
	return modernDiv, ctx.Diagnostics()
}

// renderGutter creates the outer table structure for column-level padding
// ("gutter").
//
// Takes styles (*pml_domain.StyleManager) which provides the style settings.
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes.
// Takes ctx (*pml_domain.TransformationContext) which provides transformation
// context.
//
// Returns *ast_domain.TemplateNode which is the table element with gutter
// styling applied.
func (c *Column) renderGutter(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) *ast_domain.TemplateNode {
	gutterTDStyles := map[string]string{}
	copyStyle(styles, gutterTDStyles, CSSBackgroundColor)
	copyStyle(styles, gutterTDStyles, AttrBorder)
	copyStyle(styles, gutterTDStyles, AttrBorderRadius)
	copyStyle(styles, gutterTDStyles, CSSVerticalAlign)
	top, right, bottom, left := expandPadding(styles)
	gutterTDStyles[AttrPadding] = fmt.Sprintf("%s %s %s %s", coalesce(top, ValueZero), coalesce(right, ValueZero), coalesce(bottom, ValueZero), coalesce(left, ValueZero))

	tdNode := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(gutterTDStyles)),
	}, []*ast_domain.TemplateNode{c.renderColumn(styles, children, ctx)})

	trNode := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{tdNode})
	tbodyNode := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{trNode})

	return NewElementNode(ElementTable, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrWidth, Value100),
	}, []*ast_domain.TemplateNode{tbodyNode})
}

// renderColumn creates the inner table that holds the column's children,
// stacking them vertically.
//
// Takes styles (*pml_domain.StyleManager) which provides style configuration.
// Takes children ([]*ast_domain.TemplateNode) which contains the child nodes
// to render.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation context.
//
// Returns *ast_domain.TemplateNode which is the table element containing the
// stacked children.
func (c *Column) renderColumn(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) *ast_domain.TemplateNode {
	tableStyles := map[string]string{
		CSSVerticalAlign: ValueTop,
	}
	if c.hasGutter(styles) {
		copyStyle(styles, tableStyles, AttrInnerBackgroundColor, CSSBackgroundColor)
		copyStyle(styles, tableStyles, AttrInnerBorder, AttrBorder)
		copyStyle(styles, tableStyles, AttrInnerBorderRadius, AttrBorderRadius)
	} else {
		copyStyle(styles, tableStyles, CSSBackgroundColor)
		copyStyle(styles, tableStyles, AttrBorder)
		copyStyle(styles, tableStyles, AttrBorderRadius)
	}

	childRows := make([]*ast_domain.TemplateNode, 0, len(children))
	for _, child := range children {
		if child.NodeType == ast_domain.NodeFragment {
			childRows = append(childRows, c.processFragmentChildren(child, ctx)...)
			continue
		}

		if child.NodeType != ast_domain.NodeElement {
			continue
		}

		childRows = append(childRows, c.createChildRow(child, ctx))
	}

	if len(childRows) == 0 {
		spacerTd := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
			NewHTMLAttribute(AttrStyle, fmt.Sprintf("%s:%s;%s:%s;", CSSFontSize, ValueZeroPx, CSSLineHeight, ValueZeroPx)),
		}, []*ast_domain.TemplateNode{NewRawHTMLNode("&nbsp;")})
		childRows = append(childRows, NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{spacerTd}))
	}

	tbodyNode := NewElementNode(ElementTbody, nil, childRows)

	return NewElementNode(ElementTable, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrStyle, mapToStyleString(tableStyles)),
		NewHTMLAttribute(AttrWidth, Value100),
	}, []*ast_domain.TemplateNode{tbodyNode})
}

// processFragmentChildren processes all children of a fragment node and
// returns their rows.
//
// Takes fragment (*ast_domain.TemplateNode) which is the fragment node whose
// children will be processed.
// Takes ctx (*pml_domain.TransformationContext) which provides the current
// transformation state.
//
// Returns []*ast_domain.TemplateNode which contains the processed child rows.
func (c *Column) processFragmentChildren(fragment *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []*ast_domain.TemplateNode {
	rows := make([]*ast_domain.TemplateNode, 0, len(fragment.Children))

	rawAlign := ValueLeft
	for _, fc := range fragment.Children {
		if fc.NodeType == ast_domain.NodeElement {
			if align := c.getDataAttr(fc, "data-pml-align"); align != "" {
				rawAlign = align
				break
			}
		}
	}

	for _, fragmentChild := range fragment.Children {
		if fragmentChild.NodeType == ast_domain.NodeElement {
			rows = append(rows, c.createChildRow(fragmentChild, ctx))
			continue
		}

		if fragmentChild.NodeType == ast_domain.NodeRawHTML || fragmentChild.NodeType == ast_domain.NodeComment {
			rawTDNode := NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
				NewHTMLAttribute(AttrAlign, rawAlign),
				NewHTMLAttribute(AttrStyle, fmt.Sprintf("%s:%s;%s:%s;%s:%s;", CSSFontSize, ValueZeroPx, AttrPadding, "0 0 0 0", CSSWordBreak, ValueBreakWord)),
			}, []*ast_domain.TemplateNode{fragmentChild})

			rows = append(rows, NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{rawTDNode}))
		}
	}

	return rows
}

// createChildRow creates a <tr><td> wrapper for a single child element
// with proper padding and alignment, reading PML attributes from
// data-pml-* attributes that were preserved during the child's
// transformation.
//
// Takes child (*ast_domain.TemplateNode) which is the element to wrap.
//
// Returns *ast_domain.TemplateNode which is the tr/td wrapper containing
// the child.
func (c *Column) createChildRow(child *ast_domain.TemplateNode, _ *pml_domain.TransformationContext) *ast_domain.TemplateNode {
	childPadding := c.getDataAttr(child, "data-pml-padding")
	childAlign := c.getDataAttr(child, "data-pml-align")
	childContainerBg := c.getDataAttr(child, "data-pml-container-background-color")

	childPaddingTop := c.getDataAttr(child, "data-pml-padding-top")
	childPaddingRight := c.getDataAttr(child, "data-pml-padding-right")
	childPaddingBottom := c.getDataAttr(child, "data-pml-padding-bottom")
	childPaddingLeft := c.getDataAttr(child, "data-pml-padding-left")

	if childPadding == "" {
		childPadding = defaultColumnPadding
	}

	cellStyles := map[string]string{
		CSSFontSize:  ValueZeroPx,
		CSSWordBreak: ValueBreakWord,
		AttrPadding:  childPadding,
	}

	if childPaddingTop != "" {
		cellStyles["padding-top"] = childPaddingTop
	}
	if childPaddingRight != "" {
		cellStyles["padding-right"] = childPaddingRight
	}
	if childPaddingBottom != "" {
		cellStyles["padding-bottom"] = childPaddingBottom
	}
	if childPaddingLeft != "" {
		cellStyles["padding-left"] = childPaddingLeft
	}

	if childContainerBg != "" {
		cellStyles[CSSBackground] = childContainerBg
	}

	c.removeDataPMLAttributes(child)

	tdAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(cellStyles)),
	}
	if childAlign != "" {
		tdAttrs = append(tdAttrs, NewHTMLAttribute(AttrAlign, childAlign))
	}

	tdNode := NewElementNode(ElementTd, tdAttrs, []*ast_domain.TemplateNode{child})

	return NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{tdNode})
}

// getDataAttr retrieves a data-* attribute value from a node.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes to
// search.
// Takes attributeName (string) which specifies the attribute name to find.
//
// Returns string which is the attribute value, or empty if not found.
func (*Column) getDataAttr(node *ast_domain.TemplateNode, attributeName string) string {
	for i := range node.Attributes {
		if node.Attributes[i].Name == attributeName {
			return node.Attributes[i].Value
		}
	}
	return ""
}

// removeDataPMLAttributes removes all data-pml-* attributes from a node after
// they have been read. This means the final HTML does not contain internal
// metadata.
//
// Takes node (*ast_domain.TemplateNode) which is the node to clean.
func (*Column) removeDataPMLAttributes(node *ast_domain.TemplateNode) {
	filtered := make([]ast_domain.HTMLAttribute, 0, len(node.Attributes))
	for i := range node.Attributes {
		if !strings.HasPrefix(node.Attributes[i].Name, "data-pml-") {
			filtered = append(filtered, node.Attributes[i])
		}
	}
	node.Attributes = filtered
}

// calculateWidth computes the width values for a column element.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes.
// Takes ctx (*pml_domain.TransformationContext) which contains the parent
// container width and sibling count.
//
// Returns widthPercent (float64) which is the column width as a percentage.
// Returns widthPixels (float64) which is the column width in pixels.
// Returns cssClassName (string) which is the CSS class name for the width.
func (*Column) calculateWidth(styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) (widthPercent float64, widthPixels float64, cssClassName string) {
	parentWidth := ctx.ContainerWidth
	if parentWidth == 0 {
		parentWidth = WidthMobileBreakpoint
	}

	widthAttr, hasWidth := styles.Get(AttrWidth)

	widthPercent = calculateWidthPercentage(widthAttr, hasWidth, parentWidth, ctx.SiblingCount)

	widthPixels = parentWidth * widthPercent / PercentFull
	formattedPercent := fmt.Sprintf("%.2f", widthPercent)
	formattedPercent = strings.TrimRight(strings.TrimRight(formattedPercent, "0"), ".")
	classNameNb := strings.ReplaceAll(formattedPercent, ".", "-")
	cssClassName = ClassColumnPrefix + classNameNb

	return widthPercent, widthPixels, cssClassName
}

// getDivStyles builds the CSS style map for the outer column div element.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values.
//
// Returns map[string]string which contains the CSS property-value pairs
// for the div.
func (c *Column) getDivStyles(styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) map[string]string {
	textAlign := defaultColumnAlign
	if ctx.InheritedTextAlign != "" {
		textAlign = ctx.InheritedTextAlign
	}

	divStyles := map[string]string{
		CSSFontSize:      ValueZeroPx,
		CSSTextAlign:     textAlign,
		AttrDirection:    getStyleWithDefault(styles, AttrDirection, defaultColumnDirection),
		CSSDisplay:       ValueInlineBlock,
		CSSVerticalAlign: getStyleWithDefault(styles, CSSVerticalAlign, defaultColumnVerticalAlign),
	}
	if !c.hasGutter(styles) {
		copyStyle(styles, divStyles, CSSBackgroundColor)
	}
	return divStyles
}

// hasGutter reports whether any padding style is set.
//
// Takes styles (*pml_domain.StyleManager) which provides access to styles.
//
// Returns bool which is true if any padding attribute exists.
func (*Column) hasGutter(styles *pml_domain.StyleManager) bool {
	attrs := []string{AttrPadding, "padding-top", "padding-right", "padding-bottom", "padding-left"}
	for _, attr := range attrs {
		if _, ok := styles.Get(attr); ok {
			return true
		}
	}
	return false
}

// calculateWidthPercentage determines the width percentage based on attribute
// value and context.
//
// Takes widthAttr (string) which specifies the width as percentage or pixels.
// Takes hasWidth (bool) which indicates if a width attribute was provided.
// Takes parentWidth (float64) which is the parent container width in pixels.
// Takes siblingCount (int) which is the number of sibling elements.
//
// Returns float64 which is the calculated width as a percentage.
func calculateWidthPercentage(widthAttr string, hasWidth bool, parentWidth float64, siblingCount int) float64 {
	defaultWidth := calculateDefaultWidth(siblingCount)

	if !hasWidth || widthAttr == "" {
		return defaultWidth
	}

	if strings.HasSuffix(widthAttr, FormatPercent) {
		var widthPercent float64
		if _, err := fmt.Sscanf(strings.TrimSuffix(widthAttr, FormatPercent), "%f", &widthPercent); err != nil {
			return defaultWidth
		}
		return widthPercent
	}

	widthPixels := float64(mustParsePixels(widthAttr))
	if parentWidth > 0 {
		return (widthPixels / parentWidth) * PercentFull
	}
	return 0
}

// calculateDefaultWidth returns the default width percentage for each sibling.
//
// Takes siblingCount (int) which specifies the number of sibling elements.
//
// Returns float64 which is the width percentage, dividing the full width
// equally among siblings.
func calculateDefaultWidth(siblingCount int) float64 {
	if siblingCount > 0 {
		return PercentFull / float64(siblingCount)
	}
	return PercentFull
}

// coalesce returns the first non-empty string from the given values.
//
// Takes values (...string) which are the candidate strings to check.
//
// Returns string which is the first non-empty value, or an empty string if
// none is found.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
