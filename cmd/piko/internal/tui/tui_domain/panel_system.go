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
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/safeconv"

	"piko.sh/piko/wdk/clock"
)

const (
	// maxHistorySize is the number of samples to keep for sparklines.
	maxHistorySize = 1800

	// minSparklineWidth is the smallest allowed width for sparkline charts.
	minSparklineWidth = 20

	// sparklineWidthAdjust is the adjustment for label and padding width.
	sparklineWidthAdjust = 40

	// millicoresPerCore is the number of millicores in one CPU core.
	millicoresPerCore = 1000

	// systemSectionLabelWidth is the fixed width for section labels in characters.
	systemSectionLabelWidth = 12

	// systemRecentPausesLimit is the number of recent GC pauses to display.
	systemRecentPausesLimit = 5

	// nsToMicrosDiv is the divisor to convert nanoseconds to microseconds.
	nsToMicrosDiv = 1000

	// expandedLinesCPUBase is the base line count for the expanded CPU section.
	expandedLinesCPUBase = 4

	// expandedLinesCPUHistory is the number of extra lines added when CPU
	// history is displayed.
	expandedLinesCPUHistory = 3

	// expandedLinesMemory is the number of lines shown in the expanded memory section.
	expandedLinesMemory = 13

	// expandedLinesGoroutinesBase is the base line count for the goroutines section.
	expandedLinesGoroutinesBase = 1

	// expandedLinesGoroutinesHist is the number of extra lines added when
	// the goroutine histogram is shown.
	expandedLinesGoroutinesHist = 3

	// expandedLinesGCBase is the base number of lines for the GC section display.
	expandedLinesGCBase = 5

	// expandedLinesBuild is the number of lines shown in the build section when
	// expanded.
	expandedLinesBuild = 6

	// expandedLinesProcess is the line count when the process section is expanded.
	expandedLinesProcess = 4

	// expandedLinesRuntime is the number of lines shown when the runtime section
	// is expanded.
	expandedLinesRuntime = 2

	// sectionCPU is the section key for the CPU usage display.
	sectionCPU = "cpu"

	// sectionMemory is the key for the memory statistics section.
	sectionMemory = "memory"

	// sectionGoroutines is the key for the goroutines panel section.
	sectionGoroutines = "goroutines"

	// sectionGC is the section key for garbage collection pause data.
	sectionGC = "gc"

	// sectionBuild is the section key for build information.
	sectionBuild = "build"

	// sectionProcess is the key for the process information section.
	sectionProcess = "process"

	// sectionRuntime identifies the Go runtime settings section.
	sectionRuntime = "runtime"

	// sectionCache is the section key for render cache statistics.
	sectionCache = "cache"

	// expandedLinesCache is the number of lines shown when the cache section
	// is expanded.
	expandedLinesCache = 2
)

var (
	_ Panel = (*SystemPanel)(nil)

	_ ItemRenderer[systemSection] = (*systemRenderer)(nil)
)

// systemSection represents a display section in the system panel.
type systemSection struct {
	// key identifies the section for tracking expanded or collapsed state.
	key string
}

// SystemRefreshMessage signals that new system statistics are available.
type SystemRefreshMessage struct {
	// Stats holds the system statistics from this refresh.
	Stats *SystemStats

	// Err holds any error from the refresh operation; nil means success.
	Err error
}

// SystemPanel shows Go runtime statistics from the Piko server.
// It implements the Panel interface.
type SystemPanel struct {
	*AssetViewer[systemSection]

	// clock provides time values for refresh timestamps and GC calculations.
	clock clock.Clock

	// lastRefresh records when the panel data was last updated.
	lastRefresh time.Time

	// provider supplies system statistics for display.
	provider SystemProvider

	// err holds the last refresh error; nil means the refresh succeeded.
	err error

	// stats holds the most recent system statistics snapshot.
	stats *SystemStats

	// cpuHistory stores CPU usage samples in millicores for sparkline display.
	cpuHistory *HistoryRing

	// memHistory stores memory allocation values for the sparkline display.
	memHistory *HistoryRing

	// heapHistory stores recent heap allocation values for the graph.
	heapHistory *HistoryRing

	// goroutineHistory stores past goroutine counts for sparkline display.
	goroutineHistory *HistoryRing

	// gcPauseHistory stores recent GC pause times in microseconds.
	gcPauseHistory *HistoryRing

	// stateMutex guards the err and stats fields for safe concurrent access.
	stateMutex sync.RWMutex
}

