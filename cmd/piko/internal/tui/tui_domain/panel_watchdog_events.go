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
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogEventsPanelID is the identifier used by the model to look
	// up the watchdog Events panel.
	WatchdogEventsPanelID = "watchdog-events"

	// WatchdogEventsPanelTitle is the display title shown in the tab
	// bar and panel frame.
	WatchdogEventsPanelTitle = "Watchdog Events"

	// eventsLocalCap is the maximum number of events buffered in the
	// panel's local ring. Older events are evicted as new ones arrive.
	eventsLocalCap = 256

	// eventsBackfillWindow is how far back the dispatcher subscription
	// should deliver historical events at panel attach time.
	eventsBackfillWindow = time.Hour
)

// eventsViewMessage carries a single live event delivered through the
// dispatcher subscription into the Events panel.
type eventsViewMessage struct {
	// Event is the watchdog event delivered by the dispatcher.
	Event WatchdogEvent

	// Done is true when the subscription channel has closed.
	Done bool
}

// WatchdogEventsPanel renders the live event stream with priority
// colouring, pause/resume, and inline detail expansion of the selected
// row.
type WatchdogEventsPanel struct {
	// clock yields the current time and supports test injection.
	clock clock.Clock

	// dispatcher streams events; nil renders a placeholder body.
	dispatcher *EventDispatcher

	// theme is the active colour theme, or nil to fall back to defaults.
	theme *Theme

	// subscription is the active dispatcher subscription, or nil.
	subscription *WatchdogSubscription

	// events is the local ring of recent events, oldest first.
	events []WatchdogEvent

	// BasePanel provides the shared panel boilerplate.
	BasePanel

	// minimumPriority filters out events below this priority.
	minimumPriority WatchdogEventPriority

	// totalReceived counts every event delivered to the panel.
	totalReceived uint64

	// mu guards events, totalReceived, and minimumPriority.
	mu sync.RWMutex

	// subscriptionMu guards subscription against concurrent reopening.
	subscriptionMu sync.Mutex

	// paused indicates whether incoming events should auto-advance the cursor.
	paused bool

	// expanded indicates whether the focused row's details are visible.
	expanded bool
}

// Compile-time interface assertions.
var (
	_ Panel = (*WatchdogEventsPanel)(nil)

	_ ThemeAware = (*WatchdogEventsPanel)(nil)
)

// NewWatchdogEventsPanel constructs the Events panel.
//
// Takes dispatcher (*EventDispatcher) which streams live events. Pass
// nil to render a placeholder; the panel functions but only shows a
// "no event source" hint.
// Takes clk (clock.Clock) which yields the current time.
//
// Returns *WatchdogEventsPanel ready for AddPanel.
func NewWatchdogEventsPanel(dispatcher *EventDispatcher, clk clock.Clock) *WatchdogEventsPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogEventsPanel{
		BasePanel:  NewBasePanel(WatchdogEventsPanelID, WatchdogEventsPanelTitle),
		dispatcher: dispatcher,
		clock:      clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "j / Down", Description: "Next event"},
		{Key: "k / Up", Description: "Previous event"},
		{Key: "g", Description: "Top"},
		{Key: "G", Description: "Bottom"},
		{Key: "Space", Description: "Pause / resume"},
		{Key: "Enter", Description: "Toggle event details"},
		{Key: "f", Description: "Cycle priority filter"},
	})
	return panel
}

// SetTheme implements ThemeAware.
//
// Takes theme (*Theme) which is the new colour theme to apply.
func (p *WatchdogEventsPanel) SetTheme(theme *Theme) {
	p.theme = theme
}

// Init opens the dispatcher subscription if one is configured and
// returns the command that awaits the first event.
//
// Returns tea.Cmd which is the wait-for-first-event command, or nil.
func (p *WatchdogEventsPanel) Init() tea.Cmd {
	return p.subscribeCmd()
}

