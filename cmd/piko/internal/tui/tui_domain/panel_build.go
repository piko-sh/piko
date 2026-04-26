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

// BuildPanel surfaces the server's build metadata and Go runtime config.
//
// It is the TUI counterpart of `piko info build` and `piko info runtime`,
// rendered as a read-only key/value list. Implements Panel.
type BuildPanel struct {
	// lastRefresh records when the panel last received a stats payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the SystemStats fetched on each refresh.
	provider SystemProvider

	// err holds the last refresh error, or nil after a successful one.
	err error

	// stats holds the most recent stats snapshot.
	stats *SystemStats

	BasePanel

	// stateMutex guards stats / err for safe concurrent reads.
	stateMutex sync.RWMutex
}

var _ Panel = (*BuildPanel)(nil)

// NewBuildPanel constructs a BuildPanel sharing the supplied SystemProvider.
//
// Takes provider (SystemProvider) which supplies system statistics.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *BuildPanel ready to register with a group.
func NewBuildPanel(provider SystemProvider, c clock.Clock) *BuildPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &BuildPanel{
		BasePanel:  NewBasePanel("build", titleBuild),
		clock:      c,
		provider:   provider,
		stateMutex: sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{
		{Key: "r", Description: "Refresh"},
	})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd that fetches the current system stats.
func (p *BuildPanel) Init() tea.Cmd { return p.refresh() }

// Update reacts to data refreshes and the 'r' refresh key.
//
// Takes message (tea.Msg) which is the message to handle.
//
// Returns Panel which is the receiver after the update.
// Returns tea.Cmd which is the resulting command.
func (p *BuildPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
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
// Takes width (int) which is the available render width.
// Takes height (int) which is the available render height.
//
// Returns string which is the rendered panel body.
func (p *BuildPanel) View(width, height int) string {
	p.SetSize(width, height)
	stats, err := p.snapshot()
	body := p.renderBody(stats, err)
	return p.RenderFrame(body)
}

// DetailView renders the right-pane body, a denser key/value table than
// the centre summary so users can copy values out of the detail.
//
// Takes width (int) and height (int) which are the inner detail-pane
// dimensions.
//
// Returns string with the rendered body sized to width x height.
func (p *BuildPanel) DetailView(width, height int) string {
	stats, err := p.snapshot()
	body := p.buildDetailBody(stats, err)
	return RenderDetailBody(nil, body, width, height)
}

// snapshot returns the latest stats / error pair under a read lock.
//
// Returns *SystemStats which is the most recent stats snapshot or nil.
// Returns error which is the last refresh error or nil.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *BuildPanel) snapshot() (*SystemStats, error) {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return p.stats, p.err
}

// handleStats applies a refresh message to the panel state.
//
// Takes msg (systemStatsMessage) which carries the new stats or error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *BuildPanel) handleStats(msg systemStatsMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.err = msg.err
	} else {
		p.stats = msg.stats
		p.err = nil
	}
	p.lastRefresh = p.clock.Now()
}

// renderBody renders the centre-summary body for the panel.
//
// Takes stats (*SystemStats) which is the snapshot, possibly nil.
// Takes err (error) which is the last refresh error, possibly nil.
//
// Returns string which is the rendered body.
func (p *BuildPanel) renderBody(stats *SystemStats, err error) string {
	if err != nil {
		var b strings.Builder
		RenderErrorState(&b, err)
		return strings.TrimSuffix(b.String(), stringNewline)
	}
	if stats == nil {
		return RenderDimText("Fetching build info...")
	}
	body := p.buildDetailBody(stats, nil)
	return RenderDetailBody(nil, body.WithoutHeader(), p.ContentWidth(), p.ContentHeight())
}

// buildDetailBody assembles the detail-pane inspector.DetailBody for the panel.
//
// Takes stats (*SystemStats) which is the snapshot, possibly nil.
// Takes err (error) which is the last refresh error, possibly nil.
//
// Returns inspector.DetailBody describing the build and runtime sections.
func (*BuildPanel) buildDetailBody(stats *SystemStats, err error) inspector.DetailBody {
	if err != nil {
		return inspector.DetailBody{
			Title:    titleBuild,
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: err.Error()}}}},
		}
	}
	if stats == nil {
		return inspector.DetailBody{
			Title:    titleBuild,
			Subtitle: "no data yet",
		}
	}
	return inspector.DetailBody{
		Title:    "Build & Runtime",
		Subtitle: stats.Build.Version,
		Sections: []inspector.DetailSection{
			{Heading: titleBuild, Rows: []inspector.DetailRow{
				{Label: "Version", Value: stats.Build.Version},
				{Label: "Commit", Value: stats.Build.Commit},
				{Label: "Built", Value: stats.Build.BuildTime},
				{Label: "Go", Value: stats.Build.GoVersion},
				{Label: "OS", Value: stats.Build.OS},
				{Label: "Arch", Value: stats.Build.Arch},
			}},
			{Heading: "Runtime", Rows: []inspector.DetailRow{
				{Label: "GOGC", Value: stats.Runtime.GOGC},
				{Label: "GOMEMLIMIT", Value: stats.Runtime.GOMEMLIMIT},
				{Label: "GOMAXPROCS", Value: formatInt(stats.GOMAXPROCS)},
				{Label: "NumCPU", Value: formatInt(stats.NumCPU)},
			}},
		},
	}
}

// refresh returns a tea.Cmd that fetches a fresh stats snapshot.
//
// Returns tea.Cmd which performs the refresh.
func (p *BuildPanel) refresh() tea.Cmd {
	return refreshSystemStatsCmd(p.provider)
}
