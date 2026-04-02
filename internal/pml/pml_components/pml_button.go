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

// Button represents a clickable button element for email templates.
// It implements the pml_domain.Component interface for the <pml-button> tag.
type Button struct {
	BaseComponent
}

var _ pml_domain.Component = (*Button)(nil)

const (
	// defaultButtonAlign is the default horizontal alignment for buttons.
	defaultButtonAlign = "center"

	// vmlInsetDefault is the fallback inset value for VML textbox padding.
	vmlInsetDefault = "0"

	// defaultButtonBgColor is the default background colour for buttons.
	defaultButtonBgColor = "#414141"

	// defaultButtonBorder is the border style used when none is set.
	defaultButtonBorder = "none"

	// defaultButtonBorderRadius is the default border radius for button corners.
	defaultButtonBorderRadius = "3px"

	// defaultButtonColor is the default text colour for button elements.
	defaultButtonColor = "#ffffff"

	// defaultButtonFontFamily is the default list of fonts for button text.
	defaultButtonFontFamily = "Ubuntu, Helvetica, Arial, sans-serif"

	// defaultButtonFontSize is the default font size for button text.
	defaultButtonFontSize = "13px"

	// defaultButtonFontWeight is the default font weight for button text.
	defaultButtonFontWeight = "normal"

	// defaultButtonInnerPadding is the default space inside buttons around the text.
	defaultButtonInnerPadding = "10px 25px"

	// defaultButtonLineHeight is the default CSS line height for button text.
	defaultButtonLineHeight = "120%"

	// defaultButtonPadding is the default padding for button elements.
	defaultButtonPadding = "10px 25px"

	// defaultButtonTarget is the link target used when none is set.
	defaultButtonTarget = "_blank"

	// defaultButtonTextDecoration is the default CSS text decoration for buttons.
	defaultButtonTextDecoration = "none"

	// defaultButtonTextTransform is the default CSS text-transform value for buttons.
	defaultButtonTextTransform = "none"

	// defaultButtonVerticalAlign is the default vertical alignment for buttons.
	defaultButtonVerticalAlign = "middle"
)

