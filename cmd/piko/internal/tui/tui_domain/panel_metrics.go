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
	"slices"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/wdk/clock"
)

const (
	// metricsCursorWidth is the width of the cursor column in characters.
	metricsCursorWidth = 2

	// metricsExpandWidth is the width of the expand or collapse indicator.
	metricsExpandWidth = 2

	// metricsSpacingWidth is the space between columns in the metrics panel.
	metricsSpacingWidth = 2

	// metricsValueColWidth is the fixed width for the metrics value column.
	metricsValueColWidth = 15

	// metricsMinNameWidth is the smallest width in characters for the name column.
	metricsMinNameWidth = 30

	// metricsMinSparkWidth is the minimum width for the sparkline column.
	metricsMinSparkWidth = 15

	// metricsNameWidthRatio is the portion of available width used for metric names.
	metricsNameWidthRatio = 0.6

	// valueThresholdGiga is the threshold for values shown with a G suffix.
	valueThresholdGiga = 1e9

	// valueThresholdMega is the threshold for mega (million) value formatting.
	valueThresholdMega = 1e6

	// valueThresholdKilo is the threshold (1,000) for formatting values with K suffix.
	valueThresholdKilo = 1e3

	// valueThresholdOne is the threshold for displaying values without scaling.
	valueThresholdOne = 1

	// valueThresholdMilli is the smallest value shown in milli-scale format.
	valueThresholdMilli = 0.001

	// metricsHistorySize is the number of samples to keep per metric.
	// At 2 second refresh intervals, 1800 samples = 1 hour of history.
	metricsHistorySize = 1800
)

var (
	_ Panel = (*MetricsPanel)(nil)

	_ ItemRenderer[metricDisplay] = (*metricsRenderer)(nil)
)

// MetricsPanel displays OpenTelemetry metrics with sparklines.
// It implements the Panel interface.
type MetricsPanel struct {
	*AssetViewer[metricDisplay]

	// clock provides time functions for use in tests.
	clock clock.Clock

	// lastRefresh is when the metrics were last updated.
	lastRefresh time.Time

	// provider supplies metrics data for display; nil means no data source is set.
	provider MetricsProvider

	// err holds the most recent refresh error; nil means success.
	err error

	// metricHistory stores past values for each metric, keyed by metric name.
	metricHistory map[string]*metricHistoryEntry

	// stateMutex protects panel state from concurrent access by multiple goroutines.
	stateMutex sync.RWMutex
}

// metricHistoryEntry stores the history of values for a single metric.
type metricHistoryEntry struct {
	// description is a brief explanation of what this metric measures.
	description string

	// unit is the measurement unit for the metric values.
	unit string

	// values holds past metric readings, limited to metricsHistorySize entries.
	values []float64
}

// metricDisplay holds a single metric and its display state for the TUI.
type metricDisplay struct {
	// name is the metric identifier used for display and history lookup.
	name string

	// description is a short explanation of what the metric measures.
	description string

	// unit specifies the measurement unit for formatting metric values.
	unit string

	// values holds the sample data used for the sparkline and statistics.
	values []float64

	// current is the most recent metric value.
	current float64
}

// metricsRenderer shows metric items in the metrics panel.
type metricsRenderer struct {
	// panel is the parent panel used to check expansion state and render rows.
	panel *MetricsPanel
}

// MetricsRefreshMessage signals that new metrics data is ready to display.
type MetricsRefreshMessage struct {
	// Err holds any error from the refresh; nil means success.
	Err error

	// Metrics holds the refreshed metric data to display.
	Metrics []metricDisplay
}

// NewMetricsPanel creates a new metrics panel.
//
// Takes provider (MetricsProvider) which supplies the metrics data.
// Takes c (clock.Clock) which provides time functions. If nil, uses the real
// clock.
//
// Returns *MetricsPanel which is the configured panel ready for use.
func NewMetricsPanel(provider MetricsProvider, c clock.Clock) *MetricsPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &MetricsPanel{
		AssetViewer:   nil,
		clock:         c,
		lastRefresh:   time.Time{},
		provider:      provider,
		err:           nil,
		metricHistory: make(map[string]*metricHistoryEntry),
		stateMutex:    sync.RWMutex{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[metricDisplay]{
		ID:           "metrics",
		Title:        "Metrics",
		Renderer:     &metricsRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "j/↓", Description: "Move down"},
			{Key: "k/↑", Description: "Move up"},
			{Key: "Enter", Description: "Toggle expand"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear search"},
			{Key: "r", Description: "Refresh"},
		},
	})

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which triggers a refresh of the metrics data.
func (p *MetricsPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles messages and updates the panel state.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after processing the message.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (p *MetricsPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case MetricsRefreshMessage:
		p.handleRefreshMessage(message)
		return p, nil
	case DataUpdatedMessage:
		command := p.refresh()
		return p, command
	}
	return p, nil
}

