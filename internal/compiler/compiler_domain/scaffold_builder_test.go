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

package compiler_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	es_ast "piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/css_ast"
	"piko.sh/piko/internal/esbuild/css_lexer"
)

func TestNewScaffoldBuilder(t *testing.T) {
	t.Run("creates scaffold builder", func(t *testing.T) {
		builder := NewScaffoldBuilder()

		require.NotNil(t, builder)
		_, ok := builder.(*scaffoldBuilder)
		assert.True(t, ok, "should return *scaffoldBuilder")
	})
}

func TestNewUsedSelectors(t *testing.T) {
	t.Run("creates initialised used selectors", func(t *testing.T) {
		selectors := newUsedSelectors()

		require.NotNil(t, selectors)
		require.NotNil(t, selectors.classes)
		require.NotNil(t, selectors.ids)
		require.NotNil(t, selectors.tags)
		assert.Empty(t, selectors.classes)
		assert.Empty(t, selectors.ids)
		assert.Empty(t, selectors.tags)
	})

	t.Run("maps are usable immediately", func(t *testing.T) {
		selectors := newUsedSelectors()

		selectors.classes["my-class"] = true
		selectors.ids["my-id"] = true
		selectors.tags["div"] = true

		assert.True(t, selectors.classes["my-class"])
		assert.True(t, selectors.ids["my-id"])
		assert.True(t, selectors.tags["div"])
	})
}

func TestSelfClosingTags(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{name: "br is self-closing", tag: "br", expected: true},
		{name: "img is self-closing", tag: "img", expected: true},
		{name: "input is self-closing", tag: "input", expected: true},
		{name: "hr is self-closing", tag: "hr", expected: true},
		{name: "meta is self-closing", tag: "meta", expected: true},
		{name: "link is self-closing", tag: "link", expected: true},
		{name: "div is not self-closing", tag: "div", expected: false},
		{name: "span is not self-closing", tag: "span", expected: false},
		{name: "p is not self-closing", tag: "p", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, selfClosingTags[tt.tag])
		})
	}
}

func TestInteractivePseudoClasses(t *testing.T) {
	tests := []struct {
		name     string
		pseudo   string
		expected bool
	}{
		{name: "hover is interactive", pseudo: "hover", expected: true},
		{name: "focus is interactive", pseudo: "focus", expected: true},
		{name: "focus-visible is interactive", pseudo: "focus-visible", expected: true},
		{name: "focus-within is interactive", pseudo: "focus-within", expected: true},
		{name: "active is interactive", pseudo: "active", expected: true},
		{name: "visited is interactive", pseudo: "visited", expected: true},
		{name: "target is interactive", pseudo: "target", expected: true},
		{name: "first-child is not interactive", pseudo: "first-child", expected: false},
		{name: "last-child is not interactive", pseudo: "last-child", expected: false},
		{name: "nth-child is not interactive", pseudo: "nth-child", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, interactivePseudoClasses[tt.pseudo])
		})
	}
}

func TestPropertiesToStrip(t *testing.T) {
	tests := []struct {
		name     string
		property string
		expected bool
	}{
		{name: "transition is stripped", property: "transition", expected: true},
		{name: "animation is stripped", property: "animation", expected: true},
		{name: "cursor is stripped", property: "cursor", expected: true},
		{name: "pointer-events is stripped", property: "pointer-events", expected: true},
		{name: "user-select is stripped", property: "user-select", expected: true},
		{name: "will-change is stripped", property: "will-change", expected: true},
		{name: "outline is stripped", property: "outline", expected: true},
		{name: "color is not stripped", property: "color", expected: false},
		{name: "background is not stripped", property: "background", expected: false},
		{name: "margin is not stripped", property: "margin", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, propertiesToStrip[tt.property])
		})
	}
}

