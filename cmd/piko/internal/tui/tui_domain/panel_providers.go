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

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/wdk/clock"
)

const (
	// providersRefreshTimeout caps the providers RPC fetch.
	providersRefreshTimeout = 5 * time.Second
)

// providersRefreshMessage carries the result of a ListProviders fetch
// back to the panel.
type providersRefreshMessage struct {
	// err is the RPC error, or nil on success.
	err error

	// entries is the list of provider entries returned by the inspector.
	entries []ProviderEntry
}

// providersDescribeMessage carries a per-row describe result back to
// the panel for detail rendering.
type providersDescribeMessage struct {
	// detail is the cached describe-provider response, or nil on error.
	detail *ProviderDetail

	// err is the RPC error, or nil on success.
	err error

	// row is the "type/name" key identifying the row.
	row string
}

// ProvidersPanel surfaces the provider registry.
//
// It is the TUI counterpart of `piko get providers` / `piko describe
// provider`. Implements Panel and the Detailer interface.
type ProvidersPanel struct {
	// lastRefresh records when the panel last received a list payload.
	lastRefresh time.Time

	// clock supplies time for tests; defaults to the real clock.
	clock clock.Clock

	// provider supplies the providers inspector port.
	provider ProvidersInspector

	// err holds the last list refresh error, or nil after success.
	err error

	*AssetViewer[ProviderEntry]

	// detailCache caches DescribeProvider responses keyed by
	// "type/name".
	detailCache map[string]*ProviderDetail

	// detailErrors records per-row describe errors, keyed by
	// "type/name".
	detailErrors map[string]error

	// subCursorByProvider records the sub-resource cursor for each
	// provider whose sub-resource list has been navigated. Keyed by
	// "type/name".
	subCursorByProvider map[string]int

	// pendingDetail tracks the "type/name" keys with an in-flight
	// describe-provider RPC. Used to dedup overlapping fetches when
	// the cursor moves rapidly or the periodic tick fires before a
	// previous fetch has completed.
	pendingDetail map[string]struct{}

	// stateMutex guards err / detailCache / detailErrors /
	// pendingDetail / subCursorByProvider for safe concurrent reads.
	stateMutex sync.RWMutex

	// subMode reports whether keyboard navigation drives the
	// sub-resource cursor (true) or the providers list (false). Tab
	// toggles.
	subMode bool
}

var _ Panel = (*ProvidersPanel)(nil)

// providersRenderer renders ProviderEntry rows.
type providersRenderer struct {
	// panel is the owning ProvidersPanel.
	panel *ProvidersPanel
}

// NewProvidersPanel constructs a ProvidersPanel.
//
// Takes provider (ProvidersInspector) which supplies provider metadata.
// Takes c (clock.Clock) for testing; nil falls back to the real clock.
//
// Returns *ProvidersPanel ready to register with a group.
func NewProvidersPanel(provider ProvidersInspector, c clock.Clock) *ProvidersPanel {
	if c == nil {
		c = clock.RealClock()
	}
	p := &ProvidersPanel{
		AssetViewer:         nil,
		clock:               c,
		provider:            provider,
		stateMutex:          sync.RWMutex{},
		detailCache:         map[string]*ProviderDetail{},
		detailErrors:        map[string]error{},
		subCursorByProvider: map[string]int{},
		pendingDetail:       map[string]struct{}{},
	}
	p.AssetViewer = NewAssetViewer(AssetViewerConfig[ProviderEntry]{
		ID:           "providers",
		Title:        "Providers",
		Renderer:     &providersRenderer{panel: p},
		NavMode:      NavigationSkipLine,
		EnableSearch: true,
		UseMutex:     true,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓ or j/k", Description: "Navigate"},
			{Key: "Tab", Description: "Toggle providers / sub-resource cursor"},
			{Key: "/", Description: "Search"},
			{Key: "r", Description: "Refresh"},
		},
	})
	return p
}