// NewButton creates a new Button component instance.
//
// A Button renders a button that works across different email clients, with
// VML fallback for Outlook.
//
// Returns *Button which is the configured button ready for use.
func NewButton() *Button {
	return &Button{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML custom element tag name for this component.
//
// Returns string which is the custom element tag name.
func (*Button) TagName() string {
	return "pml-button"
}

// IsEndingTag returns true because Button treats its children as raw HTML
// content.
//
// Returns bool which is always true for this element type.
func (*Button) IsEndingTag() bool {
	return true
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent component names.
func (*Button) AllowedParents() []string {
	return []string{"pml-col", "pml-hero"}
}

// buttonAllowedAttributes defines the valid attributes for buttons.
var buttonAllowedAttributes = map[string]pml_domain.AttributeDefinition{
	AttrHref:                     NewAttributeDefinition(pml_domain.TypeString),
	AttrAlign:                    NewEnumAttributeDefinition([]string{ValueLeft, ValueCentre, ValueRight}),
	CSSBackgroundColor:           NewAttributeDefinition(pml_domain.TypeColor),
	AttrBorderRadius:             NewAttributeDefinition(pml_domain.TypeUnit),
	AttrBorder:                   NewAttributeDefinition(pml_domain.TypeString),
	AttrColour:                   NewAttributeDefinition(pml_domain.TypeColor),
	CSSFontFamily:                NewAttributeDefinition(pml_domain.TypeString),
	CSSFontSize:                  NewAttributeDefinition(pml_domain.TypeUnit),
	CSSFontWeight:                NewAttributeDefinition(pml_domain.TypeString),
	AttrHeight:                   NewAttributeDefinition(pml_domain.TypeUnit),
	AttrInnerPadding:             NewAttributeDefinition(pml_domain.TypeUnit),
	CSSLineHeight:                NewAttributeDefinition(pml_domain.TypeUnit),
	AttrPadding:                  NewAttributeDefinition(pml_domain.TypeUnit),
	AttrTarget:                   NewAttributeDefinition(pml_domain.TypeString),
	CSSTextDecoration:            NewAttributeDefinition(pml_domain.TypeString),
	CSSTextTransform:             NewAttributeDefinition(pml_domain.TypeString),
	CSSVerticalAlign:             NewEnumAttributeDefinition([]string{ValueTop, ValueMiddle, ValueBottom}),
	AttrWidth:                    NewAttributeDefinition(pml_domain.TypeUnit),
	AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
	AttrTitle:                    NewAttributeDefinition(pml_domain.TypeString),
	AttrRel:                      NewAttributeDefinition(pml_domain.TypeString),
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which maps attribute names
// to their definitions.
func (*Button) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return buttonAllowedAttributes
}

// DefaultAttributes returns the default attribute values for the pml-button
// element.
//
// Returns map[string]string which contains the default styling attributes.
func (*Button) DefaultAttributes() map[string]string {
	return map[string]string{
		AttrAlign:          defaultButtonAlign,
		CSSBackgroundColor: defaultButtonBgColor,
		AttrBorder:         defaultButtonBorder,
		AttrBorderRadius:   defaultButtonBorderRadius,
		AttrColour:         defaultButtonColor,
		CSSFontFamily:      defaultButtonFontFamily,
		CSSFontSize:        defaultButtonFontSize,
		CSSFontWeight:      defaultButtonFontWeight,
		AttrInnerPadding:   defaultButtonInnerPadding,
		CSSLineHeight:      defaultButtonLineHeight,
		AttrPadding:        defaultButtonPadding,
		AttrTarget:         defaultButtonTarget,
		CSSTextDecoration:  defaultButtonTextDecoration,
		CSSTextTransform:   defaultButtonTextTransform,
		CSSVerticalAlign:   defaultButtonVerticalAlign,
	}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which maps CSS properties to their
// rendering targets (container, cell, or link).
func (*Button) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: AttrAlign, Target: TargetContainer},
		{Property: AttrWidth, Target: TargetContainer},
		{Property: CSSBackgroundColor, Target: TargetCell},
		{Property: AttrBorderRadius, Target: TargetCell},
		{Property: AttrBorder, Target: TargetCell},
		{Property: AttrHeight, Target: TargetCell},
		{Property: AttrInnerPadding, Target: TargetLink},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: CSSVerticalAlign, Target: TargetCell},
		{Property: AttrColour, Target: TargetLink},
		{Property: CSSFontFamily, Target: TargetLink},
		{Property: CSSFontSize, Target: TargetLink},
		{Property: CSSFontWeight, Target: TargetLink},
		{Property: CSSLineHeight, Target: TargetLink},
		{Property: CSSTextDecoration, Target: TargetLink},
		{Property: CSSTextTransform, Target: TargetLink},
	}
}

// Transform converts a pml-button element into either a linked or non-linked
// HTML button.
//
// pml-button does NOT apply its padding, align, or container-background-colour
// to itself. These attributes are stored as data-pml-* attributes so the
// parent pml-col can read them and apply them to the wrapper <td> element.
//
// Takes node (*ast_domain.TemplateNode) which is the button element to
// transform.
// Takes ctx (*pml_domain.TransformationContext) which provides style and
// diagnostic management.
//
// Returns *ast_domain.TemplateNode which is the transformed HTML element.
// Returns []*pml_domain.Error which contains any diagnostics from the
// transformation.
func (c *Button) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	var contentNodes []*ast_domain.TemplateNode
	if len(node.Children) > 0 {
		contentNodes = processInlineChildren(node.Children)
	}

	href, hasLink := styles.Get(AttrHref)

	var buttonNode *ast_domain.TemplateNode
	if hasLink {
		buttonNode = c.renderLinkedButton(styles, contentNodes, href)
	} else {
		buttonNode = c.renderNonLinkedButton(styles, contentNodes)
	}

	transferPikoDirectives(node, buttonNode)
	return buttonNode, ctx.Diagnostics()
}

// renderLinkedButton creates the structure for a button with a link, using an
// anchor tag and VML fallback for older email clients.
//
// Takes styles (*pml_domain.StyleManager) which provides the button styling.
// Takes contentNodes ([]*ast_domain.TemplateNode) which contains the button
// content.
// Takes href (string) which specifies the link destination.
//
// Returns *ast_domain.TemplateNode which is the button fragment with VML
// fallback.
func (c *Button) renderLinkedButton(styles *pml_domain.StyleManager, contentNodes []*ast_domain.TemplateNode, href string) *ast_domain.TemplateNode {
	linkStyles := buildLinkStyles(styles)
	cellStyles := buildCellStyles(styles)
	tableStyles := buildTableStyles(styles)

	linkAttrs := buildLinkAttributes(styles, linkStyles, href)
	linkNode := NewElementNode(ElementA, linkAttrs, contentNodes)
	tableNode := createButtonTable(styles, tableStyles, cellStyles, linkNode)

	textContent := extractTextContent(contentNodes)
	vmlButton := c.renderVML(styles, textContent, href)

	return buildButtonFragment(vmlButton, tableNode)
}

