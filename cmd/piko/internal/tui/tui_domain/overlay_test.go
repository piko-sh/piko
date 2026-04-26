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

type stubOverlay struct {
	id        string
	body      string
	keys      []KeyBinding
	min       struct{ w, h int }
	dismissed bool
}

func newStubOverlay(id, body string) *stubOverlay {
	o := &stubOverlay{id: id, body: body}
	o.min.w, o.min.h = 20, 4
	return o
}

func (o *stubOverlay) ID() string             { return o.id }
func (o *stubOverlay) Render(_, _ int) string { return o.body }
func (o *stubOverlay) KeyMap() []KeyBinding   { return o.keys }
func (o *stubOverlay) Dismiss() bool          { return o.dismissed }
func (o *stubOverlay) MinSize() (int, int)    { return o.min.w, o.min.h }
func (o *stubOverlay) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "esc" {
			o.dismissed = true
		}
	}
	return o, nil
}

func TestOverlayManagerPushPop(t *testing.T) {
	m := NewOverlayManager(nil)
	if !m.Empty() {
		t.Error("new manager should be empty")
	}

	a := newStubOverlay("a", "AAA")
	b := newStubOverlay("b", "BBB")
	m.Push(a)
	m.Push(b)

	if m.Top().ID() != "b" {
		t.Errorf("Top = %q, want b", m.Top().ID())
	}
	popped := m.Pop()
	if popped.ID() != "b" {
		t.Errorf("Pop = %q, want b", popped.ID())
	}
	if m.Top().ID() != "a" {
		t.Errorf("Top after pop = %q, want a", m.Top().ID())
	}

	m.Pop()
	if !m.Empty() {
		t.Error("manager should be empty after popping all")
	}
	if got := m.Pop(); got != nil {
		t.Error("Pop on empty manager should return nil")
	}
}

func TestOverlayManagerUpdateConsumesInput(t *testing.T) {
	m := NewOverlayManager(nil)
	o := newStubOverlay("a", "body")
	m.Push(o)

	_, consumed := m.Update(tea.KeyPressMsg{Code: 'x'})
	if !consumed {
		t.Error("Update should report consumed when overlay is active")
	}
}

func TestOverlayManagerEmptyUpdate(t *testing.T) {
	m := NewOverlayManager(nil)
	cmd, consumed := m.Update(nil)
	if consumed {
		t.Error("empty manager should not consume messages")
	}
	if cmd != nil {
		t.Error("empty manager should not produce a command")
	}
}

func TestOverlayManagerAutoPopOnDismiss(t *testing.T) {
	m := NewOverlayManager(nil)
	o := newStubOverlay("a", "body")
	m.Push(o)

	m.Update(tea.KeyPressMsg{Code: 'a'})
	if m.Empty() {
		t.Error("non-dismissive key should not pop overlay")
	}

	m.Update(tea.KeyPressMsg{Mod: 0, Code: 27})

	o.dismissed = true
	if !m.Empty() {

		m.Update(tea.KeyPressMsg{Code: 'x'})
	}
	if !m.Empty() {
		t.Error("dismissed overlay should be popped on next Update")
	}
}

func TestComposeOverlayCentresContent(t *testing.T) {
	bg := strings.Repeat(strings.Repeat(".", 20)+"\n", 5)
	bg = strings.TrimRight(bg, "\n")
	overlay := "AAAA\nAAAA"

	composed := ComposeOverlay(bg, overlay, 20, 5, nil)

	if !strings.Contains(composed, "AAAA") {
		t.Errorf("composed should contain overlay body: %q", composed)
	}
	if !strings.Contains(composed, ".") {
		t.Errorf("composed should retain background dots: %q", composed)
	}
	rows := strings.Split(composed, "\n")
	if len(rows) != 5 {
		t.Errorf("expected 5 rows, got %d", len(rows))
	}
}

func TestComposeOverlayEmptyBody(t *testing.T) {
	bg := "..."
	composed := ComposeOverlay(bg, "", 3, 1, nil)
	if composed != bg {
		t.Errorf("empty overlay should not modify background: %q", composed)
	}
}

func TestHelpOverlayDismissOnEsc(t *testing.T) {
	overlay := NewHelpOverlay(nil, []KeyBinding{{Key: "q", Description: "Quit"}}, "", nil, nil)
	if overlay.Dismiss() {
		t.Error("overlay should not start dismissed")
	}
	overlay.Update(tea.KeyPressMsg{Code: 27})

	overlay.Update(makeKey("esc"))
	if !overlay.Dismiss() {
		t.Error("overlay should be dismissed after Esc")
	}
}

func TestHelpOverlayRenders(t *testing.T) {
	overlay := NewHelpOverlay(nil,
		[]KeyBinding{{Key: "q", Description: "Quit"}},
		"Test",
		[]KeyBinding{{Key: "j", Description: "Down"}},
		nil,
	)

	body := overlay.Render(60, 14)
	if !strings.Contains(body, "Quit") {
		t.Errorf("expected Quit in help body: %q", body)
	}
	if !strings.Contains(body, "Down") {
		t.Errorf("expected Down in help body: %q", body)
	}
	if !strings.Contains(body, "Panel: Test") {
		t.Errorf("expected panel section in help body: %q", body)
	}
}

func makeKey(label string) tea.KeyPressMsg {
	switch label {
	case "esc":
		return tea.KeyPressMsg{Code: 27}
	case "?":
		return tea.KeyPressMsg{Code: '?'}
	case "q":
		return tea.KeyPressMsg{Code: 'q'}
	}
	return tea.KeyPressMsg{}
}
