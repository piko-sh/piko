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

package premailer

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func getStyleAttr(t *testing.T, node *ast_domain.TemplateNode) string {
	t.Helper()
	require.NotNil(t, node, "Node should not be nil when getting style attribute")
	style, _ := node.GetAttribute("style")
	return style
}

var whitespaceRegex = regexp.MustCompile(`\s+`)

func getHeadStyleContent(t *testing.T, tree *ast_domain.TemplateAST) string {
	t.Helper()
	head := ast_domain.MustQuery(tree, "head")
	if head == nil {
		return ""
	}
	styleNode := head.MustQuery("style")
	if styleNode == nil {
		return ""
	}

	return whitespaceRegex.ReplaceAllString(strings.TrimSpace(styleNode.Text(context.Background())), " ")
}

func TestPremailerTransform(t *testing.T) {
	testCases := []struct {
		name                    string
		htmlInput               string
		cssInput                string
		expectedPStyle          string
		expectedSpanStyle       string
		expectedLeftoverCSS     string
		expectHeadCreated       bool
		expectNoStyleTagRemoved bool
		keepBangImportant       bool
	}{
		{
			name:              "Basic inlining",
			htmlInput:         `<p class="message">Hello</p>`,
			cssInput:          `.message { color: red; }`,
			expectedPStyle:    "color: #ff0000",
			expectedSpanStyle: "",
		},
		{
			name:           "Specificity override",
			htmlInput:      `<p id="main">Hello</p>`,
			cssInput:       `p { color: blue; } #main { color: green; }`,
			expectedPStyle: "color: #008000",
		},
		{
			name:                "Important CSS rule overrides inline style (dual placement)",
			htmlInput:           `<p style="color: blue;">Hello</p>`,
			cssInput:            `p { color: red !important; }`,
			expectedPStyle:      "color: #ff0000",
			expectedLeftoverCSS: "p { color: #ff0000 !important; }",
			keepBangImportant:   true,
		},
		{
			name:           "Normal CSS rule does NOT override important inline style",
			htmlInput:      `<p style="color: blue !important;">Hello</p>`,
			cssInput:       `p { color: red; }`,
			expectedPStyle: "color: #0000ff",
		},
		{
			name:           "Important rule strips !important by default (keepImportant=false)",
			htmlInput:      `<p>Hello</p>`,
			cssInput:       `p { color: red !important; font-size: 14px; }`,
			expectedPStyle: "color: #ff0000; font-size: 14px",
		},
		{
			name:                "Leftover rules are preserved in a new style tag",
			htmlInput:           `<html><head></head><body><a>Link</a></body></html>`,
			cssInput:            `a { color: blue; } a:hover { color: red; }`,
			expectedLeftoverCSS: `a:hover { color: #ff0000; }`,
		},
		{
			name:                "Head is created if it doesn't exist for leftover rules",
			htmlInput:           `<html><body><a>Link</a></body></html>`,
			cssInput:            `a:hover { color: red; }`,
			expectedLeftoverCSS: `a:hover { color: #ff0000; }`,
			expectHeadCreated:   true,
		},
		{
			name:           "Multiple rules are merged correctly",
			htmlInput:      `<p class="bold" id="main">Test</p>`,
			cssInput:       `.bold { font-weight: bold; } #main { font-size: 16px; } p { color: black; }`,
			expectedPStyle: "color: #000000; font-size: 16px; font-weight: bold",
		},
		{
			name:      "No styles in, no changes out",
			htmlInput: `<p>Hello</p>`,
			cssInput:  ``,
		},
		{
			name:                    "Ignores style tag with data-premailer-ignore",
			htmlInput:               `<html><head><style data-premailer="ignore">p { font-style: italic; }</style></head><body><p>Hello</p></body></html>`,
			cssInput:                `p { color: red; }`,
			expectedPStyle:          "color: #ff0000",
			expectNoStyleTagRemoved: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			var fullHTML string
			styleTagHTML := ""
			if tc.cssInput != "" {
				styleTagHTML = "<style>" + tc.cssInput + "</style>"
			}

			if strings.Contains(tc.htmlInput, "</head>") {
				fullHTML = strings.Replace(tc.htmlInput, "</head>", styleTagHTML+"</head>", 1)
			} else {
				fullHTML = styleTagHTML + tc.htmlInput
			}

			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.keepBangImportant {
				premailer = New(tree, WithKeepBangImportant(true))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			pNode := ast_domain.MustQuery(transformedTree, "p")
			if tc.expectedPStyle != "" {
				assert.Equal(t, tc.expectedPStyle, getStyleAttr(t, pNode))
			} else if pNode != nil {
				_, hasStyle := pNode.GetAttribute("style")
				assert.False(t, hasStyle, "P node should not have a style attribute")
			}

			spanNode := ast_domain.MustQuery(transformedTree, "span")
			if tc.expectedSpanStyle != "" {
				assert.Equal(t, tc.expectedSpanStyle, getStyleAttr(t, spanNode))
			}

			if tc.expectedLeftoverCSS != "" {
				assert.Equal(t, tc.expectedLeftoverCSS, getHeadStyleContent(t, transformedTree))
			}

			if tc.expectHeadCreated {
				assert.NotNil(t, ast_domain.MustQuery(transformedTree, "head"), "A <head> tag should have been created")
			}

			if tc.expectNoStyleTagRemoved {
				ignoredStyle := ast_domain.MustQuery(transformedTree, `style[data-premailer="ignore"]`)
				require.NotNil(t, ignoredStyle, "The ignored style tag should be preserved")
				assert.Equal(t, "p { font-style: italic; }", strings.TrimSpace(ignoredStyle.Text(context.Background())))
			} else {

				styleTags := ast_domain.MustQuery(transformedTree, "style")
				if tc.expectedLeftoverCSS == "" && !tc.keepBangImportant {

					assert.Nil(t, styleTags, "No style tags should remain when no leftover CSS and keepBangImportant=false")
				}
			}
		})
	}
}

