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
	"math"
	"strings"
	"testing"
)

func TestParseClipPath_None(t *testing.T) {
	shape := ParseClipPath("none", 100, 100)
	if shape.Type != ClipShapeNone {
		t.Errorf("expected ClipShapeNone, got %d", shape.Type)
	}
}

func TestParseClipPath_Empty(t *testing.T) {
	shape := ParseClipPath("", 100, 100)
	if shape.Type != ClipShapeNone {
		t.Errorf("expected ClipShapeNone for empty string, got %d", shape.Type)
	}
}

func TestParseClipPath_Circle(t *testing.T) {
	shape := ParseClipPath("circle(50%)", 200, 200)
	if shape.Type != ClipShapeCircle {
		t.Fatalf("expected ClipShapeCircle, got %d", shape.Type)
	}
	if math.Abs(shape.RadiusX-0.5) > 0.01 {
		t.Errorf("expected radius 0.5, got %v", shape.RadiusX)
	}
	if shape.CenterX != 0.5 || shape.CenterY != 0.5 {
		t.Errorf("expected center (0.5, 0.5), got (%v, %v)", shape.CenterX, shape.CenterY)
	}
}

func TestParseClipPath_CircleWithPosition(t *testing.T) {
	shape := ParseClipPath("circle(30% at 25% 75%)", 200, 200)
	if shape.Type != ClipShapeCircle {
		t.Fatalf("expected ClipShapeCircle, got %d", shape.Type)
	}
	if math.Abs(shape.CenterX-0.25) > 0.01 {
		t.Errorf("expected centerX 0.25, got %v", shape.CenterX)
	}
	if math.Abs(shape.CenterY-0.75) > 0.01 {
		t.Errorf("expected centerY 0.75, got %v", shape.CenterY)
	}
}

func TestParseClipPath_Ellipse(t *testing.T) {
	shape := ParseClipPath("ellipse(40% 60%)", 200, 100)
	if shape.Type != ClipShapeEllipse {
		t.Fatalf("expected ClipShapeEllipse, got %d", shape.Type)
	}
	if math.Abs(shape.RadiusX-0.4) > 0.01 {
		t.Errorf("expected radiusX 0.4, got %v", shape.RadiusX)
	}
	if math.Abs(shape.RadiusY-0.6) > 0.01 {
		t.Errorf("expected radiusY 0.6, got %v", shape.RadiusY)
	}
}

func TestParseClipPath_Inset(t *testing.T) {
	shape := ParseClipPath("inset(10px 20px 30px 40px)", 200, 200)
	if shape.Type != ClipShapeInset {
		t.Fatalf("expected ClipShapeInset, got %d", shape.Type)
	}

	if math.Abs(shape.InsetTop-7.5) > 0.1 {
		t.Errorf("expected insetTop 7.5, got %v", shape.InsetTop)
	}
	if math.Abs(shape.InsetRight-15) > 0.1 {
		t.Errorf("expected insetRight 15, got %v", shape.InsetRight)
	}
}

func TestParseClipPath_InsetWithRound(t *testing.T) {
	shape := ParseClipPath("inset(10% round 5px)", 200, 200)
	if shape.Type != ClipShapeInset {
		t.Fatalf("expected ClipShapeInset, got %d", shape.Type)
	}
	if shape.InsetRadius <= 0 {
		t.Error("expected non-zero inset radius")
	}
}

func TestParseClipPath_Polygon(t *testing.T) {
	shape := ParseClipPath("polygon(50% 0%, 100% 100%, 0% 100%)", 200, 200)
	if shape.Type != ClipShapePolygon {
		t.Fatalf("expected ClipShapePolygon, got %d", shape.Type)
	}
	if len(shape.Points) != 3 {
		t.Fatalf("expected 3 vertices, got %d", len(shape.Points))
	}
	if math.Abs(shape.Points[0][0]-0.5) > 0.01 {
		t.Errorf("expected first vertex x=0.5, got %v", shape.Points[0][0])
	}
	if math.Abs(shape.Points[0][1]-0.0) > 0.01 {
		t.Errorf("expected first vertex y=0.0, got %v", shape.Points[0][1])
	}
}

func TestParseClipPath_Unknown(t *testing.T) {
	shape := ParseClipPath("url(#myClip)", 100, 100)
	if shape.Type != ClipShapeNone {
		t.Errorf("expected ClipShapeNone for url(), got %d", shape.Type)
	}
}

