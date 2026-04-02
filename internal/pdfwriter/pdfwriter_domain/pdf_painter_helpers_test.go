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
			if math.Abs(got-test.want) > 1e-9 {
				t.Errorf("parseOriginComponent(%q) = %f, want %f", test.input, got, test.want)
			}
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
			if math.Abs(gotX-test.wantX) > 1e-9 {
				t.Errorf("x = %f, want %f", gotX, test.wantX)
			}
			if math.Abs(gotY-test.wantY) > 1e-9 {
				t.Errorf("y = %f, want %f", gotY, test.wantY)
			}
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
			if math.Abs(gotW-test.wantW) > 1e-9 {
				t.Errorf("width = %f, want %f", gotW, test.wantW)
			}
			if math.Abs(gotH-test.wantH) > 1e-9 {
				t.Errorf("height = %f, want %f", gotH, test.wantH)
			}
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
		if math.Abs(gotW-smallW) > 1e-9 || math.Abs(gotH-smallH) > 1e-9 {
			t.Errorf("scale-down should not enlarge: got (%f, %f), want (%f, %f)", gotW, gotH, smallW, smallH)
		}
	})
}

func TestExtractTextContent(t *testing.T) {
	t.Parallel()

	t.Run("direct text", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().WithText("Hello World").Build()
		got := extractTextContent(box)
		if got != "Hello World" {
			t.Errorf("got %q, want %q", got, "Hello World")
		}
	})

	t.Run("nested children", func(t *testing.T) {
		t.Parallel()
		child1 := newLayoutBox().WithText("Hello ").Build()
		child2 := newLayoutBox().WithText("World").Build()
		parent := newLayoutBox().WithChildren(child1, child2).Build()
		got := extractTextContent(parent)
		if got != "Hello World" {
			t.Errorf("got %q, want %q", got, "Hello World")
		}
	})

	t.Run("deeply nested", func(t *testing.T) {
		t.Parallel()
		leaf := newLayoutBox().WithText("deep").Build()
		mid := newLayoutBox().WithChildren(leaf).Build()
		root := newLayoutBox().WithChildren(mid).Build()
		got := extractTextContent(root)
		if got != "deep" {
			t.Errorf("got %q, want %q", got, "deep")
		}
	})

	t.Run("no text returns empty", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().Build()
		got := extractTextContent(box)
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
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
			if math.Abs(got.Red-test.wantR) > 1e-9 {
				t.Errorf("red = %f, want %f", got.Red, test.wantR)
			}
			if math.Abs(got.Green-test.wantG) > 1e-9 {
				t.Errorf("green = %f, want %f", got.Green, test.wantG)
			}
			if math.Abs(got.Blue-test.wantB) > 1e-9 {
				t.Errorf("blue = %f, want %f", got.Blue, test.wantB)
			}
			if got.Alpha != test.colour.Alpha {
				t.Errorf("alpha should be preserved: got %f, want %f", got.Alpha, test.colour.Alpha)
			}
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
			if math.Abs(got.Red-test.wantR) > 1e-9 {
				t.Errorf("red = %f, want %f", got.Red, test.wantR)
			}
			if math.Abs(got.Green-test.wantG) > 1e-9 {
				t.Errorf("green = %f, want %f", got.Green, test.wantG)
			}
			if math.Abs(got.Blue-test.wantB) > 1e-9 {
				t.Errorf("blue = %f, want %f", got.Blue, test.wantB)
			}
			if got.Alpha != test.colour.Alpha {
				t.Errorf("alpha should be preserved: got %f, want %f", got.Alpha, test.colour.Alpha)
			}
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
			if got != test.want {
				t.Errorf("hasAnyBorderRadius() = %v, want %v", got, test.want)
			}
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
			if got != test.want {
				t.Errorf("isEditableFormElement(%q) = %v, want %v", test.tagName, got, test.want)
			}
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
		if !isUniformBorder(box) {
			t.Error("expected uniform border")
		}
	})

	t.Run("non-uniform widths", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 3, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()
		if isUniformBorder(box) {
			t.Error("expected non-uniform border due to different widths")
		}
	})

	t.Run("non-uniform colours", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 2, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()

		box.Style.BorderRightColour = testColour(0, 1, 0, 1)
		if isUniformBorder(box) {
			t.Error("expected non-uniform border due to different colours")
		}
	})

	t.Run("non-uniform styles", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(2, 2, 2, 2).
			WithBorderStyle(layouter_domain.BorderStyleSolid).
			WithBorderColour(testColour(1, 0, 0, 1)).
			Build()
		box.Style.BorderBottomStyle = layouter_domain.BorderStyleDashed
		if isUniformBorder(box) {
			t.Error("expected non-uniform border due to different styles")
		}
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
		if top != 10 || right != 10 || bottom != 10 || left != 10 {
			t.Errorf("expected all edges 10, got %f %f %f %f", top, right, bottom, left)
		}
	})

	t.Run("falls back to border widths", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithBorder(1, 2, 3, 4).
			Build()
		top, right, bottom, left := resolveBorderImageEdges(box)
		if top != 1 || right != 2 || bottom != 3 || left != 4 {
			t.Errorf("expected 1 2 3 4, got %f %f %f %f", top, right, bottom, left)
		}
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
		if math.Abs(got-15.5) > 1e-9 {
			t.Errorf("got %f, want 15.5", got)
		}
	})

	t.Run("falls back to font size ratio when zero", func(t *testing.T) {
		t.Parallel()
		box := newLayoutBox().
			WithFontStyle("sans-serif", 400, 0, 20).
			Build()
		got := resolveBaselineOffset(box)
		want := 20.0 * 0.8
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("got %f, want %f", got, want)
		}
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
			if got != test.want {
				t.Errorf("pdfEscapeString(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestPaintOuterBoxShadows_EmptyShadowList(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(10, 10, 100, 50).WithBorder(2, 2, 2, 2).Build()
	painter.paintOuterBoxShadows(&stream, box)
	if stream.String() != "" {
		t.Errorf("expected empty stream, got %q", stream.String())
	}
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
	if qCount < 2 {
		t.Errorf("blurred shadow expected at least 2 save states, got %d", qCount)
	}
	requireStreamContains(t, &stream, "f*")
}

func TestPaintOuterBoxShadows_SkipsInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5), Inset: true}}
	painter.paintOuterBoxShadows(&stream, box)
	if stream.String() != "" {
		t.Errorf("expected empty stream for inset-only shadows, got %q", stream.String())
	}
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
	if stream.String() != "" {
		t.Errorf("expected empty stream, got %q", stream.String())
	}
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
	if qCount < 2 {
		t.Errorf("blurred inset shadow expected at least 2 save states, got %d", qCount)
	}
}

