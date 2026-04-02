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

package ast_domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestNode(name string) *TemplateNode {
	return &TemplateNode{
		NodeType:    NodeElement,
		TagName:     name,
		Location:    Location{Line: 1, Column: 1},
		TextContent: "some text",
		Attributes: []HTMLAttribute{
			{Name: "id", Value: name},
		},
		DynamicAttributes: []DynamicAttribute{
			{Name: "title", RawExpression: "'My Title'", Expression: &StringLiteral{Value: "My Title"}},
		},
		RichText: []TextPart{
			{IsLiteral: true, Literal: "Hello"},
		},
		OnEvents: map[string][]Directive{
			"click": {{Type: DirectiveOn, Arg: "click", Expression: &Identifier{Name: "doClick"}}},
		},
		Binds: map[string]*Directive{
			"prop": {Type: DirectiveBind, Arg: "prop", Expression: &Identifier{Name: "propVal"}},
		},
		DirIf:             &Directive{Type: DirectiveIf, Expression: &Identifier{Name: "show"}},
		Key:               &StringLiteral{Value: name + "-key"},
		IsContentEditable: true,
		Diagnostics: []*Diagnostic{
			{Message: "A test diagnostic"},
		},
		GoAnnotations: &GoGeneratorAnnotation{
			OriginalPackageAlias: new("main"),
			PartialInfo:          &PartialInvocationInfo{InvocationKey: "key123"},
			DynamicAttributeOrigins: map[string]string{
				"title": "parent",
			},
		},
		Children: []*TemplateNode{},
	}
}

