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
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogHistoryPanelID identifies the watchdog History panel.
	WatchdogHistoryPanelID = "watchdog-history"

	// WatchdogHistoryPanelTitle is the display title.
	WatchdogHistoryPanelTitle = "Watchdog History"
)

// historyFilter selects which subset of history entries the table shows.
type historyFilter int

const (
	// historyFilterAll keeps every entry.
	historyFilterAll historyFilter = iota

	// historyFilterClean keeps entries with reason "clean".
	historyFilterClean

	// historyFilterUnclean keeps entries marked unclean or panic.
	historyFilterUnclean

	// historyFilterRunning keeps entries that have not yet stopped.
	historyFilterRunning
)

// historyFilterCount is the number of distinct history filters cycled
// through by the `f` key.
const historyFilterCount = 4

// historySnapshotMsg carries a refreshed history list. Err is non-nil
// when either RPC failed; the panel surfaces it as a banner.
type historySnapshotMsg struct {
	// Status is the latest watchdog status snapshot.
	Status *WatchdogStatus

	// Err is the fetch error, or nil on success.
	Err error

	// Entries is the latest startup history list.
	Entries []WatchdogStartupEntry
}

// WatchdogHistoryPanel renders the startup-history ring including a
// crash-loop indicator derived from the current watchdog status.
type WatchdogHistoryPanel struct {
	// provider supplies startup-history and status snapshots.
	provider WatchdogProvider

	// clock yields the current time for window calculations.
	clock clock.Clock

	// lastFetchErr is the most recent fetch error, or nil after success.
	lastFetchErr error

	// theme is the active theme used to render styles.
	theme *Theme

	// status is the cached watchdog status used for the crash-loop banner.
	status *WatchdogStatus

	// entries is the cached startup history list.
	entries []WatchdogStartupEntry

	BasePanel

	// filter is the currently active history filter.
	filter historyFilter

	// mu guards lastFetchErr, status, and entries.
	mu sync.RWMutex
}

// Compile-time interface assertions.
var (
	_ Panel = (*WatchdogHistoryPanel)(nil)

	_ ThemeAware = (*WatchdogHistoryPanel)(nil)
)

// NewWatchdogHistoryPanel constructs the History panel.
//
// Takes provider (WatchdogProvider) which supplies the history list.
// Takes clk (clock.Clock) which yields the current time.
//
// Returns *WatchdogHistoryPanel ready for AddPanel.
func NewWatchdogHistoryPanel(provider WatchdogProvider, clk clock.Clock) *WatchdogHistoryPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogHistoryPanel{
		BasePanel: NewBasePanel(WatchdogHistoryPanelID, WatchdogHistoryPanelTitle),
		provider:  provider,
		clock:     clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "j / Down", Description: "Next entry"},
		{Key: "k / Up", Description: "Previous entry"},
		{Key: "g", Description: "Top"},
		{Key: "G", Description: "Bottom"},
		{Key: "f", Description: "Cycle filter"},
		{Key: "R", Description: "Refresh"},
	})
	return panel
}

// SetTheme implements ThemeAware.
//
// Takes theme (*Theme) which becomes the active theme.
func (p *WatchdogHistoryPanel) SetTheme(theme *Theme) { p.theme = theme }

// Init kicks off the first history fetch.
//
// Returns tea.Cmd which schedules the first fetch.
func (p *WatchdogHistoryPanel) Init() tea.Cmd { return p.fetchCmd() }

// Update handles tick, snapshot, and key messages.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogHistoryPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case historySnapshotMsg:
		p.mu.Lock()
		if msg.Err == nil {
			p.entries = msg.Entries
			p.status = msg.Status
		}
		p.lastFetchErr = msg.Err
		p.mu.Unlock()
		p.clampCursor()
	case TickMessage:
		cmd := p.fetchCmd()
		return p, cmd
	case tea.KeyPressMsg:
		if cmd := p.handleKey(msg); cmd != nil {
			return p, cmd
		}
	}
	return p, nil
}