func TestKeepBangImportantOption(t *testing.T) {
	testCases := []struct {
		name              string
		htmlInput         string
		cssInput          string
		expectedPStyle    string
		expectedStyleCSS  string
		keepBangImportant bool
		expectStyleBlock  bool
	}{
		{
			name:              "KeepBangImportant=false removes rule from style block",
			htmlInput:         `<p>Hello</p>`,
			cssInput:          `p { color: red !important; font-size: 14px; }`,
			keepBangImportant: false,
			expectedPStyle:    "color: #ff0000; font-size: 14px",
			expectStyleBlock:  false,
		},
		{
			name:              "KeepBangImportant=true keeps rule in style block with dual placement",
			htmlInput:         `<p>Hello</p>`,
			cssInput:          `p { color: red !important; font-size: 14px; }`,
			keepBangImportant: true,
			expectedPStyle:    "color: #ff0000; font-size: 14px",
			expectStyleBlock:  true,
			expectedStyleCSS:  "p {\n  color: #ff0000 !important;\n  font-size: 14px;\n}",
		},
		{
			name:              "KeepBangImportant=true with multiple !important properties",
			htmlInput:         `<p>Hello</p>`,
			cssInput:          `p { color: red !important; font-size: 14px !important; margin: 0; }`,
			keepBangImportant: true,
			expectedPStyle:    "color: #ff0000; font-size: 14px; margin-bottom: 0; margin-left: 0; margin-right: 0; margin-top: 0",
			expectStyleBlock:  true,
			expectedStyleCSS:  "p {\n  color: #ff0000 !important;\n  font-size: 14px !important;\n  margin-top: 0;\n  margin-right: 0;\n  margin-bottom: 0;\n  margin-left: 0;\n}",
		},
		{
			name:              "KeepBangImportant=false with multiple !important",
			htmlInput:         `<p>Hello</p>`,
			cssInput:          `p { color: red !important; font-size: 14px !important; }`,
			keepBangImportant: false,
			expectedPStyle:    "color: #ff0000; font-size: 14px",
			expectStyleBlock:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<style>" + tc.cssInput + "</style>" + tc.htmlInput
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.keepBangImportant {
				premailer = New(tree, WithKeepBangImportant(true))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			pNode := ast_domain.MustQuery(transformedTree, "p")
			require.NotNil(t, pNode, "P node should exist")
			assert.Equal(t, tc.expectedPStyle, getStyleAttr(t, pNode), "Inline style mismatch")

			styleNodes, _ := ast_domain.QueryAll(transformedTree, "style", "test")
			if tc.expectStyleBlock {
				require.NotEmpty(t, styleNodes, "Expected <style> block to exist")
				styleContent := styleNodes[0].Text(context.Background())

				assert.Contains(t, styleContent, "!important", "Style block should contain !important")
				assert.Contains(t, styleContent, "p {", "Style block should contain p selector")
			} else {
				assert.Empty(t, styleNodes, "Expected no <style> block (rule should be fully inlined)")
			}
		})
	}
}

