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

// LayoutPicker chooses the active Layout based on the current terminal
// dimensions and a configured set of breakpoints. A single LayoutPicker is
// owned by the model; Reflow is called when the terminal resizes and Active
// returns the cached layout for rendering.
type LayoutPicker struct {
	// breakpoints lists the configured responsive thresholds.
	breakpoints []Breakpoint

	// layouts is the registry of available layouts keyed by Name.
	layouts map[string]Layout

	// active is the currently-selected layout.
	active Layout

	// overrideTo forces a specific layout name when non-empty.
	overrideTo string

	// currentBP is the breakpoint last matched by Reflow.
	currentBP Breakpoint
}

// NewLayoutPicker creates a picker seeded with the default layouts and
// breakpoints. The picker's active layout is the smallest variant until
// Reflow is called with real dimensions.
//
// Returns *LayoutPicker which is ready to render the single-pane fallback.
func NewLayoutPicker() *LayoutPicker {
	return NewLayoutPickerWith(DefaultBreakpoints, defaultLayouts())
}

// NewLayoutPickerWith creates a picker with custom breakpoints and layouts.
// Used by tests and by callers wanting to substitute layouts for their own
// implementations.
//
// Takes breakpoints ([]Breakpoint) which configures responsive thresholds.
// Takes layouts (map[string]Layout) which is the registry of available
// layouts indexed by Name.
//
// Returns *LayoutPicker which is configured but not yet sized.
func NewLayoutPickerWith(breakpoints []Breakpoint, layouts map[string]Layout) *LayoutPicker {
	if len(breakpoints) == 0 {
		breakpoints = DefaultBreakpoints
	}
	if layouts == nil {
		layouts = defaultLayouts()
	}

	picker := &LayoutPicker{
		breakpoints: breakpoints,
		layouts:     layouts,
	}
	picker.active = picker.layoutByName(LayoutNameSingle)
	picker.currentBP = breakpoints[0]
	return picker
}

// Reflow updates the picker's active layout based on the current terminal
// dimensions. The selected breakpoint and resolved Layout are cached so
// Active can return them without recomputing.
//
// Takes width (int) which is the terminal width.
// Takes height (int) which is the terminal height.
//
// Returns Layout which is the new active layout.
func (p *LayoutPicker) Reflow(width, height int) Layout {
	if p.overrideTo != "" {
		if layout := p.layoutByName(p.overrideTo); layout != nil {
			p.active = layout
			return layout
		}
	}

	bp := PickBreakpoint(p.breakpoints, width, height)
	p.currentBP = bp
	if layout := p.layoutByName(bp.LayoutName); layout != nil {
		p.active = layout
	}
	return p.active
}

// Active returns the currently-selected Layout.
//
// Returns Layout which is the layout to render with; never nil after
// construction.
func (p *LayoutPicker) Active() Layout {
	return p.active
}

// Breakpoint returns the last breakpoint matched by Reflow.
//
// Returns Breakpoint which describes the active responsive band.
func (p *LayoutPicker) Breakpoint() Breakpoint {
	return p.currentBP
}

// Override forces the picker to return a specific layout regardless of
// terminal dimensions. Pass "" to clear the override and resume responsive
// selection.
//
// Takes name (string) which is the layout name to force, or "" to clear.
func (p *LayoutPicker) Override(name string) {
	p.overrideTo = name
}

// layoutByName resolves a layout from the registry, returning nil when the
// name is unknown.
//
// Takes name (string) which is the registered layout identifier.
//
// Returns Layout which is the matched implementation or nil.
func (p *LayoutPicker) layoutByName(name string) Layout {
	return p.layouts[name]
}

// defaultLayouts returns the canonical Layout registry used by the picker
// when no custom set is supplied.
//
// Returns map[string]Layout indexed by Layout.Name().
func defaultLayouts() map[string]Layout {
	return map[string]Layout{
		LayoutNameSingle:      NewSinglePaneLayout(),
		LayoutNameTwoColumn:   NewTwoColumnLayout(),
		LayoutNameThreeColumn: NewThreeColumnLayout(),
	}
}
