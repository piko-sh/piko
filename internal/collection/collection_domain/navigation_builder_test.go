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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

func TestNavigationBuilder_BuildNavigationGroups(t *testing.T) {
	builder := NewNavigationBuilder()

	t.Run("SimpleTwoLevelHierarchy", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "intro",
				Locale: "en",
				URL:    "/docs/intro",
				Metadata: map[string]any{
					"Title": "Introduction",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "get-started",
								Order:   1,
							},
						},
					},
				},
			},
			{
				Slug:   "install",
				Locale: "en",
				URL:    "/docs/install",
				Metadata: map[string]any{
					"Title": "Installation",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "get-started",
								Order:   2,
							},
						},
					},
				},
			},
			{
				Slug:   "api-intro",
				Locale: "en",
				URL:    "/docs/api-intro",
				Metadata: map[string]any{
					"Title": "API Introduction",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "api",
								Order:   1,
							},
						},
					},
				},
			},
		}

		config := collection_dto.NavigationConfig{
			IncludeHidden:  false,
			DefaultOrder:   999,
			GroupBySection: true,
		}

		groups := builder.BuildNavigationGroups(context.Background(), items, config)
		require.NotNil(t, groups)
		require.Contains(t, groups.Groups, "sidebar")

		tree := groups.Groups["sidebar"]
		assert.Equal(t, "en", tree.Locale)
		assert.Len(t, tree.Sections, 2, "Should have 2 sections: get-started and api")

		var getStartedSection *collection_dto.NavigationNode
		for _, section := range tree.Sections {
			if section.Section == "get-started" {
				getStartedSection = section
				break
			}
		}
		require.NotNil(t, getStartedSection)
		assert.Len(t, getStartedSection.Children, 2, "Should have intro and install")
		assert.Equal(t, "intro", getStartedSection.Children[0].ID)
		assert.Equal(t, "install", getStartedSection.Children[1].ID)
	})

	t.Run("ThreeLevelHierarchyWithSubsections", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "renderer",
				Locale: "en",
				URL:    "/docs/api/core/renderer",
				Metadata: map[string]any{
					"Title": "Renderer",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section:    "api",
								Subsection: "core",
								Order:      1,
							},
						},
					},
				},
			},
			{
				Slug:   "parser",
				Locale: "en",
				URL:    "/docs/api/core/parser",
				Metadata: map[string]any{
					"Title": "Parser",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section:    "api",
								Subsection: "core",
								Order:      2,
							},
						},
					},
				},
			},
			{
				Slug:   "http-client",
				Locale: "en",
				URL:    "/docs/api/adapters/http-client",
				Metadata: map[string]any{
					"Title": "HTTP Client",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section:    "api",
								Subsection: "adapters",
								Order:      1,
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		tree := groups.Groups["sidebar"]
		assert.Len(t, tree.Sections, 1, "Should have 1 section: api")

		apiSection := tree.Sections[0]
		assert.Equal(t, "api", apiSection.Section)
		assert.Len(t, apiSection.Children, 2, "Should have 2 subsections: core and adapters")

		var coreSubsection *collection_dto.NavigationNode
		for _, child := range apiSection.Children {
			if child.Subsection == "core" {
				coreSubsection = child
				break
			}
		}
		require.NotNil(t, coreSubsection)
		assert.Len(t, coreSubsection.Children, 2, "Core should have renderer and parser")
	})

	t.Run("HiddenItemsExcluded", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "visible",
				Locale: "en",
				URL:    "/docs/visible",
				Metadata: map[string]any{
					"Title": "Visible",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Hidden:  false,
								Order:   1,
							},
						},
					},
				},
			},
			{
				Slug:   "hidden",
				Locale: "en",
				URL:    "/docs/hidden",
				Metadata: map[string]any{
					"Title": "Hidden",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Hidden:  true,
								Order:   2,
							},
						},
					},
				},
			},
		}

		config := collection_dto.NavigationConfig{
			IncludeHidden:  false,
			DefaultOrder:   999,
			GroupBySection: true,
		}

		groups := builder.BuildNavigationGroups(context.Background(), items, config)
		tree := groups.Groups["sidebar"]

		guidesSection := tree.Sections[0]
		assert.Len(t, guidesSection.Children, 1, "Should only have 1 visible item")
		assert.Equal(t, "visible", guidesSection.Children[0].ID)
	})

	t.Run("MultipleNavigationGroups", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "intro",
				Locale: "en",
				URL:    "/docs/intro",
				Metadata: map[string]any{
					"Title": "Introduction",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "get-started",
								Order:   1,
							},
							"footer": {
								Section: "quick-links",
								Order:   1,
								Label:   "Get Started",
							},
						},
					},
				},
			},
			{
				Slug:   "api",
				Locale: "en",
				URL:    "/docs/api",
				Metadata: map[string]any{
					"Title": "API Reference",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "api",
								Order:   2,
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		assert.Len(t, groups.Groups, 2, "Should have sidebar and footer groups")
		assert.Contains(t, groups.Groups, "sidebar")
		assert.Contains(t, groups.Groups, "footer")

		sidebarTree := groups.Groups["sidebar"]
		assert.Len(t, sidebarTree.Sections, 2)

		footerTree := groups.Groups["footer"]
		assert.Len(t, footerTree.Sections, 1)
		assert.Equal(t, "quick-links", footerTree.Sections[0].Section)
	})

	t.Run("MultiLocaleSeparation", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "intro",
				Locale: "en",
				URL:    "/docs/intro",
				Metadata: map[string]any{
					"Title": "Introduction",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "get-started",
								Order:   1,
							},
						},
					},
				},
			},
			{
				Slug:   "intro",
				Locale: "fr",
				URL:    "/fr/docs/intro",
				Metadata: map[string]any{
					"Title": "Introduction",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "get-started",
								Order:   1,
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)
		tree := groups.Groups["sidebar"]
		assert.NotNil(t, tree)
		assert.Contains(t, []string{"en", "fr"}, tree.Locale)

		configEN := collection_dto.DefaultNavigationConfig()
		configEN.Locale = "en"
		groupsEN := builder.BuildNavigationGroups(context.Background(), items, configEN)
		treeEN := groupsEN.Groups["sidebar"]
		assert.NotNil(t, treeEN)
		assert.Equal(t, "en", treeEN.Locale)

		configFR := collection_dto.DefaultNavigationConfig()
		configFR.Locale = "fr"
		groupsFR := builder.BuildNavigationGroups(context.Background(), items, configFR)
		treeFR := groupsFR.Groups["sidebar"]
		assert.NotNil(t, treeFR)
		assert.Equal(t, "fr", treeFR.Locale)

		configDE := collection_dto.DefaultNavigationConfig()
		configDE.Locale = "de"
		groupsDE := builder.BuildNavigationGroups(context.Background(), items, configDE)
		treeDE := groupsDE.Groups["sidebar"]
		assert.NotNil(t, treeDE)

		assert.Contains(t, []string{"en", "fr"}, treeDE.Locale)
	})

	t.Run("OrderSorting", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "third",
				Locale: "en",
				URL:    "/docs/third",
				Metadata: map[string]any{
					"Title": "Third Item",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   30,
							},
						},
					},
				},
			},
			{
				Slug:   "first",
				Locale: "en",
				URL:    "/docs/first",
				Metadata: map[string]any{
					"Title": "First Item",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   10,
							},
						},
					},
				},
			},
			{
				Slug:   "second",
				Locale: "en",
				URL:    "/docs/second",
				Metadata: map[string]any{
					"Title": "Second Item",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   20,
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		tree := groups.Groups["sidebar"]
		guidesSection := tree.Sections[0]

		assert.Equal(t, "first", guidesSection.Children[0].ID, "Items should be sorted by Order")
		assert.Equal(t, "second", guidesSection.Children[1].ID)
		assert.Equal(t, "third", guidesSection.Children[2].ID)
	})

	t.Run("AlphabeticalFallbackForSameOrder", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "zebra",
				Locale: "en",
				URL:    "/docs/zebra",
				Metadata: map[string]any{
					"Title": "Zebra",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   10,
							},
						},
					},
				},
			},
			{
				Slug:   "apple",
				Locale: "en",
				URL:    "/docs/apple",
				Metadata: map[string]any{
					"Title": "Apple",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   10,
							},
						},
					},
				},
			},
			{
				Slug:   "banana",
				Locale: "en",
				URL:    "/docs/banana",
				Metadata: map[string]any{
					"Title": "Banana",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   10,
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		tree := groups.Groups["sidebar"]
		guidesSection := tree.Sections[0]

		assert.Equal(t, "apple", guidesSection.Children[0].ID, "Same order should sort alphabetically")
		assert.Equal(t, "banana", guidesSection.Children[1].ID)
		assert.Equal(t, "zebra", guidesSection.Children[2].ID)
	})

	t.Run("CustomLabelOverridesTitle", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "long-title-page",
				Locale: "en",
				URL:    "/docs/long-title-page",
				Metadata: map[string]any{
					"Title": "A Very Long and Descriptive Title for Documentation",
					"Navigation": &markdown_dto.NavigationMetadata{
						Groups: map[string]*markdown_dto.NavGroupMetadata{
							"sidebar": {
								Section: "guides",
								Order:   1,
								Label:   "Quick Ref",
							},
						},
					},
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		tree := groups.Groups["sidebar"]
		guidesSection := tree.Sections[0]
		item := guidesSection.Children[0]

		assert.Equal(t, "Quick Ref", item.Title, "Label should override Title in navigation")
	})

	t.Run("EmptyCollection", func(t *testing.T) {
		items := []collection_dto.ContentItem{}
		config := collection_dto.DefaultNavigationConfig()

		groups := builder.BuildNavigationGroups(context.Background(), items, config)
		require.NotNil(t, groups)

		assert.Empty(t, groups.Groups)
	})

	t.Run("ItemsWithNoNavigationMetadata", func(t *testing.T) {
		items := []collection_dto.ContentItem{
			{
				Slug:   "no-nav",
				Locale: "en",
				URL:    "/docs/no-nav",
				Metadata: map[string]any{
					"Title": "No Navigation",
				},
			},
		}

		config := collection_dto.DefaultNavigationConfig()
		groups := builder.BuildNavigationGroups(context.Background(), items, config)

		assert.Empty(t, groups.Groups)
	})
}

