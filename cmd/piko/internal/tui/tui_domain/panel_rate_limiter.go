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
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// rateLimiterRefreshTimeout caps the rate-limiter status RPC fetch.
	rateLimiterRefreshTimeout = 5 * time.Second

	// rateLimiterHistorySize bounds the allowed/denied history kept
	// for the detail-pane chart.
	rateLimiterHistorySize = 600

	// rateLimiterTitle is the panel title shown in the centre and
	// detail headings.
	rateLimiterTitle = "Rate limiter"
)

// rateLimiterRefreshMessage carries the result of a status fetch.
type rateLimiterRefreshMessage struct {
	// status is the freshly fetched rate-limiter status, or nil on error.
	status *RateLimiterStatus

	// err is the fetch error, or nil on success.
	err error
}

// RateLimiterPanel surfaces the rate-limiter counters.
//
// It is the TUI counterpart of `piko get ratelimiter`. Implements Panel.
type RateLimiterPanel struct {
	// lastRefresh records when the panel last received a status payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the rate-limiter status fetched on each
	// refresh.
	provider RateLimiterInspector

	// err holds the last refresh error, or nil after success.
	err error

	// status holds the most recent status snapshot.
	status *RateLimiterStatus

	// allowedHistory holds per-refresh deltas of the allowed counter
	// for the detail-pane chart.
	allowedHistory *HistoryRing

	// deniedHistory holds per-refresh deltas of the denied counter
	// for the detail-pane chart.
	deniedHistory *HistoryRing

	BasePanel

	// prevAllowed is the allowed counter at the previous refresh,
	// used to compute the per-refresh delta pushed into allowedHistory.
	prevAllowed int64

	// prevDenied is the denied counter at the previous refresh,
	// used to compute the per-refresh delta pushed into deniedHistory.
	prevDenied int64

	// stateMutex guards status / err / history for safe concurrent
	// reads.
	stateMutex sync.RWMutex
}

var _ Panel = (*RateLimiterPanel)(nil)

// NewRateLimiterPanel constructs a RateLimiterPanel.
//
// Takes provider (RateLimiterInspector) which supplies the status RPC port.
// Takes c (clock.Clock) which yields the current time; nil falls back
// to the real system clock.
//
// Returns *RateLimiterPanel ready to register with the model.
func NewRateLimiterPanel(provider RateLimiterInspector, c clock.Clock) *RateLimiterPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &RateLimiterPanel{
		BasePanel:      NewBasePanel("rate-limiter", "Rate Limiter"),
		clock:          c,
		provider:       provider,
		stateMutex:     sync.RWMutex{},
		allowedHistory: NewHistoryRing(rateLimiterHistorySize),
		deniedHistory:  NewHistoryRing(rateLimiterHistorySize),
	}
	p.SetKeyMap([]KeyBinding{{Key: "r", Description: "Refresh"}})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which is the initial refresh command.
func (p *RateLimiterPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel (always the receiver).
// Returns tea.Cmd which is the resulting command.
func (p *RateLimiterPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "r" {
			cmd := p.refresh()
			return p, cmd
		}
	case rateLimiterRefreshMessage:
		p.handleStatus(msg)
		return p, nil
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the panel. Falls back to a "feature disabled" hint
// when the server does not expose the rate-limiter inspector service.
//
// Takes width (int) which is the allocated panel width.
// Takes height (int) which is the allocated panel height.
//
// Returns string which is the framed panel body.
func (p *RateLimiterPanel) View(width, height int) string {
	p.SetSize(width, height)
	snap := p.snapshot()
	if IsServiceUnavailable(snap.err) {
		hint := ServiceUnavailableHint(rateLimiterTitle,
			"Rate-limit metrics are unavailable. The server build omits the rate limiter, or no policy is configured.")
		return p.RenderFrame(RenderDimText(hint))
	}
	body := p.renderBody(snap.status, snap.err)
	return p.RenderFrame(body)
}

// DetailView renders the right-pane detail with allow/deny chart.
//
// Takes width (int) which is the detail pane width.
// Takes height (int) which is the detail pane height.
//
// Returns string which is the rendered detail body with chart.
func (p *RateLimiterPanel) DetailView(width, height int) string {
	snap := p.snapshot()
	body := p.detailBody(snap.status, snap.err)

	if len(snap.allowed) < 2 && len(snap.denied) < 2 {
		return RenderDetailBody(nil, body, width, height)
	}
	now := p.clock.Now()
	series := []ChartSeries{}
	if len(snap.allowed) >= 2 {
		series = append(series, ChartSeries{Name: "allow/min", Points: pointsFromHistory(snap.allowed, now), Severity: SeverityHealthy})
	}
	if len(snap.denied) >= 2 {
		series = append(series, ChartSeries{Name: "deny/min", Points: pointsFromHistory(snap.denied, now), Severity: SeverityWarning})
	}
	return RenderDetailBodyWithChart(nil, body, series, "Throughput", width, height)
}

// rateLimiterSnapshot bundles all values rendered together so they can
// be read under a single lock acquisition.
type rateLimiterSnapshot struct {
	// status is the cached status snapshot.
	status *RateLimiterStatus

	// err is the cached refresh error.
	err error

	// allowed is the per-refresh delta history of allowed checks.
	allowed []float64

	// denied is the per-refresh delta history of denied checks.
	denied []float64
}

// snapshot reads status, err, and both history rings under stateMutex
// so renders cannot race against concurrent handleStatus writes.
//
// Returns rateLimiterSnapshot which bundles the cached values.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *RateLimiterPanel) snapshot() rateLimiterSnapshot {
	p.stateMutex.RLock()
	defer p.stateMutex.RUnlock()
	return rateLimiterSnapshot{
		status:  p.status,
		err:     p.err,
		allowed: p.allowedHistory.Values(),
		denied:  p.deniedHistory.Values(),
	}
}

// handleStatus stores a fresh status payload, updating delta history.
//
// Takes msg (rateLimiterRefreshMessage) which carries the status and error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *RateLimiterPanel) handleStatus(msg rateLimiterRefreshMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.err = msg.err
		return
	}
	p.err = nil
	if msg.status != nil {
		if p.status != nil {
			deltaAllowed := msg.status.TotalAllowed - p.prevAllowed
			deltaDenied := msg.status.TotalDenied - p.prevDenied
			if deltaAllowed >= 0 {
				p.allowedHistory.Append(float64(deltaAllowed))
			}
			if deltaDenied >= 0 {
				p.deniedHistory.Append(float64(deltaDenied))
			}
		}
		p.prevAllowed = msg.status.TotalAllowed
		p.prevDenied = msg.status.TotalDenied
		p.status = msg.status
	}
	p.lastRefresh = p.clock.Now()
}