// systemRenderer implements ItemRenderer for systemSection entries.
type systemRenderer struct {
	// panel is the parent SystemPanel used for rendering.
	panel *SystemPanel
}

// NewSystemPanel creates a new system stats panel.
//
// Takes provider (SystemProvider) which supplies system statistics data.
// Takes c (clock.Clock) which provides time functions; if nil, uses the real
// clock.
//
// Returns *SystemPanel which is set up with history tracking and an asset
// viewer for showing system information.
func NewSystemPanel(provider SystemProvider, c clock.Clock) *SystemPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &SystemPanel{
		AssetViewer:      nil,
		clock:            c,
		lastRefresh:      time.Time{},
		provider:         provider,
		err:              nil,
		stats:            nil,
		cpuHistory:       NewHistoryRing(maxHistorySize),
		memHistory:       NewHistoryRing(maxHistorySize),
		heapHistory:      NewHistoryRing(maxHistorySize),
		goroutineHistory: NewHistoryRing(maxHistorySize),
		gcPauseHistory:   NewHistoryRing(maxHistorySize),
		stateMutex:       sync.RWMutex{},
	}

	sections := []systemSection{
		{key: sectionBuild},
		{key: sectionProcess},
		{key: sectionRuntime},
		{key: sectionCPU},
		{key: sectionMemory},
		{key: sectionGoroutines},
		{key: sectionGC},
		{key: sectionCache},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[systemSection]{
		ID:           "system",
		Title:        "System",
		Renderer:     &systemRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: false,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Move up/down"},
			{Key: "Space", Description: "Toggle expand"},
			{Key: "g/G", Description: "Go to top/bottom"},
			{Key: "r", Description: "Refresh"},
		},
	})

	p.SetItems(sections)

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which refreshes the panel data.
func (p *SystemPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles messages for the system panel.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after handling the message.
// Returns tea.Cmd which is the command to run, or nil if there is none.
func (p *SystemPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case SystemRefreshMessage:
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
func (p *SystemPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderSystemHeader,
		RenderEmptyState:    p.renderSystemEmptyState,
		RenderItems:         p.renderSystemItems,
		TrimTrailingNewline: true,
	})
}

// handleRefreshMessage processes a system refresh message.
//
// Takes message (SystemRefreshMessage) which holds the refresh data or error.
//
// Safe for concurrent use. Locks both the panel mutex and state mutex.
func (p *SystemPanel) handleRefreshMessage(message SystemRefreshMessage) {
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
		p.stats = message.Stats
		p.err = nil
		p.updateHistory()
	}
	p.lastRefresh = p.clock.Now()
}

// updateHistory adds the current stats to the sparkline history.
// Caller must hold stateMutex.
func (p *SystemPanel) updateHistory() {
	if p.stats == nil {
		return
	}

	p.cpuHistory.Append(p.stats.CPUMillicores)
	p.memHistory.Append(float64(p.stats.Memory.Alloc))
	p.heapHistory.Append(float64(p.stats.Memory.HeapAlloc))
	p.goroutineHistory.Append(float64(p.stats.NumGoroutines))

	if p.stats.GC.LastPauseNs > 0 {
		pauseMicros := float64(p.stats.GC.LastPauseNs) / nsToMicrosDiv
		p.gcPauseHistory.Append(pauseMicros)
	}
}

// handleKey handles key events for the system panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to handle.
//
// Returns Panel which is this panel after handling the key event.
// Returns tea.Cmd which is the command to run, or nil if there is none.
func (p *SystemPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
	if result.Handled {
		return p, result.Cmd
	}

	return p, nil
}

