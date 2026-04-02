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
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/wdk/clock"
)

// DefaultRefreshTimeout is the default timeout for panel refresh operations.
const DefaultRefreshTimeout = 5 * time.Second

// PanelState holds common state that most panels need for error handling
// and refresh tracking. Embed this in panel structs to reduce boilerplate.
type PanelState struct {
	// LastRefresh is the time of the last refresh, whether it succeeded or failed.
	LastRefresh time.Time

	// Err holds the last error from a refresh, or nil if successful.
	Err error

	// clock provides time operations; nil uses the real clock.
	clock clock.Clock

	// mu guards concurrent access to the panel state fields.
	mu sync.RWMutex
}

// NewPanelState creates a new PanelState with the given clock.
//
// If clock is nil, defaults to the real system clock.
//
// Takes c (clock.Clock) which provides time functionality.
//
// Returns PanelState which is the initialised panel state.
func NewPanelState(c clock.Clock) PanelState {
	if c == nil {
		c = clock.RealClock()
	}
	return PanelState{
		LastRefresh: time.Time{},
		Err:         nil,
		clock:       c,
		mu:          sync.RWMutex{},
	}
}

// Clock returns the panel's clock for time operations.
//
// Returns clock.Clock which provides time functions, defaulting to the real
// clock when none is set.
func (s *PanelState) Clock() clock.Clock {
	if s.clock == nil {
		return clock.RealClock()
	}
	return s.clock
}

// Now returns the current time from the panel's clock.
//
// Returns time.Time which is the current time according to the panel's clock.
func (s *PanelState) Now() time.Time {
	return s.Clock().Now()
}

// Since returns the duration since t using the panel's clock.
//
// Takes t (time.Time) which is the start time to measure from.
//
// Returns time.Duration which is the elapsed time since t.
func (s *PanelState) Since(t time.Time) time.Duration {
	return s.Clock().Now().Sub(t)
}

// Lock acquires the write lock on the panel state.
//
// Safe for concurrent use. Callers must call Unlock when done.
func (s *PanelState) Lock() {
	s.mu.Lock()
}

// Unlock releases the write lock on the panel state.
//
// Safe for concurrent use. Call this only after a successful Lock call.
func (s *PanelState) Unlock() {
	s.mu.Unlock()
}

// RLock acquires the read lock on the panel state.
//
// Safe for concurrent use. Call RUnlock to release the lock.
func (s *PanelState) RLock() {
	s.mu.RLock()
}

// RUnlock releases the read lock on the panel state.
//
// Safe for concurrent use. Must be called after a successful RLock call.
func (s *PanelState) RUnlock() {
	s.mu.RUnlock()
}

// UpdateError sets the error state and updates the last refresh time.
//
// Takes err (error) which is the error to store, or nil to clear the error.
func (s *PanelState) UpdateError(err error) {
	s.Err = err
	s.LastRefresh = s.Now()
}

// UpdateSuccess clears any error and updates the last refresh time.
func (s *PanelState) UpdateSuccess() {
	s.Err = nil
	s.LastRefresh = s.Now()
}

// GetError returns the current error state.
//
// Returns error when the panel has encountered a failure.
//
// Safe for concurrent use.
func (s *PanelState) GetError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Err
}

// GetLastRefresh returns the last refresh time.
//
// Returns time.Time which is the timestamp of the most recent refresh.
//
// Safe for concurrent use.
func (s *PanelState) GetLastRefresh() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastRefresh
}

// KeyResult holds the outcome of a key handler, indicating whether the key was
// processed and any command to run as a result.
type KeyResult struct {
	// Cmd holds the command to run after handling the key, or nil if none.
	Cmd tea.Cmd

	// Handled indicates whether a handler processed the key.
	Handled bool
}

// RefreshConfig configures the BuildRefreshCmd helper.
type RefreshConfig struct {
	// NoProvider returns the message to send when the provider is nil. This is
	// usually an error message for the panel type.
	NoProvider func() tea.Msg

	// Fetch performs the data fetch and returns the result message. The
	// context has the configured timeout applied.
	Fetch func(ctx context.Context) tea.Msg

	// Timeout is the maximum time allowed for the refresh operation.
	// Defaults to DefaultRefreshTimeout if zero.
	Timeout time.Duration
}

// ViewBuilder helps build panel views with a consistent structure.
// It manages the content buffer, header line counting, and mutex handling.
type ViewBuilder struct {
	// panel is the base panel used for sizing and rendering frames.
	panel *BasePanel

	// search provides optional filtering for panel content; nil means no filtering.
	search *SearchMixin

	// viewerMutex protects concurrent access during view rendering;
	// nil means no locking.
	viewerMutex *sync.RWMutex

	// content holds the rendered panel text before final framing is applied.
	content strings.Builder

	// headerLines counts the header lines added to the view so far.
	headerLines int
}

