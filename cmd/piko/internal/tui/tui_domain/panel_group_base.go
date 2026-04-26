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

// BasePanelGroup is the shared implementation embedded by every concrete
// PanelGroup. It holds the identifier, title, hotkey, and items, and
// supplies default Items / DefaultItemID / Visible behaviour.
//
// Concrete groups embed BasePanelGroup and only override behaviour that
// genuinely diverges. The four built-in groups all use the defaults.
type BasePanelGroup struct {
	// id uniquely identifies the group within a model.
	id GroupID

	// title is the rendered tab label.
	title string

	// items is the ordered menu of panels exposed by the group.
	items []MenuItem

	// hotkey is the keyboard accelerator that activates the group.
	hotkey rune
}

// NewBasePanelGroup constructs a BasePanelGroup. Concrete groups embed
// it via composition.
//
// Takes id (GroupID) which uniquely identifies the group.
// Takes title (string) which is the tab label.
// Takes hotkey (rune) which is the keyboard accelerator.
// Takes items ([]MenuItem) which is the ordered item list.
//
// Returns BasePanelGroup ready for embedding.
func NewBasePanelGroup(id GroupID, title string, hotkey rune, items []MenuItem) BasePanelGroup {
	return BasePanelGroup{id: id, title: title, hotkey: hotkey, items: items}
}

// ID implements PanelGroup.
//
// Returns GroupID which is the group's stable identifier.
func (g *BasePanelGroup) ID() GroupID { return g.id }

// Title implements PanelGroup.
//
// Returns string which is the rendered tab label.
func (g *BasePanelGroup) Title() string { return g.title }

// Hotkey implements PanelGroup.
//
// Returns rune which is the keyboard accelerator that activates the group.
func (g *BasePanelGroup) Hotkey() rune { return g.hotkey }

// Items implements PanelGroup.
//
// Returns []MenuItem which is the ordered menu of panels in the group.
func (g *BasePanelGroup) Items() []MenuItem { return g.items }

// DefaultItemID implements PanelGroup.
//
// Returns ItemID which is the first item's ID, or an empty ItemID when
// the group is unpopulated.
func (g *BasePanelGroup) DefaultItemID() ItemID {
	if len(g.items) == 0 {
		return ""
	}
	return g.items[0].ID
}

// Visible implements PanelGroup. Hidden when no items are configured.
//
// Returns bool which is true when the group has at least one item.
func (g *BasePanelGroup) Visible() bool { return len(g.items) > 0 }

// MenuItemSpec describes one menu entry in a group. Specs whose Panel
// is nil are filtered out by buildGroup so groups can declare every
// possible item up-front and let the service drop the ones whose
// providers are not configured.
type MenuItemSpec struct {
	// Panel is the panel rendered when this item is active. Specs
	// with a nil Panel are dropped.
	Panel Panel

	// ID is the menu item identifier.
	ID ItemID

	// Label is the rendered left-column label.
	Label string

	// Hotkey is the keyboard accelerator string (see MenuItem.Hotkey
	// for the supported syntax: "1"-"0", "shift+1"-"shift+0", ...).
	Hotkey string
}

// collectMenuItems materialises a non-nil-Panel subset of specs into
// MenuItem values, preserving order.
//
// Takes specs ([]MenuItemSpec) which is the ordered candidate list.
//
// Returns []MenuItem with one entry per non-nil spec.
func collectMenuItems(specs []MenuItemSpec) []MenuItem {
	items := make([]MenuItem, 0, len(specs))
	for _, s := range specs {
		if s.Panel == nil {
			continue
		}
		items = append(items, MenuItem{
			ID:     s.ID,
			Label:  s.Label,
			Hotkey: s.Hotkey,
			Panel:  s.Panel,
		})
	}
	return items
}

// buildGroup is the shared body of every NewXGroup constructor:
// filter nil panels and return a BasePanelGroup wrapping the rest.
//
// Takes id (GroupID), title (string), hotkey (rune) which configure
// the surrounding tab.
// Takes specs ([]MenuItemSpec) which is the ordered candidate menu
// list; nil-Panel specs are silently dropped.
//
// Returns PanelGroup ready for registration with the model.
func buildGroup(id GroupID, title string, hotkey rune, specs []MenuItemSpec) PanelGroup {
	pg := NewBasePanelGroup(id, title, hotkey, collectMenuItems(specs))
	return &pg
}