// renderBody renders the centre-pane body for the rate-limiter panel.
//
// Takes status (*RateLimiterStatus) which is the latest status snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns string which is the rendered body.
func (p *RateLimiterPanel) renderBody(status *RateLimiterStatus, err error) string {
	if err != nil {
		var b strings.Builder
		RenderErrorState(&b, err)
		return strings.TrimSuffix(b.String(), stringNewline)
	}
	if status == nil {
		return RenderDimText("Fetching rate-limiter status...")
	}
	body := p.detailBody(status, nil)
	return RenderDetailBody(nil, body.WithoutHeader(), p.ContentWidth(), p.ContentHeight())
}

// detailBody builds the structured inspector.DetailBody for the rate-limiter panel.
//
// Takes status (*RateLimiterStatus) which is the latest status snapshot.
// Takes err (error) which is the latest refresh error.
//
// Returns inspector.DetailBody describing the panel state.
func (*RateLimiterPanel) detailBody(status *RateLimiterStatus, err error) inspector.DetailBody {
	if err != nil {
		return inspector.DetailBody{
			Title:    rateLimiterTitle,
			Sections: []inspector.DetailSection{{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: err.Error()}}}},
		}
	}
	if status == nil {
		return inspector.DetailBody{Title: rateLimiterTitle, Subtitle: "no data yet"}
	}
	denyRatio := 0.0
	if status.TotalChecks > 0 {
		denyRatio = float64(status.TotalDenied) / float64(status.TotalChecks)
	}
	return inspector.DetailBody{
		Title:    rateLimiterTitle,
		Subtitle: status.FailPolicy,
		Sections: []inspector.DetailSection{
			{Heading: "Configuration", Rows: []inspector.DetailRow{
				{Label: "Token bucket", Value: status.TokenBucketStore},
				{Label: "Counter store", Value: status.CounterStore},
				{Label: "Fail policy", Value: status.FailPolicy},
				{Label: "Key prefix", Value: status.KeyPrefix},
			}},
			{Heading: "Counters", Rows: []inspector.DetailRow{
				{Label: "Checks", Value: fmt.Sprintf(fmtDecimal, status.TotalChecks)},
				{Label: "Allowed", Value: fmt.Sprintf(fmtDecimal, status.TotalAllowed)},
				{Label: "Denied", Value: fmt.Sprintf(fmtDecimal, status.TotalDenied)},
				{Label: "Errors", Value: fmt.Sprintf(fmtDecimal, status.TotalErrors)},
				{Label: "Deny ratio", Value: percentageString(denyRatio)},
			}},
		},
	}
}

// refresh returns a Cmd that fetches the latest rate-limiter status.
//
// Returns tea.Cmd which delivers a rateLimiterRefreshMessage.
func (p *RateLimiterPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return rateLimiterRefreshMessage{err: errNoRateLimiterInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), rateLimiterRefreshTimeout,
			errors.New("rate-limiter status exceeded timeout"))
		defer cancel()
		status, err := p.provider.GetStatus(ctx)
		return rateLimiterRefreshMessage{status: status, err: err}
	}
}