func TestEmitClipPath_Circle(t *testing.T) {
	stream := &ContentStream{}
	shape := ClipShape{Type: ClipShapeCircle, CenterX: 0.5, CenterY: 0.5, RadiusX: 0.5}

	EmitClipPath(stream, shape, 0, 0, 100, 100)

	output := stream.String()
	if output == "" {
		t.Error("expected non-empty output for circle clip")
	}

	if len(output) < 10 {
		t.Error("expected substantial output for circle clip path")
	}
}

func TestEmitClipPath_Polygon(t *testing.T) {
	stream := &ContentStream{}
	shape := ClipShape{
		Type:   ClipShapePolygon,
		Points: [][2]float64{{0.5, 0}, {1, 1}, {0, 1}},
	}

	EmitClipPath(stream, shape, 0, 0, 100, 100)

	output := stream.String()
	if output == "" {
		t.Error("expected non-empty output for polygon clip")
	}
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
	if output == "" {
		t.Error("expected non-empty output for ellipse clip")
	}

	if !strings.Contains(output, " c\n") && !strings.Contains(output, " c ") {
		t.Error("expected Bezier curve operators in ellipse path")
	}

	if !strings.Contains(output, " m\n") && !strings.Contains(output, " m ") {
		t.Error("expected MoveTo operator in ellipse path")
	}

	if !strings.Contains(output, "h") {
		t.Error("expected ClosePath (h) operator in ellipse path")
	}
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
	if !strings.Contains(output, "re") {
		t.Error("expected rectangle (re) operator for inset without radius")
	}
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

	if !strings.Contains(output, " c\n") && !strings.Contains(output, " c ") {
		t.Error("expected Bezier curve operators for rounded inset")
	}
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
	if output != "" {
		t.Errorf("expected empty output for polygon with fewer than 3 vertices, got %q", output)
	}
}

func TestEmitClipPath_None_Noop(t *testing.T) {
	t.Parallel()

	stream := &ContentStream{}
	shape := ClipShape{Type: ClipShapeNone}

	EmitClipPath(stream, shape, 0, 0, 200, 100)

	output := stream.String()
	if output != "" {
		t.Errorf("expected empty output for ClipShapeNone, got %q", output)
	}
}

func TestResolveClipLength_Percent(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("50%", 200)
	if math.Abs(result-100) > 0.01 {
		t.Errorf("expected 100 for 50%% of 200, got %v", result)
	}
}

func TestResolveClipLength_Px(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("100px", 200)
	expected := 100 * 0.75
	if math.Abs(result-expected) > 0.01 {
		t.Errorf("expected %v for 100px, got %v", expected, result)
	}
}

func TestResolveClipLength_Pt(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("72pt", 200)
	if math.Abs(result-72) > 0.01 {
		t.Errorf("expected 72 for 72pt, got %v", result)
	}
}

func TestResolveClipLength_BareNumber(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("100", 200)
	expected := 100 * 0.75
	if math.Abs(result-expected) > 0.01 {
		t.Errorf("expected %v for bare number 100, got %v", expected, result)
	}
}

func TestResolveClipLength_InvalidPercent(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abc%", 200)
	if result != 0 {
		t.Errorf("expected 0 for invalid percent, got %v", result)
	}
}

func TestResolveClipLength_InvalidPx(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abcpx", 200)
	if result != 0 {
		t.Errorf("expected 0 for invalid px, got %v", result)
	}
}

func TestResolveClipLength_InvalidPt(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("abcpt", 200)
	if result != 0 {
		t.Errorf("expected 0 for invalid pt, got %v", result)
	}
}

func TestResolveClipLength_InvalidBareNumber(t *testing.T) {
	t.Parallel()

	result := resolveClipLength("notanumber", 200)
	if result != 0 {
		t.Errorf("expected 0 for invalid bare number, got %v", result)
	}
}

func TestParsePercentOrKeyword_Left(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("left")
	if result != 0 {
		t.Errorf("expected 0 for 'left', got %v", result)
	}
}

func TestParsePercentOrKeyword_Right(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("right")
	if result != 1 {
		t.Errorf("expected 1 for 'right', got %v", result)
	}
}

func TestParsePercentOrKeyword_Top(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("top")
	if result != 0 {
		t.Errorf("expected 0 for 'top', got %v", result)
	}
}

func TestParsePercentOrKeyword_Bottom(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("bottom")
	if result != 1 {
		t.Errorf("expected 1 for 'bottom', got %v", result)
	}
}

func TestParsePercentOrKeyword_Center(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("center")
	if result != 0.5 {
		t.Errorf("expected 0.5 for 'center', got %v", result)
	}
}

