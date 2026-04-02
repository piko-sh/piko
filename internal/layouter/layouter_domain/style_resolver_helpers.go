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
	"math"
	"strconv"
	"strings"
)

// copyPropertyFromStyle copies a single named CSS property
// from src to dst, used to implement inherit, initial, and
// unset keywords.
//
// Takes dst (*ComputedStyle) which is the target style.
// Takes property (string) which is the CSS property name.
// Takes src (*ComputedStyle) which is the source style.
//
//nolint:revive // mirrors applyProperty
func copyPropertyFromStyle(dst *ComputedStyle, property string, src *ComputedStyle) {
	switch property {
	case "display":
		dst.Display = src.Display
	case "position":
		dst.Position = src.Position
	case "transform":
		dst.HasTransform = src.HasTransform
		dst.TransformValue = src.TransformValue
	case "transform-origin":
		dst.TransformOrigin = src.TransformOrigin
	case "box-sizing":
		dst.BoxSizing = src.BoxSizing
	case "float":
		dst.Float = src.Float
	case "clear":
		dst.Clear = src.Clear
	case "visibility":
		dst.Visibility = src.Visibility
	case "overflow-x":
		dst.OverflowX = src.OverflowX
	case "overflow-y":
		dst.OverflowY = src.OverflowY
	case "z-index":
		dst.ZIndex = src.ZIndex
		dst.ZIndexAuto = src.ZIndexAuto
	case "width":
		dst.Width = src.Width
	case "height":
		dst.Height = src.Height
	case "min-width":
		dst.MinWidth = src.MinWidth
	case "min-height":
		dst.MinHeight = src.MinHeight
	case "max-width":
		dst.MaxWidth = src.MaxWidth
	case "max-height":
		dst.MaxHeight = src.MaxHeight
	case "margin-top":
		dst.MarginTop = src.MarginTop
	case "margin-right":
		dst.MarginRight = src.MarginRight
	case "margin-bottom":
		dst.MarginBottom = src.MarginBottom
	case "margin-left":
		dst.MarginLeft = src.MarginLeft
	case "margin-inline-start", "margin-inline-end", "margin-inline":
		dst.MarginLeft = src.MarginLeft
		dst.MarginRight = src.MarginRight
	case "padding-top":
		dst.PaddingTop = src.PaddingTop
	case "padding-right":
		dst.PaddingRight = src.PaddingRight
	case "padding-bottom":
		dst.PaddingBottom = src.PaddingBottom
	case "padding-left":
		dst.PaddingLeft = src.PaddingLeft
	case "padding-inline-start", "padding-inline-end", "padding-inline":
		dst.PaddingLeft = src.PaddingLeft
		dst.PaddingRight = src.PaddingRight
	case "border-top-width":
		dst.BorderTopWidth = src.BorderTopWidth
	case "border-right-width":
		dst.BorderRightWidth = src.BorderRightWidth
	case "border-bottom-width":
		dst.BorderBottomWidth = src.BorderBottomWidth
	case "border-left-width":
		dst.BorderLeftWidth = src.BorderLeftWidth
	case "border-inline-start-width", "border-inline-end-width":
		dst.BorderLeftWidth = src.BorderLeftWidth
		dst.BorderRightWidth = src.BorderRightWidth
	case "border-top-style":
		dst.BorderTopStyle = src.BorderTopStyle
	case "border-right-style":
		dst.BorderRightStyle = src.BorderRightStyle
	case "border-bottom-style":
		dst.BorderBottomStyle = src.BorderBottomStyle
	case "border-left-style":
		dst.BorderLeftStyle = src.BorderLeftStyle
	case "border-top-left-radius":
		dst.BorderTopLeftRadius = src.BorderTopLeftRadius
	case "border-top-right-radius":
		dst.BorderTopRightRadius = src.BorderTopRightRadius
	case "border-bottom-right-radius":
		dst.BorderBottomRightRadius = src.BorderBottomRightRadius
	case "border-bottom-left-radius":
		dst.BorderBottomLeftRadius = src.BorderBottomLeftRadius
	case "top":
		dst.Top = src.Top
	case "right":
		dst.Right = src.Right
	case "bottom":
		dst.Bottom = src.Bottom
	case "left":
		dst.Left = src.Left
	case "font-family":
		dst.FontFamily = src.FontFamily
	case "font-size":
		dst.FontSize = src.FontSize
	case "font-weight":
		dst.FontWeight = src.FontWeight
	case "font-style":
		dst.FontStyle = src.FontStyle
	case "line-height":
		dst.LineHeight = src.LineHeight
		dst.LineHeightAuto = src.LineHeightAuto
	case "text-align":
		dst.TextAlign = src.TextAlign
	case "text-decoration", "text-decoration-line":
		dst.TextDecoration = src.TextDecoration
	case "text-decoration-color":
		dst.TextDecorationColour = src.TextDecorationColour
		dst.TextDecorationColourSet = src.TextDecorationColourSet
	case "text-decoration-style":
		dst.TextDecorationStyle = src.TextDecorationStyle
	case "text-transform":
		dst.TextTransform = src.TextTransform
	case "letter-spacing":
		dst.LetterSpacing = src.LetterSpacing
	case "word-spacing":
		dst.WordSpacing = src.WordSpacing
	case "-webkit-text-stroke-width":
		dst.TextStrokeWidth = src.TextStrokeWidth
		if dst.TextStrokeWidth > 0 {
			dst.TextRenderingMode = TextRenderFillStroke
		}
	case "-webkit-text-stroke-color":
		dst.TextStrokeColour = src.TextStrokeColour
	case "-webkit-text-stroke":
		dst.TextStrokeWidth = src.TextStrokeWidth
		dst.TextStrokeColour = src.TextStrokeColour
		if dst.TextStrokeWidth > 0 {
			dst.TextRenderingMode = TextRenderFillStroke
		}
	case "white-space":
		dst.WhiteSpace = src.WhiteSpace
	case "word-break":
		dst.WordBreak = src.WordBreak
	case "overflow-wrap", "word-wrap":
		dst.OverflowWrap = src.OverflowWrap
	case "text-indent":
		dst.TextIndent = src.TextIndent
	case "color":
		dst.Colour = src.Colour
	case "background-color":
		dst.BackgroundColour = src.BackgroundColour
	case "border-top-color":
		dst.BorderTopColour = src.BorderTopColour
	case "border-right-color":
		dst.BorderRightColour = src.BorderRightColour
	case "border-bottom-color":
		dst.BorderBottomColour = src.BorderBottomColour
	case "border-left-color":
		dst.BorderLeftColour = src.BorderLeftColour
	case "flex-direction":
		dst.FlexDirection = src.FlexDirection
	case "flex-wrap":
		dst.FlexWrap = src.FlexWrap
	case "justify-content":
		dst.JustifyContent = src.JustifyContent
	case "align-items":
		dst.AlignItems = src.AlignItems
	case "align-self":
		dst.AlignSelf = src.AlignSelf
	case "align-content":
		dst.AlignContent = src.AlignContent
	case "justify-items":
		dst.JustifyItems = src.JustifyItems
	case "justify-self":
		dst.JustifySelf = src.JustifySelf
	case "flex-grow":
		dst.FlexGrow = src.FlexGrow
	case "flex-shrink":
		dst.FlexShrink = src.FlexShrink
	case "flex-basis":
		dst.FlexBasis = src.FlexBasis
	case "order":
		dst.Order = src.Order
	case "row-gap":
		dst.RowGap = src.RowGap
	case "column-gap":
		dst.ColumnGap = src.ColumnGap
	case "table-layout":
		dst.TableLayout = src.TableLayout
	case "border-collapse":
		dst.BorderCollapse = src.BorderCollapse
	case "border-spacing":
		dst.BorderSpacing = src.BorderSpacing
	case "caption-side":
		dst.CaptionSide = src.CaptionSide
	case "vertical-align":
		dst.VerticalAlign = src.VerticalAlign
	case "list-style-type":
		dst.ListStyleType = src.ListStyleType
	case "list-style-position":
		dst.ListStylePosition = src.ListStylePosition
	case "page-break-before":
		dst.PageBreakBefore = src.PageBreakBefore
	case "page-break-after":
		dst.PageBreakAfter = src.PageBreakAfter
	case "page-break-inside":
		dst.PageBreakInside = src.PageBreakInside
	case "break-before":
		dst.PageBreakBefore = src.PageBreakBefore
	case "break-after":
		dst.PageBreakAfter = src.PageBreakAfter
	case "break-inside":
		dst.PageBreakInside = src.PageBreakInside
	case "orphans":
		dst.Orphans = src.Orphans
	case "widows":
		dst.Widows = src.Widows
	case "opacity":
		dst.Opacity = src.Opacity
	case "box-shadow":
		dst.BoxShadow = src.BoxShadow
	case "grid-template-columns":
		dst.GridTemplateColumns = src.GridTemplateColumns
	case "grid-template-rows":
		dst.GridTemplateRows = src.GridTemplateRows
	case "grid-auto-columns":
		dst.GridAutoColumns = src.GridAutoColumns
	case "grid-auto-rows":
		dst.GridAutoRows = src.GridAutoRows
	case "grid-column-start":
		dst.GridColumnStart = src.GridColumnStart
	case "grid-column-end":
		dst.GridColumnEnd = src.GridColumnEnd
	case "grid-row-start":
		dst.GridRowStart = src.GridRowStart
	case "grid-row-end":
		dst.GridRowEnd = src.GridRowEnd
	case "grid-template-areas":
		dst.GridTemplateAreas = src.GridTemplateAreas
	case "grid-area":
		dst.GridArea = src.GridArea
	case "writing-mode":
		dst.WritingMode = src.WritingMode
	case "grid-auto-flow":
		dst.GridAutoFlow = src.GridAutoFlow
	case "aspect-ratio":
		dst.AspectRatio = src.AspectRatio
		dst.AspectRatioAuto = src.AspectRatioAuto
	case "text-overflow":
		dst.TextOverflow = src.TextOverflow
	case "content":
		dst.Content = src.Content
	case "column-count":
		dst.ColumnCount = src.ColumnCount
	case "column-width":
		dst.ColumnWidth = src.ColumnWidth
	case "column-fill":
		dst.ColumnFill = src.ColumnFill
	case "column-rule-width":
		dst.ColumnRuleWidth = src.ColumnRuleWidth
	case "column-rule-style":
		dst.ColumnRuleStyle = src.ColumnRuleStyle
	case "column-rule-color":
		dst.ColumnRuleColour = src.ColumnRuleColour
	case "column-span":
		dst.ColumnSpan = src.ColumnSpan
	case "mix-blend-mode":
		dst.MixBlendMode = src.MixBlendMode
	case "text-shadow":
		dst.TextShadow = src.TextShadow
	case "filter":
		dst.Filter = src.Filter
	case "backdrop-filter":
		dst.BackdropFilter = src.BackdropFilter
	case "outline-width":
		dst.OutlineWidth = src.OutlineWidth
	case "outline-style":
		dst.OutlineStyle = src.OutlineStyle
	case "outline-color":
		dst.OutlineColour = src.OutlineColour
	case "outline-offset":
		dst.OutlineOffset = src.OutlineOffset
	case "background-image":
		dst.BgImages = src.BgImages
	case "background-size":
		dst.BgSize = src.BgSize
	case "background-position":
		dst.BgPosition = src.BgPosition
	case "background-repeat":
		dst.BgRepeat = src.BgRepeat
	case "background-attachment":
		dst.BgAttachment = src.BgAttachment
	case "background-origin":
		dst.BgOrigin = src.BgOrigin
	case "background-clip":
		dst.BgClip = src.BgClip
	case "object-fit":
		dst.ObjectFit = src.ObjectFit
	case "object-position":
		dst.ObjectPosition = src.ObjectPosition
	case "border-image-source":
		dst.BorderImageSource = src.BorderImageSource
	case "border-image-slice":
		dst.BorderImageSlice = src.BorderImageSlice
	case "border-image-width":
		dst.BorderImageWidth = src.BorderImageWidth
	case "border-image-outset":
		dst.BorderImageOutset = src.BorderImageOutset
	case "border-image-repeat":
		dst.BorderImageRepeat = src.BorderImageRepeat
	case "clip-path":
		dst.ClipPath = src.ClipPath
	case "mask-image":
		dst.MaskImage = src.MaskImage
	case "direction":
		dst.Direction = src.Direction
	case "unicode-bidi":
		dst.UnicodeBidi = src.UnicodeBidi
	case "hyphens":
		dst.Hyphens = src.Hyphens
	case "tab-size":
		dst.TabSize = src.TabSize
	case "tab-stops":
		dst.TabStops = src.TabStops
	case "counter-reset":
		dst.CounterReset = src.CounterReset
	case "counter-increment":
		dst.CounterIncrement = src.CounterIncrement
	}
}

