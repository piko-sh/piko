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

package annotator_domain

import (
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsWhitespaceOrCommentNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "comment node returns true",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeComment,
			},
			expected: true,
		},
		{
			name: "text node with only whitespace returns true",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "   \t\n  ",
				RichText:    nil,
			},
			expected: true,
		},
		{
			name: "text node with empty content returns true",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "",
				RichText:    nil,
			},
			expected: true,
		},
		{
			name: "text node with actual content returns false",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "Hello",
				RichText:    nil,
			},
			expected: false,
		},
		{
			name: "text node with rich text returns false",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "",
				RichText: []ast_domain.TextPart{
					{Literal: "text", IsLiteral: true},
				},
			},
			expected: false,
		},
		{
			name: "element node returns false",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isWhitespaceOrCommentNode(tc.node)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindEffectiveRootElements(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		nodes         []*ast_domain.TemplateNode
		expectedCount int
	}{
		{
			name:          "empty nodes returns empty slice",
			nodes:         []*ast_domain.TemplateNode{},
			expectedCount: 0,
		},
		{
			name: "only element nodes",
			nodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
			},
			expectedCount: 2,
		},
		{
			name: "mixed nodes filters correctly",
			nodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "text"},
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeComment},
				{NodeType: ast_domain.NodeElement, TagName: "p"},
			},
			expectedCount: 2,
		},
		{
			name: "no element nodes returns empty",
			nodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "text"},
				{NodeType: ast_domain.NodeComment},
			},
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findEffectiveRootElements(tc.nodes)

			assert.Len(t, result, tc.expectedCount)
			for _, node := range result {
				assert.Equal(t, ast_domain.NodeElement, node.NodeType)
			}
		})
	}
}

func TestShouldSkipStaticAttr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		attributeName string
		expected      bool
	}{
		{
			name:          "is attribute should be skipped",
			attributeName: "is",
			expected:      true,
		},
		{
			name:          "server prefix should be skipped",
			attributeName: "server.data",
			expected:      true,
		},
		{
			name:          "request prefix should be skipped",
			attributeName: "request.param",
			expected:      true,
		},
		{
			name:          "class attribute should not be skipped",
			attributeName: "class",
			expected:      false,
		},
		{
			name:          "id attribute should not be skipped",
			attributeName: "id",
			expected:      false,
		},
		{
			name:          "data attribute should not be skipped",
			attributeName: "data-value",
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := shouldSkipStaticAttr(tc.attributeName)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsPartialMetadataAttr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		attributeName string
		expected      bool
	}{
		{
			name:          "partial attribute returns true",
			attributeName: "partial",
			expected:      true,
		},
		{
			name:          "partial_name attribute returns true",
			attributeName: "partial_name",
			expected:      true,
		},
		{
			name:          "partial_src attribute returns true",
			attributeName: "partial_src",
			expected:      true,
		},
		{
			name:          "class attribute returns false",
			attributeName: "class",
			expected:      false,
		},
		{
			name:          "id attribute returns false",
			attributeName: "id",
			expected:      false,
		},
		{
			name:          "partial-like but different returns false",
			attributeName: "partialx",
			expected:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isPartialMetadataAttr(tc.attributeName)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindDefaultSlotLocation(t *testing.T) {
	t.Parallel()

	t.Run("returns location of first non-whitespace node", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "  ", Location: ast_domain.Location{Line: 1, Column: 1}},
			{NodeType: ast_domain.NodeComment, Location: ast_domain.Location{Line: 2, Column: 1}},
			{NodeType: ast_domain.NodeElement, TagName: "div", Location: ast_domain.Location{Line: 3, Column: 5}},
		}

		result := findDefaultSlotLocation(nodes)

		assert.Equal(t, 3, result.Line)
		assert.Equal(t, 5, result.Column)
	})

	t.Run("returns first node location when all are whitespace", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "  ", Location: ast_domain.Location{Line: 1, Column: 1}},
			{NodeType: ast_domain.NodeComment, Location: ast_domain.Location{Line: 2, Column: 2}},
		}

		result := findDefaultSlotLocation(nodes)

		assert.Equal(t, 1, result.Line)
		assert.Equal(t, 1, result.Column)
	})

	t.Run("returns zero location for empty nodes", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{}

		result := findDefaultSlotLocation(nodes)

		assert.Equal(t, 0, result.Line)
		assert.Equal(t, 0, result.Column)
		assert.Equal(t, 0, result.Offset)
	})
}

func TestCollectDefinedSlots(t *testing.T) {
	t.Parallel()

	t.Run("collects named slots", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:slot",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "name", Value: "header"},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:slot",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "name", Value: "footer"},
				},
			},
		}

		result := collectDefinedSlots(nodes)

		assert.True(t, result["header"])
		assert.True(t, result["footer"])
		assert.Len(t, result, 2)
	})

	t.Run("collects default slot with empty name", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "piko:slot",
				Attributes: []ast_domain.HTMLAttribute{},
			},
		}

		result := collectDefinedSlots(nodes)

		assert.True(t, result[""])
		assert.Len(t, result, 1)
	})

	t.Run("collects slots from nested nodes", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:slot",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "name", Value: "nested"},
						},
					},
				},
			},
		}

		result := collectDefinedSlots(nodes)

		assert.True(t, result["nested"])
	})

	t.Run("returns empty map for no slots", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
			{NodeType: ast_domain.NodeText, TextContent: "text"},
		}

		result := collectDefinedSlots(nodes)

		assert.Empty(t, result)
	})
}

func TestCollectTargetStaticAttrs(t *testing.T) {
	t.Parallel()

	t.Run("collects attributes with lowercase keys", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "Class", Value: "container"},
				{Name: "ID", Value: "main"},
				{Name: "data-value", Value: "123"},
			},
		}

		result := collectTargetStaticAttrs(node)

		assert.Equal(t, "container", result["class"].Value)
		assert.Equal(t, "main", result["id"].Value)
		assert.Equal(t, "123", result["data-value"].Value)
	})

	t.Run("returns empty map for no attributes", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{},
		}

		result := collectTargetStaticAttrs(node)

		assert.Empty(t, result)
	})
}

