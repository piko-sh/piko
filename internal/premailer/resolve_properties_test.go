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

func TestResolveProperties(t *testing.T) {
	t.Run("returns properties without modifying AST", func(t *testing.T) {
		htmlInput := `<p class="msg">Hello</p>`
		css := `.msg { color: red; font-size: 14px; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithExpandShorthands(true),
			WithSkipEmailValidation(true),
			WithSkipStyleExtraction(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		pNode := ast_domain.MustQuery(tree, "p")
		require.NotNil(t, pNode)
		props := resolved.Elements[pNode]
		require.NotNil(t, props, "should have properties for p node")
		assert.Equal(t, "#ff0000", props["color"])
		assert.Equal(t, "14px", props["font-size"])

		_, hasStyle := pNode.GetAttribute("style")
		assert.False(t, hasStyle, "AST should not be modified by ResolveProperties")
	})

	t.Run("merges inline styles with stylesheet rules", func(t *testing.T) {
		htmlInput := `<p style="font-size: 16px;" class="msg">Hello</p>`
		css := `.msg { color: blue; font-size: 12px; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithExpandShorthands(true),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		pNode := ast_domain.MustQuery(tree, "p")
		props := resolved.Elements[pNode]
		require.NotNil(t, props)

		assert.Equal(t, "16px", props["font-size"])

		assert.Equal(t, "#0000ff", props["color"])
	})

	t.Run("returns empty result for no CSS", func(t *testing.T) {
		htmlInput := `<p>Hello</p>`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree, WithSkipEmailValidation(true))
		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		assert.Empty(t, resolved.Elements)
		assert.Empty(t, resolved.PseudoElements)
	})

	t.Run("expands inline shorthand properties", func(t *testing.T) {
		htmlInput := `<div style="margin: 10px 20px;">Content</div>`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExpandShorthands(true),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		divNode := ast_domain.MustQuery(tree, "div")
		props := resolved.Elements[divNode]
		require.NotNil(t, props)
		assert.Equal(t, "10px", props["margin-top"])
		assert.Equal(t, "20px", props["margin-right"])
		assert.Equal(t, "10px", props["margin-bottom"])
		assert.Equal(t, "20px", props["margin-left"])
	})

	t.Run("inline !important overrides stylesheet !important", func(t *testing.T) {
		htmlInput := `<p style="color: green !important;" class="msg">Hello</p>`
		css := `.msg { color: red !important; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithExpandShorthands(true),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		pNode := ast_domain.MustQuery(tree, "p")
		props := resolved.Elements[pNode]
		require.NotNil(t, props)

		assert.Equal(t, "green", props["color"])
	})

	t.Run("stylesheet !important overrides inline normal", func(t *testing.T) {
		htmlInput := `<p style="color: green;" class="msg">Hello</p>`
		css := `.msg { color: red !important; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithExpandShorthands(true),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		pNode := ast_domain.MustQuery(tree, "p")
		props := resolved.Elements[pNode]
		require.NotNil(t, props)

		assert.Equal(t, "#ff0000", props["color"])
	})

	t.Run("collects pseudo-element rules when enabled", func(t *testing.T) {
		htmlInput := `<p class="msg">Hello</p>`
		css := `.msg::before { content: ">>"; color: red; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithExpandShorthands(true),
			WithResolvePseudoElements(true),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		pNode := ast_domain.MustQuery(tree, "p")
		require.NotNil(t, pNode)
		pseudoMap := resolved.PseudoElements[pNode]
		require.NotNil(t, pseudoMap, "should have pseudo-element properties")
		beforeProps := pseudoMap["before"]
		require.NotNil(t, beforeProps, "should have ::before properties")
		assert.Equal(t, "#ff0000", beforeProps["color"])
	})

	t.Run("no pseudo-elements when disabled", func(t *testing.T) {
		htmlInput := `<p class="msg">Hello</p>`
		css := `.msg::before { content: ">>"; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithResolvePseudoElements(false),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)

		assert.Empty(t, resolved.PseudoElements)
	})
}

func TestParseInlineStyleWithImportance(t *testing.T) {
	testCases := []struct {
		expected map[string]property
		name     string
		input    string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]property{},
		},
		{
			name:  "single property",
			input: "color: red",
			expected: map[string]property{
				"color": {value: "red", important: false},
			},
		},
		{
			name:  "property with !important",
			input: "color: red !important",
			expected: map[string]property{
				"color": {value: "red", important: true},
			},
		},
		{
			name:  "multiple properties mixed importance",
			input: "color: red !important; font-size: 14px; margin: 0 !important",
			expected: map[string]property{
				"color":     {value: "red", important: true},
				"font-size": {value: "14px", important: false},
				"margin":    {value: "0", important: true},
			},
		},
		{
			name:  "HTML-encoded single quotes are decoded",
			input: "grid-template-areas: &#39;header header&#39; &#39;main sidebar&#39;",
			expected: map[string]property{
				"grid-template-areas": {
					value:     "'header header' 'main sidebar'",
					important: false,
				},
			},
		},
		{
			name:  "trailing semicolon is handled",
			input: "color: blue;",
			expected: map[string]property{
				"color": {value: "blue", important: false},
			},
		},
		{
			name:  "malformed entry with no colon is skipped",
			input: "invalidentry; color: red",
			expected: map[string]property{
				"color": {value: "red", important: false},
			},
		},
		{
			name:  "empty value is skipped",
			input: "color:; font-size: 12px",
			expected: map[string]property{
				"font-size": {value: "12px", important: false},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseInlineStyleWithImportance(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandInlineShorthandsWithImportance(t *testing.T) {
	t.Run("expands shorthand preserving importance", func(t *testing.T) {
		input := map[string]property{
			"margin": {value: "10px 20px", important: true},
			"color":  {value: "red", important: false},
		}

		result := expandInlineShorthandsWithImportance(input)

		assert.Equal(t, property{value: "10px", important: true}, result["margin-top"])
		assert.Equal(t, property{value: "20px", important: true}, result["margin-right"])
		assert.Equal(t, property{value: "10px", important: true}, result["margin-bottom"])
		assert.Equal(t, property{value: "20px", important: true}, result["margin-left"])

		assert.Equal(t, property{value: "red", important: false}, result["color"])

		_, hasMargin := result["margin"]
		assert.False(t, hasMargin)
	})

	t.Run("expands CSS-wide keywords on shorthands", func(t *testing.T) {
		input := map[string]property{
			"margin": {value: "inherit", important: false},
		}

		result := expandInlineShorthandsWithImportance(input)

		assert.Equal(t, property{value: "inherit", important: false}, result["margin-top"])
		assert.Equal(t, property{value: "inherit", important: false}, result["margin-right"])
		assert.Equal(t, property{value: "inherit", important: false}, result["margin-bottom"])
		assert.Equal(t, property{value: "inherit", important: false}, result["margin-left"])
	})

	t.Run("passes through unknown properties", func(t *testing.T) {
		input := map[string]property{
			"color": {value: "blue", important: false},
		}

		result := expandInlineShorthandsWithImportance(input)

		assert.Equal(t, property{value: "blue", important: false}, result["color"])
	})

	t.Run("expands flex shorthand", func(t *testing.T) {
		input := map[string]property{
			"flex": {value: "1 0 auto", important: false},
		}

		result := expandInlineShorthandsWithImportance(input)

		assert.Equal(t, property{value: "1", important: false}, result["flex-grow"])
		assert.Equal(t, property{value: "0", important: false}, result["flex-shrink"])
		assert.Equal(t, property{value: "auto", important: false}, result["flex-basis"])
	})
}

func TestMergeRuleAndInlineProperties(t *testing.T) {
	t.Run("inline normal overrides stylesheet normal", func(t *testing.T) {
		ruleProps := map[string]property{
			"color":     {value: "red", important: false},
			"font-size": {value: "12px", important: false},
		}
		inline := map[string]property{
			"color": {value: "blue", important: false},
		}

		result := mergeRuleAndInlineProperties(ruleProps, inline)

		assert.Equal(t, "blue", result["color"])
		assert.Equal(t, "12px", result["font-size"])
	})

	t.Run("stylesheet !important overrides inline normal", func(t *testing.T) {
		ruleProps := map[string]property{
			"color": {value: "red", important: true},
		}
		inline := map[string]property{
			"color": {value: "blue", important: false},
		}

		result := mergeRuleAndInlineProperties(ruleProps, inline)

		assert.Equal(t, "red", result["color"])
	})

	t.Run("inline !important overrides stylesheet !important", func(t *testing.T) {
		ruleProps := map[string]property{
			"color": {value: "red", important: true},
		}
		inline := map[string]property{
			"color": {value: "green", important: true},
		}

		result := mergeRuleAndInlineProperties(ruleProps, inline)

		assert.Equal(t, "green", result["color"])
	})

	t.Run("nil rule props works with inline only", func(t *testing.T) {
		inline := map[string]property{
			"color": {value: "blue", important: false},
		}

		result := mergeRuleAndInlineProperties(nil, inline)

		assert.Equal(t, "blue", result["color"])
	})

	t.Run("nil inline works with rule props only", func(t *testing.T) {
		ruleProps := map[string]property{
			"color": {value: "red", important: false},
		}

		result := mergeRuleAndInlineProperties(ruleProps, nil)

		assert.Equal(t, "red", result["color"])
	})
}

func TestExpandCSSWideKeyword(t *testing.T) {
	testCases := []struct {
		name     string
		prop     string
		value    string
		expected map[string]string
	}{
		{
			name:  "inherit on margin",
			prop:  "margin",
			value: "inherit",
			expected: map[string]string{
				"margin-top":    "inherit",
				"margin-right":  "inherit",
				"margin-bottom": "inherit",
				"margin-left":   "inherit",
			},
		},
		{
			name:  "initial on padding",
			prop:  "padding",
			value: "initial",
			expected: map[string]string{
				"padding-top":    "initial",
				"padding-right":  "initial",
				"padding-bottom": "initial",
				"padding-left":   "initial",
			},
		},
		{
			name:  "unset on flex",
			prop:  "flex",
			value: "unset",
			expected: map[string]string{
				"flex-grow":   "unset",
				"flex-shrink": "unset",
				"flex-basis":  "unset",
			},
		},
		{
			name:     "unknown property returns nil",
			prop:     "color",
			value:    "inherit",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandCSSWideKeyword(tc.prop, tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestResolvePropertiesSkipEmailValidation(t *testing.T) {

	t.Run("skip email validation suppresses CSS diagnostics", func(t *testing.T) {
		htmlInput := `<p class="msg">Hello</p>`
		css := `.msg { color: red; }`

		tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
		require.NoError(t, err)

		pm := New(tree,
			WithExternalCSS(css),
			WithSkipEmailValidation(true),
		)

		resolved, err := pm.ResolveProperties()
		require.NoError(t, err)
		assert.NotNil(t, resolved)
	})
}

func TestResolvePropertiesSkipStyleExtraction(t *testing.T) {
	htmlInput := `<html><head><style>p { color: red; }</style></head><body><p>Hello</p></body></html>`

	tree, err := ast_domain.Parse(context.Background(), htmlInput, "test.html", nil)
	require.NoError(t, err)

	pm := New(tree,
		WithSkipStyleExtraction(true),
		WithSkipEmailValidation(true),
	)

	_, err = pm.Transform()
	require.NoError(t, err)

	styleNode := ast_domain.MustQuery(tree, "style")
	assert.NotNil(t, styleNode, "style tag should be preserved when SkipStyleExtraction is true")
}
