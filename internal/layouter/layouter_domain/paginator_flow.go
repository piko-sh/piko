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
	"cmp"
	"context"
	"slices"
	"sort"
)

const (
	// pageEpsilon absorbs floating-point error when comparing positions with page edges.
	pageEpsilon = 0.01
)

// pageAt returns the page an effective position falls on and moves the offset so that a
// position inside a data-layout-role header or footer starts below or above it instead.
//
// Takes top (float64) which is an effective position.
//
// Returns int which is the page.
// Returns float64 which is the position after any adjustment.
func (s *paginationState) pageAt(top float64) (int, float64) {
	page := max(s.geometry.pageForY(top), 0)
	if s.footerHeight > 0 && top >= s.contentEnd(page)-pageEpsilon {
		return s.advanceToPage(page, top)
	}
	if s.headerHeight > 0 && top < s.contentStart(page)-pageEpsilon {
		s.offset += s.contentStart(page) - top
		return page, s.contentStart(page)
	}
	return page, top
}

// capacity returns the height of a page's content area.
//
// Takes page (int) which is the page.
//
// Returns float64 which is its content height in points.
func (s *paginationState) capacity(page int) float64 {
	return s.contentEnd(page) - s.contentStart(page)
}

// atPageStart reports whether an effective position is at the start of its page, where
// moving content to the next page gains nothing.
//
// Takes page (int) which is the page.
// Takes top (float64) which is the position.
//
// Returns bool which is true at the page's content start.
func (s *paginationState) atPageStart(page int, top float64) bool {
	return top <= s.contentStart(page)+pageEpsilon
}

// borderBottom returns a box's border-box bottom in layout coordinates. Its bottom margin
// is left out: a margin is truncated where a page breaks.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns float64 which is the bottom edge.
func borderBottom(box *LayoutBox) float64 {
	return box.BorderBoxY() + box.BorderBoxHeight()
}

// isOutOfFlow reports whether a box is absolutely or fixed positioned, so takes no part
// in the flow and never moves the content around it.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns bool which is true for an out-of-flow box.
func isOutOfFlow(box *LayoutBox) bool {
	return box.Style.Position == PositionAbsolute || box.Style.Position == PositionFixed
}

// isMonolithic reports whether a box is never split across pages (CSS Fragmentation 3
// section 4.4): replaced elements, atomic inlines, list markers, table rows and cells,
// scroll containers, transformed, clipped or masked boxes, floats, out-of-flow boxes and
// multi-column containers.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns bool which is true for a monolithic box.
func isMonolithic(box *LayoutBox) bool {
	switch box.Type { //nolint:exhaustive // only the always-monolithic types are listed
	case BoxReplaced, BoxInlineBlock, BoxListMarker, BoxTableRow, BoxTableCell:
		return true
	}
	style := &box.Style
	switch style.Display { //nolint:exhaustive // only atomic inline displays are listed
	case DisplayInlineBlock, DisplayInlineFlex, DisplayInlineGrid:
		return true
	}
	return style.OverflowY == OverflowScroll || style.OverflowY == OverflowAuto ||
		style.HasTransform || style.ClipPath != "" || style.MaskImage != "" ||
		style.Float != FloatNone || isOutOfFlow(box) ||
		style.ColumnCount > 1 || !style.ColumnWidth.IsAuto()
}

// avoidsBreakInside reports whether a box asks not to be split and fits on a page, so is
// kept whole. A box taller than a page breaks regardless.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which supplies the page capacity.
//
// Returns bool which is true when the box is kept whole.
func avoidsBreakInside(box *LayoutBox, state *paginationState) bool {
	return box.Style.PageBreakInside == PageBreakAvoid && box.BorderBoxHeight() <= state.capacity(0)+pageEpsilon
}

// keptWhole reports whether pagination treats a box as one unbreakable piece.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which supplies the page capacity.
//
// Returns bool which is true for a monolithic box or one kept whole by break-inside.
func keptWhole(box *LayoutBox, state *paginationState) bool {
	return isMonolithic(box) || avoidsBreakInside(box, state)
}

// isForcedBreak reports whether a break value forces a page break.
//
// Takes value (PageBreakType) which is the break value.
//
// Returns bool which is true for page, always, left and right.
func isForcedBreak(value PageBreakType) bool {
	return value == PageBreakAlways || value == PageBreakLeft || value == PageBreakRight
}

