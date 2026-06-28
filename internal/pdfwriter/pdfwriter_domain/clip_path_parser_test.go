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

package pdfwriter_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClipPath_None(t *testing.T) {
	shape := ParseClipPath("none", 100, 100)
	assert.Equal(t, ClipShapeNone, shape.Type)
}

func TestParseClipPath_Empty(t *testing.T) {
	shape := ParseClipPath("", 100, 100)
	assert.Equal(t, ClipShapeNone, shape.Type, "expected ClipShapeNone for empty string")
}

func TestParseClipPath_Circle(t *testing.T) {
	shape := ParseClipPath("circle(50%)", 200, 200)
	require.Equal(t, ClipShapeCircle, shape.Type)
	assert.InDelta(t, 0.5, shape.RadiusX, 0.01)
	assert.Equal(t, 0.5, shape.CenterX)
	assert.Equal(t, 0.5, shape.CenterY)
}

func TestParseClipPath_CircleWithPosition(t *testing.T) {
	shape := ParseClipPath("circle(30% at 25% 75%)", 200, 200)
	require.Equal(t, ClipShapeCircle, shape.Type)
	assert.InDelta(t, 0.25, shape.CenterX, 0.01)
	assert.InDelta(t, 0.75, shape.CenterY, 0.01)
}

func TestParseClipPath_Ellipse(t *testing.T) {
	shape := ParseClipPath("ellipse(40% 60%)", 200, 100)
	require.Equal(t, ClipShapeEllipse, shape.Type)
	assert.InDelta(t, 0.4, shape.RadiusX, 0.01)
	assert.InDelta(t, 0.6, shape.RadiusY, 0.01)
}

func TestParseClipPath_Inset(t *testing.T) {
	shape := ParseClipPath("inset(10px 20px 30px 40px)", 200, 200)
	require.Equal(t, ClipShapeInset, shape.Type)

	assert.InDelta(t, 7.5, shape.InsetTop, 0.1)
	assert.InDelta(t, 15, shape.InsetRight, 0.1)
}

func TestParseClipPath_InsetWithRound(t *testing.T) {
	shape := ParseClipPath("inset(10% round 5px)", 200, 200)
	require.Equal(t, ClipShapeInset, shape.Type)
	assert.Greater(t, shape.InsetRadius, 0.0, "expected non-zero inset radius")
}

func TestParseClipPath_Polygon(t *testing.T) {
	shape := ParseClipPath("polygon(50% 0%, 100% 100%, 0% 100%)", 200, 200)
	require.Equal(t, ClipShapePolygon, shape.Type)
	require.Len(t, shape.Points, 3)
	assert.InDelta(t, 0.5, shape.Points[0][0], 0.01)
	assert.InDelta(t, 0.0, shape.Points[0][1], 0.01)
}

func TestParseClipPath_Unknown(t *testing.T) {
	shape := ParseClipPath("url(#myClip)", 100, 100)
	assert.Equal(t, ClipShapeNone, shape.Type, "expected ClipShapeNone for url()")
}

func TestEmitClipPath_Circle(t *testing.T) {
	stream := &ContentStream{}
	shape := ClipShape{Type: ClipShapeCircle, CenterX: 0.5, CenterY: 0.5, RadiusX: 0.5}

	EmitClipPath(stream, shape, 0, 0, 100, 100)

	output := stream.String()
	assert.NotEmpty(t, output, "expected non-empty output for circle clip")

	assert.GreaterOrEqual(t, len(output), 10, "expected substantial output for circle clip path")
}

func TestEmitClipPath_Polygon(t *testing.T) {
	stream := &ContentStream{}
	shape := ClipShape{
		Type:   ClipShapePolygon,
		Points: [][2]float64{{0.5, 0}, {1, 1}, {0, 1}},
	}

	EmitClipPath(stream, shape, 0, 0, 100, 100)

	output := stream.String()
	assert.NotEmpty(t, output, "expected non-empty output for polygon clip")
}

func TestEmitClipPath_Ellipse_ProducesBeziersOutput(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{
		Type:    ClipShapeEllipse,
		CenterX: 0.5,
		CenterY: 0.5,
		RadiusX: 0.4,
		RadiusY: 0.3,
	}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()
	assert.NotEmpty(t, output, "expected non-empty output for ellipse clip")

	assert.True(t, strings.Contains(output, " c\n") || strings.Contains(output, " c "), "expected Bezier curve operators in ellipse path")

	assert.True(t, strings.Contains(output, " m\n") || strings.Contains(output, " m "), "expected MoveTo operator in ellipse path")

	assert.Contains(t, output, "h", "expected ClosePath (h) operator in ellipse path")
}

func TestEmitClipPath_Inset_WithoutRadius(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{
		Type:        ClipShapeInset,
		InsetTop:    10,
		InsetRight:  20,
		InsetBottom: 10,
		InsetLeft:   20,
		InsetRadius: 0,
	}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()
	assert.Contains(t, output, "re", "expected rectangle (re) operator for inset without radius")
}

func TestEmitClipPath_Inset_WithRadius(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{
		Type:        ClipShapeInset,
		InsetTop:    5,
		InsetRight:  5,
		InsetBottom: 5,
		InsetLeft:   5,
		InsetRadius: 10,
	}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()

	assert.True(t, strings.Contains(output, " c\n") || strings.Contains(output, " c "), "expected Bezier curve operators for rounded inset")
}

