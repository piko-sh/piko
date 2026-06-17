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
	"context"
	"math"
	"strings"
)

const (
	// softHyphen is the Unicode soft hyphen character.
	softHyphen = "\u00AD"

	// spaceChar is a single ASCII space.
	spaceChar = " "

	// tabChar is the horizontal tab character.
	tabChar = "\t"

	// subSuperOffsetFactor scales the parent font's x-height to obtain the vertical
	// displacement applied to vertical-align: sub and super items per CSS 2.1 ss10.8.1. Sub
	// lowers by +0.5 * x-height, super raises by -0.5 * x-height.
	subSuperOffsetFactor = 0.5
)

// lineItem pairs a fragment with its horizontal position and width within a line box.
type lineItem struct {
	// fragment is the layout result for this item.
	fragment *Fragment

	// parentMetrics holds the font metrics of the item's containing inline element. CSS
	// vertical-align values sub, super, text-top, and text-bottom are defined relative to
	// the parent element's font, not the item's own font (CSS 2.1 ss10.8.1).
	parentMetrics FontMetrics

	// x is the horizontal offset within the line.
	x float64

	// width is the measured width of this item.
	width float64

	// effectiveVerticalAlign overrides the fragment box's own vertical-align value. CSS
	// vertical-align does not inherit, but a non-baseline value on an inline wrapper (such
	// as <sub> or <sup>) must shift its descendant glyphs alongside the wrapper, so the
	// wrapper's alignment is propagated to text descendants here.
	effectiveVerticalAlign VerticalAlignType
}

// lineBox represents a single horizontal line of inline items with its vertical position
// and dimensions.
type lineBox struct {
	// items is the ordered set of items on this line.
	items []lineItem

	// y is the vertical offset of this line.
	y float64

	// width is the total used width of this line.
	width float64

	// height is the computed height of this line.
	height float64
}

// inlineLayoutContext holds mutable state during inline formatting context layout.
type inlineLayoutContext struct {
	// fontMetrics provides text measurement.
	fontMetrics FontMetricsPort

	// currentLineItems accumulates items for the current line before it is flushed.
	currentLineItems []lineItem

	// lines stores the completed line boxes.
	lines []lineBox

	// floats provides access to the parent BFC's float state for per-line width narrowing.
	// Nil when no floats are active.
	floats *floatContext

	// childReplacements maps original text boxes to their line-segment clones when word
	// wrapping splits a text run across multiple lines. Used to patch the container's
	// Children after layout so each segment has its own LayoutBox with correct Text and
	// Glyphs.
	childReplacements map[*LayoutBox][]*LayoutBox

	// input carries the layout constraints from the parent context.
	input layoutInput

	// contentOffsetX is the horizontal offset from the container's ContentX to the content
	// start (padding + border).
	contentOffsetX float64

	// availableWidth is the maximum line width.
	availableWidth float64

	// cursorX is the current horizontal write position.
	cursorX float64

	// cursorY is the current vertical write position, relative to the container's ContentY.
	cursorY float64

	// currentLineHeight is the tallest item height on the current line.
	currentLineHeight float64

	// floatBFCOffsetY is the Y offset from the BFC root to this container's content top.
	floatBFCOffsetY float64

	// floatContainerX is the X of the BFC content area.
	floatContainerX float64

	// floatContainerWidth is the width of the BFC content area.
	floatContainerWidth float64

	// wrapsText is true when word wrapping is enabled.
	wrapsText bool

	// isEllipsisMode is true when text-overflow: ellipsis is active (requires overflow:
	// hidden and no wrapping).
	isEllipsisMode bool
}

// resolveEffectiveVerticalAlign chooses between an item's own vertical-align and a
// non-baseline alignment inherited from an inline wrapper ancestor. The wrapper-inherited
// value wins when set, so text inside <sub>/<sup> shifts alongside the wrapper instead of
// sitting on the baseline.
//
// Takes ownVA (VerticalAlignType) which is the item's own vertical-align style.
// Takes inheritedVA (VerticalAlignType) which is the alignment propagated by an enclosing
// inline wrapper, or VerticalAlignBaseline when none applies.
//
// Returns VerticalAlignType which is the effective alignment to use during layout.
func resolveEffectiveVerticalAlign(ownVA, inheritedVA VerticalAlignType) VerticalAlignType {
	if inheritedVA != VerticalAlignBaseline {
		return inheritedVA
	}
	return ownVA
}

// extendLineHeightForVerticalAlign grows the current line height by the additional
// vertical extent that a non-baseline vertical-align value introduces. Sub and super
// shift the item by half the parent's x-height; text-top and text-bottom stretch the line
// to the parent's content area when the parent's font is taller than the item's own text.
//
// Takes lineHeight (float64) which is the line height accumulated so far from regular
// text metrics.
// Takes itemHeight (float64) which is the natural margin-box height of the item being
// appended.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics.
// Takes va (VerticalAlignType) which is the effective alignment.
//
// Returns float64 which is the line height after growth.
func extendLineHeightForVerticalAlign(lineHeight, itemHeight float64, parentMetrics FontMetrics, va VerticalAlignType) float64 {
	switch va {
	case VerticalAlignSub, VerticalAlignSuper:
		grown := itemHeight + parentMetrics.XHeight*subSuperOffsetFactor
		if grown > lineHeight {
			return grown
		}
	case VerticalAlignTextTop, VerticalAlignTextBottom:
		parentBox := parentMetrics.Ascent + parentMetrics.Descent
		if parentBox > lineHeight {
			return parentBox
		}
	default:
	}
	return lineHeight
}

// parentFontMetrics resolves the FontMetrics for an inline item's containing element.
// Used so sub, super, text-top, and text-bottom vertical alignment can offset relative to
// the parent's font (CSS 2.1 ss10.8.1).
//
// Takes fontMetrics (FontMetricsPort) which resolves font descriptors to metrics.
// Takes parentStyle (*ComputedStyle) which is the containing element's resolved style.
//
// Returns FontMetrics which holds the parent's vertical metrics, or a zero value when no
// parent style is supplied.
func parentFontMetrics(fontMetrics FontMetricsPort, parentStyle *ComputedStyle) FontMetrics {
	if parentStyle == nil || fontMetrics == nil {
		return FontMetrics{}
	}
	font := FontDescriptor{
		Family: parentStyle.FontFamily,
		Weight: parentStyle.FontWeight,
		Style:  parentStyle.FontStyle,
	}
	return fontMetrics.GetMetrics(font, parentStyle.FontSize)
}

