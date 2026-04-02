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

import "context"

// PageGeometry provides page content-area heights, supporting a
// distinct first-page height for @page :first CSS rules. When
// FirstPageHeight is zero, all pages use DefaultHeight.
//
// Page 0 uses FirstPageHeight (when non-zero) and all subsequent
// pages use DefaultHeight. The cumulative page start positions
// are computed by PageStart, which accounts for the first page
// being a different height.
type PageGeometry struct {
	// DefaultHeight is the content-area height in points for all
	// pages (and the first page when FirstPageHeight is zero).
	DefaultHeight float64

	// FirstPageHeight is the content-area height in points for
	// the first page only. Zero means same as DefaultHeight.
	FirstPageHeight float64
}

// UniformPageGeometry returns a PageGeometry where every page has
// the same content-area height.
//
// Takes height (float64) which specifies the content-area height in
// points for all pages.
//
// Returns PageGeometry which holds the uniform page geometry.
func UniformPageGeometry(height float64) PageGeometry {
	return PageGeometry{DefaultHeight: height}
}

// PageStart returns the cumulative Y coordinate at which the
// given page index begins. Exported for use by position
// extraction in wdk/runtime.
//
// Takes index (int) which specifies the zero-based page number.
//
// Returns float64 which holds the cumulative Y coordinate in points.
func (g PageGeometry) PageStart(index int) float64 {
	if index <= 0 {
		return 0
	}
	return g.heightForPage(0) + float64(index-1)*g.DefaultHeight
}

// heightForPage returns the content-area height for the given
// page index.
//
// Takes index (int) which specifies the zero-based page number.
//
// Returns float64 which holds the content-area height in points.
func (g PageGeometry) heightForPage(index int) float64 {
	if index == 0 && g.FirstPageHeight > 0 {
		return g.FirstPageHeight
	}
	return g.DefaultHeight
}

// pageEnd returns the cumulative Y coordinate at which the given
// page index ends.
//
// Takes index (int) which specifies the zero-based page number.
//
// Returns float64 which holds the cumulative Y coordinate of the
// page bottom in points.
func (g PageGeometry) pageEnd(index int) float64 {
	return g.PageStart(index) + g.heightForPage(index)
}

// pageForY returns the page index that the given effective Y
// coordinate falls on.
//
// Takes y (float64) which specifies the effective Y coordinate
// in points.
//
// Returns int which holds the zero-based page index.
func (g PageGeometry) pageForY(y float64) int {
	if y < 0 {
		return 0
	}
	firstH := g.heightForPage(0)
	if y < firstH {
		return 0
	}
	return 1 + int((y-firstH)/g.DefaultHeight)
}

// Paginate walks the box tree and assigns PageIndex to each box
// based on which page it falls on, accounting for break-before,
// break-after, break-inside, orphans/widows, table header/footer
// repetition, position:fixed cloning, and data-layout-role
// header/footer repetition.
//
// Each box receives a PageIndex (zero-based page number) and a
// PageYOffset (cumulative Y displacement from forced page breaks).
// The position extraction layer uses these to compute
// page-relative coordinates:
//
//	effectiveY = absoluteY + pageYOffset
//	pageRelativeY = effectiveY - PageStart(pageIndex)
//
// Takes root (*LayoutBox) which is the root of the laid-out box
// tree with absolute Y coordinates.
// Takes geometry (PageGeometry) which provides the page
// content-area height(s) in points.
//
// Returns int which is the maximum page index assigned
// (zero-based).
func Paginate(ctx context.Context, root *LayoutBox, geometry PageGeometry) int {
	if geometry.DefaultHeight <= 0 {
		return 0
	}
	state := &paginationState{
		geometry: geometry,
	}

	var headerBox, footerBox *LayoutBox
	for _, child := range root.Children {
		switch layoutRole(child) {
		case "header":
			headerBox = child
		case "footer":
			footerBox = child
		}
	}
	if headerBox != nil {
		state.headerHeight = headerBox.MarginBoxHeight()
	}
	if footerBox != nil {
		state.footerHeight = footerBox.MarginBoxHeight()
	}
	if headerBox != nil || footerBox != nil {
		filtered := make([]*LayoutBox, 0, len(root.Children))
		for _, child := range root.Children {
			if child != headerBox && child != footerBox {
				filtered = append(filtered, child)
			}
		}
		root.Children = filtered
	}

	paginateBox(ctx, root, state)
	cloneFixedElements(root, state)
	cloneLayoutRoleElements(root, state, headerBox, footerBox)

	return state.maxPage
}

