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

package tui_domain

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// MenuColumnDefaultWidth is the default cell width allocated to the
// left-column contextual menu when the breakpoint shows it.
const MenuColumnDefaultWidth = 20

// MenuColumnMinWidth is the lower bound below which the menu column
// folds into a chip-bar above the centre column.
const MenuColumnMinWidth = 16

// DetailColumnMinFraction is the fraction of the residual width
// (after the menu column) reserved for the right detail column when
// it is visible. The remainder goes to centre.
const DetailColumnMinFraction = 0.4

// groupedColumnCount is the maximum number of columns the GroupedView
// composer ever joins horizontally: left menu, centre, right detail.
const groupedColumnCount = 3

// FocusTarget identifies which pane within the GroupedView currently owns
// keyboard focus. Centre is the default; Tab cycles to Detail and back
// when the right column is visible.
type FocusTarget int

const (
	// FocusCentre means the centre pane consumes key presses.
	FocusCentre FocusTarget = iota

	// FocusDetail means the right detail pane consumes key presses.
	FocusDetail

	// FocusMenu means the left menu owns focus (item navigation via
	// j/k in the menu, Enter activates).
	FocusMenu
)

// GroupVisibilityState records the per-group toggle state for left and
// right column visibility. It overrides the breakpoint defaults so the
// user's intent persists across resizes within the same group.
//
// A nil pointer means "use the breakpoint default"; non-nil values
// override.
type GroupVisibilityState struct {
	// LeftOverride, when non-nil, forces the left column visible (true)
	// or hidden (false) regardless of breakpoint default.
	LeftOverride *bool

	// RightOverride, when non-nil, forces the right column visible
	// (true) or hidden (false) regardless of breakpoint default.
	RightOverride *bool
}

// columnVisible resolves an override against a default.
//
// Takes override (*bool) which is the user's per-group override; nil
// means "no override".
// Takes fallback (bool) which is the breakpoint default.
//
// Returns bool which is true when the column should be visible.
func columnVisible(override *bool, fallback bool) bool {
	if override == nil {
		return fallback
	}
	return *override
}

// GroupedViewArgs bundles the inputs to GroupedView.Compose. Bundled into
// a struct because the call site has many independent fields and a
// positional signature would be hard to read.
type GroupedViewArgs struct {
	// Group is the active panel group.
	Group PanelGroup

	// Item is the active menu entry within the group.
	Item MenuItem

	// Visibility carries per-group column overrides.
	Visibility GroupVisibilityState

	// Theme drives styling for frames, dim text, and selected entries.
	Theme *Theme

	// Breakpoint supplies the responsive column-visibility defaults.
	Breakpoint Breakpoint

	// MenuCursor is the cursor row in the contextual menu.
	MenuCursor int

	// Focus identifies which pane currently owns keyboard focus.
	Focus FocusTarget

	// Width is the total composition width in cells.
	Width int

	// Height is the total composition height in rows.
	Height int
}

// GroupedView composes the menu / centre / detail layout for the active
// group. It owns no state; the model passes the active group, item,
// focus, and visibility on each frame.
type GroupedView struct {
	// menu is the contextual menu renderer.
	menu *ContextualMenu
}

// NewGroupedView constructs a GroupedView with the supplied theme.
//
// Takes theme (*Theme) which is the active theme.
//
// Returns *GroupedView ready to call Compose on.
func NewGroupedView(theme *Theme) *GroupedView {
	return &GroupedView{menu: NewContextualMenu(theme)}
}

// SetTheme propagates the theme to the contextual-menu renderer.
//
// Takes theme (*Theme) which is the new theme.
func (v *GroupedView) SetTheme(theme *Theme) {
	v.menu.SetTheme(theme)
}