func TestRebuildSortedStaticAttrs(t *testing.T) {
	t.Parallel()

	t.Run("returns attributes sorted alphabetically", func(t *testing.T) {
		t.Parallel()

		attrs := map[string]ast_domain.HTMLAttribute{
			"class": {Name: "class", Value: "container"},
			"id":    {Name: "id", Value: "main"},
			"aria":  {Name: "aria", Value: "label"},
		}

		result := rebuildSortedStaticAttrs(attrs)

		require.Len(t, result, 3)
		assert.Equal(t, "aria", result[0].Name)
		assert.Equal(t, "class", result[1].Name)
		assert.Equal(t, "id", result[2].Name)
	})

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		attrs := map[string]ast_domain.HTMLAttribute{}

		result := rebuildSortedStaticAttrs(attrs)

		assert.Empty(t, result)
	})
}

func TestMergeClassAttr(t *testing.T) {
	t.Parallel()

	t.Run("merges class with existing", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"class": {Name: "class", Value: "existing"},
		}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "class", Value: "new-class"}

		mergeClassAttr(invokerAttr, finalAttrs)

		assert.Equal(t, "existing new-class", finalAttrs["class"].Value)
	})

	t.Run("adds class when none exists", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "class", Value: "new-class"}

		mergeClassAttr(invokerAttr, finalAttrs)

		assert.Equal(t, "new-class", finalAttrs["class"].Value)
	})
}

func TestMergePartialMetadataAttr(t *testing.T) {
	t.Parallel()

	t.Run("prepends invoker value to existing", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"partial": {Name: "partial", Value: "existing"},
		}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "partial", Value: "new"}

		mergePartialMetadataAttr("partial", invokerAttr, finalAttrs)

		assert.Equal(t, "new existing", finalAttrs["partial"].Value)
	})

	t.Run("adds attribute when none exists", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "partial", Value: "new"}

		mergePartialMetadataAttr("partial", invokerAttr, finalAttrs)

		assert.Equal(t, "new", finalAttrs["partial"].Value)
	})
}

func TestGetSortedAttrKeys(t *testing.T) {
	t.Parallel()

	t.Run("returns keys in sorted order", func(t *testing.T) {
		t.Parallel()

		attrs := map[string]ast_domain.DynamicAttribute{
			"zebra":  {},
			"alpha":  {},
			"middle": {},
		}

		result := getSortedAttrKeys(attrs)

		require.Len(t, result, 3)
		assert.Equal(t, "alpha", result[0])
		assert.Equal(t, "middle", result[1])
		assert.Equal(t, "zebra", result[2])
	})

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		attrs := map[string]ast_domain.DynamicAttribute{}

		result := getSortedAttrKeys(attrs)

		assert.Empty(t, result)
	})
}

func TestGetPartialOrigin(t *testing.T) {
	t.Parallel()

	t.Run("returns package alias when set", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("mypackage"),
			},
		}

		result := getPartialOrigin(node)

		assert.Equal(t, "mypackage", result)
	})

	t.Run("returns empty when GoAnnotations is nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		result := getPartialOrigin(node)

		assert.Equal(t, "", result)
	})

	t.Run("returns empty when OriginalPackageAlias is nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: nil,
			},
		}

		result := getPartialOrigin(node)

		assert.Equal(t, "", result)
	})
}

func TestEnsureDynamicAttributeAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("creates GoAnnotations when nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}

		ensureDynamicAttributeAnnotations(node)

		require.NotNil(t, node.GoAnnotations)
		require.NotNil(t, node.GoAnnotations.DynamicAttributeOrigins)
	})

	t.Run("creates DynamicAttributeOrigins when nil", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: nil,
			},
		}

		ensureDynamicAttributeAnnotations(node)

		require.NotNil(t, node.GoAnnotations.DynamicAttributeOrigins)
	})

	t.Run("preserves existing values", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("existingalias"),
				DynamicAttributeOrigins: map[string]string{
					"existing": "value",
				},
			},
		}

		ensureDynamicAttributeAnnotations(node)

		assert.Equal(t, "existingalias", *node.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "value", node.GoAnnotations.DynamicAttributeOrigins["existing"])
	})
}

func TestGroupContentBySlot(t *testing.T) {
	t.Parallel()

	t.Run("groups piko:slot elements by name", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:slot",
				Location: ast_domain.Location{Line: 1, Column: 1},
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "name", Value: "header"},
				},
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "header content"},
				},
			},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result["header"].Nodes, 1)
		assert.Equal(t, "header content", result["header"].Nodes[0].TextContent)
	})

	t.Run("groups elements with p-slot directive", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "article",
				Location: ast_domain.Location{Line: 1, Column: 1},
				DirSlot:  &ast_domain.Directive{RawExpression: "content"},
			},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result["content"].Nodes, 1)
		assert.Equal(t, "article", result["content"].Nodes[0].TagName)
		assert.Nil(t, result["content"].Nodes[0].DirSlot, "DirSlot should be cleared")
	})

	t.Run("puts non-slot content in default slot", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div", Location: ast_domain.Location{Line: 1, Column: 1}},
			{NodeType: ast_domain.NodeElement, TagName: "span", Location: ast_domain.Location{Line: 2, Column: 1}},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result[""].Nodes, 2)
	})

	t.Run("ignores whitespace and comment nodes for default slot", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "  \n  "},
			{NodeType: ast_domain.NodeComment},
			{NodeType: ast_domain.NodeElement, TagName: "div", Location: ast_domain.Location{Line: 3, Column: 1}},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result[""].Nodes, 1)
		assert.Equal(t, "div", result[""].Nodes[0].TagName)
	})
}

func TestPartialExpansionTaskPool(t *testing.T) {

	t.Run("getPartialExpansionTask returns configured task", func(t *testing.T) {
		invokerNode := &ast_domain.TemplateNode{TagName: "TestPartial"}
		aliasToPath := map[string]string{"alias": "/path/to/partial"}

		task := getPartialExpansionTask(nil, invokerNode, nil, aliasToPath, "alias")
		defer putPartialExpansionTask(task)

		assert.Same(t, invokerNode, task.invokerNode)
		assert.Equal(t, "alias", task.userAlias)
		assert.Equal(t, "/path/to/partial", task.aliasToRawPath["alias"])
	})

	t.Run("putPartialExpansionTask clears task fields", func(t *testing.T) {
		invokerNode := &ast_domain.TemplateNode{TagName: "TestPartial"}
		task := getPartialExpansionTask(nil, invokerNode, nil, nil, "alias")

		putPartialExpansionTask(task)

		assert.Nil(t, task.ec)
		assert.Nil(t, task.invokerNode)
		assert.Nil(t, task.invokerComponent)
		assert.Nil(t, task.aliasToRawPath)
		assert.Equal(t, "", task.userAlias)
		assert.False(t, task.hasError)
	})
}

