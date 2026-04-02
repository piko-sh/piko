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

func TestResolveStyle_FillOpacity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want float64
	}{
		{name: "half", val: "0.5", want: 0.5},
		{name: "zero", val: "0", want: 0},
		{name: "one", val: "1", want: 1},
		{name: "clamped_above", val: "2.0", want: 1},
		{name: "clamped_below", val: "-0.5", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"fill-opacity": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if math.Abs(s.FillOpacity-tt.want) > 1e-9 {
				t.Errorf("FillOpacity = %v, want %v", s.FillOpacity, tt.want)
			}
		})
	}
}

func TestResolveStyle_FillRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "nonzero", val: "nonzero", want: "nonzero"},
		{name: "evenodd", val: "evenodd", want: "evenodd"},
		{name: "invalid_keeps_default", val: "winding", want: "nonzero"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"fill-rule": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.FillRule != tt.want {
				t.Errorf("FillRule = %q, want %q", s.FillRule, tt.want)
			}
		})
	}
}

func TestResolveStyle_StrokeWidth(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-width": "3.5"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.StrokeWidth-3.5) > 1e-9 {
		t.Errorf("StrokeWidth = %v, want 3.5", s.StrokeWidth)
	}
}

func TestResolveStyle_StrokeLineCap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "butt", val: "butt", want: "butt"},
		{name: "round", val: "round", want: "round"},
		{name: "square", val: "square", want: "square"},
		{name: "invalid_keeps_default", val: "flat", want: "butt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"stroke-linecap": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.StrokeLineCap != tt.want {
				t.Errorf("StrokeLineCap = %q, want %q", s.StrokeLineCap, tt.want)
			}
		})
	}
}

func TestResolveStyle_StrokeLineJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "miter", val: "miter", want: "miter"},
		{name: "round", val: "round", want: "round"},
		{name: "bevel", val: "bevel", want: "bevel"},
		{name: "invalid_keeps_default", val: "arcs", want: "miter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"stroke-linejoin": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.StrokeLineJoin != tt.want {
				t.Errorf("StrokeLineJoin = %q, want %q", s.StrokeLineJoin, tt.want)
			}
		})
	}
}

func TestResolveStyle_StrokeMitreLimit(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-miterlimit": "8"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.StrokeMitreLimit-8) > 1e-9 {
		t.Errorf("StrokeMitreLimit = %v, want 8", s.StrokeMitreLimit)
	}
}

func TestResolveStyle_StrokeDashArray(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-dasharray": "5,10,15"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	want := []float64{5, 10, 15}
	if len(s.StrokeDashArray) != len(want) {
		t.Fatalf("StrokeDashArray length = %d, want %d", len(s.StrokeDashArray), len(want))
	}
	for i, v := range want {
		if math.Abs(s.StrokeDashArray[i]-v) > 1e-9 {
			t.Errorf("StrokeDashArray[%d] = %v, want %v", i, s.StrokeDashArray[i], v)
		}
	}
}

func TestResolveStyle_StrokeDashArrayNone(t *testing.T) {
	t.Parallel()
	parent := DefaultStyle()

	parent.StrokeDashArray = []float64{1, 2}
	node := &Node{Attrs: map[string]string{"stroke-dasharray": "none"}}
	s := ResolveStyle(node, &parent)
	if s.StrokeDashArray != nil {
		t.Errorf("StrokeDashArray = %v, want nil for 'none'", s.StrokeDashArray)
	}
}

func TestResolveStyle_StrokeDashOffset(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-dashoffset": "3.5"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.StrokeDashOffset-3.5) > 1e-9 {
		t.Errorf("StrokeDashOffset = %v, want 3.5", s.StrokeDashOffset)
	}
}

func TestResolveStyle_Opacity(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"opacity": "0.7"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.Opacity-0.7) > 1e-9 {
		t.Errorf("Opacity = %v, want 0.7", s.Opacity)
	}
}

func TestResolveStyle_DisplayNone(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"display": "none"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.Display != "none" {
		t.Errorf("Display = %q, want %q", s.Display, "none")
	}
}

func TestResolveStyle_Visibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "visible", val: "visible", want: "visible"},
		{name: "hidden", val: "hidden", want: "hidden"},
		{name: "collapse", val: "collapse", want: "collapse"},
		{name: "invalid_keeps_parent", val: "bogus", want: "visible"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"visibility": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.Visibility != tt.want {
				t.Errorf("Visibility = %q, want %q", s.Visibility, tt.want)
			}
		})
	}
}

