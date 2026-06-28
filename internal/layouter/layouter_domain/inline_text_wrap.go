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
	"strings"
)

// wrappedWordState holds mutable state and per-run constants during wrapped text layout.
type wrappedWordState struct {
	// segmentText holds the accumulated text for the current line segment.
	segmentText string

	// child holds the original text run box being wrapped.
	child *LayoutBox

	// font holds the resolved font descriptor for measurement.
	font FontDescriptor

	// segmentStartX holds the horizontal start position of the current segment.
	segmentStartX float64

	// fontSize holds the font size in points.
	fontSize float64

	// spaceWidth holds the measured width of a single space character.
	spaceWidth float64

	// textLineHeight holds the computed line height for the text run.
	textLineHeight float64

	// parentMetrics holds the parent element's font metrics for vertical-align
	// sub/super/text-top/text-bottom calculation.
	parentMetrics FontMetrics

	// direction holds the text direction (LTR or RTL).
	direction DirectionType

	// effectiveVA holds the alignment to apply to each emitted segment.
	effectiveVA VerticalAlignType

	// segmentEmpty indicates whether the current segment has no text yet.
	segmentEmpty bool

	// lastWasSoftHyphen indicates whether the last word ended with a soft hyphen.
	lastWasSoftHyphen bool
}

// layoutWrappedTextRun splits a text run across lines at word boundaries when it exceeds
// the available width. Each line segment gets its own cloned LayoutBox with the segment's
// text and shaped glyphs, so that writeFragmentsToBoxTree assigns correct positions and
// the PDF painter renders each segment independently.
//
// Takes child (*LayoutBox) which is the text run box to lay out.
// Takes font (FontDescriptor) which is the resolved font descriptor for measurement.
// Takes fontSize (float64) which is the font size in points.
// Takes textLineHeight (float64) which is the line height for the text run.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply.
func (c *inlineLayoutContext) layoutWrappedTextRun(
	child *LayoutBox, font FontDescriptor, fontSize, _ float64, textLineHeight float64,
	parentMetrics FontMetrics, effectiveVA VerticalAlignType,
) {
	text := child.Text

	collapsible := !preservesWhitespace(child.Style.WhiteSpace)
	hasLeadingSpace := collapsible && c.cursorX > 0 && strings.HasPrefix(text, spaceChar)
	hasTrailingSpace := collapsible && strings.HasSuffix(text, spaceChar)

	words := prepareWrappedWords(text, child.Style.Hyphens, child.Style.Language)
	if len(words) == 0 {
		return
	}

	direction := child.Style.Direction
	spaceWidth := c.fontMetrics.MeasureText(font, fontSize, spaceChar, direction)
	if child.Style.WordSpacing != 0 {
		spaceWidth += child.Style.WordSpacing
	}

	if hasLeadingSpace {
		c.cursorX += spaceWidth
	}

	state := wrappedWordState{
		child:          child,
		font:           font,
		fontSize:       fontSize,
		spaceWidth:     spaceWidth,
		textLineHeight: textLineHeight,
		parentMetrics:  parentMetrics,
		direction:      direction,
		effectiveVA:    effectiveVA,
		segmentStartX:  c.cursorX,
		segmentEmpty:   true,
	}

	for _, word := range words {
		isHyphenFragment := strings.HasSuffix(word, softHyphen)
		displayWord := strings.ReplaceAll(word, softHyphen, "")
		wordWidth := c.fontMetrics.MeasureText(font, fontSize, displayWord, direction)
		if child.Style.LetterSpacing != 0 {
			wordWidth += child.Style.LetterSpacing * float64(len([]rune(displayWord)))
		}

		c.tryFitWordOnLine(displayWord, wordWidth, isHyphenFragment, &state)
	}

	c.emitFinalWrappedSegment(&state, hasTrailingSpace)
}

// prepareWrappedWords normalises a text run's soft hyphens and splits it into the word
// list that the wrapping loop consumes.
//
// Takes text (string) which is the run's text after whitespace handling.
// Takes hyphens (HyphensType) which selects the hyphenation behaviour.
// Takes language (string) which is the language code for auto-hyphenation.
//
// Returns []string which is the (possibly fragment-expanded) word list.
func prepareWrappedWords(text string, hyphens HyphensType, language string) []string {
	if hyphens == HyphensNone {
		text = strings.ReplaceAll(text, softHyphen, "")
	}

	if hyphens == HyphensAuto {
		text = autoHyphenateText(text, language)
	}

	words := splitIntoWords(text)
	if len(words) == 0 {
		return nil
	}

	if hyphens != HyphensNone {
		words = expandSoftHyphens(words)
	}
	return words
}

