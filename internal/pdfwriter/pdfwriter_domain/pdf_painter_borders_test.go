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
	"strings"
	"testing"

	"piko.sh/piko/internal/layouter/layouter_domain"
)

func TestPaintBorders_SolidBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "2 w")
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "0 0 0 RG")

	if strings.Contains(got, " d") {
		t.Error("solid border should not set a dash pattern")
	}
}

func TestPaintBorders_DashedBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleDashed).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	requireStreamContains(t, &stream, "d")
	requireStreamContains(t, &stream, "S")
}

func TestPaintBorders_DottedBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleDotted).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	requireStreamContains(t, &stream, "1 J")
	requireStreamContains(t, &stream, "d")
	requireStreamContains(t, &stream, "S")
}

func TestPaintBorders_DoubleBorderThickEnough(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(6, 6, 6, 6).
		WithBorderStyle(layouter_domain.BorderStyleDouble).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()

	requireStreamContains(t, &stream, "2 w")

	strokeCount := strings.Count(got, "\nS\n") + strings.Count(got, " S\n")

	if strokeCount < 8 {
		t.Errorf("double border expected at least 8 strokes, got %d", strokeCount)
	}
}

func TestPaintBorders_DoubleBorderTooThinFallsBackToSolid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleDouble).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "2 w")

	if strings.Contains(got, "0.6667 w") || strings.Contains(got, "0.66667 w") {
		t.Error("double border < 3px should not split into thirds")
	}
}

func TestPaintBorders_GrooveBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(4, 4, 4, 4).
		WithBorderStyle(layouter_domain.BorderStyleGroove).
		WithBorderColour(testColour(0.5, 0.5, 0.5, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()

	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")

	requireStreamContains(t, &stream, "2 w")

	rgCount := strings.Count(got, "RG")
	if rgCount < 4 {
		t.Errorf("groove border expected at least 4 colour changes, got %d", rgCount)
	}
}

func TestPaintBorders_RidgeBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(4, 4, 4, 4).
		WithBorderStyle(layouter_domain.BorderStyleRidge).
		WithBorderColour(testColour(0.5, 0.5, 0.5, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "2 w")

	rgCount := strings.Count(got, "RG")
	if rgCount < 4 {
		t.Errorf("ridge border expected at least 4 colour changes, got %d", rgCount)
	}
}

func TestPaintBorders_InsetBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleInset).
		WithBorderColour(testColour(0.6, 0.6, 0.6, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")

	rgCount := strings.Count(got, "RG")
	if rgCount < 2 {
		t.Errorf("inset border expected at least 2 colour settings, got %d", rgCount)
	}
}

func TestPaintBorders_OutsetBorder(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleOutset).
		WithBorderColour(testColour(0.6, 0.6, 0.6, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "S")
	rgCount := strings.Count(got, "RG")
	if rgCount < 2 {
		t.Errorf("outset border expected at least 2 colour settings, got %d", rgCount)
	}
}

func TestPaintBorders_ZeroWidthSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithBorder(0, 0, 0, 0).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	if strings.Contains(got, "S") {
		t.Error("zero-width borders should produce no stroke operators")
	}
}

func TestPaintBorders_NoneStyleSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleNone).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()
	if strings.Contains(got, "S") {
		t.Error("border-style: none should produce no stroke operators")
	}
}

func TestPaintBorders_RoundedUniformSolid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(1, 0, 0, 1)).
		WithBorderRadius(10, 10, 10, 10).
		Build()

	painter.paintBorders(&stream, box)

	requireStreamContains(t, &stream, "S")
	requireStreamContains(t, &stream, "2 w")
}

func TestPaintBorders_RoundedNonUniform(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 4, 2, 4).
		WithBorderRadius(10, 10, 10, 10).
		Build()
	box.Style.BorderTopStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderRightStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderBottomStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderLeftStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderTopColour = testColour(1, 0, 0, 1)
	box.Style.BorderRightColour = testColour(0, 1, 0, 1)
	box.Style.BorderBottomColour = testColour(0, 0, 1, 1)
	box.Style.BorderLeftColour = testColour(1, 1, 0, 1)

	painter.paintBorders(&stream, box)

	got := stream.String()

	requireStreamContains(t, &stream, "W")
	requireStreamContains(t, &stream, "S")

	rgCount := strings.Count(got, "RG")
	if rgCount < 4 {
		t.Errorf("non-uniform rounded border expected at least 4 colour settings, got %d", rgCount)
	}
}

func TestPaintBorders_RoundedUniformDouble(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(6, 6, 6, 6).
		WithBorderStyle(layouter_domain.BorderStyleDouble).
		WithBorderColour(testColour(0, 0, 0, 1)).
		WithBorderRadius(10, 10, 10, 10).
		Build()

	painter.paintBorders(&stream, box)

	got := stream.String()

	strokeCount := strings.Count(got, "\nS\n") + strings.Count(got, " S\n")
	if strokeCount < 2 {
		t.Errorf("rounded double border expected at least 2 strokes, got %d", strokeCount)
	}
}

func TestIsUniformBorder_AllSame(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	if !isUniformBorder(box) {
		t.Error("expected uniform border when all sides are the same")
	}
}

func TestIsUniformBorder_DifferentWidths(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 4, 2, 4).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	if isUniformBorder(box) {
		t.Error("expected non-uniform border with different widths")
	}
}

