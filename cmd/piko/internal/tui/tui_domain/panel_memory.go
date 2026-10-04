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

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// memoryHistorySize bounds the heap-alloc history kept for the detail chart. At a
	// typical 2s refresh rate this is ~30 minutes.
	memoryHistorySize = 900
)

// MemoryPanel surfaces heap and garbage-collection statistics.
//
// It is the TUI counterpart of `piko info memory` / `piko info gc`. Implements Panel.
type MemoryPanel struct {
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

	// gcHistory holds GC pause-duration samples (microseconds) for the detail-pane chart.
	gcHistory *HistoryRing

	BasePanel

	// stateMutex guards stats / err / history for safe concurrent reads.
	stateMutex sync.RWMutex
}

var (
	_ Panel = (*MemoryPanel)(nil)
)

// NewMemoryPanel constructs a MemoryPanel sharing the supplied SystemProvider.
//
// Takes provider (SystemProvider) which supplies system statistics.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *MemoryPanel ready to register with a group.
func NewMemoryPanel(provider SystemProvider, c clock.Clock) *MemoryPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &MemoryPanel{
		BasePanel:   NewBasePanel("memory", "Memory"),
		clock:       c,
		provider:    provider,
		heapHistory: NewHistoryRing(memoryHistorySize),
		gcHistory:   NewHistoryRing(memoryHistorySize),
		stateMutex:  sync.RWMutex{},
		err:         nil,
		stats:       nil,
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which schedules the first stats fetch.
func (p *MemoryPanel) Init() tea.Cmd { return p.refresh() }

// Update reacts to refreshes and the 'r' refresh key.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
func (p *MemoryPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
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
func (p *MemoryPanel) View(width, height int) string {
	p.SetSize(width, height)
	snap := p.snapshot()
	body := p.renderBody(snap.stats, snap.err)
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail with heap and GC charts.
//
// Takes width (int) which sets the available width in terminal cells.
// Takes height (int) which sets the available height in terminal rows.
//
// Returns string with the rendered body.
func (p *MemoryPanel) DetailView(width, height int) string {
	snap := p.snapshot()
	body := p.detailBody(snap.stats, snap.err)

	if len(snap.heap) < 2 && len(snap.gc) < 2 {
		return RenderDetailBody(nil, body, width, height)
	}

	now := p.clock.Now()
	series := make([]ChartSeries, 0, 2)
	if len(snap.heap) >= 2 {
		series = append(series, ChartSeries{Name: "Heap", Points: pointsFromHistory(snap.heap, now), Severity: 0})
	}
	if len(snap.gc) >= 2 {
		series = append(series, ChartSeries{Name: "GC pause us", Points: pointsFromHistory(snap.gc, now), Severity: SeverityWarning})
	}
	return RenderDetailBodyWithChart(nil, body, series, "Heap & GC", width, height)
}

// memorySnapshot bundles the values rendered together so they can all be read under one
// lock acquisition.
type memorySnapshot struct {
	// stats is the most recent SystemStats payload, or nil before any refresh has succeeded.
	stats *SystemStats

	// err is the most recent refresh error, or nil after success.
	err error

	// heap is a copy of the heap-alloc history ring values.
	heap []float64

	// gc is a copy of the GC-pause-duration history ring values (in microseconds).
	gc []float64
}

// snapshot returns the current stats, error, and copies of the heap and GC history rings,
// all read under stateMutex so they cannot race against concurrent handleStats writes.
//
// Returns memorySnapshot containing a coherent view of the state.
func (p *MemoryPanel) snapshot() memorySnapshot {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return memorySnapshot{
		stats: p.stats,
		err:   p.err,
		heap:  p.heapHistory.Values(),
		gc:    p.gcHistory.Values(),
	}
}

// handleStats applies an incoming systemStatsMessage to the panel state, updating the
// rolling history rings and the last-error.
//
// Takes msg (systemStatsMessage) which carries the fresh stats and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *MemoryPanel) handleStats(msg systemStatsMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.err = msg.err
		return
	}
	p.stats = msg.stats
	p.err = nil
	if msg.stats != nil {
		p.heapHistory.Append(float64(msg.stats.Memory.HeapAlloc))
		if msg.stats.GC.LastPauseNs > 0 {
			p.gcHistory.Append(float64(msg.stats.GC.LastPauseNs) / 1000.0)
		}
	}
}

// renderBody renders the centre-pane tile body for the supplied stats and error.
//
// Takes stats (*SystemStats) which is the latest stats snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns string which is the rendered body.
func (p *MemoryPanel) renderBody(stats *SystemStats, err error) string {
	if err != nil {
		var b strings.Builder
		RenderErrorState(&b, err)
		return strings.TrimSuffix(b.String(), stringNewline)
	}
	if stats == nil {
		return RenderDimText("Fetching memory stats...")
	}
	body := p.detailBody(stats, nil)
	return RenderDetailBody(nil, body.WithoutHeader(), p.ContentWidth(), p.ContentHeight())
}

// detailBody composes the detail-pane inspector.DetailBody from stats and any refresh
// error.
//
// Takes stats (*SystemStats) which is the latest stats snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns inspector.DetailBody describing the panel state.
func (*MemoryPanel) detailBody(stats *SystemStats, err error) inspector.DetailBody {
	if err != nil {
		return inspector.DetailBody{
			Title: "Memory",
			Sections: []inspector.DetailSection{inspector.NewDetailSection(
				"Error",
				[]inspector.DetailRow{inspector.NewDetailRow("Reason", err.Error())},
			)},
			Subtitle: "",
		}
	}
	if stats == nil {
		return inspector.DetailBody{Title: "Memory", Subtitle: "no data yet", Sections: nil}
	}

	heap := []inspector.DetailRow{
		inspector.NewDetailRow("Alloc", inspector.FormatBytes(stats.Memory.Alloc)),
		inspector.NewDetailRow("Heap alloc", inspector.FormatBytes(stats.Memory.HeapAlloc)),
		inspector.NewDetailRow("Heap inuse", inspector.FormatBytes(stats.Memory.HeapInuse)),
		inspector.NewDetailRow("Heap idle", inspector.FormatBytes(stats.Memory.HeapIdle)),
		inspector.NewDetailRow("Heap released", inspector.FormatBytes(stats.Memory.HeapReleased)),
		inspector.NewDetailRow("Heap objects", formatUint64(stats.Memory.HeapObjects)),
		inspector.NewDetailRow("Sys", inspector.FormatBytes(stats.Memory.Sys)),
		inspector.NewDetailRow("Total alloc", inspector.FormatBytes(stats.Memory.TotalAlloc)),
	}

	gc := []inspector.DetailRow{
		inspector.NewDetailRow("Cycles", formatUint64(uint64(stats.GC.NumGC))),
		inspector.NewDetailRow("Last pause", formatDurationNs(stats.GC.LastPauseNs)),
		inspector.NewDetailRow("Total pause", formatDurationNs(stats.GC.PauseTotalNs)),
		inspector.NewDetailRow("GC CPU", percentageString(stats.GC.GCCPUFraction)),
		inspector.NewDetailRow("Next GC", inspector.FormatBytes(stats.GC.NextGC)),
	}

	return inspector.DetailBody{
		Title:    "Memory & GC",
		Subtitle: inspector.FormatBytes(stats.Memory.HeapAlloc) + " heap · " + formatUint64(uint64(stats.GC.NumGC)) + " cycles",
		Sections: []inspector.DetailSection{
			inspector.NewDetailSection("Heap", heap),
			inspector.NewDetailSection("Garbage Collection", gc),
		},
	}
}

// refresh returns a command that fetches fresh SystemStats from the configured provider.
//
// Returns tea.Cmd which delivers a systemStatsMessage.
func (p *MemoryPanel) refresh() tea.Cmd { return refreshSystemStatsCmd(p.provider) }
