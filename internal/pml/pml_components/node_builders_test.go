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

package pml_components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewElementNode_SetsRequiredFields(t *testing.T) {
	attrs := []ast_domain.HTMLAttribute{
		{Name: "class", Value: "container"},
		{Name: "id", Value: "main"},
	}
	children := []*ast_domain.TemplateNode{
		{NodeType: ast_domain.NodeText, TextContent: "Hello"},
	}

	result := NewElementNode("div", attrs, children)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, "div", result.TagName)
	assert.Equal(t, attrs, result.Attributes)
	assert.Equal(t, children, result.Children)
	assert.NotNil(t, result.Location)
	assert.NotNil(t, result.NodeRange)
}

func TestNewElementNode_EmptyChildren(t *testing.T) {
	result := NewElementNode("span", []ast_domain.HTMLAttribute{}, nil)

	assert.Equal(t, ast_domain.NodeElement, result.NodeType)
	assert.Equal(t, "span", result.TagName)
	assert.Empty(t, result.Attributes)
	assert.Empty(t, result.Children)
}

func TestNewElementNode_SelfClosingTag(t *testing.T) {
	result := NewElementNode("br", nil, nil)

	assert.Equal(t, "br", result.TagName)
	assert.Empty(t, result.Children)
}

func TestNewFragmentNode_SetsRequiredFields(t *testing.T) {
	children := []*ast_domain.TemplateNode{
		{NodeType: ast_domain.NodeText, TextContent: "Child 1"},
		{NodeType: ast_domain.NodeText, TextContent: "Child 2"},
	}

	result := NewFragmentNode(children)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	assert.Equal(t, children, result.Children)
	assert.Empty(t, result.TagName)
	assert.Empty(t, result.Attributes)
}

func TestNewFragmentNode_EmptyChildren(t *testing.T) {
	result := NewFragmentNode(nil)

	assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	assert.Empty(t, result.Children)
}

func TestNewFragmentNode_SingleChild(t *testing.T) {
	child := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeElement,
		TagName:     "div",
		TextContent: "content",
	}

	result := NewFragmentNode([]*ast_domain.TemplateNode{child})

	require.Len(t, result.Children, 1)
	assert.Equal(t, "div", result.Children[0].TagName)
}

func TestNewRawHTMLNode_SetsRequiredFields(t *testing.T) {
	html := "<!--[if mso]><div><![endif]-->"

	result := NewRawHTMLNode(html)

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
	assert.Equal(t, html, result.TextContent)
	assert.Empty(t, result.TagName)
	assert.Empty(t, result.Children)
}

func TestNewRawHTMLNode_OutlookConditional(t *testing.T) {
	html := "<!--[if mso | IE]><table><tr><td><![endif]-->"

	result := NewRawHTMLNode(html)

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
	assert.Contains(t, result.TextContent, "<!--[if mso")
	assert.Contains(t, result.TextContent, "<![endif]-->")
}

func TestNewRawHTMLNode_EmptyHTML(t *testing.T) {
	result := NewRawHTMLNode("")

	assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
	assert.Equal(t, "", result.TextContent)
}

func TestNewSimpleTextNode_SetsRequiredFields(t *testing.T) {
	text := "Hello World"

	result := NewSimpleTextNode(text)

	assert.Equal(t, ast_domain.NodeText, result.NodeType)
	assert.Equal(t, text, result.TextContent)
	assert.Empty(t, result.TagName)
	assert.Empty(t, result.Children)
	assert.NotNil(t, result.Location)
}

func TestNewSimpleTextNode_EmptyString(t *testing.T) {
	result := NewSimpleTextNode("")

	assert.Equal(t, ast_domain.NodeText, result.NodeType)
	assert.Equal(t, "", result.TextContent)
}

func TestNewSimpleTextNode_SpecialCharacters(t *testing.T) {
	text := "&#8202;"

	result := NewSimpleTextNode(text)

	assert.Equal(t, text, result.TextContent)
}

