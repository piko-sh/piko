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

func TestNewRGBA(t *testing.T) {
	tests := []struct {
		name                            string
		red, green, blue, alpha         float64
		expectedR, expectedG, expectedB float64
		expectedA                       float64
		expectedSpace                   ColourSpace
	}{
		{
			name: "black",
			red:  0, green: 0, blue: 0, alpha: 1,
			expectedR: 0, expectedG: 0, expectedB: 0,
			expectedA:     1,
			expectedSpace: ColourSpaceRGB,
		},
		{
			name: "white",
			red:  1, green: 1, blue: 1, alpha: 1,
			expectedR: 1, expectedG: 1, expectedB: 1,
			expectedA:     1,
			expectedSpace: ColourSpaceRGB,
		},
		{
			name: "half alpha",
			red:  0.5, green: 0.5, blue: 0.5, alpha: 0.5,
			expectedR: 0.5, expectedG: 0.5, expectedB: 0.5,
			expectedA:     0.5,
			expectedSpace: ColourSpaceRGB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRGBA(tt.red, tt.green, tt.blue, tt.alpha)
			assert.InDelta(t, tt.expectedR, c.Red, 0.001)
			assert.InDelta(t, tt.expectedG, c.Green, 0.001)
			assert.InDelta(t, tt.expectedB, c.Blue, 0.001)
			assert.InDelta(t, tt.expectedA, c.Alpha, 0.001)
			assert.Equal(t, tt.expectedSpace, c.Space)
		})
	}
}

func TestNewGrey(t *testing.T) {
	tests := []struct {
		name          string
		value         float64
		alpha         float64
		expectedSpace ColourSpace
	}{
		{
			name:          "mid grey full alpha",
			value:         0.5,
			alpha:         1.0,
			expectedSpace: ColourSpaceGrey,
		},
		{
			name:          "black half alpha",
			value:         0,
			alpha:         0.5,
			expectedSpace: ColourSpaceGrey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewGrey(tt.value, tt.alpha)

			assert.InDelta(t, tt.value, c.Red, 0.001)
			assert.InDelta(t, tt.value, c.Green, 0.001)
			assert.InDelta(t, tt.value, c.Blue, 0.001)
			assert.InDelta(t, tt.alpha, c.Alpha, 0.001)
			assert.Equal(t, tt.expectedSpace, c.Space)
		})
	}
}

func TestNewCMYK(t *testing.T) {
	tests := []struct {
		name                       string
		cyan, magenta, yellow, key float64
	}{
		{
			name: "pure cyan",
			cyan: 1, magenta: 0, yellow: 0, key: 0,
		},
		{
			name: "custom values",
			cyan: 0.3, magenta: 0.6, yellow: 0.1, key: 0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCMYK(tt.cyan, tt.magenta, tt.yellow, tt.key)
			assert.InDelta(t, tt.cyan, c.Cyan, 0.001)
			assert.InDelta(t, tt.magenta, c.Magenta, 0.001)
			assert.InDelta(t, tt.yellow, c.Yellow, 0.001)
			assert.InDelta(t, tt.key, c.Key, 0.001)
			assert.InDelta(t, 1.0, c.Alpha, 0.001)
			assert.Equal(t, ColourSpaceCMYK, c.Space)
		})
	}
}

func TestNewHSLA(t *testing.T) {
	tests := []struct {
		name       string
		hue        float64
		saturation float64
		lightness  float64
		alpha      float64
		expectedR  float64
		expectedG  float64
		expectedB  float64
	}{
		{

			name: "red",
			hue:  0, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 1, expectedG: 0, expectedB: 0,
		},
		{

			name: "yellow",
			hue:  60, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 1, expectedG: 1, expectedB: 0,
		},
		{

			name: "green",
			hue:  120, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 0, expectedG: 1, expectedB: 0,
		},
		{

			name: "cyan",
			hue:  180, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 0, expectedG: 1, expectedB: 1,
		},
		{

			name: "blue",
			hue:  240, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 0, expectedG: 0, expectedB: 1,
		},
		{

			name: "magenta",
			hue:  300, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 1, expectedG: 0, expectedB: 1,
		},
		{

			name: "negative hue wraps to magenta",
			hue:  -60, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 1, expectedG: 0, expectedB: 1,
		},
		{

			name: "hue above 360 wraps to yellow",
			hue:  420, saturation: 1, lightness: 0.5, alpha: 1,
			expectedR: 1, expectedG: 1, expectedB: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewHSLA(tt.hue, tt.saturation, tt.lightness, tt.alpha)
			assert.InDelta(t, tt.expectedR, c.Red, 0.001)
			assert.InDelta(t, tt.expectedG, c.Green, 0.001)
			assert.InDelta(t, tt.expectedB, c.Blue, 0.001)
		})
	}
}

