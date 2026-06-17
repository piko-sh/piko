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

//go:build !integration

package pdfwriter_domain

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestParseOriginComponent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "left keyword", input: "left", want: 0},
		{name: "top keyword", input: "top", want: 0},
		{name: "right keyword", input: "right", want: 1},
		{name: "bottom keyword", input: "bottom", want: 1},
		{name: "center keyword", input: "center", want: 0.5},
		{name: "centre keyword (British)", input: "centre", want: 0.5},
		{name: "0%", input: "0%", want: 0},
		{name: "50%", input: "50%", want: 0.5},
		{name: "100%", input: "100%", want: 1},
		{name: "25%", input: "25%", want: 0.25},
		{name: "unknown value defaults to 50%", input: "bogus", want: 0.5},
		{name: "empty string defaults to 50%", input: "", want: 0.5},
		{name: "case insensitive LEFT", input: "LEFT", want: 0},
		{name: "whitespace trimmed", input: "  right  ", want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := parseOriginComponent(test.input)
			assert.InDelta(t, test.want, got, 1e-9)
		})
	}
}

func TestParseObjectPosition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		wantX float64
		wantY float64
	}{
		{name: "empty defaults to centre", input: "", wantX: 0.5, wantY: 0.5},
		{name: "single value left", input: "left", wantX: 0, wantY: 0.5},
		{name: "two values", input: "left top", wantX: 0, wantY: 0},
		{name: "percentages", input: "25% 75%", wantX: 0.25, wantY: 0.75},
		{name: "right bottom", input: "right bottom", wantX: 1, wantY: 1},
		{name: "centre centre", input: "centre centre", wantX: 0.5, wantY: 0.5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotX, gotY := parseObjectPosition(test.input)
			assert.InDelta(t, test.wantX, gotX, 1e-9, "x")
			assert.InDelta(t, test.wantY, gotY, 1e-9, "y")
		})
	}
}

func TestResolveObjectFitSize(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	contentW := 200.0
	contentH := 100.0
	intrinsicW := 400.0
	intrinsicH := 300.0

	tests := []struct {
		name  string
		fit   layouter_domain.ObjectFitType
		wantW float64
		wantH float64
	}{
		{
			name:  "fill stretches to content box",
			fit:   layouter_domain.ObjectFitFill,
			wantW: contentW,
			wantH: contentH,
		},
		{
			name:  "contain scales to fit within content box",
			fit:   layouter_domain.ObjectFitContain,
			wantW: intrinsicW * math.Min(contentW/intrinsicW, contentH/intrinsicH),
			wantH: intrinsicH * math.Min(contentW/intrinsicW, contentH/intrinsicH),
		},
		{
			name:  "cover scales to fill content box",
			fit:   layouter_domain.ObjectFitCover,
			wantW: intrinsicW * math.Max(contentW/intrinsicW, contentH/intrinsicH),
			wantH: intrinsicH * math.Max(contentW/intrinsicW, contentH/intrinsicH),
		},
		{
			name:  "none uses intrinsic dimensions",
			fit:   layouter_domain.ObjectFitNone,
			wantW: intrinsicW,
			wantH: intrinsicH,
		},
		{
			name:  "scale-down shrinks when larger than content",
			fit:   layouter_domain.ObjectFitScaleDown,
			wantW: intrinsicW * math.Min(contentW/intrinsicW, contentH/intrinsicH),
			wantH: intrinsicH * math.Min(contentW/intrinsicW, contentH/intrinsicH),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			gotW, gotH := painter.resolveObjectFitSize(test.fit, contentW, contentH, intrinsicW, intrinsicH)
			assert.InDelta(t, test.wantW, gotW, 1e-9, "width")
			assert.InDelta(t, test.wantH, gotH, 1e-9, "height")
		})
	}

	t.Run("scale-down does not enlarge when smaller than content", func(t *testing.T) {
		t.Parallel()
		smallW := 50.0
		smallH := 30.0
		gotW, gotH := painter.resolveObjectFitSize(
			layouter_domain.ObjectFitScaleDown,
			contentW, contentH, smallW, smallH,
		)
		assert.InDelta(t, smallW, gotW, 1e-9, "scale-down should not enlarge width")
		assert.InDelta(t, smallH, gotH, 1e-9, "scale-down should not enlarge height")
	})
}