func TestNavigationNode_Helpers(t *testing.T) {
	t.Run("IsCategory", func(t *testing.T) {
		categoryNode := &collection_dto.NavigationNode{
			ID:    "guides",
			Title: "Guides",
			URL:   "",
		}
		assert.True(t, categoryNode.IsCategory())

		contentNode := &collection_dto.NavigationNode{
			ID:    "intro",
			Title: "Introduction",
			URL:   "/docs/intro",
		}
		assert.False(t, contentNode.IsCategory())
	})

	t.Run("IsLeaf", func(t *testing.T) {
		leafNode := &collection_dto.NavigationNode{
			ID:       "intro",
			Children: []*collection_dto.NavigationNode{},
		}
		assert.True(t, leafNode.IsLeaf())

		parentNode := &collection_dto.NavigationNode{
			ID: "guides",
			Children: []*collection_dto.NavigationNode{
				{ID: "child1"},
			},
		}
		assert.False(t, parentNode.IsLeaf())
	})

	t.Run("HasContent", func(t *testing.T) {
		contentNode := &collection_dto.NavigationNode{
			ID: "intro",
			ContentItem: &collection_dto.ContentItem{
				ID: "intro",
			},
		}
		assert.True(t, contentNode.HasContent())

		categoryNode := &collection_dto.NavigationNode{
			ID:          "guides",
			ContentItem: nil,
		}
		assert.False(t, categoryNode.HasContent())
	})

	t.Run("GetBreadcrumb", func(t *testing.T) {
		root := &collection_dto.NavigationNode{
			ID:    "guides",
			Title: "Guides",
			Level: 0,
		}

		subsection := &collection_dto.NavigationNode{
			ID:     "advanced",
			Title:  "Advanced",
			Level:  1,
			Parent: root,
		}

		leaf := &collection_dto.NavigationNode{
			ID:     "custom-directives",
			Title:  "Custom Directives",
			Level:  2,
			Parent: subsection,
		}

		breadcrumb := leaf.GetBreadcrumb()
		assert.Len(t, breadcrumb, 3)
		assert.Equal(t, "guides", breadcrumb[0].ID)
		assert.Equal(t, "advanced", breadcrumb[1].ID)
		assert.Equal(t, "custom-directives", breadcrumb[2].ID)
	})

	t.Run("CountDescendants", func(t *testing.T) {
		root := &collection_dto.NavigationNode{
			ID: "guides",
			Children: []*collection_dto.NavigationNode{
				{
					ID: "child1",
					Children: []*collection_dto.NavigationNode{
						{ID: "grandchild1"},
						{ID: "grandchild2"},
					},
				},
				{
					ID: "child2",
				},
			},
		}

		count := root.CountDescendants()
		assert.Equal(t, 4, count, "Should count all descendants: 2 children + 2 grandchildren")
	})

	t.Run("FindNodeByID", func(t *testing.T) {
		root := &collection_dto.NavigationNode{
			ID: "guides",
			Children: []*collection_dto.NavigationNode{
				{
					ID: "basics",
					Children: []*collection_dto.NavigationNode{
						{ID: "intro"},
						{ID: "install"},
					},
				},
				{
					ID: "advanced",
					Children: []*collection_dto.NavigationNode{
						{ID: "custom-directives"},
					},
				},
			},
		}

		found := root.FindNodeByID("install")
		require.NotNil(t, found)
		assert.Equal(t, "install", found.ID)

		found = root.FindNodeByID("custom-directives")
		require.NotNil(t, found)
		assert.Equal(t, "custom-directives", found.ID)

		found = root.FindNodeByID("nonexistent")
		assert.Nil(t, found)
	})
}

