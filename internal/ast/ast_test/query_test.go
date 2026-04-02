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

package ast_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func newTestAST(t *testing.T, html string) *ast_domain.TemplateAST {
	t.Helper()
	templateAST, err := ast_domain.ParseAndTransform(context.Background(), html, "test.html")
	require.NoError(t, err)
	return templateAST
}

func hasClass(node *ast_domain.TemplateNode, className string) bool {
	for _, attr := range node.Attributes {
		if attr.Name == "class" {
			for class := range strings.FieldsSeq(attr.Value) {
				if class == className {
					return true
				}
			}
			return false
		}
	}
	return false
}

func assertHasAttribute(t *testing.T, node *ast_domain.TemplateNode, name, value string) {
	t.Helper()
	found := false
	for _, attr := range node.Attributes {
		if attr.Name == name {
			assert.Equal(t, value, attr.Value)
			found = true
			break
		}
	}
	require.True(t, found, "Expected node <%s> to have attribute %s=\"%s\"", node.TagName, name, value)
}

func TestASTQueryEngine(t *testing.T) {
	html := `
		<main id="app" data-root lang="en-US">
			<div class="card active" data-id="1">
				<h1>Title</h1>
				<p>Paragraph 1</p>
			</div>
			<div class="card" lang="en">
				<p>Paragraph 2</p>
				<div class="nested">
					<p class="special">Paragraph 3</p>
				</div>
			</div>
			<p class="footer">Footer</p>
			<Fragment>
				<div class="card from-fragment">Final Card</div>
			</Fragment>
		</main>
	`
	templateAST := newTestAST(t, html)

	t.Run("Query single node", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "h1")
		require.Empty(t, diagnostics, "Valid query should produce no diagnostics")
		require.NotNil(t, node)
		assert.Equal(t, "h1", node.TagName)
	})

	t.Run("Query by class", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".card", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("Query by tag and class", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "div.active")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "h1", node.FirstElementChild().TagName)
	})

	t.Run("Query by ID", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#app")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "main", node.TagName)
	})

	t.Run("Universal Selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "*", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 10)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, "*.card", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("Descendant selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".card p", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("Child selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".card > p", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "Paragraph 1", nodes[0].Text(context.Background()))
		assert.Equal(t, "Paragraph 2", nodes[1].Text(context.Background()))
	})

	t.Run("Complex selector", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "main#app > div.active p")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Paragraph 1", node.Text(context.Background()))
	})

	t.Run("Fragment traversal", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "main .from-fragment")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "div", node.TagName)

		node, diagnostics = ast_domain.Query(templateAST, "main > .from-fragment")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "div", node.TagName)
	})

	t.Run("No match returns empty slice or nil", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "section.non-existent")
		require.Empty(t, diagnostics)
		assert.Nil(t, node)

		nodes, diagnostics := ast_domain.QueryAll(templateAST, "section.non-existent", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)
	})

	t.Run("Invalid selector returns diagnostics", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "div[type='text'")
		assert.Nil(t, node)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "expected ']' to close attribute selector")
	})

	t.Run("Attribute Selectors", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "[data-root]", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assert.Equal(t, "main", nodes[0].TagName)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[data-id="1"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[class~="active"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assertHasAttribute(t, nodes[0], "data-id", "1")

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[lang|="en"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[class^="foot"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assert.Equal(t, "p", nodes[0].TagName)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[class$="fragment"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assert.Equal(t, "div", nodes[0].TagName)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[class*="from-frag"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
	})
}

func TestASTQueryEngine_ExtendedFeatures(t *testing.T) {
	html := `
		<div id="container">
			<h1 class="heading">Main Title</h1>
			<!-- comment -->
			<p id="p1" data-type="INTRO">First paragraph.</p>
			<div class="content" lang="en-gb">
				<p id="p2" data-type="body">Second paragraph.</p>
			</div>
			<h2 class="heading">Subtitle</h2>
			<p id="p3" data-type="body">Third paragraph.</p>
			<p id="p4" data-type="conclusion">Fourth paragraph.</p>
			<span>A span</span>
			<p id="p5" data-type="footer">Fifth paragraph.</p>
		</div>
	`
	templateAST := newTestAST(t, html)

	t.Run("Adjacent Sibling Combinator (+)", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "h1 + p")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p1")

		node, diagnostics = ast_domain.Query(templateAST, "h1 + h2")
		require.Empty(t, diagnostics)
		assert.Nil(t, node)

		node, diagnostics = ast_domain.Query(templateAST, "h2 + p + p")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p4")

		node, diagnostics = ast_domain.Query(templateAST, ".content + h2")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Subtitle", node.Text(context.Background()))
	})

	t.Run("General Sibling Combinator (~)", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "h2 ~ p", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "p3")
		assertHasAttribute(t, nodes[1], "id", "p4")
		assertHasAttribute(t, nodes[2], "id", "p5")

		nodes, diagnostics = ast_domain.QueryAll(templateAST, ".content ~ *", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 5)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, "h2 ~ h1", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)
	})

	t.Run("Selector List (,)", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "h1, h2, span", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, "div.content > p, .heading ~ p, #p5", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 5)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, "p, [data-type=body]", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 5)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, "nonexistent, h1", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
	})

	t.Run("Attribute Case-Insensitivity (i)", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `[data-type="intro"]`, "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[data-type="intro" i]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assertHasAttribute(t, nodes[0], "id", "p1")

		nodes, diagnostics = ast_domain.QueryAll(templateAST, `[data-type^="con" i]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assertHasAttribute(t, nodes[0], "id", "p4")
	})

	t.Run(":not() with attribute selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `p:not([data-type="body"])`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p4")
		assertHasAttribute(t, nodes[2], "id", "p5")
	})

	t.Run(":not() with universal selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > *:not(.heading)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 6)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assert.Equal(t, "div", nodes[1].TagName)
		assertHasAttribute(t, nodes[5], "id", "p5")
	})
}

func TestASTQueryEngine_PseudoClasses(t *testing.T) {
	html := `
		<div id="container">
			<p id="p1" class="intro first">First P</p>
			<h2 id="h2-1">First H2</h2>
			<p id="p2" class="body">Second P</p>
			<p id="p3" class="body special">Third P</p>
			<h2 id="h2-2">Second H2</h2>
			<span id="s1">First Span</span>
			<p id="p4" class="outro">Fourth P</p>
			<p id="p5" class="outro last">Fifth P</p>
		</div>
	`
	templateAST := newTestAST(t, html)

	t.Run(":not() with simple class", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(.special)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p2")
		assertHasAttribute(t, nodes[2], "id", "p4")
		assertHasAttribute(t, nodes[3], "id", "p5")
	})

	t.Run(":not() with child combinator and tag", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :not(p)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assert.Equal(t, "h2", nodes[0].TagName)
		assert.Equal(t, "h2", nodes[1].TagName)
		assert.Equal(t, "span", nodes[2].TagName)
	})

	t.Run(":not() with ID selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(#p1)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
		assertHasAttribute(t, nodes[0], "id", "p2")
	})

	t.Run(":not() chained", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(.body):not(.intro)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "p4")
		assertHasAttribute(t, nodes[1], "id", "p5")
	})

	t.Run(":not() with descendant combinator is parsed correctly", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "div:not(#container) p", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 0)
	})

	t.Run(":not() with chained classes", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(.body.special)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p2")
		assertHasAttribute(t, nodes[2], "id", "p4")
		assertHasAttribute(t, nodes[3], "id", "p5")
	})

	t.Run(":not() containing :first-child", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :not(:first-child)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 7)
		assertHasAttribute(t, nodes[0], "id", "h2-1")
	})

	t.Run(":not() containing :last-of-type", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(:last-of-type)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
	})

	t.Run(":not() does not match when selector is empty", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not()", "test.selector")
		require.NotEmpty(t, diagnostics, "An empty :not() should produce a diagnostic")
		assert.Contains(t, diagnostics[0].Message, "expected a selector")
		assert.Empty(t, nodes)
	})

	t.Run(":not() with a selector that matches nothing", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:not(.non-existent-class)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 5, "Should match all 5 p tags")
	})

	t.Run(":first-child selects first element", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > :first-child")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p1")
	})

	t.Run(":last-child selects last element", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > :last-child")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p5")
	})

	t.Run(":first-child with matching tag", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > p:first-child")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p1")
	})

	t.Run(":first-child with non-matching tag", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > h2:first-child")
		require.Empty(t, diagnostics)
		assert.Nil(t, node)
	})

	t.Run(":nth-child(n) with number", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > :nth-child(2)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "h2-1")
	})

	t.Run(":nth-child(n) with 'even' keyword", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :nth-child(even)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
		assertHasAttribute(t, nodes[0], "id", "h2-1")
		assertHasAttribute(t, nodes[1], "id", "p3")
		assertHasAttribute(t, nodes[2], "id", "s1")
		assertHasAttribute(t, nodes[3], "id", "p5")
	})

	t.Run(":nth-child(n) with 'odd' keyword", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :nth-child(odd)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p2")
		assertHasAttribute(t, nodes[2], "id", "h2-2")
		assertHasAttribute(t, nodes[3], "id", "p4")
	})

	t.Run(":nth-child(n) with 'an+b' formula", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :nth-child(3n+2)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "h2-1")
		assertHasAttribute(t, nodes[1], "id", "h2-2")
		assertHasAttribute(t, nodes[2], "id", "p5")
	})

	t.Run(":nth-child(n) with '-an+b' formula", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :nth-child(-n+3)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "h2-1")
		assertHasAttribute(t, nodes[2], "id", "p2")
	})

	t.Run(":first-of-type with p", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "p:first-of-type")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "p1")
	})

	t.Run(":first-of-type with h2", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "h2:first-of-type")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "h2-1")
	})

	t.Run(":last-of-type with p", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "p:last-of-type")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "p5")
	})

	t.Run(":last-of-type with h2", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "h2:last-of-type")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "h2-2")
	})

	t.Run(":only-child selects lone child elements", func(t *testing.T) {

		html := `
			<div id="parent-single">
				<p id="lonely">Only child</p>
			</div>
			<div id="parent-multiple">
				<p id="first">First</p>
				<p id="second">Second</p>
			</div>
			<div id="parent-empty"></div>
		`
		testAST := newTestAST(t, html)

		nodes, diagnostics := ast_domain.QueryAll(testAST, "p:only-child", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1, "Should match only the lone child")
		assertHasAttribute(t, nodes[0], "id", "lonely")
	})

	t.Run(":only-child does not match when multiple children", func(t *testing.T) {
		html := `
			<div>
				<p id="first">First</p>
				<p id="second">Second</p>
			</div>
		`
		testAST := newTestAST(t, html)

		nodes, diagnostics := ast_domain.QueryAll(testAST, "p:only-child", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 0, "Should not match when multiple siblings exist")
	})

	t.Run(":only-of-type selects elements with unique tag", func(t *testing.T) {
		html := `
			<div id="mixed">
				<p id="solo-p">Only paragraph</p>
				<span id="span1">Span 1</span>
				<span id="span2">Span 2</span>
				<div id="solo-div">Only div</div>
			</div>
		`
		testAST := newTestAST(t, html)

		node, diagnostics := ast_domain.Query(testAST, "p:only-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "solo-p")

		node, diagnostics = ast_domain.Query(testAST, "#mixed > div:only-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "solo-div")
	})

	t.Run(":only-of-type does not match when multiple of same type", func(t *testing.T) {
		html := `
			<div>
				<span id="span1">Span 1</span>
				<span id="span2">Span 2</span>
			</div>
		`
		testAST := newTestAST(t, html)

		nodes, diagnostics := ast_domain.QueryAll(testAST, "span:only-of-type", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 0, "Should not match when multiple elements of same type exist")
	})

	t.Run(":only-child and :only-of-type work together", func(t *testing.T) {
		html := `
			<div id="parent1">
				<p id="both">Only child and only of type</p>
			</div>
			<div id="parent2">
				<p id="only-type">Only p, but has sibling</p>
				<span>A span</span>
			</div>
		`
		testAST := newTestAST(t, html)

		node, diagnostics := ast_domain.Query(testAST, "p:only-child:only-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "both")

		node, diagnostics = ast_domain.Query(testAST, "#parent2 > p:only-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "only-type")
	})

	t.Run(":nth-of-type(n) with number", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "p:nth-of-type(2)")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "p2")
	})

	t.Run(":nth-of-type(n) with 'odd'", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:nth-of-type(odd)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p3")
		assertHasAttribute(t, nodes[2], "id", "p5")
	})

	t.Run(":nth-of-type(n) with '2n'", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "h2:nth-of-type(2n)")
		require.Empty(t, diagnostics)
		assertHasAttribute(t, node, "id", "h2-2")
	})

	t.Run("Chained pseudo-classes :nth-of-type and :not", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "p:nth-of-type(odd):not(.special)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "p1")
		assertHasAttribute(t, nodes[1], "id", "p5")
	})

	t.Run("Chained pseudo-classes :nth-child and :not", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#container > :nth-child(even):not(h2)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assertHasAttribute(t, nodes[0], "id", "p3")
		assertHasAttribute(t, nodes[1], "id", "s1")
		assertHasAttribute(t, nodes[2], "id", "p5")
	})

	t.Run("Chained pseudo-classes :last-child and :not", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > :last-child:not(span)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p5")
	})

	t.Run("Chained pseudo-classes fail case", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#container > :last-child:not(p)")
		require.Empty(t, diagnostics)
		assert.Nil(t, node)
	})
}

func TestASTQueryEngine_ComplexDOM(t *testing.T) {

	html := `
		<div id="app" class="theme-dark" lang="en">
			<header class="main-header" role="banner">
				<h1 id="site-title">My Complex App</h1>
				<nav>
					<ul id="main-nav">
						<li><a href="/" class="nav-link active">Home</a></li>
						<li><a href="/about" class="nav-link">About</a></li>
						<li><a href="/contact" class="nav-link disabled" aria-disabled="true">Contact</a></li>
					</ul>
				</nav>
			</header>
			<main class="content-area">
				<article class="post" data-id="123" data-category="tech">
					<h2>Post Title 1</h2>
					<p>First paragraph.</p>
					<p class="standout">Second paragraph, important.</p>
					<div class="meta">
						<span>Author: Alex</span>
						<time>2024-08-20</time>
					</div>
				</article>

				<section id="comments">
					<h3>Comments</h3>
					<div class="comment-list">
						<!-- Comment 1 -->
						<div class="comment" id="comment-1">
							<p>This is the first comment.</p>
							<span>User A</span>
						</div>
						<!-- Comment 2 (nested in a fragment) -->
						<Fragment>
							<div class="comment" id="comment-2">
								<p>Second comment, with a <a href="#">link</a>.</p>
								<span>User B</span>
							</div>
						</Fragment>
						<!-- Comment 3 -->
						<div class="comment" id="comment-3">
							<p>Third comment is here.</p>
							<span>User C</span>
						</div>
					</div>
				</section>

				<article class="post" data-id="456" data-category="lifestyle">
					<h2>Post Title 2</h2>
					<p>Another first paragraph.</p>
					<!-- This is an empty P tag for testing :empty -->
					<p></p>
					<ul class="tag-list">
						<li><span>Tag 1</span></li>
						<li><span>Tag 2</span></li>
						<li><span>Tag 3</span></li>
						<li><span>Tag 4</span></li>
					</ul>
				</article>
			</main>
			<footer role="contentinfo">
				<p id="copyright">&copy; 2024 My Complex App</p>
				<p id="back-to-top"><a href="#app">Back to top</a></p>
			</footer>
		</div>
	`
	templateAST := newTestAST(t, html)

	t.Run("1. Child selector through multiple levels", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "nav > ul > li > a", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
		assert.Equal(t, "Home", nodes[0].Text(context.Background()))
	})

	t.Run("2. Descendant selector skipping levels", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "header nav a", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("3. General sibling selector for all subsequent posts", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "article[data-id='123'] ~ article", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assertHasAttribute(t, nodes[0], "data-id", "456")
	})

	t.Run("4. Adjacent sibling selector for meta after p", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "p.standout + .meta")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "div", node.TagName)
	})

	t.Run("5. Grouping selector for all titles", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "h1, h2, h3", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
	})

	t.Run("6. Attribute selector with exact match", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `[data-category="tech"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "data-id", "123")
	})

	t.Run("7. Attribute selector with prefix match", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `[id^="comment-"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("8. Attribute selector with substring match", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `[class*="nav-"]`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 3)
	})

	t.Run("9. Chained attribute selectors", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `a[href="/contact"][aria-disabled="true"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Contact", node.Text(context.Background()))
	})

	t.Run("10. :first-child on nav links", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#main-nav li:first-child a")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Home", node.Text(context.Background()))
	})

	t.Run("11. :last-child on nav links", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#main-nav li:last-child a")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Contact", node.Text(context.Background()))
	})

	t.Run("12. :nth-child(even) on tags", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".tag-list li:nth-child(even)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "Tag 2", nodes[0].Text(context.Background()))
		assert.Equal(t, "Tag 4", nodes[1].Text(context.Background()))
	})

	t.Run("13. :nth-child(2n+1) on comments (includes fragment)", func(t *testing.T) {

		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".comment-list > .comment:nth-child(2n+1)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "comment-1")
		assertHasAttribute(t, nodes[1], "id", "comment-3")
	})

	t.Run("14. :first-of-type for paragraphs in a post", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "article.post > p:first-of-type", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "First paragraph.", nodes[0].Text(context.Background()))
		assert.Equal(t, "Another first paragraph.", nodes[1].Text(context.Background()))
	})

	t.Run("15. :last-of-type for spans in a comment", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#comment-1 span:last-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "User A", node.Text(context.Background()))
	})

	t.Run("16. :nth-of-type(2) for paragraphs in the footer", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "footer p:nth-of-type(2)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "back-to-top")
	})

	t.Run("17. :not() to exclude active nav link", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "a.nav-link:not(.active)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
	})

	t.Run("18. :not() chained to exclude multiple classes", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "a.nav-link:not(.active):not(.disabled)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "About", node.Text(context.Background()))
	})

	t.Run("19. :not() with attribute selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `article:not([data-category="tech"])`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assertHasAttribute(t, nodes[0], "data-category", "lifestyle")
	})

	t.Run("20. Combining descendant, child, and pseudo-class selectors", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "main article .meta > span:first-child")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "Author: Alex", node.Text(context.Background()))
	})

	t.Run("21. Universal selector with adjacent sibling", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#site-title + *")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "nav", node.TagName)
	})

	t.Run("22. Universal selector with general sibling", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#comments ~ *", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 1)
		assert.Equal(t, "article", nodes[0].TagName)
	})

	t.Run("23. Selecting an element inside a fragment by ID", func(t *testing.T) {

		node, diagnostics := ast_domain.Query(templateAST, "#comment-2")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "div", node.TagName)
	})

	t.Run("24. Chaining :not() with positional selectors", func(t *testing.T) {

		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".comment:not(:first-child)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "comment-2")
		assertHasAttribute(t, nodes[1], "id", "comment-3")
	})

	t.Run("25. Combining multiple combinators in a single group", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "article.post > .meta ~ *", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes, "There are no siblings after .meta div")
	})

	t.Run("26. Combining multiple combinators in a single group (successful)", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "header + main")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "main", node.TagName)
	})

	t.Run("27. Complex :nth-child formula", func(t *testing.T) {

		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".tag-list li:nth-child(-n+4)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4)
	})

	t.Run("28. Querying for a link within a specific context", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#back-to-top a")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "href", "#app")
	})

	t.Run("29. Selector that results in no matches", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "header > h2", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)
	})

	t.Run("30. Combining tag, class, and attribute selector", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `a.nav-link[href="/about"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "About", node.Text(context.Background()))
	})

	t.Run("31. Querying for direct children of fragment-sibling's parent", func(t *testing.T) {

		node, diagnostics := ast_domain.Query(templateAST, "#comment-1 + .comment")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "comment-2")
	})

	t.Run("32. Querying for general siblings after fragment", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "#comment-1 ~ .comment", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "comment-2")
		assertHasAttribute(t, nodes[1], "id", "comment-3")
	})

	t.Run("33. Chained adjacent sibling combinators", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "article > h2 + p + p")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.True(t, hasClass(node, "standout"))
	})

	t.Run("34. Attribute [~=] for space-separated classes", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `[class~="theme-dark"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "div", node.TagName)
		assertHasAttribute(t, node, "id", "app")
	})

	t.Run("35. Attribute [$=] for suffix match on ID", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `[id$="-top"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "p", node.TagName)
	})

	t.Run("36. :nth-last-child to get second to last post", func(t *testing.T) {

		node, diagnostics := ast_domain.Query(templateAST, "main > :nth-last-child(2)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "section", node.TagName)
		assertHasAttribute(t, node, "id", "comments")
	})

	t.Run("37. :nth-last-of-type to get second to last paragraph in footer", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "footer > p:nth-last-of-type(2)")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "copyright")
	})

	t.Run("38. Native :only-child support (previously simulated)", func(t *testing.T) {

		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".tag-list li > span:only-child", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 4, "Should match all span elements that are only children of their li")
	})

	t.Run("39. Native :only-of-type support (previously simulated)", func(t *testing.T) {

		node, diagnostics := ast_domain.Query(templateAST, ".meta time:only-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
	})

	t.Run("40. Grouping selector with overlapping matches to test uniqueness", func(t *testing.T) {

		nodes, diagnostics := ast_domain.QueryAll(templateAST, `p.standout, [data-id="123"] p`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
	})

	t.Run("41. Complex :not() excluding a tag with a specific role", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `#app > *:not([role="banner"])`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "main", nodes[0].TagName)
		assert.Equal(t, "footer", nodes[1].TagName)
	})

	t.Run("42. :not() with chained simple selector", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, `a.nav-link:not(.active)`, "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "About", nodes[0].Text(context.Background()))
		assert.Equal(t, "Contact", nodes[1].Text(context.Background()))
	})

	t.Run("43. Adjacent sibling of an element from a fragment", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, "#comment-2 + .comment")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "comment-3")
	})

	t.Run("44. Universal selector combined with :nth-of-type", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, ".meta > *:nth-of-type(1)", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assert.Equal(t, "span", nodes[0].TagName)
		assert.Equal(t, "time", nodes[1].TagName)
	})

	t.Run("45. Selector targeting nothing with a valid but impossible chain", func(t *testing.T) {
		nodes, diagnostics := ast_domain.QueryAll(templateAST, "h1 + h2", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)
	})

	t.Run("46. Selecting based on a less common attribute", func(t *testing.T) {
		node, diagnostics := ast_domain.Query(templateAST, `[role="contentinfo"]`)
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Equal(t, "footer", node.TagName)
	})
}