// NewViewBuilder creates a new view builder for the given panel.
// Pass the search mixin and viewer mutex if the panel uses them, or nil if not.
//
// Takes panel (*BasePanel) which specifies the panel to build views for.
// Takes search (*SearchMixin) which provides search functionality, or nil.
// Takes viewerMutex (*sync.RWMutex) which protects viewer access, or nil.
//
// Returns *ViewBuilder which is ready to build view content.
func NewViewBuilder(panel *BasePanel, search *SearchMixin, viewerMutex *sync.RWMutex) *ViewBuilder {
	return &ViewBuilder{
		content:     strings.Builder{},
		panel:       panel,
		search:      search,
		viewerMutex: viewerMutex,
		headerLines: 0,
	}
}

// SetupView sets up the view with standard settings: panel size and search
// width.
//
// Takes width (int) which specifies the panel width.
// Takes height (int) which specifies the panel height.
func (vb *ViewBuilder) SetupView(width, height int) {
	vb.panel.SetSize(width, height)
	if vb.search != nil {
		vb.search.SetWidth(vb.panel.ContentWidth())
	}
}

// WithReadLock executes the render function with the viewer mutex held.
// If no mutex was provided, it executes the function directly.
//
// Takes renderFunction (func()) which is the function to execute while holding
// the read lock.
//
// Safe for concurrent use. Acquires a read lock for the duration of the
// render function call.
func (vb *ViewBuilder) WithReadLock(renderFunction func()) {
	if vb.viewerMutex != nil {
		vb.viewerMutex.RLock()
		defer vb.viewerMutex.RUnlock()
	}
	renderFunction()
}

// RenderSearchHeader renders the search header if search is enabled.
//
// Takes totalCount (int) which is the total number of search results.
//
// Returns int which is the number of header lines added.
func (vb *ViewBuilder) RenderSearchHeader(totalCount int) int {
	if vb.search == nil {
		return 0
	}
	lines := vb.search.RenderHeader(&vb.content, totalCount)
	vb.headerLines += lines
	return lines
}

// RenderErrorState renders an error message if the error is not nil.
//
// Takes err (error) which is the error to render, or nil if there is no error.
//
// Returns bool which is true if an error was rendered, false if err was nil.
func (vb *ViewBuilder) RenderErrorState(err error) bool {
	if err == nil {
		return false
	}
	RenderErrorState(&vb.content, err)
	vb.headerLines++
	return true
}

// Content returns the content builder for custom rendering.
//
// Returns *strings.Builder which allows direct access to the view content.
func (vb *ViewBuilder) Content() *strings.Builder {
	return &vb.content
}

// HeaderLines returns the total number of header lines used so far.
//
// Returns int which is the count of header lines added to the view.
func (vb *ViewBuilder) HeaderLines() int {
	return vb.headerLines
}

// AddHeaderLines increases the header line count by the given amount.
//
// Takes count (int) which specifies how many header lines to add.
func (vb *ViewBuilder) AddHeaderLines(count int) {
	vb.headerLines += count
}

// Finish completes the view and returns the rendered frame.
// This trims any trailing newline and applies the panel frame.
//
// Returns string which is the final rendered view with panel framing applied.
func (vb *ViewBuilder) Finish() string {
	return vb.panel.RenderFrame(strings.TrimSuffix(vb.content.String(), stringNewline))
}

// Panel returns the underlying BasePanel.
//
// Returns *BasePanel which provides access to the panel configuration.
func (vb *ViewBuilder) Panel() *BasePanel {
	return vb.panel
}

// ContentWidth returns the panel's content width.
//
// Returns int which is the width available for content in characters.
func (vb *ViewBuilder) ContentWidth() int {
	return vb.panel.ContentWidth()
}

// ContentHeight returns the height of the panel's content area in rows.
//
// Returns int which is the content height.
func (vb *ViewBuilder) ContentHeight() int {
	return vb.panel.ContentHeight()
}

// RenderExpandableItemsConfig configures the RenderExpandableItems helper.
// It reduces boilerplate in panel render methods by handling the common
// loop structure.
type RenderExpandableItemsConfig[T any] struct {
	// Ctx is the scroll context used for writing lines.
	Ctx *ScrollContext

	// GetID returns the unique identifier for an item, used for expansion lookup.
	GetID func(T) string

	// IsExpanded checks whether the given ID is currently expanded.
	IsExpanded func(string) bool

	// RenderRow renders a single item row. It receives the item, its selected
	// state, and its expanded state, and returns the formatted row string.
	RenderRow func(T, bool, bool) string

	// RenderExpand renders expanded content for an item. May be nil
	// to skip expansion rendering.
	RenderExpand func(*ScrollContext, T)

	// Items is the complete list of items to render.
	Items []T

	// DisplayItems is the list of visible item indices after filtering.
	DisplayItems []int

	// Cursor is the current position in the list.
	Cursor int
}