// renderSystemHeader renders the header section showing error state and uptime
// information.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to the builder.
//
// Safe for concurrent use. Acquires a read lock on state before accessing
// error and stats fields.
func (p *SystemPanel) renderSystemHeader(content *strings.Builder) int {
	usedLines := 0

	p.stateMutex.RLock()
	err := p.err
	stats := p.stats
	p.stateMutex.RUnlock()

	if err != nil {
		RenderErrorState(content, err)
		content.WriteString(stringNewline)
		usedLines += 2
	}

	if stats != nil {
		uptimeString := formatUptime(stats.Uptime)
		header := lipgloss.NewStyle().
			Foreground(colourForegroundDim).
			Render(fmt.Sprintf("Uptime: %s  |  CPUs: %d  |  GOMAXPROCS: %d",
				uptimeString, stats.NumCPU, stats.GOMAXPROCS))
		content.WriteString(header)
		content.WriteString("\n\n")
		usedLines += 2
	}

	return usedLines
}

// renderSystemEmptyState renders the empty state message.
//
// Takes content (*strings.Builder) which receives the rendered output.
//
// Safe for concurrent use. Acquires a read lock on stats state.
func (p *SystemPanel) renderSystemEmptyState(content *strings.Builder) {
	p.stateMutex.RLock()
	stats := p.stats
	p.stateMutex.RUnlock()

	if stats == nil {
		content.WriteString(RenderDimText("Waiting for system stats..."))
	}
}

// renderSystemItems draws all system sections with their full content.
//
// Takes content (*strings.Builder) which receives the output text.
// Takes displayItems ([]int) which lists the items to show.
// Takes headerLines (int) which tells how many lines the header uses.
//
// Safe for concurrent use. Gets a read lock on stateMutex to read stats.
func (p *SystemPanel) renderSystemItems(content *strings.Builder, displayItems []int, headerLines int) {
	p.stateMutex.RLock()
	stats := p.stats
	p.stateMutex.RUnlock()

	if stats == nil {
		return
	}

	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()
	sparkWidth := max(minSparklineWidth, p.ContentWidth()-sparklineWidthAdjust)

	for _, itemIndex := range displayItems {
		if itemIndex >= len(items) {
			continue
		}

		section := items[itemIndex]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()
		expanded := p.IsExpanded(section.key)

		ctx.WriteLineIfVisible(func() string {
			return p.renderSectionRow(section, selected, expanded, sparkWidth)
		})

		if expanded {
			p.renderSectionDetails(ctx, section.key)
		}
	}
}

// sectionContent holds the computed content for a section row.
type sectionContent struct {
	// label is the display text for this section.
	label string

	// sparkline is the rendered graph showing the metric values over time.
	sparkline string

	// value is the formatted text to display for this section.
	value string
}

// getSectionContent builds the label, sparkline and value for a section.
//
// Takes section (systemSection) which identifies the system metric to show.
// Takes config (*SparklineConfig) which controls sparkline display settings.
//
// Returns sectionContent which holds the formatted label, sparkline and value.
//
// Safe for concurrent use. Acquires a read lock on stateMutex to access the
// current stats and history data.
func (p *SystemPanel) getSectionContent(section systemSection, config *SparklineConfig) sectionContent {
	p.stateMutex.RLock()
	stats := p.stats
	cpuHist := p.cpuHistory.Values()
	memHist := p.memHistory.Values()
	goroutineHist := p.goroutineHistory.Values()
	gcPauseHist := p.gcPauseHistory.Values()
	p.stateMutex.RUnlock()

	if stats == nil {
		return sectionContent{label: sectionLabel(section.key), sparkline: "", value: ""}
	}

	switch section.key {
	case sectionCPU:
		return sectionContent{label: "CPU", sparkline: Sparkline(cpuHist, config), value: formatMillicores(stats.CPUMillicores)}
	case sectionMemory:
		return sectionContent{label: "Memory", sparkline: Sparkline(memHist, config), value: inspector.FormatBytes(stats.Memory.Alloc)}
	case sectionGoroutines:
		return sectionContent{label: "Goroutines", sparkline: Sparkline(goroutineHist, config), value: strconv.Itoa(stats.NumGoroutines)}
	case sectionGC:
		lastPause := time.Duration(safeconv.Uint64ToInt64(stats.GC.LastPauseNs))
		return sectionContent{label: "GC Pause", sparkline: Sparkline(gcPauseHist, config), value: lastPause.Round(time.Microsecond).String()}
	case sectionBuild:
		return sectionContent{label: "Build", sparkline: "", value: fmt.Sprintf("%s (%s)", stats.Build.Version, stats.Build.GoVersion)}
	case sectionProcess:
		return sectionContent{label: "Process", sparkline: "", value: fmt.Sprintf("PID %d | RSS %s", stats.Process.PID, inspector.FormatBytes(stats.Process.RSS))}
	case sectionRuntime:
		return sectionContent{label: "Runtime", sparkline: "", value: fmt.Sprintf("GOGC=%s GOMEMLIMIT=%s", stats.Runtime.GOGC, stats.Runtime.GOMEMLIMIT)}
	case sectionCache:
		return sectionContent{label: "Cache", sparkline: "", value: fmt.Sprintf("Components: %d | SVGs: %d", stats.Cache.ComponentCacheSize, stats.Cache.SVGCacheSize)}
	default:
		return sectionContent{label: section.key, sparkline: "", value: ""}
	}
}

