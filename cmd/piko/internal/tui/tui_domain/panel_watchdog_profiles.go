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

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// WatchdogProfilesPanelID identifies the watchdog Profiles panel.
	WatchdogProfilesPanelID = "watchdog-profiles"

	// WatchdogProfilesPanelTitle is the display title.
	WatchdogProfilesPanelTitle = "Watchdog Profiles"

	// profilesAllTypeLabel is the "no type filter" sentinel shown in
	// the type filter cycle.
	profilesAllTypeLabel = "all"
)

// profileSortMode selects which column the profile table is ordered by.
type profileSortMode int

const (
	// profileSortAgeDesc orders profiles newest-first by capture timestamp.
	profileSortAgeDesc profileSortMode = iota

	// profileSortType orders profiles alphabetically by type.
	profileSortType

	// profileSortSizeDesc orders profiles largest-first by size.
	profileSortSizeDesc

	// profileSortFilename orders profiles alphabetically by file name.
	profileSortFilename
)

// profilesSnapshotMsg carries a refreshed profile list from the provider.
// Err is non-nil when the fetch failed.
type profilesSnapshotMsg struct {
	// Err is the fetch error, or nil on success.
	Err error

	// Profiles is the latest profile inventory.
	Profiles []WatchdogProfile
}

// profilesPruneCompletedMsg notifies the panel that a prune RPC
// completed.
type profilesPruneCompletedMsg struct {
	// Err is the prune error, or nil on success.
	Err error

	// ProfileType is the type that was pruned.
	ProfileType string

	// Removed is the number of profiles removed.
	Removed int
}

// WatchdogProfilesPanel renders the inventory of stored profile artefacts
// with sort, type-filter, and prune actions.
type WatchdogProfilesPanel struct {
	// provider supplies the profile inventory.
	provider WatchdogProvider

	// clock yields the current time for age calculations.
	clock clock.Clock

	// lastFetchErr is the most recent fetch error, or nil after success.
	lastFetchErr error

	// theme is the active theme used to render styles.
	theme *Theme

	// typeFilter constrains the visible inventory to a single profile type.
	typeFilter string

	// profiles is the cached profile inventory.
	profiles []WatchdogProfile

	BasePanel

	// sortMode is the currently active sort mode.
	sortMode profileSortMode

	// mu guards lastFetchErr and profiles.
	mu sync.RWMutex

	// pruneInFlight is true while a prune RPC is awaiting completion.
	pruneInFlight bool
}

// Compile-time interface assertions.
var (
	_ Panel = (*WatchdogProfilesPanel)(nil)

	_ ThemeAware = (*WatchdogProfilesPanel)(nil)
)

// NewWatchdogProfilesPanel constructs the Profiles panel.
//
// Takes provider (WatchdogProvider) which supplies the profile list.
// Takes clk (clock.Clock) which yields the current time.
//
// Returns *WatchdogProfilesPanel ready for AddPanel.
func NewWatchdogProfilesPanel(provider WatchdogProvider, clk clock.Clock) *WatchdogProfilesPanel {
	if clk == nil {
		clk = clock.RealClock()
	}
	panel := &WatchdogProfilesPanel{
		BasePanel: NewBasePanel(WatchdogProfilesPanelID, WatchdogProfilesPanelTitle),
		provider:  provider,
		clock:     clk,
	}
	panel.SetKeyMap([]KeyBinding{
		{Key: "j / Down", Description: "Next profile"},
		{Key: "k / Up", Description: "Previous profile"},
		{Key: "g", Description: "Top"},
		{Key: "G", Description: "Bottom"},
		{Key: "s", Description: "Cycle sort"},
		{Key: "t", Description: "Cycle type filter"},
		{Key: "P", Description: "Prune current type"},
		{Key: "R", Description: "Refresh"},
	})
	return panel
}

// SetTheme implements ThemeAware.
//
// Takes theme (*Theme) which becomes the active theme.
func (p *WatchdogProfilesPanel) SetTheme(theme *Theme) { p.theme = theme }

// Init kicks off the first profile-list fetch.
//
// Returns tea.Cmd which schedules the first fetch.
func (p *WatchdogProfilesPanel) Init() tea.Cmd {
	return p.fetchCmd()
}