func TestRemoveClassesOption(t *testing.T) {
	testCases := []struct {
		name          string
		htmlInput     string
		cssInput      string
		expectedStyle string
		removeClasses bool
		expectClasses bool
	}{
		{
			name:          "RemoveClasses=false (default) keeps class attributes",
			htmlInput:     `<div class="container"><p class="text">Hello</p></div>`,
			cssInput:      `.container { width: 600px; } .text { color: blue; }`,
			removeClasses: false,
			expectClasses: true,
			expectedStyle: "color: #0000ff",
		},
		{
			name:          "RemoveClasses=true removes all class attributes",
			htmlInput:     `<div class="container"><p class="text">Hello</p></div>`,
			cssInput:      `.container { width: 600px; } .text { color: blue; }`,
			removeClasses: true,
			expectClasses: false,
			expectedStyle: "color: #0000ff",
		},
		{
			name:          "RemoveClasses=true removes classes but keeps styles",
			htmlInput:     `<div class="box red bold"><span class="small">Text</span></div>`,
			cssInput:      `.box { padding: 10px; } .red { color: red; } .bold { font-weight: bold; }`,
			removeClasses: true,
			expectClasses: false,
			expectedStyle: "color: #ff0000; font-weight: bold; padding-bottom: 10px; padding-left: 10px; padding-right: 10px; padding-top: 10px",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<style>" + tc.cssInput + "</style>" + tc.htmlInput
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.removeClasses {
				premailer = New(tree, WithRemoveClasses(true))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			divNode := ast_domain.MustQuery(transformedTree, "div")
			require.NotNil(t, divNode, "Div node should exist")

			_, hasClassOnDiv := divNode.GetAttribute("class")
			if tc.expectClasses {
				assert.True(t, hasClassOnDiv, "Class attribute should be present on div")
			} else {
				assert.False(t, hasClassOnDiv, "Class attribute should be removed from div")
			}

			if tc.expectedStyle != "" {

				if strings.Contains(tc.htmlInput, "<p class=\"text\">") {
					pNode := ast_domain.MustQuery(transformedTree, "p")
					require.NotNil(t, pNode, "P node should exist")
					assert.Equal(t, tc.expectedStyle, getStyleAttr(t, pNode))
				} else {

					assert.Equal(t, tc.expectedStyle, getStyleAttr(t, divNode))
				}
			}
		})
	}
}

