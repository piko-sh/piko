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

package layouter_adapters

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/premailer"
)

const (
	// defaultRootFontSize is the fallback root font size in points.
	defaultRootFontSize = 12.0

	// tagTable is the HTML table tag name.
	tagTable = "table"

	// tagTD is the HTML table data cell tag name.
	tagTD = "td"

	// tagTH is the HTML table header cell tag name.
	tagTH = "th"

	// cssPxSuffix is the CSS pixel unit suffix.
	cssPxSuffix = "px"

	// cssBorderStyleOutset is the CSS border style "outset" keyword.
	cssBorderStyleOutset = "outset"

	// cssBorderStyleInset is the CSS border style "inset" keyword.
	cssBorderStyleInset = "inset"

	// cssBorderWidth1px is the CSS border width for table cell borders.
	cssBorderWidth1px = "1px"
)

// CSSResolutionAdapter implements StylesheetPort by delegating CSS parsing,
// selector matching, and cascade resolution to the premailer, then converting
// the resolved property maps into ComputedStyle values.
type CSSResolutionAdapter struct {
	// rootFontSize is the root font size in points used
	// for rem unit resolution.
	rootFontSize float64

	// viewportWidth is the viewport width in points used
	// for vw and vmin/vmax unit resolution.
	viewportWidth float64

	// viewportHeight is the viewport height in points used
	// for vh and vmin/vmax unit resolution.
	viewportHeight float64
}

// NewCSSResolutionAdapter creates a new CSS resolution adapter
// with the given root font size in points.
//
// Takes rootFontSize (float64) which specifies the root font
// size; values <= 0 fall back to defaultRootFontSize.
//
// Returns a pointer to the initialised adapter.
func NewCSSResolutionAdapter(rootFontSize float64) *CSSResolutionAdapter {
	if rootFontSize <= 0 {
		rootFontSize = defaultRootFontSize
	}
	return &CSSResolutionAdapter{rootFontSize: rootFontSize}
}

// SetViewportDimensions configures the viewport dimensions
// used for resolving vw, vh, vmin, and vmax units.
//
// Takes width (float64) which specifies the viewport width
// in points.
//
// Takes height (float64) which specifies the viewport height
// in points.
func (a *CSSResolutionAdapter) SetViewportDimensions(width, height float64) {
	a.viewportWidth = width
	a.viewportHeight = height
}

// ResolveStyles resolves CSS styles for every node in the AST by delegating
// CSS parsing and cascade resolution to the premailer, then converting the
// resolved property maps into ComputedStyle values with inheritance.
//
// Takes tree (*ast_domain.TemplateAST) which is the parsed template AST to
// resolve styles for.
// Takes styling (string) which is the raw CSS text from style elements.
// Takes additionalStylesheets ([]string) which provides extra stylesheet
// sources to apply.
//
// Returns the computed StyleMap, PseudoStyleMap, and nil error on success.
func (a *CSSResolutionAdapter) ResolveStyles(
	ctx context.Context,
	tree *ast_domain.TemplateAST,
	styling string,
	additionalStylesheets []string,
) (layouter_domain.StyleMap, layouter_domain.PseudoStyleMap, error) {
	combinedCSS := a.buildCombinedCSS(styling, additionalStylesheets)

	pmOpts := []premailer.Option{
		premailer.WithResolvePseudoElements(true),
		premailer.WithSkipEmailValidation(true),
		premailer.WithSkipHTMLAttributeMapping(true),
		premailer.WithSkipStyleExtraction(true),
		premailer.WithExpandShorthands(true),
		premailer.WithExternalCSS(combinedCSS),
	}

	pm := premailer.New(tree, pmOpts...)
	resolved, err := pm.ResolveProperties()
	if err != nil {
		return nil, nil, fmt.Errorf("resolving CSS properties: %w", err)
	}

	tree.Diagnostics = append(tree.Diagnostics, resolved.Diagnostics...)

	styleMap := make(layouter_domain.StyleMap)
	pseudoStyleMap := make(layouter_domain.PseudoStyleMap)
	parentMap := make(map[*ast_domain.TemplateNode]*ast_domain.TemplateNode)

	for _, rootNode := range tree.RootNodes {
		a.resolveSubtree(ctx, rootNode, nil, resolved, styleMap, pseudoStyleMap, parentMap)
	}

	return styleMap, pseudoStyleMap, nil
}