func TestMergeStaticAttrByType(t *testing.T) {
	t.Parallel()

	t.Run("class attribute is merged via mergeClassAttr", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"class": {Name: "class", Value: "existing"},
		}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "class", Value: "added"}

		mergeStaticAttrByType("class", invokerAttr, finalAttrs)

		assert.Equal(t, "existing added", finalAttrs["class"].Value)
	})

	t.Run("partial metadata attribute is merged via mergePartialMetadataAttr", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"partial": {Name: "partial", Value: "inner"},
		}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "partial", Value: "outer"}

		mergeStaticAttrByType("partial", invokerAttr, finalAttrs)

		assert.Equal(t, "outer inner", finalAttrs["partial"].Value)
	})

	t.Run("partial_name metadata attribute is merged correctly", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "partial_name", Value: "my-partial"}

		mergeStaticAttrByType("partial_name", invokerAttr, finalAttrs)

		assert.Equal(t, "my-partial", finalAttrs["partial_name"].Value)
	})

	t.Run("partial_src metadata attribute is merged correctly", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "partial_src", Value: "/src/partial.pk"}

		mergeStaticAttrByType("partial_src", invokerAttr, finalAttrs)

		assert.Equal(t, "/src/partial.pk", finalAttrs["partial_src"].Value)
	})

	t.Run("regular attribute overwrites directly", func(t *testing.T) {
		t.Parallel()

		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"id": {Name: "id", Value: "old"},
		}
		invokerAttr := &ast_domain.HTMLAttribute{Name: "id", Value: "new"}

		mergeStaticAttrByType("id", invokerAttr, finalAttrs)

		assert.Equal(t, "new", finalAttrs["id"].Value)
	})
}

func TestMergeInvokerStaticAttrs(t *testing.T) {
	t.Parallel()

	t.Run("skips is attribute", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "is", Value: "MyPartial"},
				{Name: "title", Value: "hello"},
			},
		}
		finalAttrs := make(map[string]ast_domain.HTMLAttribute)

		mergeInvokerStaticAttrs(invokerNode, finalAttrs)

		assert.NotContains(t, finalAttrs, "is")
		assert.Equal(t, "hello", finalAttrs["title"].Value)
	})

	t.Run("skips server prefix attributes", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "server.data", Value: "secret"},
				{Name: "id", Value: "main"},
			},
		}
		finalAttrs := make(map[string]ast_domain.HTMLAttribute)

		mergeInvokerStaticAttrs(invokerNode, finalAttrs)

		assert.NotContains(t, finalAttrs, "server.data")
		assert.Equal(t, "main", finalAttrs["id"].Value)
	})

	t.Run("skips request prefix attributes", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "request.param", Value: "value"},
				{Name: "data-x", Value: "42"},
			},
		}
		finalAttrs := make(map[string]ast_domain.HTMLAttribute)

		mergeInvokerStaticAttrs(invokerNode, finalAttrs)

		assert.NotContains(t, finalAttrs, "request.param")
		assert.Equal(t, "42", finalAttrs["data-x"].Value)
	})

	t.Run("merges class attributes by joining", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "invoker-class"},
			},
		}
		finalAttrs := map[string]ast_domain.HTMLAttribute{
			"class": {Name: "class", Value: "target-class"},
		}

		mergeInvokerStaticAttrs(invokerNode, finalAttrs)

		assert.Equal(t, "target-class invoker-class", finalAttrs["class"].Value)
	})
}

func TestMergeStaticAttributes(t *testing.T) {
	t.Parallel()

	t.Run("merges target and invoker attributes sorted", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "target-id"},
				{Name: "class", Value: "target-cls"},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "invoker-cls"},
				{Name: "aria-label", Value: "button"},
			},
		}

		mergeStaticAttributes(targetNode, invokerNode)

		require.Len(t, targetNode.Attributes, 3)
		assert.Equal(t, "aria-label", targetNode.Attributes[0].Name)
		assert.Equal(t, "class", targetNode.Attributes[1].Name)
		assert.Equal(t, "target-cls invoker-cls", targetNode.Attributes[1].Value)
		assert.Equal(t, "id", targetNode.Attributes[2].Name)
	})

	t.Run("handles empty target attributes", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{},
		}
		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "title", Value: "invoker-title"},
			},
		}

		mergeStaticAttributes(targetNode, invokerNode)

		require.Len(t, targetNode.Attributes, 1)
		assert.Equal(t, "invoker-title", targetNode.Attributes[0].Value)
	})

	t.Run("handles empty invoker attributes", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "data-x", Value: "1"},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{},
		}

		mergeStaticAttributes(targetNode, invokerNode)

		require.Len(t, targetNode.Attributes, 1)
		assert.Equal(t, "1", targetNode.Attributes[0].Value)
	})
}

func TestStampInvokerOriginOnAttr(t *testing.T) {
	t.Parallel()

	t.Run("sets origin when origin is non-empty and GoAnnotations is nil", func(t *testing.T) {
		t.Parallel()

		attr := &ast_domain.DynamicAttribute{Name: "title"}

		stampInvokerOriginOnAttr(attr, "my_pkg")

		require.NotNil(t, attr.GoAnnotations)
		require.NotNil(t, attr.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "my_pkg", *attr.GoAnnotations.OriginalPackageAlias)
	})

	t.Run("sets origin when GoAnnotations already exists", func(t *testing.T) {
		t.Parallel()

		attr := &ast_domain.DynamicAttribute{
			Name:          "title",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{NeedsCSRF: true},
		}

		stampInvokerOriginOnAttr(attr, "other_pkg")

		require.NotNil(t, attr.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "other_pkg", *attr.GoAnnotations.OriginalPackageAlias)
		assert.True(t, attr.GoAnnotations.NeedsCSRF, "existing annotations should be preserved")
	})

	t.Run("does nothing when origin is empty", func(t *testing.T) {
		t.Parallel()

		attr := &ast_domain.DynamicAttribute{Name: "title"}

		stampInvokerOriginOnAttr(attr, "")

		assert.Nil(t, attr.GoAnnotations)
	})
}

func TestCollectPartialDynamicAttrs(t *testing.T) {
	t.Parallel()

	t.Run("collects dynamic attributes with lowercase keys", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "Title", RawExpression: "state.title"},
				{Name: "VISIBLE", RawExpression: "state.visible"},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectPartialDynamicAttrs(targetNode, finalDynAttrs, "partial_pkg")

		assert.Len(t, finalDynAttrs, 2)
		assert.Equal(t, "state.title", finalDynAttrs["title"].RawExpression)
		assert.Equal(t, "state.visible", finalDynAttrs["visible"].RawExpression)
		assert.Equal(t, "partial_pkg", targetNode.GoAnnotations.DynamicAttributeOrigins["title"])
		assert.Equal(t, "partial_pkg", targetNode.GoAnnotations.DynamicAttributeOrigins["visible"])
	})

	t.Run("skips origin recording when partialOrigin is empty", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.title"},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectPartialDynamicAttrs(targetNode, finalDynAttrs, "")

		assert.Len(t, finalDynAttrs, 1)
		assert.Empty(t, targetNode.GoAnnotations.DynamicAttributeOrigins)
	})
}

