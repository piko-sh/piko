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
	// threeColumnContextFraction is the fraction of width allocated to the
	// left context strip in the three-column layout.
	threeColumnContextFraction = 0.22

	// threeColumnPrimaryFraction is the fraction of width allocated to the
	// centre primary pane in the three-column layout.
	threeColumnPrimaryFraction = 0.40
)

// ThreeColumnLayout renders three panes side-by-side. The focused panel
// claims the centre (primary) slot; the PaneAssigner supplies the context
// (left) and detail (right) panes.
type ThreeColumnLayout struct{}

// NewThreeColumnLayout returns the canonical three-column layout.
//
// Returns *ThreeColumnLayout which is stateless and reusable.
func NewThreeColumnLayout() *ThreeColumnLayout {
	return &ThreeColumnLayout{}
}

// Name returns the layout's registered identifier.
//
// Returns string equal to LayoutNameThreeColumn.
func (*ThreeColumnLayout) Name() string { return LayoutNameThreeColumn }

// MaxPanes reports that the layout renders three panes.
//
// Returns int equal to 3.
func (*ThreeColumnLayout) MaxPanes() int { return ThreeColumnMinPanes }

// Allocate splits the layout rectangle into context, primary, and detail
// panes. Falls back to two-column behaviour when there is not room for
// three at their minimum widths, and to single-pane behaviour when there is
// not even room for two.
//
// Takes width (int) which is the layout area width.
// Takes height (int) which is the layout area height.
// Takes paneCount (int) which is the candidate pane count; truncated to 3.
//
// Returns []PaneRect of length 1, 2, or 3 depending on available space.
func (*ThreeColumnLayout) Allocate(width, height, paneCount int) []PaneRect {
	if paneCount <= 0 || height < MinPaneHeight {
		return nil
	}

	if paneCount == 1 || width < 2*MinPaneWidth {
		return []PaneRect{{X: 0, Y: 0, Width: width, Height: height}}
	}

	if paneCount == 2 || width < ThreeColumnMinPanes*MinPaneWidth {
		return NewTwoColumnLayout().Allocate(width, height, 2)
	}

	contextWidth := int(float64(width) * threeColumnContextFraction)
	primaryWidth := int(float64(width) * threeColumnPrimaryFraction)

	if contextWidth < MinPaneWidth {
		contextWidth = MinPaneWidth
	}
	if primaryWidth < MinPaneWidth {
		primaryWidth = MinPaneWidth
	}

	detailWidth := width - contextWidth - primaryWidth
	if detailWidth < MinPaneWidth {
		shortfall := MinPaneWidth - detailWidth
		primaryWidth -= shortfall
		detailWidth = MinPaneWidth
	}
	if primaryWidth < MinPaneWidth {
		shortfall := MinPaneWidth - primaryWidth
		contextWidth -= shortfall
		primaryWidth = MinPaneWidth
	}

	if contextWidth < MinPaneWidth || primaryWidth < MinPaneWidth || detailWidth < MinPaneWidth {
		return NewTwoColumnLayout().Allocate(width, height, 2)
	}

	return []PaneRect{
		{X: 0, Y: 0, Width: contextWidth, Height: height},
		{X: contextWidth, Y: 0, Width: primaryWidth, Height: height},
		{X: contextWidth + primaryWidth, Y: 0, Width: detailWidth, Height: height},
	}
}

// Compose joins the rendered panes horizontally.
//
// Takes panes ([]RenderedPane) which holds the pre-rendered pane bodies.
//
// Returns string which is the rendered layout.
func (*ThreeColumnLayout) Compose(panes []RenderedPane, _, _ int) string {
	return joinHorizontal(panes)
}