// paginationState holds the mutable state accumulated during
// pagination of the box tree.
type paginationState struct {
	// geometry holds the page content-area heights.
	geometry PageGeometry

	// offset holds the cumulative Y displacement from forced
	// page breaks and overflow adjustments.
	offset float64

	// maxPage holds the highest page index assigned so far.
	maxPage int

	// headerHeight holds the margin-box height of the
	// data-layout-role header, or 0 when no header exists.
	headerHeight float64

	// footerHeight holds the margin-box height of the
	// data-layout-role footer, or 0 when no footer exists.
	footerHeight float64
}

// contentStart returns the Y coordinate where content begins on
// the given page, accounting for data-layout-role headers.
//
// Takes page (int) which specifies the zero-based page index.
//
// Returns float64 which holds the Y coordinate of the content
// start in points.
func (s *paginationState) contentStart(page int) float64 {
	return s.geometry.PageStart(page) + s.headerHeight
}

// contentEnd returns the Y coordinate where content ends on the
// given page, accounting for data-layout-role footers.
//
// Takes page (int) which specifies the zero-based page index.
//
// Returns float64 which holds the Y coordinate of the content
// end in points.
func (s *paginationState) contentEnd(page int) float64 {
	return s.geometry.pageEnd(page) - s.footerHeight
}

// advanceToPage advances the offset so that effectiveY lands at
// the content start of the target page.
//
// Takes page (int) which specifies the current page index.
// Takes effectiveY (float64) which specifies the current
// effective Y coordinate.
//
// Returns int which holds the updated page index.
// Returns float64 which holds the updated effective Y coordinate.
func (s *paginationState) advanceToPage(page int, effectiveY float64) (int, float64) {
	page++
	newStart := s.contentStart(page)
	s.offset += newStart - effectiveY
	return page, newStart
}

// trackPage updates maxPage if page exceeds it.
//
// Takes page (int) which specifies the page index to track.
func (s *paginationState) trackPage(page int) {
	if page > s.maxPage {
		s.maxPage = page
	}
}

// paginateBox assigns page indices and Y offsets to a single box
// and recurses into its children.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes box (*LayoutBox) which is the box to paginate.
// Takes state (*paginationState) which holds the accumulated
// pagination state.
func paginateBox(ctx context.Context, box *LayoutBox, state *paginationState) {
	if ctx.Err() != nil {
		return
	}

	effectiveY := box.ContentY + state.offset
	page := max(state.geometry.pageForY(effectiveY), 0)

	page, effectiveY = applyLayoutRoleOverflow(box, state, page, effectiveY)
	page, effectiveY = applyBreakBefore(box, state, page, effectiveY)
	page, effectiveY = applyBreakInsideAvoid(box, state, page, effectiveY)

	box.PageIndex = page
	box.PageYOffset = state.offset
	state.trackPage(page)

	if isTextBlock(box) {
		paginateTextBlock(ctx, box, state)
		return
	}
	if box.Type == BoxTable {
		paginateTable(ctx, box, state)
		return
	}

	for i, child := range box.Children {
		applyChildOverflow(child, state)
		paginateBox(ctx, child, state)
		applyBreakAfter(box, child, i, state)
	}
}

// applyChildOverflow pushes a child box to the next page when
// its margin-box bottom overflows the current page boundary.
// Implements the design doc's splitBox algorithm for block
// containers: split between children by pushing overflowing
// children to the next page.
//
// A child is pushed only when:
//
//	(1) its bottom exceeds the page content end,
//	(2) its top is past the page content start (avoids infinite
//	    loops for boxes taller than a page), and
//	(3) it fits on a fresh page.
//
// Takes child (*LayoutBox) which is the child box to check.
// Takes state (*paginationState) which holds the accumulated
// pagination state.
func applyChildOverflow(child *LayoutBox, state *paginationState) {
	effectiveY := child.ContentY + state.offset
	page := max(state.geometry.pageForY(effectiveY), 0)
	bottom := effectiveY + child.MarginBoxHeight()
	contentEnd := state.contentEnd(page)

	if bottom > contentEnd &&
		effectiveY > state.contentStart(page) &&
		child.MarginBoxHeight() <= state.geometry.heightForPage(page+1) {
		state.advanceToPage(page, effectiveY)
	}
}

