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
	"cmp"
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
	// routesFetchLimit is the maximum number of routes to fetch when refreshing.
	routesFetchLimit = 5000

	// routesStatsWidth is the fixed width in characters for route statistics.
	routesStatsWidth = 40

	// routesMinPathWidth is the smallest character width for the route path
	// column.
	routesMinPathWidth = 20

	// routesSpacingAdjust is the spacing offset used when calculating path width.
	routesSpacingAdjust = 6

	// routesErrorRateHigh is the error rate threshold above which a route is
	// considered unhealthy.
	routesErrorRateHigh = 10

	// routesErrorRateLow is the error rate threshold below which routes are healthy.
	routesErrorRateLow = 1

	// routesRecentSpansLimit is the maximum number of recent spans to store per route.
	routesRecentSpansLimit = 10

	// routesRecentDisplayLimit is the maximum number of recent requests to display.
	routesRecentDisplayLimit = 5

	// routesPercentileP50 is the percentile value for calculating the median
	// response time.
	routesPercentileP50 = 50

	// routesPercentileP90 is the 90th percentile threshold for route statistics.
	routesPercentileP90 = 90

	// routesPercentileP95 is the 95th percentile for route latency.
	routesPercentileP95 = 95

	// routesPercentileP99 is the 99th percentile threshold for latency calculations.
	routesPercentileP99 = 99

	// routesPercentileMax is the highest allowed percentile value.
	routesPercentileMax = 100

	// routesPercentUnit is the multiplier to convert a ratio to a percentage.
	routesPercentUnit = 100.0
)

var (
	_ Panel = (*RoutesPanel)(nil)

	// _ verifies that routesRenderer implements ItemRenderer[RouteStats].
	_ ItemRenderer[RouteStats] = (*routesRenderer)(nil)
)

// RouteStats holds statistics for a single HTTP route.
type RouteStats struct {
	// Method is the HTTP method for this route; "MIXED" if multiple methods are used.
	Method string

	// Path is the route pattern used to match incoming requests.
	Path string

	// Durations holds request durations in milliseconds for calculating
	// statistics.
	Durations []float64

	// RecentSpans holds the most recent request spans for this route.
	RecentSpans []Span

	// P50Ms is the 50th percentile (median) latency in milliseconds.
	P50Ms float64

	// AverageMs is the average response time in milliseconds.
	AverageMs float64

	// ErrorCount is the number of requests that returned an error.
	ErrorCount int

	// P90Ms is the 90th percentile latency in milliseconds.
	P90Ms float64

	// P95Ms is the 95th percentile latency in milliseconds.
	P95Ms float64

	// P99Ms is the 99th percentile request latency in milliseconds.
	P99Ms float64

	// MinMs is the shortest response time in milliseconds.
	MinMs float64

	// MaxMs is the longest request duration in milliseconds.
	MaxMs float64

	// Count is the total number of requests for this route.
	Count int
}

// RoutesPanel displays route-level metrics from traces and implements Panel.
type RoutesPanel struct {
	*AssetViewer[RouteStats]

	// clock provides the current time for tracking when the panel was last refreshed.
	clock clock.Clock

	// lastRefresh is the time when the routes were last refreshed.
	lastRefresh time.Time

	// provider supplies trace data for the routes panel; nil means no data source.
	provider TracesProvider

	// err holds the last error from refreshing routes data; nil means no error.
	err error

	// sortBy is the current sort column: "count", "avg", "errors", or "path".
	sortBy string

	// sortDesc indicates whether to sort in descending order.
	sortDesc bool

	// totalCount is the total number of routes in the list.
	totalCount int

	// totalErrors is the count of routes that failed to load.
	totalErrors int

	// stateMutex guards panel state during reads and writes from multiple goroutines.
	stateMutex sync.RWMutex
}

// routesRenderer implements ItemRenderer for RouteStats.
type routesRenderer struct {
	// panel is the parent routes panel used for expansion state and rendering.
	panel *RoutesPanel
}

// RoutesRefreshMessage signals that new route data is ready to display.
type RoutesRefreshMessage struct {
	// Err holds any error from the refresh; nil means success.
	Err error

	// Routes holds the current route statistics for display.
	Routes []RouteStats

	// TotalCount is the total number of routes.
	TotalCount int

	// TotalErrors is the count of routes that failed to load.
	TotalErrors int
}

