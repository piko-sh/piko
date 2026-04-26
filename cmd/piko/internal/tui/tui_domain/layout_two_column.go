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

const (
	// twoColumnPrimaryFraction is the fraction of width allocated to the
	// primary pane in the two-column layout.
	twoColumnPrimaryFraction = 0.62
)

// TwoColumnLayout renders two panes side-by-side. The focused panel is
// allocated to the primary slot; an associated panel (chosen by the
// PaneAssigner) takes the detail slot.
type TwoColumnLayout struct{}

// NewTwoColumnLayout returns the canonical two-column layout.
//
// Returns *TwoColumnLayout which is stateless and reusable.
func NewTwoColumnLayout() *TwoColumnLayout {
	return &TwoColumnLayout{}
}

// Name returns the layout's registered identifier.
//
// Returns string equal to LayoutNameTwoColumn.
func (*TwoColumnLayout) Name() string { return LayoutNameTwoColumn }

// MaxPanes reports that the layout renders two panes.
//
// Returns int equal to 2.
func (*TwoColumnLayout) MaxPanes() int { return 2 }

// Allocate splits the layout rectangle into a primary and detail pane.
//
// When the rectangle is too narrow to fit two panes at their minimum width the
// layout falls back to a single full-width pane, so callers always receive
// at least one rectangle when there is room for any content.
//
// Takes width (int) which is the layout area width.
// Takes height (int) which is the layout area height.
// Takes paneCount (int) which is the candidate pane count; truncated to 2.
//
// Returns []PaneRect of length 1 or 2 depending on available space.
func (*TwoColumnLayout) Allocate(width, height, paneCount int) []PaneRect {
	if paneCount <= 0 || height < MinPaneHeight {
		return nil
	}

	if paneCount == 1 || width < 2*MinPaneWidth {
		return []PaneRect{{X: 0, Y: 0, Width: width, Height: height}}
	}

	primaryWidth := min(max(int(float64(width)*twoColumnPrimaryFraction), MinPaneWidth), width-MinPaneWidth)

	detailWidth := width - primaryWidth

	return []PaneRect{
		{X: 0, Y: 0, Width: primaryWidth, Height: height},
		{X: primaryWidth, Y: 0, Width: detailWidth, Height: height},
	}
}

// Compose joins the rendered panes horizontally.
//
// Takes panes ([]RenderedPane) which holds the pre-rendered pane bodies.
//
// Returns string which is the rendered layout.
func (*TwoColumnLayout) Compose(panes []RenderedPane, _, _ int) string {
	return joinHorizontal(panes)
}