// Init triggers an initial refresh.
//
// Returns tea.Cmd which is the initial refresh command.
func (p *ProvidersPanel) Init() tea.Cmd { return p.refresh() }

// Update handles messages.
//
// Takes message (tea.Msg) which is the incoming update message.
//
// Returns Panel which is the (possibly mutated) panel.
// Returns tea.Cmd which is the next command to execute, or nil.
func (p *ProvidersPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if cmd, handled := forwardSearchUpdate(p.AssetViewer, message); handled {
		return p, cmd
	}

	switch msg := message.(type) {
	case tea.KeyPressMsg:
		return p.handleKey(msg)
	case providersRefreshMessage:
		p.handleList(msg)
		cmd := p.refreshSelectedDetailCmd()
		return p, cmd
	case providersDescribeMessage:
		p.handleDescribe(msg)
		return p, nil
	case DataUpdatedMessage, TickMessage:
		cmd := p.refresh()
		return p, cmd
	}
	return p, nil
}

// View renders the panel. Falls back to a "feature disabled" hint
// when the server does not expose the provider inspector service.
//
// Takes width (int) which is the rendering width in columns.
// Takes height (int) which is the rendering height in rows.
//
// Returns string which is the rendered panel content.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) View(width, height int) string {
	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()
	if IsServiceUnavailable(err) {
		p.SetSize(width, height)
		hint := ServiceUnavailableHint("Provider inspection",
			"Provider listing is unavailable. The server build omits the provider inspector.")
		return p.RenderFrame(RenderDimText(hint))
	}
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderHeader,
		RenderEmptyState:    p.renderEmptyState,
		RenderItems:         p.renderItems,
		TrimTrailingNewline: false,
	})
}

// DetailView renders the right-pane detail.
//
// Takes width (int) which is the detail-pane width in columns.
// Takes height (int) which is the detail-pane height in rows.
//
// Returns string which is the rendered detail body.
func (p *ProvidersPanel) DetailView(width, height int) string {
	body := p.detailBody()
	return RenderDetailBody(nil, body, width, height)
}

// renderHeader writes the search and error header lines.
//
// Takes content (*strings.Builder) which receives the rendered header lines.
//
// Returns int which is the number of header lines written.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) renderHeader(content *strings.Builder) int {
	used := 0
	if p.Search() != nil {
		used += p.Search().RenderHeader(content, len(p.Items()))
	}
	p.stateMutex.RLock()
	err := p.err
	p.stateMutex.RUnlock()
	if err != nil {
		RenderErrorState(content, err)
		used++
	}
	return used
}

// renderEmptyState writes the placeholder shown when no providers exist.
//
// Takes content (*strings.Builder) which receives the placeholder text.
func (*ProvidersPanel) renderEmptyState(content *strings.Builder) {
	content.WriteString(RenderDimText("No providers configured"))
}

// renderItems writes one row per visible provider entry.
//
// Takes content (*strings.Builder) which receives the rendered rows.
// Takes displayItems ([]int) which is the indices of items to render.
// Takes headerLines (int) which is the number of header lines already
// consumed by renderHeader, used to size the scroll context.
func (p *ProvidersPanel) renderItems(content *strings.Builder, displayItems []int, headerLines int) {
	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()
	for _, index := range displayItems {
		if index >= len(items) {
			continue
		}
		entry := items[index]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()
		ctx.WriteLineIfVisible(func() string {
			return p.renderRow(entry, selected)
		})
	}
}

// renderRow renders a single provider row with cursor and default-marker.
//
// Takes entry (ProviderEntry) which is the row to render.
// Takes selected (bool) which is true when the cursor sits on this row.
//
// Returns string which is the rendered row.
func (p *ProvidersPanel) renderRow(e ProviderEntry, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	star := "  "
	if e.IsDefault {
		star = "* "
	}
	label := fmt.Sprintf("%s/%s", e.ResourceType, e.Name)
	return cursor + star + RenderName(label, max(0, p.ContentWidth()-6), selected, p.Focused())
}

