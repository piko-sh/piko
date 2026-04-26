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

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// registryItemType represents the kind of item shown in the registry display
// list.
type registryItemType int

const (
	// registryItemKind marks a row as a resource kind entry in the registry panel.
	registryItemKind registryItemType = iota

	// registryItemResource marks a display item as a resource entry.
	registryItemResource

	// registryItemMetadata marks a display item as a metadata row.
	registryItemMetadata
)

const (
	// resourcesMetadataIndent is the number of spaces used to indent metadata
	// rows.
	resourcesMetadataIndent = 8

	// resourcesMetadataWidthAdj is the width offset for metadata rows.
	resourcesMetadataWidthAdj = 12

	// resourcesCursorIndent is the spacing before the selection cursor arrow.
	resourcesCursorIndent = "  "

	// resourcesCursorPadding is the blank space shown when no cursor is present.
	resourcesCursorPadding = "    "

	// resourcesNameWidthAdj is the width to subtract from the content area when
	// truncating resource names.
	resourcesNameWidthAdj = 12
)

var (
	_ Panel = (*RegistryPanel)(nil)

	_ ItemRenderer[registryDisplayItem] = (*registryRenderer)(nil)
)

// registryDisplayItem represents a selectable item in the registry panel.
type registryDisplayItem struct {
	// kind is the resource kind for kind-type rows.
	kind string

	// resource holds the Resource data for resource rows; nil for other row types.
	resource *Resource

	// metadataKey is the key name for metadata row items.
	metadataKey string

	// metadataVal holds the value for metadata rows.
	metadataVal string

	// resourceID is the parent resource identifier for metadata rows.
	resourceID string

	// resourceIndex is the position of this item in the parent's resources slice.
	resourceIndex int

	// itemType specifies the entry type: kind, group, or resource.
	itemType registryItemType
}

// RegistryPanel provides a view of all resource kinds and their status
// summaries. It implements the Panel interface.
type RegistryPanel struct {
	*AssetViewer[registryDisplayItem]

	// summary maps resource kinds to their status counts.
	summary map[string]map[ResourceStatus]int

	// selectedKind is the kind currently expanded in the registry tree.
	selectedKind string

	// expandedResource is the ID of the currently expanded resource, or empty
	// if no resource is expanded.
	expandedResource string

	// kinds holds the sorted list of resource kinds for display ordering.
	kinds []string

	// resources holds the resources for the currently selected kind.
	resources []Resource
}

// registryRenderer shows registry items in the panel.
type registryRenderer struct {
	// panel is the parent panel used for rendering registry rows.
	panel *RegistryPanel
}

// NewRegistryPanel creates a new registry overview panel.
//
// Returns *RegistryPanel which is the configured panel ready for use.
func NewRegistryPanel() *RegistryPanel {
	p := &RegistryPanel{
		AssetViewer:      nil,
		summary:          make(map[string]map[ResourceStatus]int),
		kinds:            []string{},
		resources:        nil,
		selectedKind:     "",
		expandedResource: "",
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[registryDisplayItem]{
		ID:           "registry",
		Title:        "Registry",
		Renderer:     &registryRenderer{panel: p},
		NavMode:      NavigationSimple,
		EnableSearch: true,
		UseMutex:     false,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Move up/down"},
			{Key: "←/→", Description: "Collapse/Expand"},
			{Key: "Space", Description: "Toggle expand"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear/Collapse"},
			{Key: "g/G", Description: "Go to top/bottom"},
		},
	})

	return p
}

// SetSummary updates the resource summary data.
//
// Takes summary (map[string]map[ResourceStatus]int) which provides the count
// of resources grouped by kind and status.
func (p *RegistryPanel) SetSummary(summary map[string]map[ResourceStatus]int) {
	p.summary = summary

	p.kinds = make([]string, 0, len(summary))
	for kind := range summary {
		p.kinds = append(p.kinds, kind)
	}

	slices.Sort(p.kinds)

	p.rebuildDisplayItems()
}

// SetResources updates the detailed resources for the selected kind.
//
// Takes resources ([]Resource) which contains the resources to display.
func (p *RegistryPanel) SetResources(resources []Resource) {
	p.resources = resources
	p.rebuildDisplayItems()
}

// SelectedKind returns the currently selected/expanded kind.
//
// Returns string which is the kind that is currently selected.
func (p *RegistryPanel) SelectedKind() string {
	return p.selectedKind
}

// Update processes an input message and returns the updated panel state.
//
// Takes message (tea.Msg) which is the input message to process.
//
// Returns Panel which is the updated panel after processing.
// Returns tea.Cmd which is the command to run, or nil if there is none.
func (p *RegistryPanel) Update(message tea.Msg) (Panel, tea.Cmd) {
	if p.Search() != nil && p.Search().IsActive() {
		handled, command := p.Search().Update(message)
		if handled {
			return p, command
		}
	}

	keyMessage, ok := message.(tea.KeyPressMsg)
	if !ok {
		return p, nil
	}

	if p.HandleNavigation(keyMessage) {
		return p, nil
	}

	return p.handleKeyAction(keyMessage)
}