// parseTabStops parses a custom tab-stops CSS property value into a
// slice of TabStop structs.
//
// The syntax is semicolon-separated entries, each containing a position
// (CSS length), an optional alignment keyword (left, right, centre), and
// an optional quoted leader character:
//
//	tab-stops: 200pt right '.'; 400pt right
//
// Takes value (string) which is the CSS tab-stops property value.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns []TabStop which is the parsed list, or nil for "none" or empty values.
func parseTabStops(value string, context ResolutionContext) []TabStop {
	value = strings.TrimSpace(value)
	if value == "" || value == "none" {
		return nil
	}

	parts := strings.Split(value, ";")
	var stops []TabStop

	for _, part := range parts {
		if stop, ok := parseSingleTabStop(strings.TrimSpace(part), context); ok {
			stops = append(stops, stop)
		}
	}

	return stops
}

// parseSingleTabStop parses one semicolon-delimited tab stop definition
// into a TabStop.
//
// Takes part (string) which is the single tab stop definition text.
// Takes context (ResolutionContext) which provides unit resolution values.
//
// Returns TabStop which is the parsed tab stop.
// Returns bool which is false when the definition is empty or has an
// invalid position.
func parseSingleTabStop(part string, context ResolutionContext) (TabStop, bool) {
	if part == "" {
		return TabStop{}, false
	}

	tokens := tokeniseTabStop(part)
	if len(tokens) == 0 {
		return TabStop{}, false
	}

	position := resolveLength(tokens[0], context)
	if position <= 0 {
		return TabStop{}, false
	}

	stop := TabStop{Position: position, Align: TabAlignLeft}
	for _, token := range tokens[1:] {
		applyTabStopToken(&stop, token)
	}
	return stop, true
}