func TestCollectInvokerDynamicAttrs(t *testing.T) {
	t.Parallel()

	t.Run("collects normal dynamic attributes from invoker", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.title"},
			},
		}
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectInvokerDynamicAttrs(invokerNode, targetNode, finalDynAttrs, "invoker_pkg")

		assert.Len(t, finalDynAttrs, 1)
		assert.Equal(t, "state.title", finalDynAttrs["title"].RawExpression)
		assert.Equal(t, "invoker_pkg", targetNode.GoAnnotations.DynamicAttributeOrigins["title"])
	})

	t.Run("skips server prefix attributes", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "server.data", RawExpression: "state.data"},
				{Name: "normal", RawExpression: "state.normal"},
			},
		}
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectInvokerDynamicAttrs(invokerNode, targetNode, finalDynAttrs, "pkg")

		assert.Len(t, finalDynAttrs, 1)
		assert.NotContains(t, finalDynAttrs, "server.data")
		assert.Contains(t, finalDynAttrs, "normal")
	})

	t.Run("skips request prefix attributes", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "request.path", RawExpression: "state.path"},
				{Name: "colour", RawExpression: "state.colour"},
			},
		}
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectInvokerDynamicAttrs(invokerNode, targetNode, finalDynAttrs, "pkg")

		assert.Len(t, finalDynAttrs, 1)
		assert.NotContains(t, finalDynAttrs, "request.path")
		assert.Contains(t, finalDynAttrs, "colour")
	})

	t.Run("stamps origin on collected attributes", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "label", RawExpression: "state.label"},
			},
		}
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		finalDynAttrs := make(map[string]ast_domain.DynamicAttribute)

		collectInvokerDynamicAttrs(invokerNode, targetNode, finalDynAttrs, "origin_pkg")

		attr := finalDynAttrs["label"]
		require.NotNil(t, attr.GoAnnotations)
		require.NotNil(t, attr.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "origin_pkg", *attr.GoAnnotations.OriginalPackageAlias)
	})
}

func TestFinaliseAttrOrigin(t *testing.T) {
	t.Parallel()

	t.Run("sets origin when attribute is from invoker", func(t *testing.T) {
		t.Parallel()

		invokerOrigin := "invoker_pkg"
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: map[string]string{
					"title": "invoker_pkg",
				},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/invoker/source.pk"),
			},
		}
		attr := &ast_domain.DynamicAttribute{Name: "title"}

		finaliseAttrOrigin(attr, targetNode, invokerNode, invokerOrigin, "title")

		require.NotNil(t, attr.GoAnnotations)
		require.NotNil(t, attr.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "invoker_pkg", *attr.GoAnnotations.OriginalPackageAlias)
		require.NotNil(t, attr.GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/invoker/source.pk", *attr.GoAnnotations.OriginalSourcePath)
	})

	t.Run("does not modify when key is not from invoker", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: map[string]string{},
			},
		}
		invokerNode := &ast_domain.TemplateNode{}
		attr := &ast_domain.DynamicAttribute{Name: "title"}

		finaliseAttrOrigin(attr, targetNode, invokerNode, "invoker_pkg", "title")

		assert.Nil(t, attr.GoAnnotations)
	})

	t.Run("does not modify when origin does not match invoker", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: map[string]string{
					"title": "different_pkg",
				},
			},
		}
		invokerNode := &ast_domain.TemplateNode{}
		attr := &ast_domain.DynamicAttribute{Name: "title"}

		finaliseAttrOrigin(attr, targetNode, invokerNode, "invoker_pkg", "title")

		assert.Nil(t, attr.GoAnnotations)
	})

	t.Run("handles nil invoker GoAnnotations gracefully", func(t *testing.T) {
		t.Parallel()

		invokerOrigin := "invoker_pkg"
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: map[string]string{
					"title": "invoker_pkg",
				},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			GoAnnotations: nil,
		}
		attr := &ast_domain.DynamicAttribute{Name: "title"}

		finaliseAttrOrigin(attr, targetNode, invokerNode, invokerOrigin, "title")

		require.NotNil(t, attr.GoAnnotations)
		require.NotNil(t, attr.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "invoker_pkg", *attr.GoAnnotations.OriginalPackageAlias)
		assert.Nil(t, attr.GoAnnotations.OriginalSourcePath)
	})
}

func TestApplyInvokerDirectives(t *testing.T) {
	t.Parallel()

	t.Run("copies all directive types from invoker to target", func(t *testing.T) {
		t.Parallel()

		invokerOrigin := "inv_pkg"
		targetNode := &ast_domain.TemplateNode{TagName: "div"}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/invoker.pk"),
			},
			DirIf:    &ast_domain.Directive{RawExpression: "state.show"},
			DirFor:   &ast_domain.Directive{RawExpression: "item in state.items"},
			DirShow:  &ast_domain.Directive{RawExpression: "state.visible"},
			DirKey:   &ast_domain.Directive{RawExpression: "item.id"},
			DirClass: &ast_domain.Directive{RawExpression: "state.classes"},
			DirStyle: &ast_domain.Directive{RawExpression: "state.styles"},
		}

		applyInvokerDirectives(targetNode, invokerNode, invokerOrigin)

		require.NotNil(t, targetNode.DirIf)
		assert.Equal(t, "state.show", targetNode.DirIf.RawExpression)
		require.NotNil(t, targetNode.DirIf.GoAnnotations)
		assert.Equal(t, "inv_pkg", *targetNode.DirIf.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "/invoker.pk", *targetNode.DirIf.GoAnnotations.OriginalSourcePath)

		require.NotNil(t, targetNode.DirFor)
		assert.Equal(t, "item in state.items", targetNode.DirFor.RawExpression)

		require.NotNil(t, targetNode.DirShow)
		assert.Equal(t, "state.visible", targetNode.DirShow.RawExpression)

		require.NotNil(t, targetNode.DirKey)
		assert.Equal(t, "item.id", targetNode.DirKey.RawExpression)

		require.NotNil(t, targetNode.DirClass)
		assert.Equal(t, "state.classes", targetNode.DirClass.RawExpression)

		require.NotNil(t, targetNode.DirStyle)
		assert.Equal(t, "state.styles", targetNode.DirStyle.RawExpression)
	})

	t.Run("copies else-if and else directives", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{TagName: "div"}
		invokerNode := &ast_domain.TemplateNode{
			TagName:   "piko:partial",
			DirElseIf: &ast_domain.Directive{RawExpression: "state.other"},
			DirElse:   &ast_domain.Directive{RawExpression: ""},
		}

		applyInvokerDirectives(targetNode, invokerNode, "pkg")

		require.NotNil(t, targetNode.DirElseIf)
		assert.Equal(t, "state.other", targetNode.DirElseIf.RawExpression)

		require.NotNil(t, targetNode.DirElse)
	})

	t.Run("does not overwrite target directives when invoker has none", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			TagName: "div",
			DirIf:   &ast_domain.Directive{RawExpression: "original"},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
		}

		applyInvokerDirectives(targetNode, invokerNode, "pkg")

		assert.Equal(t, "original", targetNode.DirIf.RawExpression)
	})
}

