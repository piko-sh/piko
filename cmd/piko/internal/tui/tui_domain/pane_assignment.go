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

// PaneSlot identifies which conceptual position a panel occupies within a
// multi-pane layout.
type PaneSlot int

const (
	// SlotPrimary is the focused panel; always visible.
	SlotPrimary PaneSlot = iota

	// SlotContext sits to the left of Primary in the three-column layout.
	SlotContext

	// SlotDetail sits to the right of Primary in the two- and three-column
	// layouts.
	SlotDetail
)

// PaneAssignment links a Panel to the slot it should occupy. Layouts read
// the slot to decide where in the rectangle each panel renders.
type PaneAssignment struct {
	// Panel is the panel to render in the slot.
	Panel Panel

	// Slot is the conceptual position the panel occupies.
	Slot PaneSlot
}

// PaneAssigner decides which panels appear alongside the focused panel
// when a multi-pane layout is active. The default implementation pairs the
// focused panel with neighbours from the panel list, but services may
// substitute custom assigners that respect domain-specific pairings (for
// example, a registry panel paired with a storage panel).
type PaneAssigner interface {
	// Assign returns up to maxPanes assignments, ordered for the layout's
	// rendering convention (left-to-right). The focused panel always
	// appears, occupying the layout's primary slot.
	Assign(panels []Panel, focusedID string, layoutName string, maxPanes int) []PaneAssignment
}

// DefaultPaneAssigner pairs the focused panel with the previous and next
// panels in tab order. The focused panel takes the primary slot; the
// previous panel takes the context slot in three-column mode; the next
// panel takes the detail slot in two- and three-column modes.
type DefaultPaneAssigner struct {
	// Pairings maps a panel ID to a preferred detail-slot panel ID.
	//
	// When the focused panel has a pairing, that panel takes the detail slot
	// in preference to the next-in-order panel. Used to express domain
	// affinities (e.g. a list panel paired with its detail counterpart).
	Pairings map[string]string
}

// NewDefaultPaneAssigner returns an assigner with no explicit pairings.
//
// Returns *DefaultPaneAssigner ready for use.
func NewDefaultPaneAssigner() *DefaultPaneAssigner {
	return &DefaultPaneAssigner{Pairings: nil}
}

// Assign implements the PaneAssigner contract.
//
// The returned slice is at most maxPanes long and contains the focused panel
// first (primary). Additional panels are filled in based on layout name.
//
// Takes panels ([]Panel) which is the full ordered panel list.
// Takes focusedID (string) which identifies the currently-focused panel.
// Takes layoutName (string) which is the active layout's name.
// Takes maxPanes (int) which is the cap from the breakpoint or layout.
//
// Returns []PaneAssignment ordered for the layout's rendering convention.
// An empty slice is returned when no panels match.
func (a *DefaultPaneAssigner) Assign(panels []Panel, focusedID string, layoutName string, maxPanes int) []PaneAssignment {
	if maxPanes <= 0 || len(panels) == 0 {
		return nil
	}

	focusIdx := max(indexOfPanel(panels, focusedID), 0)

	result := make([]PaneAssignment, 0, maxPanes)

	if maxPanes == 1 || layoutName == LayoutNameSingle {
		result = append(result, PaneAssignment{Panel: panels[focusIdx], Slot: SlotPrimary})
		return result
	}

	if maxPanes == 2 || layoutName == LayoutNameTwoColumn {
		result = append(result, PaneAssignment{Panel: panels[focusIdx], Slot: SlotPrimary})
		if detail := a.detailPanel(panels, focusIdx, focusedID); detail != nil {
			result = append(result, PaneAssignment{Panel: detail, Slot: SlotDetail})
		}
		return result
	}

	if context := a.contextPanel(panels, focusIdx); context != nil {
		result = append(result, PaneAssignment{Panel: context, Slot: SlotContext})
	}
	result = append(result, PaneAssignment{Panel: panels[focusIdx], Slot: SlotPrimary})
	if detail := a.detailPanel(panels, focusIdx, focusedID); detail != nil {
		result = append(result, PaneAssignment{Panel: detail, Slot: SlotDetail})
	}
	return result
}

// detailPanel selects the panel that should occupy the detail slot. A
// configured pairing wins; otherwise the next panel in tab order is used.
//
// Takes panels ([]Panel) which is the full ordered panel list.
// Takes focusIdx (int) which is the focused panel's index.
// Takes focusedID (string) which identifies the focused panel.
//
// Returns Panel which is the detail panel, or nil when none is suitable.
func (a *DefaultPaneAssigner) detailPanel(panels []Panel, focusIdx int, focusedID string) Panel {
	if pair, ok := a.Pairings[focusedID]; ok {
		if index := indexOfPanel(panels, pair); index >= 0 && index != focusIdx {
			return panels[index]
		}
	}
	if focusIdx+1 < len(panels) {
		return panels[focusIdx+1]
	}
	if focusIdx > 0 {
		return panels[focusIdx-1]
	}
	return nil
}

// contextPanel selects the panel that should occupy the context slot in a
// three-column layout. Returns the panel to the left of focus (wrapping if
// needed), and never the same panel as detail or primary.
//
// Takes panels ([]Panel) which is the full ordered panel list.
// Takes focusIdx (int) which is the focused panel's index.
//
// Returns Panel which is the context panel, or nil when none is suitable.
func (*DefaultPaneAssigner) contextPanel(panels []Panel, focusIdx int) Panel {
	if focusIdx-1 >= 0 {
		return panels[focusIdx-1]
	}
	if len(panels) >= ThreeColumnMinPanes {
		return panels[len(panels)-1]
	}
	return nil
}

// indexOfPanel returns the index of the panel with the given ID, or -1 when
// not found.
//
// Takes panels ([]Panel) which is the panel list to search.
// Takes id (string) which is the panel ID to find.
//
// Returns int which is the index, or -1 when missing.
func indexOfPanel(panels []Panel, id string) int {
	for i, p := range panels {
		if p.ID() == id {
			return i
		}
	}
	return -1
}