// View renders the panel sized to width x height.
//
// Takes width (int) which is the column width for the rendered frame.
// Takes height (int) which is the row height for the rendered frame.
//
// Returns string with the rendered frame.
func (p *WatchdogHistoryPanel) View(width, height int) string {
	p.SetSize(width, height)

	cw := p.ContentWidth()
	ch := p.ContentHeight()
	if cw <= 0 || ch <= 0 {
		return p.RenderFrame("")
	}
	body := p.composeBody(cw, ch)
	return p.RenderFrame(body)
}

// composeBody arranges the crash-loop banner, header, table, and
// footer.
//
// Takes width (int) which is the body width.
// Takes height (int) which is the body height.
//
// Returns string with the assembled body.
func (p *WatchdogHistoryPanel) composeBody(width, height int) string {
	rows := make([]string, 0, height)
	rows = append(rows,
		p.renderHeaderRow(width),
		p.renderCrashLoopRow(width),
		"",
		p.renderTableHeader(width),
	)

	listHeight := max(height-5, 1)
	rows = append(rows, p.renderRows(width, listHeight)...)

	for len(rows) < height-1 {
		rows = append(rows, strings.Repeat(" ", width))
	}
	rows = append(rows, p.renderFooterRow(width))

	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderHeaderRow shows the panel title and the active filter.
//
// Takes width (int) which is the row width.
//
// Returns string with the assembled row.
func (p *WatchdogHistoryPanel) renderHeaderRow(width int) string {
	left := p.boldStyle().Render(WatchdogHistoryPanelTitle)
	right := p.dimStyle().Render("filter: " + p.filterLabel())
	gap := max(1, width-TextWidth(left)-TextWidth(right))
	return PadRightANSI(left+strings.Repeat(" ", gap)+right, width)
}

// renderCrashLoopRow shows the crash-loop indicator. Detected when the
// number of unclean entries within CrashLoopWindow meets the configured
// threshold from the watchdog status.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered indicator row.
func (p *WatchdogHistoryPanel) renderCrashLoopRow(width int) string {
	status := p.snapshotStatus()
	if status == nil {
		return PadRightANSI(p.dimStyle().Render("crash loop: unknown (status unavailable)"), width)
	}

	uncleanCount, windowStart := p.uncleanWithinWindow(status.CrashLoopWindow)
	threshold := status.CrashLoopThreshold

	if threshold > 0 && uncleanCount >= threshold {
		body := fmt.Sprintf("⚠ crash loop detected: %d unclean exits since %s",
			uncleanCount, inspector.FormatTimeSince(p.clock.Now(), windowStart))
		return PadRightANSI(p.errorStyle().Render(body), width)
	}
	body := fmt.Sprintf("✓ no crash loop · %d unclean within window (threshold %d)",
		uncleanCount, threshold)
	return PadRightANSI(p.healthyStyle().Render(body), width)
}

// renderTableHeader returns the PID / STARTED / STOPPED / DURATION /
// REASON / VERSION header row.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered header row.
func (p *WatchdogHistoryPanel) renderTableHeader(width int) string {
	cols := p.columnLayout(width)
	header := p.dimStyle().Render(p.composeColumns(
		[]string{"PID", "STARTED", "STOPPED", "DURATION", "REASON", "HOST", "VERSION"},
		cols,
	))
	return PadRightANSI(header, width)
}

// renderRows returns up to height rows of filtered history.
//
// Takes width (int) which is the row width.
// Takes height (int) which is the maximum number of rows to return.
//
// Returns []string which is the rendered list rows.
func (p *WatchdogHistoryPanel) renderRows(width, height int) []string {
	entries := p.visibleEntries()
	if len(entries) == 0 {
		return []string{p.dimStyle().Render(PadRightANSI("No history entries match the current filter.", width))}
	}

	cursor := p.Cursor()
	if cursor >= len(entries) {
		cursor = len(entries) - 1
		p.SetCursor(cursor)
	}

	cols := p.columnLayout(width)
	startIdx := p.ScrollOffset()
	endIdx := min(startIdx+height, len(entries))

	rows := make([]string, 0, height)
	for i := startIdx; i < endIdx; i++ {
		entry := entries[i]
		stopLabel := "-"
		if !entry.StoppedAt.IsZero() {
			stopLabel = entry.StoppedAt.Format("2006-01-02 15:04:05")
		}
		duration := inspector.FormatDuration(entry.Duration(p.clock))
		reasonLabel := entry.Reason
		if reasonLabel == "" {
			reasonLabel = "running"
		}
		row := p.composeColumns(
			[]string{
				strconv.Itoa(entry.PID),
				entry.StartedAt.Format("2006-01-02 15:04:05"),
				stopLabel,
				duration,
				reasonLabel,
				entry.Hostname,
				entry.Version,
			},
			cols,
		)
		row = p.styleReasonRow(entry, row)
		if i == cursor {
			row = p.cursorStyle().Render(PadRightANSI(row, width))
		}
		rows = append(rows, PadRightANSI(row, width))
	}
	return rows
}

// styleReasonRow applies a colour to entries that exited uncleanly.
//
// Takes entry (WatchdogStartupEntry) which is the source entry.
// Takes row (string) which is the pre-rendered row text.
//
// Returns string which is the row with reason-specific styling applied.
func (p *WatchdogHistoryPanel) styleReasonRow(entry WatchdogStartupEntry, row string) string {
	if entry.IsUnclean() {
		return p.warningStyle().Render(row)
	}
	return row
}

// renderFooterRow shows a short hint.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered footer row.
func (p *WatchdogHistoryPanel) renderFooterRow(width int) string {
	hint := "f to cycle filter · R to refresh"
	return PadRightANSI(p.dimStyle().Render(hint), width)
}

// columnLayout computes the per-column widths.
//
// Takes width (int) which is the total row width.
//
// Returns []int which is the per-column widths in order.
func (*WatchdogHistoryPanel) columnLayout(width int) []int {
	used := HistoryColPID + HistoryColStarted + HistoryColStopped + HistoryColDuration + HistoryColReason + HistoryColHostname
	versionWidth := max(HistoryColPID, width-used)
	return []int{HistoryColPID, HistoryColStarted, HistoryColStopped, HistoryColDuration, HistoryColReason, HistoryColHostname, versionWidth}
}

// composeColumns formats cells into a padded row.
//
// Takes cells ([]string) which are the per-column values.
// Takes widths ([]int) which are the per-column widths.
//
// Returns string with the cells padded and joined.
func (*WatchdogHistoryPanel) composeColumns(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		w := 0
		if i < len(widths) {
			w = widths[i]
		}
		if w <= 0 {
			parts[i] = cell
			continue
		}
		parts[i] = PadRightANSI(TruncateANSI(cell, w-1), w)
	}
	return strings.Join(parts, " ")
}

