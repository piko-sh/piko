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
)

const (
	// storageRowSuffixWidth is the number of characters reserved for the row suffix.
	storageRowSuffixWidth = 26

	// storageRowMinNameWidth is the minimum width in characters for artefact names.
	storageRowMinNameWidth = 20
)

var (
	_ Panel = (*StoragePanel)(nil)

	_ ItemRenderer[Resource] = (*storageRenderer)(nil)
)

// StoragePanel displays storage items and their variants.
// It implements the Panel interface.
type StoragePanel struct {
	*AssetViewer[Resource]

	*StatusFilterMixin

	// artefacts holds all resources before status filtering is applied.
	artefacts []Resource
}

// storageRenderer implements ItemRenderer for storage resources.
type storageRenderer struct {
	// panel is the parent panel used to check expansion state and render rows.
	panel *StoragePanel
}

// NewStoragePanel creates a new storage detail panel.
//
// Returns *StoragePanel which is set up to show storage resources.
func NewStoragePanel() *StoragePanel {
	p := &StoragePanel{
		AssetViewer:       nil,
		StatusFilterMixin: NewStatusFilterMixin(),
		artefacts:         []Resource{},
	}

	p.AssetViewer = NewAssetViewer(AssetViewerConfig[Resource]{
		ID:           "storage",
		Title:        "Storage",
		Renderer:     &storageRenderer{panel: p},
		NavMode:      NavigationSimple,
		EnableSearch: true,
		UseMutex:     false,
		KeyBindings: []KeyBinding{
			{Key: "↑/↓", Description: "Move up/down"},
			{Key: "Space", Description: "Expand / Collapse"},
			{Key: "/", Description: "Search"},
			{Key: "Esc", Description: "Clear search/collapse"},
			{Key: "f", Description: "Filter by status"},
			{Key: "g/G", Description: "Go to top/bottom"},
		},
	})

	return p
}

// SetArtefacts updates the artefact list.
//
// Takes artefacts ([]Resource) which specifies the new list of artefacts to
// display.
func (p *StoragePanel) SetArtefacts(artefacts []Resource) {
	p.artefacts = artefacts
	p.applyStatusFilter()
}

