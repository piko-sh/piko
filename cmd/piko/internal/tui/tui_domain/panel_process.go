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
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// processHistorySize bounds the per-metric history kept for the
	// detail-pane chart.
	processHistorySize = 900

	// processChartSeriesCap is the maximum number of distinct chart
	// series the detail-pane composer ever allocates (RSS, FDs,
	// Threads).
	processChartSeriesCap = 3
)

// ProcessPanel surfaces process-level metadata (PID, threads, FDs, RSS).
//
// It is the TUI counterpart of `piko info process`. Implements Panel.
type ProcessPanel struct {
	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the SystemStats fetched on each refresh.
	provider SystemProvider

	// err holds the last refresh error, or nil after success.
	err error

	// stats holds the most recent stats snapshot.
	stats *SystemStats

	// threadHistory holds thread-count samples for the detail chart.
	threadHistory *HistoryRing

	// fdHistory holds open-FD count samples for the detail chart.
	fdHistory *HistoryRing

	// rssHistory holds resident-set-size samples for the detail chart.
	rssHistory *HistoryRing

	BasePanel

	// stateMutex guards stats / err / history for safe concurrent reads.
	stateMutex sync.RWMutex
}

var _ Panel = (*ProcessPanel)(nil)

// NewProcessPanel constructs a ProcessPanel sharing the supplied
// SystemProvider.
//
// Takes provider (SystemProvider) which supplies system statistics.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *ProcessPanel ready to register with a group.
func NewProcessPanel(provider SystemProvider, c clock.Clock) *ProcessPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &ProcessPanel{
		BasePanel:     NewBasePanel("process", titleProcess),
		clock:         c,
		provider:      provider,
		threadHistory: NewHistoryRing(processHistorySize),
		fdHistory:     NewHistoryRing(processHistorySize),
		rssHistory:    NewHistoryRing(processHistorySize),
		stateMutex:    sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which schedules the first stats fetch.
func (p *ProcessPanel) Init() tea.Cmd { return p.refresh() }

// Update reacts to refreshes and the 'r' refresh key.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
func (p *ProcessPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "r" {
			cmd := p.refresh()
			return p, cmd
		}
	case systemStatsMessage:
		p.handleStats(msg)
		return p, nil
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the panel within (width, height).
//
// Takes width (int) which is the column width allocated by the layout.
// Takes height (int) which is the row height allocated by the layout.
//
// Returns string with the rendered panel.
func (p *ProcessPanel) View(width, height int) string {
	p.SetSize(width, height)
	snap := p.snapshot()
	body := p.renderBody(snap.stats, snap.err)
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail with thread/FD/RSS charts.
//
// Takes width (int) and height (int) for the inner content.
//
// Returns string with the rendered body.
func (p *ProcessPanel) DetailView(width, height int) string {
	snap := p.snapshot()
	body := p.detailBody(snap.stats, snap.err)

	if len(snap.threads) < 2 && len(snap.fds) < 2 && len(snap.rss) < 2 {
		return RenderDetailBody(nil, body, width, height)
	}

	now := p.clock.Now()
	series := make([]ChartSeries, 0, processChartSeriesCap)
	if len(snap.rss) >= 2 {
		series = append(series, ChartSeries{Name: "RSS", Points: pointsFromHistory(snap.rss, now)})
	}
	if len(snap.fds) >= 2 {
		series = append(series, ChartSeries{Name: "FDs", Points: pointsFromHistory(snap.fds, now)})
	}
	if len(snap.threads) >= 2 {
		series = append(series, ChartSeries{Name: "Threads", Points: pointsFromHistory(snap.threads, now), Severity: SeverityWarning})
	}
	return RenderDetailBodyWithChart(nil, body, series, "Process history", width, height)
}

// processSnapshot bundles all values rendered together so they can be
// read under a single lock acquisition.
type processSnapshot struct {
	// stats is the most recent SystemStats payload, or nil before any
	// refresh has succeeded.
	stats *SystemStats

	// err is the most recent refresh error, or nil after success.
	err error

	// threads is a copy of the thread-count history ring values.
	threads []float64

	// fds is a copy of the file-descriptor count history ring values.
	fds []float64

	// rss is a copy of the resident-set-size history ring values.
	rss []float64
}

// snapshot reads stats, err, and all history rings under stateMutex
// so renders cannot race against concurrent handleStats writes.
//
// Returns processSnapshot containing a coherent view of the state.
func (p *ProcessPanel) snapshot() processSnapshot {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return processSnapshot{
		stats:   p.stats,
		err:     p.err,
		threads: p.threadHistory.Values(),
		fds:     p.fdHistory.Values(),
		rss:     p.rssHistory.Values(),
	}
}

// handleStats applies an incoming systemStatsMessage to the panel
// state, updating the rolling history rings and the last-error.
//
// Takes msg (systemStatsMessage) which carries the fresh stats and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProcessPanel) handleStats(msg systemStatsMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.err = msg.err
		return
	}
	p.stats = msg.stats
	p.err = nil
	if msg.stats != nil {
		p.threadHistory.Append(float64(msg.stats.Process.ThreadCount))
		p.fdHistory.Append(float64(msg.stats.Process.FDCount))
		p.rssHistory.Append(float64(msg.stats.Process.RSS))
	}
}

// renderBody renders the centre-pane tile body for the supplied stats
// and error.
//
// Takes stats (*SystemStats) which is the latest stats snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns string which is the rendered body.
func (p *ProcessPanel) renderBody(stats *SystemStats, err error) string {
	if err != nil {
		var b strings.Builder
		RenderErrorState(&b, err)
		return strings.TrimSuffix(b.String(), stringNewline)
	}
	if stats == nil {
		return RenderDimText("Fetching process info...")
	}
	body := p.detailBody(stats, nil)
	return RenderDetailBody(nil, body.WithoutHeader(), p.ContentWidth(), p.ContentHeight())
}

// detailBody composes the detail-pane inspector.DetailBody from stats and any
// refresh error.
//
// Takes stats (*SystemStats) which is the latest stats snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns inspector.DetailBody describing the panel state.
func (*ProcessPanel) detailBody(stats *SystemStats, err error) inspector.DetailBody {
	if err != nil {
		return inspector.DetailBody{
			Title:    titleProcess,
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: err.Error()}}}},
		}
	}
	if stats == nil {
		return inspector.DetailBody{Title: titleProcess, Subtitle: "no data yet"}
	}

	rows := []inspector.DetailRow{
		{Label: "PID", Value: formatInt(stats.Process.PID)},
		{Label: "Threads", Value: formatInt(stats.Process.ThreadCount)},
		{Label: "FDs", Value: formatInt(stats.Process.FDCount)},
		{Label: "RSS", Value: inspector.FormatBytes(stats.Process.RSS)},
		{Label: "Goroutines", Value: formatInt(stats.NumGoroutines)},
		{Label: "Uptime", Value: stats.Uptime.Truncate(time.Second).String()},
	}

	return inspector.DetailBody{
		Title:    titleProcess,
		Subtitle: "PID " + formatInt(stats.Process.PID),
		Sections: []inspector.DetailSection{{Heading: "Snapshot", Rows: rows}},
	}
}

// refresh returns a command that fetches fresh SystemStats from the
// configured provider.
//
// Returns tea.Cmd which delivers a systemStatsMessage.
func (p *ProcessPanel) refresh() tea.Cmd { return refreshSystemStatsCmd(p.provider) }