// applyLayoutRoleOverflow pushes a box to the next page when
// data-layout-role headers or footers reduce the content area
// and the box overflows into the footer zone.
//
// Takes box (*LayoutBox) which is the box to check.
// Takes state (*paginationState) which holds the pagination state.
// Takes page (int) which specifies the current page index.
// Takes effectiveY (float64) which specifies the effective Y
// coordinate of the box.
//
// Returns int which holds the updated page index.
// Returns float64 which holds the updated effective Y coordinate.
func applyLayoutRoleOverflow(
	box *LayoutBox, state *paginationState, page int, effectiveY float64,
) (int, float64) {
	if state.headerHeight == 0 && state.footerHeight == 0 {
		return page, effectiveY
	}
	boxBottom := effectiveY + box.MarginBoxHeight()
	contentEnd := state.contentEnd(page)
	if boxBottom > contentEnd && effectiveY > state.contentStart(page) {
		return state.advanceToPage(page, effectiveY)
	}
	return page, effectiveY
}

// needsParityAdvance reports whether the page must be advanced to
// satisfy a left/right parity constraint. Even pages (0, 2, 4...)
// are right; odd pages are left.
//
// Takes breakVal (PageBreakType) which specifies the break type
// to check.
// Takes page (int) which specifies the current page index.
//
// Returns bool which indicates whether an extra page advance is
// needed for parity.
func needsParityAdvance(breakVal PageBreakType, page int) bool {
	return (breakVal == PageBreakLeft && page%2 == 0) ||
		(breakVal == PageBreakRight && page%2 != 0)
}

// applyBreakBefore handles break-before: page/always/left/right
// by forcing this box to the next page (or the next page with
// the required parity).
//
// Takes box (*LayoutBox) which is the box whose break-before
// property is checked.
// Takes state (*paginationState) which holds the pagination state.
// Takes page (int) which specifies the current page index.
// Takes effectiveY (float64) which specifies the effective Y
// coordinate of the box.
//
// Returns int which holds the updated page index.
// Returns float64 which holds the updated effective Y coordinate.
func applyBreakBefore(
	box *LayoutBox, state *paginationState, page int, effectiveY float64,
) (int, float64) {
	switch box.Style.PageBreakBefore {
	case PageBreakAlways, PageBreakLeft, PageBreakRight:
	default:
		return page, effectiveY
	}

	ps := state.contentStart(page)
	if effectiveY > ps {
		page, effectiveY = state.advanceToPage(page, effectiveY)
	}

	if needsParityAdvance(box.Style.PageBreakBefore, page) {
		page, effectiveY = state.advanceToPage(page, effectiveY)
	}
	return page, effectiveY
}

// applyBreakInsideAvoid moves the entire box to the next page if
// it would be split across a page boundary and the box fits on a
// single page. If the box is taller than a page, the move is
// skipped and the existing child recursion distributes children
// across pages naturally.
//
// Takes box (*LayoutBox) which is the box whose break-inside
// property is checked.
// Takes state (*paginationState) which holds the pagination state.
// Takes page (int) which specifies the current page index.
// Takes effectiveY (float64) which specifies the effective Y
// coordinate of the box.
//
// Returns int which holds the updated page index.
// Returns float64 which holds the updated effective Y coordinate.
func applyBreakInsideAvoid(
	box *LayoutBox, state *paginationState, page int, effectiveY float64,
) (int, float64) {
	if box.Style.PageBreakInside != PageBreakAvoid {
		return page, effectiveY
	}
	boxBottom := effectiveY + box.MarginBoxHeight()
	contentEnd := state.contentEnd(page)
	if boxBottom > contentEnd && box.MarginBoxHeight() <= state.geometry.heightForPage(page) {
		return state.advanceToPage(page, effectiveY)
	}
	return page, effectiveY
}

// applyBreakAfter handles break-after: page/always/left/right on
// a child by adjusting the offset so the next sibling starts on
// a new page (with the required parity for left/right).
//
// Takes parent (*LayoutBox) which is the parent box containing
// the child.
// Takes child (*LayoutBox) which is the child whose break-after
// property is checked.
// Takes childIdx (int) which specifies the index of the child
// within the parent's children.
// Takes state (*paginationState) which holds the pagination state.
func applyBreakAfter(parent, child *LayoutBox, childIdx int, state *paginationState) {
	switch child.Style.PageBreakAfter {
	case PageBreakAlways, PageBreakLeft, PageBreakRight:
	default:
		return
	}

	childBottom := child.ContentY + state.offset + child.MarginBoxHeight()
	childPage := child.PageIndex
	contentEnd := state.contentEnd(childPage)
	if childBottom < contentEnd && childIdx+1 < len(parent.Children) {
		state.offset += contentEnd - childBottom
	}

	if childIdx+1 >= len(parent.Children) {
		return
	}

	nextEffY := parent.Children[childIdx+1].ContentY + state.offset
	nextPage := max(state.geometry.pageForY(nextEffY), 0)
	if needsParityAdvance(child.Style.PageBreakAfter, nextPage) {
		newStart := state.contentStart(nextPage + 1)
		state.offset += newStart - nextEffY
	}
}