// effectiveAvailableWidth returns the available width for the current line, narrowed by
// any floats that overlap at the current cursorY. The result is always capped at the
// container's own content width, since floats from an ancestor BFC cannot widen the
// available space beyond what the container provides.
//
// Returns float64 which is the effective available width in points.
func (c *inlineLayoutContext) effectiveAvailableWidth() float64 {
	if c.floats == nil {
		return c.availableWidth
	}
	bfcY := c.floatBFCOffsetY + c.cursorY
	floatWidth := c.floats.availableWidthAtY(bfcY, 0, c.floatContainerX, c.floatContainerWidth)
	return math.Min(c.availableWidth, floatWidth)
}

// lineContentOffsetX returns the horizontal content start position for the current line,
// incorporating the float left offset when floats overlap at the current cursor Y.
//
// Returns float64 which is the horizontal content start position.
func (c *inlineLayoutContext) lineContentOffsetX() float64 {
	if c.floats == nil {
		return c.contentOffsetX
	}
	bfcY := c.floatBFCOffsetY + c.cursorY
	return c.contentOffsetX + c.floats.leftOffsetAtY(bfcY, 0, c.floatContainerX)
}

// flushLine finalises the current line, applies vertical alignment, and advances the
// cursor to the next line.
func (c *inlineLayoutContext) flushLine() {
	if len(c.currentLineItems) > 0 || c.currentLineHeight > 0 {
		applyInlineVerticalAlign(c.currentLineItems, c.currentLineHeight)

		c.lines = append(c.lines, lineBox{
			items:  c.currentLineItems,
			y:      c.cursorY,
			width:  c.cursorX,
			height: c.currentLineHeight,
		})
	}
	c.currentLineItems = nil
	c.cursorY += c.currentLineHeight
	c.cursorX = 0
	c.currentLineHeight = 0
}

// layoutNonTextChild sizes and positions a non-text inline child.
//
// Inline-block and replaced elements are laid out as block formatting contexts and placed
// as atomic inline units. BoxInline children (e.g. spans) are transparent wrappers whose
// children are recursively processed within the same inline formatting context, then the
// wrapper fragment is sized to cover its children's extent.
//
// Takes ctx (context.Context) which carries the cancellation signal through recursive
// layout.
// Takes child (*LayoutBox) which is the non-text inline box to position.
// Takes parentStyle (*ComputedStyle) which is the style of the containing inline element;
// used to resolve parent font metrics for vertical-align.
// Takes inheritedVA (VerticalAlignType) which is the vertical-align value propagated by
// an enclosing wrapper, so that nested wrappers and their text descendants share the
// outer wrapper's offset.
func (c *inlineLayoutContext) layoutNonTextChild(ctx context.Context, child *LayoutBox, parentStyle *ComputedStyle, inheritedVA VerticalAlignType) {
	if child.Type == BoxInlineBlock || child.Type == BoxReplaced {
		c.layoutAtomicInline(ctx, child, parentStyle, inheritedVA)
		return
	}

	marginLeft := child.Style.MarginLeft.Resolve(c.availableWidth, 0)
	marginRight := child.Style.MarginRight.Resolve(c.availableWidth, 0)

	c.cursorX += marginLeft

	startX := c.cursorX
	childFragment := &Fragment{
		Box:     child,
		OffsetX: c.lineContentOffsetX() + c.cursorX,
		OffsetY: c.cursorY,
	}
	childFragment.Margin.Left = marginLeft
	childFragment.Margin.Right = marginRight
	effectiveVA := resolveEffectiveVerticalAlign(child.Style.VerticalAlign, inheritedVA)
	itemIndex := len(c.currentLineItems)
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               childFragment,
		parentMetrics:          parentFontMetrics(c.fontMetrics, parentStyle),
		x:                      c.cursorX - marginLeft,
		width:                  0,
		effectiveVerticalAlign: effectiveVA,
	})

	for _, grandchild := range child.Children {
		if grandchild.Type == BoxTextRun {
			c.layoutTextChild(grandchild, &child.Style, effectiveVA)
		} else {
			c.layoutNonTextChild(ctx, grandchild, &child.Style, effectiveVA)
		}
	}

	c.cursorX += marginRight

	inlineWidth := c.cursorX - startX + marginLeft
	if itemIndex < len(c.currentLineItems) {
		c.currentLineItems[itemIndex].width = inlineWidth
	}
	childFragment.ContentWidth = c.cursorX - marginRight - startX
	childFragment.ContentHeight = c.currentLineHeight
}

// layoutAtomicInline sizes an inline-block or replaced element as a block formatting
// context and places it on the current line as an atomic inline unit.
//
// Takes ctx (context.Context) which carries the cancellation signal through recursive
// layout.
// Takes child (*LayoutBox) which is the inline-block or replaced element to lay out.
// Takes parentStyle (*ComputedStyle) which is the style of the containing inline element;
// used to resolve parent font metrics for vertical-align.
// Takes inheritedVA (VerticalAlignType) which is the vertical-align value propagated by
// an enclosing inline wrapper.
func (c *inlineLayoutContext) layoutAtomicInline(ctx context.Context, child *LayoutBox, parentStyle *ComputedStyle, inheritedVA VerticalAlignType) {
	childFragment := layoutBox(ctx, child, layoutInput{AvailableWidth: c.availableWidth, FontMetrics: c.fontMetrics, Cache: c.input.Cache})
	childFragment.Margin.Left = child.Style.MarginLeft.Resolve(c.availableWidth, 0)
	childFragment.Margin.Right = child.Style.MarginRight.Resolve(c.availableWidth, 0)
	childFragment.Margin.Top = child.Style.MarginTop.Resolve(c.availableWidth, 0)
	childFragment.Margin.Bottom = child.Style.MarginBottom.Resolve(c.availableWidth, 0)

	inlineWidth := childFragment.MarginBoxWidth()
	inlineHeight := childFragment.MarginBoxHeight()

	if c.wrapsText && c.cursorX+inlineWidth > c.effectiveAvailableWidth() && c.cursorX > 0 {
		c.flushLine()
	}

	childFragment.OffsetX = c.lineContentOffsetX() + c.cursorX + childFragment.Margin.Left + childFragment.Padding.Left + childFragment.Border.Left
	childFragment.OffsetY = c.cursorY + childFragment.Margin.Top + childFragment.Padding.Top + childFragment.Border.Top

	parentMetrics := parentFontMetrics(c.fontMetrics, parentStyle)
	effectiveVA := resolveEffectiveVerticalAlign(child.Style.VerticalAlign, inheritedVA)
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               childFragment,
		parentMetrics:          parentMetrics,
		x:                      c.cursorX,
		width:                  inlineWidth,
		effectiveVerticalAlign: effectiveVA,
	})
	c.cursorX += inlineWidth
	if inlineHeight > c.currentLineHeight {
		c.currentLineHeight = inlineHeight
	}
	c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, inlineHeight, parentMetrics, effectiveVA)
}

