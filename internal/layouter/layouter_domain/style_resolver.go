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

package layouter_domain

import (
	"maps"
	"strings"
)

const (
	// defaultLineHeightMultiplier is the factor applied to font
	// size when line-height is "normal", set to 1.4 to match
	// the piko CSS reset.
	defaultLineHeightMultiplier = 1.4

	// maxVarResolutionDepth is the recursion limit for
	// resolving nested var() references.
	maxVarResolutionDepth = 16

	// viewportUnitDivisor converts viewport units to fractions.
	viewportUnitDivisor = 100.0

	// varPrefixLength is the character length of the "var("
	// prefix.
	varPrefixLength = len("var(")

	// cssKeywordTop is the CSS "top" keyword string.
	cssKeywordTop = "top"

	// cssKeywordBottom is the CSS "bottom" keyword string.
	cssKeywordBottom = "bottom"

	// commaDelimiter is the comma separator for CSS value lists.
	commaDelimiter = ","

	// percentSuffix is the percent sign used in CSS percentage values.
	percentSuffix = "%"
)

// ResolutionContext provides the values needed to resolve relative CSS units.
type ResolutionContext struct {
	// ParentFontSize is the computed font size of the parent
	// element in points.
	ParentFontSize float64

	// RootFontSize is the computed font size of the root
	// element in points, used for rem units.
	RootFontSize float64

	// ContainingBlockWidth is the width of the containing
	// block in points, used for percentage resolution.
	ContainingBlockWidth float64

	// ViewportWidth is the viewport width in points, used
	// for vw and vmin/vmax units.
	ViewportWidth float64

	// ViewportHeight is the viewport height in points, used
	// for vh and vmin/vmax units.
	ViewportHeight float64
}

// ResolveStyle converts a flat CSS property map into a fully
// resolved ComputedStyle with inheritance and unit conversion.
//
// Takes properties (map[string]string) which is the CSS
// property map to resolve.
// Takes parentStyle (*ComputedStyle) which is the parent
// element style for inheritance, or nil for the root.
// Takes context (ResolutionContext) which provides the
// values needed for relative unit resolution.
//
// Returns the fully resolved ComputedStyle.
func ResolveStyle(properties map[string]string, parentStyle *ComputedStyle, context ResolutionContext) ComputedStyle {
	style := DefaultComputedStyle()

	if parentStyle != nil {
		inheritFromParent(&style, parentStyle)
	}

	collectCustomProperties(&style, properties)

	if fontSize, ok := properties["font-size"]; ok {
		resolved := resolveVarReferences(fontSize, style.CustomProperties, 0)
		applyProperty(&style, "font-size", strings.TrimSpace(resolved), context, parentStyle)
	}

	for property, value := range properties {
		if strings.HasPrefix(property, "--") || property == "font-size" {
			continue
		}
		resolved := resolveVarReferences(value, style.CustomProperties, 0)
		applyProperty(&style, property, strings.TrimSpace(resolved), context, parentStyle)
	}

	if style.LineHeightAuto {
		style.LineHeight = style.FontSize * defaultLineHeightMultiplier
	}

	return style
}

// collectCustomProperties copies CSS custom properties (--*)
// from the property map into the style.
//
// Takes style (*ComputedStyle) which is the target style
// to populate with custom properties.
// Takes properties (map[string]string) which is the CSS
// property map to scan for custom properties.
func collectCustomProperties(style *ComputedStyle, properties map[string]string) {
	for property, value := range properties {
		if strings.HasPrefix(property, "--") {
			if style.CustomProperties == nil {
				style.CustomProperties = make(map[string]string)
			}
			style.CustomProperties[property] = value
		}
	}
}

