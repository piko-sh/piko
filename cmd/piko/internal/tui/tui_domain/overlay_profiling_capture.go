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
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/wdk/clock"
)

const (
	// profilingCaptureOverlayID identifies the capture overlay on the
	// overlay-manager stack so the profiling panel can pop it when the
	// capture finishes without disturbing other overlays.
	profilingCaptureOverlayID = "profiling-capture"

	// captureOverlayMinWidth is the smallest width at which the capture
	// overlay remains legible.
	captureOverlayMinWidth = 40

	// captureOverlayMinHeight is the smallest height at which the
	// overlay's content fits without clipping.
	captureOverlayMinHeight = 7

	// captureProgressWidth is the cell width of the progress bar.
	captureProgressWidth = 32
)

// captureOverlaySpinnerFrames is the small Braille spinner cycle used
// while the capture is in flight. Six frames per second feels alive
// without flashing.
var captureOverlaySpinnerFrames = []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"}

// ProfilingCaptureOverlay is the modal shown during a one-shot
// profile capture.
//
// It displays the profile name, an elapsed-time progress bar against
// the total duration, and a small spinner. The overlay self-dismisses
// when the panel emits a popOverlayMessage on completion.
type ProfilingCaptureOverlay struct {
	// clock supplies the current time for progress calculation.
	clock clock.Clock

	// startedAt is the wall-clock instant the capture began.
	startedAt time.Time

	// profile names the profile being captured (e.g. "cpu", "heap").
	profile string

	// duration is the total length of the capture window.
	duration time.Duration

	// frame is the current spinner frame index.
	frame int

	// dismissed is true once the user has hidden the overlay.
	dismissed bool
}

var _ Overlay = (*ProfilingCaptureOverlay)(nil)

// NewProfilingCaptureOverlay constructs the overlay.
//
// Takes profile (string) which names the profile being captured.
// Takes duration (time.Duration) which is the sampling window.
// Takes c (clock.Clock); nil falls back to the real clock.
//
// Returns *ProfilingCaptureOverlay ready to push onto the manager.
func NewProfilingCaptureOverlay(profile string, duration time.Duration, c clock.Clock) *ProfilingCaptureOverlay {
	if c == nil {
		c = clock.RealClock()
	}
	return &ProfilingCaptureOverlay{
		clock:     c,
		profile:   profile,
		duration:  duration,
		startedAt: c.Now(),
	}
}

// ID identifies the overlay.
//
// Returns string which is the stable overlay identifier.
func (*ProfilingCaptureOverlay) ID() string { return profilingCaptureOverlayID }

// MinSize returns the smallest acceptable overlay dimensions.
//
// Returns width (int) which is the minimum legible width.
// Returns height (int) which is the minimum legible height.
func (*ProfilingCaptureOverlay) MinSize() (width, height int) {
	return captureOverlayMinWidth, captureOverlayMinHeight
}

// KeyMap describes the keys the overlay accepts.
//
// Returns []KeyBinding which is the overlay's key bindings.
func (*ProfilingCaptureOverlay) KeyMap() []KeyBinding {
	return []KeyBinding{
		{Key: "Esc", Description: "Hide the overlay (capture continues)"},
	}
}

// Dismiss reports whether the overlay should be popped.
//
// Returns bool which is true when the overlay has been dismissed.
func (o *ProfilingCaptureOverlay) Dismiss() bool { return o.dismissed }

// Update advances the spinner on each tick and dismisses on Esc.
//
// Takes msg (tea.Msg) which is the message routed to the overlay.
//
// Returns Overlay which is the (possibly updated) receiver.
// Returns tea.Cmd which is always nil; the overlay produces no
// commands of its own.
func (o *ProfilingCaptureOverlay) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyPressMsg:
		if m.String() == "esc" {
			o.dismissed = true
		}
	case TickMessage:
		o.frame = (o.frame + 1) % len(captureOverlaySpinnerFrames)
	}
	return o, nil
}

// Render produces the overlay body sized to the supplied dimensions.
//
// Takes width (int) which is the outer width of the overlay.
// Takes height (int) which is the outer height of the overlay.
//
// Returns string which is the rendered overlay frame.
func (o *ProfilingCaptureOverlay) Render(width, height int) string {
	innerWidth := max(1, width-PanelChromeWidth)

	elapsed := min(o.clock.Now().Sub(o.startedAt), o.duration)
	bar := captureProgressBar(elapsed, o.duration, captureProgressWidth)
	spinner := captureOverlaySpinnerFrames[o.frame%len(captureOverlaySpinnerFrames)]

	rows := []string{
		PadRightANSI(spinner+SingleSpace+"Capturing "+o.profile+" profile", innerWidth),
		"",
		PadRightANSI(bar, innerWidth),
		PadRightANSI(fmt.Sprintf("%s / %s elapsed", elapsed.Truncate(time.Millisecond), o.duration), innerWidth),
		"",
		PadRightANSI("Press Esc to hide (capture continues in the background)", innerWidth),
	}
	body := strings.Join(rows, "\n")

	return RenderPaneFrame(PaneFrameOpts{
		Title:   "Profile capture",
		Body:    body,
		Width:   width,
		Height:  height,
		Focused: true,
	})
}

// captureProgressBar renders the elapsed-vs-total progress bar.
//
// Takes elapsed (time.Duration), total (time.Duration), width (int)
// which sets the bar character width.
//
// Returns string with a "[#####.....]" style bar.
func captureProgressBar(elapsed, total time.Duration, width int) string {
	if width <= 2 {
		return ""
	}
	inner := width - 2
	filled := 0
	if total > 0 {
		filled = int(float64(elapsed) / float64(total) * float64(inner))
	}
	if filled < 0 {
		filled = 0
	}
	if filled > inner {
		filled = inner
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("·", inner-filled) + "]"
}