func TestNavigationBuilder_Humanise(t *testing.T) {
	builder := NewNavigationBuilder()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HyphenatedSlug",
			input:    "get-started",
			expected: "Get Started",
		},
		{
			name:     "UnderscoreSlug",
			input:    "api_reference",
			expected: "Api Reference",
		},
		{
			name:     "MixedDelimiters",
			input:    "quick-start_guide",
			expected: "Quick Start Guide",
		},
		{
			name:     "SingleWord",
			input:    "introduction",
			expected: "Introduction",
		},
		{
			name:     "AlreadyTitleCase",
			input:    "Getting Started",
			expected: "Getting Started",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.humanise(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractGroupNamesFromMapNav(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		navMap := map[string]any{
			"Groups": map[string]any{
				"docs":    map[string]any{"Section": "Guide"},
				"sidebar": map[string]any{"Section": "Sidebar"},
			},
		}
		groups := make(map[string]bool)
		extractGroupNamesFromMapNav(navMap, groups)
		assert.True(t, groups["docs"])
		assert.True(t, groups["sidebar"])
		assert.Equal(t, 2, len(groups))
	})

	t.Run("MissingGroupsKey", func(t *testing.T) {
		navMap := map[string]any{"Other": "data"}
		groups := make(map[string]bool)
		extractGroupNamesFromMapNav(navMap, groups)
		assert.Empty(t, groups)
	})

	t.Run("WrongGroupsType", func(t *testing.T) {
		navMap := map[string]any{"Groups": "not-a-map"}
		groups := make(map[string]bool)
		extractGroupNamesFromMapNav(navMap, groups)
		assert.Empty(t, groups)
	})
}

