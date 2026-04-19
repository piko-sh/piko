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

type fakePanel struct {
	selection Selection
	body      string
	id        string
	title     string
	keyMap    []KeyBinding
	focused   bool
}

func (f *fakePanel) ID() string                        { return f.id }
func (f *fakePanel) Title() string                     { return f.title }
func (*fakePanel) Init() tea.Cmd                       { return nil }
func (f *fakePanel) Update(_ tea.Msg) (Panel, tea.Cmd) { return f, nil }
func (f *fakePanel) View(width, _ int) string {
	return strings.Repeat(f.body, width/max(1, len(f.body)))
}
func (f *fakePanel) Focused() bool            { return f.focused }
func (f *fakePanel) SetFocused(focused bool)  { f.focused = focused }
func (f *fakePanel) KeyMap() []KeyBinding     { return f.keyMap }
func (*fakePanel) DetailView(_, _ int) string { return "" }
func (f *fakePanel) Selection() Selection     { return f.selection }

type fakeGroup struct {
	id      GroupID
	title   string
	items   []MenuItem
	hotkey  rune
	visible bool
}

func (f *fakeGroup) ID() GroupID           { return f.id }
func (f *fakeGroup) Title() string         { return f.title }
func (f *fakeGroup) Hotkey() rune          { return f.hotkey }
func (f *fakeGroup) Items() []MenuItem     { return f.items }
func (f *fakeGroup) DefaultItemID() ItemID { return f.items[0].ID }
func (f *fakeGroup) Visible() bool         { return f.visible }

func newTestGroup() *fakeGroup {
	items := []MenuItem{
		{ID: "alpha", Label: "Alpha", Hotkey: "1", Panel: &fakePanel{id: "alpha", title: "Alpha", body: "α"}},
		{ID: "beta", Label: "Beta", Hotkey: "2", Panel: &fakePanel{id: "beta", title: "Beta", body: "β"}},
	}
	return &fakeGroup{id: "test", title: "Test", hotkey: '1', items: items, visible: true}
}

func TestGroupedViewComposeWide(t *testing.T) {
	g := newTestGroup()
	view := NewGroupedView(nil)
	body := view.Compose(GroupedViewArgs{
		Group:      g,
		Item:       g.items[0],
		Breakpoint: DefaultBreakpoints[2],
		Width:      180,
		Height:     30,
	})
	if body == "" {
		t.Fatal("expected non-empty body")
	}
	if !strings.Contains(body, "Alpha") {
		t.Errorf("expected menu to contain Alpha; got %q", body)
	}
}

func TestGroupedViewComposeNarrow(t *testing.T) {
	g := newTestGroup()
	view := NewGroupedView(nil)
	body := view.Compose(GroupedViewArgs{
		Group:      g,
		Item:       g.items[0],
		Breakpoint: DefaultBreakpoints[0],
		Width:      80,
		Height:     20,
	})
	if body == "" {
		t.Fatal("expected non-empty body for narrow terminal")
	}
}

func TestGroupedViewToggleRight(t *testing.T) {
	g := newTestGroup()
	view := NewGroupedView(nil)
	body := view.Compose(GroupedViewArgs{
		Group:      g,
		Item:       g.items[0],
		Visibility: GroupVisibilityState{RightOverride: new(true)},
		Breakpoint: DefaultBreakpoints[1],
		Width:      130,
		Height:     30,
	})
	if body == "" {
		t.Fatal("expected non-empty body with right override on")
	}
}

func TestGroupHotkeyAccelerator(t *testing.T) {
	cases := []struct {
		group PanelGroup
		want  rune
	}{
		{group: NewContentGroup(ContentPanels{}), want: '1'},
		{group: NewTelemetryGroup(TelemetryPanels{}), want: '2'},
		{group: NewRuntimeGroup(RuntimePanels{}), want: '3'},
		{group: NewWatchdogGroup(WatchdogPanels{}), want: '4'},
	}
	for _, c := range cases {
		if got := c.group.Hotkey(); got != c.want {
			t.Errorf("%s.Hotkey() = %q, want %q", c.group.ID(), got, c.want)
		}
	}
}

func TestEmptyGroupVisibility(t *testing.T) {
	g := NewContentGroup(ContentPanels{})
	if g.Visible() {
		t.Errorf("empty content group should report Visible == false")
	}
}
