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

// GlobalKeyBindings returns the key bindings that work across all panels.
//
// Returns []KeyBinding which contains the global keyboard shortcuts.
func GlobalKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "q", Description: "Quit"},
		{Key: "Ctrl+C", Description: "Quit"},
		{Key: "?", Description: "Toggle help"},
		{Key: "Tab", Description: "Next panel"},
		{Key: "Shift+Tab", Description: "Previous panel"},
		{Key: "1-9", Description: "Jump to panel"},
		{Key: "r", Description: "Force refresh"},
	}
}

// NavigationKeyBindings returns key bindings for list and tree navigation.
//
// Returns []KeyBinding which contains the standard navigation key mappings.
func NavigationKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j / Down", Description: "Move down"},
		{Key: "k / Up", Description: "Move up"},
		{Key: "g", Description: "Go to top"},
		{Key: "G", Description: "Go to bottom"},
		{Key: keyEnter, Description: "Expand / Select"},
		{Key: "Esc", Description: "Collapse / Back"},
		{Key: "/", Description: "Search / Filter"},
	}
}

// TableKeyBindings returns key bindings for table navigation.
//
// Returns []KeyBinding which contains the key mappings for moving around and
// working with tables.
func TableKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "j / Down", Description: "Move down"},
		{Key: "k / Up", Description: "Move up"},
		{Key: "h / Left", Description: "Scroll left"},
		{Key: "l / Right", Description: "Scroll right"},
		{Key: "g", Description: "Go to top"},
		{Key: "G", Description: "Go to bottom"},
		{Key: keyEnter, Description: "View details"},
		{Key: "/", Description: "Search"},
	}
}

// MetricsPanelKeyBindings returns the key bindings for the metrics panel.
//
// Returns []KeyBinding which lists the available key commands.
func MetricsPanelKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "p", Description: "Toggle percentiles"},
		{Key: "t", Description: "Change time range"},
		{Key: "s", Description: "Sort metrics"},
	}
}

// TracesPanelKeyBindings returns the key bindings for the traces panel.
//
// Returns []KeyBinding which contains the key actions available for the panel.
func TracesPanelKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "e", Description: "Show only errors"},
		{Key: "s", Description: "Sort by duration"},
		{Key: keyEnter, Description: "View trace details"},
	}
}

// ResourcesPanelKeyBindings returns the key bindings for the resources panel.
//
// Returns []KeyBinding which lists the available key actions.
func ResourcesPanelKeyBindings() []KeyBinding {
	return []KeyBinding{
		{Key: "f", Description: "Filter by status"},
		{Key: keyEnter, Description: "Expand / View details"},
	}
}
