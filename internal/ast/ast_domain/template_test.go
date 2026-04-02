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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateNode_DistributeDirectives(t *testing.T) {
	t.Parallel()

	mockExprIf := &Identifier{Name: "show"}
	mockExprFor := &ForInExpression{ItemVariable: &Identifier{Name: "item"}, Collection: &Identifier{Name: "items"}}
	mockExprShow := &Identifier{Name: "isVisible"}
	mockExprBind := &Identifier{Name: "url"}
	mockExprModel := &Identifier{Name: "form.name"}

	rawRefValue := "myCanvas"
	mockExprClass := &Identifier{Name: "classObject"}
	mockExprStyle := &Identifier{Name: "styleObject"}
	mockExprText := &Identifier{Name: "user.name"}
	mockExprHTML := &Identifier{Name: "rawHTML"}
	mockExprOnClick := &Identifier{Name: "handleClick"}
	mockExprOnUpdate := &Identifier{Name: "handleUpdate"}

	t.Run("distributes all directive types correctly", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			Directives: []Directive{
				{Type: DirectiveIf, Expression: mockExprIf},
				{Type: DirectiveFor, Expression: mockExprFor},
				{Type: DirectiveShow, Expression: mockExprShow},
				{Type: DirectiveBind, Arg: "href", Expression: mockExprBind},
				{Type: DirectiveModel, Expression: mockExprModel},
				{Type: DirectiveRef, RawExpression: rawRefValue},
				{Type: DirectiveClass, Expression: mockExprClass},
				{Type: DirectiveStyle, Expression: mockExprStyle},
				{Type: DirectiveText, Expression: mockExprText},
				{Type: DirectiveHTML, Expression: mockExprHTML},
				{Type: DirectiveOn, Arg: "click", Expression: mockExprOnClick},
				{Type: DirectiveEvent, Arg: "update", Expression: mockExprOnUpdate},
			},
		}

		node.distributeDirectives()

		require.NotNil(t, node.DirIf)
		assert.Same(t, mockExprIf, node.DirIf.Expression, "DirIf expression mismatch")

		require.NotNil(t, node.DirFor)
		assert.Same(t, mockExprFor, node.DirFor.Expression, "DirFor expression mismatch")

		require.NotNil(t, node.DirShow)
		assert.Same(t, mockExprShow, node.DirShow.Expression, "DirShow expression mismatch")

		require.NotNil(t, node.DirModel)
		assert.Same(t, mockExprModel, node.DirModel.Expression, "DirModel expression mismatch")

		require.NotNil(t, node.DirRef)
		assert.Equal(t, rawRefValue, node.DirRef.RawExpression, "DirRef RawExpression mismatch")

		require.NotNil(t, node.DirClass)
		assert.Same(t, mockExprClass, node.DirClass.Expression, "DirClass expression mismatch")

		require.NotNil(t, node.DirStyle)
		assert.Same(t, mockExprStyle, node.DirStyle.Expression, "DirStyle expression mismatch")

		require.NotNil(t, node.DirText)
		assert.Same(t, mockExprText, node.DirText.Expression, "DirText expression mismatch")

		require.NotNil(t, node.DirHTML)
		assert.Same(t, mockExprHTML, node.DirHTML.Expression, "DirHTML expression mismatch")

		require.NotNil(t, node.Binds)
		require.Contains(t, node.Binds, "href")
		require.NotNil(t, node.Binds["href"])
		assert.Same(t, mockExprBind, node.Binds["href"].Expression)

		require.Contains(t, node.OnEvents, "click")
		require.Len(t, node.OnEvents["click"], 1)
		assert.Same(t, mockExprOnClick, node.OnEvents["click"][0].Expression)

		require.Contains(t, node.CustomEvents, "update")
		require.Len(t, node.CustomEvents["update"], 1)
		assert.Same(t, mockExprOnUpdate, node.CustomEvents["update"][0].Expression)

		assert.Empty(t, node.Directives, "Directives slice should be empty after distribution")
	})

	t.Run("handles multiple event handlers for the same event", func(t *testing.T) {
		t.Parallel()

		mockHandler1 := &Identifier{Name: "handler1"}
		mockHandler2 := &Identifier{Name: "handler2"}
		node := &TemplateNode{
			Directives: []Directive{
				{Type: DirectiveOn, Arg: "click", Expression: mockHandler1},
				{Type: DirectiveOn, Arg: "click", Expression: mockHandler2},
			},
		}

		node.distributeDirectives()

		require.Contains(t, node.OnEvents, "click")
		assert.Len(t, node.OnEvents["click"], 2)
		assert.Same(t, mockHandler1, node.OnEvents["click"][0].Expression)
		assert.Same(t, mockHandler2, node.OnEvents["click"][1].Expression)
	})

	t.Run("handles p-else directive without expression", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			Directives: []Directive{
				{Type: DirectiveElse},
			},
		}

		node.distributeDirectives()

		require.NotNil(t, node.DirElse, "DirElse should be populated with the directive pointer")
		assert.Equal(t, DirectiveElse, node.DirElse.Type)
		assert.Nil(t, node.DirIf, "DirIf should not be affected by p-else at this stage")
	})

	t.Run("ignores directives with nil Expression (except p-else)", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			Directives: []Directive{
				{Type: DirectiveIf, Expression: nil},
			},
		}

		node.distributeDirectives()

		assert.Nil(t, node.DirIf, "DirIf should be nil when Expression is nil")
		assert.Empty(t, node.Directives, "Directives slice should be empty as nil expression directives are skipped")
	})

	t.Run("initialises maps if they are nil but only when needed", func(t *testing.T) {
		t.Parallel()

		mockExprOnClick := &Identifier{Name: "handleClick"}
		mockExprBind := &Identifier{Name: "url"}

		node := &TemplateNode{
			Directives: []Directive{
				{Type: DirectiveOn, Arg: "click", Expression: mockExprOnClick},
				{Type: DirectiveBind, Arg: "src", Expression: mockExprBind},
			},
		}
		require.Nil(t, node.OnEvents)
		require.Nil(t, node.CustomEvents)
		require.Nil(t, node.Binds)

		node.distributeDirectives()

		assert.NotNil(t, node.OnEvents)
		assert.NotNil(t, node.Binds)

		assert.Nil(t, node.CustomEvents, "CustomEvents map should not be initialised if no p-event directives are present")

		assert.Contains(t, node.OnEvents, "click")
		assert.Contains(t, node.Binds, "src")
	})
}

