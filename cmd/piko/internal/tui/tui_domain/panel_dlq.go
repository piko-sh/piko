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
	// dlqRefreshTimeout caps the dispatcher summary RPC fetch.
	dlqRefreshTimeout = 5 * time.Second

	// dlqEntriesPerType is the per-dispatcher slice cap for the
	// detail-pane entry preview.
	dlqEntriesPerType = 25
)

// dlqRefreshMessage carries the result of a DispatcherSummaries fetch.
type dlqRefreshMessage struct {
	// err is the RPC error, or nil on success.
	err error

	// summaries is the dispatcher summary list returned by the inspector.
	summaries []DispatcherSummary
}

// dlqEntriesMessage carries the result of a ListDLQEntries fetch.
type dlqEntriesMessage struct {
	// err is the RPC error, or nil on success.
	err error

	// dispatcherType identifies the dispatcher whose entries were fetched.
	dispatcherType string

	// entries is the list of DLQ entries returned by the inspector.
	entries []DLQEntry
}

// DLQPanel surfaces dispatcher dead-letter queues. It is the TUI
// counterpart of `piko get dlq` / `piko describe dlq`.
//
// The panel runs in two cursor modes: dispatcherMode navigates the
// summary list in the centre column, entryMode navigates the
// dead-lettered entries shown in the detail column for the active
// dispatcher. Tab toggles between modes.
type DLQPanel struct {
	// lastRefresh records when the panel last received a summary
	// payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the dispatcher summaries + DLQ entries.
	provider DLQInspector

	// err holds the last summary refresh error, or nil after success.
	err error

	*AssetViewer[DispatcherSummary]

	// entries caches per-dispatcher DLQ entry slices, keyed by type.
	entries map[string][]DLQEntry

	// entryErrors records per-dispatcher fetch errors, keyed by type.
	entryErrors map[string]error

	// entryCursorByType records the per-dispatcher cursor into the
	// entry list shown in the detail pane.
	entryCursorByType map[string]int

	// stateMutex guards err / entries / entryErrors / entry cursors
	// for safe concurrent reads.
	stateMutex sync.RWMutex

	// entryMode reports whether keyboard navigation drives the
	// detail-pane entry cursor (true) or the centre dispatcher list
	// (false). Tab toggles.
	entryMode bool
}

var _ Panel = (*DLQPanel)(nil)

// dlqRenderer renders DispatcherSummary rows for the DLQ panel.
type dlqRenderer struct {
	// panel is the owning DLQPanel.
	panel *DLQPanel
}

// NewDLQPanel constructs a DLQPanel.
//
// Takes provider (DLQInspector) which supplies dispatcher summaries.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *DLQPanel ready to register with a group.
func NewDLQPanel(provider DLQInspector, c clock.Clock) *DLQPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &DLQPanel{
		AssetViewer:       nil,
		clock:             c,
		provider:          provider,
		stateMutex:        sync.RWMutex{},
		entries:           map[string][]DLQEntry{},
		entryErrors:       map[string]error{},
		entryCursorByType: map[string]int{},
	}
	p.AssetViewer = NewAssetViewer(AssetViewerConfig[DispatcherSummary]{
		ID:           "dlq",
		Title:        "DLQ",
		Renderer:     &dlqRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓ or j/k", Description: "Navigate"},
			{Key: "Tab", Description: "Toggle dispatcher / entry cursor"},
			{Key: "/", Description: "Search"},
			{Key: "r", Description: "Refresh"},
		},
	})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which is the initial refresh command.
func (p *DLQPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the incoming update message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
func (p *DLQPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if cmd, handled := forwardSearchUpdate(p.AssetViewer, message); handled {
		return p, cmd
	}
	switch msg := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	case dlqRefreshMessage:
		p.handleSummaries(msg)
		cmd := p.fetchSelectedEntriesCmd()
		return p, cmd
	case dlqEntriesMessage:
		p.handleEntries(msg)
		return p, nil
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the centre. Falls back to a "feature disabled" hint
// when the server does not expose the dispatcher inspector service.
//
// Takes width (int) which is the rendering width in columns.
// Takes height (int) which is the rendering height in rows.
//
// Returns string which is the rendered panel content.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) View(width, height int) string {
	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()
	if IsServiceUnavailable(err) {
		p.SetSize(width, height)
		hint := ServiceUnavailableHint("DLQ inspection",
			"This deployment has no dispatchers, or the server build omits the dispatcher inspector.")
		return p.RenderFrame(RenderDimText(hint))
	}
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderHeader,
		RenderEmptyState:    p.renderEmptyState,
		RenderItems:         p.renderItems,
		TrimTrailingNewline: false,
	})
}