func TestResolveStyle_Color(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"color": "#ff0000"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.Colour.R-1.0) > 0.01 || s.Colour.G > 0.01 || s.Colour.B > 0.01 {
		t.Errorf("Color = (%v,%v,%v), want ~(1,0,0)", s.Colour.R, s.Colour.G, s.Colour.B)
	}
}

func TestResolveStyle_FontFamily(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-family": "Helvetica"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.FontFamily != "Helvetica" {
		t.Errorf("FontFamily = %q, want %q", s.FontFamily, "Helvetica")
	}
}

func TestResolveStyle_FontSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want float64
	}{
		{name: "bare_number", val: "14", want: 14},
		{name: "px_suffix", val: "14px", want: 14},
		{name: "decimal", val: "10.5px", want: 10.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"font-size": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if math.Abs(s.FontSize-tt.want) > 1e-9 {
				t.Errorf("FontSize = %v, want %v", s.FontSize, tt.want)
			}
		})
	}
}

func TestResolveStyle_FontWeight(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-weight": "bold"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.FontWeight != "bold" {
		t.Errorf("FontWeight = %q, want %q", s.FontWeight, "bold")
	}
}

func TestResolveStyle_FontStyle(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-style": "italic"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.FontStyle != "italic" {
		t.Errorf("FontStyle = %q, want %q", s.FontStyle, "italic")
	}
}

func TestResolveStyle_TextAnchor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "start", val: "start", want: "start"},
		{name: "middle", val: "middle", want: "middle"},
		{name: "end", val: "end", want: "end"},
		{name: "invalid_keeps_default", val: "left", want: "start"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"text-anchor": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.TextAnchor != tt.want {
				t.Errorf("TextAnchor = %q, want %q", s.TextAnchor, tt.want)
			}
		})
	}
}

func TestResolveStyle_DominantBaseline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want string
	}{
		{name: "auto", val: "auto", want: "auto"},
		{name: "middle", val: "middle", want: "middle"},
		{name: "hanging", val: "hanging", want: "hanging"},
		{name: "central", val: "central", want: "central"},
		{name: "alphabetic", val: "alphabetic", want: "alphabetic"},
		{name: "text_before_edge", val: "text-before-edge", want: "text-before-edge"},
		{name: "text_after_edge", val: "text-after-edge", want: "text-after-edge"},
		{name: "invalid_keeps_default", val: "top", want: "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"dominant-baseline": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if s.DominantBaseline != tt.want {
				t.Errorf("DominantBaseline = %q, want %q", s.DominantBaseline, tt.want)
			}
		})
	}
}

func TestResolveStyle_LetterSpacing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want float64
	}{
		{name: "normal", val: "normal", want: 0},
		{name: "bare_number", val: "2", want: 2},
		{name: "px_suffix", val: "2px", want: 2},
		{name: "decimal_px", val: "1.5px", want: 1.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"letter-spacing": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if math.Abs(s.LetterSpacing-tt.want) > 1e-9 {
				t.Errorf("LetterSpacing = %v, want %v", s.LetterSpacing, tt.want)
			}
		})
	}
}

func TestResolveStyle_WordSpacing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  string
		want float64
	}{
		{name: "normal", val: "normal", want: 0},
		{name: "bare_number", val: "4", want: 4},
		{name: "px_suffix", val: "4px", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node := &Node{Attrs: map[string]string{"word-spacing": tt.val}}
			s := ResolveStyle(node, new(DefaultStyle()))
			if math.Abs(s.WordSpacing-tt.want) > 1e-9 {
				t.Errorf("WordSpacing = %v, want %v", s.WordSpacing, tt.want)
			}
		})
	}
}

func TestResolveStyle_InlineStyle(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{
		"style": "fill:red;stroke:blue",
	}}
	s := ResolveStyle(node, new(DefaultStyle()))

	if s.Fill == nil {
		t.Fatal("Fill is nil, want red")
	}
	if math.Abs(s.Fill.R-1.0) > 0.01 || s.Fill.G > 0.01 || s.Fill.B > 0.01 {
		t.Errorf("Fill = (%v,%v,%v), want ~(1,0,0)", s.Fill.R, s.Fill.G, s.Fill.B)
	}

	if s.Stroke == nil {
		t.Fatal("Stroke is nil, want blue")
	}
	if s.Stroke.R > 0.01 || s.Stroke.G > 0.01 || math.Abs(s.Stroke.B-1.0) > 0.01 {
		t.Errorf("Stroke = (%v,%v,%v), want ~(0,0,1)", s.Stroke.R, s.Stroke.G, s.Stroke.B)
	}
}