func TestTemplateNode_DistributeDirectivesRecursively(t *testing.T) {
	t.Parallel()

	mockParentExpr := &Identifier{Name: "parentCondition"}
	mockChildExpr := &ForInExpression{ItemVariable: &Identifier{Name: "item"}, Collection: &Identifier{Name: "items"}}

	childNode := &TemplateNode{
		Directives: []Directive{
			{Type: DirectiveFor, Expression: mockChildExpr},
		},
	}
	parentNode := &TemplateNode{
		Directives: []Directive{
			{Type: DirectiveIf, Expression: mockParentExpr},
		},
		Children: []*TemplateNode{childNode},
	}

	parentNode.distributeDirectivesRecursively()

	require.NotNil(t, parentNode.DirIf)
	assert.Same(t, mockParentExpr, parentNode.DirIf.Expression, "Parent node's DirIf should be populated")
	assert.Empty(t, parentNode.Directives, "Parent's directives should be cleared")

	require.NotNil(t, childNode.DirFor)
	assert.Same(t, mockChildExpr, childNode.DirFor.Expression, "Child node's DirFor should be populated")
	assert.Empty(t, childNode.Directives, "Child's directives should be cleared")
}

func TestTemplateNode_Text(t *testing.T) {
	t.Parallel()

	tree := mustParse(t, `
		<div id="main">
			Hello
			<!-- a comment -->
			<span>World</span>
			<p>Count: {{ counter }}!</p>
		</div>
	`)
	mainDiv := findNodeByTagFromRoots(t, tree.RootNodes, "div")

	t.Run("Text() concatenates literal text and ignores expressions and comments", func(t *testing.T) {
		t.Parallel()

		expected := "Hello World Count: !"
		assert.Equal(t, expected, mainDiv.Text(context.Background()))
	})

	t.Run("RawText() concatenates literal text and includes raw expressions", func(t *testing.T) {
		t.Parallel()

		expected := "Hello World Count: {{ counter }}!"
		assert.Equal(t, expected, mainDiv.RawText(context.Background()))
	})

	t.Run("Text() on a simple text node", func(t *testing.T) {
		t.Parallel()

		pNode := findNodeByTag(t, mainDiv, "p")
		require.NotNil(t, pNode)
		assert.Equal(t, "Count: !", pNode.Text(context.Background()))
	})

	t.Run("Text() on a nil node returns empty string", func(t *testing.T) {
		t.Parallel()

		var nilNode *TemplateNode
		assert.Equal(t, "", nilNode.Text(context.Background()))
		assert.Equal(t, "", nilNode.RawText(context.Background()))
	})
}

