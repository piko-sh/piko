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
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SearchMixin provides search and filter features for panels.
// Embed this in panels that need search support.
type SearchMixin struct {
	// searchBox is the search input widget for user queries.
	searchBox *SearchBox

	// onFilter is called when the search filter changes.
	onFilter func()

	// searchQuery stores the current search filter text; empty means no filter.
	searchQuery string

	// filteredItems holds the indices of items that match the current search query;
	// nil when no filter is active.
	filteredItems []int
}

// NewSearchMixin creates a new search mixin with the given filter callback.
// The callback is called when the search query changes.
//
// Takes onFilter (func()) which is called when the search query changes.
//
// Returns *SearchMixin which is the configured search mixin ready for use.
func NewSearchMixin(onFilter func()) *SearchMixin {
	searchBox := NewSearchBox()

	m := &SearchMixin{
		searchBox:     searchBox,
		onFilter:      onFilter,
		searchQuery:   "",
		filteredItems: nil,
	}

	searchBox.SetOnClose(func(query string, confirmed bool) {
		if confirmed && query != "" {
			m.searchQuery = query
		} else if !confirmed {
			m.searchQuery = ""
		}
		if m.onFilter != nil {
			m.onFilter()
		}
	})

	return m
}

// SearchBox returns the underlying search box widget.
//
// Returns *SearchBox which provides access to the search input component.
func (m *SearchMixin) SearchBox() *SearchBox {
	return m.searchBox
}

// Query returns the current search query.
//
// Returns string which is the search query text.
func (m *SearchMixin) Query() string {
	return m.searchQuery
}

// SetQuery sets the search query directly (without opening the search box).
//
// Takes query (string) which specifies the search text to set.
func (m *SearchMixin) SetQuery(query string) {
	m.searchQuery = query
}

// ClearQuery clears the search query.
func (m *SearchMixin) ClearQuery() {
	m.searchQuery = ""
}

// IsActive returns true if the search box is currently active (open for input).
//
// Returns bool which indicates whether the search box is open for input.
func (m *SearchMixin) IsActive() bool {
	return m.searchBox.Active()
}

// HasQuery returns true if there is an active search query.
//
// Returns bool which is true when a search query has been set.
func (m *SearchMixin) HasQuery() bool {
	return m.searchQuery != ""
}

// FilteredItems returns the indices of items that match the current filter.
//
// Returns []int which contains the matching indices, or nil if no filter is
// active.
func (m *SearchMixin) FilteredItems() []int {
	return m.filteredItems
}

// SetWidth sets the width of the search box.
//
// Takes width (int) which specifies the width in characters.
func (m *SearchMixin) SetWidth(width int) {
	m.searchBox.SetWidth(width)
}

// Update handles search box input when the search box is active.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns bool which is true if the message was handled by the search box.
// Returns tea.Cmd which is a command to run, or nil if there is none.
func (m *SearchMixin) Update(message tea.Msg) (bool, tea.Cmd) {
	if !m.searchBox.Active() {
		return false, nil
	}
	_, command := m.searchBox.Update(message)
	return true, command
}

// HandleKey processes search-related key presses.
//
// Takes message (tea.KeyPressMsg) which contains the key press event to process.
// Takes width (int) which sets the width for the search box.
//
// Returns bool which is true if the key was handled by the search handler.
// Returns tea.Cmd which is the command to run, or nil if none.
func (m *SearchMixin) HandleKey(message tea.KeyPressMsg, width int) (bool, tea.Cmd) {
	switch message.String() {
	case "/":
		m.searchBox.SetWidth(width)
		return true, m.searchBox.Open()
	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			if m.onFilter != nil {
				m.onFilter()
			}
			return true, nil
		}
	}
	return false, nil
}

