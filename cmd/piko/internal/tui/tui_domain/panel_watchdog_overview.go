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
	"charm.land/lipgloss/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogOverviewPanelID is the identifier used by the model to
	// look up the watchdog Overview panel.
	WatchdogOverviewPanelID = "watchdog-overview"

	// WatchdogOverviewPanelTitle is the display title rendered in the
	// tab bar and the panel frame.
	WatchdogOverviewPanelTitle = "Watchdog"

	// overviewMinPaneWidthForSections is the minimum width before the
	// section nav is rendered alongside the dashboard.
	overviewMinPaneWidthForSections = 90

	// overviewAlertTapeMaxRows caps the alert-tape body even when there
	// is plenty of room.
	overviewAlertTapeMaxRows = 5

	// overviewLocalEventCap is the size of the panel-local event ring
	// retained for the alert tape.
	overviewLocalEventCap = 32
)

// overviewSections lists the categories shown in the panel's left-hand
// section navigation. Selecting one filters which gauges are emphasised
// and which events appear in the drill-down area.
var overviewSections = []overviewSection{
	{ID: "all", Label: "All"},
	{ID: "lifecycle", Label: "Lifecycle"},
	{ID: "capture", Label: "Capture"},
	{ID: "heap", Label: "Heap"},
	{ID: "goroutines", Label: "Goroutines"},
	{ID: "gc", Label: "GC"},
	{ID: "fd", Label: "FD"},
	{ID: "scheduler", Label: "Scheduler"},
	{ID: "continuous", Label: "Continuous"},
}

// overviewSection describes a single entry in the section nav.
type overviewSection struct {
	// ID identifies the section programmatically.
	ID string

	// Label is the rendered display label.
	Label string
}

// overviewSnapshotMsg carries a refreshed status snapshot from the
// background fetcher into the panel. Err is non-nil when the fetch
// failed; the panel exposes it to the user as a banner instead of
// rendering stale data without explanation.
type overviewSnapshotMsg struct {
	// Status is the freshly fetched watchdog status, or nil on error.
	Status *WatchdogStatus

	// Err is the fetch error, or nil on success.
	Err error
}

// overviewEventMsg carries a single live event delivered through the
// dispatcher subscription into the panel.
type overviewEventMsg struct {
	// Event is the watchdog event delivered by the subscription.
	Event WatchdogEvent

	// Done is true when the subscription channel closed.
	Done bool
}

// WatchdogOverviewPanel renders the at-a-glance watchdog dashboard:
// status header, budget gauges, recent-event tape, and a section nav
// for drilling into a specific category.
type WatchdogOverviewPanel struct {
	// statusFetched is when the most recent snapshot landed.
	statusFetched time.Time

	// provider supplies snapshot data via the gRPC inspector.
	provider WatchdogProvider

	// clock yields the current time; tests can substitute a fake.
	clock clock.Clock

	// lastFetchErr is the most recent fetch error, or nil on success.
	lastFetchErr error

	// dispatcher streams live events into the panel; may be nil.
	dispatcher *EventDispatcher

	// theme drives styling for status colours and headings.
	theme *Theme

	// status is the most recent watchdog status snapshot.
	status *WatchdogStatus

	// subscription is the active dispatcher subscription, when any.
	subscription *WatchdogSubscription

	// events is the local ring of recently observed events.
	events []WatchdogEvent

	BasePanel

	// mu guards status / events / lastFetchErr / statusFetched.
	mu sync.RWMutex

	// subscriptionMu guards subscription pointer mutations.
	subscriptionMu sync.Mutex
}

// Compile-time interface assertions so changes to the Panel interface
// surface here rather than at every call site.
var (
	_ Panel = (*WatchdogOverviewPanel)(nil)

	_ ThemeAware = (*WatchdogOverviewPanel)(nil)
)

