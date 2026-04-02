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
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
)

// AssetViewer provides standardised list viewing functionality for any item type.
// It handles navigation, expansion, search/filtering, and scroll-aware rendering.
//
// Generic parameter T is the item type (e.g., Resource, Span, metricDisplay).
type AssetViewer[T any] struct {
	// renderer displays each item in the viewer.
	renderer ItemRenderer[T]

	// expanded tracks which tree nodes are currently open.
	expanded map[string]bool

	// search allows finding assets in the viewer.
	search *SearchMixin

	// mu protects the asset viewer's state from concurrent access.
	mu *sync.RWMutex

	// items holds the list of assets to display.
	items []T

	BasePanel

	// navMode specifies the current navigation mode for the viewer.
	navMode NavigationMode
}

// AssetViewerConfig holds settings for creating an AssetViewer.
type AssetViewerConfig[T any] struct {
	// ID is the unique identifier for the asset viewer.
	ID string

	// Title is the display name for the asset viewer.
	Title string

	// Renderer converts items into asset representations for display.
	Renderer ItemRenderer[T]

	// KeyBindings lists the key bindings for the asset viewer.
	KeyBindings []KeyBinding

	// NavMode specifies how users move between assets when browsing.
	NavMode NavigationMode

	// EnableSearch enables the search feature in the asset viewer.
	EnableSearch bool

	// UseMutex enables mutex-based synchronisation for safe concurrent access.
	UseMutex bool
}

// SetRenderer sets the item renderer for this panel.
//
// Takes renderer (ItemRenderer[T]) which renders items. Use this when the
// renderer needs a reference back to the panel to access panel state.
func (v *AssetViewer[T]) SetRenderer(renderer ItemRenderer[T]) {
	v.renderer = renderer
}

// SetItems replaces the current items with the given slice.
// This does not clear expansion state.
//
// Takes items ([]T) which is the new slice of items to display.
//
// Safe for concurrent use when the viewer was created with a mutex.
func (v *AssetViewer[T]) SetItems(items []T) {
	if v.mu != nil {
		v.mu.Lock()
		defer v.mu.Unlock()
	}
	v.items = items
	v.updateFilter()
}

// Items returns the current items.
//
// Returns []T which contains all items currently held by the viewer.
//
// Safe for concurrent use when a mutex is configured.
func (v *AssetViewer[T]) Items() []T {
	if v.mu != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
	}
	return v.items
}

// ItemCount returns the number of items in the viewer.
//
// Returns int which is the current count of items.
//
// Safe for concurrent use when the viewer was created with a mutex.
func (v *AssetViewer[T]) ItemCount() int {
	if v.mu != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
	}
	return len(v.items)
}

// GetDisplayItems returns the indices of items to display.
//
// Returns []int which contains the filtered indices if a search is active,
// or all indices if no search is set.
func (v *AssetViewer[T]) GetDisplayItems() []int {
	if v.search != nil {
		return v.search.GetDisplayIndices(len(v.items))
	}
	items := make([]int, len(v.items))
	for i := range v.items {
		items[i] = i
	}
	return items
}

// IsExpanded returns true if the item with the given ID is expanded.
//
// Takes id (string) which is the unique identifier of the item to check.
//
// Returns bool which indicates whether the item is currently expanded.
func (v *AssetViewer[T]) IsExpanded(id string) bool {
	return v.expanded[id]
}

// ToggleExpanded toggles the expansion state of the item with the given ID.
//
// Takes id (string) which identifies the item to expand or collapse.
func (v *AssetViewer[T]) ToggleExpanded(id string) {
	if v.expanded[id] {
		delete(v.expanded, id)
	} else {
		v.expanded[id] = true
	}
}

// SetExpanded sets the expansion state of the item with the given ID.
//
// Takes id (string) which identifies the item to expand or collapse.
// Takes expanded (bool) which specifies whether the item should be expanded.
func (v *AssetViewer[T]) SetExpanded(id string, expanded bool) {
	if expanded {
		v.expanded[id] = true
	} else {
		delete(v.expanded, id)
	}
}