func TestParsePercentOrKeyword_Percent(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("75%")
	if math.Abs(result-0.75) > 0.01 {
		t.Errorf("expected 0.75 for 75%%, got %v", result)
	}
}

func TestParsePercentOrKeyword_InvalidPercent(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("abc%")
	if result != 0.5 {
		t.Errorf("expected 0.5 fallback for invalid percent, got %v", result)
	}
}

func TestParsePercentOrKeyword_Unknown(t *testing.T) {
	t.Parallel()

	result := parsePercentOrKeyword("unknown")
	if result != 0.5 {
		t.Errorf("expected 0.5 fallback for unknown keyword, got %v", result)
	}
}

func TestParseClipInset_OneValue(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(10%)", 200, 200)
	if shape.Type != ClipShapeInset {
		t.Fatalf("expected ClipShapeInset, got %d", shape.Type)
	}

	expected := 20.0
	if math.Abs(shape.InsetTop-expected) > 0.1 {
		t.Errorf("expected insetTop %v, got %v", expected, shape.InsetTop)
	}
	if math.Abs(shape.InsetBottom-expected) > 0.1 {
		t.Errorf("expected insetBottom %v, got %v", expected, shape.InsetBottom)
	}
	if math.Abs(shape.InsetRight-expected) > 0.1 {
		t.Errorf("expected insetRight %v, got %v", expected, shape.InsetRight)
	}
	if math.Abs(shape.InsetLeft-expected) > 0.1 {
		t.Errorf("expected insetLeft %v, got %v", expected, shape.InsetLeft)
	}
}

func TestParseClipInset_TwoValues(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(5% 10%)", 200, 200)
	if shape.Type != ClipShapeInset {
		t.Fatalf("expected ClipShapeInset, got %d", shape.Type)
	}

	if math.Abs(shape.InsetTop-10) > 0.1 {
		t.Errorf("expected insetTop 10, got %v", shape.InsetTop)
	}
	if math.Abs(shape.InsetBottom-10) > 0.1 {
		t.Errorf("expected insetBottom 10, got %v", shape.InsetBottom)
	}
	if math.Abs(shape.InsetRight-20) > 0.1 {
		t.Errorf("expected insetRight 20, got %v", shape.InsetRight)
	}
	if math.Abs(shape.InsetLeft-20) > 0.1 {
		t.Errorf("expected insetLeft 20, got %v", shape.InsetLeft)
	}
}

func TestParseClipInset_ThreeValues(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("inset(5% 10% 15%)", 200, 200)
	if shape.Type != ClipShapeInset {
		t.Fatalf("expected ClipShapeInset, got %d", shape.Type)
	}
	if math.Abs(shape.InsetTop-10) > 0.1 {
		t.Errorf("expected insetTop 10 (5%% of 200), got %v", shape.InsetTop)
	}
	if math.Abs(shape.InsetBottom-30) > 0.1 {
		t.Errorf("expected insetBottom 30 (15%% of 200), got %v", shape.InsetBottom)
	}

	if math.Abs(shape.InsetLeft-shape.InsetRight) > 0.1 {
		t.Errorf("expected insetLeft == insetRight, got left=%v right=%v", shape.InsetLeft, shape.InsetRight)
	}
}

func TestParseClipEllipse_WithPosition(t *testing.T) {
	t.Parallel()

	shape := ParseClipPath("ellipse(30% 40% at 20% 80%)", 200, 100)
	if shape.Type != ClipShapeEllipse {
		t.Fatalf("expected ClipShapeEllipse, got %d", shape.Type)
	}
	if math.Abs(shape.CenterX-0.2) > 0.01 {
		t.Errorf("expected centerX 0.2, got %v", shape.CenterX)
	}
	if math.Abs(shape.CenterY-0.8) > 0.01 {
		t.Errorf("expected centerY 0.8, got %v", shape.CenterY)
	}
}

func TestParsePosition_Empty(t *testing.T) {
	t.Parallel()

	x, y := parsePosition("")
	if x != 0.5 || y != 0.5 {
		t.Errorf("expected (0.5, 0.5) for empty position, got (%v, %v)", x, y)
	}
}

func TestParsePosition_SingleValue(t *testing.T) {
	t.Parallel()

	x, y := parsePosition("25%")
	if math.Abs(x-0.25) > 0.01 {
		t.Errorf("expected x=0.25, got %v", x)
	}
	if y != 0.5 {
		t.Errorf("expected y=0.5 for single value, got %v", y)
	}
}
