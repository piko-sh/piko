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

import "charm.land/lipgloss/v2"

const (
	// LayoutNameSingle renders the focused panel full-width.
	LayoutNameSingle = "single"

	// LayoutNameTwoColumn renders two panels side-by-side.
	LayoutNameTwoColumn = "two"

	// LayoutNameThreeColumn renders three panels side-by-side.
	LayoutNameThreeColumn = "three"
)

const (
	// MinPaneWidth is the minimum visible width a pane may receive.
	MinPaneWidth = 24

	// MinPaneHeight is the minimum visible height a pane may receive.
	MinPaneHeight = 6

	// MinSinglePaneWidth is the minimum width for the single-pane layout.
	// Below this the user is told the terminal is too narrow.
	MinSinglePaneWidth = 20
)

// PaneRect describes the on-screen rectangle a Layout has allocated for a
// single pane. Coordinates are in terminal cells; the origin is the top-left
// of the layout area (not the screen).
type PaneRect struct {
	// X is the column offset of the pane's left edge within the layout
	// rectangle.
	X int

	// Y is the row offset of the pane's top edge within the layout
	// rectangle.
	Y int

	// Width is the pane's width in columns.
	Width int

	// Height is the pane's height in rows.
	Height int
}

// RenderedPane couples a panel's already-rendered body with metadata the
// Layout needs to compose the final image. Bodies are produced by calling
// each panel's View with the dimensions Allocate returned.
type RenderedPane struct {
	// ID is the panel identifier; layouts use it for diagnostic logging.
	ID string

	// Title is the panel display title; layouts may render it as part of
	// chrome they own (none today, though Phase 4 will).
	Title string

	// Body is the rendered string the panel produced from View(width,
	// height). It is expected to fill exactly the dimensions Allocate
	// returned for this pane.
	Body string

	// Width is the cell width Allocate gave to this pane. Stored on the
	// rendered pane so Compose can verify or pad as needed.
	Width int

	// Height is the cell height Allocate gave to this pane.
	Height int

	// Focused indicates whether this pane is the active focus target.
	// Panels render their own focus borders today, so layouts do not need
	// to read this; future phases (focus rings owned by the layout) will.
	Focused bool
}

// Layout owns the geometry of a multi-pane render. Implementations are
// responsible for both computing the per-pane rectangles (Allocate) and
// stitching the rendered bodies together (Compose).
type Layout interface {
	// Name returns the registered layout identifier.
	Name() string

	// Allocate splits a layout rectangle into per-pane rectangles.
	//
	// Splits the area between header and status bar. The returned slice has
	// length min(paneCount, MaxPanes) where MaxPanes depends on the layout.
	// The supplied paneCount may be larger than the layout supports; surplus
	// panes are dropped.
	Allocate(width, height, paneCount int) []PaneRect

	// Compose stitches rendered pane bodies into the final layout string.
	// The number of panes must equal the number of rectangles Allocate
	// returned for the same dimensions.
	Compose(panes []RenderedPane, width, height int) string

	// MaxPanes is the maximum number of panes this layout renders.
	MaxPanes() int
}

// joinHorizontal concatenates pane bodies side-by-side.
//
// Bodies are expected to be rectangular blocks of text;
// lipgloss.JoinHorizontal handles padding rows that are shorter than the
// tallest body.
//
// Takes panes ([]RenderedPane) which is the ordered list of panes to join.
//
// Returns string which is the concatenated horizontal layout.
func joinHorizontal(panes []RenderedPane) string {
	if len(panes) == 0 {
		return ""
	}
	if len(panes) == 1 {
		return panes[0].Body
	}
	bodies := make([]string, len(panes))
	for i, p := range panes {
		bodies[i] = p.Body
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, bodies...)
}
