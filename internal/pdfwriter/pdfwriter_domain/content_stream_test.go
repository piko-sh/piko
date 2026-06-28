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
)

func TestFormatFloat_CoercesNonFiniteToZero(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{name: "NaN", value: math.NaN()},
		{name: "positive infinity", value: math.Inf(1)},
		{name: "negative infinity", value: math.Inf(-1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := formatFloat(test.value)
			assert.Equal(t, "0", got, "formatFloat(%v)", test.value)
		})
	}
}

func TestCurveTo(t *testing.T) {
	t.Run("integer values omit decimal points", func(t *testing.T) {
		var stream ContentStream
		stream.CurveTo(10, 20, 30, 40, 50, 60)
		got := stream.String()
		want := "10 20 30 40 50 60 c\n"
		assert.Equal(t, want, got)
	})

	t.Run("non-integer values use two decimal places", func(t *testing.T) {
		var stream ContentStream
		stream.CurveTo(1.5, 2.75, 3.1, 4.99, 5.123, 6.009)
		got := stream.String()
		want := "1.50 2.75 3.10 4.99 5.12 6.01 c\n"
		assert.Equal(t, want, got)
	})

	t.Run("zero values", func(t *testing.T) {
		var stream ContentStream
		stream.CurveTo(0, 0, 0, 0, 0, 0)
		got := stream.String()
		want := "0 0 0 0 0 0 c\n"
		assert.Equal(t, want, got)
	})
}

func TestClosePath(t *testing.T) {
	var stream ContentStream
	stream.ClosePath()
	got := stream.String()
	want := "h\n"
	assert.Equal(t, want, got)
}

func TestClipNonZero(t *testing.T) {
	var stream ContentStream
	stream.ClipNonZero()
	got := stream.String()
	want := "W n\n"
	assert.Equal(t, want, got)
}

func TestSetDashPattern(t *testing.T) {
	t.Run("two-element dash array with integer values", func(t *testing.T) {
		var stream ContentStream
		stream.SetDashPattern([]float64{3, 5}, 0)
		got := stream.String()
		want := "[3 5] 0 d\n"
		assert.Equal(t, want, got)
	})

	t.Run("non-integer dash values use two decimal places", func(t *testing.T) {
		var stream ContentStream
		stream.SetDashPattern([]float64{2.5, 1.75}, 0.5)
		got := stream.String()
		want := "[2.50 1.75] 0.50 d\n"
		assert.Equal(t, want, got)
	})

	t.Run("empty array produces a solid line", func(t *testing.T) {
		var stream ContentStream
		stream.SetDashPattern([]float64{}, 0)
		got := stream.String()
		want := "[] 0 d\n"
		assert.Equal(t, want, got)
	})

	t.Run("single-element array", func(t *testing.T) {
		var stream ContentStream
		stream.SetDashPattern([]float64{4}, 0)
		got := stream.String()
		want := "[4] 0 d\n"
		assert.Equal(t, want, got)
	})

	t.Run("non-zero phase offset", func(t *testing.T) {
		var stream ContentStream
		stream.SetDashPattern([]float64{6, 2}, 3)
		got := stream.String()
		want := "[6 2] 3 d\n"
		assert.Equal(t, want, got)
	})
}

