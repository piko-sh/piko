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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestApplyAttributes(t *testing.T) {
	testCases := []struct {
		styleMap           map[string]property
		expectedAttrs      map[string]string
		name               string
		html               string
		existingAttrToKeep string
		unexpectedAttrs    []string
	}{
		{
			name:     "Applies width to table",
			html:     "<table></table>",
			styleMap: map[string]property{"width": {value: "100px"}},
			expectedAttrs: map[string]string{
				"width": "100",
			},
		},
		{
			name:     "Applies height with percentage to img",
			html:     "<img>",
			styleMap: map[string]property{"height": {value: "50%"}},
			expectedAttrs: map[string]string{
				"height": "50%",
			},
		},
		{
			name:     "Applies bgcolor to td",
			html:     "<td></td>",
			styleMap: map[string]property{"background-color": {value: "#FFFFFF"}},
			expectedAttrs: map[string]string{
				"bgcolor": "#FFFFFF",
			},
		},
		{
			name:     "Applies bgcolor from background shorthand",
			html:     "<table></table>",
			styleMap: map[string]property{"background": {value: "#DDD url(bg.png) no-repeat"}},
			expectedAttrs: map[string]string{
				"bgcolor": "#DDD",
			},
		},
		{
			name:     "Applies align and valign to th",
			html:     "<th></th>",
			styleMap: map[string]property{"text-align": {value: "center"}, "vertical-align": {value: "top"}},
			expectedAttrs: map[string]string{
				"align":  "center",
				"valign": "top",
			},
		},
		{
			name:     "Applies table attributes from zero-value styles",
			html:     "<table></table>",
			styleMap: map[string]property{"border": {value: "0"}, "border-spacing": {value: "0px"}, "padding": {value: "0"}},
			expectedAttrs: map[string]string{
				"border":      "0",
				"cellspacing": "0",
				"cellpadding": "0",
			},
		},
		{
			name:            "Does not apply bgcolor to a p tag",
			html:            "<p></p>",
			styleMap:        map[string]property{"background-color": {value: "red"}},
			unexpectedAttrs: []string{"bgcolor"},
		},
		{
			name:            "Does not map non-zero padding to cellpadding",
			html:            "<table></table>",
			styleMap:        map[string]property{"padding": {value: "10px"}},
			unexpectedAttrs: []string{"cellpadding"},
		},
		{
			name:               "Does not overwrite existing attributes",
			html:               `<table width="200" align="right"></table>`,
			styleMap:           map[string]property{"width": {value: "100px"}, "text-align": {value: "center"}},
			existingAttrToKeep: "width",
		},
		{
			name:     "Handles !important in width value",
			html:     "<img>",
			styleMap: map[string]property{"width": {value: "50px !important"}},
			expectedAttrs: map[string]string{
				"width": "50",
			},
		},
		{
			name:     "Handles zero value for width",
			html:     "<td></td>",
			styleMap: map[string]property{"width": {value: "0"}},
			expectedAttrs: map[string]string{
				"width": "0",
			},
		},
		{
			name:     "Applies bgcolor from background shorthand with rgb color",
			html:     "<td></td>",
			styleMap: map[string]property{"background": {value: "rgb(255,0,0) url(bg.png)"}},
			expectedAttrs: map[string]string{
				"bgcolor": "rgb(255,0,0)",
			},
		},
		{
			name:     "Applies bgcolor from background shorthand with color name",
			html:     "<table></table>",
			styleMap: map[string]property{"background": {value: "red url(bg.png) no-repeat"}},
			expectedAttrs: map[string]string{
				"bgcolor": "red",
			},
		},
		{
			name:     "Handles height with !important",
			html:     "<img>",
			styleMap: map[string]property{"height": {value: "100px !important"}},
			expectedAttrs: map[string]string{
				"height": "100",
			},
		},
		{
			name:     "Applies border=0 for border:none",
			html:     "<table></table>",
			styleMap: map[string]property{"border": {value: "none"}},
			expectedAttrs: map[string]string{
				"border": "0",
			},
		},
		{
			name:            "Does not apply non-zero border",
			html:            "<table></table>",
			styleMap:        map[string]property{"border": {value: "1px solid black"}},
			unexpectedAttrs: []string{"border"},
		},
		{
			name:     "Applies cellspacing from border-spacing",
			html:     "<table></table>",
			styleMap: map[string]property{"border-spacing": {value: "0"}},
			expectedAttrs: map[string]string{
				"cellspacing": "0",
			},
		},
		{
			name:     "Applies width with percentage and !important",
			html:     "<td></td>",
			styleMap: map[string]property{"width": {value: "50% !important"}},
			expectedAttrs: map[string]string{
				"width": "50%",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tree, err := ast_domain.Parse(context.Background(), tc.html, "test.html", nil)
			require.NoError(t, err)
			require.NotEmpty(t, tree.RootNodes, "Expected at least one root node, but got none. HTML: %s", tc.html)
			node := tree.RootNodes[0]
			if node.TagName == "html" {
				node = ast_domain.MustQuery(tree, "body").Children[0]
			}
			require.NotNil(t, node)

			ApplyAttributesFromStyle(node, tc.styleMap)

			for attr, value := range tc.expectedAttrs {
				actualValue, ok := node.GetAttribute(attr)
				assert.True(t, ok, "Expected attribute '%s' to be present", attr)
				assert.Equal(t, value, actualValue)
			}
			for _, attr := range tc.unexpectedAttrs {
				_, ok := node.GetAttribute(attr)
				assert.False(t, ok, "Did not expect attribute '%s' to be present", attr)
			}
			if tc.existingAttrToKeep != "" {
				value, _ := node.GetAttribute(tc.existingAttrToKeep)
				assert.NotEqual(t, tc.expectedAttrs[tc.existingAttrToKeep], value,
					"Attribute '%s' should not have been overwritten", tc.existingAttrToKeep)
			}
		})
	}
}

