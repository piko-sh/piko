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
	// healthStateCount is the number of displayable health states (healthy,
	// degraded, unhealthy).
	healthStateCount = 3

	// probeKeyLiveness is the key used to identify the liveness probe.
	probeKeyLiveness = "liveness"

	// probeKeyReadiness identifies the readiness probe in the health panel.
	probeKeyReadiness = "readiness"

	// healthHistorySize is the number of historical check results to retain.
	healthHistorySize = 1800

	// healthSparklineWidth is the width in columns for sparklines in the health
	// panel.
	healthSparklineWidth = 30

	// healthWeightHealthy is the weight for healthy state in average health
	// calculations.
	healthWeightHealthy = 1.0

	// healthWeightDegraded is the weight value for a degraded health state.
	healthWeightDegraded = 0.5

	// healthWeightOther is the weight value for unknown health states.
	healthWeightOther = 0.0
)

var (
	_ Panel = (*HealthPanel)(nil)

	_ ItemRenderer[healthDisplayItem] = (*healthRenderer)(nil)
)

// healthDisplayItem represents a single row in the health panel list.
// It can be either a probe header or a dependency entry under a probe.
type healthDisplayItem struct {
	// probeStatus holds the health check result for this probe.
	probeStatus *HealthStatus

	// dependency holds the health status when this item represents a dependency.
	dependency *HealthStatus

	// probeKey identifies the health probe and tracks expansion state.
	probeKey string

	// dependencyIndex is the position of the dependency within the probe for
	// unique identification.
	dependencyIndex int

	// isProbeRow indicates whether this item is a probe header row.
	isProbeRow bool
}

// HealthPanel displays the status of liveness and readiness probes.
// It implements the Panel interface.
type HealthPanel struct {
	*AssetViewer[healthDisplayItem]

	// clock provides the current time for tracking when the last refresh happened.
	clock clock.Clock

	// lastRefresh is the time when the health data was last updated.
	lastRefresh time.Time

	// provider supplies liveness and readiness health checks.
	provider HealthProvider

	// err holds the last error from a health check refresh; nil means success.
	err error

	// liveness holds the current liveness probe status; nil until the first
	// refresh.
	liveness *HealthStatus

	// readiness holds the current readiness probe status; nil until the first
	// refresh.
	readiness *HealthStatus

	// livenessHistory stores recent liveness probe results for the sparkline
	// display.
	livenessHistory *HistoryRing

	// readinessHistory holds recent readiness probe results for display.
	readinessHistory *HistoryRing

	// stateMutex guards the panel state for safe access from multiple goroutines.
	stateMutex sync.RWMutex
}

// healthRenderer implements ItemRenderer for health display items.
type healthRenderer struct {
	// panel is the parent panel used for rendering and expansion state checks.
	panel *HealthPanel
}

// HealthRefreshMessage signals that new health data is ready for the UI.
type HealthRefreshMessage struct {
	// Liveness holds the liveness probe result; nil if not yet checked.
	Liveness *HealthStatus

	// Readiness holds the current readiness probe status.
	Readiness *HealthStatus

	// Err holds any error that occurred during the health refresh.
	Err error
}

// NewHealthPanel creates a new health panel.
//
// Takes provider (HealthProvider) which supplies health check data.
// Takes c (clock.Clock) which provides time functions; if nil, uses the real
// clock.
//
// Returns *HealthPanel which is the configured panel ready for use.
func NewHealthPanel(provider HealthProvider, c clock.Clock) *HealthPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &HealthPanel{
		AssetViewer:      nil,
		clock:            c,
		lastRefresh:      time.Time{},
		provider:         provider,
		err:              nil,
		liveness:         nil,
		readiness:        nil,
		livenessHistory:  NewHistoryRing(healthHistorySize),
		readinessHistory: NewHistoryRing(healthHistorySize),
		stateMutex:       sync.RWMutex{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[healthDisplayItem]{
		ID:           "health",
		Title:        "Health",
		Renderer:     &healthRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: false,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Navigate"},
			{Key: "Space", Description: "Expand/Collapse"},
			{Key: "g/G", Description: "Go to top/bottom"},
			{Key: "r", Description: "Refresh"},
		},
	})

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which triggers the initial data refresh.
func (p *HealthPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles a message and returns the updated panel state.
//
// Takes message (tea.Msg) which is the message to handle.
//
// Returns Panel which is the updated panel after handling the message.
// Returns tea.Cmd which is a command to run, or nil if none is needed.
func (p *HealthPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case HealthRefreshMessage:
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
func (p *HealthPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderHealthHeader,
		RenderEmptyState:    p.renderHealthEmptyState,
		RenderItems:         p.renderHealthItems,
		TrimTrailingNewline: true,
	})
}