// NewWatchdogOverviewPanel constructs the panel. The dispatcher may be
// nil; events will then be polled from provider.ListEvents on each
// snapshot tick.
//
// Takes provider (WatchdogProvider) which supplies snapshot data.
// Takes dispatcher (*EventDispatcher) which streams live events.
// Takes clk (clock.Clock) which yields the current time. Pass nil to
// use the real system clock.
//
// Returns *WatchdogOverviewPanel ready for AddPanel.
func NewWatchdogOverviewPanel(provider WatchdogProvider, dispatcher *EventDispatcher, clk clock.Clock) *WatchdogOverviewPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogOverviewPanel{
		BasePanel:  NewBasePanel(WatchdogOverviewPanelID, WatchdogOverviewPanelTitle),
		provider:   provider,
		dispatcher: dispatcher,
		clock:      clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "j / Down", Description: "Next section"},
		{Key: "k / Up", Description: "Previous section"},
		{Key: "g", Description: "Top section"},
		{Key: "G", Description: "Bottom section"},
		{Key: "R", Description: "Refresh now"},
	})
	return panel
}

// SetTheme implements ThemeAware so the model propagates theme changes
// to the panel.
//
// Takes theme (*Theme) which becomes the new theme.
func (p *WatchdogOverviewPanel) SetTheme(theme *Theme) {
	p.theme = theme
}

// Init kicks off the first snapshot fetch and subscribes to the
// dispatcher when one is configured.
//
// Returns tea.Cmd which delivers the first overviewSnapshotMsg and, when
// applicable, the first overviewEventMsg from the subscription.
func (p *WatchdogOverviewPanel) Init() tea.Cmd {
	cmds := []tea.Cmd{p.fetchSnapshotCmd()}
	if cmd := p.subscribeCmd(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// Update handles tick, snapshot, event, and key messages.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel (always the receiver).
// Returns tea.Cmd which is the resulting command.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case overviewSnapshotMsg:
		p.mu.Lock()
		if msg.Err == nil {
			p.status = msg.Status
		}
		p.lastFetchErr = msg.Err
		p.statusFetched = p.clock.Now()
		p.mu.Unlock()
	case overviewEventMsg:
		if msg.Done {
			cmd := p.subscribeCmd()
			return p, cmd
		}
		p.recordEvent(msg.Event)
		cmd := p.waitForNextEventCmd()
		return p, cmd
	case TickMessage:
		cmd := p.fetchSnapshotCmd()
		return p, cmd
	case tea.KeyPressMsg:
		if cmd := p.handleKey(msg); cmd != nil {
			return p, cmd
		}
	}
	return p, nil
}

// View renders the panel sized to the supplied dimensions.
//
// Takes width (int) and height (int) which are the allocated panel
// dimensions including frame and padding.
//
// Returns string which is the framed body.
func (p *WatchdogOverviewPanel) View(width, height int) string {
	p.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()
	if contentWidth <= 0 || contentHeight <= 0 {
		return p.RenderFrame("")
	}

	body := p.composeBody(contentWidth, contentHeight)
	return p.RenderFrame(body)
}

// composeBody arranges the section nav, dashboard, and alert tape
// according to the available width.
//
// Takes width (int) and height (int) which are the inner content
// dimensions.
//
// Returns string which is the composed body.
func (p *WatchdogOverviewPanel) composeBody(width, height int) string {
	dashboard := p.renderDashboard(width, height)
	if width < overviewMinPaneWidthForSections {
		return dashboard
	}

	sectionWidth := overviewSectionsWidth(width)
	dashboardWidth := width - sectionWidth - 1

	sections := p.renderSectionNav(sectionWidth, height)
	dashboard = p.renderDashboard(dashboardWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, sections, " ", dashboard)
}

// overviewSectionsWidth chooses an appropriate width for the section
// nav given the panel's available width.
//
// Takes width (int) which is the inner panel width.
//
// Returns int which is the section nav width in cells.
func overviewSectionsWidth(width int) int {
	target := min(max(width/overviewSectionsWidthDivisor, SectionNavMinWidth), SectionNavMaxWidth)
	return target
}