func TestRemoveIDsOption(t *testing.T) {
	testCases := []struct {
		name          string
		htmlInput     string
		cssInput      string
		expectedStyle string
		removeIDs     bool
		expectIDs     bool
	}{
		{
			name:          "RemoveIDs=false (default) keeps id attributes",
			htmlInput:     `<div id="container"><p id="main-text">Hello</p></div>`,
			cssInput:      `#container { width: 600px; } #main-text { color: blue; }`,
			removeIDs:     false,
			expectIDs:     true,
			expectedStyle: "color: #0000ff",
		},
		{
			name:          "RemoveIDs=true removes all id attributes",
			htmlInput:     `<div id="container"><p id="main-text">Hello</p></div>`,
			cssInput:      `#container { width: 600px; } #main-text { color: blue; }`,
			removeIDs:     true,
			expectIDs:     false,
			expectedStyle: "color: #0000ff",
		},
		{
			name:          "RemoveIDs=true removes IDs but keeps styles",
			htmlInput:     `<div id="wrapper"><span id="highlight">Text</span></div>`,
			cssInput:      `#wrapper { padding: 10px; } #highlight { color: red; font-weight: bold; }`,
			removeIDs:     true,
			expectIDs:     false,
			expectedStyle: "color: #ff0000; font-weight: bold",
		},
		{
			name:          "RemoveIDs=true works with mixed selectors",
			htmlInput:     `<div id="box" class="container"><p id="text" class="paragraph">Content</p></div>`,
			cssInput:      `#box { margin: 5px; } .container { padding: 10px; } #text { color: green; }`,
			removeIDs:     true,
			expectIDs:     false,
			expectedStyle: "color: #008000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<style>" + tc.cssInput + "</style>" + tc.htmlInput
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.removeIDs {
				premailer = New(tree, WithRemoveIDs(true))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			divNode := ast_domain.MustQuery(transformedTree, "div")
			require.NotNil(t, divNode, "Div node should exist")

			_, hasIDOnDiv := divNode.GetAttribute("id")
			if tc.expectIDs {
				assert.True(t, hasIDOnDiv, "ID attribute should be present on div")
			} else {
				assert.False(t, hasIDOnDiv, "ID attribute should be removed from div")
			}

			if tc.expectedStyle != "" {

				if strings.Contains(tc.htmlInput, "<p") {
					pNode := ast_domain.MustQuery(transformedTree, "p")
					require.NotNil(t, pNode, "P node should exist")
					assert.Equal(t, tc.expectedStyle, getStyleAttr(t, pNode))

					if tc.removeIDs {
						_, hasIDOnP := pNode.GetAttribute("id")
						assert.False(t, hasIDOnP, "ID attribute should be removed from p")
					}
				} else if strings.Contains(tc.htmlInput, "<span") {

					spanNode := ast_domain.MustQuery(transformedTree, "span")
					require.NotNil(t, spanNode, "Span node should exist")
					assert.Equal(t, tc.expectedStyle, getStyleAttr(t, spanNode))

					if tc.removeIDs {
						_, hasIDOnSpan := spanNode.GetAttribute("id")
						assert.False(t, hasIDOnSpan, "ID attribute should be removed from span")
					}
				} else {

					assert.Equal(t, tc.expectedStyle, getStyleAttr(t, divNode))
				}
			}
		})
	}
}

func TestStructuralPseudoClasses(t *testing.T) {
	testCases := []struct {
		name                string
		htmlInput           string
		cssInput            string
		expectedInlineStyle string
		expectedLeftover    bool
	}{
		{
			name:                "first-child is inlined",
			htmlInput:           `<div><p>First</p><p>Second</p></div>`,
			cssInput:            `p:first-child { color: red; }`,
			expectedInlineStyle: "color: #ff0000",
			expectedLeftover:    false,
		},
		{
			name:                "last-child is inlined",
			htmlInput:           `<div><p>First</p><p>Last</p></div>`,
			cssInput:            `p:last-child { color: blue; }`,
			expectedInlineStyle: "color: #0000ff",
			expectedLeftover:    false,
		},
		{
			name:                "only-child is inlined (now natively supported)",
			htmlInput:           `<div><p>Only</p></div>`,
			cssInput:            `p:only-child { font-weight: bold; }`,
			expectedInlineStyle: "font-weight: bold",
			expectedLeftover:    false,
		},
		{
			name:             "hover is NOT inlined (kept as leftover)",
			htmlInput:        `<a href="#">Link</a>`,
			cssInput:         `a:hover { color: #ff0000; }`,
			expectedLeftover: true,
		},
		{
			name:             "active is NOT inlined (kept as leftover)",
			htmlInput:        `<button>Click</button>`,
			cssInput:         `button:active { background: #0000ff; }`,
			expectedLeftover: true,
		},
		{
			name:                "first-of-type is inlined",
			htmlInput:           `<div><span>A</span><p>First P</p><p>Second P</p></div>`,
			cssInput:            `p:first-of-type { color: green; }`,
			expectedInlineStyle: "color: #008000",
			expectedLeftover:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><head><style>" + tc.cssInput + "</style></head><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			premailer := New(tree)

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			if tc.expectedInlineStyle != "" {

				var targetNode *ast_domain.TemplateNode

				selector := strings.TrimSpace(tc.cssInput[:strings.Index(tc.cssInput, "{")])

				targetNode, _ = ast_domain.Query(transformedTree, selector)

				if targetNode == nil {
					if strings.Contains(tc.htmlInput, "<p>") {
						targetNode = ast_domain.MustQuery(transformedTree, "p")
					} else if strings.Contains(tc.htmlInput, "<a") {
						targetNode = ast_domain.MustQuery(transformedTree, "a")
					} else if strings.Contains(tc.htmlInput, "<button") {
						targetNode = ast_domain.MustQuery(transformedTree, "button")
					}
				}

				require.NotNil(t, targetNode, "Target node should exist")
				assert.Equal(t, tc.expectedInlineStyle, getStyleAttr(t, targetNode))
			}

			headContent := getHeadStyleContent(t, transformedTree)
			if tc.expectedLeftover {
				assert.NotEmpty(t, headContent, "Leftover styles should exist in head")
			} else {

				if headContent != "" {

					assert.NotContains(t, headContent, tc.expectedInlineStyle)
				}
			}
		})
	}
}