func TestExtractTextContent(t *testing.T) {
	t.Parallel()

	t.Run("direct text", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().WithText("Hello World").Build()
		got := extractTextContent(box)
		assert.Equal(t, "Hello World", got)
	})

	t.Run("nested children", func(t *testing.T) {
		t.Parallel()
		child1 := newLayoutBox().WithText("Hello ").Build()
		child2 := newLayoutBox().WithText("World").Build()
		parent := newLayoutBox().WithChildren(child1, child2).Build()
		got := extractTextContent(parent)
		assert.Equal(t, "Hello World", got)
	})

	t.Run("deeply nested", func(t *testing.T) {
		t.Parallel()
		leaf := newLayoutBox().WithText("deep").Build()
		mid := newLayoutBox().WithChildren(leaf).Build()
		root := newLayoutBox().WithChildren(mid).Build()
		got := extractTextContent(root)
		assert.Equal(t, "deep", got)
	})

	t.Run("no text returns empty", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().Build()
		got := extractTextContent(box)
		assert.Equal(t, "", got)
	})
}

func TestDarkenColour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		colour layouter_domain.Colour
		factor float64
		wantR  float64
		wantG  float64
		wantB  float64
	}{
		{
			name:   "darken by 50%",
			colour: layouter_domain.Colour{Red: 1.0, Green: 0.8, Blue: 0.6, Alpha: 1.0},
			factor: 0.5,
			wantR:  0.5,
			wantG:  0.4,
			wantB:  0.3,
		},
		{
			name:   "darken to black",
			colour: layouter_domain.Colour{Red: 0.5, Green: 0.5, Blue: 0.5, Alpha: 1.0},
			factor: 0,
			wantR:  0,
			wantG:  0,
			wantB:  0,
		},
		{
			name:   "no darkening (factor 1)",
			colour: layouter_domain.Colour{Red: 0.3, Green: 0.6, Blue: 0.9, Alpha: 0.8},
			factor: 1.0,
			wantR:  0.3,
			wantG:  0.6,
			wantB:  0.9,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := darkenColour(test.colour, test.factor)
			assert.InDelta(t, test.wantR, got.Red, 1e-9, "red")
			assert.InDelta(t, test.wantG, got.Green, 1e-9, "green")
			assert.InDelta(t, test.wantB, got.Blue, 1e-9, "blue")
			assert.Equal(t, test.colour.Alpha, got.Alpha, "alpha should be preserved")
		})
	}
}

func TestLightenColour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		colour layouter_domain.Colour
		factor float64
		wantR  float64
		wantG  float64
		wantB  float64
	}{
		{
			name:   "lighten by 50%",
			colour: layouter_domain.Colour{Red: 0.0, Green: 0.0, Blue: 0.0, Alpha: 1.0},
			factor: 0.5,
			wantR:  0.5,
			wantG:  0.5,
			wantB:  0.5,
		},
		{
			name:   "lighten to white",
			colour: layouter_domain.Colour{Red: 0.5, Green: 0.5, Blue: 0.5, Alpha: 1.0},
			factor: 1.0,
			wantR:  1.0,
			wantG:  1.0,
			wantB:  1.0,
		},
		{
			name:   "no lightening (factor 0)",
			colour: layouter_domain.Colour{Red: 0.3, Green: 0.6, Blue: 0.9, Alpha: 0.7},
			factor: 0,
			wantR:  0.3,
			wantG:  0.6,
			wantB:  0.9,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := lightenColour(test.colour, test.factor)
			assert.InDelta(t, test.wantR, got.Red, 1e-9, "red")
			assert.InDelta(t, test.wantG, got.Green, 1e-9, "green")
			assert.InDelta(t, test.wantB, got.Blue, 1e-9, "blue")
			assert.Equal(t, test.colour.Alpha, got.Alpha, "alpha should be preserved")
		})
	}
}

