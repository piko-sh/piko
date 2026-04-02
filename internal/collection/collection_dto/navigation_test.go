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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigationNode_IsCategory(t *testing.T) {
	t.Parallel()

	t.Run("category node", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{Title: "Guides", URL: ""}
		assert.True(t, node.IsCategory())
	})

	t.Run("content node", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{Title: "Install", URL: "/docs/install"}
		assert.False(t, node.IsCategory())
	})
}

func TestNavigationNode_IsLeaf(t *testing.T) {
	t.Parallel()

	t.Run("leaf node", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{Title: "Install"}
		assert.True(t, node.IsLeaf())
	})

	t.Run("non-leaf node", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{
			Title:    "Guides",
			Children: []*NavigationNode{{Title: "Child"}},
		}
		assert.False(t, node.IsLeaf())
	})
}

func TestNavigationNode_HasContent(t *testing.T) {
	t.Parallel()

	t.Run("with content", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{ContentItem: &ContentItem{ID: "item-1"}}
		assert.True(t, node.HasContent())
	})

	t.Run("without content", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{}
		assert.False(t, node.HasContent())
	})
}

func TestNavigationNode_GetBreadcrumb(t *testing.T) {
	t.Parallel()

	t.Run("root node", func(t *testing.T) {
		t.Parallel()

		root := &NavigationNode{Title: "Root"}
		breadcrumb := root.GetBreadcrumb()

		require.Len(t, breadcrumb, 1)
		assert.Equal(t, "Root", breadcrumb[0].Title)
	})

	t.Run("nested node", func(t *testing.T) {
		t.Parallel()

		root := &NavigationNode{Title: "Guides"}
		child := &NavigationNode{Title: "Advanced", Parent: root}
		grandchild := &NavigationNode{Title: "Custom Directives", Parent: child}

		breadcrumb := grandchild.GetBreadcrumb()

		require.Len(t, breadcrumb, 3)
		assert.Equal(t, "Guides", breadcrumb[0].Title)
		assert.Equal(t, "Advanced", breadcrumb[1].Title)
		assert.Equal(t, "Custom Directives", breadcrumb[2].Title)
	})
}

func TestNavigationNode_CountDescendants(t *testing.T) {
	t.Parallel()

	t.Run("leaf node", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{}
		assert.Equal(t, 0, node.CountDescendants())
	})

	t.Run("with children", func(t *testing.T) {
		t.Parallel()

		node := &NavigationNode{
			Children: []*NavigationNode{
				{Title: "A"},
				{Title: "B", Children: []*NavigationNode{
					{Title: "B1"},
					{Title: "B2"},
				}},
				{Title: "C"},
			},
		}
		assert.Equal(t, 5, node.CountDescendants())
	})
}

func TestNavigationNode_FindNodeByID(t *testing.T) {
	t.Parallel()

	tree := &NavigationNode{
		ID: "root",
		Children: []*NavigationNode{
			{ID: "child-1"},
			{
				ID: "child-2",
				Children: []*NavigationNode{
					{ID: "grandchild-1"},
					{ID: "grandchild-2"},
				},
			},
		},
	}

	t.Run("find root", func(t *testing.T) {
		t.Parallel()

		found := tree.FindNodeByID("root")
		require.NotNil(t, found)
		assert.Equal(t, "root", found.ID)
	})

	t.Run("find child", func(t *testing.T) {
		t.Parallel()

		found := tree.FindNodeByID("child-1")
		require.NotNil(t, found)
		assert.Equal(t, "child-1", found.ID)
	})

	t.Run("find grandchild", func(t *testing.T) {
		t.Parallel()

		found := tree.FindNodeByID("grandchild-2")
		require.NotNil(t, found)
		assert.Equal(t, "grandchild-2", found.ID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, tree.FindNodeByID("nonexistent"))
	})
}

func TestDefaultNavigationConfig(t *testing.T) {
	t.Parallel()

	config := DefaultNavigationConfig()

	assert.False(t, config.IncludeHidden)
	assert.Equal(t, 999, config.DefaultOrder)
	assert.True(t, config.GroupBySection)
	assert.Empty(t, config.Locale)
}