// Update handles tick, snapshot, prune-completion, and key messages.
//
// Takes message (tea.Msg) which is the routed message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProfilesPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	switch msg := message.(type) {
	case profilesSnapshotMsg:
		p.mu.Lock()
		if msg.Err == nil {
			p.profiles = msg.Profiles
		}
		p.lastFetchErr = msg.Err
		p.mu.Unlock()
		p.clampCursor()
	case profilesPruneCompletedMsg:
		p.pruneInFlight = false
		cmd := p.fetchCmd()
		return p, cmd
	case TickMessage:
		cmd := p.fetchCmd()
		return p, cmd
	case tea.KeyPressMsg:
		if cmd := p.handleKey(msg); cmd != nil {
			return p, cmd
		}
	}
	return p, nil
}

// View renders the panel sized to the supplied dimensions.
//
// Takes width (int) which is the column width for the rendered frame.
// Takes height (int) which is the row height for the rendered frame.
//
// Returns string with the rendered frame.
func (p *WatchdogProfilesPanel) View(width, height int) string {
	p.SetSize(width, height)

	cw := p.ContentWidth()
	ch := p.ContentHeight()
	if cw <= 0 || ch <= 0 {
		return p.RenderFrame("")
	}
	body := p.composeBody(cw, ch)
	return p.RenderFrame(body)
}

// composeBody arranges the header, table, and footer.
//
// Takes width (int) which is the body width.
// Takes height (int) which is the body height.
//
// Returns string with the assembled body.
func (p *WatchdogProfilesPanel) composeBody(width, height int) string {
	rows := make([]string, 0, height)
	rows = append(rows, p.renderHeaderRow(width), "", p.renderTableHeader(width))

	listHeight := max(height-4, 1)
	rows = append(rows, p.renderRows(width, listHeight)...)

	for len(rows) < height-1 {
		rows = append(rows, strings.Repeat(" ", width))
	}
	rows = append(rows, p.renderFooterRow(width))

	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// renderHeaderRow shows the panel title and aggregate counts.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered header row.
func (p *WatchdogProfilesPanel) renderHeaderRow(width int) string {
	visible := p.visibleProfiles()
	totalSize := uint64(0)
	for _, prof := range visible {
		totalSize += safeconv.Int64ToUint64(prof.SizeBytes)
	}

	left := p.boldStyle().Render(WatchdogProfilesPanelTitle)
	right := p.dimStyle().Render(fmt.Sprintf("%d profiles · %s · type: %s · sort: %s",
		len(visible), inspector.FormatBytes(totalSize), p.activeTypeFilterLabel(), p.sortModeLabel()))

	leftWidth := TextWidth(left)
	rightWidth := TextWidth(right)
	gap := max(1, width-leftWidth-rightWidth)
	return PadRightANSI(left+strings.Repeat(" ", gap)+right, width)
}

// renderTableHeader returns the AGE / TYPE / SIZE / SIDECAR /
// FILENAME header row.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered header row.
func (p *WatchdogProfilesPanel) renderTableHeader(width int) string {
	cols := p.columnLayout(width)
	header := p.dimStyle().Render(p.composeColumns([]string{"AGE", "TYPE", "SIZE", "SIDECAR", "FILENAME"}, cols))
	return PadRightANSI(header, width)
}

// renderRows returns up to height rows of the visible, sorted, filtered
// profile inventory.
//
// Takes width (int) which is the row width.
// Takes height (int) which is the maximum number of rows to return.
//
// Returns []string which is the rendered list rows.
func (p *WatchdogProfilesPanel) renderRows(width, height int) []string {
	profiles := p.visibleProfiles()
	if len(profiles) == 0 {
		hint := "No profiles match the current filter."
		if len(p.snapshotProfiles()) == 0 {
			hint = "No profiles stored yet."
		}
		return []string{p.dimStyle().Render(PadRightANSI(hint, width))}
	}

	cursor := p.Cursor()
	if cursor >= len(profiles) {
		cursor = len(profiles) - 1
		p.SetCursor(cursor)
	}

	cols := p.columnLayout(width)

	startIdx := p.ScrollOffset()
	endIdx := min(startIdx+height, len(profiles))

	now := p.clock.Now()
	rows := make([]string, 0, height)
	for i := startIdx; i < endIdx; i++ {
		prof := profiles[i]
		age := inspector.FormatTimeSince(now, prof.Timestamp)
		size := prof.DisplaySize()
		sidecar := "—"
		if prof.HasSidecar {
			sidecar = "✓"
		}
		row := p.composeColumns([]string{age, prof.Type, size, sidecar, prof.Filename}, cols)
		if i == cursor {
			row = p.cursorStyle().Render(PadRightANSI(row, width))
		}
		rows = append(rows, PadRightANSI(row, width))
	}
	return rows
}

// renderFooterRow shows the in-flight prune indicator and a key hint.
//
// Takes width (int) which is the row width.
//
// Returns string with the rendered footer row.
func (p *WatchdogProfilesPanel) renderFooterRow(width int) string {
	if p.pruneInFlight {
		return PadRightANSI(p.warningStyle().Render("Prune in progress…"), width)
	}
	hint := "press s to sort · t to filter · P to prune the active type · R to refresh"
	return PadRightANSI(p.dimStyle().Render(hint), width)
}

// columnLayout computes the per-column widths for the table.
//
// Takes width (int) which is the total row width.
//
// Returns []int which is the per-column widths in order.
func (*WatchdogProfilesPanel) columnLayout(width int) []int {
	const profilesColumnGutter = 4
	cols := []int{
		ProfilesColAge,
		ProfilesColType,
		ProfilesColSize,
		ProfilesColSidecar,
		max(ProfilesColFilename, width-ProfilesColAge-ProfilesColType-ProfilesColSize-ProfilesColSidecar-profilesColumnGutter),
	}
	return cols
}

// composeColumns formats a sequence of cells into a single padded row.
//
// Takes cells ([]string) and widths ([]int); their lengths must match.
//
// Returns string which is the formatted row.
func (*WatchdogProfilesPanel) composeColumns(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		w := 0
		if i < len(widths) {
			w = widths[i]
		}
		if w <= 0 {
			parts[i] = cell
			continue
		}
		parts[i] = PadRightANSI(TruncateANSI(cell, w-1), w)
	}
	return strings.Join(parts, " ")
}

