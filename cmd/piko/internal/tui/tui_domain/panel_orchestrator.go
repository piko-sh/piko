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
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"piko.sh/piko/wdk/clock"
)

// ViewMode controls what the orchestrator panel displays.
type ViewMode int

const (
	// ViewModeTasks displays the list of individual tasks in the panel.
	ViewModeTasks ViewMode = iota

	// ViewModeWorkflows shows workflow aggregates.
	ViewModeWorkflows
)

const (
	// taskRowSuffixWidth is the fixed width for the status and time at the end of
	// a task row.
	taskRowSuffixWidth = 27

	// taskRowMinNameWidth is the minimum width in characters for task names.
	taskRowMinNameWidth = 20

	// workflowRowSuffixWidth is the fixed width reserved for the row suffix.
	workflowRowSuffixWidth = 29

	// workflowRowMinNameWidth is the minimum character width for workflow names.
	workflowRowMinNameWidth = 30

	// hoursPerDay is the number of hours in a day, used for duration formatting.
	hoursPerDay = 24
)

var (
	// _ is a compile-time assertion that OrchestratorPanel implements Panel.
	_ Panel = (*OrchestratorPanel)(nil)

	_ ItemRenderer[Resource] = (*orchestratorRenderer)(nil)
)

// OrchestratorPanel displays task queue and workflow status. It implements
// the Panel interface.
type OrchestratorPanel struct {
	*AssetViewer[Resource]

	*StatusFilterMixin

	// clock provides the current time for calculating elapsed time.
	clock clock.Clock

	// tasks holds the task resources displayed in task view mode.
	tasks []Resource

	// workflows holds the workflow resources shown in the panel.
	workflows []Resource

	// viewMode holds the current display mode (tasks or workflows).
	viewMode ViewMode
}

// orchestratorRenderer implements ItemRenderer for Resource types.
type orchestratorRenderer struct {
	// panel references the parent panel for accessing expansion state and rendering.
	panel *OrchestratorPanel
}

// NewOrchestratorPanel creates a new orchestrator panel.
//
// Takes c (clock.Clock) which specifies the clock for time operations. If nil,
// defaults to a real clock.
//
// Returns *OrchestratorPanel which is the configured panel ready for use.
func NewOrchestratorPanel(c clock.Clock) *OrchestratorPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &OrchestratorPanel{
		AssetViewer:       nil,
		StatusFilterMixin: NewStatusFilterMixin(),
		clock:             c,
		tasks:             []Resource{},
		workflows:         []Resource{},
		viewMode:          ViewModeTasks,
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[Resource]{
		ID:           "orchestrator",
		Title:        "Orchestrator",
		Renderer:     &orchestratorRenderer{panel: p},
		NavMode:      NavigationSimple,
		EnableSearch: true,
		UseMutex:     false,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Move up/down"},
			{Key: "←/→", Description: "Tasks/Workflows view"},
			{Key: "Space", Description: "Expand / Collapse"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear search/collapse"},
			{Key: "f", Description: "Filter by status"},
			{Key: "g/G", Description: "Go to top/bottom"},
		},
	})

	return p
}

// SetTasks updates the task list with the given tasks.
//
// Takes tasks ([]Resource) which contains the new tasks to display.
func (p *OrchestratorPanel) SetTasks(tasks []Resource) {
	p.tasks = tasks
	if p.viewMode == ViewModeTasks {
		p.applyFilters()
	}
}

// SetWorkflows updates the workflow list with the given entries.
//
// Takes workflows ([]Resource) which contains the new workflow entries.
func (p *OrchestratorPanel) SetWorkflows(workflows []Resource) {
	p.workflows = workflows
	if p.viewMode == ViewModeWorkflows {
		p.applyFilters()
	}
}

// Update handles input messages for the orchestrator panel.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel state.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *OrchestratorPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	if keyMessage, ok := message.(tea.KeyPressMsg); ok {
		return p.handleKey(keyMessage)
	}
	return p, nil
}

// View renders the panel with the given dimensions.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *OrchestratorPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderOrchestratorHeader,
		RenderEmptyState:    p.renderOrchestratorEmptyState,
		RenderItems:         p.renderOrchestratorItems,
		TrimTrailingNewline: true,
	})
}

// applyFilters filters the current items by status and updates the viewer.
func (p *OrchestratorPanel) applyFilters() {
	items := p.currentItems()

	if p.FilterStatus() == nil {
		p.SetItems(items)
		return
	}

	filtered := make([]Resource, 0)
	for i := range items {
		if p.MatchesFilter(items[i].Status) {
			filtered = append(filtered, items[i])
		}
	}
	p.SetItems(filtered)
}