// isTextBlock reports whether a box is a block-level container whose
// children are all text-run lines (ignoring list markers).
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which indicates whether the box is a text block
// subject to orphans and widows constraints.
func isTextBlock(box *LayoutBox) bool {
	if len(box.Children) == 0 {
		return false
	}
	switch box.Type {
	case BoxBlock, BoxAnonymousBlock, BoxListItem:
	default:
		return false
	}
	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		if child.Type != BoxTextRun {
			return false
		}
	}
	return true
}

// collectTextLines separates text-run children from list markers.
// List markers are paginated immediately; text-run lines are
// returned for orphans/widows processing.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes box (*LayoutBox) which is the text block whose children
// are separated.
//
// Returns []*LayoutBox which holds the text-run lines for
// orphans/widows processing.
func collectTextLines(ctx context.Context, box *LayoutBox, state *paginationState) []*LayoutBox {
	var lines []*LayoutBox
	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			paginateBox(ctx, child, state)
			continue
		}
		lines = append(lines, child)
	}
	return lines
}

// findTextSplitIndex returns the index of the first text line
// whose bottom exceeds the page boundary.
//
// Takes lines ([]*LayoutBox) which holds the text-run lines to
// check.
// Takes state (*paginationState) which holds the pagination state.
// Takes contentEnd (float64) which specifies the Y coordinate of
// the page content end.
//
// Returns int which holds the split index, or len(lines) when
// all lines fit on the current page.
func findTextSplitIndex(lines []*LayoutBox, state *paginationState, contentEnd float64) int {
	for i, line := range lines {
		lineBottom := line.ContentY + state.offset + line.MarginBoxHeight()
		if lineBottom > contentEnd {
			return i
		}
	}
	return len(lines)
}

// paginateAllLines paginates every line in the slice.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes lines ([]*LayoutBox) which holds the text-run lines to
// paginate.
func paginateAllLines(ctx context.Context, lines []*LayoutBox, state *paginationState) {
	for _, line := range lines {
		paginateBox(ctx, line, state)
	}
}

// paginateTextBlock handles pagination for a text block (paragraph),
// enforcing orphans and widows constraints.
//
// The algorithm finds the natural split point (first line whose bottom
// exceeds the page boundary), then adjusts the split to satisfy the
// orphans (minimum lines before the break) and widows (minimum lines
// after the break) properties. If both constraints cannot be satisfied
// simultaneously, the entire paragraph is moved to the next page.
//
// Lines before the split are paginated normally. Lines after the split
// are pushed to the next page by adding an offset equal to the gap
// between the split line's position and the next page start.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes box (*LayoutBox) which is the text block to paginate.
func paginateTextBlock(ctx context.Context, box *LayoutBox, state *paginationState) {
	lines := collectTextLines(ctx, box, state)
	if len(lines) == 0 {
		return
	}

	boxTop := box.ContentY + state.offset
	boxBottom := boxTop + box.MarginBoxHeight()
	contentEnd := state.contentEnd(box.PageIndex)

	if boxBottom <= contentEnd {
		paginateAllLines(ctx, lines, state)
		return
	}

	splitIndex := findTextSplitIndex(lines, state, contentEnd)

	if splitIndex == len(lines) {
		paginateAllLines(ctx, lines, state)
		return
	}

	splitIndex = adjustSplitForOrphansWidows(box, state, lines, splitIndex, boxTop)
	paginateSplitLines(ctx, box, state, lines, splitIndex)
}

