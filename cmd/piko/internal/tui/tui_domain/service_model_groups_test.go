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

	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
)

func TestModelSetGroupsActivatesFirstVisible(t *testing.T) {
	config := tui_dto.DefaultConfig()
	model := NewModel(config)

	g := newTestGroup()
	model.SetGroups([]PanelGroup{g})

	if model.ActiveGroupID() != g.ID() {
		t.Errorf("ActiveGroupID = %q, want %q", model.ActiveGroupID(), g.ID())
	}
	if got := model.ActiveGroup(); got == nil || got.ID() != g.ID() {
		t.Errorf("ActiveGroup did not return registered group")
	}
	if got := model.ActiveItem(); got.Panel == nil || got.ID != g.items[0].ID {
		t.Errorf("ActiveItem did not return default item")
	}
}

func TestModelMoveMenuCursor(t *testing.T) {
	config := tui_dto.DefaultConfig()
	model := NewModel(config)
	g := newTestGroup()
	model.SetGroups([]PanelGroup{g})

	model.moveMenuCursor(+1)
	if got := model.menuCursorByGroup[g.ID()]; got != 1 {
		t.Errorf("after +1: cursor = %d, want 1", got)
	}

	model.moveMenuCursor(+1)
	if got := model.menuCursorByGroup[g.ID()]; got != 1 {
		t.Errorf("after second +1: cursor = %d, want 1 (clamped)", got)
	}

	model.moveMenuCursor(-2)
	if got := model.menuCursorByGroup[g.ID()]; got != 0 {
		t.Errorf("after -2: cursor = %d, want 0 (clamped)", got)
	}
}

func TestModelCommitMenuCursorActivatesItem(t *testing.T) {
	config := tui_dto.DefaultConfig()
	model := NewModel(config)
	g := newTestGroup()
	model.SetGroups([]PanelGroup{g})

	model.moveMenuCursor(+1)
	model.commitMenuCursor()

	if got := model.activeItemByGroup[g.ID()]; got != g.items[1].ID {
		t.Errorf("active item = %q, want %q", got, g.items[1].ID)
	}
}

func TestModelToggleColumnsOverridesBreakpoint(t *testing.T) {
	config := tui_dto.DefaultConfig()
	model := NewModel(config)
	g := newTestGroup()
	model.SetGroups([]PanelGroup{g})
	model.width = 180
	model.height = 40

	model.toggleMenuColumn()
	model.toggleDetailColumn()

	v := model.groupVisibility[g.ID()]
	if v.LeftOverride == nil || *v.LeftOverride {
		t.Errorf("LeftOverride = %v, want non-nil & false", v.LeftOverride)
	}
	if v.RightOverride == nil || *v.RightOverride {
		t.Errorf("RightOverride = %v, want non-nil & false", v.RightOverride)
	}
}
