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

package collection_domain

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_dto"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// logKeyGroup is the log field key for the navigation group name.
	logKeyGroup = "group"

	// logKeySlug is the logging key for content item slugs.
	logKeySlug = "slug"

	// logKeySection is the logger key for section identifiers.
	logKeySection = "section"
)

// NavigationBuilder constructs hierarchical navigation trees from flat
// content lists. It uses single-pass O(n) construction per group and supports
// locale-aware, multi-group navigation such as sidebars, footers, and
// breadcrumbs.
type NavigationBuilder struct{}

// NewNavigationBuilder creates a new navigation builder.
//
// Returns *NavigationBuilder which is ready to build navigation structures.
func NewNavigationBuilder() *NavigationBuilder {
	return &NavigationBuilder{}
}

// BuildNavigationGroups creates navigation groups from content items.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes items ([]collection_dto.ContentItem) which contains the content to
// organise into navigation structures.
// Takes config (collection_dto.NavigationConfig) which specifies locale and
// grouping options.
//
// Returns *collection_dto.NavigationGroups which contains all named groups
// (sidebar, footer, and similar) with their navigation trees.
//
// The method first collects all group names from items, then builds a tree
// for each group. If config.Locale is set, only that locale is used.
// Otherwise, trees are built for all locales and the first is returned.
func (nb *NavigationBuilder) BuildNavigationGroups(
	ctx context.Context,
	items []collection_dto.ContentItem,
	config collection_dto.NavigationConfig,
) *collection_dto.NavigationGroups {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Building navigation groups",
		logger_domain.Int("item_count", len(items)),
		logger_domain.String("locale_filter", config.Locale))

	groupNames := nb.collectGroupNames(items)

	groups := &collection_dto.NavigationGroups{
		Groups: make(map[string]*collection_dto.NavigationTree),
	}

	for groupName := range groupNames {
		trees := nb.buildTreeForGroup(ctx, items, groupName, config)

		tree := nb.selectTreeForLocale(ctx, trees, config.Locale, groupName)
		if tree != nil {
			groups.Groups[groupName] = tree

			l.Trace("Built navigation tree for group",
				logger_domain.String(logKeyGroup, groupName),
				logger_domain.Int("locale_count", len(trees)),
				logger_domain.String("selected_locale", tree.Locale))
		}
	}

	l.Internal("Navigation groups built",
		logger_domain.Int("group_count", len(groups.Groups)))

	return groups
}

// selectTreeForLocale selects the appropriate tree based on locale.
//
// If a specific locale is requested, returns that locale's tree or nil if
// not found. If no locale is specified, returns the first available tree.
//
// Takes trees (map[string]*collection_dto.NavigationTree) which maps locale
// codes to their navigation trees.
// Takes requestedLocale (string) which specifies the desired locale, or empty
// for any available tree.
// Takes groupName (string) which identifies the navigation group for logging.
//
// Returns *collection_dto.NavigationTree which is the selected tree, or nil
// if no trees are available.
func (*NavigationBuilder) selectTreeForLocale(
	ctx context.Context,
	trees map[string]*collection_dto.NavigationTree,
	requestedLocale string,
	groupName string,
) *collection_dto.NavigationTree {
	_, l := logger_domain.From(ctx, log)
	if len(trees) == 0 {
		return nil
	}

	if requestedLocale != "" {
		if tree, ok := trees[requestedLocale]; ok {
			return tree
		}
		l.Trace("Requested locale not found in navigation trees, using fallback",
			logger_domain.String(logKeyGroup, groupName),
			logger_domain.String("requested_locale", requestedLocale))
	}

	for _, tree := range trees {
		return tree
	}

	return nil
}

// collectGroupNames finds all unique group names across the given items.
//
// Takes items ([]collection_dto.ContentItem) which contains the content items
// to scan for group names.
//
// Returns map[string]bool which is a set of group names found.
func (*NavigationBuilder) collectGroupNames(
	items []collection_dto.ContentItem,
) map[string]bool {
	groups := make(map[string]bool)
	for i := range items {
		extractGroupNamesFromItem(&items[i], groups)
	}
	return groups
}

