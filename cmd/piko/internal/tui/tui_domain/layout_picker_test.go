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

func TestLayoutPickerInitialState(t *testing.T) {
	p := NewLayoutPicker()
	if p.Active().Name() != LayoutNameSingle {
		t.Errorf("initial Active = %q, want %q", p.Active().Name(), LayoutNameSingle)
	}
}

func TestLayoutPickerReflowSelectsLayout(t *testing.T) {
	p := NewLayoutPicker()

	if p.Reflow(80, 24).Name() != LayoutNameSingle {
		t.Errorf("Reflow(80,24) wrong layout: %q", p.Active().Name())
	}
	if p.Reflow(120, 28).Name() != LayoutNameTwoColumn {
		t.Errorf("Reflow(120,28) wrong layout: %q", p.Active().Name())
	}
	if p.Reflow(200, 40).Name() != LayoutNameThreeColumn {
		t.Errorf("Reflow(200,40) wrong layout: %q", p.Active().Name())
	}
}

func TestLayoutPickerOverride(t *testing.T) {
	p := NewLayoutPicker()

	p.Override(LayoutNameThreeColumn)
	got := p.Reflow(60, 18).Name()
	if got != LayoutNameThreeColumn {
		t.Errorf("override ignored: %q", got)
	}

	p.Override("")
	got = p.Reflow(60, 18).Name()
	if got != LayoutNameSingle {
		t.Errorf("clearing override failed: %q", got)
	}
}

func TestLayoutPickerOverrideUnknownIgnored(t *testing.T) {
	p := NewLayoutPicker()
	p.Override("no-such-layout")
	got := p.Reflow(120, 28).Name()
	if got != LayoutNameTwoColumn {
		t.Errorf("unknown override should fall through to breakpoint: %q", got)
	}
}

func TestLayoutPickerBreakpoint(t *testing.T) {
	p := NewLayoutPicker()
	p.Reflow(200, 40)

	bp := p.Breakpoint()
	if bp.LayoutName != LayoutNameThreeColumn {
		t.Errorf("Breakpoint().LayoutName = %q, want %q", bp.LayoutName, LayoutNameThreeColumn)
	}
}