// layoutUnwrappedTextRun places a text run that fits on the current line without
// wrapping.
//
// Takes child (*LayoutBox) which is the text run box to position.
// Takes totalWidth (float64) which is the measured width of the text.
// Takes textLineHeight (float64) which is the line height for the text run.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics, used by
// vertical-align sub/super/text-top/text-bottom.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply, either the text
// run's own or one inherited from a wrapping inline.
func (c *inlineLayoutContext) layoutUnwrappedTextRun(child *LayoutBox, totalWidth, textLineHeight float64, parentMetrics FontMetrics, effectiveVA VerticalAlignType) {
	childFragment := &Fragment{
		Box:           child,
		OffsetX:       c.lineContentOffsetX() + c.cursorX,
		OffsetY:       c.cursorY,
		ContentWidth:  totalWidth,
		ContentHeight: textLineHeight,
	}
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               childFragment,
		parentMetrics:          parentMetrics,
		x:                      c.cursorX,
		width:                  totalWidth,
		effectiveVerticalAlign: effectiveVA,
	})
	c.cursorX += totalWidth
	if textLineHeight > c.currentLineHeight {
		c.currentLineHeight = textLineHeight
	}
	c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, textLineHeight, parentMetrics, effectiveVA)
}

// layoutEllipsisTextRun truncates a text run to fit the available width and appends an
// ellipsis character.
//
// Truncation operates on grapheme clusters so that multi-codepoint sequences (emoji with
// ZWJ, base + combining marks) are never split. Used when text-overflow: ellipsis is
// active with overflow: hidden and white-space: nowrap.
//
// Takes child (*LayoutBox) which is the text run box.
// Takes font (FontDescriptor) which is the resolved font.
// Takes fontSize (float64) which is the font size in points.
// Takes textLineHeight (float64) which is the line height.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics, used by
// vertical-align sub/super/text-top/text-bottom.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply.
func (c *inlineLayoutContext) layoutEllipsisTextRun(
	child *LayoutBox, font FontDescriptor, fontSize, _, textLineHeight float64,
	parentMetrics FontMetrics, effectiveVA VerticalAlignType,
) {
	direction := child.Style.Direction
	ellipsis := "\u2026"
	ellipsisWidth := c.fontMetrics.MeasureText(font, fontSize, ellipsis, direction)
	targetWidth := c.effectiveAvailableWidth() - c.cursorX - ellipsisWidth

	clusters := c.fontMetrics.SplitGraphemeClusters(child.Text)
	truncated := ""
	for _, cluster := range clusters {
		candidate := truncated + cluster
		candidateWidth := c.fontMetrics.MeasureText(font, fontSize, candidate, direction)
		if candidateWidth > targetWidth {
			break
		}
		truncated = candidate
	}

	child.Text = truncated + ellipsis
	child.Glyphs = c.fontMetrics.ShapeText(font, fontSize, child.Text, direction)
	displayWidth := c.fontMetrics.MeasureText(font, fontSize, child.Text, direction)
	childFragment := &Fragment{
		Box:           child,
		OffsetX:       c.lineContentOffsetX() + c.cursorX,
		OffsetY:       c.cursorY,
		ContentWidth:  displayWidth,
		ContentHeight: textLineHeight,
	}
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               childFragment,
		parentMetrics:          parentMetrics,
		x:                      c.cursorX,
		width:                  displayWidth,
		effectiveVerticalAlign: effectiveVA,
	})
	c.cursorX += displayWidth
	if textLineHeight > c.currentLineHeight {
		c.currentLineHeight = textLineHeight
	}
	c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, textLineHeight, parentMetrics, effectiveVA)
}

// verticalAlignInputs bundles the parent-element font metrics and the effective
// vertical-align value applied to a generated fragment. It exists to keep call sites
// under the seven-argument limit; sub, super, text-top, and text-bottom alignments are
// computed relative to the parent's font per CSS 2.1 ss10.8.1, so both values are needed
// together.
type verticalAlignInputs struct {
	// parentMetrics holds the containing inline element's font metrics.
	parentMetrics FontMetrics

	// effective is the resolved vertical-align value to apply.
	effective VerticalAlignType
}

// emitTextSegment creates a cloned LayoutBox for a line segment of wrapped text and adds
// it to the current line. The clone has its own Text and Glyphs (shaped for that
// segment), so writeFragmentsToBoxTree assigns it a unique position.
//
// Takes original (*LayoutBox) which is the original text run box being wrapped.
// Takes font (FontDescriptor) which is the font for measurement and shaping.
// Takes fontSize (float64) which is the font size.
// Takes segmentText (string) which is the pre-built text for this segment.
// Takes startX (float64) which is the horizontal start position of this segment on the
// current line.
// Takes textLineHeight (float64) which is the line height for the text.
// Takes va (verticalAlignInputs) which bundles the parent-element font metrics and the
// effective vertical-align value to apply.
func (c *inlineLayoutContext) emitTextSegment(
	original *LayoutBox, font FontDescriptor, fontSize float64,
	segmentText string, startX, textLineHeight float64,
	va verticalAlignInputs,
) {
	segmentBox := &LayoutBox{
		Type:           original.Type,
		Style:          original.Style,
		Text:           segmentText,
		BaselineOffset: original.BaselineOffset,
	}
	segmentBox.Glyphs = c.fontMetrics.ShapeText(font, fontSize, segmentText, original.Style.Direction)
	segmentWidth := c.fontMetrics.MeasureText(font, fontSize, segmentText, original.Style.Direction)
	if original.Style.LetterSpacing != 0 {
		segmentWidth += original.Style.LetterSpacing * float64(len([]rune(segmentText)))
	}

	fragment := &Fragment{
		Box:           segmentBox,
		OffsetX:       c.lineContentOffsetX() + startX,
		OffsetY:       c.cursorY,
		ContentWidth:  segmentWidth,
		ContentHeight: textLineHeight,
	}
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               fragment,
		parentMetrics:          va.parentMetrics,
		x:                      startX,
		width:                  segmentWidth,
		effectiveVerticalAlign: va.effective,
	})

	if c.childReplacements == nil {
		c.childReplacements = make(map[*LayoutBox][]*LayoutBox)
	}
	c.childReplacements[original] = append(c.childReplacements[original], segmentBox)
}

