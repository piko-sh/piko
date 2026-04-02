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

// OrderedList represents the <pml-ol> tag and implements the
// pml_domain.Component interface. It renders HTML ordered lists that work
// well with screen readers and email clients, including Gmail and Outlook.
type OrderedList struct {
	BaseComponent
}

var _ pml_domain.Component = (*OrderedList)(nil)

// NewOrderedList creates a new OrderedList component instance.
//
// An OrderedList renders semantic HTML lists (<ul> or <ol>) with proper
// email client compatibility.
//
// Returns *OrderedList which is the component ready for configuration.
func NewOrderedList() *OrderedList {
	return &OrderedList{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the HTML custom element tag name for this component.
//
// Returns string which is the tag name "pml-ol".
func (*OrderedList) TagName() string {
	return "pml-ol"
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as ordered lists have no
// default attributes.
func (*OrderedList) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent component names.
func (*OrderedList) AllowedParents() []string {
	return []string{"pml-col", "pml-hero", "pml-li"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute definitions for list style, bullet type, typography, spacing,
// alignment, and container styling.
func (*OrderedList) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrListStyle: NewEnumAttributeDefinition([]string{ValueUnordered, ValueOrdered}),

		AttrType: NewAttributeDefinition(pml_domain.TypeString),

		CSSFontFamily:     NewAttributeDefinition(pml_domain.TypeString),
		CSSFontSize:       NewAttributeDefinition(pml_domain.TypeUnit),
		CSSLineHeight:     NewAttributeDefinition(pml_domain.TypeUnit),
		CSSColor:          NewAttributeDefinition(pml_domain.TypeColor),
		CSSMarginLeft:     NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPadding:       NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingTop:    NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingBottom: NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingLeft:   NewAttributeDefinition(pml_domain.TypeUnit),
		AttrPaddingRight:  NewAttributeDefinition(pml_domain.TypeUnit),

		AttrAlign: NewEnumAttributeDefinition([]string{ValueLeft, ValueCentre, ValueRight}),

		AttrContainerBackgroundColor: NewAttributeDefinition(pml_domain.TypeColor),
	}
}

// GetStyleTargets returns the style targets for this ordered list.
//
// Returns []pml_domain.StyleTarget which lists the CSS properties and their
// target elements that can be styled.
func (*OrderedList) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: CSSFontFamily, Target: TargetContainer},
		{Property: CSSFontSize, Target: TargetContainer},
		{Property: CSSLineHeight, Target: TargetContainer},
		{Property: CSSColor, Target: TargetContainer},
		{Property: CSSMarginLeft, Target: TargetContainer},
		{Property: AttrPadding, Target: TargetContainer},
		{Property: AttrAlign, Target: TargetContainer},
	}
}

// Transform converts the <pml-ol> node into its final, email-safe HTML
// structure. This generates a properly formatted `<ul>` or `<ol>` element with
// all necessary workarounds for Gmail and Outlook compatibility.
//
// Takes node (*ast_domain.TemplateNode) which is the source node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the style
// manager and collects diagnostics.
//
// Returns *ast_domain.TemplateNode which is the transformed list wrapped in an
// Outlook compatibility div.
// Returns []*pml_domain.Error which contains any diagnostics collected during
// transformation.
//
// The transformation implements several key patterns from the Litmus guide:
//
//  1. **Semantic HTML Structure**:
//     Renders as proper `<ul>` or `<ol>` tags with `<li>` children for
//     accessibility. Screen readers will correctly announce "OrderedList with
//     X items", "Bullet", "Out of list".
//
//  2. **Margin-Left for Bullet Alignment**:
//     Applies `margin-left: 25px` (or custom value) to ensure bullets render
//     inside container boundaries rather than being cut off or misaligned.
//
//  3. **Gmail Webmail Fix**:
//     Adds a CSS class that can be targeted with Gmail-specific selectors to
//     remove the margin-left indentation in Gmail webmail only (not mobile
//     app).
//
//  4. **Outlook Container**:
//     Wraps the list in a `<div class="pml-list-wrapper">` to eliminate
//     Outlook's large default margins. This wrapper is styled via Outlook
//     conditional CSS.
//
//  5. **Style Application**:
//     Applies typography (font-family, font-size, colour, line-height) directly
//     to the list element. Individual list items inherit these styles and can
//     override them.
//
//  6. **Piko Directive Preservation**:
//     All `p-*` directives are transferred to the outermost element (the
//     Outlook wrapper div), allowing for dynamic lists:
//     `<pml-ol p-for="item in items">`.
func (*OrderedList) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	tagName, listType, marginLeft, align := determineListProperties(styles)

	listStyles := buildOrderedListStyles(styles, marginLeft)

	listAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrType, listType),
		NewHTMLAttribute(AttrAlign, align),
		NewHTMLAttribute(AttrClass, ClassList),
		NewHTMLAttribute(AttrStyle, mapToStyleString(listStyles)),
	}
	listNode := NewElementNode(tagName, listAttrs, node.Children)

	registerListMSOStyles(ctx)

	wrapperDiv := createListWrapper(listNode, styles)

	transferPikoDirectives(node, wrapperDiv)

	return wrapperDiv, ctx.Diagnostics()
}

