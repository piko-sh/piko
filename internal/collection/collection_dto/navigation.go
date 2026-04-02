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

package collection_dto

// defaultNavigationOrder is the default sort order for navigation items.
const defaultNavigationOrder = 999

// NavigationGroups contains multiple named navigation structures.
//
// Each group represents a distinct navigation UI component, enabling a single
// collection to power multiple navigation contexts (e.g., sidebar, footer,
// breadcrumbs).
//
// Design Philosophy:
//   - Multi-context support: One collection can drive multiple navigation UIs
//   - Named groups: Explicit group names prevent conflicts and aid debugging
//   - Provider-agnostic: Works identically for markdown and headless CMS
type NavigationGroups struct {
	// Groups maps each group name to its navigation tree. Group names are set in
	// content frontmatter, such as "sidebar", "footer", or "breadcrumb".
	Groups map[string]*NavigationTree
}

// NavigationTree represents a hierarchical navigation structure for a specific
// group and locale.
//
// The tree is pre-sorted and ready for rendering. All nodes are organised into
// top-level sections, which may contain subsections and items.
//
// Design Characteristics:
//   - Locale-specific: Each tree represents one language
//   - Pre-sorted: Nodes sorted by Order field, then Title
//   - Hierarchical: Section -> Subsection -> Item structure
//   - Renderable: Direct iteration in templates
type NavigationTree struct {
	// Locale is the ISO 639-1 language code for this tree (e.g. "en", "fr", "de").
	Locale string

	// Sections holds the top-level navigation groups.
	//
	// These are the root nodes of the tree. Each one represents a major
	// section or content category. Sorted by Order (ascending), then
	// Title (alphabetically).
	Sections []*NavigationNode
}

// NavigationNode represents a single node in the navigation hierarchy.
//
// Nodes can represent:
//   - Category nodes: Grouping containers (no URL, has children)
//   - Content nodes: Actual pages/documents (has URL, may have children)
//   - Section nodes: Top-level organisational units
//
// Design Philosophy:
//   - Self-describing: All metadata needed for rendering
//   - Recursive: Children are also NavigationNodes (tree structure)
//   - Type-agnostic: Check URL to distinguish categories from content
type NavigationNode struct {
	// Parent is the parent node; nil for root-level sections.
	// Used for breadcrumb generation and tree traversal.
	Parent *NavigationNode

	// ContentItem is a reference to the full content item this node represents.
	//
	// nil for category nodes (pure grouping, no content).
	// Populated for content nodes.
	//
	// Provides access to full metadata, AST, excerpt, etc.
	ContentItem *ContentItem

	// ID is the unique identifier for this node within the tree.
	//
	// For content nodes, this is usually the slug (e.g. "installation").
	// For category nodes, this is the section or subsection name
	// (e.g. "get-started").
	ID string

	// Title is the display name shown in navigation menus.
	//
	// The value is chosen from these sources, in order of priority:
	//  1. nav.label from frontmatter (if set)
	//  2. title from frontmatter
	//  3. A readable form of the slug (e.g. "get-started" becomes "Get Started")
	Title string

	// Section is the top-level section this node belongs to.
	Section string

	// Subsection is the second-level section within a section.
	//
	// Empty string if this node is directly under a section.
	Subsection string

	// URL is the link target for this node. Empty for category nodes that only
	// group other nodes; set for content nodes such as pages or documents.
	URL string

	// Icon is an optional icon name for showing in the user interface.
	//
	// The format is flexible to work with different frontends:
	//   - Icon set name: "download", "book", "code", "settings"
	//   - Emoji name: "rocket", "book", "gear"
	//   - CSS class: "fas fa-download"
	//
	// An empty string means no icon is set.
	Icon string

	// Children are the child nodes of this node.
	//
	// Empty slice for leaf nodes (actual content pages).
	// Non-empty for category/section nodes.
	//
	// Pre-sorted by Order (ascending), then Title (alphabetical).
	Children []*NavigationNode

	// Level indicates the depth in the tree hierarchy. 0 is the top-level
	// section, 1 is a direct child of a section, 2 is a child of a subsection,
	// and 3 or higher is reserved for future use.
	Level int

	// Order sets the sort position within the parent node.
	//
	// Lower values appear first. Nodes with the same Order value are sorted
	// by Title in alphabetical order.
	//
	// Default: 999 (appears last).
	Order int

	// Hidden indicates whether this node should be left out of navigation.
	//
	// Use cases:
	//   - Draft pages not ready for public navigation
	//   - Redirect pages that should not appear in menus
	//   - Old content kept for backward compatibility
	//
	// Hidden nodes are usually filtered out during tree building, so this
	// field may always be false in practice.
	Hidden bool
}

