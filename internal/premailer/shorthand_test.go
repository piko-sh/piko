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
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestExpandShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		propName string
		value    string
	}{

		{
			name:     "margin with 1 value",
			propName: "margin",
			value:    "10px",
			expected: map[string]string{
				"margin-top":    "10px",
				"margin-right":  "10px",
				"margin-bottom": "10px",
				"margin-left":   "10px",
			},
		},
		{
			name:     "margin with 2 values",
			propName: "margin",
			value:    "10px 20px",
			expected: map[string]string{
				"margin-top":    "10px",
				"margin-right":  "20px",
				"margin-bottom": "10px",
				"margin-left":   "20px",
			},
		},
		{
			name:     "margin with 3 values",
			propName: "margin",
			value:    "10px 20px 30px",
			expected: map[string]string{
				"margin-top":    "10px",
				"margin-right":  "20px",
				"margin-bottom": "30px",
				"margin-left":   "20px",
			},
		},
		{
			name:     "margin with 4 values",
			propName: "margin",
			value:    "10px 20px 30px 40px",
			expected: map[string]string{
				"margin-top":    "10px",
				"margin-right":  "20px",
				"margin-bottom": "30px",
				"margin-left":   "40px",
			},
		},

		{
			name:     "padding with 1 value",
			propName: "padding",
			value:    "15px",
			expected: map[string]string{
				"padding-top":    "15px",
				"padding-right":  "15px",
				"padding-bottom": "15px",
				"padding-left":   "15px",
			},
		},
		{
			name:     "padding with 2 values",
			propName: "padding",
			value:    "10px 5px",
			expected: map[string]string{
				"padding-top":    "10px",
				"padding-right":  "5px",
				"padding-bottom": "10px",
				"padding-left":   "5px",
			},
		},

		{
			name:     "border-width with 1 value",
			propName: "border-width",
			value:    "2px",
			expected: map[string]string{
				"border-top-width":    "2px",
				"border-right-width":  "2px",
				"border-bottom-width": "2px",
				"border-left-width":   "2px",
			},
		},
		{
			name:     "border-width with 2 values",
			propName: "border-width",
			value:    "1px 2px",
			expected: map[string]string{
				"border-top-width":    "1px",
				"border-right-width":  "2px",
				"border-bottom-width": "1px",
				"border-left-width":   "2px",
			},
		},

		{
			name:     "border-style with 1 value",
			propName: "border-style",
			value:    "solid",
			expected: map[string]string{
				"border-top-style":    "solid",
				"border-right-style":  "solid",
				"border-bottom-style": "solid",
				"border-left-style":   "solid",
			},
		},

		{
			name:     "border-color with 1 value",
			propName: "border-color",
			value:    "#FF0000",
			expected: map[string]string{
				"border-top-color":    "#FF0000",
				"border-right-color":  "#FF0000",
				"border-bottom-color": "#FF0000",
				"border-left-color":   "#FF0000",
			},
		},

		{
			name:     "border with all 3 values",
			propName: "border",
			value:    "1px solid black",
			expected: map[string]string{
				"border-top-width":    "1px",
				"border-right-width":  "1px",
				"border-bottom-width": "1px",
				"border-left-width":   "1px",
				"border-top-style":    "solid",
				"border-right-style":  "solid",
				"border-bottom-style": "solid",
				"border-left-style":   "solid",
				"border-top-color":    "black",
				"border-right-color":  "black",
				"border-bottom-color": "black",
				"border-left-color":   "black",
			},
		},
		{
			name:     "border with 2 values (width and style)",
			propName: "border",
			value:    "2px dashed",
			expected: map[string]string{
				"border-top-width":    "2px",
				"border-right-width":  "2px",
				"border-bottom-width": "2px",
				"border-left-width":   "2px",
				"border-top-style":    "dashed",
				"border-right-style":  "dashed",
				"border-bottom-style": "dashed",
				"border-left-style":   "dashed",
			},
		},
		{
			name:     "border with only style",
			propName: "border",
			value:    "solid",
			expected: map[string]string{
				"border-top-style":    "solid",
				"border-right-style":  "solid",
				"border-bottom-style": "solid",
				"border-left-style":   "solid",
			},
		},

		{
			name:     "border-top",
			propName: "border-top",
			value:    "1px solid red",
			expected: map[string]string{
				"border-top-width": "1px",
				"border-top-style": "solid",
				"border-top-color": "red",
			},
		},
		{
			name:     "border-bottom with 2 values",
			propName: "border-bottom",
			value:    "3px dotted",
			expected: map[string]string{
				"border-bottom-width": "3px",
				"border-bottom-style": "dotted",
			},
		},

		{
			name:     "background with color only",
			propName: "background",
			value:    "#FFFFFF",
			expected: map[string]string{
				"background-color": "#FFFFFF",
			},
		},
		{
			name:     "background with url only",
			propName: "background",
			value:    "url(image.png)",
			expected: map[string]string{
				"background-image": "url(image.png)",
			},
		},
		{
			name:     "background with color and url",
			propName: "background",
			value:    "#CCC url(bg.png)",
			expected: map[string]string{
				"background-color": "#CCC",
				"background-image": "url(bg.png)",
			},
		},
		{
			name:     "background with url and repeat",
			propName: "background",
			value:    "url(image.png) no-repeat",
			expected: map[string]string{
				"background-image":  "url(image.png)",
				"background-repeat": "no-repeat",
			},
		},
		{
			name:     "background with url, repeat, and position",
			propName: "background",
			value:    "url(image.png) repeat-x center",
			expected: map[string]string{
				"background-image":    "url(image.png)",
				"background-repeat":   "repeat-x",
				"background-position": "center",
			},
		},
		{
			name:     "background with attachment (fixed)",
			propName: "background",
			value:    "url(hero.jpg) no-repeat center fixed",
			expected: map[string]string{
				"background-image":      "url(hero.jpg)",
				"background-repeat":     "no-repeat",
				"background-position":   "center",
				"background-attachment": "fixed",
			},
		},
		{
			name:     "background with position/size separator",
			propName: "background",
			value:    "url(bg.png) center / cover",
			expected: map[string]string{
				"background-image":    "url(bg.png)",
				"background-position": "center",
				"background-size":     "cover",
			},
		},
		{
			name:     "background with all properties",
			propName: "background",
			value:    "url(bg.jpg) top right / 50% 100% no-repeat fixed #f0f0f0",
			expected: map[string]string{
				"background-image":      "url(bg.jpg)",
				"background-position":   "top right",
				"background-size":       "50% 100%",
				"background-repeat":     "no-repeat",
				"background-attachment": "fixed",
				"background-color":      "#f0f0f0",
			},
		},
		{
			name:     "background with size (contain)",
			propName: "background",
			value:    "url(image.png) center / contain",
			expected: map[string]string{
				"background-image":    "url(image.png)",
				"background-position": "center",
				"background-size":     "contain",
			},
		},
		{
			name:     "background with attachment (scroll)",
			propName: "background",
			value:    "url(img.png) scroll",
			expected: map[string]string{
				"background-image":      "url(img.png)",
				"background-attachment": "scroll",
			},
		},

		{
			name:     "background with linear-gradient",
			propName: "background",
			value:    "linear-gradient(to right, #3498db, #e74c3c)",
			expected: map[string]string{
				"background-image": "linear-gradient(to right, #3498db, #e74c3c)",
			},
		},
		{
			name:     "background with radial-gradient",
			propName: "background",
			value:    "radial-gradient(circle, #3498db, #2c3e50)",
			expected: map[string]string{
				"background-image": "radial-gradient(circle, #3498db, #2c3e50)",
			},
		},
		{
			name:     "background with gradient and no-repeat",
			propName: "background",
			value:    "linear-gradient(to bottom, red, blue) no-repeat",
			expected: map[string]string{
				"background-image":  "linear-gradient(to bottom, red, blue)",
				"background-repeat": "no-repeat",
			},
		},
		{
			name:     "background with repeating-linear-gradient",
			propName: "background",
			value:    "repeating-linear-gradient(45deg, red, blue 20px)",
			expected: map[string]string{
				"background-image": "repeating-linear-gradient(45deg, red, blue 20px)",
			},
		},

		{
			name:     "border-radius with 1 value",
			propName: "border-radius",
			value:    "5px",
			expected: map[string]string{
				"border-top-left-radius":     "5px",
				"border-top-right-radius":    "5px",
				"border-bottom-right-radius": "5px",
				"border-bottom-left-radius":  "5px",
			},
		},
		{
			name:     "border-radius with 2 values",
			propName: "border-radius",
			value:    "5px 10px",
			expected: map[string]string{
				"border-top-left-radius":     "5px",
				"border-top-right-radius":    "10px",
				"border-bottom-right-radius": "5px",
				"border-bottom-left-radius":  "10px",
			},
		},
		{
			name:     "border-radius with 4 values",
			propName: "border-radius",
			value:    "5px 10px 15px 20px",
			expected: map[string]string{
				"border-top-left-radius":     "5px",
				"border-top-right-radius":    "10px",
				"border-bottom-right-radius": "15px",
				"border-bottom-left-radius":  "20px",
			},
		},

		{
			name:     "CSS-wide keyword inherit should not expand",
			propName: "margin",
			value:    "inherit",
			expected: nil,
		},
		{
			name:     "CSS-wide keyword initial should not expand",
			propName: "padding",
			value:    "initial",
			expected: nil,
		},
		{
			name:     "CSS-wide keyword unset should not expand",
			propName: "border",
			value:    "unset",
			expected: nil,
		},
		{
			name:     "Non-shorthand property returns nil",
			propName: "color",
			value:    "red",
			expected: nil,
		},
		{
			name:     "Font shorthand expands correctly",
			propName: "font",
			value:    "12px Arial",
			expected: map[string]string{
				"font-size":   "12px",
				"font-family": "Arial",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandShorthand(tc.propName, tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSplitSpaceDelimited(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected []string
	}{
		{
			name:     "single value",
			value:    "10px",
			expected: []string{"10px"},
		},
		{
			name:     "multiple values",
			value:    "10px 20px 30px",
			expected: []string{"10px", "20px", "30px"},
		},
		{
			name:     "value with function",
			value:    "url(image.png) no-repeat",
			expected: []string{"url(image.png)", "no-repeat"},
		},
		{
			name:     "value with quotes",
			value:    `"Helvetica Neue" Arial`,
			expected: []string{`"Helvetica Neue"`, "Arial"},
		},
		{
			name:     "extra whitespace",
			value:    "  10px   20px  ",
			expected: []string{"10px", "20px"},
		},
		{
			name:     "empty string",
			value:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := splitSpaceDelimited(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsBorderWidth(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{value: "1px", expected: true},
		{value: "2em", expected: true},
		{value: "0", expected: true},
		{value: "thin", expected: true},
		{value: "medium", expected: true},
		{value: "thick", expected: true},
		{value: "solid", expected: false},
		{value: "red", expected: false},
		{value: "auto", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			actual := isBorderWidth(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsBorderStyle(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{value: "solid", expected: true},
		{value: "dashed", expected: true},
		{value: "dotted", expected: true},
		{value: "double", expected: true},
		{value: "none", expected: true},
		{value: "hidden", expected: true},
		{value: "1px", expected: false},
		{value: "red", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			actual := isBorderStyle(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandShorthandIntegration(t *testing.T) {

	t.Run("Shorthand expansion in ProcessCSS", func(t *testing.T) {
		css := `
			.box {
				margin: 10px 20px;
				padding: 15px;
				border: 1px solid black;
			}
		`
		cssAST := parseTestCSS(t, css)

		var diagnostics []*ast_domain.Diagnostic
		ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: true}, &diagnostics, "test.css")

		assert.Len(t, ruleSet.InlineableRules, 1)
		rule := ruleSet.InlineableRules[0]

		assert.Contains(t, rule.properties, "margin-top")
		assert.Contains(t, rule.properties, "margin-right")
		assert.Contains(t, rule.properties, "padding-top")
		assert.Contains(t, rule.properties, "border-top-width")
		assert.Contains(t, rule.properties, "border-top-style")
		assert.Contains(t, rule.properties, "border-top-color")

		assert.NotContains(t, rule.properties, "margin")
		assert.NotContains(t, rule.properties, "padding")
		assert.NotContains(t, rule.properties, "border")

		assert.Equal(t, "10px", rule.properties["margin-top"].value)
		assert.Equal(t, "20px", rule.properties["margin-right"].value)
		assert.Equal(t, "15px", rule.properties["padding-top"].value)
		assert.Equal(t, "1px", rule.properties["border-top-width"].value)
		assert.Equal(t, "solid", rule.properties["border-top-style"].value)
		assert.Equal(t, "#000000", rule.properties["border-top-color"].value)
	})

	t.Run("Shorthand expansion disabled", func(t *testing.T) {
		css := `
			.box {
				margin: 10px 20px;
				padding: 15px;
			}
		`
		cssAST := parseTestCSS(t, css)

		var diagnostics []*ast_domain.Diagnostic
		ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: false}, &diagnostics, "test.css")

		assert.Len(t, ruleSet.InlineableRules, 1)
		rule := ruleSet.InlineableRules[0]

		assert.Contains(t, rule.properties, "margin")
		assert.Contains(t, rule.properties, "padding")

		assert.NotContains(t, rule.properties, "margin-top")
		assert.NotContains(t, rule.properties, "padding-top")
	})

	t.Run("Important flag preserved on expanded properties", func(t *testing.T) {
		css := `.box { margin: 10px !important; }`
		cssAST := parseTestCSS(t, css)

		var diagnostics []*ast_domain.Diagnostic
		ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: true}, &diagnostics, "test.css")

		assert.Len(t, ruleSet.InlineableRules, 1)
		rule := ruleSet.InlineableRules[0]

		assert.True(t, rule.properties["margin-top"].important)
		assert.True(t, rule.properties["margin-right"].important)
		assert.True(t, rule.properties["margin-bottom"].important)
		assert.True(t, rule.properties["margin-left"].important)
	})
}

func TestExpandFontShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "Minimal font (size and family only)",
			value: "12px Arial",
			expected: map[string]string{
				"font-size":   "12px",
				"font-family": "Arial",
			},
		},
		{
			name:  "Font with style",
			value: "italic 14px Helvetica",
			expected: map[string]string{
				"font-style":  "italic",
				"font-size":   "14px",
				"font-family": "Helvetica",
			},
		},
		{
			name:  "Font with style and weight",
			value: "italic bold 16px Georgia",
			expected: map[string]string{
				"font-style":  "italic",
				"font-weight": "bold",
				"font-size":   "16px",
				"font-family": "Georgia",
			},
		},
		{
			name:  "Font with all optional properties",
			value: "italic small-caps bold 18px Verdana",
			expected: map[string]string{
				"font-style":   "italic",
				"font-variant": "small-caps",
				"font-weight":  "bold",
				"font-size":    "18px",
				"font-family":  "Verdana",
			},
		},
		{
			name:  "Font with line-height",
			value: "12px/1.5 Arial",
			expected: map[string]string{
				"font-size":   "12px",
				"line-height": "1.5",
				"font-family": "Arial",
			},
		},
		{
			name:  "Font with all properties including line-height",
			value: "italic small-caps 700 16px/1.6 'Times New Roman'",
			expected: map[string]string{
				"font-style":   "italic",
				"font-variant": "small-caps",
				"font-weight":  "700",
				"font-size":    "16px",
				"line-height":  "1.6",
				"font-family":  "'Times New Roman'",
			},
		},
		{
			name:  "Font family with multiple words",
			value: "14px Times New Roman",
			expected: map[string]string{
				"font-size":   "14px",
				"font-family": "Times New Roman",
			},
		},
		{
			name:  "Numeric font-weight",
			value: "400 12px Arial",
			expected: map[string]string{
				"font-weight": "400",
				"font-size":   "12px",
				"font-family": "Arial",
			},
		},
		{
			name:     "System font (caption) - not expanded",
			value:    "caption",
			expected: nil,
		},
		{
			name:     "System font (icon) - not expanded",
			value:    "icon",
			expected: nil,
		},
		{
			name:     "System font (menu) - not expanded",
			value:    "menu",
			expected: nil,
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
		{
			name:     "Only size (missing required family)",
			value:    "12px",
			expected: nil,
		},
		{
			name:  "Font with oblique style",
			value: "oblique 14px Arial",
			expected: map[string]string{
				"font-style":  "oblique",
				"font-size":   "14px",
				"font-family": "Arial",
			},
		},
		{
			name:  "Font with lighter weight",
			value: "lighter 12px Arial",
			expected: map[string]string{
				"font-weight": "lighter",
				"font-size":   "12px",
				"font-family": "Arial",
			},
		},
		{
			name:  "Font with bolder weight",
			value: "bolder 14px Arial",
			expected: map[string]string{
				"font-weight": "bolder",
				"font-size":   "14px",
				"font-family": "Arial",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandFontShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandOutlineShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "All 3 values (width, style, color)",
			value: "1px solid black",
			expected: map[string]string{
				"outline-width": "1px",
				"outline-style": "solid",
				"outline-color": "black",
			},
		},
		{
			name:  "Width and style only",
			value: "2px dashed",
			expected: map[string]string{
				"outline-width": "2px",
				"outline-style": "dashed",
			},
		},
		{
			name:  "Style and color only",
			value: "dotted red",
			expected: map[string]string{
				"outline-style": "dotted",
				"outline-color": "red",
			},
		},
		{
			name:  "Style only",
			value: "double",
			expected: map[string]string{
				"outline-style": "double",
			},
		},
		{
			name:  "Keyword width (medium)",
			value: "medium solid blue",
			expected: map[string]string{
				"outline-width": "medium",
				"outline-style": "solid",
				"outline-color": "blue",
			},
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
		{
			name:  "None keyword",
			value: "none",
			expected: map[string]string{
				"outline-style": "none",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandOutlineShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandColumnRuleShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "width style and colour",
			value: "2px solid #3498db",
			expected: map[string]string{
				"column-rule-width": "2px",
				"column-rule-style": "solid",
				"column-rule-color": "#3498db",
			},
		},
		{
			name:  "style only",
			value: "dashed",
			expected: map[string]string{
				"column-rule-style": "dashed",
			},
		},
		{
			name:  "width and style",
			value: "1px dotted",
			expected: map[string]string{
				"column-rule-width": "1px",
				"column-rule-style": "dotted",
			},
		},
		{
			name:     "empty value",
			value:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandColumnRuleShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandListStyleShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "Type only",
			value: "disc",
			expected: map[string]string{
				"list-style-type": "disc",
			},
		},
		{
			name:  "Type and position",
			value: "circle inside",
			expected: map[string]string{
				"list-style-type":     "circle",
				"list-style-position": "inside",
			},
		},
		{
			name:  "All 3 values",
			value: "square outside url(marker.png)",
			expected: map[string]string{
				"list-style-type":     "square",
				"list-style-position": "outside",
				"list-style-image":    "url(marker.png)",
			},
		},
		{
			name:  "Image only",
			value: "url(custom-marker.svg)",
			expected: map[string]string{
				"list-style-image": "url(custom-marker.svg)",
			},
		},
		{
			name:  "Position only",
			value: "inside",
			expected: map[string]string{
				"list-style-position": "inside",
			},
		},
		{
			name:  "Decimal type",
			value: "decimal",
			expected: map[string]string{
				"list-style-type": "decimal",
			},
		},
		{
			name:  "Lower-alpha type",
			value: "lower-alpha",
			expected: map[string]string{
				"list-style-type": "lower-alpha",
			},
		},
		{
			name:  "Lower-roman type with position",
			value: "lower-roman outside",
			expected: map[string]string{
				"list-style-type":     "lower-roman",
				"list-style-position": "outside",
			},
		},
		{
			name:  "None keyword",
			value: "none",
			expected: map[string]string{
				"list-style-type": "none",
			},
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
		{
			name:  "CJK decimal type",
			value: "cjk-decimal",
			expected: map[string]string{
				"list-style-type": "cjk-decimal",
			},
		},
		{
			name:  "Georgian type",
			value: "georgian",
			expected: map[string]string{
				"list-style-type": "georgian",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandListStyleShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandOverflowShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "Single value (both axes)",
			value: "hidden",
			expected: map[string]string{
				"overflow-x": "hidden",
				"overflow-y": "hidden",
			},
		},
		{
			name:  "Two values (x and y)",
			value: "scroll auto",
			expected: map[string]string{
				"overflow-x": "scroll",
				"overflow-y": "auto",
			},
		},
		{
			name:  "Visible on both axes",
			value: "visible",
			expected: map[string]string{
				"overflow-x": "visible",
				"overflow-y": "visible",
			},
		},
		{
			name:  "Auto on x, hidden on y",
			value: "auto hidden",
			expected: map[string]string{
				"overflow-x": "auto",
				"overflow-y": "hidden",
			},
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
		{
			name:     "Too many values (invalid)",
			value:    "scroll auto hidden",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandOverflowShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandInsetShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "Single value (all sides)",
			value: "10px",
			expected: map[string]string{
				"top":    "10px",
				"right":  "10px",
				"bottom": "10px",
				"left":   "10px",
			},
		},
		{
			name:  "Two values (vertical, horizontal)",
			value: "10px 20px",
			expected: map[string]string{
				"top":    "10px",
				"right":  "20px",
				"bottom": "10px",
				"left":   "20px",
			},
		},
		{
			name:  "Three values (top, horizontal, bottom)",
			value: "10px 20px 30px",
			expected: map[string]string{
				"top":    "10px",
				"right":  "20px",
				"bottom": "30px",
				"left":   "20px",
			},
		},
		{
			name:  "Four values (clockwise from top)",
			value: "10px 20px 30px 40px",
			expected: map[string]string{
				"top":    "10px",
				"right":  "20px",
				"bottom": "30px",
				"left":   "40px",
			},
		},
		{
			name:  "Auto value",
			value: "auto",
			expected: map[string]string{
				"top":    "auto",
				"right":  "auto",
				"bottom": "auto",
				"left":   "auto",
			},
		},
		{
			name:  "Mixed units",
			value: "0 20px 1em 5%",
			expected: map[string]string{
				"top":    "0",
				"right":  "20px",
				"bottom": "1em",
				"left":   "5%",
			},
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
		{
			name:     "Too many values (invalid)",
			value:    "10px 20px 30px 40px 50px",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandInsetShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandTextDecorationShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "None keyword",
			value: "none",
			expected: map[string]string{
				"text-decoration-line": "none",
			},
		},
		{
			name:  "Simple underline",
			value: "underline",
			expected: map[string]string{
				"text-decoration-line": "underline",
			},
		},
		{
			name:  "Line and color",
			value: "underline blue",
			expected: map[string]string{
				"text-decoration-line":  "underline",
				"text-decoration-color": "blue",
			},
		},
		{
			name:  "All properties",
			value: "underline wavy red 2px",
			expected: map[string]string{
				"text-decoration-line":      "underline",
				"text-decoration-style":     "wavy",
				"text-decoration-color":     "red",
				"text-decoration-thickness": "2px",
			},
		},
		{
			name:  "Line and style",
			value: "line-through dotted",
			expected: map[string]string{
				"text-decoration-line":  "line-through",
				"text-decoration-style": "dotted",
			},
		},
		{
			name:  "Overline with color",
			value: "overline #ff0000",
			expected: map[string]string{
				"text-decoration-line":  "overline",
				"text-decoration-color": "#ff0000",
			},
		},
		{
			name:  "Style and thickness",
			value: "underline solid 1px",
			expected: map[string]string{
				"text-decoration-line":      "underline",
				"text-decoration-style":     "solid",
				"text-decoration-thickness": "1px",
			},
		},
		{
			name:  "Thickness keyword (auto)",
			value: "underline auto",
			expected: map[string]string{
				"text-decoration-line":      "underline",
				"text-decoration-thickness": "auto",
			},
		},
		{
			name:  "Thickness keyword (from-font)",
			value: "line-through from-font",
			expected: map[string]string{
				"text-decoration-line":      "line-through",
				"text-decoration-thickness": "from-font",
			},
		},
		{
			name:  "Double style",
			value: "underline double",
			expected: map[string]string{
				"text-decoration-line":  "underline",
				"text-decoration-style": "double",
			},
		},
		{
			name:     "Empty value",
			value:    "",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandTextDecorationShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
