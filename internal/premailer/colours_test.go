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
)

func TestConvertColourValues(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple rgb",
			input:    "rgb(255, 0, 128)",
			expected: "#ff0080",
		},
		{
			name:     "rgba (alpha is ignored)",
			input:    "rgba(0, 255, 0, 0.5)",
			expected: "#00ff00",
		},
		{
			name:     "rgb with percentages",
			input:    "rgb(100%, 0%, 50%)",
			expected: "#ff007f",
		},
		{
			name:     "rgb with extra whitespace",
			input:    "rgb( 255 ,  128,0)",
			expected: "#ff8000",
		},
		{
			name:     "Colour name",
			input:    "red",
			expected: "#ff0000",
		},
		{
			name:     "Colour name uppercase",
			input:    "BLUE",
			expected: "#0000ff",
		},
		{
			name:     "Value with colour name",
			input:    "1px solid black",
			expected: "1px solid #000000",
		},
		{
			name:     "Value with rgb function",
			input:    "5px dotted rgb(0, 0, 255)",
			expected: "5px dotted #0000ff",
		},
		{
			name:     "Value with no colour",
			input:    "10px",
			expected: "10px",
		},
		{
			name:     "URL value should not be touched",
			input:    "url(http://example.com/red.png)",
			expected: "url(http://example.com/red.png)",
		},
		{
			name:     "Invalid rgb is unchanged",
			input:    "rgb(255, 0)",
			expected: "#190500",
		},
		{
			name:     "Special keyword transparent",
			input:    "transparent",
			expected: "transparent",
		},
		{
			name:     "Multiple rgb values",
			input:    "linear-gradient(rgb(255,0,0), rgb(0,0,255))",
			expected: "linear-gradient(#ff0000, #0000ff)",
		},

		{
			name:     "Simple hsl - red",
			input:    "hsl(0, 100%, 50%)",
			expected: "#ff0000",
		},
		{
			name:     "Simple hsl - green",
			input:    "hsl(120, 100%, 50%)",
			expected: "#00ff00",
		},
		{
			name:     "Simple hsl - blue",
			input:    "hsl(240, 100%, 50%)",
			expected: "#0000ff",
		},
		{
			name:     "hsl - cyan",
			input:    "hsl(180, 100%, 50%)",
			expected: "#00ffff",
		},
		{
			name:     "hsl - magenta",
			input:    "hsl(300, 100%, 50%)",
			expected: "#ff00ff",
		},
		{
			name:     "hsl - yellow",
			input:    "hsl(60, 100%, 50%)",
			expected: "#ffff00",
		},
		{
			name:     "hsla (alpha is ignored)",
			input:    "hsla(120, 100%, 50%, 0.5)",
			expected: "#00ff00",
		},
		{
			name:     "hsl with extra whitespace",
			input:    "hsl( 240 ,  100% , 50% )",
			expected: "#0000ff",
		},
		{
			name:     "hsl - desaturated (gray)",
			input:    "hsl(0, 0%, 50%)",
			expected: "#808080",
		},
		{
			name:     "hsl - white",
			input:    "hsl(0, 0%, 100%)",
			expected: "#ffffff",
		},
		{
			name:     "hsl - black",
			input:    "hsl(0, 0%, 0%)",
			expected: "#000000",
		},
		{
			name:     "hsl - dark red",
			input:    "hsl(0, 100%, 25%)",
			expected: "#800000",
		},
		{
			name:     "hsl - light blue",
			input:    "hsl(210, 100%, 75%)",
			expected: "#80bfff",
		},
		{
			name:     "hsl - desaturated orange",
			input:    "hsl(30, 50%, 50%)",
			expected: "#bf8040",
		},
		{
			name:     "Value with hsl function",
			input:    "5px dotted hsl(240, 100%, 50%)",
			expected: "5px dotted #0000ff",
		},
		{
			name:     "Multiple hsl values",
			input:    "linear-gradient(hsl(0,100%,50%), hsl(240,100%,50%))",
			expected: "linear-gradient(#ff0000, #0000ff)",
		},
		{
			name:     "Mixed rgb and hsl",
			input:    "linear-gradient(rgb(255,0,0), hsl(240,100%,50%))",
			expected: "linear-gradient(#ff0000, #0000ff)",
		},
		{
			name:     "hsl with decimal hue",
			input:    "hsl(120.5, 100%, 50%)",
			expected: "#00ff00",
		},
		{
			name:     "hsl without % on saturation (treated as percentage)",
			input:    "hsl(120, 100, 50)",
			expected: "#00ff00",
		},
		{
			name:     "hsla with slash separator",
			input:    "hsla(120 100% 50% / 0.8)",
			expected: "#00ff00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := convertColorValues(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestClamp(t *testing.T) {
	testCases := []struct {
		name     string
		value    int
		minVal   int
		maxVal   int
		expected int
	}{
		{name: "value within range", value: 128, minVal: 0, maxVal: 255, expected: 128},
		{name: "value below minimum", value: -10, minVal: 0, maxVal: 255, expected: 0},
		{name: "value above maximum", value: 300, minVal: 0, maxVal: 255, expected: 255},
		{name: "value equals minimum", value: 0, minVal: 0, maxVal: 255, expected: 0},
		{name: "value equals maximum", value: 255, minVal: 0, maxVal: 255, expected: 255},
		{name: "zero range", value: 5, minVal: 3, maxVal: 3, expected: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, clamp(tc.value, tc.minVal, tc.maxVal))
		})
	}
}