// renderSectionNav builds the section navigation column.
//
// Takes width (int) and height (int) which are the column dimensions.
//
// Returns string which is the section nav body.
func (p *WatchdogOverviewPanel) renderSectionNav(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	rows := make([]string, 0, len(overviewSections)+1)
	rows = append(rows, p.styledHeader("SECTIONS", width))

	cursor := p.Cursor()
	for i, section := range overviewSections {
		marker := " "
		label := section.Label
		row := fmt.Sprintf(" %s %s", marker, label)
		if i == cursor {
			marker = SectionMarker
			row = fmt.Sprintf(" %s %s", marker, label)
			row = p.cursorStyle().Render(row)
		} else {
			row = p.dimStyle().Render(row)
		}
		rows = append(rows, PadRightANSI(row, width))
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderDashboard composes the right-hand dashboard area.
//
// Takes width (int) and height (int) which are the dashboard dimensions.
//
// Returns string which is the dashboard body.
func (p *WatchdogOverviewPanel) renderDashboard(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	rows := make([]string, 0, AlertTapeMaxRows)
	rows = append(rows, p.renderHeaderStrip(width))
	if err := p.fetchError(); err != nil {
		rows = append(rows, p.errorStyle().Render(PadRightANSI("✗ refresh failed: "+err.Error(), width)))
	}
	rows = append(rows, "")
	rows = append(rows, p.renderGauges(width)...)
	rows = append(rows, "", p.styledHeader("RECENT ALERTS", width))
	rows = append(rows, p.renderAlertTape(width, p.alertTapeBudget(height, len(rows)))...)

	for len(rows) < height {
		rows = append(rows, strings.Repeat(SingleSpace, width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// alertTapeBudget computes the rows available for the alert tape after
// the rest of the dashboard has been laid out.
//
// Takes contentHeight (int) which is the dashboard's total height.
// Takes consumed (int) which is how many rows the dashboard has already
// produced.
//
// Returns int which is the alert tape budget capped at the panel cap.
func (*WatchdogOverviewPanel) alertTapeBudget(contentHeight, consumed int) int {
	available := contentHeight - consumed
	if available < 1 {
		return 0
	}
	if available > overviewAlertTapeMaxRows {
		return overviewAlertTapeMaxRows
	}
	return available
}

// renderHeaderStrip composes the "[ENABLED] | uptime: X | last refresh:
// Y" line at the top of the dashboard.
//
// Takes width (int) which is the dashboard width.
//
// Returns string which is the styled header row.
func (p *WatchdogOverviewPanel) renderHeaderStrip(width int) string {
	status := p.snapshot()
	now := p.clock.Now()

	state := "DISABLED"
	stateStyle := p.warningStyle()
	if status != nil && status.Enabled && !status.Stopped {
		state = "ENABLED"
		stateStyle = p.healthyStyle()
	}
	if status != nil && status.Stopped {
		state = "STOPPED"
		stateStyle = p.errorStyle()
	}

	parts := []string{stateStyle.Render("[" + state + "]")}
	if status != nil {
		uptime := "—"
		if !status.StartedAt.IsZero() {
			uptime = inspector.FormatDuration(now.Sub(status.StartedAt))
		}
		parts = append(parts, p.dimStyle().Render("uptime: ")+p.boldStyle().Render(uptime))

		warmupLabel := "warm-up: complete"
		if status.WarmUpRemaining > 0 {
			warmupLabel = "warm-up: " + inspector.FormatDuration(status.WarmUpRemaining)
		}
		parts = append(parts, p.dimStyle().Render(warmupLabel))
	}

	refreshLabel := "last refresh: —"
	if !p.statusFetched.IsZero() {
		refreshLabel = "last refresh: " + inspector.FormatTimeSince(now, p.statusFetched)
	}
	parts = append(parts, p.dimStyle().Render(refreshLabel))

	return PadRightANSI(strings.Join(parts, "  "), width)
}

// renderGauges composes the four budget gauges.
//
// Takes width (int) which is the dashboard width.
//
// Returns []string with one gauge per row.
func (p *WatchdogOverviewPanel) renderGauges(width int) []string {
	status := p.snapshot()
	if status == nil {
		return []string{p.dimStyle().Render("Awaiting first refresh…")}
	}

	specs := []struct {
		label string
		gauge UtilisationGauge
	}{
		{"Capture window", status.CaptureBudget},
		{"Warning window", status.WarningBudget},
		{"Heap allocation", status.HeapBudget},
		{"Goroutines    ", status.Goroutines},
	}

	rows := make([]string, 0, len(specs))
	for _, spec := range specs {
		rows = append(rows, Gauge(GaugeConfig{
			Theme:    p.theme,
			Label:    spec.label,
			Width:    width,
			Used:     spec.gauge.Used,
			Max:      spec.gauge.Max,
			Severity: spec.gauge.Severity(),
			ShowText: true,
		}))
	}
	return rows
}

// renderAlertTape returns up to budget rows of recent high-priority
// events.
//
// Takes width (int) and budget (int).
//
// Returns []string with one event per row, padded to width.
func (p *WatchdogOverviewPanel) renderAlertTape(width, budget int) []string {
	if budget <= 0 {
		return nil
	}
	events := p.recentHighPriorityEvents(budget)
	if len(events) == 0 {
		return []string{p.dimStyle().Render(PadRightANSI("No high-priority events recorded.", width))}
	}

	now := p.clock.Now()
	rows := make([]string, 0, len(events))
	for _, ev := range events {
		row := p.formatAlertRow(now, ev, width)
		rows = append(rows, row)
	}
	return rows
}

// formatAlertRow formats a single event for the alert tape.
//
// Takes now (time.Time) which is the reference time.
// Takes ev (WatchdogEvent) which is the event.
// Takes width (int) which is the available width.
//
// Returns string which is the formatted, padded row.
func (p *WatchdogOverviewPanel) formatAlertRow(now time.Time, ev WatchdogEvent, width int) string {
	timestamp := inspector.FormatTimeSince(now, ev.EmittedAt)
	glyph := StyledPriorityGlyph(p.theme, ev.Priority)
	typeLabel := string(ev.EventType)
	message := ev.Message
	if message == "" {
		message = "(no message)"
	}

	left := fmt.Sprintf("%s %s  %s", glyph, p.dimStyle().Render(PadRightANSI(timestamp, EventsTimeColumnWidth)), p.boldStyle().Render(typeLabel))
	prefixWidth := TextWidth(left)
	available := max(width-prefixWidth-overlayScreenMargin, EventsTimeColumnWidth)
	message = TruncateANSI(message, available)
	return PadRightANSI(left+DoubleSpace+p.dimStyle().Render(message), width)
}

// recentHighPriorityEvents returns the most-recent events at high
// priority or above, capped at limit.
//
// Takes limit (int) which is the maximum number of events.
//
// Returns []WatchdogEvent in newest-first order.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) recentHighPriorityEvents(limit int) []WatchdogEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]WatchdogEvent, 0, limit)
	for i := len(p.events) - 1; i >= 0 && len(out) < limit; i-- {
		ev := p.events[i]
		if !ev.IsHighOrAbove() {
			continue
		}
		out = append(out, ev)
	}
	return out
}

// recordEvent appends ev to the local event ring, evicting the oldest
// entry when at capacity.
//
// Takes ev (WatchdogEvent) which is the event to record.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) recordEvent(ev WatchdogEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.events) >= overviewLocalEventCap {
		copy(p.events, p.events[1:])
		p.events[len(p.events)-1] = ev
		return
	}
	p.events = append(p.events, ev)
}

// snapshot returns the most recently fetched status under a read lock.
//
// Returns *WatchdogStatus which is the cached snapshot or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) snapshot() *WatchdogStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// fetchError returns the most recent fetch error, if any, under a read
// lock.
//
// Returns error which is the most recent fetch error, or nil when the
// last fetch succeeded.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogOverviewPanel) fetchError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastFetchErr
}

// fetchSnapshotCmd returns a Cmd that asks the provider for a fresh
// status snapshot and posts overviewSnapshotMsg back to Update.
//
// Returns tea.Cmd which performs the fetch.
func (p *WatchdogOverviewPanel) fetchSnapshotCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, errors.New("watchdog overview fetch timed out"))
		defer cancel()
		status, err := p.provider.GetStatus(ctx)
		return overviewSnapshotMsg{Status: status, Err: err}
	}
}