// DetailView renders the right-pane detail with per-dispatcher entries.
//
// Takes width (int) which is the detail-pane width in columns.
// Takes height (int) which is the detail-pane height in rows.
//
// Returns string which is the rendered detail body.
func (p *DLQPanel) DetailView(width, height int) string {
	body := p.detailBody()
	return RenderDetailBody(nil, body, width, height)
}

// renderHeader writes the search and error header lines.
//
// Takes content (*strings.Builder) which receives the rendered header lines.
//
// Returns int which is the number of header lines written.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) renderHeader(content *strings.Builder) int {
	used := 0
	if p.Search() != nil {
		used += p.Search().RenderHeader(content, len(p.Items()))
	}
	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()
	if err != nil {
		RenderErrorState(content, err)
		used++
	}
	return used
}

// renderEmptyState writes the placeholder shown when no dispatchers exist.
//
// Takes content (*strings.Builder) which receives the placeholder text.
func (*DLQPanel) renderEmptyState(content *strings.Builder) {
	content.WriteString(RenderDimText("No dispatchers configured"))
}

// renderItems writes one row per visible dispatcher summary.
//
// Takes content (*strings.Builder) which receives the rendered rows.
// Takes displayItems ([]int) which is the indices of items to render.
// Takes headerLines (int) which is the number of header lines already
// consumed by renderHeader, used to size the scroll context.
func (p *DLQPanel) renderItems(content *strings.Builder, displayItems []int, headerLines int) {
	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()
	for _, index := range displayItems {
		if index >= len(items) {
			continue
		}
		row := items[index]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()
		ctx.WriteLineIfVisible(func() string {
			return p.renderRow(row, selected)
		})
	}
}

// renderRow renders a single dispatcher row with cursor and counters.
//
// Takes summary (DispatcherSummary) which is the row to render.
// Takes selected (bool) which is true when the cursor sits on this row.
//
// Returns string which is the rendered row.
func (p *DLQPanel) renderRow(summary DispatcherSummary, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	label := fmt.Sprintf("%-12s queued=%d  dlq=%d  failed=%d", summary.Type, summary.QueuedItems, summary.DeadLetterCount, summary.TotalFailed)
	return cursor + " " + RenderName(label, max(0, p.ContentWidth()-3), selected, p.Focused())
}

// handleKey routes a key press to the appropriate navigation handler.
//
// Takes msg (tea.KeyPressMsg) which is the key event.
//
// Returns Panel which is the receiver.
// Returns tea.Cmd which is the resulting command, or nil.
func (p *DLQPanel) handleKey(msg tea.KeyPressMsg) (Panel, tea.Cmd) {
	if msg.String() == "tab" {
		p.toggleEntryMode()
		return p, nil
	}
	if p.entryMode {
		if p.handleEntryNavigation(msg) {
			return p, nil
		}
	}
	result := HandleCommonKeys(p.AssetViewer, msg, p.refresh)
	if result.Handled {
		return p, tea.Batch(result.Cmd, p.fetchSelectedEntriesCmd())
	}
	return p, nil
}

// handleSummaries processes a DispatcherSummaries refresh result.
//
// Takes msg (dlqRefreshMessage) which carries the summaries and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) handleSummaries(msg dlqRefreshMessage) {
	p.stateMutex.Lock()
	if msg.err != nil {
		p.err = msg.err
		p.stateMutex.Unlock()
		return
	}
	p.err = nil
	p.lastRefresh = p.clock.Now()
	p.stateMutex.Unlock()

	p.SetItems(msg.summaries)
}

// handleEntries records a ListDLQEntries result in the entries cache.
//
// Takes msg (dlqEntriesMessage) which carries the entries and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) handleEntries(msg dlqEntriesMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if msg.err != nil {
		p.entryErrors[msg.dispatcherType] = msg.err
		return
	}
	delete(p.entryErrors, msg.dispatcherType)
	p.entries[msg.dispatcherType] = msg.entries
}

// refresh issues a DispatcherSummaries RPC and returns the result message.
//
// Returns tea.Cmd which delivers a dlqRefreshMessage.
func (p *DLQPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return dlqRefreshMessage{err: errNoDLQInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), dlqRefreshTimeout,
			errors.New("DLQ summary exceeded timeout"))
		defer cancel()
		summaries, err := p.provider.DispatcherSummaries(ctx)
		return dlqRefreshMessage{summaries: summaries, err: err}
	}
}