// buildCombinedCSS concatenates the user-agent stylesheet, additional
// stylesheets, and component styling into a single CSS string.
//
// Takes styling (string) which is the component's own CSS.
// Takes additionalStylesheets ([]string) which provides extra CSS sources.
//
// Returns string which contains the combined CSS.
func (*CSSResolutionAdapter) buildCombinedCSS(styling string, additionalStylesheets []string) string {
	var builder strings.Builder
	builder.WriteString(layouter_domain.UserAgentStylesheet)
	for _, sheet := range additionalStylesheets {
		builder.WriteByte('\n')
		builder.WriteString(sheet)
	}
	if styling != "" {
		builder.WriteByte('\n')
		builder.WriteString(styling)
	}
	return builder.String()
}

// resolveSubtree recursively resolves styles for a node and its children,
// applying presentational attributes, premailer-resolved CSS properties,
// and inheritance.
//
// Takes node (*ast_domain.TemplateNode) which is the current node.
// Takes parentStyle (*layouter_domain.ComputedStyle) which is the parent's
// computed style, or nil for root nodes.
// Takes resolved (*premailer.ResolvedProperties) which holds the CSS
// properties from the premailer.
// Takes styleMap (layouter_domain.StyleMap) which accumulates resolved styles.
// Takes pseudoStyleMap (layouter_domain.PseudoStyleMap) which accumulates
// pseudo-element styles.
// Takes parentMap (map) which maps each node to its parent.
func (a *CSSResolutionAdapter) resolveSubtree(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentStyle *layouter_domain.ComputedStyle,
	resolved *premailer.ResolvedProperties,
	styleMap layouter_domain.StyleMap,
	pseudoStyleMap layouter_domain.PseudoStyleMap,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
) {
	if ctx.Err() != nil {
		return
	}

	if node == nil {
		return
	}

	if node.NodeType == ast_domain.NodeText || node.NodeType == ast_domain.NodeComment {
		return
	}

	parentFontSize := a.rootFontSize
	if parentStyle != nil {
		parentFontSize = parentStyle.FontSize
	}

	containingBlockWidth := a.viewportWidth
	if parentStyle != nil && !parentStyle.Width.IsAuto() {
		containingBlockWidth = parentStyle.Width.Resolve(a.viewportWidth, 0)
	}

	resolutionContext := layouter_domain.ResolutionContext{
		ParentFontSize:       parentFontSize,
		RootFontSize:         a.rootFontSize,
		ContainingBlockWidth: containingBlockWidth,
		ViewportWidth:        a.viewportWidth,
		ViewportHeight:       a.viewportHeight,
	}

	properties := buildNodeProperties(node, resolved.Elements[node], parentMap)
	style := layouter_domain.ResolveStyle(properties, parentStyle, resolutionContext)
	styleMap[node] = &style

	a.resolvePseudoElements(node, &style, resolved, pseudoStyleMap, resolutionContext)

	for _, child := range node.Children {
		parentMap[child] = node
		a.resolveSubtree(ctx, child, &style, resolved, styleMap, pseudoStyleMap, parentMap)
	}
}

// buildNodeProperties merges presentational HTML attributes with CSS
// properties from the premailer. Presentational attributes have the lowest
// priority: any CSS property overrides them.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes cssProps (map[string]string) which contains the premailer-resolved
// CSS properties, or nil if the node had no matching rules.
// Takes parentMap (map) which maps each node to its parent for ancestor
// lookups.
//
// Returns map[string]string which contains the merged property map.
func buildNodeProperties(
	node *ast_domain.TemplateNode,
	cssProps map[string]string,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
) map[string]string {
	presAttrs := make(map[string]string)
	mapPresentationalAttributes(node, presAttrs, parentMap)

	if len(presAttrs) == 0 && cssProps == nil {
		return nil
	}

	if len(presAttrs) == 0 {
		return cssProps
	}

	if cssProps == nil {
		return presAttrs
	}

	merged := make(map[string]string, len(presAttrs)+len(cssProps))
	maps.Copy(merged, presAttrs)
	maps.Copy(merged, cssProps)
	return merged
}