// buildTreeForGroup constructs navigation trees for a single group across
// all locales.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes items ([]collection_dto.ContentItem) which contains the content items
// to organise into trees.
// Takes groupName (string) which identifies the navigation group.
// Takes config (collection_dto.NavigationConfig) which controls tree building
// behaviour.
//
// Returns map[string]*collection_dto.NavigationTree which maps locale codes
// to their navigation trees.
func (nb *NavigationBuilder) buildTreeForGroup(
	ctx context.Context,
	items []collection_dto.ContentItem,
	groupName string,
	config collection_dto.NavigationConfig,
) map[string]*collection_dto.NavigationTree {
	itemsByLocale := nb.groupByLocale(items)
	trees := make(map[string]*collection_dto.NavigationTree)

	for locale, localeItems := range itemsByLocale {
		tree := nb.buildTreeForLocale(ctx, localeItems, locale, groupName, config)
		trees[locale] = tree
	}

	return trees
}

// buildTreeForLocale constructs a navigation tree for a single locale.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes localeItems ([]collection_dto.ContentItem) which provides the content
// items to include in the tree.
// Takes locale (string) which identifies the locale for the tree.
// Takes groupName (string) which specifies the navigation group.
// Takes config (collection_dto.NavigationConfig) which controls tree building.
//
// Returns *collection_dto.NavigationTree which contains the sorted sections.
func (nb *NavigationBuilder) buildTreeForLocale(
	ctx context.Context,
	localeItems []collection_dto.ContentItem,
	locale, groupName string,
	config collection_dto.NavigationConfig,
) *collection_dto.NavigationTree {
	tree := &collection_dto.NavigationTree{
		Locale:   locale,
		Sections: []*collection_dto.NavigationNode{},
	}

	sections := nb.processItemsIntoSections(ctx, localeItems, groupName, config)
	tree.Sections = nb.sectionsMapToSlice(sections)
	nb.sortNodes(tree.Sections)

	logTreeStructure(ctx, tree, locale, groupName)
	return tree
}

// processItemsIntoSections processes content items into section nodes.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes items ([]collection_dto.ContentItem) which contains the items to
// process.
// Takes groupName (string) which identifies the group these items belong to.
// Takes config (collection_dto.NavigationConfig) which controls processing
// behaviour.
//
// Returns map[string]*collection_dto.NavigationNode which maps section names
// to their navigation nodes.
func (nb *NavigationBuilder) processItemsIntoSections(
	ctx context.Context,
	items []collection_dto.ContentItem,
	groupName string,
	config collection_dto.NavigationConfig,
) map[string]*collection_dto.NavigationNode {
	ctx, l := logger_domain.From(ctx, log)
	sections := make(map[string]*collection_dto.NavigationNode)
	var stats processingStats

	for i := range items {
		nb.processItem(ctx, &items[i], groupName, config, sections, &stats)
	}

	l.Trace("Items processing complete",
		logger_domain.String(logKeyGroup, groupName),
		logger_domain.Int("total_items", len(items)),
		logger_domain.Int("processed", stats.processed),
		logger_domain.Int("skipped_no_metadata", stats.skippedNoMeta),
		logger_domain.Int("skipped_hidden", stats.skippedHidden),
		logger_domain.Int("sections_created", len(sections)))

	return sections
}

// processingStats tracks counts of items processed, skipped, or hidden.
type processingStats struct {
	// processed is the count of items added to sections.
	processed int

	// skippedNoMeta counts items skipped because they have no group metadata.
	skippedNoMeta int

	// skippedHidden counts items that were skipped because they are marked as hidden.
	skippedHidden int
}

