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

// runtimeOverviewHistory bounds the heap/goroutine history kept for
// the Runtime overview detail-pane chart.
const runtimeOverviewHistory = 600

// RuntimeOverviewPanel is the at-a-glance Runtime -> Overview panel:
// system snapshot tiles in the centre, ntcharts heap + goroutines in
// the detail. Implements Panel.
type RuntimeOverviewPanel struct {
	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the SystemStats fetched on each refresh.
	provider SystemProvider

	// err holds the last refresh error, or nil after success.
	err error

	// stats holds the most recent stats snapshot.
	stats *SystemStats

	// heapHistory holds heap-alloc samples for the detail-pane chart.
	heapHistory *HistoryRing

	// goroutineHistory holds goroutine-count samples for the detail
	// chart.
	goroutineHistory *HistoryRing

	BasePanel

	// stateMutex guards stats / err / history for safe concurrent
	// reads.
	stateMutex sync.RWMutex
}

var _ Panel = (*RuntimeOverviewPanel)(nil)

// NewRuntimeOverviewPanel constructs the Runtime overview panel.
//
// Takes provider (SystemProvider) which supplies system stats.
// Takes c (clock.Clock); nil falls back to the real clock.
//
// Returns *RuntimeOverviewPanel ready to register with the group.
func NewRuntimeOverviewPanel(provider SystemProvider, c clock.Clock) *RuntimeOverviewPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &RuntimeOverviewPanel{
		BasePanel:        NewBasePanel("runtime-overview", "Overview"),
		clock:            c,
		provider:         provider,
		heapHistory:      NewHistoryRing(runtimeOverviewHistory),
		goroutineHistory: NewHistoryRing(runtimeOverviewHistory),
		stateMutex:       sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which schedules the first stats fetch.
func (p *RuntimeOverviewPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *RuntimeOverviewPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "r" {
			cmd := p.refresh()
			return p, cmd
		}
	case systemStatsMessage:
		p.stateMutex.Lock()
		if msg.err != nil {
			p.err = msg.err
		} else {
			p.err = nil
			p.stats = msg.stats
			if msg.stats != nil {
				p.heapHistory.Append(float64(msg.stats.Memory.HeapAlloc))
				p.goroutineHistory.Append(float64(msg.stats.NumGoroutines))
			}
		}
		p.stateMutex.Unlock()
		return p, nil
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the centre as a compact tile view: uptime, CPU,
// memory, goroutines. The denser per-field table lives in the
// detail pane.
//
// Takes width (int) which is the column width for the tile.
// Takes height (int) which is the row height for the tile.
//
// Returns string with the rendered tile.
func (p *RuntimeOverviewPanel) View(width, height int) string {
	p.SetSize(width, height)
	snap := p.snapshot()
	if snap.err != nil {
		var b strings.Builder
		RenderErrorState(&b, snap.err)
		return p.RenderFrame(strings.TrimSuffix(b.String(), stringNewline))
	}
	if snap.stats == nil {
		return p.RenderFrame(RenderDimText("Fetching runtime overview..."))
	}
	body := RenderDetailBody(nil, p.tileBody(snap.stats).WithoutHeader(), p.ContentWidth(), p.ContentHeight())
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail with the full per-field
// table plus a heap + goroutines chart.
//
// Takes width (int) which is the column width for the detail body.
// Takes height (int) which is the row height for the detail body.
//
// Returns string with the rendered detail body.
func (p *RuntimeOverviewPanel) DetailView(width, height int) string {
	snap := p.snapshot()
	if snap.err != nil || snap.stats == nil {
		return RenderDetailBody(nil, p.detailBody(snap.stats), width, height)
	}
	if len(snap.heap) < 2 && len(snap.goroutines) < 2 {
		return RenderDetailBody(nil, p.detailBody(snap.stats), width, height)
	}
	now := p.clock.Now()
	series := []ChartSeries{}
	if len(snap.heap) >= 2 {
		series = append(series, ChartSeries{Name: "Heap", Points: pointsFromHistory(snap.heap, now)})
	}
	if len(snap.goroutines) >= 2 {
		series = append(series, ChartSeries{Name: "Goroutines", Points: pointsFromHistory(snap.goroutines, now), Severity: SeverityWarning})
	}
	return RenderDetailBodyWithChart(nil, p.detailBody(snap.stats), series, "Heap & goroutines", width, height)
}

// tileBody renders the centre-pane tile summary: a short list of
// "hero" stats with no fine-grained breakdown. The detail pane shows
// the full table and history chart.
//
// Takes stats (*SystemStats) which is the most recent snapshot.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
func (*RuntimeOverviewPanel) tileBody(stats *SystemStats) inspector.DetailBody {
	if stats == nil {
		return inspector.DetailBody{Title: "Runtime", Subtitle: "no data yet"}
	}
	rows := []inspector.DetailRow{
		{Label: "Uptime", Value: stats.Uptime.Truncate(time.Second).String()},
		{Label: "Heap alloc", Value: inspector.FormatBytes(stats.Memory.HeapAlloc)},
		{Label: "Goroutines", Value: formatInt(stats.NumGoroutines)},
		{Label: "RSS", Value: inspector.FormatBytes(stats.Process.RSS)},
	}
	return inspector.DetailBody{
		Title:    "Runtime",
		Subtitle: stats.Build.Version,
		Sections: []inspector.DetailSection{{Heading: "At a glance", Rows: rows}},
	}
}

// runtimeOverviewSnapshot bundles all values rendered together so they
// can be read under a single lock acquisition.
type runtimeOverviewSnapshot struct {
	// stats is the most recent SystemStats payload, or nil before any
	// refresh has succeeded.
	stats *SystemStats

	// err is the most recent refresh error, or nil after success.
	err error

	// heap is a copy of the heap-alloc history ring values.
	heap []float64

	// goroutines is a copy of the goroutine-count history ring values.
	goroutines []float64
}

// snapshot reads stats, err, and both history rings under stateMutex
// so renders cannot race against concurrent stats writes.
//
// Returns runtimeOverviewSnapshot containing a coherent view of the
// state.
func (p *RuntimeOverviewPanel) snapshot() runtimeOverviewSnapshot {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return runtimeOverviewSnapshot{
		stats:      p.stats,
		err:        p.err,
		heap:       p.heapHistory.Values(),
		goroutines: p.goroutineHistory.Values(),
	}
}

// detailBody renders the right-pane detail body with the full per-field
// runtime table.
//
// Takes stats (*SystemStats) which is the latest stats snapshot.
//
// Returns inspector.DetailBody describing the panel state.
func (*RuntimeOverviewPanel) detailBody(stats *SystemStats) inspector.DetailBody {
	if stats == nil {
		return inspector.DetailBody{Title: "Runtime", Subtitle: "no data yet"}
	}
	rows := []inspector.DetailRow{
		{Label: "Uptime", Value: stats.Uptime.Truncate(time.Second).String()},
		{Label: "CPU", Value: percentageString(stats.CPUMillicores / 1000.0)},
		{Label: "Goroutines", Value: formatInt(stats.NumGoroutines)},
		{Label: "Heap alloc", Value: inspector.FormatBytes(stats.Memory.HeapAlloc)},
		{Label: "RSS", Value: inspector.FormatBytes(stats.Process.RSS)},
		{Label: "GOMAXPROCS", Value: formatInt(stats.GOMAXPROCS)},
		{Label: "GC cycles", Value: formatUint64(uint64(stats.GC.NumGC))},
		{Label: "Last GC", Value: formatDurationNs(stats.GC.LastPauseNs)},
	}
	return inspector.DetailBody{
		Title:    "Runtime overview",
		Subtitle: stats.Build.Version,
		Sections: []inspector.DetailSection{{Heading: "Snapshot", Rows: rows}},
	}
}

// refresh returns a command that fetches fresh SystemStats from the
// configured provider.
//
// Returns tea.Cmd which delivers a systemStatsMessage.
func (p *RuntimeOverviewPanel) refresh() tea.Cmd { return refreshSystemStatsCmd(p.provider) }