func TestEmitClipPath_Polygon_TooFewVertices_Noop(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{
		Type:   ClipShapePolygon,
		Points: [][2]float64{{0.5, 0}, {1, 1}},
	}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()
	assert.Empty(t, output, "expected empty output for polygon with fewer than 3 vertices")
}

func TestEmitClipPath_None_Noop(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{Type: ClipShapeNone}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()
	assert.Empty(t, output, "expected empty output for ClipShapeNone")
}

func TestResolveClipLength_Percent(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("50%", 200)
	assert.InDelta(t, 100, result, 0.01, "expected 100 for 50%% of 200")
}

func TestResolveClipLength_Px(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("100px", 200)
	expected := 100 * 0.75
	assert.InDelta(t, expected, result, 0.01, "expected %v for 100px", expected)
}

func TestResolveClipLength_Pt(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("72pt", 200)
	assert.InDelta(t, 72, result, 0.01, "expected 72 for 72pt")
}

func TestResolveClipLength_BareNumber(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("100", 200)
	expected := 100 * 0.75
	assert.InDelta(t, expected, result, 0.01, "expected %v for bare number 100", expected)
}

func TestResolveClipLength_InvalidPercent(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abc%", 200)
	assert.Equal(t, 0.0, result, "expected 0 for invalid percent")
}

func TestResolveClipLength_InvalidPx(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abcpx", 200)
	assert.Equal(t, 0.0, result, "expected 0 for invalid px")
}

func TestResolveClipLength_InvalidPt(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abcpt", 200)
	assert.Equal(t, 0.0, result, "expected 0 for invalid pt")
}

func TestResolveClipLength_InvalidBareNumber(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("notanumber", 200)
	assert.Equal(t, 0.0, result, "expected 0 for invalid bare number")
}

func TestParsePercentOrKeyword_Left(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("left")
	assert.Equal(t, 0.0, result, "expected 0 for 'left'")
}

func TestParsePercentOrKeyword_Right(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("right")
	assert.Equal(t, 1.0, result, "expected 1 for 'right'")
}

func TestParsePercentOrKeyword_Top(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("top")
	assert.Equal(t, 0.0, result, "expected 0 for 'top'")
}

func TestParsePercentOrKeyword_Bottom(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("bottom")
	assert.Equal(t, 1.0, result, "expected 1 for 'bottom'")
}

func TestParsePercentOrKeyword_Center(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("center")
	assert.Equal(t, 0.5, result, "expected 0.5 for 'center'")
}

func TestParsePercentOrKeyword_Percent(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("75%")
	assert.InDelta(t, 0.75, result, 0.01, "expected 0.75 for 75%%")
}

func TestParsePercentOrKeyword_InvalidPercent(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("abc%")
	assert.Equal(t, 0.5, result, "expected 0.5 fallback for invalid percent")
}

func TestParsePercentOrKeyword_Unknown(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("unknown")
	assert.Equal(t, 0.5, result, "expected 0.5 fallback for unknown keyword")
}

func TestParseClipInset_OneValue(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(10%)", 200, 200)
	require.Equal(t, ClipShapeInset, shape.Type)

	expected := 20.0
	assert.InDelta(t, expected, shape.InsetTop, 0.1, "expected insetTop %v", expected)
	assert.InDelta(t, expected, shape.InsetBottom, 0.1, "expected insetBottom %v", expected)
	assert.InDelta(t, expected, shape.InsetRight, 0.1, "expected insetRight %v", expected)
	assert.InDelta(t, expected, shape.InsetLeft, 0.1, "expected insetLeft %v", expected)
}

func TestParseClipInset_TwoValues(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(5% 10%)", 200, 200)
	require.Equal(t, ClipShapeInset, shape.Type)

	assert.InDelta(t, 10, shape.InsetTop, 0.1, "expected insetTop 10")
	assert.InDelta(t, 10, shape.InsetBottom, 0.1, "expected insetBottom 10")
	assert.InDelta(t, 20, shape.InsetRight, 0.1, "expected insetRight 20")
	assert.InDelta(t, 20, shape.InsetLeft, 0.1, "expected insetLeft 20")
}

func TestParseClipInset_ThreeValues(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(5% 10% 15%)", 200, 200)
	require.Equal(t, ClipShapeInset, shape.Type)
	assert.InDelta(t, 10, shape.InsetTop, 0.1, "expected insetTop 10 (5%% of 200)")
	assert.InDelta(t, 30, shape.InsetBottom, 0.1, "expected insetBottom 30 (15%% of 200)")

	assert.InDelta(t, shape.InsetLeft, shape.InsetRight, 0.1, "expected insetLeft == insetRight")
}

func TestParseClipEllipse_WithPosition(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("ellipse(30% 40% at 20% 80%)", 200, 100)
	require.Equal(t, ClipShapeEllipse, shape.Type)
	assert.InDelta(t, 0.2, shape.CenterX, 0.01, "expected centerX 0.2")
	assert.InDelta(t, 0.8, shape.CenterY, 0.01, "expected centerY 0.8")
}

func TestParsePosition_Empty(t *testing.T) {
	t.Parallel()

	x, y := parsePosition("")
	assert.Equal(t, 0.5, x, "expected x=0.5 for empty position")
	assert.Equal(t, 0.5, y, "expected y=0.5 for empty position")
}

func TestParsePosition_SingleValue(t *testing.T) {
	t.Parallel()

	x, y := parsePosition("25%")
	assert.InDelta(t, 0.25, x, 0.01, "expected x=0.25")
	assert.Equal(t, 0.5, y, "expected y=0.5 for single value")
}
