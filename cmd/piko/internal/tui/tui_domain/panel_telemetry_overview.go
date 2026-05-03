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
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// telemetryOverviewTimeout caps the combined health + traces fetch.
	telemetryOverviewTimeout = 5 * time.Second

	// telemetryOverviewSpanLimit is the maximum number of recent
	// spans pulled from the traces provider for the overview tile.
	telemetryOverviewSpanLimit = 200
)

// telemetryOverviewMessage carries combined health/error counts back
// to the overview panel.
type telemetryOverviewMessage struct {
	// livenessErr is the error returned by the liveness probe, or nil.
	livenessErr error

	// readinessErr is the error returned by the readiness probe, or nil.
	readinessErr error

	// err is the error returned by the traces probe, or nil.
	err error

	// liveness is the freshly fetched liveness status, or nil on error.
	liveness *HealthStatus

	// readiness is the freshly fetched readiness status, or nil on error.
	readiness *HealthStatus

	// totalSpans is the count of recent spans observed.
	totalSpans int

	// errorSpans is the count of recent spans flagged as errors.
	errorSpans int
}

// TelemetryOverviewPanel renders the at-a-glance Telemetry overview:
// liveness/readiness summary plus span counts. Implements Panel.
type TelemetryOverviewPanel struct {
	// lastRefresh records when the panel last received a payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// healthProvider supplies liveness / readiness; may be nil.
	healthProvider HealthProvider

	// tracesProvider supplies recent spans; may be nil.
	tracesProvider TracesProvider

	// last is the most recently received message.
	last telemetryOverviewMessage

	BasePanel

	// stateMutex guards last / lastRefresh / hasData for safe
	// concurrent reads.
	stateMutex sync.RWMutex

	// hasData reports whether at least one refresh has completed.
	hasData bool
}

var _ Panel = (*TelemetryOverviewPanel)(nil)

// NewTelemetryOverviewPanel constructs the panel.
//
// Takes health (HealthProvider), traces (TracesProvider) which may
// each be nil; missing providers fall back to placeholder rows.
// Takes c (clock.Clock); nil falls back to the real clock.
//
// Returns *TelemetryOverviewPanel ready to register with the group.
func NewTelemetryOverviewPanel(health HealthProvider, traces TracesProvider, c clock.Clock) *TelemetryOverviewPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &TelemetryOverviewPanel{
		BasePanel:      NewBasePanel("telemetry-overview", "Overview"),
		clock:          c,
		healthProvider: health,
		tracesProvider: traces,
		stateMutex:     sync.RWMutex{},
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which is the initial refresh command.
func (p *TelemetryOverviewPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel (always the receiver).
// Returns tea.Cmd which is the resulting command.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *TelemetryOverviewPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if cmd, handled := handleOverviewControlMessage(message, p.refresh); handled {
		return p, cmd
	}
	if msg, ok := message.(telemetryOverviewMessage); ok {
		p.stateMutex.Lock()
		p.last = msg
		p.lastRefresh = p.clock.Now()
		p.hasData = true
		p.stateMutex.Unlock()
	}
	return p, nil
}

// View renders the centre as a compact at-a-glance tile: liveness,
// readiness, span totals. The full breakdown lives in DetailView.
//
// Takes width (int) which is the allocated panel width.
// Takes height (int) which is the allocated panel height.
//
// Returns string which is the framed panel body.
func (p *TelemetryOverviewPanel) View(width, height int) string {
	p.SetSize(width, height)
	body := RenderDetailBody(nil, p.tileBody().WithoutHeader(), p.ContentWidth(), p.ContentHeight())
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail with the full per-probe
// breakdown.
//
// Takes width (int) which is the detail pane width.
// Takes height (int) which is the detail pane height.
//
// Returns string which is the rendered detail body.
func (p *TelemetryOverviewPanel) DetailView(width, height int) string {
	return RenderDetailBody(nil, p.detailBody(), width, height)
}

// tileBody renders the centre-pane tile summary. Skips the timestamp
// and per-probe error breakdown so the tile stays scannable; those
// details surface in the detail pane.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *TelemetryOverviewPanel) tileBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if !p.hasData {
		return inspector.DetailBody{Title: "Telemetry", Subtitle: "fetching..."}
	}

	livenessLabel := "-"
	switch {
	case p.last.liveness != nil:
		livenessLabel = p.last.liveness.State.String()
	case p.last.livenessErr != nil:
		livenessLabel = "error"
	}
	readinessLabel := "-"
	switch {
	case p.last.readiness != nil:
		readinessLabel = p.last.readiness.State.String()
	case p.last.readinessErr != nil:
		readinessLabel = "error"
	}

	rows := []inspector.DetailRow{
		{Label: "Liveness", Value: livenessLabel},
		{Label: "Readiness", Value: readinessLabel},
		{Label: "Spans (recent)", Value: formatInt(p.last.totalSpans)},
		{Label: "Errors", Value: formatInt(p.last.errorSpans)},
	}
	return inspector.DetailBody{
		Title:    "Telemetry",
		Sections: []inspector.DetailSection{{Heading: "At a glance", Rows: rows}},
	}
}

