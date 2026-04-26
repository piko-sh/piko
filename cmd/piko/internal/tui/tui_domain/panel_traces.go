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
	// tracesFetchLimit is the maximum number of spans to fetch from the provider.
	tracesFetchLimit = 100

	// tracesSlowThreshold is the duration above which a span is marked as slow.
	tracesSlowThreshold = 100 * time.Millisecond

	// tracesHoursPerDay is the number of hours in a day, used to format time.
	tracesHoursPerDay = 24

	// tracesCursorWidth is the width of the cursor column in the traces table.
	tracesCursorWidth = 2

	// tracesStatusWidth is the column width for the trace status indicator.
	tracesStatusWidth = 2

	// tracesDurationWidth is the column width for the trace duration field.
	tracesDurationWidth = 10

	// tracesTimeWidth is the column width for trace timestamps.
	tracesTimeWidth = 8

	// tracesSpacingWidth is the total spacing between columns in the traces panel.
	tracesSpacingWidth = 5

	// tracesMinNameWidth is the minimum width for the name column.
	tracesMinNameWidth = 30

	// tracesMinServiceW is the minimum width in characters for the service column.
	tracesMinServiceW = 10

	// tracesNameWidthRatio is the portion of available width given to the name column.
	tracesNameWidthRatio = 0.7

	// tracesMaxHistory is the maximum number of spans to keep in history.
	tracesMaxHistory = 5000
)

var (
	_ Panel = (*TracesPanel)(nil)

	_ ItemRenderer[Span] = (*spanRenderer)(nil)
)

// TracesPanel shows recent traces and spans in the debug interface.
// It implements the Panel interface and supports filtering to show only errors.
type TracesPanel struct {
	*AssetViewer[Span]

	// clock provides time functions to track when data was last updated.
	clock clock.Clock

	// lastRefresh is when the data was last fetched from the source.
	lastRefresh time.Time

	// provider fetches trace spans; nil means traces are not available.
	provider TracesProvider

	// err holds the most recent refresh error; nil means success.
	err error

	// seenSpanIDs tracks which span IDs have been seen to prevent duplicates.
	seenSpanIDs map[string]struct{}

	// errorsOnly filters the trace list to show only traces that contain errors.
	errorsOnly bool

	// stateMutex guards TracesPanel-specific state including
	// errorsOnly and seenSpanIDs.
	stateMutex sync.RWMutex
}

// spanRenderer renders trace spans and implements ItemRenderer[Span].
type spanRenderer struct {
	// panel is the parent panel that renders span lines.
	panel *TracesPanel
}

// TracesRefreshMessage signals that new trace data is ready to display.
type TracesRefreshMessage struct {
	// Err holds any error from the refresh; nil on success.
	Err error

	// Spans holds the trace spans to display in the panel.
	Spans []Span
}

// NewTracesPanel creates a new traces panel.
//
// Takes provider (TracesProvider) which supplies the trace data.
// Takes c (clock.Clock) which provides time functions. If nil, uses the real
// system clock.
//
// Returns *TracesPanel which is the configured panel ready for use.
func NewTracesPanel(provider TracesProvider, c clock.Clock) *TracesPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &TracesPanel{
		AssetViewer: nil,
		clock:       c,
		lastRefresh: time.Time{},
		provider:    provider,
		err:         nil,
		seenSpanIDs: make(map[string]struct{}),
		errorsOnly:  false,
		stateMutex:  sync.RWMutex{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[Span]{
		ID:           "traces",
		Title:        "Traces",
		Renderer:     &spanRenderer{panel: p},
		NavMode:      NavigationSimple,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "j/↓", Description: "Move down"},
			{Key: "k/↑", Description: "Move up"},
			{Key: "Enter", Description: "View trace details"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear search"},
			{Key: "e", Description: "Toggle errors only"},
			{Key: "r", Description: "Refresh"},
		},
	})

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which refreshes the panel content.
func (p *TracesPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles messages and updates the panel state.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after processing the message.
// Returns tea.Cmd which is a command to run, or nil if none needed.
func (p *TracesPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case TracesRefreshMessage:
		p.handleRefreshMessage(message)
		return p, nil
	case DataUpdatedMessage:
		command := p.refresh()
		return p, command
	}
	return p, nil
}