func TestApplyInvokerAttributesToExpandedRoot(t *testing.T) {
	t.Parallel()

	t.Run("nil target does not panic", func(t *testing.T) {
		t.Parallel()

		invokerNode := &ast_domain.TemplateNode{TagName: "div"}

		assert.NotPanics(t, func() {
			applyInvokerAttributesToExpandedRoot(nil, invokerNode)
		})
	})

	t.Run("nil invoker does not panic", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{TagName: "div"}

		assert.NotPanics(t, func() {
			applyInvokerAttributesToExpandedRoot(targetNode, nil)
		})
	})

	t.Run("merges static and dynamic attributes and directives", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			TagName: "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "target-cls"},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("inv_pkg"),
			},
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "invoker-cls"},
				{Name: "data-test", Value: "true"},
			},
			DirIf: &ast_domain.Directive{RawExpression: "state.show"},
		}

		applyInvokerAttributesToExpandedRoot(targetNode, invokerNode)

		var classAttr *ast_domain.HTMLAttribute
		for i := range targetNode.Attributes {
			if targetNode.Attributes[i].Name == "class" {
				classAttr = &targetNode.Attributes[i]
				break
			}
		}
		require.NotNil(t, classAttr)
		assert.Contains(t, classAttr.Value, "target-cls")
		assert.Contains(t, classAttr.Value, "invoker-cls")

		require.NotNil(t, targetNode.DirIf)
		assert.Equal(t, "state.show", targetNode.DirIf.RawExpression)
	})
}

func TestProcessExpandedNodes(t *testing.T) {
	t.Parallel()

	t.Run("single root element gets partial info attached directly", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "test_key",
			PartialAlias:       "card",
			PartialPackageName: "card_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].GoAnnotations)
		assert.Equal(t, pInfo, result[0].GoAnnotations.PartialInfo)
	})

	t.Run("single root with whitespace siblings returns all nodes with info on element", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: " "},
			{NodeType: ast_domain.NodeElement, TagName: "div"},
			{NodeType: ast_domain.NodeText, TextContent: "\n"},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "test_key",
			PartialAlias:       "card",
			PartialPackageName: "card_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 3)

		assert.Equal(t, pInfo, result[1].GoAnnotations.PartialInfo)
	})

	t.Run("multiple root elements get wrapped in a fragment", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "header"},
			{NodeType: ast_domain.NodeElement, TagName: "footer"},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "multi_key",
			PartialAlias:       "layout",
			PartialPackageName: "layout_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		fragmentNode := result[0]
		require.NotNil(t, fragmentNode.GoAnnotations)
		assert.Equal(t, pInfo, fragmentNode.GoAnnotations.PartialInfo)

		require.Len(t, fragmentNode.Children, 2)
		for _, child := range fragmentNode.Children {
			found := false
			for _, attr := range child.Attributes {
				if attr.Name == "p-fragment" {
					found = true
					assert.Equal(t, "multi_key", attr.Value)
				}
			}
			assert.True(t, found, "expected p-fragment attribute on child %s", child.TagName)
		}
	})

	t.Run("no element nodes returns empty", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "just text"},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "empty_key",
			PartialAlias:       "text-only",
			PartialPackageName: "text_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].GoAnnotations)
		assert.Equal(t, pInfo, result[0].GoAnnotations.PartialInfo)
	})

	t.Run("fragment node inherits invoker origin annotations", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
			{NodeType: ast_domain.NodeElement, TagName: "span"},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalPackageAlias: new("invoker_pkg"),
				OriginalSourcePath:   new("/invoker.pk"),
			},
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "origin_key",
			PartialAlias:       "comp",
			PartialPackageName: "comp_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		fragmentNode := result[0]
		require.NotNil(t, fragmentNode.GoAnnotations)
		require.NotNil(t, fragmentNode.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "invoker_pkg", *fragmentNode.GoAnnotations.OriginalPackageAlias)
		require.NotNil(t, fragmentNode.GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/invoker.pk", *fragmentNode.GoAnnotations.OriginalSourcePath)
	})
}

func TestHandleCSSError(t *testing.T) {
	t.Parallel()

	t.Run("returns false and does not set error when err is nil", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{
			ec: ec,
			loadedPartial: &annotator_dto.ParsedComponent{
				SourcePath: "/test/partial.pk",
			},
		}

		result := task.handleCSSError(nil, "Some prefix", "<style>", ast_domain.Location{Line: 1, Column: 1})

		assert.False(t, result)
		assert.False(t, task.hasError)
		assert.Empty(t, ec.diagnostics)
	})

	t.Run("returns true and records diagnostic when err is non-nil", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{
			ec: ec,
			loadedPartial: &annotator_dto.ParsedComponent{
				SourcePath: "/test/partial.pk",
			},
		}

		result := task.handleCSSError(
			errors.New("CSS parse failure"),
			"Fatal error processing CSS",
			"<style>",
			ast_domain.Location{Line: 5, Column: 3},
		)

		assert.True(t, result)
		assert.True(t, task.hasError)
		require.Len(t, ec.diagnostics, 1)
		assert.Contains(t, ec.diagnostics[0].Message, "Fatal error processing CSS")
		assert.Contains(t, ec.diagnostics[0].Message, "/test/partial.pk")
		assert.Contains(t, ec.diagnostics[0].Message, "CSS parse failure")
		assert.Equal(t, ast_domain.Error, ec.diagnostics[0].Severity)
	})
}

func TestHandleCSSDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("does nothing when diagnostics is empty", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{ec: ec}

		task.handleCSSDiagnostics(nil)

		assert.Empty(t, ec.diagnostics)
		assert.False(t, task.hasError)
	})

	t.Run("appends warning diagnostics without setting hasError", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{ec: ec}

		diagnostics := []*ast_domain.Diagnostic{
			{Severity: ast_domain.Warning, Message: "unused selector"},
		}

		task.handleCSSDiagnostics(diagnostics)

		require.Len(t, ec.diagnostics, 1)
		assert.Equal(t, "unused selector", ec.diagnostics[0].Message)
		assert.False(t, task.hasError)
	})

	t.Run("appends error diagnostics and sets hasError", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{ec: ec}

		diagnostics := []*ast_domain.Diagnostic{
			{Severity: ast_domain.Warning, Message: "warning"},
			{Severity: ast_domain.Error, Message: "parse error"},
		}

		task.handleCSSDiagnostics(diagnostics)

		require.Len(t, ec.diagnostics, 2)
		assert.True(t, task.hasError)
	})
}

func TestMergeDynamicAttributes(t *testing.T) {
	t.Parallel()

	t.Run("merges target and invoker dynamic attributes sorted", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			TagName: "div",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "zebra", RawExpression: "state.z"},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "alpha", RawExpression: "state.a"},
			},
		}

		mergeDynamicAttributes(targetNode, invokerNode, "inv_pkg")

		require.Len(t, targetNode.DynamicAttributes, 2)
		assert.Equal(t, "alpha", targetNode.DynamicAttributes[0].Name)
		assert.Equal(t, "zebra", targetNode.DynamicAttributes[1].Name)
	})

	t.Run("invoker attribute overrides target with same name", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			TagName: "div",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.old"},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.new"},
			},
		}

		mergeDynamicAttributes(targetNode, invokerNode, "inv_pkg")

		require.Len(t, targetNode.DynamicAttributes, 1)
		assert.Equal(t, "state.new", targetNode.DynamicAttributes[0].RawExpression)
	})

	t.Run("initialises GoAnnotations when nil", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{TagName: "div"}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}

		mergeDynamicAttributes(targetNode, invokerNode, "pkg")

		require.NotNil(t, targetNode.GoAnnotations)
		require.NotNil(t, targetNode.GoAnnotations.DynamicAttributeOrigins)
	})
}

func TestGroupContentBySlotMultiplePSlotDirectives(t *testing.T) {
	t.Parallel()

	t.Run("multiple p-slot elements accumulate in same slot", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Location: ast_domain.Location{Line: 1, Column: 1},
				DirSlot:  &ast_domain.Directive{RawExpression: "sidebar"},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "nav",
				Location: ast_domain.Location{Line: 2, Column: 1},
				DirSlot:  &ast_domain.Directive{RawExpression: "sidebar"},
			},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result["sidebar"].Nodes, 2)
		assert.Equal(t, "div", result["sidebar"].Nodes[0].TagName)
		assert.Equal(t, "nav", result["sidebar"].Nodes[1].TagName)
		assert.Equal(t, 1, result["sidebar"].Location.Line, "location should come from first node")
	})

	t.Run("mixed slot types and default content", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:slot",
				Location: ast_domain.Location{Line: 1, Column: 1},
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "name", Value: "header"},
				},
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "Header content"},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "aside",
				Location: ast_domain.Location{Line: 2, Column: 1},
				DirSlot:  &ast_domain.Directive{RawExpression: "sidebar"},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Location: ast_domain.Location{Line: 3, Column: 1},
			},
		}

		result := groupContentBySlot(nodes)

		assert.Len(t, result, 3)
		assert.Len(t, result["header"].Nodes, 1)
		assert.Len(t, result["sidebar"].Nodes, 1)
		assert.Len(t, result[""].Nodes, 1)
	})
}

func TestCollectDefinedSlotsDoesNotRecurseIntoSlots(t *testing.T) {
	t.Parallel()

	t.Run("does not recurse into piko:slot children", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:slot",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "name", Value: "outer"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:slot",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "name", Value: "inner"},
						},
					},
				},
			},
		}

		result := collectDefinedSlots(nodes)

		assert.True(t, result["outer"])
		assert.False(t, result["inner"], "should not recurse into piko:slot children")
	})
}

func TestValidateSlots(t *testing.T) {
	t.Parallel()

	t.Run("no warning when all provided slots are defined", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec:        ec,
			userAlias: "card",
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.pk",
			},
			loadedPartial: &annotator_dto.ParsedComponent{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "piko:slot",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "name", Value: "header"},
							},
						},
						{
							NodeType:   ast_domain.NodeElement,
							TagName:    "piko:slot",
							Attributes: []ast_domain.HTMLAttribute{},
						},
					},
				},
			},
			groupedSlotContent: map[string]invokerSlotContent{
				"header": {
					Nodes:    []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeElement, TagName: "h1"}},
					Location: ast_domain.Location{Line: 1, Column: 1},
				},
				"": {
					Nodes:    []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeElement, TagName: "p"}},
					Location: ast_domain.Location{Line: 2, Column: 1},
				},
			},
		}

		task.validateSlots()

		assert.Empty(t, ec.diagnostics)
	})

	t.Run("warning when providing content for undefined named slot", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec:        ec,
			userAlias: "card",
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.pk",
			},
			loadedPartial: &annotator_dto.ParsedComponent{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
						},
					},
				},
			},
			groupedSlotContent: map[string]invokerSlotContent{
				"nonexistent": {
					Nodes:    []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeElement, TagName: "h1"}},
					Location: ast_domain.Location{Line: 5, Column: 3},
				},
			},
		}

		task.validateSlots()

		require.Len(t, ec.diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, ec.diagnostics[0].Severity)
		assert.Contains(t, ec.diagnostics[0].Message, "does not have a slot named 'nonexistent'")
	})

	t.Run("warning when providing default slot content but no default slot defined", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec:        ec,
			userAlias: "card",
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.pk",
			},
			loadedPartial: &annotator_dto.ParsedComponent{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
						},
					},
				},
			},
			groupedSlotContent: map[string]invokerSlotContent{
				"": {
					Nodes:    []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeElement, TagName: "p"}},
					Location: ast_domain.Location{Line: 3, Column: 1},
				},
			},
		}

		task.validateSlots()

		require.Len(t, ec.diagnostics, 1)
		assert.Equal(t, ast_domain.Warning, ec.diagnostics[0].Severity)
		assert.Contains(t, ec.diagnostics[0].Message, "does not have a default slot")
	})

	t.Run("empty grouped slot content produces no diagnostics", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec:        ec,
			userAlias: "card",
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/test/invoker.pk",
			},
			loadedPartial: &annotator_dto.ParsedComponent{
				Template: &ast_domain.TemplateAST{
					RootNodes: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
						},
					},
				},
			},
			groupedSlotContent: map[string]invokerSlotContent{},
		}

		task.validateSlots()

		assert.Empty(t, ec.diagnostics)
	})
}