// adjustSplitForOrphansWidows adjusts the split index to satisfy
// orphans and widows constraints.
//
// Takes box (*LayoutBox) which is the text block whose orphans
// and widows properties are read.
// Takes lines ([]*LayoutBox) which holds the text-run lines.
// Takes splitIndex (int) which specifies the natural split point.
//
// Returns int which holds the adjusted split index, or -1 when
// the entire paragraph must be moved to the next page.
func adjustSplitForOrphansWidows(
	box *LayoutBox, _ *paginationState, lines []*LayoutBox, splitIndex int, _ float64,
) int {
	orphans := box.Style.Orphans
	widows := box.Style.Widows
	total := len(lines)

	if splitIndex < orphans {
		return -1
	}
	if total-splitIndex < widows {
		adjusted := total - widows
		if adjusted < orphans {
			return -1
		}
		return adjusted
	}
	return splitIndex
}

// paginateSplitLines paginates lines across a page break.
//
// When splitIndex is negative, the entire paragraph is moved to the
// next page. Otherwise, lines before the split stay on the current
// page and lines from the split onward are pushed to the next page.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes box (*LayoutBox) which is the parent text block.
// Takes state (*paginationState) which holds the pagination state.
// Takes lines ([]*LayoutBox) which holds the text-run lines to
// paginate.
func paginateSplitLines(
	ctx context.Context, box *LayoutBox, state *paginationState, lines []*LayoutBox, splitIndex int,
) {
	boxTop := box.ContentY + state.offset
	if splitIndex < 0 {
		nextStart := state.contentStart(box.PageIndex + 1)
		state.offset += nextStart - boxTop
		paginateAllLines(ctx, lines, state)
		return
	}

	for i := range splitIndex {
		paginateBox(ctx, lines[i], state)
	}
	if splitIndex < len(lines) {
		splitLineTop := lines[splitIndex].ContentY + state.offset
		nextStart := state.contentStart(box.PageIndex + 1)
		state.offset += nextStart - splitLineTop
		for i := splitIndex; i < len(lines); i++ {
			paginateBox(ctx, lines[i], state)
		}
	}
}

// tableGroups holds the thead and tfoot row groups extracted from
// a table, along with their precomputed heights.
type tableGroups struct {
	// thead holds the first thead row group, or nil when absent.
	thead *LayoutBox

	// tfoot holds the first tfoot row group, or nil when absent.
	tfoot *LayoutBox

	// theadHeight holds the computed height of the thead row group.
	theadHeight float64

	// tfootHeight holds the computed height of the tfoot row group.
	tfootHeight float64

	// tfootRefY is the reference Y for tfoot offset calculations.
	// Uses the first row's ContentY when available, since the row
	// group's ContentY may not reflect its visual position.
	tfootRefY float64
}

// findTableGroups locates the first thead and tfoot row groups in
// the table's children and computes their heights.
//
// Takes box (*LayoutBox) which is the table box to scan.
//
// Returns *tableGroups which holds the extracted row groups, or
// nil when neither thead nor tfoot is present.
func findTableGroups(box *LayoutBox) *tableGroups {
	var tg tableGroups
	for _, child := range box.Children {
		if child.Type != BoxTableRowGroup {
			continue
		}
		if child.Style.Display == DisplayTableHeaderGroup && tg.thead == nil {
			tg.thead = child
		}
		if child.Style.Display == DisplayTableFooterGroup && tg.tfoot == nil {
			tg.tfoot = child
		}
	}
	if tg.thead == nil && tg.tfoot == nil {
		return nil
	}
	if tg.thead != nil {
		tg.theadHeight = rowGroupHeight(tg.thead)
	}
	if tg.tfoot != nil {
		tg.tfootHeight = rowGroupHeight(tg.tfoot)
		tg.tfootRefY = tg.tfoot.ContentY
		if len(tg.tfoot.Children) > 0 {
			tg.tfootRefY = tg.tfoot.Children[0].ContentY
		}
	}
	return &tg
}

// adjustTfootSourceOffset compensates for source-order tfoot
// placement.
//
// When the tfoot appears between thead and tbody in the box tree
// (because HTML source order places tfoot before tbody), the body
// rows' ContentY values are shifted by the tfoot's height. This
// adjusts the offset so body rows are positioned as if the tfoot
// were not visually present.
//
// Takes box (*LayoutBox) which is the table box.
// Takes tg (*tableGroups) which holds the extracted thead/tfoot.
// Takes state (*paginationState) which holds the pagination state.
func adjustTfootSourceOffset(box *LayoutBox, tg *tableGroups, state *paginationState) {
	if tg.tfoot == nil {
		return
	}
	for _, child := range box.Children {
		if child == tg.thead || child == tg.tfoot {
			continue
		}
		if len(child.Children) > 0 && tg.tfoot.ContentY < child.Children[0].ContentY {
			state.offset -= tg.tfootHeight
		}
		break
	}
}