// View renders the panel.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *TracesPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderTracesViewHeader,
		RenderEmptyState:    p.renderTracesEmptyState,
		RenderItems:         p.renderTracesItems,
		TrimTrailingNewline: false,
	})
}

// handleRefreshMessage processes a traces refresh message.
//
// Takes message (TracesRefreshMessage) which contains the refresh
// result with spans or an error.
//
// Safe for concurrent use. Acquires the panel mutex before updating state.
func (p *TracesPanel) handleRefreshMessage(message TracesRefreshMessage) {
	mu := p.Mutex()
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	if message.Err != nil {
		p.err = message.Err
	} else {
		p.mergeSpansWithHistory(message.Spans)
		p.err = nil
	}
	p.lastRefresh = p.clock.Now()
}

// handleKey processes key events for the traces panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns Panel which is the panel after handling the key event.
// Returns tea.Cmd which is a command to run, or nil if none is needed.
//
// Safe for concurrent use. Uses a mutex when toggling the errors-only filter.
func (p *TracesPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	if message.String() != "esc" {
		result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
		if result.Handled {
			return p, result.Cmd
		}
	}

	switch message.String() {
	case "esc":
		if p.Search() != nil && p.Search().HasQuery() {
			p.Search().ClearQuery()
			p.updateFilter()
			return p, nil
		}
	case "e":
		p.stateMutex.Lock()
		p.errorsOnly = !p.errorsOnly
		p.stateMutex.Unlock()
		command := p.refresh()
		return p, command
	}

	return p, nil
}

// updateFilter updates the search filter for the current trace items.
func (p *TracesPanel) updateFilter() {
	if p.Search() == nil {
		return
	}
	items := p.Items()
	p.Search().UpdateFilter(len(items), func(index int, query string) bool {
		if index >= len(items) {
			return false
		}
		span := items[index]
		return strings.Contains(strings.ToLower(span.Name), query) ||
			strings.Contains(strings.ToLower(span.Service), query) ||
			strings.Contains(strings.ToLower(span.TraceID), query)
	})
}

// renderTracesViewHeader renders the search box, filter status, and error
// message for the traces view.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to the builder.
func (p *TracesPanel) renderTracesViewHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.Items()))
	}

	header := p.renderHeader()
	content.WriteString(header)
	content.WriteString(stringNewline)
	usedLines++

	if p.err != nil {
		RenderErrorState(content, p.err)
		usedLines++
	}

	return usedLines
}

// renderTracesEmptyState writes the empty state message to the output.
//
// Takes content (*strings.Builder) which receives the rendered output.
//
// Safe for concurrent use. Reads the errorsOnly state under mutex protection.
func (p *TracesPanel) renderTracesEmptyState(content *strings.Builder) {
	p.stateMutex.RLock()
	errorsOnly := p.errorsOnly
	p.stateMutex.RUnlock()

	message := "No traces available"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No traces match filter"
	} else if errorsOnly {
		message = "No error traces"
	}
	content.WriteString(RenderDimText(message))
}

// renderTracesItems renders the table header and span rows.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which contains indices of items to display.
// Takes headerLines (int) which specifies the number of header lines to use
// when working out the visible height.
func (p *TracesPanel) renderTracesItems(content *strings.Builder, displayItems []int, headerLines int) {
	tableHeader := p.renderTableHeader()
	content.WriteString(tableHeader)
	content.WriteString(stringNewline)

	visibleHeight := max(1, p.ContentHeight()-headerLines-1)
	startIndex := p.ScrollOffset()
	endIndex := min(startIndex+visibleHeight, len(displayItems))

	items := p.Items()
	for i := startIndex; i < endIndex; i++ {
		spanIndex := displayItems[i]
		if spanIndex >= len(items) {
			continue
		}
		span := items[spanIndex]
		line := p.renderSpanLine(span, i)
		content.WriteString(line)
		if i < endIndex-1 {
			content.WriteString(stringNewline)
		}
	}
}