// processItem processes a single content item into the sections map.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes item (*collection_dto.ContentItem) which is the content to process.
// Takes groupName (string) which identifies the group containing the item.
// Takes config (collection_dto.NavigationConfig) which controls navigation
// behaviour.
// Takes sections (map[string]*collection_dto.NavigationNode) which collects
// the processed items by section.
// Takes stats (*processingStats) which tracks processing outcomes.
func (nb *NavigationBuilder) processItem(
	ctx context.Context,
	item *collection_dto.ContentItem,
	groupName string,
	config collection_dto.NavigationConfig,
	sections map[string]*collection_dto.NavigationNode,
	stats *processingStats,
) {
	ctx, l := logger_domain.From(ctx, log)
	groupMeta := nb.getGroupMetadata(ctx, item, groupName)
	if groupMeta == nil {
		stats.skippedNoMeta++
		return
	}

	if groupMeta.Hidden && !config.IncludeHidden {
		l.Trace("Skipping hidden item",
			logger_domain.String(logKeySlug, item.Slug),
			logger_domain.String(logKeyGroup, groupName))
		stats.skippedHidden++
		return
	}

	section := nb.ensureSection(ctx, sections, groupMeta.Section, config)
	nb.addItemToSection(ctx, section, item, groupMeta, config)
	stats.processed++
}

// addItemToSection adds an item to the correct place within a section.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes section (*collection_dto.NavigationNode) which is the parent section.
// Takes item (*collection_dto.ContentItem) which is the content to add.
// Takes groupMeta (*markdown_dto.NavGroupMetadata) which specifies grouping.
// Takes config (collection_dto.NavigationConfig) which controls navigation.
func (nb *NavigationBuilder) addItemToSection(
	ctx context.Context,
	section *collection_dto.NavigationNode,
	item *collection_dto.ContentItem,
	groupMeta *markdown_dto.NavGroupMetadata,
	config collection_dto.NavigationConfig,
) {
	if groupMeta.Subsection == "" {
		nb.addItemToNode(section, item, groupMeta, 1)
		return
	}
	subsection := nb.ensureSubsection(ctx, section, groupMeta.Subsection, config)
	nb.addItemToNode(subsection, item, groupMeta, 2)
}

// sectionsMapToSlice converts a sections map to a slice.
//
// Takes sections (map[string]*collection_dto.NavigationNode) which contains
// the navigation nodes keyed by their identifiers.
//
// Returns []*collection_dto.NavigationNode which contains all nodes from the
// map in an unspecified order.
func (*NavigationBuilder) sectionsMapToSlice(
	sections map[string]*collection_dto.NavigationNode,
) []*collection_dto.NavigationNode {
	result := make([]*collection_dto.NavigationNode, 0, len(sections))
	for _, section := range sections {
		result = append(result, section)
	}
	return result
}

// groupByLocale organises items by their locale.
//
// Takes items ([]collection_dto.ContentItem) which contains the items to group.
//
// Returns map[string][]collection_dto.ContentItem which maps locale codes to
// their items. Items without a locale default to "en".
func (*NavigationBuilder) groupByLocale(
	items []collection_dto.ContentItem,
) map[string][]collection_dto.ContentItem {
	grouped := make(map[string][]collection_dto.ContentItem)

	for i := range items {
		locale := cmp.Or(items[i].Locale, "en")
		grouped[locale] = append(grouped[locale], items[i])
	}

	return grouped
}