// View renders the panel.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *RegistryPanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderRegistryHeader,
		RenderEmptyState:    p.renderRegistryEmptyState,
		RenderItems:         p.renderRegistryItems,
		TrimTrailingNewline: true,
	})
}

// rebuildDisplayItems rebuilds the flat list of display items from the current
// state.
func (p *RegistryPanel) rebuildDisplayItems() {
	items := make([]registryDisplayItem, 0)

	for _, kind := range p.kinds {
		items = append(items, p.buildKindItem(kind))
		if p.selectedKind == kind {
			items = append(items, p.buildResourceItems(kind)...)
		}
	}

	p.SetItems(items)
}

// buildKindItem creates a display item for a resource kind.
//
// Takes kind (string) which specifies the resource kind to display.
//
// Returns registryDisplayItem which represents the kind as a display item.
func (*RegistryPanel) buildKindItem(kind string) registryDisplayItem {
	return registryDisplayItem{
		kind:          kind,
		resource:      nil,
		metadataKey:   "",
		metadataVal:   "",
		resourceID:    "",
		resourceIndex: 0,
		itemType:      registryItemKind,
	}
}

// buildResourceItems creates display items for all resources of a kind.
//
// Takes kind (string) which specifies the resource type to build items for.
//
// Returns []registryDisplayItem which contains the display items for all
// resources, including expanded metadata items for the currently selected
// resource.
func (p *RegistryPanel) buildResourceItems(kind string) []registryDisplayItem {
	items := make([]registryDisplayItem, 0, len(p.resources))

	for i := range p.resources {
		resource := &p.resources[i]
		items = append(items, registryDisplayItem{
			kind:          kind,
			resource:      resource,
			metadataKey:   "",
			metadataVal:   "",
			resourceID:    "",
			resourceIndex: i,
			itemType:      registryItemResource,
		})

		if p.expandedResource == resource.ID {
			items = append(items, p.buildMetadataItems(kind, resource)...)
		}
	}

	return items
}

// buildMetadataItems creates display items for a resource's metadata.
//
// Takes kind (string) which specifies the type label for the display items.
// Takes resource (*Resource) which provides the resource containing metadata.
//
// Returns []registryDisplayItem which contains sorted metadata display items.
func (*RegistryPanel) buildMetadataItems(kind string, resource *Resource) []registryDisplayItem {
	keys := make([]string, 0, len(resource.Metadata))
	for key := range resource.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	items := make([]registryDisplayItem, 0, len(keys))
	for _, key := range keys {
		value := resource.Metadata[key]
		if value == "" {
			continue
		}
		items = append(items, registryDisplayItem{
			kind:          kind,
			resource:      resource,
			metadataKey:   key,
			metadataVal:   value,
			resourceID:    resource.ID,
			resourceIndex: 0,
			itemType:      registryItemMetadata,
		})
	}

	return items
}

// handleKeyAction processes key press events for the registry panel.
//
// Takes keyMessage (tea.KeyPressMsg) which contains the key press event.
//
// Returns Panel which is the panel after handling the key.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *RegistryPanel) handleKeyAction(keyMessage tea.KeyPressMsg) (Panel, tea.Cmd) {
	switch keyMessage.String() {
	case "right", "l":
		p.handleExpandKey()
		return p, nil
	case "left", "h":
		p.handleCollapseKey()
		return p, nil
	case "enter", "space":
		p.handleToggleKey()
		return p, nil
	case "/":
		if p.Search() != nil {
			p.Search().SetWidth(p.ContentWidth())
			return p, p.Search().SearchBox().Open()
		}
	case "esc":
		p.handleEscKey()
		return p, nil
	}
	return p, nil
}

// handleExpandKey expands the selected kind or resource when the user presses
// the right arrow key or the l key.
func (p *RegistryPanel) handleExpandKey() {
	item := p.GetItemAtCursor()
	if item == nil {
		return
	}

	switch item.itemType {
	case registryItemKind:
		if p.selectedKind != item.kind {
			p.selectedKind = item.kind
			p.expandedResource = ""
			p.rebuildDisplayItems()
		}
	case registryItemResource:
		if item.resource != nil && p.expandedResource != item.resource.ID {
			p.expandedResource = item.resource.ID
			p.rebuildDisplayItems()
		}
	default:
	}
}

