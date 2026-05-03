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
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

// SectionTreeOption is a functional option for configuring BuildSectionTree.
type SectionTreeOption func(*collection_dto.SectionTreeConfig)

// WithMinLevel sets the minimum heading level to include.
//
// Headings below this level are not included in the tree. The default is 2.
//
// Takes level (int) which specifies the minimum heading level.
//
// Returns SectionTreeOption which configures the section tree builder.
func WithMinLevel(level int) SectionTreeOption {
	return func(config *collection_dto.SectionTreeConfig) {
		config.MinLevel = level
	}
}

// WithMaxLevel sets the maximum heading level to include (default: 4). Headings
// above this level are filtered out.
//
// Takes level (int) which specifies the maximum heading level.
//
// Returns SectionTreeOption which configures the section tree builder.
func WithMaxLevel(level int) SectionTreeOption {
	return func(config *collection_dto.SectionTreeConfig) {
		config.MaxLevel = level
	}
}

// BuildSectionTree converts a flat list of sections into a hierarchical tree.
//
// Transforms flat section data from any content provider into a nested tree
// structure suitable for rendering a table of contents. Provider-agnostic and
// returns collection_dto.SectionNode regardless of the source.
//
// When sections is empty or no sections match the filter criteria, returns nil.
//
// Takes sections ([]markdown_dto.SectionData) which is a flat list of sections
// from markdown, CMS, or any content provider.
// Takes opts (...SectionTreeOption) which provides optional configuration such
// as WithMinLevel and WithMaxLevel.
//
// Returns []collection_dto.SectionNode which contains top-level nodes with
// nested children preserving the heading hierarchy.
//
// Default behaviour uses MinLevel 2 (starts at h2, skips h1 page title) and
// MaxLevel 4 (includes up to h4).
func BuildSectionTree(sections []markdown_dto.SectionData, opts ...SectionTreeOption) []collection_dto.SectionNode {
	config := collection_dto.DefaultSectionTreeConfig()

	for _, opt := range opts {
		opt(&config)
	}

	if len(sections) == 0 {
		return nil
	}

	filtered := filterSectionsByLevel(sections, config.MinLevel, config.MaxLevel)
	if len(filtered) == 0 {
		return nil
	}

	return buildSectionHierarchy(filtered, config.MinLevel, config.MaxLevel)
}

// filterSectionsByLevel returns only sections within the given level range.
//
// Takes sections ([]markdown_dto.SectionData) which contains the sections to
// filter.
// Takes minLevel (int) which sets the lowest heading level to include.
// Takes maxLevel (int) which sets the highest heading level to include.
//
// Returns []markdown_dto.SectionData which contains only sections where the
// level is between minLevel and maxLevel, including both.
func filterSectionsByLevel(sections []markdown_dto.SectionData, minLevel, maxLevel int) []markdown_dto.SectionData {
	result := make([]markdown_dto.SectionData, 0, len(sections))
	for _, s := range sections {
		if s.Level >= minLevel && s.Level <= maxLevel {
			result = append(result, s)
		}
	}
	return result
}

// buildSectionHierarchy converts a filtered flat list into a nested tree
// structure.
//
// Takes sections ([]markdown_dto.SectionData) which is the filtered list
// of sections to organise into a tree.
// Takes maxLevel (int) which limits the deepest heading level to include
// in the hierarchy.
//
// Returns []collection_dto.SectionNode which contains the top-level nodes
// with nested children, or nil if sections is empty.
func buildSectionHierarchy(sections []markdown_dto.SectionData, _, maxLevel int) []collection_dto.SectionNode {
	if len(sections) == 0 {
		return nil
	}

	rootLevel := maxLevel + 1
	for _, s := range sections {
		if s.Level < rootLevel {
			rootLevel = s.Level
		}
	}

	items, _ := buildSectionLevel(sections, 0, rootLevel, maxLevel)
	return items
}

// buildSectionLevel builds section nodes for a given heading level.
//
// It walks through a flat list of sections and groups them into a tree. When
// it finds a section at the current level, it creates a node. When it finds a
// deeper level, it builds child nodes. It stops when it finds a shallower
// level.
//
// Takes sections ([]markdown_dto.SectionData) which contains the flat list of
// sections to process.
// Takes startIndex (int) which specifies where to start processing.
// Takes currentLevel (int) which indicates the heading level being built.
// Takes maxLevel (int) which limits the maximum depth of the tree.
//
// Returns []collection_dto.SectionNode which contains the built nodes for this
// level.
// Returns int which is the index where processing stopped.
func buildSectionLevel(sections []markdown_dto.SectionData, startIndex, currentLevel, maxLevel int) ([]collection_dto.SectionNode, int) {
	hint := 0
	for j := startIndex; j < len(sections) && sections[j].Level >= currentLevel; j++ {
		if sections[j].Level == currentLevel {
			hint++
		}
	}
	items := make([]collection_dto.SectionNode, 0, hint)
	i := startIndex

	for i < len(sections) {
		section := sections[i]

		if section.Level < currentLevel {
			break
		}

		if section.Level > currentLevel {
			items, i = handleDeeperSection(sections, items, i, currentLevel, maxLevel)
			continue
		}

		items, i = handleCurrentSection(sections, items, i, currentLevel, maxLevel)
	}

	return items, i
}