func TestGetGroupMetadata(t *testing.T) {
	builder := NewNavigationBuilder()

	t.Run("MapBasedNavigation", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: map[string]any{
					"Groups": map[string]any{
						"docs": map[string]any{
							"Section":    "Guide",
							"Subsection": "Basics",
							"Order":      3,
							"Icon":       "book",
							"Hidden":     false,
							"Parent":     "root",
							"Label":      "Custom Label",
						},
					},
				},
			},
		}
		meta := builder.getGroupMetadata(context.Background(), item, "docs")
		require.NotNil(t, meta)
		assert.Equal(t, "Guide", meta.Section)
		assert.Equal(t, "Basics", meta.Subsection)
		assert.Equal(t, 3, meta.Order)
		assert.Equal(t, "book", meta.Icon)
		assert.False(t, meta.Hidden)
		assert.Equal(t, "root", meta.Parent)
		assert.Equal(t, "Custom Label", meta.Label)
	})

	t.Run("MapNavMissingGroups", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: map[string]any{
					"NotGroups": "value",
				},
			},
		}
		meta := builder.getGroupMetadata(context.Background(), item, "docs")
		assert.Nil(t, meta)
	})

	t.Run("MapNavGroupNotFound", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: map[string]any{
					"Groups": map[string]any{
						"other": map[string]any{"Section": "Other"},
					},
				},
			},
		}
		meta := builder.getGroupMetadata(context.Background(), item, "docs")
		assert.Nil(t, meta)
	})

	t.Run("UnsupportedNavType", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: 42,
			},
		}
		meta := builder.getGroupMetadata(context.Background(), item, "docs")
		assert.Nil(t, meta)
	})

	t.Run("NoNavigationField", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug:     "test",
			Metadata: map[string]any{},
		}
		meta := builder.getGroupMetadata(context.Background(), item, "docs")
		assert.Nil(t, meta)
	})
}