// mergeSpansWithHistory merges new spans with the accumulated history.
//
// Takes newSpans ([]Span) which contains the spans to merge into history.
//
// Safe for concurrent use; acquires stateMutex internally.
func (p *TracesPanel) mergeSpansWithHistory(newSpans []Span) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	newUniqueSpans := make([]Span, 0, len(newSpans))
	for i := range newSpans {
		span := &newSpans[i]
		if _, seen := p.seenSpanIDs[span.SpanID]; !seen {
			p.seenSpanIDs[span.SpanID] = struct{}{}
			newUniqueSpans = append(newUniqueSpans, *span)
		}
	}

	if len(newUniqueSpans) == 0 {
		return
	}

	currentItems := p.items
	allSpans := make([]Span, 0, len(newUniqueSpans)+len(currentItems))
	allSpans = append(allSpans, newUniqueSpans...)
	allSpans = append(allSpans, currentItems...)

	if len(allSpans) > tracesMaxHistory {
		for i := tracesMaxHistory; i < len(allSpans); i++ {
			delete(p.seenSpanIDs, allSpans[i].SpanID)
		}
		allSpans = allSpans[:tracesMaxHistory]
	}

	p.setItemsUnlocked(allSpans)
}

// setItemsUnlocked sets the items without locking the mutex.
//
// The caller must hold the AssetViewer mutex.
//
// Takes items ([]Span) which provides the spans to display.
func (p *TracesPanel) setItemsUnlocked(items []Span) {
	p.items = items
	if p.Search() != nil {
		p.Search().UpdateFilter(len(items), func(index int, query string) bool {
			if index >= len(items) {
				return false
			}
			span := items[index]
			return strings.Contains(strings.ToLower(span.Name), query) ||
				strings.Contains(strings.ToLower(span.Service), query) ||
				strings.Contains(strings.ToLower(span.TraceID), query)
		})
	}
}

// refresh fetches new traces data from the provider.
//
// Returns tea.Cmd which spawns a goroutine to fetch traces and sends a
// TracesRefreshMessage when complete.
//
// Concurrent use is safe. The returned command runs in a separate goroutine
// and uses a read lock to access filter state safely.
func (p *TracesPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return TracesRefreshMessage{Err: errNoTracesProvider, Spans: nil}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("traces panel data fetch exceeded 5s timeout"))
		defer cancel()

		p.stateMutex.RLock()
		errorsOnly := p.errorsOnly
		p.stateMutex.RUnlock()

		var spans []Span
		var err error

		if errorsOnly {
			spans, err = p.provider.Errors(ctx, tracesFetchLimit)
		} else {
			spans, err = p.provider.Recent(ctx, tracesFetchLimit)
		}

		if err != nil {
			return TracesRefreshMessage{Err: err, Spans: nil}
		}

		return TracesRefreshMessage{Err: nil, Spans: spans}
	}
}

// renderHeader builds the panel header text with the current filter state.
//
// Returns string which shows the error filter state, span count, and time
// since the last refresh.
//
// Safe for concurrent use. Acquires a read lock to access the errorsOnly
// filter state.
func (p *TracesPanel) renderHeader() string {
	p.stateMutex.RLock()
	errorsOnly := p.errorsOnly
	p.stateMutex.RUnlock()

	var parts []string

	if errorsOnly {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(colourError).
			Render("[Errors Only]"))
	}

	parts = append(parts, RenderDimText(fmt.Sprintf("%d spans", len(p.Items()))))

	if !p.lastRefresh.IsZero() {
		age := p.clock.Now().Sub(p.lastRefresh).Round(time.Second)
		parts = append(parts, RenderDimText(fmt.Sprintf("(updated %s ago)", age)))
	}

	return strings.Join(parts, " ")
}

// renderTableHeader renders the column headers for the traces table.
//
// Returns string which is the styled header row.
func (p *TracesPanel) renderTableHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(colourForegroundDim).
		Bold(true)

	nameW, serviceW := p.calculateColumnWidths()

	header := fmt.Sprintf("%-3s %-*s %-*s %10s %8s",
		"St",
		nameW, "Name",
		serviceW, "Service",
		"Duration",
		"Time")

	return headerStyle.Render(header)
}

// calculateColumnWidths returns the best column widths based on terminal width.
//
// Returns nameW (int) which is the width for the name column.
// Returns serviceW (int) which is the width for the service column.
func (p *TracesPanel) calculateColumnWidths() (nameW, serviceW int) {
	fixedWidth := tracesCursorWidth + tracesStatusWidth + tracesDurationWidth + tracesTimeWidth + tracesSpacingWidth
	availableWidth := p.ContentWidth() - fixedWidth

	nameW = max(tracesMinNameWidth, int(float64(availableWidth)*tracesNameWidthRatio))
	serviceW = max(tracesMinServiceW, availableWidth-nameW)
	return nameW, serviceW
}