// preprocessTextChild expands tabs, applies text transforms, and strips leading
// whitespace at the start of a line. Returns false if the child text becomes empty after
// preprocessing and should be skipped.
//
// Takes child (*LayoutBox) which is the text run to preprocess.
//
// Returns bool which is true if the text is non-empty and layout should continue.
func (c *inlineLayoutContext) preprocessTextChild(child *LayoutBox) bool {
	if preservesWhitespace(child.Style.WhiteSpace) && strings.Contains(child.Text, tabChar) {
		tabSpaces := int(child.Style.TabSize)
		if tabSpaces < 1 {
			tabSpaces = defaultTabSize
		}
		child.Text = strings.ReplaceAll(child.Text, tabChar, strings.Repeat(spaceChar, tabSpaces))
	}

	child.Text = applyTextTransform(child.Text, child.Style.TextTransform)

	if !preservesWhitespace(child.Style.WhiteSpace) && c.cursorX == 0 {
		child.Text = strings.TrimLeft(child.Text, " \t\n\r")
	}

	return child.Text != ""
}

// layoutTextChild measures a text child and delegates to the wrapped or unwrapped layout
// path.
//
// Takes child (*LayoutBox) which is the text run box to measure and lay out.
// Takes parentStyle (*ComputedStyle) which is the style of the containing inline element;
// used to resolve parent font metrics for vertical-align sub/super/text-top/text-bottom.
// Takes inheritedVA (VerticalAlignType) which is the vertical-align value propagated from
// an enclosing inline wrapper.
func (c *inlineLayoutContext) layoutTextChild(child *LayoutBox, parentStyle *ComputedStyle, inheritedVA VerticalAlignType) {
	if strings.Contains(child.Text, tabChar) && len(child.Style.TabStops) > 0 {
		c.layoutTextWithTabStops(child, parentStyle, inheritedVA)
		return
	}

	if !c.preprocessTextChild(child) {
		return
	}

	font := FontDescriptor{
		Family: child.Style.FontFamily,
		Weight: child.Style.FontWeight,
		Style:  child.Style.FontStyle,
	}
	fontSize := child.Style.FontSize
	metrics := c.fontMetrics.GetMetrics(font, fontSize)
	fontLineHeight := metrics.Ascent + metrics.Descent + metrics.LineGap
	textLineHeight := child.Style.LineHeight
	if textLineHeight < fontLineHeight {
		textLineHeight = fontLineHeight
	}

	halfLeading := (textLineHeight - fontLineHeight) / 2
	child.BaselineOffset = halfLeading + metrics.Ascent

	runs := splitIntoBidiRuns(child.Text, child.Style.Direction, child.Style.UnicodeBidi)
	direction := resolveRunDirection(runs, child.Style.Direction)

	totalWidth := c.fontMetrics.MeasureText(font, fontSize, child.Text, direction)
	if child.Style.LetterSpacing != 0 {
		totalWidth += child.Style.LetterSpacing * float64(len([]rune(child.Text)))
	}
	child.Glyphs = c.fontMetrics.ShapeText(font, fontSize, child.Text, direction)

	parentMetrics := parentFontMetrics(c.fontMetrics, parentStyle)
	effectiveVA := resolveEffectiveVerticalAlign(child.Style.VerticalAlign, inheritedVA)

	if c.isEllipsisMode && totalWidth > c.effectiveAvailableWidth()-c.cursorX {
		c.layoutEllipsisTextRun(child, font, fontSize, totalWidth, textLineHeight, parentMetrics, effectiveVA)
		return
	}

	if !c.wrapsText || totalWidth <= c.effectiveAvailableWidth()-c.cursorX {
		c.layoutUnwrappedTextRun(child, totalWidth, textLineHeight, parentMetrics, effectiveVA)
	} else if child.Style.WordBreak == WordBreakBreakAll {
		c.layoutCharacterBreakTextRun(child, font, fontSize, textLineHeight, parentMetrics, effectiveVA)
	} else {
		c.layoutWrappedTextRun(child, font, fontSize, totalWidth, textLineHeight, parentMetrics, effectiveVA)
	}
}

// resolveTabStopTargetX computes the horizontal target position for a tab stop,
// accounting for right and centre alignment by measuring the upcoming segment width and
// offsetting accordingly. The result is clamped so it never falls before the current
// cursor position.
//
// Takes stop (TabStop) which is the tab stop definition.
// Takes segmentWidth (float64) which is the measured width of the segment that follows
// this tab.
// Takes cursorX (float64) which is the current horizontal position.
//
// Returns float64 which is the resolved target X coordinate.
func resolveTabStopTargetX(stop TabStop, segmentWidth float64, cursorX float64) float64 {
	targetX := stop.Position

	switch stop.Align { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case TabAlignRight:
		targetX = stop.Position - segmentWidth
	case TabAlignCenter:
		targetX = stop.Position - segmentWidth/2
	}

	if targetX < cursorX {
		targetX = cursorX
	}

	return targetX
}

// tabTextParams groups the font and style parameters shared by the tab leader and segment
// emission helpers, reducing their argument counts to stay within the linter limit.
type tabTextParams struct {
	// Style holds the computed style for the text run.
	Style *ComputedStyle

	// Font holds the resolved font descriptor for measurement.
	Font FontDescriptor

	// ParentMetrics holds the parent element's font metrics, used by vertical-align
	// sub/super/text-top/text-bottom calculation.
	ParentMetrics FontMetrics

	// FontSize holds the font size in points.
	FontSize float64

	// TextLineHeight holds the computed line height.
	TextLineHeight float64

	// Direction holds the text direction (LTR or RTL).
	Direction DirectionType

	// EffectiveVA holds the vertical-align value to apply.
	EffectiveVA VerticalAlignType
}