func TestGetMapKeys(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		keys := getMapKeys(map[string]any{})
		assert.Empty(t, keys)
	})

	t.Run("Multiple", func(t *testing.T) {
		keys := getMapKeys(map[string]any{"a": 1, "b": 2})
		assert.Len(t, keys, 2)
		assert.Contains(t, keys, "a")
		assert.Contains(t, keys, "b")
	})
}

func TestGetStringFromMap(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		result := getStringFromMap(map[string]any{"key": "value"}, "key")
		assert.Equal(t, "value", result)
	})

	t.Run("NotFound", func(t *testing.T) {
		result := getStringFromMap(map[string]any{}, "key")
		assert.Equal(t, "", result)
	})

	t.Run("WrongType", func(t *testing.T) {
		result := getStringFromMap(map[string]any{"key": 42}, "key")
		assert.Equal(t, "", result)
	})
}

func TestGetIntFromMap(t *testing.T) {
	t.Run("IntValue", func(t *testing.T) {
		result := getIntFromMap(map[string]any{"key": 42}, "key")
		assert.Equal(t, 42, result)
	})

	t.Run("Float64Value", func(t *testing.T) {
		result := getIntFromMap(map[string]any{"key": float64(3.14)}, "key")
		assert.Equal(t, 3, result)
	})

	t.Run("NotFound", func(t *testing.T) {
		result := getIntFromMap(map[string]any{}, "key")
		assert.Equal(t, 0, result)
	})
}

func TestGetBoolFromMap(t *testing.T) {
	t.Run("Found", func(t *testing.T) {
		result := getBoolFromMap(map[string]any{"key": true}, "key")
		assert.True(t, result)
	})

	t.Run("NotFound", func(t *testing.T) {
		result := getBoolFromMap(map[string]any{}, "key")
		assert.False(t, result)
	})

	t.Run("StringTrue", func(t *testing.T) {
		result := getBoolFromMap(map[string]any{"key": "true"}, "key")
		assert.True(t, result)
	})

	t.Run("WrongType", func(t *testing.T) {
		result := getBoolFromMap(map[string]any{"key": struct{}{}}, "key")
		assert.False(t, result)
	})
}

func TestSelectTreeForLocale(t *testing.T) {
	builder := NewNavigationBuilder()

	t.Run("EmptyTrees", func(t *testing.T) {
		result := builder.selectTreeForLocale(context.Background(), map[string]*collection_dto.NavigationTree{}, "en", "docs")
		assert.Nil(t, result)
	})

	t.Run("RequestedLocaleFound", func(t *testing.T) {
		trees := map[string]*collection_dto.NavigationTree{
			"en": {Locale: "en"},
			"fr": {Locale: "fr"},
		}
		result := builder.selectTreeForLocale(context.Background(), trees, "en", "docs")
		require.NotNil(t, result)
		assert.Equal(t, "en", result.Locale)
	})

	t.Run("RequestedLocaleNotFound", func(t *testing.T) {
		trees := map[string]*collection_dto.NavigationTree{
			"en": {Locale: "en"},
		}
		result := builder.selectTreeForLocale(context.Background(), trees, "de", "docs")

		require.NotNil(t, result)
	})

	t.Run("NoLocaleRequested", func(t *testing.T) {
		trees := map[string]*collection_dto.NavigationTree{
			"en": {Locale: "en"},
		}
		result := builder.selectTreeForLocale(context.Background(), trees, "", "docs")
		require.NotNil(t, result)
	})
}