func TestUniversalSelector(t *testing.T) {
	testCases := []struct {
		name             string
		htmlInput        string
		cssInput         string
		expectedLeftover bool
		expectInline     bool
	}{
		{
			name:             "Universal selector is kept as leftover",
			htmlInput:        `<div><p>Text</p><span>More</span></div>`,
			cssInput:         `* { margin: 0; padding: 0; }`,
			expectedLeftover: true,
			expectInline:     false,
		},
		{
			name:             "Universal selector with class is kept as leftover",
			htmlInput:        `<div class="container"><p>Text</p></div>`,
			cssInput:         `*.container { border: 1px solid red; }`,
			expectedLeftover: true,
			expectInline:     false,
		},
		{
			name:             "Regular selectors are still inlined",
			htmlInput:        `<p>Hello</p>`,
			cssInput:         `p { color: blue; } * { margin: 0; }`,
			expectedLeftover: true,
			expectInline:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><head><style>" + tc.cssInput + "</style></head><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			premailer := New(tree)

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			headContent := getHeadStyleContent(t, transformedTree)
			if tc.expectedLeftover {
				assert.NotEmpty(t, headContent, "Universal selector should be kept as leftover")
				assert.Contains(t, headContent, "*", "Leftover should contain universal selector")
			}

			if tc.expectInline {
				pNode := ast_domain.MustQuery(transformedTree, "p")
				if pNode != nil {
					style := getStyleAttr(t, pNode)
					assert.NotEmpty(t, style, "Regular selector should still be inlined")
				}
			}
		})
	}
}

func TestMakeLeftoverImportantOption(t *testing.T) {
	testCases := []struct {
		name                  string
		htmlInput             string
		cssInput              string
		makeLeftoverImportant bool
		expectedWithImportant bool
	}{
		{
			name:                  "MakeLeftoverImportant=false (default) keeps rules as-is",
			htmlInput:             `<a href="#">Link</a>`,
			cssInput:              `a { color: blue; } a:hover { color: red; }`,
			makeLeftoverImportant: false,
			expectedWithImportant: false,
		},
		{
			name:                  "MakeLeftoverImportant=true adds !important to pseudo-class",
			htmlInput:             `<a href="#">Link</a>`,
			cssInput:              `a { color: blue; } a:hover { color: red; }`,
			makeLeftoverImportant: true,
			expectedWithImportant: true,
		},
		{
			name:      "MakeLeftoverImportant=true adds !important to media query rules",
			htmlInput: `<div class="container">Content</div>`,
			cssInput: `.container { width: 600px; }
@media only screen and (max-width: 600px) {
	.container { width: 100%; }
}`,
			makeLeftoverImportant: true,
			expectedWithImportant: true,
		},
		{
			name:      "MakeLeftoverImportant=false keeps media query rules without !important",
			htmlInput: `<div class="container">Content</div>`,
			cssInput: `.container { width: 600px; }
@media only screen and (max-width: 600px) {
	.container { width: 100%; }
}`,
			makeLeftoverImportant: false,
			expectedWithImportant: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><head><style>" + tc.cssInput + "</style></head><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.makeLeftoverImportant {
				premailer = New(tree, WithMakeLeftoverImportant(true))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			headContent := getHeadStyleContent(t, transformedTree)
			if tc.expectedWithImportant {
				assert.Contains(t, headContent, "!important", "Leftover rules should contain !important")
			} else {

				if headContent != "" {
					assert.NotContains(t, headContent, "!important", "Leftover rules should NOT contain !important")
				}
			}
		})
	}
}

