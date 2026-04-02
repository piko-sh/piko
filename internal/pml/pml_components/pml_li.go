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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// ListItem implements pml_domain.Component for the <pml-li> tag.
// It renders list items with separate styling for bullet points and content.
type ListItem struct {
	BaseComponent
}

var _ pml_domain.Component = (*ListItem)(nil)

// NewListItem creates a new ListItem component instance.
// A ListItem renders individual list items with separate bullet and text
// styling.
//
// Returns *ListItem which is the initialised component ready for use.
func NewListItem() *ListItem {
	return &ListItem{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the HTML tag name "pml-li".
func (*ListItem) TagName() string {
	return "pml-li"
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent component names.
func (*ListItem) AllowedParents() []string {
	return []string{"pml-ol"}
}

// listItemAllowedAttributes holds the allowed attributes for list items.
var listItemAllowedAttributes = map[string]pml_domain.AttributeDefinition{
	AttrBulletColor:      NewAttributeDefinition(pml_domain.TypeColor),
	AttrBulletFontSize:   NewAttributeDefinition(pml_domain.TypeUnit),
	AttrBulletFontFamily: NewAttributeDefinition(pml_domain.TypeString),
	AttrBulletFontWeight: NewAttributeDefinition(pml_domain.TypeString),
	AttrBulletFontStyle:  NewAttributeDefinition(pml_domain.TypeString),
	AttrBulletLineHeight: NewAttributeDefinition(pml_domain.TypeUnit),

	AttrColour:    NewAttributeDefinition(pml_domain.TypeColor),
	CSSFontSize:   NewAttributeDefinition(pml_domain.TypeUnit),
	CSSFontFamily: NewAttributeDefinition(pml_domain.TypeString),
	CSSFontWeight: NewAttributeDefinition(pml_domain.TypeString),
	CSSFontStyle:  NewAttributeDefinition(pml_domain.TypeString),
	CSSLineHeight: NewAttributeDefinition(pml_domain.TypeUnit),

	AttrMarginBottom: NewAttributeDefinition(pml_domain.TypeUnit),
	AttrPaddingLeft:  NewAttributeDefinition(pml_domain.TypeUnit),
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which maps attribute names
// to their definitions.
func (*ListItem) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return listItemAllowedAttributes
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as list items have no
// default attributes.
func (*ListItem) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// GetStyleTargets returns the style targets for this list item.
//
// Returns []pml_domain.StyleTarget which links style properties to their
// render targets, keeping bullet styles separate from text styles.
func (*ListItem) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: AttrBulletColor, Target: TargetBullet},
		{Property: AttrBulletFontSize, Target: TargetBullet},
		{Property: AttrBulletFontFamily, Target: TargetBullet},
		{Property: AttrBulletFontWeight, Target: TargetBullet},
		{Property: AttrBulletFontStyle, Target: TargetBullet},
		{Property: AttrBulletLineHeight, Target: TargetBullet},
		{Property: AttrMarginBottom, Target: TargetBullet},
		{Property: AttrPaddingLeft, Target: TargetBullet},

		{Property: AttrColour, Target: TargetText},
		{Property: CSSFontSize, Target: TargetText},
		{Property: CSSFontFamily, Target: TargetText},
		{Property: CSSFontWeight, Target: TargetText},
		{Property: CSSFontStyle, Target: TargetText},
		{Property: CSSLineHeight, Target: TargetText},
	}
}

// Transform converts the <pml-li> node into its final HTML structure.
//
// This generates a <li> element with bullet styling applied to it, and a
// <span> wrapper for the text content with separate styling.
//
// Takes node (*ast_domain.TemplateNode) which is the <pml-li> node to convert.
// Takes ctx (*pml_domain.TransformationContext) which provides the style
// manager and collects diagnostics.
//
// Returns *ast_domain.TemplateNode which is the transformed <li> element.
// Returns []*pml_domain.Error which contains any diagnostics from the
// transformation.
//
// The transformation implements the Litmus guide pattern for separate
// bullet/text styling:
//
//  1. Build <li> Styles (Bullet Styling):
//     Extract all bullet-* attributes and apply them to the <li> element.
//     These styles affect both the bullet and the text initially.
//
//  2. Build <span> Styles (Paragraph Styling):
//     Extract all text styling attributes (colour, font-size, etc.) and apply
//     them to a <span> that wraps the text content. This resets the text back
//     to desired styles, leaving only the bullet with the bullet-* styles.
//
//  3. Separate Nested Lists from Paragraph Content:
//     Walk through children. Paragraph and inline content go inside the
//     <span>. Any <pml-ol> children are placed directly inside the <li> after
//     the <span>. This structure prevents rogue bullets in Gmail IMAP (GANGA).
//
//  4. Piko Directive Preservation:
//     All p-* directives are transferred to the <li> element, allowing for
//     dynamic list items: <pml-li p-for="item in items">.
func (*ListItem) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager

	liStyles := buildBulletStyles(styles)
	spanStyles := buildTextStyles(styles)

	textContent, nestedLists := separateListChildren(node)
	processedTextContent := processInlineChildren(textContent)
	liChildren := wrapListItemContent(processedTextContent, nestedLists, spanStyles)

	cssClasses := getListItemPositionClasses(node, ctx)

	liAttrs := buildListItemAttributes(liStyles, cssClasses)
	liNode := NewElementNode(ElementLi, liAttrs, liChildren)

	transferPikoDirectives(node, liNode)

	return liNode, ctx.Diagnostics()
}