// strongerBreak combines two break values as CSS Fragmentation 3 section 3.1 does: a
// forced break wins over avoid, which wins over auto.
//
// Takes a (PageBreakType) which is one value.
// Takes b (PageBreakType) which is the other.
//
// Returns PageBreakType which is the stronger.
func strongerBreak(a, b PageBreakType) PageBreakType {
	switch {
	case isForcedBreak(a):
		return a
	case isForcedBreak(b):
		return b
	case a == PageBreakAvoid || b == PageBreakAvoid:
		return PageBreakAvoid
	}
	return PageBreakAuto
}

// firstInFlowChild returns a box's first child that takes part in the flow.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns *LayoutBox which is the child, or nil.
func firstInFlowChild(box *LayoutBox) *LayoutBox {
	for _, child := range box.Children {
		if !isOutOfFlow(child) && child.Type != BoxListMarker {
			return child
		}
	}
	return nil
}

// lastInFlowChild returns a box's last child that takes part in the flow.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns *LayoutBox which is the child, or nil.
func lastInFlowChild(box *LayoutBox) *LayoutBox {
	for _, child := range slices.Backward(box.Children) {
		if !isOutOfFlow(child) && child.Type != BoxListMarker {
			return child
		}
	}
	return nil
}

// effectiveBreakBefore is a box's break-before combined with that of its first child,
// which propagates to the parent's edge (CSS Fragmentation 3 section 3.1).
//
// Takes box (*LayoutBox) which is the box.
//
// Returns PageBreakType which is the combined value.
func effectiveBreakBefore(box *LayoutBox) PageBreakType {
	value := box.Style.PageBreakBefore
	if box.Type == BoxTable || len(inlineLines(box)) > 0 {
		return value
	}
	if first := firstInFlowChild(box); first != nil {
		value = strongerBreak(value, effectiveBreakBefore(first))
	}
	return value
}

// effectiveBreakAfter is a box's break-after combined with that of its last child.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns PageBreakType which is the combined value.
func effectiveBreakAfter(box *LayoutBox) PageBreakType {
	value := box.Style.PageBreakAfter
	if box.Type == BoxTable || len(inlineLines(box)) > 0 {
		return value
	}
	if last := lastInFlowChild(box); last != nil {
		value = strongerBreak(value, effectiveBreakAfter(last))
	}
	return value
}

// avoidsBreakBetween reports whether siblings ask not to be separated by a page break.
//
// Takes previous (*LayoutBox) which is the earlier sibling.
// Takes next (*LayoutBox) which is the later sibling.
//
// Returns bool which is true when break-after or break-before is avoid between them.
func avoidsBreakBetween(previous, next *LayoutBox) bool {
	return strongerBreak(effectiveBreakAfter(previous), effectiveBreakBefore(next)) == PageBreakAvoid
}

// inlineLines returns the line boxes of the inline formatting context a box holds: the
// lines layout recorded, or, for a tree built without layout, one line for each distinct
// row of text-run children.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns []lineRecord which holds its lines, or nil when it holds no inline content.
func inlineLines(box *LayoutBox) []lineRecord {
	if len(box.lines) > 0 {
		return box.lines
	}
	var lines []lineRecord
	for _, child := range box.Children {
		if child.Type == BoxListMarker || isOutOfFlow(child) {
			continue
		}
		if child.Type != BoxTextRun {
			return nil
		}
		top := child.BorderBoxY() - box.ContentY
		if n := len(lines); n > 0 && lines[n-1].top == top {
			lines[n-1].height = max(lines[n-1].height, child.MarginBoxHeight())
			continue
		}
		lines = append(lines, lineRecord{top: top, height: child.MarginBoxHeight()})
	}
	box.lines = lines
	return lines
}

