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

// SinglePaneLayout renders the focused panel full-width. It is the fallback
// layout used on terminals too narrow for any multi-pane variant.
type SinglePaneLayout struct{}

// NewSinglePaneLayout returns the canonical single-pane layout.
//
// Returns *SinglePaneLayout which is stateless and reusable.
func NewSinglePaneLayout() *SinglePaneLayout {
	return &SinglePaneLayout{}
}

// Name returns the layout's registered identifier.
//
// Returns string equal to LayoutNameSingle.
func (*SinglePaneLayout) Name() string { return LayoutNameSingle }

// MaxPanes reports that the layout renders one pane.
//
// Returns int equal to 1.
func (*SinglePaneLayout) MaxPanes() int { return 1 }

// Allocate returns a single rectangle filling the layout area.
//
// Takes width (int) which is the layout area width.
// Takes height (int) which is the layout area height.
// Takes paneCount (int) which is the candidate pane count; values > 1 are
// truncated to 1.
//
// Returns []PaneRect with at most one rectangle. An empty slice is returned
// when the layout area is too small to render anything.
func (*SinglePaneLayout) Allocate(width, height, paneCount int) []PaneRect {
	if paneCount <= 0 || width < MinSinglePaneWidth || height < MinPaneHeight {
		return nil
	}
	return []PaneRect{{X: 0, Y: 0, Width: width, Height: height}}
}

// Compose returns the body of the single allocated pane.
//
// Takes panes ([]RenderedPane) which holds the pre-rendered pane bodies.
//
// Returns string which is the rendered layout. An empty string is returned
// when there are no panes.
func (*SinglePaneLayout) Compose(panes []RenderedPane, _, _ int) string {
	if len(panes) == 0 {
		return ""
	}
	return panes[0].Body
}