// currentItems returns the items for the current view mode without filtering.
//
// Returns []Resource which contains tasks or workflows based on the view mode.
func (p *OrchestratorPanel) currentItems() []Resource {
	if p.viewMode == ViewModeTasks {
		return p.tasks
	}
	return p.workflows
}

// handleKey processes key events for panel navigation and interaction.
//
// Takes message (tea.KeyPressMsg) which contains the key event to process.
//
// Returns Panel which is the panel to show after handling the key.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *OrchestratorPanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	if p.HandleNavigation(message) {
		return p, nil
	}

	switch message.String() {
	case "left", "h":
		p.switchToViewMode(ViewModeTasks)
		return p, nil
	case "right", "l":
		p.switchToViewMode(ViewModeWorkflows)
		return p, nil
	case "enter", "space":
		p.HandleExpansionToggle()
		return p, nil
	case "/":
		if p.Search() != nil {
			p.Search().SetWidth(p.ContentWidth())
			return p, p.Search().SearchBox().Open()
		}
	case "esc":
		if p.Search() != nil && p.Search().HasQuery() {
			p.Search().ClearQuery()
			p.applyFilters()
			return p, nil
		}
		if len(p.ExpandedMap()) > 0 {
			p.CollapseAll()
			return p, nil
		}
	case "f":
		p.CycleFilter()
		p.applyFilters()
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return p, nil
	}

	return p, nil
}

// switchToViewMode changes to the given view mode and resets the panel state.
//
// Takes mode (ViewMode) which specifies the view mode to switch to.
func (p *OrchestratorPanel) switchToViewMode(mode ViewMode) {
	if p.viewMode == mode {
		return
	}
	p.viewMode = mode
	p.SetCursor(0)
	p.SetScrollOffset(0)
	p.CollapseAll()
	if p.Search() != nil {
		p.Search().ClearQuery()
	}
	p.ClearFilter()
	p.applyFilters()
}

// renderOrchestratorHeader renders the search box, mode indicator, and filters.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines written to the content builder.
func (p *OrchestratorPanel) renderOrchestratorHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.currentItems()))
	}

	modeStyle := lipgloss.NewStyle().Foreground(colorForegroundDim).Bold(true)
	modeText := "Tasks [Workflows]"
	if p.viewMode == ViewModeTasks {
		modeText = "[Tasks] Workflows"
	}
	content.WriteString(modeStyle.Render(modeText))
	content.WriteString(stringNewline)
	usedLines++

	usedLines += p.RenderFilterStatus(content)

	return usedLines
}

// renderOrchestratorEmptyState writes the empty state message to the content
// builder.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *OrchestratorPanel) renderOrchestratorEmptyState(content *strings.Builder) {
	message := "No items"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No items match search"
	} else if p.HasFilter() {
		message = "No items match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderOrchestratorItems renders all items with their metadata.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies which item indices to show.
// Takes headerLines (int) which is the number of header lines to exclude from
// the content height.
func (p *OrchestratorPanel) renderOrchestratorItems(content *strings.Builder, displayItems []int, headerLines int) {
	RenderExpandableItems(RenderExpandableItemsConfig[Resource]{
		Ctx:          NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines),
		Items:        p.Items(),
		DisplayItems: displayItems,
		Cursor:       p.Cursor(),
		GetID:        func(r Resource) string { return r.ID },
		IsExpanded:   p.IsExpanded,
		RenderRow:    p.renderItemRow,
		RenderExpand: p.renderItemMetadata,
	})
}

// renderItemRow renders a task or workflow row based on current view mode.
//
// Takes item (Resource) which provides the resource data to render.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused expanded state.
//
// Returns string which is the formatted row for the current view
// mode.
func (p *OrchestratorPanel) renderItemRow(item Resource, selected, _ bool) string {
	if p.viewMode == ViewModeTasks {
		return p.renderTaskRow(item, selected)
	}
	return p.renderWorkflowRow(item, selected)
}

// renderTaskRow renders a single task row for display.
//
// Takes task (Resource) which provides the task data to display.
// Takes selected (bool) which indicates if this row is currently selected.
//
// Returns string which is the formatted row ready for display.
func (p *OrchestratorPanel) renderTaskRow(task Resource, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	indicator := StatusIndicator(task.Status)

	priority := task.Metadata["priority"]
	attempt := task.Metadata["attempt"]

	elapsed := p.clock.Now().Sub(task.UpdatedAt)
	timeString := formatDuration(elapsed)

	nameWidth := max(taskRowMinNameWidth, p.ContentWidth()-taskRowSuffixWidth)

	name := RenderName(task.Name, nameWidth, selected, p.Focused())

	statusText := StatusStyle(task.Status).Render(fmt.Sprintf("%-8s", task.StatusText))

	prioChar := "-"
	if len(priority) > 0 {
		prioChar = priority[:1]
	}

	return fmt.Sprintf("%s%s %s %s P:%s A:%s %s",
		cursor, indicator, name, statusText, prioChar, attempt, timeString)
}