func TestTemplateNode_Clone(t *testing.T) {
	t.Run("should return nil when cloning a nil node", func(t *testing.T) {
		var original *TemplateNode
		clone := original.Clone()
		assert.Nil(t, clone)
	})

	t.Run("should create a shallow copy of a simple node", func(t *testing.T) {
		original := &TemplateNode{
			NodeType:    NodeElement,
			TagName:     "div",
			TextContent: "simple",
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.NotSame(t, original, clone, "Clone should be a new object in memory")
		assert.Equal(t, original.TagName, clone.TagName)
		assert.Equal(t, original.TextContent, clone.TextContent)
	})

	t.Run("should not copy children", func(t *testing.T) {
		original := createTestNode("parent")
		original.Children = append(original.Children, createTestNode("child"))

		clone := original.Clone()

		require.NotNil(t, clone)
		assert.Len(t, original.Children, 1, "Original should have one child")
		assert.Nil(t, clone.Children, "Shallow clone's Children slice should be nil")
	})

	t.Run("should create new slice instances, proving two-way independence", func(t *testing.T) {
		original := createTestNode("node-with-slices")
		original.Diagnostics = []*Diagnostic{{Message: "An error"}}

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.NotSame(t, &original.Attributes, &clone.Attributes)
		assert.NotSame(t, &original.DynamicAttributes, &clone.DynamicAttributes)
		assert.NotSame(t, &original.RichText, &clone.RichText)
		assert.NotSame(t, &original.Diagnostics, &clone.Diagnostics)

		original.Attributes = append(original.Attributes, HTMLAttribute{Name: "class", Value: "test"})
		original.Diagnostics[0].Message = "Modified Message"

		assert.Len(t, original.Attributes, 2, "Original attributes should be modified")
		assert.Len(t, clone.Attributes, 1, "Clone's attributes should NOT be modified")
		assert.Equal(t, "An error", clone.Diagnostics[0].Message, "Modifying original's deep value should not affect clone's")

		clone.Attributes[0].Value = "cloned-value"
		assert.Equal(t, "node-with-slices", original.Attributes[0].Value, "Original attribute should be unaffected by clone modification")
	})

	t.Run("should handle nil vs empty slices correctly", func(t *testing.T) {

		withNilSlice := createTestNode("nil-slice-node")
		withNilSlice.RichText = nil
		cloneWithNil := withNilSlice.Clone()
		assert.Nil(t, cloneWithNil.RichText, "A nil slice should be cloned as a nil slice")

		withEmptySlice := createTestNode("empty-slice-node")
		withEmptySlice.RichText = []TextPart{}
		cloneWithEmpty := withEmptySlice.Clone()
		assert.NotNil(t, cloneWithEmpty.RichText, "An empty slice should be cloned as a non-nil empty slice")
		assert.Len(t, cloneWithEmpty.RichText, 0, "An empty slice should be cloned as a slice with length 0")
	})

	t.Run("should create new map instances, proving two-way independence", func(t *testing.T) {
		original := createTestNode("node-with-maps")

		clone := original.Clone()
		require.NotNil(t, clone)
		require.NotNil(t, clone.GoAnnotations)

		assert.NotEqual(t, fmt.Sprintf("%p", original.OnEvents), fmt.Sprintf("%p", clone.OnEvents), "OnEvents maps should be different instances")
		assert.NotEqual(t, fmt.Sprintf("%p", original.Binds), fmt.Sprintf("%p", clone.Binds), "Binds maps should be different instances")
		assert.NotEqual(t, fmt.Sprintf("%p", original.GoAnnotations.DynamicAttributeOrigins), fmt.Sprintf("%p", clone.GoAnnotations.DynamicAttributeOrigins), "DynamicAttributeOrigins maps should be different instances")

		original.OnEvents["mouseover"] = []Directive{{}}
		delete(original.Binds, "prop")
		original.GoAnnotations.DynamicAttributeOrigins["disabled"] = "self"

		assert.NotContains(t, clone.OnEvents, "mouseover")
		assert.Contains(t, clone.Binds, "prop")
		assert.NotContains(t, clone.GoAnnotations.DynamicAttributeOrigins, "disabled")

		clone.OnEvents["blur"] = []Directive{{}}
		assert.NotContains(t, original.OnEvents, "blur", "Original map should be unaffected by clone modification")
	})

	t.Run("should handle nil vs empty maps correctly", func(t *testing.T) {

		withNilMap := createTestNode("nil-map-node")
		withNilMap.Binds = nil
		cloneWithNil := withNilMap.Clone()
		assert.Nil(t, cloneWithNil.Binds, "A nil map should be cloned as a nil map")

		withEmptyMap := createTestNode("empty-map-node")
		withEmptyMap.Binds = make(map[string]*Directive)
		cloneWithEmpty := withEmptyMap.Clone()
		assert.NotNil(t, cloneWithEmpty.Binds, "An empty map should be cloned as a non-nil empty map")
		assert.Len(t, cloneWithEmpty.Binds, 0, "An empty map should be cloned as a map with length 0")
	})

	t.Run("should create new instances of pointer fields, proving two-way independence", func(t *testing.T) {
		original := createTestNode("node-with-pointers")

		clone := original.Clone()
		require.NotNil(t, clone)

		assert.NotSame(t, original.DirIf, clone.DirIf)
		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations)
		assert.NotSame(t, original.Key, clone.Key)

		original.DirIf.RawExpression = "modified"
		original.GoAnnotations.PartialInfo.InvocationKey = "modified-key"

		assert.Equal(t, "", clone.DirIf.RawExpression, "Modification to original DirIf should not be visible in clone")
		assert.Equal(t, "key123", clone.GoAnnotations.PartialInfo.InvocationKey, "Modification to original GoAnnotation should not be visible in clone")

		clone.DirIf.Expression = &BooleanLiteral{Value: false}
		assert.IsType(t, &Identifier{}, original.DirIf.Expression, "Original directive expression should be unaffected by clone modification")
	})

	t.Run("should handle pointers to built-in types correctly", func(t *testing.T) {
		pkgAlias := "original"
		original := &TemplateNode{
			GoAnnotations: &GoGeneratorAnnotation{OriginalPackageAlias: &pkgAlias},
		}
		clone := original.Clone()
		require.NotNil(t, clone.GoAnnotations)
		require.NotNil(t, clone.GoAnnotations.OriginalPackageAlias)

		assert.NotSame(t, original.GoAnnotations, clone.GoAnnotations, "The containing GoAnnotations struct should be a new instance")

		assert.Equal(t, original.GoAnnotations.OriginalPackageAlias, clone.GoAnnotations.OriginalPackageAlias, "Pointers to the string should be copied, pointing to the same memory address")
		assert.Equal(t, "original", *clone.GoAnnotations.OriginalPackageAlias)

		pkgAlias = "modified"

		assert.Equal(t, "modified", *original.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "modified", *clone.GoAnnotations.OriginalPackageAlias, "Change to the shared underlying variable should be visible via the clone's pointer")
	})
}