// Update handles event delivery, pause/resume, navigation, and filter
// cycling.
//
// Takes message (tea.Msg) which is the incoming update message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
func (p *WatchdogEventsPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case eventsViewMessage:
		if msg.Done {
			cmd := p.subscribeCmd()
			return p, cmd
		}
		p.recordEvent(msg.Event)
		cmd := p.waitForNextEventCmd()
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
// Takes width (int) which is the rendering width in columns.
// Takes height (int) which is the rendering height in rows.
//
// Returns string which is the rendered panel content.
func (p *WatchdogEventsPanel) View(width, height int) string {
	p.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()
	if contentWidth <= 0 || contentHeight <= 0 {
		return p.RenderFrame("")
	}

	body := p.composeBody(contentWidth, contentHeight)
	return p.RenderFrame(body)
}

// composeBody arranges the header, event rows, and footer of the panel.
//
// Takes width (int) and height (int) which are the inner content
// dimensions.
//
// Returns string which is the composed body.
func (p *WatchdogEventsPanel) composeBody(width, height int) string {
	rows := make([]string, 0, height)
	rows = append(rows, p.renderHeaderRow(width), "")

	listHeight := max(

		height-3, 1)
	rows = append(rows, p.renderEventList(width, listHeight)...)

	for len(rows) < height-1 {
		rows = append(rows, strings.Repeat(" ", width))
	}
	rows = append(rows, p.renderFooterRow(width))

	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderHeaderRow shows the streaming state, total received, and
// dropped counters.
//
// Takes width (int) which is the available width.
//
// Returns string which is the styled header row.
func (p *WatchdogEventsPanel) renderHeaderRow(width int) string {
	stateLabel := p.streamStateLabel()
	pauseLabel := "▶ live"
	if p.paused {
		pauseLabel = "⏸ paused"
	}

	var droppedTotal uint64
	if p.dispatcher != nil {
		droppedTotal = p.dispatcher.DroppedTotal()
	}

	left := p.boldStyle().Render(WatchdogEventsPanelTitle) + "  " +
		p.streamStateStyle().Render(pauseLabel)
	right := p.dimStyle().Render(fmt.Sprintf("state: %s · received %d · dropped %d",
		stateLabel, p.receivedCount(), droppedTotal))

	leftWidth := TextWidth(left)
	rightWidth := TextWidth(right)
	gap := max(1, width-leftWidth-rightWidth)
	return PadRightANSI(left+strings.Repeat(" ", gap)+right, width)
}

// renderEventList composes the event rows shown in the list area.
//
// Takes width (int) and height (int).
//
// Returns []string with at most height rows.
func (p *WatchdogEventsPanel) renderEventList(width, height int) []string {
	events := p.visibleEvents()
	if len(events) == 0 {
		return []string{p.dimStyle().Render(PadRightANSI("No events received yet.", width))}
	}

	cursor := p.Cursor()
	if cursor >= len(events) {
		cursor = len(events) - 1
		p.SetCursor(cursor)
	}

	startIdx := len(events) - 1 - p.ScrollOffset()
	endIdx := max(startIdx-height+1, 0)

	now := p.clock.Now()
	rows := make([]string, 0, height)
	for i := startIdx; i >= endIdx && len(rows) < height; i-- {
		ev := events[i]
		selected := (len(events) - 1 - i) == cursor
		row := p.renderEventRow(now, ev, width, selected)
		rows = append(rows, row)

		if selected && p.expanded {
			detail := p.renderEventDetail(ev, width)
			for _, dr := range detail {
				if len(rows) >= height {
					break
				}
				rows = append(rows, dr)
			}
		}
	}

	for len(rows) < height {
		rows = append(rows, strings.Repeat(" ", width))
	}
	return rows
}

// renderEventRow formats a single event into a row.
//
// Takes ev (WatchdogEvent) which is the event to render.
// Takes width (int) which is the row width in columns.
// Takes selected (bool) which is true when the row is the focused row.
//
// Returns string which is the formatted row.
func (p *WatchdogEventsPanel) renderEventRow(_ time.Time, ev WatchdogEvent, width int, selected bool) string {
	timestamp := ev.EmittedAt.Format("15:04:05")
	glyph := StyledPriorityGlyph(p.theme, ev.Priority)
	typeLabel := string(ev.EventType)
	message := ev.Message
	if message == "" {
		message = "(no message)"
	}

	cursorMark := DoubleSpace
	if selected {
		cursorMark = SectionMarker + SingleSpace
	}

	prefix := cursorMark +
		p.dimStyle().Render(timestamp) + DoubleSpace +
		glyph + SingleSpace +
		p.boldStyle().Render(typeLabel)
	prefixWidth := TextWidth(prefix)

	available := max(width-prefixWidth-overlayScreenMargin, EventsTimeColumnWidth)
	message = TruncateANSI(message, available)
	row := prefix + DoubleSpace + p.dimStyle().Render(message)

	if selected {
		row = p.cursorStyle().Render(PadRightANSI(row, width))
	}
	return PadRightANSI(row, width)
}

// renderEventDetail returns the indented key-value rows for the
// expanded event detail view.
//
// Takes ev (WatchdogEvent) and width (int).
//
// Returns []string with one row per detail line.
func (p *WatchdogEventsPanel) renderEventDetail(ev WatchdogEvent, width int) []string {
	if width <= EventsTimeColumnWidth {
		return nil
	}

	keys := make([]string, 0, len(ev.Fields))
	for k := range ev.Fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	rows := make([]string, 0, len(keys)+2)
	rows = append(rows, PadRightANSI("    "+p.dimStyle().Render("emitted: ")+
		p.boldStyle().Render(ev.EmittedAt.Format(time.RFC3339Nano)), width))
	for _, k := range keys {
		row := "    " + p.boldStyle().Render(k) + ": " + p.dimStyle().Render(ev.Fields[k])
		rows = append(rows, PadRightANSI(row, width))
	}
	return rows
}

// renderFooterRow shows the active priority filter and any keymap
// hint relevant to the current state.
//
// Takes width (int).
//
// Returns string which is the styled footer.
func (p *WatchdogEventsPanel) renderFooterRow(width int) string {
	parts := []string{
		p.dimStyle().Render("filter: " + p.priorityFilterLabel()),
	}
	if p.paused {
		parts = append(parts, p.warningStyle().Render("PAUSED"))
	}
	body := strings.Join(parts, "  ")
	return PadRightANSI(body, width)
}

// streamStateLabel returns a human-readable label for the dispatcher's
// connection state.
//
// Returns string which is the human-readable state label.
func (p *WatchdogEventsPanel) streamStateLabel() string {
	if p.dispatcher == nil {
		return "no source"
	}
	return p.dispatcher.State()
}

// streamStateStyle styles the live indicator according to the
// dispatcher's connection state and the panel's pause flag.
//
// Returns lipgloss.Style which is the styled live indicator.
func (p *WatchdogEventsPanel) streamStateStyle() lipgloss.Style {
	if p.paused {
		return p.warningStyle()
	}
	if p.dispatcher != nil && p.dispatcher.State() == WatchdogStreamConnected {
		return p.healthyStyle()
	}
	return p.dimStyle()
}

// priorityFilterLabel returns the label for the active priority filter.
//
// Returns string which is the priority filter label.
func (p *WatchdogEventsPanel) priorityFilterLabel() string {
	switch p.minimumPriority {
	case WatchdogPriorityCritical:
		return "critical only"
	case WatchdogPriorityHigh:
		return "high or above"
	default:
		return "all priorities"
	}
}

// recordEvent appends ev to the panel's local ring, evicting the
// oldest entry when at capacity. When the panel is paused, the event
// is still recorded but the cursor does not auto-advance to the new
// row.
//
// Takes ev (WatchdogEvent) which is the event to record.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogEventsPanel) recordEvent(ev WatchdogEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalReceived++
	if len(p.events) >= eventsLocalCap {
		copy(p.events, p.events[1:])
		p.events[len(p.events)-1] = ev
		return
	}
	p.events = append(p.events, ev)
}

// receivedCount returns the cumulative event count.
//
// Returns uint64 which is the total number of events received.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogEventsPanel) receivedCount() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.totalReceived
}