// fetchSelectedEntriesCmd issues a ListDLQEntries RPC for the selected
// dispatcher when its dead-letter count is non-zero.
//
// Returns tea.Cmd which delivers a dlqEntriesMessage, or nil when no
// fetch is required.
func (p *DLQPanel) fetchSelectedEntriesCmd() tea.Cmd {
	current := p.GetItemAtCursor()
	if current == nil || p.provider == nil || current.DeadLetterCount == 0 {
		return nil
	}
	dispatcherType := current.Type
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), dlqRefreshTimeout,
			errors.New("DLQ entries exceeded timeout"))
		defer cancel()
		entries, err := p.provider.ListDLQEntries(ctx, dispatcherType, dlqEntriesPerType)
		return dlqEntriesMessage{dispatcherType: dispatcherType, entries: entries, err: err}
	}
}

// detailBody composes the detail-pane body for the selected dispatcher.
//
// Returns inspector.DetailBody describing the panel state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) detailBody() inspector.DetailBody {
	current := p.GetItemAtCursor()
	if current == nil {
		return p.overviewBody()
	}
	rows := []inspector.DetailRow{
		{Label: "Type", Value: current.Type},
		{Label: "Queued", Value: fmt.Sprintf(fmtDecimal, current.QueuedItems)},
		{Label: "Dead-letter", Value: fmt.Sprintf(fmtDecimal, current.DeadLetterCount)},
		{Label: "Retry queue", Value: fmt.Sprintf(fmtDecimal, current.RetryQueueSize)},
		{Label: "Processed", Value: fmt.Sprintf(fmtDecimal, current.TotalProcessed)},
		{Label: "Successful", Value: fmt.Sprintf(fmtDecimal, current.TotalSuccessful)},
		{Label: "Failed", Value: fmt.Sprintf(fmtDecimal, current.TotalFailed)},
		{Label: "Retries", Value: fmt.Sprintf(fmtDecimal, current.TotalRetries)},
		{Label: "Uptime", Value: current.Uptime.Truncate(time.Second).String()},
	}
	sections := []inspector.DetailSection{{Heading: "Counters", Rows: rows}}

	p.stateMutex.RLock()
	entries := p.entries[current.Type]
	entriesErr := p.entryErrors[current.Type]
	entryCursor := p.entryCursorByType[current.Type]
	entryMode := p.entryMode
	p.stateMutex.RUnlock()

	switch {
	case entriesErr != nil:
		sections = append(sections, inspector.DetailSection{Heading: "Entries", Rows: []inspector.DetailRow{{Label: "Error", Value: entriesErr.Error()}}})
	case len(entries) > 0:
		sections = append(sections, dlqEntriesListSection(entries, entryCursor, entryMode))
		if entryMode && entryCursor >= 0 && entryCursor < len(entries) {
			sections = append(sections, dlqEntryFocusSection(entries[entryCursor]))
		}
	}

	subtitle := fmt.Sprintf("%d dead-lettered · %d failed", current.DeadLetterCount, current.TotalFailed)
	if entryMode {
		subtitle += " · entry-mode (Tab to switch)"
	} else if len(entries) > 0 {
		subtitle += " · Tab to navigate entries"
	}

	return inspector.DetailBody{
		Title:    current.Type + " dispatcher",
		Subtitle: subtitle,
		Sections: sections,
	}
}

// dlqEntriesListSection renders the dispatcher's recent
// dead-lettered entries with a cursor when entryMode is active.
//
// Takes entries ([]DLQEntry) which is the cached list.
// Takes cursor (int) which is the index of the focused entry.
// Takes entryMode (bool) which is true when the entry cursor is the
// active navigation target.
//
// Returns inspector.DetailSection ready to append to the body.
func dlqEntriesListSection(entries []DLQEntry, cursor int, entryMode bool) inspector.DetailSection {
	rows := make([]inspector.DetailRow, 0, len(entries))
	for i, e := range entries {
		label := e.ID
		if label == "" {
			label = "—"
		}
		marker := DoubleSpace
		if entryMode && i == cursor {
			marker = MenuMarker + SingleSpace
		}
		summary := fmt.Sprintf("attempts=%d · %s", e.TotalAttempts, oneLineError(e.OriginalError))
		rows = append(rows, inspector.DetailRow{
			Label: marker + label,
			Value: summary,
		})
	}
	return inspector.DetailSection{Heading: "Recent dead-lettered", Rows: rows}
}