// emitTabLeaderFill creates a leader-fill fragment spanning the gap between leaderStartX
// and targetX using the given leader rune. If the leader character has zero width or the
// gap is too narrow for even one repetition, no fragment is emitted.
//
// Takes stop (TabStop) which defines the leader character.
// Takes leaderStartX (float64) which is the gap's left edge.
// Takes targetX (float64) which is the gap's right edge.
// Takes params (tabTextParams) which groups font, style, line height, and direction.
func (c *inlineLayoutContext) emitTabLeaderFill(
	stop TabStop, leaderStartX float64, targetX float64,
	params tabTextParams,
) {
	if stop.Leader == 0 || targetX <= leaderStartX {
		return
	}

	leaderText := string(stop.Leader)
	leaderWidth := c.fontMetrics.MeasureText(params.Font, params.FontSize, leaderText, params.Direction)
	if leaderWidth <= 0 {
		return
	}

	gap := targetX - leaderStartX
	repeatCount := int(gap / leaderWidth)
	if repeatCount <= 0 {
		return
	}

	fullLeader := strings.Repeat(leaderText, repeatCount)
	leaderBox := &LayoutBox{
		Text:  fullLeader,
		Type:  BoxTextRun,
		Style: *params.Style,
	}
	leaderBox.Glyphs = c.fontMetrics.ShapeText(params.Font, params.FontSize, fullLeader, params.Direction)
	leaderTotal := c.fontMetrics.MeasureText(params.Font, params.FontSize, fullLeader, params.Direction)

	frag := &Fragment{
		Box:           leaderBox,
		OffsetX:       c.lineContentOffsetX() + leaderStartX,
		OffsetY:       c.cursorY,
		ContentWidth:  leaderTotal,
		ContentHeight: params.TextLineHeight,
	}
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               frag,
		parentMetrics:          params.ParentMetrics,
		x:                      leaderStartX,
		width:                  leaderTotal,
		effectiveVerticalAlign: params.EffectiveVA,
	})
}

// emitTabSegment measures a non-empty text segment, creates a LayoutBox and Fragment for
// it, and appends it to the current line items. The cursor advances by the segment width.
//
// Takes segment (string) which is the text to emit.
// Takes params (tabTextParams) which groups font, style, line height, and direction.
func (c *inlineLayoutContext) emitTabSegment(
	segment string, params tabTextParams,
) {
	segWidth := c.fontMetrics.MeasureText(params.Font, params.FontSize, segment, params.Direction)
	segBox := &LayoutBox{
		Text:  segment,
		Type:  BoxTextRun,
		Style: *params.Style,
	}
	segBox.Glyphs = c.fontMetrics.ShapeText(params.Font, params.FontSize, segment, params.Direction)

	frag := &Fragment{
		Box:           segBox,
		OffsetX:       c.lineContentOffsetX() + c.cursorX,
		OffsetY:       c.cursorY,
		ContentWidth:  segWidth,
		ContentHeight: params.TextLineHeight,
	}
	c.currentLineItems = append(c.currentLineItems, lineItem{
		fragment:               frag,
		parentMetrics:          params.ParentMetrics,
		x:                      c.cursorX,
		width:                  segWidth,
		effectiveVerticalAlign: params.EffectiveVA,
	})
	c.cursorX += segWidth
}

// layoutTextWithTabStops splits text at tab characters and positions each segment
// according to the tab stop list.
//
// For right-aligned stops, the text after the tab is right-aligned at the stop position.
// Leader characters fill the gap before each tab stop.
//
// Takes child (*LayoutBox) which is the text run box containing tab characters.
// Takes parentStyle (*ComputedStyle) which is the style of the containing inline element;
// used to resolve parent font metrics for vertical-align.
// Takes inheritedVA (VerticalAlignType) which is the vertical-align value propagated by
// an enclosing inline wrapper.
func (c *inlineLayoutContext) layoutTextWithTabStops(child *LayoutBox, parentStyle *ComputedStyle, inheritedVA VerticalAlignType) {
	child.Text = applyTextTransform(child.Text, child.Style.TextTransform)

	font := FontDescriptor{
		Family: child.Style.FontFamily,
		Weight: child.Style.FontWeight,
		Style:  child.Style.FontStyle,
	}
	fontSize := child.Style.FontSize
	metrics := c.fontMetrics.GetMetrics(font, fontSize)
	fontLineHeight := metrics.Ascent + metrics.Descent + metrics.LineGap
	textLineHeight := child.Style.LineHeight
	if textLineHeight < fontLineHeight {
		textLineHeight = fontLineHeight
	}

	halfLeading := (textLineHeight - fontLineHeight) / 2
	child.BaselineOffset = halfLeading + metrics.Ascent

	direction := child.Style.Direction
	segments := strings.Split(child.Text, tabChar)
	stops := child.Style.TabStops
	stopIndex := 0

	params := tabTextParams{
		Font:           font,
		Style:          &child.Style,
		ParentMetrics:  parentFontMetrics(c.fontMetrics, parentStyle),
		FontSize:       fontSize,
		TextLineHeight: textLineHeight,
		Direction:      direction,
		EffectiveVA:    resolveEffectiveVerticalAlign(child.Style.VerticalAlign, inheritedVA),
	}

	for i, segment := range segments {
		if i > 0 && stopIndex < len(stops) {
			stop := stops[stopIndex]
			stopIndex++

			leaderStartX := c.cursorX
			segWidth := c.fontMetrics.MeasureText(font, fontSize, segment, direction)
			targetX := resolveTabStopTargetX(stop, segWidth, c.cursorX)
			c.emitTabLeaderFill(stop, leaderStartX, targetX, params)
			c.cursorX = targetX
		}

		if segment != "" {
			c.emitTabSegment(segment, params)
		}
	}

	if textLineHeight > c.currentLineHeight {
		c.currentLineHeight = textLineHeight
	}
	c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, textLineHeight, params.ParentMetrics, params.EffectiveVA)
}

// resolveRunDirection returns the direction to use for text measurement.
//
// For single-run text (the common case), it uses the run's resolved direction. Otherwise
// it falls back to the style direction.
//
// Takes runs ([]bidiRun) which holds the bidirectional text runs.
// Takes styleDirection (DirectionType) which is the fallback direction from the style.
//
// Returns DirectionType which is the resolved text direction.
func resolveRunDirection(runs []bidiRun, styleDirection DirectionType) DirectionType {
	if len(runs) == 1 {
		return runs[0].direction
	}
	return styleDirection
}