func TestHasAnyBorderRadius(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()

	tests := []struct {
		name           string
		tl, tr, br, bl float64
		want           bool
	}{
		{name: "no radii", tl: 0, tr: 0, br: 0, bl: 0, want: false},
		{name: "top-left only", tl: 5, tr: 0, br: 0, bl: 0, want: true},
		{name: "top-right only", tl: 0, tr: 5, br: 0, bl: 0, want: true},
		{name: "bottom-right only", tl: 0, tr: 0, br: 5, bl: 0, want: true},
		{name: "bottom-left only", tl: 0, tr: 0, br: 0, bl: 5, want: true},
		{name: "all corners", tl: 10, tr: 10, br: 10, bl: 10, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			box := newLayoutBox().WithBorderRadius(test.tl, test.tr, test.br, test.bl).Build()
			got := painter.hasAnyBorderRadius(box)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestIsEditableFormElement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tagName string
		want    bool
	}{
		{name: "input is editable", tagName: "input", want: true},
		{name: "textarea is editable", tagName: "textarea", want: true},
		{name: "select is editable", tagName: "select", want: true},
		{name: "div is not editable", tagName: "div", want: false},
		{name: "span is not editable", tagName: "span", want: false},
		{name: "button is not editable", tagName: "button", want: false},
		{name: "empty string is not editable", tagName: "", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := isEditableFormElement(test.tagName)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestIsUniformBorder(t *testing.T) {
	t.Parallel()

	t.Run("uniform border", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 2, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()
		assert.True(t, isUniformBorder(box), "expected uniform border")
	})

	t.Run("non-uniform widths", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 3, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()
		assert.False(t, isUniformBorder(box), "expected non-uniform border due to different widths")
	})

	t.Run("non-uniform colours", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 2, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()

		box.Style.BorderRightColour = testColour(0, 1, 0, 1)
		assert.False(t, isUniformBorder(box), "expected non-uniform border due to different colours")
	})

	t.Run("non-uniform styles", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 2, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()
		box.Style.BorderBottomStyle = layouter_domain.BorderStyleDashed
		assert.False(t, isUniformBorder(box), "expected non-uniform border due to different styles")
	})
}

func TestResolveBorderImageEdges(t *testing.T) {
	t.Parallel()

	t.Run("uses border-image-width when set", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(1, 2, 3, 4).
			Build()
		box.Style.BorderImageWidth = 10
		top, right, bottom, left := resolveBorderImageEdges(box)
		assert.Equal(t, 10.0, top, "top")
		assert.Equal(t, 10.0, right, "right")
		assert.Equal(t, 10.0, bottom, "bottom")
		assert.Equal(t, 10.0, left, "left")
	})

	t.Run("falls back to border widths", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(1, 2, 3, 4).
			Build()
		top, right, bottom, left := resolveBorderImageEdges(box)
		assert.Equal(t, 1.0, top, "top")
		assert.Equal(t, 2.0, right, "right")
		assert.Equal(t, 3.0, bottom, "bottom")
		assert.Equal(t, 4.0, left, "left")
	})
}

func TestResolveBaselineOffset(t *testing.T) {
	t.Parallel()

	t.Run("uses layout-computed offset when available", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBaselineOffset(15.5).
			WithFontStyle("sans-serif", 400, 0, 12).
			Build()
		got := resolveBaselineOffset(box)
		assert.InDelta(t, 15.5, got, 1e-9)
	})

	t.Run("falls back to font size ratio when zero", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithFontStyle("sans-serif", 400, 0, 20).
			Build()
		got := resolveBaselineOffset(box)
		want := 20.0 * 0.8
		assert.InDelta(t, want, got, 1e-9)
	})
}

func TestPdfEscapeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no special characters", input: "Hello", want: "Hello"},
		{name: "parentheses escaped", input: "a(b)c", want: "a\\(b\\)c"},
		{name: "backslash escaped", input: "a\\b", want: "a\\\\b"},
		{name: "empty string", input: "", want: ""},
		{name: "mixed specials", input: "(test\\)", want: "\\(test\\\\\\)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := pdfEscapeString(test.input)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestPaintOuterBoxShadows_EmptyShadowList(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(10, 10, 100, 50).WithBorder(2, 2, 2, 2).Build()
	painter.paintOuterBoxShadows(&stream, box)
	assert.Equal(t, "", stream.String(), "expected empty stream")
}

func TestPaintOuterBoxShadows_SharpShadow(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5)}}
	painter.paintOuterBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "rg")
	requireStreamContains(t, &stream, "re")
	requireStreamContains(t, &stream, "f*")
}

func TestPaintOuterBoxShadows_BlurredShadow(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 3, OffsetY: 3, BlurRadius: 10, Colour: testColour(0, 0, 0, 0.5)}}
	painter.paintOuterBoxShadows(&stream, box)
	got := stream.String()
	qCount := strings.Count(got, "\nq\n") + strings.Count(got, "q\n")
	assert.GreaterOrEqual(t, qCount, 2, "blurred shadow expected at least 2 save states")
	requireStreamContains(t, &stream, "f*")
}

func TestPaintOuterBoxShadows_SkipsInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintOuterBoxShadows(&stream, box)
	assert.Equal(t, "", stream.String(), "expected empty stream for inset-only shadows")
}