func TestRemoveComments(t *testing.T) {
	testCases := []struct {
		name               string
		htmlInput          string
		cssInput           string
		commentTextToCheck string
		expectCommentsGone bool
	}{
		{
			name:               "HTML comments are removed",
			htmlInput:          `<!-- This is a comment --><p>Text</p><!-- Another comment -->`,
			cssInput:           `p { color: blue; }`,
			expectCommentsGone: true,
			commentTextToCheck: "This is a comment",
		},
		{
			name:               "Nested comments in elements are removed",
			htmlInput:          `<div><!-- Comment inside div --><p>Text</p></div>`,
			cssInput:           `p { color: red; }`,
			expectCommentsGone: true,
			commentTextToCheck: "Comment inside div",
		},
		{
			name:               "Multiple comments are all removed",
			htmlInput:          `<!-- Start --><!-- Middle --><!-- End --><p>Content</p>`,
			cssInput:           `p { font-size: 14px; }`,
			expectCommentsGone: true,
			commentTextToCheck: "Start",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><head><style>" + tc.cssInput + "</style></head><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			premailer := New(tree)

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			if tc.expectCommentsGone {

				foundComment := false
				transformedTree.Walk(func(node *ast_domain.TemplateNode) bool {
					if node.NodeType == ast_domain.NodeComment {
						foundComment = true
						return false
					}
					return true
				})
				assert.False(t, foundComment, "Comments should have been removed from the tree")

				renderedHTML := renderTreeToHTML(transformedTree)
				assert.NotContains(t, renderedHTML, tc.commentTextToCheck, "Rendered HTML should not contain comment text")
			}
		})
	}
}

func TestRemoveScripts(t *testing.T) {
	testCases := []struct {
		name              string
		htmlInput         string
		cssInput          string
		scriptTextToCheck string
		expectScriptsGone bool
	}{
		{
			name:              "Script tags are removed",
			htmlInput:         `<script>console.log('test');</script><p>Text</p>`,
			cssInput:          `p { color: blue; }`,
			expectScriptsGone: true,
			scriptTextToCheck: "console.log",
		},
		{
			name:              "Multiple script tags are removed",
			htmlInput:         `<script src="external.js"></script><p>Content</p><script>alert('hello');</script>`,
			cssInput:          `p { font-size: 14px; }`,
			expectScriptsGone: true,
			scriptTextToCheck: "alert",
		},
		{
			name:              "Script tags with attributes are removed",
			htmlInput:         `<script type="text/javascript" async defer>var x = 1;</script><div>Content</div>`,
			cssInput:          `div { padding: 10px; }`,
			expectScriptsGone: true,
			scriptTextToCheck: "var x",
		},
		{
			name:              "Nested script tags are removed",
			htmlInput:         `<div><script>document.write('test');</script><p>Text</p></div>`,
			cssInput:          `p { color: red; }`,
			expectScriptsGone: true,
			scriptTextToCheck: "document.write",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><head><style>" + tc.cssInput + "</style></head><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			premailer := New(tree)

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			if tc.expectScriptsGone {

				foundScript := false
				transformedTree.Walk(func(node *ast_domain.TemplateNode) bool {
					if node.NodeType == ast_domain.NodeElement && node.TagName == "script" {
						foundScript = true
						return false
					}
					return true
				})
				assert.False(t, foundScript, "Script tags should have been removed from the tree")

				renderedHTML := renderTreeToHTML(transformedTree)
				assert.NotContains(t, renderedHTML, tc.scriptTextToCheck, "Rendered HTML should not contain script content")
				assert.NotContains(t, renderedHTML, "<script", "Rendered HTML should not contain <script tags")
			}
		})
	}
}