func TestNewSimpleTextNode_Multiline(t *testing.T) {
	text := "Line 1\nLine 2\nLine 3"

	result := NewSimpleTextNode(text)

	assert.Equal(t, text, result.TextContent)
}

func TestNewHTMLAttribute_SetsRequiredFields(t *testing.T) {
	result := NewHTMLAttribute("class", "container")

	assert.Equal(t, "class", result.Name)
	assert.Equal(t, "container", result.Value)
	assert.NotNil(t, result.Location)
	assert.NotNil(t, result.NameLocation)
	assert.NotNil(t, result.AttributeRange)
}

func TestNewHTMLAttribute_EmptyValue(t *testing.T) {
	result := NewHTMLAttribute("disabled", "")

	assert.Equal(t, "disabled", result.Name)
	assert.Equal(t, "", result.Value)
}

func TestNewHTMLAttribute_StyleAttribute(t *testing.T) {
	style := "color:red;background-color:blue;"

	result := NewHTMLAttribute("style", style)

	assert.Equal(t, "style", result.Name)
	assert.Equal(t, style, result.Value)
}

func TestNewLocation_ReturnsValidLocation(t *testing.T) {
	result := NewLocation()

	assert.NotNil(t, result)

	assert.Equal(t, 0, result.Line)
	assert.Equal(t, 0, result.Column)
}

func TestNewRange_ReturnsValidRange(t *testing.T) {
	result := NewRange()

	assert.NotNil(t, result)

	assert.Equal(t, 0, result.Start.Line)
	assert.Equal(t, 0, result.Start.Column)
	assert.Equal(t, 0, result.End.Line)
	assert.Equal(t, 0, result.End.Column)
}

func TestNodeBuilders_BuildComplexElement(t *testing.T) {

	tdCell := NewElementNode("td", []ast_domain.HTMLAttribute{
		NewHTMLAttribute("class", "cell"),
	}, []*ast_domain.TemplateNode{
		NewSimpleTextNode("Cell Content"),
	})

	trRow := NewElementNode("tr", nil, []*ast_domain.TemplateNode{tdCell})
	tbody := NewElementNode("tbody", nil, []*ast_domain.TemplateNode{trRow})

	table := NewElementNode("table", []ast_domain.HTMLAttribute{
		NewHTMLAttribute("border", "0"),
		NewHTMLAttribute("cellspacing", "0"),
	}, []*ast_domain.TemplateNode{tbody})

	assert.Equal(t, "table", table.TagName)
	require.Len(t, table.Children, 1)
	assert.Equal(t, "tbody", table.Children[0].TagName)
	require.Len(t, table.Children[0].Children, 1)
	assert.Equal(t, "tr", table.Children[0].Children[0].TagName)
}

func TestNodeBuilders_BuildFragmentWithMixedContent(t *testing.T) {

	fragment := NewFragmentNode([]*ast_domain.TemplateNode{
		NewRawHTMLNode("<!--[if mso]>"),
		NewElementNode("div", []ast_domain.HTMLAttribute{
			NewHTMLAttribute("class", "outlook-only"),
		}, nil),
		NewRawHTMLNode("<![endif]-->"),
		NewElementNode("div", []ast_domain.HTMLAttribute{
			NewHTMLAttribute("class", "modern"),
		}, []*ast_domain.TemplateNode{
			NewSimpleTextNode("Modern content"),
		}),
	})

	assert.Equal(t, ast_domain.NodeFragment, fragment.NodeType)
	require.Len(t, fragment.Children, 4)
	assert.Equal(t, ast_domain.NodeRawHTML, fragment.Children[0].NodeType)
	assert.Equal(t, ast_domain.NodeElement, fragment.Children[1].NodeType)
	assert.Equal(t, ast_domain.NodeRawHTML, fragment.Children[2].NodeType)
	assert.Equal(t, ast_domain.NodeElement, fragment.Children[3].NodeType)
}