// CollapseAll collapses all expanded items.
func (v *AssetViewer[T]) CollapseAll() {
	v.expanded = make(map[string]bool)
}

// ExpandedMap returns the expansion state map.
// Use this for preserving expansion state across refreshes.
//
// Returns map[string]bool which maps item identifiers to their expanded state.
func (v *AssetViewer[T]) ExpandedMap() map[string]bool {
	return v.expanded
}

// SetExpandedMap sets the expansion state map directly.
//
// Takes expanded (map[string]bool) which maps asset identifiers to their
// expanded state.
func (v *AssetViewer[T]) SetExpandedMap(expanded map[string]bool) {
	v.expanded = expanded
}

// Search returns the search mixin for the asset viewer.
//
// Returns *SearchMixin which provides search functionality, or nil if search
// is not enabled.
func (v *AssetViewer[T]) Search() *SearchMixin {
	return v.search
}

// Mutex returns the read-write mutex, or nil if not using mutex protection.
//
// Returns *sync.RWMutex which is the underlying lock for manual
// synchronisation.
func (v *AssetViewer[T]) Mutex() *sync.RWMutex {
	return v.mu
}

// updateFilter updates the filtered items list based on the current search query.
func (v *AssetViewer[T]) updateFilter() {
	if v.search == nil || v.renderer == nil {
		return
	}

	v.search.UpdateFilter(len(v.items), func(index int, query string) bool {
		if index >= len(v.items) {
			return false
		}
		return v.renderer.MatchesFilter(v.items[index], query)
	})

	displayItems := v.GetDisplayItems()
	if v.cursor >= len(displayItems) && len(displayItems) > 0 {
		v.cursor = len(displayItems) - 1
	}
}

// CalculateLineCount returns the total number of lines including expanded
// content.
//
// Returns int which is the line count for all display items.
func (v *AssetViewer[T]) CalculateLineCount() int {
	displayItems := v.GetDisplayItems()
	return CalculateLineCount(
		len(displayItems),
		func(index int) bool {
			if index >= len(displayItems) {
				return false
			}
			itemIndex := displayItems[index]
			if itemIndex >= len(v.items) {
				return false
			}
			id := v.renderer.GetID(v.items[itemIndex])
			return v.expanded[id]
		},
		func(index int) int {
			if index >= len(displayItems) {
				return 0
			}
			itemIndex := displayItems[index]
			if itemIndex >= len(v.items) {
				return 0
			}
			return v.renderer.ExpandedLineCount(v.items[itemIndex])
		},
	)
}

// NavigablePositions returns the line positions of all items that can be
// navigated to.
//
// Returns []int which contains the line numbers where navigation can stop.
func (v *AssetViewer[T]) NavigablePositions() []int {
	displayItems := v.GetDisplayItems()
	return NavigablePositions(
		len(displayItems),
		func(index int) bool {
			if index >= len(displayItems) {
				return false
			}
			itemIndex := displayItems[index]
			if itemIndex >= len(v.items) {
				return false
			}
			id := v.renderer.GetID(v.items[itemIndex])
			return v.expanded[id]
		},
		func(index int) int {
			if index >= len(displayItems) {
				return 0
			}
			itemIndex := displayItems[index]
			if itemIndex >= len(v.items) {
				return 0
			}
			return v.renderer.ExpandedLineCount(v.items[itemIndex])
		},
		v.navMode,
	)
}

// GetItemAtCursor returns the item at the current cursor position.
//
// Returns *T which is the item at the cursor, or nil if the cursor is out of
// bounds or points to a position that is not valid.
func (v *AssetViewer[T]) GetItemAtCursor() *T {
	displayItems := v.GetDisplayItems()
	positions := v.NavigablePositions()

	itemIndex := CursorToItemIndex(
		v.cursor,
		len(displayItems),
		func(index int) bool {
			if index >= len(displayItems) {
				return false
			}
			realIndex := displayItems[index]
			if realIndex >= len(v.items) {
				return false
			}
			id := v.renderer.GetID(v.items[realIndex])
			return v.expanded[id]
		},
		func(index int) int {
			if index >= len(displayItems) {
				return 0
			}
			realIndex := displayItems[index]
			if realIndex >= len(v.items) {
				return 0
			}
			return v.renderer.ExpandedLineCount(v.items[realIndex])
		},
		v.navMode,
	)

	_ = positions

	if itemIndex < 0 || itemIndex >= len(displayItems) {
		return nil
	}

	realIndex := displayItems[itemIndex]
	if realIndex >= len(v.items) {
		return nil
	}

	return &v.items[realIndex]
}

