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
	"slices"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

// contentOverviewTimeout is the maximum time the refresh command will
// wait for the resource provider to reply before cancelling the call.
const contentOverviewTimeout = 5 * time.Second

// contentOverviewMessage carries combined kind summaries.
type contentOverviewMessage struct {
	// err carries any error encountered while gathering the summary.
	err error

	// summary maps each kind to its per-status counts.
	summary map[string]map[ResourceStatus]int

	// totalsByKind maps each kind to the sum across all statuses.
	totalsByKind map[string]int

	// kinds is the sorted list of kinds present in the summary.
	kinds []string
}

// ContentOverviewPanel is the at-a-glance Content -> Overview panel.
type ContentOverviewPanel struct {
	// last is the most recently received summary payload.
	last contentOverviewMessage

	// lastRefresh records when the panel last received a payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies resource counts for the overview tile.
	provider ResourceProvider

	BasePanel

	// stateMutex guards last / lastRefresh / hasData for safe
	// concurrent reads.
	stateMutex sync.RWMutex

	// hasData reports whether at least one refresh has completed.
	hasData bool
}

var _ Panel = (*ContentOverviewPanel)(nil)

// NewContentOverviewPanel constructs the panel.
//
// Takes provider (ResourceProvider) which supplies per-kind resource counts.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *ContentOverviewPanel ready to register with the group.
func NewContentOverviewPanel(provider ResourceProvider, c clock.Clock) *ContentOverviewPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &ContentOverviewPanel{
		BasePanel:  NewBasePanel("content-overview", "Overview"),
		clock:      c,
		provider:   provider,
		stateMutex: sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which schedules the first summary fetch.
func (p *ContentOverviewPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ContentOverviewPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if cmd, handled := handleOverviewControlMessage(message, p.refresh); handled {
		return p, cmd
	}
	if msg, ok := message.(contentOverviewMessage); ok {
		p.stateMutex.Lock()
		p.last = msg
		p.lastRefresh = p.clock.Now()
		p.hasData = true
		p.stateMutex.Unlock()
	}
	return p, nil
}

// View renders the centre as a compact tile.
//
// Shows the total artefact count plus the largest-by-count kind. The
// per-status breakdown lives in DetailView.
//
// Takes width (int) which is the column width allocated by the layout.
// Takes height (int) which is the row height allocated by the layout.
//
// Returns string with the rendered tile.
func (p *ContentOverviewPanel) View(width, height int) string {
	p.SetSize(width, height)
	return p.RenderFrame(RenderDetailBody(nil, p.tileBody().WithoutHeader(), p.ContentWidth(), p.ContentHeight()))
}

// DetailView renders the right-pane detail with per-kind and per-status counts.
//
// Takes width (int) which is the column width for the detail body.
// Takes height (int) which is the row height for the detail body.
//
// Returns string with the rendered detail body.
func (p *ContentOverviewPanel) DetailView(width, height int) string {
	return RenderDetailBody(nil, p.detailBody(), width, height)
}

// tileBody renders the centre-pane tile summary: just the per-kind
// total counts. The detail pane breaks each kind down by status.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ContentOverviewPanel) tileBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if !p.hasData {
		return inspector.DetailBody{Title: "Content", Subtitle: "fetching..."}
	}
	if p.last.err != nil {
		return inspector.DetailBody{
			Title:    "Content",
			Subtitle: "refresh failed",
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: p.last.err.Error()}}}},
		}
	}

	total := 0
	for _, kind := range p.last.kinds {
		total += p.last.totalsByKind[kind]
	}
	rows := []inspector.DetailRow{
		{Label: "Total artefacts", Value: formatInt(total)},
		{Label: "Kinds", Value: formatInt(len(p.last.kinds))},
	}
	for _, kind := range p.last.kinds {
		rows = append(rows, inspector.DetailRow{
			Label: kind,
			Value: formatInt(p.last.totalsByKind[kind]),
		})
	}
	return inspector.DetailBody{
		Title:    "Content",
		Sections: []inspector.DetailSection{{Heading: "At a glance", Rows: rows}},
	}
}

// detailBody renders the right-pane detail body with per-kind and
// per-status counts. Each kind expands into one row per status with
// its count, surfacing degraded / unhealthy items at a glance.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ContentOverviewPanel) detailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if !p.hasData {
		return inspector.DetailBody{Title: "Content overview", Subtitle: "fetching..."}
	}
	if p.last.err != nil {
		return inspector.DetailBody{
			Title:    "Content overview",
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: p.last.err.Error()}}}},
		}
	}

	sections := make([]inspector.DetailSection, 0, len(p.last.kinds)+1)
	for _, kind := range p.last.kinds {
		statuses := p.last.summary[kind]
		rows := make([]inspector.DetailRow, 0, len(statuses)+1)
		rows = append(rows, inspector.DetailRow{Label: "Total", Value: formatInt(p.last.totalsByKind[kind])})
		for _, status := range sortedResourceStatuses(statuses) {
			rows = append(rows, inspector.DetailRow{Label: status.String(), Value: formatInt(statuses[status])})
		}
		sections = append(sections, inspector.DetailSection{Heading: kind, Rows: rows})
	}
	if !p.lastRefresh.IsZero() {
		sections = append(sections, inspector.DetailSection{
			Heading: "Refresh",
			Rows:    []inspector.DetailRow{{Label: "Last", Value: p.lastRefresh.Format(time.RFC3339)}},
		})
	}
	return inspector.DetailBody{
		Title:    "Content overview",
		Sections: sections,
	}
}

// sortedResourceStatuses returns the statuses in a stable
// human-readable order. Used by the detail pane so per-kind
// breakdowns line up across renders.
//
// Takes statuses (map[ResourceStatus]int) which is the per-status
// count map.
//
// Returns []ResourceStatus in declared-order: Healthy, Degraded,
// Unhealthy, Pending, Unknown, but only those present in the map.
func sortedResourceStatuses(statuses map[ResourceStatus]int) []ResourceStatus {
	order := []ResourceStatus{
		ResourceStatusHealthy,
		ResourceStatusDegraded,
		ResourceStatusUnhealthy,
		ResourceStatusPending,
		ResourceStatusUnknown,
	}
	out := make([]ResourceStatus, 0, len(statuses))
	for _, s := range order {
		if _, ok := statuses[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

// refresh returns a command that fetches the current resource summary
// from the configured provider and emits it as a contentOverviewMessage.
//
// Returns tea.Cmd which performs the fetch when executed.
func (p *ContentOverviewPanel) refresh() tea.Cmd {
	provider := p.provider
	return func() tea.Msg {
		if provider == nil {
			return contentOverviewMessage{err: errNoResourceProvider}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), contentOverviewTimeout,
			errors.New("content overview exceeded timeout"))
		defer cancel()
		summary, err := provider.Summary(ctx)
		if err != nil {
			return contentOverviewMessage{err: err}
		}
		kinds := make([]string, 0, len(summary))
		totals := make(map[string]int, len(summary))
		for kind, statuses := range summary {
			total := 0
			for _, n := range statuses {
				total += n
			}
			totals[kind] = total
			kinds = append(kinds, kind)
		}
		slices.Sort(kinds)
		return contentOverviewMessage{summary: summary, kinds: kinds, totalsByKind: totals}
	}
}