// NewRoutesPanel creates a new routes panel.
//
// Takes provider (TracesProvider) which supplies trace data for route display.
// Takes c (clock.Clock) which provides time functions; if nil, uses the real
// clock.
//
// Returns *RoutesPanel which is the configured panel ready for use.
func NewRoutesPanel(provider TracesProvider, c clock.Clock) *RoutesPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &RoutesPanel{
		AssetViewer: nil,
		clock:       c,
		lastRefresh: time.Time{},
		provider:    provider,
		err:         nil,
		sortBy:      "count",
		sortDesc:    true,
		totalCount:  0,
		totalErrors: 0,
		stateMutex:  sync.RWMutex{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[RouteStats]{
		ID:           "routes",
		Title:        "Routes",
		Renderer:     &routesRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Navigate"},
			{Key: "Space", Description: "Expand details"},
			{Key: "/", Description: "Search"},
			{Key: "s", Description: "Cycle sort"},
			{Key: "Esc", Description: "Clear/Collapse"},
			{Key: "g/G", Description: "Top/Bottom"},
		},
	})

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which triggers the initial data refresh.
func (p *RoutesPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles messages for the routes panel.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after processing the message.
// Returns tea.Cmd which is the command to run, or nil if there is none.
func (p *RoutesPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case RoutesRefreshMessage:
		p.handleRefreshMessage(message)
		return p, nil
	case DataUpdatedMessage, TickMessage:
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
func (p *RoutesPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderRoutesHeader,
		RenderEmptyState:    p.renderRoutesEmptyState,
		RenderItems:         p.renderRoutesItems,
		TrimTrailingNewline: false,
	})
}

// handleRefreshMessage processes a routes refresh message.
//
// Takes message (RoutesRefreshMessage) which contains the refresh data or error.
//
// Safe for concurrent use. Acquires both the panel mutex and state mutex.
func (p *RoutesPanel) handleRefreshMessage(message RoutesRefreshMessage) {
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
		p.totalCount = message.TotalCount
		p.totalErrors = message.TotalErrors
		p.err = nil

		routes := message.Routes
		p.sortRoutesSlice(routes)

		p.items = routes
	}
	p.lastRefresh = p.clock.Now()
}

// handleKey processes key events for the routes panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns Panel which is the panel after handling the key event.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *RoutesPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
	if result.Handled {
		return p, result.Cmd
	}

	if message.String() == "s" {
		p.cycleSortAndReSort()
		return p, nil
	}

	return p, nil
}

// cycleSortAndReSort changes to the next sort field and re-sorts the items.
//
// Safe for concurrent use. Acquires both the panel mutex and the state mutex.
func (p *RoutesPanel) cycleSortAndReSort() {
	mu := p.Mutex()
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	p.cycleSortBy()
	routes := p.items
	p.sortRoutesSlice(routes)
}

// cycleSortBy moves to the next sort option in the list.
func (p *RoutesPanel) cycleSortBy() {
	switch p.sortBy {
	case "count":
		p.sortBy = "avg"
	case "avg":
		p.sortBy = "errors"
	case "errors":
		p.sortBy = "path"
		p.sortDesc = false
	case "path":
		p.sortBy = "count"
		p.sortDesc = true
	}
}

// sortRoutesSlice sorts the given routes slice using the current sort settings.
//
// Takes routes ([]RouteStats) which is the slice to sort in place.
func (p *RoutesPanel) sortRoutesSlice(routes []RouteStats) {
	slices.SortFunc(routes, func(a, b RouteStats) int {
		var c int
		switch p.sortBy {
		case "avg":
			c = cmp.Compare(a.AverageMs, b.AverageMs)
		case "errors":
			c = cmp.Compare(a.ErrorCount, b.ErrorCount)
		case "path":
			c = cmp.Compare(a.Path, b.Path)
		default:
			c = cmp.Compare(a.Count, b.Count)
		}

		if c == 0 && p.sortBy != "path" {
			c = cmp.Compare(a.Path, b.Path)
		}

		if p.sortDesc {
			return -c
		}
		return c
	})
}

// renderRoutesHeader renders the search box, totals header, filter status,
// and error message.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to content.
//
// Safe for concurrent use. Acquires a read lock to access panel state.
func (p *RoutesPanel) renderRoutesHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.Items()))
	}

	p.stateMutex.RLock()
	totalCount := p.totalCount
	totalErrors := p.totalErrors
	sortBy := p.sortBy
	err := p.err
	p.stateMutex.RUnlock()

	errorRate := 0.0
	if totalCount > 0 {
		errorRate = float64(totalErrors) / float64(totalCount) * routesPercentUnit
	}
	header := RenderDimText(fmt.Sprintf("Total: %d requests  |  Errors: %d (%.1f%%)  |  Sort: %s",
		totalCount, totalErrors, errorRate, sortBy))
	content.WriteString(header)
	content.WriteString(stringNewline)
	usedLines++

	if err != nil {
		RenderErrorState(content, err)
		usedLines++
	}

	return usedLines
}