// layoutInlineContent performs inline formatting context layout for a container's
// children, returning child fragments with parent-relative offsets, the intrinsic content
// height, and margin information.
//
// Takes ctx (context.Context) which carries the cancellation signal through recursive
// layout.
// Takes container (*LayoutBox) which is the box whose inline children are laid out.
// Takes input (layoutInput) which carries the available width, font metrics, and cache
// from the parent context.
//
// Returns formattingContextResult with the layout results for this container and its
// inline children.
func layoutInlineContent(ctx context.Context, container *LayoutBox, input layoutInput) formattingContextResult {
	contentOffsetX := 0.0

	inlineLayoutCtx := &inlineLayoutContext{
		input:               input,
		fontMetrics:         input.FontMetrics,
		contentOffsetX:      contentOffsetX,
		availableWidth:      input.AvailableWidth,
		wrapsText:           allowsWrapping(container.Style.WhiteSpace),
		cursorX:             container.Style.TextIndent,
		cursorY:             0,
		floats:              input.Floats,
		floatBFCOffsetY:     input.FloatBFCOffsetY,
		floatContainerX:     input.FloatContainerX,
		floatContainerWidth: input.FloatContainerWidth,
		isEllipsisMode: container.Style.TextOverflow == TextOverflowEllipsis &&
			container.Style.OverflowX != OverflowVisible &&
			!allowsWrapping(container.Style.WhiteSpace),
	}

	for _, child := range container.Children {
		if child.Type == BoxListMarker {
			continue
		}
		if child.Type != BoxTextRun {
			inlineLayoutCtx.layoutNonTextChild(ctx, child, &container.Style, VerticalAlignBaseline)
			continue
		}
		inlineLayoutCtx.layoutTextChild(child, &container.Style, VerticalAlignBaseline)
	}

	inlineLayoutCtx.flushLine()

	if len(inlineLayoutCtx.childReplacements) > 0 {
		applyChildReplacements(container, inlineLayoutCtx.childReplacements)
	}

	applyTextAlignment(inlineLayoutCtx.lines, container.Style.TextAlign, container.Style.Direction, inlineLayoutCtx.availableWidth, contentOffsetX)

	if container.Style.Direction == DirectionRTL {
		mirrorInlineLines(inlineLayoutCtx.lines, inlineLayoutCtx.availableWidth, contentOffsetX)
	}

	lineFragments, totalHeight := buildLineFragments(inlineLayoutCtx.lines)

	return formattingContextResult{
		Children:      lineFragments,
		ContentHeight: totalHeight,
		Margin: BoxEdges{
			Top:    input.Edges.MarginTop,
			Bottom: input.Edges.MarginBottom,
		},
	}
}

// applyChildReplacements recursively swaps original text-run boxes for their line-segment
// clones throughout the inline subtree.
//
// Takes box (*LayoutBox) which is the subtree root to rewrite in place.
// Takes replacements (map[*LayoutBox][]*LayoutBox) which maps each original text run to
// its line-segment clones.
func applyChildReplacements(box *LayoutBox, replacements map[*LayoutBox][]*LayoutBox) {
	if len(box.Children) == 0 {
		return
	}
	newChildren := make([]*LayoutBox, 0, len(box.Children))
	for _, child := range box.Children {
		if repl, ok := replacements[child]; ok {
			newChildren = append(newChildren, repl...)
			continue
		}
		applyChildReplacements(child, replacements)
		newChildren = append(newChildren, child)
	}
	box.Children = newChildren
}

// buildLineFragments constructs fragment trees from completed line boxes and computes the
// total content height.
//
// Takes lines ([]lineBox) which holds the completed line boxes.
//
// Returns []*Fragment which holds the constructed fragment trees.
// Returns float64 which is the total content height.
func buildLineFragments(lines []lineBox) ([]*Fragment, float64) {
	lineFragments := make([]*Fragment, 0, len(lines))
	totalHeight := 0.0
	for _, line := range lines {
		lineChildren := make([]*Fragment, 0, len(line.items))
		for _, item := range line.items {
			item.fragment.OffsetY -= line.y
			lineChildren = append(lineChildren, item.fragment)
		}
		lineFragment := &Fragment{
			Children:      lineChildren,
			OffsetY:       line.y,
			ContentWidth:  line.width,
			ContentHeight: line.height,
		}
		lineFragments = append(lineFragments, lineFragment)
		totalHeight += line.height
	}
	return lineFragments, totalHeight
}

// applyTextAlignment adjusts fragment positions in each line according to the text-align
// mode.
//
// Takes lines ([]lineBox) which is the set of line boxes to adjust.
// Takes textAlign (TextAlignType) which is the alignment mode to apply.
// Takes availableWidth (float64) which is the maximum line width for offset calculation.
// Takes contentOffsetX (float64) which is the horizontal offset from the container's
// ContentX to the content start (padding + border).
func applyTextAlignment(lines []lineBox, textAlign TextAlignType, direction DirectionType, availableWidth, contentOffsetX float64) {
	textAlign = resolveTextAlign(textAlign, direction)

	if textAlign == TextAlignLeft {
		return
	}

	for lineIndex, line := range lines {
		switch textAlign {
		case TextAlignJustify:
			applyJustifyToLine(line, lineIndex, len(lines), availableWidth, contentOffsetX)
		case TextAlignCentre, TextAlignRight:
			applyOffsetToLine(line, textAlign, availableWidth, contentOffsetX)
		default:
			continue
		}
	}
}

// resolveTextAlign maps logical text-align values (start, end) to physical values (left,
// right) based on the inline base direction. Physical values pass through unchanged.
//
// Takes textAlign (TextAlignType) which is the text-align value.
// Takes direction (DirectionType) which is the inline base direction.
//
// Returns TextAlignType which is the resolved physical alignment.
func resolveTextAlign(textAlign TextAlignType, direction DirectionType) TextAlignType {
	switch textAlign {
	case TextAlignStart:
		if direction == DirectionRTL {
			return TextAlignRight
		}
		return TextAlignLeft
	case TextAlignEnd:
		if direction == DirectionRTL {
			return TextAlignLeft
		}
		return TextAlignRight
	default:
		return textAlign
	}
}

// mirrorInlineLines reverses the visual order of items within each line for RTL inline
// flow. Each item's x-position is mirrored so the first item in logical order appears at
// the right edge of the line.
//
// Takes lines ([]lineBox) which holds the line boxes to mirror.
// Takes availableWidth (float64) which is the maximum line width.
// Takes contentOffsetX (float64) which is the horizontal content start offset.
func mirrorInlineLines(lines []lineBox, availableWidth, contentOffsetX float64) {
	for _, line := range lines {
		for _, item := range line.items {
			mirroredX := availableWidth - item.x - item.width
			item.fragment.OffsetX = contentOffsetX + mirroredX
		}
	}
}

