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
	"slices"
	"strings"
)

// isFlexOrGridContainer reports whether a box lays its children out as flex or grid
// items.
//
// A flex item can itself be a flex container (display: inline-flex on a child of a flex
// container), so the box type alone is not enough.
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which is true for a flex or grid container.
func isFlexOrGridContainer(box *LayoutBox) bool {
	switch box.Type { //nolint:exhaustive // only flex and grid containers matter here
	case BoxFlex, BoxGrid:
		return true
	case BoxFlexItem, BoxGridItem:
		switch box.Style.Display { //nolint:exhaustive // only the flex and grid displays matter
		case DisplayFlex, DisplayInlineFlex, DisplayGrid, DisplayInlineGrid:
			return true
		}
	}
	return false
}

// collapsesSpaces reports whether a white-space mode collapses runs of spaces and tabs.
//
// Takes mode (WhiteSpaceType) which is the computed white-space value.
//
// Returns bool which is true for normal, nowrap and pre-line.
func collapsesSpaces(mode WhiteSpaceType) bool {
	return mode == WhiteSpaceNormal || mode == WhiteSpaceNowrap || mode == WhiteSpacePreLine
}

// collapsesSegmentBreaks reports whether a white-space mode turns segment breaks
// (newlines) into collapsible spaces.
//
// Takes mode (WhiteSpaceType) which is the computed white-space value.
//
// Returns bool which is true for normal and nowrap.
func collapsesSegmentBreaks(mode WhiteSpaceType) bool {
	return mode == WhiteSpaceNormal || mode == WhiteSpaceNowrap
}

// isCollapsibleSpace reports whether r is white space that CSS collapses: space, tab,
// line feed, carriage return or form feed. A no-break space is not.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true for collapsible white space.
func isCollapsibleSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	}
	return false
}

// isBlankRun reports whether a text run is collapsible white space alone, which a flex or
// grid container does not render at all.
//
// Takes box (*LayoutBox) which is the text run.
//
// Returns bool which is true for such a run.
func isBlankRun(box *LayoutBox) bool {
	if box.Type != BoxTextRun || !collapsesSegmentBreaks(box.Style.WhiteSpace) {
		return false
	}
	for _, r := range box.Text {
		if !isCollapsibleSpace(r) {
			return false
		}
	}
	return true
}

// wrapFlexGridTextRuns rebuilds a flex or grid container's children so that each
// contiguous run of text children becomes one anonymous item, dropping any run that is
// white space alone.
//
// Takes box (*LayoutBox) which is the flex or grid container.
func wrapFlexGridTextRuns(box *LayoutBox) {
	if len(box.Children) == 0 {
		return
	}
	children := make([]*LayoutBox, 0, len(box.Children))
	start := 0
	flush := func(end int) {
		if end <= start {
			return
		}
		run := box.Children[start:end]
		for _, child := range run {
			if !isBlankRun(child) {
				children = append(children, wrapInAnonymousBlock(append([]*LayoutBox(nil), run...), box))
				return
			}
		}
	}
	for i, child := range box.Children {
		if child.Type == BoxTextRun {
			continue
		}
		flush(i)
		children = append(children, child)
		start = i + 1
	}
	flush(len(box.Children))
	box.Children = children
}

// isInlineLevelChild reports whether a child takes part in its parent's inline formatting
// context: an inline-level box, or a replaced element such as an image or icon whose
// outer display is inline.
//
// Takes child (*LayoutBox) which is the child to check.
//
// Returns bool which is true when it is inline-level.
func isInlineLevelChild(child *LayoutBox) bool {
	if child.Type.IsInlineLevel() {
		return true
	}
	if child.Type != BoxReplaced {
		return false
	}
	switch child.Style.Display { //nolint:exhaustive // only the inline outer displays matter
	case DisplayInline, DisplayInlineBlock, DisplayInlineFlex, DisplayInlineGrid:
		return true
	}
	return false
}

// holdsInlineContext reports whether a box is the root of an inline formatting context:
// it is not itself an inline box taking part in its parent's context, and every child is
// inline-level.
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which is true when the box holds an inline formatting context.
func holdsInlineContext(box *LayoutBox) bool {
	switch box.Type { //nolint:exhaustive // only the boxes inside another context are excluded
	case BoxInline, BoxAnonymousInline, BoxTextRun, BoxReplaced:
		return false
	}
	if len(box.Children) == 0 {
		return false
	}
	for _, child := range box.Children {
		if child.Type != BoxListMarker && !isInlineLevelChild(child) {
			return false
		}
	}
	return true
}

// isInlineBox reports whether a child is an inline box whose content belongs to the same
// inline formatting context as its parent, as opposed to an atomic inline that starts a
// context of its own.
//
// Takes box (*LayoutBox) which is the child.
//
// Returns bool which is true for inline and anonymous inline boxes.
func isInlineBox(box *LayoutBox) bool {
	return box.Type == BoxInline || box.Type == BoxAnonymousInline
}