// Update handles input messages for the storage panel.
//
// Takes message (tea.Msg) which is the message to process.
//
// Returns Panel which is the updated panel after processing.
// Returns tea.Cmd which is the command to run, or nil if none.
func (p *StoragePanel) Update(message tea.Msg) (Panel, tea.Cmd) {
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

// View renders the panel at the specified dimensions.
//
// Takes width (int) which specifies the panel width in characters.
// Takes height (int) which specifies the panel height in lines.
//
// Returns string which contains the rendered panel content.
func (p *StoragePanel) View(width, height int) string {
	return p.RenderViewWith(width, height, ViewCallbacks{
		RenderHeader:        p.renderStorageHeader,
		RenderEmptyState:    p.renderStorageEmptyState,
		RenderItems:         p.renderStorageItems,
		TrimTrailingNewline: true,
	})
}

// applyStatusFilter filters items by status and updates the viewer.
func (p *StoragePanel) applyStatusFilter() {
	if p.FilterStatus() == nil {
		p.SetItems(p.artefacts)
		return
	}

	filtered := make([]Resource, 0)
	for i := range p.artefacts {
		if p.MatchesFilter(p.artefacts[i].Status) {
			filtered = append(filtered, p.artefacts[i])
		}
	}
	p.SetItems(filtered)
}

// handleKey handles key events for the storage panel.
//
// Takes message (tea.KeyPressMsg) which contains the key event to handle.
//
// Returns Panel which is the updated panel after handling the key event.
// Returns tea.Cmd which is a command to run, or nil if none is needed.
func (p *StoragePanel) handleKey(message tea.KeyPressMsg) (Panel, tea.Cmd) {
	if message.String() != "esc" {
		result := HandleCommonKeys(p.AssetViewer, message, nil)
		if result.Handled {
			return p, result.Cmd
		}
	}

	switch message.String() {
	case "esc":
		if p.Search() != nil && p.Search().HasQuery() {
			p.Search().ClearQuery()
			p.applyStatusFilter()
			return p, nil
		}
		if len(p.ExpandedMap()) > 0 {
			p.CollapseAll()
			return p, nil
		}
	case "f":
		p.CycleFilter()
		p.applyStatusFilter()
		p.SetCursor(0)
		p.SetScrollOffset(0)
		return p, nil
	}

	return p, nil
}

// renderStorageHeader renders the search box and filter status.
//
// Takes content (*strings.Builder) which receives the rendered header output.
//
// Returns int which is the number of lines used by the header.
func (p *StoragePanel) renderStorageHeader(content *strings.Builder) int {
	usedLines := 0

	if p.Search() != nil {
		usedLines += p.Search().RenderHeader(content, len(p.artefacts))
	}

	usedLines += p.RenderFilterStatus(content)

	return usedLines
}

// renderStorageEmptyState writes the empty state message to the output.
//
// Takes content (*strings.Builder) which receives the rendered output.
func (p *StoragePanel) renderStorageEmptyState(content *strings.Builder) {
	message := "No artefacts"
	if p.Search() != nil && p.Search().HasQuery() {
		message = "No artefacts match search"
	} else if p.HasFilter() {
		message = "No artefacts match filter"
	}
	content.WriteString(RenderDimText(message))
}

// renderStorageItems renders all storage items with their metadata.
//
// Takes content (*strings.Builder) which receives the rendered output.
// Takes displayItems ([]int) which specifies which item indices to show.
// Takes headerLines (int) which is the number of header lines to subtract
// from the content height.
func (p *StoragePanel) renderStorageItems(content *strings.Builder, displayItems []int, headerLines int) {
	RenderExpandableItems(RenderExpandableItemsConfig[Resource]{
		Ctx:          NewScrollContext(content, p.ScrollOffset(), p.ContentHeight()-headerLines),
		Items:        p.Items(),
		DisplayItems: displayItems,
		Cursor:       p.Cursor(),
		GetID:        func(r Resource) string { return r.ID },
		IsExpanded:   p.IsExpanded,
		RenderRow:    p.renderArtefactRow,
		RenderExpand: p.renderArtefactMetadata,
	})
}

// renderArtefactRow renders a single artefact row.
//
// Takes artefact (Resource) which provides the artefact data to display.
// Takes selected (bool) which indicates whether this row is selected.
// Takes expanded (bool) which indicates whether the row details are expanded.
//
// Returns string which is the formatted row ready for display.
func (p *StoragePanel) renderArtefactRow(artefact Resource, selected, expanded bool) string {
	cursor := RenderCursor(selected, p.Focused())
	indicator := StatusIndicator(artefact.Status)
	expandChar := RenderExpandIndicator(expanded)

	variantCount := artefact.Metadata["variant_count"]
	totalSize := artefact.Metadata["total_size"]

	nameWidth := max(storageRowMinNameWidth, p.ContentWidth()-storageRowSuffixWidth)

	name := RenderName(artefact.Name, nameWidth, selected, p.Focused())

	suffix := RenderDimText(fmt.Sprintf("%sv %s", variantCount, totalSize))

	return fmt.Sprintf("%s%s %s %s %s", cursor, indicator, expandChar, name, suffix)
}

// renderArtefactMetadata shows sorted metadata for an expanded artefact.
//
// Takes ctx (*ScrollContext) which tracks scroll position and visible lines.
// Takes artefact (Resource) which provides the metadata to show.
func (p *StoragePanel) renderArtefactMetadata(ctx *ScrollContext, artefact Resource) {
	keys := make([]string, 0, len(artefact.Metadata))
	for key := range artefact.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		value := artefact.Metadata[key]
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

// RenderRow renders an artefact row.
//
// Takes artefact (Resource) which provides the artefact data to
// render.
// Takes _ (int) which is the unused line index.
// Takes selected (bool) which indicates if this row is selected.
// Takes _ (bool) which is the unused focused state.
// Takes _ (int) which is the unused content width.
//
// Returns string which is the formatted artefact row for display.
func (r *storageRenderer) RenderRow(artefact Resource, _ int, selected, _ bool, _ int) string {
	expanded := r.panel.IsExpanded(artefact.ID)
	return r.panel.renderArtefactRow(artefact, selected, expanded)
}

// RenderExpanded returns metadata lines for an artefact.
//
// Takes artefact (Resource) which provides the metadata to render.
// Takes width (int) which sets the content width for formatting.
//
// Returns []string which contains the formatted metadata rows sorted by key.
func (*storageRenderer) RenderExpanded(artefact Resource, width int) []string {
	keys := make([]string, 0, len(artefact.Metadata))
	for key := range artefact.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := artefact.Metadata[key]
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

// GetID returns the artefact's unique identifier.
//
// Takes artefact (Resource) which provides the resource to identify.
//
// Returns string which is the unique identifier of the artefact.
func (*storageRenderer) GetID(artefact Resource) string {
	return artefact.ID
}

// MatchesFilter returns true if the artefact matches the search query.
//
// Takes artefact (Resource) which is the resource to check against the query.
// Takes query (string) which is the search term to match.
//
// Returns bool which is true if the artefact name or ID contains the query.
func (*storageRenderer) MatchesFilter(artefact Resource, query string) bool {
	return strings.Contains(strings.ToLower(artefact.Name), query) ||
		strings.Contains(strings.ToLower(artefact.ID), query)
}

// IsExpandable reports whether the artefact has metadata to show.
//
// Takes artefact (Resource) which is the resource to check for metadata.
//
// Returns bool which is true if the artefact has one or more metadata entries.
func (*storageRenderer) IsExpandable(artefact Resource) bool {
	return len(artefact.Metadata) > 0
}

// ExpandedLineCount returns the number of metadata lines.
//
// Takes artefact (Resource) which provides the resource to count lines for.
//
// Returns int which is the number of metadata entries in the artefact.
func (*storageRenderer) ExpandedLineCount(artefact Resource) int {
	return len(artefact.Metadata)
}