// handleCollapseKey collapses the expanded kind or resource when left/h is
// pressed.
func (p *RegistryPanel) handleCollapseKey() {
	if p.expandedResource != "" {
		p.expandedResource = ""
		p.rebuildDisplayItems()
		return
	}
	if p.selectedKind != "" {
		p.selectedKind = ""
		p.resources = nil
		p.rebuildDisplayItems()
	}
}

// handleToggleKey toggles the expansion state of the item at the cursor.
func (p *RegistryPanel) handleToggleKey() {
	item := p.GetItemAtCursor()
	if item == nil {
		return
	}

	switch item.itemType {
	case registryItemKind:
		if p.selectedKind == item.kind {
			p.selectedKind = ""
			p.resources = nil
		} else {
			p.selectedKind = item.kind
			p.expandedResource = ""
		}
		p.rebuildDisplayItems()
	case registryItemResource:
		if item.resource == nil {
			return
		}
		if p.expandedResource == item.resource.ID {
			p.expandedResource = ""
		} else {
			p.expandedResource = item.resource.ID
		}
		p.rebuildDisplayItems()
	default:
	}
}

// handleEscKey clears state in order: search query first, then expanded
// resource, then selected kind.
func (p *RegistryPanel) handleEscKey() {
	if p.Search() != nil && p.Search().HasQuery() {
		p.Search().ClearQuery()
		p.rebuildDisplayItems()
		return
	}
	if p.expandedResource != "" {
		p.expandedResource = ""
		p.rebuildDisplayItems()
		return
	}
	if p.selectedKind != "" {
		p.selectedKind = ""
		p.resources = nil
		p.rebuildDisplayItems()
	}
}

// renderRegistryHeader renders the search box and filter status.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines used by the header.
func (p *RegistryPanel) renderRegistryHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.kinds))
	}

	return usedLines
}

// renderRegistryEmptyState writes the empty state message to the builder.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *RegistryPanel) renderRegistryEmptyState(content *strings.Builder) {
	message := "No resources available"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No resources match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderRegistryItems writes all registry items to the output buffer.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies the indices of items to render.
// Takes headerLines (int) which is the number of header lines to exclude from
// the content height.
func (p *RegistryPanel) renderRegistryItems(content *strings.Builder, displayItems []int, headerLines int) {
	ctx := NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines)
	items := p.Items()

	for _, itemIndex := range displayItems {
		if itemIndex >= len(items) {
			continue
		}

		item := items[itemIndex]
		lineIndex := ctx.LineIndex()
		selected := lineIndex == p.Cursor()

		ctx.WriteLineIfVisible(func() string {
			return p.renderRegistryRow(item, selected)
		})
	}
}

// renderRegistryRow renders a single registry row based on item type.
//
// Takes item (registryDisplayItem) which specifies the row data to render.
// Takes selected (bool) which indicates if the row is currently selected.
//
// Returns string which contains the rendered row, or empty if the type is not
// handled.
func (p *RegistryPanel) renderRegistryRow(item registryDisplayItem, selected bool) string {
	switch item.itemType {
	case registryItemKind:
		return p.renderKindRow(item.kind, selected)
	case registryItemResource:
		if item.resource != nil {
			return p.renderResourceRow(*item.resource, selected)
		}
	case registryItemMetadata:
		return p.renderMetadataRow(item.metadataKey, item.metadataVal, selected)
	}
	return ""
}

// renderKindRow renders a single kind summary row.
//
// Takes kind (string) which specifies the type of registry item to render.
// Takes selected (bool) which indicates whether this row is currently selected.
//
// Returns string which is the formatted row ready for display.
func (p *RegistryPanel) renderKindRow(kind string, selected bool) string {
	cursor := RenderCursor(selected, p.Focused())
	counts := p.summary[kind]

	status := determineKindOverallStatus(counts)
	indicator := StatusIndicator(status)
	expanded := p.selectedKind == kind
	expandChar := RenderExpandIndicator(expanded)

	countString := buildKindCountString(counts)

	caser := cases.Title(language.English)
	kindName := caser.String(kind)
	if selected && p.Focused() {
		kindName = lipgloss.NewStyle().Bold(true).Render(kindName)
	}

	return fmt.Sprintf("%s%s %s %s %s", cursor, indicator, expandChar, kindName, countString)
}

// renderResourceRow renders a single resource row when the registry panel is
// expanded.
//
// Takes resource (Resource) which is the resource to render.
// Takes selected (bool) which shows whether this row is selected.
//
// Returns string which is the formatted row with cursor, status marker, expand
// marker, and shortened name.
func (p *RegistryPanel) renderResourceRow(resource Resource, selected bool) string {
	cursor := resourcesCursorPadding
	if selected {
		cursor = resourcesCursorIndent + cursorArrow
		if p.Focused() {
			cursor = resourcesCursorIndent + lipgloss.NewStyle().Foreground(colourPrimary).Render(cursorArrow)
		}
	}

	indicator := StatusIndicator(resource.Status)

	expandChar := " "
	if len(resource.Metadata) > 0 {
		expanded := p.expandedResource == resource.ID
		expandChar = RenderExpandIndicator(expanded)
	}

	name := TruncateString(resource.Name, p.ContentWidth()-resourcesNameWidthAdj)
	if selected && p.Focused() {
		name = lipgloss.NewStyle().Bold(true).Render(name)
	}

	return fmt.Sprintf("%s%s %s %s", cursor, indicator, expandChar, name)
}