// Handled returns a KeyResult that shows the key was processed with no
// command.
//
// Returns KeyResult which shows that a key press was handled and needs no
// further action.
func Handled() KeyResult {
	return KeyResult{Handled: true, Cmd: nil}
}

// HandledWithCmd returns a KeyResult showing the key was handled with a
// command.
//
// Takes command (tea.Cmd) which is the command to run after handling.
//
// Returns KeyResult which shows the key was handled and includes the command.
func HandledWithCmd(command tea.Cmd) KeyResult {
	return KeyResult{Handled: true, Cmd: command}
}

// NotHandled returns a KeyResult that shows the key was not handled.
//
// Returns KeyResult which signals that the key event was not processed.
func NotHandled() KeyResult {
	return KeyResult{Handled: false, Cmd: nil}
}

// HandleCommonKeys processes key events that are shared by most panels.
//
// This includes navigation (j/k, arrows, g/G, pgup/pgdown), expansion
// (enter, space), search (/, esc), and refresh (r).
//
// Panels should call this first in their key handler, then handle
// panel-specific keys if the result is NotHandled.
//
// Takes viewer (*AssetViewer[T]) which is the panel to handle keys for.
// Takes message (tea.KeyPressMsg) which is the key event to process.
// Takes refreshFunction (func() tea.Cmd) which is the panel's refresh function.
// Pass nil to disable the 'r' key refresh.
//
// Returns KeyResult which shows whether the key was handled.
func HandleCommonKeys[T any](
	viewer *AssetViewer[T],
	message tea.KeyPressMsg,
	refreshFunction func() tea.Cmd,
) KeyResult {
	if viewer.HandleNavigation(message) {
		return Handled()
	}

	switch message.String() {
	case "enter", "space":
		if viewer.HandleExpansionToggle() {
			return Handled()
		}

	case "/":
		if viewer.Search() != nil {
			viewer.Search().SetWidth(viewer.ContentWidth())
			return HandledWithCmd(viewer.Search().SearchBox().Open())
		}

	case "esc":
		if viewer.Search() != nil && viewer.Search().HasQuery() {
			viewer.Search().ClearQuery()
			return Handled()
		}
		if len(viewer.ExpandedMap()) > 0 {
			viewer.CollapseAll()
			return Handled()
		}

	case "r":
		if refreshFunction != nil {
			return HandledWithCmd(refreshFunction())
		}
	}

	return NotHandled()
}

// BuildRefreshCmd creates a standard refresh command with timeout handling.
// This reduces repeated code in panel refresh methods.
//
// Takes config (RefreshConfig) which specifies the fetch function, timeout,
// and fallback behaviour when no provider is set.
//
// Returns tea.Cmd which wraps the fetch operation with timeout handling.
func BuildRefreshCmd(config RefreshConfig) tea.Cmd {
	return func() tea.Msg {
		if config.Fetch == nil {
			return config.NoProvider()
		}

		timeout := config.Timeout
		if timeout == 0 {
			timeout = DefaultRefreshTimeout
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), timeout,
			fmt.Errorf("lifecycle panel data fetch exceeded %s timeout", timeout))
		defer cancel()

		return config.Fetch(ctx)
	}
}

// RenderExpandableItems renders a scrollable list of items that can be
// expanded. It loops through the display items, checks if each one is selected
// or expanded, and draws rows with optional expanded content.
//
// Takes config (RenderExpandableItemsConfig[T]) which provides all settings
// for rendering, including items, display indices, cursor position, and
// callbacks for drawing rows and expanded content.
func RenderExpandableItems[T any](config RenderExpandableItemsConfig[T]) {
	for _, itemIndex := range config.DisplayItems {
		if itemIndex >= len(config.Items) {
			continue
		}

		item := config.Items[itemIndex]
		lineIndex := config.Ctx.LineIndex()
		selected := lineIndex == config.Cursor
		itemID := config.GetID(item)
		expanded := config.IsExpanded(itemID)

		config.Ctx.WriteLineIfVisible(func() string {
			return config.RenderRow(item, selected, expanded)
		})

		if expanded && config.RenderExpand != nil {
			config.RenderExpand(config.Ctx, item)
		}
	}
}