// renderSectionRow renders a single section row for the system panel.
//
// Takes section (systemSection) which sets which system section to render.
// Takes selected (bool) which shows if this row is currently selected.
// Takes expanded (bool) which shows if this section is expanded.
// Takes sparkWidth (int) which sets the width of the sparkline chart.
//
// Returns string which holds the formatted row with cursor, expand marker,
// label, sparkline, and value.
func (p *SystemPanel) renderSectionRow(section systemSection, selected, expanded bool, sparkWidth int) string {
	cursor := RenderCursor(selected, p.Focused())
	expandChar := RenderExpandIndicator(expanded)

	config := DefaultSparklineConfig()
	config.Width = sparkWidth
	config.ShowCurrent = false

	sc := p.getSectionContent(section, &config)

	label := PadRight(sc.label, systemSectionLabelWidth)
	if selected && p.Focused() {
		label = lipgloss.NewStyle().Bold(true).Render(label)
	}

	value := lipgloss.NewStyle().Foreground(colourForeground).Render(sc.value)

	return fmt.Sprintf("%s%s %s %s %s", cursor, expandChar, label, sc.sparkline, value)
}

// renderSectionDetails renders the expanded details for a section.
//
// Takes ctx (*ScrollContext) which provides the scroll context for rendering.
// Takes sectionKey (string) which identifies the section to render.
//
// Safe for concurrent use. Acquires a read lock on the state mutex to access
// statistics and history data.
func (p *SystemPanel) renderSectionDetails(ctx *ScrollContext, sectionKey string) {
	p.stateMutex.RLock()
	stats := p.stats
	cpuHist := p.cpuHistory.Values()
	goroutineHist := p.goroutineHistory.Values()
	p.stateMutex.RUnlock()

	if stats == nil {
		return
	}

	dimStyle := lipgloss.NewStyle().Foreground(colourForegroundDim)
	indent := "      "

	var lines []string

	switch sectionKey {
	case sectionCPU:
		lines = renderCPUDetails(stats, cpuHist, &dimStyle)
	case sectionMemory:
		lines = renderMemoryDetails(stats, &dimStyle)
	case sectionGoroutines:
		lines = renderGoroutineDetails(stats, goroutineHist, &dimStyle)
	case sectionGC:
		lines = renderGCDetails(stats, p.clock.Now(), &dimStyle)
	case sectionBuild:
		lines = renderBuildDetails(stats, &dimStyle)
	case sectionProcess:
		lines = renderProcessDetails(stats, &dimStyle)
	case sectionRuntime:
		lines = renderRuntimeDetails(stats, &dimStyle)
	case sectionCache:
		lines = renderCacheDetails(stats, &dimStyle)
	}

	for _, line := range lines {
		ctx.WriteLineIfVisible(func() string {
			return indent + line
		})
	}
}

