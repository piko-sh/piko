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
	"testing"

	tea "charm.land/bubbletea/v2"
)

func commandBarKeyPress(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func TestCommandBarOpenAndClose(t *testing.T) {
	registry := NewCommandRegistry()
	bar := NewCommandBar(registry, nil)

	if bar.Active() {
		t.Errorf("new bar should be inactive")
	}

	bar.Open(CommandModeCommand)
	if !bar.Active() {
		t.Errorf("Open should activate the bar")
	}
	if bar.Mode() != CommandModeCommand {
		t.Errorf("Mode = %v, want CommandModeCommand", bar.Mode())
	}

	bar.Close()
	if bar.Active() {
		t.Errorf("Close should deactivate the bar")
	}
}

func TestCommandBarOpenOffIsNoop(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	bar.Open(CommandModeOff)
	if bar.Active() {
		t.Errorf("CommandModeOff should not activate")
	}
}

func TestCommandBarUpdateInactiveIsNoop(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	cmd := bar.Update(commandBarKeyPress('a'), nil)
	if cmd != nil {
		t.Errorf("inactive Update should return nil cmd")
	}
}

func TestCommandBarSubmitsCommand(t *testing.T) {
	called := false
	registry := NewCommandRegistry()
	registry.Register(Command{
		Name: "ping",
		Run: func(_ []string, _ *Model) tea.Cmd {
			called = true
			return nil
		},
	})

	bar := NewCommandBar(registry, nil)
	bar.Open(CommandModeCommand)
	for _, r := range "ping" {
		bar.Update(commandBarKeyPress(r), nil)
	}
	bar.Update(tea.KeyPressMsg{Code: 13}, nil)

	if !called {
		t.Errorf("registered command was not invoked")
	}
	if bar.Active() {
		t.Errorf("bar should close after submit")
	}
}

func TestCommandBarSubmitsFilterMessage(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	bar.Open(CommandModeFilter)
	for _, r := range "needle" {
		bar.Update(commandBarKeyPress(r), nil)
	}
	cmd := bar.Update(tea.KeyPressMsg{Code: 13}, nil)
	if cmd == nil {
		t.Fatalf("expected filter cmd from Enter")
	}
	msg := cmd()
	apply, ok := msg.(filterApplyMessage)
	if !ok {
		t.Fatalf("Enter on filter mode returned %T, want filterApplyMessage", msg)
	}
	if apply.Query != "needle" {
		t.Errorf("filter query = %q, want %q", apply.Query, "needle")
	}
}

func TestCommandBarSubmitsSearchMessage(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	bar.Open(CommandModeSearch)
	for _, r := range "term" {
		bar.Update(commandBarKeyPress(r), nil)
	}
	cmd := bar.Update(tea.KeyPressMsg{Code: 13}, nil)
	if cmd == nil {
		t.Fatalf("expected search cmd from Enter")
	}
	msg := cmd()
	if _, ok := msg.(searchApplyMessage); !ok {
		t.Fatalf("Enter on search mode returned %T, want searchApplyMessage", msg)
	}
}

func TestCommandBarEscClosesWithoutSubmitting(t *testing.T) {
	called := false
	registry := NewCommandRegistry()
	registry.Register(Command{
		Name: "ping",
		Run: func(_ []string, _ *Model) tea.Cmd {
			called = true
			return nil
		},
	})

	bar := NewCommandBar(registry, nil)
	bar.Open(CommandModeCommand)
	for _, r := range "ping" {
		bar.Update(commandBarKeyPress(r), nil)
	}
	bar.Update(tea.KeyPressMsg{Code: 27}, nil)

	if called {
		t.Errorf("Esc should not invoke the command")
	}
	if bar.Active() {
		t.Errorf("Esc should close the bar")
	}
}

func TestCommandBarUnknownCommandPushesToast(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	model := NewModel(newTestConfig())

	bar.Open(CommandModeCommand)
	for _, r := range "unknown" {
		bar.Update(commandBarKeyPress(r), model)
	}
	bar.Update(tea.KeyPressMsg{Code: 13}, model)

	toast, ok := model.toasts.Current()
	if !ok {
		t.Fatalf("expected a warn toast for unknown command")
	}
	if !strings.Contains(toast.Body, "Unknown command") {
		t.Errorf("toast body = %q, want unknown-command label", toast.Body)
	}
}

func TestCommandBarViewWhenInactive(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	if got := bar.View(60); got != "" {
		t.Errorf("inactive View = %q, want empty", got)
	}
}

func TestCommandBarViewWhenActive(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		mode   CommandBarMode
	}{
		{name: "command", mode: CommandModeCommand, prefix: ":"},
		{name: "filter", mode: CommandModeFilter, prefix: "/"},
		{name: "search", mode: CommandModeSearch, prefix: "?"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bar := NewCommandBar(NewCommandRegistry(), nil)
			bar.Open(c.mode)
			out := bar.View(60)
			if out == "" {
				t.Errorf("active View returned empty in %s mode", c.name)
			}
			if !strings.Contains(out, c.prefix) {
				t.Errorf("active View %s mode missing prompt %q: %q", c.name, c.prefix, out)
			}
		})
	}
}

func TestCommandBarValueExposesInput(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	bar.Open(CommandModeCommand)
	for _, r := range "abc" {
		bar.Update(commandBarKeyPress(r), nil)
	}
	if bar.Value() != "abc" {
		t.Errorf("Value = %q, want %q", bar.Value(), "abc")
	}
}

func TestCommandBarSetWidthClampsToMinimum(t *testing.T) {
	bar := NewCommandBar(NewCommandRegistry(), nil)
	bar.SetWidth(2)
	bar.Open(CommandModeCommand)
	out := bar.View(2)
	if out == "" {
		t.Errorf("View with tiny width returned empty")
	}
}