// determineListProperties gets the style settings for a list element.
//
// Takes styles (*pml_domain.StyleManager) which provides access to the style
// values for the list element.
//
// Returns tagName (string) which is the HTML element name (ul or ol).
// Returns listType (string) which is the bullet or number style.
// Returns marginLeft (string) which is the left margin for bullet position.
// Returns align (string) which is the text alignment for the list.
func determineListProperties(styles *pml_domain.StyleManager) (tagName, listType, marginLeft, align string) {
	listStyle := mustGetStyle(styles, AttrListStyle)
	if listStyle == "" {
		listStyle = defaultListStyle
	}

	tagName = ElementUl
	if listStyle == ValueOrdered {
		tagName = ElementOl
	}

	listType = mustGetStyle(styles, AttrType)
	if listType == "" {
		if listStyle == ValueUnordered {
			listType = defaultUnorderedType
		} else {
			listType = defaultOrderedType
		}
	}

	marginLeft = mustGetStyle(styles, CSSMarginLeft)
	if marginLeft == "" {
		marginLeft = defaultListMarginLeft
	}

	align = mustGetStyle(styles, AttrAlign)
	if align == "" {
		align = defaultListAlign
	}

	return tagName, listType, marginLeft, align
}

// buildOrderedListStyles builds the style map for an ordered list element.
//
// Takes styles (*pml_domain.StyleManager) which provides the base styles to
// copy from.
// Takes marginLeft (string) which sets the left margin value for the list.
//
// Returns map[string]string which contains the CSS styles for the ordered
// list.
func buildOrderedListStyles(styles *pml_domain.StyleManager, marginLeft string) map[string]string {
	listStyles := map[string]string{
		CSSMargin:     ValueZero,
		CSSMarginLeft: marginLeft,
		CSSPadding:    ValueZero,
	}

	copyStyle(styles, listStyles, CSSFontFamily)
	copyStyle(styles, listStyles, CSSFontSize)
	copyStyle(styles, listStyles, CSSLineHeight)
	copyStyle(styles, listStyles, CSSColor)

	top, right, bottom, left := expandPadding(styles)
	if top != "" || right != "" || bottom != "" || left != "" {
		if top == "" {
			top = ValueZero
		}
		if right == "" {
			right = ValueZero
		}
		if bottom == "" {
			bottom = ValueZero
		}
		if left == "" {
			left = ValueZero
		}
		listStyles[CSSPadding] = top + ValueSpace + right + ValueSpace + bottom + ValueSpace + left
	}

	return listStyles
}

// registerListMSOStyles registers Outlook-specific conditional styles for
// correct list display in Microsoft Outlook email clients.
//
// Takes ctx (*pml_domain.TransformationContext) which provides access to the
// MSO conditional style collector.
func registerListMSOStyles(ctx *pml_domain.TransformationContext) {
	if ctx.MSOConditionalCollector == nil {
		return
	}

	ctx.MSOConditionalCollector.RegisterStyle(ElementUl, msoUlMargin)

	ctx.MSOConditionalCollector.RegisterStyle(ElementLi, msoListItemMarginLeft)

	ctx.MSOConditionalCollector.RegisterStyle(ElementLi+"."+ClassListFirst, msoFirstListItemMarginTop)

	ctx.MSOConditionalCollector.RegisterStyle(ElementLi+"."+ClassListLast, msoLastListItemMarginBottom)
}

// createListWrapper wraps a list node in a div for Outlook email clients.
// It also adds a background colour wrapper if one is set in the styles.
//
// Takes listNode (*ast_domain.TemplateNode) which is the list to wrap.
// Takes styles (*pml_domain.StyleManager) which provides style values.
//
// Returns *ast_domain.TemplateNode which is the wrapped list node.
func createListWrapper(listNode *ast_domain.TemplateNode, styles *pml_domain.StyleManager) *ast_domain.TemplateNode {
	wrapperDiv := NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrClass, ClassListWrapper),
	}, []*ast_domain.TemplateNode{listNode})

	if bgColor, ok := styles.Get(AttrContainerBackgroundColor); ok && bgColor != "" {
		bgStyle := fmt.Sprintf("%s:%s;", CSSBackgroundColor, bgColor)
		wrapperDiv = NewElementNode(ElementTd, []ast_domain.HTMLAttribute{
			NewHTMLAttribute(AttrStyle, bgStyle),
		}, []*ast_domain.TemplateNode{wrapperDiv})
	}

	return wrapperDiv
}