// renderRoutesEmptyState writes the empty state message to the output.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *RoutesPanel) renderRoutesEmptyState(content *strings.Builder) {
	message := "No routes available"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No routes match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderRoutesItems renders the routes list.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies which items to display.
// Takes headerLines (int) which indicates how many lines the header uses.
func (p *RoutesPanel) renderRoutesItems(content *strings.Builder, displayItems []int, headerLines int) {
	RenderExpandableItems(RenderExpandableItemsConfig[RouteStats]{
		Ctx:          NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines-1),
		Items:        p.Items(),
		DisplayItems: displayItems,
		Cursor:       p.Cursor(),
		GetID:        func(r RouteStats) string { return r.Path },
		IsExpanded:   p.IsExpanded,
		RenderRow:    p.renderRouteLine,
		RenderExpand: p.renderExpandedRouteDetails,
	})
}

// renderRouteLine renders a single route row for display.
//
// Takes route (RouteStats) which provides the route data to show.
// Takes selected (bool) which shows if this row is selected.
// Takes expanded (bool) which shows if the route details are visible.
//
// Returns string which is the formatted line ready for display.
func (p *RoutesPanel) renderRouteLine(route RouteStats, selected, expanded bool) string {
	cursor := RenderCursor(selected, p.Focused())
	expandChar := RenderExpandIndicator(expanded)

	var indicator string
	if route.ErrorCount > 0 {
		errorRate := float64(route.ErrorCount) / float64(route.Count) * routesPercentUnit
		if errorRate > routesErrorRateHigh {
			indicator = statusUnhealthyStyle.Render(statusIndicatorDot)
		} else if errorRate > routesErrorRateLow {
			indicator = statusDegradedStyle.Render(statusIndicatorDot)
		} else {
			indicator = statusHealthyStyle.Render(statusIndicatorDot)
		}
	} else {
		indicator = statusHealthyStyle.Render(statusIndicatorDot)
	}

	pathWidth := max(routesMinPathWidth, p.ContentWidth()-routesStatsWidth-routesSpacingAdjust)

	path := RenderName(route.Path, pathWidth, selected, p.Focused())

	errorPct := 0.0
	if route.Count > 0 {
		errorPct = float64(route.ErrorCount) / float64(route.Count) * routesPercentUnit
	}
	stats := RenderDimText(fmt.Sprintf("%5d  %6.0fms  %6.0fms  %4.1f%%",
		route.Count, route.AverageMs, route.P50Ms, errorPct))

	return fmt.Sprintf("%s%s %s %s %s", cursor, indicator, expandChar, path, stats)
}

// renderExpandedRouteDetails renders the full details for a route when it is
// expanded.
//
// Takes ctx (*ScrollContext) which manages which lines are shown on screen.
// Takes route (RouteStats) which holds the route data to display.
func (p *RoutesPanel) renderExpandedRouteDetails(ctx *ScrollContext, route RouteStats) {
	indent := "      "
	dimStyle := lipgloss.NewStyle().Foreground(colorForegroundDim)

	ctx.WriteLineIfVisible(func() string {
		text := fmt.Sprintf("Latency: p50=%.0fms  p90=%.0fms  p95=%.0fms  p99=%.0fms",
			route.P50Ms, route.P90Ms, route.P95Ms, route.P99Ms)
		return indent + dimStyle.Render(text)
	})

	ctx.WriteLineIfVisible(func() string {
		text := fmt.Sprintf("Range:   min=%.0fms  max=%.0fms  avg=%.0fms",
			route.MinMs, route.MaxMs, route.AverageMs)
		return indent + dimStyle.Render(text)
	})

	if route.ErrorCount > 0 {
		ctx.WriteLineIfVisible(func() string {
			errorPct := float64(route.ErrorCount) / float64(route.Count) * routesPercentUnit
			text := fmt.Sprintf("Errors:  %d (%.1f%%)", route.ErrorCount, errorPct)
			return indent + statusUnhealthyStyle.Render(text)
		})
	}

	if len(route.RecentSpans) > 0 {
		ctx.WriteLineIfVisible(func() string {
			return indent + dimStyle.Render("Recent requests:")
		})

		for i := range route.RecentSpans {
			if i >= routesRecentDisplayLimit {
				break
			}
			span := route.RecentSpans[i]
			ctx.WriteLineIfVisible(func() string {
				return p.renderSpanLine(span, indent, &dimStyle)
			})
		}
	}
}