func TestGetIntFromMap_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want int
	}{
		{name: "int", m: map[string]any{"k": 42}, key: "k", want: 42},
		{name: "int8", m: map[string]any{"k": int8(10)}, key: "k", want: 10},
		{name: "int16", m: map[string]any{"k": int16(200)}, key: "k", want: 200},
		{name: "int32", m: map[string]any{"k": int32(300)}, key: "k", want: 300},
		{name: "int64", m: map[string]any{"k": int64(400)}, key: "k", want: 400},
		{name: "uint", m: map[string]any{"k": uint(500)}, key: "k", want: 500},
		{name: "uint8", m: map[string]any{"k": uint8(60)}, key: "k", want: 60},
		{name: "uint16", m: map[string]any{"k": uint16(700)}, key: "k", want: 700},
		{name: "uint32", m: map[string]any{"k": uint32(800)}, key: "k", want: 800},
		{name: "uint64", m: map[string]any{"k": uint64(900)}, key: "k", want: 900},
		{name: "float32", m: map[string]any{"k": float32(3.7)}, key: "k", want: 3},
		{name: "float64", m: map[string]any{"k": float64(7.9)}, key: "k", want: 7},
		{name: "string valid integer", m: map[string]any{"k": "42"}, key: "k", want: 42},
		{name: "string invalid", m: map[string]any{"k": "not a number"}, key: "k", want: 0},
		{name: "nil value", m: map[string]any{"k": nil}, key: "k", want: 0},
		{name: "key not found", m: map[string]any{}, key: "k", want: 0},
		{name: "unsupported type bool", m: map[string]any{"k": true}, key: "k", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getIntFromMap(tt.m, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBoolFromMap_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want bool
	}{
		{name: "bool true", m: map[string]any{"k": true}, key: "k", want: true},
		{name: "bool false", m: map[string]any{"k": false}, key: "k", want: false},
		{name: "int non-zero", m: map[string]any{"k": 1}, key: "k", want: true},
		{name: "int zero", m: map[string]any{"k": 0}, key: "k", want: false},
		{name: "int64 non-zero", m: map[string]any{"k": int64(1)}, key: "k", want: true},
		{name: "int64 zero", m: map[string]any{"k": int64(0)}, key: "k", want: false},
		{name: "float64 non-zero", m: map[string]any{"k": float64(1.0)}, key: "k", want: true},
		{name: "float64 zero", m: map[string]any{"k": float64(0)}, key: "k", want: false},
		{name: "string true", m: map[string]any{"k": "true"}, key: "k", want: true},
		{name: "string 1", m: map[string]any{"k": "1"}, key: "k", want: true},
		{name: "string false", m: map[string]any{"k": "false"}, key: "k", want: false},
		{name: "string other", m: map[string]any{"k": "yes"}, key: "k", want: false},
		{name: "nil value", m: map[string]any{"k": nil}, key: "k", want: false},
		{name: "key not found", m: map[string]any{}, key: "k", want: false},
		{name: "unsupported type struct", m: map[string]any{"k": struct{}{}}, key: "k", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getBoolFromMap(tt.m, tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractGroupNamesFromItem(t *testing.T) {
	t.Run("TypedNavigationMetadata", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: &markdown_dto.NavigationMetadata{
					Groups: map[string]*markdown_dto.NavGroupMetadata{
						"sidebar": {Section: "guides"},
						"footer":  {Section: "links"},
					},
				},
			},
		}
		groups := make(map[string]bool)
		extractGroupNamesFromItem(item, groups)
		assert.True(t, groups["sidebar"])
		assert.True(t, groups["footer"])
		assert.Len(t, groups, 2)
	})

	t.Run("MapBasedNavigation", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: map[string]any{
					"Groups": map[string]any{
						"docs": map[string]any{"Section": "Guide"},
					},
				},
			},
		}
		groups := make(map[string]bool)
		extractGroupNamesFromItem(item, groups)
		assert.True(t, groups["docs"])
		assert.Len(t, groups, 1)
	})

	t.Run("NoNavigationMetadata", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug:     "test",
			Metadata: map[string]any{"Title": "No Nav"},
		}
		groups := make(map[string]bool)
		extractGroupNamesFromItem(item, groups)
		assert.Empty(t, groups)
	})

	t.Run("UnsupportedNavType", func(t *testing.T) {
		item := &collection_dto.ContentItem{
			Slug: "test",
			Metadata: map[string]any{
				collection_dto.MetaKeyNavigation: 42,
			},
		}
		groups := make(map[string]bool)
		extractGroupNamesFromItem(item, groups)
		assert.Empty(t, groups)
	})
}