// applyTabStopToken applies a single token (alignment keyword or quoted
// leader character) to a tab stop.
//
// Takes stop (*TabStop) which is the tab stop to modify.
// Takes token (string) which is the token to apply.
func applyTabStopToken(stop *TabStop, token string) {
	switch token {
	case "left":
		stop.Align = TabAlignLeft
	case "right":
		stop.Align = TabAlignRight
	case "center":
		stop.Align = TabAlignCenter
	default:

		leader := strings.Trim(token, "'\"")
		if len(leader) > 0 {
			stop.Leader = rune(leader[0])
		}
	}
}

// tokeniseTabStop splits a single tab stop definition into tokens,
// respecting quoted strings as single tokens.
//
// Takes s (string) which is the tab stop definition string.
//
// Returns []string which is the list of tokens.
func tokeniseTabStop(s string) []string {
	var ts tabStopTokeniser

	for i := 0; i < len(s); i++ {
		ts.processByte(s[i])
	}
	ts.flush()

	return ts.tokens
}

// tabStopTokeniser accumulates tokens from a tab stop definition string,
// keeping track of whether we are inside a quoted region.
type tabStopTokeniser struct {
	// tokens holds the accumulated tokens.
	tokens []string

	// current accumulates characters for the token being built.
	current strings.Builder

	// inQuote indicates whether the tokeniser is inside a quoted region.
	inQuote bool

	// quoteCh is the quote character that opened the current quoted region.
	quoteCh byte
}