func TestASTQueryEngine_NodeMethods(t *testing.T) {
	html := `
		<div id="root">
			<section id="sec1">
				<p id="p1" class="item">A</p>
				<p id="p2" class="item special">B</p>
			</section>
			<section id="sec2">
				<p id="p3" class="item">C</p>
				<div class="nested">
					<p id="p4" class="item">D</p>
				</div>
			</section>
			<p id="p5" class="item">E</p>
		</div>
	`
	templateAST := newTestAST(t, html)
	rootNode := ast_domain.MustQuery(templateAST, "#root")
	require.NotNil(t, rootNode, "Setup failed: could not find #root")
	section2Node := ast_domain.MustQuery(templateAST, "#sec2")
	require.NotNil(t, section2Node, "Setup failed: could not find #sec2")

	t.Run("QueryAll finds descendants within the node", func(t *testing.T) {
		nodes, diagnostics := section2Node.QueryAll(".item", "test.selector")
		require.Empty(t, diagnostics)
		require.Len(t, nodes, 2)
		assertHasAttribute(t, nodes[0], "id", "p3")
		assertHasAttribute(t, nodes[1], "id", "p4")
	})

	t.Run("Query finds first descendant", func(t *testing.T) {
		node, diagnostics := section2Node.Query(".item")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p3")
	})

	t.Run("Query can select the node itself", func(t *testing.T) {
		node, diagnostics := section2Node.Query("#sec2")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Same(t, section2Node, node)
	})

	t.Run("Positional pseudo-classes are scoped to the node's context", func(t *testing.T) {

		node, diagnostics := section2Node.Query("section:first-of-type")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assert.Same(t, section2Node, node)
	})

	t.Run("Sibling combinators work relative to the node's children", func(t *testing.T) {

		node, diagnostics := rootNode.Query("#sec1 + section")
		require.Empty(t, diagnostics)
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "sec2")

		node, diagnostics = section2Node.Query("#p1 + div")
		require.Empty(t, diagnostics)
		assert.Nil(t, node)
	})

	t.Run("Query for something outside the subtree returns no results", func(t *testing.T) {
		nodes, diagnostics := section2Node.QueryAll("#p1", "test.selector")
		require.Empty(t, diagnostics)
		assert.Empty(t, nodes)
	})

	t.Run("MustQuery on node returns first result or nil", func(t *testing.T) {
		node := section2Node.MustQuery(".item")
		require.NotNil(t, node)
		assertHasAttribute(t, node, "id", "p3")

		node = section2Node.MustQuery(".nonexistent")
		assert.Nil(t, node)
	})

	t.Run("Query methods on a nil node do not panic", func(t *testing.T) {
		var nilNode *ast_domain.TemplateNode
		assert.NotPanics(t, func() {
			nodes, diagnostics := nilNode.QueryAll(".item", "test.selector")
			assert.Nil(t, nodes)
			assert.Nil(t, diagnostics)

			node, diagnostics := nilNode.Query(".item")
			assert.Nil(t, node)
			assert.Nil(t, diagnostics)

			mustNode := nilNode.MustQuery(".item")
			assert.Nil(t, mustNode)
		})
	})
}