func TestTemplateNode_DeepClone(t *testing.T) {
	t.Run("should return nil when deep-cloning a nil node", func(t *testing.T) {
		var original *TemplateNode
		clone := original.DeepClone()
		assert.Nil(t, clone)
	})

	t.Run("should behave like shallow clone for a node with no children", func(t *testing.T) {
		original := createTestNode("simple-node")
		clone := original.DeepClone()

		require.NotNil(t, clone)
		assert.NotSame(t, original, clone)
		assert.Equal(t, original.TagName, clone.TagName)
		assert.Empty(t, clone.Children)
	})

	t.Run("should recursively clone children, making them independent", func(t *testing.T) {
		root := createTestNode("root")
		child := createTestNode("child")
		grandchild := createTestNode("grandchild")

		child.Children = append(child.Children, grandchild)
		root.Children = append(root.Children, child)

		clone := root.DeepClone()
		require.NotNil(t, clone)
		require.Len(t, clone.Children, 1)
		require.Len(t, clone.Children[0].Children, 1)

		cloneChild := clone.Children[0]
		cloneGrandchild := cloneChild.Children[0]

		assert.NotSame(t, root, clone, "Root should be a new object")
		assert.NotSame(t, child, cloneChild, "Child should be a new object")
		assert.NotSame(t, grandchild, cloneGrandchild, "Grandchild should be a new object")
		assert.NotSame(t, &root.Children, &clone.Children, "Children slice header should be different")
	})

	t.Run("should deep clone complex expression trees", func(t *testing.T) {
		originalIdent := &Identifier{Name: "originalName"}
		original := &TemplateNode{
			DirIf: &Directive{
				Expression: &BinaryExpression{
					Left:     &IntegerLiteral{Value: 1},
					Operator: OpPlus,
					Right:    originalIdent,
				},
			},
		}

		clone := original.DeepClone()
		require.NotNil(t, clone.DirIf)
		require.NotNil(t, clone.DirIf.Expression)
		clonedExpr, ok := clone.DirIf.Expression.(*BinaryExpression)
		require.True(t, ok)
		clonedIdent, ok := clonedExpr.Right.(*Identifier)
		require.True(t, ok)

		originalIdent.Name = "modifiedName"

		assert.NotSame(t, originalIdent, clonedIdent)
		assert.Equal(t, "originalName", clonedIdent.Name)
	})

	t.Run("should create a completely independent tree (full modification test)", func(t *testing.T) {
		root := createTestNode("root")
		child1 := createTestNode("child1")
		child2 := createTestNode("child2")
		grandchild := createTestNode("grandchild")

		child2.Children = append(child2.Children, grandchild)
		root.Children = []*TemplateNode{child1, child2}

		clone := root.DeepClone()
		require.NotNil(t, clone)
		cloneChild1 := clone.Children[0]
		cloneChild2 := clone.Children[1]
		cloneGrandchild := cloneChild2.Children[0]

		root.TagName = "new-root-name"
		root.TextContent = "modified text"
		root.Attributes = append(root.Attributes, HTMLAttribute{Name: "data-mod", Value: "true"})
		root.DynamicAttributes[0].RawExpression = "new expression"
		root.OnEvents["mouseover"] = []Directive{{}}
		root.OnEvents["click"][0].Arg = "modified-click"
		root.DirIf.Expression = &BooleanLiteral{Value: false}
		root.Key = &StringLiteral{Value: "new-key"}
		root.GoAnnotations.OriginalPackageAlias = new("new-pkg")
		child1.TagName = "new-child1-name"
		grandchild.TagName = "new-grandchild-name"
		child2.Children = append(child2.Children, createTestNode("new-sibling"))

		assert.Equal(t, "root", clone.TagName)
		assert.Equal(t, "some text", clone.TextContent)
		assert.Len(t, clone.Attributes, 1)
		assert.Equal(t, "'My Title'", clone.DynamicAttributes[0].RawExpression)
		assert.Len(t, clone.OnEvents, 1)
		assert.NotContains(t, clone.OnEvents, "mouseover")
		assert.Equal(t, "click", clone.OnEvents["click"][0].Arg)
		assert.Equal(t, "show", clone.DirIf.Expression.String())
		assert.Equal(t, `"root-key"`, clone.Key.String())
		require.NotNil(t, clone.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "main", *clone.GoAnnotations.OriginalPackageAlias)
		assert.Equal(t, "child1", cloneChild1.TagName)
		assert.Equal(t, "grandchild", cloneGrandchild.TagName)
		assert.Len(t, cloneChild2.Children, 1)

		clone.TagName = "cloned-root-name"
		clone.Attributes[0].Value = "cloned-id"
		cloneGrandchild.TagName = "cloned-grandchild"
		clone.Children = clone.Children[:1]

		assert.Equal(t, "new-root-name", root.TagName)
		assert.Equal(t, "root", root.Attributes[0].Value)
		assert.Equal(t, "new-grandchild-name", grandchild.TagName)
		assert.Len(t, root.Children, 2, "Original should still have 2 children")
	})
}