// refresh fetches new system stats from the provider.
//
// Returns tea.Cmd which gets stats in the background and sends a
// SystemRefreshMessage with the result.
func (p *SystemPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return SystemRefreshMessage{Stats: nil, Err: errNoSystemProvider}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("system panel data fetch exceeded 5s timeout"))
		defer cancel()

		stats, err := p.provider.GetStats(ctx)
		if err != nil {
			return SystemRefreshMessage{Stats: nil, Err: err}
		}

		return SystemRefreshMessage{Stats: stats, Err: nil}
	}
}

// RenderRow renders a system section row.
//
// Takes section (systemSection) which identifies the section to
// render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes width (int) which sets the available width for rendering.
//
// Returns string which is the formatted section row for display.
func (r *systemRenderer) RenderRow(section systemSection, _ int, selected, _ bool, width int) string {
	expanded := r.panel.IsExpanded(section.key)
	sparkWidth := max(minSparklineWidth, width-sparklineWidthAdjust)
	return r.panel.renderSectionRow(section, selected, expanded, sparkWidth)
}

// RenderExpanded returns detail lines for an expanded system section.
//
// Takes section (systemSection) which identifies the section to
// expand.
// Takes _ (int) which is the unused content width.
//
// Returns []string which contains the indented detail lines for
// the section, or nil if no stats are available.
//
// Safe for concurrent use. Acquires a read lock on the panel
// state mutex to access stats and history data.
func (r *systemRenderer) RenderExpanded(section systemSection, _ int) []string {
	r.panel.stateMutex.RLock()
	stats := r.panel.stats
	cpuHist := r.panel.cpuHistory.Values()
	goroutineHist := r.panel.goroutineHistory.Values()
	r.panel.stateMutex.RUnlock()

	if stats == nil {
		return nil
	}

	dimStyle := lipgloss.NewStyle().Foreground(colourForegroundDim)
	indent := "      "

	var lines []string

	switch section.key {
	case sectionCPU:
		lines = renderCPUDetails(stats, cpuHist, &dimStyle)
	case sectionMemory:
		lines = renderMemoryDetails(stats, &dimStyle)
	case sectionGoroutines:
		lines = renderGoroutineDetails(stats, goroutineHist, &dimStyle)
	case sectionGC:
		lines = renderGCDetails(stats, r.panel.clock.Now(), &dimStyle)
	case sectionBuild:
		lines = renderBuildDetails(stats, &dimStyle)
	case sectionProcess:
		lines = renderProcessDetails(stats, &dimStyle)
	case sectionRuntime:
		lines = renderRuntimeDetails(stats, &dimStyle)
	case sectionCache:
		lines = renderCacheDetails(stats, &dimStyle)
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		result[i] = indent + line
	}
	return result
}

// GetID returns the system section's unique identifier.
//
// Takes section (systemSection) which specifies the section to identify.
//
// Returns string which is the unique key for the given section.
func (*systemRenderer) GetID(section systemSection) string {
	return section.key
}

// MatchesFilter reports whether the section matches the search query.
//
// Takes section (systemSection) which is the section to check.
// Takes query (string) which is the search term to match against.
//
// Returns bool which is true if the section key or label contains the query.
func (*systemRenderer) MatchesFilter(section systemSection, query string) bool {
	return strings.Contains(strings.ToLower(section.key), strings.ToLower(query)) ||
		strings.Contains(strings.ToLower(sectionLabel(section.key)), strings.ToLower(query))
}

// IsExpandable reports whether a section can be expanded to show more detail.
//
// Returns bool which is always true for system sections.
func (*systemRenderer) IsExpandable(_ systemSection) bool {
	return true
}

