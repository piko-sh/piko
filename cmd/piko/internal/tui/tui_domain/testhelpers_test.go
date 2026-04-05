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
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
	"piko.sh/piko/wdk/clock"
)

type mockPanel struct {
	id         string
	title      string
	keyMap     []KeyBinding
	focused    bool
	updateFunc func(tea.Msg) (Panel, tea.Cmd)
}

func (p *mockPanel) ID() string               { return p.id }
func (p *mockPanel) Title() string            { return p.title }
func (p *mockPanel) Init() tea.Cmd            { return nil }
func (p *mockPanel) View(_ int, _ int) string { return "" }
func (p *mockPanel) Focused() bool            { return p.focused }
func (p *mockPanel) SetFocused(focused bool)  { p.focused = focused }
func (p *mockPanel) KeyMap() []KeyBinding     { return p.keyMap }

func (p *mockPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.updateFunc != nil {
		return p.updateFunc(message)
	}
	return p, nil
}

func newMockPanel(id string) *mockPanel {
	return &mockPanel{
		id:         id,
		title:      id,
		focused:    false,
		keyMap:     nil,
		updateFunc: nil,
	}
}

func createTestKeyMessage(key string) tea.KeyPressMsg {
	switch key {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	default:
		r := []rune(key)
		return tea.KeyPressMsg{Code: r[0], Text: key}
	}
}

func newTestClock() *clock.MockClock {
	return clock.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
}

func newTestConfig() *tui_dto.Config {
	return &tui_dto.Config{
		RefreshInterval: time.Second,
		Clock:           newTestClock(),
	}
}

var _ Panel = (*mockPanel)(nil)