// HandleNavigation processes navigation key presses.
//
// Takes message (tea.KeyPressMsg) which contains the key press event to process.
//
// Returns bool which is true if the key was handled.
//
// Safe for concurrent use when the mutex is set.
func (v *AssetViewer[T]) HandleNavigation(message tea.KeyPressMsg) bool {
	if !v.focused {
		return false
	}

	if v.mu != nil {
		v.mu.Lock()
		defer v.mu.Unlock()
	}

	lineCount := v.CalculateLineCount()
	positions := v.NavigablePositions()

	newCursor, newOffset, handled := HandleNavigationKey(
		message,
		v.cursor,
		v.scrollOffset,
		v.ContentHeight(),
		positions,
		lineCount,
	)

	if handled {
		v.cursor = newCursor
		v.scrollOffset = newOffset
	}

	return handled
}

// HandleExpansionToggle handles enter or space key to toggle expansion.
//
// Returns bool which is true if expansion was toggled.
//
// Safe for concurrent use when the viewer has a mutex.
func (v *AssetViewer[T]) HandleExpansionToggle() bool {
	if v.mu != nil {
		v.mu.Lock()
		defer v.mu.Unlock()
	}

	item := v.GetItemAtCursor()
	if item == nil {
		return false
	}

	if !v.renderer.IsExpandable(*item) {
		return false
	}

	id := v.renderer.GetID(*item)
	v.ToggleExpanded(id)
	return true
}

// Update handles messages for the asset viewer.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns bool which is true if the message was handled.
// Returns tea.Cmd which is a command to run, or nil if none.
func (v *AssetViewer[T]) Update(message tea.Msg) (bool, tea.Cmd) {
	if v.search != nil && v.search.IsActive() {
		return v.search.Update(message)
	}

	keyMessage, ok := message.(tea.KeyPressMsg)
	if !ok {
		return false, nil
	}

	if v.HandleNavigation(keyMessage) {
		return true, nil
	}

	switch keyMessage.String() {
	case "enter", "space":
		if v.HandleExpansionToggle() {
			return true, nil
		}
	case "esc":
		if v.search != nil && v.search.HasQuery() {
			v.search.ClearQuery()
			v.updateFilter()
			return true, nil
		}
		if len(v.expanded) > 0 {
			v.CollapseAll()
			return true, nil
		}
	}

	if v.search != nil {
		handled, command := v.search.HandleKey(keyMessage, v.ContentWidth())
		if handled {
			return true, command
		}
	}

	return false, nil
}

// RenderHeader writes the header section to the given builder.
//
// Takes content (*strings.Builder) which receives the rendered header.
//
// Returns int which is the number of lines written.
func (v *AssetViewer[T]) RenderHeader(content *strings.Builder) int {
	usedLines := 0

	if v.search != nil {
		usedLines += v.search.RenderHeader(content, len(v.items))
	}

	return usedLines
}

// RenderItems writes all items to the output, handling scroll position.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes headerLines (int) which specifies how many header lines to exclude
// from the content height calculation.
func (v *AssetViewer[T]) RenderItems(content *strings.Builder, headerLines int) {
	displayItems := v.GetDisplayItems()
	if len(displayItems) == 0 {
		return
	}

	ctx := NewScrollContext(content, v.scrollOffset, v.ContentHeight()-headerLines)

	for _, itemIndex := range displayItems {
		if itemIndex >= len(v.items) {
			continue
		}

		item := v.items[itemIndex]
		id := v.renderer.GetID(item)
		expanded := v.expanded[id]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == v.cursor

		ctx.WriteLineIfVisible(func() string {
			return v.renderer.RenderRow(item, lineIndex, selected, v.focused, v.ContentWidth())
		})

		if expanded {
			details := v.renderer.RenderExpanded(item, v.ContentWidth())
			for _, detail := range details {
				detailLineIndex := ctx.LineIndex()
				detailSelected := detailLineIndex == v.cursor && v.navMode == NavigationSimple
				_ = detailSelected
				ctx.WriteLine(detail)
			}
		}
	}
}