// ExpandedLineCount returns the number of detail lines for an expanded section.
//
// Takes section (systemSection) which identifies the section to measure.
//
// Returns int which is the line count for the specified section when expanded.
//
// Safe for concurrent use. Acquires a read lock on the panel state.
func (r *systemRenderer) ExpandedLineCount(section systemSection) int {
	r.panel.stateMutex.RLock()
	stats := r.panel.stats
	cpuHist := r.panel.cpuHistory.Values()
	goroutineHist := r.panel.goroutineHistory.Values()
	r.panel.stateMutex.RUnlock()

	if stats == nil {
		return 0
	}

	switch section.key {
	case sectionCPU:
		lines := expandedLinesCPUBase
		if len(cpuHist) > 1 {
			lines += expandedLinesCPUHistory
		}
		return lines
	case sectionMemory:
		return expandedLinesMemory
	case sectionGoroutines:
		lines := expandedLinesGoroutinesBase
		if len(goroutineHist) > 0 {
			lines += expandedLinesGoroutinesHist
		}
		return lines
	case sectionGC:
		lines := expandedLinesGCBase
		if len(stats.GC.RecentPauses) > 0 {
			lines++
		}
		return lines
	case sectionBuild:
		return expandedLinesBuild
	case sectionProcess:
		return expandedLinesProcess
	case sectionRuntime:
		return expandedLinesRuntime
	case sectionCache:
		return expandedLinesCache
	}
	return 0
}

// sectionLabel returns a display label for a section key.
//
// Takes key (string) which is the section name to look up.
//
// Returns string which is the human-readable label, or the key itself if no
// label exists.
func sectionLabel(key string) string {
	labels := map[string]string{
		sectionCPU:        "CPU",
		sectionMemory:     "Memory",
		sectionGoroutines: "Goroutines",
		sectionGC:         "GC Pause",
		sectionBuild:      "Build",
		sectionProcess:    "Process",
		sectionRuntime:    "Runtime",
		sectionCache:      "Cache",
	}
	if label, ok := labels[key]; ok {
		return label
	}
	return key
}

// renderCPUDetails renders CPU section details.
//
// Takes stats (*SystemStats) which provides current CPU metrics.
// Takes cpuHist ([]float64) which holds recent CPU usage history for
// min/max/avg calculations.
// Takes dimStyle (*lipgloss.Style) which applies visual styling to the output.
//
// Returns []string which contains the formatted CPU detail lines.
func renderCPUDetails(stats *SystemStats, cpuHist []float64, dimStyle *lipgloss.Style) []string {
	lines := []string{
		dimStyle.Render(fmt.Sprintf("Current: %s", formatMillicores(stats.CPUMillicores))),
		dimStyle.Render(fmt.Sprintf("NumCPU: %d", stats.NumCPU)),
		dimStyle.Render(fmt.Sprintf("GOMAXPROCS: %d", stats.GOMAXPROCS)),
		dimStyle.Render(fmt.Sprintf("CGO calls: %d", stats.NumCGOCalls)),
	}
	if len(cpuHist) > 1 {
		minC, maxC, avgC := minMaxAverage(cpuHist)
		lines = append(lines,
			dimStyle.Render(fmt.Sprintf("Min (last %ds): %s", len(cpuHist), formatMillicores(minC))),
			dimStyle.Render(fmt.Sprintf("Max (last %ds): %s", len(cpuHist), formatMillicores(maxC))),
			dimStyle.Render(fmt.Sprintf("Average (last %ds): %s", len(cpuHist), formatMillicores(avgC))),
		)
	}
	return lines
}

