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

package driven_svgwriter

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColour_HSL_Red(t *testing.T) {
	c, ok := ParseColour("hsl(0, 100%, 50%)")
	require.True(t, ok)
	assert.True(t, math.Abs(c.R-1.0) <= 0.01 && c.G <= 0.01 && c.B <= 0.01, "want ~(1,0,0)")
	assert.Equal(t, 1.0, c.A)
}

func TestParseColour_HSL_Green(t *testing.T) {
	c, ok := ParseColour("hsl(120, 100%, 50%)")
	require.True(t, ok)
	assert.True(t, c.R <= 0.01 && math.Abs(c.G-1.0) <= 0.01 && c.B <= 0.01, "want ~(0,1,0)")
}

func TestParseColour_HSL_Blue(t *testing.T) {
	c, ok := ParseColour("hsl(240, 100%, 50%)")
	require.True(t, ok)
	assert.True(t, c.R <= 0.01 && c.G <= 0.01 && math.Abs(c.B-1.0) <= 0.01, "want ~(0,0,1)")
}

func TestParseColour_HSL_Grey(t *testing.T) {
	c, ok := ParseColour("hsl(0, 0%, 50%)")
	require.True(t, ok)
	assert.True(t, math.Abs(c.R-0.5) <= 0.01 && math.Abs(c.G-0.5) <= 0.01 && math.Abs(c.B-0.5) <= 0.01, "want ~(0.5,0.5,0.5)")
}

func TestParseColour_HSLA_WithAlpha(t *testing.T) {
	c, ok := ParseColour("hsla(0, 100%, 50%, 0.5)")
	require.True(t, ok)
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 0.5, c.A, 0.01)
}

func TestParseColour_HSL_NegativeHue(t *testing.T) {

	c, ok := ParseColour("hsl(-120, 100%, 50%)")
	require.True(t, ok)
	assert.True(t, c.R <= 0.01 && c.G <= 0.01 && math.Abs(c.B-1.0) <= 0.01, "want ~(0,0,1)")
}

func TestParseColour_HSL_DegSuffix(t *testing.T) {
	c, ok := ParseColour("hsl(120deg, 100%, 50%)")
	require.True(t, ok)
	assert.True(t, c.R <= 0.01 && math.Abs(c.G-1.0) <= 0.01 && c.B <= 0.01, "want ~(0,1,0)")
}

func TestParseColour_HSL_Invalid(t *testing.T) {
	_, ok := ParseColour("hsl(abc, 100%, 50%)")
	assert.False(t, ok, "expected ok=false for invalid hue")
	_, ok = ParseColour("hsl(0, 100%)")
	assert.False(t, ok, "expected ok=false for too few args")
}

func TestParseColour_Existing_Named(t *testing.T) {
	c, ok := ParseColour("red")
	assert.True(t, ok)
	assert.Equal(t, 1.0, c.R)
}

func TestParseColour_Existing_Hex(t *testing.T) {
	c, ok := ParseColour("#ff0000")
	assert.True(t, ok)
	assert.Equal(t, 1.0, c.R)
	assert.Equal(t, 0.0, c.G)
	assert.Equal(t, 0.0, c.B)
}

func TestParseColour_Existing_None(t *testing.T) {
	_, ok := ParseColour("none")
	assert.False(t, ok, "expected ok=false for 'none'")
}

func TestParseColour_CurrentColour(t *testing.T) {
	c, ok := ParseColour("currentColor")
	require.True(t, ok)
	assert.True(t, c.IsCurrentColour(), "expected currentColor sentinel")
}

func TestParseColour_RGB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		wantR  float64
		wantG  float64
		wantB  float64
		wantOK bool
	}{
		{
			name:   "pure_red",
			input:  "rgb(255,0,0)",
			wantR:  1.0,
			wantG:  0.0,
			wantB:  0.0,
			wantOK: true,
		},
		{
			name:   "mixed_values",
			input:  "rgb(0,128,255)",
			wantR:  0.0,
			wantG:  128.0 / 255.0,
			wantB:  1.0,
			wantOK: true,
		},
		{
			name:   "percentage_red",
			input:  "rgb(100%,0%,0%)",
			wantR:  1.0,
			wantG:  0.0,
			wantB:  0.0,
			wantOK: true,
		},
		{
			name:   "percentage_mixed",
			input:  "rgb(0%,50%,100%)",
			wantR:  0.0,
			wantG:  0.5,
			wantB:  1.0,
			wantOK: true,
		},
		{
			name:   "too_few_args",
			input:  "rgb(255,0)",
			wantOK: false,
		},
		{
			name:   "too_many_args",
			input:  "rgb(255,0,0,0.5)",
			wantOK: false,
		},
		{
			name:   "invalid_component",
			input:  "rgb(abc,0,0)",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, ok := ParseColour(tt.input)
			require.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.InDelta(t, tt.wantR, c.R, 0.01)
			assert.InDelta(t, tt.wantG, c.G, 0.01)
			assert.InDelta(t, tt.wantB, c.B, 0.01)
			assert.Equal(t, 1.0, c.A)
		})
	}
}