func TestSetLineCap(t *testing.T) {

	tests := []struct {
		name string
		want string
		cap  int
	}{
		{name: "butt cap", cap: 0, want: "0 J\n"},
		{name: "round cap", cap: 1, want: "1 J\n"},
		{name: "projecting square cap", cap: 2, want: "2 J\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetLineCap(test.cap)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetLineJoin(t *testing.T) {

	tests := []struct {
		name string
		want string
		join int
	}{
		{name: "mitre join", join: 0, want: "0 j\n"},
		{name: "round join", join: 1, want: "1 j\n"},
		{name: "bevel join", join: 2, want: "2 j\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetLineJoin(test.join)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetExtGState(t *testing.T) {
	var stream ContentStream
	stream.SetExtGState("GS0")
	got := stream.String()
	want := "/GS0 gs\n"
	assert.Equal(t, want, got)
}

func TestConcatMatrix(t *testing.T) {
	t.Run("identity matrix with integer values", func(t *testing.T) {
		var stream ContentStream
		stream.ConcatMatrix(1, 0, 0, 1, 0, 0)
		got := stream.String()
		want := "1 0 0 1 0 0 cm\n"
		assert.Equal(t, want, got)
	})

	t.Run("translation matrix", func(t *testing.T) {
		var stream ContentStream
		stream.ConcatMatrix(1, 0, 0, 1, 100, 200)
		got := stream.String()
		want := "1 0 0 1 100 200 cm\n"
		assert.Equal(t, want, got)
	})

	t.Run("scaling matrix with non-integer values", func(t *testing.T) {
		var stream ContentStream
		stream.ConcatMatrix(0.5, 0, 0, 0.75, 0, 0)
		got := stream.String()
		want := "0.50 0 0 0.75 0 0 cm\n"
		assert.Equal(t, want, got)
	})
}

func TestPaintXObject(t *testing.T) {
	var stream ContentStream
	stream.PaintXObject("Im0")
	got := stream.String()
	want := "/Im0 Do\n"
	assert.Equal(t, want, got)
}

func TestPaintShading(t *testing.T) {
	var stream ContentStream
	stream.PaintShading("Sh1")
	got := stream.String()
	want := "/Sh1 sh\n"
	assert.Equal(t, want, got)
}

func TestFormatFloat_IntegerValues(t *testing.T) {
	tests := []struct {
		want  string
		value float64
	}{
		{want: "0", value: 0},
		{want: "1", value: 1},
		{want: "10", value: 10},
		{want: "-5", value: -5},
		{want: "100", value: 100},
	}
	for _, test := range tests {
		got := formatFloat(test.value)
		assert.Equal(t, test.want, got, "formatFloat(%v)", test.value)
	}
}

func TestFormatFloat_NonIntegerValues(t *testing.T) {
	tests := []struct {
		want  string
		value float64
	}{
		{want: "0.50", value: 0.5},
		{want: "1.10", value: 1.1},
		{want: "3.14", value: 3.14},
		{want: "2.75", value: 2.755},
		{want: "-0.33", value: -0.33},
	}
	for _, test := range tests {
		got := formatFloat(test.value)
		assert.Equal(t, test.want, got, "formatFloat(%v)", test.value)
	}
}

func TestSetFillColourGrey(t *testing.T) {
	tests := []struct {
		name string
		want string
		grey float64
	}{
		{name: "black", grey: 0, want: "0 g\n"},
		{name: "white", grey: 1, want: "1 g\n"},
		{name: "mid grey", grey: 0.5, want: "0.50 g\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetFillColourGrey(test.grey)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetStrokeColourGrey(t *testing.T) {
	tests := []struct {
		name string
		want string
		grey float64
	}{
		{name: "black", grey: 0, want: "0 G\n"},
		{name: "white", grey: 1, want: "1 G\n"},
		{name: "mid grey", grey: 0.5, want: "0.50 G\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetStrokeColourGrey(test.grey)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetFillColourCMYK(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		cyan    float64
		magenta float64
		yellow  float64
		key     float64
	}{
		{name: "black", cyan: 0, magenta: 0, yellow: 0, key: 1, want: "0 0 0 1 k\n"},
		{name: "cyan", cyan: 1, magenta: 0, yellow: 0, key: 0, want: "1 0 0 0 k\n"},
		{name: "non-integer values", cyan: 0.2, magenta: 0.4, yellow: 0.6, key: 0.1, want: "0.20 0.40 0.60 0.10 k\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetFillColourCMYK(test.cyan, test.magenta, test.yellow, test.key)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetStrokeColourCMYK(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		cyan    float64
		magenta float64
		yellow  float64
		key     float64
	}{
		{name: "black", cyan: 0, magenta: 0, yellow: 0, key: 1, want: "0 0 0 1 K\n"},
		{name: "magenta", cyan: 0, magenta: 1, yellow: 0, key: 0, want: "0 1 0 0 K\n"},
		{name: "non-integer values", cyan: 0.15, magenta: 0.35, yellow: 0.55, key: 0.05, want: "0.15 0.35 0.55 0.05 K\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetStrokeColourCMYK(test.cyan, test.magenta, test.yellow, test.key)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetTextRenderingMode(t *testing.T) {
	tests := []struct {
		name string
		want string
		mode int
	}{
		{name: "fill", mode: 0, want: "0 Tr\n"},
		{name: "stroke", mode: 1, want: "1 Tr\n"},
		{name: "fill then stroke", mode: 2, want: "2 Tr\n"},
		{name: "invisible", mode: 3, want: "3 Tr\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetTextRenderingMode(test.mode)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetCharSpacing(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		spacing float64
	}{
		{name: "zero", spacing: 0, want: "0 Tc\n"},
		{name: "positive integer", spacing: 5, want: "5 Tc\n"},
		{name: "positive float", spacing: 2.5, want: "2.50 Tc\n"},
		{name: "negative", spacing: -1, want: "-1 Tc\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetCharSpacing(test.spacing)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetWordSpacing(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		spacing float64
	}{
		{name: "zero", spacing: 0, want: "0 Tw\n"},
		{name: "positive integer", spacing: 10, want: "10 Tw\n"},
		{name: "positive float", spacing: 3.75, want: "3.75 Tw\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stream ContentStream
			stream.SetWordSpacing(test.spacing)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestOperatorSequence(t *testing.T) {
	var stream ContentStream
	stream.MoveTo(10, 20)
	stream.CurveTo(15, 25, 30, 35, 50, 60)
	stream.ClosePath()
	stream.ClipNonZero()

	got := stream.String()
	want := "10 20 m\n15 25 30 35 50 60 c\nh\nW n\n"
	assert.Equal(t, want, got)
}

func TestSetStrokeColourRGB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		want    string
		r, g, b float64
	}{
		{name: "pure red", r: 1, g: 0, b: 0, want: "1 0 0 RG\n"},
		{name: "pure green", r: 0, g: 1, b: 0, want: "0 1 0 RG\n"},
		{name: "pure blue", r: 0, g: 0, b: 1, want: "0 0 1 RG\n"},
		{name: "non-integer values", r: 0.25, g: 0.50, b: 0.75, want: "0.25 0.50 0.75 RG\n"},
		{name: "black", r: 0, g: 0, b: 0, want: "0 0 0 RG\n"},
		{name: "white", r: 1, g: 1, b: 1, want: "1 1 1 RG\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetStrokeColourRGB(test.r, test.g, test.b)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetFillColourRGB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		want    string
		r, g, b float64
	}{
		{name: "pure red", r: 1, g: 0, b: 0, want: "1 0 0 rg\n"},
		{name: "non-integer values", r: 0.33, g: 0.66, b: 0.99, want: "0.33 0.66 0.99 rg\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetFillColourRGB(test.r, test.g, test.b)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetLineWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		width float64
	}{
		{name: "integer width", width: 2, want: "2 w\n"},
		{name: "fractional width", width: 0.5, want: "0.50 w\n"},
		{name: "zero width", width: 0, want: "0 w\n"},
		{name: "large width", width: 10, want: "10 w\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetLineWidth(test.width)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestRectangle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		want       string
		x, y, w, h float64
	}{
		{name: "integer values", x: 10, y: 20, w: 100, h: 50, want: "10 20 100 50 re\n"},
		{name: "fractional values", x: 1.5, y: 2.5, w: 50.25, h: 30.75, want: "1.50 2.50 50.25 30.75 re\n"},
		{name: "zero origin", x: 0, y: 0, w: 595, h: 842, want: "0 0 595 842 re\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.Rectangle(test.x, test.y, test.w, test.h)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestFill(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.Fill()
	got := stream.String()
	want := "f\n"
	assert.Equal(t, want, got)
}

func TestFillEvenOdd(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.FillEvenOdd()
	got := stream.String()
	want := "f*\n"
	assert.Equal(t, want, got)
}

func TestStroke(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.Stroke()
	got := stream.String()
	want := "S\n"
	assert.Equal(t, want, got)
}

func TestFillAndStroke(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.FillAndStroke()
	got := stream.String()
	want := "B\n"
	assert.Equal(t, want, got)
}

func TestCircle(t *testing.T) {
	t.Parallel()

	t.Run("emits four cubic Bezier curves and close path", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream
		stream.Circle(50, 50, 25)
		got := stream.String()

		lines := 0
		for _, c := range got {
			if c == '\n' {
				lines++
			}
		}
		assert.Equal(t, 6, lines, "expected 6 operator lines")

		assert.Equal(t, "75", got[:2], "expected to start with 75")

		assert.Equal(t, "h\n", got[len(got)-2:], "expected to end with close path")
	})

	t.Run("zero radius produces degenerate circle", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream
		stream.Circle(10, 20, 0)
		got := stream.String()

		assert.NotEmpty(t, got, "expected non-empty output for zero radius circle")
	})
}

func TestClipEvenOdd(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.ClipEvenOdd()
	got := stream.String()
	want := "W* n\n"
	assert.Equal(t, want, got)
}

func TestFillEvenOddAndStroke(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.FillEvenOddAndStroke()
	got := stream.String()
	want := "B*\n"
	assert.Equal(t, want, got)
}

func TestEndPath(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.EndPath()
	got := stream.String()
	want := "n\n"
	assert.Equal(t, want, got)
}

func TestSetMiterLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		limit float64
	}{
		{name: "default mitre limit", limit: 10, want: "10 M\n"},
		{name: "low limit", limit: 1, want: "1 M\n"},
		{name: "fractional limit", limit: 2.5, want: "2.50 M\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetMiterLimit(test.limit)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMoveText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		x    float64
		y    float64
	}{
		{name: "integer offsets", x: 100, y: 200, want: "100 200 Td\n"},
		{name: "fractional offsets", x: 10.5, y: 20.75, want: "10.50 20.75 Td\n"},
		{name: "zero offsets", x: 0, y: 0, want: "0 0 Td\n"},
		{name: "negative offsets", x: -5, y: -10, want: "-5 -10 Td\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.MoveText(test.x, test.y)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestSetTextMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		want             string
		a, b, c, d, e, f float64
	}{
		{name: "identity", a: 1, b: 0, c: 0, d: 1, e: 0, f: 0, want: "1 0 0 1 0 0 Tm\n"},
		{name: "translation", a: 1, b: 0, c: 0, d: 1, e: 72, f: 720, want: "1 0 0 1 72 720 Tm\n"},
		{name: "scaled", a: 0.5, b: 0, c: 0, d: 0.5, e: 10, f: 20, want: "0.50 0 0 0.50 10 20 Tm\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetTextMatrix(test.a, test.b, test.c, test.d, test.e, test.f)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestShowGlyphs(t *testing.T) {
	t.Parallel()

	t.Run("empty glyph list produces no output", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream
		stream.ShowGlyphs(nil, nil, nil, 12)
		got := stream.String()
		assert.Empty(t, got, "expected empty output")
	})

	t.Run("single glyph no adjustment", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream
		stream.ShowGlyphs(
			[]uint16{0x0041},
			[]float64{6.0},
			[]float64{6.0},
			12,
		)
		got := stream.String()
		want := "[<0041>] TJ\n"
		assert.Equal(t, want, got)
	})

	t.Run("two glyphs with adjustment", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream

		stream.ShowGlyphs(
			[]uint16{0x0041, 0x0056},
			[]float64{6.0, 7.0},
			[]float64{7.0, 7.0},
			12,
		)
		got := stream.String()

		require.NotEmpty(t, got, "expected non-empty output")
		assert.Equal(t, byte('['), got[0], "expected output to start with [")
		assert.True(t, strings.HasSuffix(got, "TJ\n"), "expected output to end with TJ, got %q", got)
	})

	t.Run("zero font size skips adjustment", func(t *testing.T) {
		t.Parallel()
		var stream ContentStream
		stream.ShowGlyphs(
			[]uint16{0x0041, 0x0042},
			[]float64{6.0, 7.0},
			[]float64{7.0, 7.0},
			0,
		)
		got := stream.String()
		want := "[<0041><0042>] TJ\n"
		assert.Equal(t, want, got)
	})
}

func TestContentStream_BeginMarkedContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		tag  string
		mcid int
	}{
		{name: "paragraph tag", tag: "P", mcid: 0, want: "/P <</MCID 0>> BDC\n"},
		{name: "span tag with mcid 5", tag: "Span", mcid: 5, want: "/Span <</MCID 5>> BDC\n"},
		{name: "heading tag", tag: "H1", mcid: 42, want: "/H1 <</MCID 42>> BDC\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.BeginMarkedContent(test.tag, test.mcid)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestContentStream_EndMarkedContent(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.EndMarkedContent()
	got := stream.String()
	want := "EMC\n"
	assert.Equal(t, want, got)
}

func TestSaveAndRestoreState(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.SaveState()
	stream.RestoreState()
	got := stream.String()
	want := "q\nQ\n"
	assert.Equal(t, want, got)
}

func TestBeginEndText(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.BeginText()
	stream.EndText()
	got := stream.String()
	want := "BT\nET\n"
	assert.Equal(t, want, got)
}

func TestSetFont(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		want     string
		fontName string
		size     float64
	}{
		{name: "integer size", fontName: "F1", size: 12, want: "/F1 12 Tf\n"},
		{name: "fractional size", fontName: "F2", size: 10.5, want: "/F2 10.50 Tf\n"},
		{name: "Helvetica", fontName: "Helvetica", size: 14, want: "/Helvetica 14 Tf\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.SetFont(test.fontName, test.size)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestShowText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		text string
	}{
		{name: "simple text", text: "Hello", want: "(Hello) Tj\n"},
		{name: "text with parentheses", text: "a(b)c", want: "(a\\(b\\)c) Tj\n"},
		{name: "text with backslash", text: "a\\b", want: "(a\\\\b) Tj\n"},
		{name: "empty string", text: "", want: "() Tj\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stream ContentStream
			stream.ShowText(test.text)
			got := stream.String()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestMoveTo(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.MoveTo(10, 20)
	got := stream.String()
	want := "10 20 m\n"
	assert.Equal(t, want, got)
}

func TestLineTo(t *testing.T) {
	t.Parallel()

	var stream ContentStream
	stream.LineTo(30, 40)
	got := stream.String()
	want := "30 40 l\n"
	assert.Equal(t, want, got)
}