func TestLinkQueryParams(t *testing.T) {
	testCases := []struct {
		queryParams   map[string]string
		expectedHrefs map[string]string
		name          string
		htmlInput     string
	}{
		{
			name:      "Basic parameter appending to simple URL",
			htmlInput: `<a href="https://example.com/product">Buy Now</a>`,
			queryParams: map[string]string{
				"utm_source":   "newsletter",
				"utm_campaign": "october_promo",
			},
			expectedHrefs: map[string]string{
				"Buy Now": "https://example.com/product?utm_campaign=october_promo&utm_source=newsletter",
			},
		},
		{
			name:      "Intelligent merging with existing query parameters",
			htmlInput: `<a href="https://example.com/product?id=123">Product</a>`,
			queryParams: map[string]string{
				"utm_source": "email",
			},
			expectedHrefs: map[string]string{
				"Product": "https://example.com/product?id=123&utm_source=email",
			},
		},
		{
			name: "Multiple links get parameters appended",
			htmlInput: `
				<a href="https://example.com/page1">Link 1</a>
				<a href="https://example.com/page2">Link 2</a>
			`,
			queryParams: map[string]string{
				"source": "test",
			},
			expectedHrefs: map[string]string{
				"Link 1": "https://example.com/page1?source=test",
				"Link 2": "https://example.com/page2?source=test",
			},
		},
		{
			name: "Skips mailto: links",
			htmlInput: `
				<a href="mailto:test@example.com">Email</a>
				<a href="https://example.com">Website</a>
			`,
			queryParams: map[string]string{
				"source": "newsletter",
			},
			expectedHrefs: map[string]string{
				"Email":   "mailto:test@example.com",
				"Website": "https://example.com?source=newsletter",
			},
		},
		{
			name: "Skips tel: links",
			htmlInput: `
				<a href="tel:+1234567890">Call</a>
				<a href="https://example.com">Visit</a>
			`,
			queryParams: map[string]string{
				"campaign": "ads",
			},
			expectedHrefs: map[string]string{
				"Call":  "tel:+1234567890",
				"Visit": "https://example.com?campaign=ads",
			},
		},
		{
			name: "Skips javascript: pseudo-protocol",
			htmlInput: `
				<a href="javascript:alert('test')">Alert</a>
				<a href="https://example.com">Link</a>
			`,
			queryParams: map[string]string{
				"test": "value",
			},
			expectedHrefs: map[string]string{
				"Alert": "javascript:alert('test')",
				"Link":  "https://example.com?test=value",
			},
		},
		{
			name: "Skips anchor-only links",
			htmlInput: `
				<a href="#section">Section</a>
				<a href="https://example.com">External</a>
			`,
			queryParams: map[string]string{
				"ref": "email",
			},
			expectedHrefs: map[string]string{
				"Section":  "#section",
				"External": "https://example.com?ref=email",
			},
		},
		{
			name:      "Handles relative URLs",
			htmlInput: `<a href="/products/widget">Widget</a>`,
			queryParams: map[string]string{
				"source": "campaign",
			},
			expectedHrefs: map[string]string{
				"Widget": "/products/widget?source=campaign",
			},
		},
		{
			name:      "Handles links with existing complex query strings",
			htmlInput: `<a href="https://example.com/search?q=test&category=electronics&sort=price">Search</a>`,
			queryParams: map[string]string{
				"utm_source": "newsletter",
				"utm_medium": "email",
			},
			expectedHrefs: map[string]string{
				"Search": "https://example.com/search?category=electronics&q=test&sort=price&utm_medium=email&utm_source=newsletter",
			},
		},
		{
			name:        "Does nothing when no parameters configured",
			htmlInput:   `<a href="https://example.com">Link</a>`,
			queryParams: nil,
			expectedHrefs: map[string]string{
				"Link": "https://example.com",
			},
		},
		{
			name:        "Does nothing when empty parameters map",
			htmlInput:   `<a href="https://example.com">Link</a>`,
			queryParams: map[string]string{},
			expectedHrefs: map[string]string{
				"Link": "https://example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fullHTML := "<html><body>" + tc.htmlInput + "</body></html>"
			tree, err := ast_domain.Parse(context.Background(), fullHTML, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			var premailer *Premailer
			if tc.queryParams != nil {
				premailer = New(tree, WithLinkQueryParams(tc.queryParams))
			} else {
				premailer = New(tree)
			}

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Premailer transformation should not fail")

			for linkText, expectedHref := range tc.expectedHrefs {

				var foundLink *ast_domain.TemplateNode
				transformedTree.Walk(func(node *ast_domain.TemplateNode) bool {
					if node.NodeType == ast_domain.NodeElement && node.TagName == "a" {

						if len(node.Children) > 0 && node.Children[0].NodeType == ast_domain.NodeText {
							if strings.TrimSpace(node.Children[0].TextContent) == linkText {
								foundLink = node
								return false
							}
						}
					}
					return true
				})

				require.NotNil(t, foundLink, "Should find link with text: %s", linkText)
				actualHref, exists := foundLink.GetAttribute("href")
				assert.True(t, exists, "Link should have href attribute")
				assert.Equal(t, expectedHref, actualHref, "Link href should match expected for: %s", linkText)
			}
		})
	}
}