// handleKey routes a key press to the appropriate navigation handler.
//
// Takes msg (tea.KeyPressMsg) which is the key event.
//
// Returns Panel which is the receiver.
// Returns tea.Cmd which is the resulting command, or nil.
func (p *ProvidersPanel) handleKey(msg tea.KeyPressMsg) (Panel, tea.Cmd) {
	if msg.String() == "tab" {
		p.toggleSubMode()
		return p, nil
	}
	if p.subMode && p.handleSubResourceNavigation(msg) {
		return p, nil
	}
	result := HandleCommonKeys(p.AssetViewer, msg, p.refresh)
	if result.Handled {
		return p, tea.Batch(result.Cmd, p.refreshSelectedDetailCmd())
	}
	return p, nil
}

// toggleSubMode flips between providers-list navigation and
// sub-resource-list navigation.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) toggleSubMode() {
	p.stateMutex.Lock()
	p.subMode = !p.subMode
	p.stateMutex.Unlock()
}

// handleSubResourceNavigation drives the sub-resource cursor for the
// selected provider. Returns true when the key was consumed.
//
// Takes msg (tea.KeyPressMsg) which is the key event.
//
// Returns bool which is true when the key was an entry-cursor move.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) handleSubResourceNavigation(msg tea.KeyPressMsg) bool {
	current := p.GetItemAtCursor()
	if current == nil {
		return false
	}
	key := providerKey(*current)
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	detail, ok := p.detailCache[key]
	if !ok || len(detail.SubResources) == 0 {
		return false
	}
	cursor := p.subCursorByProvider[key]
	switch msg.String() {
	case "up", "k":
		cursor = max(0, cursor-1)
	case "down", "j":
		cursor = min(len(detail.SubResources)-1, cursor+1)
	case "g":
		cursor = 0
	case "G":
		cursor = len(detail.SubResources) - 1
	default:
		return false
	}
	p.subCursorByProvider[key] = cursor
	return true
}

// handleList processes a ListProviders refresh result.
//
// Takes msg (providersRefreshMessage) which carries the entries and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) handleList(msg providersRefreshMessage) {
	var entries []ProviderEntry
	if msg.err == nil {
		entries = msg.entries
		slices.SortStableFunc(entries, compareProviderEntries)
	}

	p.stateMutex.Lock()
	if msg.err != nil {
		p.err = msg.err
	} else {
		p.err = nil
	}
	p.lastRefresh = p.clock.Now()
	p.stateMutex.Unlock()

	if msg.err == nil {
		p.SetItems(entries)
	}
}

// compareProviderEntries orders entries by resource type then name.
// Used as the comparator for the ListProviders result so the panel
// row order is stable and deterministic.
//
// Takes a (ProviderEntry) which is the first entry to compare.
// Takes b (ProviderEntry) which is the second entry to compare.
//
// Returns int which is negative when a sorts first, positive when b
// sorts first, and 0 when they are equal.
func compareProviderEntries(a, b ProviderEntry) int {
	return cmp.Or(
		cmp.Compare(a.ResourceType, b.ResourceType),
		cmp.Compare(a.Name, b.Name),
	)
}

// handleDescribe records a DescribeProvider result in the cache.
//
// Takes msg (providersDescribeMessage) which carries the detail and any error.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) handleDescribe(msg providersDescribeMessage) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	delete(p.pendingDetail, msg.row)
	if msg.err != nil {
		p.detailErrors[msg.row] = msg.err
		return
	}
	delete(p.detailErrors, msg.row)
	p.detailCache[msg.row] = msg.detail
}

