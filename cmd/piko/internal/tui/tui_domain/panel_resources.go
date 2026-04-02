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

	"piko.sh/piko/wdk/clock"
)

const (
	// resourcesFDTargetMinWidth is the minimum width for file descriptor targets.
	resourcesFDTargetMinWidth = 20

	// resourcesMaxHistory is the maximum number of past data points to keep.
	resourcesMaxHistory = 1800

	// resourcesFDFixedWidth is the fixed width in characters for file descriptor rows.
	resourcesFDFixedWidth = 22

	// resourcesAgeWarningThreshold is the age after which resource data is
	// highlighted with a warning style.
	resourcesAgeWarningThreshold = 30 * time.Minute

	// resourcesAgeErrorThreshold is the age after which a resource is shown as
	// an error.
	resourcesAgeErrorThreshold = 1 * time.Hour

	// resourcesSummarySparklineWidth is the character width of the sparkline in the
	// resources summary line.
	resourcesSummarySparklineWidth = 20
)

var (
	// _ is a compile-time check that ResourcesPanel implements Panel.
	_ Panel = (*ResourcesPanel)(nil)

	_ ItemRenderer[FDCategory] = (*resourcesRenderer)(nil)
)

// ResourcesPanel displays OS-level resources like file descriptors,
// categorised by type (files, TCP, UDP, pipes, etc.).
type ResourcesPanel struct {
	*AssetViewer[FDCategory]

	// clock provides time functions for tracking data freshness.
	clock clock.Clock

	// lastRefresh is when the resources data was last updated.
	lastRefresh time.Time

	// provider fetches file descriptor data; nil causes refresh to return an error.
	provider FDsProvider

	// err holds the last refresh error, or nil if refresh succeeded.
	err error

	// data holds the current file descriptor details for display.
	data *FDsData

	// fdCountHistory holds recent file descriptor counts to show trends.
	fdCountHistory *HistoryRing

	// stateMutex guards panel state for safe access from multiple goroutines.
	stateMutex sync.RWMutex
}

// resourcesRenderer renders resource items for a ResourcesPanel.
type resourcesRenderer struct {
	// panel is the parent panel used for rendering and search.
	panel *ResourcesPanel
}

// ResourcesRefreshMessage signals that new resource data is available.
type ResourcesRefreshMessage struct {
	// Err holds any error that occurred during the refresh.
	Err error

	// Data holds the file descriptor information from a successful refresh.
	Data *FDsData
}

// NewResourcesPanel creates a panel for viewing OS resource information.
//
// Takes provider (FDsProvider) which supplies file descriptor data.
// Takes c (clock.Clock) which provides time functions; if nil, uses the real
// clock.
//
// Returns *ResourcesPanel which is ready for use with an embedded AssetViewer.
func NewResourcesPanel(provider FDsProvider, c clock.Clock) *ResourcesPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &ResourcesPanel{
		AssetViewer:    nil,
		clock:          c,
		lastRefresh:    time.Time{},
		provider:       provider,
		err:            nil,
		data:           nil,
		fdCountHistory: NewHistoryRing(resourcesMaxHistory),
		stateMutex:     sync.RWMutex{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[FDCategory]{
		ID:           "resources",
		Title:        "Resources",
		Renderer:     &resourcesRenderer{panel: p},
		NavMode:      NavigationSimple,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "j/↓", Description: "Move down"},
			{Key: "k/↑", Description: "Move up"},
			{Key: "Space", Description: "Expand/collapse"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear/collapse"},
			{Key: "s", Description: "Toggle sort (age/fd)"},
			{Key: "r", Description: "Refresh"},
		},
	})

	return p
}

// Init initialises the panel.
//
// Returns tea.Cmd which triggers the initial data refresh.
func (p *ResourcesPanel) Init() tea.Cmd {
	return p.refresh()
}

// Update handles messages for the resources panel.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after processing.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *ResourcesPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	switch message := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(message)
	case ResourcesRefreshMessage:
		p.handleRefreshMessage(message)
		return p, nil
	case DataUpdatedMessage:
		command := p.refresh()
		return p, command
	}
	return p, nil
}

// View renders the panel with the specified dimensions.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *ResourcesPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderResourcesHeader,
		RenderEmptyState:    p.renderEmptyState,
		RenderItems:         p.renderCategories,
		TrimTrailingNewline: false,
	})
}