// View renders the panel with the given dimensions.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *MetricsPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderMetricsHeader,
		RenderEmptyState:    p.renderMetricsEmptyState,
		RenderItems:         p.renderMetricsItems,
		TrimTrailingNewline: false,
	})
}

// handleRefreshMessage processes a metrics refresh message.
//
// Takes message (MetricsRefreshMessage) which contains the refresh data or error.
//
// Safe for concurrent use. Locks both the panel mutex and state mutex.
func (p *MetricsPanel) handleRefreshMessage(message MetricsRefreshMessage) {
	mu := p.Mutex()
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	if message.Err != nil {
		p.err = message.Err
	} else {
		p.mergeMetricsWithHistory(message.Metrics)
		p.err = nil
	}
	p.lastRefresh = p.clock.Now()
}

// handleKey processes key events for the metrics panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns Panel which is the panel after handling the key event.
// Returns tea.Cmd which is nil as the metrics panel has no special keys.
func (p *MetricsPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
	if result.Handled {
		return p, result.Cmd
	}

	return p, nil
}

// renderMetricsHeader renders the search box, filter status, and error
// message.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to the content buffer.
//
// Safe for concurrent use. Acquires a read lock to access error state.
func (p *MetricsPanel) renderMetricsHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.Items()))
	}

	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()

	if err != nil {
		RenderErrorState(content, err)
		usedLines++
	}

	return usedLines
}

// renderMetricsEmptyState writes the empty state message to the builder.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *MetricsPanel) renderMetricsEmptyState(content *strings.Builder) {
	message := "No metrics available"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No metrics match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderMetricsItems writes the metrics list to the output.
//
// Takes content (*strings.Builder) which collects the rendered output.
// Takes displayItems ([]int) which lists the indices of metrics to show.
// Takes headerLines (int) which sets how many lines to keep for headers.
func (p *MetricsPanel) renderMetricsItems(content *strings.Builder, displayItems []int, headerLines int) {
	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()
	nameWidth, sparkWidth := p.calculateColumnWidths()

	for _, metricIndex := range displayItems {
		if metricIndex >= len(items) {
			continue
		}

		m := items[metricIndex]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()
		expanded := p.IsExpanded(m.name)

		ctx.WriteLineIfVisible(func() string {
			return p.renderMetricLine(m, selected, expanded, nameWidth, sparkWidth)
		})

		if expanded {
			p.renderExpandedDetails(ctx, m)
		}
	}
}

// renderMetricLine renders a single metric row for display.
//
// Takes m (metricDisplay) which holds the metric data to show.
// Takes selected (bool) which indicates if this row is the current selection.
// Takes expanded (bool) which indicates if this metric is expanded.
// Takes nameWidth (int) which sets the width for the name column.
// Takes sparkWidth (int) which sets the width for the sparkline chart.
//
// Returns string which is the formatted line ready for display.
func (p *MetricsPanel) renderMetricLine(m metricDisplay, selected, expanded bool, nameWidth, sparkWidth int) string {
	cursor := RenderCursor(selected, p.Focused())
	expandChar := RenderExpandIndicator(expanded)

	name := RenderName(m.name, nameWidth, selected, p.Focused())

	config := DefaultSparklineConfig()
	config.Width = sparkWidth
	config.ShowCurrent = false
	spark := Sparkline(m.values, &config)

	current := formatMetricValue(m.current, m.unit)
	current = lipgloss.NewStyle().Foreground(colourForeground).Render(current)

	return fmt.Sprintf("%s%s %s %s %s", cursor, expandChar, name, spark, current)
}

// renderExpandedDetails shows the full details for a metric when expanded.
//
// Takes ctx (*ScrollContext) which controls which lines are visible.
// Takes m (metricDisplay) which holds the metric data to display.
func (*MetricsPanel) renderExpandedDetails(ctx *ScrollContext, m metricDisplay) {
	indent := "     "

	if m.description != "" {
		ctx.WriteLineIfVisible(func() string {
			return indent + RenderItalicDimText(m.description)
		})
	}

	if len(m.values) > 0 {
		ctx.WriteLineIfVisible(func() string {
			minVal, maxVal, avg := calculateMetricStats(m.values)
			stats := fmt.Sprintf("min: %s  max: %s  avg: %s  samples: %d",
				formatMetricValue(minVal, m.unit),
				formatMetricValue(maxVal, m.unit),
				formatMetricValue(avg, m.unit),
				len(m.values))
			return indent + RenderDimText(stats)
		})
	}
}

