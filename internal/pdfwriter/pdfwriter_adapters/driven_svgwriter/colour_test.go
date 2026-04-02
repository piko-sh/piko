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
)

func TestParseColour_HSL_Red(t *testing.T) {
	c, ok := ParseColour("hsl(0, 100%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if math.Abs(c.R-1.0) > 0.01 || c.G > 0.01 || c.B > 0.01 {
		t.Errorf("hsl(0,100%%,50%%) = (%v,%v,%v), want ~(1,0,0)", c.R, c.G, c.B)
	}
	if c.A != 1 {
		t.Errorf("alpha = %v, want 1", c.A)
	}
}

func TestParseColour_HSL_Green(t *testing.T) {
	c, ok := ParseColour("hsl(120, 100%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if c.R > 0.01 || math.Abs(c.G-1.0) > 0.01 || c.B > 0.01 {
		t.Errorf("hsl(120,100%%,50%%) = (%v,%v,%v), want ~(0,1,0)", c.R, c.G, c.B)
	}
}

func TestParseColour_HSL_Blue(t *testing.T) {
	c, ok := ParseColour("hsl(240, 100%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if c.R > 0.01 || c.G > 0.01 || math.Abs(c.B-1.0) > 0.01 {
		t.Errorf("hsl(240,100%%,50%%) = (%v,%v,%v), want ~(0,0,1)", c.R, c.G, c.B)
	}
}

func TestParseColour_HSL_Grey(t *testing.T) {
	c, ok := ParseColour("hsl(0, 0%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if math.Abs(c.R-0.5) > 0.01 || math.Abs(c.G-0.5) > 0.01 || math.Abs(c.B-0.5) > 0.01 {
		t.Errorf("hsl(0,0%%,50%%) = (%v,%v,%v), want ~(0.5,0.5,0.5)", c.R, c.G, c.B)
	}
}

func TestParseColour_HSLA_WithAlpha(t *testing.T) {
	c, ok := ParseColour("hsla(0, 100%, 50%, 0.5)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if math.Abs(c.R-1.0) > 0.01 {
		t.Errorf("R = %v, want ~1.0", c.R)
	}
	if math.Abs(c.A-0.5) > 0.01 {
		t.Errorf("A = %v, want ~0.5", c.A)
	}
}

func TestParseColour_HSL_NegativeHue(t *testing.T) {

	c, ok := ParseColour("hsl(-120, 100%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if c.R > 0.01 || c.G > 0.01 || math.Abs(c.B-1.0) > 0.01 {
		t.Errorf("hsl(-120,100%%,50%%) = (%v,%v,%v), want ~(0,0,1)", c.R, c.G, c.B)
	}
}

func TestParseColour_HSL_DegSuffix(t *testing.T) {
	c, ok := ParseColour("hsl(120deg, 100%, 50%)")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if c.R > 0.01 || math.Abs(c.G-1.0) > 0.01 || c.B > 0.01 {
		t.Errorf("hsl(120deg,100%%,50%%) = (%v,%v,%v), want ~(0,1,0)", c.R, c.G, c.B)
	}
}

func TestParseColour_HSL_Invalid(t *testing.T) {
	_, ok := ParseColour("hsl(abc, 100%, 50%)")
	if ok {
		t.Error("expected ok=false for invalid hue")
	}
	_, ok = ParseColour("hsl(0, 100%)")
	if ok {
		t.Error("expected ok=false for too few args")
	}
}

func TestParseColour_Existing_Named(t *testing.T) {
	c, ok := ParseColour("red")
	if !ok || c.R != 1.0 {
		t.Errorf("expected red=(1,0,0), got ok=%v, R=%v", ok, c.R)
	}
}

func TestParseColour_Existing_Hex(t *testing.T) {
	c, ok := ParseColour("#ff0000")
	if !ok || c.R != 1.0 || c.G != 0 || c.B != 0 {
		t.Errorf("expected #ff0000=(1,0,0), got ok=%v, (%v,%v,%v)", ok, c.R, c.G, c.B)
	}
}

func TestParseColour_Existing_None(t *testing.T) {
	_, ok := ParseColour("none")
	if ok {
		t.Error("expected ok=false for 'none'")
	}
}

func TestParseColour_CurrentColour(t *testing.T) {
	c, ok := ParseColour("currentColor")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !c.IsCurrentColour() {
		t.Error("expected currentColor sentinel")
	}
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
			if ok != tt.wantOK {
				t.Fatalf("ParseColour(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if math.Abs(c.R-tt.wantR) > 0.01 {
				t.Errorf("R = %v, want ~%v", c.R, tt.wantR)
			}
			if math.Abs(c.G-tt.wantG) > 0.01 {
				t.Errorf("G = %v, want ~%v", c.G, tt.wantG)
			}
			if math.Abs(c.B-tt.wantB) > 0.01 {
				t.Errorf("B = %v, want ~%v", c.B, tt.wantB)
			}
			if c.A != 1 {
				t.Errorf("A = %v, want 1", c.A)
			}
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
			if ok != tt.wantOK {
				t.Fatalf("ParseColour(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if math.Abs(c.R-tt.wantR) > 0.01 {
				t.Errorf("R = %v, want ~%v", c.R, tt.wantR)
			}
			if math.Abs(c.G-tt.wantG) > 0.01 {
				t.Errorf("G = %v, want ~%v", c.G, tt.wantG)
			}
			if math.Abs(c.B-tt.wantB) > 0.01 {
				t.Errorf("B = %v, want ~%v", c.B, tt.wantB)
			}
			if math.Abs(c.A-tt.wantA) > 0.01 {
				t.Errorf("A = %v, want ~%v", c.A, tt.wantA)
			}
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
			if !ok {
				t.Fatalf("ParseColour(%q) ok = false, want true", tt.input)
			}
			if math.Abs(c.R-tt.wantR) > 0.01 {
				t.Errorf("R = %v, want ~%v", c.R, tt.wantR)
			}
			if math.Abs(c.G-tt.wantG) > 0.01 {
				t.Errorf("G = %v, want ~%v", c.G, tt.wantG)
			}
			if math.Abs(c.B-tt.wantB) > 0.01 {
				t.Errorf("B = %v, want ~%v", c.B, tt.wantB)
			}
			if c.A != 1 {
				t.Errorf("A = %v, want 1", c.A)
			}
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
			if ok {
				t.Errorf("ParseColour(%q) ok = true, want false", tt.input)
			}
		})
	}
}

func TestParseColour_EmptyString(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("")
	if ok {
		t.Error("ParseColour(\"\") ok = true, want false")
	}
}

func TestParseColour_UnknownString(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("notacolour")
	if ok {
		t.Error("ParseColour(\"notacolour\") ok = true, want false")
	}
}

func TestParseColour_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	_, ok := ParseColour("   ")
	if ok {
		t.Error("ParseColour(\"   \") ok = true, want false")
	}
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
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("clamp01(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