// collapseWhiteSpace applies CSS Text 3 white-space processing to every inline formatting
// context in a box tree, then removes the text runs and anonymous wrappers that
// processing left empty.
//
// Takes box (*LayoutBox) which is the root of the tree to process.
func collapseWhiteSpace(box *LayoutBox) {
	if holdsInlineContext(box) {
		collapseInlineContext(box)
	}
	for _, child := range box.Children {
		collapseWhiteSpace(child)
	}
	pruneEmptyTextBoxes(box)
}

// collapseInlineContext collapses the white space of one inline formatting context,
// across the inline boxes inside it, and trims the context's two edges.
//
// Takes root (*LayoutBox) which holds the inline formatting context.
func collapseInlineContext(root *LayoutBox) {
	afterSpace := true
	collapseInlineRuns(root, &afterSpace)
	trimInlineContextEnd(root)
}

// collapseInlineRuns walks an inline formatting context in document order, collapsing
// each text run's white space against the run before it.
//
// Takes box (*LayoutBox) which is the context root or an inline box inside it.
// Takes afterSpace (*bool) which records whether the content so far ends in a collapsible
// space; it starts true so the context's leading space goes.
func collapseInlineRuns(box *LayoutBox, afterSpace *bool) {
	for _, child := range box.Children {
		switch {
		case child.Type == BoxTextRun:
			child.Text = collapseRunText(child.Text, child.Style.WhiteSpace, afterSpace)
		case isInlineBox(child):
			collapseInlineRuns(child, afterSpace)
		case child.Type == BoxListMarker:
		default:

			*afterSpace = false
		}
	}
}

// collapseRunText collapses one run's white space according to its white-space mode.
//
// Takes text (string) which is the run's text.
// Takes mode (WhiteSpaceType) which is the run's computed white-space value.
// Takes afterSpace (*bool) which records whether the content before the run ends in a
// collapsible space, and is updated to say whether the run does.
//
// Returns string which is the processed text.
func collapseRunText(text string, mode WhiteSpaceType, afterSpace *bool) string {
	if !collapsesSpaces(mode) {
		if text != "" {
			*afterSpace = false
		}
		return text
	}
	keepBreaks := !collapsesSegmentBreaks(mode)
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		switch {
		case keepBreaks && r == '\n':

			trimmed := strings.TrimRight(out.String(), " ")
			out.Reset()
			out.WriteString(trimmed)
			out.WriteByte('\n')
			*afterSpace = true
		case isCollapsibleSpace(r):
			if !*afterSpace {
				out.WriteByte(' ')
				*afterSpace = true
			}
		default:
			_, _ = out.WriteRune(r)
			*afterSpace = false
		}
	}
	return out.String()
}

// trimInlineContextEnd removes the collapsible spaces at the end of an inline formatting
// context, walking back over runs that end up empty. An atomic inline at the end keeps
// the space before it.
//
// Takes root (*LayoutBox) which holds the inline formatting context.
func trimInlineContextEnd(root *LayoutBox) {
	trimTrailingRuns(root)
}

// trimTrailingRuns trims trailing collapsible spaces from a box's inline content, from
// its last child backwards.
//
// Takes box (*LayoutBox) which is the context root or an inline box inside it.
//
// Returns bool which is true when trimming should continue into earlier content, because
// everything from here to the end is now empty.
func trimTrailingRuns(box *LayoutBox) bool {
	for _, child := range slices.Backward(box.Children) {
		switch {
		case child.Type == BoxTextRun:
			if !collapsesSpaces(child.Style.WhiteSpace) {
				return child.Text == ""
			}
			child.Text = strings.TrimRight(child.Text, " ")
			if child.Text != "" {
				return false
			}
		case isInlineBox(child):
			if !trimTrailingRuns(child) {
				return false
			}
		case child.Type == BoxListMarker:
		default:
			return false
		}
	}
	return true
}

// pruneEmptyTextBoxes removes the children that white-space processing left with nothing
// to render: empty text runs, and anonymous boxes whose children have all gone.
//
// Takes box (*LayoutBox) which is the box whose children are pruned.
func pruneEmptyTextBoxes(box *LayoutBox) {
	kept := box.Children[:0]
	for _, child := range box.Children {
		switch {
		case child.Type == BoxTextRun && child.Text == "":
			continue
		case (child.Type == BoxAnonymousBlock || child.Type == BoxAnonymousInline) && len(child.Children) == 0:
			continue
		}
		kept = append(kept, child)
	}
	for i := len(kept); i < len(box.Children); i++ {
		box.Children[i] = nil
	}
	box.Children = kept
}