// getGroupMetadata retrieves navigation metadata for a specific group from a
// content item.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes item (*collection_dto.ContentItem) which provides the content with
// navigation metadata.
// Takes groupName (string) which specifies the navigation group to look up.
//
// Returns *markdown_dto.NavGroupMetadata which contains the group's navigation
// settings, or nil if the group is not found.
func (*NavigationBuilder) getGroupMetadata(
	ctx context.Context,
	item *collection_dto.ContentItem,
	groupName string,
) *markdown_dto.NavGroupMetadata {
	_, l := logger_domain.From(ctx, log)
	navData, ok := item.Metadata[collection_dto.MetaKeyNavigation]
	if !ok {
		l.Trace("No Navigation field in metadata",
			logger_domain.String(logKeySlug, item.Slug),
			logger_domain.String(logKeyGroup, groupName))
		return nil
	}

	if nav, ok := navData.(*markdown_dto.NavigationMetadata); ok {
		return nav.Groups[groupName]
	}

	navMap, ok := navData.(map[string]any)
	if !ok {
		l.Warn("Navigation data is neither struct nor map",
			logger_domain.String(logKeySlug, item.Slug),
			logger_domain.String("type", fmt.Sprintf("%T", navData)))
		return nil
	}

	groupsMap, ok := navMap["Groups"].(map[string]any)
	if !ok {
		l.Warn("No 'Groups' key or wrong type",
			logger_domain.String(logKeySlug, item.Slug),
			logger_domain.Strings("available_keys", getMapKeys(navMap)))
		return nil
	}

	groupData, ok := groupsMap[groupName].(map[string]any)
	if !ok {
		l.Trace("Group name not found in Groups map",
			logger_domain.String(logKeySlug, item.Slug),
			logger_domain.String(logKeyGroup, groupName),
			logger_domain.Strings("available_groups", getMapKeys(groupsMap)))
		return nil
	}

	meta := &markdown_dto.NavGroupMetadata{
		Section:    getStringFromMap(groupData, "Section"),
		Subsection: getStringFromMap(groupData, "Subsection"),
		Order:      getIntFromMap(groupData, "Order"),
		Icon:       getStringFromMap(groupData, "Icon"),
		Hidden:     getBoolFromMap(groupData, "Hidden"),
		Parent:     getStringFromMap(groupData, "Parent"),
		Label:      getStringFromMap(groupData, "Label"),
	}

	return meta
}

// ensureSection gets or creates a section node.
//
// Sections are top-level category nodes in the navigation tree.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes sections (map[string]*collection_dto.NavigationNode) which holds
// existing section nodes keyed by their ID.
// Takes sectionID (string) which identifies the section to get or create.
// Takes config (collection_dto.NavigationConfig) which provides default values
// for new sections.
//
// Returns *collection_dto.NavigationNode which is the existing or newly created
// section node.
func (nb *NavigationBuilder) ensureSection(
	ctx context.Context,
	sections map[string]*collection_dto.NavigationNode,
	sectionID string,
	config collection_dto.NavigationConfig,
) *collection_dto.NavigationNode {
	_, l := logger_domain.From(ctx, log)
	if node, exists := sections[sectionID]; exists {
		l.Trace("Section already exists",
			logger_domain.String(logKeySection, sectionID),
			logger_domain.Int("child_count", len(node.Children)))
		return node
	}

	node := &collection_dto.NavigationNode{
		ID:          sectionID,
		Title:       nb.humanise(sectionID),
		Section:     sectionID,
		Subsection:  "",
		Level:       0,
		Parent:      nil,
		Children:    []*collection_dto.NavigationNode{},
		URL:         "",
		Icon:        "",
		Order:       config.DefaultOrder,
		Hidden:      false,
		ContentItem: nil,
	}

	sections[sectionID] = node
	l.Trace("Created new section",
		logger_domain.String(logKeySection, sectionID),
		logger_domain.String("title", node.Title))
	return node
}

// ensureSubsection gets or creates a subsection node within a section.
//
// Subsections are second-level category nodes.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes sectionNode (*collection_dto.NavigationNode) which is the parent
// section to search or add the subsection to.
// Takes subsectionID (string) which identifies the subsection to find or
// create.
// Takes config (collection_dto.NavigationConfig) which provides default
// ordering for new subsections.
//
// Returns *collection_dto.NavigationNode which is the existing or newly
// created subsection node.
func (nb *NavigationBuilder) ensureSubsection(
	ctx context.Context,
	sectionNode *collection_dto.NavigationNode,
	subsectionID string,
	config collection_dto.NavigationConfig,
) *collection_dto.NavigationNode {
	_, l := logger_domain.From(ctx, log)
	for _, child := range sectionNode.Children {
		if child.ID == subsectionID && child.ContentItem == nil {
			l.Trace("Subsection already exists",
				logger_domain.String(logKeySection, sectionNode.Section),
				logger_domain.String("subsection", subsectionID),
				logger_domain.Int("child_count", len(child.Children)))
			return child
		}
	}

	node := &collection_dto.NavigationNode{
		ID:          subsectionID,
		Title:       nb.humanise(subsectionID),
		Section:     sectionNode.Section,
		Subsection:  subsectionID,
		Level:       1,
		Parent:      sectionNode,
		Children:    []*collection_dto.NavigationNode{},
		URL:         "",
		Icon:        "",
		Order:       config.DefaultOrder,
		Hidden:      false,
		ContentItem: nil,
	}

	sectionNode.Children = append(sectionNode.Children, node)
	l.Trace("Created new subsection",
		logger_domain.String(logKeySection, sectionNode.Section),
		logger_domain.String("subsection", subsectionID),
		logger_domain.String("title", node.Title))
	return node
}

