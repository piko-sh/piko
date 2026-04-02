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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_BasicStructure(t *testing.T) {
	t.Run("simple nested elements", func(t *testing.T) {
		source := `<div><p>Hello</p></div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		divNode := tree.RootNodes[0]
		assert.Equal(t, NodeElement, divNode.NodeType)
		assert.Equal(t, "div", divNode.TagName)

		require.Len(t, divNode.Children, 1)
		pNode := divNode.Children[0]
		assert.Equal(t, NodeElement, pNode.NodeType)
		assert.Equal(t, "p", pNode.TagName)

		require.Len(t, pNode.Children, 1)
		textNode := pNode.Children[0]
		assert.Equal(t, NodeText, textNode.NodeType)
		assert.Equal(t, "Hello", textNode.TextContent)
	})

	t.Run("text and element siblings", func(t *testing.T) {
		source := `Before <span>Inside</span> After`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 3)
		assert.Equal(t, NodeText, tree.RootNodes[0].NodeType)
		assert.Equal(t, "Before ", tree.RootNodes[0].TextContent)

		assert.Equal(t, NodeElement, tree.RootNodes[1].NodeType)
		assert.Equal(t, "span", tree.RootNodes[1].TagName)

		assert.Equal(t, NodeText, tree.RootNodes[2].NodeType)
		assert.Equal(t, " After", tree.RootNodes[2].TextContent)
	})

	t.Run("comment node", func(t *testing.T) {
		source := `<!-- This is a comment -->`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		commentNode := tree.RootNodes[0]
		assert.Equal(t, NodeComment, commentNode.NodeType)
		assert.Equal(t, " This is a comment ", commentNode.TextContent)
	})

	t.Run("self-closing tag", func(t *testing.T) {
		source := `<input type="text" />`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		inputNode := tree.RootNodes[0]
		assert.Equal(t, "input", inputNode.TagName)
		assert.Empty(t, inputNode.Children)
	})

	t.Run("ignores top-level html, head, body wrappers", func(t *testing.T) {
		source := `
<html>
	<head><title>Test</title></head>
	<body><h1>Title</h1><p>Content</p></body>
</html>
		`
		tree := mustParse(t, strings.TrimSpace(source))

		require.Len(t, tree.RootNodes, 3)
		assert.Equal(t, "title", tree.RootNodes[0].TagName)
		assert.Equal(t, "h1", tree.RootNodes[1].TagName)
		assert.Equal(t, "p", tree.RootNodes[2].TagName)
	})
}

func TestParse_Directives(t *testing.T) {
	t.Run("p-if", func(t *testing.T) {
		tree := mustParse(t, `<div p-if="user.isActive"></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirIf)
		assertExprString(t, "user.isActive", node.DirIf.Expression)
	})

	t.Run("p-else-if", func(t *testing.T) {
		tree := mustParse(t, `<i p-if="false"></i><div p-else-if="user.isAdmin"></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirElseIf, "DirElseIf should be populated")
		assert.Nil(t, node.DirIf, "DirIf should be nil on an else-if node")
		assertExprString(t, "user.isAdmin", node.DirElseIf.Expression)
	})

	t.Run("p-else", func(t *testing.T) {
		tree := mustParse(t, `<i p-if="false"></i><div p-else></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirElse, "DirElse should be populated")
		assert.Nil(t, node.DirIf, "DirIf should be nil on an else node")
		assert.Nil(t, node.DirElse.Expression, "p-else directive should have no expression")
	})

	t.Run("p-for", func(t *testing.T) {
		tree := mustParse(t, `<li p-for="item in items"></li>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "li")

		require.NotNil(t, node.DirFor)
		assertExprString(t, "item in items", node.DirFor.Expression)
	})

	t.Run("p-show", func(t *testing.T) {
		tree := mustParse(t, `<p p-show="isVisible"></p>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "p")

		require.NotNil(t, node.DirShow)
		assertExprString(t, "isVisible", node.DirShow.Expression)
	})

	t.Run("p-model", func(t *testing.T) {
		tree := mustParse(t, `<input p-model="form.name" />`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "input")

		require.NotNil(t, node.DirModel)
		assertExprString(t, "form.name", node.DirModel.Expression)
	})

	t.Run("p-ref", func(t *testing.T) {

		tree := mustParse(t, `<canvas p-ref="myCanvas"></canvas>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "canvas")

		require.NotNil(t, node.DirRef)
		require.Equal(t, "myCanvas", node.DirRef.RawExpression)
		require.Nil(t, node.DirRef.Expression)
	})

	t.Run("p-ref validation errors are added to AST diagnostics", func(t *testing.T) {

		tree, err := Parse(context.Background(), `<input p-ref="123invalid">`, "", nil)
		require.NoError(t, err)
		require.Len(t, tree.Diagnostics, 1)
		assert.Contains(t, tree.Diagnostics[0].Message, "valid JavaScript identifier")
		assert.Equal(t, Error, tree.Diagnostics[0].Severity)

		tree2, err := Parse(context.Background(), `<input p-ref="">`, "", nil)
		require.NoError(t, err)
		require.Len(t, tree2.Diagnostics, 1)
		assert.Contains(t, tree2.Diagnostics[0].Message, "cannot be empty")

		tree3, err := Parse(context.Background(), `<input p-ref="_private">`, "", nil)
		require.NoError(t, err)
		require.Empty(t, tree3.Diagnostics)

		tree4, err := Parse(context.Background(), `<input p-ref="$special">`, "", nil)
		require.NoError(t, err)
		require.Empty(t, tree4.Diagnostics)

		tree5, err := Parse(context.Background(), `<input p-ref="my-ref">`, "", nil)
		require.NoError(t, err)
		require.Len(t, tree5.Diagnostics, 1)
		assert.Contains(t, tree5.Diagnostics[0].Message, "valid JavaScript identifier")

		tree6, err := Parse(context.Background(), `<input p-ref="state.ref">`, "", nil)
		require.NoError(t, err)
		require.Len(t, tree6.Diagnostics, 1)
		assert.Contains(t, tree6.Diagnostics[0].Message, "valid JavaScript identifier")

		tree7, err := Parse(context.Background(), `<input p-ref="   ">`, "", nil)
		require.NoError(t, err)
		require.Len(t, tree7.Diagnostics, 1)
		assert.Contains(t, tree7.Diagnostics[0].Message, "cannot be empty")
	})

	t.Run("p-ref normalises whitespace", func(t *testing.T) {

		tree, err := Parse(context.Background(), `<input p-ref="  myRef  ">`, "", nil)
		require.NoError(t, err)
		require.Empty(t, tree.Diagnostics)

		node := findNodeByTagFromRoots(t, tree.RootNodes, "input")
		require.NotNil(t, node.DirRef)
		assert.Equal(t, "myRef", node.DirRef.RawExpression, "RawExpression should be trimmed")
	})

	t.Run("p-class", func(t *testing.T) {
		tree := mustParse(t, `<div p-class="myClasses"></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirClass)
		assertExprString(t, "myClasses", node.DirClass.Expression)
	})

	t.Run("p-style", func(t *testing.T) {
		tree := mustParse(t, `<div p-style="myStyles"></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirStyle)
		assertExprString(t, "myStyles", node.DirStyle.Expression)
	})

	t.Run("p-text", func(t *testing.T) {
		tree := mustParse(t, `<span p-text="user.name"></span>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "span")

		require.NotNil(t, node.DirText)
		assertExprString(t, "user.name", node.DirText.Expression)
	})

	t.Run("p-html", func(t *testing.T) {
		tree := mustParse(t, `<div p-html="rawContent"></div>`)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

		require.NotNil(t, node.DirHTML)
		assertExprString(t, "rawContent", node.DirHTML.Expression)
	})
}