// NavigationConfig controls navigation tree building behaviour.
//
// This configuration is passed to NavigationBuilder to customise how
// navigation hierarchies are constructed from flat content lists.
//
// Design Philosophy:
//   - Sensible defaults: Works out-of-the-box for most use cases
//   - Explicit control: Each option has a clear, single purpose
//   - Backward compatible: Adding options doesn't break existing code
type NavigationConfig struct {
	// Locale filters the navigation tree to show items from one locale only.
	//
	// When empty, items from all locales are included. This returns one tree per
	// group. If there are items in more than one locale, the first locale found
	// is used (the order is not fixed).
	//
	// When set, only items that match this locale are included. This gives you a
	// navigation tree for that locale only.
	//
	// Always set Locale when building navigation for a page to get the correct
	// locale-specific navigation.
	Locale string

	// DefaultOrder is the fallback order value for items without explicit
	// ordering. Default is 999 so that unordered items appear after ordered ones.
	DefaultOrder int

	// IncludeHidden determines whether hidden items are included in the tree.
	//
	// false (default): Hidden items are filtered out
	// true: Hidden items are included (useful for admin views)
	IncludeHidden bool

	// GroupBySection controls whether empty section category nodes appear.
	//
	// When true (default), empty sections appear as category nodes.
	// When false, only sections with content appear.
	GroupBySection bool
}

// IsCategory reports whether this node is a category (grouping) node.
//
// A category node has no URL and exists only to group other nodes.
//
// Returns bool which is true if the node is a category.
func (n *NavigationNode) IsCategory() bool {
	return n.URL == ""
}

// IsLeaf returns true if this node has no children.
//
// Leaf nodes are typically actual content pages, while non-leaf nodes
// are categories or sections.
//
// Returns bool which is true when the node has no children.
func (n *NavigationNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// HasContent returns true if this node represents actual content.
//
// Content nodes have a non-nil ContentItem reference and typically have a URL.
//
// Returns bool which is true when the node has a ContentItem reference.
func (n *NavigationNode) HasContent() bool {
	return n.ContentItem != nil
}

// GetBreadcrumb returns the breadcrumb trail from root to this node.
//
// The returned slice starts with the root ancestor and ends with this node.
//
// Returns []*NavigationNode which is the path from root to this node.
func (n *NavigationNode) GetBreadcrumb() []*NavigationNode {
	var breadcrumb []*NavigationNode

	current := n
	for current != nil {
		breadcrumb = append([]*NavigationNode{current}, breadcrumb...)
		current = current.Parent
	}

	return breadcrumb
}

// CountDescendants returns the total number of descendant nodes.
//
// This includes direct children and all nested children recursively.
//
// Returns int which is the total count of all descendant nodes.
func (n *NavigationNode) CountDescendants() int {
	count := len(n.Children)

	for _, child := range n.Children {
		count += child.CountDescendants()
	}

	return count
}

// FindNodeByID searches the tree for a node with the given ID.
//
// Takes id (string) which specifies the node identifier to find.
//
// Returns *NavigationNode which is the matching node, or nil if not found.
//
// Search is depth-first recursive.
func (n *NavigationNode) FindNodeByID(id string) *NavigationNode {
	if n.ID == id {
		return n
	}

	for _, child := range n.Children {
		if found := child.FindNodeByID(id); found != nil {
			return found
		}
	}

	return nil
}

// DefaultNavigationConfig returns a NavigationConfig with sensible defaults.
//
// This is the configuration used when none is explicitly provided.
//
// Defaults:
//   - IncludeHidden: false
//   - DefaultOrder: 999
//   - GroupBySection: true
//
// Returns NavigationConfig which contains the default navigation settings.
func DefaultNavigationConfig() NavigationConfig {
	return NavigationConfig{
		IncludeHidden:  false,
		DefaultOrder:   defaultNavigationOrder,
		GroupBySection: true,
	}
}