// paginateTable handles pagination for tables, repeating thead row
// groups at the top and tfoot row groups at the bottom of each
// continuation page.
//
// The algorithm iterates over individual rows within each body row
// group. When a row's bottom would overflow the current page (minus
// tfoot space), the row is pushed to the next page. A cloned copy
// of the thead is inserted at the top and a cloned tfoot at the
// bottom of each continuation page. Cloned boxes have their
// SourceNode set to nil to avoid duplicate data-layout-id entries
// in position extraction.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes box (*LayoutBox) which is the table box to paginate.
func paginateTable(ctx context.Context, box *LayoutBox, state *paginationState) {
	tg := findTableGroups(box)
	if tg == nil {
		for _, child := range box.Children {
			paginateBox(ctx, child, state)
		}
		return
	}

	if tg.thead != nil {
		paginateBox(ctx, tg.thead, state)
	}
	adjustTfootSourceOffset(box, tg, state)

	lastPage := box.PageIndex
	var clonedHeaders, clonedFooters []*LayoutBox

	for _, child := range box.Children {
		if child == tg.thead || child == tg.tfoot {
			continue
		}
		assignRowGroupPage(child, state)
		lastPage, clonedHeaders, clonedFooters = paginateTableRows(
			ctx, child, state, tg, lastPage, clonedHeaders, clonedFooters,
		)
	}

	positionOriginalTfoot(tg, state, lastPage)

	box.Children = append(box.Children, clonedHeaders...)
	box.Children = append(box.Children, clonedFooters...)
}

// assignRowGroupPage assigns the page index and offset to a body
// row group based on its effective Y position.
//
// Takes rg (*LayoutBox) which is the row group box to assign.
// Takes state (*paginationState) which holds the pagination state.
func assignRowGroupPage(rg *LayoutBox, state *paginationState) {
	rgEffectiveY := rg.ContentY + state.offset
	rgPage := max(state.geometry.pageForY(rgEffectiveY), 0)
	rg.PageIndex = rgPage
	rg.PageYOffset = state.offset
	state.trackPage(rgPage)
}

// paginateTableRows processes individual rows within a body row
// group, detecting page transitions and cloning thead/tfoot
// groups as needed.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes rg (*LayoutBox) which is the body row group.
// Takes state (*paginationState) which holds the pagination state.
// Takes tg (*tableGroups) which holds the extracted thead/tfoot.
// Takes lastPage (int) which specifies the most recent page index.
// Takes headers ([]*LayoutBox) which holds previously cloned
// thead boxes.
//
// Returns updatedLastPage (int) which holds the updated last page
// index.
// Returns updatedHeaders ([]*LayoutBox) which holds the extended
// cloned thead slice.
// Returns updatedFooters ([]*LayoutBox) which holds the extended
// cloned tfoot slice.
func paginateTableRows(
	ctx context.Context, rg *LayoutBox, state *paginationState, tg *tableGroups,
	lastPage int, headers, footers []*LayoutBox,
) (updatedLastPage int, updatedHeaders []*LayoutBox, updatedFooters []*LayoutBox) {
	for _, row := range rg.Children {
		rowPage := computeRowPage(row, state, tg.tfootHeight)

		if rowPage > lastPage {
			footers = cloneTfootRange(tg, state, lastPage, rowPage, footers)
			headers = cloneTheadRange(tg, state, lastPage, rowPage, headers)

			pageStart := state.contentStart(rowPage)
			desiredY := pageStart + tg.theadHeight
			rowEffectiveY := row.ContentY + state.offset
			state.offset += desiredY - rowEffectiveY
			lastPage = rowPage
		}

		paginateBox(ctx, row, state)
	}
	return lastPage, headers, footers
}

// computeRowPage determines which page a table row belongs to,
// advancing to the next page when the row's bottom would
// overflow the current page (leaving space for tfoot).
//
// Takes row (*LayoutBox) which is the table row box.
// Takes state (*paginationState) which holds the pagination state.
// Takes tfootHeight (float64) which specifies the height reserved
// for the tfoot at the page bottom.
//
// Returns int which holds the page index for the row.
func computeRowPage(row *LayoutBox, state *paginationState, tfootHeight float64) int {
	rowEffectiveY := row.ContentY + state.offset
	rowPage := max(state.geometry.pageForY(rowEffectiveY), 0)

	rowBottom := rowEffectiveY + row.MarginBoxHeight()
	effectiveEnd := state.contentEnd(rowPage) - tfootHeight
	if rowBottom > effectiveEnd {
		rowPage++
	}
	return rowPage
}