// processByte handles a single character, delegating to the appropriate
// handler based on the current quote state.
//
// Takes ch (byte) which is the character to process.
func (t *tabStopTokeniser) processByte(ch byte) {
	if t.inQuote {
		t.processQuotedByte(ch)
		return
	}
	t.processUnquotedByte(ch)
}

// processQuotedByte handles a character inside a quoted region. If the
// character closes the quote, the accumulated token is emitted.
//
// Takes ch (byte) which is the character to process.
func (t *tabStopTokeniser) processQuotedByte(ch byte) {
	t.current.WriteByte(ch)
	if ch == t.quoteCh {
		t.inQuote = false
		t.emit()
	}
}

// processUnquotedByte handles a character outside any quoted region.
//
// Takes ch (byte) which is the character to process.
func (t *tabStopTokeniser) processUnquotedByte(ch byte) {
	switch ch {
	case '\'', '"':
		t.inQuote = true
		t.quoteCh = ch
		t.current.WriteByte(ch)
	case ' ', '\t':
		t.emit()
	default:
		t.current.WriteByte(ch)
	}
}

// emit appends the current builder content as a token (if non-empty)
// and resets the builder.
func (t *tabStopTokeniser) emit() {
	if t.current.Len() > 0 {
		t.tokens = append(t.tokens, t.current.String())
		t.current.Reset()
	}
}

