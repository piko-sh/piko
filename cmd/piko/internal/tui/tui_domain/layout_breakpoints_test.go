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

func TestPickBreakpointDefaults(t *testing.T) {
	cases := []struct {
		name       string
		wantLayout string
		width      int
		height     int
	}{
		{name: "narrow", width: 60, height: 20, wantLayout: LayoutNameSingle},
		{name: "exactly single max", width: 99, height: 24, wantLayout: LayoutNameSingle},
		{name: "two threshold", width: 100, height: 24, wantLayout: LayoutNameTwoColumn},
		{name: "two band", width: 140, height: 28, wantLayout: LayoutNameTwoColumn},
		{name: "three threshold", width: 160, height: 30, wantLayout: LayoutNameThreeColumn},
		{name: "three wide", width: 220, height: 50, wantLayout: LayoutNameThreeColumn},
		{name: "wide but short", width: 220, height: 20, wantLayout: LayoutNameSingle},
		{name: "wide medium height", width: 220, height: 24, wantLayout: LayoutNameTwoColumn},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp := PickBreakpoint(DefaultBreakpoints, tc.width, tc.height)
			if bp.LayoutName != tc.wantLayout {
				t.Errorf("PickBreakpoint(%d, %d).LayoutName = %q, want %q",
					tc.width, tc.height, bp.LayoutName, tc.wantLayout)
			}
		})
	}
}

func TestPickBreakpointSmallestAlwaysMatches(t *testing.T) {
	bp := PickBreakpoint(DefaultBreakpoints, 0, 0)
	if bp.LayoutName != LayoutNameSingle {
		t.Errorf("expected single-pane fallback for 0x0, got %q", bp.LayoutName)
	}
}

func TestPickBreakpointNilUsesDefaults(t *testing.T) {
	bp := PickBreakpoint(nil, 200, 50)
	if bp.LayoutName != LayoutNameThreeColumn {
		t.Errorf("nil-breakpoints fallback failed: %q", bp.LayoutName)
	}
}