// handleRefreshMessage processes a health refresh message and updates the panel
// state.
//
// Takes message (HealthRefreshMessage) which contains the health check results.
//
// Safe for concurrent use. Locks both the panel mutex and state mutex.
func (p *HealthPanel) handleRefreshMessage(message HealthRefreshMessage) {
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
		p.liveness = message.Liveness
		p.readiness = message.Readiness
		p.err = nil
		p.updateHealthHistory()
		p.rebuildDisplayItems()
	}
	p.lastRefresh = p.clock.Now()
}

// rebuildDisplayItems builds a flat list of display items from the current
// panel state. Must be called while holding both mutexes.
func (p *HealthPanel) rebuildDisplayItems() {
	items := make([]healthDisplayItem, 0)

	if p.liveness != nil {
		items = append(items, healthDisplayItem{
			probeKey:        probeKeyLiveness,
			probeStatus:     p.liveness,
			dependency:      nil,
			dependencyIndex: -1,
			isProbeRow:      true,
		})

		if p.IsExpanded(probeKeyLiveness) {
			sortedDeps := sortHealthDependencies(p.liveness.Dependencies)
			for i, dependency := range sortedDeps {
				items = append(items, healthDisplayItem{
					probeKey:        probeKeyLiveness,
					probeStatus:     p.liveness,
					dependency:      dependency,
					dependencyIndex: i,
					isProbeRow:      false,
				})
			}
		}
	}

	if p.readiness != nil {
		items = append(items, healthDisplayItem{
			probeKey:        probeKeyReadiness,
			probeStatus:     p.readiness,
			dependency:      nil,
			dependencyIndex: -1,
			isProbeRow:      true,
		})

		if p.IsExpanded(probeKeyReadiness) {
			sortedDeps := sortHealthDependencies(p.readiness.Dependencies)
			for i, dependency := range sortedDeps {
				items = append(items, healthDisplayItem{
					probeKey:        probeKeyReadiness,
					probeStatus:     p.readiness,
					dependency:      dependency,
					dependencyIndex: i,
					isProbeRow:      false,
				})
			}
		}
	}

	p.items = items
}

// handleKey processes key events for the health panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns Panel which is the panel after handling the key event.
// Returns tea.Cmd which is the command to run, or nil if none is needed.
func (p *HealthPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	switch message.String() {
	case "enter", "space":
		p.toggleHealthExpansion()
		return p, nil
	}

	result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
	if result.Handled {
		return p, result.Cmd
	}

	return p, nil
}

// toggleHealthExpansion handles the expansion toggle and rebuilds the item
// list.
//
// Safe for concurrent use. Acquires both the parent mutex and stateMutex before
// changing the expansion state.
func (p *HealthPanel) toggleHealthExpansion() {
	mu := p.Mutex()
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}

	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	item := p.getItemAtCursorUnlocked()
	if item == nil {
		return
	}

	if !item.isProbeRow {
		return
	}

	currentExpanded := p.IsExpanded(item.probeKey)
	p.SetExpanded(item.probeKey, !currentExpanded)

	p.rebuildDisplayItems()
}

// getItemAtCursorUnlocked returns the item at the current cursor position.
// Must be called with the AssetViewer mutex held.
//
// Returns *healthDisplayItem which is the item at the cursor, or nil if the
// cursor is out of bounds.
func (p *HealthPanel) getItemAtCursorUnlocked() *healthDisplayItem {
	items := p.items
	cursor := p.Cursor()
	if cursor < 0 || cursor >= len(items) {
		return nil
	}
	return &items[cursor]
}

// renderHealthHeader renders the header section of the health panel.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to the builder.
//
// Safe for concurrent use. Acquires a read lock on stateMutex to access shared
// state.
func (p *HealthPanel) renderHealthHeader(content *strings.Builder) int {
	usedLines := 0

	p.stateMutex.RLock()
	lastRefresh := p.lastRefresh
	err := p.err
	p.stateMutex.RUnlock()

	if !lastRefresh.IsZero() {
		header := lipgloss.NewStyle().
			Foreground(colourForegroundDim).
			Render(fmt.Sprintf("Last check: %s", lastRefresh.Format("15:04:05")))
		content.WriteString(header)
		content.WriteString("\n\n")
		usedLines += 2
	}

	if err != nil {
		RenderErrorState(content, err)
		usedLines++
	}

	return usedLines
}