// renderWorkflowRow renders a single workflow row for display.
//
// Takes workflow (Resource) which provides the workflow data to display.
// Takes selected (bool) which indicates whether this row is currently selected.
//
// Returns string which is the formatted row ready for display.
func (p *OrchestratorPanel) renderWorkflowRow(workflow Resource, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	indicator := StatusIndicator(workflow.Status)

	taskCount := workflow.Metadata["task_count"]
	completeCount := workflow.Metadata["complete_count"]
	failedCount := workflow.Metadata["failed_count"]
	progress := workflow.Metadata["progress"]

	nameWidth := max(workflowRowMinNameWidth, p.ContentWidth()-workflowRowSuffixWidth)

	name := RenderName(workflow.Name, nameWidth, selected, p.Focused())

	progressInfo := RenderDimText(fmt.Sprintf("%s/%s ✓%s ✗%s", completeCount, taskCount, completeCount, failedCount))
	if progress != "" {
		progressInfo += " " + lipgloss.NewStyle().Foreground(colorInfo).Render(progress)
	}

	return fmt.Sprintf("%s%s %s %s", cursor, indicator, name, progressInfo)
}

// renderItemMetadata displays sorted metadata for an expanded item.
//
// Takes ctx (*ScrollContext) which tracks scroll position and line rendering.
// Takes item (Resource) which provides the metadata to display.
func (p *OrchestratorPanel) renderItemMetadata(ctx *ScrollContext, item Resource) {
	keys := make([]string, 0, len(item.Metadata))
	for key := range item.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		value := item.Metadata[key]
		if value == "" {
			continue
		}

		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()

		ctx.WriteLineIfVisible(func() string {
			return RenderMetadataRow(key, value, MetadataRowConfig{
				IndentSpaces:    0,
				WidthAdjustment: 0,
				Selected:        selected,
				Focused:         p.Focused(),
				ContentWidth:    p.ContentWidth(),
			})
		})
	}
}

// RenderRow renders a resource row.
//
// Takes item (Resource) which provides the resource data to render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted resource row for display.
func (r *orchestratorRenderer) RenderRow(item Resource, _ int, selected, _ bool, _ int) string {
	expanded := r.panel.IsExpanded(item.ID)
	return r.panel.renderItemRow(item, selected, expanded)
}

// RenderExpanded returns formatted metadata lines for a resource.
//
// Takes item (Resource) which provides the resource data to display.
// Takes width (int) which sets the available width for content.
//
// Returns []string which contains the formatted metadata rows sorted by key.
func (*orchestratorRenderer) RenderExpanded(item Resource, width int) []string {
	keys := make([]string, 0, len(item.Metadata))
	for key := range item.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := item.Metadata[key]
		if value == "" {
			continue
		}
		lines = append(lines, RenderMetadataRow(key, value, MetadataRowConfig{
			IndentSpaces:    0,
			WidthAdjustment: 0,
			Selected:        false,
			Focused:         false,
			ContentWidth:    width,
		}))
	}
	return lines
}

// GetID returns the unique identifier for the given resource.
//
// Takes item (Resource) which is the resource to get the identifier from.
//
// Returns string which is the unique identifier.
func (*orchestratorRenderer) GetID(item Resource) string {
	return item.ID
}

// MatchesFilter reports whether the resource matches the search query.
//
// Takes item (Resource) which is the resource to check.
// Takes query (string) which is the search term to match against.
//
// Returns bool which is true if the resource name or ID contains the query.
func (*orchestratorRenderer) MatchesFilter(item Resource, query string) bool {
	return strings.Contains(strings.ToLower(item.Name), query) ||
		strings.Contains(strings.ToLower(item.ID), query)
}

// IsExpandable reports whether the resource has any metadata to display.
//
// Takes item (Resource) which is the resource to check.
//
// Returns bool which is true when the resource has at least one non-empty
// metadata value.
func (*orchestratorRenderer) IsExpandable(item Resource) bool {
	for _, value := range item.Metadata {
		if value != "" {
			return true
		}
	}
	return false
}

// ExpandedLineCount returns the number of non-empty metadata lines.
//
// Takes item (Resource) which provides the metadata to count.
//
// Returns int which is the count of non-empty metadata entries.
func (*orchestratorRenderer) ExpandedLineCount(item Resource) int {
	count := 0
	for _, value := range item.Metadata {
		if value != "" {
			count++
		}
	}
	return count
}

// formatDuration formats a duration as a string that is easy to read.
//
// Takes d (time.Duration) which is the duration to format.
//
// Returns string which shows the duration using the largest fitting unit
// (seconds, minutes, hours, or days).
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/hoursPerDay))
	}
}
