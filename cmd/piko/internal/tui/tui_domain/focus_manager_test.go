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

import "testing"

func TestFocusManagerInitialActive(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c"))

	if got := f.ActiveID(); got != "a" {
		t.Errorf("ActiveID = %q, want %q", got, "a")
	}
}

func TestFocusManagerSetActive(t *testing.T) {
	panels := makeStubPanels("a", "b", "c")
	f := NewFocusManager()
	f.SetPanels(panels)

	if !f.SetActive("c") {
		t.Errorf("SetActive should report change to a different ID")
	}
	if f.ActiveID() != "c" {
		t.Errorf("ActiveID = %q, want %q", f.ActiveID(), "c")
	}
	if !panels[2].Focused() {
		t.Error("c panel should be focused")
	}
	if panels[0].Focused() || panels[1].Focused() {
		t.Error("non-active panels should not be focused")
	}

	if f.SetActive("c") {
		t.Errorf("SetActive should report no change for same ID")
	}

	if f.SetActive("missing") {
		t.Errorf("SetActive should report no change for unknown ID")
	}
}

func TestFocusManagerNextPrevPanel(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c"))

	if got := f.NextPanel(); got != "b" {
		t.Errorf("NextPanel from a = %q, want %q", got, "b")
	}
	f.SetActive("c")
	if got := f.NextPanel(); got != "a" {
		t.Errorf("NextPanel wrap from c = %q, want %q", got, "a")
	}

	f.SetActive("a")
	if got := f.PrevPanel(); got != "c" {
		t.Errorf("PrevPanel wrap from a = %q, want %q", got, "c")
	}
}

func TestFocusManagerVisibleCycling(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c", "d"))

	f.MarkVisible([]string{"b", "d"})

	f.SetActive("b")
	if got := f.NextVisible(); got != "d" {
		t.Errorf("NextVisible from b = %q, want %q", got, "d")
	}

	f.SetActive("d")
	if got := f.NextVisible(); got != "b" {
		t.Errorf("NextVisible wrap from d = %q, want %q", got, "b")
	}

	f.SetActive("b")
	if got := f.PrevVisible(); got != "d" {
		t.Errorf("PrevVisible wrap from b = %q, want %q", got, "d")
	}
}

func TestFocusManagerVisibleCyclingActiveNotInVisibleSet(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c"))
	f.SetActive("a")
	f.MarkVisible([]string{"b", "c"})

	if got := f.NextVisible(); got != "b" {
		t.Errorf("NextVisible from non-visible active = %q, want %q", got, "b")
	}
	if got := f.PrevVisible(); got != "c" {
		t.Errorf("PrevVisible from non-visible active = %q, want %q", got, "c")
	}
}

func TestFocusManagerEmptyPanels(t *testing.T) {
	f := NewFocusManager()

	if id := f.ActiveID(); id != "" {
		t.Errorf("ActiveID with no panels = %q, want empty", id)
	}
	if id := f.NextPanel(); id != "" {
		t.Errorf("NextPanel with no panels = %q, want empty", id)
	}
	if id := f.PrevPanel(); id != "" {
		t.Errorf("PrevPanel with no panels = %q, want empty", id)
	}
	if id := f.NextVisible(); id != "" {
		t.Errorf("NextVisible with no panels = %q, want empty", id)
	}
}

func TestFocusManagerSetPanelsPreservesActive(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c"))
	f.SetActive("b")

	f.SetPanels(makeStubPanels("a", "b", "d"))
	if f.ActiveID() != "b" {
		t.Errorf("ActiveID after panel-list update = %q, want %q", f.ActiveID(), "b")
	}

	f.SetPanels(makeStubPanels("x", "y"))
	if f.ActiveID() != "x" {
		t.Errorf("ActiveID after removing previously-active panel = %q, want %q", f.ActiveID(), "x")
	}
}

func TestFocusManagerIsVisible(t *testing.T) {
	f := NewFocusManager()
	f.SetPanels(makeStubPanels("a", "b", "c"))
	f.MarkVisible([]string{"a", "c"})

	if !f.IsVisible("a") {
		t.Error("a should be visible")
	}
	if f.IsVisible("b") {
		t.Error("b should not be visible")
	}
	if !f.IsVisible("c") {
		t.Error("c should be visible")
	}
}
