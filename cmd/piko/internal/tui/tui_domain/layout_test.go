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

func TestSinglePaneLayoutAllocate(t *testing.T) {
	layout := NewSinglePaneLayout()

	rects := layout.Allocate(80, 24, 1)
	if len(rects) != 1 {
		t.Fatalf("Allocate returned %d rects, want 1", len(rects))
	}
	if rects[0].Width != 80 || rects[0].Height != 24 {
		t.Errorf("rect = %+v, want full coverage", rects[0])
	}

	if rects := layout.Allocate(10, 24, 1); len(rects) != 0 {
		t.Errorf("expected empty allocation when width below MinSinglePaneWidth")
	}
	if rects := layout.Allocate(80, 2, 1); len(rects) != 0 {
		t.Errorf("expected empty allocation when height below MinPaneHeight")
	}
	if rects := layout.Allocate(80, 24, 0); len(rects) != 0 {
		t.Errorf("expected empty allocation for zero panes")
	}
}

func TestTwoColumnLayoutAllocate(t *testing.T) {
	layout := NewTwoColumnLayout()

	rects := layout.Allocate(120, 30, 2)
	if len(rects) != 2 {
		t.Fatalf("Allocate returned %d rects, want 2", len(rects))
	}
	if rects[0].Width+rects[1].Width != 120 {
		t.Errorf("widths %d + %d != 120", rects[0].Width, rects[1].Width)
	}
	if rects[0].Width < MinPaneWidth || rects[1].Width < MinPaneWidth {
		t.Errorf("widths below minimum: %+v / %+v", rects[0], rects[1])
	}
	if rects[1].X != rects[0].Width {
		t.Errorf("second pane should start at end of first: x=%d, want %d", rects[1].X, rects[0].Width)
	}

	rects = layout.Allocate(40, 30, 2)
	if len(rects) != 1 {
		t.Errorf("expected single-pane fallback, got %d rects", len(rects))
	}
}

func TestThreeColumnLayoutAllocate(t *testing.T) {
	layout := NewThreeColumnLayout()

	rects := layout.Allocate(180, 40, 3)
	if len(rects) != 3 {
		t.Fatalf("Allocate returned %d rects, want 3", len(rects))
	}
	total := rects[0].Width + rects[1].Width + rects[2].Width
	if total != 180 {
		t.Errorf("widths sum = %d, want 180", total)
	}
	for _, r := range rects {
		if r.Width < MinPaneWidth {
			t.Errorf("pane below MinPaneWidth: %+v", r)
		}
	}

	rects = layout.Allocate(60, 30, 3)
	if len(rects) != 2 {
		t.Errorf("expected two-column fallback at width=60, got %d rects", len(rects))
	}

	rects = layout.Allocate(40, 30, 3)
	if len(rects) != 1 {
		t.Errorf("expected single-pane fallback at width=40, got %d rects", len(rects))
	}
}

func TestLayoutCompose(t *testing.T) {
	single := NewSinglePaneLayout().Compose([]RenderedPane{{Body: "hello"}}, 80, 24)
	if single != "hello" {
		t.Errorf("single Compose = %q, want %q", single, "hello")
	}

	two := NewTwoColumnLayout().Compose([]RenderedPane{
		{Body: "AAA"},
		{Body: "BBB"},
	}, 80, 24)
	if !contains(two, "AAA") || !contains(two, "BBB") {
		t.Errorf("two-column compose missing panel bodies: %q", two)
	}
}

func TestLayoutMaxPanes(t *testing.T) {
	if got := NewSinglePaneLayout().MaxPanes(); got != 1 {
		t.Errorf("single MaxPanes = %d, want 1", got)
	}
	if got := NewTwoColumnLayout().MaxPanes(); got != 2 {
		t.Errorf("two MaxPanes = %d, want 2", got)
	}
	if got := NewThreeColumnLayout().MaxPanes(); got != 3 {
		t.Errorf("three MaxPanes = %d, want 3", got)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