// paginateBox assigns a box and its descendants to pages.
//
// Takes box (*LayoutBox) which is the box to paginate.
// Takes state (*paginationState) which holds the accumulated pagination state.
func paginateBox(ctx context.Context, box *LayoutBox, state *paginationState) {
	if ctx.Err() != nil || state.limits.failed() {
		return
	}

	page, top := 0, box.BorderBoxY()+state.offset

	if box != state.root {
		page, top = state.pageAt(top)
	}
	page, top = applyBreakBefore(box, state, page, top)
	if box != state.bandItem && keptWhole(box, state) {
		page, _ = pushWholeIfCrossing(box, state, page, top)
	}

	box.PageIndex = page
	box.PageYOffset = state.offset
	box.Fragments = nil
	if !state.trackPage(page) {
		return
	}

	switch {
	case isMonolithic(box):
		assignPageRecursive(box, page, state.offset)
	case box.Type == BoxTable:
		paginateTable(ctx, box, state)
	case len(inlineLines(box)) > 0:
		paginateLines(box, state)
	case isBandContainer(box):
		paginateBands(ctx, box, state)
	case firstInFlowChild(box) == nil:
		paginateLeaf(box, state)
		assignOutOfFlowChildren(box, state)
	default:
		paginateChildren(ctx, box, state)
	}
	finishBox(box, state, state.offset)
}

// pushWholeIfCrossing moves a box that must not be split to the next page when it would
// cross the end of this one and fits on a page of its own.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which holds the pagination state.
// Takes page (int) which is the page its top is on.
// Takes top (float64) which is its effective top.
//
// Returns int which is the page it starts on.
// Returns float64 which is its effective top.
func pushWholeIfCrossing(box *LayoutBox, state *paginationState, page int, top float64) (int, float64) {
	height := box.BorderBoxHeight()
	if top+height > state.contentEnd(page)+pageEpsilon && !state.atPageStart(page, top) &&
		height <= state.capacity(page+1)+pageEpsilon {
		return state.advanceToPage(page, top)
	}
	return page, top
}

// paginateChildren paginates a block container's children in order, deciding before each
// whether it must start on the next page.
//
// Takes box (*LayoutBox) which is the container.
// Takes state (*paginationState) which holds the pagination state.
func paginateChildren(ctx context.Context, box *LayoutBox, state *paginationState) {
	for i, child := range box.Children {
		if isOutOfFlow(child) {
			assignOutOfFlow(child, state)
			continue
		}
		if child.Type != BoxListMarker {
			pushBeforeUnbreakablePiece(box, i, state)
		}
		paginateBox(ctx, child, state)
		applyBreakAfter(box, child, i, state)
	}
}

// pushBeforeUnbreakablePiece moves a child to the next page when the first part of it
// that cannot be broken inside, together with any siblings it must not be separated from,
// does not fit on this page but would fit on the next.
//
// Takes parent (*LayoutBox) which is the container.
// Takes index (int) which is the child's index among the container's children.
// Takes state (*paginationState) which holds the pagination state.
func pushBeforeUnbreakablePiece(parent *LayoutBox, index int, state *paginationState) {
	child := parent.Children[index]
	page, top := state.pageAt(child.BorderBoxY() + state.offset)
	if state.atPageStart(page, top) {
		return
	}
	end := unbreakableRunEnd(parent, index, state)
	if end+state.offset > state.contentEnd(page)+pageEpsilon &&
		end-child.BorderBoxY() <= state.capacity(page+1)+pageEpsilon {
		state.advanceToPage(page, top)
	}
}

// unbreakableRunEnd returns where the first unbreakable piece starting at a child ends,
// in layout coordinates.
//
// Takes parent (*LayoutBox) which is the container.
// Takes index (int) which is the child's index.
// Takes state (*paginationState) which holds the pagination state.
//
// Returns float64 which is the end of the run.
//
// The child's own first piece extends over each following sibling it must not be
// separated from, and over the container's bottom edges when the run reaches its last
// child. A run longer than a page gives up its avoid links (CSS Fragmentation 3 section
// 4.4) and is just the child's first piece.
func unbreakableRunEnd(parent *LayoutBox, index int, state *paginationState) float64 {
	children := parent.Children
	start := children[index].BorderBoxY()
	end := firstPieceEnd(children[index], state)
	run := index
	for run+1 < len(children) && !isOutOfFlow(children[run+1]) && avoidsBreakBetween(children[run], children[run+1]) {
		chained := max(end, borderBottom(children[run]), firstPieceEnd(children[run+1], state))
		if chained-start > state.capacity(0)+pageEpsilon {
			break
		}
		end = chained
		run++
	}
	last := children[run]
	if last == lastInFlowChild(parent) && parent.Style.Height.IsAuto() && end >= borderBottom(last)-pageEpsilon &&
		contentBottom(parent) <= borderBottom(last)+last.Margin.Bottom+pageEpsilon {
		end = max(end, borderBottom(parent))
	}
	return end
}