// renderSpanLine renders a single span line for the recent requests list.
//
// Takes span (Span) which provides the span data to show.
// Takes indent (string) which sets the indent prefix.
// Takes dimStyle (*lipgloss.Style) which styles dimmed parts.
//
// Returns string which is the formatted span line ready to show.
func (*RoutesPanel) renderSpanLine(span Span, indent string, dimStyle *lipgloss.Style) string {
	statusStyle := statusHealthyStyle
	statusText := "OK"
	if span.IsError() {
		statusStyle = statusUnhealthyStyle
		statusText = "ERR"
	}

	timeString := span.StartTime.Format("15:04:05")
	method := span.Attributes["method"]
	if method == "" {
		method = "???"
	}
	durationMs := span.Duration.Milliseconds()

	return indent + fmt.Sprintf("  %s  %s  %4dms  %s",
		dimStyle.Render(timeString),
		dimStyle.Render(method),
		durationMs,
		statusStyle.Render(statusText))
}

// refresh fetches new route data from traces.
//
// Returns tea.Cmd which gets and groups route statistics from the traces
// provider.
func (p *RoutesPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return RoutesRefreshMessage{Err: errors.New("no traces provider"), Routes: nil, TotalCount: 0, TotalErrors: 0}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("routes panel data fetch exceeded 5s timeout"))
		defer cancel()

		spans, err := p.provider.Recent(ctx, routesFetchLimit)
		if err != nil {
			return RoutesRefreshMessage{Err: err, Routes: nil, TotalCount: 0, TotalErrors: 0}
		}

		routeMap, totalCount, totalErrors := aggregateSpansToRoutes(spans)
		routes := calculateRouteStats(routeMap)

		return RoutesRefreshMessage{
			Err:         nil,
			Routes:      routes,
			TotalCount:  totalCount,
			TotalErrors: totalErrors,
		}
	}
}

// RenderRow renders a route statistics row.
//
// Takes route (RouteStats) which holds the route data to render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted route row for display.
func (r *routesRenderer) RenderRow(route RouteStats, _ int, selected, _ bool, _ int) string {
	expanded := r.panel.IsExpanded(route.Path)
	return r.panel.renderRouteLine(route, selected, expanded)
}

// RenderExpanded returns latency and error lines for an expanded route.
//
// Takes route (RouteStats) which holds the route data to expand.
// Takes _ (int) which is the unused content width.
//
// Returns []string which contains the latency percentiles, range,
// error count, and recent request lines.
func (r *routesRenderer) RenderExpanded(route RouteStats, _ int) []string {
	indent := "      "
	dimStyle := lipgloss.NewStyle().Foreground(colorForegroundDim)
	var lines []string

	text := fmt.Sprintf("Latency: p50=%.0fms  p90=%.0fms  p95=%.0fms  p99=%.0fms",
		route.P50Ms, route.P90Ms, route.P95Ms, route.P99Ms)
	lines = append(lines, indent+dimStyle.Render(text))

	text = fmt.Sprintf("Range:   min=%.0fms  max=%.0fms  avg=%.0fms",
		route.MinMs, route.MaxMs, route.AverageMs)
	lines = append(lines, indent+dimStyle.Render(text))

	if route.ErrorCount > 0 {
		errorPct := float64(route.ErrorCount) / float64(route.Count) * routesPercentUnit
		text = fmt.Sprintf("Errors:  %d (%.1f%%)", route.ErrorCount, errorPct)
		lines = append(lines, indent+statusUnhealthyStyle.Render(text))
	}

	if len(route.RecentSpans) > 0 {
		lines = append(lines, indent+dimStyle.Render("Recent requests:"))
		for i := range route.RecentSpans {
			if i >= routesRecentDisplayLimit {
				break
			}
			lines = append(lines, r.panel.renderSpanLine(route.RecentSpans[i], indent, &dimStyle))
		}
	}

	return lines
}

// GetID returns the route's unique identifier.
//
// Takes route (RouteStats) which provides the route statistics to identify.
//
// Returns string which is the route's path used as its unique identifier.
func (*routesRenderer) GetID(route RouteStats) string {
	return route.Path
}

// MatchesFilter returns true if the route matches the search query.
//
// Takes route (RouteStats) which is the route to check.
// Takes query (string) which is the search term to match against the path.
//
// Returns bool which is true if the route path contains the query.
func (*routesRenderer) MatchesFilter(route RouteStats, query string) bool {
	return strings.Contains(strings.ToLower(route.Path), query)
}

// IsExpandable returns whether the route has extra details to show.
//
// Returns bool which is always true for route statistics.
func (*routesRenderer) IsExpandable(_ RouteStats) bool {
	return true
}

