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
			assert.InDelta(t, tt.want, s.FillOpacity, 1e-9)
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
			assert.Equal(t, tt.want, s.FillRule)
		})
	}
}

func TestResolveStyle_StrokeWidth(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-width": "3.5"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.InDelta(t, 3.5, s.StrokeWidth, 1e-9)
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
			assert.Equal(t, tt.want, s.StrokeLineCap)
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
			assert.Equal(t, tt.want, s.StrokeLineJoin)
		})
	}
}

func TestResolveStyle_StrokeMitreLimit(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-miterlimit": "8"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.InDelta(t, 8.0, s.StrokeMitreLimit, 1e-9)
}

func TestResolveStyle_StrokeDashArray(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-dasharray": "5,10,15"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	want := []float64{5, 10, 15}
	require.Len(t, s.StrokeDashArray, len(want))
	for i, v := range want {
		assert.InDelta(t, v, s.StrokeDashArray[i], 1e-9, "StrokeDashArray[%d]", i)
	}
}

func TestResolveStyle_StrokeDashArrayNone(t *testing.T) {
	t.Parallel()
	parent := DefaultStyle()

	parent.StrokeDashArray = []float64{1, 2}
	node := &Node{Attrs: map[string]string{"stroke-dasharray": "none"}}
	s := ResolveStyle(node, &parent)
	assert.Nil(t, s.StrokeDashArray, "StrokeDashArray want nil for 'none'")
}

func TestResolveStyle_StrokeDashOffset(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-dashoffset": "3.5"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.InDelta(t, 3.5, s.StrokeDashOffset, 1e-9)
}

func TestResolveStyle_Opacity(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"opacity": "0.7"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.InDelta(t, 0.7, s.Opacity, 1e-9)
}

func TestResolveStyle_DisplayNone(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"display": "none"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "none", s.Display)
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
			assert.Equal(t, tt.want, s.Visibility)
		})
	}
}

func TestResolveStyle_Color(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"color": "#ff0000"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.True(t, math.Abs(s.Colour.R-1.0) <= 0.01 && s.Colour.G <= 0.01 && s.Colour.B <= 0.01, "Color want ~(1,0,0)")
}

func TestResolveStyle_FontFamily(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-family": "Helvetica"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "Helvetica", s.FontFamily)
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
			assert.InDelta(t, tt.want, s.FontSize, 1e-9)
		})
	}
}

func TestResolveStyle_FontWeight(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-weight": "bold"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "bold", s.FontWeight)
}

func TestResolveStyle_FontStyle(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"font-style": "italic"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "italic", s.FontStyle)
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
			assert.Equal(t, tt.want, s.TextAnchor)
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
			assert.Equal(t, tt.want, s.DominantBaseline)
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
			assert.InDelta(t, tt.want, s.LetterSpacing, 1e-9)
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
			assert.InDelta(t, tt.want, s.WordSpacing, 1e-9)
		})
	}
}

func TestResolveStyle_InlineStyle(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{
		"style": "fill:red;stroke:blue",
	}}
	s := ResolveStyle(node, new(DefaultStyle()))

	require.NotNil(t, s.Fill, "Fill is nil, want red")
	assert.True(t, math.Abs(s.Fill.R-1.0) <= 0.01 && s.Fill.G <= 0.01 && s.Fill.B <= 0.01, "Fill want ~(1,0,0)")

	require.NotNil(t, s.Stroke, "Stroke is nil, want blue")
	assert.True(t, s.Stroke.R <= 0.01 && s.Stroke.G <= 0.01 && math.Abs(s.Stroke.B-1.0) <= 0.01, "Stroke want ~(0,0,1)")
}

func TestResolveStyle_InlineStyleOverridesAttribute(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{
		"fill":  "red",
		"style": "fill:blue",
	}}
	s := ResolveStyle(node, new(DefaultStyle()))
	require.NotNil(t, s.Fill, "Fill is nil, want blue")
	assert.True(t, s.Fill.R <= 0.01 && s.Fill.G <= 0.01 && math.Abs(s.Fill.B-1.0) <= 0.01, "Fill want ~(0,0,1) from inline override")
}

func TestResolveStyle_NilNode(t *testing.T) {
	t.Parallel()
	s := ResolveStyle(nil, new(DefaultStyle()))

	assert.Equal(t, "nonzero", s.FillRule)
}