func TestMapperFunctions(t *testing.T) {
	t.Run("mapWidthHeight", func(t *testing.T) {
		testCases := []struct {
			prop     string
			input    string
			expected string
			ok       bool
		}{
			{prop: "width", input: "100px", expected: "100", ok: true},
			{prop: "height", input: "50.5px", expected: "50.5", ok: true},
			{prop: "width", input: "75%", expected: "75%", ok: true},
			{prop: "height", input: "0", expected: "0", ok: true},
			{prop: "width", input: "100em", expected: "", ok: false},
			{prop: "height", input: "auto", expected: "", ok: false},
			{prop: "width", input: " 150px !important ", expected: "150", ok: true},
			{prop: "width", input: "50% !important", expected: "50%", ok: true},
		}
		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				name, value, ok := mapWidthHeight(tc.prop, tc.input)
				assert.Equal(t, tc.ok, ok)
				if ok {
					assert.Equal(t, tc.prop, name)
					assert.Equal(t, tc.expected, value)
				}
			})
		}
	})

	t.Run("mapBackgroundShorthand", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
			ok       bool
		}{
			{input: "#FF0000", expected: "#FF0000", ok: true},
			{input: "red url(foo.png)", expected: "red", ok: true},
			{input: "url(foo.png) no-repeat center / cover #eee", expected: "#eee", ok: true},
			{input: "url(foo.png)", expected: "", ok: false},
			{input: "no-repeat center", expected: "", ok: false},
		}
		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				name, value, ok := mapBackgroundShorthand("", tc.input)
				assert.Equal(t, tc.ok, ok)
				if ok {
					assert.Equal(t, "bgcolor", name)
					assert.Equal(t, tc.expected, value)
				}
			})
		}
	})

	t.Run("mapBorder", func(t *testing.T) {
		name, value, ok := mapBorder("", "0")
		assert.True(t, ok)
		assert.Equal(t, "border", name)
		assert.Equal(t, "0", value)

		name, value, ok = mapBorder("", "none")
		assert.True(t, ok)
		assert.Equal(t, "border", name)
		assert.Equal(t, "0", value)

		_, _, ok = mapBorder("", "1px solid black")
		assert.False(t, ok)
	})

	t.Run("mapCellspacing", func(t *testing.T) {
		name, value, ok := mapCellspacing("", "0px")
		assert.True(t, ok)
		assert.Equal(t, "cellspacing", name)
		assert.Equal(t, "0", value)

		_, _, ok = mapCellspacing("", "10px")
		assert.False(t, ok)
	})

	t.Run("mapPadding", func(t *testing.T) {
		name, value, ok := mapPadding("", "0")
		assert.True(t, ok)
		assert.Equal(t, "cellpadding", name)
		assert.Equal(t, "0", value)

		_, _, ok = mapPadding("", "10px")
		assert.False(t, ok)
	})

	t.Run("mapBgColor", func(t *testing.T) {
		name, value, ok := mapBgColor("", "#FF0000")
		assert.True(t, ok)
		assert.Equal(t, "bgcolor", name)
		assert.Equal(t, "#FF0000", value)
	})

	t.Run("mapTextAlign", func(t *testing.T) {
		name, value, ok := mapTextAlign("", "center")
		assert.True(t, ok)
		assert.Equal(t, "align", name)
		assert.Equal(t, "center", value)
	})

	t.Run("mapVerticalAlign", func(t *testing.T) {
		name, value, ok := mapVerticalAlign("", "middle")
		assert.True(t, ok)
		assert.Equal(t, "valign", name)
		assert.Equal(t, "middle", value)
	})

	t.Run("mapBackgroundImage", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
			ok       bool
		}{
			{name: "Simple URL", input: "url(image.png)", expected: "image.png", ok: true},
			{name: "URL with quotes", input: `url("image.jpg")`, expected: "image.jpg", ok: true},
			{name: "URL with single quotes", input: "url('image.gif')", expected: "image.gif", ok: true},
			{name: "URL with path", input: "url(https://example.com/bg.jpg)", expected: "https://example.com/bg.jpg", ok: true},
			{name: "URL with spaces", input: "url( image.png )", expected: "image.png", ok: true},
			{name: "Not a URL", input: "gradient(...)", expected: "", ok: false},
			{name: "Empty value", input: "", expected: "", ok: false},
			{name: "Empty URL", input: "url()", expected: "", ok: false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				name, value, ok := mapBackgroundImage("", tc.input)
				assert.Equal(t, tc.ok, ok)
				if ok {
					assert.Equal(t, "background", name)
					assert.Equal(t, tc.expected, value)
				}
			})
		}
	})

	t.Run("mapBorderCollapse", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
			ok       bool
		}{
			{name: "Collapse", input: "collapse", expected: "0", ok: true},
			{name: "Collapse uppercase", input: "COLLAPSE", expected: "0", ok: true},
			{name: "Separate", input: "separate", expected: "", ok: false},
			{name: "Empty", input: "", expected: "", ok: false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				name, value, ok := mapBorderCollapse("", tc.input)
				assert.Equal(t, tc.ok, ok)
				if ok {
					assert.Equal(t, "cellspacing", name)
					assert.Equal(t, tc.expected, value)
				}
			})
		}
	})

	t.Run("mapWhiteSpace", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
			ok       bool
		}{
			{name: "Nowrap", input: "nowrap", expected: "nowrap", ok: true},
			{name: "Nowrap uppercase", input: "NOWRAP", expected: "nowrap", ok: true},
			{name: "Nowrap with spaces", input: " nowrap ", expected: "nowrap", ok: true},
			{name: "Normal", input: "normal", expected: "", ok: false},
			{name: "Pre", input: "pre", expected: "", ok: false},
			{name: "Pre-wrap", input: "pre-wrap", expected: "", ok: false},
			{name: "Empty", input: "", expected: "", ok: false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				name, value, ok := mapWhiteSpace("", tc.input)
				assert.Equal(t, tc.ok, ok)
				if ok {
					assert.Equal(t, "nowrap", name)
					assert.Equal(t, tc.expected, value)
				}
			})
		}
	})
}