// visibleEntries returns the filtered history entries, newest-first.
//
// Returns []WatchdogStartupEntry which is the filtered slice in
// newest-first order.
func (p *WatchdogHistoryPanel) visibleEntries() []WatchdogStartupEntry {
	all := p.snapshotEntries()
	out := make([]WatchdogStartupEntry, 0, len(all))
	for _, entry := range all {
		switch p.filter {
		case historyFilterClean:
			if entry.Reason != "clean" {
				continue
			}
		case historyFilterUnclean:
			if !entry.IsUnclean() {
				continue
			}
		case historyFilterRunning:
			if !entry.IsRunning() {
				continue
			}
		}
		out = append(out, entry)
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// snapshotEntries returns a copy of the cached entries.
//
// Returns []WatchdogStartupEntry which is a snapshot copy.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogHistoryPanel) snapshotEntries() []WatchdogStartupEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]WatchdogStartupEntry, len(p.entries))
	copy(out, p.entries)
	return out
}

// snapshotStatus returns the cached watchdog status, used for the
// crash-loop banner.
//
// Returns *WatchdogStatus which is the cached snapshot, or nil when
// none is available.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogHistoryPanel) snapshotStatus() *WatchdogStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// uncleanWithinWindow counts unclean entries since now-window.
//
// Takes window (time.Duration) which is the look-back period.
//
// Returns int which is the count of unclean exits.
// Returns time.Time which is the start of the window.
func (p *WatchdogHistoryPanel) uncleanWithinWindow(window time.Duration) (int, time.Time) {
	if window <= 0 {
		return 0, p.clock.Now()
	}
	cutoff := p.clock.Now().Add(-window)
	count := 0
	for _, entry := range p.snapshotEntries() {
		if entry.StartedAt.Before(cutoff) {
			continue
		}
		if entry.IsUnclean() {
			count++
		}
	}
	return count, cutoff
}

// filterLabel returns the human label for the active filter.
//
// Returns string which is the active filter's label.
func (p *WatchdogHistoryPanel) filterLabel() string {
	switch p.filter {
	case historyFilterClean:
		return "clean"
	case historyFilterUnclean:
		return "unclean"
	case historyFilterRunning:
		return "running"
	default:
		return "all"
	}
}

// fetchCmd asks the provider for fresh history and status snapshots.
//
// Returns tea.Cmd which produces a historySnapshotMsg, or nil when no
// provider is configured.
func (p *WatchdogHistoryPanel) fetchCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, errors.New("watchdog history fetch timed out"))
		defer cancel()
		entries, historyErr := p.provider.GetStartupHistory(ctx)
		status, statusErr := p.provider.GetStatus(ctx)
		err := historyErr
		if err == nil {
			err = statusErr
		}
		return historySnapshotMsg{Entries: entries, Status: status, Err: err}
	}
}