// buildBulletStyles extracts bullet-* attributes and maps them to standard
// CSS properties.
//
// Takes styles (*pml_domain.StyleManager) which provides access to style
// attributes.
//
// Returns map[string]string which contains the mapped CSS properties.
func buildBulletStyles(styles *pml_domain.StyleManager) map[string]string {
	liStyles := map[string]string{}

	if value, ok := styles.Get(AttrBulletColor); ok {
		liStyles[AttrColour] = value
	}
	if value, ok := styles.Get(AttrBulletFontSize); ok {
		liStyles[CSSFontSize] = value
	}
	if value, ok := styles.Get(AttrBulletFontFamily); ok {
		liStyles[CSSFontFamily] = value
	}
	if value, ok := styles.Get(AttrBulletFontWeight); ok {
		liStyles[CSSFontWeight] = value
	}
	if value, ok := styles.Get(AttrBulletFontStyle); ok {
		liStyles[CSSFontStyle] = value
	}
	if value, ok := styles.Get(AttrBulletLineHeight); ok {
		liStyles[CSSLineHeight] = value
	}

	copyStyle(styles, liStyles, AttrMarginBottom)
	copyStyle(styles, liStyles, AttrPaddingLeft)

	return liStyles
}

// buildTextStyles extracts text style attributes for a content span.
//
// Takes styles (*pml_domain.StyleManager) which provides the source styles.
//
// Returns map[string]string which contains the extracted text style attributes.
func buildTextStyles(styles *pml_domain.StyleManager) map[string]string {
	spanStyles := map[string]string{}
	copyStyle(styles, spanStyles, AttrColour)
	copyStyle(styles, spanStyles, CSSFontSize)
	copyStyle(styles, spanStyles, CSSFontFamily)
	copyStyle(styles, spanStyles, CSSFontWeight)
	copyStyle(styles, spanStyles, CSSFontStyle)
	copyStyle(styles, spanStyles, CSSLineHeight)
	return spanStyles
}