func TestResolveStyle_InlineStyleOverridesAttribute(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{
		"fill":  "red",
		"style": "fill:blue",
	}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.Fill == nil {
		t.Fatal("Fill is nil, want blue")
	}
	if s.Fill.R > 0.01 || s.Fill.G > 0.01 || math.Abs(s.Fill.B-1.0) > 0.01 {
		t.Errorf("Fill = (%v,%v,%v), want ~(0,0,1) from inline override", s.Fill.R, s.Fill.G, s.Fill.B)
	}
}

func TestResolveStyle_NilNode(t *testing.T) {
	t.Parallel()
	s := ResolveStyle(nil, new(DefaultStyle()))

	if s.FillRule != "nonzero" {
		t.Errorf("FillRule = %q, want %q", s.FillRule, "nonzero")
	}
}

func TestResolveStyle_NilAttrs(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: nil}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.FillRule != "nonzero" {
		t.Errorf("FillRule = %q, want %q", s.FillRule, "nonzero")
	}
}

func TestResolveStyle_NonInheritedReset(t *testing.T) {
	t.Parallel()
	parent := DefaultStyle()
	parent.Opacity = 0.5
	parent.Display = "none"
	parent.StrokeDashArray = []float64{1, 2}
	parent.StrokeDashOffset = 5

	node := &Node{Attrs: map[string]string{}}
	s := ResolveStyle(node, &parent)

	if math.Abs(s.Opacity-1.0) > 1e-9 {
		t.Errorf("Opacity = %v, want 1 (reset)", s.Opacity)
	}
	if s.Display != "inline" {
		t.Errorf("Display = %q, want %q (reset)", s.Display, "inline")
	}
	if s.StrokeDashArray != nil {
		t.Errorf("StrokeDashArray = %v, want nil (reset)", s.StrokeDashArray)
	}
	if s.StrokeDashOffset != 0 {
		t.Errorf("StrokeDashOffset = %v, want 0 (reset)", s.StrokeDashOffset)
	}
}

func TestParseDashArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []float64
	}{
		{name: "comma_separated", in: "5,10,15", want: []float64{5, 10, 15}},
		{name: "space_separated", in: "5 10", want: []float64{5, 10}},
		{name: "mixed_comma_space", in: "5, 10, 15", want: []float64{5, 10, 15}},
		{name: "single_value", in: "8", want: []float64{8}},
		{name: "invalid_returns_nil", in: "5,abc,10", want: nil},
		{name: "empty_returns_nil", in: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseDashArray(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseDashArray(%q) = %v, want nil", tt.in, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseDashArray(%q) length = %d, want %d", tt.in, len(got), len(tt.want))
			}
			for i, v := range tt.want {
				if math.Abs(got[i]-v) > 1e-9 {
					t.Errorf("parseDashArray(%q)[%d] = %v, want %v", tt.in, i, got[i], v)
				}
			}
		})
	}
}

func TestParseInlineStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "empty",
			in:   "",
			want: map[string]string{},
		},
		{
			name: "single_declaration",
			in:   "fill:red",
			want: map[string]string{"fill": "red"},
		},
		{
			name: "multiple_declarations",
			in:   "fill:red;stroke:blue;opacity:0.5",
			want: map[string]string{"fill": "red", "stroke": "blue", "opacity": "0.5"},
		},
		{
			name: "trailing_semicolon",
			in:   "fill:red;",
			want: map[string]string{"fill": "red"},
		},
		{
			name: "whitespace_around_values",
			in:   " fill : red ; stroke : blue ",
			want: map[string]string{"fill": "red", "stroke": "blue"},
		},
		{
			name: "no_colon_skipped",
			in:   "fill:red;invalid;stroke:blue",
			want: map[string]string{"fill": "red", "stroke": "blue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseInlineStyle(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("parseInlineStyle(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok || gotV != wantV {
					t.Errorf("parseInlineStyle(%q)[%q] = %q, want %q", tt.in, k, gotV, wantV)
				}
			}
		})
	}
}

func TestResolveStyle_FillNone(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"fill": "none"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.Fill != nil {
		t.Errorf("Fill = %v, want nil for fill=none", s.Fill)
	}
}