// cloneTfootRange creates tfoot clones for pages lastPage..rowPage-1
// (the departing pages).
//
// Takes tg (*tableGroups) which holds the extracted thead/tfoot.
// Takes state (*paginationState) which holds the pagination state.
// Takes lastPage (int) which specifies the start of the page range.
// Takes rowPage (int) which specifies the exclusive end of the
// page range.
// Takes footers ([]*LayoutBox) which holds previously cloned
// tfoot boxes.
//
// Returns []*LayoutBox which holds the extended cloned tfoot slice.
func cloneTfootRange(
	tg *tableGroups, state *paginationState, lastPage, rowPage int, footers []*LayoutBox,
) []*LayoutBox {
	if tg.tfoot == nil {
		return footers
	}
	for p := lastPage; p < rowPage; p++ {
		cloned := cloneBoxSubtree(tg.tfoot)
		footY := state.contentEnd(p) - tg.tfootHeight
		assignPageRecursive(cloned, p, footY-tg.tfootRefY)
		footers = append(footers, cloned)
		state.trackPage(p)
	}
	return footers
}

// cloneTheadRange creates thead clones for pages
// lastPage+1..rowPage (the arriving pages).
//
// Takes tg (*tableGroups) which holds the extracted thead/tfoot.
// Takes state (*paginationState) which holds the pagination state.
// Takes lastPage (int) which specifies the page before the range
// start.
// Takes rowPage (int) which specifies the inclusive end of the
// page range.
// Takes headers ([]*LayoutBox) which holds previously cloned
// thead boxes.
//
// Returns []*LayoutBox which holds the extended cloned thead slice.
func cloneTheadRange(
	tg *tableGroups, state *paginationState, lastPage, rowPage int, headers []*LayoutBox,
) []*LayoutBox {
	if tg.thead == nil {
		return headers
	}
	for p := lastPage + 1; p <= rowPage; p++ {
		cloned := cloneBoxSubtree(tg.thead)
		pageTop := state.contentStart(p)
		assignPageRecursive(cloned, p, pageTop-tg.thead.ContentY)
		headers = append(headers, cloned)
		state.trackPage(p)
	}
	return headers
}

// positionOriginalTfoot places the original tfoot on the last
// page at the bottom of the content area.
//
// Takes tg (*tableGroups) which holds the extracted thead/tfoot.
// Takes state (*paginationState) which holds the pagination state.
// Takes lastPage (int) which specifies the page to place the
// tfoot on.
func positionOriginalTfoot(tg *tableGroups, state *paginationState, lastPage int) {
	if tg.tfoot == nil {
		return
	}
	footY := state.contentEnd(lastPage) - tg.tfootHeight
	assignPageRecursive(tg.tfoot, lastPage, footY-tg.tfootRefY)
	state.trackPage(lastPage)
}

// rowGroupHeight computes the effective height of a table row group.
//
// When the row group's own ContentHeight is zero (which can happen in
// some table layout models because the rows hold the actual dimensions),
// the height is computed by summing the individual row heights. A
// span-based calculation (lastRow bottom minus group top) is avoided
// because row ContentY values are absolute and the group's ContentY may
// not reflect its visual position in the table (e.g. when tfoot appears
// between thead and tbody in source order).
//
// Takes group (*LayoutBox) which is the row group box.
//
// Returns float64 which holds the effective height in points.
func rowGroupHeight(group *LayoutBox) float64 {
	h := group.MarginBoxHeight()
	if h == 0 && len(group.Children) > 0 {
		for _, row := range group.Children {
			h += row.MarginBoxHeight()
		}
	}
	return h
}

// collectFixedBoxes recursively finds all boxes with
// position:fixed that do not have a transform ancestor.
//
// Takes box (*LayoutBox) which is the root of the subtree to
// search.
//
// Returns []*LayoutBox which holds the collected fixed-position
// boxes.
func collectFixedBoxes(box *LayoutBox) []*LayoutBox {
	var result []*LayoutBox
	for _, child := range box.Children {
		if child.Style.Position == PositionFixed && child.TransformAncestor == nil {
			result = append(result, child)
		}
		result = append(result, collectFixedBoxes(child)...)
	}
	return result
}