// visibleEvents returns the events filtered by the active priority
// filter, oldest-first.
//
// Returns []WatchdogEvent which is the filtered event list.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogEventsPanel) visibleEvents() []WatchdogEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.minimumPriority == 0 {
		out := make([]WatchdogEvent, len(p.events))
		copy(out, p.events)
		return out
	}
	out := make([]WatchdogEvent, 0, len(p.events))
	for _, ev := range p.events {
		if ev.Priority >= p.minimumPriority {
			out = append(out, ev)
		}
	}
	return out
}

// subscribeCmd opens (or reopens) the dispatcher subscription.
//
// Returns tea.Cmd which awaits the next event, or nil if no dispatcher.
//
// Concurrency: Safe for concurrent use; guarded by subscriptionMu.
func (p *WatchdogEventsPanel) subscribeCmd() tea.Cmd {
	if p.dispatcher == nil {
		return nil
	}
	p.subscriptionMu.Lock()
	if p.subscription != nil {
		p.subscription.Cancel()
	}
	since := p.clock.Now().Add(-eventsBackfillWindow)
	p.subscription = new(p.dispatcher.Subscribe(EventFilter{}, since))
	p.subscriptionMu.Unlock()
	return p.waitForNextEventCmd()
}

// waitForNextEventCmd reads the next event from the active subscription.
//
// Returns tea.Cmd which yields the next eventsViewMessage, or nil.
//
// Concurrency: Safe for concurrent use; guarded by subscriptionMu.
func (p *WatchdogEventsPanel) waitForNextEventCmd() tea.Cmd {
	p.subscriptionMu.Lock()
	sub := p.subscription
	p.subscriptionMu.Unlock()
	if sub == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-sub.Events
		if !ok {
			return eventsViewMessage{Done: true}
		}
		return eventsViewMessage{Event: ev}
	}
}