// ViewCallbacks holds the per-panel render functions for the RenderViewWith
// template method.
type ViewCallbacks struct {
	// RenderHeader writes the panel header and returns the number of lines used.
	RenderHeader func(w *strings.Builder) int

	// RenderEmptyState writes the empty-state placeholder.
	RenderEmptyState func(w *strings.Builder)

	// RenderItems writes the item list.
	RenderItems func(w *strings.Builder, displayItems []int, usedLines int)

	// TrimTrailingNewline controls whether a trailing newline is stripped
	// before framing. Set to true for panels whose item renderers emit a
	// trailing newline that would otherwise produce a blank bottom row.
	TrimTrailingNewline bool
}

// RenderViewWith is a template method that encapsulates the common View()
// pattern shared by all panels. It handles size setup, search-mixin width,
// mutex locking, header/empty/item rendering, and framing.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
// Takes callbacks (ViewCallbacks) which provides the
// panel-specific render functions.
//
// Returns string which contains the rendered panel content.
//
// Safe for concurrent use when the viewer has a mutex set.
func (v *AssetViewer[T]) RenderViewWith(width, height int, callbacks ViewCallbacks) string {
	v.SetSize(width, height)

	if v.search != nil {
		v.search.SetWidth(v.ContentWidth())
	}

	if v.mu != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
	}

	var content strings.Builder
	usedLines := callbacks.RenderHeader(&content)

	displayItems := v.GetDisplayItems()
	if len(displayItems) == 0 {
		callbacks.RenderEmptyState(&content)
		return v.RenderFrame(content.String())
	}

	callbacks.RenderItems(&content, displayItems, usedLines)

	result := content.String()
	if callbacks.TrimTrailingNewline {
		result = strings.TrimSuffix(result, stringNewline)
	}
	return v.RenderFrame(result)
}

// View renders the full panel view.
// Override this in panels that need custom rendering.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
//
// Safe for concurrent use when the viewer has a mutex set.
func (v *AssetViewer[T]) View(width, height int) string {
	v.SetSize(width, height)

	if v.search != nil {
		v.search.SetWidth(v.ContentWidth())
	}

	if v.mu != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
	}

	var content strings.Builder
	headerLines := v.RenderHeader(&content)

	displayItems := v.GetDisplayItems()
	if len(displayItems) == 0 {
		itemName := "items"
		hasFilter := v.search != nil && v.search.HasQuery()
		RenderEmptyState(&content, hasFilter, itemName)
		return v.RenderFrame(content.String())
	}

	v.RenderItems(&content, headerLines)
	return v.RenderFrame(strings.TrimSuffix(content.String(), stringNewline))
}

// Init returns nil as no initialisation is needed.
//
// Returns tea.Cmd which is nil.
func (*AssetViewer[T]) Init() tea.Cmd {
	return nil
}

// NewAssetViewer creates a new asset viewer with the given configuration.
//
// Takes config (AssetViewerConfig[T]) which provides the viewer
// settings including ID, title, renderer, navigation mode, and
// optional search and mutex support.
//
// Returns *AssetViewer[T] which is the configured viewer ready
// for use.
func NewAssetViewer[T any](config AssetViewerConfig[T]) *AssetViewer[T] {
	v := &AssetViewer[T]{
		BasePanel: NewBasePanel(config.ID, config.Title),
		items:     make([]T, 0),
		expanded:  make(map[string]bool),
		renderer:  config.Renderer,
		navMode:   config.NavMode,
		search:    nil,
		mu:        nil,
	}

	if config.EnableSearch {
		v.search = NewSearchMixin(v.updateFilter)
	}

	if config.UseMutex {
		v.mu = &sync.RWMutex{}
	}

	if len(config.KeyBindings) > 0 {
		v.SetKeyMap(config.KeyBindings)
	}

	return v
}