// flush emits any remaining content after all bytes have been processed.
func (t *tabStopTokeniser) flush() {
	t.emit()
}

// resolveLength converts a CSS length string with units into
// a resolved point value.
//
// Takes value (string) which is the CSS length string
// to resolve.
// Takes context (ResolutionContext) which provides the
// values needed for unit resolution.
//
// Returns float64 which is the resolved length in
// points.
func resolveLength(value string, context ResolutionContext) float64 {
	value = strings.TrimSpace(value)
	if value == "0" {
		return 0
	}

	if strings.HasPrefix(value, "calc(") && strings.HasSuffix(value, ")") {
		inner := value[calcPrefixLength : len(value)-1]
		expression := parseCalc(inner)
		if expression != nil {
			return expression.resolveCalc(context, context.ContainingBlockWidth)
		}
		return 0
	}

	for _, unit := range []struct {
		suffix     string
		multiplier float64
	}{
		{"vmin", math.Min(context.ViewportWidth, context.ViewportHeight) / viewportUnitDivisor},
		{"vmax", math.Max(context.ViewportWidth, context.ViewportHeight) / viewportUnitDivisor},
		{"vw", context.ViewportWidth / viewportUnitDivisor},
		{"vh", context.ViewportHeight / viewportUnitDivisor},
		{"%", context.ContainingBlockWidth / viewportUnitDivisor},
		{"px", PixelsToPoints},
		{"pt", 1.0},
		{"rem", context.RootFontSize},
		{"em", context.ParentFontSize},
		{"cm", CentimetresToPoints},
		{"mm", MillimetresToPoints},
		{"in", InchesToPoints},
		{"pc", PicasToPoints},
	} {
		if numberString, found := strings.CutSuffix(value, unit.suffix); found {
			number, err := strconv.ParseFloat(strings.TrimSpace(numberString), 64)
			if err != nil {
				return 0
			}
			return number * unit.multiplier
		}
	}

	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return number
}

// splitShorthandValues splits a CSS shorthand value into
// whitespace-separated tokens, but preserves function calls
// like calc() as single tokens. For example,
// "calc(10px + 5px) 20px" splits into
// ["calc(10px + 5px)", "20px"].
//
// Takes value (string) which is the CSS shorthand value.
//
// Returns []string which is the list of tokens.
func splitShorthandValues(value string) []string {
	var parts []string
	depth := 0
	start := -1

	for i := 0; i < len(value); i++ {
		start, depth = classifyShorthandChar(value[i], i, start, depth, value, &parts)
	}

	if start != -1 {
		parts = append(parts, value[start:])
	}
	return parts
}

// classifyShorthandChar processes a single character during shorthand
// value splitting, updating the token start position and paren depth.
//
// Takes ch (byte) which is the character to classify.
// Takes i (int) which is the current index in the value string.
// Takes start (int) which is the start index of the current token.
// Takes depth (int) which is the current parenthesis nesting depth.
// Takes value (string) which is the full shorthand value string.
// Takes parts (*[]string) which accumulates the split tokens.
//
// Returns the updated start index and parenthesis depth.
func classifyShorthandChar(ch byte, i, start, depth int, value string, parts *[]string) (newStart, newDepth int) {
	switch {
	case ch == '(':
		if start == -1 {
			start = i
		}
		depth++
	case ch == ')':
		if depth > 0 {
			depth--
		}
	case ch == ' ' && depth == 0:
		if start != -1 {
			*parts = append(*parts, value[start:i])
			start = -1
		}
	default:
		if start == -1 {
			start = i
		}
	}
	return start, depth
}