func TestParseColourComponent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
		ok       bool
	}{
		{name: "integer value", input: "128", expected: 128, ok: true},
		{name: "zero", input: "0", expected: 0, ok: true},
		{name: "max 255", input: "255", expected: 255, ok: true},
		{name: "over 255 clamped", input: "300", expected: 255, ok: true},
		{name: "negative clamped", input: "-10", expected: 0, ok: true},
		{name: "percentage 100%", input: "100%", expected: 255, ok: true},
		{name: "percentage 50%", input: "50%", expected: 127, ok: true},
		{name: "percentage 0%", input: "0%", expected: 0, ok: true},
		{name: "whitespace trimmed", input: "  200  ", expected: 200, ok: true},
		{name: "invalid string", input: "abc", expected: 0, ok: false},
		{name: "invalid percentage", input: "abc%", expected: 0, ok: false},
		{name: "empty string", input: "", expected: 0, ok: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, ok := parseColorComponent(tc.input)
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.expected, value)
			}
		})
	}
}

func TestHslToRgb(t *testing.T) {
	testCases := []struct {
		name       string
		hue        float64
		saturation float64
		lightness  float64
		expectedR  int
		expectedG  int
		expectedB  int
	}{
		{name: "pure red", hue: 0, saturation: 100, lightness: 50, expectedR: 255, expectedG: 0, expectedB: 0},
		{name: "pure green", hue: 120, saturation: 100, lightness: 50, expectedR: 0, expectedG: 255, expectedB: 0},
		{name: "pure blue", hue: 240, saturation: 100, lightness: 50, expectedR: 0, expectedG: 0, expectedB: 255},
		{name: "white", hue: 0, saturation: 0, lightness: 100, expectedR: 255, expectedG: 255, expectedB: 255},
		{name: "black", hue: 0, saturation: 0, lightness: 0, expectedR: 0, expectedG: 0, expectedB: 0},
		{name: "grey 50%", hue: 0, saturation: 0, lightness: 50, expectedR: 128, expectedG: 128, expectedB: 128},
		{name: "negative hue wraps", hue: -60, saturation: 100, lightness: 50, expectedR: 255, expectedG: 0, expectedB: 255},
		{name: "hue above 360 wraps", hue: 480, saturation: 100, lightness: 50, expectedR: 0, expectedG: 255, expectedB: 0},
		{name: "high lightness", hue: 0, saturation: 100, lightness: 75, expectedR: 255, expectedG: 128, expectedB: 128},
		{name: "low lightness", hue: 0, saturation: 100, lightness: 25, expectedR: 128, expectedG: 0, expectedB: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b := hslToRgb(tc.hue, tc.saturation, tc.lightness)
			assert.Equal(t, tc.expectedR, r, "red component")
			assert.Equal(t, tc.expectedG, g, "green component")
			assert.Equal(t, tc.expectedB, b, "blue component")
		})
	}
}