// visibleProfiles returns the profiles after sorting and filtering.
//
// Returns []WatchdogProfile which is the filtered, sorted slice.
func (p *WatchdogProfilesPanel) visibleProfiles() []WatchdogProfile {
	all := p.snapshotProfiles()
	out := all[:0:len(all)]
	for _, prof := range all {
		if p.typeFilter != "" && p.typeFilter != profilesAllTypeLabel && prof.Type != p.typeFilter {
			continue
		}
		out = append(out, prof)
	}

	switch p.sortMode {
	case profileSortAgeDesc:
		slices.SortStableFunc(out, func(a, b WatchdogProfile) int {
			return b.Timestamp.Compare(a.Timestamp)
		})
	case profileSortType:
		slices.SortStableFunc(out, func(a, b WatchdogProfile) int {
			return cmp.Or(cmp.Compare(a.Type, b.Type), b.Timestamp.Compare(a.Timestamp))
		})
	case profileSortSizeDesc:
		slices.SortStableFunc(out, func(a, b WatchdogProfile) int {
			return cmp.Compare(b.SizeBytes, a.SizeBytes)
		})
	case profileSortFilename:
		slices.SortStableFunc(out, func(a, b WatchdogProfile) int {
			return cmp.Compare(a.Filename, b.Filename)
		})
	}
	return out
}

// snapshotProfiles returns a copy of the cached profiles slice.
//
// Returns []WatchdogProfile which is a snapshot copy.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (p *WatchdogProfilesPanel) snapshotProfiles() []WatchdogProfile {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]WatchdogProfile, len(p.profiles))
	copy(out, p.profiles)
	return out
}

// activeTypeFilterLabel returns the human-readable type filter.
//
// Returns string which is the active filter, or "all" when no filter is set.
func (p *WatchdogProfilesPanel) activeTypeFilterLabel() string {
	if p.typeFilter == "" {
		return profilesAllTypeLabel
	}
	return p.typeFilter
}

// sortModeLabel returns the human-readable sort mode.
//
// Returns string which is the active sort mode's label.
func (p *WatchdogProfilesPanel) sortModeLabel() string {
	switch p.sortMode {
	case profileSortAgeDesc:
		return "age desc"
	case profileSortType:
		return "type"
	case profileSortSizeDesc:
		return "size desc"
	case profileSortFilename:
		return "filename"
	default:
		return ""
	}
}

// fetchCmd asks the provider for a fresh profile inventory.
//
// Returns tea.Cmd which produces a profilesSnapshotMsg, or nil when no
// provider is configured.
func (p *WatchdogProfilesPanel) fetchCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, errors.New("watchdog profiles fetch timed out"))
		defer cancel()
		profiles, err := p.provider.ListProfiles(ctx)
		return profilesSnapshotMsg{Profiles: profiles, Err: err}
	}
}