func TestPaintOuterBoxShadows_WithBorderRadius(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).WithBorderRadius(10, 10, 10, 10).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5)}}
	painter.paintOuterBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "f*")
}

func TestPaintOuterBoxShadows_WithSpread(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{SpreadRadius: 5, Colour: testColour(0, 0, 0, 0.3)}}
	painter.paintOuterBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "f*")
}

func TestPaintInsetBoxShadows_EmptyShadowList(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(10, 10, 100, 50).WithBorder(2, 2, 2, 2).Build()
	painter.paintInsetBoxShadows(&stream, box)
	assert.Equal(t, "", stream.String(), "expected empty stream")
}

func TestPaintInsetBoxShadows_SharpInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 3, OffsetY: 3, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintInsetBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "rg")
	requireStreamContains(t, &stream, "W")
	requireStreamContains(t, &stream, "f")
}

func TestPaintInsetBoxShadows_BlurredInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 3, OffsetY: 3, BlurRadius: 8, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintInsetBoxShadows(&stream, box)
	got := stream.String()
	qCount := strings.Count(got, "\nq\n") + strings.Count(got, "q\n")
	assert.GreaterOrEqual(t, qCount, 2, "blurred inset shadow expected at least 2 save states")
}

func TestPaintInsetBoxShadows_SkipsNonInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5)}}
	painter.paintInsetBoxShadows(&stream, box)
	assert.Equal(t, "", stream.String(), "expected empty stream for non-inset shadows")
}

func TestPaintInsetBoxShadows_WithBorderRadius(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).WithBorderRadius(10, 10, 10, 10).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 3, OffsetY: 3, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintInsetBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "W")
}

func TestPaintInsetBoxShadows_LargeSpreadFillsEntirePadding(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{SpreadRadius: 100, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintInsetBoxShadows(&stream, box)
	requireStreamContains(t, &stream, "f")
}

func TestCollectLinkAnnotation_ExternalURI(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("a", "href", "https://example.com")).Build()
	painter.collectLinkAnnotation(box)
	require.Len(t, painter.annotations, 1)
	assert.Equal(t, "https://example.com", painter.annotations[0].uri)
	assert.Equal(t, "", painter.annotations[0].dest)
}

func TestCollectLinkAnnotation_InternalFragment(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("a", "href", "#section1")).Build()
	painter.collectLinkAnnotation(box)
	require.Len(t, painter.annotations, 1)
	assert.Equal(t, "section1", painter.annotations[0].dest)
	assert.Equal(t, "", painter.annotations[0].uri)
}

func TestCollectLinkAnnotation_NonAnchorSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("div", "href", "https://example.com")).Build()
	painter.collectLinkAnnotation(box)
	assert.Empty(t, painter.annotations, "expected 0 annotations")
}

func TestCollectLinkAnnotation_NilSourceNode(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).Build()
	painter.collectLinkAnnotation(box)
	assert.Empty(t, painter.annotations, "expected 0 annotations")
}

func TestCollectLinkAnnotation_EmptyHref(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("a")).Build()
	painter.collectLinkAnnotation(box)
	assert.Empty(t, painter.annotations, "expected 0 annotations")
}

func TestCollectLinkAnnotation_RecordsPageIndex(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithPageIndex(3).WithSourceNode(testSourceNode("a", "href", "https://example.com")).Build()
	painter.collectLinkAnnotation(box)
	require.Len(t, painter.annotations, 1)
	assert.Equal(t, 3, painter.annotations[0].pageIndex)
}

func TestCollectNamedDestination_WithId(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("div", "id", "section1")).Build()
	painter.collectNamedDestination(box)
	require.Len(t, painter.namedDests, 1)
	assert.Equal(t, "section1", painter.namedDests[0].name)
}

func TestCollectNamedDestination_NoId(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("div", "class", "content")).Build()
	painter.collectNamedDestination(box)
	assert.Empty(t, painter.namedDests, "expected 0 named dests")
}

func TestCollectNamedDestination_NilSourceNode(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).Build()
	painter.collectNamedDestination(box)
	assert.Empty(t, painter.namedDests, "expected 0 named dests")
}

func TestCollectNamedDestination_RecordsPageIndex(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithPageIndex(2).WithSourceNode(testSourceNode("h1", "id", "chapter2")).Build()
	painter.collectNamedDestination(box)
	require.Len(t, painter.namedDests, 1)
	assert.Equal(t, 2, painter.namedDests[0].pageIndex)
}