// resolveVarReferences recursively replaces var() references
// in a CSS value string with their resolved values.
//
// Takes value (string) which is the CSS value that may
// contain var() references.
// Takes customProperties (map[string]string) which is
// the map of custom property names to values.
// Takes depth (int) which is the current recursion depth
// to guard against infinite loops.
//
// Returns string which is the value with all var()
// references resolved.
func resolveVarReferences(value string, customProperties map[string]string, depth int) string {
	if depth > maxVarResolutionDepth || !strings.Contains(value, "var(") {
		return value
	}

	varStart := strings.Index(value, "var(")
	if varStart == -1 {
		return value
	}

	varEnd := findMatchingCloseParen(value, varStart+varPrefixLength)
	if varEnd == -1 {
		return value
	}

	replacement := resolveVarExpression(
		value[varStart+varPrefixLength:varEnd], customProperties, depth,
	)

	result := value[:varStart] + replacement + value[varEnd+1:]
	return resolveVarReferences(result, customProperties, depth+1)
}

// findMatchingCloseParen returns the index of the closing
// parenthesis that matches the open paren before start.
//
// Takes value (string) which is the string to search.
// Takes start (int) which is the index to begin
// scanning from.
//
// Returns int which is the index of the matching close
// paren, or -1 if not found.
func findMatchingCloseParen(value string, start int) int {
	parenDepth := 0
	for index := start; index < len(value); index++ {
		switch value[index] {
		case '(':
			parenDepth++
		case ')':
			if parenDepth == 0 {
				return index
			}
			parenDepth--
		}
	}
	return -1
}

// resolveVarExpression resolves a single var() expression
// body, handling fallback values when the variable is unset.
//
// Takes inner (string) which is the content between the
// var( and closing paren.
// Takes customProperties (map[string]string) which is
// the map of custom property names to values.
// Takes depth (int) which is the current recursion
// depth for nested resolution.
//
// Returns string which is the resolved value, or empty
// string if the variable is unset with no fallback.
func resolveVarExpression(inner string, customProperties map[string]string, depth int) string {
	inner = strings.TrimSpace(inner)
	variableName, fallback, hasFallback := strings.Cut(inner, commaDelimiter)
	variableName = strings.TrimSpace(variableName)

	if resolved, found := customProperties[variableName]; found {
		return resolveVarReferences(resolved, customProperties, depth+1)
	}
	if hasFallback {
		return resolveVarReferences(strings.TrimSpace(fallback), customProperties, depth+1)
	}
	return ""
}

// inheritFromParent copies inheritable CSS property values
// from the parent style into the child style.
//
// Takes style (*ComputedStyle) which is the child style
// to receive inherited values.
// Takes parent (*ComputedStyle) which is the parent
// style to inherit from.
func inheritFromParent(style *ComputedStyle, parent *ComputedStyle) {
	style.Colour = parent.Colour
	style.FontFamily = parent.FontFamily
	style.FontSize = parent.FontSize
	style.FontStyle = parent.FontStyle
	style.FontWeight = parent.FontWeight
	style.LetterSpacing = parent.LetterSpacing
	style.LineHeight = parent.LineHeight
	style.LineHeightAuto = parent.LineHeightAuto
	style.TextAlign = parent.TextAlign
	style.TextDecoration = parent.TextDecoration
	style.TextIndent = parent.TextIndent
	style.TextTransform = parent.TextTransform
	style.Visibility = parent.Visibility
	style.WhiteSpace = parent.WhiteSpace
	style.WordBreak = parent.WordBreak
	style.OverflowWrap = parent.OverflowWrap
	style.WordSpacing = parent.WordSpacing
	style.ListStyleType = parent.ListStyleType
	style.ListStylePosition = parent.ListStylePosition
	style.Orphans = parent.Orphans
	style.Widows = parent.Widows

	if len(parent.CustomProperties) > 0 {
		style.CustomProperties = make(map[string]string, len(parent.CustomProperties))
		maps.Copy(style.CustomProperties, parent.CustomProperties)
	}
}