// calculateColumnWidths works out the widths for the name and sparkline
// columns based on the space available.
//
// Returns nameWidth (int) which is the width for the name column.
// Returns sparkWidth (int) which is the width for the sparkline column.
func (p *MetricsPanel) calculateColumnWidths() (nameWidth, sparkWidth int) {
	contentWidth := p.ContentWidth()
	fixedWidth := metricsCursorWidth + metricsExpandWidth + metricsSpacingWidth + metricsValueColWidth
	availableWidth := contentWidth - fixedWidth

	nameWidth = max(metricsMinNameWidth, int(float64(availableWidth)*metricsNameWidthRatio))
	sparkWidth = max(metricsMinSparkWidth, availableWidth-nameWidth)
	return nameWidth, sparkWidth
}

// mergeMetricsWithHistory merges new metric data with stored history.
//
// Takes newMetrics ([]metricDisplay) which contains the latest metric values.
//
// Must be called with both mutexes held.
func (p *MetricsPanel) mergeMetricsWithHistory(newMetrics []metricDisplay) {
	expandedMap := p.ExpandedMap()

	for _, m := range newMetrics {
		p.updateMetricHistory(m)
	}

	displayMetrics := p.buildDisplayMetricsFromHistory(newMetrics)

	p.items = displayMetrics

	p.SetExpandedMap(expandedMap)
}

// updateMetricHistory records the latest value from a metric into its history.
//
// Takes m (metricDisplay) which contains the metric data to record.
func (p *MetricsPanel) updateMetricHistory(m metricDisplay) {
	entry, exists := p.metricHistory[m.name]
	if !exists {
		entry = &metricHistoryEntry{
			values:      make([]float64, 0, metricsHistorySize),
			description: m.description,
			unit:        m.unit,
		}
		p.metricHistory[m.name] = entry
	}

	if m.description != "" {
		entry.description = m.description
	}
	if m.unit != "" {
		entry.unit = m.unit
	}

	if len(m.values) > 0 {
		latestValue := m.values[len(m.values)-1]
		entry.values = append(entry.values, latestValue)

		if len(entry.values) > metricsHistorySize {
			entry.values = entry.values[len(entry.values)-metricsHistorySize:]
		}
	}
}

// buildDisplayMetricsFromHistory builds display metrics using accumulated
// history.
//
// Takes newMetrics ([]metricDisplay) which provides the current metric values.
//
// Returns []metricDisplay which contains metrics enriched with historical data.
func (p *MetricsPanel) buildDisplayMetricsFromHistory(newMetrics []metricDisplay) []metricDisplay {
	displayMetrics := make([]metricDisplay, 0, len(newMetrics))

	for _, m := range newMetrics {
		entry, exists := p.metricHistory[m.name]
		if !exists {
			displayMetrics = append(displayMetrics, m)
			continue
		}

		display := metricDisplay{
			name:        m.name,
			description: entry.description,
			unit:        entry.unit,
			values:      entry.values,
			current:     m.current,
		}
		displayMetrics = append(displayMetrics, display)
	}

	return displayMetrics
}

// refresh fetches new metrics data from the provider.
//
// Returns tea.Cmd which queries the provider and sends a MetricsRefreshMessage
// with the results.
func (p *MetricsPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return MetricsRefreshMessage{Err: errNoMetricsProvider, Metrics: nil}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("metrics panel data fetch exceeded 5s timeout"))
		defer cancel()

		names, err := p.provider.ListMetrics(ctx)
		if err != nil {
			return MetricsRefreshMessage{Err: err, Metrics: nil}
		}

		slices.Sort(names)

		metrics := make([]metricDisplay, 0, len(names))
		end := p.clock.Now()
		start := end.Add(-5 * time.Minute)

		for _, name := range names {
			series, err := p.provider.Query(ctx, name, start, end)
			if err != nil {
				continue
			}

			display := metricDisplay{
				name:        series.Name,
				description: series.Description,
				unit:        series.Unit,
				values:      make([]float64, 0, len(series.Values)),
				current:     0,
			}

			for _, v := range series.Values {
				display.values = append(display.values, v.Value)
			}

			if latest := series.Latest(); latest != nil {
				display.current = latest.Value
			}

			metrics = append(metrics, display)
		}

		return MetricsRefreshMessage{Err: nil, Metrics: metrics}
	}
}