func TestColour_String(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		colour   Colour
	}{
		{
			name:     "RGB opaque",
			colour:   NewRGBA(0.5, 0.5, 0.5, 1),
			expected: "rgb(0.50, 0.50, 0.50)",
		},
		{
			name:     "RGBA with transparency",
			colour:   NewRGBA(0.5, 0.5, 0.5, 0.5),
			expected: "rgba(0.50, 0.50, 0.50, 0.50)",
		},
		{
			name:     "gray",
			colour:   NewGrey(0.5, 1.0),
			expected: "grey(0.50, 1.00)",
		},
		{
			name:     "CMYK",
			colour:   NewCMYK(1, 0, 0, 0),
			expected: "cmyk(1.00, 0.00, 0.00, 0.00)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.colour.String())
		})
	}
}

func TestParseColour_Named(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectedR float64
		expectedG float64
		expectedB float64
		expectedA float64
		expectOK  bool
	}{
		{
			name:      "red",
			input:     "red",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{
			name:      "transparent",
			input:     "transparent",
			expectedR: 0, expectedG: 0, expectedB: 0, expectedA: 0,
			expectOK: true,
		},
		{
			name:      "rebeccapurple",
			input:     "rebeccapurple",
			expectedR: 102.0 / 255, expectedG: 51.0 / 255, expectedB: 153.0 / 255, expectedA: 1,
			expectOK: true,
		},
		{
			name:     "unknown colour returns false",
			input:    "unknown",
			expectOK: false,
		},
		{
			name:     "empty string returns false",
			input:    "",
			expectOK: false,
		},
		{

			name:      "case insensitive match",
			input:     "RED",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colour, ok := ParseColour(tt.input)
			assert.Equal(t, tt.expectOK, ok)
			if ok {
				assert.InDelta(t, tt.expectedR, colour.Red, 0.001)
				assert.InDelta(t, tt.expectedG, colour.Green, 0.001)
				assert.InDelta(t, tt.expectedB, colour.Blue, 0.001)
				assert.InDelta(t, tt.expectedA, colour.Alpha, 0.001)
			}
		})
	}
}

func TestParseColour_Hex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectedR float64
		expectedG float64
		expectedB float64
		expectedA float64
		expectOK  bool
	}{
		{
			name:      "short hex red",
			input:     "#f00",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{
			name:      "full hex red",
			input:     "#ff0000",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{

			name:      "hex with alpha",
			input:     "#ff000080",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 128.0 / 255,
			expectOK: true,
		},
		{

			name:     "invalid length",
			input:    "#f",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colour, ok := ParseColour(tt.input)
			assert.Equal(t, tt.expectOK, ok)
			if ok {
				assert.InDelta(t, tt.expectedR, colour.Red, 0.001)
				assert.InDelta(t, tt.expectedG, colour.Green, 0.001)
				assert.InDelta(t, tt.expectedB, colour.Blue, 0.001)
				assert.InDelta(t, tt.expectedA, colour.Alpha, 0.001)
			}
		})
	}
}

func TestParseColour_RGB(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectedR float64
		expectedG float64
		expectedB float64
		expectedA float64
		expectOK  bool
	}{
		{
			name:      "rgb integer notation",
			input:     "rgb(255, 0, 0)",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{
			name:      "rgba with alpha",
			input:     "rgba(255, 0, 0, 0.5)",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 0.5,
			expectOK: true,
		},
		{
			name:      "rgb percentage notation",
			input:     "rgb(100%, 0%, 0%)",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{

			name:     "too few parts",
			input:    "rgb(0)",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colour, ok := ParseColour(tt.input)
			assert.Equal(t, tt.expectOK, ok)
			if ok {
				assert.InDelta(t, tt.expectedR, colour.Red, 0.001)
				assert.InDelta(t, tt.expectedG, colour.Green, 0.001)
				assert.InDelta(t, tt.expectedB, colour.Blue, 0.001)
				assert.InDelta(t, tt.expectedA, colour.Alpha, 0.001)
			}
		})
	}
}

func TestParseColour_HSL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectedR float64
		expectedG float64
		expectedB float64
		expectedA float64
		expectOK  bool
	}{
		{

			name:      "red via hsl",
			input:     "hsl(0, 100%, 50%)",
			expectedR: 1, expectedG: 0, expectedB: 0, expectedA: 1,
			expectOK: true,
		},
		{

			name:      "green half alpha via hsla",
			input:     "hsla(120, 100%, 50%, 0.5)",
			expectedR: 0, expectedG: 1, expectedB: 0, expectedA: 0.5,
			expectOK: true,
		},
		{

			name:     "invalid hue",
			input:    "hsl(invalid, 100%, 50%)",
			expectOK: false,
		},
		{

			name:     "too few parts",
			input:    "hsl(0)",
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colour, ok := ParseColour(tt.input)
			assert.Equal(t, tt.expectOK, ok)
			if ok {
				assert.InDelta(t, tt.expectedR, colour.Red, 0.001)
				assert.InDelta(t, tt.expectedG, colour.Green, 0.001)
				assert.InDelta(t, tt.expectedB, colour.Blue, 0.001)
				assert.InDelta(t, tt.expectedA, colour.Alpha, 0.001)
			}
		})
	}
}