// dlqEntryFocusSection renders the deep-dive section for one DLQ
// entry: full error message, attempt count, and timestamps.
//
// Takes e (DLQEntry) which is the focused entry.
//
// Returns inspector.DetailSection ready to append to the body.
func dlqEntryFocusSection(e DLQEntry) inspector.DetailSection {
	rows := []inspector.DetailRow{
		{Label: "ID", Value: e.ID},
		{Label: "Type", Value: e.Type},
		{Label: "Attempts", Value: fmt.Sprintf(fmtDecimal, e.TotalAttempts)},
		{Label: "Added", Value: inspector.FormatDetailTime(e.AddedAt)},
		{Label: "Last attempt", Value: inspector.FormatDetailTime(e.LastAttempt)},
		{Label: "Error", Value: e.OriginalError},
	}
	return inspector.DetailSection{Heading: "Selected entry", Rows: rows}
}

// oneLineError collapses error messages to a single line so the row
// summary stays readable; the full message is rendered in the
// "Selected entry" focus section when entry mode is active.
//
// Takes message (string) which is the original error message.
//
// Returns string with newlines replaced by " ".
func oneLineError(message string) string {
	if !strings.ContainsAny(message, "\r\n") {
		return message
	}
	collapsed := strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(message)
	return strings.TrimSpace(collapsed)
}

// toggleEntryMode flips between dispatcher-list navigation and
// entry-list navigation. When switching into entry mode with no
// entries cached, the panel still flips so the next refresh can show
// them.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) toggleEntryMode() {
	p.stateMutex.Lock()
	p.entryMode = !p.entryMode
	p.stateMutex.Unlock()
}

// handleEntryNavigation drives the per-dispatcher entry cursor when
// entry mode is active. Returns true when the key was consumed.
//
// Takes msg (tea.KeyPressMsg) which is the key event.
//
// Returns bool which is true when the key was an entry-cursor move.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) handleEntryNavigation(msg tea.KeyPressMsg) bool {
	current := p.GetItemAtCursor()
	if current == nil {
		return false
	}
	dispatcherType := current.Type
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	entries := p.entries[dispatcherType]
	if len(entries) == 0 {
		return false
	}
	cursor := p.entryCursorByType[dispatcherType]
	switch msg.String() {
	case "up", "k":
		cursor = max(0, cursor-1)
	case "down", "j":
		cursor = min(len(entries)-1, cursor+1)
	case "g":
		cursor = 0
	case "G":
		cursor = len(entries) - 1
	default:
		return false
	}
	p.entryCursorByType[dispatcherType] = cursor
	return true
}

// overviewBody renders the overview shown when no dispatcher is selected.
//
// Returns inspector.DetailBody describing the overview state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *DLQPanel) overviewBody() inspector.DetailBody {
	itemCount := len(p.Items())
	p.stateMutex.RLock()
	lastRefresh := p.lastRefresh
	err := p.err
	p.stateMutex.RUnlock()
	return inspectorOverviewBody(inspectorOverviewArgs{
		title:       "DLQ overview",
		itemLabel:   "Dispatchers",
		itemCount:   itemCount,
		lastRefresh: lastRefresh,
		err:         err,
	})
}

// RenderRow renders a dispatcher summary as a single list row.
//
// Takes summary (DispatcherSummary) which is the row to render.
// Takes selected (bool) which is true when the cursor sits on this row.
//
// Returns string which is the rendered row.
func (r *dlqRenderer) RenderRow(summary DispatcherSummary, _ int, selected, _ bool, _ int) string {
	return r.panel.renderRow(summary, selected)
}

// RenderExpanded returns no expanded lines; dispatcher rows are not expandable.
//
// Returns []string which is always nil for dispatcher rows.
func (*dlqRenderer) RenderExpanded(_ DispatcherSummary, _ int) []string { return nil }

// GetID returns the unique dispatcher type used as the row identifier.
//
// Takes summary (DispatcherSummary) which is the row to identify.
//
// Returns string which is the dispatcher type used as the unique id.
func (*dlqRenderer) GetID(summary DispatcherSummary) string { return summary.Type }

// MatchesFilter reports whether a summary matches the supplied search query.
//
// Takes summary (DispatcherSummary) which is the row to filter.
// Takes query (string) which is the search query.
//
// Returns bool which is true when the dispatcher type contains query.
func (*dlqRenderer) MatchesFilter(summary DispatcherSummary, query string) bool {
	return strings.Contains(strings.ToLower(summary.Type), strings.ToLower(query))
}

// IsExpandable reports whether the row can be expanded; always false here.
//
// Returns bool which is always false for dispatcher rows.
func (*dlqRenderer) IsExpandable(_ DispatcherSummary) bool { return false }

// ExpandedLineCount returns the number of expanded lines; always zero.
//
// Returns int which is always zero for dispatcher rows.
func (*dlqRenderer) ExpandedLineCount(_ DispatcherSummary) int { return 0 }