// renderHealthEmptyState renders the empty state message when no health data
// is available yet.
//
// Takes content (*strings.Builder) which receives the rendered output.
//
// Safe for concurrent use. Acquires a read lock on stateMutex to access the
// error state.
func (p *HealthPanel) renderHealthEmptyState(content *strings.Builder) {
	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()

	if err == nil {
		content.WriteString(RenderDimText("Waiting for health data..."))
	}
}

// renderHealthItems renders all health items with their expanded content.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies which items to render by index.
// Takes headerLines (int) which specifies how many lines the header uses.
func (p *HealthPanel) renderHealthItems(content *strings.Builder, displayItems []int, headerLines int) {
	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()

	for _, itemIndex := range displayItems {
		if itemIndex >= len(items) {
			continue
		}

		item := items[itemIndex]
		selected := ctx.LineIndex() == p.Cursor()

		if item.isProbeRow {
			p.renderProbeItem(ctx, item, selected)
		} else {
			p.renderDependencyItem(ctx, item, selected)
		}
	}
}

// renderProbeItem renders a probe header and an optional collapsed summary.
//
// Takes ctx (*ScrollContext) which provides the scrollable output context.
// Takes item (healthDisplayItem) which contains the probe data to render.
// Takes selected (bool) which indicates if this item is currently selected.
func (p *HealthPanel) renderProbeItem(ctx *ScrollContext, item healthDisplayItem, selected bool) {
	ctx.WriteLineIfVisible(func() string {
		return p.renderProbeHeader(item, selected)
	})

	if !p.IsExpanded(item.probeKey) && len(item.probeStatus.Dependencies) > 0 {
		ctx.WriteLineIfVisible(func() string {
			return p.renderProbeSummary(item.probeStatus)
		})
	}
}

// renderDependencyItem renders a dependency row and its details.
//
// Takes ctx (*ScrollContext) which provides the scrollable rendering context.
// Takes item (healthDisplayItem) which holds the dependency to display.
// Takes selected (bool) which indicates whether this item is currently
// selected.
func (p *HealthPanel) renderDependencyItem(ctx *ScrollContext, item healthDisplayItem, selected bool) {
	ctx.WriteLineIfVisible(func() string {
		return p.renderDependencyRow(item.dependency, selected)
	})
	p.renderDependencyDetails(ctx, item.dependency)
}

// renderProbeHeader renders a header row for a probe with cursor, expand
// marker, status indicator, title, state text, sparkline, and duration.
//
// Takes item (healthDisplayItem) which contains the probe data to display.
// Takes selected (bool) which shows if this row is currently selected.
//
// Returns string which is the formatted header row ready for display.
//
// Safe for concurrent use. Acquires a read lock on stateMutex to access
// history data.
func (p *HealthPanel) renderProbeHeader(item healthDisplayItem, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	expanded := p.IsExpanded(item.probeKey)
	expandChar := RenderExpandIndicator(expanded)
	indicator := healthStateIndicator(item.probeStatus.State)

	title := item.probeKey
	switch item.probeKey {
	case probeKeyLiveness:
		title = "Liveness"
	case probeKeyReadiness:
		title = "Readiness"
	}

	titleStyle := lipgloss.NewStyle().Bold(true)
	if selected && p.Focused() {
		titleStyle = titleStyle.Foreground(colourPrimary)
	}

	stateText := RenderDimText(item.probeStatus.State.String())

	p.stateMutex.RLock()
	var history []float64
	if item.probeKey == probeKeyLiveness {
		history = p.livenessHistory.Values()
	} else {
		history = p.readinessHistory.Values()
	}
	p.stateMutex.RUnlock()

	var sparkline string
	if len(history) > 1 {
		config := DefaultSparklineConfig()
		config.Width = healthSparklineWidth
		config.ShowCurrent = false
		sparkline = " " + Sparkline(history, &config)
	}

	var durationString string
	if item.probeStatus.Duration > 0 {
		durationString = RenderDimText(fmt.Sprintf("  (%s)", item.probeStatus.Duration.Round(time.Millisecond)))
	}

	return fmt.Sprintf("%s%s %s %s  %s%s%s",
		cursor, expandChar, indicator, titleStyle.Render(title), stateText, sparkline, durationString)
}