// refreshSelectedDetailCmd issues a DescribeProvider RPC for the row
// currently under the cursor when the cache lacks an entry and no
// other fetch is already in flight for the same key. Returns nil when
// no fetch is needed.
//
// Returns tea.Cmd which is the describe-provider command, or nil.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) refreshSelectedDetailCmd() tea.Cmd {
	current := p.GetItemAtCursor()
	if current == nil {
		return nil
	}
	key := providerKey(*current)
	p.stateMutex.Lock()
	if _, cached := p.detailCache[key]; cached {
		p.stateMutex.Unlock()
		return nil
	}
	if _, pending := p.pendingDetail[key]; pending {
		p.stateMutex.Unlock()
		return nil
	}
	p.pendingDetail[key] = struct{}{}
	p.stateMutex.Unlock()

	resourceType := current.ResourceType
	name := current.Name
	provider := p.provider
	return func() tea.Msg {
		if provider == nil {
			return providersDescribeMessage{row: key, err: errNoProvidersInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), providersRefreshTimeout,
			errors.New("describe provider exceeded timeout"))
		defer cancel()
		detail, err := provider.DescribeProvider(ctx, resourceType, name)
		return providersDescribeMessage{row: key, detail: detail, err: err}
	}
}

// refresh issues a ListProviders RPC and returns the result message.
//
// Returns tea.Cmd which delivers a providersRefreshMessage.
func (p *ProvidersPanel) refresh() tea.Cmd {
	return func() tea.Msg {
		if p.provider == nil {
			return providersRefreshMessage{err: errNoProvidersInspector}
		}
		ctx, cancel := context.WithTimeoutCause(context.Background(), providersRefreshTimeout,
			errors.New("providers list exceeded timeout"))
		defer cancel()
		entries, err := p.provider.ListProviders(ctx)
		return providersRefreshMessage{entries: entries, err: err}
	}
}

// detailBody composes the detail-pane body for the selected row.
//
// Returns inspector.DetailBody describing the panel state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) detailBody() inspector.DetailBody {
	current := p.GetItemAtCursor()
	if current == nil {
		return p.overviewBody()
	}
	key := providerKey(*current)
	p.stateMutex.RLock()
	detail, ok := p.detailCache[key]
	derr := p.detailErrors[key]
	subCursor := p.subCursorByProvider[key]
	subMode := p.subMode
	p.stateMutex.RUnlock()
	if !ok {
		return providerEntryDetail(*current, derr)
	}
	return providerDetailBody(*current, detail, subCursor, subMode)
}

// overviewBody renders the overview shown when no row is selected.
//
// Returns inspector.DetailBody describing the overview state.
//
// Concurrency: Safe for concurrent use; guarded by stateMutex.
func (p *ProvidersPanel) overviewBody() inspector.DetailBody {
	itemCount := len(p.Items())
	p.stateMutex.RLock()
	lastRefresh := p.lastRefresh
	err := p.err
	p.stateMutex.RUnlock()
	return inspectorOverviewBody(inspectorOverviewArgs{
		title:       "Providers overview",
		itemLabel:   "Providers",
		itemCount:   itemCount,
		lastRefresh: lastRefresh,
		err:         err,
	})
}

// providerEntryDetail renders the cache-miss detail body using the
// row's own metadata while a describe RPC is in flight.
//
// Takes entry (ProviderEntry) which is the selected provider row.
// Takes derr (error) which is the latest describe error, may be nil.
//
// Returns inspector.DetailBody describing the cache-miss state.
func providerEntryDetail(e ProviderEntry, derr error) inspector.DetailBody {
	const providerEntryFixedRows = 3
	rows := make([]inspector.DetailRow, 0, providerEntryFixedRows+len(e.Values))
	rows = append(rows,
		inspector.DetailRow{Label: "Type", Value: e.ResourceType},
		inspector.DetailRow{Label: "Name", Value: e.Name},
		inspector.DetailRow{Label: "Default", Value: boolLabel(e.IsDefault)},
	)
	for _, k := range sortedStringKeys(e.Values) {
		rows = append(rows, inspector.DetailRow{Label: k, Value: e.Values[k]})
	}
	sections := []inspector.DetailSection{{Heading: "Summary", Rows: rows}}
	if derr != nil {
		sections = append(sections, inspector.DetailSection{Heading: "Error", Rows: []inspector.DetailRow{{Label: "Reason", Value: derr.Error()}}})
	} else {
		sections = append(sections, inspector.DetailSection{Heading: "Detail", Rows: []inspector.DetailRow{{Label: "Status", Value: "loading..."}}})
	}
	return inspector.DetailBody{
		Title:    e.ResourceType + "/" + e.Name,
		Subtitle: "press r to refresh detail",
		Sections: sections,
	}
}