// renderMemoryDetails formats memory statistics for display.
//
// Takes stats (*SystemStats) which provides the memory data to show.
// Takes dimStyle (*lipgloss.Style) which applies dimmed styling to the output.
//
// Returns []string which contains the formatted memory detail lines.
func renderMemoryDetails(stats *SystemStats, dimStyle *lipgloss.Style) []string {
	return []string{
		dimStyle.Render(fmt.Sprintf("Alloc: %s", inspector.FormatBytes(stats.Memory.Alloc))),
		dimStyle.Render(fmt.Sprintf("TotalAlloc: %s", inspector.FormatBytes(stats.Memory.TotalAlloc))),
		dimStyle.Render(fmt.Sprintf("Sys: %s", inspector.FormatBytes(stats.Memory.Sys))),
		dimStyle.Render(fmt.Sprintf("HeapAlloc: %s", inspector.FormatBytes(stats.Memory.HeapAlloc))),
		dimStyle.Render(fmt.Sprintf("HeapSys: %s", inspector.FormatBytes(stats.Memory.HeapSys))),
		dimStyle.Render(fmt.Sprintf("HeapIdle: %s", inspector.FormatBytes(stats.Memory.HeapIdle))),
		dimStyle.Render(fmt.Sprintf("HeapInuse: %s", inspector.FormatBytes(stats.Memory.HeapInuse))),
		dimStyle.Render(fmt.Sprintf("HeapReleased: %s", inspector.FormatBytes(stats.Memory.HeapReleased))),
		dimStyle.Render(fmt.Sprintf("HeapObjects: %d", stats.Memory.HeapObjects)),
		dimStyle.Render(fmt.Sprintf("LiveObjects: %d", stats.Memory.LiveObjects)),
		dimStyle.Render(fmt.Sprintf("Mallocs: %d", stats.Memory.Mallocs)),
		dimStyle.Render(fmt.Sprintf("Frees: %d", stats.Memory.Frees)),
		dimStyle.Render(fmt.Sprintf("StackSys: %s", inspector.FormatBytes(stats.Memory.StackSys))),
	}
}

// renderGoroutineDetails renders the expanded details for the
// active tasks section.
//
// Takes stats (*SystemStats) which provides the current task
// count.
// Takes goroutineHist ([]float64) which holds recent task counts
// for min/max/avg calculations.
// Takes dimStyle (*lipgloss.Style) which applies dimmed styling to
// the output.
//
// Returns []string which contains the formatted detail lines.
func renderGoroutineDetails(stats *SystemStats, goroutineHist []float64, dimStyle *lipgloss.Style) []string {
	lines := []string{
		dimStyle.Render(fmt.Sprintf("Current: %d", stats.NumGoroutines)),
	}
	if len(goroutineHist) > 0 {
		minG, maxG, avgG := minMaxAverage(goroutineHist)
		lines = append(lines,
			dimStyle.Render(fmt.Sprintf("Min (last %ds): %.0f", len(goroutineHist), minG)),
			dimStyle.Render(fmt.Sprintf("Max (last %ds): %.0f", len(goroutineHist), maxG)),
			dimStyle.Render(fmt.Sprintf("Average (last %ds): %.1f", len(goroutineHist), avgG)),
		)
	}
	return lines
}

// renderGCDetails renders the garbage collection section details.
//
// Takes stats (*SystemStats) which contains the garbage collection metrics.
// Takes now (time.Time) which is the current time for working out relative
// timestamps.
// Takes dimStyle (*lipgloss.Style) which styles the output text.
//
// Returns []string which contains the formatted GC detail lines.
func renderGCDetails(stats *SystemStats, now time.Time, dimStyle *lipgloss.Style) []string {
	lastGCTime := time.Unix(0, stats.GC.LastGC)
	pauseTotalDur := time.Duration(safeconv.Uint64ToInt64(stats.GC.PauseTotalNs))
	lines := []string{
		dimStyle.Render(fmt.Sprintf("NumGC: %d", stats.GC.NumGC)),
		dimStyle.Render(fmt.Sprintf("LastGC: %s ago", now.Sub(lastGCTime).Round(time.Second))),
		dimStyle.Render(fmt.Sprintf("PauseTotalNs: %s", pauseTotalDur)),
		dimStyle.Render(fmt.Sprintf("GCCPUFraction: %.4f%%", stats.GC.GCCPUFraction*100)),
		dimStyle.Render(fmt.Sprintf("NextGC: %s", inspector.FormatBytes(stats.GC.NextGC))),
	}
	if len(stats.GC.RecentPauses) > 0 {
		numPauses := min(systemRecentPausesLimit, len(stats.GC.RecentPauses))
		pauses := make([]string, 0, numPauses)
		for i := range numPauses {
			pauseNs := stats.GC.RecentPauses[i]
			pause := time.Duration(safeconv.Uint64ToInt64(pauseNs))
			pauses = append(pauses, pause.String())
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("Recent pauses: %s", strings.Join(pauses, ", "))))
	}
	return lines
}