// renderNonLinkedButton creates a table with a paragraph element for buttons
// that do not have links.
//
// Takes styles (*pml_domain.StyleManager) which provides the style settings.
// Takes contentNodes ([]*ast_domain.TemplateNode) which contains the button
// content.
//
// Returns *ast_domain.TemplateNode which is the table structure with the
// styled paragraph.
func (*Button) renderNonLinkedButton(styles *pml_domain.StyleManager, contentNodes []*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	pStyles := buildParagraphStyles(styles)
	cellStyles := buildCellStyles(styles)
	tableStyles := buildTableStyles(styles)

	pNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  ElementP,
		Attributes: []ast_domain.HTMLAttribute{
			{
				Name:           AttrStyle,
				Value:          mapToStyleString(pStyles),
				Location:       NewLocation(),
				NameLocation:   NewLocation(),
				AttributeRange: NewRange(),
			},
		},
		Children:           contentNodes,
		Key:                nil,
		DirKey:             nil,
		DirHTML:            nil,
		GoAnnotations:      nil,
		RuntimeAnnotations: nil,
		CustomEvents:       nil,
		OnEvents:           nil,
		Binds:              nil,
		DirContext:         nil,
		DirElse:            nil,
		DirText:            nil,
		DirStyle:           nil,
		DirClass:           nil,
		DirIf:              nil,
		DirElseIf:          nil,
		DirFor:             nil,
		DirShow:            nil,
		DirRef:             nil,
		DirModel:           nil,
		DirScaffold:        nil,
		TextContent:        "",
		InnerHTML:          "",
		RichText:           nil,
		Diagnostics:        nil,
		DynamicAttributes:  nil,
		Directives:         nil,
		Location:           NewLocation(),
		NodeRange:          NewRange(),
		OpeningTagRange:    NewRange(),
		ClosingTagRange:    NewRange(),
		PreferredFormat:    0,
		IsPooled:           false,
		IsContentEditable:  false,
	}

	return createButtonTable(styles, tableStyles, cellStyles, pNode)
}

// renderVML builds a VML roundrect element for Outlook email clients.
//
// Takes styles (*pml_domain.StyleManager) which provides button styling.
// Takes content (string) which is the button label text.
// Takes href (string) which is the button link URL.
//
// Returns *ast_domain.TemplateNode which contains the VML markup wrapped in
// conditional comments.
func (*Button) renderVML(styles *pml_domain.StyleManager, content, href string) *ast_domain.TemplateNode {
	width := mustGetStyle(styles, AttrWidth)
	height := mustGetStyle(styles, AttrHeight)
	bgColor := getStyleWithDefault(styles, CSSBackgroundColor, defaultButtonBgColor)
	borderRadius := getStyleWithDefault(styles, AttrBorderRadius, defaultButtonBorderRadius)
	border := getStyleWithDefault(styles, AttrBorder, defaultButtonBorder)

	vRectStyle := buildButtonVMLRectStyle(width, height)
	arcsize := calculateArcsize(borderRadius, height)
	strokeColor, strokeWeight, hasStroke := parseVMLStroke(border, bgColor)
	vCentreStyles := buildVMLCentreStyles(styles)
	vmlAttrs := buildVMLAttributes(href, vRectStyle, arcsize, hasStroke, strokeColor, strokeWeight, bgColor)
	vmlInset := buildVMLInset(styles)
	vmlHTML := `<v:roundrect ` + vmlAttrs +
		`><w:anchorlock/><v:textbox style="mso-fit-shape-to-text:true" inset="` + vmlInset +
		`"><center style=` + fmt.Sprintf("%q", vCentreStyles) +
		`>` + content + `</center></v:textbox></v:roundrect>`

	return NewRawHTMLNode(fmt.Sprintf("%s%s%s", ConditionalCommentStart, vmlHTML, ConditionalCommentEnd))
}