func TestCheckCircularDependencies(t *testing.T) {
	t.Parallel()

	t.Run("no error when hasError is already true", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec:       ec,
			hasError: true,
		}

		task.checkCircularDependencies()

		assert.True(t, task.hasError)
		assert.Empty(t, ec.diagnostics)
	})

	t.Run("no error when path is not circular", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			expansionPath: []string{"/a.pk", "/b.pk"},
			diagnostics:   make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec: ec,
			loadedPartial: &annotator_dto.ParsedComponent{
				SourcePath: "/c.pk",
			},
			invokerNode: &ast_domain.TemplateNode{
				Location: ast_domain.Location{Line: 1, Column: 1},
			},
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/b.pk",
			},
			userAlias: "c",
		}

		task.checkCircularDependencies()

		assert.False(t, task.hasError)
		assert.Empty(t, ec.diagnostics)
	})

	t.Run("detects circular dependency", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			expansionPath: []string{"/a.pk", "/b.pk"},
			diagnostics:   make([]*ast_domain.Diagnostic, 0),
		}

		task := &partialExpansionTask{
			ec: ec,
			loadedPartial: &annotator_dto.ParsedComponent{
				SourcePath: "/a.pk",
			},
			invokerNode: &ast_domain.TemplateNode{
				Location: ast_domain.Location{Line: 5, Column: 3},
			},
			invokerComponent: &annotator_dto.ParsedComponent{
				SourcePath: "/b.pk",
			},
			userAlias: "a",
		}

		task.checkCircularDependencies()

		assert.True(t, task.hasError)
		require.Len(t, ec.diagnostics, 1)
		assert.Equal(t, ast_domain.Error, ec.diagnostics[0].Severity)
		assert.Contains(t, ec.diagnostics[0].Message, "Circular dependency detected")
		assert.Contains(t, ec.diagnostics[0].Message, "/a.pk")
		assert.Contains(t, ec.diagnostics[0].Message, "/b.pk")
	})
}

func TestRebuildSortedDynamicAttrs(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty map", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		invokerNode := &ast_domain.TemplateNode{}
		finalDynAttrs := map[string]ast_domain.DynamicAttribute{}

		result := rebuildSortedDynamicAttrs(finalDynAttrs, targetNode, invokerNode, "pkg")

		assert.Empty(t, result)
	})

	t.Run("returns attributes sorted by key", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: make(map[string]string),
			},
		}
		invokerNode := &ast_domain.TemplateNode{}

		finalDynAttrs := map[string]ast_domain.DynamicAttribute{
			"zebra": {Name: "zebra", RawExpression: "state.z"},
			"alpha": {Name: "alpha", RawExpression: "state.a"},
			"mid":   {Name: "mid", RawExpression: "state.m"},
		}

		result := rebuildSortedDynamicAttrs(finalDynAttrs, targetNode, invokerNode, "pkg")

		require.Len(t, result, 3)
		assert.Equal(t, "alpha", result[0].Name)
		assert.Equal(t, "mid", result[1].Name)
		assert.Equal(t, "zebra", result[2].Name)
	})

	t.Run("sets origin on attributes from invoker", func(t *testing.T) {
		t.Parallel()

		invokerOrigin := "invoker_pkg"
		targetNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				DynamicAttributeOrigins: map[string]string{
					"title": "invoker_pkg",
				},
			},
		}
		invokerNode := &ast_domain.TemplateNode{
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/invoker.pk"),
			},
		}

		finalDynAttrs := map[string]ast_domain.DynamicAttribute{
			"title": {Name: "title", RawExpression: "state.title"},
		}

		result := rebuildSortedDynamicAttrs(finalDynAttrs, targetNode, invokerNode, invokerOrigin)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].GoAnnotations)
		require.NotNil(t, result[0].GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "invoker_pkg", *result[0].GoAnnotations.OriginalPackageAlias)
		require.NotNil(t, result[0].GoAnnotations.OriginalSourcePath)
		assert.Equal(t, "/invoker.pk", *result[0].GoAnnotations.OriginalSourcePath)
	})
}

func TestProcessExpandedNodesFragmentIDAttributes(t *testing.T) {
	t.Parallel()

	t.Run("fragment children get sequential p-fragment-id attributes", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "header"},
			{NodeType: ast_domain.NodeElement, TagName: "main"},
			{NodeType: ast_domain.NodeElement, TagName: "footer"},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "frag_key",
			PartialAlias:       "layout",
			PartialPackageName: "layout_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		fragmentNode := result[0]
		require.Len(t, fragmentNode.Children, 3)

		for i, child := range fragmentNode.Children {
			var fragmentIDFound bool
			var fragmentIDValue string
			for _, attr := range child.Attributes {
				if attr.Name == "p-fragment-id" {
					fragmentIDFound = true
					fragmentIDValue = attr.Value
				}
			}
			assert.True(t, fragmentIDFound, "child %d should have p-fragment-id", i)
			assert.Equal(t, fmt.Sprintf("%d", i), fragmentIDValue, "child %d should have sequential id", i)
		}
	})

	t.Run("fragment node nil invoker GoAnnotations does not copy origin", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
			{NodeType: ast_domain.NodeElement, TagName: "span"},
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName:       "piko:partial",
			GoAnnotations: nil,
		}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "test_key",
			PartialAlias:       "comp",
			PartialPackageName: "comp_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		fragmentNode := result[0]
		require.NotNil(t, fragmentNode.GoAnnotations)
		assert.Nil(t, fragmentNode.GoAnnotations.OriginalPackageAlias)
		assert.Nil(t, fragmentNode.GoAnnotations.OriginalSourcePath)
	})

	t.Run("single root element with existing GoAnnotations", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("existing_pkg"),
				},
			},
		}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "single_key",
			PartialAlias:       "card",
			PartialPackageName: "card_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].GoAnnotations)
		assert.Equal(t, pInfo, result[0].GoAnnotations.PartialInfo)
		assert.Equal(t, "existing_pkg", *result[0].GoAnnotations.OriginalPackageAlias)
	})
}

