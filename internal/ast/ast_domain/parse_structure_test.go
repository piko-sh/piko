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

func TestParse_Structure_VoidElements(t *testing.T) {
	t.Parallel()

	t.Run("void element does not prematurely close its parent", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
    <p>Some text</p>
    <input type="text">
    <button>A button after the input</button>
</div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1, "Should only have one root node (the div)")
		divNode := tree.RootNodes[0]
		require.Equal(t, "div", divNode.TagName)

		require.Len(t, divNode.Children, 3, "Div should contain three children")

		assert.Equal(t, "p", divNode.Children[0].TagName)
		assert.Equal(t, "input", divNode.Children[1].TagName)
		assert.Equal(t, "button", divNode.Children[2].TagName)
	})

	t.Run("multiple sequential void elements are parsed correctly", func(t *testing.T) {
		t.Parallel()

		source := `
<section>
    <hr>
    <input type="text">
    <br>
    <p>Content</p>
</section>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		sectionNode := tree.RootNodes[0]
		require.Equal(t, "section", sectionNode.TagName)

		require.Len(t, sectionNode.Children, 4, "Section should have 4 children")
		assert.Equal(t, "hr", sectionNode.Children[0].TagName)
		assert.Equal(t, "input", sectionNode.Children[1].TagName)
		assert.Equal(t, "br", sectionNode.Children[2].TagName)
		assert.Equal(t, "p", sectionNode.Children[3].TagName)
	})

	t.Run("void element with self-closing syntax does not prematurely close parent", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
    <button>Button Before</button>
    <input type="text" />
    <button>Button After</button>
</div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1, "Should only have one root node")
		divNode := tree.RootNodes[0]

		require.Len(t, divNode.Children, 3, "Div should contain three children")
		assert.Equal(t, "button", divNode.Children[0].TagName, "First child should be the 'before' button")
		assert.Equal(t, "input", divNode.Children[1].TagName, "Second child should be the input")
		assert.Equal(t, "button", divNode.Children[2].TagName, "Third child should be the 'after' button")
	})

	t.Run("nested structure with void elements at different levels", func(t *testing.T) {
		t.Parallel()

		source := `
<main>
    <article>
        <h1>Title</h1>
        <hr />
        <p>Paragraph 1</p>
    </article>
    <aside>
        <input type="search">
        <button>Search</button>
    </aside>
</main>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		mainNode := tree.RootNodes[0]
		require.Equal(t, "main", mainNode.TagName)

		require.Len(t, mainNode.Children, 2, "Main should have two children: article and aside")
		articleNode := mainNode.Children[0]
		asideNode := mainNode.Children[1]

		require.Equal(t, "article", articleNode.TagName)
		require.Len(t, articleNode.Children, 3, "Article should have h1, hr, and p")
		assert.Equal(t, "h1", articleNode.Children[0].TagName)
		assert.Equal(t, "hr", articleNode.Children[1].TagName)
		assert.Equal(t, "p", articleNode.Children[2].TagName)

		require.Equal(t, "aside", asideNode.TagName)
		require.Len(t, asideNode.Children, 2, "Aside should have input and button")
		assert.Equal(t, "input", asideNode.Children[0].TagName)
		assert.Equal(t, "button", asideNode.Children[1].TagName)
	})
}

