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
	"fmt"
)

// StateSlot identifies which sub-pane within a menu item a state snapshot
// belongs to. The model keeps centre and detail snapshots separately so
// scrolling in one does not displace scrolling in the other.
type StateSlot int

const (
	// StateSlotMenu is the per-group left-column menu cursor. State for
	// this slot lives keyed by group only; the ItemID component of a
	// StateKey is empty when the slot is StateSlotMenu.
	StateSlotMenu StateSlot = iota

	// StateSlotCentre is the centre column of a menu item.
	StateSlotCentre

	// StateSlotDetail is the right column of a menu item.
	StateSlotDetail
)

// String renders the slot for diagnostic and key-encoding use.
//
// Returns string identifier for the slot.
func (s StateSlot) String() string {
	switch s {
	case StateSlotMenu:
		return "menu"
	case StateSlotCentre:
		return "centre"
	case StateSlotDetail:
		return "detail"
	default:
		return "unknown"
	}
}

// StateKey addresses per (group, item, pane) state.
//
// Used by the grouped UI on top of the existing PanelStateStore;
// callers store and reload state under StateKey.String().
type StateKey struct {
	// Group identifies the parent group. Required.
	Group GroupID

	// Item identifies the menu item within the group. Empty only when
	// Pane == StateSlotMenu (in which case the key addresses the group's
	// menu cursor as a whole).
	Item ItemID

	// Pane identifies which sub-pane within the item the state belongs
	// to.
	Pane StateSlot
}

// String produces a stable encoding suitable for use as a flat-key in the
// existing PanelStateStore implementations. The encoding is "group/item/pane"
// with a special "menu" pane that omits the item segment.
//
// Returns string suitable for keying flat-map stores.
func (k StateKey) String() string {
	if k.Pane == StateSlotMenu {
		return fmt.Sprintf("%s/menu", k.Group)
	}
	return fmt.Sprintf("%s/%s/%s", k.Group, k.Item, k.Pane)
}

// MenuKey returns a StateKey addressing the per-group menu cursor.
//
// Takes group (GroupID) which is the parent group.
//
// Returns StateKey with Pane == StateSlotMenu and an empty Item.
func MenuKey(group GroupID) StateKey {
	return StateKey{Group: group, Pane: StateSlotMenu}
}

// CentreKey returns a StateKey addressing the centre pane of an item.
//
// Takes group (GroupID) and item (ItemID) which together identify the
// menu item.
//
// Returns StateKey with Pane == StateSlotCentre.
func CentreKey(group GroupID, item ItemID) StateKey {
	return StateKey{Group: group, Item: item, Pane: StateSlotCentre}
}

// DetailKey returns a StateKey addressing the detail pane of an item.
//
// Takes group (GroupID) and item (ItemID) which together identify the
// menu item.
//
// Returns StateKey with Pane == StateSlotDetail.
func DetailKey(group GroupID, item ItemID) StateKey {
	return StateKey{Group: group, Item: item, Pane: StateSlotDetail}
}
