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
	"strings"

	tea "charm.land/bubbletea/v2"
)

// RegisterBuiltinCommands populates a registry with the canonical command
// set. Callers can extend the registry afterwards by calling Register with
// additional Command values.
//
// Takes registry (*CommandRegistry) which receives the built-in commands.
func RegisterBuiltinCommands(registry *CommandRegistry) {
	if registry == nil {
		return
	}

	registry.Register(Command{Name: "quit", Aliases: []string{"q", "exit"}, Description: "Quit the TUI", Run: runQuitCommand})
	registry.Register(Command{Name: "refresh", Aliases: []string{"r"}, Description: "Force-refresh all providers", Run: runRefreshCommand})
	registry.Register(Command{Name: "help", Aliases: []string{"?"}, Description: "Open the help overlay", Run: runHelpCommand})
	registry.Register(Command{Name: "theme", Description: "Switch theme. Without arguments, lists registered themes", Run: runThemeCommand})
	registry.Register(Command{Name: "focus", Description: "Focus the panel with the given ID", Run: runFocusCommand})
	registry.Register(Command{Name: "layout", Description: "Force a layout (single|two|three) or clear with no arguments", Run: runLayoutCommand})
}

// runQuitCommand emits the quit message.
//
// Returns tea.Cmd which delivers a quitMessage to the runtime.
func runQuitCommand(_ []string, _ *Model) tea.Cmd {
	return func() tea.Msg { return quitMessage{} }
}

// runRefreshCommand emits the force-refresh message.
//
// Returns tea.Cmd which delivers a forceRefreshMessage to the runtime.
func runRefreshCommand(_ []string, _ *Model) tea.Cmd {
	return func() tea.Msg { return forceRefreshMessage{} }
}

// runHelpCommand emits the toggle-help message.
//
// Returns tea.Cmd which delivers a toggleHelpMessage to the runtime.
func runHelpCommand(_ []string, _ *Model) tea.Cmd {
	return func() tea.Msg { return toggleHelpMessage{} }
}

// runThemeCommand applies a theme by name or lists registered themes when
// no name is given.
//
// Takes args ([]string) which are the parsed command arguments; the first
// argument names the theme to apply.
// Takes model (*Model) which is the active TUI model.
//
// Returns tea.Cmd which is always nil; mutations occur on the model directly.
func runThemeCommand(args []string, model *Model) tea.Cmd {
	if model == nil {
		return nil
	}
	if len(args) == 0 {
		names := GlobalThemeRegistry().Names()
		if model.toasts != nil {
			model.toasts.Push(ToastInfo, "Available themes: "+strings.Join(names, ", "))
		}
		return nil
	}
	theme, ok := GlobalThemeRegistry().Get(args[0])
	if !ok {
		if model.toasts != nil {
			model.toasts.Push(ToastWarn, "Unknown theme: "+args[0])
		}
		return nil
	}
	model.SetTheme(&theme)
	if model.toasts != nil {
		model.toasts.Push(ToastSuccess, "Theme: "+theme.Name)
	}
	return nil
}

// runFocusCommand emits a focus-panel message for the given panel ID.
//
// Takes args ([]string) which are the parsed command arguments; the first
// argument is the panel ID.
// Takes model (*Model) which is the active TUI model.
//
// Returns tea.Cmd which delivers a focusPanelMessage, or nil when args are missing.
func runFocusCommand(args []string, model *Model) tea.Cmd {
	if model == nil || len(args) == 0 {
		return nil
	}
	id := args[0]
	return func() tea.Msg { return focusPanelMessage{panelID: id} }
}

// runLayoutCommand forces a specific layout, or clears the override when
// invoked without arguments.
//
// Takes args ([]string) which are the parsed command arguments; the first
// argument names the layout (single|two|three).
// Takes model (*Model) which is the active TUI model.
//
// Returns tea.Cmd which is always nil; mutations occur on the model directly.
func runLayoutCommand(args []string, model *Model) tea.Cmd {
	if model == nil || model.layoutPicker == nil {
		return nil
	}
	if len(args) == 0 {
		model.layoutPicker.Override("")
		model.layoutPicker.Reflow(model.width, model.height-LayoutChromeHeight)
		return nil
	}
	name := strings.ToLower(args[0])
	switch name {
	case LayoutNameSingle, LayoutNameTwoColumn, LayoutNameThreeColumn:
		model.layoutPicker.Override(name)
	default:
		if model.toasts != nil {
			model.toasts.Push(ToastWarn, "Unknown layout: "+name)
		}
	}
	return nil
}
