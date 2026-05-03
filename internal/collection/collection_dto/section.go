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

const (
	// defaultSectionMinLevel is the smallest heading level used when parsing sections.
	defaultSectionMinLevel = 2

	// defaultSectionMaxLevel is the deepest heading level to include in the tree.
	defaultSectionMaxLevel = 4
)

// SectionNode represents a section entry in a table of contents tree.
//
// Unlike flat section data from content providers, SectionNode holds nested
// children to build a tree structure for navigation. It works with any content
// source such as markdown or CMS systems.
type SectionNode struct {
	// Title is the heading text, such as "Introduction".
	Title string

	// Slug is the URL-safe anchor ID (e.g., "getting-started").
	Slug string

	// Children holds nested sections, such as h3 headings under an h2.
	Children []SectionNode

	// Level is the heading level from 2 to 6, where 2 means h2.
	Level int
}

// SectionTreeConfig controls how section trees are built from flat section lists.
//
// This configuration is passed to BuildSectionTree to customise filtering
// and hierarchy generation.
//
// Design Philosophy:
//   - Sensible defaults: Works out-of-the-box for typical ToC use cases
//   - Explicit filtering: Clear min/max level boundaries
//   - Provider-agnostic: Works identically for any content source
//
// Default Values:
//   - MinLevel: 2 (start at h2, skip h1 page title)
//   - MaxLevel: 4 (include up to h4)
type SectionTreeConfig struct {
	// MinLevel is the smallest heading level to include in the tree.
	//
	// Headings below this level are filtered out. Default is 2, which starts at
	// h2 and skips h1 (typically the page title).
	MinLevel int

	// MaxLevel is the highest heading level to include in the tree.
	//
	// Headings with a level greater than MaxLevel are not included.
	// Default: 4 (includes h2, h3, h4).
	MaxLevel int
}

// DefaultSectionTreeConfig returns a SectionTreeConfig with sensible defaults.
//
// Defaults:
//   - MinLevel: 2 (skips h1 page title)
//   - MaxLevel: 4 (includes h2, h3, h4)
//
// Returns SectionTreeConfig which contains the default level settings.
func DefaultSectionTreeConfig() SectionTreeConfig {
	return SectionTreeConfig{
		MinLevel: defaultSectionMinLevel,
		MaxLevel: defaultSectionMaxLevel,
	}
}