func TestCollectOutlineEntry_HeadingWithText(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	child := newLayoutBox().WithText("Chapter One").Build()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("h1")).WithChildren(child).Build()
	painter.collectOutlineEntry(box)
	assert.True(t, painter.outlineBuilder.HasEntries(), "expected outline entry")
}

func TestCollectOutlineEntry_H2ThroughH6(t *testing.T) {
	t.Parallel()
	for _, tag := range []string{"h2", "h3", "h4", "h5", "h6"} {
		t.Run(tag, func(t *testing.T) {
			t.Parallel()
			painter := newPainterWithDefaults()
			child := newLayoutBox().WithText("Heading").Build()
			box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode(tag)).WithChildren(child).Build()
			painter.collectOutlineEntry(box)
			assert.True(t, painter.outlineBuilder.HasEntries(), "expected outline entry for %s", tag)
		})
	}
}

func TestCollectOutlineEntry_NonHeadingSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithSourceNode(testSourceNode("p")).Build()
	painter.collectOutlineEntry(box)
	assert.False(t, painter.outlineBuilder.HasEntries(), "expected no outline entry for non-heading")
}

func TestCollectOutlineEntry_EmptyTextSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithSourceNode(testSourceNode("h1")).Build()
	painter.collectOutlineEntry(box)
	assert.False(t, painter.outlineBuilder.HasEntries(), "expected no outline entry for empty heading")
}

func TestCollectOutlineEntry_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).Build()
	painter.collectOutlineEntry(box)
	assert.False(t, painter.outlineBuilder.HasEntries(), "expected no outline entry for nil source node")
}

func TestEmitOverflowClip_WithoutBorderRadius(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	painter.emitOverflowClip(&stream, box)
	requireStreamContains(t, &stream, "re")
	requireStreamContains(t, &stream, "W")
}

func TestEmitOverflowClip_WithBorderRadius(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).WithBorderRadius(10, 10, 10, 10).Build()
	painter.emitOverflowClip(&stream, box)
	requireStreamContains(t, &stream, "W")
	assert.Contains(t, stream.String(), "c", "expected Bezier curves for rounded overflow clip")
}

func TestEmitOverflowClip_ClipsPaddingBox(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(50, 50, 200, 100).WithPadding(10, 10, 10, 10).WithBorder(3, 3, 3, 3).Build()
	painter.emitOverflowClip(&stream, box)
	requireStreamContains(t, &stream, "220")
	requireStreamContains(t, &stream, "120")
}

func TestResolveTransformOrigin_DefaultCentre(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithPadding(5, 5, 5, 5).WithBorder(2, 2, 2, 2).Build()
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX() + box.BorderBoxWidth()*0.5
	expectedY := painter.pageHeight - box.BorderBoxY() - box.BorderBoxHeight()*0.5
	assert.InDelta(t, expectedX, ox, 1e-9, "originX")
	assert.InDelta(t, expectedY, oy, 1e-9, "originY")
}

func TestResolveTransformOrigin_LeftTop(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "left top"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX()
	expectedY := painter.pageHeight - box.BorderBoxY()
	assert.InDelta(t, expectedX, ox, 1e-9, "originX")
	assert.InDelta(t, expectedY, oy, 1e-9, "originY")
}

func TestResolveTransformOrigin_RightBottom(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "right bottom"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX() + box.BorderBoxWidth()
	expectedY := painter.pageHeight - box.BorderBoxY() - box.BorderBoxHeight()
	assert.InDelta(t, expectedX, ox, 1e-9, "originX")
	assert.InDelta(t, expectedY, oy, 1e-9, "originY")
}

func TestResolveTransformOrigin_PercentageValues(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "25% 75%"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX() + box.BorderBoxWidth()*0.25
	expectedY := painter.pageHeight - box.BorderBoxY() - box.BorderBoxHeight()*0.75
	assert.InDelta(t, expectedX, ox, 1e-9, "originX")
	assert.InDelta(t, expectedY, oy, 1e-9, "originY")
}

func TestEmitBorderBoxCutout_Rectangle(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).Build()
	painter.emitBorderBoxCutout(&stream, box, 18, 750, 104, 54, false)
	requireStreamContains(t, &stream, "re")
}

func TestEmitBorderBoxCutout_RoundedRect(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).WithBorderRadius(10, 10, 10, 10).Build()
	painter.emitBorderBoxCutout(&stream, box, 18, 750, 104, 54, true)
	assert.Contains(t, stream.String(), "c", "expected rounded rect path with curves")
}