// subscribeCmd opens a dispatcher subscription and returns a Cmd that
// awaits the first event. Subsequent events are awaited via
// waitForNextEventCmd.
//
// Returns tea.Cmd which awaits the first event, or nil when no
// dispatcher is configured.
//
// Concurrency: Safe for concurrent use; guarded by subscriptionMu.
func (p *WatchdogOverviewPanel) subscribeCmd() tea.Cmd {
	if p.dispatcher == nil {
		return nil
	}
	p.subscriptionMu.Lock()
	if p.subscription != nil {
		p.subscription.Cancel()
	}
	sub := p.dispatcher.Subscribe(EventFilter{}, p.clock.Now().Add(-time.Hour))
	p.subscription = &sub
	p.subscriptionMu.Unlock()

	return p.waitForNextEventCmd()
}

// waitForNextEventCmd reads a single event from the active subscription
// and packages it as overviewEventMsg.
//
// Returns tea.Cmd which awaits the next event, or nil when the panel is
// not subscribed.
//
// Concurrency: Safe for concurrent use; guarded by subscriptionMu.
func (p *WatchdogOverviewPanel) waitForNextEventCmd() tea.Cmd {
	p.subscriptionMu.Lock()
	sub := p.subscription
	p.subscriptionMu.Unlock()
	if sub == nil {
		return nil
	}

	return func() tea.Msg {
		ev, ok := <-sub.Events
		if !ok {
			return overviewEventMsg{Done: true}
		}
		return overviewEventMsg{Event: ev}
	}
}