// handleKey processes panel-specific keys.
//
// Takes message (tea.KeyPressMsg) which is the key event.
//
// Returns tea.Cmd which schedules any follow-up command, or nil.
func (p *WatchdogHistoryPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "f":
		p.cycleFilter()
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return nil
	case "R":
		return p.fetchCmd()
	}
	if p.HandleNavigation(message, len(p.visibleEntries())) {
		return nil
	}
	return nil
}

// cycleFilter advances the filter through (all, clean, unclean,
// running, all).
func (p *WatchdogHistoryPanel) cycleFilter() {
	p.filter = (p.filter + 1) % historyFilterCount
}

// clampCursor adjusts the cursor when data shrinks.
func (p *WatchdogHistoryPanel) clampCursor() {
	visible := len(p.visibleEntries())
	if visible == 0 {
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return
	}
	if p.Cursor() >= visible {
		p.SetCursor(visible - 1)
	}
}

// boldStyle returns the bold-text style.
//
// Returns lipgloss.Style which is the themed bold style or a fallback.
func (p *WatchdogHistoryPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim-text style.
//
// Returns lipgloss.Style which is the themed dim style or a fallback.
func (p *WatchdogHistoryPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// healthyStyle returns the healthy-status style.
//
// Returns lipgloss.Style which is the themed healthy style or a fallback.
func (p *WatchdogHistoryPanel) healthyStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusHealthy
	}
	return statusHealthyStyle
}

// warningStyle returns the warning-status style.
//
// Returns lipgloss.Style which is the themed warning style or a fallback.
func (p *WatchdogHistoryPanel) warningStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusDegraded
	}
	return statusDegradedStyle
}

// errorStyle returns the error-status style.
//
// Returns lipgloss.Style which is the themed error style or a fallback.
func (p *WatchdogHistoryPanel) errorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusUnhealthy
	}
	return statusUnhealthyStyle
}

// cursorStyle returns the selection style.
//
// Returns lipgloss.Style which is the themed selection style or a fallback.
func (p *WatchdogHistoryPanel) cursorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Selected
	}
	return navItemActiveStyle
}