// renderBuildDetails formats build information as styled text lines.
//
// Takes stats (*SystemStats) which holds the build details to display.
// Takes dimStyle (*lipgloss.Style) which sets the dimmed text style.
//
// Returns []string which contains the styled build detail lines.
func renderBuildDetails(stats *SystemStats, dimStyle *lipgloss.Style) []string {
	return []string{
		dimStyle.Render(fmt.Sprintf("Version: %s", stats.Build.Version)),
		dimStyle.Render(fmt.Sprintf("Commit: %s", stats.Build.Commit)),
		dimStyle.Render(fmt.Sprintf("BuildTime: %s", stats.Build.BuildTime)),
		dimStyle.Render(fmt.Sprintf("GoVersion: %s", stats.Build.GoVersion)),
		dimStyle.Render(fmt.Sprintf("OS: %s", stats.Build.OS)),
		dimStyle.Render(fmt.Sprintf("Arch: %s", stats.Build.Arch)),
	}
}

// renderProcessDetails builds the process information section as styled text.
//
// Takes stats (*SystemStats) which provides the process metrics to display.
// Takes dimStyle (*lipgloss.Style) which applies the visual styling to output.
//
// Returns []string which contains the formatted process detail lines.
func renderProcessDetails(stats *SystemStats, dimStyle *lipgloss.Style) []string {
	return []string{
		dimStyle.Render(fmt.Sprintf("PID: %d", stats.Process.PID)),
		dimStyle.Render(fmt.Sprintf("ThreadCount: %d", stats.Process.ThreadCount)),
		dimStyle.Render(fmt.Sprintf("FDCount: %d", stats.Process.FDCount)),
		dimStyle.Render(fmt.Sprintf("RSS: %s", inspector.FormatBytes(stats.Process.RSS))),
	}
}

// renderRuntimeDetails renders the runtime settings section.
//
// Takes stats (*SystemStats) which provides the runtime data to display.
// Takes dimStyle (*lipgloss.Style) which sets the dimmed text style.
//
// Returns []string which contains the formatted GOGC and GOMEMLIMIT values.
func renderRuntimeDetails(stats *SystemStats, dimStyle *lipgloss.Style) []string {
	return []string{
		dimStyle.Render(fmt.Sprintf("GOGC: %s", stats.Runtime.GOGC)),
		dimStyle.Render(fmt.Sprintf("GOMEMLIMIT: %s", stats.Runtime.GOMEMLIMIT)),
	}
}

// renderCacheDetails formats render cache statistics for display.
//
// Takes stats (*SystemStats) which provides the cache data to show.
// Takes dimStyle (*lipgloss.Style) which applies dimmed styling to the output.
//
// Returns []string which contains the formatted cache detail lines.
func renderCacheDetails(stats *SystemStats, dimStyle *lipgloss.Style) []string {
	return []string{
		dimStyle.Render(fmt.Sprintf("ComponentCacheSize: %d", stats.Cache.ComponentCacheSize)),
		dimStyle.Render(fmt.Sprintf("SVGCacheSize: %d", stats.Cache.SVGCacheSize)),
	}
}

// formatMillicores formats CPU usage in millicores, as used by Kubernetes.
//
// Takes m (float64) which is the CPU usage in millicores.
//
// Returns string which shows cores if m is at least 1000, or millicores with
// an "m" suffix otherwise.
func formatMillicores(m float64) string {
	if m >= millicoresPerCore {
		return fmt.Sprintf("%.2f", m/millicoresPerCore)
	}
	return fmt.Sprintf("%.0fm", m)
}

// formatUptime formats a duration as a readable uptime string.
//
// Takes d (time.Duration) which is the duration to format.
//
// Returns string which contains the formatted uptime in hours, minutes, and
// seconds (e.g. "2h30m15s", "5m30s", or "45s").
func formatUptime(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// minMaxAverage calculates the smallest, largest, and average values in a slice.
//
// Takes values ([]float64) which contains the numbers to analyse.
//
// Returns minVal (float64) which is the smallest value in the slice.
// Returns maxVal (float64) which is the largest value in the slice.
// Returns avg (float64) which is the mean of all values.
func minMaxAverage(values []float64) (minVal, maxVal, avg float64) {
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