// handleDeeperSection handles a section that is deeper than the current level.
// It either creates an orphan node or attaches children to the last item.
//
// Takes sections ([]markdown_dto.SectionData) which contains all the
// section data being processed.
// Takes items ([]collection_dto.SectionNode) which holds the nodes built
// so far at the current level.
// Takes index (int) which is the current position in the sections slice.
// Takes maxLevel (int) which is the deepest heading level to include.
//
// Returns []collection_dto.SectionNode which is the updated list of nodes
// with the deeper section handled.
// Returns int which is the next index to process after this section.
func handleDeeperSection(
	sections []markdown_dto.SectionData,
	items []collection_dto.SectionNode,
	index, _, maxLevel int,
) ([]collection_dto.SectionNode, int) {
	section := sections[index]

	if len(items) == 0 {
		item, newIndex := createOrphanSection(sections, index, maxLevel)
		return append(items, item), newIndex
	}

	lastIndex := len(items) - 1
	children, newIndex := buildSectionLevel(sections, index, section.Level, maxLevel)
	items[lastIndex].Children = children
	return items, newIndex
}

// createOrphanSection creates a section node for a section that has no parent
// at a higher level.
//
// Takes sections ([]markdown_dto.SectionData) which contains the section data
// to process.
// Takes index (int) which specifies the current position in the sections slice.
// Takes maxLevel (int) which defines the deepest heading level to include.
//
// Returns collection_dto.SectionNode which is the built node with any nested
// children.
// Returns int which is the updated index after processing.
func createOrphanSection(sections []markdown_dto.SectionData, index, maxLevel int) (collection_dto.SectionNode, int) {
	section := sections[index]
	item := sectionDataToNode(section)
	index++

	if section.Level < maxLevel && index < len(sections) {
		children, newIndex := buildSectionLevel(sections, index, section.Level+1, maxLevel)
		item.Children = children
		index = newIndex
	}
	return item, index
}

// handleCurrentSection processes a single section at the current heading level.
//
// Takes sections ([]markdown_dto.SectionData) which contains all sections to
// process.
// Takes items ([]collection_dto.SectionNode) which holds the nodes built so
// far.
// Takes index (int) which is the current position in the sections slice.
// Takes currentLevel (int) which is the heading level being processed.
// Takes maxLevel (int) which is the deepest heading level to include.
//
// Returns []collection_dto.SectionNode which contains the updated list with
// the new node added.
// Returns int which is the next index to process after this section.
func handleCurrentSection(
	sections []markdown_dto.SectionData,
	items []collection_dto.SectionNode,
	index, currentLevel, maxLevel int,
) ([]collection_dto.SectionNode, int) {
	item := sectionDataToNode(sections[index])
	index++

	if canHaveChildren(sections, index, currentLevel, maxLevel) {
		children, newIndex := buildSectionLevel(sections, index, sections[index].Level, maxLevel)
		item.Children = children
		index = newIndex
	}

	return append(items, item), index
}

// canHaveChildren checks if child sections can be collected at this position.
//
// Takes sections ([]markdown_dto.SectionData) which holds all section data.
// Takes index (int) which is the current position in the sections slice.
// Takes currentLevel (int) which is the nesting level being processed.
// Takes maxLevel (int) which is the maximum allowed nesting depth.
//
// Returns bool which is true when children can exist at this position.
func canHaveChildren(sections []markdown_dto.SectionData, index, currentLevel, maxLevel int) bool {
	return currentLevel < maxLevel && index < len(sections) && sections[index].Level > currentLevel
}

// sectionDataToNode converts a flat SectionData into a SectionNode.
//
// Takes section (markdown_dto.SectionData) which provides the source data.
//
// Returns collection_dto.SectionNode which is the converted node with nil
// children.
func sectionDataToNode(section markdown_dto.SectionData) collection_dto.SectionNode {
	return collection_dto.SectionNode{
		Title:    section.Title,
		Slug:     section.Slug,
		Level:    section.Level,
		Children: nil,
	}
}
