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
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

// NoStack implements the pml_domain.Component interface for the <pml-no-stack>
// tag. It stops child columns from stacking on mobile devices by creating a
// ghost table for Outlook and passing a specific context to its children.
type NoStack struct {
	BaseComponent
}

var _ pml_domain.Component = (*NoStack)(nil)

const (
	// defaultGroupWidth is the fallback width for a group when no width is set.
	defaultGroupWidth = "100%"

	// defaultGroupDirection is the default text direction for group elements.
	defaultGroupDirection = "ltr"
)

// NewNoStack creates a new NoStack component instance. A NoStack stops its
// child columns from stacking on mobile devices by using an Outlook ghost
// table.
//
// Returns *NoStack which is the configured component ready for use.
func NewNoStack() *NoStack {
	return &NoStack{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the PML tag name "pml-no-stack".
func (*NoStack) TagName() string {
	return "pml-no-stack"
}

// AllowedParents returns the list of valid parent components for this component.
//
// Returns []string which contains the allowed parent component names.
func (*NoStack) AllowedParents() []string {
	return []string{"pml-row"}
}

// DefaultAttributes returns the default attribute values for this component.
//
// Returns map[string]string which is an empty map as NoStack has no defaults.
func (*NoStack) DefaultAttributes() map[string]string {
	return map[string]string{}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which maps attribute names
// to their type definitions and allowed values.
func (*NoStack) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		CSSWidth: {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		CSSVerticalAlign: {
			Type:          pml_domain.TypeEnum,
			AllowedValues: []string{ValueTop, ValueMiddle, ValueBottom},
		},
		CSSBackgroundColor: {
			Type:          pml_domain.TypeColor,
			AllowedValues: nil,
		},
		CSSDirection: {
			Type:          pml_domain.TypeEnum,
			AllowedValues: []string{ValueLTR, ValueRTL},
		},
	}
}

// GetStyleTargets returns the list of style targets for this component.
//
// Returns []pml_domain.StyleTarget which lists the supported CSS properties
// and their target elements.
func (*NoStack) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: CSSWidth, Target: TargetContainer},
		{Property: CSSVerticalAlign, Target: TargetContainer},
		{Property: CSSBackgroundColor, Target: TargetContainer},
	}
}

// Transform converts the <pml-no-stack> element into its final HTML output.
// The main complexity is generating the Outlook-specific table structure
// within conditional comments that forces columns to remain side-by-side.
//
// Takes node (*ast_domain.TemplateNode) which is the node to transform.
// Takes ctx (*pml_domain.TransformationContext) which provides the
// transformation state and settings.
//
// Returns *ast_domain.TemplateNode which is the transformed HTML structure.
// Returns []*pml_domain.Error which contains any diagnostics from the
// transformation.
func (c *NoStack) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager
	parentContainerWidth := ctx.ContainerWidth
	if parentContainerWidth == 0 {
		parentContainerWidth = WidthMobileBreakpoint
	}

	groupWidthAttr, hasWidth := styles.Get(CSSWidth)
	if !hasWidth || groupWidthAttr == "" {
		groupWidthAttr = defaultGroupWidth
	}

	var groupPixelWidth float64
	if strings.HasSuffix(groupWidthAttr, FormatPercent) {
		var percent float64
		if _, err := fmt.Sscanf(strings.TrimSuffix(groupWidthAttr, FormatPercent), "%f", &percent); err != nil {
			groupPixelWidth = parentContainerWidth
		} else {
			groupPixelWidth = parentContainerWidth * percent / PercentFull
		}
	} else {
		groupPixelWidth = float64(mustParsePixels(groupWidthAttr))
	}

	childCtx := ctx.CloneForChild(nil, nil, node, c)
	childCtx.ContainerWidth = groupPixelWidth
	childCtx.IsInsideGroup = true
	childCtx.SiblingCount = len(node.Children)

	rootNode := c.renderStructure(styles, node.Children, groupWidthAttr, ctx)

	transferPikoDirectives(node, rootNode)
	return rootNode, ctx.Diagnostics()
}

// renderStructure creates the complete group structure.
//
// Takes styles (*pml_domain.StyleManager) which provides the style values.
// Takes children ([]*ast_domain.TemplateNode) which are the child nodes
// to include.
//
// Returns *ast_domain.TemplateNode which is the rendered group div
// element.
func (c *NoStack) renderStructure(styles *pml_domain.StyleManager, children []*ast_domain.TemplateNode, _ string, ctx *pml_domain.TransformationContext) *ast_domain.TemplateNode {
	divStyles := map[string]string{
		CSSFontSize:   ValueZeroPx,
		CSSLineHeight: ValueZero,
		CSSTextAlign:  ValueLeft,
		CSSDisplay:    ValueInlineBlock,
		CSSWidth:      Value100,
		CSSDirection:  getStyleWithDefault(styles, CSSDirection, defaultGroupDirection),
	}
	copyStyle(styles, divStyles, CSSVerticalAlign)
	copyStyle(styles, divStyles, CSSBackgroundColor)

	outlookTableStart := c.renderOutlookGhostTableStart(styles)
	outlookTableEnd := c.renderOutlookGhostTableEnd()

	wrappedChildren := make([]*ast_domain.TemplateNode, 0, len(children)*3+2)
	wrappedChildren = append(wrappedChildren, outlookTableStart)
	for _, child := range children {
		if shouldWrapChild(child) {
			wrappedChildren = append(wrappedChildren, wrapChildWithOutlookTD(child, ctx)...)
		} else {
			wrappedChildren = append(wrappedChildren, child)
		}
	}
	wrappedChildren = append(wrappedChildren, outlookTableEnd)

	return NewElementNode(ElementDiv, []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(divStyles)),
	}, wrappedChildren)
}

// renderOutlookGhostTableStart generates the opening Outlook conditional
// comment with the table and tr start.
//
// Takes styles (*pml_domain.StyleManager) which provides CSS styles to apply
// as table attributes.
//
// Returns *ast_domain.TemplateNode which contains the conditional comment
// wrapped HTML table opening tags.
func (*NoStack) renderOutlookGhostTableStart(styles *pml_domain.StyleManager) *ast_domain.TemplateNode {
	tableAttrs := map[string]string{
		AttrRole:        ValuePresentation,
		AttrBorder:      ValueZero,
		AttrCellPadding: ValueZero,
		AttrCellSpacing: ValueZero,
	}
	if directory := mustGetStyle(styles, CSSDirection); directory != "" {
		tableAttrs[AttrDir] = directory
	}
	if bgColor := mustGetStyle(styles, CSSBackgroundColor); bgColor != "" && bgColor != ValueNone {
		tableAttrs[AttrBgColor] = bgColor
	}

	keys := slices.Sorted(maps.Keys(tableAttrs))

	attrsString := make([]string, 0, len(keys))
	for _, k := range keys {
		attrsString = append(attrsString, fmt.Sprintf("%s=%q", k, tableAttrs[k]))
	}

	tableHTML := fmt.Sprintf(
		"<table %s><tr>",
		strings.Join(attrsString, " "),
	)

	return NewRawHTMLNode(fmt.Sprintf("<!--[if mso | IE]>%s<![endif]-->", tableHTML))
}

// renderOutlookGhostTableEnd creates the closing Outlook conditional comment.
//
// Returns *ast_domain.TemplateNode which contains the closing ghost table HTML.
func (*NoStack) renderOutlookGhostTableEnd() *ast_domain.TemplateNode {
	return NewRawHTMLNode("<!--[if mso | IE]></tr></table><![endif]-->")
}
