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

package pikotest_domain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pikotest/pikotest_domain"
	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/templater/templater_dto"
)

func renderView(t *testing.T, nodes []*ast_domain.TemplateNode, metadata templater_dto.InternalMetadata) *pikotest_domain.TestView {
	t.Helper()

	buildAST := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*pikotest_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
			RootNodes: nodes,
		}, metadata, nil
	}

	tester := pikotest_domain.NewComponentTester(t, buildAST)
	request := pikotest_domain.NewRequest("GET", "/test").Build(context.Background())
	return tester.Render(request, nil)
}

func emptyMeta() templater_dto.InternalMetadata {
	return templater_dto.InternalMetadata{}
}

func TestTestView_State(t *testing.T) {

	view := renderView(t, nil, emptyMeta())
	assert.Nil(t, view.State())
}

func TestTestView_AssertState(t *testing.T) {
	view := renderView(t, nil, emptyMeta())

	var called bool
	view.AssertState(func(state any) {
		called = true
	})
	assert.True(t, called)
}

func TestTestView_Metadata(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			Title:       "Test Title",
			Description: "A description",
			Language:    "en",
		},
	}
	view := renderView(t, nil, meta)

	assert.Equal(t, "Test Title", view.Metadata().Title)
	assert.Equal(t, "A description", view.Metadata().Description)
	assert.Equal(t, "en", view.Metadata().Language)
}

func TestTestView_AST(t *testing.T) {
	h1 := makeElementNode("h1")
	view := renderView(t, []*ast_domain.TemplateNode{h1}, emptyMeta())

	ast := view.AST()
	require.NotNil(t, ast)
	require.Len(t, ast.RootNodes, 1)
	assert.Equal(t, "h1", ast.RootNodes[0].TagName)
}

func TestTestView_AssertTitle(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{name: "matching title", title: "My Page", expected: "My Page"},
		{name: "empty title", title: "", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Title: tc.title},
			}
			view := renderView(t, nil, meta)
			view.AssertTitle(tc.expected)
		})
	}
}

func TestTestView_AssertStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected int
	}{
		{name: "explicit 200", status: 200, expected: 200},
		{name: "zero means 200", status: 0, expected: 200},
		{name: "404 status", status: 404, expected: 404},
		{name: "500 status", status: 500, expected: 500},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			meta := templater_dto.InternalMetadata{
				Metadata: templater_dto.Metadata{Status: tc.status},
			}
			view := renderView(t, nil, meta)
			view.AssertStatusCode(tc.expected)
		})
	}
}

func TestTestView_AssertDefaultStatusCode(t *testing.T) {
	view := renderView(t, nil, emptyMeta())
	view.AssertDefaultStatusCode()
}

func TestTestView_AssertDescription(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{Description: "A test page"},
	}
	view := renderView(t, nil, meta)
	view.AssertDescription("A test page")
}

func TestTestView_AssertHasMetaTag(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			MetaTags: []templater_dto.MetaTag{
				{Name: "author", Content: "Piko"},
				{Name: "robots", Content: "noindex"},
			},
		},
	}
	view := renderView(t, nil, meta)
	view.AssertHasMetaTag("author", "Piko")
	view.AssertHasMetaTag("robots", "noindex")
}

func TestTestView_AssertHasOGTag(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{
			OGTags: []templater_dto.OGTag{
				{Property: "og:title", Content: "My Page"},
				{Property: "og:type", Content: "website"},
			},
		},
	}
	view := renderView(t, nil, meta)
	view.AssertHasOGTag("og:title", "My Page")
	view.AssertHasOGTag("og:type", "website")
}

func TestTestView_AssertClientRedirect(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{ClientRedirect: "/dashboard"},
	}
	view := renderView(t, nil, meta)
	view.AssertClientRedirect("/dashboard")
}

func TestTestView_AssertServerRedirect(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{ServerRedirect: "/login"},
	}
	view := renderView(t, nil, meta)
	view.AssertServerRedirect("/login")
}

func TestTestView_AssertLanguage(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{Language: "fr"},
	}
	view := renderView(t, nil, meta)
	view.AssertLanguage("fr")
}

func TestTestView_AssertCanonicalURL(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		Metadata: templater_dto.Metadata{CanonicalURL: "https://example.com/page"},
	}
	view := renderView(t, nil, meta)
	view.AssertCanonicalURL("https://example.com/page")
}

func TestTestView_AssertJSScriptURLs(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		JSScriptMetas: []templater_dto.JSScriptMeta{
			{URL: "/js/app.js"},
			{URL: "/js/vendor.js"},
		},
	}
	view := renderView(t, nil, meta)
	view.AssertJSScriptURLs([]string{"/js/app.js", "/js/vendor.js"})
}

func TestTestView_AssertHasJSScript(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		JSScriptMetas: []templater_dto.JSScriptMeta{
			{URL: "/js/app.js"},
		},
	}
	view := renderView(t, nil, meta)
	view.AssertHasJSScript()
}

func TestTestView_AssertNoJSScript(t *testing.T) {
	view := renderView(t, nil, emptyMeta())
	view.AssertNoJSScript()
}

func TestTestView_AssertJSScriptURLContains(t *testing.T) {
	meta := templater_dto.InternalMetadata{
		JSScriptMetas: []templater_dto.JSScriptMeta{
			{URL: "/js/app.abc123.js"},
		},
	}
	view := renderView(t, nil, meta)
	view.AssertJSScriptURLContains("abc123")
}

func TestTestView_QueryAST(t *testing.T) {
	h1 := makeElementWithChildren("h1", makeTextNode("Title"))
	p := makeElementWithChildren("p", makeTextNode("Body"))
	view := renderView(t, []*ast_domain.TemplateNode{h1, p}, emptyMeta())

	result := view.QueryAST("h1")
	require.Equal(t, 1, result.Len())
	result.HasText("Title")
}

func TestTestView_QueryAST_NoMatch(t *testing.T) {
	h1 := makeElementNode("h1")
	view := renderView(t, []*ast_domain.TemplateNode{h1}, emptyMeta())

	result := view.QueryAST("h2")
	assert.Equal(t, 0, result.Len())
}