// resolvePseudoElements builds ComputedStyle values for ::before and ::after
// pseudo-elements from the premailer-resolved pseudo-element properties.
//
// Takes node (*ast_domain.TemplateNode) which is the element.
// Takes elementStyle (*layouter_domain.ComputedStyle) which is the element's
// own resolved style for inheritance.
// Takes resolved (*premailer.ResolvedProperties) which holds the resolved
// pseudo-element properties.
// Takes pseudoStyleMap (layouter_domain.PseudoStyleMap) which accumulates
// pseudo-element styles.
// Takes resolutionContext (layouter_domain.ResolutionContext) which provides
// unit resolution values.
func (*CSSResolutionAdapter) resolvePseudoElements(
	node *ast_domain.TemplateNode,
	elementStyle *layouter_domain.ComputedStyle,
	resolved *premailer.ResolvedProperties,
	pseudoStyleMap layouter_domain.PseudoStyleMap,
	resolutionContext layouter_domain.ResolutionContext,
) {
	pseudos := resolved.PseudoElements[node]
	if len(pseudos) == 0 {
		return
	}

	nodeMap := make(map[layouter_domain.PseudoType]*layouter_domain.ComputedStyle)

	for pseudoName, props := range pseudos {
		pseudoType := parsePseudoType(pseudoName)
		if pseudoType == layouter_domain.PseudoNone {
			continue
		}
		nodeMap[pseudoType] = new(layouter_domain.ResolveStyle(props, elementStyle, resolutionContext))
	}

	if len(nodeMap) > 0 {
		pseudoStyleMap[node] = nodeMap
	}
}

// parsePseudoType converts a pseudo-element name string to the domain
// PseudoType enum.
//
// Takes name (string) which is the pseudo-element name ("before" or "after").
//
// Returns layouter_domain.PseudoType which is the corresponding enum value,
// or PseudoNone for unrecognised names.
func parsePseudoType(name string) layouter_domain.PseudoType {
	switch name {
	case "before":
		return layouter_domain.PseudoBefore
	case "after":
		return layouter_domain.PseudoAfter
	default:
		return layouter_domain.PseudoNone
	}
}

// getAttributeValue returns the value of the named attribute on the node,
// or an empty string if not found.
//
// Takes node (*ast_domain.TemplateNode) which is the node to inspect.
// Takes name (string) which is the attribute name to look up.
//
// Returns string which is the attribute value, or empty if the attribute
// is not present.
func getAttributeValue(node *ast_domain.TemplateNode, name string) string {
	for index := range node.Attributes {
		if node.Attributes[index].Name == name {
			return node.Attributes[index].Value
		}
	}
	return ""
}

// mapPresentationalAttributes maps HTML presentational attributes to their
// CSS equivalents. These are applied before CSS rules so that any CSS rule
// (including the user-agent stylesheet) overrides them.
//
// Takes node (*ast_domain.TemplateNode) which is the element node to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes parentMap (map) which maps each node to its parent for ancestor
// lookups.
func mapPresentationalAttributes(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
) {
	tagName := strings.ToLower(node.TagName)

	mapDimensionAttributes(node, properties, tagName)
	mapAlignAttribute(node, properties, tagName)
	mapValignAttribute(node, properties, tagName)
	mapTableBorderAttribute(node, properties, parentMap, tagName)
	mapBgcolourAttribute(node, properties, tagName)
	mapCellpaddingAttribute(node, properties, parentMap, tagName)
	mapCellspacingAttribute(node, properties, tagName)
	mapFontElementAttributes(node, properties, tagName)
}