// renderMetadataRow formats a metadata key-value pair as a display row.
//
// Takes key (string) which is the metadata field name.
// Takes value (string) which is the metadata field value.
// Takes selected (bool) which shows whether the row is selected.
//
// Returns string which is the formatted row ready for display.
func (p *RegistryPanel) renderMetadataRow(key, value string, selected bool) string {
	return RenderMetadataRow(key, value, MetadataRowConfig{
		IndentSpaces:    resourcesMetadataIndent,
		WidthAdjustment: resourcesMetadataWidthAdj,
		Selected:        selected,
		Focused:         p.Focused(),
		ContentWidth:    p.ContentWidth(),
	})
}

// RenderRow renders a registry display item row.
//
// Takes item (registryDisplayItem) which holds the registry data
// to render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted registry row for display.
func (r *registryRenderer) RenderRow(item registryDisplayItem, _ int, selected, _ bool, _ int) string {
	return r.panel.renderRegistryRow(item, selected)
}

// RenderExpanded returns nil as expansion is handled by rebuilding items.
//
// Returns []string which is always nil for this renderer.
func (*registryRenderer) RenderExpanded(_ registryDisplayItem, _ int) []string {
	return nil
}

// GetID returns a unique identifier for the registry item.
//
// Takes item (registryDisplayItem) which is the item to get an ID for.
//
// Returns string which is a prefixed identifier based on item type, or empty
// if the type is not known or the resource is nil.
func (*registryRenderer) GetID(item registryDisplayItem) string {
	switch item.itemType {
	case registryItemKind:
		return "kind:" + item.kind
	case registryItemResource:
		if item.resource != nil {
			return "resource:" + item.resource.ID
		}
	case registryItemMetadata:
		return "meta:" + item.resourceID + ":" + item.metadataKey
	}
	return ""
}

// MatchesFilter checks whether the item matches the search query.
//
// Takes item (registryDisplayItem) which is the display item to check.
// Takes query (string) which is the search text to match against.
//
// Returns bool which is true if the item contains the query text.
func (*registryRenderer) MatchesFilter(item registryDisplayItem, query string) bool {
	q := strings.ToLower(query)
	switch item.itemType {
	case registryItemKind:
		return strings.Contains(strings.ToLower(item.kind), q)
	case registryItemResource:
		if item.resource != nil {
			return strings.Contains(strings.ToLower(item.resource.Name), q) ||
				strings.Contains(strings.ToLower(item.resource.ID), q)
		}
	case registryItemMetadata:
		return strings.Contains(strings.ToLower(item.metadataKey), q) ||
			strings.Contains(strings.ToLower(item.metadataVal), q)
	}
	return false
}

// IsExpandable reports whether the item can be expanded.
//
// Takes item (registryDisplayItem) which is the display item to check.
//
// Returns bool which is true if the item is a kind, or a resource with
// metadata.
func (*registryRenderer) IsExpandable(item registryDisplayItem) bool {
	switch item.itemType {
	case registryItemKind:
		return true
	case registryItemResource:
		return item.resource != nil && len(item.resource.Metadata) > 0
	default:
		return false
	}
}

// ExpandedLineCount returns the number of extra lines for expanded items.
// This always returns zero because expansion rebuilds the item list instead.
//
// Returns int which is always zero.
func (*registryRenderer) ExpandedLineCount(_ registryDisplayItem) int {
	return 0
}

// determineKindOverallStatus finds the most severe status from a set of counts.
//
// Takes counts (map[ResourceStatus]int) which maps each status to its count.
//
// Returns ResourceStatus which is the most severe status found. Statuses are
// checked in order: unhealthy, degraded, pending, healthy. Returns unknown if
// counts is empty.
func determineKindOverallStatus(counts map[ResourceStatus]int) ResourceStatus {
	switch {
	case counts[ResourceStatusUnhealthy] > 0:
		return ResourceStatusUnhealthy
	case counts[ResourceStatusDegraded] > 0:
		return ResourceStatusDegraded
	case counts[ResourceStatusPending] > 0:
		return ResourceStatusPending
	case counts[ResourceStatusHealthy] > 0:
		return ResourceStatusHealthy
	default:
		return ResourceStatusUnknown
	}
}