func TestResolveStyle_StrokeNone(t *testing.T) {
	t.Parallel()
	parent := DefaultStyle()
	parent.Stroke = &Colour{R: 1, G: 0, B: 0, A: 1}
	node := &Node{Attrs: map[string]string{"stroke": "none"}}
	s := ResolveStyle(node, &parent)
	if s.Stroke != nil {
		t.Errorf("Stroke = %v, want nil for stroke=none", s.Stroke)
	}
}

func TestResolveStyle_StrokeColour(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke": "#00ff00"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.Stroke == nil {
		t.Fatal("Stroke is nil, want green")
	}
	if s.Stroke.R > 0.01 || math.Abs(s.Stroke.G-1.0) > 0.01 || s.Stroke.B > 0.01 {
		t.Errorf("Stroke = (%v,%v,%v), want ~(0,1,0)", s.Stroke.R, s.Stroke.G, s.Stroke.B)
	}
}

func TestResolveStyle_StrokeOpacity(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-opacity": "0.3"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if math.Abs(s.StrokeOpacity-0.3) > 1e-9 {
		t.Errorf("StrokeOpacity = %v, want 0.3", s.StrokeOpacity)
	}
}

func TestResolveStyle_TextDecoration(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"text-decoration": "underline"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.TextDecoration != "underline" {
		t.Errorf("TextDecoration = %q, want %q", s.TextDecoration, "underline")
	}
}

func TestResolveStyle_FillURLRef(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"fill": "url(#grad1)"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.FillRef != "grad1" {
		t.Errorf("FillRef = %q, want %q", s.FillRef, "grad1")
	}
}

func TestResolveStyle_StrokeURLRef(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke": "url(#grad2)"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	if s.StrokeRef != "grad2" {
		t.Errorf("StrokeRef = %q, want %q", s.StrokeRef, "grad2")
	}
}

func TestDefaultStyle(t *testing.T) {
	t.Parallel()
	s := DefaultStyle()

	if s.Fill == nil {
		t.Fatal("Fill is nil, want black")
	}
	if s.Fill.R != 0 || s.Fill.G != 0 || s.Fill.B != 0 || s.Fill.A != 1 {
		t.Errorf("Fill = (%v,%v,%v,%v), want (0,0,0,1)", s.Fill.R, s.Fill.G, s.Fill.B, s.Fill.A)
	}
	if math.Abs(s.FillOpacity-1) > 1e-9 {
		t.Errorf("FillOpacity = %v, want 1", s.FillOpacity)
	}
	if s.FillRule != "nonzero" {
		t.Errorf("FillRule = %q, want %q", s.FillRule, "nonzero")
	}
	if math.Abs(s.StrokeOpacity-1) > 1e-9 {
		t.Errorf("StrokeOpacity = %v, want 1", s.StrokeOpacity)
	}
	if math.Abs(s.StrokeWidth-1) > 1e-9 {
		t.Errorf("StrokeWidth = %v, want 1", s.StrokeWidth)
	}
	if s.StrokeLineCap != "butt" {
		t.Errorf("StrokeLineCap = %q, want %q", s.StrokeLineCap, "butt")
	}
	if s.StrokeLineJoin != "miter" {
		t.Errorf("StrokeLineJoin = %q, want %q", s.StrokeLineJoin, "miter")
	}
	if math.Abs(s.StrokeMitreLimit-4) > 1e-9 {
		t.Errorf("StrokeMitreLimit = %v, want 4", s.StrokeMitreLimit)
	}
	if math.Abs(s.Opacity-1) > 1e-9 {
		t.Errorf("Opacity = %v, want 1", s.Opacity)
	}
	if s.Display != "inline" {
		t.Errorf("Display = %q, want %q", s.Display, "inline")
	}
	if s.Visibility != "visible" {
		t.Errorf("Visibility = %q, want %q", s.Visibility, "visible")
	}
	if s.FontFamily != "sans-serif" {
		t.Errorf("FontFamily = %q, want %q", s.FontFamily, "sans-serif")
	}
	if math.Abs(s.FontSize-16) > 1e-9 {
		t.Errorf("FontSize = %v, want 16", s.FontSize)
	}
	if s.FontWeight != "normal" {
		t.Errorf("FontWeight = %q, want %q", s.FontWeight, "normal")
	}
	if s.FontStyle != "normal" {
		t.Errorf("FontStyle = %q, want %q", s.FontStyle, "normal")
	}
	if s.TextAnchor != "start" {
		t.Errorf("TextAnchor = %q, want %q", s.TextAnchor, "start")
	}
	if s.DominantBaseline != "auto" {
		t.Errorf("DominantBaseline = %q, want %q", s.DominantBaseline, "auto")
	}
}