// separateListChildren splits the children of a node into text content and
// nested lists.
//
// Takes node (*ast_domain.TemplateNode) which contains the children to split.
//
// Returns textContent ([]*ast_domain.TemplateNode) which holds all child nodes
// that are not lists.
// Returns nestedLists ([]*ast_domain.TemplateNode) which holds all child nodes
// that are nested lists.
func separateListChildren(node *ast_domain.TemplateNode) (textContent, nestedLists []*ast_domain.TemplateNode) {
	for _, child := range node.Children {
		if isNestedListNode(child) {
			nestedLists = append(nestedLists, child)
		} else {
			textContent = append(textContent, child)
		}
	}
	return textContent, nestedLists
}

// wrapListItemContent wraps text content in a span element when styles are
// given, then adds any nested lists after the text.
//
// Takes textContent ([]*ast_domain.TemplateNode) which holds the text nodes
// to wrap.
// Takes nestedLists ([]*ast_domain.TemplateNode) which holds any nested list
// nodes to add after the text content.
// Takes spanStyles (map[string]string) which sets the styles to apply to the
// wrapping span element.
//
// Returns []*ast_domain.TemplateNode which holds the combined list item
// children ready to use.
func wrapListItemContent(textContent, nestedLists []*ast_domain.TemplateNode, spanStyles map[string]string) []*ast_domain.TemplateNode {
	var liChildren []*ast_domain.TemplateNode

	if len(spanStyles) > 0 {
		spanNode := NewElementNode(ElementSpan, []ast_domain.HTMLAttribute{
			NewHTMLAttribute(AttrStyle, mapToStyleString(spanStyles)),
		}, textContent)
		liChildren = append(liChildren, spanNode)
	} else {
		liChildren = append(liChildren, textContent...)
	}

	liChildren = append(liChildren, nestedLists...)
	return liChildren
}

// getListItemPositionClasses returns CSS classes based on the position of a
// list item within its parent list.
//
// Takes node (*ast_domain.TemplateNode) which is the list item to check.
// Takes ctx (*pml_domain.TransformationContext) which provides the parent node
// and sibling context.
//
// Returns []string which contains position classes such as "first" or "last".
func getListItemPositionClasses(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) []string {
	var cssClasses []string

	if ctx.ParentNode == nil || len(ctx.ParentNode.Children) == 0 {
		return cssClasses
	}

	nodeIndex := -1
	for i, child := range ctx.ParentNode.Children {
		if child == node {
			nodeIndex = i
			break
		}
	}

	if nodeIndex == 0 {
		cssClasses = append(cssClasses, ClassListFirst)
	}
	if nodeIndex == len(ctx.ParentNode.Children)-1 {
		cssClasses = append(cssClasses, ClassListLast)
	}

	return cssClasses
}

// buildListItemAttributes builds the HTML attributes for a list item element.
//
// Takes liStyles (map[string]string) which specifies the inline CSS styles.
// Takes cssClasses ([]string) which provides the CSS class names to apply.
//
// Returns []ast_domain.HTMLAttribute which contains the built attributes.
func buildListItemAttributes(liStyles map[string]string, cssClasses []string) []ast_domain.HTMLAttribute {
	liAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(liStyles)),
	}

	if len(cssClasses) > 0 {
		var builder strings.Builder
		for i, class := range cssClasses {
			if i > 0 {
				builder.WriteString(ValueSpace)
			}
			builder.WriteString(class)
		}
		liAttrs = append(liAttrs, NewHTMLAttribute(AttrClass, builder.String()))
	}

	return liAttrs
}

// isNestedListNode checks if a child node is a nested list. This includes
// pml-ol elements or ul/ol elements wrapped in a div.
//
// Takes child (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the child is a nested list node.
func isNestedListNode(child *ast_domain.TemplateNode) bool {
	if child.TagName == "pml-ol" {
		return true
	}
	if child.NodeType == ast_domain.NodeElement && child.TagName == ElementDiv && len(child.Children) > 0 {
		firstChildTag := child.Children[0].TagName
		return firstChildTag == ElementUl || firstChildTag == ElementOl
	}
	return false
}