// mapDimensionAttributes maps HTML width and height attributes to CSS
// width and height properties for elements that support them.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapDimensionAttributes(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	switch tagName {
	case "img", tagTable, tagTD, tagTH, "canvas", "video":
	default:
		return
	}

	if widthValue := getAttributeValue(node, "width"); widthValue != "" {
		properties["width"] = normaliseDimensionValue(widthValue)
	}
	if heightValue := getAttributeValue(node, "height"); heightValue != "" {
		properties["height"] = normaliseDimensionValue(heightValue)
	}
}

// normaliseDimensionValue converts an HTML dimension attribute value to a
// CSS length. Bare numbers get a "px" suffix; percentage values are returned
// as-is.
//
// Takes value (string) which is the raw HTML dimension attribute value.
//
// Returns string which is the normalised CSS length value.
func normaliseDimensionValue(value string) string {
	if strings.HasSuffix(value, "%") {
		return value
	}
	if _, parseError := strconv.ParseFloat(value, 64); parseError == nil {
		return value + cssPxSuffix
	}
	return value
}

// mapAlignAttribute maps the HTML align attribute to the appropriate CSS
// property depending on the element type.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapAlignAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	alignValue := strings.ToLower(getAttributeValue(node, "align"))
	if alignValue == "" {
		return
	}

	switch tagName {
	case tagTable:
		switch alignValue {
		case "center":
			properties["margin-left"] = "auto"
			properties["margin-right"] = "auto"
		case "left":
			properties["float"] = "left"
		case "right":
			properties["float"] = "right"
		}
	case tagTD, tagTH, "tr",
		"div", "p",
		"h1", "h2", "h3", "h4", "h5", "h6":
		properties["text-align"] = alignValue
	}
}

// mapValignAttribute maps the HTML valign attribute to the CSS vertical-align
// property for table-related elements.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapValignAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	valignValue := strings.ToLower(getAttributeValue(node, "valign"))
	if valignValue == "" {
		return
	}

	switch tagName {
	case tagTD, tagTH, "tr", "tbody", "thead", "tfoot":
		properties["vertical-align"] = valignValue
	}
}

// mapTableBorderAttribute maps the HTML border attribute on table elements
// to CSS border properties. For td/th cells, it walks up the parent chain
// to find the ancestor table's border attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes parentMap (map) which maps each node to its parent.
// Takes tagName (string) which is the lowercased tag name.
func mapTableBorderAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
	tagName string,
) {
	switch tagName {
	case tagTable:
		borderValue := getAttributeValue(node, "border")
		if borderValue == "" {
			return
		}
		borderWidth, parseError := strconv.Atoi(borderValue)
		if parseError != nil || borderWidth <= 0 {
			return
		}
		widthString := borderValue + cssPxSuffix
		properties["border-top-width"] = widthString
		properties["border-right-width"] = widthString
		properties["border-bottom-width"] = widthString
		properties["border-left-width"] = widthString
		properties["border-top-style"] = cssBorderStyleOutset
		properties["border-right-style"] = cssBorderStyleOutset
		properties["border-bottom-style"] = cssBorderStyleOutset
		properties["border-left-style"] = cssBorderStyleOutset

	case tagTD, tagTH:
		tableNode := findAncestorByTag(node, tagTable, parentMap)
		if tableNode == nil {
			return
		}
		borderValue := getAttributeValue(tableNode, "border")
		if borderValue == "" {
			return
		}
		borderWidth, parseError := strconv.Atoi(borderValue)
		if parseError != nil || borderWidth < 1 {
			return
		}
		properties["border-top-width"] = cssBorderWidth1px
		properties["border-right-width"] = cssBorderWidth1px
		properties["border-bottom-width"] = cssBorderWidth1px
		properties["border-left-width"] = cssBorderWidth1px
		properties["border-top-style"] = cssBorderStyleInset
		properties["border-right-style"] = cssBorderStyleInset
		properties["border-bottom-style"] = cssBorderStyleInset
		properties["border-left-style"] = cssBorderStyleInset
	}
}