func TestTemplateNode_AttributeHelpers(t *testing.T) {
	node := &TemplateNode{
		NodeType: NodeElement,
		TagName:  "div",
		Attributes: []HTMLAttribute{
			{Name: "id", Value: "main"},
			{Name: "class", Value: "container"},
			{Name: "disabled", Value: ""},
		},
	}

	t.Run("GetAttribute", func(t *testing.T) {
		value, found := node.GetAttribute("id")
		assert.True(t, found)
		assert.Equal(t, "main", value)

		value, found = node.GetAttribute("class")
		assert.True(t, found)
		assert.Equal(t, "container", value)

		value, found = node.GetAttribute("disabled")
		assert.True(t, found)
		assert.Equal(t, "", value)

		_, found = node.GetAttribute("href")
		assert.False(t, found)
	})

	t.Run("HasAttribute", func(t *testing.T) {
		assert.True(t, node.HasAttribute("id"))
		assert.True(t, node.HasAttribute("class"))
		assert.False(t, node.HasAttribute("href"))
	})

	t.Run("SetAttribute", func(t *testing.T) {
		node.SetAttribute("id", "app")
		value, _ := node.GetAttribute("id")
		assert.Equal(t, "app", value)

		node.SetAttribute("data-test", "true")
		value, _ = node.GetAttribute("data-test")
		assert.Equal(t, "true", value)
		assert.Len(t, node.Attributes, 4)
	})

	t.Run("RemoveAttribute", func(t *testing.T) {
		node.RemoveAttribute("disabled")
		assert.False(t, node.HasAttribute("disabled"))
		assert.Len(t, node.Attributes, 3)

		node.RemoveAttribute("class")
		assert.False(t, node.HasAttribute("class"))
		assert.Len(t, node.Attributes, 2)
	})
}

func TestTemplateNode_ClassHelpers(t *testing.T) {
	node := &TemplateNode{
		NodeType: NodeElement,
		TagName:  "div",
	}

	t.Run("Classes returns nil when no class attribute exists", func(t *testing.T) {
		assert.Nil(t, node.Classes())
	})

	node.SetAttribute("class", "  card   active   ")

	t.Run("Classes returns a clean slice of class names", func(t *testing.T) {
		assert.Equal(t, []string{"card", "active"}, node.Classes())
	})

	t.Run("HasClass checks for existence", func(t *testing.T) {
		assert.True(t, node.HasClass("card"))
		assert.True(t, node.HasClass("active"))
		assert.False(t, node.HasClass("container"))
	})

	t.Run("AddClass adds new classes and handles duplicates", func(t *testing.T) {
		node.AddClass("featured", "card", "  new-class  ")

		assert.Equal(t, "active card featured new-class", node.Attributes[0].Value)
	})
}

func TestTemplateNode_StructuralHelpers(t *testing.T) {
	t.Parallel()

	tree := mustParse(t, `
		<div id="root">
			<!-- comment -->
			First text
			<p id="p1"></p>
			<hr>
			Second text
			<p id="p2"></p>
		</div>
	`)
	root := findNodeByTagFromRoots(t, tree.RootNodes, "div")
	require.Len(t, root.Children, 6)

	t.Run("ChildElementCount ignores non-element nodes", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 3, root.ChildElementCount())
	})

	t.Run("FirstElementChild finds the first element", func(t *testing.T) {
		t.Parallel()

		first := root.FirstElementChild()
		require.NotNil(t, first)
		assert.Equal(t, "p", first.TagName)
		assert.True(t, first.HasAttribute("id"))
		value, _ := first.GetAttribute("id")
		assert.Equal(t, "p1", value)
	})

	t.Run("LastElementChild finds the last element", func(t *testing.T) {
		t.Parallel()

		last := root.LastElementChild()
		require.NotNil(t, last)
		assert.Equal(t, "p", last.TagName)
		value, _ := last.GetAttribute("id")
		assert.Equal(t, "p2", value)
	})

	t.Run("helpers on node with no element children return nil/zero", func(t *testing.T) {
		t.Parallel()

		p1 := findNodeByTag(t, root, "p")
		assert.Equal(t, 0, p1.ChildElementCount())
		assert.Nil(t, p1.FirstElementChild())
		assert.Nil(t, p1.LastElementChild())
	})
}

func TestTemplateNode_DirectiveHelpers(t *testing.T) {
	t.Parallel()

	createNodeWithRawDirectives := func() *TemplateNode {
		return &TemplateNode{
			NodeType: NodeElement,
			Directives: []Directive{
				{Type: DirectiveIf, RawExpression: "cond1"},
				{Type: DirectiveOn, Arg: "click", RawExpression: "handler1"},
				{Type: DirectiveOn, Arg: "submit", RawExpression: "handler2"},
				{Type: DirectiveBind, Arg: "href", RawExpression: "url"},
			},
		}
	}

	t.Run("helpers return nil/false before distribution pass", func(t *testing.T) {
		t.Parallel()

		node := createNodeWithRawDirectives()

		assert.Nil(t, node.GetDirective(DirectiveIf), "GetDirective should be nil before distribution")
		assert.Nil(t, node.GetDirectives(DirectiveOn), "GetDirectives should be nil for repeatable directives before distribution")
		assert.False(t, node.HasDirective(DirectiveIf), "HasDirective should be false before distribution")
		assert.False(t, node.HasDirective(DirectiveOn), "HasDirective should be false for repeatable directives before distribution")
	})

	t.Run("helpers on nil node", func(t *testing.T) {
		t.Parallel()

		var nilNode *TemplateNode
		assert.Nil(t, nilNode.GetDirective(DirectiveIf))
		assert.Nil(t, nilNode.GetDirectives(DirectiveOn))
		assert.False(t, nilNode.HasDirective(DirectiveIf))
	})
}
