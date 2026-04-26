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

// Breakpoint expresses a "this terminal is wide and tall enough for layout
// X" rule. The picker walks Breakpoints from largest to smallest and
// returns the first whose minimums the terminal satisfies.
//
// ShowsLeftByDefault and ShowsRightByDefault drive the GroupedView's
// initial column visibility. Per the redesign, the right column collapses
// before the left as the terminal narrows; below the smallest breakpoint
// only the centre column is shown by default and the user toggles either
// side on demand.
type Breakpoint struct {
	// LayoutName names the layout to use when this breakpoint matches.
	LayoutName string

	// MinWidth is the minimum terminal width (cols) required.
	MinWidth int

	// MinHeight is the minimum terminal height (rows) required.
	MinHeight int

	// MaxVisiblePanes is the upper bound on simultaneously visible
	// panes for this breakpoint.
	MaxVisiblePanes int

	// ShowsLeftByDefault selects whether the left column is visible
	// when the layout activates.
	ShowsLeftByDefault bool

	// ShowsRightByDefault selects whether the right column is visible
	// when the layout activates.
	ShowsRightByDefault bool
}

// DefaultBreakpoints describes the standard responsive breakpoints used by
// the TUI. They are listed smallest-first; PickLayout walks them in reverse
// so the widest matching layout wins.
//
// Visibility defaults follow the right-collapses-first rule:
//   - >=160 cols: left + centre + right (all visible)
//   - 100-159 cols: left + centre (right hidden by default; ']' toggles)
//   - <100 cols: centre only (both hidden by default; '[' / ']' toggle)
var DefaultBreakpoints = []Breakpoint{
	{
		MinWidth: 0, MinHeight: 0,
		LayoutName:          LayoutNameSingle,
		MaxVisiblePanes:     1,
		ShowsLeftByDefault:  false,
		ShowsRightByDefault: false,
	},
	{
		MinWidth: 100, MinHeight: 24,
		LayoutName:          LayoutNameTwoColumn,
		MaxVisiblePanes:     2,
		ShowsLeftByDefault:  true,
		ShowsRightByDefault: false,
	},
	{
		MinWidth: 160, MinHeight: 30,
		LayoutName:          LayoutNameThreeColumn,
		MaxVisiblePanes:     3,
		ShowsLeftByDefault:  true,
		ShowsRightByDefault: true,
	},
}

// PickBreakpoint walks breakpoints from largest to smallest and returns the
// first whose MinWidth and MinHeight are satisfied by the supplied terminal
// dimensions.
//
// Takes breakpoints ([]Breakpoint) which is the configured set; passing nil
// uses DefaultBreakpoints.
// Takes width (int) which is the terminal width.
// Takes height (int) which is the terminal height.
//
// Returns Breakpoint which is the matched breakpoint. The smallest
// breakpoint always matches because its minimums are zero, so the function
// is total.
func PickBreakpoint(breakpoints []Breakpoint, width, height int) Breakpoint {
	if len(breakpoints) == 0 {
		breakpoints = DefaultBreakpoints
	}

	for i := len(breakpoints) - 1; i >= 0; i-- {
		bp := breakpoints[i]
		if width >= bp.MinWidth && height >= bp.MinHeight {
			return bp
		}
	}

	return breakpoints[0]
}