// Compose renders the menu / centre / detail composition at the
// supplied width and height. The right column collapses before the left
// when the terminal narrows; below the smallest breakpoint only the
// centre is rendered by default.
//
// Takes args (GroupedViewArgs) which bundles the rendering inputs.
//
// Returns string with the composed body. The caller frames the body
// with surrounding chrome (breadcrumb above, status bar below).
func (v *GroupedView) Compose(args GroupedViewArgs) string {
	if args.Width <= 0 || args.Height <= 0 || args.Group == nil || args.Item.Panel == nil {
		return ""
	}

	leftVisible := columnVisible(args.Visibility.LeftOverride, args.Breakpoint.ShowsLeftByDefault)
	rightVisible := columnVisible(args.Visibility.RightOverride, args.Breakpoint.ShowsRightByDefault)

	leftWidth, centreWidth, rightWidth := v.allocate(args.Width, leftVisible, rightVisible)

	parts := make([]string, 0, groupedColumnCount)

	if leftVisible && leftWidth > 0 {
		parts = append(parts, v.renderMenu(args, leftWidth, args.Height))
	}

	parts = append(parts, v.renderCentre(args, centreWidth, args.Height))

	if rightVisible && rightWidth > 0 {
		parts = append(parts, v.renderDetail(args, rightWidth, args.Height))
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// allocate divides width among the visible columns. The menu takes a
// fixed 20-cell column (down to a 16-cell minimum); the right takes
// roughly 40% of the residual width when visible; the centre takes the
// remainder.
//
// Takes width (int) which is the total available width.
// Takes leftVisible, rightVisible (bool) which are the column visibility
// flags.
//
// Returns leftWidth (int) which is the cells allocated to the left
// menu column; 0 when the menu is hidden.
// Returns centreWidth (int) which is the cells allocated to the centre
// pane.
// Returns rightWidth (int) which is the cells allocated to the right
// detail column; 0 when the right column is hidden.
func (*GroupedView) allocate(width int, leftVisible, rightVisible bool) (leftWidth, centreWidth, rightWidth int) {
	if !leftVisible && !rightVisible {
		return 0, width, 0
	}

	left := 0
	if leftVisible {
		left = MenuColumnDefaultWidth
		if width-left < MinPaneWidth {
			left = MenuColumnMinWidth
			if width-left < MinPaneWidth {
				left = 0
				leftVisible = false
			}
		}
	}

	residual := width - left

	right := 0
	if rightVisible {
		right = max(int(float64(residual)*DetailColumnMinFraction), MinPaneWidth)
		if residual-right < MinPaneWidth {
			right = 0
			rightVisible = false
		}
	}

	centre := width - left - right
	if !leftVisible {
		left = 0
	}
	if !rightVisible {
		right = 0
		centre = width - left
	}

	return left, centre, right
}

// renderMenu renders the left contextual-menu column with its frame.
//
// Takes args (GroupedViewArgs) which is the full compose argument set.
// Takes width (int) which is the column cell width.
// Takes height (int) which is the column cell height.
//
// Returns string which is the framed menu column.
func (v *GroupedView) renderMenu(args GroupedViewArgs, width, height int) string {
	innerW := max(1, width-PanelChromeWidth)
	innerH := max(1, height-PanelChromeHeight)
	body := v.menu.Render(args.Group.Items(), args.Item.ID, args.MenuCursor, innerW, innerH)

	return RenderPaneFrame(PaneFrameOpts{
		Theme:   args.Theme,
		Title:   args.Group.Title(),
		Body:    body,
		Width:   width,
		Height:  height,
		Focused: args.Focus == FocusMenu,
	})
}

// renderCentre renders the centre column. Panels render their own
// frames internally (via BasePanel.RenderFrame), so the composer hands
// them the full column dimensions without adding another frame.
//
// Takes args (GroupedViewArgs) which is the full compose argument set.
// Takes width (int) which is the column cell width.
// Takes height (int) which is the column cell height.
//
// Returns string which is the centre body.
func (*GroupedView) renderCentre(args GroupedViewArgs, width, height int) string {
	centre := args.Item.Panel
	if centre == nil {
		return strings.Repeat(" ", width)
	}
	return centre.View(width, height)
}

// renderDetail renders the right detail column. The active panel
// supplies its own DetailView body; panels with no per-row detail
// return "" and the composer falls back to a centred placeholder.
//
// Takes args (GroupedViewArgs) which is the full compose argument set.
// Takes width (int) which is the column cell width.
// Takes height (int) which is the column cell height.
//
// Returns string which is the framed detail column.
func (*GroupedView) renderDetail(args GroupedViewArgs, width, height int) string {
	innerW := max(1, width-PanelChromeWidth)
	innerH := max(1, height-PanelChromeHeight)

	body := strings.Repeat(SingleSpace, innerW)
	if args.Item.Panel != nil {
		if rendered := args.Item.Panel.DetailView(innerW, innerH); rendered != "" {
			body = rendered
		} else {
			body = placeholderDetailBody(args.Theme, innerW, innerH)
		}
	}

	return RenderPaneFrame(PaneFrameOpts{
		Theme:   args.Theme,
		Title:   "Detail",
		Body:    body,
		Width:   width,
		Height:  height,
		Focused: args.Focus == FocusDetail,
	})
}

// placeholderDetailBody renders the "no detail available" placeholder
// the composer uses when the active panel returns "" from DetailView.
//
// Takes theme (*Theme) for the dim style; may be nil during tests.
// Takes width (int) and height (int) which size the placeholder.
//
// Returns string with a centred placeholder body.
func placeholderDetailBody(theme *Theme, width, height int) string {
	hint := "no detail available"
	hintWidth := TextWidth(hint)
	if width < hintWidth+2 {
		hint = "..."
		hintWidth = TextWidth(hint)
	}
	pad := max(0, (width-hintWidth)/2)
	line := strings.Repeat(SingleSpace, pad) + hint
	line = PadRightANSI(line, width)
	if theme != nil {
		line = theme.Dim.Render(line)
	}
	rows := make([]string, height)
	for i := range rows {
		if i == height/2 {
			rows[i] = line
			continue
		}
		rows[i] = strings.Repeat(SingleSpace, width)
	}
	return strings.Join(rows, "\n")
}
