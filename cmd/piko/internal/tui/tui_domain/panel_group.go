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

// GroupID is a stable identifier for a top-level group tab.
//
// The model uses GroupID to track the active group and to look up persisted
// menu cursor state per group. New groups should declare a constant in this
// file so cross-references are searchable.
type GroupID string

const (
	// GroupContent groups everything related to the artefact pipeline:
	// registry, storage, orchestrator, providers, and dead-letter queues.
	GroupContent GroupID = "content"

	// GroupTelemetry groups observability surfaces: health, metrics,
	// traces, routes, and rate-limit state.
	GroupTelemetry GroupID = "telemetry"

	// GroupRuntime groups host and process state: system, resources,
	// lifecycle, memory, process, build info, and live profiling.
	GroupRuntime GroupID = "runtime"

	// GroupWatchdog groups the watchdog inspector surfaces: overview,
	// events, profiles, history, diagnostic, and config.
	GroupWatchdog GroupID = "watchdog"
)

// ItemID is a stable identifier for a single menu item inside a group. The
// pair (GroupID, ItemID) uniquely identifies a panel in the model's state.
type ItemID string

// Selection describes what is currently selected inside a panel.
//
// Cross-panel coordination (e.g. selecting a span in Traces and reading it
// from Routes) reads this through Panel.Selection. An empty RowID means
// "nothing selected: render an overview".
type Selection struct {
	// RowID identifies the currently-selected row. Empty when nothing is
	// selected.
	RowID string

	// Kind optionally narrows the selection's type.
	//
	// Examples include "task", "workflow", or "metric". Empty when the panel
	// only deals with one kind.
	Kind string
}

// IsEmpty reports whether the selection is the "no row" state.
//
// Returns bool which is true when both RowID and Kind are empty.
func (s Selection) IsEmpty() bool {
	return s.RowID == "" && s.Kind == ""
}

// Badge is a small status indicator rendered next to a menu item label.
//
// Use Count for numeric badges (queue depth, error count) and Severity to
// drive the badge colour. An empty badge (Count == 0 && Severity == 0
// && Glyph == "") suppresses the indicator.
type Badge struct {
	// Glyph is rendered verbatim when set.
	//
	// Takes precedence over Count. Useful for status pips like a filled
	// circle or warning triangle.
	Glyph string

	// Count is rendered as a numeric badge when Glyph is empty and Count
	// is positive.
	Count int

	// Severity drives the badge colour through the active Theme.
	Severity Severity
}

// IsEmpty reports whether the badge would render nothing.
//
// Returns bool which is true when no glyph, no count, and no severity is
// set.
func (b Badge) IsEmpty() bool {
	return b.Glyph == "" && b.Count == 0 && b.Severity == SeverityHealthy
}

// MenuItem is a single entry in a group's left column.
//
// It is a plain struct: the centre is the bound Panel itself, and the
// right-hand detail is rendered via Panel.DetailView so there is no separate
// detail panel to wire.
type MenuItem struct {
	// Panel is the panel rendered when this item is active.
	Panel Panel

	// ID is the stable item identifier used for state look-ups,
	// command-palette navigation, and per-group menu cursor restore.
	ID ItemID

	// Label is the rendered left-column label.
	Label string

	// Hotkey is the per-group keyboard accelerator.
	//
	// Single-key accelerators ("1" through "0") cover the first ten items;
	// "shift+1" through "shift+0" cover items 11-20; further sets can use
	// "ctrl+1" etc. Empty when no hotkey is assigned.
	Hotkey string

	// Badge is the optional status indicator rendered next to the
	// label. Zero value suppresses the indicator.
	Badge Badge
}

// PanelGroup is a top-level tab containing an ordered list of menu
// items. Groups own no rendering of their own; the GroupedView
// renderer composes group + active item + menu cursor each frame.
type PanelGroup interface {
	// ID returns the stable group identifier used for state look-ups
	// and command-palette navigation.
	ID() GroupID

	// Title returns the short human label rendered in the top tab bar.
	Title() string

	// Hotkey returns the rune that activates this group when pressed.
	Hotkey() rune

	// Items returns the ordered list of menu items shown in the left
	// column when this group is active.
	Items() []MenuItem

	// DefaultItemID returns the menu item that should be active when
	// the group is first opened.
	DefaultItemID() ItemID

	// Visible reports whether this group should appear in the tab bar.
	Visible() bool
}