func TestAssembleFinalScaffold(t *testing.T) {
	tests := []struct {
		name       string
		staticHTML string
		css        string
		expected   string
	}{
		{
			name:       "with CSS and HTML",
			staticHTML: "<div>Hello</div>",
			css:        ".test{color:red}",
			expected:   `<template shadowrootmode="open"><style>.test{color:red}</style><div>Hello</div></template>`,
		},
		{
			name:       "with HTML only no CSS",
			staticHTML: "<span>World</span>",
			css:        "",
			expected:   `<template shadowrootmode="open"><span>World</span></template>`,
		},
		{
			name:       "empty HTML with CSS",
			staticHTML: "",
			css:        "body{margin:0}",
			expected:   `<template shadowrootmode="open"><style>body{margin:0}</style></template>`,
		},
		{
			name:       "both empty",
			staticHTML: "",
			css:        "",
			expected:   `<template shadowrootmode="open"></template>`,
		},
		{
			name:       "complex HTML structure",
			staticHTML: `<div class="container"><p>Text</p></div>`,
			css:        ".container{padding:10px}p{margin:0}",
			expected:   `<template shadowrootmode="open"><style>.container{padding:10px}p{margin:0}</style><div class="container"><p>Text</p></div></template>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := assembleFinalScaffold(tt.staticHTML, tt.css)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollectAttributeSelectors(t *testing.T) {
	t.Run("collects class names", func(t *testing.T) {
		selectors := newUsedSelectors()
		attr := &ast_domain.HTMLAttribute{
			Name:  "class",
			Value: "foo bar baz",
		}

		collectAttributeSelectors(attr, selectors)

		assert.True(t, selectors.classes["foo"])
		assert.True(t, selectors.classes["bar"])
		assert.True(t, selectors.classes["baz"])
		assert.Len(t, selectors.classes, 3)
	})

	t.Run("collects single class", func(t *testing.T) {
		selectors := newUsedSelectors()
		attr := &ast_domain.HTMLAttribute{
			Name:  "class",
			Value: "single-class",
		}

		collectAttributeSelectors(attr, selectors)

		assert.True(t, selectors.classes["single-class"])
		assert.Len(t, selectors.classes, 1)
	})

	t.Run("collects ID", func(t *testing.T) {
		selectors := newUsedSelectors()
		attr := &ast_domain.HTMLAttribute{
			Name:  "id",
			Value: "my-element",
		}

		collectAttributeSelectors(attr, selectors)

		assert.True(t, selectors.ids["my-element"])
		assert.Len(t, selectors.ids, 1)
	})

	t.Run("handles uppercase attribute names", func(t *testing.T) {
		selectors := newUsedSelectors()
		classAttr := &ast_domain.HTMLAttribute{Name: "CLASS", Value: "test"}
		idAttr := &ast_domain.HTMLAttribute{Name: "ID", Value: "myId"}

		collectAttributeSelectors(classAttr, selectors)
		collectAttributeSelectors(idAttr, selectors)

		assert.True(t, selectors.classes["test"])
		assert.True(t, selectors.ids["myId"])
	})

	t.Run("ignores other attributes", func(t *testing.T) {
		selectors := newUsedSelectors()
		attr := &ast_domain.HTMLAttribute{
			Name:  "data-value",
			Value: "something",
		}

		collectAttributeSelectors(attr, selectors)

		assert.Empty(t, selectors.classes)
		assert.Empty(t, selectors.ids)
	})

	t.Run("handles empty class value", func(t *testing.T) {
		selectors := newUsedSelectors()
		attr := &ast_domain.HTMLAttribute{
			Name:  "class",
			Value: "",
		}

		collectAttributeSelectors(attr, selectors)

		assert.Empty(t, selectors.classes)
	})
}

func TestWriteAttributesAndCollect(t *testing.T) {
	t.Run("writes standard attributes", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "test"},
			{Name: "id", Value: "myid"},
			{Name: "data-value", Value: "123"},
		}

		writeAttributesAndCollect(&builder, attrs, selectors)

		result := builder.String()
		assert.Contains(t, result, `class="test"`)
		assert.Contains(t, result, `id="myid"`)
		assert.Contains(t, result, `data-value="123"`)
	})

	t.Run("skips p- prefixed attributes", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "visible"},
			{Name: "p-if", Value: "condition"},
			{Name: "p-for", Value: "item in items"},
		}

		writeAttributesAndCollect(&builder, attrs, selectors)

		result := builder.String()
		assert.Contains(t, result, `class="visible"`)
		assert.NotContains(t, result, "p-if")
		assert.NotContains(t, result, "p-for")
	})

	t.Run("skips colon prefixed attributes", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		attrs := []ast_domain.HTMLAttribute{
			{Name: "id", Value: "elem"},
			{Name: ":class", Value: "dynamicClass"},
			{Name: ":style", Value: "dynamicStyle"},
		}

		writeAttributesAndCollect(&builder, attrs, selectors)

		result := builder.String()
		assert.Contains(t, result, `id="elem"`)
		assert.NotContains(t, result, ":class")
		assert.NotContains(t, result, ":style")
	})

	t.Run("escapes HTML in attribute values", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		attrs := []ast_domain.HTMLAttribute{
			{Name: "data-html", Value: `<script>"alert('xss')"</script>`},
		}

		writeAttributesAndCollect(&builder, attrs, selectors)

		result := builder.String()
		assert.Contains(t, result, "&lt;script&gt;")
		assert.Contains(t, result, "&#34;")
	})

	t.Run("collects selectors while writing", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		attrs := []ast_domain.HTMLAttribute{
			{Name: "class", Value: "btn primary"},
			{Name: "id", Value: "submit-btn"},
		}

		writeAttributesAndCollect(&builder, attrs, selectors)

		assert.True(t, selectors.classes["btn"])
		assert.True(t, selectors.classes["primary"])
		assert.True(t, selectors.ids["submit-btn"])
	})
}

func TestWriteStaticNodeAndCollectSelectors(t *testing.T) {
	t.Run("writes text node", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "Hello World",
		}

		err := writeStaticNodeAndCollectSelectors(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, "Hello World", builder.String())
	})

	t.Run("escapes HTML in text content", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "<script>alert('xss')</script>",
		}

		err := writeStaticNodeAndCollectSelectors(&builder, node, selectors)

		require.NoError(t, err)
		assert.Contains(t, builder.String(), "&lt;script&gt;")
	})

	t.Run("skips nodes with DirIf", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirIf:    &ast_domain.Directive{Type: ast_domain.DirectiveIf, RawExpression: "show"},
		}

		err := writeStaticNodeAndCollectSelectors(&builder, node, selectors)

		require.NoError(t, err)
		assert.Empty(t, builder.String())
	})

	t.Run("skips nodes with DirFor", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			DirFor:   &ast_domain.Directive{Type: ast_domain.DirectiveFor, RawExpression: "item in items"},
		}

		err := writeStaticNodeAndCollectSelectors(&builder, node, selectors)

		require.NoError(t, err)
		assert.Empty(t, builder.String())
	})
}

func TestWriteElementNode(t *testing.T) {
	t.Run("writes element with tag name", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, "<div></div>", builder.String())
		assert.True(t, selectors.tags["div"])
	})

	t.Run("converts tag name to lowercase", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "DIV",
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, "<div></div>", builder.String())
		assert.True(t, selectors.tags["div"])
	})

	t.Run("writes self-closing tag without end tag", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "br",
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, "<br>", builder.String())
	})

	t.Run("writes self-closing img with attributes", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "image.png"},
				{Name: "alt", Value: "An image"},
			},
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Contains(t, builder.String(), `<img`)
		assert.Contains(t, builder.String(), `src="image.png"`)
		assert.Contains(t, builder.String(), `alt="An image"`)
		assert.NotContains(t, builder.String(), "</img>")
	})

	t.Run("writes element with children", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Hello"},
			},
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, "<div>Hello</div>", builder.String())
	})

	t.Run("writes nested elements", func(t *testing.T) {
		var builder strings.Builder
		selectors := newUsedSelectors()
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "span",
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Nested"},
					},
				},
			},
		}

		err := writeElementNode(&builder, node, selectors)

		require.NoError(t, err)
		assert.Equal(t, `<div class="container"><span>Nested</span></div>`, builder.String())
		assert.True(t, selectors.tags["div"])
		assert.True(t, selectors.tags["span"])
		assert.True(t, selectors.classes["container"])
	})
}

func TestAnyMatches(t *testing.T) {
	tests := []struct {
		matches  map[string]bool
		name     string
		items    []string
		expected bool
	}{
		{
			name:     "single match",
			items:    []string{"foo"},
			matches:  map[string]bool{"foo": true},
			expected: true,
		},
		{
			name:     "one of multiple matches",
			items:    []string{"foo", "bar"},
			matches:  map[string]bool{"bar": true},
			expected: true,
		},
		{
			name:     "no match",
			items:    []string{"foo", "bar"},
			matches:  map[string]bool{"baz": true},
			expected: false,
		},
		{
			name:     "empty items",
			items:    []string{},
			matches:  map[string]bool{"foo": true},
			expected: false,
		},
		{
			name:     "empty matches",
			items:    []string{"foo"},
			matches:  map[string]bool{},
			expected: false,
		},
		{
			name:     "both empty",
			items:    []string{},
			matches:  map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyMatches(tt.items, tt.matches)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVarFunction(t *testing.T) {
	tests := []struct {
		name     string
		tok      css_ast.Token
		expected bool
	}{
		{
			name: "var function with children",
			tok: css_ast.Token{
				Kind:     css_lexer.TFunction,
				Text:     "var",
				Children: &[]css_ast.Token{},
			},
			expected: true,
		},
		{
			name: "VAR uppercase",
			tok: css_ast.Token{
				Kind:     css_lexer.TFunction,
				Text:     "VAR",
				Children: &[]css_ast.Token{},
			},
			expected: true,
		},
		{
			name: "var without children",
			tok: css_ast.Token{
				Kind:     css_lexer.TFunction,
				Text:     "var",
				Children: nil,
			},
			expected: false,
		},
		{
			name: "different function name",
			tok: css_ast.Token{
				Kind:     css_lexer.TFunction,
				Text:     "calc",
				Children: &[]css_ast.Token{},
			},
			expected: false,
		},
		{
			name: "not a function",
			tok: css_ast.Token{
				Kind:     css_lexer.TIdent,
				Text:     "var",
				Children: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVarFunction(tt.tok)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractVarName(t *testing.T) {
	tests := []struct {
		name     string
		children *[]css_ast.Token
		expected string
	}{
		{
			name: "extracts CSS variable",
			children: &[]css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "--my-var"},
			},
			expected: "--my-var",
		},
		{
			name: "skips whitespace",
			children: &[]css_ast.Token{
				{Kind: css_lexer.TWhitespace, Text: " "},
				{Kind: css_lexer.TIdent, Text: "--color"},
			},
			expected: "--color",
		},
		{
			name:     "nil children",
			children: nil,
			expected: "",
		},
		{
			name:     "empty children",
			children: &[]css_ast.Token{},
			expected: "",
		},
		{
			name: "no variable prefix",
			children: &[]css_ast.Token{
				{Kind: css_lexer.TIdent, Text: "not-a-var"},
			},
			expected: "",
		},
		{
			name: "only whitespace",
			children: &[]css_ast.Token{
				{Kind: css_lexer.TWhitespace, Text: " "},
				{Kind: css_lexer.TWhitespace, Text: "  "},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVarName(tt.children)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasInteractivePseudo(t *testing.T) {
	t.Run("returns true for hover", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "hover"}},
			},
		}

		assert.True(t, hasInteractivePseudo(component))
	})

	t.Run("returns true for focus", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "focus"}},
			},
		}

		assert.True(t, hasInteractivePseudo(component))
	})

	t.Run("returns true for active", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "active"}},
			},
		}

		assert.True(t, hasInteractivePseudo(component))
	})

	t.Run("returns false for first-child", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "first-child"}},
			},
		}

		assert.False(t, hasInteractivePseudo(component))
	})

	t.Run("returns false for empty subclass selectors", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		assert.False(t, hasInteractivePseudo(component))
	})

	t.Run("returns false for non-pseudo subclass", func(t *testing.T) {
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
			},
		}

		assert.False(t, hasInteractivePseudo(component))
	})
}

func TestIsHostSelector(t *testing.T) {
	t.Run("returns true for host", func(t *testing.T) {
		selector := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "host"}},
			},
		}

		assert.True(t, isHostSelector(selector))
	})

	t.Run("returns true for host-context", func(t *testing.T) {
		selector := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "host-context"}},
			},
		}

		assert.True(t, isHostSelector(selector))
	})

	t.Run("returns false for other pseudo", func(t *testing.T) {
		selector := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSPseudoClass{Name: "before"}},
			},
		}

		assert.False(t, isHostSelector(selector))
	})

	t.Run("returns false for empty subclass selectors", func(t *testing.T) {
		selector := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		assert.False(t, isHostSelector(selector))
	})
}

func TestTypeMatchesUsedTags(t *testing.T) {
	t.Run("matches used tag", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true
		component := css_ast.CompoundSelector{
			TypeSelector: &css_ast.NamespacedName{
				Name: css_ast.NameToken{Text: "div"},
			},
		}

		assert.True(t, typeMatchesUsedTags(component, selectors))
	})

	t.Run("does not match unused tag", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true
		component := css_ast.CompoundSelector{
			TypeSelector: &css_ast.NamespacedName{
				Name: css_ast.NameToken{Text: "span"},
			},
		}

		assert.False(t, typeMatchesUsedTags(component, selectors))
	})

	t.Run("matches case insensitively", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true
		component := css_ast.CompoundSelector{
			TypeSelector: &css_ast.NamespacedName{
				Name: css_ast.NameToken{Text: "DIV"},
			},
		}

		assert.True(t, typeMatchesUsedTags(component, selectors))
	})

	t.Run("returns true for nil type selector", func(t *testing.T) {
		selectors := newUsedSelectors()
		component := css_ast.CompoundSelector{
			TypeSelector: nil,
		}

		assert.True(t, typeMatchesUsedTags(component, selectors))
	})
}

func TestGetClassesFromComponent(t *testing.T) {
	t.Run("extracts class names from component", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "class1"},
			{OriginalName: "class2"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 1}}}},
			},
		}

		classes := getClassesFromComponent(component, symbols)

		assert.Equal(t, []string{"class1", "class2"}, classes)
	})

	t.Run("skips non-class selectors", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "myclass"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSPseudoClass{Name: "hover"}},
			},
		}

		classes := getClassesFromComponent(component, symbols)

		assert.Equal(t, []string{"myclass"}, classes)
	})

	t.Run("returns empty for empty component", func(t *testing.T) {
		symbols := []es_ast.Symbol{}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		classes := getClassesFromComponent(component, symbols)

		assert.Empty(t, classes)
	})

	t.Run("skips out of bounds references", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "only-one"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 99}}}},
			},
		}

		classes := getClassesFromComponent(component, symbols)

		assert.Equal(t, []string{"only-one"}, classes)
	})
}

func TestGetIDsFromComponent(t *testing.T) {
	t.Run("extracts ID names from component", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "id1"},
			{OriginalName: "id2"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 1}}}},
			},
		}

		ids := getIDsFromComponent(component, symbols)

		assert.Equal(t, []string{"id1", "id2"}, ids)
	})

	t.Run("skips non-hash selectors", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "myid"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{}}},
			},
		}

		ids := getIDsFromComponent(component, symbols)

		assert.Equal(t, []string{"myid"}, ids)
	})

	t.Run("returns empty for empty component", func(t *testing.T) {
		symbols := []es_ast.Symbol{}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		ids := getIDsFromComponent(component, symbols)

		assert.Empty(t, ids)
	})

	t.Run("skips out of bounds references", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "only-id"},
		}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 100}}}},
			},
		}

		ids := getIDsFromComponent(component, symbols)

		assert.Equal(t, []string{"only-id"}, ids)
	})
}

func TestClassMatchesUsedClasses(t *testing.T) {
	t.Run("returns true when class matches", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.classes["btn"] = true
		symbols := []es_ast.Symbol{{OriginalName: "btn"}}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
			},
		}

		assert.True(t, classMatchesUsedClasses(component, selectors, symbols))
	})

	t.Run("returns false when class does not match", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.classes["other"] = true
		symbols := []es_ast.Symbol{{OriginalName: "btn"}}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
			},
		}

		assert.False(t, classMatchesUsedClasses(component, selectors, symbols))
	})

	t.Run("returns true when no class selectors", func(t *testing.T) {
		selectors := newUsedSelectors()
		symbols := []es_ast.Symbol{}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		assert.True(t, classMatchesUsedClasses(component, selectors, symbols))
	})
}

func TestIdMatchesUsedIDs(t *testing.T) {
	t.Run("returns true when ID matches", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.ids["header"] = true
		symbols := []es_ast.Symbol{{OriginalName: "header"}}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
			},
		}

		assert.True(t, idMatchesUsedIDs(component, selectors, symbols))
	})

	t.Run("returns false when ID does not match", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.ids["footer"] = true
		symbols := []es_ast.Symbol{{OriginalName: "header"}}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{
				{Data: &css_ast.SSHash{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
			},
		}

		assert.False(t, idMatchesUsedIDs(component, selectors, symbols))
	})

	t.Run("returns true when no ID selectors", func(t *testing.T) {
		selectors := newUsedSelectors()
		symbols := []es_ast.Symbol{}
		component := css_ast.CompoundSelector{
			SubclassSelectors: []css_ast.SubclassSelector{},
		}

		assert.True(t, idMatchesUsedIDs(component, selectors, symbols))
	})
}

func TestCollectUsedVariables(t *testing.T) {
	t.Run("collects variables from declarations", func(t *testing.T) {
		usedVars := make(map[string]bool)
		rules := []css_ast.Rule{
			{
				Data: &css_ast.RDeclaration{
					Value: []css_ast.Token{
						{
							Kind:     css_lexer.TFunction,
							Text:     "var",
							Children: &[]css_ast.Token{{Kind: css_lexer.TIdent, Text: "--primary-color"}},
						},
					},
				},
			},
		}

		collectUsedVariables(rules, usedVars)

		assert.True(t, usedVars["--primary-color"])
	})

	t.Run("collects variables from nested selector rules", func(t *testing.T) {
		usedVars := make(map[string]bool)
		rules := []css_ast.Rule{
			{
				Data: &css_ast.RSelector{
					Rules: []css_ast.Rule{
						{
							Data: &css_ast.RDeclaration{
								Value: []css_ast.Token{
									{
										Kind:     css_lexer.TFunction,
										Text:     "var",
										Children: &[]css_ast.Token{{Kind: css_lexer.TIdent, Text: "--nested-var"}},
									},
								},
							},
						},
					},
				},
			},
		}

		collectUsedVariables(rules, usedVars)

		assert.True(t, usedVars["--nested-var"])
	})

	t.Run("collects variables from nested at-rules", func(t *testing.T) {
		usedVars := make(map[string]bool)
		rules := []css_ast.Rule{
			{
				Data: &css_ast.RKnownAt{
					Rules: []css_ast.Rule{
						{
							Data: &css_ast.RDeclaration{
								Value: []css_ast.Token{
									{
										Kind:     css_lexer.TFunction,
										Text:     "var",
										Children: &[]css_ast.Token{{Kind: css_lexer.TIdent, Text: "--media-var"}},
									},
								},
							},
						},
					},
				},
			},
		}

		collectUsedVariables(rules, usedVars)

		assert.True(t, usedVars["--media-var"])
	})

	t.Run("handles empty rules", func(t *testing.T) {
		usedVars := make(map[string]bool)
		rules := []css_ast.Rule{}

		collectUsedVariables(rules, usedVars)

		assert.Empty(t, usedVars)
	})
}

func TestStripUnusedDeclarations(t *testing.T) {
	t.Run("strips transition properties", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "transition"}},
			{Data: &css_ast.RDeclaration{KeyText: "color"}},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		assert.Equal(t, "color", result[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("strips animation properties", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "animation"}},
			{Data: &css_ast.RDeclaration{KeyText: "animation-name"}},
			{Data: &css_ast.RDeclaration{KeyText: "background"}},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		assert.Equal(t, "background", result[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("strips cursor and pointer-events", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "cursor"}},
			{Data: &css_ast.RDeclaration{KeyText: "pointer-events"}},
			{Data: &css_ast.RDeclaration{KeyText: "display"}},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		assert.Equal(t, "display", result[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("strips unused CSS variables", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "--unused-var"}},
			{Data: &css_ast.RDeclaration{KeyText: "--used-var"}},
		}
		usedVars := map[string]bool{"--used-var": true}

		result := stripUnusedDeclarations(rules, usedVars)

		require.Len(t, result, 1)
		assert.Equal(t, "--used-var", result[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("keeps used CSS variables", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "--my-var"}},
		}
		usedVars := map[string]bool{"--my-var": true}

		result := stripUnusedDeclarations(rules, usedVars)

		require.Len(t, result, 1)
	})

	t.Run("handles case insensitive property names", func(t *testing.T) {
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "TRANSITION"}},
			{Data: &css_ast.RDeclaration{KeyText: "Color"}},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		assert.Equal(t, "Color", result[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("processes nested selector rules", func(t *testing.T) {
		rules := []css_ast.Rule{
			{
				Data: &css_ast.RSelector{
					Rules: []css_ast.Rule{
						{Data: &css_ast.RDeclaration{KeyText: "transition"}},
						{Data: &css_ast.RDeclaration{KeyText: "color"}},
					},
				},
			},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		selectorRule, ok := result[0].Data.(*css_ast.RSelector)
		require.True(t, ok, "result[0].Data should be *css_ast.RSelector")
		require.Len(t, selectorRule.Rules, 1)
		assert.Equal(t, "color", selectorRule.Rules[0].Data.(*css_ast.RDeclaration).KeyText)
	})

	t.Run("processes nested at-rules", func(t *testing.T) {
		rules := []css_ast.Rule{
			{
				Data: &css_ast.RKnownAt{
					Rules: []css_ast.Rule{
						{Data: &css_ast.RDeclaration{KeyText: "cursor"}},
						{Data: &css_ast.RDeclaration{KeyText: "margin"}},
					},
				},
			},
		}

		result := stripUnusedDeclarations(rules, map[string]bool{})

		require.Len(t, result, 1)
		atRule, ok := result[0].Data.(*css_ast.RKnownAt)
		require.True(t, ok, "result[0].Data should be *css_ast.RKnownAt")
		require.Len(t, atRule.Rules, 1)
		assert.Equal(t, "margin", atRule.Rules[0].Data.(*css_ast.RDeclaration).KeyText)
	})
}

func TestBuildStaticScaffold(t *testing.T) {
	ctx := context.Background()
	builder := NewScaffoldBuilder()

	t.Run("returns error for nil template AST", func(t *testing.T) {
		result, err := builder.BuildStaticScaffold(ctx, nil, "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil TemplateAST")
		assert.Empty(t, result)
	})

	t.Run("builds scaffold for empty template", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.Equal(t, `<template shadowrootmode="open"></template>`, result)
	})

	t.Run("builds scaffold with text node", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Hello"},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.Equal(t, `<template shadowrootmode="open">Hello</template>`, result)
	})

	t.Run("builds scaffold with element", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "container"},
					},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Content"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.Contains(t, result, `<div class="container">Content</div>`)
	})

	t.Run("includes CSS in scaffold", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "test"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, ".test{color:red}")

		require.NoError(t, err)
		assert.Contains(t, result, `<style>`)

		assert.Contains(t, result, `color`)
	})

	t.Run("skips conditional nodes", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Static"},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf:    &ast_domain.Directive{Type: ast_domain.DirectiveIf, RawExpression: "show"},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Conditional"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.Contains(t, result, "Static")
		assert.NotContains(t, result, "Conditional")
	})

	t.Run("skips loop nodes", func(t *testing.T) {
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Before"},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirFor:   &ast_domain.Directive{Type: ast_domain.DirectiveFor, RawExpression: "item in items"},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Loop Item"},
					},
				},
				{NodeType: ast_domain.NodeText, TextContent: "After"},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.Contains(t, result, "Before")
		assert.Contains(t, result, "After")
		assert.NotContains(t, result, "Loop Item")
	})
}

func TestTreeShakeCSSWithFallback(t *testing.T) {
	ctx := context.Background()

	t.Run("returns empty for empty CSS", func(t *testing.T) {
		selectors := newUsedSelectors()

		result := treeShakeCSSWithFallback(ctx, "", selectors)

		assert.Empty(t, result)
	})

	t.Run("returns empty for whitespace-only CSS", func(t *testing.T) {
		selectors := newUsedSelectors()

		result := treeShakeCSSWithFallback(ctx, "   \n\t  ", selectors)

		assert.Empty(t, result)
	})

	t.Run("returns original CSS on parse error", func(t *testing.T) {
		selectors := newUsedSelectors()
		invalidCSS := "@invalid {{{ broken"

		result := treeShakeCSSWithFallback(ctx, invalidCSS, selectors)

		assert.Equal(t, invalidCSS, result)
	})
}

func TestWithScaffoldConfig(t *testing.T) {
	t.Run("stores and retrieves config from context", func(t *testing.T) {
		config := ScaffoldBuilderConfig{
			CSSTreeShaking:         true,
			CSSTreeShakingSafelist: []string{"active", "open"},
		}
		ctx := WithScaffoldConfig(context.Background(), config)

		got := GetScaffoldConfig(ctx)

		assert.True(t, got.CSSTreeShaking)
		assert.Equal(t, []string{"active", "open"}, got.CSSTreeShakingSafelist)
	})

	t.Run("returns empty config when not set in context", func(t *testing.T) {
		got := GetScaffoldConfig(context.Background())

		assert.False(t, got.CSSTreeShaking)
		assert.Nil(t, got.CSSTreeShakingSafelist)
	})

	t.Run("stores config with tree shaking disabled", func(t *testing.T) {
		config := ScaffoldBuilderConfig{
			CSSTreeShaking: false,
		}
		ctx := WithScaffoldConfig(context.Background(), config)

		got := GetScaffoldConfig(ctx)

		assert.False(t, got.CSSTreeShaking)
	})

	t.Run("later config overrides earlier config", func(t *testing.T) {
		first := ScaffoldBuilderConfig{CSSTreeShaking: true}
		second := ScaffoldBuilderConfig{CSSTreeShaking: false}
		ctx := WithScaffoldConfig(context.Background(), first)
		ctx = WithScaffoldConfig(ctx, second)

		got := GetScaffoldConfig(ctx)

		assert.False(t, got.CSSTreeShaking)
	})
}

func TestFilterRules(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps selector rules matching used classes", func(t *testing.T) {
		symbols := []es_ast.Symbol{{OriginalName: "btn"}}
		selectors := newUsedSelectors()
		selectors.classes["btn"] = true
		selectors.tags["div"] = true

		rules := []css_ast.Rule{
			{
				Data: &css_ast.RSelector{
					Selectors: []css_ast.ComplexSelector{
						{
							Selectors: []css_ast.CompoundSelector{
								{
									SubclassSelectors: []css_ast.SubclassSelector{
										{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
									},
								},
							},
						},
					},
					Rules: []css_ast.Rule{
						{Data: &css_ast.RDeclaration{KeyText: "color"}},
					},
				},
			},
		}

		result := filterRules(ctx, rules, selectors, symbols)

		assert.Len(t, result, 1)
	})

	t.Run("removes selector rules not matching any used selector", func(t *testing.T) {
		symbols := []es_ast.Symbol{{OriginalName: "unused-class"}}
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		rules := []css_ast.Rule{
			{
				Data: &css_ast.RSelector{
					Selectors: []css_ast.ComplexSelector{
						{
							Selectors: []css_ast.CompoundSelector{
								{
									SubclassSelectors: []css_ast.SubclassSelector{
										{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
									},
								},
							},
						},
					},
					Rules: []css_ast.Rule{
						{Data: &css_ast.RDeclaration{KeyText: "color"}},
					},
				},
			},
		}

		result := filterRules(ctx, rules, selectors, symbols)

		assert.Empty(t, result)
	})

	t.Run("keeps declaration rules by default", func(t *testing.T) {
		selectors := newUsedSelectors()
		rules := []css_ast.Rule{
			{Data: &css_ast.RDeclaration{KeyText: "color"}},
		}

		result := filterRules(ctx, rules, selectors, nil)

		assert.Len(t, result, 1)
	})

	t.Run("handles empty rules slice", func(t *testing.T) {
		selectors := newUsedSelectors()

		result := filterRules(ctx, []css_ast.Rule{}, selectors, nil)

		assert.Empty(t, result)
	})
}

func TestShouldKeepRule(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps selector rule with matching tag", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		rule := css_ast.Rule{
			Data: &css_ast.RSelector{
				Selectors: []css_ast.ComplexSelector{
					{
						Selectors: []css_ast.CompoundSelector{
							{
								TypeSelector: &css_ast.NamespacedName{
									Name: css_ast.NameToken{Text: "div"},
								},
							},
						},
					},
				},
				Rules: []css_ast.Rule{
					{Data: &css_ast.RDeclaration{KeyText: "margin"}},
				},
			},
		}

		assert.True(t, shouldKeepRule(ctx, rule, selectors, nil))
	})

	t.Run("removes selector rule with unmatched tag", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		rule := css_ast.Rule{
			Data: &css_ast.RSelector{
				Selectors: []css_ast.ComplexSelector{
					{
						Selectors: []css_ast.CompoundSelector{
							{
								TypeSelector: &css_ast.NamespacedName{
									Name: css_ast.NameToken{Text: "span"},
								},
							},
						},
					},
				},
				Rules: []css_ast.Rule{
					{Data: &css_ast.RDeclaration{KeyText: "margin"}},
				},
			},
		}

		assert.False(t, shouldKeepRule(ctx, rule, selectors, nil))
	})

	t.Run("keeps at-rule with matching inner rules", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		rule := css_ast.Rule{
			Data: &css_ast.RKnownAt{
				AtToken: "media",
				Rules: []css_ast.Rule{
					{
						Data: &css_ast.RSelector{
							Selectors: []css_ast.ComplexSelector{
								{
									Selectors: []css_ast.CompoundSelector{
										{
											TypeSelector: &css_ast.NamespacedName{
												Name: css_ast.NameToken{Text: "div"},
											},
										},
									},
								},
							},
							Rules: []css_ast.Rule{
								{Data: &css_ast.RDeclaration{KeyText: "display"}},
							},
						},
					},
				},
			},
		}

		assert.True(t, shouldKeepRule(ctx, rule, selectors, nil))
	})

	t.Run("removes at-rule when all inner rules are removed", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		rule := css_ast.Rule{
			Data: &css_ast.RKnownAt{
				AtToken: "media",
				Rules: []css_ast.Rule{
					{
						Data: &css_ast.RSelector{
							Selectors: []css_ast.ComplexSelector{
								{
									Selectors: []css_ast.CompoundSelector{
										{
											TypeSelector: &css_ast.NamespacedName{
												Name: css_ast.NameToken{Text: "span"},
											},
										},
									},
								},
							},
							Rules: []css_ast.Rule{
								{Data: &css_ast.RDeclaration{KeyText: "display"}},
							},
						},
					},
				},
			},
		}

		assert.False(t, shouldKeepRule(ctx, rule, selectors, nil))
	})

	t.Run("keeps declaration rule unconditionally", func(t *testing.T) {
		selectors := newUsedSelectors()

		rule := css_ast.Rule{
			Data: &css_ast.RDeclaration{KeyText: "color"},
		}

		assert.True(t, shouldKeepRule(ctx, rule, selectors, nil))
	})
}

func TestFilterSelectorRule(t *testing.T) {
	ctx := context.Background()

	t.Run("keeps rule with matching selectors", func(t *testing.T) {
		symbols := []es_ast.Symbol{{OriginalName: "active"}}
		selectors := newUsedSelectors()
		selectors.classes["active"] = true

		r := &css_ast.RSelector{
			Selectors: []css_ast.ComplexSelector{
				{
					Selectors: []css_ast.CompoundSelector{
						{
							SubclassSelectors: []css_ast.SubclassSelector{
								{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
							},
						},
					},
				},
			},
			Rules: []css_ast.Rule{
				{Data: &css_ast.RDeclaration{KeyText: "color"}},
			},
		}

		assert.True(t, filterSelectorRule(ctx, r, selectors, symbols))
		assert.Len(t, r.Selectors, 1)
	})

	t.Run("returns false when no selectors match", func(t *testing.T) {
		symbols := []es_ast.Symbol{{OriginalName: "missing"}}
		selectors := newUsedSelectors()
		selectors.classes["other"] = true

		r := &css_ast.RSelector{
			Selectors: []css_ast.ComplexSelector{
				{
					Selectors: []css_ast.CompoundSelector{
						{
							SubclassSelectors: []css_ast.SubclassSelector{
								{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
							},
						},
					},
				},
			},
			Rules: []css_ast.Rule{
				{Data: &css_ast.RDeclaration{KeyText: "color"}},
			},
		}

		assert.False(t, filterSelectorRule(ctx, r, selectors, symbols))
	})

	t.Run("filters multiple selectors keeping only matching ones", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "used"},
			{OriginalName: "unused"},
		}
		selectors := newUsedSelectors()
		selectors.classes["used"] = true

		r := &css_ast.RSelector{
			Selectors: []css_ast.ComplexSelector{
				{
					Selectors: []css_ast.CompoundSelector{
						{
							SubclassSelectors: []css_ast.SubclassSelector{
								{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
							},
						},
					},
				},
				{
					Selectors: []css_ast.CompoundSelector{
						{
							SubclassSelectors: []css_ast.SubclassSelector{
								{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 1}}}},
							},
						},
					},
				},
			},
			Rules: []css_ast.Rule{
				{Data: &css_ast.RDeclaration{KeyText: "color"}},
			},
		}

		assert.True(t, filterSelectorRule(ctx, r, selectors, symbols))
		assert.Len(t, r.Selectors, 1, "should keep only the matching selector")
	})

	t.Run("recursively filters nested rules", func(t *testing.T) {
		symbols := []es_ast.Symbol{{OriginalName: "outer"}}
		selectors := newUsedSelectors()
		selectors.classes["outer"] = true
		selectors.tags["div"] = true

		r := &css_ast.RSelector{
			Selectors: []css_ast.ComplexSelector{
				{
					Selectors: []css_ast.CompoundSelector{
						{
							SubclassSelectors: []css_ast.SubclassSelector{
								{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
							},
						},
					},
				},
			},
			Rules: []css_ast.Rule{
				{
					Data: &css_ast.RSelector{
						Selectors: []css_ast.ComplexSelector{
							{
								Selectors: []css_ast.CompoundSelector{
									{
										TypeSelector: &css_ast.NamespacedName{
											Name: css_ast.NameToken{Text: "div"},
										},
									},
								},
							},
						},
						Rules: []css_ast.Rule{
							{Data: &css_ast.RDeclaration{KeyText: "margin"}},
						},
					},
				},
			},
		}

		assert.True(t, filterSelectorRule(ctx, r, selectors, symbols))
		assert.Len(t, r.Rules, 1, "nested matching rule should be kept")
	})
}

func TestFilterMatchingSelectors(t *testing.T) {
	t.Run("keeps selectors that match used tags", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		complexSelectors := []css_ast.ComplexSelector{
			{
				Selectors: []css_ast.CompoundSelector{
					{
						TypeSelector: &css_ast.NamespacedName{
							Name: css_ast.NameToken{Text: "div"},
						},
					},
				},
			},
		}

		result := filterMatchingSelectors(complexSelectors, selectors, nil)

		assert.Len(t, result, 1)
	})

	t.Run("removes selectors that do not match", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		complexSelectors := []css_ast.ComplexSelector{
			{
				Selectors: []css_ast.CompoundSelector{
					{
						TypeSelector: &css_ast.NamespacedName{
							Name: css_ast.NameToken{Text: "span"},
						},
					},
				},
			},
		}

		result := filterMatchingSelectors(complexSelectors, selectors, nil)

		assert.Empty(t, result)
	})

	t.Run("returns empty for empty input", func(t *testing.T) {
		selectors := newUsedSelectors()

		result := filterMatchingSelectors([]css_ast.ComplexSelector{}, selectors, nil)

		assert.Empty(t, result)
	})

	t.Run("keeps host selectors regardless of used selectors", func(t *testing.T) {
		selectors := newUsedSelectors()

		complexSelectors := []css_ast.ComplexSelector{
			{
				Selectors: []css_ast.CompoundSelector{
					{
						SubclassSelectors: []css_ast.SubclassSelector{
							{Data: &css_ast.SSPseudoClass{Name: "host"}},
						},
					},
				},
			},
		}

		result := filterMatchingSelectors(complexSelectors, selectors, nil)

		assert.Len(t, result, 1)
	})

	t.Run("removes selectors with interactive pseudo-classes", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		complexSelectors := []css_ast.ComplexSelector{
			{
				Selectors: []css_ast.CompoundSelector{
					{
						TypeSelector: &css_ast.NamespacedName{
							Name: css_ast.NameToken{Text: "div"},
						},
						SubclassSelectors: []css_ast.SubclassSelector{
							{Data: &css_ast.SSPseudoClass{Name: "hover"}},
						},
					},
				},
			},
		}

		result := filterMatchingSelectors(complexSelectors, selectors, nil)

		assert.Empty(t, result)
	})

	t.Run("filters mix of matching and non-matching selectors", func(t *testing.T) {
		symbols := []es_ast.Symbol{
			{OriginalName: "used"},
			{OriginalName: "unused"},
		}
		selectors := newUsedSelectors()
		selectors.classes["used"] = true
		selectors.tags["p"] = true

		complexSelectors := []css_ast.ComplexSelector{

			{
				Selectors: []css_ast.CompoundSelector{
					{
						SubclassSelectors: []css_ast.SubclassSelector{
							{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 0}}}},
						},
					},
				},
			},

			{
				Selectors: []css_ast.CompoundSelector{
					{
						SubclassSelectors: []css_ast.SubclassSelector{
							{Data: &css_ast.SSClass{Name: es_ast.LocRef{Ref: es_ast.Ref{InnerIndex: 1}}}},
						},
					},
				},
			},

			{
				Selectors: []css_ast.CompoundSelector{
					{
						TypeSelector: &css_ast.NamespacedName{
							Name: css_ast.NameToken{Text: "p"},
						},
					},
				},
			},
		}

		result := filterMatchingSelectors(complexSelectors, selectors, symbols)

		assert.Len(t, result, 2, "should keep the class-matching and tag-matching selectors")
	})
}

func TestFilterAtRule(t *testing.T) {
	ctx := context.Background()

	t.Run("returns false for nil rules", func(t *testing.T) {
		selectors := newUsedSelectors()

		r := &css_ast.RKnownAt{
			AtToken: "media",
			Rules:   nil,
		}

		assert.False(t, filterAtRule(ctx, r, selectors, nil))
	})

	t.Run("returns false when all inner rules are filtered out", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		r := &css_ast.RKnownAt{
			AtToken: "media",
			Rules: []css_ast.Rule{
				{
					Data: &css_ast.RSelector{
						Selectors: []css_ast.ComplexSelector{
							{
								Selectors: []css_ast.CompoundSelector{
									{
										TypeSelector: &css_ast.NamespacedName{
											Name: css_ast.NameToken{Text: "span"},
										},
									},
								},
							},
						},
						Rules: []css_ast.Rule{
							{Data: &css_ast.RDeclaration{KeyText: "color"}},
						},
					},
				},
			},
		}

		assert.False(t, filterAtRule(ctx, r, selectors, nil))
	})

	t.Run("returns true when some inner rules remain", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		r := &css_ast.RKnownAt{
			AtToken: "media",
			Rules: []css_ast.Rule{
				{
					Data: &css_ast.RSelector{
						Selectors: []css_ast.ComplexSelector{
							{
								Selectors: []css_ast.CompoundSelector{
									{
										TypeSelector: &css_ast.NamespacedName{
											Name: css_ast.NameToken{Text: "div"},
										},
									},
								},
							},
						},
						Rules: []css_ast.Rule{
							{Data: &css_ast.RDeclaration{KeyText: "color"}},
						},
					},
				},
			},
		}

		assert.True(t, filterAtRule(ctx, r, selectors, nil))
		assert.Len(t, r.Rules, 1)
	})

	t.Run("returns true for at-rule with empty but non-nil rules after filtering", func(t *testing.T) {
		selectors := newUsedSelectors()

		r := &css_ast.RKnownAt{
			AtToken: "media",
			Rules: []css_ast.Rule{
				{Data: &css_ast.RDeclaration{KeyText: "color"}},
			},
		}

		assert.True(t, filterAtRule(ctx, r, selectors, nil))
	})
}

func TestShakeCSS(t *testing.T) {
	ctx := context.Background()

	t.Run("removes unused class selectors", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true
		selectors.classes["used"] = true

		css := `div { margin: 0; } .used { color: red; } .unused { color: blue; }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "margin")
		assert.Contains(t, result, "red")
		assert.NotContains(t, result, "blue")
	})

	t.Run("removes unused tag selectors", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `div { color: red; } span { color: blue; }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.NotContains(t, result, "blue")
	})

	t.Run("keeps all rules when all selectors are used", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true
		selectors.tags["span"] = true

		css := `div { color: red; } span { color: blue; }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.Contains(t, result, "blue")
	})

	t.Run("strips interactive properties from output", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `div { color: red; transition: all 0.3s; cursor: pointer; }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.NotContains(t, result, "transition")
		assert.NotContains(t, result, "cursor")
	})

	t.Run("removes hover pseudo-class rules", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `div { color: red; } div:hover { color: blue; }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.NotContains(t, result, "blue")
	})

	t.Run("preserves media queries with matching inner rules", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `@media (max-width: 768px) { div { color: red; } }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.Contains(t, result, "768px")
	})

	t.Run("preserves media queries containing used selectors", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `@media (max-width: 768px) { div { color: red; } }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "red", "used div rule inside media query should remain")
		assert.Contains(t, result, "768px", "media query should be preserved")
	})

	t.Run("handles empty CSS gracefully", func(t *testing.T) {
		selectors := newUsedSelectors()

		result, err := shakeCSS(ctx, "", selectors)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("strips unused CSS variables", func(t *testing.T) {
		selectors := newUsedSelectors()
		selectors.tags["div"] = true

		css := `div { --used: red; --unused: blue; color: var(--used); }`

		result, err := shakeCSS(ctx, css, selectors)

		require.NoError(t, err)
		assert.Contains(t, result, "--used")
		assert.NotContains(t, result, "--unused")
	})
}

func TestBuildStaticScaffoldWithTreeShaking(t *testing.T) {
	ctx := context.Background()

	t.Run("tree-shakes unused CSS when enabled", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: true,
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "used"},
					},
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "Hello"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, `div { margin: 0; } .used { color: red; } .unused { color: blue; }`)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.Contains(t, result, "margin")
		assert.NotContains(t, result, "blue")
		assert.Contains(t, result, `<div class="used">Hello</div>`)
	})

	t.Run("preserves all CSS when tree shaking is disabled", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: false,
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "used"},
					},
				},
			},
		}
		fullCSS := `.used { color: red; } .unused { color: blue; }`

		result, err := builder.BuildStaticScaffold(ctx, tAST, fullCSS)

		require.NoError(t, err)
		assert.Contains(t, result, fullCSS)
	})

	t.Run("safelist classes are preserved during tree shaking", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking:         true,
			CSSTreeShakingSafelist: []string{"dynamic"},
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "static"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, `.static { color: red; } .dynamic { color: green; } .unused { color: blue; }`)

		require.NoError(t, err)
		assert.Contains(t, result, "red", "static class should be preserved")
		assert.Contains(t, result, "green", "safelisted class should be preserved")
		assert.NotContains(t, result, "blue", "unused non-safelisted class should be removed")
	})

	t.Run("returns error for nil AST with tree shaking enabled", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: true,
		})

		result, err := builder.BuildStaticScaffold(ctx, nil, ".test { color: red; }")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil TemplateAST")
		assert.Empty(t, result)
	})

	t.Run("tree shaking with empty CSS produces no style tag", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: true,
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST, "")

		require.NoError(t, err)
		assert.NotContains(t, result, "<style>")
		assert.Contains(t, result, "<div></div>")
	})

	t.Run("tree shaking with multiple root nodes", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: true,
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "header",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "id", Value: "top"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "main",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "content"},
					},
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST,
			`header { padding: 10px; } main { margin: 0; } footer { display: none; }`)

		require.NoError(t, err)
		assert.Contains(t, result, "padding")
		assert.Contains(t, result, "margin")
		assert.NotContains(t, result, "footer", "unused tag selector should be removed")
	})

	t.Run("tree shaking removes hover rules", func(t *testing.T) {
		builder := NewScaffoldBuilder(ScaffoldBuilderConfig{
			CSSTreeShaking: true,
		})
		tAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
				},
			},
		}

		result, err := builder.BuildStaticScaffold(ctx, tAST,
			`button { color: red; } button:hover { color: blue; }`)

		require.NoError(t, err)
		assert.Contains(t, result, "red")
		assert.NotContains(t, result, "blue")
	})
}