func TestParse_Structure_VoidElementsWithAttributes(t *testing.T) {
	t.Parallel()

	t.Run("void element with static attribute is parsed correctly", func(t *testing.T) {
		t.Parallel()

		source := `
<main>
    <p>Before</p>
    <input type="text" value="hello">
    <p>After</p>
</main>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		mainNode := tree.RootNodes[0]

		require.Len(t, mainNode.Children, 3, "Main should have three children")
		assert.Equal(t, "p", mainNode.Children[0].TagName)
		assert.Equal(t, "input", mainNode.Children[1].TagName)
		assert.Equal(t, "p", mainNode.Children[2].TagName)

		pAfter := mainNode.Children[2]
		require.Len(t, pAfter.Children, 1)
		assert.Equal(t, "After", pAfter.Children[0].TextContent)
	})

	t.Run("void element with dynamic attribute is parsed correctly", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
    <input :readonly="isEditable">
    <button>Sibling Button</button>
</div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1, "Should only have one root node")
		divNode := tree.RootNodes[0]

		require.Len(t, divNode.Children, 2, "Div should have two children: input and button")
		assert.Equal(t, "input", divNode.Children[0].TagName)
		assert.Equal(t, "button", divNode.Children[1].TagName)

		inputNode := divNode.Children[0]
		require.Len(t, inputNode.DynamicAttributes, 1)
		assert.Equal(t, "readonly", inputNode.DynamicAttributes[0].Name)
	})

	t.Run("void element with self-closing slash and dynamic attribute is parsed correctly", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
    <input :readonly="isEditable" />
    <button>Sibling Button</button>
</div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1, "Should only have one root node")
		divNode := tree.RootNodes[0]

		require.Len(t, divNode.Children, 2, "Div should have two children: input and button")
		assert.Equal(t, "input", divNode.Children[0].TagName)
		assert.Equal(t, "button", divNode.Children[1].TagName)
	})

	t.Run("void element with multiple dynamic attributes and directives", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
	<p>Paragraph before</p>
    <hr p-if="showLine" :class="lineClass" />
    <p>Paragraph after</p>
</div>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)
		divNode := tree.RootNodes[0]
		require.Len(t, divNode.Children, 3)

		assert.Equal(t, "p", divNode.Children[0].TagName)
		assert.Equal(t, "hr", divNode.Children[1].TagName)
		assert.Equal(t, "p", divNode.Children[2].TagName)

		hrNode := divNode.Children[1]
		require.NotNil(t, hrNode.DirIf)
		assert.Equal(t, "showLine", hrNode.DirIf.RawExpression)
		require.Len(t, hrNode.DynamicAttributes, 1)
		assert.Equal(t, "class", hrNode.DynamicAttributes[0].Name)
		assert.Equal(t, "lineClass", hrNode.DynamicAttributes[0].RawExpression)
	})
}

func TestParse_Structure_ExactFailingCase(t *testing.T) {
	t.Parallel()

	t.Run("replicates the exact structure of the boolean attribute binding component", func(t *testing.T) {
		t.Parallel()

		source := `
<div>
    <button :disabled="state.isButtonDisabled">Click me</button>
    <input type="text" :readonly="!state.isEditable">
    <button p-on:click="toggle">Toggle</button>
</div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")

		t.Logf("AST DUMP FOR EXACT FAILING CASE:\n%s", DumpAST(context.Background(), tree))

		require.NoError(t, err, "The parser returned a fatal error")
		assertNoError(t, tree.Diagnostics, "The parser produced diagnostics for a valid template")

		require.Len(t, tree.RootNodes, 1, "The root of the AST should be a single <div> element")

		divNode := tree.RootNodes[0]
		require.Equal(t, "div", divNode.TagName)

		var elementChildren []*TemplateNode
		for _, child := range divNode.Children {
			if child.NodeType == NodeElement {
				elementChildren = append(elementChildren, child)
			}
		}
		require.Len(t, elementChildren, 3, "The <div> should contain exactly three element children")

		child1 := elementChildren[0]
		assert.Equal(t, "button", child1.TagName, "First child should be the disabled button")
		require.Len(t, child1.DynamicAttributes, 1, "First button should have one dynamic attribute")
		assert.Equal(t, "disabled", child1.DynamicAttributes[0].Name)

		child2 := elementChildren[1]
		assert.Equal(t, "input", child2.TagName, "Second child should be the input")
		require.Len(t, child2.Attributes, 1, "Input should have a static 'type' attribute")
		assert.Equal(t, "type", child2.Attributes[0].Name)
		require.Len(t, child2.DynamicAttributes, 1, "Input should have one dynamic attribute")
		assert.Equal(t, "readonly", child2.DynamicAttributes[0].Name)

		child3 := elementChildren[2]
		assert.Equal(t, "button", child3.TagName, "Third child should be the toggle button")

		require.Empty(t, child3.Directives, "The raw Directives slice should be empty after distribution")
		require.Contains(t, child3.OnEvents, "click", "Toggle button should have a click event handler in its OnEvents map")
		require.Len(t, child3.OnEvents["click"], 1, "Toggle button should have exactly one click handler")

		clickDirective := child3.OnEvents["click"][0]
		assert.Equal(t, DirectiveOn, clickDirective.Type)
		assert.Equal(t, "toggle", clickDirective.RawExpression)
		require.NotNil(t, clickDirective.Expression, "The directive's expression should have been parsed")
		assert.Equal(t, "toggle", clickDirective.Expression.String())
	})
}