// applyProperty sets a single CSS property on the style.
// Global CSS keywords (inherit, initial, unset) are handled
// before the main property dispatch.
//
// Takes style (*ComputedStyle) which is the style to modify.
// Takes property (string) which is the CSS property name.
// Takes value (string) which is the resolved CSS value.
// Takes context (ResolutionContext) which provides unit
// resolution values.
// Takes parentStyle (*ComputedStyle) which is the parent
// style for inherit resolution, or nil for root elements.
//
//nolint:revive // CSS property dispatch
func applyProperty(style *ComputedStyle, property, value string, context ResolutionContext, parentStyle *ComputedStyle) {
	switch value {
	case "inherit":
		if parentStyle != nil {
			copyPropertyFromStyle(style, property, parentStyle)
		}
		return
	case "initial":
		copyPropertyFromStyle(style, property, new(DefaultComputedStyle()))
		return
	case "unset":
		if InheritableProperties[property] {
			if parentStyle != nil {
				copyPropertyFromStyle(style, property, parentStyle)
			}
		} else {
			copyPropertyFromStyle(style, property, new(DefaultComputedStyle()))
		}
		return
	}

	switch property {
	case "display":
		style.Display = parseDisplay(value)
	case "position":
		style.Position = parsePosition(value)
	case "transform":
		style.HasTransform = value != cssKeywordNone && value != ""
		if style.HasTransform {
			style.TransformValue = value
		}
	case "transform-origin":
		style.TransformOrigin = value
	case "box-sizing":
		style.BoxSizing = parseBoxSizing(value)
	case "float":
		style.Float = parseFloat(value)
	case "clear":
		style.Clear = parseClear(value)
	case "visibility":
		style.Visibility = parseVisibility(value)
	case "overflow":
		overflow := parseOverflow(value)
		style.OverflowX = overflow
		style.OverflowY = overflow
	case "overflow-x":
		style.OverflowX = parseOverflow(value)
	case "overflow-y":
		style.OverflowY = parseOverflow(value)
	case "z-index":
		if value == cssKeywordAuto {
			style.ZIndexAuto = true
		} else {
			style.ZIndex = parseIntValue(value)
			style.ZIndexAuto = false
		}

	case "width":
		style.Width = parseDimension(value, context)
	case "height":
		style.Height = parseDimension(value, context)
	case "min-width":
		style.MinWidth = parseDimension(value, context)
	case "min-height":
		style.MinHeight = parseDimension(value, context)
	case "max-width":
		style.MaxWidth = parseDimension(value, context)
	case "max-height":
		style.MaxHeight = parseDimension(value, context)

	case "margin-top":
		style.MarginTop = parseDimension(value, context)
	case "margin-right":
		style.MarginRight = parseDimension(value, context)
	case "margin-bottom":
		style.MarginBottom = parseDimension(value, context)
	case "margin-left":
		style.MarginLeft = parseDimension(value, context)

	case "margin-inline-start":
		dim := parseDimension(value, context)
		if style.Direction == DirectionRTL {
			style.MarginRight = dim
		} else {
			style.MarginLeft = dim
		}
	case "margin-inline-end":
		dim := parseDimension(value, context)
		if style.Direction == DirectionRTL {
			style.MarginLeft = dim
		} else {
			style.MarginRight = dim
		}
	case "margin-inline":
		dim := parseDimension(value, context)
		style.MarginLeft = dim
		style.MarginRight = dim

	case "padding-top":
		style.PaddingTop = resolveLength(value, context)
	case "padding-right":
		style.PaddingRight = resolveLength(value, context)
	case "padding-bottom":
		style.PaddingBottom = resolveLength(value, context)
	case "padding-left":
		style.PaddingLeft = resolveLength(value, context)

	case "padding-inline-start":
		resolved := resolveLength(value, context)
		if style.Direction == DirectionRTL {
			style.PaddingRight = resolved
		} else {
			style.PaddingLeft = resolved
		}
	case "padding-inline-end":
		resolved := resolveLength(value, context)
		if style.Direction == DirectionRTL {
			style.PaddingLeft = resolved
		} else {
			style.PaddingRight = resolved
		}
	case "padding-inline":
		resolved := resolveLength(value, context)
		style.PaddingLeft = resolved
		style.PaddingRight = resolved

	case "border-top-width":
		style.BorderTopWidth = resolveLength(value, context)
	case "border-right-width":
		style.BorderRightWidth = resolveLength(value, context)
	case "border-bottom-width":
		style.BorderBottomWidth = resolveLength(value, context)
	case "border-left-width":
		style.BorderLeftWidth = resolveLength(value, context)

	case "border-inline-start-width":
		resolved := resolveLength(value, context)
		if style.Direction == DirectionRTL {
			style.BorderRightWidth = resolved
		} else {
			style.BorderLeftWidth = resolved
		}
	case "border-inline-end-width":
		resolved := resolveLength(value, context)
		if style.Direction == DirectionRTL {
			style.BorderLeftWidth = resolved
		} else {
			style.BorderRightWidth = resolved
		}

	case "border-top-style":
		style.BorderTopStyle = parseBorderStyle(value)
	case "border-right-style":
		style.BorderRightStyle = parseBorderStyle(value)
	case "border-bottom-style":
		style.BorderBottomStyle = parseBorderStyle(value)
	case "border-left-style":
		style.BorderLeftStyle = parseBorderStyle(value)

	case "border-top-left-radius":
		style.BorderTopLeftRadius = resolveLength(value, context)
	case "border-top-right-radius":
		style.BorderTopRightRadius = resolveLength(value, context)
	case "border-bottom-right-radius":
		style.BorderBottomRightRadius = resolveLength(value, context)
	case "border-bottom-left-radius":
		style.BorderBottomLeftRadius = resolveLength(value, context)

	case "top":
		style.Top = parseDimension(value, context)
	case "right":
		style.Right = parseDimension(value, context)
	case "bottom":
		style.Bottom = parseDimension(value, context)
	case "left":
		style.Left = parseDimension(value, context)

	case "font-family":
		style.FontFamily = parseFontFamily(value)
	case "font-size":
		style.FontSize = resolveFontSize(value, context)
	case "font-weight":
		style.FontWeight = parseFontWeight(value)
	case "font-style":
		style.FontStyle = parseFontStyle(value)
	case "line-height":
		if value == cssKeywordNormal {
			style.LineHeightAuto = true
			style.LineHeight = style.FontSize * defaultLineHeightMultiplier
		} else {
			style.LineHeightAuto = false
			style.LineHeight = resolveLineHeight(value, style.FontSize, context)
		}
	case "text-align":
		style.TextAlign = parseTextAlign(value)
	case "text-decoration", "text-decoration-line":
		style.TextDecoration = parseTextDecoration(value)
	case "text-decoration-color":
		if colour, ok := ParseColour(value); ok {
			style.TextDecorationColour = colour
			style.TextDecorationColourSet = true
		}
	case "text-decoration-style":
		style.TextDecorationStyle = parseTextDecorationStyle(value)
	case "text-transform":
		style.TextTransform = parseTextTransform(value)
	case "letter-spacing":
		if value != cssKeywordNormal {
			style.LetterSpacing = resolveLength(value, context)
		}
	case "word-spacing":
		if value != cssKeywordNormal {
			style.WordSpacing = resolveLength(value, context)
		}
	case "-webkit-text-stroke-width":
		style.TextStrokeWidth = resolveLength(value, context)
		if style.TextStrokeWidth > 0 {
			style.TextRenderingMode = TextRenderFillStroke
		}
	case "-webkit-text-stroke-color":
		if colour, ok := ParseColour(value); ok {
			style.TextStrokeColour = colour
		}
	case "-webkit-text-stroke":
		parts := strings.Fields(value)
		for _, part := range parts {
			if colour, ok := ParseColour(part); ok {
				style.TextStrokeColour = colour
			} else {
				style.TextStrokeWidth = resolveLength(part, context)
			}
		}
		if style.TextStrokeWidth > 0 {
			style.TextRenderingMode = TextRenderFillStroke
		}
	case "white-space":
		style.WhiteSpace = parseWhiteSpace(value)
	case "word-break":
		style.WordBreak = parseWordBreak(value)
	case "overflow-wrap", "word-wrap":
		style.OverflowWrap = parseOverflowWrap(value)
	case "text-indent":
		style.TextIndent = resolveLength(value, context)

	case "color":
		style.Colour = parseColour(value)
	case "background-color":
		style.BackgroundColour = parseColour(value)
	case "border-top-color":
		style.BorderTopColour = parseColour(value)
	case "border-right-color":
		style.BorderRightColour = parseColour(value)
	case "border-bottom-color":
		style.BorderBottomColour = parseColour(value)
	case "border-left-color":
		style.BorderLeftColour = parseColour(value)

	case "flex-direction":
		style.FlexDirection = parseFlexDirection(value)
	case "flex-wrap":
		style.FlexWrap = parseFlexWrap(value)
	case "justify-content":
		style.JustifyContent = parseJustifyContent(value)
	case "align-items":
		style.AlignItems = parseAlignItems(value)
	case "align-self":
		style.AlignSelf = parseAlignSelf(value)
	case "align-content":
		style.AlignContent = parseAlignContent(value)
	case "justify-items":
		style.JustifyItems = parseJustifyItems(value)
	case "justify-self":
		style.JustifySelf = parseJustifySelf(value)
	case "flex-grow":
		style.FlexGrow = parseFloatValue(value)
	case "flex-shrink":
		style.FlexShrink = parseFloatValue(value)
	case "flex-basis":
		style.FlexBasis = parseDimension(value, context)
	case "order":
		style.Order = parseIntValue(value)
	case "gap":
		parts := splitShorthandValues(value)
		if len(parts) == 1 {
			style.RowGap = resolveLength(value, context)
			style.ColumnGap = resolveLength(value, context)
		} else if len(parts) >= 2 {
			style.RowGap = resolveLength(parts[0], context)
			style.ColumnGap = resolveLength(parts[1], context)
		}
	case "row-gap":
		style.RowGap = resolveLength(value, context)
	case "column-gap":
		style.ColumnGap = resolveLength(value, context)

	case "table-layout":
		style.TableLayout = parseTableLayout(value)
	case "border-collapse":
		style.BorderCollapse = parseBorderCollapse(value)
	case "border-spacing":
		style.BorderSpacing = resolveLength(value, context)
	case "caption-side":
		style.CaptionSide = parseCaptionSide(value)
	case "vertical-align":
		style.VerticalAlign = parseVerticalAlign(value)

	case "list-style-type":
		style.ListStyleType = parseListStyleType(value)
	case "list-style-position":
		style.ListStylePosition = parseListStylePosition(value)
	case "list-style":
		parseListStyleShorthand(style, value)

	case "page-break-before":
		style.PageBreakBefore = parsePageBreak(value)
	case "page-break-after":
		style.PageBreakAfter = parsePageBreak(value)
	case "page-break-inside":
		style.PageBreakInside = parsePageBreak(value)
	case "break-before":
		style.PageBreakBefore = parsePageBreak(value)
	case "break-after":
		style.PageBreakAfter = parsePageBreak(value)
	case "break-inside":
		style.PageBreakInside = parsePageBreak(value)
	case "orphans":
		style.Orphans = parseIntValue(value)
	case "widows":
		style.Widows = parseIntValue(value)

	case "opacity":
		style.Opacity = parseFloatValue(value)

	case "box-shadow":
		style.BoxShadow = parseBoxShadow(value, context)

	case "grid-template-columns":
		result := parseGridTrackList(value, context)
		style.GridTemplateColumns = result.tracks
		style.GridAutoRepeatColumns = result.autoRepeat
	case "grid-template-rows":
		result := parseGridTrackList(value, context)
		style.GridTemplateRows = result.tracks
		style.GridAutoRepeatRows = result.autoRepeat
	case "grid-auto-columns":
		style.GridAutoColumns = parseGridTrackList(value, context).tracks
	case "grid-auto-rows":
		style.GridAutoRows = parseGridTrackList(value, context).tracks
	case "grid-column-start":
		style.GridColumnStart = parseGridLine(value)
	case "grid-column-end":
		style.GridColumnEnd = parseGridLine(value)
	case "grid-row-start":
		style.GridRowStart = parseGridLine(value)
	case "grid-row-end":
		style.GridRowEnd = parseGridLine(value)
	case "grid-column":
		start, end := parseGridShorthand(value)
		style.GridColumnStart = start
		style.GridColumnEnd = end
	case "grid-row":
		start, end := parseGridShorthand(value)
		style.GridRowStart = start
		style.GridRowEnd = end
	case "grid-template-areas":
		style.GridTemplateAreas = parseGridTemplateAreas(value)
	case "grid-area":
		parseGridAreaShorthand(style, value)
	case "writing-mode":
		style.WritingMode = parseWritingMode(value)
	case "grid-auto-flow":
		style.GridAutoFlow = parseGridAutoFlow(value)
	case "aspect-ratio":
		style.AspectRatio, style.AspectRatioAuto = parseAspectRatio(value)
	case "text-overflow":
		style.TextOverflow = parseTextOverflow(value)
	case "content":
		style.Content = parseContent(value)
	case "column-count":
		style.ColumnCount = parseColumnCount(value)
	case "column-width":
		style.ColumnWidth = parseDimension(value, context)
	case "columns":
		parseColumnsShorthand(style, value, context)
	case "column-fill":
		style.ColumnFill = parseColumnFill(value)
	case "column-rule-width":
		style.ColumnRuleWidth = resolveLength(value, context)
	case "column-rule-style":
		style.ColumnRuleStyle = parseBorderStyle(value)
	case "column-rule-color":
		style.ColumnRuleColour = parseColour(value)
	case "column-span":
		style.ColumnSpan = parseColumnSpan(value)
	case "mix-blend-mode":
		style.MixBlendMode = ParseBlendMode(value)

	case "text-shadow":
		style.TextShadow = parseTextShadow(value, context)
	case "filter":
		style.Filter = parseFilterList(value)
	case "backdrop-filter":
		style.BackdropFilter = parseFilterList(value)
	case "outline":
		parseOutlineShorthand(style, value, context)
	case "outline-width":
		style.OutlineWidth = resolveLength(value, context)
	case "outline-style":
		style.OutlineStyle = parseBorderStyle(value)
	case "outline-color":
		style.OutlineColour = parseColour(value)
	case "outline-offset":
		style.OutlineOffset = resolveLength(value, context)
	case "background-image":
		style.BgImages = parseBackgroundImages(value, context)
	case "background-size":
		style.BgSize = value
	case "background-position":
		style.BgPosition = value
	case "background-repeat":
		style.BgRepeat = value
	case "background-attachment":
		style.BgAttachment = value
	case "background-origin":
		style.BgOrigin = value
	case "background-clip":
		style.BgClip = value
	case "object-fit":
		style.ObjectFit = parseObjectFit(value)
	case "object-position":
		style.ObjectPosition = value
	case "border-image-source":
		style.BorderImageSource = value
	case "border-image-slice":
		style.BorderImageSlice = parseFloatValue(value)
	case "border-image-width":
		style.BorderImageWidth = resolveLength(value, context)
	case "border-image-outset":
		style.BorderImageOutset = resolveLength(value, context)
	case "border-image-repeat":
		style.BorderImageRepeat = parseBorderImageRepeat(value)
	case "clip-path":
		style.ClipPath = value
	case "mask-image":
		style.MaskImage = value
	case "direction":
		style.Direction = parseDirection(value)
	case "unicode-bidi":
		style.UnicodeBidi = parseUnicodeBidi(value)
	case "hyphens":
		style.Hyphens = parseHyphens(value)
	case "tab-size":
		style.TabSize = parseFloatValue(value)
	case "tab-stops":
		style.TabStops = parseTabStops(value, context)
	case "counter-reset":
		style.CounterReset = parseCounterOperations(value, 0)
	case "counter-increment":
		style.CounterIncrement = parseCounterOperations(value, 1)
	}
}