// providerDetailBody renders the cached describe-provider sections
// plus a navigable sub-resource list. When the panel is in sub-mode
// and a sub-resource is selected, an additional "Selected
// sub-resource" section displays its full key/value metadata.
//
// Takes e (ProviderEntry) which is the selected provider row.
// Takes d (*ProviderDetail) which is the cached describe output.
// Takes subCursor (int) which is the focused sub-resource index.
// Takes subMode (bool) which is true when keyboard navigation drives
// the sub-resource cursor.
//
// Returns inspector.DetailBody ready to pass to RenderDetailBody.
func providerDetailBody(e ProviderEntry, d *ProviderDetail, subCursor int, subMode bool) inspector.DetailBody {
	sections := make([]inspector.DetailSection, 0, len(d.Sections)+2)
	for _, s := range d.Sections {
		sections = append(sections, providerSectionToDetail(s))
	}
	sections = append(sections, providerSubResourceSections(d.SubResources, subCursor, subMode)...)
	return inspector.DetailBody{
		Title:    e.ResourceType + "/" + e.Name,
		Subtitle: providerDetailSubtitle(e, len(d.SubResources) > 0, subMode),
		Sections: sections,
	}
}

// providerSectionToDetail flattens a ProviderSection's entries into
// a inspector.DetailSection.
//
// Takes s (ProviderSection) which is the source section.
//
// Returns inspector.DetailSection which is the flattened detail section.
func providerSectionToDetail(s ProviderSection) inspector.DetailSection {
	rows := make([]inspector.DetailRow, 0, len(s.Entries))
	for _, en := range s.Entries {
		rows = append(rows, inspector.DetailRow{Label: en.Key, Value: en.Value})
	}
	return inspector.DetailSection{Heading: s.Title, Rows: rows}
}

// providerSubResourceSections renders the navigable sub-resource
// list and the optional focused-entry section. Returns an empty
// slice when there are no sub-resources.
//
// Takes subs ([]ProviderSubResource) which is the sub-resource list.
// Takes subCursor (int) which is the focused sub-resource index.
// Takes subMode (bool) which is true when keyboard navigation drives
// the sub-resource cursor.
//
// Returns []inspector.DetailSection which is the rendered sections.
func providerSubResourceSections(subs []ProviderSubResource, subCursor int, subMode bool) []inspector.DetailSection {
	if len(subs) == 0 {
		return nil
	}
	rows := make([]inspector.DetailRow, 0, len(subs))
	for i, r := range subs {
		marker := DoubleSpace
		if subMode && i == subCursor {
			marker = MenuMarker + SingleSpace
		}
		label := marker + r.Type + "/" + r.Name
		rows = append(rows, inspector.DetailRow{Label: label, Value: subResourceSummary(r.Values)})
	}
	out := []inspector.DetailSection{{Heading: "Sub-resources", Rows: rows}}
	if subMode && subCursor >= 0 && subCursor < len(subs) {
		out = append(out, providerSubResourceFocusSection(subs[subCursor]))
	}
	return out
}

// providerDetailSubtitle composes the detail-pane subtitle with a
// hint about Tab when sub-resources exist.
//
// Takes e (ProviderEntry) which is the selected provider.
// Takes hasSubs (bool) which is true when sub-resources exist.
// Takes subMode (bool) which is true when in sub-resource navigation mode.
//
// Returns string which is the composed subtitle.
func providerDetailSubtitle(e ProviderEntry, hasSubs, subMode bool) string {
	subtitle := boolLabel(e.IsDefault) + " default"
	if !hasSubs {
		return subtitle
	}
	if subMode {
		return subtitle + " · sub-resource mode (Tab to switch)"
	}
	return subtitle + " · Tab to navigate sub-resources"
}