func TestTemplateAST_Cloning(t *testing.T) {
	createOriginalAST := func() *TemplateAST {
		ast := &TemplateAST{
			RootNodes: []*TemplateNode{
				createTestNode("root1"),
				createTestNode("root2"),
			},
			Diagnostics: []*Diagnostic{{Message: "An error"}},
		}
		ast.RootNodes[0].Children = append(ast.RootNodes[0].Children, createTestNode("child"))
		return ast
	}

	t.Run("shallow clone of TemplateAST", func(t *testing.T) {
		originalAST := createOriginalAST()
		clone := originalAST.Clone()
		require.NotNil(t, clone)
		assert.NotSame(t, originalAST, clone)

		assert.NotSame(t, &originalAST.RootNodes, &clone.RootNodes)
		assert.NotSame(t, &originalAST.Diagnostics, &clone.Diagnostics)

		assert.Same(t, originalAST.RootNodes[0], clone.RootNodes[0])
		assert.Same(t, originalAST.Diagnostics[0], clone.Diagnostics[0])

		originalAST.RootNodes[0].TagName = "modified-root"
		assert.Equal(t, "modified-root", clone.RootNodes[0].TagName)
	})

	t.Run("deep clone of TemplateAST", func(t *testing.T) {
		originalAST := createOriginalAST()
		deepClone := originalAST.DeepClone()
		require.NotNil(t, deepClone)
		assert.NotSame(t, originalAST, deepClone)

		assert.NotSame(t, &originalAST.RootNodes, &deepClone.RootNodes)

		assert.NotSame(t, originalAST.RootNodes[0], deepClone.RootNodes[0])
		assert.NotSame(t, originalAST.RootNodes[0].Children[0], deepClone.RootNodes[0].Children[0])

		originalAST.RootNodes[0].Children[0].TagName = "modified-child"
		assert.Equal(t, "child", deepClone.RootNodes[0].Children[0].TagName)

		deepClone.RootNodes[1].TagName = "cloned-root2"
		assert.Equal(t, "root2", originalAST.RootNodes[1].TagName)
	})
}

func TestDeepCloneWithScopeAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node           *TemplateNode
		checkClone     func(t *testing.T, clone *TemplateNode)
		name           string
		partialScopeID string
	}{
		{
			name: "simple div gets partial and p-key attributes",
			node: &TemplateNode{
				NodeType: NodeElement,
				TagName:  "div",
				Key:      &StringLiteral{Value: "r.0"},
				Attributes: []HTMLAttribute{
					{Name: "class", Value: "container"},
				},
			},
			partialScopeID: "my_partial_abc",
			checkClone: func(t *testing.T, clone *TemplateNode) {
				assert.True(t, hasAttributeByName(clone.Attributes, "partial"),
					"Clone should have a 'partial' attribute")
				assert.True(t, hasAttributeByName(clone.Attributes, "p-key"),
					"Clone should have a 'p-key' attribute")

				for _, attr := range clone.Attributes {
					if attr.Name == "partial" {
						assert.Equal(t, "my_partial_abc", attr.Value)
					}
					if attr.Name == "p-key" {
						assert.Equal(t, "r.0", attr.Value)
					}
				}
			},
		},
		{
			name: "nested nodes all receive scope attributes",
			node: &TemplateNode{
				NodeType: NodeElement,
				TagName:  "div",
				Key:      &StringLiteral{Value: "r.0"},
				Children: []*TemplateNode{
					{
						NodeType: NodeElement,
						TagName:  "span",
						Key:      &StringLiteral{Value: "r.0:0"},
					},
				},
			},
			partialScopeID: "scope_xyz",
			checkClone: func(t *testing.T, clone *TemplateNode) {
				assert.True(t, hasAttributeByName(clone.Attributes, "partial"),
					"Root clone should have 'partial' attribute")

				require.Len(t, clone.Children, 1)
				child := clone.Children[0]
				assert.True(t, hasAttributeByName(child.Attributes, "partial"),
					"Child clone should have 'partial' attribute")
				assert.True(t, hasAttributeByName(child.Attributes, "p-key"),
					"Child clone should have 'p-key' attribute")
			},
		},
		{
			name: "node with StringLiteral key gets p-key value",
			node: &TemplateNode{
				NodeType: NodeElement,
				TagName:  "p",
				Key:      &StringLiteral{Value: "my-key"},
			},
			partialScopeID: "scope_123",
			checkClone: func(t *testing.T, clone *TemplateNode) {
				var pkeyValue string
				for _, attr := range clone.Attributes {
					if attr.Name == "p-key" {
						pkeyValue = attr.Value
					}
				}
				assert.Equal(t, "my-key", pkeyValue)
			},
		},
		{
			name: "node already has partial attribute skips injection",
			node: &TemplateNode{
				NodeType: NodeElement,
				TagName:  "div",
				Key:      &StringLiteral{Value: "r.0"},
				Attributes: []HTMLAttribute{
					{Name: "partial", Value: "existing_scope"},
				},
			},
			partialScopeID: "new_scope",
			checkClone: func(t *testing.T, clone *TemplateNode) {
				partialCount := 0
				for _, attr := range clone.Attributes {
					if attr.Name == "partial" {
						partialCount++
						assert.Equal(t, "existing_scope", attr.Value,
							"Existing partial attribute should not be overwritten")
					}
				}
				assert.Equal(t, 1, partialCount,
					"Should not add a second 'partial' attribute")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clone := tc.node.DeepCloneWithScopeAttributes(tc.partialScopeID)

			require.NotNil(t, clone)
			assert.NotSame(t, tc.node, clone)
			tc.checkClone(t, clone)
		})
	}
}

