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

import "sync"

// MouseTargetKind classifies a hit-region registered with the MouseRouter.
type MouseTargetKind int

const (
	// MouseTargetPane covers a panel's rendered rectangle. Clicks focus
	// the panel; wheel events scroll it.
	MouseTargetPane MouseTargetKind = iota

	// MouseTargetTab covers a single tab in the tab bar. Clicks focus the
	// associated panel.
	MouseTargetTab

	// MouseTargetStatusBar covers the bottom-of-screen status bar.
	//
	// Clicks here are reserved for future segments (e.g. clickable
	// provider indicators).
	MouseTargetStatusBar
)

// MouseTarget describes a hit-region the router can resolve from a click
// or wheel event.
type MouseTarget struct {
	// PanelID identifies the panel whose region this target covers.
	PanelID string

	// Rect is the rectangular hit region in screen coordinates.
	Rect PaneRect

	// Kind classifies the target for routing.
	Kind MouseTargetKind
}

// MouseRouter buffers the set of hit-regions registered during the most
// recent render.
//
// The Model resets the router before rendering, layouts (and the Model
// itself for tabs) add targets via Add, and click/wheel handlers query
// Hit. Concurrent reads are safe; mutation must be single-writer.
type MouseRouter struct {
	// targets is the registered hit-region list, ordered as added.
	targets []MouseTarget

	// mu guards targets for safe concurrent reads alongside
	// single-writer mutation.
	mu sync.RWMutex
}

// NewMouseRouter creates an empty router.
//
// Returns *MouseRouter ready to receive Add calls.
func NewMouseRouter() *MouseRouter {
	return &MouseRouter{}
}

// Reset clears the registered targets. Called once per frame before the
// new render registers fresh ones.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *MouseRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targets = r.targets[:0]
}

// Add appends a target to the router. The first registered target whose
// rectangle contains the click coordinates wins; register foreground
// elements (overlays, tab bar) before background elements (panes) when
// they overlap.
//
// Takes target (MouseTarget) which is the hit-region to register.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *MouseRouter) Add(target MouseTarget) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.targets = append(r.targets, target)
}

// Hit returns the first registered target whose rectangle contains (x, y).
//
// The most-recently-added target wins, allowing layered chrome (e.g.
// tabs over panes) to be queried correctly when the registration order
// is "background first, foreground last".
//
// Takes x (int) which is the column.
// Takes y (int) which is the row.
//
// Returns MouseTarget which is the matched target (zero value when no
// hit).
// Returns bool which is true when a target contains (x, y).
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *MouseRouter) Hit(x, y int) (MouseTarget, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.targets) - 1; i >= 0; i-- {
		if rectContains(r.targets[i].Rect, x, y) {
			return r.targets[i], true
		}
	}
	return MouseTarget{}, false
}

// Targets returns a snapshot of the registered targets, primarily for
// tests and diagnostics.
//
// Returns []MouseTarget which is a copy of the current target slice.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (r *MouseRouter) Targets() []MouseTarget {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MouseTarget, len(r.targets))
	copy(out, r.targets)
	return out
}

// rectContains reports whether (x, y) falls within the half-open
// rectangle [Rect.X, Rect.X+Rect.Width) x [Rect.Y, Rect.Y+Rect.Height).
//
// Takes rect (PaneRect) which is the rectangle to test.
// Takes x (int) which is the column.
// Takes y (int) which is the row.
//
// Returns bool which is true when the point falls inside the rectangle.
func rectContains(rect PaneRect, x, y int) bool {
	if rect.Width <= 0 || rect.Height <= 0 {
		return false
	}
	return x >= rect.X && x < rect.X+rect.Width && y >= rect.Y && y < rect.Y+rect.Height
}