// contentBottom returns where a box's content area ends, in layout coordinates. When it
// is further down than its last child, as for a root box sized to the viewport, the box's
// bottom edges do not travel with that child.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns float64 which is the bottom of its content area.
func contentBottom(box *LayoutBox) float64 {
	return borderBottom(box) - box.Padding.Bottom - box.Border.Bottom
}

// firstPieceEnd returns where the first part of a box that cannot be broken inside ends,
// in layout coordinates. It looks ahead only: nothing is paginated.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which memoises the answers.
//
// Returns float64 which is the end of the box's first piece.
func firstPieceEnd(box *LayoutBox, state *paginationState) float64 {
	if end, ok := state.firstPieces[box]; ok {
		return end
	}
	end := computeFirstPieceEnd(box, state)
	if state.firstPieces == nil {
		state.firstPieces = make(map[*LayoutBox]float64)
	}
	state.firstPieces[box] = end
	return end
}

// computeFirstPieceEnd works out firstPieceEnd for a box.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which holds the pagination state.
//
// Returns float64 which is the end of the box's first piece.
func computeFirstPieceEnd(box *LayoutBox, state *paginationState) float64 {
	switch {
	case keptWhole(box, state):
		return borderBottom(box)
	case box.Type == BoxTable:
		return tableFirstPieceEnd(box)
	case len(inlineLines(box)) > 0:
		return linesFirstPieceEnd(box)
	case isBandContainer(box):
		end := box.BorderBoxY()
		for _, item := range firstBand(box) {
			end = max(end, firstPieceEnd(item, state))
		}
		return end
	}
	first := firstInFlowChild(box)
	if first == nil {
		return box.BorderBoxY() + box.Border.Top + box.Padding.Top
	}
	for i, child := range box.Children {
		if child == first {
			return unbreakableRunEnd(box, i, state)
		}
	}
	return borderBottom(box)
}

// linesFirstPieceEnd returns where the lines a paragraph must keep together at its start
// end: its first orphans lines, or all of them when it is too short to split without
// leaving fewer than widows lines after a break.
//
// Takes box (*LayoutBox) which holds the inline formatting context.
//
// Returns float64 which is the end of the first piece.
func linesFirstPieceEnd(box *LayoutBox) float64 {
	count := len(box.lines)
	keep := min(max(box.Style.Orphans, 1), count)
	if count < box.Style.Orphans+box.Style.Widows {
		keep = count
	}
	last := box.lines[keep-1]
	end := box.ContentY + last.top + last.height
	if keep == count {
		end = max(end, borderBottom(box))
	}
	return end
}

// tableFirstPieceEnd returns where a table's first piece ends: its header group and first
// body row, which must not be separated.
//
// Takes box (*LayoutBox) which is the table.
//
// Returns float64 which is the end of the first piece.
func tableFirstPieceEnd(box *LayoutBox) float64 {
	end := box.BorderBoxY()
	for _, group := range box.Children {
		if group.Style.Display == DisplayTableFooterGroup {
			continue
		}
		for _, row := range group.Children {
			end = max(end, borderBottom(row))
			if group.Style.Display != DisplayTableHeaderGroup {
				return end
			}
		}
	}
	return borderBottom(box)
}

// paginateLines breaks an inline formatting context between its lines, keeping at least
// orphans lines before each break and widows lines after the last, then gives each inline
// descendant the page of the line it sits on.
//
// Takes box (*LayoutBox) which holds the inline formatting context.
// Takes state (*paginationState) which holds the pagination state.
func paginateLines(box *LayoutBox, state *paginationState) {
	count := len(box.lines)
	placement := &linePlacement{
		box:     box,
		state:   state,
		offsets: make([]float64, count),
		pages:   make([]int, count),
	}
	pageFirst := 0
	for i := range count {
		page, top := state.pageAt(placement.top(i))
		crossing := top+box.lines[i].height > state.contentEnd(page)+pageEpsilon && !state.atPageStart(page, top)
		natural := i > pageFirst && page > placement.pages[i-1]
		if crossing || natural {
			split := lineSplit(box, i, pageFirst, count)
			if split < i || crossing {
				page = placement.breakBefore(split, i, page)
			}
			pageFirst = split
		}
		placement.offsets[i] = state.offset
		placement.pages[i] = page
		state.trackPage(page)
	}
	assignLineDescendants(box, box, placement.offsets, placement.pages)
}