func TestHasAttributeByName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		attributeName string
		attrs         []HTMLAttribute
		want          bool
	}{
		{
			name: "attribute present returns true",
			attrs: []HTMLAttribute{
				{Name: "class", Value: "foo"},
				{Name: "id", Value: "bar"},
			},
			attributeName: "class",
			want:          true,
		},
		{
			name: "attribute absent returns false",
			attrs: []HTMLAttribute{
				{Name: "class", Value: "foo"},
			},
			attributeName: "id",
			want:          false,
		},
		{
			name:          "empty slice returns false",
			attrs:         []HTMLAttribute{},
			attributeName: "class",
			want:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasAttributeByName(tc.attrs, tc.attributeName)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestExtractStaticKeyString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		key  Expression
		want string
	}{
		{
			name: "nil key returns empty string",
			key:  nil,
			want: "",
		},
		{
			name: "StringLiteral key returns value",
			key:  &StringLiteral{Value: "my-key-value"},
			want: "my-key-value",
		},
		{
			name: "non-string expression returns empty string",
			key:  &Identifier{Name: "dynamicKey"},
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractStaticKeyString(tc.key)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestHasAttributeByName_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		attributeName string
		attrs         []HTMLAttribute
		want          bool
	}{
		{
			name:          "nil slice returns false",
			attrs:         nil,
			attributeName: "class",
			want:          false,
		},
		{
			name: "case-sensitive match required",
			attrs: []HTMLAttribute{
				{Name: "Class", Value: "foo"},
			},
			attributeName: "class",
			want:          false,
		},
		{
			name: "matches first attribute in long list",
			attrs: []HTMLAttribute{
				{Name: "target", Value: "_blank"},
				{Name: "href", Value: "/page"},
				{Name: "class", Value: "link"},
			},
			attributeName: "target",
			want:          true,
		},
		{
			name: "matches last attribute in long list",
			attrs: []HTMLAttribute{
				{Name: "href", Value: "/page"},
				{Name: "class", Value: "link"},
				{Name: "target", Value: "_blank"},
			},
			attributeName: "target",
			want:          true,
		},
		{
			name: "attribute with empty value still matches",
			attrs: []HTMLAttribute{
				{Name: "disabled", Value: ""},
			},
			attributeName: "disabled",
			want:          true,
		},
		{
			name: "empty name does not match named attributes",
			attrs: []HTMLAttribute{
				{Name: "class", Value: "foo"},
			},
			attributeName: "",
			want:          false,
		},
		{
			name: "empty name matches attribute with empty name",
			attrs: []HTMLAttribute{
				{Name: "", Value: "foo"},
			},
			attributeName: "",
			want:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasAttributeByName(tc.attrs, tc.attributeName)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestExtractStaticKeyString_EdgeCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		key  Expression
		want string
	}{
		{
			name: "empty string literal returns empty string",
			key:  &StringLiteral{Value: ""},
			want: "",
		},
		{
			name: "integer literal returns empty string",
			key:  &IntegerLiteral{Value: 42},
			want: "",
		},
		{
			name: "boolean literal returns empty string",
			key:  &BooleanLiteral{Value: true},
			want: "",
		},
		{
			name: "string literal with special characters",
			key:  &StringLiteral{Value: "key-with-special_chars.and/slashes"},
			want: "key-with-special_chars.and/slashes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractStaticKeyString(tc.key)
			assert.Equal(t, tc.want, result)
		})
	}
}

func TestInjectScopeAttributes(t *testing.T) {
	t.Parallel()

	t.Run("adds partial attribute when not present", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		result := injectScopeAttributes(attrs, nil, "scope-123")

		require.Len(t, result, 2)
		assert.Equal(t, "class", result[0].Name)
		assert.Equal(t, "partial", result[1].Name)
		assert.Equal(t, "scope-123", result[1].Value)
	})

	t.Run("does not add partial when already present", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "partial", Value: "existing-scope"},
			{Name: "class", Value: "container"},
		}
		result := injectScopeAttributes(attrs, nil, "scope-123")

		assert.Len(t, result, 2)
		assert.Equal(t, "existing-scope", result[0].Value)
	})

	t.Run("does not add partial when partialScopeID is empty", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		result := injectScopeAttributes(attrs, nil, "")

		assert.Len(t, result, 1)
	})

	t.Run("adds p-key when key is present and p-key is absent", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		key := &StringLiteral{Value: "item-1"}
		result := injectScopeAttributes(attrs, key, "")

		require.Len(t, result, 2)
		assert.Equal(t, "p-key", result[1].Name)
		assert.Equal(t, "item-1", result[1].Value)
	})

	t.Run("does not add p-key when already present", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "p-key", Value: "existing-key"},
		}
		key := &StringLiteral{Value: "item-1"}
		result := injectScopeAttributes(attrs, key, "")

		assert.Len(t, result, 1)
		assert.Equal(t, "existing-key", result[0].Value)
	})

	t.Run("does not add p-key when key is nil", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		result := injectScopeAttributes(attrs, nil, "")

		assert.Len(t, result, 1)
	})

	t.Run("does not add p-key when key is not a string literal", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		key := &Identifier{Name: "dynamicKey"}
		result := injectScopeAttributes(attrs, key, "")

		assert.Len(t, result, 1)
	})

	t.Run("adds both partial and p-key when both are needed", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "class", Value: "container"},
		}
		key := &StringLiteral{Value: "item-1"}
		result := injectScopeAttributes(attrs, key, "scope-abc")

		require.Len(t, result, 3)
		assert.Equal(t, "class", result[0].Name)
		assert.Equal(t, "partial", result[1].Name)
		assert.Equal(t, "scope-abc", result[1].Value)
		assert.Equal(t, "p-key", result[2].Name)
		assert.Equal(t, "item-1", result[2].Value)
	})

	t.Run("returns original slice when nothing needs adding", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{
			{Name: "partial", Value: "scope-existing"},
			{Name: "p-key", Value: "key-existing"},
		}
		key := &StringLiteral{Value: "some-key"}
		result := injectScopeAttributes(attrs, key, "scope-new")

		assert.Len(t, result, 2)

		assert.Equal(t, "scope-existing", result[0].Value)
		assert.Equal(t, "key-existing", result[1].Value)
	})

	t.Run("handles empty attributes slice", func(t *testing.T) {
		t.Parallel()

		var attrs []HTMLAttribute
		key := &StringLiteral{Value: "item-key"}
		result := injectScopeAttributes(attrs, key, "scope-id")

		require.Len(t, result, 2)
		assert.Equal(t, "partial", result[0].Name)
		assert.Equal(t, "p-key", result[1].Name)
	})

	t.Run("does not add p-key for empty string literal key", func(t *testing.T) {
		t.Parallel()

		attrs := []HTMLAttribute{}
		key := &StringLiteral{Value: ""}
		result := injectScopeAttributes(attrs, key, "")

		assert.Empty(t, result)
	})
}

