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

// ThematicBreak implements the Component interface for the <pml-hr> tag.
// It renders a horizontal line separator with Outlook email client support.
type ThematicBreak struct {
	BaseComponent
}

var _ pml_domain.Component = (*ThematicBreak)(nil)

const (
	// defaultBorderColor is the default border colour for thematic breaks.
	defaultBorderColor = "#000000"

	// defaultBorderStyle is the default CSS border style for thematic breaks.
	defaultBorderStyle = "solid"

	// defaultBorderWidth is the default border width for thematic breaks.
	defaultBorderWidth = "4px"

	// defaultHRPadding is the default CSS padding for horizontal rules.
	defaultHRPadding = "10px 25px"

	// defaultHRAlign is the default alignment for thematic breaks.
	defaultHRAlign = "center"
)

// NewThematicBreak creates a new ThematicBreak component instance.
// A ThematicBreak renders a horizontal line separator with styling control.
//
// Returns *ThematicBreak which is a configured component ready for use.
func NewThematicBreak() *ThematicBreak {
	return &ThematicBreak{
		BaseComponent: BaseComponent{},
	}
}

// TagName returns the tag name for this component.
//
// Returns string which is the HTML tag name "pml-hr".
func (*ThematicBreak) TagName() string {
	return "pml-hr"
}

// IsEndingTag returns true because ThematicBreak is a void element and cannot
// have children.
//
// Returns bool which is always true for this element type.
func (*ThematicBreak) IsEndingTag() bool {
	return true
}

// AllowedParents returns the list of valid parent components for this element.
//
// Returns []string which contains the allowed parent element names.
func (*ThematicBreak) AllowedParents() []string {
	return []string{"pml-col", "pml-hero"}
}