// linePlacement records the page and offset each line of an inline formatting context
// lands on while paginateLines walks its lines.
type linePlacement struct {
	// box holds the inline formatting context and its lines.
	box *LayoutBox

	// state holds the pagination state.
	state *paginationState

	// offsets holds the offset each line was placed with.
	offsets []float64

	// pages holds the page each line was placed on.
	pages []int
}

// top returns a line's top in effective coordinates.
//
// Takes line (int) which is the line index.
//
// Returns float64 which is its top at the current offset.
func (p *linePlacement) top(line int) float64 {
	return p.box.ContentY + p.box.lines[line].top + p.state.offset
}

// breakBefore moves the lines from split up to index onto the next page, unless split is
// the box's first line and the box already starts a page, where moving gains nothing.
//
// Takes split (int) which is the first line to move.
// Takes index (int) which is the line being placed.
// Takes page (int) which is the page that line currently falls on.
//
// Returns int which is the page the line falls on afterwards.
func (p *linePlacement) breakBefore(split, index, page int) int {
	if split == 0 && p.state.atPageStart(p.box.PageIndex, p.box.BorderBoxY()+p.box.PageYOffset) {
		return page
	}
	from := page
	if split < index {
		from = p.pages[split]
	}
	next, _ := p.state.advanceToPage(from, p.top(split))
	for k := split; k < index; k++ {
		p.offsets[k] = p.state.offset
		p.pages[k] = next
	}
	page, _ = p.state.pageAt(p.top(index))
	return page
}

// lineSplit chooses the first line to start the next page with when a page ends at line
// index, keeping at least orphans lines of the paragraph before the break and widows
// lines after it.
//
// When both cannot be kept, a paragraph that starts on this page moves to the next page
// whole.
//
// Takes box (*LayoutBox) which holds the paragraph and its orphans and widows values.
// Takes index (int) which is the line the page would naturally break before.
// Takes pageFirst (int) which is the paragraph's first line on this page.
// Takes count (int) which is how many lines the paragraph has.
//
// Returns int which is the line the next page starts with.
func lineSplit(box *LayoutBox, index, pageFirst, count int) int {
	orphans, widows := max(box.Style.Orphans, 1), max(box.Style.Widows, 1)
	split := index
	if count-split < widows && count-widows > pageFirst {
		split = count - widows
	}
	if pageFirst == 0 && split < orphans {
		return 0
	}
	return split
}

// assignLineDescendants gives each descendant of an inline formatting context the page
// and offset of the line containing its vertical middle; an atomic inline takes its whole
// subtree with it.
//
// Takes root (*LayoutBox) which holds the inline formatting context and its lines.
// Takes box (*LayoutBox) which is the root or an inline box inside it.
// Takes offsets ([]float64) which holds each line's offset.
// Takes pages ([]int) which holds each line's page.
func assignLineDescendants(root, box *LayoutBox, offsets []float64, pages []int) {
	for _, child := range box.Children {
		line := lineContaining(root, child.ContentY+child.ContentHeight/2)
		if isOutOfFlow(child) {
			line = lineContaining(root, child.ContentY)
		}
		child.Fragments = nil
		if isInlineBox(child) {
			child.PageIndex = pages[line]
			child.PageYOffset = offsets[line]
			child.LastPageIndex = child.PageIndex
			assignLineDescendants(root, child, offsets, pages)
			for _, grandchild := range child.Children {
				child.LastPageIndex = max(child.LastPageIndex, grandchild.LastPageIndex)
			}
			continue
		}
		assignPageRecursive(child, pages[line], offsets[line])
	}
}