// addItemToNode adds a content item as a child of the given parent node.
//
// Takes parentNode (*collection_dto.NavigationNode) which is the parent node
// to attach the new child to.
// Takes item (*collection_dto.ContentItem) which provides the content data.
// Takes navMeta (*markdown_dto.NavGroupMetadata) which provides the display
// settings for navigation.
// Takes level (int) which sets the depth in the navigation tree.
func (*NavigationBuilder) addItemToNode(
	parentNode *collection_dto.NavigationNode,
	item *collection_dto.ContentItem,
	navMeta *markdown_dto.NavGroupMetadata,
	level int,
) {
	title := item.GetMetadataString(collection_dto.MetaKeyTitle, item.Slug)

	if navMeta.Label != "" {
		title = navMeta.Label
	}

	node := &collection_dto.NavigationNode{
		ID:          item.Slug,
		Title:       title,
		Section:     navMeta.Section,
		Subsection:  navMeta.Subsection,
		Level:       level,
		Parent:      parentNode,
		Children:    []*collection_dto.NavigationNode{},
		URL:         item.URL,
		Icon:        navMeta.Icon,
		Order:       navMeta.Order,
		Hidden:      navMeta.Hidden,
		ContentItem: item,
	}

	parentNode.Children = append(parentNode.Children, node)
}

// sortNodes recursively sorts navigation nodes by Order (ascending), then by
// Title (alphabetical).
//
// Takes nodes ([]*collection_dto.NavigationNode) which is the slice of nodes
// to sort in place.
//
// This produces consistent, predictable navigation order.
func (nb *NavigationBuilder) sortNodes(nodes []*collection_dto.NavigationNode) {
	titleCache := make(map[*collection_dto.NavigationNode]string, len(nodes))
	for _, n := range nodes {
		titleCache[n] = strings.ToLower(n.Title)
	}

	slices.SortFunc(nodes, func(a, b *collection_dto.NavigationNode) int {
		return cmp.Or(
			cmp.Compare(a.Order, b.Order),
			cmp.Compare(titleCache[a], titleCache[b]),
		)
	})

	for _, node := range nodes {
		if len(node.Children) > 0 {
			nb.sortNodes(node.Children)
		}
	}
}