func TestParse_EventAndBindDirectives(t *testing.T) {
	t.Run("p-on simple", func(t *testing.T) {
		source := `<button p-on:click="handleClick"></button>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "button")

		require.Contains(t, node.OnEvents, "click")
		clickHandlers := node.OnEvents["click"]
		require.Len(t, clickHandlers, 1)

		directive := clickHandlers[0]
		assert.Equal(t, "click", directive.Arg)
		assert.Empty(t, directive.Modifier)
		assert.Equal(t, "handleClick", directive.RawExpression)
		require.NotNil(t, directive.Expression)
		assertExprString(t, "handleClick", directive.Expression)
	})

	t.Run("p-on with modifier", func(t *testing.T) {
		source := `<form p-on:submit.prevent="submitForm"></form>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "form")

		require.Contains(t, node.OnEvents, "submit")
		submitHandlers := node.OnEvents["submit"]
		require.Len(t, submitHandlers, 1)

		directive := submitHandlers[0]
		assert.Equal(t, "submit", directive.Arg)
		assert.Equal(t, []string{"prevent"}, directive.EventModifiers)
		assert.Empty(t, directive.Modifier)
		require.NotNil(t, directive.Expression)
		assertExprString(t, "submitForm", directive.Expression)
	})

	t.Run("p-event with modifier", func(t *testing.T) {
		source := `<custom-comp p-event:update.once="handleUpdate"></custom-comp>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "custom-comp")

		require.Contains(t, node.CustomEvents, "update")
		updateHandlers := node.CustomEvents["update"]
		require.Len(t, updateHandlers, 1)

		directive := updateHandlers[0]
		assert.Equal(t, "update", directive.Arg)
		assert.Equal(t, []string{"once"}, directive.EventModifiers)
		assert.Empty(t, directive.Modifier)
		require.NotNil(t, directive.Expression)
		assertExprString(t, "handleUpdate", directive.Expression)
	})

	t.Run("p-bind single", func(t *testing.T) {
		source := `<a p-bind:href="user.url"></a>`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "a")

		require.NotNil(t, node.Binds, "Binds map should be populated")
		require.Contains(t, node.Binds, "href")
		bindDir := node.Binds["href"]
		require.NotNil(t, bindDir)
		assert.Equal(t, "href", bindDir.Arg)
		assertExprString(t, "user.url", bindDir.Expression)
	})

	t.Run("p-bind multiple", func(t *testing.T) {
		source := `<img p-bind:src="img.src" p-bind:alt="img.alt">`
		tree := mustParse(t, source)
		node := findNodeByTagFromRoots(t, tree.RootNodes, "img")

		require.NotNil(t, node.Binds)
		require.Len(t, node.Binds, 2)

		require.Contains(t, node.Binds, "src")
		assertExprString(t, "img.src", node.Binds["src"].Expression)

		require.Contains(t, node.Binds, "alt")
		assertExprString(t, "img.alt", node.Binds["alt"].Expression)
	})
}

func TestParse_Attributes(t *testing.T) {
	source := `<div class="card" id="main" :title="pageTitle" disabled></div>`
	tree := mustParse(t, source)
	node := findNodeByTagFromRoots(t, tree.RootNodes, "div")

	require.Len(t, node.Attributes, 3)
	classAttr := getAttribute(t, node, "class")
	require.NotNil(t, classAttr)
	assert.Equal(t, "card", classAttr.Value)
	idAttr := getAttribute(t, node, "id")
	require.NotNil(t, idAttr)
	assert.Equal(t, "main", idAttr.Value)
	disabledAttr := getAttribute(t, node, "disabled")
	require.NotNil(t, disabledAttr)
	assert.Equal(t, "", disabledAttr.Value)

	require.Len(t, node.DynamicAttributes, 1)
	titleAttr := getDynamicAttribute(t, node, "title")
	require.NotNil(t, titleAttr)
	assert.Equal(t, "pageTitle", titleAttr.RawExpression)
	require.NotNil(t, titleAttr.Expression)
	assertExprString(t, "pageTitle", titleAttr.Expression)
}

func TestParse_LocationTracking(t *testing.T) {
	source := `
<div class="container">
  <p
    class="text-xl"
    p-if="user.isActive"
    :style="user.style"
  >
    Hello
  </p>
</div>
`
	tree := mustParse(t, source)

	divNode := findNodeByTagFromRoots(t, tree.RootNodes, "div")
	assert.Equal(t, 2, divNode.Location.Line, "div tag line")
	assert.Equal(t, 1, divNode.Location.Column, "div tag column")

	pNode := findNodeByTag(t, divNode, "p")
	assert.Equal(t, 3, pNode.Location.Line, "p tag line")
	assert.Equal(t, 3, pNode.Location.Column, "p tag column")

	styleAttr := getDynamicAttribute(t, pNode, "style")
	require.NotNil(t, styleAttr)
	assert.Equal(t, 6, styleAttr.Location.Line, ":style line")
	assert.Equal(t, 13, styleAttr.Location.Column, ":style value column (start of quote)")

	require.NotNil(t, pNode.DirIf, "p-if directive should populate DirIf field after transformation")
	assert.Equal(t, 5, pNode.DirIf.Location.Line, "p-if line")
	assert.Equal(t, 11, pNode.DirIf.Location.Column, "p-if value column")
	assertExprString(t, "user.isActive", pNode.DirIf.Expression)

	assert.Empty(t, pNode.Directives, "p-if directive should not be in the raw Directives slice after transformation")
}

func TestParse_TextInterpolation(t *testing.T) {
	t.Run("simple static text node", func(t *testing.T) {
		source := `<div>Hello World</div>`
		tree := mustParse(t, source)
		div := findNodeByTagFromRoots(t, tree.RootNodes, "div")
		require.Len(t, div.Children, 1)

		textNode := div.Children[0]
		assert.Equal(t, NodeText, textNode.NodeType)
		assert.Equal(t, "Hello World", textNode.TextContent, "Static text should use TextContent field")
		assert.Nil(t, textNode.RichText, "RichText should be nil for static nodes")
	})

	t.Run("simple interpolation", func(t *testing.T) {
		source := `<div>Hello, {{ user.name }}!</div>`
		tree := mustParse(t, source)
		div := findNodeByTagFromRoots(t, tree.RootNodes, "div")
		require.Len(t, div.Children, 1)

		textNode := div.Children[0]
		assert.Equal(t, NodeText, textNode.NodeType)
		assert.Empty(t, textNode.TextContent, "TextContent should be empty for dynamic nodes")
		require.NotNil(t, textNode.RichText, "RichText should be populated")
		require.Len(t, textNode.RichText, 3, "Should have three parts: literal, expression, literal")

		assert.True(t, textNode.RichText[0].IsLiteral)
		assert.Equal(t, "Hello, ", textNode.RichText[0].Literal)

		assert.False(t, textNode.RichText[1].IsLiteral)
		require.NotNil(t, textNode.RichText[1].Expression)
		assertExprString(t, "user.name", textNode.RichText[1].Expression)

		assert.True(t, textNode.RichText[2].IsLiteral)
		assert.Equal(t, "!", textNode.RichText[2].Literal)
	})

	t.Run("multiple interpolations", func(t *testing.T) {
		source := `<p>User: {{ user.name }}, Age: {{ user.age + 1 }}</p>`
		tree := mustParse(t, source)
		p := findNodeByTagFromRoots(t, tree.RootNodes, "p")
		require.Len(t, p.Children, 1)

		textNode := p.Children[0]
		require.Len(t, textNode.RichText, 4)

		assert.Equal(t, "User: ", textNode.RichText[0].Literal)
		assertExprString(t, "user.name", textNode.RichText[1].Expression)
		assert.Equal(t, ", Age: ", textNode.RichText[2].Literal)
		assertExprString(t, "(user.age + 1)", textNode.RichText[3].Expression)
	})

	t.Run("interpolation at start and end", func(t *testing.T) {
		source := `{{ greeting }} World {{ punctuation }}`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		textNode := tree.RootNodes[0]
		require.Len(t, textNode.RichText, 3)

		assertExprString(t, "greeting", textNode.RichText[0].Expression)
		assert.Equal(t, " World ", textNode.RichText[1].Literal)
		assertExprString(t, "punctuation", textNode.RichText[2].Expression)
	})

	t.Run("adjacent interpolations", func(t *testing.T) {
		source := `{{ firstName }}{{ lastName }}`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		textNode := tree.RootNodes[0]
		require.Len(t, textNode.RichText, 2)

		assertExprString(t, "firstName", textNode.RichText[0].Expression)
		assertExprString(t, "lastName", textNode.RichText[1].Expression)
	})

	t.Run("empty interpolation is ignored", func(t *testing.T) {
		source := `Hello{{}}World`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		textNode := tree.RootNodes[0]
		require.NotNil(t, textNode.RichText)
		require.Len(t, textNode.RichText, 2)
		assert.Equal(t, "Hello", textNode.RichText[0].Literal)
		assert.Equal(t, "World", textNode.RichText[1].Literal)
	})

	t.Run("interpolation with only whitespace is ignored", func(t *testing.T) {
		source := `Hello{{  }}World`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		textNode := tree.RootNodes[0]
		require.NotNil(t, textNode.RichText)
		require.Len(t, textNode.RichText, 2)
		assert.Equal(t, "Hello", textNode.RichText[0].Literal)
		assert.Equal(t, "World", textNode.RichText[1].Literal)
	})

	t.Run("error: unterminated interpolation", func(t *testing.T) {
		source := `<div>Hello {{ user.name</div>`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Unterminated text interpolation")
		div := findNodeByTagFromRoots(t, tree.RootNodes, "div")
		require.Len(t, div.Children, 1)
		textNode := div.Children[0]

		require.Len(t, textNode.RichText, 2)
		assert.Equal(t, "Hello ", textNode.RichText[0].Literal)
		assert.True(t, textNode.RichText[1].IsLiteral, "Fallback should be a literal part")
		assert.Equal(t, "{{ user.name", textNode.RichText[1].Literal)
	})

	t.Run("error: syntax error inside interpolation with correct location", func(t *testing.T) {
		source := `
		<p>
			Count: {{ 1 + }}
		</p>
		`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected expression on the right side of the operator")
		require.Len(t, tree.Diagnostics, 1)
		diagnostic := tree.Diagnostics[0]

		assert.Equal(t, 3, diagnostic.Location.Line, "Diagnostic line number is incorrect")
		assert.Equal(t, 16, diagnostic.Location.Column, "Diagnostic column number is incorrect")
	})

	t.Run("error: syntax error inside multi-line interpolation", func(t *testing.T) {
		source := `
		<p>
			Result: {{
				user.name +
				user.
			}}
		</p>
		`
		tree, err := ParseAndTransform(context.Background(), source, "test")
		require.NoError(t, err)

		assertHasError(t, tree.Diagnostics, "Expected identifier after '.'")
		require.Len(t, tree.Diagnostics, 1)
		diagnostic := tree.Diagnostics[0]

		assert.Equal(t, 5, diagnostic.Location.Line, "Diagnostic line number is incorrect for multi-line expression")
		assert.Equal(t, 9, diagnostic.Location.Column, "Diagnostic column number is incorrect for multi-line expression")
	})
}

func TestParse_WhitespaceHandling(t *testing.T) {
	t.Run("preserves single space between elements but discards newlines", func(t *testing.T) {
		source := `
<div>
    <span>First</span> <span>Second</span>
    <p>A</p>
    <p>B</p>
    <strong>One</strong><strong>Two</strong>
</div>
`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		div := findNodeByTagFromRoots(t, tree.RootNodes, "div")
		require.NotNil(t, div)

		require.Len(t, div.Children, 7, "Expected 7 children: span, text, span, p, p, strong, strong")

		assert.Equal(t, NodeElement, div.Children[0].NodeType)
		assert.Equal(t, "span", div.Children[0].TagName)

		assert.Equal(t, NodeText, div.Children[1].NodeType, "The second child MUST be a text node for the space")
		assert.Equal(t, " ", div.Children[1].TextContent, "The text node should contain exactly one space")

		assert.Equal(t, NodeElement, div.Children[2].NodeType)
		assert.Equal(t, "span", div.Children[2].TagName)

		assert.Equal(t, NodeElement, div.Children[3].NodeType)
		assert.Equal(t, "p", div.Children[3].TagName)

		assert.Equal(t, NodeElement, div.Children[4].NodeType)
		assert.Equal(t, "p", div.Children[4].TagName)

		assert.Equal(t, NodeElement, div.Children[5].NodeType)
		assert.Equal(t, "strong", div.Children[5].TagName)

		assert.Equal(t, NodeElement, div.Children[6].NodeType)
		assert.Equal(t, "strong", div.Children[6].TagName)
	})
}

func TestParse_CharactersAndEntities(t *testing.T) {
	source := `
		<div
			title="Header with © symbol and &copy; entity"
			data-emoji="🚀"
			data-numeric-entity="&#169;"
			data-hex-entity="&#xA9;"
		>
			<!-- Comment with © symbol and &copy; entity -->
			<p>Text with © symbol</p>
			<span>Text with 🚀 emoji</span>
			<b>Text with &copy; entity</b>
			<i>Combined &amp; more: &lt; &gt; &quot; &apos;</i>
		</div>
	`

	tree := mustParse(t, source)

	require.Len(t, tree.RootNodes, 1, "Should be one root div element")
	divNode := tree.RootNodes[0]
	require.Equal(t, "div", divNode.TagName)

	titleAttr := getAttribute(t, divNode, "title")
	require.NotNil(t, titleAttr)
	assert.Equal(t, "Header with © symbol and &copy; entity", titleAttr.Value)

	emojiAttr := getAttribute(t, divNode, "data-emoji")
	require.NotNil(t, emojiAttr)
	assert.Equal(t, "🚀", emojiAttr.Value)

	numericAttr := getAttribute(t, divNode, "data-numeric-entity")
	require.NotNil(t, numericAttr)
	assert.Equal(t, "&#169;", numericAttr.Value, "Numeric entity should be preserved in attributes")

	hexAttr := getAttribute(t, divNode, "data-hex-entity")
	require.NotNil(t, hexAttr)
	assert.Equal(t, "&#xA9;", hexAttr.Value, "Hexadecimal entity should be preserved in attributes")

	var contentNodes []*TemplateNode
	for _, child := range divNode.Children {
		if child.NodeType != NodeText || strings.TrimSpace(child.TextContent) != "" {
			contentNodes = append(contentNodes, child)
		}
	}
	require.Len(t, contentNodes, 5, "Div should have a comment and four element nodes")

	commentNode := contentNodes[0]
	assert.Equal(t, NodeComment, commentNode.NodeType)
	assert.Equal(t, " Comment with © symbol and &copy; entity ", commentNode.TextContent)

	pNode := contentNodes[1]
	require.Equal(t, "p", pNode.TagName)
	require.Len(t, pNode.Children, 1)
	pTextNode := pNode.Children[0]
	assert.Equal(t, "Text with © symbol", pTextNode.TextContent, "UTF-8 in text node should be preserved")

	spanNode := contentNodes[2]
	require.Equal(t, "span", spanNode.TagName)
	require.Len(t, spanNode.Children, 1)
	spanTextNode := spanNode.Children[0]
	assert.Equal(t, "Text with 🚀 emoji", spanTextNode.TextContent, "Emoji in text node should be preserved")

	bNode := contentNodes[3]
	require.Equal(t, "b", bNode.TagName)
	require.Len(t, bNode.Children, 1)
	bTextNode := bNode.Children[0]
	assert.Equal(t, "Text with © entity", bTextNode.TextContent, "Named entity in text should be DECODED")

	iNode := contentNodes[4]
	require.Equal(t, "i", iNode.TagName)
	require.Len(t, iNode.Children, 1)
	iTextNode := iNode.Children[0]
	assert.Equal(t, "Combined & more: < > \" '", iTextNode.TextContent, "Common entities in text should be DECODED")
}

func TestParse_UTF8AndEntitiesEverywhere(t *testing.T) {
	source := `
		<!-- Comment with © and 🚀 -->
		<div
			static-attr="© &copy; 🚀"
			:dynamic-attr="'© &copy; 🚀'"
			p-if="status == 'active-©-🚀'"
		>
			Plain text with © and 🚀.
			Interpolation: {{ "©" + " &amp; " + "🚀" }}
		</div>
	`
	tree := mustParse(t, source)
	assertNoError(t, tree.Diagnostics, "Should parse without errors")

	var contentNodes []*TemplateNode
	for _, node := range tree.RootNodes {
		if node.NodeType != NodeText || strings.TrimSpace(node.TextContent) != "" {
			contentNodes = append(contentNodes, node)
		}
	}
	require.Len(t, contentNodes, 2, "Should have two root nodes: Comment and Div")

	commentNode := contentNodes[0]
	assert.Equal(t, NodeComment, commentNode.NodeType)
	assert.Equal(t, " Comment with © and 🚀 ", commentNode.TextContent)

	divNode := contentNodes[1]
	require.Equal(t, "div", divNode.TagName)

	staticAttr := getAttribute(t, divNode, "static-attr")
	require.NotNil(t, staticAttr)
	assert.Equal(t, "© &copy; 🚀", staticAttr.Value, "Static attribute value should be RAW")

	dynAttr := getDynamicAttribute(t, divNode, "dynamic-attr")
	require.NotNil(t, dynAttr)
	assert.Equal(t, "'© &copy; 🚀'", dynAttr.RawExpression)

	require.NotNil(t, dynAttr.Expression)
	require.IsType(t, &StringLiteral{}, dynAttr.Expression)
	assert.Equal(t, "© © 🚀", dynAttr.Expression.(*StringLiteral).Value)

	require.NotNil(t, divNode.DirIf)
	assert.Equal(t, "status == 'active-©-🚀'", divNode.DirIf.RawExpression)
	require.NotNil(t, divNode.DirIf.Expression)
	assert.Equal(t, "(status == \"active-©-🚀\")", divNode.DirIf.Expression.String())

	var textNodes []*TemplateNode
	for _, node := range divNode.Children {
		if node.NodeType == NodeText && (node.RichText != nil || strings.TrimSpace(node.TextContent) != "") {
			textNodes = append(textNodes, node)
		}
	}
	require.Len(t, textNodes, 1, "Div should contain ONE text node for all its mixed content")

	theOnlyTextNode := textNodes[0]
	assert.Equal(t, NodeText, theOnlyTextNode.NodeType)
	assert.Empty(t, theOnlyTextNode.TextContent, "The single text node is a RichText node, so its TextContent should be empty")
	require.NotNil(t, theOnlyTextNode.RichText, "The text node must contain RichText parts")

	require.Len(t, theOnlyTextNode.RichText, 3, "The RichText slice should have three parts")

	assert.True(t, theOnlyTextNode.RichText[0].IsLiteral)
	expectedTextBefore := "Plain text with © and 🚀. Interpolation:"
	actualTextBefore := strings.Join(strings.Fields(theOnlyTextNode.RichText[0].Literal), " ")
	assert.Equal(t, expectedTextBefore, actualTextBefore)

	assert.False(t, theOnlyTextNode.RichText[1].IsLiteral)
	assert.Equal(t, `("©" + (" & " + "🚀"))`, theOnlyTextNode.RichText[1].Expression.String())

	assert.True(t, theOnlyTextNode.RichText[2].IsLiteral)
}

func TestParse_UnicodeAndEntities(t *testing.T) {
	t.Run("handles Arabic text and attributes", func(t *testing.T) {
		source := `<div عنوان="ترحيب"><p>مرحباً بالعالم</p></div>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		div := tree.RootNodes[0]
		assert.Equal(t, "div", div.TagName)
		attr := getAttribute(t, div, "عنوان")
		assert.Equal(t, "ترحيب", attr.Value)

		p := findNodeByTag(t, div, "p")
		require.Len(t, p.Children, 1)
		textNode := p.Children[0]
		assert.Equal(t, "مرحباً بالعالم", textNode.TextContent)
	})

	t.Run("handles Unicode in dynamic attributes and interpolations", func(t *testing.T) {
		source := `<div :aria-label="صفحة.عنوان">اسم المستخدم: {{ مستخدم.اسم }}</div>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		div := tree.RootNodes[0]
		dynAttr := getDynamicAttribute(t, div, "aria-label")
		assert.Equal(t, "صفحة.عنوان", dynAttr.RawExpression)
		assertExprString(t, "صفحة.عنوان", dynAttr.Expression)

		require.Len(t, div.Children, 1)
		textNode := div.Children[0]
		require.NotNil(t, textNode.RichText)
		require.Len(t, textNode.RichText, 2)
		assert.Equal(t, "اسم المستخدم: ", textNode.RichText[0].Literal)
		assertExprString(t, "مستخدم.اسم", textNode.RichText[1].Expression)
	})

	t.Run("handles right-to-left (RTL) directionality attribute", func(t *testing.T) {
		source := `<html lang="ar" dir="rtl"><body><p>هذا اختبار.</p></body></html>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		p := tree.RootNodes[0]
		assert.Equal(t, "p", p.TagName)
		require.Len(t, p.Children, 1)
		assert.Equal(t, "هذا اختبار.", p.Children[0].TextContent)
	})

	t.Run("handles mixed LTR and RTL content", func(t *testing.T) {
		source := `<p>The arabic word for "book" is <bdo dir="rtl">كتاب</bdo>.</p>`
		tree := mustParse(t, source)
		require.Len(t, tree.RootNodes, 1)

		p := tree.RootNodes[0]
		require.Len(t, p.Children, 3)

		assert.Equal(t, "The arabic word for \"book\" is ", p.Children[0].TextContent)
		bdo := p.Children[1]
		assert.Equal(t, "bdo", bdo.TagName)
		dirAttr := getAttribute(t, bdo, "dir")
		assert.Equal(t, "rtl", dirAttr.Value)
		require.Len(t, bdo.Children, 1)
		assert.Equal(t, "كتاب", bdo.Children[0].TextContent)
		assert.Equal(t, ".", p.Children[2].TextContent)
	})
}

func TestParse_ZalgoText(t *testing.T) {
	zalgoVar := "u̸s̸e̵r̸"
	zalgoText := "ţ̸̘͔̩̼̳̪̙͕̆́̾̈̓̔̓̿̽̉̈́́̚h̴͉̼̩̳͙̺̺̙̘̐̃͛͐͑̇͛̏͐̾̂̀͂̃̍͝͠í̵̧̖̺̣̱̙̉̐̄̓̽̔̇̎̀́̌̍͊s̵͕̼͖̙͌̆̀͂̑̄̀̅̎͒̔̚̚͠͠ ̶̡̹̯̼̞̑̾̏͊͐̓̽͒̊͆̆̎̐̎͠ï̷̢̩̱̦͓̳̩͖͍͙͖̥̩͙̔͂͐̚͘s̸̥̗̣͕̀̀̓͌̀̈̍̆̂̽̍͛̚͝ ̸̢̙̻̯̳̼͚̖̫͍̳̩́̑̃̈̽̒̚͜͠z̷̛͙̯͚̟̞̺̙͎͇̼̃̓͊͊͌ả̵̧̨͚̬͓̾̊̌̌̓̆̆͝l̵̥̮̞̩͐͌͆̂͋̐͂͂̆̉͋͝͠g̶͎͚̳̲̣̖̻͚̭̥̤̪͌̈́o̶̧̜̟̻̟̝̹̪̱͇̮͎̞̣̹̩͈͌́̏̀͂̌̇̍̏̌̕͠"
	source := fmt.Sprintf(`<div title="%s" p-if="%s.isActive">{{ %s.name }}</div>`, zalgoText, zalgoVar, zalgoVar)

	tree := mustParse(t, source)
	require.Len(t, tree.RootNodes, 1)

	div := tree.RootNodes[0]
	assert.Equal(t, "div", div.TagName)

	titleAttr := getAttribute(t, div, "title")
	require.NotNil(t, titleAttr)
	assert.Equal(t, zalgoText, titleAttr.Value)

	require.NotNil(t, div.DirIf)
	assert.Equal(t, fmt.Sprintf("%s.isActive", zalgoVar), div.DirIf.Expression.String())

	require.Len(t, div.Children, 1)
	textNode := div.Children[0]
	require.NotNil(t, textNode.RichText)
	require.Len(t, textNode.RichText, 1)
	assert.False(t, textNode.RichText[0].IsLiteral)
	assert.Equal(t, fmt.Sprintf("%s.name", zalgoVar), textNode.RichText[0].Expression.String())
}

func TestHandleRawText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		source    string
		parentTag string
		wantText  string
	}{
		{
			name:      "textarea preserves raw content",
			source:    `<textarea>raw content here</textarea>`,
			parentTag: "textarea",
			wantText:  "raw content here",
		},
		{
			name:      "pre preserves raw content with whitespace",
			source:    `<pre>  indented  text  </pre>`,
			parentTag: "pre",
			wantText:  "  indented  text  ",
		},
		{
			name:      "code preserves raw content",
			source:    `<code>func main() {}</code>`,
			parentTag: "code",
			wantText:  "func main() {}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tree := mustParse(t, tc.source)

			require.Len(t, tree.RootNodes, 1)
			parent := tree.RootNodes[0]
			assert.Equal(t, tc.parentTag, parent.TagName)

			require.Len(t, parent.Children, 1, "Raw text parent should have one text child")
			textChild := parent.Children[0]
			assert.Equal(t, NodeText, textChild.NodeType, "Child should be a text node")
			assert.Equal(t, tc.wantText, textChild.TextContent, "Text content should match raw input")
			assert.True(t, textChild.PreserveWhitespace, "Raw text child should have PreserveWhitespace set")
		})
	}
}

func TestHandleFlagDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawExpr  string
		wantExpr string
	}{
		{
			name:     "empty value normalises to true",
			rawExpr:  "",
			wantExpr: "true",
		},
		{
			name:     "true value stays true",
			rawExpr:  "true",
			wantExpr: "true",
		},
		{
			name:     "TRUE value normalises to true",
			rawExpr:  "TRUE",
			wantExpr: "true",
		},
		{
			name:     "false value stays false",
			rawExpr:  "false",
			wantExpr: "false",
		},
		{
			name:     "FALSE value normalises to false",
			rawExpr:  "FALSE",
			wantExpr: "false",
		},
		{
			name:     "dynamic expression left unchanged",
			rawExpr:  "someCondition",
			wantExpr: "someCondition",
		},
		{
			name:     "whitespace around true is trimmed",
			rawExpr:  "  true  ",
			wantExpr: "true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &Directive{
				Type:          DirectiveScaffold,
				RawExpression: tc.rawExpr,
			}
			handleFlagDirective(d)
			assert.Equal(t, tc.wantExpr, d.RawExpression)
		})
	}
}

func TestParse_SVGForeignToken(t *testing.T) {
	t.Run("basic svg with children", func(t *testing.T) {
		source := `<div><svg viewBox="0 0 48 48"><circle cx="24" cy="24" r="20"></circle></svg></div>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		divNode := tree.RootNodes[0]
		assert.Equal(t, "div", divNode.TagName)

		require.Len(t, divNode.Children, 1)
		svgNode := divNode.Children[0]
		assert.Equal(t, NodeElement, svgNode.NodeType)
		assert.Equal(t, "svg", svgNode.TagName)

		viewBoxAttr := getAttribute(t, svgNode, "viewBox")
		assert.Equal(t, "0 0 48 48", viewBoxAttr.Value)

		require.Len(t, svgNode.Children, 1)
		circleNode := svgNode.Children[0]
		assert.Equal(t, NodeElement, circleNode.NodeType)
		assert.Equal(t, "circle", circleNode.TagName)
		assert.Equal(t, "24", getAttribute(t, circleNode, "cx").Value)
		assert.Equal(t, "24", getAttribute(t, circleNode, "cy").Value)
		assert.Equal(t, "20", getAttribute(t, circleNode, "r").Value)
	})

	t.Run("svg preserves case-sensitive attributes", func(t *testing.T) {
		source := `<svg viewBox="0 0 24 24"><path d="M10 20v-6" stroke-linecap="round"></path></svg>`
		tree := mustParse(t, source)

		require.Len(t, tree.RootNodes, 1)
		svgNode := tree.RootNodes[0]
		assert.Equal(t, "svg", svgNode.TagName)

		viewBoxAttr := getAttribute(t, svgNode, "viewBox")
		assert.Equal(t, "0 0 24 24", viewBoxAttr.Value, "viewBox case should be preserved")

		require.Len(t, svgNode.Children, 1)
		pathNode := svgNode.Children[0]
		assert.Equal(t, "path", pathNode.TagName)
		assert.Equal(t, "round", getAttribute(t, pathNode, "stroke-linecap").Value)
	})

	t.Run("svg lowercases non-spec attributes but preserves spec ones", func(t *testing.T) {
		source := `<svg viewBox="0 0 100 100" preserveAspectRatio="xMidYMid" class="icon">` +
			`<linearGradient gradientUnits="userSpaceOnUse" gradientTransform="rotate(45)" spreadMethod="pad"></linearGradient>` +
			`<circle pathLength="100" fill="red"></circle>` +
			`<text textLength="60" lengthAdjust="spacing"></text>` +
			`</svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]

		assert.NotNil(t, getAttribute(t, svgNode, "viewBox"), "viewBox must keep capital B")
		assert.NotNil(t, getAttribute(t, svgNode, "preserveAspectRatio"), "preserveAspectRatio must keep case")

		assert.NotNil(t, getAttribute(t, svgNode, "class"), "class must be lowercased")

		gradientNode := svgNode.Children[0]
		assert.NotNil(t, getAttribute(t, gradientNode, "gradientUnits"))
		assert.NotNil(t, getAttribute(t, gradientNode, "gradientTransform"))
		assert.NotNil(t, getAttribute(t, gradientNode, "spreadMethod"))

		circleNode := svgNode.Children[1]
		assert.NotNil(t, getAttribute(t, circleNode, "pathLength"))
		assert.NotNil(t, getAttribute(t, circleNode, "fill"), "fill must be lowercased")

		textNode := svgNode.Children[2]
		assert.NotNil(t, getAttribute(t, textNode, "textLength"))
		assert.NotNil(t, getAttribute(t, textNode, "lengthAdjust"))
	})

	t.Run("svg with nested groups", func(t *testing.T) {
		source := `<svg><g transform="translate(0,0)"><rect x="0" y="0" width="10" height="10"></rect></g></svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]
		assert.Equal(t, "svg", svgNode.TagName)

		require.Len(t, svgNode.Children, 1)
		gNode := svgNode.Children[0]
		assert.Equal(t, "g", gNode.TagName)
		assert.Equal(t, "translate(0,0)", getAttribute(t, gNode, "transform").Value)

		require.Len(t, gNode.Children, 1)
		rectNode := gNode.Children[0]
		assert.Equal(t, "rect", rectNode.TagName)
	})

	t.Run("svg with self-closing elements", func(t *testing.T) {
		source := `<svg viewBox="0 0 48 48"><circle cx="24" cy="24" r="20" fill="none" stroke-width="4"/><circle cx="24" cy="24" r="20" fill="none"/></svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]
		require.Len(t, svgNode.Children, 2, "should have two self-closing circle children")
		assert.Equal(t, "circle", svgNode.Children[0].TagName)
		assert.Equal(t, "circle", svgNode.Children[1].TagName)
	})

	t.Run("svg with text content", func(t *testing.T) {
		source := `<svg><text x="10" y="20">Hello SVG</text></svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]
		require.Len(t, svgNode.Children, 1)
		textEl := svgNode.Children[0]
		assert.Equal(t, "text", textEl.TagName)

		require.Len(t, textEl.Children, 1)
		textNode := textEl.Children[0]
		assert.Equal(t, NodeText, textNode.NodeType)
		assert.Equal(t, "Hello SVG", textNode.TextContent)
	})

	t.Run("svg with piko directives", func(t *testing.T) {
		source := `<svg><circle p-if="state.show" cx="12" cy="12" r="10"></circle></svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]
		require.Len(t, svgNode.Children, 1)
		circleNode := svgNode.Children[0]
		assert.Equal(t, "circle", circleNode.TagName)

		require.NotNil(t, circleNode.DirIf, "p-if directive should be parsed on SVG child")
	})

	t.Run("svg with dynamic attributes", func(t *testing.T) {
		source := `<svg :class="state.svgClass"><circle :r="state.radius" cx="24" cy="24"></circle></svg>`
		tree := mustParse(t, source)

		svgNode := tree.RootNodes[0]
		require.Len(t, svgNode.DynamicAttributes, 1)
		assert.Equal(t, "class", svgNode.DynamicAttributes[0].Name)
		assert.Equal(t, "state.svgClass", svgNode.DynamicAttributes[0].RawExpression)

		circleNode := svgNode.Children[0]
		require.Len(t, circleNode.DynamicAttributes, 1)
		assert.Equal(t, "r", circleNode.DynamicAttributes[0].Name)
		assert.Equal(t, "state.radius", circleNode.DynamicAttributes[0].RawExpression)
	})

	t.Run("svg siblings with html elements", func(t *testing.T) {
		source := `<div><span>Label</span><svg viewBox="0 0 24 24"><path d="M0 0"/></svg><span>After</span></div>`
		tree := mustParse(t, source)

		divNode := tree.RootNodes[0]
		require.Len(t, divNode.Children, 3)
		assert.Equal(t, "span", divNode.Children[0].TagName)
		assert.Equal(t, "svg", divNode.Children[1].TagName)
		assert.Equal(t, "span", divNode.Children[2].TagName)

		svgNode := divNode.Children[1]
		require.Len(t, svgNode.Children, 1)
		assert.Equal(t, "path", svgNode.Children[0].TagName)
	})

	t.Run("multiple svgs in same template", func(t *testing.T) {
		source := `<div><svg viewBox="0 0 24 24"><circle r="10"/></svg><svg viewBox="0 0 48 48"><rect width="48" height="48"/></svg></div>`
		tree := mustParse(t, source)

		divNode := tree.RootNodes[0]
		require.Len(t, divNode.Children, 2)

		svg1 := divNode.Children[0]
		assert.Equal(t, "svg", svg1.TagName)
		require.Len(t, svg1.Children, 1)
		assert.Equal(t, "circle", svg1.Children[0].TagName)

		svg2 := divNode.Children[1]
		assert.Equal(t, "svg", svg2.TagName)
		require.Len(t, svg2.Children, 1)
		assert.Equal(t, "rect", svg2.Children[0].TagName)
	})
}