func TestTemplateNode_Reset(t *testing.T) {
	t.Parallel()

	node := createTestNode("reset-test")
	node.DirFor = &Directive{Type: DirectiveFor, RawExpression: "item in items"}
	node.DirShow = &Directive{Type: DirectiveShow, RawExpression: "visible"}
	node.DirModel = &Directive{Type: DirectiveModel, RawExpression: "inputVal"}
	node.DirElse = &Directive{Type: DirectiveElse}
	node.DirElseIf = &Directive{Type: DirectiveElseIf, RawExpression: "cond2"}
	node.DirRef = &Directive{Type: DirectiveRef, RawExpression: "myRef"}
	node.DirSlot = &Directive{Type: DirectiveSlot, RawExpression: "header"}
	node.DirClass = &Directive{Type: DirectiveClass}
	node.DirStyle = &Directive{Type: DirectiveStyle}
	node.DirText = &Directive{Type: DirectiveText}
	node.DirContext = &Directive{Type: DirectiveContext}
	node.DirScaffold = &Directive{Type: DirectiveScaffold}

	node.Reset()

	assert.Equal(t, NodeType(0), node.NodeType, "NodeType should be zeroed")
	assert.Equal(t, "", node.TagName, "TagName should be empty")
	assert.Equal(t, "", node.TextContent, "TextContent should be empty")
	assert.Equal(t, "", node.InnerHTML, "InnerHTML should be empty")
	assert.Equal(t, FormatAuto, node.PreferredFormat, "PreferredFormat should be FormatAuto")
	assert.False(t, node.IsPooled, "IsPooled should be false")
	assert.False(t, node.IsContentEditable, "IsContentEditable should be false")
	assert.False(t, node.PreserveWhitespace, "PreserveWhitespace should be false")

	assert.Nil(t, node.Attributes, "Attributes should be nil")
	assert.Nil(t, node.Children, "Children should be nil")
	assert.Nil(t, node.PrerenderedHTML, "PrerenderedHTML should be nil")

	assert.Nil(t, node.DirIf, "DirIf should be nil")
	assert.Nil(t, node.DirElseIf, "DirElseIf should be nil")
	assert.Nil(t, node.DirElse, "DirElse should be nil")
	assert.Nil(t, node.DirFor, "DirFor should be nil")
	assert.Nil(t, node.DirShow, "DirShow should be nil")
	assert.Nil(t, node.DirModel, "DirModel should be nil")
	assert.Nil(t, node.DirRef, "DirRef should be nil")
	assert.Nil(t, node.DirSlot, "DirSlot should be nil")
	assert.Nil(t, node.DirClass, "DirClass should be nil")
	assert.Nil(t, node.DirStyle, "DirStyle should be nil")
	assert.Nil(t, node.DirText, "DirText should be nil")
	assert.Nil(t, node.DirHTML, "DirHTML should be nil")
	assert.Nil(t, node.DirKey, "DirKey should be nil")
	assert.Nil(t, node.DirContext, "DirContext should be nil")
	assert.Nil(t, node.DirScaffold, "DirScaffold should be nil")
	assert.Nil(t, node.Key, "Key should be nil")
	assert.Nil(t, node.GoAnnotations, "GoAnnotations should be nil")
}

func TestPropValue_GetSetGoAnnotation(t *testing.T) {
	t.Parallel()

	t.Run("set annotation and get it back", func(t *testing.T) {
		t.Parallel()
		pv := &PropValue{}
		ann := &GoGeneratorAnnotation{
			DynamicAttributeOrigins: map[string]string{"foo": "bar"},
		}
		pv.SetGoAnnotation(ann)
		result := pv.GetGoAnnotation()
		assert.Same(t, ann, result, "GetGoAnnotation should return the same annotation that was set")
	})

	t.Run("nil annotation returns nil", func(t *testing.T) {
		t.Parallel()
		pv := &PropValue{}
		result := pv.GetGoAnnotation()
		assert.Nil(t, result, "GetGoAnnotation on a fresh PropValue should return nil")
	})
}