// UpdateFilter rebuilds the filtered items list based on the current search
// query by testing each item against the provided match function.
//
// Takes itemCount (int) which is the total number of items to filter.
// Takes matchFunc (func(...)) which tests if an item at the given index
// matches the query; the query passed to matchFunc is already lowercased.
func (m *SearchMixin) UpdateFilter(itemCount int, matchFunc func(index int, query string) bool) {
	if m.searchQuery == "" {
		m.filteredItems = nil
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filteredItems = make([]int, 0)
	for i := range itemCount {
		if matchFunc(i, query) {
			m.filteredItems = append(m.filteredItems, i)
		}
	}
}

// GetDisplayIndices returns the indices of items to display.
//
// Takes totalItems (int) which specifies the total number of items when no
// filter is active.
//
// Returns []int which contains the filtered indices if a search filter is
// active, or all indices from 0 to totalItems-1 otherwise.
func (m *SearchMixin) GetDisplayIndices(totalItems int) []int {
	if len(m.filteredItems) > 0 || m.searchQuery != "" {
		return m.filteredItems
	}
	items := make([]int, totalItems)
	for i := range totalItems {
		items[i] = i
	}
	return items
}

// RenderHeader renders the search box and filter status into the content
// builder.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes totalCount (int) which is the total number of items before filtering.
//
// Returns int which is the number of lines written to the builder.
func (m *SearchMixin) RenderHeader(content *strings.Builder, totalCount int) int {
	usedLines := 0

	if m.searchBox.Active() {
		content.WriteString(m.searchBox.View())
		content.WriteString(stringNewline)
		usedLines += 2
	}

	if m.searchQuery != "" {
		filterInfo := lipgloss.NewStyle().
			Foreground(colourInfo).
			Render(fmt.Sprintf("Filter: %q (%d of %d)", m.searchQuery, len(m.filteredItems), totalCount))
		content.WriteString(filterInfo)
		content.WriteString(stringNewline)
		usedLines++
	}

	return usedLines
}

// AdjustCursorAfterFilter adjusts the cursor position after a filter update.
// If the cursor is beyond the filtered items, it is moved to the last item.
//
// Takes cursor (int) which is the current cursor position to adjust.
//
// Returns int which is the adjusted cursor position within valid bounds.
func (m *SearchMixin) AdjustCursorAfterFilter(cursor int) int {
	if m.searchQuery == "" {
		return cursor
	}
	if cursor >= len(m.filteredItems) {
		return max(0, len(m.filteredItems)-1)
	}
	return cursor
}

// StatusFilterMixin provides status-based filtering for panels.
// This is separate from search filtering and can be combined with it.
type StatusFilterMixin struct {
	// filterStatus holds the current status filter; nil means no filter is set.
	filterStatus *ResourceStatus
}

// NewStatusFilterMixin creates a new status filter mixin.
//
// Returns *StatusFilterMixin which is ready for use with no active filter.
func NewStatusFilterMixin() *StatusFilterMixin {
	return &StatusFilterMixin{
		filterStatus: nil,
	}
}

// FilterStatus returns the current status filter.
//
// Returns *ResourceStatus which is the active filter, or nil if not set.
func (m *StatusFilterMixin) FilterStatus() *ResourceStatus {
	return m.filterStatus
}

// SetFilterStatus sets the status filter.
//
// Takes status (*ResourceStatus) which specifies the status to filter by.
func (m *StatusFilterMixin) SetFilterStatus(status *ResourceStatus) {
	m.filterStatus = status
}

// ClearFilter removes the status filter, allowing all statuses to pass.
func (m *StatusFilterMixin) ClearFilter() {
	m.filterStatus = nil
}

// HasFilter returns true if a status filter is active.
//
// Returns bool which indicates whether a filter is currently set.
func (m *StatusFilterMixin) HasFilter() bool {
	return m.filterStatus != nil
}

// CycleFilter moves to the next status filter value in the cycle.
// The order is: nil -> Healthy -> Degraded -> Unhealthy -> Pending -> nil.
func (m *StatusFilterMixin) CycleFilter() {
	if m.filterStatus == nil {
		m.filterStatus = new(ResourceStatusHealthy)
		return
	}

	switch *m.filterStatus {
	case ResourceStatusHealthy:
		m.filterStatus = new(ResourceStatusDegraded)
	case ResourceStatusDegraded:
		m.filterStatus = new(ResourceStatusUnhealthy)
	case ResourceStatusUnhealthy:
		m.filterStatus = new(ResourceStatusPending)
	default:
		m.filterStatus = nil
	}
}

// MatchesFilter checks whether the given status matches the current filter.
// Returns true if no filter is active.
//
// Takes status (ResourceStatus) which is the status to check against the filter.
//
// Returns bool which is true if the status matches or no filter is set.
func (m *StatusFilterMixin) MatchesFilter(status ResourceStatus) bool {
	if m.filterStatus == nil {
		return true
	}
	return status == *m.filterStatus
}

// RenderFilterStatus renders the current filter status into the content
// builder.
//
// Takes content (*strings.Builder) which receives the rendered filter status.
//
// Returns int which is the number of lines used (0 or 1).
func (m *StatusFilterMixin) RenderFilterStatus(content *strings.Builder) int {
	if m.filterStatus == nil {
		return 0
	}

	filterText := fmt.Sprintf("Status: %s", m.filterStatus.String())
	content.WriteString(lipgloss.NewStyle().
		Foreground(colourForegroundDim).
		Italic(true).
		Render(filterText))
	content.WriteString(stringNewline)
	return 1
}