// applyJustifyToLine distributes extra space evenly between items to fill the available
// width.
//
// Takes line (lineBox) which is the line to justify.
// Takes lineIndex (int) which is the zero-based index of this line.
// Takes lineCount (int) which is the total number of lines, used to skip the last line.
// Takes availableWidth (float64) which is the maximum line width.
// Takes contentOffsetX (float64) which is the horizontal offset from the container's
// ContentX to the content start (padding + border).
func applyJustifyToLine(line lineBox, lineIndex, lineCount int, availableWidth, contentOffsetX float64) {
	if lineIndex == lineCount-1 {
		return
	}

	gapCount := len(line.items) - 1
	freeSpace := availableWidth - line.width
	if freeSpace <= 0 {
		return
	}
	if gapCount <= 0 {
		if len(line.items) == 1 {
			line.items[0].fragment.ContentWidth = availableWidth
		}
		return
	}

	gapIncrement := freeSpace / float64(gapCount)
	for itemIndex, item := range line.items {
		item.fragment.OffsetX = contentOffsetX + item.x + gapIncrement*float64(itemIndex)
		if itemIndex < gapCount {
			item.fragment.ContentWidth += gapIncrement
		} else {
			item.fragment.ContentWidth = availableWidth -
				(item.fragment.OffsetX - contentOffsetX)
		}
	}
}

// applyOffsetToLine shifts all items in a line by a horizontal offset for centre or right
// alignment.
//
// Takes line (lineBox) which is the line to shift.
// Takes textAlign (TextAlignType) which is the alignment mode, either centre or right.
// Takes availableWidth (float64) which is the maximum line width.
// Takes contentOffsetX (float64) which is the horizontal offset from the container's
// ContentX to the content start (padding + border).
func applyOffsetToLine(line lineBox, textAlign TextAlignType, availableWidth, contentOffsetX float64) {
	var offset float64
	switch textAlign { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case TextAlignCentre:
		offset = (availableWidth - line.width) / 2
	case TextAlignRight:
		offset = availableWidth - line.width
	}

	if offset <= 0 {
		return
	}

	for _, item := range line.items {
		item.fragment.OffsetX = contentOffsetX + item.x + offset
	}
}

// itemVerticalAlign returns the alignment value an item should be laid out with,
// preferring the wrapper-inherited value when set so that text inside <sub>/<sup> shifts
// alongside the wrapper.
//
// Takes item (lineItem) which is the line item to inspect.
//
// Returns VerticalAlignType which is the effective alignment.
func itemVerticalAlign(item lineItem) VerticalAlignType {
	if item.effectiveVerticalAlign != VerticalAlignBaseline {
		return item.effectiveVerticalAlign
	}
	return item.fragment.Box.Style.VerticalAlign
}

// measureBaselineGroup scans items for baseline-aligned fragments and returns the tallest
// margin-box height and the deepest top-edge baseline (margin + border + padding top)
// found.
//
// Sub and super items belong in the baseline group per CSS 2.1 ss10.8.1 because their
// offsets are expressed relative to the parent's baseline; their margin boxes (including
// the sub/super shift) must therefore grow the line box.
//
// Takes items ([]lineItem) which is the set of line items to scan.
//
// Returns height (float64) which is the tallest baseline-aligned margin-box height.
// Returns maxBaseline (float64) which is the deepest baseline offset.
func measureBaselineGroup(items []lineItem) (height float64, maxBaseline float64) {
	for _, item := range items {
		if item.fragment.Box == nil {
			continue
		}
		va := itemVerticalAlign(item)
		if !belongsInBaselineGroup(va) {
			continue
		}
		h := baselineGroupItemHeight(item, va)
		if h > height {
			height = h
		}
		baseline := item.fragment.Margin.Top + item.fragment.Border.Top + item.fragment.Padding.Top
		if baseline > maxBaseline {
			maxBaseline = baseline
		}
	}
	return height, maxBaseline
}

// belongsInBaselineGroup reports whether items with the given vertical-align participate
// in baseline-group height computation. Baseline, sub, super, text-top, and text-bottom
// all anchor relative to the parent baseline or content area, so they jointly determine
// how tall the line box must grow.
//
// Takes va (VerticalAlignType) which is the alignment to test.
//
// Returns bool which is true when the alignment contributes to the baseline group.
func belongsInBaselineGroup(va VerticalAlignType) bool {
	switch va {
	case VerticalAlignBaseline, VerticalAlignSub, VerticalAlignSuper,
		VerticalAlignTextTop, VerticalAlignTextBottom:
		return true
	default:
		return false
	}
}

// baselineGroupItemHeight returns the effective margin-box height of an item once its
// sub/super shift has been folded in. Plain baseline items return their natural
// margin-box height; sub and super items add the half-XHeight shift so the line box grows
// to contain the shifted glyphs.
//
// Takes item (lineItem) which is the line item being measured.
// Takes va (VerticalAlignType) which is its effective alignment.
//
// Returns float64 which is the height to use for line-box sizing.
func baselineGroupItemHeight(item lineItem, va VerticalAlignType) float64 {
	height := item.fragment.MarginBoxHeight()
	switch va {
	case VerticalAlignSub, VerticalAlignSuper:
		height += item.parentMetrics.XHeight * subSuperOffsetFactor
	default:
	}
	return height
}

// computeBaselineShift determines how far the baseline group must shift down when a
// middle-aligned item is taller than the baseline group. This keeps both groups centred
// relative to each other within the expanded line box (CSS 2.1 ss10.8).
//
// Takes items ([]lineItem) which is the set of line items to scan.
// Takes lineHeight (float64) which is the line box height.
// Takes baselineGroupHeight (float64) which is the tallest baseline-aligned item height.
//
// Returns float64 which is the downward shift for the baseline group.
func computeBaselineShift(items []lineItem, lineHeight float64, baselineGroupHeight float64) float64 {
	baselineShift := 0.0
	for _, item := range items {
		if item.fragment.Box == nil {
			continue
		}
		if itemVerticalAlign(item) != VerticalAlignMiddle {
			continue
		}
		itemHeight := item.fragment.MarginBoxHeight()
		if itemHeight > baselineGroupHeight {
			shift := (lineHeight - baselineGroupHeight) / 2
			if shift > baselineShift {
				baselineShift = shift
			}
		}
	}
	return baselineShift
}