// handleRefreshMessage processes a resources refresh message.
//
// Takes message (ResourcesRefreshMessage) which contains the
// refresh data or error.
//
// Safe for concurrent use. Acquires both the panel mutex and state mutex.
func (p *ResourcesPanel) handleRefreshMessage(message ResourcesRefreshMessage) {
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
		p.data = message.Data
		p.err = nil
		p.updateFDCountHistory()
		if p.data != nil {
			p.setItemsUnlocked(p.data.Categories)
		}
	}
	p.lastRefresh = p.clock.Now()
}

// setItemsUnlocked sets items without locking. The caller must hold the lock.
//
// Takes categories ([]FDCategory) which specifies the items to set.
func (p *ResourcesPanel) setItemsUnlocked(categories []FDCategory) {
	p.items = categories
}

// handleKey handles key presses for the resources panel.
//
// Takes message (tea.KeyPressMsg) which contains the key press to handle.
//
// Returns Panel which is the panel to show after handling the key.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *ResourcesPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	switch message.String() {
	case "enter", "space":
		p.toggleCategoryExpansion()
		return p, nil
	}

	result := HandleCommonKeys(p.AssetViewer, message, p.refresh)
	if result.Handled {
		return p, result.Cmd
	}

	return p, nil
}

// toggleCategoryExpansion expands or collapses the category at the cursor.
// Only one category may be open at a time.
func (p *ResourcesPanel) toggleCategoryExpansion() {
	item := p.GetItemAtCursor()
	if item == nil {
		return
	}

	currentID := item.Category
	wasExpanded := p.IsExpanded(currentID)

	p.CollapseAll()
	if !wasExpanded {
		p.SetExpanded(currentID, true)
	}
}

// renderResourcesHeader renders the header section of the resources panel.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to content.
//
// Safe for concurrent use. Uses a read lock to access the error state.
func (p *ResourcesPanel) renderResourcesHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.Items()))
	}

	summary := p.renderSummaryLine()
	content.WriteString(summary)
	content.WriteString(stringNewline)
	usedLines++

	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()

	if err != nil {
		RenderErrorState(content, err)
		usedLines++
	}

	return usedLines
}

// renderSummaryLine builds the summary text showing file descriptor count and
// sparkline.
//
// Returns string which contains the formatted summary line with file descriptor
// count, optional sparkline history, and refresh timestamp.
//
// Safe for concurrent use. Acquires a read lock on stateMutex to access panel
// state.
func (p *ResourcesPanel) renderSummaryLine() string {
	var parts []string

	p.stateMutex.RLock()
	data := p.data
	lastRefresh := p.lastRefresh
	history := p.fdCountHistory.Values()
	p.stateMutex.RUnlock()

	if data != nil {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(colorForeground).
			Bold(true).
			Render(fmt.Sprintf("%d file descriptors", data.Total)))

		if len(history) > 1 {
			config := DefaultSparklineConfig()
			config.Width = resourcesSummarySparklineWidth
			sparkline := Sparkline(history, &config)
			parts = append(parts, RenderDimText(sparkline))
		}
	} else {
		parts = append(parts, RenderDimText("Loading..."))
	}

	if !lastRefresh.IsZero() {
		age := p.clock.Now().Sub(lastRefresh).Round(time.Second)
		parts = append(parts, RenderDimText(fmt.Sprintf("(updated %s ago)", age)))
	}

	return strings.Join(parts, " ")
}

// renderEmptyState writes the empty state message to the output.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *ResourcesPanel) renderEmptyState(content *strings.Builder) {
	message := "No resources available"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No resources match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderCategories writes all file descriptor categories to the output.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies the indices of items to display.
// Takes headerLines (int) which is the number of header lines to exclude from
// content height.
func (p *ResourcesPanel) renderCategories(content *strings.Builder, displayItems []int, headerLines int) {
	RenderExpandableItems(RenderExpandableItemsConfig[FDCategory]{
		Ctx:          NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines-1),
		Items:        p.Items(),
		DisplayItems: displayItems,
		Cursor:       p.Cursor(),
		GetID:        func(cat FDCategory) string { return cat.Category },
		IsExpanded:   p.IsExpanded,
		RenderRow:    p.renderCategoryRow,
		RenderExpand: p.renderCategoryFDs,
	})
}