func TestIsUniformBorder_DifferentColours(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		Build()
	box.Style.BorderTopColour = testColour(1, 0, 0, 1)
	box.Style.BorderRightColour = testColour(0, 1, 0, 1)
	box.Style.BorderBottomColour = testColour(0, 0, 1, 1)
	box.Style.BorderLeftColour = testColour(1, 1, 0, 1)

	if isUniformBorder(box) {
		t.Error("expected non-uniform border with different colours")
	}
}

func TestIsUniformBorder_DifferentStyles(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()
	box.Style.BorderTopStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderRightStyle = layouter_domain.BorderStyleDashed
	box.Style.BorderBottomStyle = layouter_domain.BorderStyleSolid
	box.Style.BorderLeftStyle = layouter_domain.BorderStyleDashed

	if isUniformBorder(box) {
		t.Error("expected non-uniform border with different styles")
	}
}

func TestPaintOutline_Solid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.OutlineWidth = 3
	box.Style.OutlineStyle = layouter_domain.BorderStyleSolid
	box.Style.OutlineColour = testColour(0, 0, 1, 1)

	painter.paintOutline(&stream, box)

	requireStreamContains(t, &stream, "3 w")
	requireStreamContains(t, &stream, "RG")
	requireStreamContains(t, &stream, "re")
	requireStreamContains(t, &stream, "S")
}

func TestPaintOutline_ZeroWidthSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		Build()
	box.Style.OutlineWidth = 0
	box.Style.OutlineStyle = layouter_domain.BorderStyleSolid

	painter.paintOutline(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for zero-width outline, got %q", got)
	}
}

func TestPaintOutline_NoneStyleSkipped(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		Build()
	box.Style.OutlineWidth = 3
	box.Style.OutlineStyle = layouter_domain.BorderStyleNone

	painter.paintOutline(&stream, box)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream for none outline, got %q", got)
	}
}

func TestPaintOutline_DashedEmitsDashPattern(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	box := newLayoutBox().
		WithContentRect(20, 20, 100, 50).
		WithPadding(5, 5, 5, 5).
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.OutlineWidth = 2
	box.Style.OutlineStyle = layouter_domain.BorderStyleDashed
	box.Style.OutlineColour = testColour(0, 0, 0, 1)

	painter.paintOutline(&stream, box)

	requireStreamContains(t, &stream, "d")
	requireStreamContains(t, &stream, "S")
}

func TestPaintColumnRules_DrawsRulesBetweenChildren(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	child1 := newLayoutBox().
		WithContentRect(10, 10, 80, 100).
		WithBorder(0, 0, 0, 0).
		Build()
	child2 := newLayoutBox().
		WithContentRect(110, 10, 80, 100).
		WithBorder(0, 0, 0, 0).
		Build()
	parent := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithChildren(child1, child2).
		Build()
	parent.Style.ColumnRuleWidth = 1
	parent.Style.ColumnRuleStyle = layouter_domain.BorderStyleSolid
	parent.Style.ColumnRuleColour = testColour(0, 0, 0, 1)

	painter.paintColumnRules(&stream, parent)

	requireStreamContains(t, &stream, "1 w")
	requireStreamContains(t, &stream, "S")
}

func TestPaintColumnRules_SkippedWithSingleChild(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	child := newLayoutBox().
		WithContentRect(10, 10, 80, 100).
		Build()
	parent := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		WithChildren(child).
		Build()
	parent.Style.ColumnRuleWidth = 1
	parent.Style.ColumnRuleStyle = layouter_domain.BorderStyleSolid

	painter.paintColumnRules(&stream, parent)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream with single child, got %q", got)
	}
}

func TestPaintColumnRules_SkippedWithZeroWidth(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	parent := newLayoutBox().
		WithContentRect(10, 10, 200, 100).
		Build()
	parent.Style.ColumnRuleWidth = 0
	parent.Style.ColumnRuleStyle = layouter_domain.BorderStyleSolid

	painter.paintColumnRules(&stream, parent)

	got := stream.String()
	if got != "" {
		t.Errorf("expected empty stream with zero rule width, got %q", got)
	}
}

func TestResolveBorderImageEdges_UsesBorderImageWidth(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.BorderImageWidth = 10

	top, right, bottom, left := resolveBorderImageEdges(box)

	if top != 10 || right != 10 || bottom != 10 || left != 10 {
		t.Errorf("expected all 10, got (%v, %v, %v, %v)", top, right, bottom, left)
	}
}

func TestResolveBorderImageEdges_FallsToBorderWidths(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 3, 4, 5).
		Build()

	top, right, bottom, left := resolveBorderImageEdges(box)

	if top != 2 || right != 3 || bottom != 4 || left != 5 {
		t.Errorf("expected (2, 3, 4, 5), got (%v, %v, %v, %v)", top, right, bottom, left)
	}
}

func TestApplyBorderDashPattern_Solid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	painter.applyBorderDashPattern(&stream, layouter_domain.BorderStyleSolid, 2)

	got := stream.String()
	if got != "" {
		t.Errorf("solid should not set dash pattern, got %q", got)
	}
}

func TestApplyBorderDashPattern_Dashed(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	painter.applyBorderDashPattern(&stream, layouter_domain.BorderStyleDashed, 2)

	requireStreamContains(t, &stream, "d")
}

func TestApplyBorderDashPattern_Dotted(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	painter.applyBorderDashPattern(&stream, layouter_domain.BorderStyleDotted, 2)

	requireStreamContains(t, &stream, "1 J")
	requireStreamContains(t, &stream, "d")
}