// humanise converts a slug identifier to a human-readable title.
//
// Takes slug (string) which is the identifier to convert.
//
// Returns string which is the title-cased human-readable form.
func (*NavigationBuilder) humanise(slug string) string {
	s := strings.ReplaceAll(slug, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// extractGroupNamesFromItem extracts group names from a single content item.
//
// Takes item (*collection_dto.ContentItem) which is the content item to get
// group names from.
// Takes groups (map[string]bool) which collects the found group names.
func extractGroupNamesFromItem(item *collection_dto.ContentItem, groups map[string]bool) {
	navData, ok := item.Metadata[collection_dto.MetaKeyNavigation]
	if !ok {
		return
	}

	if nav, ok := navData.(*markdown_dto.NavigationMetadata); ok {
		extractGroupNamesFromTypedNav(nav, groups)
		return
	}

	if navMap, ok := navData.(map[string]any); ok {
		extractGroupNamesFromMapNav(navMap, groups)
	}
}

// extractGroupNamesFromTypedNav gets group names from navigation metadata and
// adds them to a map.
//
// Takes nav (*markdown_dto.NavigationMetadata) which holds the navigation data
// with group definitions.
// Takes groups (map[string]bool) which is the map to fill with group names.
func extractGroupNamesFromTypedNav(nav *markdown_dto.NavigationMetadata, groups map[string]bool) {
	for groupName := range nav.Groups {
		groups[groupName] = true
	}
}

// extractGroupNamesFromMapNav extracts group names from map-based navigation
// metadata.
//
// Takes navMap (map[string]any) which contains navigation metadata with a
// Groups key.
// Takes groups (map[string]bool) which collects found group names.
func extractGroupNamesFromMapNav(navMap map[string]any, groups map[string]bool) {
	groupsMap, ok := navMap["Groups"].(map[string]any)
	if !ok {
		return
	}
	for groupName := range groupsMap {
		groups[groupName] = true
	}
}

// logTreeStructure logs the tree structure for debugging purposes.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes tree (*collection_dto.NavigationTree) which holds the navigation
// structure to log.
// Takes locale (string) which identifies the locale being processed.
// Takes groupName (string) which identifies the collection group.
func logTreeStructure(ctx context.Context, tree *collection_dto.NavigationTree, locale, groupName string) {
	_, l := logger_domain.From(ctx, log)
	const maxSectionsToLog = 3
	const maxChildrenToLog = 2

	for i, section := range tree.Sections {
		if i >= maxSectionsToLog {
			break
		}
		l.Trace("Section structure",
			logger_domain.String(logKeySection, section.ID),
			logger_domain.String("title", section.Title),
			logger_domain.Int("direct_children", len(section.Children)))

		for j, child := range section.Children {
			if j >= maxChildrenToLog {
				break
			}
			l.Trace("  Child node",
				logger_domain.String("child_id", child.ID),
				logger_domain.String("child_title", child.Title),
				logger_domain.String("child_url", child.URL),
				logger_domain.Int("grandchildren", len(child.Children)))
		}
	}

	l.Trace("Built tree for locale",
		logger_domain.String("locale", locale),
		logger_domain.String(logKeyGroup, groupName),
		logger_domain.Int("section_count", len(tree.Sections)))
}

// getMapKeys returns all keys from a map for debugging purposes.
//
// Takes m (map[string]any) which is the map to extract keys from.
//
// Returns []string which contains all keys from the map in no set order.
func getMapKeys(m map[string]any) []string {
	return slices.Collect(maps.Keys(m))
}

// getStringFromMap gets a string value from a map by key.
//
// Takes m (map[string]any) which is the map to get the value from.
// Takes key (string) which is the key to look up.
//
// Returns string which is the value if found and is a string, or an empty
// string if not found or not a string.
func getStringFromMap(m map[string]any, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

// getIntFromMap gets an integer value from a map using the given key.
//
// Handles int, int8, int16, int32, int64, uint, uint8, uint16, uint32,
// uint64, float32, float64, and string types to account for different JSON
// decoders (e.g. sonic with UseInt64 returns int64 instead of float64).
//
// Takes m (map[string]any) which is the map to search.
// Takes key (string) which is the key to look up.
//
// Returns int which is the value at the key, or 0 if not found or not a
// number.
func getIntFromMap(m map[string]any, key string) int {
	v := m[key]
	if v == nil {
		return 0
	}

	switch value := v.(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	case uint:
		return safeconv.Uint64ToInt(uint64(value))
	case uint8:
		return int(value)
	case uint16:
		return int(value)
	case uint32:
		return int(value)
	case uint64:
		return safeconv.Uint64ToInt(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case string:
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}

	return 0
}

// getBoolFromMap retrieves a boolean value from a map by key.
//
// Handles bool, int types (0 = false, non-zero = true), and string
// representations ("true", "1", etc.) to account for different JSON
// decoders and data sources.
//
// Takes m (map[string]any) which is the map to search in.
// Takes key (string) which is the key to look up.
//
// Returns bool which is the value if found, or false if the key does not
// exist or the value is not a boolean.
func getBoolFromMap(m map[string]any, key string) bool {
	v := m[key]
	if v == nil {
		return false
	}

	switch value := v.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	case string:
		return value == "true" || value == "1"
	}

	return false
}