// renderSpanLine renders a single span as a formatted table row.
//
// Takes span (Span) which contains the trace span data to display.
// Takes index (int) which is the row index in the span list.
//
// Returns string which is the formatted line with cursor, status, name,
// service, duration, and time ago columns.
func (p *TracesPanel) renderSpanLine(span Span, index int) string {
	cursor := " "
	if index == p.Cursor() {
		cursor = cursorArrow
		if p.Focused() {
			cursor = lipgloss.NewStyle().Foreground(colourPrimary).Render(cursorArrow)
		}
	}

	var statusIcon string
	switch span.Status {
	case SpanStatusOK:
		statusIcon = lipgloss.NewStyle().Foreground(colourSuccess).Render(SymbolStatusFilled)
	case SpanStatusError:
		statusIcon = lipgloss.NewStyle().Foreground(colourError).Render(SymbolStatusFilled)
	default:
		statusIcon = lipgloss.NewStyle().Foreground(colourForegroundDim).Render(SymbolStatusEmpty)
	}

	nameW, serviceW := p.calculateColumnWidths()

	name := TruncateString(span.Name, nameW)
	name = PadRight(name, nameW)
	if index == p.Cursor() && p.Focused() {
		name = lipgloss.NewStyle().Bold(true).Render(name)
	}

	service := TruncateString(span.Service, serviceW)
	if service == "" {
		service = "-"
	}
	service = PadRight(service, serviceW)
	service = RenderDimText(service)

	duration := inspector.FormatDuration(span.Duration)
	durationStyle := lipgloss.NewStyle().Foreground(colourForeground)
	if span.Duration > tracesSlowThreshold {
		durationStyle = durationStyle.Foreground(colourWarning)
	}
	if span.Duration > 1*time.Second {
		durationStyle = durationStyle.Foreground(colourError)
	}
	duration = durationStyle.Render(fmt.Sprintf("%10s", duration))

	timeAgo := formatTimeAgo(span.StartTime, p.clock.Now())
	timeAgo = RenderDimText(fmt.Sprintf("%8s", timeAgo))

	return fmt.Sprintf("%s %s %s %s %s %s", cursor, statusIcon, name, service, duration, timeAgo)
}

// RenderRow renders a span row.
//
// Takes span (Span) which holds the trace span data to render.
// Takes lineIndex (int) which is the row index in the span list.
// Takes _ (bool) which is the unused selected state.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted span row for display.
func (r *spanRenderer) RenderRow(span Span, lineIndex int, _, _ bool, _ int) string {
	return r.panel.renderSpanLine(span, lineIndex)
}

// RenderExpanded returns nil as traces do not use expansion.
//
// Returns []string which is always nil for this renderer.
func (*spanRenderer) RenderExpanded(_ Span, _ int) []string {
	return nil
}

// GetID returns the span's unique identifier.
//
// Takes span (Span) which provides the span to extract the identifier from.
//
// Returns string which is the unique identifier for the given span.
func (*spanRenderer) GetID(span Span) string {
	return span.SpanID
}

// MatchesFilter reports whether the span matches the search query.
//
// Takes span (Span) which is the span to check against the query.
// Takes query (string) which is the lowercase search term to match.
//
// Returns bool which is true if the span's name, service, or trace ID
// contains the query.
func (*spanRenderer) MatchesFilter(span Span, query string) bool {
	return strings.Contains(strings.ToLower(span.Name), query) ||
		strings.Contains(strings.ToLower(span.Service), query) ||
		strings.Contains(strings.ToLower(span.TraceID), query)
}

// IsExpandable reports whether the span can be expanded.
//
// Returns bool which is always false for span renderers.
func (*spanRenderer) IsExpandable(_ Span) bool {
	return false
}

// ExpandedLineCount returns the number of lines when expanded.
//
// Returns int which is always zero as span renderers do not support expansion.
func (*spanRenderer) ExpandedLineCount(_ Span) int {
	return 0
}

// formatTimeAgo formats a time as a short relative string like "5s", "3m",
// "2h", or "1d".
//
// Takes t (time.Time) which is the time to format.
// Takes now (time.Time) which is the current time used as a reference point.
//
// Returns string which is the relative time in short format.
func formatTimeAgo(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < tracesHoursPerDay*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/tracesHoursPerDay))
	}
}