// pruneCmd issues a prune RPC for the active type filter.
//
// Returns tea.Cmd which produces a profilesPruneCompletedMsg, or nil when
// no provider is configured.
func (p *WatchdogProfilesPanel) pruneCmd() tea.Cmd {
	if p.provider == nil {
		return nil
	}
	pruneType := p.typeFilter
	if pruneType == profilesAllTypeLabel {
		pruneType = ""
	}
	p.pruneInFlight = true
	return func() tea.Msg {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 30*time.Second, errors.New("watchdog prune timed out"))
		defer cancel()
		removed, err := p.provider.PruneProfiles(ctx, pruneType)
		return profilesPruneCompletedMsg{ProfileType: pruneType, Removed: removed, Err: err}
	}
}

// handleKey processes panel-specific keys.
//
// Takes message (tea.KeyPressMsg) which is the key event.
//
// Returns tea.Cmd which schedules any follow-up command, or nil.
func (p *WatchdogProfilesPanel) handleKey(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "s":
		p.cycleSort()
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return nil
	case "t":
		p.cycleTypeFilter()
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return nil
	case "P":
		if p.pruneInFlight {
			return nil
		}
		return p.pruneCmd()
	case "R":
		return p.fetchCmd()
	}
	if p.HandleNavigation(message, len(p.visibleProfiles())) {
		return nil
	}
	return nil
}

// cycleSort advances through the available sort modes.
func (p *WatchdogProfilesPanel) cycleSort() {
	switch p.sortMode {
	case profileSortAgeDesc:
		p.sortMode = profileSortType
	case profileSortType:
		p.sortMode = profileSortSizeDesc
	case profileSortSizeDesc:
		p.sortMode = profileSortFilename
	default:
		p.sortMode = profileSortAgeDesc
	}
}

// cycleTypeFilter advances through the distinct types present in the
// snapshot, with a leading "all" entry.
func (p *WatchdogProfilesPanel) cycleTypeFilter() {
	types := p.distinctTypes()
	choices := append([]string{profilesAllTypeLabel}, types...)
	current := p.activeTypeFilterLabel()
	for i, c := range choices {
		if c == current {
			next := choices[(i+1)%len(choices)]
			if next == profilesAllTypeLabel {
				p.typeFilter = ""
			} else {
				p.typeFilter = next
			}
			return
		}
	}
	p.typeFilter = ""
}

// distinctTypes returns the unique profile types in alphabetical order.
//
// Returns []string which is the sorted list of unique profile types.
func (p *WatchdogProfilesPanel) distinctTypes() []string {
	const distinctTypesInitialCapacity = 8
	all := p.snapshotProfiles()
	seen := make(map[string]struct{}, distinctTypesInitialCapacity)
	out := make([]string, 0, distinctTypesInitialCapacity)
	for _, prof := range all {
		if _, ok := seen[prof.Type]; ok {
			continue
		}
		seen[prof.Type] = struct{}{}
		out = append(out, prof.Type)
	}
	slices.Sort(out)
	return out
}

// clampCursor ensures the cursor stays within the visible range when
// the underlying data shrinks.
func (p *WatchdogProfilesPanel) clampCursor() {
	visible := len(p.visibleProfiles())
	cursor := p.Cursor()
	if visible == 0 {
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return
	}
	if cursor >= visible {
		p.SetCursor(visible - 1)
	}
}

// boldStyle returns the bold-text style.
//
// Returns lipgloss.Style which is the themed bold style or a fallback.
func (p *WatchdogProfilesPanel) boldStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Bold
	}
	return lipgloss.NewStyle().Bold(true)
}

// dimStyle returns the dim-text style.
//
// Returns lipgloss.Style which is the themed dim style or a fallback.
func (p *WatchdogProfilesPanel) dimStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Dim
	}
	return statusUnknownStyle
}

// warningStyle returns the warning style.
//
// Returns lipgloss.Style which is the themed warning style or a fallback.
func (p *WatchdogProfilesPanel) warningStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.StatusDegraded
	}
	return statusDegradedStyle
}

// cursorStyle returns the selection style.
//
// Returns lipgloss.Style which is the themed selection style or a fallback.
func (p *WatchdogProfilesPanel) cursorStyle() lipgloss.Style {
	if p.theme != nil {
		return p.theme.Selected
	}
	return navItemActiveStyle
}