// AllowedAttributes returns the map of valid attributes for this component.
//
// Returns map[string]pml_domain.AttributeDefinition which contains the
// attribute definitions keyed by attribute name.
func (*ThematicBreak) AllowedAttributes() map[string]pml_domain.AttributeDefinition {
	return map[string]pml_domain.AttributeDefinition{
		AttrAlign: {
			Type:          pml_domain.TypeEnum,
			AllowedValues: []string{"left", ValueCentre, "right"},
		},
		CSSBorderColor: {
			Type:          pml_domain.TypeColor,
			AllowedValues: nil,
		},
		CSSBorderStyle: {
			Type:          pml_domain.TypeEnum,
			AllowedValues: []string{"dashed", "dotted", "solid"},
		},
		CSSBorderWidth: {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		AttrContainerBackgroundColor: {
			Type:          pml_domain.TypeColor,
			AllowedValues: nil,
		},
		AttrPadding: {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		"padding-top": {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		"padding-bottom": {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		"padding-left": {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		"padding-right": {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
		AttrWidth: {
			Type:          pml_domain.TypeUnit,
			AllowedValues: nil,
		},
	}
}

// GetStyleTargets returns the style targets for this component.
//
// Returns []pml_domain.StyleTarget which links CSS properties to their targets.
func (*ThematicBreak) GetStyleTargets() []pml_domain.StyleTarget {
	return []pml_domain.StyleTarget{
		{Property: CSSBorderColor, Target: TargetContainer},
		{Property: CSSBorderStyle, Target: TargetContainer},
		{Property: CSSBorderWidth, Target: TargetContainer},
		{Property: CSSWidth, Target: TargetContainer},
		{Property: CSSPadding, Target: TargetContainer},
		{Property: CSSAlign, Target: TargetContainer},
	}
}

// DefaultAttributes returns the default attribute values for pml-hr.
//
// Returns map[string]string which contains the CSS property defaults.
func (*ThematicBreak) DefaultAttributes() map[string]string {
	return map[string]string{
		CSSBorderColor: defaultBorderColor,
		CSSBorderStyle: defaultBorderStyle,
		CSSBorderWidth: defaultBorderWidth,
		CSSPadding:     defaultHRPadding,
		CSSWidth:       Value100,
		CSSAlign:       defaultHRAlign,
	}
}

// Transform converts the <pml-hr> node into its final HTML structure for use
// in emails. The output is more complex than a simple <hr> tag to ensure that
// padding and alignment work well across all email clients, including Outlook.
//
// Takes node (*ast_domain.TemplateNode) which is the <pml-hr> node to convert.
// Takes ctx (*pml_domain.TransformationContext) which provides styles and
// diagnostics.
//
// Returns *ast_domain.TemplateNode which contains the HTML structure for email
// clients.
// Returns []*pml_domain.Error which contains any issues found during the
// conversion.
func (*ThematicBreak) Transform(node *ast_domain.TemplateNode, ctx *pml_domain.TransformationContext) (*ast_domain.TemplateNode, []*pml_domain.Error) {
	styles := ctx.StyleManager
	pStyles := buildHRParagraphStyles(styles)

	pAttrs := []ast_domain.HTMLAttribute{
		NewHTMLAttribute(AttrStyle, mapToStyleString(pStyles)),
		NewHTMLAttribute("data-pml-padding", mustGetStyle(styles, CSSPadding)),
	}

	if styles.IsExplicit(CSSAlign) {
		pAttrs = append(pAttrs, NewHTMLAttribute("data-pml-align", mustGetStyle(styles, CSSAlign)))
	}

	pNode := NewElementNode("p", pAttrs, nil)

	transferPikoDirectives(node, pNode)

	outlookTable := renderOutlookHRTable(styles, ctx)
	containerFragment := NewFragmentNode([]*ast_domain.TemplateNode{pNode, outlookTable})

	return containerFragment, ctx.Diagnostics()
}

// buildHRParagraphStyles creates the style map for a horizontal rule paragraph.
//
// Takes styles (*pml_domain.StyleManager) which holds the style settings for
// border, width, and alignment.
//
// Returns map[string]string which contains the CSS properties for the
// horizontal rule paragraph.
func buildHRParagraphStyles(styles *pml_domain.StyleManager) map[string]string {
	pStyles := map[string]string{
		CSSFontSize: "1px",
		"margin":    ValueMarginAuto,
	}

	borderStyle := mustGetStyle(styles, CSSBorderStyle)
	borderWidth := mustGetStyle(styles, CSSBorderWidth)
	borderColor := mustGetStyle(styles, CSSBorderColor)
	pStyles["border-top"] = fmt.Sprintf("%s %s %s", borderStyle, borderWidth, borderColor)

	width := mustGetStyle(styles, CSSWidth)
	pStyles[CSSWidth] = width

	align := mustGetStyle(styles, CSSAlign)
	applyHRAlignment(pStyles, align)

	return pStyles
}

// applyHRAlignment sets the margin style based on the given alignment.
//
// Takes pStyles (map[string]string) which holds the paragraph style properties
// to modify.
// Takes align (string) which sets the alignment: "left", "right", or "centre"
// (the default).
func applyHRAlignment(pStyles map[string]string, align string) {
	switch align {
	case "left":
		pStyles["margin"] = ValueZero
	case "right":
		pStyles["margin"] = ValueZero + ValueSpace + ValueZero + ValueSpace + ValueZero + ValueSpace + ValueAuto
	}
}

// renderOutlookHRTable creates a table that shows the horizontal rule in
// Outlook email clients.
//
// Takes styles (*pml_domain.StyleManager) which provides the CSS styles for
// the horizontal rule.
// Takes ctx (*pml_domain.TransformationContext) which provides the context
// for width calculations.
//
// Returns *ast_domain.TemplateNode which contains the raw HTML wrapped in
// Outlook conditional comments.
func renderOutlookHRTable(styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) *ast_domain.TemplateNode {
	widthString := mustGetStyle(styles, CSSWidth)
	alignString := mustGetStyle(styles, CSSAlign)
	borderStyle := mustGetStyle(styles, CSSBorderStyle)
	borderWidth := mustGetStyle(styles, CSSBorderWidth)
	borderColor := mustGetStyle(styles, CSSBorderColor)

	outlookWidthPx := getOutlookHRWidth(widthString, styles, ctx)

	tableHTML := buildOutlookHRTableHTML(alignString, borderStyle, borderWidth, borderColor, outlookWidthPx)
	return NewRawHTMLNode(fmt.Sprintf("<!--[if mso | IE]>%s<![endif]-->", tableHTML))
}

// buildOutlookHRTableHTML builds the HTML table string for a horizontal rule
// in Outlook email clients.
//
// Takes align (string) which sets the table alignment.
// Takes borderStyle (string) which sets the border style (e.g. "solid").
// Takes borderWidth (string) which sets the border width (e.g. "1px").
// Takes borderColor (string) which sets the border colour.
// Takes widthPx (int) which sets the table width in pixels.
//
// Returns string which is the formatted HTML table for Outlook clients.
func buildOutlookHRTableHTML(align, borderStyle, borderWidth, borderColor string, widthPx int) string {
	return fmt.Sprintf(
		`<table align=%q border="0" cellpadding="0" cellspacing="0" `+
			`style="border-top:%s %s %s;font-size:1px;margin:0px auto;width:%dpx;" `+
			`role="presentation" width="%dpx">`+
			`<tr><td style="height:0;line-height:0;">&nbsp;</td></tr></table>`,
		align,
		borderStyle,
		borderWidth,
		borderColor,
		widthPx,
		widthPx,
	)
}

// getOutlookHRWidth works out the pixel width for an Outlook horizontal rule
// table, adjusting for any padding that has been set.
//
// Takes width (string) which is the desired width in pixels or as a percent.
// Takes styles (*pml_domain.StyleManager) which gives access to CSS styles.
// Takes ctx (*pml_domain.TransformationContext) which holds the container width.
//
// Returns int which is the final width in pixels after padding is removed.
func getOutlookHRWidth(width string, styles *pml_domain.StyleManager, ctx *pml_domain.TransformationContext) int {
	paddingLeft := 0
	paddingRight := 0

	if padding := mustGetStyle(styles, CSSPadding); padding != "" {
		parts := strings.Fields(padding)
		switch len(parts) {
		case paddingPartsAll:
			paddingRight = mustParsePixels(parts[0])
			paddingLeft = mustParsePixels(parts[0])
		case paddingPartsVertHoriz:
			paddingRight = mustParsePixels(parts[1])
			paddingLeft = mustParsePixels(parts[1])
		case paddingPartsFull:
			paddingRight = mustParsePixels(parts[1])
			paddingLeft = mustParsePixels(parts[3])
		}
	}

	if pl := mustGetStyle(styles, "padding-left"); pl != "" {
		paddingLeft = mustParsePixels(pl)
	}
	if pr := mustGetStyle(styles, "padding-right"); pr != "" {
		paddingRight = mustParsePixels(pr)
	}

	totalPadding := paddingLeft + paddingRight

	containerWidth := ctx.ContainerWidth
	if containerWidth == 0 {
		containerWidth = WidthMobileBreakpoint
	}

	if strings.HasSuffix(width, "px") {
		return mustParsePixels(width)
	}

	if strings.HasSuffix(width, FormatPercent) {
		effectiveWidth := int(containerWidth) - totalPadding
		percentString := strings.TrimSuffix(width, FormatPercent)
		var percent float64
		if _, err := fmt.Sscanf(percentString, "%f", &percent); err != nil {
			return effectiveWidth
		}
		return int(float64(effectiveWidth) * percent / PercentFull)
	}

	return int(containerWidth) - totalPadding
}