// mapBgcolourAttribute maps the HTML bgcolor attribute to the CSS
// background-color property.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapBgcolourAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	bgcolourValue := getAttributeValue(node, "bgcolor")
	if bgcolourValue == "" {
		return
	}

	switch tagName {
	case "body", tagTable, tagTD, tagTH, "tr":
		properties["background-color"] = bgcolourValue
	}
}

// mapCellpaddingAttribute maps the HTML cellpadding attribute from an
// ancestor table to CSS padding on td/th cells.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes parentMap (map) which maps each node to its parent.
// Takes tagName (string) which is the lowercased tag name.
func mapCellpaddingAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
	tagName string,
) {
	if tagName != tagTD && tagName != tagTH {
		return
	}

	tableNode := findAncestorByTag(node, tagTable, parentMap)
	if tableNode == nil {
		return
	}

	cellpaddingValue := getAttributeValue(tableNode, "cellpadding")
	if cellpaddingValue == "" {
		return
	}

	if _, parseError := strconv.ParseFloat(cellpaddingValue, 64); parseError != nil {
		return
	}

	paddingCSS := cellpaddingValue + cssPxSuffix
	properties["padding-top"] = paddingCSS
	properties["padding-right"] = paddingCSS
	properties["padding-bottom"] = paddingCSS
	properties["padding-left"] = paddingCSS
}

// mapCellspacingAttribute maps the HTML cellspacing attribute to the CSS
// border-spacing property.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapCellspacingAttribute(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	if tagName != tagTable {
		return
	}

	cellspacingValue := getAttributeValue(node, "cellspacing")
	if cellspacingValue == "" {
		return
	}

	if _, parseError := strconv.ParseFloat(cellspacingValue, 64); parseError != nil {
		return
	}

	properties["border-spacing"] = cellspacingValue + cssPxSuffix
}

// mapFontElementAttributes maps the HTML color and size attributes on the
// legacy font element to CSS properties.
//
// Takes node (*ast_domain.TemplateNode) which is the element to inspect.
// Takes properties (map[string]string) which is the property map to populate.
// Takes tagName (string) which is the lowercased tag name.
func mapFontElementAttributes(
	node *ast_domain.TemplateNode,
	properties map[string]string,
	tagName string,
) {
	if tagName != "font" {
		return
	}

	if colourValue := getAttributeValue(node, "color"); colourValue != "" {
		properties["color"] = colourValue
	}

	if sizeValue := getAttributeValue(node, "size"); sizeValue != "" {
		if cssSize := mapFontSizeToCSS(sizeValue); cssSize != "" {
			properties["font-size"] = cssSize
		}
	}
}

// mapFontSizeToCSS converts an HTML font size attribute value (1-7) to a
// CSS absolute font size keyword.
//
// Takes size (string) which is the HTML font size value.
//
// Returns string which is the CSS absolute font size keyword, or empty
// for unrecognised values.
func mapFontSizeToCSS(size string) string {
	switch size {
	case "1":
		return "x-small"
	case "2":
		return "small"
	case "3":
		return "medium"
	case "4":
		return "large"
	case "5":
		return "x-large"
	case "6":
		return "xx-large"
	case "7":
		return "xxx-large"
	default:
		return ""
	}
}

// findAncestorByTag walks the parent chain to find the nearest ancestor
// element with the given tag name.
//
// Takes node (*ast_domain.TemplateNode) which is the starting node.
// Takes tagName (string) which is the tag name to search for.
// Takes parentMap (map) which maps each node to its parent.
//
// Returns *ast_domain.TemplateNode which is the matched ancestor, or nil
// if no matching ancestor is found.
func findAncestorByTag(
	node *ast_domain.TemplateNode,
	tagName string,
	parentMap map[*ast_domain.TemplateNode]*ast_domain.TemplateNode,
) *ast_domain.TemplateNode {
	current := parentMap[node]
	for current != nil {
		if strings.EqualFold(current.TagName, tagName) {
			return current
		}
		current = parentMap[current]
	}
	return nil
}