// handleKey processes keyboard input directed at the panel.
//
// Takes message (tea.KeyPressMsg) which is the key event.
//
// Returns tea.Cmd which is the command resulting from the keystroke,
// or nil when no action was taken.
func (p *WatchdogOverviewPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	if message.String() == "R" {
		return p.fetchSnapshotCmd()
	}
	if p.HandleNavigation(message, len(overviewSections)) {
		return nil
	}
	return nil
}

// styledHeader renders a section header inside the dashboard.
//
// Takes label (string) which is the heading text.
// Takes width (int) which is the available width.
//
// Returns string which is the padded, styled heading.
func (p *WatchdogOverviewPanel) styledHeader(label string, width int) string {
	style := p.boldStyle()
	if p.theme != nil {
		style = p.theme.PanelTitle
	}
	return PadRightANSI(style.Render(label), width)
}

// boldStyle returns a bold-text style with theme support.
//
// Returns lipgloss.Style which is the styled output for bold text.
func (p *WatchdogOverviewPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim-text style.
//
// Returns lipgloss.Style which is the dim-foreground style.
func (p *WatchdogOverviewPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// healthyStyle returns the healthy-status style.
//
// Returns lipgloss.Style which is the healthy-foreground style.
func (p *WatchdogOverviewPanel) healthyStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusHealthy
	}
	return statusHealthyStyle
}

// warningStyle returns the warning-status style.
//
// Returns lipgloss.Style which is the warning-foreground style.
func (p *WatchdogOverviewPanel) warningStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusDegraded
	}
	return statusDegradedStyle
}

// errorStyle returns the error-status style.
//
// Returns lipgloss.Style which is the error-foreground style.
func (p *WatchdogOverviewPanel) errorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusUnhealthy
	}
	return statusUnhealthyStyle
}

// cursorStyle returns the style applied to the focused section nav
// entry.
//
// Returns lipgloss.Style which is the cursor style.
func (p *WatchdogOverviewPanel) cursorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Selected
	}
	return navItemActiveStyle
}