func TestPaintInsetBoxShadows_SkipsNonInset(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).WithBorder(2, 2, 2, 2).Build()
	box.Style.BoxShadow = []layouter_domain.BoxShadowValue{{OffsetX: 5, OffsetY: 5, Colour: testColour(0, 0, 0, 0.5)}}
	painter.paintInsetBoxShadows(&stream, box)
	if stream.String() != "" {
		t.Errorf("expected empty stream for non-inset shadows, got %q", stream.String())
	}
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
	if len(painter.annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(painter.annotations))
	}
	if painter.annotations[0].uri != "https://example.com" {
		t.Errorf("expected URI, got %q", painter.annotations[0].uri)
	}
	if painter.annotations[0].dest != "" {
		t.Errorf("expected empty dest, got %q", painter.annotations[0].dest)
	}
}

func TestCollectLinkAnnotation_InternalFragment(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("a", "href", "#section1")).Build()
	painter.collectLinkAnnotation(box)
	if len(painter.annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(painter.annotations))
	}
	if painter.annotations[0].dest != "section1" {
		t.Errorf("expected dest 'section1', got %q", painter.annotations[0].dest)
	}
	if painter.annotations[0].uri != "" {
		t.Errorf("expected empty URI, got %q", painter.annotations[0].uri)
	}
}

func TestCollectLinkAnnotation_NonAnchorSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("div", "href", "https://example.com")).Build()
	painter.collectLinkAnnotation(box)
	if len(painter.annotations) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(painter.annotations))
	}
}

func TestCollectLinkAnnotation_NilSourceNode(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).Build()
	painter.collectLinkAnnotation(box)
	if len(painter.annotations) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(painter.annotations))
	}
}

func TestCollectLinkAnnotation_EmptyHref(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("a")).Build()
	painter.collectLinkAnnotation(box)
	if len(painter.annotations) != 0 {
		t.Errorf("expected 0 annotations, got %d", len(painter.annotations))
	}
}

func TestCollectLinkAnnotation_RecordsPageIndex(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithPageIndex(3).WithSourceNode(testSourceNode("a", "href", "https://example.com")).Build()
	painter.collectLinkAnnotation(box)
	if len(painter.annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(painter.annotations))
	}
	if painter.annotations[0].pageIndex != 3 {
		t.Errorf("expected page index 3, got %d", painter.annotations[0].pageIndex)
	}
}

func TestCollectNamedDestination_WithId(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("div", "id", "section1")).Build()
	painter.collectNamedDestination(box)
	if len(painter.namedDests) != 1 {
		t.Fatalf("expected 1 named dest, got %d", len(painter.namedDests))
	}
	if painter.namedDests[0].name != "section1" {
		t.Errorf("expected name 'section1', got %q", painter.namedDests[0].name)
	}
}

