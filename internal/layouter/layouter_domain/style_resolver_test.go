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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveStyle_EmptyProperties(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{}, nil, context)

	assert.Equal(t, DisplayInline, result.Display)
}

func TestResolveStyle_Display(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"display": "flex"}, nil, context)

	assert.Equal(t, DisplayFlex, result.Display)
}

func TestResolveStyle_Position(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"position": "absolute"}, nil, context)

	assert.Equal(t, PositionAbsolute, result.Position)
}

func TestResolveStyle_Width(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"width": "100px"}, nil, context)

	assert.Equal(t, DimensionPt(75), result.Width)
}

func TestResolveStyle_Height(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"height": "50%"}, nil, context)

	assert.Equal(t, DimensionPct(50), result.Height)
}

func TestResolveStyle_Colour(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"color": "red"}, nil, context)

	expected, ok := ParseColour("red")
	assert.True(t, ok)
	assert.Equal(t, expected, result.Colour)
}

func TestResolveStyle_FontSize(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected float64
	}{
		{
			name:     "pixel value is converted to points",
			value:    "16px",
			expected: 12,
		},
		{
			name:     "large keyword resolves relative to root font size",
			value:    "large",
			expected: 16 * 1.125,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			context := defaultResolutionContext()
			result := ResolveStyle(map[string]string{"font-size": tc.value}, nil, context)

			assert.InDelta(t, tc.expected, result.FontSize, 0.001)
		})
	}
}

func TestResolveStyle_Margin(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"margin-top": "10px"}, nil, context)

	assert.Equal(t, DimensionPt(7.5), result.MarginTop)
}

func TestResolveStyle_Padding(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"padding-left": "20px"}, nil, context)

	assert.InDelta(t, 15.0, result.PaddingLeft, 0.001)
}

func TestResolveStyle_Border(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{
		"border-top-width": "2px",
		"border-top-style": "solid",
		"border-top-color": "red",
	}, nil, context)

	assert.InDelta(t, 1.5, result.BorderTopWidth, 0.001)
	assert.Equal(t, BorderStyleSolid, result.BorderTopStyle)

	expectedColour, ok := ParseColour("red")
	assert.True(t, ok)
	assert.Equal(t, expectedColour, result.BorderTopColour)
}

func TestResolveStyle_Background(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"background-color": "#ff0000"}, nil, context)

	expectedColour, ok := ParseColour("#ff0000")
	assert.True(t, ok)
	assert.Equal(t, expectedColour, result.BackgroundColour)
}

func TestResolveStyle_Inheritance(t *testing.T) {

	parent := DefaultComputedStyle()
	parent.FontSize = 20
	redColour, _ := ParseColour("red")
	parent.Colour = redColour

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{}, &parent, context)

	assert.InDelta(t, 20.0, result.FontSize, 0.001)
	assert.Equal(t, redColour, result.Colour)
}

func TestResolveStyle_InheritKeyword(t *testing.T) {
	blueColour, _ := ParseColour("blue")
	parent := DefaultComputedStyle()
	parent.Display = DisplayFlex
	parent.Colour = blueColour

	context := defaultResolutionContext()

	t.Run("colour is inherited via the inherit keyword", func(t *testing.T) {
		result := ResolveStyle(map[string]string{"color": "inherit"}, &parent, context)
		assert.Equal(t, blueColour, result.Colour)
	})

	t.Run("display is inherited via the inherit keyword", func(t *testing.T) {
		result := ResolveStyle(map[string]string{"display": "inherit"}, &parent, context)
		assert.Equal(t, DisplayFlex, result.Display)
	})
}

func TestResolveStyle_InitialKeyword(t *testing.T) {

	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{"color": "initial"}, nil, context)

	assert.Equal(t, ColourBlack, result.Colour)
}

func TestResolveStyle_VarReferences(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{
		"--my-color": "red",
		"color":      "var(--my-color)",
	}, nil, context)

	expectedColour, ok := ParseColour("red")
	assert.True(t, ok)
	assert.Equal(t, expectedColour, result.Colour)
}

func TestResolveStyle_Flex(t *testing.T) {
	context := defaultResolutionContext()
	result := ResolveStyle(map[string]string{
		"display":        "flex",
		"flex-direction": "column",
		"flex-wrap":      "wrap",
	}, nil, context)

	assert.Equal(t, DisplayFlex, result.Display)
	assert.Equal(t, FlexDirectionColumn, result.FlexDirection)
	assert.Equal(t, FlexWrapWrap, result.FlexWrap)
}

func TestResolveLength(t *testing.T) {
	context := defaultResolutionContext()

	tests := []struct {
		name     string
		value    string
		expected float64
	}{
		{
			name:     "pixels are converted to points",
			value:    "100px",
			expected: 75,
		},
		{
			name:     "points remain unchanged",
			value:    "12pt",
			expected: 12,
		},
		{
			name:     "em units use parent font size",
			value:    "2em",
			expected: 24,
		},
		{
			name:     "rem units use root font size",
			value:    "2rem",
			expected: 32,
		},
		{
			name:     "inches convert to points",
			value:    "1in",
			expected: 72,
		},
		{
			name:     "centimetres convert to points",
			value:    "1cm",
			expected: 28.3465,
		},
		{
			name:     "millimetres convert to points",
			value:    "10mm",
			expected: 28.3465,
		},
		{
			name:     "picas convert to points",
			value:    "1pc",
			expected: 12,
		},
		{
			name:     "percentage resolves against containing block width",
			value:    "50%",
			expected: 297.64,
		},
		{
			name:     "zero string returns zero",
			value:    "0",
			expected: 0,
		},
		{
			name:     "unitless number is returned as-is",
			value:    "42",
			expected: 42,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveLength(tc.value, context)
			assert.InDelta(t, tc.expected, result, 0.001)
		})
	}
}

func TestResolveVarReferences(t *testing.T) {
	tests := []struct {
		customProperties map[string]string
		name             string
		value            string
		expected         string
		depth            int
	}{
		{
			name:             "simple variable reference is resolved",
			value:            "var(--foo)",
			customProperties: map[string]string{"--foo": "red"},
			depth:            0,
			expected:         "red",
		},
		{
			name:             "fallback is used when variable is missing",
			value:            "var(--missing, blue)",
			customProperties: map[string]string{},
			depth:            0,
			expected:         "blue",
		},
		{
			name:             "nested variable references are resolved",
			value:            "var(--a)",
			customProperties: map[string]string{"--a": "var(--b)", "--b": "green"},
			depth:            0,
			expected:         "green",
		},
		{
			name:             "exceeding max depth returns the unresolved string",
			value:            "var(--a)",
			customProperties: map[string]string{"--a": "resolved"},
			depth:            maxVarResolutionDepth + 1,
			expected:         "var(--a)",
		},
		{
			name:             "plain text without var() is returned unchanged",
			value:            "plain",
			customProperties: map[string]string{},
			depth:            0,
			expected:         "plain",
		},
		{
			name:             "missing variable with no fallback returns empty string",
			value:            "var(--missing)",
			customProperties: map[string]string{},
			depth:            0,
			expected:         "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveVarReferences(tc.value, tc.customProperties, tc.depth)
			assert.Equal(t, tc.expected, result)
		})
	}
}