// buildVMLInset constructs the v:textbox inset attribute value from the
// button's inner-padding. The VML inset format is "left,top,right,bottom".
//
// Takes styles (*pml_domain.StyleManager) which provides the padding values.
//
// Returns string which is the comma-separated inset value for VML.
func buildVMLInset(styles *pml_domain.StyleManager) string {
	padding := getButtonPadding(styles)
	top, right, bottom, left := parsePadding(padding)
	return fmt.Sprintf("%s,%s,%s,%s",
		coalesce(left, vmlInsetDefault),
		coalesce(top, vmlInsetDefault),
		coalesce(right, vmlInsetDefault),
		coalesce(bottom, vmlInsetDefault),
	)
}

// extractTextContent walks through AST nodes and collects all text content.
// This is used for VML rendering, which cannot handle rich HTML like <span>
// tags.
//
// Takes nodes ([]*ast_domain.TemplateNode) which contains the AST nodes to
// extract text from.
//
// Returns string which contains the combined text content, trimmed of
// surrounding whitespace.
func extractTextContent(nodes []*ast_domain.TemplateNode) string {
	var result strings.Builder
	for _, node := range nodes {
		if node == nil {
			continue
		}
		switch node.NodeType {
		case ast_domain.NodeText:
			result.WriteString(node.TextContent)
		case ast_domain.NodeElement:
			result.WriteString(extractTextContent(node.Children))
		default:
		}
	}
	return strings.TrimSpace(result.String())
}

// buildLinkAttributes builds the HTML attributes for the button link.
//
// Takes styles (*pml_domain.StyleManager) which provides style values for
// target, rel, and title attributes.
// Takes linkStyles (map[string]string) which contains the inline CSS styles.
// Takes href (string) which specifies the link destination URL.
//
// Returns []ast_domain.HTMLAttribute which contains the sorted link attributes
// including href, target, style, and optionally rel and title.
func buildLinkAttributes(styles *pml_domain.StyleManager, linkStyles map[string]string, href string) []ast_domain.HTMLAttribute {
	target := getStyleWithDefault(styles, AttrTarget, defaultButtonTarget)

	linkAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrHref, href),
		NewHTMLAttribute(AttrTarget, target),
		NewHTMLAttribute(AttrStyle, mapToStyleString(linkStyles)),
	}

	if rel := mustGetStyle(styles, AttrRel); rel != "" {
		linkAttrs = append(linkAttrs, NewHTMLAttribute(AttrRel, rel))
	}
	if title := mustGetStyle(styles, AttrTitle); title != "" {
		linkAttrs = append(linkAttrs, NewHTMLAttribute(AttrTitle, title))
	}

	return sortHTMLAttributes(linkAttrs)
}

// buildButtonFragment creates a fragment node containing VML and the modern
// button table.
//
// Takes vmlButton (*ast_domain.TemplateNode) which provides the VML button
// markup for Outlook compatibility.
// Takes tableNode (*ast_domain.TemplateNode) which provides the modern HTML
// table button for other email clients.
//
// Returns *ast_domain.TemplateNode which contains both button variants wrapped
// in conditional comments.
func buildButtonFragment(vmlButton, tableNode *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	return NewFragmentNode([]*ast_domain.TemplateNode{
		vmlButton,
		NewRawHTMLNode(ConditionalNotMsoStart),
		tableNode,
		NewRawHTMLNode(ConditionalNotMsoEnd),
	})
}

// getStyleWithDefault retrieves a style value from the manager or returns a
// default value if the key is missing or empty.
//
// Takes sm (*pml_domain.StyleManager) which provides access to style values.
// Takes key (string) which identifies the style to retrieve.
// Takes defaultValue (string) which is returned if the key is not found.
//
// Returns string which is the style value or the default.
func getStyleWithDefault(sm *pml_domain.StyleManager, key, defaultValue string) string {
	if value, ok := sm.Get(key); ok && value != "" {
		return value
	}
	return defaultValue
}

// getButtonPadding returns the inner padding value for a button element.
//
// It checks styles in order: inner-padding, then padding, then the default
// value. This handles the rule where padding on a button refers to its inner
// padding.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes.
//
// Returns string which is the padding value to use.
func getButtonPadding(styles *pml_domain.StyleManager) string {
	if value, ok := styles.Get(AttrInnerPadding); ok {
		return value
	}
	if value, ok := styles.Get(AttrPadding); ok {
		return value
	}
	return defaultButtonInnerPadding
}