// providerSubResourceFocusSection renders the deep-dive section for
// the focused sub-resource: every key/value field laid out one per
// row instead of squashed into a single summary line.
//
// Takes r (ProviderSubResource) which is the focused sub-resource.
//
// Returns inspector.DetailSection ready to append to the body.
func providerSubResourceFocusSection(r ProviderSubResource) inspector.DetailSection {
	const subResourceFixedRows = 2
	rows := make([]inspector.DetailRow, 0, subResourceFixedRows+len(r.Values))
	rows = append(rows,
		inspector.DetailRow{Label: "Type", Value: r.Type},
		inspector.DetailRow{Label: "Name", Value: r.Name},
	)
	for _, k := range sortedStringKeys(r.Values) {
		rows = append(rows, inspector.DetailRow{Label: k, Value: r.Values[k]})
	}
	return inspector.DetailSection{Heading: "Selected sub-resource", Rows: rows}
}

// subResourceSummary collapses a sub-resource's values into a single
// space-delimited "k=v" line for compact list display.
//
// Takes subs (map[string]string) which is the sub-resource's key/value pairs.
//
// Returns string which is the joined "k=v" line, or empty when no values.
func subResourceSummary(values map[string]string) string {
	keys := sortedStringKeys(values)
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+values[k])
	}
	return strings.Join(parts, " ")
}

// providerKey composes the "type/name" cache key for a provider entry.
//
// Takes entry (ProviderEntry) which is the entry to key.
//
// Returns string which is the cache key in the form "type/name".
func providerKey(e ProviderEntry) string { return e.ResourceType + "/" + e.Name }

// sortedStringKeys returns the sorted keys of a string-keyed map.
//
// Takes m (map[string]string) which is the map to read keys from.
//
// Returns []string which is the keys sorted in ascending order.
func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// boolLabel maps a bool to the human-readable "yes" or "no".
//
// Takes b (bool) which is the value to label.
//
// Returns string which is "yes" when b is true, otherwise "no".
func boolLabel(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// RenderRow renders a provider entry as a single list row.
//
// Takes entry (ProviderEntry) which is the row to render.
// Takes selected (bool) which is true when the cursor sits on this row.
//
// Returns string which is the rendered row.
func (r *providersRenderer) RenderRow(item ProviderEntry, _ int, selected, _ bool, _ int) string {
	return r.panel.renderRow(item, selected)
}

// RenderExpanded returns no expanded lines; provider rows are not expandable.
//
// Returns []string which is always nil for provider rows.
func (*providersRenderer) RenderExpanded(_ ProviderEntry, _ int) []string { return nil }

// GetID returns the unique "type/name" identifier for an entry.
//
// Takes entry (ProviderEntry) which is the row to identify.
//
// Returns string which is the "type/name" identifier.
func (*providersRenderer) GetID(item ProviderEntry) string { return providerKey(item) }

// MatchesFilter reports whether an entry matches the supplied search query.
//
// Takes entry (ProviderEntry) which is the row to filter.
// Takes query (string) which is the search query.
//
// Returns bool which is true when the resource type or name contains query.
func (*providersRenderer) MatchesFilter(item ProviderEntry, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(item.ResourceType), q) ||
		strings.Contains(strings.ToLower(item.Name), q)
}

// IsExpandable reports whether the entry can be expanded; always false here.
//
// Returns bool which is always false for provider rows.
func (*providersRenderer) IsExpandable(_ ProviderEntry) bool { return false }

// ExpandedLineCount returns the number of expanded lines; always zero.
//
// Returns int which is always zero for provider rows.
func (*providersRenderer) ExpandedLineCount(_ ProviderEntry) int { return 0 }