// emitFinalWrappedSegment emits the trailing segment left over after the word loop
// completes, appending a collapsible trailing space when the run abuts following inline
// content on the same line. A segment that is empty produces no fragment.
//
// Takes state (*wrappedWordState) which holds the residual segment to emit.
// Takes hasTrailingSpace (bool) which indicates a collapsible trailing space.
func (c *inlineLayoutContext) emitFinalWrappedSegment(state *wrappedWordState, hasTrailingSpace bool) {
	if state.segmentEmpty {
		return
	}
	emitText := state.segmentText

	if hasTrailingSpace {
		emitText += spaceChar
		c.cursorX += state.spaceWidth
	}
	c.emitTextSegment(
		state.child, state.font, state.fontSize, emitText,
		state.segmentStartX, state.textLineHeight,
		verticalAlignInputs{parentMetrics: state.parentMetrics, effective: state.effectiveVA},
	)
}

// tryFitWordOnLine checks whether a word fits on the current line, flushing the line if
// not, and handles overflow-wrap break-word for single oversized words that exceed the
// full available width.
//
// Takes displayWord (string) which is the visible word text.
// Takes wordWidth (float64) which is the measured word width.
// Takes isHyphenFragment (bool) which indicates a soft-hyphen break.
// Takes state (*wrappedWordState) which holds the mutable wrap state.
func (c *inlineLayoutContext) tryFitWordOnLine(
	displayWord string, wordWidth float64, isHyphenFragment bool,
	state *wrappedWordState,
) {
	needsSpace := !state.segmentEmpty && !isHyphenFragment
	requiredWidth := wordWidth
	if needsSpace {
		requiredWidth += state.spaceWidth
	}

	if c.cursorX+requiredWidth > c.effectiveAvailableWidth() && c.cursorX > 0 {
		if !state.segmentEmpty {
			emitText := state.segmentText
			if state.lastWasSoftHyphen {
				emitText += "-"
			}
			c.emitTextSegment(
				state.child, state.font, state.fontSize, emitText,
				state.segmentStartX, state.textLineHeight,
				verticalAlignInputs{parentMetrics: state.parentMetrics, effective: state.effectiveVA},
			)
		}
		c.flushLine()
		state.segmentStartX = 0
		state.segmentText = ""
		state.segmentEmpty = true
		state.lastWasSoftHyphen = false
		needsSpace = false
	}

	if wordWidth > c.effectiveAvailableWidth() && c.cursorX == 0 &&
		state.child.Style.OverflowWrap != OverflowWrapNormal {
		c.layoutCharacterBreakSingleWord(state.child, state.font, state.fontSize, displayWord, state.textLineHeight, state.parentMetrics, state.effectiveVA)
		state.segmentStartX = c.cursorX
		state.segmentText = ""
		state.segmentEmpty = true
		state.lastWasSoftHyphen = false
		return
	}

	if needsSpace {
		c.cursorX += state.spaceWidth
		state.segmentText += spaceChar
	}
	c.cursorX += wordWidth
	state.segmentText += displayWord
	state.segmentEmpty = false
	state.lastWasSoftHyphen = isHyphenFragment
	if state.textLineHeight > c.currentLineHeight {
		c.currentLineHeight = state.textLineHeight
	}
	c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, state.textLineHeight, state.parentMetrics, state.effectiveVA)

	if isHyphenFragment {
		c.handleSoftHyphenBreak(state)
	}
}

// handleSoftHyphenBreak checks whether a visible hyphen fits at the end of the current
// line after a soft-hyphen fragment. If it does not fit, the segment is emitted with a
// trailing hyphen and the line is flushed.
//
// Takes state (*wrappedWordState) which holds the mutable wrap state.
func (c *inlineLayoutContext) handleSoftHyphenBreak(state *wrappedWordState) {
	hyphenWidth := c.fontMetrics.MeasureText(state.font, state.fontSize, "-", state.direction)
	if c.cursorX+hyphenWidth > c.effectiveAvailableWidth() {
		if !state.segmentEmpty {
			c.cursorX += hyphenWidth
			c.emitTextSegment(
				state.child, state.font, state.fontSize, state.segmentText+"-",
				state.segmentStartX, state.textLineHeight,
				verticalAlignInputs{parentMetrics: state.parentMetrics, effective: state.effectiveVA},
			)
		}
		c.flushLine()
		state.segmentStartX = 0
		state.segmentText = ""
		state.segmentEmpty = true
		state.lastWasSoftHyphen = false
	}
}

// autoHyphenateText inserts soft hyphens into each word using the Liang-Knuth algorithm
// for the given language. The existing soft hyphen pipeline then handles the rest.
//
// Takes text (string) which is the text to hyphenate.
// Takes language (string) which is the language code for hyphenation patterns.
//
// Returns string which is the text with soft hyphens inserted.
func autoHyphenateText(text, language string) string {
	h := DefaultRegistry().Get(language)
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = h.InsertSoftHyphens(word)
	}
	return strings.Join(words, spaceChar)
}