// lineContaining returns the index of the line a layout position falls on.
//
// Takes root (*LayoutBox) which holds the lines.
// Takes y (float64) which is the layout position.
//
// Returns int which is the line index.
func lineContaining(root *LayoutBox, y float64) int {
	relative := y - root.ContentY
	index := sort.Search(len(root.lines), func(i int) bool {
		return root.lines[i].top > relative
	}) - 1
	return min(max(index, 0), len(root.lines)-1)
}

// paginateLeaf handles a splittable box with no in-flow content that crosses a page
// break. With data-layout-role headers or footers, the space they take at the break is
// added, so the rest of the box continues below the next page's header.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which holds the pagination state.
func paginateLeaf(box *LayoutBox, state *paginationState) {
	if state.headerHeight == 0 && state.footerHeight == 0 {
		return
	}
	page := box.PageIndex
	bottom := borderBottom(box) + state.offset
	for bottom > state.contentEnd(page)+pageEpsilon && state.withinPageLimit(page+1) {
		gap := state.contentStart(page+1) - state.contentEnd(page)
		state.offset += gap
		bottom += gap
		page++
	}
}

// assignOutOfFlow places an absolutely or fixed-positioned box, and everything inside it,
// on the page its top falls on, without moving any other content.
//
// Takes box (*LayoutBox) which is the out-of-flow box.
// Takes state (*paginationState) which holds the pagination state.
func assignOutOfFlow(box *LayoutBox, state *paginationState) {
	page := max(state.geometry.pageForY(box.BorderBoxY()+state.offset), 0)
	assignPageRecursive(box, page, state.offset)
	state.trackPage(page)
}

// assignOutOfFlowChildren places the out-of-flow children of a box that has no in-flow
// content.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which holds the pagination state.
func assignOutOfFlowChildren(box *LayoutBox, state *paginationState) {
	for _, child := range box.Children {
		if isOutOfFlow(child) || child.Type == BoxListMarker {
			assignOutOfFlow(child, state)
		}
	}
}

// isBandContainer reports whether a box lays its children out side by side in rows: a
// grid container whose items' rows layout recorded, or a row-direction flex container.
//
// Takes box (*LayoutBox) which is the box.
//
// Returns bool which is true for such a container.
func isBandContainer(box *LayoutBox) bool {
	if isMonolithic(box) {
		return false
	}
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			continue
		}
		return child.gridPlaced || child.flexLine > 0
	}
	return false
}

// bandKey returns the row range an item occupies, for grouping items into bands.
//
// Takes item (*LayoutBox) which is the grid or flex item.
//
// Returns int which is its first row.
// Returns int which is the row after its last.
func bandKey(item *LayoutBox) (start, end int) {
	if item.gridPlaced {
		return item.gridRowStart, max(item.gridRowEnd, item.gridRowStart+1)
	}
	return item.flexLine, item.flexLine + 1
}

// bands groups a container's in-flow items into bands: maximal sets of items whose rows
// overlap, in block order.
//
// Takes box (*LayoutBox) which is the grid or flex container.
//
// Returns [][]*LayoutBox which holds the bands.
func bands(box *LayoutBox) [][]*LayoutBox {
	items := make([]*LayoutBox, 0, len(box.Children))
	for _, child := range box.Children {
		if !isOutOfFlow(child) {
			items = append(items, child)
		}
	}
	slices.SortStableFunc(items, func(a, b *LayoutBox) int {
		aStart, _ := bandKey(a)
		bStart, _ := bandKey(b)
		return cmp.Or(cmp.Compare(aStart, bStart), cmp.Compare(a.BorderBoxY(), b.BorderBoxY()))
	})
	var groups [][]*LayoutBox
	bandEnd := -1
	for _, item := range items {
		start, end := bandKey(item)
		if len(groups) == 0 || start >= bandEnd {
			groups = append(groups, nil)
			bandEnd = end
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], item)
		bandEnd = max(bandEnd, end)
	}
	return groups
}

// firstBand returns a container's first band of items.
//
// Takes box (*LayoutBox) which is the grid or flex container.
//
// Returns []*LayoutBox which holds the first band, or nil.
func firstBand(box *LayoutBox) []*LayoutBox {
	if groups := bands(box); len(groups) > 0 {
		return groups[0]
	}
	return nil
}

