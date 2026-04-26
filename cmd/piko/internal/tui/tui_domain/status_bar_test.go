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
)

func TestStatusBarRendererThreeSegments(t *testing.T) {
	r := NewStatusBarRenderer(nil)
	got := r.Render(StatusBarSegments{
		Left:   "q quit",
		Middle: "Refreshed",
		Right:  "● connected",
	}, 80)

	if !strings.Contains(got, "q quit") {
		t.Errorf("missing left segment: %q", got)
	}
	if !strings.Contains(got, "Refreshed") {
		t.Errorf("missing middle segment: %q", got)
	}
	if !strings.Contains(got, "connected") {
		t.Errorf("missing right segment: %q", got)
	}
	if w := TextWidth(got); w != 80 {
		t.Errorf("rendered width = %d, want 80", w)
	}
}

func TestStatusBarRendererDropsMiddleWhenTight(t *testing.T) {
	r := NewStatusBarRenderer(nil)
	got := r.Render(StatusBarSegments{
		Left:   "q quit | r refresh | ? help",
		Middle: "this middle has too much content for 30 cols",
		Right:  "● connected",
	}, 30)

	if strings.Contains(got, "too much content") {
		t.Errorf("middle should be dropped when tight: %q", got)
	}
	if !strings.Contains(got, "connected") {
		t.Errorf("right segment should survive tight render: %q", got)
	}
	if w := TextWidth(got); w != 30 {
		t.Errorf("rendered width = %d, want 30", w)
	}
}

func TestStatusBarRendererTruncatesLeftWhenVeryTight(t *testing.T) {
	r := NewStatusBarRenderer(nil)
	got := r.Render(StatusBarSegments{
		Left:   "q quit | r refresh | ? help | tab cycle | g top",
		Middle: "",
		Right:  "● connected to remote endpoint",
	}, 40)

	if w := TextWidth(got); w != 40 {
		t.Errorf("rendered width = %d, want 40", w)
	}
}

func TestStatusBarRendererZeroWidth(t *testing.T) {
	r := NewStatusBarRenderer(nil)
	if got := r.Render(StatusBarSegments{Left: "x"}, 0); got != "" {
		t.Errorf("zero-width render should return empty, got %q", got)
	}
}