// renderDependencyRow renders a single dependency row for the health panel.
//
// Takes dependency (*HealthStatus) which provides the health status to show.
// Takes selected (bool) which indicates whether this row is selected.
//
// Returns string which is the formatted row with cursor, indicator, name,
// state, and duration.
func (p *HealthPanel) renderDependencyRow(dependency *HealthStatus, selected bool) string {
	cursor := RenderCursorStyled(selected, p.Focused(), ChildCursorConfig())
	indicator := healthStateIndicator(dependency.State)
	name := dependency.Name
	if selected && p.Focused() {
		name = lipgloss.NewStyle().Bold(true).Render(name)
	}

	stateText := RenderDimText(dependency.State.String())

	var durationString string
	if dependency.Duration > 0 {
		durationString = RenderDimText(fmt.Sprintf("  (%s)", dependency.Duration.Round(time.Millisecond)))
	}

	return fmt.Sprintf("%s%s %s  %s%s", cursor, indicator, name, stateText, durationString)
}

// renderDependencyDetails renders the full details for a dependency that
// cannot be selected.
//
// Takes ctx (*ScrollContext) which provides the scroll context for output.
// Takes dependency (*HealthStatus) which specifies the dependency to render.
func (*HealthPanel) renderDependencyDetails(ctx *ScrollContext, dependency *HealthStatus) {
	if dependency.Message != "" && dependency.State != HealthStateHealthy {
		ctx.WriteLineIfVisible(func() string {
			return fmt.Sprintf("        %s",
				lipgloss.NewStyle().
					Foreground(colourForegroundDim).
					Italic(true).
					Render(dependency.Message))
		})
	}

	for _, nested := range dependency.Dependencies {
		ctx.WriteLineIfVisible(func() string {
			nestedIndicator := healthStateIndicator(nested.State)
			return fmt.Sprintf("        %s %s",
				nestedIndicator,
				RenderDimText(fmt.Sprintf("%s (%s)", nested.Name, nested.State.String())))
		})
	}
}

// renderProbeSummary renders the collapsed summary line.
//
// Takes status (*HealthStatus) which provides the health status to summarise.
//
// Returns string which contains the formatted summary showing component counts
// by state. Returns an empty string if no components exist.
func (*HealthPanel) renderProbeSummary(status *HealthStatus) string {
	counts := status.CountByState()
	parts := make([]string, 0, healthStateCount)

	if c := counts[HealthStateHealthy]; c > 0 {
		parts = append(parts, statusHealthyStyle.Render(fmt.Sprintf("%d healthy", c)))
	}
	if c := counts[HealthStateDegraded]; c > 0 {
		parts = append(parts, statusDegradedStyle.Render(fmt.Sprintf("%d degraded", c)))
	}
	if c := counts[HealthStateUnhealthy]; c > 0 {
		parts = append(parts, statusUnhealthyStyle.Render(fmt.Sprintf("%d unhealthy", c)))
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("      %s",
		RenderDimText(fmt.Sprintf("(%d components: %s)", len(status.Dependencies), strings.Join(parts, ", "))))
}

// updateHealthHistory appends the current health states to the history.
// Must be called while holding stateMutex.
func (p *HealthPanel) updateHealthHistory() {
	if p.liveness != nil {
		p.livenessHistory.Append(healthStateToValue(p.liveness.State))
	}
	if p.readiness != nil {
		p.readinessHistory.Append(healthStateToValue(p.readiness.State))
	}
}

// refresh fetches new health data from the provider.
//
// Returns tea.Cmd which produces a HealthRefreshMessage with liveness and
// readiness status.
func (p *HealthPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return HealthRefreshMessage{Liveness: nil, Readiness: nil, Err: errNoHealthProvider}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("health panel data fetch exceeded 5s timeout"))
		defer cancel()

		liveness, _ := p.provider.Liveness(ctx)
		readiness, _ := p.provider.Readiness(ctx)

		return HealthRefreshMessage{
			Liveness:  liveness,
			Readiness: readiness,
			Err:       nil,
		}
	}
}

// RenderRow renders a health display item row.
//
// Takes item (healthDisplayItem) which holds the health data to
// render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted row for display.
func (r *healthRenderer) RenderRow(item healthDisplayItem, _ int, selected, _ bool, _ int) string {
	if item.isProbeRow {
		return r.panel.renderProbeHeader(item, selected)
	}
	return r.panel.renderDependencyRow(item.dependency, selected)
}

