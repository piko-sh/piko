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

package markdown_dto

// NavigationMetadata contains navigation-specific metadata for content
// organisation.
//
// Supports multiple named navigation groups (sidebar, footer, breadcrumb),
// allowing a single piece of content to appear in multiple navigation contexts
// with different metadata for each.
//
// Design Philosophy:
//   - Multi-context support: Content can appear differently in different navs
//   - Optional: All fields have sensible zero values
//   - Convention-friendly: Works with path-based defaults
type NavigationMetadata struct {
	// Groups maps group names to their navigation metadata.
	Groups map[string]*NavGroupMetadata
}

// NavGroupMetadata holds metadata for a single navigation group.
// It defines how a piece of content should appear within a specific
// navigation context.
type NavGroupMetadata struct {
	// Section is the top-level navigation group name, such as "get-started",
	// "api-reference", or "guides". If empty, it is derived from the first path
	// segment.
	Section string

	// Subsection is the secondary-level group within a section.
	// If empty, derived from second path segment.
	Subsection string

	// Icon is an optional icon name for display in the UI.
	//
	// The format depends on your icon system, for example: "download",
	// "🚀", or "fas fa-book".
	Icon string

	// Parent allows explicit override of the parent node.
	//
	// Format: "{section}" or "{section}/{subsection}"
	// If empty, derived from file path hierarchy.
	Parent string

	// Label overrides the page title in the navigation.
	//
	// When set, this value is used instead of the main title from
	// the frontmatter. Use it when the page title is long but the
	// navigation needs a short name.
	Label string

	// Order determines sort position within the current level.
	//
	// Lower values appear first. Default: 999 (appears last).
	Order int

	// Hidden indicates this item should not appear in navigation.
	// Useful for drafts, redirects, or old content.
	Hidden bool
}