// cloneFixedElements creates clones of all fixed-position boxes
// for every page they do not already appear on.
//
// Takes root (*LayoutBox) which is the root box of the tree.
// Takes state (*paginationState) which holds the pagination state.
func cloneFixedElements(root *LayoutBox, state *paginationState) {
	fixedBoxes := collectFixedBoxes(root)
	if len(fixedBoxes) == 0 {
		return
	}

	var clones []*LayoutBox
	for _, fb := range fixedBoxes {
		for p := 0; p <= state.maxPage; p++ {
			if p == fb.PageIndex {
				continue
			}
			clone := cloneBoxSubtree(fb)

			cloneOffset := fb.PageYOffset + state.geometry.PageStart(p) - state.geometry.PageStart(fb.PageIndex)
			assignPageRecursive(clone, p, cloneOffset)
			clones = append(clones, clone)
		}
	}
	root.Children = append(root.Children, clones...)
}

// layoutRole returns the value of the data-layout-role attribute
// on the box's source node, or an empty string if no such
// attribute exists.
//
// Takes box (*LayoutBox) which is the box to inspect.
//
// Returns string which holds the attribute value, or empty
// when absent.
func layoutRole(box *LayoutBox) string {
	if box.SourceNode == nil {
		return ""
	}
	for i := range box.SourceNode.Attributes {
		if box.SourceNode.Attributes[i].Name == "data-layout-role" {
			return box.SourceNode.Attributes[i].Value
		}
	}
	return ""
}

// cloneLayoutRoleElements re-inserts the header/footer boxes
// (previously removed from root.Children) back onto page 0 and
// clones them for all continuation pages.
//
// Takes root (*LayoutBox) which is the root box of the tree.
// Takes state (*paginationState) which holds the pagination state.
// Takes headerBox (*LayoutBox) which is the header box, or nil
// when absent.
// Takes footerBox (*LayoutBox) which is the footer box, or nil
// when absent.
func cloneLayoutRoleElements(root *LayoutBox, state *paginationState, headerBox, footerBox *LayoutBox) {
	if headerBox == nil && footerBox == nil {
		return
	}

	var roleClones []*LayoutBox

	if headerBox != nil {
		assignPageRecursive(headerBox, 0, 0)
		root.Children = append(root.Children, headerBox)
	}

	if footerBox != nil {
		footerY := state.geometry.pageEnd(0) - state.footerHeight
		footerOffset := footerY - footerBox.ContentY
		assignPageRecursive(footerBox, 0, footerOffset)
		root.Children = append(root.Children, footerBox)
	}

	for p := 1; p <= state.maxPage; p++ {
		if headerBox != nil {
			clone := cloneBoxSubtree(headerBox)
			headerOffset := state.geometry.PageStart(p) - headerBox.ContentY
			assignPageRecursive(clone, p, headerOffset)
			roleClones = append(roleClones, clone)
		}
		if footerBox != nil {
			clone := cloneBoxSubtree(footerBox)
			footerY := state.geometry.pageEnd(p) - state.footerHeight
			footerOffset := footerY - footerBox.ContentY
			assignPageRecursive(clone, p, footerOffset)
			roleClones = append(roleClones, clone)
		}
	}

	root.Children = append(root.Children, roleClones...)
}

// cloneBoxSubtree deep-clones a LayoutBox and all its descendants.
// SourceNode is set to nil on clones to prevent duplicate
// data-layout-id entries during position extraction.
//
// Takes original (*LayoutBox) which is the box to clone.
//
// Returns *LayoutBox which holds the deep-cloned box tree.
func cloneBoxSubtree(original *LayoutBox) *LayoutBox {
	clone := *original
	clone.SourceNode = nil
	clone.Parent = nil
	if len(original.Children) > 0 {
		clone.Children = make([]*LayoutBox, len(original.Children))
		for i, child := range original.Children {
			clone.Children[i] = cloneBoxSubtree(child)
			clone.Children[i].Parent = &clone
		}
	}
	return &clone
}

// assignPageRecursive assigns the given PageIndex and PageYOffset to
// a box and all its descendants.
//
// Takes box (*LayoutBox) which is the root of the subtree to
// update.
// Takes page (int) which specifies the page index to assign.
// Takes offset (float64) which specifies the PageYOffset to
// assign.
func assignPageRecursive(box *LayoutBox, page int, offset float64) {
	box.PageIndex = page
	box.PageYOffset = offset
	for _, child := range box.Children {
		assignPageRecursive(child, page, offset)
	}
}