// renderCategoryRow renders a single category row.
//
// Takes cat (FDCategory) which is the category to render.
// Takes selected (bool) which indicates whether this row is selected.
// Takes expanded (bool) which indicates whether the category is expanded.
//
// Returns string which is the formatted row for display.
func (p *ResourcesPanel) renderCategoryRow(cat FDCategory, selected, expanded bool) string {
	cursor := RenderCursor(selected, p.Focused())
	expandChar := RenderExpandIndicator(expanded)

	icon := p.categoryIcon(cat.Category)
	name := fmt.Sprintf("%s %s", icon, p.categoryDisplayName(cat.Category))
	if selected && p.Focused() {
		name = lipgloss.NewStyle().Bold(true).Render(name)
	}

	count := RenderDimText(fmt.Sprintf("(%d)", cat.Count))

	return fmt.Sprintf("%s %s %s %s", cursor, expandChar, name, count)
}

// renderCategoryFDs renders file descriptors for an expanded category.
//
// Takes ctx (*ScrollContext) which tracks the current scroll position.
// Takes cat (FDCategory) which holds the file descriptors to render.
func (p *ResourcesPanel) renderCategoryFDs(ctx *ScrollContext, cat FDCategory) {
	searchQuery := ""
	if p.Search() != nil {
		searchQuery = strings.ToLower(p.Search().Query())
	}

	for _, fd := range cat.FDs {
		if searchQuery != "" && !p.fdMatchesFilter(fd, searchQuery) {
			continue
		}

		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()

		ctx.WriteLineIfVisible(func() string {
			return p.renderFDRow(fd, selected)
		})
	}
}

// renderFDRow renders a single file descriptor row.
//
// Takes fd (FDInfo) which provides the file descriptor details to display.
// Takes selected (bool) which shows whether this row is currently selected.
//
// Returns string which is the formatted row ready for display.
func (p *ResourcesPanel) renderFDRow(fd FDInfo, selected bool) string {
	cursor := RenderCursorStyled(selected, p.Focused(), ChildCursorConfig())

	fdNum := RenderDimText(fmt.Sprintf("%4d ", fd.FD))

	age := p.formatAge(fd.AgeMs)

	availableWidth := p.ContentWidth() - resourcesFDFixedWidth
	targetWidth := max(resourcesFDTargetMinWidth, availableWidth)
	target := TruncateString(fd.Target, targetWidth)
	if selected && p.Focused() {
		target = lipgloss.NewStyle().Bold(true).Render(target)
	}

	return fmt.Sprintf("%s%s%s %s", cursor, fdNum, target, age)
}

// formatAge formats an age value and applies colour based on thresholds.
//
// Takes ageMs (int64) which is the age in milliseconds.
//
// Returns string which is the formatted age with colour styling.
func (*ResourcesPanel) formatAge(ageMs int64) string {
	age := time.Duration(ageMs) * time.Millisecond
	ageString := formatDuration(age)

	style := lipgloss.NewStyle().Foreground(colorForegroundDim)
	if age > resourcesAgeErrorThreshold {
		style = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	} else if age > resourcesAgeWarningThreshold {
		style = lipgloss.NewStyle().Foreground(colorWarning)
	}

	return style.Render(fmt.Sprintf("%8s", ageString))
}

// categoryIcon returns an emoji icon for the given resource category.
//
// Takes category (string) which identifies the resource type.
//
// Returns string which is the emoji icon for the category, or a question mark
// if the category is not recognised.
func (*ResourcesPanel) categoryIcon(category string) string {
	icons := map[string]string{
		"file":   "📄",
		"tcp":    "🌐",
		"udp":    "📡",
		"unix":   "🔌",
		"pipe":   "📥",
		"socket": "🔗",
		"other":  "❓",
	}
	if icon, ok := icons[category]; ok {
		return icon
	}
	return "❓"
}

// categoryDisplayName returns a display name for the given category.
//
// Takes category (string) which is the internal category identifier.
//
// Returns string which is the readable display name, or the original category
// if no mapping exists.
func (*ResourcesPanel) categoryDisplayName(category string) string {
	names := map[string]string{
		"file":   "Files",
		"tcp":    "TCP Connections",
		"udp":    "UDP Sockets",
		"unix":   "Unix Sockets",
		"pipe":   "Pipes",
		"socket": "Other Sockets",
		"other":  "Other",
	}
	if name, ok := names[category]; ok {
		return name
	}
	return category
}

