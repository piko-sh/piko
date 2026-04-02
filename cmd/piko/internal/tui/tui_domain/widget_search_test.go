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
)

func TestNewSearchBox(t *testing.T) {
	builder := NewSearchBox()

	if builder == nil {
		t.Fatal("expected non-nil SearchBox")
	}
	if builder.Active() {
		t.Error("expected new SearchBox to be inactive")
	}
	if builder.Query() != "" {
		t.Error("expected empty query")
	}
	if builder.width != searchBoxInitialWidth {
		t.Errorf("expected width %d, got %d", searchBoxInitialWidth, builder.width)
	}
}

func TestSearchBox_SetWidth(t *testing.T) {
	builder := NewSearchBox()

	builder.SetWidth(80)
	if builder.width != 80 {
		t.Errorf("expected width 80, got %d", builder.width)
	}

	builder.SetWidth(20)
	if builder.width != 20 {
		t.Errorf("expected width 20, got %d", builder.width)
	}
}

func TestSearchBox_OpenClose(t *testing.T) {
	builder := NewSearchBox()

	command := builder.Open()
	if !builder.Active() {
		t.Error("expected active after Open")
	}
	if command == nil {
		t.Error("expected blink command from Open")
	}

	builder.Close(true)
	if builder.Active() {
		t.Error("expected inactive after Close")
	}
}

func TestSearchBox_QueryGetSet(t *testing.T) {
	builder := NewSearchBox()
	builder.Open()

	builder.SetQuery("test query")
	if builder.Query() != "test query" {
		t.Errorf("expected 'test query', got %q", builder.Query())
	}

	builder.SetQuery("")
	if builder.Query() != "" {
		t.Error("expected empty query")
	}
}

func TestSearchBox_OnCloseCallback(t *testing.T) {
	builder := NewSearchBox()

	var callbackQuery string
	var callbackConfirmed bool
	builder.SetOnClose(func(query string, confirmed bool) {
		callbackQuery = query
		callbackConfirmed = confirmed
	})

	builder.Open()
	builder.SetQuery("search term")

	builder.Close(true)
	if callbackQuery != "search term" {
		t.Errorf("expected 'search term', got %q", callbackQuery)
	}
	if !callbackConfirmed {
		t.Error("expected confirmed=true")
	}

	builder.Open()
	builder.SetQuery("another term")
	builder.Close(false)
	if callbackQuery != "" {
		t.Errorf("expected empty query when not confirmed, got %q", callbackQuery)
	}
	if callbackConfirmed {
		t.Error("expected confirmed=false")
	}
}

func TestSearchBox_Update_EnterConfirms(t *testing.T) {
	builder := NewSearchBox()
	builder.Open()
	builder.SetQuery("my search")

	var confirmed bool
	builder.SetOnClose(func(_ string, c bool) {
		confirmed = c
	})

	message := createTestKeyMessage("enter")
	builder.Update(message)

	if builder.Active() {
		t.Error("expected inactive after Enter")
	}
	if !confirmed {
		t.Error("expected confirmed on Enter")
	}
}

func TestSearchBox_Update_EscCancels(t *testing.T) {
	builder := NewSearchBox()
	builder.Open()
	builder.SetQuery("my search")

	var confirmed bool
	builder.SetOnClose(func(_ string, c bool) {
		confirmed = c
	})

	message := createTestKeyMessage("esc")
	builder.Update(message)

	if builder.Active() {
		t.Error("expected inactive after Esc")
	}
	if confirmed {
		t.Error("expected not confirmed on Esc")
	}
}

func TestSearchBox_Update_InactiveDoesNothing(t *testing.T) {
	builder := NewSearchBox()

	message := createTestKeyMessage("enter")
	_, command := builder.Update(message)

	if command != nil {
		t.Error("expected nil command when inactive")
	}
}

func TestSearchBox_View_Inactive(t *testing.T) {
	builder := NewSearchBox()

	view := builder.View()
	if view != "" {
		t.Errorf("expected empty view when inactive, got %q", view)
	}
}

func TestSearchBox_View_Active(t *testing.T) {
	builder := NewSearchBox()
	builder.Open()

	view := builder.View()
	if view == "" {
		t.Error("expected non-empty view when active")
	}
}