func TestResolveStyle_NilAttrs(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: nil}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "nonzero", s.FillRule)
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

	assert.InDelta(t, 1.0, s.Opacity, 1e-9, "Opacity want 1 (reset)")
	assert.Equal(t, "inline", s.Display, "Display want inline (reset)")
	assert.Nil(t, s.StrokeDashArray, "StrokeDashArray want nil (reset)")
	assert.Equal(t, 0.0, s.StrokeDashOffset, "StrokeDashOffset want 0 (reset)")
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
				assert.Nil(t, got)
				return
			}
			require.Len(t, got, len(tt.want))
			for i, v := range tt.want {
				assert.InDelta(t, v, got[i], 1e-9, "parseDashArray(%q)[%d]", tt.in, i)
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
			require.Len(t, got, len(tt.want))
			for k, wantV := range tt.want {
				gotV, ok := got[k]
				assert.True(t, ok, "parseInlineStyle(%q) missing key %q", tt.in, k)
				assert.Equal(t, wantV, gotV, "parseInlineStyle(%q)[%q]", tt.in, k)
			}
		})
	}
}

func TestResolveStyle_FillNone(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"fill": "none"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Nil(t, s.Fill, "Fill want nil for fill=none")
}

func TestResolveStyle_StrokeNone(t *testing.T) {
	t.Parallel()
	parent := DefaultStyle()
	parent.Stroke = &Colour{R: 1, G: 0, B: 0, A: 1}
	node := &Node{Attrs: map[string]string{"stroke": "none"}}
	s := ResolveStyle(node, &parent)
	assert.Nil(t, s.Stroke, "Stroke want nil for stroke=none")
}

func TestResolveStyle_StrokeColour(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke": "#00ff00"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	require.NotNil(t, s.Stroke, "Stroke is nil, want green")
	assert.True(t, s.Stroke.R <= 0.01 && math.Abs(s.Stroke.G-1.0) <= 0.01 && s.Stroke.B <= 0.01, "Stroke want ~(0,1,0)")
}

func TestResolveStyle_StrokeOpacity(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke-opacity": "0.3"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.InDelta(t, 0.3, s.StrokeOpacity, 1e-9)
}

func TestResolveStyle_TextDecoration(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"text-decoration": "underline"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "underline", s.TextDecoration)
}

func TestResolveStyle_FillURLRef(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"fill": "url(#grad1)"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "grad1", s.FillRef)
}

func TestResolveStyle_StrokeURLRef(t *testing.T) {
	t.Parallel()
	node := &Node{Attrs: map[string]string{"stroke": "url(#grad2)"}}
	s := ResolveStyle(node, new(DefaultStyle()))
	assert.Equal(t, "grad2", s.StrokeRef)
}

func TestDefaultStyle(t *testing.T) {
	t.Parallel()
	s := DefaultStyle()

	require.NotNil(t, s.Fill, "Fill is nil, want black")
	assert.True(t, s.Fill.R == 0 && s.Fill.G == 0 && s.Fill.B == 0 && s.Fill.A == 1, "Fill want (0,0,0,1)")
	assert.InDelta(t, 1.0, s.FillOpacity, 1e-9)
	assert.Equal(t, "nonzero", s.FillRule)
	assert.InDelta(t, 1.0, s.StrokeOpacity, 1e-9)
	assert.InDelta(t, 1.0, s.StrokeWidth, 1e-9)
	assert.Equal(t, "butt", s.StrokeLineCap)
	assert.Equal(t, "miter", s.StrokeLineJoin)
	assert.InDelta(t, 4.0, s.StrokeMitreLimit, 1e-9)
	assert.InDelta(t, 1.0, s.Opacity, 1e-9)
	assert.Equal(t, "inline", s.Display)
	assert.Equal(t, "visible", s.Visibility)
	assert.Equal(t, "sans-serif", s.FontFamily)
	assert.InDelta(t, 16.0, s.FontSize, 1e-9)
	assert.Equal(t, "normal", s.FontWeight)
	assert.Equal(t, "normal", s.FontStyle)
	assert.Equal(t, "start", s.TextAnchor)
	assert.Equal(t, "auto", s.DominantBaseline)
}