// detailBody builds the structured inspector.DetailBody for the detail pane.
//
// Returns inspector.DetailBody describing the panel state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *TelemetryOverviewPanel) detailBody() inspector.DetailBody {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()

	if !p.hasData {
		return inspector.DetailBody{Title: "Telemetry overview", Subtitle: "fetching..."}
	}

	rows := []inspector.DetailRow{}
	if p.last.liveness != nil {
		rows = append(rows, inspector.DetailRow{Label: "Liveness", Value: p.last.liveness.State.String()})
	} else if p.last.livenessErr != nil {
		rows = append(rows, inspector.DetailRow{Label: "Liveness", Value: "error: " + p.last.livenessErr.Error()})
	}
	if p.last.readiness != nil {
		rows = append(rows, inspector.DetailRow{Label: "Readiness", Value: p.last.readiness.State.String()})
	} else if p.last.readinessErr != nil {
		rows = append(rows, inspector.DetailRow{Label: "Readiness", Value: "error: " + p.last.readinessErr.Error()})
	}
	rows = append(rows,
		inspector.DetailRow{Label: "Recent spans", Value: formatInt(p.last.totalSpans)},
		inspector.DetailRow{Label: "Error spans", Value: formatInt(p.last.errorSpans)},
	)
	if !p.lastRefresh.IsZero() {
		rows = append(rows, inspector.DetailRow{Label: "Last refresh", Value: p.lastRefresh.Format(time.RFC3339)})
	}
	if p.last.err != nil {
		rows = append(rows, inspector.DetailRow{Label: "Error", Value: p.last.err.Error()})
	}

	return inspector.DetailBody{
		Title:    "Telemetry overview",
		Sections: []inspector.DetailSection{{Heading: "Status", Rows: rows}},
	}
}

// refresh returns a Cmd that probes liveness, readiness, and traces
// concurrently and posts a telemetryOverviewMessage.
//
// Returns tea.Cmd which delivers the combined telemetry overview message.
func (p *TelemetryOverviewPanel) refresh() tea.Cmd {
	health := p.healthProvider
	traces := p.tracesProvider
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), telemetryOverviewTimeout,
			errors.New("telemetry overview exceeded timeout"))
		defer cancel()
		msg := telemetryOverviewMessage{}

		var wg sync.WaitGroup
		if health != nil {
			wg.Go(func() { telemetryProbeLiveness(ctx, health, &msg) })
			wg.Go(func() { telemetryProbeReadiness(ctx, health, &msg) })
		}
		if traces != nil {
			wg.Go(func() { telemetryProbeTraces(ctx, traces, &msg) })
		}
		wg.Wait()
		return msg
	}
}

// telemetryProbeLiveness fetches the liveness probe and stores the
// result into the per-field slot on msg.
//
// Takes ctx (context.Context) for cancellation.
// Takes health (HealthProvider) which supplies the liveness call.
// Takes msg (*telemetryOverviewMessage) which receives the result.
func telemetryProbeLiveness(ctx context.Context, health HealthProvider, msg *telemetryOverviewMessage) {
	liveness, err := health.Liveness(ctx)
	if err != nil {
		msg.livenessErr = err
		return
	}
	msg.liveness = liveness
}

// telemetryProbeReadiness fetches the readiness probe and stores the
// result into the per-field slot on msg.
//
// Takes ctx (context.Context) for cancellation.
// Takes health (HealthProvider) which supplies the readiness call.
// Takes msg (*telemetryOverviewMessage) which receives the result.
func telemetryProbeReadiness(ctx context.Context, health HealthProvider, msg *telemetryOverviewMessage) {
	readiness, err := health.Readiness(ctx)
	if err != nil {
		msg.readinessErr = err
		return
	}
	msg.readiness = readiness
}

// telemetryProbeTraces counts recent and error spans, writing the
// totals to msg.
//
// Takes ctx (context.Context) for cancellation.
// Takes traces (TracesProvider) which supplies the recent-spans call.
// Takes msg (*telemetryOverviewMessage) which receives the counts.
func telemetryProbeTraces(ctx context.Context, traces TracesProvider, msg *telemetryOverviewMessage) {
	recent, err := traces.Recent(ctx, telemetryOverviewSpanLimit)
	if err != nil {
		msg.err = err
		return
	}
	msg.totalSpans = len(recent)
	count := 0
	for i := range recent {
		if recent[i].IsError() {
			count++
		}
	}
	msg.errorSpans = count
}