func TestParseColour_RGBA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		wantR  float64
		wantG  float64
		wantB  float64
		wantA  float64
		wantOK bool
	}{
		{
			name:   "red_half_alpha",
			input:  "rgba(255,0,0,0.5)",
			wantR:  1.0,
			wantG:  0.0,
			wantB:  0.0,
			wantA:  0.5,
			wantOK: true,
		},
		{
			name:   "full_alpha",
			input:  "rgba(0,128,255,1)",
			wantR:  0.0,
			wantG:  128.0 / 255.0,
			wantB:  1.0,
			wantA:  1.0,
			wantOK: true,
		},
		{
			name:   "alpha_clamped_above",
			input:  "rgba(255,255,255,2.0)",
			wantR:  1.0,
			wantG:  1.0,
			wantB:  1.0,
			wantA:  1.0,
			wantOK: true,
		},
		{
			name:   "invalid_arg_count_three",
			input:  "rgba(255,0,0)",
			wantOK: false,
		},
		{
			name:   "invalid_arg_count_five",
			input:  "rgba(255,0,0,0.5,1)",
			wantOK: false,
		},
		{
			name:   "bad_alpha",
			input:  "rgba(255,0,0,abc)",
			wantOK: false,
		},
		{
			name:   "bad_colour_component",
			input:  "rgba(abc,0,0,0.5)",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, ok := ParseColour(tt.input)
			require.Equal(t, tt.wantOK, ok)
			if !ok {
				return
			}
			assert.InDelta(t, tt.wantR, c.R, 0.01)
			assert.InDelta(t, tt.wantG, c.G, 0.01)
			assert.InDelta(t, tt.wantB, c.B, 0.01)
			assert.InDelta(t, tt.wantA, c.A, 0.01)
		})
	}
}

func TestParseColour_ShortHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		wantR float64
		wantG float64
		wantB float64
	}{
		{name: "red", input: "#f00", wantR: 1.0, wantG: 0.0, wantB: 0.0},
		{name: "green", input: "#0f0", wantR: 0.0, wantG: 1.0, wantB: 0.0},
		{name: "blue", input: "#00f", wantR: 0.0, wantG: 0.0, wantB: 1.0},
		{name: "white", input: "#fff", wantR: 1.0, wantG: 1.0, wantB: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, ok := ParseColour(tt.input)
			require.True(t, ok)
			assert.InDelta(t, tt.wantR, c.R, 0.01)
			assert.InDelta(t, tt.wantG, c.G, 0.01)
			assert.InDelta(t, tt.wantB, c.B, 0.01)
			assert.Equal(t, 1.0, c.A)
		})
	}
}

func TestParseColour_InvalidHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "invalid_chars", input: "#gg0000"},
		{name: "wrong_length_two", input: "#12"},
		{name: "wrong_length_four", input: "#1234"},
		{name: "wrong_length_five", input: "#12345"},
		{name: "wrong_length_seven", input: "#1234567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := ParseColour(tt.input)
			assert.False(t, ok)
		})
	}
}

func TestParseColour_EmptyString(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("")
	assert.False(t, ok)
}

func TestParseColour_UnknownString(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("notacolour")
	assert.False(t, ok)
}

func TestParseColour_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("   ")
	assert.False(t, ok)
}

func TestClamp01(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{name: "negative", in: -0.5, want: 0},
		{name: "zero", in: 0, want: 0},
		{name: "mid_range", in: 0.5, want: 0.5},
		{name: "one", in: 1.0, want: 1.0},
		{name: "above_one", in: 1.5, want: 1.0},
		{name: "large_negative", in: -100, want: 0},
		{name: "large_positive", in: 100, want: 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := clamp01(tt.in)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}