func TestCollectNamedDestination_NoId(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithSourceNode(testSourceNode("div", "class", "content")).Build()
	painter.collectNamedDestination(box)
	if len(painter.namedDests) != 0 {
		t.Errorf("expected 0 named dests, got %d", len(painter.namedDests))
	}
}

func TestCollectNamedDestination_NilSourceNode(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).Build()
	painter.collectNamedDestination(box)
	if len(painter.namedDests) != 0 {
		t.Errorf("expected 0 named dests, got %d", len(painter.namedDests))
	}
}

func TestCollectNamedDestination_RecordsPageIndex(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 100, 20).WithBorder(0, 0, 0, 0).WithPageIndex(2).WithSourceNode(testSourceNode("h1", "id", "chapter2")).Build()
	painter.collectNamedDestination(box)
	if len(painter.namedDests) != 1 {
		t.Fatalf("expected 1 named dest, got %d", len(painter.namedDests))
	}
	if painter.namedDests[0].pageIndex != 2 {
		t.Errorf("expected page index 2, got %d", painter.namedDests[0].pageIndex)
	}
}

func TestCollectOutlineEntry_HeadingWithText(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	child := newLayoutBox().WithText("Chapter One").Build()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithBorder(0, 0, 0, 0).WithSourceNode(testSourceNode("h1")).WithChildren(child).Build()
	painter.collectOutlineEntry(box)
	if !painter.outlineBuilder.HasEntries() {
		t.Error("expected outline entry")
	}
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
			if !painter.outlineBuilder.HasEntries() {
				t.Errorf("expected outline entry for %s", tag)
			}
		})
	}
}

func TestCollectOutlineEntry_NonHeadingSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithSourceNode(testSourceNode("p")).Build()
	painter.collectOutlineEntry(box)
	if painter.outlineBuilder.HasEntries() {
		t.Error("expected no outline entry for non-heading")
	}
}

func TestCollectOutlineEntry_EmptyTextSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).WithSourceNode(testSourceNode("h1")).Build()
	painter.collectOutlineEntry(box)
	if painter.outlineBuilder.HasEntries() {
		t.Error("expected no outline entry for empty heading")
	}
}

func TestCollectOutlineEntry_NilSourceNodeSkipped(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(10, 10, 200, 30).Build()
	painter.collectOutlineEntry(box)
	if painter.outlineBuilder.HasEntries() {
		t.Error("expected no outline entry for nil source node")
	}
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
	if !strings.Contains(stream.String(), "c") {
		t.Error("expected Bezier curves for rounded overflow clip")
	}
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
	if math.Abs(ox-expectedX) > 1e-9 {
		t.Errorf("originX: got %v, want %v", ox, expectedX)
	}
	if math.Abs(oy-expectedY) > 1e-9 {
		t.Errorf("originY: got %v, want %v", oy, expectedY)
	}
}

func TestResolveTransformOrigin_LeftTop(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "left top"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX()
	expectedY := painter.pageHeight - box.BorderBoxY()
	if math.Abs(ox-expectedX) > 1e-9 {
		t.Errorf("originX: got %v, want %v", ox, expectedX)
	}
	if math.Abs(oy-expectedY) > 1e-9 {
		t.Errorf("originY: got %v, want %v", oy, expectedY)
	}
}

func TestResolveTransformOrigin_RightBottom(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "right bottom"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX() + box.BorderBoxWidth()
	expectedY := painter.pageHeight - box.BorderBoxY() - box.BorderBoxHeight()
	if math.Abs(ox-expectedX) > 1e-9 {
		t.Errorf("originX: got %v, want %v", ox, expectedX)
	}
	if math.Abs(oy-expectedY) > 1e-9 {
		t.Errorf("originY: got %v, want %v", oy, expectedY)
	}
}

func TestResolveTransformOrigin_PercentageValues(t *testing.T) {
	t.Parallel()
	painter := newPainterWithDefaults()
	box := newLayoutBox().WithContentRect(20, 20, 100, 50).Build()
	box.Style.TransformOrigin = "25% 75%"
	ox, oy := painter.resolveTransformOrigin(box)
	expectedX := box.BorderBoxX() + box.BorderBoxWidth()*0.25
	expectedY := painter.pageHeight - box.BorderBoxY() - box.BorderBoxHeight()*0.75
	if math.Abs(ox-expectedX) > 1e-9 {
		t.Errorf("originX: got %v, want %v", ox, expectedX)
	}
	if math.Abs(oy-expectedY) > 1e-9 {
		t.Errorf("originY: got %v, want %v", oy, expectedY)
	}
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
	if !strings.Contains(stream.String(), "c") {
		t.Error("expected rounded rect path with curves")
	}
}