// handleKey handles panel-specific key bindings.
//
// Takes message (tea.KeyPressMsg) which is the key press event.
//
// Returns tea.Cmd which is the resulting command, or nil.
func (p *WatchdogEventsPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "space", " ":
		p.paused = !p.paused
		return nil
	case "enter":
		p.expanded = !p.expanded
		return nil
	case "f":
		p.cyclePriorityFilter()

		p.SetCursor(0)
		p.SetScrollOffset(0)
		return nil
	}

	if p.HandleNavigation(message, len(p.visibleEvents())) {
		return nil
	}
	return nil
}

// cyclePriorityFilter advances the priority filter through (all,
// high+, critical only, all).
func (p *WatchdogEventsPanel) cyclePriorityFilter() {
	switch p.minimumPriority {
	case 0:
		p.minimumPriority = WatchdogPriorityHigh
	case WatchdogPriorityHigh:
		p.minimumPriority = WatchdogPriorityCritical
	default:
		p.minimumPriority = 0
	}
}

// boldStyle returns the bold-text style with theme support.
//
// Returns lipgloss.Style which is the bold style.
func (p *WatchdogEventsPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim-text style.
//
// Returns lipgloss.Style which is the dim style.
func (p *WatchdogEventsPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// healthyStyle returns the healthy-status style.
//
// Returns lipgloss.Style which is the healthy-status style.
func (p *WatchdogEventsPanel) healthyStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusHealthy
	}
	return statusHealthyStyle
}

// warningStyle returns the warning-status style.
//
// Returns lipgloss.Style which is the warning-status style.
func (p *WatchdogEventsPanel) warningStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusDegraded
	}
	return statusDegradedStyle
}

// cursorStyle returns the style applied to the focused event row.
//
// Returns lipgloss.Style which is the focused-row style.
func (p *WatchdogEventsPanel) cursorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Selected
	}
	return navItemActiveStyle
}