func TestGroupContentBySlotEmptyNodes(t *testing.T) {
	t.Parallel()

	t.Run("nil nodes returns empty map", func(t *testing.T) {
		t.Parallel()

		result := groupContentBySlot(nil)

		assert.Empty(t, result)
	})

	t.Run("only whitespace and comment nodes produces empty map", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "   \n  "},
			{NodeType: ast_domain.NodeComment},
			{NodeType: ast_domain.NodeText, TextContent: "  "},
		}

		result := groupContentBySlot(nodes)

		assert.Empty(t, result)
	})

	t.Run("piko:slot with no name goes to default slot", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType:   ast_domain.NodeElement,
				TagName:    "piko:slot",
				Location:   ast_domain.Location{Line: 1, Column: 1},
				Attributes: []ast_domain.HTMLAttribute{},
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText, TextContent: "default content"},
				},
			},
		}

		result := groupContentBySlot(nodes)

		require.Len(t, result[""].Nodes, 1)
		assert.Equal(t, "default content", result[""].Nodes[0].TextContent)
	})
}

func TestApplyInvokerDirectivesNilNodes(t *testing.T) {
	t.Parallel()

	t.Run("nil target and nil invoker does not panic", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			applyInvokerAttributesToExpandedRoot(nil, nil)
		})
	})

	t.Run("invoker with no GoAnnotations still copies directives", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{TagName: "div"}
		invokerNode := &ast_domain.TemplateNode{
			TagName:       "piko:partial",
			GoAnnotations: nil,
			DirIf:         &ast_domain.Directive{RawExpression: "state.show"},
		}

		applyInvokerDirectives(targetNode, invokerNode, "pkg")

		require.NotNil(t, targetNode.DirIf)
		assert.Equal(t, "state.show", targetNode.DirIf.RawExpression)
		require.NotNil(t, targetNode.DirIf.GoAnnotations)
		assert.Equal(t, "pkg", *targetNode.DirIf.GoAnnotations.OriginalPackageAlias)
		assert.Nil(t, targetNode.DirIf.GoAnnotations.OriginalSourcePath)
	})
}

func TestFindDefaultSlotLocationNilSlice(t *testing.T) {
	t.Parallel()

	t.Run("nil nodes returns zero location", func(t *testing.T) {
		t.Parallel()

		result := findDefaultSlotLocation(nil)

		assert.Equal(t, 0, result.Line)
		assert.Equal(t, 0, result.Column)
		assert.Equal(t, 0, result.Offset)
	})
}

func TestHandleCSSErrorMessageFormat(t *testing.T) {
	t.Parallel()

	t.Run("diagnostic message includes prefix, source path, and error", func(t *testing.T) {
		t.Parallel()

		ec := &expansionContext{
			diagnostics: make([]*ast_domain.Diagnostic, 0),
		}
		task := &partialExpansionTask{
			ec: ec,
			loadedPartial: &annotator_dto.ParsedComponent{
				SourcePath: "/components/card.pk",
			},
		}

		_ = task.handleCSSError(
			errors.New("unexpected token at line 10"),
			"Fatal error scoping CSS",
			"<style> block",
			ast_domain.Location{Line: 3, Column: 1},
		)

		require.Len(t, ec.diagnostics, 1)
		diagnostic := ec.diagnostics[0]
		assert.Equal(t, ast_domain.Error, diagnostic.Severity)
		assert.Contains(t, diagnostic.Message, "Fatal error scoping CSS")
		assert.Contains(t, diagnostic.Message, "/components/card.pk")
		assert.Contains(t, diagnostic.Message, "unexpected token at line 10")
		assert.Equal(t, "<style> block", diagnostic.Expression)
	})
}

func TestCollectDefinedSlotsDeeplyNested(t *testing.T) {
	t.Parallel()

	t.Run("collects slots from deeply nested elements", func(t *testing.T) {
		t.Parallel()

		nodes := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "section",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "article",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "piko:slot",
										Attributes: []ast_domain.HTMLAttribute{
											{Name: "name", Value: "deep-slot"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result := collectDefinedSlots(nodes)

		assert.True(t, result["deep-slot"])
		assert.Len(t, result, 1)
	})
}

func TestMergeDynamicAttributesServerAndRequestSkip(t *testing.T) {
	t.Parallel()

	t.Run("skips server and request prefixed dynamic attributes from invoker", func(t *testing.T) {
		t.Parallel()

		targetNode := &ast_domain.TemplateNode{
			TagName: "div",
		}
		invokerNode := &ast_domain.TemplateNode{
			TagName: "piko:partial",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "server.data", RawExpression: "state.data"},
				{Name: "request.param", RawExpression: "state.param"},
				{Name: "title", RawExpression: "state.title"},
			},
		}

		mergeDynamicAttributes(targetNode, invokerNode, "inv_pkg")

		require.Len(t, targetNode.DynamicAttributes, 1)
		assert.Equal(t, "title", targetNode.DynamicAttributes[0].Name)
	})
}

func TestProcessExpandedNodesEmptyList(t *testing.T) {
	t.Parallel()

	t.Run("empty expanded nodes produces fragment with no children", func(t *testing.T) {
		t.Parallel()

		expandedNodes := []*ast_domain.TemplateNode{}
		invokerNode := &ast_domain.TemplateNode{TagName: "piko:partial"}
		pInfo := &ast_domain.PartialInvocationInfo{
			InvocationKey:      "empty_key",
			PartialAlias:       "empty",
			PartialPackageName: "empty_hash",
		}

		result := processExpandedNodes(expandedNodes, invokerNode, pInfo)

		require.Len(t, result, 1)
		require.NotNil(t, result[0].GoAnnotations)
		assert.Equal(t, pInfo, result[0].GoAnnotations.PartialInfo)
	})
}

func TestGetSortedAttrKeys_Deterministic(t *testing.T) {
	t.Parallel()

	attrs := map[string]ast_domain.DynamicAttribute{
		"zebra": {Name: "zebra", Expression: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, NameLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
		"alpha": {Name: "alpha", Expression: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, NameLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
		"mango": {Name: "mango", Expression: nil, Location: ast_domain.Location{Line: 0, Column: 0, Offset: 0}, NameLocation: ast_domain.Location{Line: 0, Column: 0, Offset: 0}},
	}

	result1 := getSortedAttrKeys(attrs)
	result2 := getSortedAttrKeys(attrs)

	require.Len(t, result1, 3)
	assert.True(t, sort.StringsAreSorted(result1))
	assert.Equal(t, result1, result2)
}