// expandSoftHyphens splits words containing soft hyphens into fragments. Each fragment
// except the last retains a trailing soft hyphen marker so the caller knows a visible
// hyphen should appear when the break is taken.
//
// Takes words ([]string) which is the word list to expand.
//
// Returns []string which is the expanded fragment list.
func expandSoftHyphens(words []string) []string {
	var result []string
	for _, word := range words {
		if !strings.Contains(word, softHyphen) {
			result = append(result, word)
			continue
		}
		parts := strings.Split(word, softHyphen)
		for partIndex, part := range parts {
			if part == "" {
				continue
			}
			if partIndex < len(parts)-1 {
				result = append(result, part+softHyphen)
			} else {
				result = append(result, part)
			}
		}
	}
	return result
}

// layoutCharacterBreakTextRun splits a text run at grapheme cluster boundaries when
// word-break: break-all is set.
//
// Each line segment gets its own cloned LayoutBox. The growing string is measured as a
// whole to account for kerning. Grapheme cluster iteration ensures multi-codepoint
// sequences (emoji ZWJ, combining marks) are kept intact.
//
// Takes child (*LayoutBox) which is the text run box to lay out.
// Takes font (FontDescriptor) which is the resolved font descriptor for measurement.
// Takes fontSize (float64) which is the font size in points.
// Takes textLineHeight (float64) which is the line height for the text run.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply.
func (c *inlineLayoutContext) layoutCharacterBreakTextRun(
	child *LayoutBox, font FontDescriptor, fontSize, textLineHeight float64,
	parentMetrics FontMetrics, effectiveVA VerticalAlignType,
) {
	clusters := c.fontMetrics.SplitGraphemeClusters(child.Text)
	c.breakClustersAtBoundaries(child, clusters, font, fontSize, textLineHeight, parentMetrics, effectiveVA)
}

// layoutCharacterBreakSingleWord breaks a single word at grapheme cluster boundaries when
// overflow-wrap: break-word or anywhere is active and the word exceeds the available
// width.
//
// Takes original (*LayoutBox) which is the text run box.
// Takes font (FontDescriptor) which is the font descriptor.
// Takes fontSize (float64) which is the font size.
// Takes word (string) which is the word to break.
// Takes textLineHeight (float64) which is the line height.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply.
func (c *inlineLayoutContext) layoutCharacterBreakSingleWord(
	original *LayoutBox, font FontDescriptor, fontSize float64,
	word string, textLineHeight float64,
	parentMetrics FontMetrics, effectiveVA VerticalAlignType,
) {
	clusters := c.fontMetrics.SplitGraphemeClusters(word)
	c.breakClustersAtBoundaries(original, clusters, font, fontSize, textLineHeight, parentMetrics, effectiveVA)
}

// breakClustersAtBoundaries splits grapheme clusters across lines when they exceed the
// available width. Each line segment gets its own cloned LayoutBox via emitTextSegment.
//
// Takes box (*LayoutBox) which is the original text run box.
// Takes clusters ([]string) which holds the grapheme clusters.
// Takes font (FontDescriptor) which is the resolved font.
// Takes fontSize (float64) which is the font size in points.
// Takes textLineHeight (float64) which is the line height.
// Takes parentMetrics (FontMetrics) which are the parent element's font metrics.
// Takes effectiveVA (VerticalAlignType) which is the alignment to apply.
func (c *inlineLayoutContext) breakClustersAtBoundaries(
	box *LayoutBox, clusters []string, font FontDescriptor, fontSize, textLineHeight float64,
	parentMetrics FontMetrics, effectiveVA VerticalAlignType,
) {
	segmentStart := 0
	segmentStartX := c.cursorX

	direction := box.Style.Direction
	for i := range len(clusters) {
		candidate := strings.Join(clusters[segmentStart:i+1], "")
		candidateWidth := c.fontMetrics.MeasureText(font, fontSize, candidate, direction)

		if segmentStartX+candidateWidth > c.effectiveAvailableWidth() && i > segmentStart {
			segment := strings.Join(clusters[segmentStart:i], "")
			segmentWidth := c.fontMetrics.MeasureText(font, fontSize, segment, direction)
			c.cursorX = segmentStartX + segmentWidth
			if textLineHeight > c.currentLineHeight {
				c.currentLineHeight = textLineHeight
			}
			c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, textLineHeight, parentMetrics, effectiveVA)
			c.emitTextSegment(box, font, fontSize, segment, segmentStartX, textLineHeight, verticalAlignInputs{parentMetrics: parentMetrics, effective: effectiveVA})
			c.flushLine()
			segmentStart = i
			segmentStartX = 0
		}
	}

	if segmentStart < len(clusters) {
		segment := strings.Join(clusters[segmentStart:], "")
		segmentWidth := c.fontMetrics.MeasureText(font, fontSize, segment, direction)
		c.cursorX = segmentStartX + segmentWidth
		if textLineHeight > c.currentLineHeight {
			c.currentLineHeight = textLineHeight
		}
		c.currentLineHeight = extendLineHeightForVerticalAlign(c.currentLineHeight, textLineHeight, parentMetrics, effectiveVA)
		c.emitTextSegment(box, font, fontSize, segment, segmentStartX, textLineHeight, verticalAlignInputs{parentMetrics: parentMetrics, effective: effectiveVA})
	}
}
