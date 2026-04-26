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

type stubPanel struct {
	id      string
	title   string
	focused bool
}

func newStubPanel(id, title string) *stubPanel {
	return &stubPanel{id: id, title: title}
}

func (p *stubPanel) ID() string                        { return p.id }
func (p *stubPanel) Title() string                     { return p.title }
func (p *stubPanel) Init() tea.Cmd                     { return nil }
func (p *stubPanel) Update(_ tea.Msg) (Panel, tea.Cmd) { return p, nil }
func (p *stubPanel) View(_, _ int) string              { return p.id }
func (p *stubPanel) Focused() bool                     { return p.focused }
func (p *stubPanel) SetFocused(focused bool)           { p.focused = focused }
func (p *stubPanel) KeyMap() []KeyBinding              { return nil }
func (*stubPanel) DetailView(_, _ int) string          { return "" }
func (*stubPanel) Selection() Selection                { return Selection{} }

func makeStubPanels(ids ...string) []Panel {
	panels := make([]Panel, len(ids))
	for i, id := range ids {
		panels[i] = newStubPanel(id, id)
	}
	return panels
}

func TestDefaultPaneAssignerSingle(t *testing.T) {
	panels := makeStubPanels("a", "b", "c")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "b", LayoutNameSingle, 1)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Panel.ID() != "b" {
		t.Errorf("panel ID = %q, want %q", got[0].Panel.ID(), "b")
	}
	if got[0].Slot != SlotPrimary {
		t.Errorf("slot = %d, want SlotPrimary", got[0].Slot)
	}
}

func TestDefaultPaneAssignerTwoColumn(t *testing.T) {
	panels := makeStubPanels("a", "b", "c")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "b", LayoutNameTwoColumn, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Panel.ID() != "b" || got[0].Slot != SlotPrimary {
		t.Errorf("primary mismatch: %+v", got[0])
	}
	if got[1].Panel.ID() != "c" || got[1].Slot != SlotDetail {
		t.Errorf("detail mismatch: %+v", got[1])
	}
}

func TestDefaultPaneAssignerThreeColumn(t *testing.T) {
	panels := makeStubPanels("a", "b", "c")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "b", LayoutNameThreeColumn, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Panel.ID() != "a" || got[0].Slot != SlotContext {
		t.Errorf("context mismatch: %+v", got[0])
	}
	if got[1].Panel.ID() != "b" || got[1].Slot != SlotPrimary {
		t.Errorf("primary mismatch: %+v", got[1])
	}
	if got[2].Panel.ID() != "c" || got[2].Slot != SlotDetail {
		t.Errorf("detail mismatch: %+v", got[2])
	}
}

func TestDefaultPaneAssignerPairings(t *testing.T) {
	panels := makeStubPanels("registry", "storage", "metrics", "traces")
	assigner := &DefaultPaneAssigner{
		Pairings: map[string]string{"registry": "storage"},
	}

	got := assigner.Assign(panels, "registry", LayoutNameTwoColumn, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[1].Panel.ID() != "storage" {
		t.Errorf("paired detail = %q, want %q", got[1].Panel.ID(), "storage")
	}
}

func TestDefaultPaneAssignerFocusOnFirst(t *testing.T) {
	panels := makeStubPanels("a", "b", "c")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "a", LayoutNameThreeColumn, 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Panel.ID() != "c" {
		t.Errorf("expected last panel as context when focus is on first, got %q", got[0].Panel.ID())
	}
	if got[1].Panel.ID() != "a" {
		t.Errorf("primary should remain focused panel, got %q", got[1].Panel.ID())
	}
}

func TestDefaultPaneAssignerSinglePanel(t *testing.T) {
	panels := makeStubPanels("only")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "only", LayoutNameThreeColumn, 3)
	if len(got) == 0 || got[0].Panel.ID() != "only" {
		t.Errorf("expected single assignment for single-panel input, got %+v", got)
	}
}

func TestDefaultPaneAssignerMissingFocus(t *testing.T) {
	panels := makeStubPanels("a", "b")
	assigner := NewDefaultPaneAssigner()

	got := assigner.Assign(panels, "missing", LayoutNameTwoColumn, 2)
	if len(got) == 0 || got[0].Panel.ID() != "a" {
		t.Errorf("expected fallback to first panel, got %+v", got)
	}
}