// paginateBands paginates a grid or row flex container band by band.
//
// Takes box (*LayoutBox) which is the grid or flex container.
// Takes state (*paginationState) which holds the pagination state.
//
// Every item in a band starts from the band's offset, so the items stay side by side
// while each breaks inside itself; the next band starts below the furthest any item
// reached. An item that fills its band to the bottom (the stretched default) ends where
// the band ends, so its borders and background continue down every page the band covers.
func paginateBands(ctx context.Context, box *LayoutBox, state *paginationState) {
	for _, child := range box.Children {
		if isOutOfFlow(child) {
			assignOutOfFlow(child, state)
		}
	}
	for _, band := range bands(box) {
		paginateBand(ctx, band, placeBand(band, state), state)
	}
}

// placeBand moves a band to the next page when its largest first piece does not fit on
// this page but would on the next, so that its items start on the same page.
//
// Takes band ([]*LayoutBox) which holds the band's items.
// Takes state (*paginationState) which holds the pagination state.
//
// Returns float64 which is the band's bottom in layout coordinates.
func placeBand(band []*LayoutBox, state *paginationState) float64 {
	bandTop, bandBottom := band[0].BorderBoxY(), borderBottom(band[0])
	piece := 0.0
	for _, item := range band {
		bandTop = min(bandTop, item.BorderBoxY())
		bandBottom = max(bandBottom, borderBottom(item))
		piece = max(piece, firstPieceEnd(item, state))
	}
	page, top := state.pageAt(bandTop + state.offset)
	if piece+state.offset > state.contentEnd(page)+pageEpsilon && !state.atPageStart(page, top) &&
		piece-bandTop <= state.capacity(page+1)+pageEpsilon {
		state.advanceToPage(page, top)
	}
	return bandBottom
}

// paginateBand paginates each item of a band from the band's offset and starts what
// follows below the furthest any item reached; an item stretched to the band's bottom
// ends where the band ends.
//
// Takes band ([]*LayoutBox) which holds the band's items.
// Takes bandBottom (float64) which is the band's bottom in layout coordinates.
// Takes state (*paginationState) which holds the pagination state.
func paginateBand(ctx context.Context, band []*LayoutBox, bandBottom float64, state *paginationState) {
	start := state.offset
	ends := make([]float64, len(band))
	furthest := bandBottom + start
	outer := state.bandItem
	for i, item := range band {
		state.offset = start
		state.bandItem = item
		paginateBox(ctx, item, state)
		ends[i] = state.offset
		furthest = max(furthest, borderBottom(item)+ends[i])
	}
	state.bandItem = outer
	state.offset = max(start, furthest-bandBottom)
	for i, item := range band {
		if borderBottom(item) >= bandBottom-pageEpsilon && ends[i] < state.offset {
			finishBox(item, state, state.offset)
		}
	}
}

// finishBox records the pages a paginated box covers: its last page, and the slice of it
// on each page when it spans more than one.
//
// Takes box (*LayoutBox) which is the box.
// Takes state (*paginationState) which holds the pagination state.
// Takes endOffset (float64) which is the offset at the box's bottom edge.
func finishBox(box *LayoutBox, state *paginationState, endOffset float64) {
	last := box.PageIndex
	for _, child := range box.Children {
		last = max(last, child.LastPageIndex)
	}
	top := box.BorderBoxY() + box.PageYOffset
	bottom := borderBottom(box) + endOffset
	if bottom > top {
		last = max(last, max(state.geometry.pageForY(bottom-pageEpsilon), 0))
	}
	box.LastPageIndex = last
	box.Fragments = nil
	state.trackPage(last)
	if box == state.root || isMonolithic(box) || last <= box.PageIndex || !state.withinPageLimit(last) {
		if isMonolithic(box) {
			box.LastPageIndex = box.PageIndex
		}
		return
	}
	box.Fragments = make([]BoxFragment, 0, last-box.PageIndex+1)
	for page := box.PageIndex; page <= last; page++ {
		slice := BoxFragment{PageIndex: page, Top: state.contentStart(page), Bottom: state.contentEnd(page)}
		if page == box.PageIndex {
			slice.Top = top
		}
		if page == last {
			slice.Bottom = min(bottom, slice.Bottom)
		}
		box.Fragments = append(box.Fragments, slice)
	}
}
