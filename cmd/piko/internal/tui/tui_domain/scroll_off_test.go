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

func TestApplyScrollOffMaintainsMargin(t *testing.T) {
	testCases := []struct {
		name        string
		scrollIn    int
		cursor      int
		visible     int
		lineCount   int
		margin      int
		expectedOut int
	}{
		{
			name: "cursor at top with margin scrolls up",

			scrollIn: 2, cursor: 2, visible: 10, lineCount: 50, margin: 3,
			expectedOut: 0,
		},
		{
			name: "cursor at middle leaves scroll alone",

			scrollIn: 10, cursor: 15, visible: 10, lineCount: 50, margin: 3,
			expectedOut: 10,
		},
		{
			name: "cursor near bottom scrolls down",

			scrollIn: 10, cursor: 18, visible: 10, lineCount: 50, margin: 3,
			expectedOut: 12,
		},
		{
			name: "tiny viewport halves margin to fit",

			scrollIn: 10, cursor: 11, visible: 4, lineCount: 50, margin: 3,
			expectedOut: 10,
		},
		{
			name:     "clamps to zero when scrollOffset would go negative",
			scrollIn: 0, cursor: 0, visible: 10, lineCount: 50, margin: 3,
			expectedOut: 0,
		},
		{
			name:     "clamps to maxScroll when scrollOffset would exceed",
			scrollIn: 0, cursor: 49, visible: 10, lineCount: 50, margin: 3,
			expectedOut: 40,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyScrollOff(tc.scrollIn, tc.cursor, tc.visible, tc.lineCount, tc.margin)
			if got != tc.expectedOut {
				t.Errorf("ApplyScrollOff(%d, %d, %d, %d, %d) = %d, want %d",
					tc.scrollIn, tc.cursor, tc.visible, tc.lineCount, tc.margin, got, tc.expectedOut)
			}
		})
	}
}

func TestApplyScrollOffZeroMarginMatchesSnapToEdge(t *testing.T) {
	cases := []struct {
		scrollIn, cursor, visible, lineCount int
	}{
		{scrollIn: 0, cursor: 0, visible: 10, lineCount: 50},
		{scrollIn: 0, cursor: 9, visible: 10, lineCount: 50},
		{scrollIn: 5, cursor: 14, visible: 10, lineCount: 50},
		{scrollIn: 5, cursor: 4, visible: 10, lineCount: 50},
		{scrollIn: 0, cursor: 49, visible: 10, lineCount: 50},
	}
	for _, c := range cases {
		legacy := AdjustScrollForCursor(c.cursor, c.scrollIn, c.visible, c.lineCount)
		fresh := ApplyScrollOff(c.scrollIn, c.cursor, c.visible, c.lineCount, 0)
		if legacy != fresh {
			t.Errorf("zero-margin divergence at cursor=%d scrollIn=%d: legacy=%d, fresh=%d", c.cursor, c.scrollIn, legacy, fresh)
		}
	}
}

func TestApplyScrollOffInvalidViewport(t *testing.T) {
	if got := ApplyScrollOff(7, 3, 0, 50, 3); got != 7 {
		t.Errorf("expected pass-through on visible=0, got %d", got)
	}
	if got := ApplyScrollOff(7, 3, -1, 50, 3); got != 7 {
		t.Errorf("expected pass-through on visible<0, got %d", got)
	}
}

func TestAdjustScrollForCursorWithMarginDelegates(t *testing.T) {
	zeroMargin := AdjustScrollForCursorWithMargin(15, 10, 10, 50, 0)
	legacy := AdjustScrollForCursor(15, 10, 10, 50)
	if zeroMargin != legacy {
		t.Errorf("zero-margin wrapper diverged from legacy: %d vs %d", zeroMargin, legacy)
	}

	withMargin := AdjustScrollForCursorWithMargin(18, 10, 10, 50, 3)
	expected := ApplyScrollOff(10, 18, 10, 50, 3)
	if withMargin != expected {
		t.Errorf("margin wrapper diverged from ApplyScrollOff: %d vs %d", withMargin, expected)
	}
}