// buildLinkStyles builds a map of CSS styles for link-styled buttons.
//
// Takes styles (*pml_domain.StyleManager) which provides style overrides.
//
// Returns map[string]string which contains CSS property-value pairs with
// default values used for any missing styles.
func buildLinkStyles(styles *pml_domain.StyleManager) map[string]string {
	return map[string]string{
		CSSBackground:     getStyleWithDefault(styles, CSSBackgroundColor, defaultButtonBgColor),
		AttrBorderRadius:  getStyleWithDefault(styles, AttrBorderRadius, defaultButtonBorderRadius),
		AttrColour:        getStyleWithDefault(styles, AttrColour, defaultButtonColor),
		CSSDisplay:        ValueInlineBlock,
		CSSFontFamily:     getStyleWithDefault(styles, CSSFontFamily, defaultButtonFontFamily),
		CSSFontSize:       getStyleWithDefault(styles, CSSFontSize, defaultButtonFontSize),
		CSSFontWeight:     getStyleWithDefault(styles, CSSFontWeight, defaultButtonFontWeight),
		CSSLineHeight:     getStyleWithDefault(styles, CSSLineHeight, defaultButtonLineHeight),
		CSSMargin:         ValueZero,
		CSSPadding:        getButtonPadding(styles),
		CSSTextDecoration: getStyleWithDefault(styles, CSSTextDecoration, defaultButtonTextDecoration),
		CSSTextTransform:  getStyleWithDefault(styles, CSSTextTransform, defaultButtonTextTransform),
	}
}

// buildParagraphStyles builds a CSS style map for paragraph elements.
//
// Takes styles (*pml_domain.StyleManager) which provides the style settings.
//
// Returns map[string]string which contains the paragraph styles, including link
// styles and Outlook padding reset.
func buildParagraphStyles(styles *pml_domain.StyleManager) map[string]string {
	pStyles := buildLinkStyles(styles)
	pStyles[CSSMsoPaddingAlt] = ValueZeroPx
	return pStyles
}

// buildCellStyles creates CSS style mappings for table cell elements.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values to
// apply.
//
// Returns map[string]string which contains the CSS properties for the cell.
func buildCellStyles(styles *pml_domain.StyleManager) map[string]string {
	cellStyles := map[string]string{
		CSSBackgroundColor: getStyleWithDefault(styles, CSSBackgroundColor, defaultButtonBgColor),
		AttrBorderRadius:   getStyleWithDefault(styles, AttrBorderRadius, defaultButtonBorderRadius),
		CSSCursor:          ValueAuto,
		CSSMsoPaddingAlt:   getButtonPadding(styles),
	}

	if border, ok := styles.Get(AttrBorder); ok && border != ValueNone {
		cellStyles[AttrBorder] = border
	} else {
		cellStyles[AttrBorder] = ValueNone
	}

	if height := mustGetStyle(styles, AttrHeight); height != "" {
		cellStyles[AttrHeight] = height
	}
	return cellStyles
}

// buildTableStyles creates a CSS style map for table elements.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values.
//
// Returns map[string]string which holds the CSS properties for the table.
func buildTableStyles(styles *pml_domain.StyleManager) map[string]string {
	tableStyles := map[string]string{
		CSSBorderCollapse: ValueSeparate,
		CSSLineHeight:     Value100,
	}
	if width := mustGetStyle(styles, AttrWidth); width != "" {
		tableStyles[AttrWidth] = width
	}
	return tableStyles
}

// createButtonTable creates the button's table structure with data-pml-*
// attributes.
//
// The table itself does not have padding or alignment applied. These are stored
// as data-pml-* attributes so the parent pml-col can apply them.
//
// Takes styles (*pml_domain.StyleManager) which provides style management.
// Takes tableStyles (map[string]string) which specifies the table styling.
// Takes cellStyles (map[string]string) which specifies the cell styling.
// Takes contentNode (*ast_domain.TemplateNode) which is the button content.
//
// Returns *ast_domain.TemplateNode which is the constructed table element.
func createButtonTable(styles *pml_domain.StyleManager, tableStyles, cellStyles map[string]string, contentNode *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	tableAttrs := buildButtonTableAttributes(styles, tableStyles)
	tdNode := buildButtonTDNode(styles, cellStyles, contentNode)
	trNode := NewElementNode(ElementTr, nil, []*ast_domain.TemplateNode{tdNode})
	tbody := NewElementNode(ElementTbody, nil, []*ast_domain.TemplateNode{trNode})

	return NewElementNode(ElementTable, tableAttrs, []*ast_domain.TemplateNode{tbody})
}