// fdMatchesFilter checks if a file descriptor matches the search filter.
//
// Takes fd (FDInfo) which is the file descriptor to check.
// Takes query (string) which is the lowercase search term.
//
// Returns bool which is true if the fd target contains the query.
func (*ResourcesPanel) fdMatchesFilter(fd FDInfo, query string) bool {
	return strings.Contains(strings.ToLower(fd.Target), query)
}

// updateFDCountHistory appends the current file descriptor count to history.
// Must be called with stateMutex held.
func (p *ResourcesPanel) updateFDCountHistory() {
	if p.data == nil {
		return
	}
	p.fdCountHistory.Append(float64(p.data.Total))
}

// refresh fetches new resource data.
//
// Returns tea.Cmd which loads resource data in the background and sends a
// ResourcesRefreshMessage with the results or error.
func (p *ResourcesPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return ResourcesRefreshMessage{Err: errors.New("no resources provider"), Data: nil}
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second,
			errors.New("resources panel data fetch exceeded 5s timeout"))
		defer cancel()

		data, err := p.provider.GetFDs(ctx)
		if err != nil {
			return ResourcesRefreshMessage{Err: err, Data: nil}
		}

		return ResourcesRefreshMessage{Err: nil, Data: data}
	}
}

// RenderRow renders a file descriptor category row.
//
// Takes cat (FDCategory) which holds the category data to render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted category row for display.
func (r *resourcesRenderer) RenderRow(cat FDCategory, _ int, selected, _ bool, _ int) string {
	expanded := r.panel.IsExpanded(cat.Category)
	return r.panel.renderCategoryRow(cat, selected, expanded)
}

// RenderExpanded returns file descriptor lines for an expanded category.
//
// Takes cat (FDCategory) which holds the category to expand.
// Takes _ (int) which is the unused content width.
//
// Returns []string which contains the formatted file descriptor
// rows, filtered by the active search query.
func (r *resourcesRenderer) RenderExpanded(cat FDCategory, _ int) []string {
	searchQuery := ""
	if r.panel.Search() != nil {
		searchQuery = strings.ToLower(r.panel.Search().Query())
	}

	lines := make([]string, 0, len(cat.FDs))
	for _, fd := range cat.FDs {
		if searchQuery != "" && !r.panel.fdMatchesFilter(fd, searchQuery) {
			continue
		}
		lines = append(lines, r.panel.renderFDRow(fd, false))
	}
	return lines
}

// GetID returns the category's unique identifier.
//
// Takes cat (FDCategory) which specifies the category to retrieve the ID from.
//
// Returns string which is the category's identifier value.
func (*resourcesRenderer) GetID(cat FDCategory) string {
	return cat.Category
}

// MatchesFilter reports whether the category matches the search query.
//
// Takes cat (FDCategory) which is the category to check.
// Takes query (string) which is the lowercase search term to match.
//
// Returns bool which is true if the category name, display name, or any file
// descriptor within the category matches the query.
func (r *resourcesRenderer) MatchesFilter(cat FDCategory, query string) bool {
	if strings.Contains(strings.ToLower(cat.Category), query) {
		return true
	}
	displayName := r.panel.categoryDisplayName(cat.Category)
	if strings.Contains(strings.ToLower(displayName), query) {
		return true
	}
	for _, fd := range cat.FDs {
		if r.panel.fdMatchesFilter(fd, query) {
			return true
		}
	}
	return false
}

// IsExpandable returns true if the category has file descriptors to show.
//
// Takes cat (FDCategory) which specifies the category to check.
//
// Returns bool which is true when the category contains file descriptors.
func (*resourcesRenderer) IsExpandable(cat FDCategory) bool {
	return len(cat.FDs) > 0
}

// ExpandedLineCount returns the number of file descriptor lines.
//
// Takes cat (FDCategory) which specifies the category to count.
//
// Returns int which is the count of matching file descriptors, filtered by
// the current search query if one is active.
func (r *resourcesRenderer) ExpandedLineCount(cat FDCategory) int {
	if r.panel.Search() == nil || !r.panel.Search().HasQuery() {
		return len(cat.FDs)
	}

	query := strings.ToLower(r.panel.Search().Query())
	count := 0
	for _, fd := range cat.FDs {
		if r.panel.fdMatchesFilter(fd, query) {
			count++
		}
	}
	return count
}
