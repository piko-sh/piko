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
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// CommandBarMode selects the prompt and routing of input typed into the
// command bar. The bar is a single line at the bottom of the screen; the
// mode controls whether submitted text is treated as a command, a filter
// query, or a search query.
type CommandBarMode int

const (
	// CommandModeOff means the bar is hidden.
	CommandModeOff CommandBarMode = iota

	// CommandModeCommand routes Enter through the CommandRegistry.
	CommandModeCommand

	// CommandModeFilter routes Enter into the focused panel's filter,
	// consuming only that panel's items list.
	CommandModeFilter

	// CommandModeSearch routes Enter into a substring search across the
	// focused panel.
	CommandModeSearch
)

// commandBarMaxHistory caps the in-memory ring of recently-submitted
// command lines.
const commandBarMaxHistory = 20

// CommandBar is the bottom-of-screen single-line input used for ":"
// commands, "/" filters, and "?" searches. Only one mode is active at a
// time.
type CommandBar struct {
	// registry resolves typed command names into handlers.
	registry *CommandRegistry

	// theme drives prompt and bar colours.
	theme *Theme

	// history is the in-memory ring of recently submitted lines.
	history []string

	// input is the underlying textinput model.
	input textinput.Model

	// historyIndex is the cursor into history for up/down navigation.
	historyIndex int

	// mode controls how submitted text is routed.
	mode CommandBarMode
}

// NewCommandBar constructs a command bar bound to the supplied registry.
//
// Takes registry (*CommandRegistry) which resolves typed commands.
// Takes theme (*Theme) which styles the bar.
//
// Returns *CommandBar ready to call Open on.
func NewCommandBar(registry *CommandRegistry, theme *Theme) *CommandBar {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = CommandBarMaxLen
	ti.SetWidth(CommandBarVisibleWidth)

	return &CommandBar{
		registry:     registry,
		theme:        theme,
		input:        ti,
		history:      make([]string, 0, commandBarMaxHistory),
		historyIndex: 0,
		mode:         CommandModeOff,
	}
}

// SetTheme replaces the theme used to style the bar.
//
// Takes theme (*Theme) which becomes the new theme.
func (c *CommandBar) SetTheme(theme *Theme) {
	c.theme = theme
}

// Active reports whether the bar is currently consuming input.
//
// Returns bool which is true when the bar is open.
func (c *CommandBar) Active() bool {
	return c.mode != CommandModeOff
}

// Mode returns the current command bar mode.
//
// Returns CommandBarMode which is the current routing mode.
func (c *CommandBar) Mode() CommandBarMode {
	return c.mode
}

// Open activates the bar in the supplied mode and returns the cursor-blink
// command from the underlying textinput.
//
// Takes mode (CommandBarMode) which selects routing; passing
// CommandModeOff is a no-op.
//
// Returns tea.Cmd which is the cursor-blink command.
func (c *CommandBar) Open(mode CommandBarMode) tea.Cmd {
	if mode == CommandModeOff {
		return nil
	}
	c.mode = mode
	c.input.Reset()
	c.input.Focus()
	return textinput.Blink
}

// Close deactivates the bar without submitting.
func (c *CommandBar) Close() {
	c.mode = CommandModeOff
	c.input.Reset()
	c.input.Blur()
}

// Value returns the current input text.
//
// Returns string which is the typed line (without the leading prompt).
func (c *CommandBar) Value() string {
	return c.input.Value()
}

// SetWidth sizes the underlying input to fit width-2 cells (leaving room
// for the prompt).
//
// Takes width (int) which is the bar's total width.
func (c *CommandBar) SetWidth(width int) {
	width = max(width, CommandBarChromeWidth)
	c.input.SetWidth(width - CommandBarPromptReserve)
}

// Update handles input messages.
//
// When Enter is pressed the registered handler runs (in command mode) or a
// submission message is returned (in filter/search modes). Esc closes the bar.
//
// Takes msg (tea.Msg) which is the message to process.
// Takes model (*Model) which is the host model passed to command handlers
// in command mode.
//
// Returns tea.Cmd which is the resulting command.
func (c *CommandBar) Update(msg tea.Msg, model *Model) tea.Cmd {
	if !c.Active() {
		return nil
	}

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc":
			c.Close()
			return nil
		case "enter":
			return c.submit(model)
		}
	}

	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return cmd
}

// View renders the command bar to a single-line string of the supplied
// width. Returns the empty string when the bar is inactive.
//
// Takes width (int) which is the available width.
//
// Returns string which is the rendered bar.
func (c *CommandBar) View(width int) string {
	if !c.Active() {
		return ""
	}

	c.SetWidth(width)
	prompt := c.prompt()
	input := c.input.View()

	body := prompt + input
	body = PadRightANSI(body, width)

	if c.theme != nil {
		return c.theme.Search.UnsetPadding().Render(body)
	}
	return body
}

// submit acts on the current input value according to the active mode.
//
// Takes model (*Model) which is passed to command handlers in command
// mode.
//
// Returns tea.Cmd which is the resulting command.
func (c *CommandBar) submit(model *Model) tea.Cmd {
	value := c.input.Value()
	mode := c.mode
	c.recordHistory(value)
	c.Close()

	switch mode {
	case CommandModeCommand:
		return c.runCommand(value, model)
	case CommandModeFilter:
		return func() tea.Msg {
			return filterApplyMessage{Query: value}
		}
	case CommandModeSearch:
		return func() tea.Msg {
			return searchApplyMessage{Query: value}
		}
	default:
		return nil
	}
}

// runCommand parses value and dispatches to the registered handler, if
// any.
//
// Takes value (string) which is the typed input.
// Takes model (*Model) which is passed to the handler.
//
// Returns tea.Cmd which is the handler's command, or a toast about the
// missing command.
func (c *CommandBar) runCommand(value string, model *Model) tea.Cmd {
	name, args, ok := ParseCommand(value)
	if !ok || c.registry == nil {
		return nil
	}

	cmd, found := c.registry.Lookup(name)
	if !found {
		if model != nil && model.toasts != nil {
			model.toasts.Push(ToastWarn, "Unknown command: "+name)
		}
		return nil
	}
	return cmd.Run(args, model)
}

// recordHistory pushes value onto the history ring, evicting the oldest
// entry when full.
//
// Takes value (string) which is the typed line.
func (c *CommandBar) recordHistory(value string) {
	if value == "" {
		return
	}
	c.history = append(c.history, value)
	if len(c.history) > commandBarMaxHistory {
		c.history = c.history[len(c.history)-commandBarMaxHistory:]
	}
	c.historyIndex = len(c.history)
}

// prompt returns the leading prompt string for the active mode.
//
// Returns string which is the styled prompt (single character + space).
func (c *CommandBar) prompt() string {
	var prompt string
	switch c.mode {
	case CommandModeCommand:
		prompt = ":"
	case CommandModeFilter:
		prompt = "/"
	case CommandModeSearch:
		prompt = "?"
	default:
		prompt = " "
	}
	if c.theme != nil {
		return c.theme.SearchLabel.Render(prompt) + " "
	}
	return prompt + " "
}
