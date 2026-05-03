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

import "errors"

const (
	// SingleSpace is a single ASCII space character. Used as a separator
	// in many breadcrumb, status, and key/value renderings.
	SingleSpace = " "

	// DoubleSpace is two ASCII spaces. Used as the standard indent for
	// list items where a single space would not be visually distinct.
	DoubleSpace = "  "

	// HyphenGlyph is a long dash placeholder for missing or unknown
	// numeric values.
	HyphenGlyph = "-"

	// FormatPercentInt is the common format for an integer percent column.
	FormatPercentInt = "%d"
)

const (
	// HoursPerDay is the number of hours in a day, used when collapsing a
	// time-since render into a coarse "Nd ago" form.
	HoursPerDay = 24
)

const (
	// CommandBarMaxLen is the upper bound on the command-bar input length.
	// Long enough for any practical command argument; short enough to keep
	// rendering tight.
	CommandBarMaxLen = 256

	// CommandBarVisibleWidth is the input field width in cells when the
	// command bar is open.
	CommandBarVisibleWidth = 40

	// CommandBarChromeWidth is the columns the command bar reserves for its
	// own chrome (prompt prefix and trailing label) before allocating the
	// rest of the row to the input field.
	CommandBarChromeWidth = 4

	// CommandBarPromptReserve is the width set aside for the prompt
	// glyph; the remaining width is given to the input field.
	CommandBarPromptReserve = 2

	// OverlayMinWidthChrome is the columns the overlay reserves for its
	// own border + padding chrome before allocating the rest of the
	// content.
	OverlayMinWidthChrome = 4

	// OverlayMinHeightChrome is the lines the overlay reserves for its
	// own border + padding chrome.
	OverlayMinHeightChrome = 5

	// OverlayDefaultWidth is the overlay width used when no caller-driven
	// preference is supplied.
	OverlayDefaultWidth = 32

	// OverlayDefaultHeight is the overlay height used when no caller-driven
	// preference is supplied.
	OverlayDefaultHeight = 8

	// HelpOverlayMinWidth is the minimum width the help overlay accepts.
	HelpOverlayMinWidth = 50

	// HelpOverlayMinHeight is the minimum height the help overlay accepts.
	HelpOverlayMinHeight = 12

	// overlayScreenNumerator is the numerator of the cap fraction that
	// keeps an overlay at 80% of the screen so the layout behind it
	// remains partly visible.
	overlayScreenNumerator = 4

	// overlayScreenDenominator is the denominator paired with
	// overlayScreenNumerator to compute the 4/5 = 80% screen cap.
	overlayScreenDenominator = 5

	// overlayScreenMargin is the smallest gap retained between an
	// overlay's edge and the screen edge.
	overlayScreenMargin = 2
)

const (
	// ThreeColumnMinPanes is the smallest pane count that triggers
	// three-column composition; below this the layout falls back to a
	// simpler form.
	ThreeColumnMinPanes = 3

	// ThreeColumnContextWidthDivisor divides the available width into
	// equal-share thirds for the context pane in three-column mode.
	ThreeColumnContextWidthDivisor = 3

	// AutoLayoutSwitchPaneCount is the pane count above which the
	// layout picker promotes from two-column to three-column.
	AutoLayoutSwitchPaneCount = 3
)

const (
	// EventStreamBufferSize is the buffered channel depth for streamed
	// watchdog events. Small enough to apply back-pressure quickly, large
	// enough to bridge typical render-loop pauses.
	EventStreamBufferSize = 64
)

const (
	// ConfigKeyMinWidth is the minimum width allocated to the key column
	// in a key/value config row.
	ConfigKeyMinWidth = 16

	// ConfigKeyMaxWidth is the maximum width allocated to the key column.
	ConfigKeyMaxWidth = 32
)

const (
	// SectionNavMinWidth is the minimum width the section nav column
	// occupies in the overview panel.
	SectionNavMinWidth = 14

	// SectionNavMaxWidth is the maximum width the section nav column
	// occupies in the overview panel.
	SectionNavMaxWidth = 22

	// AlertTapeMaxRows is the upper bound on how many alert tape rows the
	// overview panel will render.
	AlertTapeMaxRows = 16

	// overviewSectionsWidthDivisor divides the available width to compute
	// the section nav width before clamping to its min/max bounds.
	overviewSectionsWidthDivisor = 6

	// SparklineMinHistory is the minimum number of samples required before
	// rendering a sparkline; below this the sparkline is suppressed to
	// avoid misleading the viewer.
	SparklineMinHistory = 8
)

const (
	// HistoryColPID is the cell width of the PID column in the watchdog
	// history table.
	HistoryColPID = 8

	// HistoryColStarted is the cell width of the started-at column.
	HistoryColStarted = 21

	// HistoryColStopped is the cell width of the stopped-at column.
	HistoryColStopped = 21

	// HistoryColDuration is the cell width of the duration column.
	HistoryColDuration = 11

	// HistoryColReason is the cell width of the reason column.
	HistoryColReason = 11

	// HistoryColHostname is the cell width of the hostname column.
	HistoryColHostname = 16
)

const (
	// ProfilesColAge is the cell width of the age column in the
	// watchdog profiles table.
	ProfilesColAge = 10

	// ProfilesColType is the cell width of the profile-type column.
	ProfilesColType = 16

	// ProfilesColSize is the cell width of the size column.
	ProfilesColSize = 12

	// ProfilesColSidecar is the cell width of the sidecar-presence column.
	ProfilesColSidecar = 9

	// ProfilesColFilename is the cell width of the filename column.
	ProfilesColFilename = 8

	// ProfilesPaneSidecarWidth is the right-hand sidebar width devoted to
	// rendering the sidecar JSON tree.
	ProfilesPaneSidecarWidth = 8
)

const (
	// EventsTimeColumnWidth is the width of the timestamp column.
	EventsTimeColumnWidth = 8
)

// DispatcherJitterDivisor divides the computed jitter window before
// applying it to the next backoff delay.
const DispatcherJitterDivisor = 4

// FormatLatencyMs is the standard format for a millisecond latency
// value rendered as e.g. "12.3 ms".
const FormatLatencyMs = "%.1f ms"

// DetailRecentRowLimit caps how many "recent" items a detail pane
// renders inline (e.g. recent spans for a route, recent FDs for a
// resource category) before truncating.
const DetailRecentRowLimit = 5

// defaultRenderWidth is the fallback width used when the terminal has
// not yet reported its size. Set to the standard 80-column width so
// the first frame renders sensibly before the first WindowSizeMsg.
const defaultRenderWidth = 80

const (
	// SeverityCriticalThreshold is the percent above which utilisation
	// severity becomes critical.
	SeverityCriticalThreshold = 0.8

	// SeverityWarningThreshold is the percent above which utilisation
	// severity becomes warning.
	SeverityWarningThreshold = 0.6
)

// ErrWatchdogNoConnection is the sentinel returned when a watchdog
// provider call is made without a configured gRPC connection. Callers
// can detect it with errors.Is.
var ErrWatchdogNoConnection = errors.New("watchdog provider has no gRPC connection")