// RenderExpanded returns expanded content lines for a health item.
//
// Takes item (healthDisplayItem) which holds the health data to
// expand.
// Takes _ (int) which is the unused content width.
//
// Returns []string which contains the detail lines, or nil if no
// expanded content is available.
func (r *healthRenderer) RenderExpanded(item healthDisplayItem, _ int) []string {
	if item.isProbeRow {
		if !r.panel.IsExpanded(item.probeKey) && len(item.probeStatus.Dependencies) > 0 {
			return []string{r.panel.renderProbeSummary(item.probeStatus)}
		}
		return nil
	}

	lines := make([]string, 0)
	dependency := item.dependency

	if dependency.Message != "" && dependency.State != HealthStateHealthy {
		lines = append(lines, fmt.Sprintf("        %s",
			lipgloss.NewStyle().
				Foreground(colourForegroundDim).
				Italic(true).
				Render(dependency.Message)))
	}

	for _, nested := range dependency.Dependencies {
		indicator := healthStateIndicator(nested.State)
		lines = append(lines, fmt.Sprintf("        %s %s",
			indicator,
			RenderDimText(fmt.Sprintf("%s (%s)", nested.Name, nested.State.String()))))
	}

	return lines
}

// GetID returns a unique identifier for the health item.
//
// Takes item (healthDisplayItem) which specifies the health display item to
// identify.
//
// Returns string which is the probe key for probe rows, or a composite key
// combining probe key and dependency index for other items.
func (*healthRenderer) GetID(item healthDisplayItem) string {
	if item.isProbeRow {
		return item.probeKey
	}
	return fmt.Sprintf("%s:%d", item.probeKey, item.dependencyIndex)
}

// MatchesFilter checks whether the item matches the search query.
//
// Takes item (healthDisplayItem) which is the item to check.
// Takes query (string) which is the search term to match against.
//
// Returns bool which is true if the item matches the query.
func (*healthRenderer) MatchesFilter(item healthDisplayItem, query string) bool {
	q := strings.ToLower(query)
	if item.isProbeRow {
		return strings.Contains(strings.ToLower(item.probeKey), q)
	}
	return strings.Contains(strings.ToLower(item.dependency.Name), q)
}

// IsExpandable reports whether the item can show more details.
//
// Takes item (healthDisplayItem) which is the display item to check.
//
// Returns bool which is true when the item has content that can be shown.
func (*healthRenderer) IsExpandable(item healthDisplayItem) bool {
	if item.isProbeRow {
		return len(item.probeStatus.Dependencies) > 0
	}
	dependency := item.dependency
	return (dependency.Message != "" && dependency.State != HealthStateHealthy) || len(dependency.Dependencies) > 0
}

// ExpandedLineCount returns the number of detail lines when expanded.
//
// Takes item (healthDisplayItem) which specifies the display item to measure.
//
// Returns int which is the number of lines shown in expanded view.
func (r *healthRenderer) ExpandedLineCount(item healthDisplayItem) int {
	if item.isProbeRow {
		if !r.panel.IsExpanded(item.probeKey) && len(item.probeStatus.Dependencies) > 0 {
			return 1
		}
		return 0
	}

	count := 0
	dependency := item.dependency
	if dependency.Message != "" && dependency.State != HealthStateHealthy {
		count++
	}
	count += len(dependency.Dependencies)
	return count
}

// sortHealthDependencies returns a sorted copy of the given dependencies.
//
// Takes deps ([]*HealthStatus) which contains the dependencies to sort.
//
// Returns []*HealthStatus which is a new slice sorted by name.
func sortHealthDependencies(deps []*HealthStatus) []*HealthStatus {
	sorted := make([]*HealthStatus, len(deps))
	copy(sorted, deps)
	slices.SortFunc(sorted, func(a, b *HealthStatus) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return sorted
}

// healthStateIndicator returns a coloured indicator for a health state.
//
// Takes state (HealthState) which specifies the health state to display.
//
// Returns string which is the styled indicator symbol.
func healthStateIndicator(state HealthState) string {
	switch state {
	case HealthStateHealthy:
		return statusHealthyStyle.Render(SymbolStatusFilled)
	case HealthStateDegraded:
		return statusDegradedStyle.Render(SymbolStatusFilled)
	case HealthStateUnhealthy:
		return statusUnhealthyStyle.Render(SymbolStatusFilled)
	default:
		return lipgloss.NewStyle().Foreground(colourForegroundDim).Render(SymbolStatusEmpty)
	}
}