// ExpandedLineCount returns the number of lines shown when a route is expanded.
//
// Takes route (RouteStats) which provides the route data to count lines for.
//
// Returns int which is the total line count for the expanded view.
func (*routesRenderer) ExpandedLineCount(route RouteStats) int {
	lines := 2

	if route.ErrorCount > 0 {
		lines++
	}

	if len(route.RecentSpans) > 0 {
		lines++
		lines += min(routesRecentDisplayLimit, len(route.RecentSpans))
	}

	return lines
}

// aggregateSpansToRoutes groups spans by their path and gathers statistics.
//
// Takes spans ([]Span) which contains the spans to group.
//
// Returns routeMap (map[string]*RouteStats) which maps paths to their stats.
// Returns totalCount (int) which is the number of spans with valid paths.
// Returns totalErrors (int) which is the number of spans marked as errors.
func aggregateSpansToRoutes(spans []Span) (routeMap map[string]*RouteStats, totalCount, totalErrors int) {
	routeMap = make(map[string]*RouteStats, len(spans))

	for i := range spans {
		path := spans[i].Attributes["path"]
		if path == "" {
			continue
		}

		totalCount++
		if spans[i].IsError() {
			totalErrors++
		}

		route := getOrCreateRoute(routeMap, path, spans[i].Attributes["method"])
		updateRouteWithSpan(route, spans[i])
	}

	return routeMap, totalCount, totalErrors
}

// getOrCreateRoute gets a route from the map or creates a new one if missing.
//
// Takes routeMap (map[string]*RouteStats) which holds the current routes.
// Takes path (string) which identifies the route.
// Takes method (string) which sets the HTTP method for new routes.
//
// Returns *RouteStats which is the existing or newly created route entry.
func getOrCreateRoute(routeMap map[string]*RouteStats, path, method string) *RouteStats {
	route, exists := routeMap[path]
	if !exists {
		route = &RouteStats{
			Method:      method,
			Path:        path,
			Durations:   make([]float64, 0),
			RecentSpans: make([]Span, 0, routesRecentSpansLimit),
			P50Ms:       0,
			AverageMs:   0,
			ErrorCount:  0,
			P90Ms:       0,
			P95Ms:       0,
			P99Ms:       0,
			MinMs:       0,
			MaxMs:       0,
			Count:       0,
		}
		routeMap[path] = route
	}
	return route
}

// updateRouteWithSpan updates route statistics with data from a span.
//
// Takes route (*RouteStats) which is the route statistics to update.
// Takes span (Span) which provides the span data to add.
func updateRouteWithSpan(route *RouteStats, span Span) {
	route.Count++
	if span.IsError() {
		route.ErrorCount++
	}

	method := span.Attributes["method"]
	if route.Method != "" && route.Method != method {
		route.Method = "MIXED"
	}

	route.Durations = append(route.Durations, float64(span.Duration.Milliseconds()))

	if len(route.RecentSpans) < routesRecentSpansLimit {
		route.RecentSpans = append(route.RecentSpans, span)
	}
}

// calculateRouteStats computes statistics for each route and returns them.
//
// Takes routeMap (map[string]*RouteStats) which contains the routes to
// process.
//
// Returns []RouteStats which contains the computed stats with average,
// minimum, maximum, and percentile values (p50, p90, p95, p99).
func calculateRouteStats(routeMap map[string]*RouteStats) []RouteStats {
	routes := make([]RouteStats, 0, len(routeMap))

	for _, route := range routeMap {
		if len(route.Durations) > 0 {
			slices.Sort(route.Durations)
			route.AverageMs = average(route.Durations)
			route.MinMs = route.Durations[0]
			route.MaxMs = route.Durations[len(route.Durations)-1]
			route.P50Ms = percentile(route.Durations, routesPercentileP50)
			route.P90Ms = percentile(route.Durations, routesPercentileP90)
			route.P95Ms = percentile(route.Durations, routesPercentileP95)
			route.P99Ms = percentile(route.Durations, routesPercentileP99)
		}
		routes = append(routes, *route)
	}

	return routes
}

// average calculates the arithmetic mean of a slice of numbers.
//
// Takes values ([]float64) which contains the numbers to average.
//
// Returns float64 which is the mean value, or zero if the slice is empty.
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// percentile calculates the p-th percentile from a sorted slice of values.
//
// Takes sorted ([]float64) which is the sorted slice of values.
// Takes p (int) which is the percentile to calculate, from 0 to 100.
//
// Returns float64 which is the calculated percentile value.
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= routesPercentileMax {
		return sorted[len(sorted)-1]
	}
	index := float64(len(sorted)-1) * float64(p) / routesPercentUnit
	lower := int(index)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