func TestNewScaffoldBuilderWithConfig(t *testing.T) {
	t.Run("accepts config parameter", func(t *testing.T) {
		config := ScaffoldBuilderConfig{
			CSSTreeShaking:         true,
			CSSTreeShakingSafelist: []string{"keep-me"},
		}

		builder := NewScaffoldBuilder(config)

		require.NotNil(t, builder)
		inner, ok := builder.(*scaffoldBuilder)
		require.True(t, ok)
		assert.True(t, inner.config.CSSTreeShaking)
		assert.Equal(t, []string{"keep-me"}, inner.config.CSSTreeShakingSafelist)
	})

	t.Run("uses first config when multiple provided", func(t *testing.T) {
		first := ScaffoldBuilderConfig{CSSTreeShaking: true}
		second := ScaffoldBuilderConfig{CSSTreeShaking: false}

		builder := NewScaffoldBuilder(first, second)

		inner, ok := builder.(*scaffoldBuilder)
		require.True(t, ok)
		assert.True(t, inner.config.CSSTreeShaking, "should use the first config")
	})

	t.Run("defaults to tree shaking disabled when no config", func(t *testing.T) {
		builder := NewScaffoldBuilder()

		inner, ok := builder.(*scaffoldBuilder)
		require.True(t, ok)
		assert.False(t, inner.config.CSSTreeShaking)
	})
}
