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

	"github.com/stretchr/testify/assert"

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

	assert.NotContains(t, got, " d", "solid border should not set a dash pattern")
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

	assert.GreaterOrEqual(t, strokeCount, 8, "double border expected at least 8 strokes")
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

	assert.NotContains(t, got, "0.6667 w", "double border < 3px should not split into thirds")
	assert.NotContains(t, got, "0.66667 w", "double border < 3px should not split into thirds")
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
	assert.GreaterOrEqual(t, rgCount, 4, "groove border expected at least 4 colour changes")
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
	assert.GreaterOrEqual(t, rgCount, 4, "ridge border expected at least 4 colour changes")
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
	assert.GreaterOrEqual(t, rgCount, 2, "inset border expected at least 2 colour settings")
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
	assert.GreaterOrEqual(t, rgCount, 2, "outset border expected at least 2 colour settings")
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
	assert.NotContains(t, got, "S", "zero-width borders should produce no stroke operators")
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
	assert.NotContains(t, got, "S", "border-style: none should produce no stroke operators")
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
	assert.GreaterOrEqual(t, rgCount, 4, "non-uniform rounded border expected at least 4 colour settings")
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
	assert.GreaterOrEqual(t, strokeCount, 2, "rounded double border expected at least 2 strokes")
}

func TestIsUniformBorder_AllSame(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	assert.True(t, isUniformBorder(box), "expected uniform border when all sides are the same")
}

func TestIsUniformBorder_DifferentWidths(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 4, 2, 4).
		WithBorderStyle(layouter_domain.BorderStyleSolid).
		WithBorderColour(testColour(0, 0, 0, 1)).
		Build()

	assert.False(t, isUniformBorder(box), "expected non-uniform border with different widths")
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

	assert.False(t, isUniformBorder(box), "expected non-uniform border with different colours")
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

	assert.False(t, isUniformBorder(box), "expected non-uniform border with different styles")
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
	assert.Empty(t, got, "expected empty stream for zero-width outline")
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
	assert.Empty(t, got, "expected empty stream for none outline")
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
	assert.Empty(t, got, "expected empty stream with single child")
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
	assert.Empty(t, got, "expected empty stream with zero rule width")
}

func TestResolveBorderImageEdges_UsesBorderImageWidth(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 2, 2, 2).
		Build()
	box.Style.BorderImageWidth = 10

	top, right, bottom, left := resolveBorderImageEdges(box)

	assert.EqualValues(t, 10, top, "top edge")
	assert.EqualValues(t, 10, right, "right edge")
	assert.EqualValues(t, 10, bottom, "bottom edge")
	assert.EqualValues(t, 10, left, "left edge")
}

func TestResolveBorderImageEdges_FallsToBorderWidths(t *testing.T) {
	t.Parallel()

	box := newLayoutBox().
		WithBorder(2, 3, 4, 5).
		Build()

	top, right, bottom, left := resolveBorderImageEdges(box)

	assert.EqualValues(t, 2, top, "top edge")
	assert.EqualValues(t, 3, right, "right edge")
	assert.EqualValues(t, 4, bottom, "bottom edge")
	assert.EqualValues(t, 5, left, "left edge")
}

func TestApplyBorderDashPattern_Solid(t *testing.T) {
	t.Parallel()

	painter := newPainterWithDefaults()
	var stream ContentStream
	painter.applyBorderDashPattern(&stream, layouter_domain.BorderStyleSolid, 2)

	got := stream.String()
	assert.Empty(t, got, "solid should not set dash pattern")
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
