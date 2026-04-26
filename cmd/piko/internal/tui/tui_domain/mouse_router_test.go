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
	"sync"
	"testing"
)

func TestMouseRouterEmpty(t *testing.T) {
	r := NewMouseRouter()
	if _, ok := r.Hit(5, 5); ok {
		t.Errorf("empty router should not report a hit")
	}
}

func TestMouseRouterHit(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{
		Rect:    PaneRect{X: 10, Y: 5, Width: 20, Height: 8},
		PanelID: "primary",
		Kind:    MouseTargetPane,
	})

	target, ok := r.Hit(15, 7)
	if !ok {
		t.Fatalf("Hit at (15,7) should match, got miss")
	}
	if target.PanelID != "primary" {
		t.Errorf("PanelID = %q, want primary", target.PanelID)
	}
}

func TestMouseRouterMissOutside(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{
		Rect: PaneRect{X: 10, Y: 5, Width: 20, Height: 8},
	})

	cases := []struct{ x, y int }{
		{0, 0},
		{9, 5},
		{30, 5},
		{15, 4},
		{15, 13},
	}
	for _, c := range cases {
		if _, ok := r.Hit(c.x, c.y); ok {
			t.Errorf("Hit(%d,%d) should miss", c.x, c.y)
		}
	}
}

func TestMouseRouterLatestWins(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{Rect: PaneRect{X: 0, Y: 0, Width: 50, Height: 20}, PanelID: "background", Kind: MouseTargetPane})
	r.Add(MouseTarget{Rect: PaneRect{X: 5, Y: 5, Width: 10, Height: 5}, PanelID: "foreground", Kind: MouseTargetTab})

	target, ok := r.Hit(7, 7)
	if !ok || target.PanelID != "foreground" {
		t.Errorf("expected foreground to win, got %+v ok=%v", target, ok)
	}
}

func TestMouseRouterReset(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{Rect: PaneRect{X: 0, Y: 0, Width: 10, Height: 10}})
	r.Reset()
	if len(r.Targets()) != 0 {
		t.Errorf("Reset should empty target list, got %d", len(r.Targets()))
	}
	if _, ok := r.Hit(1, 1); ok {
		t.Errorf("Hit after Reset should miss")
	}
}

func TestMouseRouterEmptyRect(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{Rect: PaneRect{X: 0, Y: 0, Width: 0, Height: 10}})
	r.Add(MouseTarget{Rect: PaneRect{X: 0, Y: 0, Width: 10, Height: 0}})
	if _, ok := r.Hit(0, 0); ok {
		t.Errorf("zero-area rectangles should not hit")
	}
}

func TestMouseRouterConcurrent(t *testing.T) {
	r := NewMouseRouter()
	r.Add(MouseTarget{Rect: PaneRect{X: 0, Y: 0, Width: 10, Height: 10}, PanelID: "x"})

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			_, _ = r.Hit(5, 5)
		})
		wg.Go(func() {
			_ = r.Targets()
		})
	}
	wg.Wait()
}