// applyVerticalAlignOffsets adjusts each item's vertical offset according to its
// vertical-align value, using the previously computed baseline shift and maximum
// baseline.
//
// Sub and super lower or raise the box by half the parent's x-height (CSS 2.1 ss10.8.1).
// Text-top and text-bottom align the box edges to the parent's content area, defined by
// the parent's ascent and descent lines.
//
// Takes items ([]lineItem) which is the set of line items to adjust.
// Takes lineHeight (float64) which is the line box height.
// Takes baselineShift (float64) which is the downward shift for baseline-aligned items.
// Takes maxBaseline (float64) which is the deepest baseline offset among baseline-aligned
// items.
func applyVerticalAlignOffsets(items []lineItem, lineHeight float64, baselineShift float64, maxBaseline float64) {
	for _, item := range items {
		if item.fragment.Box == nil {
			continue
		}
		switch itemVerticalAlign(item) {
		case VerticalAlignBaseline:
			baseline := item.fragment.Margin.Top + item.fragment.Border.Top + item.fragment.Padding.Top
			item.fragment.OffsetY += baselineShift + maxBaseline - baseline
		case VerticalAlignTop:
			continue
		case VerticalAlignMiddle:
			itemHeight := item.fragment.MarginBoxHeight()
			item.fragment.OffsetY += (lineHeight - itemHeight) / 2
		case VerticalAlignBottom:
			itemHeight := item.fragment.MarginBoxHeight()
			item.fragment.OffsetY += lineHeight - itemHeight
		case VerticalAlignSub:
			baseline := item.fragment.Margin.Top + item.fragment.Border.Top + item.fragment.Padding.Top
			item.fragment.OffsetY += baselineShift + maxBaseline - baseline + item.parentMetrics.XHeight*subSuperOffsetFactor
		case VerticalAlignSuper:
			baseline := item.fragment.Margin.Top + item.fragment.Border.Top + item.fragment.Padding.Top
			item.fragment.OffsetY += baselineShift + maxBaseline - baseline - item.parentMetrics.XHeight*subSuperOffsetFactor
		case VerticalAlignTextTop:
			item.fragment.OffsetY += baselineShift + maxBaseline - item.parentMetrics.Ascent
		case VerticalAlignTextBottom:
			item.fragment.OffsetY += baselineShift + maxBaseline + item.parentMetrics.Descent - item.fragment.MarginBoxHeight()
		}
	}
}

// applyInlineVerticalAlign adjusts the vertical position of each item according to its
// vertical-align style.
//
// When a middle-aligned item is taller than the baseline group, the baseline group shifts
// down so both groups remain centred relative to each other within the expanded line box
// (CSS 2.1 ss10.8).
//
// Takes items ([]lineItem) which is the set of line items to adjust vertically.
// Takes lineHeight (float64) which is the height of the line box for alignment
// calculation.
func applyInlineVerticalAlign(items []lineItem, lineHeight float64) {
	baselineGroupHeight, maxBaseline := measureBaselineGroup(items)
	baselineShift := computeBaselineShift(items, lineHeight, baselineGroupHeight)
	applyVerticalAlignOffsets(items, lineHeight, baselineShift, maxBaseline)
}

// allowsWrapping reports whether the given white-space mode permits word wrapping.
//
// Takes whiteSpace (WhiteSpaceType) which is the white-space mode to check.
//
// Returns bool which is true if word wrapping is permitted.
func allowsWrapping(whiteSpace WhiteSpaceType) bool {
	switch whiteSpace {
	case WhiteSpaceNowrap, WhiteSpacePre:
		return false
	default:
		return true
	}
}

// preservesWhitespace reports whether the white-space mode preserves whitespace
// characters (including tabs) as-is.
//
// Takes whiteSpace (WhiteSpaceType) which is the mode to check.
//
// Returns bool which is true for pre, pre-wrap, and pre-line modes.
func preservesWhitespace(whiteSpace WhiteSpaceType) bool {
	switch whiteSpace {
	case WhiteSpacePre, WhiteSpacePreWrap, WhiteSpacePreLine:
		return true
	default:
		return false
	}
}

// layoutTextRun measures a standalone text run and returns a Fragment with the measured
// dimensions.
//
// Takes box (*LayoutBox) which is the text run box to measure.
// Takes fontMetrics (FontMetricsPort) which provides text measurement for sizing.
//
// Returns *Fragment with the text run's layout results.
func layoutTextRun(box *LayoutBox, fontMetrics FontMetricsPort) *Fragment {
	box.Text = applyTextTransform(box.Text, box.Style.TextTransform)

	font := FontDescriptor{
		Family: box.Style.FontFamily,
		Weight: box.Style.FontWeight,
		Style:  box.Style.FontStyle,
	}
	fontSize := box.Style.FontSize
	metrics := fontMetrics.GetMetrics(font, fontSize)

	contentWidth := fontMetrics.MeasureText(font, fontSize, box.Text, box.Style.Direction)
	if box.Style.LetterSpacing != 0 {
		contentWidth += box.Style.LetterSpacing * float64(len([]rune(box.Text)))
	}
	box.Glyphs = fontMetrics.ShapeText(font, fontSize, box.Text, box.Style.Direction)
	fontLineHeight := metrics.Ascent + metrics.Descent + metrics.LineGap
	contentHeight := box.Style.LineHeight
	if contentHeight < fontLineHeight {
		contentHeight = fontLineHeight
	}

	halfLeading := (contentHeight - fontLineHeight) / 2
	box.BaselineOffset = halfLeading + metrics.Ascent

	return &Fragment{
		Box:           box,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
	}
}

// splitIntoWords splits text on breakable whitespace boundaries into individual words.
// Non-breaking spaces (U+00A0) are preserved within words because they must not allow
// line breaks.
//
// Takes text (string) which is the text to split.
//
// Returns []string which is the list of words.
func splitIntoWords(text string) []string {
	const nbsp = '\u00A0'
	const placeholder = '\x00'
	replaced := strings.ReplaceAll(text, string(nbsp), string(placeholder))
	words := strings.Fields(replaced)
	for i, w := range words {
		words[i] = strings.ReplaceAll(w, string(placeholder), string(nbsp))
	}
	return words
}