func TestHueToRgb(t *testing.T) {
	testCases := []struct {
		name     string
		pValue   float64
		qValue   float64
		hueTemp  float64
		expected float64
	}{
		{name: "hue in first sector (< 60)", pValue: 0.0, qValue: 1.0, hueTemp: 30, expected: 0.5},
		{name: "hue in second sector (60-180)", pValue: 0.0, qValue: 1.0, hueTemp: 120, expected: 1.0},
		{name: "hue in third sector (180-240)", pValue: 0.0, qValue: 1.0, hueTemp: 200, expected: 0.6666666666666666},
		{name: "hue in fourth sector (>= 240)", pValue: 0.0, qValue: 1.0, hueTemp: 300, expected: 0.0},
		{name: "negative hue wraps to 300", pValue: 0.0, qValue: 1.0, hueTemp: -60, expected: 0.0},
		{name: "hue above 360 wraps", pValue: 0.0, qValue: 1.0, hueTemp: 420, expected: 1.0},
		{name: "pValue equals qValue (zero saturation)", pValue: 0.5, qValue: 0.5, hueTemp: 120, expected: 0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hueToRgb(tc.pValue, tc.qValue, tc.hueTemp)
			assert.InDelta(t, tc.expected, result, 0.001, "hueToRgb result")
		})
	}
}

func TestConvertColourNameWord(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "known colour name", input: "red", expected: "#ff0000"},
		{name: "uppercase colour name", input: "BLUE", expected: "#0000ff"},
		{name: "mixed case colour name", input: "GrEeN", expected: "#008000"},
		{name: "transparent is preserved", input: "transparent", expected: "transparent"},
		{name: "unknown word unchanged", input: "foobar", expected: "foobar"},
		{name: "numeric value unchanged", input: "10px", expected: "10px"},
		{name: "colour with trailing comma", input: "red,", expected: "#ff0000,"},
		{name: "colour with trailing semicolon", input: "blue;", expected: "#0000ff;"},
		{name: "empty string", input: "", expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertColorNameWord(tc.input))
		})
	}
}

func TestConvertColourNames(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single colour name", input: "red", expected: "#ff0000"},
		{name: "multiple colour names", input: "red blue green", expected: "#ff0000 #0000ff #008000"},
		{name: "mixed words and colours", input: "1px solid black", expected: "1px solid #000000"},
		{name: "no colour names", input: "10px 20px", expected: "10px 20px"},
		{name: "transparent preserved", input: "transparent", expected: "transparent"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertColorNames(tc.input))
		})
	}
}

func TestConvertRgbMatch(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "valid rgb", input: "rgb(255, 0, 128)", expected: "#ff0080"},
		{name: "valid rgba", input: "rgba(0, 255, 0, 0.5)", expected: "#00ff00"},
		{name: "percentages", input: "rgb(100%, 0%, 50%)", expected: "#ff007f"},
		{name: "no match returns input", input: "not-rgb", expected: "not-rgb"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertRgbMatch(tc.input))
		})
	}
}

func TestConvertHslMatch(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "valid hsl red", input: "hsl(0, 100%, 50%)", expected: "#ff0000"},
		{name: "valid hsl green", input: "hsl(120, 100%, 50%)", expected: "#00ff00"},
		{name: "valid hsla", input: "hsla(240, 100%, 50%, 0.5)", expected: "#0000ff"},
		{name: "no match returns input", input: "not-hsl", expected: "not-hsl"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, convertHslMatch(tc.input))
		})
	}
}
