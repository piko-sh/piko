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
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCommandRegistryRegisterAndLookup(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{
		Name:        "ping",
		Description: "Ping",
		Run:         func(_ []string, _ *Model) tea.Cmd { return nil },
	})

	cmd, ok := r.Lookup("ping")
	if !ok || cmd.Name != "ping" {
		t.Errorf("Lookup ping = %+v, ok=%v", cmd, ok)
	}
}

func TestCommandRegistryAliases(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{
		Name:    "quit",
		Aliases: []string{"q", "exit"},
		Run:     func(_ []string, _ *Model) tea.Cmd { return nil },
	})

	for _, name := range []string{"quit", "q", "exit"} {
		cmd, ok := r.Lookup(name)
		if !ok || cmd.Name != "quit" {
			t.Errorf("Lookup %q did not resolve to quit (got %+v)", name, cmd)
		}
	}
}

func TestCommandRegistryDuplicatePanics(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "ping", Run: func(_ []string, _ *Model) tea.Cmd { return nil }})

	defer func() {
		if recover() == nil {
			t.Errorf("expected panic on duplicate register")
		}
	}()
	r.Register(Command{Name: "ping", Run: func(_ []string, _ *Model) tea.Cmd { return nil }})
}

func TestCommandRegistryComplete(t *testing.T) {
	r := NewCommandRegistry()
	for _, name := range []string{"layout", "list", "log", "quit"} {
		r.Register(Command{Name: name, Run: func(_ []string, _ *Model) tea.Cmd { return nil }})
	}

	out := r.Complete("l")
	if len(out) != 3 {
		t.Errorf("Complete returned %d, want 3", len(out))
	}
	if out[0].Name != "layout" || out[1].Name != "list" || out[2].Name != "log" {
		t.Errorf("Complete order wrong: %+v", out)
	}
}

func TestParseCommand(t *testing.T) {
	cases := []struct {
		line     string
		wantName string
		wantArgs []string
		wantOK   bool
	}{
		{":quit", "quit", []string{}, true},
		{"quit", "quit", []string{}, true},
		{":theme piko-light", "theme", []string{"piko-light"}, true},
		{"  :focus    panel-id  ", "focus", []string{"panel-id"}, true},
		{":Theme dark", "theme", []string{"dark"}, true},
		{"", "", nil, false},
		{":   ", "", nil, false},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			name, args, ok := ParseCommand(c.line)
			if ok != c.wantOK {
				t.Errorf("ok = %v, want %v", ok, c.wantOK)
			}
			if name != c.wantName {
				t.Errorf("name = %q, want %q", name, c.wantName)
			}
			if len(args) != len(c.wantArgs) {
				t.Errorf("args len = %d, want %d (%v)", len(args), len(c.wantArgs), args)
			}
		})
	}
}

func TestRegisterBuiltinCommands(t *testing.T) {
	r := NewCommandRegistry()
	RegisterBuiltinCommands(r)

	for _, name := range []string{"quit", "q", "refresh", "r", "help", "?", "theme", "focus", "layout"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("missing built-in: %q", name)
		}
	}
}