// buildButtonTableAttributes builds the HTML attributes for a button table.
//
// Takes styles (*pml_domain.StyleManager) which provides style values for
// padding and alignment.
// Takes tableStyles (map[string]string) which contains the inline CSS styles
// to apply.
//
// Returns []ast_domain.HTMLAttribute which contains the sorted attributes
// including the presentation role, styling, and directional padding data.
func buildButtonTableAttributes(styles *pml_domain.StyleManager, tableStyles map[string]string) []ast_domain.HTMLAttribute {
	tableAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrBorder, ValueZero),
		NewHTMLAttribute(AttrCellPadding, ValueZero),
		NewHTMLAttribute(AttrCellSpacing, ValueZero),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrStyle, mapToStyleString(tableStyles)),
		NewHTMLAttribute("data-pml-padding", mustGetStyle(styles, AttrPadding)),
		NewHTMLAttribute("data-pml-align", mustGetStyle(styles, AttrAlign)),
	}

	addDirectionalPaddingAttributes(&tableAttrs, styles)
	addContainerBgAttribute(&tableAttrs, styles)

	return sortHTMLAttributes(tableAttrs)
}

// addDirectionalPaddingAttributes adds directional padding data attributes
// when they exist in the style manager.
//
// Takes tableAttrs (*[]ast_domain.HTMLAttribute) which receives the padding
// attributes.
// Takes styles (*pml_domain.StyleManager) which provides the style values.
func addDirectionalPaddingAttributes(tableAttrs *[]ast_domain.HTMLAttribute, styles *pml_domain.StyleManager) {
	paddingAttrs := map[string]string{
		"padding-top":    "data-pml-padding-top",
		"padding-right":  "data-pml-padding-right",
		"padding-bottom": "data-pml-padding-bottom",
		"padding-left":   "data-pml-padding-left",
	}

	for styleKey, attributeName := range paddingAttrs {
		if value := mustGetStyle(styles, styleKey); value != "" {
			*tableAttrs = append(*tableAttrs, NewHTMLAttribute(attributeName, value))
		}
	}
}

// addContainerBgAttribute adds a container background colour attribute to the
// table attributes if the style is set.
//
// Takes tableAttrs (*[]ast_domain.HTMLAttribute) which receives the new
// attribute if the background colour style is set.
// Takes styles (*pml_domain.StyleManager) which provides the style values to
// check.
func addContainerBgAttribute(tableAttrs *[]ast_domain.HTMLAttribute, styles *pml_domain.StyleManager) {
	if containerBg := mustGetStyle(styles, AttrContainerBackgroundColor); containerBg != "" {
		*tableAttrs = append(*tableAttrs, NewHTMLAttribute("data-pml-container-background-color", containerBg))
	}
}

// buildButtonTDNode creates the TD element that wraps the button content.
//
// Takes styles (*pml_domain.StyleManager) which provides style values for the
// cell attributes.
// Takes cellStyles (map[string]string) which contains CSS styles for the cell.
// Takes contentNode (*ast_domain.TemplateNode) which is the button content to
// wrap.
//
// Returns *ast_domain.TemplateNode which is the configured TD element.
func buildButtonTDNode(styles *pml_domain.StyleManager, cellStyles map[string]string, contentNode *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	tdAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrAlign, ValueCentre),
		NewHTMLAttribute(AttrBgColor, getStyleWithDefault(styles, CSSBackgroundColor, defaultButtonBgColor)),
		NewHTMLAttribute(AttrRole, ValuePresentation),
		NewHTMLAttribute(AttrStyle, mapToStyleString(cellStyles)),
		NewHTMLAttribute(AttrValign, getStyleWithDefault(styles, CSSVerticalAlign, defaultButtonVerticalAlign)),
	}

	return NewElementNode(ElementTd, tdAttrs, []*ast_domain.TemplateNode{contentNode})
}

// buildButtonVMLRectStyle constructs the VML rectangle style string for button
// components.
//
// Takes width (string) which specifies the button width.
// Takes height (string) which specifies the button height.
//
// Returns string which contains the formatted VML style attributes, defaulting
// to "height:40px;width:200px;" when both parameters are empty.
func buildButtonVMLRectStyle(width, height string) string {
	var vRectStyle strings.Builder
	if height != "" {
		_, _ = fmt.Fprintf(&vRectStyle, "%s:%s;", AttrHeight, height)
	}
	if width != "" {
		_, _ = fmt.Fprintf(&vRectStyle, "%s:%s;", AttrWidth, width)
	}
	if vRectStyle.Len() == 0 {
		vRectStyle.WriteString("height:40px;width:200px;")
	}
	return vRectStyle.String()
}

