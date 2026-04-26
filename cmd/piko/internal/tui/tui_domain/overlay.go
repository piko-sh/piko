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
	tea "charm.land/bubbletea/v2"
)

// Overlay is a transient view rendered on top of the main layout.
//
// Includes help screens, confirmation dialogues, and detail popups. Overlays
// are stacked in an OverlayManager; the topmost overlay receives input until
// it is dismissed.
type Overlay interface {
	// ID identifies the overlay for diagnostic purposes and to allow
	// callers to query the stack for a specific kind.
	ID() string

	// Render returns the framed overlay body.
	//
	// The OverlayManager is responsible for compositing this body over the
	// background; the Overlay itself just produces the inner content. The
	// supplied dimensions are the overlay's own width and height, not the
	// screen.
	Render(width, height int) string

	// Update processes a message and returns the (possibly updated)
	// Overlay plus any commands. Returning a different value from the
	// receiver allows immutable update patterns; returning the receiver
	// is the conventional in-place form.
	Update(msg tea.Msg) (Overlay, tea.Cmd)

	// KeyMap returns key bindings the overlay accepts; surfaced in the
	// help screen and the status bar hint area.
	KeyMap() []KeyBinding

	// Dismiss reports whether the overlay should be popped after the most
	// recent Update. Implementations typically set an internal flag on
	// Esc/Enter and return it here.
	Dismiss() bool

	// MinSize suggests the smallest dimensions at which the overlay
	// remains usable. The OverlayManager grows the overlay to fit its
	// content but will not shrink it below this minimum.
	MinSize() (width, height int)
}

// OverlayManager maintains a stack of overlays. The top of the stack
// receives input; underlying overlays remain visible (composited) but
// inactive.
type OverlayManager struct {
	// theme drives the overlay frame and dim styles for compositing.
	theme *Theme

	// stack is the ordered overlay stack; the last entry is the active one.
	stack []Overlay
}

// NewOverlayManager creates an empty manager bound to the supplied theme.
//
// Takes theme (*Theme) which provides border and dim styles for
// composition.
//
// Returns *OverlayManager which has no overlays pushed.
func NewOverlayManager(theme *Theme) *OverlayManager {
	return &OverlayManager{theme: theme}
}

// SetTheme updates the theme used for compositing. Called when the user
// switches themes via the command bar.
//
// Takes theme (*Theme) which becomes the new theme.
func (m *OverlayManager) SetTheme(theme *Theme) {
	m.theme = theme
}

// Push adds an overlay to the top of the stack. Subsequent input is routed
// to it until Pop or Dismiss is called.
//
// Takes overlay (Overlay) which is the overlay to show.
func (m *OverlayManager) Push(overlay Overlay) {
	if overlay == nil {
		return
	}
	m.stack = append(m.stack, overlay)
}

// Pop removes the topmost overlay and returns it. Returns nil when the
// stack is empty.
//
// Returns Overlay which was at the top of the stack, or nil.
func (m *OverlayManager) Pop() Overlay {
	if len(m.stack) == 0 {
		return nil
	}
	top := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return top
}

// Top returns the overlay currently receiving input, or nil when the stack
// is empty.
//
// Returns Overlay which is the active overlay.
func (m *OverlayManager) Top() Overlay {
	if len(m.stack) == 0 {
		return nil
	}
	return m.stack[len(m.stack)-1]
}

// Empty reports whether the overlay stack is empty.
//
// Returns bool which is true when no overlays are pushed.
func (m *OverlayManager) Empty() bool {
	return len(m.stack) == 0
}

// Update routes a message to the topmost overlay. Returns any command the
// overlay produced, plus a flag indicating whether the message was
// consumed; consumed messages are not forwarded to the underlying panels.
//
// Takes msg (tea.Msg) which is the message to dispatch.
//
// Returns tea.Cmd which is the resulting command from the active overlay.
// Returns bool which is true when an overlay was active and consumed the
// message.
func (m *OverlayManager) Update(msg tea.Msg) (tea.Cmd, bool) {
	if len(m.stack) == 0 {
		return nil, false
	}

	index := len(m.stack) - 1
	updated, cmd := m.stack[index].Update(msg)
	if updated != nil {
		m.stack[index] = updated
	}

	if updated != nil && updated.Dismiss() {
		m.stack = m.stack[:index]
	}

	return cmd, true
}

// Render composites every pushed overlay over the supplied background.
//
// The returned string is the same shape as the background. When the stack is
// empty the background is returned unchanged.
//
// Takes background (string) which is the rendered layout body.
// Takes screenWidth (int) which is the terminal width.
// Takes screenHeight (int) which is the terminal height.
//
// Returns string which is background with overlays composited over it.
func (m *OverlayManager) Render(background string, screenWidth, screenHeight int) string {
	if len(m.stack) == 0 {
		return background
	}

	current := background
	for _, overlay := range m.stack {
		w, h := overlaySize(overlay, screenWidth, screenHeight)
		body := overlay.Render(w, h)
		current = ComposeOverlay(current, body, screenWidth, screenHeight, m.theme)
	}
	return current
}

// overlaySize chooses dimensions for an overlay given its minimum and the
// available screen. Caps at 80% of the screen so the underlying layout is
// always partly visible.
//
// Takes overlay (Overlay) which provides its preferred minimum size.
// Takes screenWidth (int) which is the available width.
// Takes screenHeight (int) which is the available height.
//
// Returns width (int) which is the overlay width.
// Returns height (int) which is the overlay height.
func overlaySize(overlay Overlay, screenWidth, screenHeight int) (width, height int) {
	minW, minH := overlay.MinSize()
	if minW <= 0 {
		minW = OverlayDefaultWidth
	}
	if minH <= 0 {
		minH = OverlayDefaultHeight
	}

	w := minW
	if maxW := screenWidth * overlayScreenNumerator / overlayScreenDenominator; maxW > w {
		w = maxW
	}
	if w > screenWidth-overlayScreenMargin {
		w = screenWidth - overlayScreenMargin
	}
	w = max(w, minW)

	h := minH
	if maxH := screenHeight * overlayScreenNumerator / overlayScreenDenominator; maxH > h {
		h = maxH
	}
	if h > screenHeight-overlayScreenMargin {
		h = screenHeight - overlayScreenMargin
	}
	h = max(h, minH)

	return w, h
}
