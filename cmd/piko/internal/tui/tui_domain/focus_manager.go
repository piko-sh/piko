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

// FocusManager tracks which panel currently has focus and which panels are
// currently visible (i.e. assigned to a layout slot).
//
// The active panel is always considered visible. Tab and 1-9 hotkeys cycle
// through every registered panel; ctrl+w + arrow cycles through only the
// panels currently visible.
type FocusManager struct {
	// visibleIDs is the set of panel IDs currently rendered in the layout.
	visibleIDs map[string]struct{}

	// activeID is the panel currently holding focus.
	activeID string

	// panels is the registered list of all panels in tab order.
	panels []Panel
}

// NewFocusManager creates an empty FocusManager. SetPanels populates the
// list and SetActive picks the initial focus.
//
// Returns *FocusManager which is empty and ready for SetPanels.
func NewFocusManager() *FocusManager {
	return &FocusManager{
		visibleIDs: make(map[string]struct{}),
	}
}

// SetPanels replaces the managed panel list. The active ID is preserved if
// the corresponding panel still exists; otherwise focus reverts to the
// first panel.
//
// Takes panels ([]Panel) which is the new ordered panel list.
func (f *FocusManager) SetPanels(panels []Panel) {
	f.panels = panels
	if f.activeID != "" {
		for _, p := range panels {
			if p.ID() == f.activeID {
				return
			}
		}
		f.activeID = ""
	}
	if len(panels) > 0 && f.activeID == "" {
		f.activeID = panels[0].ID()
	}
}

// SetActive marks the panel with the given ID as focused.
//
// The previously focused panel has SetFocused(false) called on it; the new
// focused panel has SetFocused(true) called on it. Unknown IDs are ignored.
//
// Takes id (string) which is the panel to focus.
//
// Returns bool which is true when focus changed.
func (f *FocusManager) SetActive(id string) bool {
	if id == f.activeID {
		return false
	}
	for _, p := range f.panels {
		if p.ID() == id {
			f.applyFocus(id)
			return true
		}
	}
	return false
}

// ActiveID returns the ID of the currently-focused panel.
//
// Returns string which is the active panel ID, or "" when no panel is
// registered.
func (f *FocusManager) ActiveID() string {
	return f.activeID
}

// MarkVisible records the IDs of panels currently visible in the layout.
// This is invoked after each layout assignment so cycling visible panes
// can ignore hidden panels.
//
// Takes ids ([]string) which lists the visible panel IDs.
func (f *FocusManager) MarkVisible(ids []string) {
	visible := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		visible[id] = struct{}{}
	}
	f.visibleIDs = visible
}

// IsVisible reports whether the panel with the given ID is currently
// rendered.
//
// Takes id (string) which is the panel to check.
//
// Returns bool which is true when the panel is part of the visible set.
func (f *FocusManager) IsVisible(id string) bool {
	_, ok := f.visibleIDs[id]
	return ok
}

// NextPanel returns the ID of the panel after the current one in the
// registered list, wrapping at the end. Cycles through every panel
// regardless of layout visibility, equivalent to Tab.
//
// Returns string which is the next panel ID, or "" when no panels exist.
func (f *FocusManager) NextPanel() string {
	if len(f.panels) == 0 {
		return ""
	}
	index := f.indexOfActive()
	if index < 0 {
		return f.panels[0].ID()
	}
	return f.panels[(index+1)%len(f.panels)].ID()
}

// PrevPanel is the reverse of NextPanel, wrapping from the start back to
// the end.
//
// Returns string which is the previous panel ID, or "" when no panels
// exist.
func (f *FocusManager) PrevPanel() string {
	if len(f.panels) == 0 {
		return ""
	}
	index := f.indexOfActive()
	if index < 0 {
		return f.panels[len(f.panels)-1].ID()
	}
	index--
	if index < 0 {
		index = len(f.panels) - 1
	}
	return f.panels[index].ID()
}

// NextVisible returns the next panel ID that is currently visible in the
// layout.
//
// When only one panel is visible, the active ID is returned. Cycling among
// visible panels is the equivalent of ctrl+w + right.
//
// Returns string which is the next visible panel ID, or "" when none are
// visible.
func (f *FocusManager) NextVisible() string {
	visible := f.orderedVisible()
	if len(visible) == 0 {
		return ""
	}
	for i, id := range visible {
		if id == f.activeID {
			return visible[(i+1)%len(visible)]
		}
	}
	return visible[0]
}

// PrevVisible is the reverse of NextVisible.
//
// Returns string which is the previous visible panel ID.
func (f *FocusManager) PrevVisible() string {
	visible := f.orderedVisible()
	if len(visible) == 0 {
		return ""
	}
	for i, id := range visible {
		if id == f.activeID {
			j := i - 1
			if j < 0 {
				j = len(visible) - 1
			}
			return visible[j]
		}
	}
	return visible[len(visible)-1]
}

// orderedVisible returns the visible panel IDs in registered-panel order.
//
// Returns []string which is the ordered list of visible panel IDs.
func (f *FocusManager) orderedVisible() []string {
	out := make([]string, 0, len(f.visibleIDs))
	for _, p := range f.panels {
		if _, ok := f.visibleIDs[p.ID()]; ok {
			out = append(out, p.ID())
		}
	}
	return out
}

// indexOfActive returns the position of the active panel in the registered
// list, or -1 if the active ID is not registered.
//
// Returns int which is the active panel's index, or -1 when not found.
func (f *FocusManager) indexOfActive() int {
	for i, p := range f.panels {
		if p.ID() == f.activeID {
			return i
		}
	}
	return -1
}

// applyFocus updates the SetFocused state on each panel and records the new
// active ID.
//
// Takes id (string) which is the new active panel ID. Caller has already
// verified the ID exists.
func (f *FocusManager) applyFocus(id string) {
	for _, p := range f.panels {
		p.SetFocused(p.ID() == id)
	}
	f.activeID = id
}