// parseVMLStroke extracts stroke properties from a border string for VML
// rendering.
//
// Takes border (string) which contains the border definition to parse.
// Takes bgColor (string) which provides the fallback stroke colour.
//
// Returns strokeColor (string) which is the extracted or fallback colour.
// Returns strokeWeight (string) which is the stroke width if found.
// Returns hasStroke (bool) which indicates whether a stroke should be drawn.
func parseVMLStroke(border, bgColor string) (strokeColor, strokeWeight string, hasStroke bool) {
	hasStroke = border != ValueNone && border != ""
	strokeColor = bgColor
	strokeWeight = ""

	if hasStroke {
		parts := strings.Fields(border)
		if len(parts) >= BorderPartsMinimum {
			strokeWeight = parts[0]
			strokeColor = parts[2]
		}
	}

	return strokeColor, strokeWeight, hasStroke
}

// buildVMLCentreStyles builds the VML centre element style string.
//
// Takes styles (*pml_domain.StyleManager) which provides style values with
// defaults for colour, font family, size, and weight.
//
// Returns string which contains the formatted CSS style declarations.
func buildVMLCentreStyles(styles *pml_domain.StyleManager) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%s:%s;%s:%s;%s:%s;%s:%s;",
		AttrColour, getStyleWithDefault(styles, AttrColour, defaultButtonColor),
		CSSFontFamily, getStyleWithDefault(styles, CSSFontFamily, "Arial, Helvetica, sans-serif"),
		CSSFontSize, getStyleWithDefault(styles, CSSFontSize, defaultButtonFontSize),
		CSSFontWeight, getStyleWithDefault(styles, CSSFontWeight, defaultButtonFontWeight))
	return b.String()
}

// buildVMLAttributes constructs the complete VML roundrect attributes string.
//
// Takes href (string) which is the link destination URL.
// Takes vRectStyle (string) which defines the VML rectangle CSS style.
// Takes arcsize (string) which specifies the corner rounding percentage.
// Takes hasStroke (bool) which controls whether a border stroke is applied.
// Takes strokeColor (string) which sets the border colour when hasStroke is
// true.
// Takes strokeWeight (string) which sets the border thickness when hasStroke
// is true.
// Takes bgColor (string) which specifies the fill colour.
//
// Returns string which contains the formatted VML attribute string for use in
// a roundrect element.
func buildVMLAttributes(href, vRectStyle, arcsize string, hasStroke bool, strokeColor, strokeWeight, bgColor string) string {
	var vmlAttrs strings.Builder
	_, _ = fmt.Fprintf(&vmlAttrs, `xmlns:v="urn:schemas-microsoft-com:vml" xmlns:w="urn:schemas-microsoft-com:office:word" href=%q style=%q arcsize=%q`, href, vRectStyle, arcsize)

	if hasStroke {
		_, _ = fmt.Fprintf(&vmlAttrs, ` strokecolor=%q strokeweight=%q`, strokeColor, strokeWeight)
	} else {
		vmlAttrs.WriteString(` stroke="false"`)
	}

	_, _ = fmt.Fprintf(&vmlAttrs, ` fillcolor=%q`, bgColor)
	return vmlAttrs.String()
}

// calculateArcsize computes the VML arcsize percentage from border radius and
// height values.
//
// Takes borderRadius (string) which specifies the corner radius in pixels.
// Takes height (string) which specifies the element height in pixels.
//
// Returns string which is the arcsize as a percentage value for VML rendering.
func calculateArcsize(borderRadius, height string) string {
	radiusPx := mustParsePixels(borderRadius)
	if radiusPx == 0 {
		return "0%"
	}
	if height != "" {
		heightPx := mustParsePixels(height)
		if heightPx > 0 {
			arcsizeVal := (float64(radiusPx) / float64(heightPx)) * PercentFull
			if arcsizeVal > NumericFifty {
				arcsizeVal = NumericFifty
			}
			return fmt.Sprintf("%.0f%%", arcsizeVal)
		}
	}
	if radiusPx >= NumericTwenty {
		return "50%"
	}
	return "10%"
}