// RenderRow renders a metric display row.
//
// Takes m (metricDisplay) which holds the metric data to render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted metric row for display.
func (r *metricsRenderer) RenderRow(m metricDisplay, _ int, selected, _ bool, _ int) string {
	expanded := r.panel.IsExpanded(m.name)
	nameWidth, sparkWidth := r.panel.calculateColumnWidths()
	return r.panel.renderMetricLine(m, selected, expanded, nameWidth, sparkWidth)
}

// RenderExpanded returns expanded content lines for a metric.
//
// Takes m (metricDisplay) which holds the metric data to expand.
// Takes _ (int) which is the unused content width.
//
// Returns []string which contains the description and statistics
// lines.
func (*metricsRenderer) RenderExpanded(m metricDisplay, _ int) []string {
	var lines []string
	indent := "     "

	if m.description != "" {
		lines = append(lines, indent+RenderItalicDimText(m.description))
	}

	if len(m.values) > 0 {
		minVal, maxVal, avg := calculateMetricStats(m.values)
		stats := fmt.Sprintf("min: %s  max: %s  avg: %s  samples: %d",
			formatMetricValue(minVal, m.unit),
			formatMetricValue(maxVal, m.unit),
			formatMetricValue(avg, m.unit),
			len(m.values))
		lines = append(lines, indent+RenderDimText(stats))
	}

	return lines
}

// GetID returns a unique identifier for the metric.
//
// Takes m (metricDisplay) which specifies the metric to identify.
//
// Returns string which is the metric's name used as its identifier.
func (*metricsRenderer) GetID(m metricDisplay) string {
	return m.name
}

// MatchesFilter returns true if the metric matches the search query.
//
// Takes m (metricDisplay) which is the metric to check.
// Takes query (string) which is the lowercase search term to match.
//
// Returns bool which is true when the query matches the metric name or
// description.
func (*metricsRenderer) MatchesFilter(m metricDisplay, query string) bool {
	return strings.Contains(strings.ToLower(m.name), query) ||
		strings.Contains(strings.ToLower(m.description), query)
}

// IsExpandable returns true if the metric can show expanded details.
//
// Takes m (metricDisplay) which specifies the metric to check.
//
// Returns bool which is true when the metric has a description or values.
func (*metricsRenderer) IsExpandable(m metricDisplay) bool {
	return m.description != "" || len(m.values) > 0
}

// ExpandedLineCount returns the number of detail lines when expanded.
//
// Takes m (metricDisplay) which specifies the metric to measure.
//
// Returns int which is the count of lines shown in expanded view.
func (*metricsRenderer) ExpandedLineCount(m metricDisplay) int {
	lines := 0
	if m.description != "" {
		lines++
	}
	if len(m.values) > 0 {
		lines++
	}
	return lines
}

// calculateMetricStats works out the smallest, largest, and average values
// from a slice of numbers.
//
// Takes values ([]float64) which contains the numbers to analyse.
//
// Returns minVal (float64) which is the smallest value in the slice.
// Returns maxVal (float64) which is the largest value in the slice.
// Returns avg (float64) which is the mean of all values.
func calculateMetricStats(values []float64) (minVal, maxVal, avg float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}
	minVal, maxVal = values[0], values[0]
	sum := 0.0
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
		sum += v
	}
	avg = sum / float64(len(values))
	return minVal, maxVal, avg
}

// formatMetricValue formats a number with a scale suffix and unit.
//
// Takes v (float64) which is the value to format.
// Takes unit (string) which is the unit to add after the value.
//
// Returns string which contains the formatted value with a scale suffix
// (G, M, K) when needed, followed by the unit if given.
func formatMetricValue(v float64, unit string) string {
	var formatted string
	absV := v
	if absV < 0 {
		absV = -absV
	}

	switch {
	case absV >= valueThresholdGiga:
		formatted = fmt.Sprintf("%.2fG", v/valueThresholdGiga)
	case absV >= valueThresholdMega:
		formatted = fmt.Sprintf("%.2fM", v/valueThresholdMega)
	case absV >= valueThresholdKilo:
		formatted = fmt.Sprintf("%.2fK", v/valueThresholdKilo)
	case absV >= valueThresholdOne:
		formatted = fmt.Sprintf("%.2f", v)
	case absV >= valueThresholdMilli:
		formatted = fmt.Sprintf("%.4f", v)
	case absV == 0:
		formatted = "0"
	default:
		formatted = fmt.Sprintf("%.2e", v)
	}

	if unit != "" {
		formatted += " " + unit
	}
	return formatted
}
