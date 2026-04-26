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

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

// DetailView renders the detail-pane body for the history entry under
// the cursor, or a panel-level summary when nothing is selected.
//
// Takes width (int) and height (int) which are the inner dimensions
// of the detail pane.
//
// Returns string with the rendered body.
func (p *WatchdogHistoryPanel) DetailView(width, height int) string {
	body := p.buildDetailBody()
	return RenderDetailBody(p.theme, body, width, height)
}

// buildDetailBody assembles the structured detail content based on the
// current cursor target.
//
// Returns inspector.DetailBody describing the selected history entry or the overview.
func (p *WatchdogHistoryPanel) buildDetailBody() inspector.DetailBody {
	entries := p.visibleEntries()
	cursor := p.Cursor()
	if cursor >= 0 && cursor < len(entries) {
		return watchdogHistoryEntryDetailBody(entries[cursor], p.clock)
	}
	return p.historyOverviewDetailBody()
}

// watchdogHistoryEntryDetailBody renders detail for a single startup
// history entry.
//
// Takes e (WatchdogStartupEntry) which is the entry to render.
// Takes c (clock.Clock) which supplies the current time for duration calculations.
//
// Returns inspector.DetailBody describing the entry's PID, host, version, and lifecycle.
func watchdogHistoryEntryDetailBody(e WatchdogStartupEntry, c clock.Clock) inspector.DetailBody {
	state := "exited"
	if e.IsRunning() {
		state = "running"
	} else if e.IsUnclean() {
		state = "unclean"
	}

	rows := []inspector.DetailRow{
		{Label: "PID", Value: fmt.Sprintf(FormatPercentInt, e.PID)},
		{Label: "Hostname", Value: e.Hostname},
		{Label: "Version", Value: e.Version},
		{Label: "State", Value: state},
		{Label: "Reason", Value: defaultDash(e.Reason)},
		{Label: "Started", Value: inspector.FormatDetailTime(e.StartedAt)},
		{Label: "Stopped", Value: formatTimeOrDash(e.StoppedAt)},
		{Label: "Duration", Value: inspector.FormatDuration(e.Duration(c))},
	}
	if e.GomemlimitBytes > 0 {
		rows = append(rows, inspector.DetailRow{Label: "GOMEMLIMIT", Value: inspector.FormatBytes(uint64(e.GomemlimitBytes))})
	}

	return inspector.DetailBody{
		Title:    fmt.Sprintf("PID %d", e.PID),
		Subtitle: state + " · " + e.Hostname,
		Sections: []inspector.DetailSection{{Heading: "Startup entry", Rows: rows}},
	}
}

// historyOverviewDetailBody renders the panel-level summary.
//
// Returns inspector.DetailBody summarising entry counts and crash-loop thresholds.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogHistoryPanel) historyOverviewDetailBody() inspector.DetailBody {
	p.mu.RLock()
	entries := append([]WatchdogStartupEntry(nil), p.entries...)
	status := p.status
	p.mu.RUnlock()

	clean, unclean, running := 0, 0, 0
	for _, e := range entries {
		switch {
		case e.IsRunning():
			running++
		case e.IsUnclean():
			unclean++
		default:
			clean++
		}
	}

	rows := []inspector.DetailRow{
		{Label: "Total entries", Value: fmt.Sprintf(FormatPercentInt, len(entries))},
		{Label: "Running", Value: fmt.Sprintf(FormatPercentInt, running)},
		{Label: "Clean exits", Value: fmt.Sprintf(FormatPercentInt, clean)},
		{Label: "Unclean exits", Value: fmt.Sprintf(FormatPercentInt, unclean)},
	}
	if status != nil && status.CrashLoopThreshold > 0 {
		rows = append(rows,
			inspector.DetailRow{Label: "Crash-loop threshold", Value: fmt.Sprintf(FormatPercentInt, status.CrashLoopThreshold)},
			inspector.DetailRow{Label: "Crash-loop window", Value: status.CrashLoopWindow.String()},
		)
	}
	return inspector.DetailBody{
		Title:    "Startup history",
		Subtitle: fmt.Sprintf("%d entries", len(entries)),
		Sections: []inspector.DetailSection{{Heading: "Counts", Rows: rows}},
	}
}
